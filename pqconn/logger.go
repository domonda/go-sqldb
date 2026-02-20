package pqconn

import "github.com/domonda/go-sqldb"

// ListenerEventLogger will log all subscribed channel listener events if not nil
var ListenerEventLogger sqldb.Logger

func recoverAndLogListenerPanic(operation, channel string) {
	p := recover()
	switch {
	case p == nil:
		return

	case ListenerEventLogger != nil:
		ListenerEventLogger.Printf("%s on channel %q panicked with: %+v", operation, channel, p)

	case sqldb.ErrLogger != nil:
		sqldb.ErrLogger.Printf("%s on channel %q panicked with: %+v", operation, channel, p)
	}
}
