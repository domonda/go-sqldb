package mysqlconn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"net"
	"strconv"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/domonda/go-sqldb"
)

// Connect establishes a new [sqldb.Connection] using the passed config
// and github.com/go-sql-driver/mysql as driver implementation.
// The connection is pinged with the passed context and only returned
// when there was no error from the ping.
func Connect(ctx context.Context, config *sqldb.Config) (sqldb.Connection, error) {
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

// formatDSN converts a sqldb.Config to a MySQL DSN string
// using the go-sql-driver/mysql Config.FormatDSN method.
func formatDSN(config *sqldb.Config) string {
	mysqlCfg := mysqldriver.NewConfig()
	mysqlCfg.User = config.User
	mysqlCfg.Passwd = config.Password
	mysqlCfg.DBName = config.Database
	mysqlCfg.Net = "tcp"
	if config.Port != 0 {
		mysqlCfg.Addr = net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port)))
	} else {
		mysqlCfg.Addr = config.Host
	}
	if len(config.Extra) > 0 {
		if mysqlCfg.Params == nil {
			mysqlCfg.Params = make(map[string]string, len(config.Extra))
		}
		maps.Copy(mysqlCfg.Params, config.Extra)
	}
	return mysqlCfg.FormatDSN()
}

// MustConnect creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/go-sql-driver/mysql as driver implementation.
// The connection is pinged with the passed context and only returned
// when there was no error from the ping.
// Errors are panicked.
func MustConnect(ctx context.Context, config *sqldb.Config) sqldb.Connection {
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
	config *sqldb.Config
}

func (conn *connection) Config() *sqldb.Config {
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
	rows, err := conn.db.QueryContext(ctx, query, args...)
	if err != nil {
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}
	return rows
}

func (conn *connection) Prepare(ctx context.Context, query string) (sqldb.Stmt, error) {
	stmt, err := conn.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, wrapKnownErrors(err)
	}
	return sqldb.NewStmt(stmt, query, wrapKnownErrors), nil
}

func (*connection) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelRepeatableRead // MySQL default
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
		return nil, wrapKnownErrors(err)
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
