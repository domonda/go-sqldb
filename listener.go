package sqldb

import "fmt"

type Listener interface {
	// ListenOnChannel will call onNotify for every channel notification
	// and onUnlisten if the channel gets unlistened
	// or the listener connection gets closed for some reason.
	// It is valid to pass nil for onNotify or onUnlisten to not get those callbacks.
	// Note that the callbacks are called in sequence from a single go routine,
	// so callbacks should offload long running or potentially blocking code to other go routines.
	// Panics from callbacks will be recovered and logged.
	ListenOnChannel(conn Connection, channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error

	// UnlistenChannel will stop listening on the channel.
	// An error is returned, when the channel was not listened to
	// or the listener connection is closed.
	UnlistenChannel(conn Connection, channel string) error

	// IsListeningOnChannel returns if a channel is listened to.
	IsListeningOnChannel(conn Connection, channel string) bool

	// Close the listener.
	Close(conn Connection) error
}

func UnsupportedListener() Listener {
	return noListener{}
}

type noListener struct{}

func (noListener) ListenOnChannel(conn Connection, channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	return fmt.Errorf("listening on channels %w", ErrNotSupported)
}

func (noListener) UnlistenChannel(conn Connection, channel string) error {
	return fmt.Errorf("listening on channels %w", ErrNotSupported)
}

func (noListener) IsListeningOnChannel(conn Connection, channel string) bool {
	return false
}

func (noListener) Close(conn Connection) error {
	return nil
}
