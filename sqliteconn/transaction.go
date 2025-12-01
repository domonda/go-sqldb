package sqliteconn

import (
	"context"
	"database/sql"
	"time"

	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/domonda/go-sqldb"
)

type transaction struct {
	parent   *connection
	txOpts   *sql.TxOptions
	txID     uint64
	isNested bool // Whether this is a nested transaction (savepoint)
}

func (conn *transaction) Config() *sqldb.ConnConfig {
	return conn.parent.config
}

func (t *transaction) Ping(ctx context.Context, timeout time.Duration) error {
	return t.parent.Ping(ctx, timeout)
}

func (t *transaction) Stats() sql.DBStats {
	return t.parent.Stats()
}

func (t *transaction) Exec(ctx context.Context, query string, args ...any) error {
	err := sqlitex.Execute(t.parent.conn, query, &sqlitex.ExecOptions{
		Args: args,
	})
	if err != nil {
		return wrapKnownErrors(err)
	}
	return nil
}

func (t *transaction) Query(ctx context.Context, query string, args ...any) sqldb.Rows {
	stmt, err := t.parent.conn.Prepare(query)
	if err != nil {
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}

	// Bind arguments
	if err := bindArgs(stmt, args); err != nil {
		stmt.Finalize()
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}

	return &rows{
		stmt:               stmt,
		conn:               t.parent.conn,
		shouldFinalizeStmt: true, // Transaction Query() owns the statement
	}
}

func (t *transaction) Prepare(ctx context.Context, query string) (sqldb.Stmt, error) {
	stmt, err := t.parent.conn.Prepare(query)
	if err != nil {
		return nil, wrapKnownErrors(err)
	}

	return &statement{
		query: query,
		stmt:  stmt,
		conn:  t.parent.conn,
	}, nil
}

func (t *transaction) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelSerializable
}

func (t *transaction) Transaction() sqldb.TransactionState {
	return sqldb.TransactionState{
		ID:   t.txID,
		Opts: t.txOpts,
	}
}

func (t *transaction) Begin(ctx context.Context, id uint64, opts *sql.TxOptions) (sqldb.Connection, error) {
	// Nested transaction (savepoint)
	err := sqlitex.ExecuteTransient(t.parent.conn, `SAVEPOINT nested_tx`, nil)
	if err != nil {
		return nil, wrapKnownErrors(err)
	}

	return &transaction{
		parent:   t.parent,
		txOpts:   opts,
		txID:     id,
		isNested: true, // Mark as nested transaction
	}, nil
}

func (t *transaction) Commit() error {
	// For nested transactions, use RELEASE SAVEPOINT instead of COMMIT
	if t.isNested {
		return sqlitex.ExecuteTransient(t.parent.conn, `RELEASE SAVEPOINT nested_tx`, nil)
	}
	return sqlitex.ExecuteTransient(t.parent.conn, `COMMIT`, nil)
}

func (t *transaction) Rollback() error {
	// For nested transactions, use ROLLBACK TO SAVEPOINT instead of ROLLBACK
	if t.isNested {
		return sqlitex.ExecuteTransient(t.parent.conn, `ROLLBACK TO SAVEPOINT nested_tx`, nil)
	}
	return sqlitex.ExecuteTransient(t.parent.conn, `ROLLBACK`, nil)
}

func (t *transaction) Close() error {
	return t.Rollback()
}
