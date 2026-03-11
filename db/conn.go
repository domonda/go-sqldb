package db

import (
	"context"
	"sync"

	"github.com/domonda/go-sqldb"
)

type (
	connCtxKey            struct{}
	queryBuilderCtxKey    struct{}
	structReflectorCtxKey struct{}
)

var (
	globalConn    sqldb.Connection = sqldb.NewErrConn(sqldb.ErrNoDatabaseConnection)
	globalConnMtx sync.RWMutex

	globalQueryBuilder    sqldb.QueryBuilder = sqldb.StdQueryBuilder{}
	globalQueryBuilderMtx sync.RWMutex

	globalStructReflector    sqldb.StructReflector = sqldb.NewTaggedStructReflector()
	globalStructReflectorMtx sync.RWMutex
)

// SetConn sets the global connection that will be returned by [Conn]
// if there is no other connection in the context passed to [Conn].
// If the connection also implements [sqldb.QueryBuilder],
// it will be used by [QueryBuilder] when no context-level query builder is set.
func SetConn(c sqldb.Connection) {
	if c == nil {
		panic("unable to set nil Connection") // Prefer to panic early
	}
	globalConnMtx.Lock()
	globalConn = c
	globalConnMtx.Unlock()
}

// SetQueryBuilder sets the global [sqldb.QueryBuilder] that will be returned
// by [QueryBuilder] if there is no other query builder in the context.
func SetQueryBuilder(queryBuilder sqldb.QueryBuilder) {
	globalQueryBuilderMtx.Lock()
	globalQueryBuilder = queryBuilder
	globalQueryBuilderMtx.Unlock()
}

// SetStructReflector sets the global [sqldb.StructReflector] that will be returned
// by [StructReflector] if there is no other struct reflector in the context.
func SetStructReflector(structReflector sqldb.StructReflector) {
	globalStructReflectorMtx.Lock()
	globalStructReflector = structReflector
	globalStructReflectorMtx.Unlock()
}

// Conn returns the connection from the context
// or the global connection that was configured with [SetConn].
func Conn(ctx context.Context) sqldb.Connection {
	if c, _ := ctx.Value(connCtxKey{}).(sqldb.Connection); c != nil {
		return c
	}
	globalConnMtx.RLock()
	c := globalConn
	globalConnMtx.RUnlock()
	return c
}

// QueryBuilder returns the [sqldb.QueryBuilder] for the given context.
// It checks the following sources in order:
//  1. A query builder stored in the context via [ContextWithQueryBuilder].
//  2. The connection from [Conn] if it implements [sqldb.QueryBuilder].
//  3. The global query builder configured with [SetQueryBuilder].
func QueryBuilder(ctx context.Context) sqldb.QueryBuilder {
	if qb, _ := ctx.Value(queryBuilderCtxKey{}).(sqldb.QueryBuilder); qb != nil {
		return qb
	}
	if qb, _ := Conn(ctx).(sqldb.QueryBuilder); qb != nil {
		return qb
	}
	globalQueryBuilderMtx.RLock()
	qb := globalQueryBuilder
	globalQueryBuilderMtx.RUnlock()
	return qb
}

// StructReflector returns the [sqldb.StructReflector] from the context,
// or the global struct reflector that was configured with [SetStructReflector].
func StructReflector(ctx context.Context) sqldb.StructReflector {
	if sr, _ := ctx.Value(structReflectorCtxKey{}).(sqldb.StructReflector); sr != nil {
		return sr
	}
	globalStructReflectorMtx.RLock()
	sr := globalStructReflector
	globalStructReflectorMtx.RUnlock()
	return sr
}

// ContextWithConn returns a new context with the passed [Connection]
// added as value so it can be retrieved again using [Conn].
func ContextWithConn(ctx context.Context, conn sqldb.Connection) context.Context {
	return context.WithValue(ctx, connCtxKey{}, conn)
}

// ContextWithQueryBuilder returns a new context with the passed sqldb.QueryBuilder
// added as value so it can be retrieved again using [QueryBuilder].
func ContextWithQueryBuilder(ctx context.Context, queryBuilder sqldb.QueryBuilder) context.Context {
	return context.WithValue(ctx, queryBuilderCtxKey{}, queryBuilder)
}

// ContextWithStructReflector returns a new context with the passed sqldb.StructReflector
// added as value so it can be retrieved again using [StructReflector].
func ContextWithStructReflector(ctx context.Context, structReflector sqldb.StructReflector) context.Context {
	return context.WithValue(ctx, structReflectorCtxKey{}, structReflector)
}

// ContextWithGlobalConn returns a new context with the global connection
// added as value so it can be retrieved again using Conn(ctx).
func ContextWithGlobalConn(ctx context.Context) context.Context {
	globalConnMtx.RLock()
	c := globalConn
	globalConnMtx.RUnlock()
	return ContextWithConn(ctx, c)
}

// Close the global connection that was configured with [SetConn].
func Close() error {
	globalConnMtx.RLock()
	c := globalConn
	globalConnMtx.RUnlock()
	return c.Close()
}
