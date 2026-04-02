package pqconn

import (
	"fmt"
	"sync"
	"time"

	"github.com/lib/pq"

	"github.com/domonda/go-sqldb"
)

var (
	// ListenerMinReconnectInterval is the minimum interval between reconnection attempts
	// for the PostgreSQL LISTEN/NOTIFY listener.
	ListenerMinReconnectInterval = time.Second * 10
	// ListenerMaxReconnectInterval is the maximum interval between reconnection attempts
	// for the PostgreSQL LISTEN/NOTIFY listener.
	ListenerMaxReconnectInterval = time.Second * 60
	// ListenerPingInterval is the interval between keep-alive pings
	// for the PostgreSQL LISTEN/NOTIFY listener.
	ListenerPingInterval = time.Second * 90
)

type listener struct {
	conn     *pq.Listener
	ping     *time.Ticker
	stop     chan struct{}
	stopOnce sync.Once

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
		conn.listener = &listener{
			conn: pq.NewListener(
				conn.config.URL().String(),
				ListenerMinReconnectInterval,
				ListenerMaxReconnectInterval,
				logListenerConnectionEvent,
			),
			ping:              time.NewTicker(ListenerPingInterval),
			stop:              make(chan struct{}),
			notifyCallbacks:   make(map[string][]sqldb.OnNotifyFunc),
			unlistenCallbacks: make(map[string][]sqldb.OnUnlistenFunc),
		}

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
			if err := l.conn.Ping(); err != nil {
				sqldb.ErrLogger.Printf("pqconn: listener ping failed: %v", err)
			}
		}
	}
}

// handleConnectionEvent is the callback for pq.Listener connection events.
// On pq.ListenerEventReconnected we must re-subscribe all channels because
// a dropped TCP connection can cause the PostgreSQL backend to terminate
// the session, which removes all LISTEN registrations on the server side.
// lib/pq's Listener.resync re-issues LISTEN for channels it tracks internally,
// but does not verify that the server-side subscriptions are still active
// after reconnect, so we explicitly re-apply all channel subscriptions
// to guarantee no notifications are lost.
func (l *listener) handleConnectionEvent(event pq.ListenerEventType, err error) {
	switch {
	case err != nil:
		sqldb.ErrLogger.Printf("pqconn: got listener connection event=%q error=%v", connectionEvent(event), err)
	case ListenerEventLogger != nil:
		ListenerEventLogger.Printf("pqconn: got listener connection event=%q", connectionEvent(event))
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
	l.stopOnce.Do(func() {
		close(l.stop)

		l.ping.Stop()
		l.conn.Close() //#nosec G104 -- Don't care about close errors

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
		if err := l.conn.Listen(channel); err != nil && err != pq.ErrChannelAlreadyOpen {
			// pq.Listener.resync may have already re-subscribed the channel,
			// in which case we get ErrChannelAlreadyOpen which is expected.
			// Any other error means the channel was not re-subscribed.
			sqldb.ErrLogger.Printf("pqconn: failed to resubscribe to channel %q after reconnect: %v", channel, err)
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
	// Copy slice to be able to immediately unlock again
	callbacks := append([]sqldb.OnNotifyFunc(nil), l.notifyCallbacks[notification.Channel]...)
	l.callbacksMtx.RUnlock()

	for _, callback := range callbacks {
		l.safeNotifyCallback(callback, notification.Channel, notification.Extra)
	}
}

func (l *listener) safeNotifyCallback(callback sqldb.OnNotifyFunc, channel, payload string) {
	defer recoverAndLogListenerPanic("notify", channel)

	callback(channel, payload)
}

///////////////////////////////////////////////////////////////////////////////
// Channel subscription management
///////////////////////////////////////////////////////////////////////////////

func (l *listener) listenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	err = l.conn.Listen(channel)
	if err != nil && err != pq.ErrChannelAlreadyOpen {
		return fmt.Errorf("pqconn failed to listenOnChannel %q: %w", channel, err)
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
		return fmt.Errorf("pqconn unable to unlistenChannel %q: no db connection", channel)
	}

	err = l.conn.Unlisten(channel)
	if err != nil {
		return fmt.Errorf("pqconn failed to unlistenChannel %q: %w", channel, err)
	}

	l.callbacksMtx.Lock()
	unlistenCallbacks := l.unlistenCallbacks[channel]
	delete(l.notifyCallbacks, channel)
	delete(l.unlistenCallbacks, channel)
	l.callbacksMtx.Unlock()

	for _, callback := range unlistenCallbacks {
		l.safeUnlistenCallback(callback, channel)
	}

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

func (l *listener) close() {
	if l == nil {
		return
	}
	l.stopOnce.Do(func() {
		close(l.stop)

		globalListenersMtx.Lock()
		delete(globalListeners, l.connURL)
		globalListenersMtx.Unlock()

		l.ping.Stop()
		l.conn.Close() //#nosec G104 -- Don't care about close errors

		l.callbacksMtx.Lock()
		unlistenCallbacks := make(map[string][]sqldb.OnUnlistenFunc, len(l.unlistenCallbacks))
		for ch, cbs := range l.unlistenCallbacks {
			unlistenCallbacks[ch] = cbs
		}
		clear(l.notifyCallbacks)
		clear(l.unlistenCallbacks)
		l.callbacksMtx.Unlock()

		for channel, callbacks := range unlistenCallbacks {
			for _, callback := range callbacks {
				l.safeUnlistenCallback(callback, channel)
			}
		}
	})
}

func (l *listener) isStopped() bool {
	select {
	case <-l.stop:
		return true
	default:
		return false
	}
}

func (l *listener) safeUnlistenCallback(callback sqldb.OnUnlistenFunc, channel string) {
	defer recoverAndLogListenerPanic("unlisten", channel)

	callback(channel)
}
