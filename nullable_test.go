package sqldb

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsNull(t *testing.T) {
	assert.True(t, IsNull(nil))

	var i int
	assert.False(t, IsNull(i))

	var ni sql.NullInt64
	assert.True(t, IsNull(ni))
	ni.Valid = true
	assert.False(t, IsNull(ni))

	var iptr *int
	assert.True(t, IsNull(iptr))
	iptr = new(int)
	assert.False(t, IsNull(iptr))

	var ns sql.NullString
	assert.True(t, IsNull(ns))
	ns = sql.NullString{String: "hello", Valid: true}
	assert.False(t, IsNull(ns))

	var nt sql.NullTime
	assert.True(t, IsNull(nt))
	nt = sql.NullTime{Time: time.Now(), Valid: true}
	assert.False(t, IsNull(nt))
}

func TestNullable_Scan(t *testing.T) {
	t.Run("scan non-nil value", func(t *testing.T) {
		var n Nullable[string]
		err := n.Scan("hello")
		assert.NoError(t, err)
		assert.True(t, n.Valid)
		assert.Equal(t, "hello", n.Val)
	})

	t.Run("scan nil sets invalid", func(t *testing.T) {
		n := Nullable[string]{Val: "old", Valid: true}
		err := n.Scan(nil)
		assert.NoError(t, err)
		assert.False(t, n.Valid)
		assert.Equal(t, "", n.Val)
	})

	t.Run("scan int64", func(t *testing.T) {
		var n Nullable[int64]
		err := n.Scan(int64(42))
		assert.NoError(t, err)
		assert.True(t, n.Valid)
		assert.Equal(t, int64(42), n.Val)
	})

	t.Run("scan bool", func(t *testing.T) {
		var n Nullable[bool]
		err := n.Scan(true)
		assert.NoError(t, err)
		assert.True(t, n.Valid)
		assert.True(t, n.Val)
	})
}

func TestNullable_Value(t *testing.T) {
	t.Run("valid value", func(t *testing.T) {
		n := Nullable[string]{Val: "test", Valid: true}
		v, err := n.Value()
		assert.NoError(t, err)
		assert.Equal(t, "test", v)
	})

	t.Run("invalid returns nil", func(t *testing.T) {
		n := Nullable[string]{Val: "stale", Valid: false}
		v, err := n.Value()
		assert.NoError(t, err)
		assert.Nil(t, v)
	})

	t.Run("implements driver.Valuer", func(t *testing.T) {
		var n Nullable[int64]
		var _ driver.Valuer = n
	})
}

func TestIsNullable(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want bool
	}{
		{name: "*int (nil pointer)", typ: reflect.TypeFor[*int](), want: true},
		{name: "[]byte (nil slice)", typ: reflect.TypeFor[[]byte](), want: true},
		{name: "map (nil map)", typ: reflect.TypeFor[map[string]int](), want: true},
		{name: "int (not nullable)", typ: reflect.TypeFor[int](), want: false},
		{name: "string (not nullable)", typ: reflect.TypeFor[string](), want: false},
		{name: "bool (not nullable)", typ: reflect.TypeFor[bool](), want: false},
		{name: "sql.NullString (driver.Valuer)", typ: reflect.TypeFor[sql.NullString](), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNullable(tt.typ); got != tt.want {
				t.Errorf("IsNullable(%s) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestIsNullOrZero(t *testing.T) {
	var nilIntPtr *int
	var nilSlice []byte
	var nilMap map[string]int

	intVal := 0
	nonZeroInt := 42

	tests := []struct {
		name string
		val  any
		want bool
	}{
		// Zero/null values
		{name: "nil", val: nil, want: true},
		{name: "zero time.Time", val: time.Time{}, want: true},
		{name: "zero int", val: 0, want: true},
		{name: "zero int8", val: int8(0), want: true},
		{name: "zero float64", val: float64(0), want: true},
		{name: "zero string", val: "", want: true},
		{name: "zero bool", val: false, want: true},
		{name: "nil pointer", val: nilIntPtr, want: true},
		{name: "nil slice", val: nilSlice, want: true},
		{name: "nil map", val: nilMap, want: true},
		{name: "zero [16]byte UUID", val: [16]byte{}, want: true},
		{name: "sql.NullString invalid", val: sql.NullString{}, want: true},

		// Not null or zero
		{name: "non-nil pointer to zero", val: &intVal, want: false},
		{name: "non-nil pointer to non-zero", val: &nonZeroInt, want: false},
		{name: "non-zero int", val: 42, want: false},
		{name: "non-zero string", val: "hello", want: false},
		{name: "true bool", val: true, want: false},
		{name: "non-zero float64", val: 3.14, want: false},
		{name: "non-empty slice", val: []byte{1, 2, 3}, want: false},
		{name: "non-empty map", val: map[string]int{"a": 1}, want: false},
		{name: "non-zero [16]byte", val: [16]byte{1}, want: false},
		{name: "sql.NullString valid", val: sql.NullString{String: "x", Valid: true}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNullOrZero(tt.val); got != tt.want {
				t.Errorf("IsNullOrZero(%#v) = %t, want %t", tt.val, got, tt.want)
			}
		})
	}
}
