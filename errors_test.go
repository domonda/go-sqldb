package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceErrNoRows(t *testing.T) {
	replacement := errors.New("not found")

	t.Run("nil error returns nil", func(t *testing.T) {
		assert.Nil(t, ReplaceErrNoRows(nil, replacement))
	})

	t.Run("ErrNoRows returns replacement", func(t *testing.T) {
		assert.Equal(t, replacement, ReplaceErrNoRows(sql.ErrNoRows, replacement))
	})

	t.Run("wrapped ErrNoRows returns replacement", func(t *testing.T) {
		wrapped := fmt.Errorf("query failed: %w", sql.ErrNoRows)
		assert.Equal(t, replacement, ReplaceErrNoRows(wrapped, replacement))
	})

	t.Run("other error returned unchanged", func(t *testing.T) {
		other := errors.New("connection refused")
		assert.Equal(t, other, ReplaceErrNoRows(other, replacement))
	})

	t.Run("nil replacement for ErrNoRows", func(t *testing.T) {
		assert.Nil(t, ReplaceErrNoRows(sql.ErrNoRows, nil))
	})
}

func TestIsOtherThanErrNoRows(t *testing.T) {
	t.Run("nil returns false", func(t *testing.T) {
		assert.False(t, IsOtherThanErrNoRows(nil))
	})

	t.Run("ErrNoRows returns false", func(t *testing.T) {
		assert.False(t, IsOtherThanErrNoRows(sql.ErrNoRows))
	})

	t.Run("wrapped ErrNoRows returns false", func(t *testing.T) {
		assert.False(t, IsOtherThanErrNoRows(fmt.Errorf("wrap: %w", sql.ErrNoRows)))
	})

	t.Run("other error returns true", func(t *testing.T) {
		assert.True(t, IsOtherThanErrNoRows(errors.New("connection refused")))
	})
}

func TestErrQueryCanceled(t *testing.T) {
	t.Run("error message", func(t *testing.T) {
		assert.Equal(t, "query canceled", ErrQueryCanceled.Error())
	})

	t.Run("is context.Canceled", func(t *testing.T) {
		assert.True(t, errors.Is(ErrQueryCanceled, context.Canceled))
	})

	t.Run("is itself", func(t *testing.T) {
		assert.True(t, errors.Is(ErrQueryCanceled, ErrQueryCanceled))
	})

	t.Run("is not other error", func(t *testing.T) {
		assert.False(t, errors.Is(ErrQueryCanceled, sql.ErrNoRows))
	})
}

func TestSentinelErrors(t *testing.T) {
	assert.Equal(t, "no database connection", ErrNoDatabaseConnection.Error())
	assert.Equal(t, "within a transaction", ErrWithinTransaction.Error())
	assert.Equal(t, "not within a transaction", ErrNotWithinTransaction.Error())
	assert.Equal(t, "null value not allowed", ErrNullValueNotAllowed.Error())
	assert.Equal(t, "deadlock detected", ErrDeadlock.Error())
}

func TestErrRaisedException(t *testing.T) {
	err := ErrRaisedException{Message: "custom exception"}
	assert.Equal(t, "raised exception: custom exception", err.Error())
}

