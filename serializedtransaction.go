package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
)

var (
	// Number of retries used for a SerializedTransaction
	// before it fails
	SerializedTransactionRetries = 10

	serializedTransactionCtxKey int

	savepointCount atomic.Uint64
)

// SerializedTransaction executes txFunc "serially" within a database transaction that is passed in to txFunc via the context.
// Use db.ContextConnection(ctx) to get the transaction connection within txFunc.
// Transaction returns all errors from txFunc or transaction commit errors happening after txFunc.
// If parentConn is already a transaction, then it is passed through to txFunc unchanged as tx Connection
// and no parentConn.Begin, Commit, or Rollback calls will occour within this Transaction call.
// Errors and panics from txFunc will rollback the transaction if parentConn was not already a transaction.
// Recovered panics are re-paniced and rollback errors after a panic are logged with ErrLogger.
//
// Serialized transactions are typically necessary when an insert depends on a previous select within
// the transaction, but that pre-insert select can't lock the table like it's possible with SELECT FOR UPDATE.
// During transaction execution, the isolation level "Serializable" is set. This does not mean
// that the transaction will be run in series. On the contrary, it actually means that Postgres will
// track read/write dependencies and will report an error in case other concurrent transactions
// have altered the results of the statements within this transaction. If no serialisation is possible,
// raw Postgres error will be:
// ```
// ERROR:  could not serialize access due to read/write dependencies among transactions
// HINT:   The transaction might succeed if retried.
// ```
// or
// ```
// ERROR:  could not serialize access due to concurrent update
// HINT:   The transaction might succeed if retried.
// ```
// In this case, retry the whole transaction (as Postgres hints). This works simply
// because if you run the transaction for the second (or Nth) time, the queries will
// yield different results therefore altering the end result.
//
// SerializedTransaction calls can be nested, in which case nested calls just execute the
// txFunc within the parent's serialized transaction.
// It's not valid to nest a SerializedTransaction within a normal Transaction function
// because in this case serialization retries can't be delegated up to the
// partent transaction that doesn't know anything about serialization.
//
// Because of the retryable nature, please be careful with the size of the transaction and the retry cost.
func SerializedTransaction(ctx context.Context, txFunc func(context.Context) error) error {
	// Pass nested serialized transactions through
	if ContextConnection(ctx).IsTransaction() {
		if ctx.Value(&serializedTransactionCtxKey) == nil {
			return errors.New("SerializedTransaction called from within a non-serialized transaction")
		}
		return txFunc(ctx)
	}

	// Add value to context to check for nested serialized transactions
	ctx = context.WithValue(ctx, &serializedTransactionCtxKey, struct{}{})

	opts := sql.TxOptions{Isolation: sql.LevelSerializable}
	for i := 0; i < SerializedTransactionRetries; i++ {
		err := TransactionOpts(ctx, &opts, txFunc)
		if err == nil || !strings.Contains(err.Error(), "could not serialize access") {
			return err // nil or err
		}
	}

	return errors.New("SerializedTransaction retried too many times")
}

// TransactionSavepoint executes txFunc within a database transaction or uses savepoints for rollback.
// If the passed context already has a database transaction connection,
// then a savepoint with a random name is created before the execution of txFunc.
// If txFunc returns an error, then the transaction is rolled back to the savepoint
// but the transaction from the context is not rolled back.
// If the passed context does not have a database transaction connection,
// then Transaction(ctx, txFunc) is called without savepoints.
// Use db.ContextConnection(ctx) to get the transaction connection within txFunc.
// TransactionSavepoint returns all errors from txFunc, transaction, savepoint, and rollback errors.
// Panics from txFunc are not recovered to rollback to the savepoint,
// they should behandled by the parent Transaction function.
func TransactionSavepoint(ctx context.Context, txFunc func(context.Context) error) error {
	conn := ContextConnection(ctx)
	if !conn.IsTransaction() {
		// If not already in a transaction, then execute txFunc
		// within a as transaction instead of using savepoints:
		return Transaction(ctx, txFunc)
	}

	savepoint := fmt.Sprintf("SP%d", savepointCount.Add(1))

	err := conn.Exec(ctx, "SAVEPOINT "+savepoint)
	if err != nil {
		return err
	}

	err = txFunc(ctx)
	if err != nil {
		e := conn.Exec(ctx, "ROLLBACK TO "+savepoint)
		if e != nil && !errors.Is(e, sql.ErrTxDone) {
			// Double error situation, wrap err with e so it doesn't get lost
			err = fmt.Errorf("TransactionSavepoint error (%s) from rollback after error: %w", e, err)
		}
		return err
	}

	return conn.Exec(ctx, "RELEASE SAVEPOINT "+savepoint)
}
