package sqldb

import (
	"context"
	"database/sql"
)

// ConnExt combines a database connection
// with a struct reflector, query formatter and builder.
type ConnExt struct {
	Connection

	StructReflector StructReflector
	QueryFormatter  QueryFormatter
	QueryBuilder    QueryBuilder
}

func NewConnExt(conn Connection, structReflector StructReflector, queryFormatter QueryFormatter, queryBuilder QueryBuilder) *ConnExt {
	return &ConnExt{
		Connection:      conn,
		StructReflector: structReflector,
		QueryFormatter:  queryFormatter,
		QueryBuilder:    queryBuilder,
	}
}

func (c *ConnExt) WithConnection(conn Connection) *ConnExt {
	return &ConnExt{
		Connection:      conn,
		StructReflector: c.StructReflector,
		QueryFormatter:  c.QueryFormatter,
		QueryBuilder:    c.QueryBuilder,
	}
}

func TransactionExt(ctx context.Context, connExt *ConnExt, opts *sql.TxOptions, txFunc func(*ConnExt) error) error {
	return Transaction(ctx, connExt.Connection, opts, func(tx Connection) error {
		return txFunc(connExt.WithConnection(tx))
	})
}

func TransactionResult[T any](ctx context.Context, connExt *ConnExt, opts *sql.TxOptions, txFunc func(*ConnExt) (T, error)) (result T, err error) {
	err = TransactionExt(ctx, connExt, opts, func(connExt *ConnExt) error {
		result, err = txFunc(connExt)
		return err
	})
	return result, err
}
