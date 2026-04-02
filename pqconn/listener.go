package pqconn

import (
	"cmp"
	"fmt"
	"log"
	"maps"
	"runtime/debug"
	"sync"
	"time"

	"github.com/lib/pq"

	"github.com/domonda/go-sqldb"
)

const (
	// DefaultListenerMinReconnectInterval is the default minimum interval
	// between reconnection attempts for the PostgreSQL LISTEN/NOTIFY listener.
	DefaultListenerMinReconnectInterval = 10 * time.Second

	// DefaultListenerMaxReconnectInterval is the default maximum interval
	// between reconnection attempts for the PostgreSQL LISTEN/NOTIFY listener.
	DefaultListenerMaxReconnectInterval = 60 * time.Second

	// DefaultListenerPingInterval is the default interval between
	// keep-alive pings for the PostgreSQL LISTEN/NOTIFY listener.
	DefaultListenerPingInterval = 90 * time.Second
)

type listener struct {
	conn     *pq.Listener
	ping     *time.Ticker
	stop     chan struct{}
	stopOnce sync.Once
	config   *sqldb.Config

	callbacksMtx      sync.RWMutex
	notifyCallbacks   map[string][]sqldb.OnNotifyFunc
	unlistenCallbacks map[string][]sqldb.OnUnlistenFunc
}

// getOrCreateListener returns the global listener for the connection URL,
// creating a new one if none exists.
func (conn *connection) getOrCreateListener() *listener {
	conn.listenerMtx.Lock()
	defer conn.listenerMtx.Unlock()

	if conn.listener == nil {
		minReconnect := cmp.Or(conn.config.ListenerMinReconnectInterval, DefaultListenerMinReconnectInterval)
		maxReconnect := cmp.Or(conn.config.ListenerMaxReconnectInterval, DefaultListenerMaxReconnectInterval)
		pingInterval := cmp.Or(conn.config.ListenerPingInterval, DefaultListenerPingInterval)
		conn.listener = &listener{
			ping:              time.NewTicker(pingInterval),
			stop:              make(chan struct{}),
			config:            conn.config,
			notifyCallbacks:   make(map[string][]sqldb.OnNotifyFunc),
			unlistenCallbacks: make(map[string][]sqldb.OnUnlistenFunc),
		}
		conn.listener.conn = pq.NewListener(
			conn.config.URL().String(),
			minReconnect,
			maxReconnect,
			conn.listener.handleConnectionEvent,
		)

		go conn.listener.listen()
	}

	return conn.listener
}

func (conn *connection) getListenerOrNil() *listener {
	conn.listenerMtx.RLock()
	defer conn.listenerMtx.RUnlock()

	return conn.listener
}

///////////////////////////////////////////////////////////////////////////////
// Main loop and connection event handling
///////////////////////////////////////////////////////////////////////////////

func (l *listener) listen() {
	for {
		select {
		case <-l.stop:
			return

		case notification, isOpen := <-l.conn.Notify:
			if !isOpen {
				l.close()
				return
			}
			// Re-connect will be followed by nil notification
			if notification != nil {
				l.notify(notification)
			}

		case <-l.ping.C:
			// Ping serves as a keep-alive to detect dead connections.
			// On failure we only log because pq.Listener handles
			// reconnection internally — closing here would prevent
			// automatic recovery and channel re-subscription.
			err := l.conn.Ping()
			if err != nil {
				l.logError(fmt.Errorf("pqconn: listener ping failed: %w", err))
			}
		}
	}
}

// handleConnectionEvent is the callback for pq.Listener connection events.
// It logs the event via ListenerEventLogger if configured,
// or falls back to logError for error events.
// On pq.ListenerEventReconnected it calls resubscribeChannels
// because a dropped TCP connection can cause the PostgreSQL backend
// to terminate the session, removing all LISTEN registrations.
// lib/pq's Listener.resync re-issues LISTEN for its internally tracked
// channels but does not verify that server-side subscriptions survived
// the reconnect, so we explicitly re-apply them.
func (l *listener) handleConnectionEvent(event pq.ListenerEventType, err error) {
	if l.config.ListenerEventLogger != nil {
		if err != nil {
			l.config.ListenerEventLogger.Printf("pqconn: got listener connection event=%q error=%v", connectionEvent(event), err)
		} else {
			l.config.ListenerEventLogger.Printf("pqconn: got listener connection event=%q", connectionEvent(event))
		}
	} else if err != nil {
		l.logError(fmt.Errorf("pqconn: got listener connection event=%q error=%w", connectionEvent(event), err))
	}

	if event == pq.ListenerEventReconnected {
		l.resubscribeChannels()
	}
}

// resubscribeChannels re-issues LISTEN for every channel that has
// registered notify or unlisten callbacks. This is called after
// pq.Listener reconnects to ensure the PostgreSQL session is
// subscribed to all expected channels.
func (l *listener) resubscribeChannels() {
	if l.isStopped() {
		return
	}

	l.callbacksMtx.RLock()
	channels := make([]string, 0, len(l.notifyCallbacks)+len(l.unlistenCallbacks))
	for ch := range l.notifyCallbacks {
		channels = append(channels, ch)
	}
	for ch := range l.unlistenCallbacks {
		if _, hasNotify := l.notifyCallbacks[ch]; !hasNotify {
			channels = append(channels, ch)
		}
	}
	l.callbacksMtx.RUnlock()

	for _, channel := range channels {
		// pq.Listener.resync may have already re-subscribed the channel,
		// in which case we get ErrChannelAlreadyOpen which is expected.
		// Any other error means the channel was not re-subscribed.
		if err := l.conn.Listen(channel); err != nil && err != pq.ErrChannelAlreadyOpen {
			l.logError(fmt.Errorf("pqconn: failed to resubscribe to channel %q after reconnect: %w", channel, err))
		}
	}
}

