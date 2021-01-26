package sqldb

import (
	"database/sql"
	"errors"
	"fmt"
)

// Transaction executes txFunc within a database transaction that is passed in to txFunc as tx Connection.
// Transaction returns all errors from txFunc or transaction commit errors happening after txFunc.
// If parentConn is already a transaction, then it is passed through to txFunc unchanged as tx Connection
// and no parentConn.Begin, Commit, or Rollback calls will occour within this Transaction call.
// Errors and panics from txFunc will rollback the transaction if parentConn was not already a transaction.
// Recovered panics are re-paniced and rollback errors after a panic are logged with ErrLogger.
func Transaction(parentConn Connection, opts *sql.TxOptions, txFunc func(tx Connection) error) (err error) {
	if parentConn.IsTransaction() {
		return txFunc(parentConn)
	}
	return FreshTransaction(parentConn, opts, txFunc)
}

// FreshTransaction executes txFunc within a database transaction that is passed in to txFunc as tx Connection.
// FreshTransaction returns all errors from txFunc or transaction commit errors happening after txFunc.
// If parentConn is already a transaction, a fresh transaction will be created.
// Errors and panics from txFunc will rollback the transaction.
// Recovered panics are re-paniced and rollback errors after a panic are logged with ErrLogger.
func FreshTransaction(parentConn Connection, opts *sql.TxOptions, txFunc func(tx Connection) error) (err error) {
	tx, e := parentConn.Begin(opts)
	if e != nil {
		return fmt.Errorf("FreshTransaction Begin error: %w", e)
	}

	defer func() {
		if r := recover(); r != nil {
			// txFunc paniced
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, log e so it doesn't get lost
				ErrLogger.Printf("FreshTransaction error (%s) from rollback after panic: %+v", e, r)
			}
			panic(r) // re-throw panic after Rollback
		}

		if err != nil {
			// txFunc returned an error
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, wrap err with e so it doesn't get lost
				err = fmt.Errorf("FreshTransaction error (%s) from rollback after error: %w", e, err)
			}
			return
		}

		e := tx.Commit()
		if e != nil {
			// Set Commit error as function return value
			err = fmt.Errorf("FreshTransaction Commit error: %w", e)
		}
	}()

	return txFunc(tx)
}
