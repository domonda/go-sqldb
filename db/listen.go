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
// Note that the callbacks are called in sequence from a single go routine,
// so callbacks should offload long running or potentially blocking code to other go routines.
// Panics from callbacks will be recovered and logged.
func ListenOnChannel(ctx context.Context, channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) error {
	conn, ok := Conn(ctx).Connection.(sqldb.ListenerConnection)
	if !ok {
		return fmt.Errorf("notifications %w", errors.ErrUnsupported)
	}
	return conn.ListenOnChannel(channel, onNotify, onUnlisten)
}

// UnlistenChannel will stop listening on the channel.
// An error is returned, when the channel was not listened to
// or the listener connection is closed.
func UnlistenChannel(ctx context.Context, channel string) error {
	conn, ok := Conn(ctx).Connection.(sqldb.ListenerConnection)
	if !ok {
		return fmt.Errorf("notifications %w", errors.ErrUnsupported)
	}
	return conn.UnlistenChannel(channel)
}

// IsListeningOnChannel returns if a channel is listened to.
func IsListeningOnChannel(ctx context.Context, channel string) bool {
	conn, ok := Conn(ctx).Connection.(sqldb.ListenerConnection)
	if !ok {
		return false
	}
	return conn.IsListeningOnChannel(channel)
}
