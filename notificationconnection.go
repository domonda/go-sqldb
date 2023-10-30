package sqldb

import "context"

type (
	// OnNotifyFunc is a callback type passed to Connection.ListenChannel
	OnNotifyFunc func(channel, payload string)

	// OnUnlistenFunc is a callback type passed to Connection.ListenChannel
	OnUnlistenFunc func(channel string)
)

type NotificationConnection interface {
	Connection

	// ListenChannel will call onNotify for every channel notification
	// and onUnlisten if the channel gets unlistened
	// or the listener connection gets closed for some reason.
	//
	// It is valid to call ListenChannel multiple times for the same channel
	// to register multiple callbacks.
	//
	// It is valid to pass nil for onNotify or onUnlisten to not get those callbacks.
	//
	// Note that the callbacks are called in sequence from a single go routine,
	// so callbacks should offload long running or potentially blocking code to other go routines.
	//
	// Panics from callbacks will be recovered and logged.
	ListenChannel(ctx context.Context, channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error

	// UnlistenChannel will stop listening on the channel.
	//
	// If the passed onNotify callback function is not nil,
	// then only this callback will be unsubscribed but other
	// callback might still be active.
	// If nil is passed for onNotify, then all callbacks
	// will be unsubscribed.
	//
	// An error is returned, when the channel was not listened to
	// or the listener connection is closed
	// or the passed onNotify callback was not subscribed with ListenChannel.
	UnlistenChannel(ctx context.Context, channel string, onNotify OnNotifyFunc) error

	// IsListeningChannel returns if a channel is listened to.
	IsListeningChannel(ctx context.Context, channel string) bool

	// ListeningChannels returns all listened channel names.
	ListeningChannels(ctx context.Context) ([]string, error)

	// NotifyChannel notifies a channel with the optional payload.
	// If the payload is an empty string, then it won't be added to the notification.
	NotifyChannel(ctx context.Context, channel, payload string) error
}
