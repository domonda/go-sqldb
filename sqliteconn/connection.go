package sqliteconn

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/domonda/go-sqldb"
)

const Driver = "sqlite"

// Connect establishes a new sqldb.Connection using the passed sqldb.ConnConfig
// and zombiezen.com/go/sqlite as the underlying SQLite implementation.
func Connect(config *sqldb.ConnConfig) (sqldb.Connection, error) {
	if config.Driver != Driver {
		return nil, fmt.Errorf(`invalid driver %q, expected %q`, config.Driver, Driver)
	}

	// Build connection flags
	flags := sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenURI
	if config.ReadOnly {
		flags = sqlite.OpenReadOnly | sqlite.OpenURI
	}

	// Open the SQLite connection
	conn, err := sqlite.OpenConn(config.Database, flags)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite connection: %w", err)
	}

	// Enable foreign keys for SQLite (disabled by default)
	if err := sqlitex.ExecuteTransient(conn, `PRAGMA foreign_keys = ON`, nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Enable WAL mode for better concurrency (unless read-only)
	if !config.ReadOnly {
		if err := sqlitex.ExecuteTransient(conn, `PRAGMA journal_mode = WAL`, nil); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
		}
	}

	if config.ReadOnly {
		// Set connection to read-only mode
		if err := sqlitex.ExecuteTransient(conn, `PRAGMA query_only = ON`, nil); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to set read-only mode: %w", err)
		}
	}

	return &connection{
		conn:   conn,
		config: config,
	}, nil
}

// MustConnect creates a new sqldb.Connection using the passed sqldb.ConnConfig
// and zombiezen.com/go/sqlite as the underlying implementation.
// Errors are panicked.
func MustConnect(config *sqldb.ConnConfig) sqldb.Connection {
	conn, err := Connect(config)
	if err != nil {
		panic(err)
	}
	return conn
}

type connection struct {
	conn   *sqlite.Conn
	config *sqldb.ConnConfig
	txOpts *sql.TxOptions
	txID   uint64
}

func (c *connection) Ping(ctx context.Context, timeout time.Duration) error {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Simple query to verify connection is working
	return sqlitex.ExecuteTransient(c.conn, `SELECT 1`, nil)
}

func (c *connection) Stats() sql.DBStats {
	// zombiezen.com/go/sqlite doesn't provide pool statistics for single connections
	// Return minimal stats
	return sql.DBStats{
		OpenConnections: 1,
	}
}

func (c *connection) Exec(ctx context.Context, query string, args ...any) error {
	err := sqlitex.Execute(c.conn, query, &sqlitex.ExecOptions{
		Args: args,
	})
	if err != nil {
		return wrapKnownErrors(err)
	}
	return nil
}

func (c *connection) Query(ctx context.Context, query string, args ...any) sqldb.Rows {
	stmt, err := c.conn.Prepare(query)
	if err != nil {
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}

	// Bind arguments
	if err := bindArgs(stmt, args); err != nil {
		stmt.Finalize()
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}

	return &rows{
		stmt:               stmt,
		conn:               c.conn,
		shouldFinalizeStmt: true, // Query() owns the statement
	}
}

func (c *connection) Prepare(ctx context.Context, query string) (sqldb.Stmt, error) {
	stmt, err := c.conn.Prepare(query)
	if err != nil {
		return nil, wrapKnownErrors(err)
	}

	return &statement{
		query: query,
		stmt:  stmt,
		conn:  c.conn,
	}, nil
}

func (c *connection) Begin(ctx context.Context, id uint64, opts *sql.TxOptions) (sqldb.Connection, error) {
	// Start a transaction
	immediate := false
	if opts != nil && opts.Isolation >= sql.LevelReadCommitted {
		immediate = true
	}

	var err error
	if immediate {
		err = sqlitex.ExecuteTransient(c.conn, `BEGIN IMMEDIATE`, nil)
	} else {
		err = sqlitex.ExecuteTransient(c.conn, `BEGIN DEFERRED`, nil)
	}

	if err != nil {
		return nil, wrapKnownErrors(err)
	}

	return &transaction{
		parent: c,
		txOpts: opts,
		txID:   id,
	}, nil
}

func (c *connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (c *connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (c *connection) Transaction() sqldb.TransactionState {
	return sqldb.TransactionState{
		Opts: c.txOpts,
		ID:   c.txID,
	}
}

func (c *connection) Config() *sqldb.ConnConfig {
	return c.config
}

func (c *connection) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelSerializable
}

func (c *connection) Close() error {
	return c.conn.Close()
}

// bindArgs binds arguments to a prepared statement
func bindArgs(stmt *sqlite.Stmt, args []any) error {
	for i, arg := range args {
		pos := i + 1 // SQLite parameters are 1-indexed

		switch v := arg.(type) {
		case nil:
			stmt.BindNull(pos)
		case int:
			stmt.BindInt64(pos, int64(v))
		case int8:
			stmt.BindInt64(pos, int64(v))
		case int16:
			stmt.BindInt64(pos, int64(v))
		case int32:
			stmt.BindInt64(pos, int64(v))
		case int64:
			stmt.BindInt64(pos, v)
		case uint:
			stmt.BindInt64(pos, int64(v))
		case uint8:
			stmt.BindInt64(pos, int64(v))
		case uint16:
			stmt.BindInt64(pos, int64(v))
		case uint32:
			stmt.BindInt64(pos, int64(v))
		case uint64:
			stmt.BindInt64(pos, int64(v))
		case float32:
			stmt.BindFloat(pos, float64(v))
		case float64:
			stmt.BindFloat(pos, v)
		case bool:
			if v {
				stmt.BindInt64(pos, 1)
			} else {
				stmt.BindInt64(pos, 0)
			}
		case string:
			stmt.BindText(pos, v)
		case []byte:
			stmt.BindBytes(pos, v)
		default:
			return fmt.Errorf("unsupported argument type at position %d: %T", pos, arg)
		}
	}
	return nil
}
