package db

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplaceErrNoRows(t *testing.T) {
	replacement := errors.New("not found")

	t.Run("nil error", func(t *testing.T) {
		err := ReplaceErrNoRows(nil, replacement)
		require.NoError(t, err)
	})

	t.Run("ErrNoRows replaced", func(t *testing.T) {
		err := ReplaceErrNoRows(sql.ErrNoRows, replacement)
		require.ErrorIs(t, err, replacement)
	})

	t.Run("wrapped ErrNoRows replaced", func(t *testing.T) {
		wrapped := errors.Join(errors.New("context"), sql.ErrNoRows)
		err := ReplaceErrNoRows(wrapped, replacement)
		require.ErrorIs(t, err, replacement)
	})

	t.Run("other error unchanged", func(t *testing.T) {
		otherErr := errors.New("some other error")
		err := ReplaceErrNoRows(otherErr, replacement)
		require.ErrorIs(t, err, otherErr)
	})
}

func TestIsOtherThanErrNoRows(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		require.False(t, IsOtherThanErrNoRows(nil))
	})

	t.Run("ErrNoRows", func(t *testing.T) {
		require.False(t, IsOtherThanErrNoRows(sql.ErrNoRows))
	})

	t.Run("other error", func(t *testing.T) {
		require.True(t, IsOtherThanErrNoRows(errors.New("connection lost")))
	})

	t.Run("wrapped ErrNoRows", func(t *testing.T) {
		wrapped := errors.Join(errors.New("context"), sql.ErrNoRows)
		require.False(t, IsOtherThanErrNoRows(wrapped))
	})
}
