package db

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func setupQueryCallbackCtx(conn *sqldb.MockConn) context.Context {
	config := sqldb.NewConnExt(conn, sqldb.NewTaggedStructReflector(), sqldb.StdQueryFormatter{}, sqldb.StdQueryBuilder{})
	return ContextWithConn(context.Background(), config)
}

func TestQueryCallback_ScalarSingle(t *testing.T) {
	query := /*sql*/ `SELECT name FROM users`
	conn := sqldb.NewMockConn("$", nil, nil).
		WithQueryResult(
			[]string{"name"},
			[][]driver.Value{{"Alice"}, {"Bob"}, {"Charlie"}},
			query,
		)
	ctx := setupQueryCallbackCtx(conn)

	var names []string
	err := QueryCallback(ctx,
		func(name string) { names = append(names, name) },
		query,
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"Alice", "Bob", "Charlie"}, names)
}

func TestQueryCallback_ScalarMultiColumn(t *testing.T) {
	query := /*sql*/ `SELECT name, age FROM users`
	conn := sqldb.NewMockConn("$", nil, nil).
		WithQueryResult(
			[]string{"name", "age"},
			[][]driver.Value{
				{"Alice", int64(30)},
				{"Bob", int64(25)},
			},
			query,
		)
	ctx := setupQueryCallbackCtx(conn)

	type entry struct {
		name string
		age  int64
	}
	var entries []entry
	err := QueryCallback(ctx,
		func(name string, age int64) {
			entries = append(entries, entry{name, age})
		},
		query,
	)
	require.NoError(t, err)
	assert.Equal(t, []entry{{"Alice", 30}, {"Bob", 25}}, entries)
}

func TestQueryCallback_WithQueryArgs(t *testing.T) {
	// Regression test: callback args count differs from SQL query args count.
	// The old bug used len(args) instead of typ.NumIn() to build callbackArgs.
	query := /*sql*/ `SELECT name, value FROM kv WHERE org_id = $1 AND type = $2`
	conn := sqldb.NewMockConn("$", nil, nil).
		WithQueryResult(
			[]string{"name", "value"},
			[][]driver.Value{
				{"greeting", "hello"},
				{"farewell", "bye"},
			},
			query,
			"org-1",  // $1
			"string", // $2
		)
	ctx := setupQueryCallbackCtx(conn)

	got := make(map[string]string)
	err := QueryCallback(ctx,
		func(name, value string) { got[name] = value },
		query,
		"org-1",  // $1
		"string", // $2
	)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"greeting": "hello", "farewell": "bye"}, got)
}

func TestQueryCallback_WithContext(t *testing.T) {
	query := /*sql*/ `SELECT id FROM items`
	conn := sqldb.NewMockConn("$", nil, nil).
		WithQueryResult(
			[]string{"id"},
			[][]driver.Value{{int64(1)}, {int64(2)}},
			query,
		)
	ctx := setupQueryCallbackCtx(conn)

	var ids []int64
	err := QueryCallback(ctx,
		func(ctx context.Context, id int64) {
			require.NotNil(t, ctx)
			ids = append(ids, id)
		},
		query,
	)
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2}, ids)
}

func TestQueryCallback_WithErrorReturn(t *testing.T) {
	query := /*sql*/ `SELECT name FROM users`
	conn := sqldb.NewMockConn("$", nil, nil).
		WithQueryResult(
			[]string{"name"},
			[][]driver.Value{{"Alice"}, {"STOP"}, {"Charlie"}},
			query,
		)
	ctx := setupQueryCallbackCtx(conn)

	stopErr := errors.New("stop iteration")
	var names []string
	err := QueryCallback(ctx,
		func(name string) error {
			if name == "STOP" {
				return stopErr
			}
			names = append(names, name)
			return nil
		},
		query,
	)
	require.ErrorIs(t, err, stopErr)
	assert.Equal(t, []string{"Alice"}, names, "should stop after error")
}

func TestQueryCallback_WithContextAndErrorReturn(t *testing.T) {
	query := /*sql*/ `SELECT value FROM items`
	conn := sqldb.NewMockConn("$", nil, nil).
		WithQueryResult(
			[]string{"value"},
			[][]driver.Value{{int64(10)}, {int64(20)}},
			query,
		)
	ctx := setupQueryCallbackCtx(conn)

	var sum int64
	err := QueryCallback(ctx,
		func(ctx context.Context, value int64) error {
			sum += value
			return nil
		},
		query,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(30), sum)
}

func TestQueryCallback_ZeroRows(t *testing.T) {
	query := /*sql*/ `SELECT name FROM users WHERE 1=0`
	conn := sqldb.NewMockConn("$", nil, nil).
		WithQueryResult(
			[]string{"name"},
			[][]driver.Value{},
			query,
		)
	ctx := setupQueryCallbackCtx(conn)

	called := false
	err := QueryCallback(ctx,
		func(name string) { called = true },
		query,
	)
	require.NoError(t, err)
	assert.False(t, called, "callback should not be called for zero rows")
}

