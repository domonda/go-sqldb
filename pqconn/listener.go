package pqconn

import (
	"fmt"
	"sync"
	"time"

	"github.com/lib/pq"

	"github.com/domonda/go-sqldb"
)

var (
	globalListeners    = make(map[string]*listener)
	globalListenersMtx sync.RWMutex

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
	connURL  string
	conn     *pq.Listener
	ping     *time.Ticker
	stop     chan struct{}
	stopOnce sync.Once

	callbacksMtx      sync.RWMutex
	notifyCallbacks   map[string][]sqldb.OnNotifyFunc
	unlistenCallbacks map[string][]sqldb.OnUnlistenFunc
}

func (conn *connection) getOrCreateListener() *listener {
	connURL := conn.config.URL().String()

	globalListenersMtx.Lock()
	defer globalListenersMtx.Unlock()

	l := globalListeners[connURL]

	if l == nil {
		l = &listener{
			connURL: connURL,
			conn: pq.NewListener(
				connURL,
				ListenerMinReconnectInterval,
				ListenerMaxReconnectInterval,
				logListenerConnectionEvent,
			),
			ping:              time.NewTicker(ListenerPingInterval),
			stop:              make(chan struct{}),
			notifyCallbacks:   make(map[string][]sqldb.OnNotifyFunc),
			unlistenCallbacks: make(map[string][]sqldb.OnUnlistenFunc),
		}
		globalListeners[connURL] = l

		go l.listen()
	}

	return l
}

func (conn *connection) getListenerOrNil() *listener {
	connURL := conn.config.URL().String()

	globalListenersMtx.RLock()
	l := globalListeners[connURL]
	globalListenersMtx.RUnlock()

	return l
}

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
			err := l.conn.Ping()
			if err != nil {
				l.close()
				return
			}
		}
	}
}

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

func (l *listener) safeUnlistenCallback(callback sqldb.OnUnlistenFunc, channel string) {
	defer recoverAndLogListenerPanic("unlisten", channel)

	callback(channel)
}

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

func (l *listener) isListeningOnChannel(channel string) bool {
	if l == nil {
		return false
	}

	l.callbacksMtx.RLock()
	defer l.callbacksMtx.RUnlock()

	return len(l.notifyCallbacks[channel]) > 0 || len(l.unlistenCallbacks[channel]) > 0
}

func (l *listener) listenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	err = l.conn.Listen(channel)
	if err != nil {
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

// called on nil listener will return an error
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

func logListenerConnectionEvent(event pq.ListenerEventType, err error) {
	switch {
	case err != nil:
		sqldb.ErrLogger.Printf("pqconn: got listener connection event=%q error=%v", connectionEvent(event), err)

	case ListenerEventLogger != nil:
		ListenerEventLogger.Printf("pqconn: got listener connection event=%q", connectionEvent(event))
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
