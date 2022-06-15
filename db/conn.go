package db

import (
	"context"
	"fmt"
	"time"

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
// The returned connection will use the passed context.
// See sqldb.Connection.WithContext
func Conn(ctx context.Context) sqldb.Connection {
	return ConnDefault(ctx, globalConn)
}

// ConnDefault returns a non nil sqldb.Connection from ctx
// or the passed defaultConn.
// The returned connection will use the passed context.
// See sqldb.Connection.WithContext
func ConnDefault(ctx context.Context, defaultConn sqldb.Connection) sqldb.Connection {
	c, _ := ctx.Value(&globalConnCtxKey).(sqldb.Connection)
	if c == nil {
		c = defaultConn
	}
	if c.Context() == ctx {
		return c
	}
	return c.WithContext(ctx)
}

// ContextWithConn returns a new context with the passed sqldb.Connection
// added as value so it can be retrieved again using Conn(ctx).
// Passing a nil connection causes Conn(ctx)
// to return the global connection set with SetConn.
func ContextWithConn(ctx context.Context, conn sqldb.Connection) context.Context {
	return context.WithValue(ctx, &globalConnCtxKey, conn)
}

// ContextWithoutCancel returns a new context that inherits
// all values from parent, but not its cancellation state.
func ContextWithoutCancel(parent context.Context) context.Context {
	if _, ok := parent.(contextWithoutCancel); ok {
		return parent
	}
	return contextWithoutCancel{parent}
}

type contextWithoutCancel struct {
	parent context.Context
}

func (contextWithoutCancel) Deadline() (time.Time, bool) { return time.Time{}, false }
func (contextWithoutCancel) Done() <-chan struct{}       { return nil }
func (contextWithoutCancel) Err() error                  { return nil }
func (c contextWithoutCancel) Value(key any) any         { return c.parent.Value(key) }
func (c contextWithoutCancel) String() string            { return fmt.Sprintf("%s.WithoutCancel", c.parent) }
