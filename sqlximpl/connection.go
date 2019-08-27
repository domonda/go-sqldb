package sqlximpl

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	sqldb "github.com/domonda/go-sqldb"
)

func NewConnection(driverName, dataSourceName string) (sqldb.Connection, error) {
	db, err := sqlx.Connect(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &connection{db, driverName, dataSourceName}, nil
}

func MustNewConnection(driverName, dataSourceName string) sqldb.Connection {
	conn, err := NewConnection(driverName, dataSourceName)
	if err != nil {
		panic(err)
	}
	return conn
}

func NewPostgresConnection(user, dbname string) (sqldb.Connection, error) {
	driverName := "postgres"
	dataSourceName := fmt.Sprintf("user=%s dbname=%s sslmode=disable", user, dbname)
	return NewConnection(driverName, dataSourceName)
}

func MustNewPostgresConnection(user, dbname string) sqldb.Connection {
	conn, err := NewPostgresConnection(user, dbname)
	if err != nil {
		panic(err)
	}
	return conn
}

type connection struct {
	db             *sqlx.DB
	driverName     string
	dataSourceName string
}

func (conn connection) Exec(query string, args ...interface{}) error {
	_, err := conn.db.Exec(query, args...)
	return err
}

func (conn connection) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	row := conn.db.QueryRowx(query, args...)
	if row.Err() != nil {
		return sqldb.NewErrRowScanner(row.Err())
	}
	return rowScanner{row}
}

func (conn connection) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.db.Queryx(query, args...)
	if err != nil {
		return sqldb.NewErrRowsScanner(err)
	}
	return &rowsScanner{rows}
}

func (conn connection) Begin() (sqldb.Connection, error) {
	tx, err := conn.db.Beginx()
	if err != nil {
		return nil, err
	}
	return TransactionConnection(tx), nil
}

func (conn connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn connection) Transaction(txFunc func(tx sqldb.Connection) error) error {
	return sqldb.Transaction(conn, txFunc)
}

func (conn connection) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return getOrCreateGlobalListener(conn.dataSourceName).listenOnChannel(channel, onNotify, onUnlisten)
}

func (conn connection) UnlistenChannel(channel string) (err error) {
	return getGlobalListenerOrNil(conn.dataSourceName).unlistenChannel(channel)
}

func (conn connection) IsListeningOnChannel(channel string) bool {
	return getGlobalListenerOrNil(conn.dataSourceName).isListeningOnChannel(channel)
}

func (conn connection) Close() error {
	return conn.db.Close()
}
