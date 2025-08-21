package db

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/domonda/go-sqldb"
)

// NonConnectionForTest returns a sqldb.Connection intended
// for unit tests that should run without an actual database connection.
//
// The transaction related methods simulate a transaction
// without any actual transaction handling.
//
// All methods execpt the following will cause the test to fail:
//   - Context()
//   - WithContext(context.Context)
//   - WithStructFieldMapper(sqldb.StructFieldMapper)
//   - Placeholder(paramIndex int)
//   - ValidateColumnName(name string)
//   - IsTransaction()
//   - TransactionNo()
//   - TransactionOptions()
//   - Begin(opts *sql.TxOptions, no uint64) (sqldb.Connection, error)
//   - Commit() error
//   - Rollback() error
//   - Transaction(opts *sql.TxOptions, txFunc func(tx sqldb.Connection) error) error
func NonConnectionForTest(t *testing.T) sqldb.Connection {
	return &nonConnForTest{t: t, ctx: context.TODO(), namer: sqldb.DefaultStructFieldMapping}
}

// ContextWithNonConnectionForTest returns a new context with a sqldb.Connection
// intended for unit tests that should run without an actual database connection.
//
// The transaction related methods of that connection
// simulate a transaction without any actual transaction handling.
//
// All methods of that connection execpt the following will cause the test to fail:
//   - Context()
//   - WithContext(context.Context)
//   - WithStructFieldMapper(sqldb.StructFieldMapper)
//   - Placeholder(paramIndex int)
//   - ValidateColumnName(name string)
//   - IsTransaction()
//   - TransactionNo()
//   - TransactionOptions()
//   - Begin(opts *sql.TxOptions, no uint64) (sqldb.Connection, error)
//   - Commit() error
//   - Rollback() error
//   - Transaction(opts *sql.TxOptions, txFunc func(tx sqldb.Connection) error) error
func ContextWithNonConnectionForTest(ctx context.Context, t *testing.T) context.Context {
	return ContextWithConn(ctx, NonConnectionForTest(t))
}

type nonConnForTest struct {
	t *testing.T

	txNo   uint64
	txOpts *sql.TxOptions

	// Deprecated
	ctx   context.Context
	namer sqldb.StructFieldMapper
}

func (e *nonConnForTest) Context() context.Context { return e.ctx }

func (e *nonConnForTest) WithContext(ctx context.Context) sqldb.Connection {
	return &nonConnForTest{t: e.t, txNo: e.txNo, txOpts: e.txOpts, ctx: ctx, namer: e.namer}
}

func (e *nonConnForTest) WithStructFieldMapper(namer sqldb.StructFieldMapper) sqldb.Connection {
	return &nonConnForTest{t: e.t, txNo: e.txNo, txOpts: e.txOpts, ctx: e.ctx, namer: namer}
}

func (e *nonConnForTest) StructFieldMapper() sqldb.StructFieldMapper {
	return e.namer
}

func (e *nonConnForTest) Ping(time.Duration) error {
	e.t.Fatal("Ping() called on TestWithoutDBConnection")
	return nil
}

func (e *nonConnForTest) Stats() sql.DBStats {
	e.t.Fatal("Stats() called on TestWithoutDBConnection")
	return sql.DBStats{}
}

func (e *nonConnForTest) Config() *sqldb.Config {
	e.t.Fatal("Config() called on TestWithoutDBConnection")
	return nil
}

func (e *nonConnForTest) Placeholder(paramIndex int) string {
	return fmt.Sprintf("$%d", paramIndex+1)
}

func (e *nonConnForTest) ValidateColumnName(name string) error {
	return nil
}

func (e *nonConnForTest) Exec(query string, args ...any) error {
	e.t.Fatal("Exec() called on TestWithoutDBConnection")
	return nil
}

func (e *nonConnForTest) Update(table string, values sqldb.Values, where string, args ...any) error {
	e.t.Fatal("Update() called on TestWithoutDBConnection")
	return nil
}

func (e *nonConnForTest) UpdateReturningRow(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowScanner {
	e.t.Fatal("UpdateReturningRow() called on TestWithoutDBConnection")
	return nil
}

func (e *nonConnForTest) UpdateReturningRows(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowsScanner {
	e.t.Fatal("UpdateReturningRows() called on TestWithoutDBConnection")
	return nil
}

func (e *nonConnForTest) UpdateStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	e.t.Fatal("UpdateStruct() called on TestWithoutDBConnection")
	return nil
}

func (e *nonConnForTest) UpsertStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	e.t.Fatal("UpsertStruct() called on TestWithoutDBConnection")
	return nil
}

func (e *nonConnForTest) QueryRow(query string, args ...any) sqldb.RowScanner {
	e.t.Fatal("QueryRow() called on TestWithoutDBConnection")
	return nil
}

func (e *nonConnForTest) QueryRows(query string, args ...any) sqldb.RowsScanner {
	e.t.Fatal("QueryRows() called on TestWithoutDBConnection")
	return nil
}

func (e *nonConnForTest) IsTransaction() bool {
	return e.txNo != 0
}

func (e *nonConnForTest) TransactionNo() uint64 {
	return e.txNo
}

func (e *nonConnForTest) TransactionOptions() (*sql.TxOptions, bool) {
	// Must return false, otherwise child transaction opening is calling Config() and it throws an error.
	return e.txOpts, false
}

func (e *nonConnForTest) Begin(opts *sql.TxOptions, no uint64) (sqldb.Connection, error) {
	return &nonConnForTest{t: e.t, txNo: no, txOpts: opts, ctx: e.ctx, namer: e.namer}, nil
}

func (e *nonConnForTest) Commit() error {
	return nil
}

func (e *nonConnForTest) Rollback() error {
	return nil
}

func (e *nonConnForTest) Transaction(opts *sql.TxOptions, txFunc func(tx sqldb.Connection) error) error {
	return txFunc(e)
}

func (e *nonConnForTest) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) error {
	e.t.Fatal("ListenOnChannel() called on TestWithoutDBConnection")
	return nil
}

func (e *nonConnForTest) UnlistenChannel(channel string) error {
	e.t.Fatal("UnlistenChannel() called on TestWithoutDBConnection")
	return nil
}

func (e *nonConnForTest) IsListeningOnChannel(channel string) bool {
	return false
}

func (e *nonConnForTest) Close() error {
	return nil
}
