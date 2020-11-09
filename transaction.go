package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type txKeyType struct{} // unique type for this package
var txKey txKeyType

func Transaction(ctx context.Context, txFunc func(ctx context.Context) error) (err error) {
	return TransactionOpts(ctx, nil, txFunc)
}

func TransactionReadonly(ctx context.Context, txFunc func(ctx context.Context) error) (err error) {
	return TransactionOpts(ctx, &sql.TxOptions{ReadOnly: true}, txFunc)
}

// Transaction executes txFunc within a database transaction that is passed in to txFunc as tx Connection.
// Transaction returns all errors from txFunc or transaction commit errors happening after txFunc.
// If parentConn is already a transaction, then it is passed through to txFunc unchanged as tx Connection
// and no parentConn.Begin, Commit, or Rollback calls will occour within this Transaction call.
// Errors and panics from txFunc will rollback the transaction if parentConn was not already a transaction.
// Recovered panics are re-paniced and rollback errors after a panic are logged with ErrLogger.
func TransactionOpts(ctx context.Context, opts *sql.TxOptions, txFunc func(ctx context.Context) error) (err error) {
	if ctx.Value(txKey) != nil {
		return txFunc(ctx)
	}

	conn, err := Conn(ctx)
	if err != nil {
		return err
	}

	tx, e := conn.Begin(ctx, opts)
	if e != nil {
		return fmt.Errorf("transaction Begin error: %w", e)
	}

	defer func() {
		if r := recover(); r != nil {
			// txFunc paniced
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, log e so it doesn't get lost
				ErrLogger.Printf("transaction error (%s) from rollback after panic: %+v", e, r)
			}
			panic(r) // re-throw panic after Rollback
		}

		if err != nil {
			// txFunc returned an error
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, wrap err with e so it doesn't get lost
				err = fmt.Errorf("transaction error (%s) from rollback after error: %w", e, err)
			}
			return
		}

		e := tx.Commit()
		if e != nil {
			// Set Commit error as function return value
			err = fmt.Errorf("transaction Commit error: %w", e)
		}
	}()

	return txFunc(context.WithValue(ctx, txKey, tx))
}
