package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ConnExt combines a [Connection] with a [StructReflector], [QueryFormatter],
// and [QueryBuilder] to provide a complete interface for executing
// and building SQL queries against a database.
type ConnExt interface {
	Connection
	StructReflector
	QueryFormatter
	QueryBuilder
}

type connExtImpl struct {
	Connection
	StructReflector
	QueryFormatter
	QueryBuilder
}

// NewConnExt returns a new ConnExt combining the passed
// Connection, StructReflector, QueryFormatter, and QueryBuilder.
func NewConnExt(conn Connection, structReflector StructReflector, queryFormatter QueryFormatter, queryBuilder QueryBuilder) ConnExt {
	return &connExtImpl{
		Connection:      conn,
		StructReflector: structReflector,
		QueryFormatter:  queryFormatter,
		QueryBuilder:    queryBuilder,
	}
}

// NewConnExtWithConn returns a new ConnExt that uses conn as its [Connection]
// while reusing the [StructReflector], [QueryFormatter], and [QueryBuilder] from base.
// This is typically used to create a ConnExt for a transaction connection
// that shares the configuration of its parent ConnExt.
func NewConnExtWithConn(base ConnExt, conn Connection) ConnExt {
	return &connExtImpl{
		Connection:      conn,
		StructReflector: base,
		QueryFormatter:  base,
		QueryBuilder:    base,
	}
}

func (c *connExtImpl) ListenOnChannel(channel string, onNotify OnNotifyFunc, onUnlisten OnUnlistenFunc) error {
	if lc, ok := c.Connection.(ListenerConnection); ok {
		return lc.ListenOnChannel(channel, onNotify, onUnlisten)
	}
	return fmt.Errorf("notifications %w", errors.ErrUnsupported)
}

func (c *connExtImpl) UnlistenChannel(channel string) error {
	if lc, ok := c.Connection.(ListenerConnection); ok {
		return lc.UnlistenChannel(channel)
	}
	return fmt.Errorf("notifications %w", errors.ErrUnsupported)
}

func (c *connExtImpl) IsListeningOnChannel(channel string) bool {
	lc, ok := c.Connection.(ListenerConnection)
	return ok && lc.IsListeningOnChannel(channel)
}

// TransactionExt executes txFunc within a database transaction,
// wrapping the transaction Connection in a ConnExt with the same
// StructReflector, QueryFormatter, and QueryBuilder as connExt.
// See [Transaction] for transaction semantics.
func TransactionExt(ctx context.Context, conn ConnExt, opts *sql.TxOptions, txFunc func(ConnExt) error) error {
	return Transaction(ctx, conn, opts, func(tx Connection) error {
		return txFunc(NewConnExtWithConn(conn, tx))
	})
}

// TransactionResult is like [TransactionExt] but also returns
// a result value of generic type T from txFunc.
func TransactionResult[T any](ctx context.Context, conn ConnExt, opts *sql.TxOptions, txFunc func(ConnExt) (T, error)) (result T, err error) {
	err = TransactionExt(ctx, conn, opts, func(conn ConnExt) error {
		result, err = txFunc(conn)
		return err
	})
	return result, err
}