func connectionEvent(event pq.ListenerEventType) string {
	switch event {
	case pq.ListenerEventConnected:
		return "connected"
	case pq.ListenerEventDisconnected:
		return "disconnected"
	case pq.ListenerEventReconnected:
		return "reconnected"
	case pq.ListenerEventConnectionAttemptFailed:
		return "connection attempt failed"
	default:
		return fmt.Sprintf("unknown(%d)", event)
	}
}

///////////////////////////////////////////////////////////////////////////////
// Notification dispatch
///////////////////////////////////////////////////////////////////////////////

func (l *listener) notify(notification *pq.Notification) {
	l.callbacksMtx.RLock()
	defer l.callbacksMtx.RUnlock()

	for _, callback := range l.notifyCallbacks[notification.Channel] {
		go l.safeNotifyCallback(callback, notification.Channel, notification.Extra)
	}
}

func (l *listener) safeNotifyCallback(callback sqldb.OnNotifyFunc, channel, payload string) {
	defer func() {
		if p := recover(); p != nil {
			l.logError(fmt.Errorf("pqconn: notify callback on channel %q panicked with: %+v\n%s", channel, p, debug.Stack()))
		}
	}()

	callback(channel, payload)
}

///////////////////////////////////////////////////////////////////////////////
// Channel subscription management
///////////////////////////////////////////////////////////////////////////////

func (l *listener) listenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	if l.isStopped() {
		return fmt.Errorf("pqconn: unable to listenOnChannel %q: listener is closed", channel)
	}
	err = l.conn.Listen(channel)
	if err != nil && err != pq.ErrChannelAlreadyOpen {
		return fmt.Errorf("pqconn: failed to listenOnChannel %q: %w", channel, err)
	}

	l.callbacksMtx.Lock()
	defer l.callbacksMtx.Unlock()

	if onNotify != nil {
		l.notifyCallbacks[channel] = append(l.notifyCallbacks[channel], onNotify)
	}
	if onUnlisten != nil {
		l.unlistenCallbacks[channel] = append(l.unlistenCallbacks[channel], onUnlisten)
	}

	return nil
}

// unlistenChannel removes all callbacks for the channel
// and calls the registered unlisten callbacks.
// Called on nil listener will return an error.
func (l *listener) unlistenChannel(channel string) (err error) {
	if l == nil || l.isStopped() {
		return fmt.Errorf("pqconn: unable to unlistenChannel %q: no listener active", channel)
	}

	err = l.conn.Unlisten(channel)
	if err != nil {
		return fmt.Errorf("pqconn: failed to unlistenChannel %q: %w", channel, err)
	}

	l.callbacksMtx.Lock()
	callbacks := l.unlistenCallbacks[channel]
	delete(l.notifyCallbacks, channel)
	delete(l.unlistenCallbacks, channel)
	l.callbacksMtx.Unlock()

	var wg sync.WaitGroup
	for _, callback := range callbacks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.safeUnlistenCallback(callback, channel)
		}()
	}
	wg.Wait()

	return nil
}

func (l *listener) isListeningOnChannel(channel string) bool {
	if l == nil {
		return false
	}

	l.callbacksMtx.RLock()
	defer l.callbacksMtx.RUnlock()

	return len(l.notifyCallbacks[channel]) > 0 || len(l.unlistenCallbacks[channel]) > 0
}

///////////////////////////////////////////////////////////////////////////////
// Lifecycle
///////////////////////////////////////////////////////////////////////////////

// close stops the listener, closes the underlying connection,
// and calls all registered unlisten callbacks.
// Waits for all unlisten callbacks to complete before returning.
// Only the first call performs cleanup; subsequent calls are no-ops.
func (l *listener) close() {
	if l == nil {
		return
	}
	l.stopOnce.Do(func() {
		close(l.stop)

		l.ping.Stop()
		l.conn.Close() //#nosec G104 -- Don't care about close errors

		l.callbacksMtx.Lock()
		unlistenCallbacks := make(map[string][]sqldb.OnUnlistenFunc, len(l.unlistenCallbacks))
		maps.Copy(unlistenCallbacks, l.unlistenCallbacks)
		clear(l.notifyCallbacks)
		clear(l.unlistenCallbacks)
		l.callbacksMtx.Unlock()

		var wg sync.WaitGroup
		for channel, callbacks := range unlistenCallbacks {
			for _, callback := range callbacks {
				wg.Add(1)
				go func() {
					defer wg.Done()
					l.safeUnlistenCallback(callback, channel)
				}()
			}
		}
		wg.Wait()
	})
}

func (l *listener) safeUnlistenCallback(callback sqldb.OnUnlistenFunc, channel string) {
	defer func() {
		if p := recover(); p != nil {
			l.logError(fmt.Errorf("pqconn: unlisten callback on channel %q panicked with: %+v\n%s", channel, p, debug.Stack()))
		}
	}()

	callback(channel)
}

func (l *listener) isStopped() bool {
	select {
	case <-l.stop:
		return true
	default:
		return false
	}
}

// logError logs the error to the first available logger:
// ErrLogger, then ListenerEventLogger.
// Does nothing if err is nil or no logger is configured.
func (l *listener) logError(err error) {
	switch {
	case err == nil:
		return
	case l != nil && l.config.ErrLogger != nil:
		l.config.ErrLogger.Printf("%v", err)
	case l != nil && l.config.ListenerEventLogger != nil:
		l.config.ListenerEventLogger.Printf("%v", err)
	default:
		log.Printf("%v", err)
	}
}
