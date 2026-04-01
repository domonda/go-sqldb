package sqldb

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

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

func Test_isNonSQLScannerStruct(t *testing.T) {
	tests := []struct {
		t    reflect.Type
		want bool
	}{
		// Structs that do not implement sql.Scanner
		{t: reflect.TypeFor[struct{ X int }](), want: true},

		// Structs that implement sql.Scanner
		{t: reflect.TypeFor[time.Time](), want: false},
		{t: reflect.TypeFor[sql.NullTime](), want: false},

		// Non struct types
		{t: reflect.TypeFor[int](), want: false},
		{t: reflect.TypeFor[string](), want: false},
		{t: reflect.TypeFor[[]byte](), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.t.String(), func(t *testing.T) {
			if got := isNonSQLScannerStruct(tt.t); got != tt.want {
				t.Errorf("isNonSQLScannerStruct(%s) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}

func Test_derefStruct(t *testing.T) {
	type testStruct struct{ X int }

	t.Run("plain struct", func(t *testing.T) {
		s := testStruct{X: 42}
		got, err := derefStruct(reflect.ValueOf(s))
		require.NoError(t, err)
		assert.Equal(t, 42, got.FieldByName("X").Interface())
	})

	t.Run("pointer to struct", func(t *testing.T) {
		s := &testStruct{X: 7}
		got, err := derefStruct(reflect.ValueOf(s))
		require.NoError(t, err)
		assert.Equal(t, 7, got.FieldByName("X").Interface())
	})

	t.Run("double pointer to struct", func(t *testing.T) {
		s := &testStruct{X: 99}
		p := &s
		got, err := derefStruct(reflect.ValueOf(p))
		require.NoError(t, err)
		assert.Equal(t, 99, got.FieldByName("X").Interface())
	})

	t.Run("nil pointer", func(t *testing.T) {
		var s *testStruct
		_, err := derefStruct(reflect.ValueOf(s))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil pointer")
	})

	t.Run("nil double pointer", func(t *testing.T) {
		var s *testStruct
		p := &s
		_, err := derefStruct(reflect.ValueOf(p))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil pointer")
	})

	t.Run("non-struct type", func(t *testing.T) {
		_, err := derefStruct(reflect.ValueOf(42))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected struct")
	})

	t.Run("pointer to non-struct", func(t *testing.T) {
		v := 42
		_, err := derefStruct(reflect.ValueOf(&v))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected struct")
	})
}
