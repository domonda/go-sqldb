package db_test

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
)

func TestReplaceErrNoRows(t *testing.T) {
	replacement := errors.New("not found")

	t.Run("nil error", func(t *testing.T) {
		err := db.ReplaceErrNoRows(nil, replacement)
		require.NoError(t, err)
	})

	t.Run("ErrNoRows replaced", func(t *testing.T) {
		err := db.ReplaceErrNoRows(sql.ErrNoRows, replacement)
		require.ErrorIs(t, err, replacement)
	})

	t.Run("wrapped ErrNoRows replaced", func(t *testing.T) {
		wrapped := errors.Join(errors.New("context"), sql.ErrNoRows)
		err := db.ReplaceErrNoRows(wrapped, replacement)
		require.ErrorIs(t, err, replacement)
	})

	t.Run("other error unchanged", func(t *testing.T) {
		otherErr := errors.New("some other error")
		err := db.ReplaceErrNoRows(otherErr, replacement)
		require.ErrorIs(t, err, otherErr)
	})
}

func TestIsOtherThanErrNoRows(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		require.False(t, db.IsOtherThanErrNoRows(nil))
	})

	t.Run("ErrNoRows", func(t *testing.T) {
		require.False(t, db.IsOtherThanErrNoRows(sql.ErrNoRows))
	})

	t.Run("other error", func(t *testing.T) {
		require.True(t, db.IsOtherThanErrNoRows(errors.New("connection lost")))
	})

	t.Run("wrapped ErrNoRows", func(t *testing.T) {
		wrapped := errors.Join(errors.New("context"), sql.ErrNoRows)
		require.False(t, db.IsOtherThanErrNoRows(wrapped))
	})
}

func TestUnwrapErrorWithQuery(t *testing.T) {
	original := errors.New("original error")
	formatter := sqldb.NewQueryFormatter("$")
	const query = /*sql*/ `INSERT INTO invoice (data) VALUES ($1)`

	t.Run("nil", func(t *testing.T) {
		require.NoError(t, db.UnwrapErrorWithQuery(nil))
	})

	t.Run("error without query unchanged", func(t *testing.T) {
		// Callers must be able to use the function unconditionally
		// without knowing whether an error carries a query.
		require.Same(t, original, db.UnwrapErrorWithQuery(original))
	})

	t.Run("error with query loses the query", func(t *testing.T) {
		wrapped := sqldb.WrapErrorWithQuery(original, query, []any{"secret invoice"}, formatter)
		require.Contains(t, wrapped.Error(), "secret invoice")

		require.Same(t, original, db.UnwrapErrorWithQuery(wrapped))
	})

	t.Run("query removed from within further error wrapping", func(t *testing.T) {
		// The message of an error wrapping the query wrapper already
		// contains the query, so the outer wrapping has to be discarded
		// together with it to guarantee the query is gone.
		wrapped := sqldb.WrapErrorWithQuery(original, query, []any{"secret invoice"}, formatter)
		outer := fmt.Errorf("storing invoice: %w", wrapped)
		require.Contains(t, outer.Error(), "secret invoice")

		unwrapped := db.UnwrapErrorWithQuery(outer)
		require.Same(t, original, unwrapped)
		require.NotContains(t, unwrapped.Error(), "secret invoice")
	})
}
