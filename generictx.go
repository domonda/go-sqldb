package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var _ Connection = &genericTx{}

type genericTx struct {
	// The parent non-transaction connection is needed
	// for Ping(), Stats(), Config(), and Begin().
	parent  *genericConn
	tx      *sql.Tx
	opts    *sql.TxOptions
	id      uint64
	wrapErr func(error) error // propagated from parent
}

func newGenericTx(parent *genericConn, tx *sql.Tx, opts *sql.TxOptions, id uint64) *genericTx {
	return &genericTx{
		parent:  parent,
		tx:      tx,
		opts:    opts,
		id:      id,
		wrapErr: parent.wrapErr,
	}
}

func (conn *genericTx) Config() *ConnConfig {
	return conn.parent.config
}

func (conn *genericTx) Ping(ctx context.Context, timeout time.Duration) error {
	return conn.parent.Ping(ctx, timeout)
}

func (conn *genericTx) Stats() sql.DBStats { return conn.parent.Stats() }

func (conn *genericTx) Exec(ctx context.Context, query string, args ...any) error {
	_, err := conn.tx.ExecContext(ctx, query, args...)
	if err != nil && conn.wrapErr != nil {
		return conn.wrapErr(err)
	}
	return err
}

func (conn *genericTx) Query(ctx context.Context, query string, args ...any) Rows {
	rows, err := conn.tx.QueryContext(ctx, query, args...)
	if err != nil {
		if conn.wrapErr != nil {
			err = conn.wrapErr(err)
		}
		return NewErrRows(err)
	}
	return rows
}

func (conn *genericTx) Prepare(ctx context.Context, query string) (Stmt, error) {
	stmt, err := conn.tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return NewStmt(stmt, query), nil
}

func (conn *genericTx) DefaultIsolationLevel() sql.IsolationLevel {
	return conn.parent.defaultIsolationLevel
}

func (conn *genericTx) Transaction() TransactionState {
	return TransactionState{
		ID:   conn.id,
		Opts: conn.opts,
	}
}

func (conn *genericTx) Begin(ctx context.Context, id uint64, opts *sql.TxOptions) (Connection, error) {
	if id == 0 {
		return nil, errors.New("transaction ID must not be zero")
	}
	tx, err := conn.parent.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newGenericTx(conn.parent, tx, opts, id), nil
}

func (conn *genericTx) Commit() error {
	return conn.tx.Commit()
}

func (conn *genericTx) Rollback() error {
	return conn.tx.Rollback()
}

func (conn *genericTx) FormatTableName(name string) (string, error) {
	return conn.parent.FormatTableName(name)
}

func (conn *genericTx) FormatColumnName(name string) (string, error) {
	return conn.parent.FormatColumnName(name)
}

func (conn *genericTx) FormatPlaceholder(paramIndex int) string {
	return conn.parent.FormatPlaceholder(paramIndex)
}

func (conn *genericTx) FormatStringLiteral(str string) string {
	return conn.parent.FormatStringLiteral(str)
}

func (conn *genericTx) MaxArgs() int {
	return conn.parent.MaxArgs()
}

func (conn *genericTx) Close() error {
	return conn.Rollback()
}

// genericTxWithQueryBuilder wraps a [genericTx] with a non-nil [QueryBuilder].
// Implements [QueryBuilder], [UpsertQueryBuilder], and [ReturningQueryBuilder]
// via delegation, mirroring [genericConnQueryBuilder].
type genericTxWithQueryBuilder struct {
	*genericTx
	QueryBuilder
}

func (conn *genericTxWithQueryBuilder) InsertUnique(formatter QueryFormatter, table string, columns []ColumnInfo, onConflict string) (string, error) {
	uqb, ok := conn.QueryBuilder.(UpsertQueryBuilder)
	if !ok {
		return "", fmt.Errorf("genericTxWithQueryBuilder: QueryBuilder %T does not implement UpsertQueryBuilder", conn.QueryBuilder)
	}
	return uqb.InsertUnique(formatter, table, columns, onConflict)
}

func (conn *genericTxWithQueryBuilder) Upsert(formatter QueryFormatter, table string, columns []ColumnInfo) (string, error) {
	uqb, ok := conn.QueryBuilder.(UpsertQueryBuilder)
	if !ok {
		return "", fmt.Errorf("genericTxWithQueryBuilder: QueryBuilder %T does not implement UpsertQueryBuilder", conn.QueryBuilder)
	}
	return uqb.Upsert(formatter, table, columns)
}

func (conn *genericTxWithQueryBuilder) InsertReturning(formatter QueryFormatter, table string, columns []ColumnInfo, returning string) (string, error) {
	rqb, ok := conn.QueryBuilder.(ReturningQueryBuilder)
	if !ok {
		return "", fmt.Errorf("genericTxWithQueryBuilder: QueryBuilder %T does not implement ReturningQueryBuilder", conn.QueryBuilder)
	}
	return rqb.InsertReturning(formatter, table, columns, returning)
}

func (conn *genericTxWithQueryBuilder) UpdateReturning(formatter QueryFormatter, table string, values Values, returning, where string, whereArgs []any) (string, []any, error) {
	rqb, ok := conn.QueryBuilder.(ReturningQueryBuilder)
	if !ok {
		return "", nil, fmt.Errorf("genericTxWithQueryBuilder: QueryBuilder %T does not implement ReturningQueryBuilder", conn.QueryBuilder)
	}
	return rqb.UpdateReturning(formatter, table, values, returning, where, whereArgs)
}

// Begin overrides [genericTx.Begin] to propagate the [QueryBuilder]
// to nested transactions.
func (conn *genericTxWithQueryBuilder) Begin(ctx context.Context, id uint64, opts *sql.TxOptions) (Connection, error) {
	if id == 0 {
		return nil, errors.New("transaction ID must not be zero")
	}
	tx, err := conn.genericTx.parent.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &genericTxWithQueryBuilder{
		genericTx:    newGenericTx(conn.genericTx.parent, tx, opts, id),
		QueryBuilder: conn.QueryBuilder,
	}, nil
}
