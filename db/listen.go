package db

import (
	"context"
	"errors"

	"github.com/domonda/go-sqldb"
)

// ListenOnChannel will call onNotify for every channel notification
// and onUnlisten if the channel gets unlistened
// or the listener connection gets closed for some reason.
// It is valid to pass nil for onNotify or onUnlisten to not get those callbacks.
// Note that the callbacks are called in sequence from a single go routine,
// so callbacks should offload long running or potentially blocking code to other go routines.
// Panics from callbacks will be recovered and logged.
// Returns errors.ErrUnsupported if the connection does not implement sqldb.ListenerConnection.
func ListenOnChannel(ctx context.Context, channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) error {
	listener, ok := Conn(ctx).(sqldb.ListenerConnection)
	if !ok {
		return errors.ErrUnsupported
	}
	return listener.ListenOnChannel(channel, onNotify, onUnlisten)
}

// UnlistenChannel will stop listening on the channel.
// An error is returned, when the channel was not listened to
// or the listener connection is closed.
// Returns errors.ErrUnsupported if the connection does not implement sqldb.ListenerConnection.
func UnlistenChannel(ctx context.Context, channel string) error {
	listener, ok := Conn(ctx).(sqldb.ListenerConnection)
	if !ok {
		return errors.ErrUnsupported
	}
	return listener.UnlistenChannel(channel)
}

// IsListeningOnChannel returns if a channel is listened to.
// Returns false if the connection does not implement sqldb.ListenerConnection.
func IsListeningOnChannel(ctx context.Context, channel string) bool {
	listener, ok := Conn(ctx).(sqldb.ListenerConnection)
	if !ok {
		return false
	}
	return listener.IsListeningOnChannel(channel)
}
