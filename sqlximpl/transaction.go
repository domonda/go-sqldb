package sqlximpl

import (
	"github.com/jmoiron/sqlx"

	sqldb "github.com/domonda/go-sqldb"
)

type transaction struct {
	tx *sqlx.Tx
}

func (conn transaction) Exec(query string, args ...interface{}) error {
	_, err := conn.tx.Exec(query, args...)
	return err
}

func (conn transaction) QueryRow(query string, args ...interface{}) sqldb.RowScanner {
	row := conn.tx.QueryRowx(query, args...)
	if row.Err() != nil {
		return sqldb.NewErrRowScanner(row.Err())
	}
	return rowScanner{row}
}

func (conn transaction) QueryRows(query string, args ...interface{}) sqldb.RowsScanner {
	rows, err := conn.tx.Queryx(query, args...)
	if err != nil {
		return sqldb.NewErrRowsScanner(err)
	}
	return &rowsScanner{rows}
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
