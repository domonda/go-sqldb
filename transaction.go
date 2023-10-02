package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
)

var txCounter atomic.Uint64

// NextTxNumber returns the next globally unique number
// for a new transaction in a threadsafe way.
//
// Use TxConnection.TxNumber() to get the number
// from a transaction connection.
func NextTxNumber() uint64 {
	return txCounter.Add(1)
}

// TxConnection is a connection that supports transactions.
//
// This does not mean that every TxConnection represents
// a separate connection for an active transaction,
// only if it was returned for a new transaction by
// the Begin method.
type TxConnection interface {
	Connection

	DefaultIsolationLevel() sql.IsolationLevel

	// TxNumber returns the globally unique number of the transaction
	// or zero if the connection is not a transaction.
	// Implementations should use the package function NextTxNumber
	// to aquire a new number in a threadsafe way.
	TxNumber() uint64

	// TxOptions returns the sql.TxOptions of the
	// current transaction and true as second result value,
	// or false if the connection is not a transaction.
	TxOptions() (*sql.TxOptions, bool)

	// Begin a new transaction.
	// If the connection is already a transaction then a brand
	// new transaction will begin on the parent's connection.
	// The passed no will be returnd from the transaction's
	// Connection.TxNumber method.
	// Implementations should use the package function NextTxNumber
	// to aquire a new number in a threadsafe way.
	Begin(ctx context.Context, opts *sql.TxOptions, no uint64) (TxConnection, error)

	// Commit the current transaction.
	// Returns ErrNotWithinTransaction if the connection
	// is not within a transaction.
	Commit() error

	// Rollback the current transaction.
	// Returns ErrNotWithinTransaction if the connection
	// is not within a transaction.
	Rollback() error
}

// Transaction executes txFunc within a database transaction that is passed in to txFunc as tx Connection.
// Transaction returns all errors from txFunc or transaction commit errors happening after txFunc.
// If parentConn is already a transaction, then it is passed through to txFunc unchanged as tx Connection
// and no parentConn.Begin, Commit, or Rollback calls will occour within this Transaction call.
// An error is returned, if the requested transaction options passed via opts
// are stricter than the options of the parent transaction.
// Errors and panics from txFunc will rollback the transaction if parentConn was not already a transaction.
// Recovered panics are re-paniced and rollback errors after a panic are logged with ErrLogger.
func Transaction(ctx context.Context, parentConn TxConnection, opts *sql.TxOptions, txFunc func(tx TxConnection) error) (err error) {
	if parentOpts, parentIsTx := parentConn.TxOptions(); parentIsTx {
		// parentConn is already a transaction connection
		// so don't begin a new transaction,
		// just execute txFunc within the current transaction
		// if the TxOptions are compatible
		err = CheckTxOptionsCompatibility(parentOpts, opts, parentConn.DefaultIsolationLevel())
		if err != nil {
			return err
		}
		return txFunc(parentConn)
	}

	// Execute txFunc within new transaction
	return IsolatedTransaction(ctx, parentConn, opts, txFunc)
}

// IsolatedTransaction executes txFunc within a database transaction that is passed in to txFunc as tx Connection.
// IsolatedTransaction returns all errors from txFunc or transaction commit errors happening after txFunc.
// If parentConn is already a transaction, a brand new transaction will begin on the parent's connection.
// Errors and panics from txFunc will rollback the transaction.
// Recovered panics are re-paniced and rollback errors after a panic are logged with ErrLogger.
func IsolatedTransaction(ctx context.Context, parentConn TxConnection, opts *sql.TxOptions, txFunc func(tx TxConnection) error) (err error) {
	txNo := NextTxNumber()
	tx, e := parentConn.Begin(ctx, opts, txNo)
	if e != nil {
		return fmt.Errorf("Transaction %d Begin error: %w", txNo, e)
	}

	defer func() {
		if r := recover(); r != nil {
			// txFunc paniced
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, log e so it doesn't get lost
				ErrLogger.Printf("Transaction %d error (%s) from rollback after panic: %+v", txNo, e, r)
			}
			panic(r) // re-throw panic after Rollback
		}

		if err != nil {
			// txFunc returned an error
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, wrap err with e so it doesn't get lost
				err = fmt.Errorf("Transaction %d error (%s) from rollback after error: %w", txNo, e, err)
			}
			return
		}

		e := tx.Commit()
		if e != nil {
			// Set Commit error as function return value
			err = fmt.Errorf("Transaction %d Commit error: %w", txNo, e)
		}
	}()

	return txFunc(tx)
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