func TestConstraintViolationErrors(t *testing.T) {
	t.Run("ErrIntegrityConstraintViolation", func(t *testing.T) {
		assert.Equal(t, "integrity constraint violation", ErrIntegrityConstraintViolation{}.Error())
		assert.Equal(t, "integrity constraint violation of constraint: fk_user", ErrIntegrityConstraintViolation{Constraint: "fk_user"}.Error())
	})

	t.Run("ErrRestrictViolation", func(t *testing.T) {
		err := ErrRestrictViolation{Constraint: "restrict_x"}
		assert.Equal(t, "restrict violation of constraint: restrict_x", err.Error())
		assert.Equal(t, "restrict violation", ErrRestrictViolation{}.Error())

		var target ErrIntegrityConstraintViolation
		require.True(t, errors.As(err, &target))
		assert.Equal(t, "restrict_x", target.Constraint)
	})

	t.Run("ErrNotNullViolation", func(t *testing.T) {
		err := ErrNotNullViolation{Constraint: "nn_name"}
		assert.Equal(t, "not null violation of constraint: nn_name", err.Error())
		assert.Equal(t, "not null violation", ErrNotNullViolation{}.Error())

		var target ErrIntegrityConstraintViolation
		require.True(t, errors.As(err, &target))
		assert.Equal(t, "nn_name", target.Constraint)
	})

	t.Run("ErrForeignKeyViolation", func(t *testing.T) {
		err := ErrForeignKeyViolation{Constraint: "fk_order"}
		assert.Equal(t, "foreign key violation of constraint: fk_order", err.Error())
		assert.Equal(t, "foreign key violation", ErrForeignKeyViolation{}.Error())

		var target ErrIntegrityConstraintViolation
		require.True(t, errors.As(err, &target))
		assert.Equal(t, "fk_order", target.Constraint)
	})

	t.Run("ErrUniqueViolation", func(t *testing.T) {
		err := ErrUniqueViolation{Constraint: "uq_email"}
		assert.Equal(t, "unique violation of constraint: uq_email", err.Error())
		assert.Equal(t, "unique violation", ErrUniqueViolation{}.Error())

		var target ErrIntegrityConstraintViolation
		require.True(t, errors.As(err, &target))
		assert.Equal(t, "uq_email", target.Constraint)
	})

	t.Run("ErrCheckViolation", func(t *testing.T) {
		err := ErrCheckViolation{Constraint: "ck_positive"}
		assert.Equal(t, "check violation of constraint: ck_positive", err.Error())
		assert.Equal(t, "check violation", ErrCheckViolation{}.Error())

		var target ErrIntegrityConstraintViolation
		require.True(t, errors.As(err, &target))
		assert.Equal(t, "ck_positive", target.Constraint)
	})

	t.Run("ErrExclusionViolation", func(t *testing.T) {
		err := ErrExclusionViolation{Constraint: "excl_range"}
		assert.Equal(t, "exclusion violation of constraint: excl_range", err.Error())
		assert.Equal(t, "exclusion violation", ErrExclusionViolation{}.Error())

		var target ErrIntegrityConstraintViolation
		require.True(t, errors.As(err, &target))
		assert.Equal(t, "excl_range", target.Constraint)
	})
}

func TestWrapErrorWithQuery_AlreadyWrapped(t *testing.T) {
	// given
	original := errors.New("original error")
	formatter := StdQueryFormatter{PlaceholderPosPrefix: "$"}
	wrapped := WrapErrorWithQuery(original, "SELECT 1", nil, formatter)

	// when - wrapping an already wrapped error should return it unchanged
	doubleWrapped := WrapErrorWithQuery(wrapped, "SELECT 2", nil, formatter)

	// then
	assert.Equal(t, wrapped, doubleWrapped)
}

func TestWrapErrorWithQuery(t *testing.T) {
	type args struct {
		err      error
		query    string
		args     []any
		queryFmt QueryFormatter
	}
	tests := []struct {
		name      string
		args      args
		wantError string
	}{
		{name: "nil", args: args{err: nil}, wantError: ""},
		{
			name: "select no rows",
			args: args{
				err:      sql.ErrNoRows,
				query:    `SELECT * FROM table WHERE b = $2 AND a = $1`,
				queryFmt: StdQueryFormatter{PlaceholderPosPrefix: "$"},
				args:     []any{1, "2"},
			},
			wantError: fmt.Sprintf("%s from query: %s", sql.ErrNoRows, `SELECT * FROM table WHERE b = '2' AND a = 1`),
		},
		{
			name: "multi line",
			args: args{
				err: sql.ErrNoRows,
				query: `
					SELECT *
					FROM table
					WHERE b = $2
						AND a = $1`,
				queryFmt: StdQueryFormatter{PlaceholderPosPrefix: "$"},
				args:     []any{1, "2"},
			},
			wantError: fmt.Sprintf(
				"%s from query: %s",
				sql.ErrNoRows,
				`SELECT *
FROM table
WHERE b = '2'
	AND a = 1`,
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WrapErrorWithQuery(tt.args.err, tt.args.query, tt.args.args, tt.args.queryFmt)
			if tt.wantError == "" && err != nil || tt.wantError != "" && (err == nil || err.Error() != tt.wantError) {
				t.Errorf("WrapErrorWithQuery() error = \n%s\nwantErr\n%s", err, tt.wantError)
			}
			if !errors.Is(err, tt.args.err) {
				t.Errorf("WrapErrorWithQuery() error = %v does not wrap %v", err, tt.args.err)
			}
		})
	}
}
