package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func NonConnForTest(t *testing.T) Connection {
	t.Helper()
	return &nonConnForTest{t: t}
}

type nonConnForTest struct {
	StdQueryFormatter

	t *testing.T

	txID   uint64
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

func (e *nonConnForTest) Exec(ctx context.Context, query string, args ...any) error {
	e.t.Fatal("Exec() called on non-working connection for test. That call should have been mocked!")
	return nil
}

func (e *nonConnForTest) Query(ctx context.Context, query string, args ...any) Rows {
	e.t.Fatal("Query() called on non-working connection for test. That call should have been mocked!")
	return nil
}

func (e *nonConnForTest) Prepare(ctx context.Context, query string) (Stmt, error) {
	e.t.Fatal("Prepare() called on non-working connection for test. That call should have been mocked!")
	return nil, nil
}

func (*nonConnForTest) DefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelDefault
}

func (e *nonConnForTest) Transaction() TransactionState {
	return TransactionState{
		ID:   e.txID,
		Opts: e.txOpts,
	}
}

func (e *nonConnForTest) Begin(ctx context.Context, id uint64, opts *sql.TxOptions) (Connection, error) {
	if id == 0 {
		return nil, errors.New("transaction ID must not be zero")
	}
	if e.txID != 0 {
		return nil, ErrWithinTransaction
	}
	return &nonConnForTest{t: e.t, txID: id, txOpts: opts}, nil
}

func (e *nonConnForTest) Commit() error {
	if e.txID == 0 {
		return ErrNotWithinTransaction
	}
	return nil
}

func (e *nonConnForTest) Rollback() error {
	if e.txID == 0 {
		return ErrNotWithinTransaction
	}
	return nil
}

func (e *nonConnForTest) Close() error {
	return nil
}
