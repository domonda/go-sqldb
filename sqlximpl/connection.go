package sqlximpl

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/implhelper"
)

func NewConnection(driverName, dataSourceName string) (sqldb.Connection, error) {
	db, err := sqlx.Connect(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &Connection{db, driverName, dataSourceName}, nil
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

type Connection struct {
	db             *sqlx.DB
	driverName     string
	dataSourceName string
}

func (conn *Connection) Exec(query string, args ...interface{}) error {
	_, err := conn.db.Exec(query, args...)
	return err
}

// Insert a new row into table using the named columValues.
func (conn *Connection) Insert(table string, columValues map[string]interface{}) error {
	return implhelper.Insert(conn, table, columValues)
}

func (conn *Connection) InsertStruct(table string, rowStruct interface{}, onlyColumns ...string) error {
	return implhelper.InsertStruct(conn, table, rowStruct, onlyColumns...)
}

func (conn *Connection) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	row := conn.db.QueryRowx(query, args...)
	if row.Err() != nil {
		return sqldb.NewErrRowScanner(row.Err())
	}
	return rowScanner{row}
}

func (conn *Connection) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.db.Queryx(query, args...)
	if err != nil {
		return sqldb.NewErrRowsScanner(err)
	}
	return &rowsScanner{rows}
}

func (conn *Connection) Begin() (sqldb.Connection, error) {
	tx, err := conn.db.Beginx()
	if err != nil {
		return nil, err
	}
	return TransactionConnection(tx), nil
}

func (conn *Connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *Connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn *Connection) Transaction(txFunc func(tx sqldb.Connection) error) error {
	return implhelper.Transaction(conn, txFunc)
}

func (conn *Connection) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return getOrCreateGlobalListener(conn.dataSourceName).listenOnChannel(channel, onNotify, onUnlisten)
}

func (conn *Connection) UnlistenChannel(channel string) (err error) {
	return getGlobalListenerOrNil(conn.dataSourceName).unlistenChannel(channel)
}

func (conn *Connection) IsListeningOnChannel(channel string) bool {
	return getGlobalListenerOrNil(conn.dataSourceName).isListeningOnChannel(channel)
}

func (conn *Connection) Close() error {
	return conn.db.Close()
}
