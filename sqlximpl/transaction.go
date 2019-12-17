package sqlximpl

import (
	"github.com/jmoiron/sqlx"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/implhelper"
)

type Transaction struct {
	tx *sqlx.Tx
}

func TransactionConnection(tx *sqlx.Tx) sqldb.Connection {
	return Transaction{tx}
}

func (conn Transaction) Exec(query string, args ...interface{}) error {
	_, err := conn.tx.Exec(query, args...)
	return err
}

// Insert a new row into table using the named columValues.
func (conn Transaction) Insert(table string, columValues map[string]interface{}) error {
	return implhelper.Insert(conn, table, columValues)
}

func (conn Transaction) InsertStruct(table string, rowStruct interface{}, onlyColumns ...string) error {
	return implhelper.InsertStruct(conn, table, rowStruct, onlyColumns...)
}

func (conn Transaction) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	row := conn.tx.QueryRowx(query, args...)
	if row.Err() != nil {
		return sqldb.NewErrRowScanner(row.Err())
	}
	return rowScanner{row}
}

func (conn Transaction) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.tx.Queryx(query, args...)
	if err != nil {
		return sqldb.NewErrRowsScanner(err)
	}
	return &rowsScanner{rows}
}

func (conn Transaction) Begin() (sqldb.Connection, error) {
	return nil, sqldb.ErrWithinTransaction
}

func (conn Transaction) Commit() error {
	return conn.tx.Commit()
}

func (conn Transaction) Rollback() error {
	return conn.tx.Rollback()
}

func (conn Transaction) Transaction(txFunc func(tx sqldb.Connection) error) error {
	return sqldb.ErrWithinTransaction
}

func (conn Transaction) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return getOrCreateGlobalListener(conn.tx.DriverName()).listenOnChannel(channel, onNotify, onUnlisten)
}

func (conn Transaction) UnlistenChannel(channel string) (err error) {
	return getGlobalListenerOrNil(conn.tx.DriverName()).unlistenChannel(channel)
}

func (conn Transaction) IsListeningOnChannel(channel string) bool {
	return getGlobalListenerOrNil(conn.tx.DriverName()).isListeningOnChannel(channel)
}

func (conn Transaction) Close() error {
	conn.Rollback()
	return nil
}
