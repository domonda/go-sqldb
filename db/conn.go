package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// SetConn sets the global connection returned by Conn
// if there is no other connection in the context passed to Conn.
func SetConn(c sqldb.Connection) {
	if c == nil {
		panic("must not set nil sqldb.Connection")
	}
	globalConn = c
}

// Conn returns a non nil sqldb.Connection from ctx
// or the global connection set with SetConn.
func Conn(ctx context.Context) sqldb.Connection {
	return ConnOr(ctx, globalConn)
}

// ConnOr returns a non nil sqldb.Connection from ctx
// or the passed defaultConn.
func ConnOr(ctx context.Context, defaultConn sqldb.Connection) sqldb.Connection {
	c := ctx.Value(&globalConnCtxKey).(sqldb.Connection)
	if c == nil {
		return defaultConn
	}
	return c
}

// ContextWithConn returns a new context with the passed sqldb.Connection
// added as value so it can be retrieved again using Conn(ctx).
// Passing a nil connection causes Conn(ctx)
// to return the global connection set with SetConn.
func ContextWithConn(ctx context.Context, conn sqldb.Connection) context.Context {
	return context.WithValue(ctx, &globalConnCtxKey, conn)
}
