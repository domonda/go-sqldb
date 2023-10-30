package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
)

var txCount atomic.Uint64

// NextTxNumber returns the next globally unique number
// for a new transaction in a threadsafe way.
//
// Use TxConnection.TxNumber() to get the number
// from a transaction connection.
func NextTxNumber() uint64 {
	return txCount.Add(1)
}

// ToTxConnection returns the passed Connection
// as TxConnection if implemented
// or else an ErrorConnection with an error
// that wraps errors.ErrUnsupported.
func ToTxConnection(conn Connection) TxConnection {
	tx, err := AsTxConnection(conn)
	if err != nil {
		return ErrorConnection(err)
	}
	return tx
}

func AsTxConnection(conn Connection) (TxConnection, error) {
	if tx, ok := conn.(TxConnection); ok {
		return tx, nil
	}
	return nil, fmt.Errorf("%w: %s does not implement TxConnection", errors.ErrUnsupported, conn)
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

// TransactionOpts executes txFunc within a database transaction with sql.TxOptions that is passed in to txFunc via the context.
// Use db.ContextConnection(ctx) to get the transaction connection within txFunc.
// TransactionOpts returns all errors from txFunc or transaction commit errors happening after txFunc.
// If parentConn is already a transaction, then it is passed through to txFunc unchanged as tx Connection
// and no parentConn.Begin, Commit, or Rollback calls will occour within this TransactionOpts call.
// Errors and panics from txFunc will rollback the transaction if parentConn was not already a transaction.
// Recovered panics are re-paniced and rollback errors after a panic are logged with ErrLogger.
func TransactionOpts(ctx context.Context, opts *sql.TxOptions, txFunc func(context.Context) error) (err error) {
	// Don't shadow err result var!
	txConnection, e := AsTxConnection(ContextConnection(ctx))
	if e != nil {
		return e
	}

	if parentOpts, isTransation := txConnection.TxOptions(); isTransation {
		// txConn is already a transaction connection
		// so don't begin a new transaction,
		// just execute txFunc within the current transaction
		// if the TxOptions are compatible
		err = CheckTxOptionsCompatibility(parentOpts, opts, txConnection.DefaultIsolationLevel())
		if err != nil {
			return err
		}
		return txFunc(ContextWithConnection(ctx, txConnection))
	}

	// Execute txFunc within new transaction
	txNumber := NextTxNumber()
	// Don't shadow err result var!
	tx, e := txConnection.Begin(ctx, opts, txNumber)
	if e != nil {
		return fmt.Errorf("Transaction %d Begin error: %w", txNumber, e)
	}

	defer func() {
		if r := recover(); r != nil {
			// txFunc paniced
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, log e so it doesn't get lost
				ErrLogger.Printf("Transaction %d error (%s) from rollback after panic: %+v", txNumber, e, r)
			}
			panic(r) // re-throw panic after Rollback
		}

		if err != nil {
			// txFunc returned an error
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, wrap err with e so it doesn't get lost
				err = fmt.Errorf("Transaction %d error (%s) from rollback after error: %w", txNumber, e, err)
			}
			return
		}

		e := tx.Commit()
		if e != nil {
			// Set Commit error as function return value
			err = fmt.Errorf("Transaction %d Commit error: %w", txNumber, e)
		}
	}()

	return txFunc(ContextWithConnection(ctx, tx))
}

func Transaction(ctx context.Context, txFunc func(context.Context) error) (err error) {
	return TransactionOpts(ctx, nil, txFunc)
}

// TransactionReadOnly executes txFunc within a read-only database transaction that is passed in to txFunc via the context.
// Use db.ContextConnection(ctx) to get the transaction connection within txFunc.
// TransactionReadOnly returns all errors from txFunc or transaction commit errors happening after txFunc.
// If parentConn is already a transaction, then it is passed through to txFunc unchanged as tx Connection
// and no parentConn.Begin, Commit, or Rollback calls will occour within this TransactionReadOnly call.
// Errors and panics from txFunc will rollback the transaction if parentConn was not already a transaction.
// Recovered panics are re-paniced and rollback errors after a panic are logged with ErrLogger.
func TransactionReadOnly(ctx context.Context, txFunc func(context.Context) error) error {
	return TransactionOpts(ctx, &sql.TxOptions{ReadOnly: true}, txFunc)
}

// DebugNoTransaction executes nonTxFunc without a database transaction.
// Useful to temporarely replace Transaction to debug the same code without using a transaction.
func DebugNoTransaction(ctx context.Context, nonTxFunc func(context.Context) error) error {
	return nonTxFunc(ctx)
}

// DebugNoTransactionOpts executes nonTxFunc without a database transaction.
// Useful to temporarely replace TransactionOpts to debug the same code without using a transaction.
func DebugNoTransactionOpts(ctx context.Context, opts *sql.TxOptions, nonTxFunc func(context.Context) error) error {
	return nonTxFunc(ctx)
}

// IsTransaction indicates if the connection from the context
// (or the global connection if the context has none)
// is a transaction.
func IsTransaction(ctx context.Context) bool {
	return ContextConnection(ctx).IsTransaction()
}

// ValidateWithinTransaction returns ErrNotWithinTransaction
// if the database connection from the context is not a transaction.
func ValidateWithinTransaction(ctx context.Context) error {
	conn := ContextConnection(ctx)
	if err := conn.Err(); err != nil {
		return err
	}
	if !conn.IsTransaction() {
		return ErrNotWithinTransaction
	}
	return nil
}

// ValidateNotWithinTransaction returns ErrWithinTransaction
// if the database connection from the context is a transaction.
func ValidateNotWithinTransaction(ctx context.Context) error {
	conn := ContextConnection(ctx)
	if err := conn.Err(); err != nil {
		return err
	}
	if conn.IsTransaction() {
		return ErrWithinTransaction
	}
	return nil
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
