package mockconn

/*
import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/domonda/go-sqldb"
)

var DefaultArgFmt = "?"

func New(ctx context.Context, queryWriter io.Writer, rowsProvider RowsProvider) sqldb.Connection {
	return &connection{
		ctx:              ctx,
		queryWriter:      queryWriter,
		listening:        newBoolMap(),
		rowsProvider:     rowsProvider,
		structFieldNamer: sqldb.DefaultStructFieldMapping,
		argFmt:           DefaultArgFmt,
	}
}

type connection struct {
	ctx              context.Context
	queryWriter      io.Writer
	listening        *boolMap
	rowsProvider     RowsProvider
	structFieldNamer sqldb.StructFieldMapper
	argFmt           string
}



func (conn *connection) Stats() sql.DBStats {
	return sql.DBStats{}
}

func (conn *connection) Config() *sqldb.Config {
	return &sqldb.Config{Driver: "mockconn", Host: "localhost", Database: "mock"}
}

func (conn *connection) Ping(time.Duration) error {
	return nil
}

func (conn *connection) Exec(query string, args ...any) error {
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, query)
	}
	return nil
}

func (conn *connection) Query(query string, args ...any) sqldb.Rows {
	if err := conn.ctx.Err(); err != nil {
		return sqldb.RowsErr(err)
	}
	return conn.rowsProvider.Query(conn.structFieldNamer, query, args...)
}

// func (conn *connection) QueryRow(query string, args ...any) sqldb.RowScanner {
// 	if conn.ctx.Err() != nil {
// 		return sqldb.RowScannerWithError(conn.ctx.Err())
// 	}
// 	if conn.queryWriter != nil {
// 		fmt.Fprint(conn.queryWriter, query)
// 	}
// 	if conn.rowsProvider == nil {
// 		return sqldb.RowScannerWithError(nil)
// 	}
// 	return conn.rowsProvider.QueryRow(conn.structFieldNamer, query, args...)
// }

// func (conn *connection) QueryRows(query string, args ...any) sqldb.RowsScanner {
// 	if conn.ctx.Err() != nil {
// 		return sqldb.RowsScannerWithError(conn.ctx.Err())
// 	}
// 	if conn.queryWriter != nil {
// 		fmt.Fprint(conn.queryWriter, query)
// 	}
// 	if conn.rowsProvider == nil {
// 		return sqldb.RowsScannerWithError(nil)
// 	}
// 	return conn.rowsProvider.QueryRows(conn.structFieldNamer, query, args...)
// }

func (conn *connection) Transaction() (no uint64, opts *sql.TxOptions) {
	return 0, nil
}

func (conn *connection) Begin(no uint64, opts *sql.TxOptions) (sqldb.Connection, error) {
	if no == 0 {
		return nil, errors.New("transaction number must not be zero")
	}
	if conn.queryWriter != nil {
		fmt.Fprint(conn.queryWriter, "BEGIN")
	}
	return transaction{conn, opts, no}, nil
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
*/
