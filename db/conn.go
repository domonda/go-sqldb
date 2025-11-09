package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

var (
	globalConn = sqldb.ConnExt{
		Connection:      sqldb.NewErrConn(sqldb.ErrNoDatabaseConnection),
		StructReflector: sqldb.NewTaggedStructReflector(),
		QueryFormatter:  sqldb.StdQueryFormatter{},
		QueryBuilder:    sqldb.StdQueryBuilder{},
	}

	connCtxKey byte
)

// SetConn sets the global connection that will be returned by [Conn]
// if there is no other connection in the context passed to [Conn].
func SetConn(c *sqldb.ConnExt) {
	if c == nil {
		panic("can't set nil sqldb.ConnExt") // Prefer to panic early
	}
	globalConn = *c
}

// Conn returns the connection from the context
// or the global connection that was configured with [SetConn].
func Conn(ctx context.Context) *sqldb.ConnExt {
	if c, _ := ctx.Value(&connCtxKey).(*sqldb.ConnExt); c != nil {
		return c
	}
	return &globalConn
}

// ContextWithConn returns a new context with the passed sqldb.ConnExt
// added as value so it can be retrieved again using [Conn].
func ContextWithConn(ctx context.Context, conn *sqldb.ConnExt) context.Context {
	return context.WithValue(ctx, &connCtxKey, conn)
}

// ContextWithGlobalConn returns a new context with the global connection
// added as value so it can be retrieved again using Conn(ctx).
func ContextWithGlobalConn(ctx context.Context) context.Context {
	return ContextWithConn(ctx, &globalConn)
}

// Close the global connection that was configured with [SetConn].
func Close() error {
	return globalConn.Close()
}
