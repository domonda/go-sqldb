package mssqlconn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	_ "github.com/microsoft/go-mssqldb" // register "sqlserver" database/sql driver

	"github.com/domonda/go-sqldb"
)

// Driver is the database/sql driver name used for SQL Server connections.
const Driver = "sqlserver"

// Connect establishes a new [sqldb.Connection] using the passed config
// and github.com/microsoft/go-mssqldb as driver implementation.
// The connection is pinged with the passed context and only returned
// when there was no error from the ping.
func Connect(ctx context.Context, config *sqldb.ConnConfig) (sqldb.Connection, error) {
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
	return &connection{db: db, config: config}, nil
}

// formatDSN converts a sqldb.ConnConfig to a SQL Server connection URL.
// The go-mssqldb driver expects the database as a query parameter,
// not in the URL path (which is used for instance names).
func formatDSN(config *sqldb.ConnConfig) string {
	query := make(url.Values)
	query.Set("database", config.Database)
	for key, val := range config.Extra {
		query.Set(key, val)
	}
	u := &url.URL{
		Scheme:   Driver,
		Host:     config.Host,
		RawQuery: query.Encode(),
	}
	if config.Port != 0 {
		u.Host = fmt.Sprintf("%s:%d", config.Host, config.Port)
	}
	if config.User != "" {
		u.User = url.UserPassword(config.User, config.Password)
	}
	return u.String()
}

// MustConnect is like Connect but panics on error.
func MustConnect(ctx context.Context, config *sqldb.ConnConfig) sqldb.Connection {
	conn, err := Connect(ctx, config)
	if err != nil {
		panic(err)
	}
	return conn
}

type connection struct {
	QueryFormatter
	QueryBuilder

	db     *sql.DB
	config *sqldb.ConnConfig
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

func (conn *connection) Query(ctx context.Context, query string, args ...any) sqldb.Rows {
	rows, err := conn.db.QueryContext(ctx, query, args...)
	if err != nil {
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}
	return rows
}

func (conn *connection) Prepare(ctx context.Context, query string) (sqldb.Stmt, error) {
	stmt, err := conn.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return sqldb.NewStmt(stmt, query), nil
}

func (*connection) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelReadCommitted // SQL Server default
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
