package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/domonda/go-sqldb"
)

// ListenOnChannel will call onNotify for every channel notification
// and onUnlisten if the channel gets unlistened
// or the listener connection gets closed for some reason.
// It is valid to pass nil for onNotify or onUnlisten to not get those callbacks.
// Calling ListenOnChannel multiple times for the same channel
// adds additional callbacks; all registered callbacks will be invoked
// for each notification.
// Note that the callbacks are called in sequence from a single go routine,
// so callbacks should offload long running or potentially blocking code to other go routines.
// Panics from callbacks will be recovered and logged.
// Returns errors.ErrUnsupported if the connection does not implement sqldb.ListenerConnection.
func ListenOnChannel(ctx context.Context, channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) error {
	listener, ok := Conn(ctx).(sqldb.ListenerConnection)
	if !ok {
		return fmt.Errorf("ListenOnChannel: %w", errors.ErrUnsupported)
	}
	return listener.ListenOnChannel(channel, onNotify, onUnlisten)
}

// UnlistenChannel will stop listening on the channel
// and remove all registered callbacks for it.
// An error is returned, when the channel was not listened to
// or the listener connection is closed.
// Returns errors.ErrUnsupported if the connection does not implement sqldb.ListenerConnection.
func UnlistenChannel(ctx context.Context, channel string) error {
	listener, ok := Conn(ctx).(sqldb.ListenerConnection)
	if !ok {
		return fmt.Errorf("UnlistenChannel: %w", errors.ErrUnsupported)
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