func TestQueryCallback_InvalidNotFunc(t *testing.T) {
	query := /*sql*/ `SELECT 1`
	conn := sqldb.NewMockConn("$", nil, nil)
	ctx := setupQueryCallbackCtx(conn)

	err := QueryCallback(ctx, "not a function", query)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected callback function")
}

func TestQueryCallback_InvalidVariadic(t *testing.T) {
	query := /*sql*/ `SELECT 1`
	conn := sqldb.NewMockConn("$", nil, nil)
	ctx := setupQueryCallbackCtx(conn)

	err := QueryCallback(ctx, func(args ...string) {}, query)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "variadic")
}

func TestQueryCallback_InvalidNoArgs(t *testing.T) {
	query := /*sql*/ `SELECT 1`
	conn := sqldb.NewMockConn("$", nil, nil)
	ctx := setupQueryCallbackCtx(conn)

	err := QueryCallback(ctx, func() {}, query)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no arguments")
}

func TestQueryCallback_InvalidOnlyContext(t *testing.T) {
	query := /*sql*/ `SELECT 1`
	conn := sqldb.NewMockConn("$", nil, nil)
	ctx := setupQueryCallbackCtx(conn)

	err := QueryCallback(ctx, func(ctx context.Context) {}, query)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no arguments")
}

func TestQueryCallback_InvalidMultipleResults(t *testing.T) {
	query := /*sql*/ `SELECT 1`
	conn := sqldb.NewMockConn("$", nil, nil)
	ctx := setupQueryCallbackCtx(conn)

	err := QueryCallback(ctx, func(v int64) (int64, error) { return v, nil }, query)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "one result")
}

func TestQueryCallback_InvalidNonErrorResult(t *testing.T) {
	query := /*sql*/ `SELECT 1`
	conn := sqldb.NewMockConn("$", nil, nil)
	ctx := setupQueryCallbackCtx(conn)

	err := QueryCallback(ctx, func(v int64) string { return "" }, query)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "type error")
}

func TestQueryCallback_InvalidChanArg(t *testing.T) {
	query := /*sql*/ `SELECT 1`
	conn := sqldb.NewMockConn("$", nil, nil)
	ctx := setupQueryCallbackCtx(conn)

	err := QueryCallback(ctx, func(ch chan int) {}, query)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid argument type")
}

func TestQueryCallback_InvalidFuncArg(t *testing.T) {
	query := /*sql*/ `SELECT 1`
	conn := sqldb.NewMockConn("$", nil, nil)
	ctx := setupQueryCallbackCtx(conn)

	err := QueryCallback(ctx, func(fn func()) {}, query)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid argument type")
}

func TestQueryCallback_ColumnCountMismatch(t *testing.T) {
	query := /*sql*/ `SELECT name, age FROM users`
	conn := sqldb.NewMockConn("$", nil, nil).
		WithQueryResult(
			[]string{"name", "age"},
			[][]driver.Value{{"Alice", int64(30)}},
			query,
		)
	ctx := setupQueryCallbackCtx(conn)

	// Callback takes 1 arg but query returns 2 columns
	err := QueryCallback(ctx,
		func(name string) {},
		query,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "arguments but query result has")
}

func TestQueryCallback_ManyQueryArgsFewerCallbackArgs(t *testing.T) {
	// Specific regression test for the bug where len(args) (4 SQL args)
	// was used instead of typ.NumIn() (2 callback args) to build callbackArgs,
	// causing "reflect: Call using zero Value argument" panic.
	query := /*sql*/ `INSERT INTO kv (id, unstruct_id, name, value) VALUES ($1, $2, $3, $4) ON CONFLICT (unstruct_id, name) DO UPDATE SET value = EXCLUDED.value RETURNING name, value`
	conn := sqldb.NewMockConn("$", nil, nil).
		WithQueryResult(
			[]string{"name", "value"},
			[][]driver.Value{{"greeting", "hello"}},
			query,
			"id-1",       // $1
			"unstruct-1", // $2
			"greeting",   // $3
			"hello",      // $4
		)
	ctx := setupQueryCallbackCtx(conn)

	var gotName, gotValue string
	err := QueryCallback(ctx,
		func(name, value string) {
			gotName = name
			gotValue = value
		},
		query,
		"id-1",       // $1
		"unstruct-1", // $2
		"greeting",   // $3
		"hello",      // $4
	)
	require.NoError(t, err)
	assert.Equal(t, "greeting", gotName)
	assert.Equal(t, "hello", gotValue)
}
