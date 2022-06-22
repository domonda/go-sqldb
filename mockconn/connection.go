package mockconn

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/reflection"
)

var DefaultParamPlaceholderFormatter = sqldb.NewParamPlaceholderFormatter("$%d", 1)

func New(ctx context.Context, queryWriter io.Writer, rowsProvider RowsProvider) sqldb.Connection {
	return &connection{
		ctx:               ctx,
		queryWriter:       queryWriter,
		listening:         newBoolMap(),
		rowsProvider:      rowsProvider,
		structFieldMapper: sqldb.DefaultStructFieldMapping,
	}
}

type connection struct {
	ctx               context.Context
	queryWriter       io.Writer
	listening         *boolMap
	rowsProvider      RowsProvider
	structFieldMapper reflection.StructFieldMapper
}

func (conn *connection) Context() context.Context { return conn.ctx }

func (conn *connection) WithContext(ctx context.Context) sqldb.Connection {
	if ctx == conn.ctx {
		return conn
	}
	return &connection{
		ctx:               ctx,
		queryWriter:       conn.queryWriter,
		listening:         conn.listening,
		rowsProvider:      conn.rowsProvider,
		structFieldMapper: conn.structFieldMapper,
	}
}

func (conn *connection) WithStructFieldMapper(mapper reflection.StructFieldMapper) sqldb.Connection {
	return &connection{
		ctx:               conn.ctx,
		queryWriter:       conn.queryWriter,
		listening:         conn.listening,
		rowsProvider:      conn.rowsProvider,
		structFieldMapper: mapper,
	}
}

func (conn *connection) StructFieldMapper() reflection.StructFieldMapper {
	return conn.structFieldMapper
}

func (conn *connection) Stats() sql.DBStats {
	return sql.DBStats{}
}

func (conn *connection) Ping(time.Duration) error {
	return nil
}

func (conn *connection) Config() *sqldb.Config {
	return &sqldb.Config{Driver: "mockconn", Host: "localhost", Database: "mock"}
}

func (conn *connection) ValidateColumnName(name string) error {
	return validateColumnName(name)
}

func (*connection) ParamPlaceholder(index int) string {
	return fmt.Sprintf("$%d", index+1)
}

func (conn *connection) Err() error {
	return nil
}

func (conn *connection) Now() (time.Time, error) {
	return time.Now(), nil
}

func (conn *connection) Exec(query string, args ...any) error {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, query)
	}
	return nil
}

func (conn *connection) QueryRow(query string, args ...any) sqldb.RowScanner {
	if conn.ctx.Err() != nil {
		return sqldb.RowScannerWithError(conn.ctx.Err())
	}
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, query)
	}
	if conn.rowsProvider == nil {
		return sqldb.RowScannerWithError(nil)
	}
	return conn.rowsProvider.QueryRow(conn.structFieldMapper, query, args...)
}

func (conn *connection) QueryRows(query string, args ...any) sqldb.RowsScanner {
	if conn.ctx.Err() != nil {
		return sqldb.RowsScannerWithError(conn.ctx.Err())
	}
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, query)
	}
	if conn.rowsProvider == nil {
		return sqldb.RowsScannerWithError(nil)
	}
	return conn.rowsProvider.QueryRows(conn.structFieldMapper, query, args...)
}

func (conn *connection) IsTransaction() bool {
	return false
}

func (conn *connection) TransactionOptions() (*sql.TxOptions, bool) {
	return nil, false
}

func (conn *connection) Begin(opts *sql.TxOptions) (sqldb.Connection, error) {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, "BEGIN")
	}
	return transaction{conn, opts}, nil
}

func (conn *connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *connection) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	conn.listening.Set(channel, true)
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, "LISTEN "+channel)
	}
	return nil
}

func (conn *connection) UnlistenChannel(channel string) (err error) {
	conn.listening.Set(channel, false)
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, "UNLISTEN "+channel)
	}
	return nil
}

func (conn *connection) IsListeningOnChannel(channel string) bool {
	return conn.listening.Get(channel)
}

func (conn *connection) Close() error {
	return nil
}
