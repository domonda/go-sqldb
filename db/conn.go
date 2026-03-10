package db

import (
	"context"
	"sync"

	"github.com/domonda/go-sqldb"
)

type connCtxKey struct{}

var (
	globalConnMu sync.RWMutex
	globalConn   = sqldb.NewErrConnExt(sqldb.ErrNoDatabaseConnection)
)

// SetConn sets the global connection that will be returned by [Conn]
// if there is no other connection in the context passed to [Conn].
func SetConn(c sqldb.ConnExt) {
	if c == nil {
		panic("unable to set nil sqldb.ConnExt") // Prefer to panic early
	}
	globalConnMu.Lock()
	globalConn = c
	globalConnMu.Unlock()
}

// Conn returns the connection from the context
// or the global connection that was configured with [SetConn].
func Conn(ctx context.Context) sqldb.ConnExt {
	if c, _ := ctx.Value(connCtxKey{}).(sqldb.ConnExt); c != nil {
		return c
	}
	globalConnMu.RLock()
	c := globalConn
	globalConnMu.RUnlock()
	return c
}

// ContextWithConn returns a new context with the passed sqldb.ConnExt
// added as value so it can be retrieved again using [Conn].
func ContextWithConn(ctx context.Context, conn sqldb.ConnExt) context.Context {
	return context.WithValue(ctx, connCtxKey{}, conn)
}

// ContextWithGlobalConn returns a new context with the global connection
// added as value so it can be retrieved again using Conn(ctx).
func ContextWithGlobalConn(ctx context.Context) context.Context {
	globalConnMu.RLock()
	c := globalConn
	globalConnMu.RUnlock()
	return ContextWithConn(ctx, c)
}

// Close the global connection that was configured with [SetConn].
func Close() error {
	globalConnMu.RLock()
	c := globalConn
	globalConnMu.RUnlock()
	return c.Close()
}
