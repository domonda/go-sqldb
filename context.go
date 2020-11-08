package sqldb

import "context"

type connKeyType struct{} // unique type for this package
var connKey connKeyType

// ConnectionFromContext returns a Connection
// from a context returned by ContextWithConnection
// or nil.
func ConnectionFromContext(ctx context.Context) Connection {
	conn, _ := ctx.Value(connKey).(Connection)
	return conn
}

// ContextWithConnection returns a new context with the passed Connection
// added as value so it can be retrieved again using ConnectionFromContext(ctx).
func ContextWithConnection(ctx context.Context, conn Connection) context.Context {
	return context.WithValue(ctx, connKey, conn)
}
