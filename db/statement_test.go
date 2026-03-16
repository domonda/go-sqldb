package db

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestPrepare_Success(t *testing.T) {
	// given
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
	var gotQuery string
	mock.MockPrepare = func(ctx context.Context, query string) (sqldb.Stmt, error) {
		gotQuery = query
		return &sqldb.MockStmt{Prepared: query}, nil
	}
	ctx := testContext(t, mock)
	query := /*sql*/ `SELECT id FROM users WHERE id = $1`

	// when
	stmt, err := Prepare(ctx, query)

	// then
	require.NoError(t, err)
	require.NotNil(t, stmt)
	assert.Equal(t, query, gotQuery)
	assert.Equal(t, query, stmt.PreparedQuery())
	require.NoError(t, stmt.Close())
}

func TestPrepare_Error(t *testing.T) {
	// given
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
	prepErr := errors.New("prepare failed")
	mock.MockPrepare = func(ctx context.Context, query string) (sqldb.Stmt, error) {
		return nil, prepErr
	}
	ctx := testContext(t, mock)
	query := /*sql*/ `SELECT 1`

	// when
	stmt, err := Prepare(ctx, query)

	// then
	require.ErrorIs(t, err, prepErr)
	assert.Nil(t, stmt)
}

func TestStmtWithErrWrapping_Exec_Success(t *testing.T) {
	// given
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
	var execCount int
	var gotArgs []any
	mock.MockPrepare = func(ctx context.Context, query string) (sqldb.Stmt, error) {
		return &sqldb.MockStmt{
			Prepared: query,
			MockExec: func(ctx context.Context, args ...any) error {
				execCount++
				gotArgs = args
				return nil
			},
		}, nil
	}
	ctx := testContext(t, mock)
	query := /*sql*/ `DELETE FROM users WHERE id = $1`

	stmt, err := Prepare(ctx, query)
	require.NoError(t, err)
	defer stmt.Close()

	// when
	err = stmt.Exec(ctx, 42)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, execCount)
	assert.Equal(t, []any{42}, gotArgs)
}

func TestStmtWithErrWrapping_Exec_Error(t *testing.T) {
	// given
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
	execErr := errors.New("exec error")
	mock.MockPrepare = func(ctx context.Context, query string) (sqldb.Stmt, error) {
		return &sqldb.MockStmt{
			Prepared: query,
			MockExec: func(ctx context.Context, args ...any) error {
				return execErr
			},
		}, nil
	}
	ctx := testContext(t, mock)
	query := /*sql*/ `DELETE FROM users WHERE id = $1`

	stmt, err := Prepare(ctx, query)
	require.NoError(t, err)
	defer stmt.Close()

	// when
	err = stmt.Exec(ctx, 99)

	// then – error is wrapped with query context but still unwraps to the original
	require.ErrorIs(t, err, execErr)
}

func TestStmtWithErrWrapping_Query_Success(t *testing.T) {
	// given
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
	mock.MockPrepare = func(ctx context.Context, query string) (sqldb.Stmt, error) {
		return &sqldb.MockStmt{
			Prepared: query,
			MockQuery: func(ctx context.Context, args ...any) sqldb.Rows {
				return sqldb.NewMockRows("id").WithRow(int64(7))
			},
		}, nil
	}
	ctx := testContext(t, mock)
	query := /*sql*/ `SELECT id FROM users WHERE id = $1`

	stmt, err := Prepare(ctx, query)
	require.NoError(t, err)
	defer stmt.Close()

	// when
	rows := stmt.Query(ctx, 7)
	defer rows.Close()

	// then
	require.NoError(t, rows.Err())
	cols, err := rows.Columns()
	require.NoError(t, err)
	assert.Equal(t, []string{"id"}, cols)
	assert.True(t, rows.Next())
}

func TestStmtWithErrWrapping_Query_Error(t *testing.T) {
	// given
	mock := sqldb.NewMockConn(sqldb.NewQueryFormatter("$"))
	queryErr := errors.New("query error")
	mock.MockPrepare = func(ctx context.Context, query string) (sqldb.Stmt, error) {
		return &sqldb.MockStmt{
			Prepared: query,
			MockQuery: func(ctx context.Context, args ...any) sqldb.Rows {
				return sqldb.NewErrRows(queryErr)
			},
		}, nil
	}
	ctx := testContext(t, mock)
	query := /*sql*/ `SELECT id FROM users WHERE id = $1`

	stmt, err := Prepare(ctx, query)
	require.NoError(t, err)
	defer stmt.Close()

	// when
	rows := stmt.Query(ctx, 99)
	defer rows.Close()

	// then – error is wrapped with query context but still unwraps to the original
	require.ErrorIs(t, rows.Err(), queryErr)
}
