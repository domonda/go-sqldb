package sqldb

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/net/context"
)

var (
	globalConnection Connection = ErrorConnection{errors.New("sqldb not initialized, use sqldb.SetGlobalConnection!")}

	connectionCtxKey int
)

// GlobalConnection returns the global connection
// that will never be nil but an ErrorConnection
// if not initialized with SetGlobalConnection.
func GlobalConnection() Connection {
	return globalConnection
}

// SetGlobalConnection sets the global connection
// that will be returned by ContextConnection
// if the context has no other connection.
//
// This function is not thread-safe becaues the global connection
// is not expected to change between threads.
func SetGlobalConnection(conn Connection) {
	if conn == nil {
		panic("<nil> Connection")
	}
	globalConnection = conn
}

// ContextConnection returns the connection added
// to the context or the global connection
// if the context has no connection.
//
// See ContextWithConnection and SetGlobalConnection.
func ContextConnection(ctx context.Context) Connection {
	return ContextConnectionOr(ctx, globalConnection)
}

// ContextConnectionOr returns the connection added
// to the context or the passed defaultConn
// if the context has no connection.
//
// See ContextWithConnection and SetGlobalConnection.
func ContextConnectionOr(ctx context.Context, defaultConn Connection) Connection {
	if ctxConn, ok := ctx.Value(&connectionCtxKey).(Connection); ok {
		return ctxConn
	}
	return globalConnection
}

// ContextWithConnection returns a new context with the passed Connection
// added as value so it can be retrieved again using ContextConnection(ctx).
func ContextWithConnection(ctx context.Context, conn Connection) context.Context {
	if conn == nil {
		panic("<nil> Connection")
	}
	return context.WithValue(ctx, &connectionCtxKey, conn)
}

// IsTransaction indicates if the connection from the context
// (or the global connection if the context has none)
// is a transaction.
func IsTransaction(ctx context.Context) bool {
	return ContextConnection(ctx).IsTransaction()
}

type FullyFeaturedConnection interface {
	Connection
	TxConnection
	ListenerConnection
}

// Connection represents a database connection or transaction
type Connection interface {
	StructFieldMapper

	QueryFormatter

	ValidateColumnName(name string) error

	String() string
	DatabaseKind() string

	// Config returns the configuration used
	// to create this connection.
	Config() *Config

	// DBStats returns the sql.DBStats of this connection.
	DBStats() sql.DBStats

	// IsTransaction returns if the connection is a transaction
	IsTransaction() bool

	// Ping returns an error if the database
	// does not answer on this connection
	// with an optional timeout.
	// The passed timeout has to be greater zero
	// to be considered.
	Ping(ctx context.Context, timeout time.Duration) error

	// Exec executes a query with optional args.
	Exec(ctx context.Context, query string, args ...any) error

	Query(ctx context.Context, query string, args ...any) (Rows, error)

	// Close the connection.
	// Transactions will be rolled back.
	Close() error
}
