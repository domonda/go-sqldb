package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/domonda/go-sqldb"
)

// ContextWithNonConnectionForTest returns a new context with a sqldb.Connection
// intended for unit tests that should work without an actual database connection
// by mocking any SQL related functionality so that the connection won't be used.
//
// The transaction related methods of that connection
// simulate a transaction without any actual transaction handling.
// All other methods except Close will cause the test to fail.
func ContextWithNonConnectionForTest(ctx context.Context, t *testing.T) context.Context {
	return ContextWithConn(ctx, &nonConnForTest{t: t})
}

type nonConnForTest struct {
	sqldb.StdQueryFormatter

	t *testing.T

	txNo   uint64
	txOpts *sql.TxOptions
}

func (e *nonConnForTest) Ping(ctx context.Context, timeout time.Duration) error {
	e.t.Fatalf("Ping() called on non-working connection for test. That call should have been mocked!")
	return nil
}

func (e *nonConnForTest) Stats() sql.DBStats {
	e.t.Fatal("Stats() called on non-working connection for test. That call should have been mocked!")
	return sql.DBStats{}
}

func (e *nonConnForTest) Config() *sqldb.Config {
	e.t.Fatal("Config() called on non-working connection for test. That call should have been mocked!")
	return nil
}

func (e *nonConnForTest) Exec(ctx context.Context, query string, args ...any) error {
	e.t.Fatal("Exec() called on non-working connection for test. That call should have been mocked!")
	return nil
}

func (e *nonConnForTest) Query(ctx context.Context, query string, args ...any) sqldb.Rows {
	e.t.Fatal("Query() called on non-working connection for test. That call should have been mocked!")
	return nil
}

func (e *nonConnForTest) Prepare(ctx context.Context, query string) (sqldb.Stmt, error) {
	e.t.Fatal("Prepare() called on non-working connection for test. That call should have been mocked!")
	return nil, nil
}

func (e *nonConnForTest) TransactionInfo() sqldb.TransactionInfo {
	return sqldb.TransactionInfo{
		No:                    e.txNo,
		Opts:                  e.txOpts,
		DefaultIsolationLevel: sql.LevelDefault,
	}
}

func (e *nonConnForTest) Begin(ctx context.Context, no uint64, opts *sql.TxOptions) (sqldb.Connection, error) {
	if e.txNo != 0 {
		return nil, sqldb.ErrWithinTransaction
	}
	return &nonConnForTest{t: e.t, txNo: no, txOpts: opts}, nil
}

func (e *nonConnForTest) Commit() error {
	if e.txNo == 0 {
		return sqldb.ErrNotWithinTransaction
	}
	return nil
}

func (e *nonConnForTest) Rollback() error {
	if e.txNo == 0 {
		return sqldb.ErrNotWithinTransaction
	}
	return nil
}

func (e *nonConnForTest) Close() error {
	return nil
}
