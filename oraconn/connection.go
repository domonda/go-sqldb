package oraconn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	go_ora "github.com/sijms/go-ora/v2"

	"github.com/domonda/go-sqldb"
)

// Driver is the database/sql driver name used for Oracle connections.
const Driver = "oracle"

// Connect establishes a new [sqldb.Connection] using the passed config
// and github.com/sijms/go-ora/v2 as driver implementation.
// The connection is pinged with the passed context and only returned
// when there was no error from the ping.
//
// If lowercaseColumns is true, column names returned by [sqldb.Rows.Columns]
// are lowercased so they match the conventional lowercase Go struct tags.
// Oracle returns uppercase names for unquoted identifiers by default.
// Oracle SQL itself is case-insensitive for unquoted identifiers,
// so this only affects the Go-side column name matching.
func Connect(ctx context.Context, config *sqldb.ConnConfig, lowercaseColumns bool) (sqldb.Connection, error) {
	if config.Driver != Driver {
		return nil, fmt.Errorf(`invalid driver %q, expected %q`, config.Driver, Driver)
	}
	err := config.Validate()
	if err != nil {
		return nil, err
	}

	dsn := formatDSN(config)
	db, err := sql.Open(Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %w", err)
	}
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	err = db.PingContext(ctx)
	if err != nil {
		if e := db.Close(); e != nil {
			err = fmt.Errorf("%w, then %w", err, e)
		}
		return nil, err
	}
	return &connection{db: db, config: config, lowercaseColumns: lowercaseColumns}, nil
}

// formatDSN converts a sqldb.ConnConfig to an Oracle connection URL
// using the go-ora BuildUrl function.
func formatDSN(config *sqldb.ConnConfig) string {
	var options map[string]string
	if len(config.Extra) > 0 {
		options = config.Extra
	}
	return go_ora.BuildUrl(
		config.Host,
		int(config.Port),
		config.Database,
		config.User,
		config.Password,
		options,
	)
}

// MustConnect is like Connect but panics on error.
func MustConnect(ctx context.Context, config *sqldb.ConnConfig, lowercaseColumns bool) sqldb.Connection {
	conn, err := Connect(ctx, config, lowercaseColumns)
	if err != nil {
		panic(err)
	}
	return conn
}

type connection struct {
	QueryFormatter
	QueryBuilder

	db               *sql.DB
	config           *sqldb.ConnConfig
	lowercaseColumns bool
}

func (conn *connection) Config() *sqldb.ConnConfig {
	return conn.config
}

func (conn *connection) Ping(ctx context.Context, timeout time.Duration) error {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return conn.db.PingContext(ctx)
}

func (conn *connection) Stats() sql.DBStats {
	return conn.db.Stats()
}

func (conn *connection) Exec(ctx context.Context, query string, args ...any) error {
	_, err := conn.db.ExecContext(ctx, query, args...)
	if err != nil {
		return wrapKnownErrors(err)
	}
	return nil
}

func (conn *connection) ExecRowsAffected(ctx context.Context, query string, args ...any) (int64, error) {
	result, err := conn.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, wrapKnownErrors(err)
	}
	return result.RowsAffected()
}

func (conn *connection) Query(ctx context.Context, query string, args ...any) sqldb.Rows {
	r, err := conn.db.QueryContext(ctx, query, args...)
	if err != nil {
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}
	if conn.lowercaseColumns {
		return lowercaseRows{r}
	}
	return r
}

func (conn *connection) Prepare(ctx context.Context, query string) (sqldb.Stmt, error) {
	stmt, err := conn.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return sqldb.NewStmt(stmt, query), nil
}

func (*connection) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelReadCommitted // Oracle default
}

func (conn *connection) Transaction() sqldb.TransactionState {
	return sqldb.TransactionState{
		ID:   0,
		Opts: nil,
	}
}

func (conn *connection) Begin(ctx context.Context, id uint64, opts *sql.TxOptions) (sqldb.Connection, error) {
	if id == 0 {
		return nil, errors.New("transaction ID must not be zero")
	}
	tx, err := conn.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTransaction(conn, tx, opts, id), nil
}

func (conn *connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Close() error {
	return conn.db.Close()
}
