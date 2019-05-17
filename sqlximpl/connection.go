package sqlximpl

import (
	"github.com/jmoiron/sqlx"

	sqldb "github.com/domonda/go-sqldb"
)

func NewConnection(driverName, dataSourceName string) (sqldb.Connection, error) {
	db, err := sqlx.Connect(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &connection{db}, nil
}

type connection struct {
	db *sqlx.DB
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
	return &transaction{tx}, nil
}

func (conn connection) Commit() error {
	return sqldb.ErrNotWithinTransaction
}

func (conn connection) Rollback() error {
	return sqldb.ErrNotWithinTransaction
}
