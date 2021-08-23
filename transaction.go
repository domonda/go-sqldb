package sqldb

import (
	"database/sql"
	"errors"
	"fmt"
)

// IsolatedTransaction executes txFunc within a database transaction that is passed in to txFunc as tx Connection.
// IsolatedTransaction returns all errors from txFunc or transaction commit errors happening after txFunc.
// If parentConn is already a transaction, a brand new transaction will begin on the parent's connection.
// Errors and panics from txFunc will rollback the transaction.
// Recovered panics are re-paniced and rollback errors after a panic are logged with ErrLogger.
func IsolatedTransaction(parentConn Connection, opts *sql.TxOptions, txFunc func(tx Connection) error) (err error) {
	tx, e := parentConn.Begin(opts)
	if e != nil {
		return fmt.Errorf("Transaction Begin error: %w", e)
	}

	defer func() {
		if r := recover(); r != nil {
			// txFunc paniced
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, log e so it doesn't get lost
				ErrLogger.Printf("Transaction error (%s) from rollback after panic: %+v", e, r)
			}
			panic(r) // re-throw panic after Rollback
		}

		if err != nil {
			// txFunc returned an error
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, wrap err with e so it doesn't get lost
				err = fmt.Errorf("Transaction error (%s) from rollback after error: %w", e, err)
			}
			return
		}

		e := tx.Commit()
		if e != nil {
			// Set Commit error as function return value
			err = fmt.Errorf("Transaction Commit error: %w", e)
		}
	}()

	return txFunc(tx)
}

// Transaction executes txFunc within a database transaction that is passed in to txFunc as tx Connection.
// Transaction returns all errors from txFunc or transaction commit errors happening after txFunc.
// If parentConn is already a transaction, then it is passed through to txFunc unchanged as tx Connection
// and no parentConn.Begin, Commit, or Rollback calls will occour within this Transaction call.
// An error is returned, if the requested transaction options passed via opts
// are stricter than the options of the parent transaction.
// Errors and panics from txFunc will rollback the transaction if parentConn was not already a transaction.
// Recovered panics are re-paniced and rollback errors after a panic are logged with ErrLogger.
func Transaction(parentConn Connection, opts *sql.TxOptions, txFunc func(tx Connection) error) (err error) {
	if parentOpts, parentIsTx := parentConn.TransactionOptions(); parentIsTx {
		err = CheckTxOptionsCompatibility(parentOpts, opts, parentConn.Config().DefaultIsolationLevel)
		if err != nil {
			return err
		}
		return txFunc(parentConn)
	}
	return IsolatedTransaction(parentConn, opts, txFunc)
}

// CheckTxOptionsCompatibility returns an error
// if the parent transaction options are less strict than the child options.
func CheckTxOptionsCompatibility(parent, child *sql.TxOptions, defaultIsolation sql.IsolationLevel) error {
	var (
		parentReadOnly  = false
		parentIsolation = defaultIsolation
		childReadOnly   = false
		childIsolation  = defaultIsolation
	)
	if parent != nil {
		parentReadOnly = parent.ReadOnly
		if parent.Isolation != sql.LevelDefault {
			parentIsolation = parent.Isolation
		}
	}
	if child != nil {
		childReadOnly = child.ReadOnly
		if child.Isolation != sql.LevelDefault {
			childIsolation = child.Isolation
		}
	}

	if parentReadOnly && !childReadOnly {
		return errors.New("parent transaction is read-only but child is not")
	}
	if parentIsolation < childIsolation {
		return fmt.Errorf("parent transaction isolation level '%s' is less strict child level '%s'", parentIsolation, childIsolation)
	}
	return nil
}
