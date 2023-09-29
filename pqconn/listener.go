package pqconn

import (
	"fmt"
	"sync"
	"time"

	"github.com/lib/pq"

	"github.com/domonda/go-sqldb"
)

var (
	// ListenerEventLogger will log all subscribed channel listener events if not nil
	ListenerEventLogger sqldb.Logger

	// Listener for all Postgres notifications
	Listener sqldb.Listener = listenerImpl{}
)

type listenerImpl struct{}

func (listenerImpl) ListenOnChannel(conn sqldb.Connection, channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) error {
	if conn.IsTransaction() {
		return sqldb.ErrWithinTransaction
	}
	return getOrCreateListener(conn.Config().ConnectURL()).listenOnChannel(channel, onNotify, onUnlisten)
}

func (listenerImpl) UnlistenChannel(conn sqldb.Connection, channel string) error {
	if conn.IsTransaction() {
		return sqldb.ErrWithinTransaction
	}
	return getListenerOrNil(conn.Config().ConnectURL()).unlistenChannel(channel)
}

func (listenerImpl) IsListeningOnChannel(conn sqldb.Connection, channel string) bool {
	if conn.IsTransaction() {
		return false
	}
	return getListenerOrNil(conn.Config().ConnectURL()).isListeningOnChannel(channel)
}

func (listenerImpl) Close(conn sqldb.Connection) error {
	if !conn.IsTransaction() {
		getListenerOrNil(conn.Config().ConnectURL()).close()
	}
	return nil
}

var (
	globalListeners    = make(map[string]*listener)
	globalListenersMtx sync.RWMutex

	ListenerMinReconnectInterval = time.Second * 10
	ListenerMaxReconnectInterval = time.Second * 60
	ListenerPingInterval         = time.Second * 90
)

type listener struct {
	connURL string
	conn    *pq.Listener
	ping    *time.Ticker

	callbacksMtx      sync.RWMutex
	notifyCallbacks   map[string][]sqldb.OnNotifyFunc
	unlistenCallbacks map[string][]sqldb.OnUnlistenFunc
}

func getOrCreateListener(connURL string) *listener {
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
			notifyCallbacks:   make(map[string][]sqldb.OnNotifyFunc),
			unlistenCallbacks: make(map[string][]sqldb.OnUnlistenFunc),
		}
		globalListeners[connURL] = l

		go l.listen()
	}

	return l
}

func getListenerOrNil(connURL string) *listener {
	globalListenersMtx.RLock()
	l := globalListeners[connURL]
	globalListenersMtx.RUnlock()

	return l
}

func (l *listener) listen() {
	for {
		select {
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

func recoverAndLogListenerPanic(operation, channel string) {
	p := recover()
	switch {
	case p == nil:
		return

	case ListenerEventLogger != nil:
		ListenerEventLogger.Printf("%s on channel %q paniced with: %+v", operation, channel, p)

	case sqldb.ErrLogger != nil:
		sqldb.ErrLogger.Printf("%s on channel %q paniced with: %+v", operation, channel, p)
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

	globalListenersMtx.Lock()
	defer globalListenersMtx.Unlock()

	delete(globalListeners, l.connURL)

	l.ping.Stop()
	l.conn.Close() //#nosec G104 -- Don't care about close errors
	l.conn = nil

	l.callbacksMtx.Lock()
	defer l.callbacksMtx.Unlock()

	for channel, callbacks := range l.unlistenCallbacks {
		for _, callback := range callbacks {
			l.safeUnlistenCallback(callback, channel)
		}
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
		return fmt.Errorf("sqlxconn can't listenOnChannel %q because: %w", channel, err)
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
	if l == nil || l.conn == nil {
		return fmt.Errorf("sqlxconn can't unlistenChannel %q because: no db connection", channel)
	}

	err = l.conn.Unlisten(channel)
	if err != nil {
		return fmt.Errorf("sqlxconn can't unlistenChannel %q because: %w", channel, err)
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
		sqldb.ErrLogger.Printf("sqlxconn: got listener connection event=%q error=%v", connectionEvent(event), err)

	case ListenerEventLogger != nil:
		ListenerEventLogger.Printf("sqlxconn: got listener connection event=%q", connectionEvent(event))
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
