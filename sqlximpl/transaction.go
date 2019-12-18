package sqlximpl

import (
	"github.com/jmoiron/sqlx"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/implhelper"
)

type transaction struct {
	tx *sqlx.Tx
}

func TransactionConnection(tx *sqlx.Tx) sqldb.Connection {
	return transaction{tx}
}

func (conn transaction) Exec(query string, args ...interface{}) error {
	_, err := conn.tx.Exec(query, args...)
	return err
}

// Insert a new row into table using the named columValues.
func (conn transaction) Insert(table string, columValues sqldb.Values) error {
	return implhelper.Insert(conn, table, columValues)
}

// InsertReturning inserts a new row into table using columnValues
// and returns values from the inserted row listed in returning.
func (conn transaction) InsertReturning(table string, columnValues sqldb.Values, returning string) sqldb.RowScanner {
	return implhelper.InsertReturning(conn, table, columnValues, returning)
}

func (conn transaction) InsertStruct(table string, rowStruct interface{}, onlyColumns ...string) error {
	return implhelper.InsertStruct(conn, table, rowStruct, onlyColumns...)
}

func (conn transaction) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	row := conn.tx.QueryRowx(query, args...)
	if row.Err() != nil {
		return sqldb.NewErrRowScanner(row.Err())
	}
	return &rowScanner{query, row}
}

func (conn transaction) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.tx.Queryx(query, args...)
	if err != nil {
		return sqldb.NewErrRowsScanner(err)
	}
	return &rowsScanner{query, rows}
}

func (conn transaction) Begin() (sqldb.Connection, error) {
	return nil, sqldb.ErrWithinTransaction
}

func (conn transaction) Commit() error {
	return conn.tx.Commit()
}

func (conn transaction) Rollback() error {
	return conn.tx.Rollback()
}

func (conn transaction) Transaction(txFunc func(tx sqldb.Connection) error) error {
	return sqldb.ErrWithinTransaction
}

func (conn transaction) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return getOrCreateGlobalListener(conn.tx.DriverName()).listenOnChannel(channel, onNotify, onUnlisten)
}

func (conn transaction) UnlistenChannel(channel string) (err error) {
	return getGlobalListenerOrNil(conn.tx.DriverName()).unlistenChannel(channel)
}

func (conn transaction) IsListeningOnChannel(channel string) bool {
	return getGlobalListenerOrNil(conn.tx.DriverName()).isListeningOnChannel(channel)
}

func (conn transaction) Close() error {
	conn.Rollback()
	return nil
}
