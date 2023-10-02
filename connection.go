package sqldb

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"
)

type FullyFeaturedConnection interface {
	Connection
	TxConnection
	ListenerConnection
}

// Connection represents a database connection or transaction
type Connection interface {
	StructFieldMapper

	QueryFormatter

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
