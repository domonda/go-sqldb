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
	maxNumRowsCtxKey      struct{}
)

var (
	globalConn    sqldb.Connection = sqldb.NewErrConn(sqldb.ErrNoDatabaseConnection)
	globalConnMtx sync.RWMutex

	globalQueryBuilder    sqldb.QueryBuilder = sqldb.StdReturningQueryBuilder{}
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
	if queryBuilder == nil {
		panic("unable to set nil QueryBuilder")
	}
	globalQueryBuilderMtx.Lock()
	globalQueryBuilder = queryBuilder
	globalQueryBuilderMtx.Unlock()
}

// SetStructReflector sets the global [sqldb.StructReflector] that will be returned
// by [StructReflector] if there is no other struct reflector in the context.
func SetStructReflector(structReflector sqldb.StructReflector) {
	if structReflector == nil {
		panic("unable to set nil StructReflector")
	}
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
	if qb, ok := Conn(ctx).(sqldb.QueryBuilder); ok {
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

// UnlimitedMaxNumRows is the sentinel value for the maxNumRows argument
// of [QueryRowsAsSlice], [QueryRowsAsStrings] and [QueryRowsAsMapSlice]
// that disables the row cap. Any negative integer has the same effect,
// but using this named constant makes the intent explicit at call sites.
const UnlimitedMaxNumRows = sqldb.UnlimitedMaxNumRows

// ErrMaxNumRowsExceeded is returned by the multi-row query functions
// when the query would produce more rows than the cap set via
// [ContextWithMaxNumRows]. The rows scanned up to the cap are still
// returned alongside this error, so callers can consume the partial
// result after detecting the sentinel with [errors.As].
type ErrMaxNumRowsExceeded = sqldb.ErrMaxNumRowsExceeded

// ContextWithMaxNumRows returns a new context that caps the number of rows
// every multi-row slice-returning query function in this package will scan
// from its underlying driver rows iterator. The cap is read by
// [QueryRowsAsSlice], [QueryRowsAsStrings], and [QueryRowsAsMapSlice] (as
// well as anything else that calls [MaxNumRowsFromContext]), so a single
// call at the top of a request handler enforces the limit across every
// subsequent query on that context, including queries executed inside
// nested [Transaction] callbacks that inherit the context.
//
// The intended use is as a defensive safety net: set a generous but
// finite cap at a request or job boundary so that a buggy query or
// unexpectedly large result set cannot exhaust memory. Callers that
// need a specific business limit should still use SQL-level LIMIT
// clauses, which push the limit down to the database; the context cap
// is an application-side last line of defense.
//
// Semantics of maxNumRows:
//
//   - Negative values disable the cap. Use the [UnlimitedMaxNumRows]
//     constant for clarity at call sites. Any negative integer has the
//     same effect; the stored value is clamped to [UnlimitedMaxNumRows].
//   - A value of 0 is a deliberate hard cap that prevents any rows
//     from being returned at all. A non-empty query returns an empty
//     result together with [ErrMaxNumRowsExceeded]. An empty query
//     (zero rows from the driver) returns an empty result with no
//     error because the cap was never exceeded.
//   - A positive value N allows up to N rows to be scanned. If the
//     driver delivers an (N+1)th row, the query function stops,
//     returns the N rows already scanned, and wraps
//     [ErrMaxNumRowsExceeded] into the error chain. Use
//     [errors.As] to recognize the sentinel and decide whether to
//     consume the partial result, log, or fail the operation.
//
// The returned context value shadows any previous cap, so nested
// scopes can tighten (or loosen) the limit for a subtree of calls by
// wrapping the parent context again. Removing the cap entirely from a
// subtree requires wrapping with [UnlimitedMaxNumRows]; there is no
// "clear" operation because context values are immutable.
//
// Example:
//
//	// Apply a 1000-row safety net for the whole request.
//	ctx = db.ContextWithMaxNumRows(ctx, 1000)
//
//	users, err := db.QueryRowsAsSlice[User](ctx, `SELECT * FROM public.user`)
//	if err != nil {
//	    var capped db.ErrMaxNumRowsExceeded
//	    if errors.As(err, &capped) {
//	        // users contains the first 1000 rows, decide how to proceed
//	    }
//	    return err
//	}
//
// See also [MaxNumRowsFromContext] for reading the current cap.
func ContextWithMaxNumRows(ctx context.Context, maxNumRows int) context.Context {
	return context.WithValue(ctx, maxNumRowsCtxKey{}, max(maxNumRows, UnlimitedMaxNumRows))
}

// MaxNumRowsFromContext returns the row cap previously installed on ctx
// by [ContextWithMaxNumRows], or [UnlimitedMaxNumRows] if no cap
// is set. A return value of 0 means a hard cap of zero rows was set
// deliberately: the caller explicitly wants no rows to come back, and
// any non-empty query result will produce [ErrMaxNumRowsExceeded].
// A positive value N is the cap passed to the multi-row query
// functions; when the driver delivers more than N rows they return the
// first N rows together with a wrapped [ErrMaxNumRowsExceeded].
//
// This function is primarily used by the multi-row query wrappers in
// this package (see [QueryRowsAsSlice], [QueryRowsAsStrings],
// [QueryRowsAsMapSlice]) to plumb the context cap into the underlying
// sqldb calls. Application code rarely needs to read the cap directly.
func MaxNumRowsFromContext(ctx context.Context) int {
	if maxNumRows, ok := ctx.Value(maxNumRowsCtxKey{}).(int); ok {
		return maxNumRows
	}
	return UnlimitedMaxNumRows
}
