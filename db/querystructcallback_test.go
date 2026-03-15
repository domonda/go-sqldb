package db

import (
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestQueryStructCallback_Struct(t *testing.T) {
	type user struct {
		Name string `db:"name"`
		Age  int64  `db:"age"`
	}

	query := /*sql*/ `SELECT name, age FROM users`
	conn := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
		WithQueryResult(
			[]string{"name", "age"},
			[][]driver.Value{
				{"Alice", int64(30)},
				{"Bob", int64(25)},
			},
			query,
		)
	ctx := testContext(t, conn)

	var got []user
	err := QueryStructCallback(ctx,
		func(u user) error {
			got = append(got, u)
			return nil
		},
		query,
	)
	require.NoError(t, err)
	assert.Equal(t, []user{{"Alice", 30}, {"Bob", 25}}, got)
}

func TestQueryStructCallback_PointerToStruct(t *testing.T) {
	type user struct {
		Name string `db:"name"`
		Age  int64  `db:"age"`
	}

	query := /*sql*/ `SELECT name, age FROM users`
	conn := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
		WithQueryResult(
			[]string{"name", "age"},
			[][]driver.Value{
				{"Alice", int64(30)},
			},
			query,
		)
	ctx := testContext(t, conn)

	var got []*user
	err := QueryStructCallback(ctx,
		func(u *user) error {
			got = append(got, u)
			return nil
		},
		query,
	)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, &user{"Alice", 30}, got[0])
}

func TestQueryStructCallback_ErrorInterruptsIteration(t *testing.T) {
	type row struct {
		Name string `db:"name"`
	}

	query := /*sql*/ `SELECT name FROM users`
	conn := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
		WithQueryResult(
			[]string{"name"},
			[][]driver.Value{{"Alice"}, {"STOP"}, {"Charlie"}},
			query,
		)
	ctx := testContext(t, conn)

	stopErr := errors.New("stop iteration")
	var got []string
	err := QueryStructCallback(ctx,
		func(r row) error {
			if r.Name == "STOP" {
				return stopErr
			}
			got = append(got, r.Name)
			return nil
		},
		query,
	)
	require.ErrorIs(t, err, stopErr)
	assert.Equal(t, []string{"Alice"}, got, "should stop after error")
}

func TestQueryStructCallback_ZeroRows(t *testing.T) {
	type row struct {
		Name string `db:"name"`
	}

	query := /*sql*/ `SELECT name FROM users WHERE 1=0`
	conn := sqldb.NewMockConn(sqldb.NewQueryFormatter("$")).
		WithQueryResult(
			[]string{"name"},
			[][]driver.Value{},
			query,
		)
	ctx := testContext(t, conn)

	called := false
	err := QueryStructCallback(ctx,
		func(r row) error {
			called = true
			return nil
		},
		query,
	)
	require.NoError(t, err)
	assert.False(t, called, "callback should not be called for zero rows")
}

func TestQueryStructCallback_InvalidType(t *testing.T) {
	query := /*sql*/ `SELECT name FROM users`
	conn := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
	ctx := testContext(t, conn)

	err := QueryStructCallback(ctx,
		func(s string) error { return nil },
		query,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected struct or pointer to struct")
}
