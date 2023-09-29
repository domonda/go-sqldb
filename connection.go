package sqldb

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"
)

// Connection represents a database connection or transaction
type Connection interface {
	DBKind() DBKind

	// DBStats returns the sql.DBStats of this connection.
	DBStats() sql.DBStats

	// Config returns the configuration used
	// to create this connection.
	Config() *Config

	// IsTransaction returns if the connection is a transaction
	IsTransaction() bool

	// Ping returns an error if the database
	// does not answer on this connection
	// with an optional timeout.
	// The passed timeout has to be greater zero
	// to be considered.
	Ping(ctx context.Context, timeout time.Duration) error

	// Now returns the result of the SQL now()
	// function for the current connection.
	// Useful for getting the timestamp of a
	// SQL transaction for use in Go code.
	Now(ctx context.Context) (time.Time, error)

	// Exec executes a query with optional args.
	Exec(ctx context.Context, query string, args ...any) error

	// QueryRow queries a single row and returns a RowScanner for the results.
	QueryRow(ctx context.Context, query string, args ...any) RowScanner

	// QueryRows queries multiple rows and returns a RowsScanner for the results.
	QueryRows(ctx context.Context, query string, args ...any) RowsScanner

	// Close the connection.
	// Transactions will be rolled back.
	Close() error
}

type FullyFeaturedConnection interface {
	Connection
	TxConnection
	ListenerConnection
}

type DBKind interface {
	DatabaseKind() string

	DefaultIsolationLevel() sql.IsolationLevel

	// ValidateColumnName returns an error
	// if the passed name is not valid for a
	// column of the connection's database.
	ValidateColumnName(name string) error
}
