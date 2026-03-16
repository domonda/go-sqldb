package sqldb

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scanTestStruct struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

// mockRowScanner implements rowScanner for testing.
type mockRowScanner struct {
	columns []string
	values  []any
	scanErr error
}

func (r *mockRowScanner) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	for i, d := range dest {
		switch v := d.(type) {
		case *int64:
			*v = r.values[i].(int64)
		case *string:
			*v = r.values[i].(string)
		}
	}
	return nil
}

func TestScanStruct(t *testing.T) {
	refl := NewTaggedStructReflector()

	t.Run("scan into struct", func(t *testing.T) {
		// given
		row := &mockRowScanner{
			columns: []string{"id", "name"},
			values:  []any{int64(42), "Alice"},
		}
		var dest scanTestStruct

		// when
		err := scanStruct(row, []string{"id", "name"}, refl, &dest)

		// then
		require.NoError(t, err)
		assert.Equal(t, int64(42), dest.ID)
		assert.Equal(t, "Alice", dest.Name)
	})

	t.Run("nil pointer destination returns error", func(t *testing.T) {
		// given
		row := &mockRowScanner{
			columns: []string{"id", "name"},
			values:  []any{int64(1), "Bob"},
		}
		var dest *scanTestStruct = nil

		// when
		err := scanStruct(row, []string{"id", "name"}, refl, dest)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil pointer")
	})

	t.Run("non-struct destination returns error", func(t *testing.T) {
		// given
		row := &mockRowScanner{
			columns: []string{"id"},
			values:  []any{int64(1)},
		}
		var dest int

		// when
		err := scanStruct(row, []string{"id"}, refl, &dest)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected struct")
	})

	t.Run("scan error propagates", func(t *testing.T) {
		// given
		row := &mockRowScanner{
			columns: []string{"id", "name"},
			values:  []any{int64(1), "Bob"},
			scanErr: sql.ErrNoRows,
		}
		var dest scanTestStruct

		// when
		err := scanStruct(row, []string{"id", "name"}, refl, &dest)

		// then
		require.Error(t, err)
	})

	t.Run("allocates nil pointer to struct", func(t *testing.T) {
		// given
		row := &mockRowScanner{
			columns: []string{"id", "name"},
			values:  []any{int64(7), "Charlie"},
		}
		// dest is a pointer to a nil pointer – scanStruct should allocate it
		var inner *scanTestStruct
		dest := &inner

		// when
		err := scanStruct(row, []string{"id", "name"}, refl, dest)

		// then
		require.NoError(t, err)
		require.NotNil(t, inner)
		assert.Equal(t, int64(7), inner.ID)
		assert.Equal(t, "Charlie", inner.Name)
	})
}
