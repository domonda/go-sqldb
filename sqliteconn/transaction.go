package sqliteconn

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/domonda/go-sqldb"
)

type transaction struct {
	parent        *connection
	txOpts        *sql.TxOptions
	txID          uint64
	savepointName string // Non-empty for nested transactions (savepoints)
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
	if err := ctx.Err(); err != nil {
		return err
	}
	err := sqlitex.Execute(t.parent.conn, query, &sqlitex.ExecOptions{
		Args: args,
	})
	if err != nil {
		return wrapKnownErrors(err)
	}
	return nil
}

func (t *transaction) Query(ctx context.Context, query string, args ...any) sqldb.Rows {
	if err := ctx.Err(); err != nil {
		return sqldb.NewErrRows(err)
	}
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Nested transaction using a savepoint with a unique name derived from the transaction ID
	name := fmt.Sprintf("sp_%d", id)
	err := sqlitex.ExecuteTransient(t.parent.conn, `SAVEPOINT `+name, nil)
	if err != nil {
		return nil, wrapKnownErrors(err)
	}
	return &transaction{
		parent:        t.parent,
		txOpts:        opts,
		txID:          id,
		savepointName: name,
	}, nil
}

func (t *transaction) Commit() error {
	if t.savepointName != "" {
		return sqlitex.ExecuteTransient(t.parent.conn, `RELEASE SAVEPOINT `+t.savepointName, nil)
	}
	return sqlitex.ExecuteTransient(t.parent.conn, `COMMIT`, nil)
}

func (t *transaction) Rollback() error {
	if t.savepointName != "" {
		return sqlitex.ExecuteTransient(t.parent.conn, `ROLLBACK TO SAVEPOINT `+t.savepointName, nil)
	}
	return sqlitex.ExecuteTransient(t.parent.conn, `ROLLBACK`, nil)
}

func (t *transaction) Close() error {
	return t.Rollback()
}
