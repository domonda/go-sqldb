package sqlximpl

import (
	"fmt"
	"sync"
	"time"

	"github.com/lib/pq"

	"github.com/domonda/errors"
	"github.com/domonda/errors/wrap"
	sqldb "github.com/domonda/go-sqldb"
)

var (
	globalListeners    = make(map[string]*listener)
	globalListenersMtx sync.RWMutex

	listenerMinReconnectInterval = time.Second * 10
	listenerMaxReconnectInterval = time.Second * 60
	listenerPingInterval         = time.Second * 90
)

type listener struct {
	dataSourceName string
	conn           *pq.Listener
	ping           *time.Ticker

	callbacksMtx      sync.RWMutex
	notifyCallbacks   map[string][]sqldb.OnNotifyFunc
	unlistenCallbacks map[string][]sqldb.OnUnlistenFunc
}

func getOrCreateGlobalListener(dataSourceName string) *listener {
	globalListenersMtx.Lock()
	defer globalListenersMtx.Unlock()

	l := globalListeners[dataSourceName]

	if l == nil {
		l = &listener{
			dataSourceName: dataSourceName,
			conn: pq.NewListener(
				dataSourceName,
				listenerMinReconnectInterval,
				listenerMaxReconnectInterval,
				logListenerConnectionEvent,
			),
			ping:              time.NewTicker(listenerPingInterval),
			notifyCallbacks:   make(map[string][]sqldb.OnNotifyFunc),
			unlistenCallbacks: make(map[string][]sqldb.OnUnlistenFunc),
		}
		globalListeners[dataSourceName] = l

		go l.listen()
	}

	return l
}

func getGlobalListenerOrNil(dataSourceName string) *listener {
	globalListenersMtx.RLock()
	defer globalListenersMtx.RUnlock()

	return globalListeners[dataSourceName]
}

func (l *listener) listen() {
	for {
		select {
		case notification, isOpen := <-l.conn.Notify:
			if !isOpen {
				l.close()
				return
			}
			l.notify(notification)

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
	defer wrap.RecoverAndLogPanic(sqldb.ErrLogger, "safeNotifyCallback", channel, payload)

	callback(channel, payload)
}

func (l *listener) safeUnlistenCallback(callback sqldb.OnUnlistenFunc, channel string) {
	defer wrap.RecoverAndLogPanic(sqldb.ErrLogger, "safeUnlistenCallback", channel)

	callback(channel)
}

func (l *listener) close() {
	globalListenersMtx.Lock()
	defer globalListenersMtx.Unlock()

	delete(globalListeners, l.dataSourceName)

	l.ping.Stop()
	l.conn.Close()
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
	defer wrap.ResultError(&err, "listenOnChannel", channel)

	err = l.conn.Listen(channel)
	if err != nil {
		return err
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
	defer wrap.ResultError(&err, "unlistenChannel", channel)

	if l == nil || l.conn == nil {
		return errors.New("postgresdb: no listener")
	}

	err = l.conn.Unlisten(channel)
	if err != nil {
		return err
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
	var message string
	switch event {
	case pq.ListenerEventConnected:
		message = "connected"
	case pq.ListenerEventDisconnected:
		message = "disconnected"
	case pq.ListenerEventReconnected:
		message = "reconnected"
	case pq.ListenerEventConnectionAttemptFailed:
		message = "connectionAttemptFailed"
	default:
		message = fmt.Sprintf("unknown(%d)", event)
	}

	fmt.Printf("sqlximpl: got listener connection event message=%q error=%v\n", message, err)
}
