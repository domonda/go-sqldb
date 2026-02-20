package sqldb

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/domonda/go-types/date"
	"github.com/domonda/go-types/notnull"
	"github.com/domonda/go-types/nullable"
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

	d := date.Date("2000-01-01")
	assert.False(t, IsNull(d))

	nd := date.Null
	assert.True(t, IsNull(nd))
	nd = "2000-01-01"
	assert.False(t, IsNull(nd))

	var nj nullable.JSON
	assert.True(t, IsNull(nj))
	nj = nullable.JSON("{}")
	assert.False(t, IsNull(nj))

	var na nullable.IntArray
	assert.True(t, IsNull(na))
	na = []int64{}
	assert.False(t, IsNull(na))

	var nnj notnull.JSON
	assert.False(t, IsNull(nnj))
	nnj = notnull.JSON("{}")
	assert.False(t, IsNull(nnj))

	var nna notnull.IntArray
	assert.False(t, IsNull(nna))
	nna = []int64{}
	assert.False(t, IsNull(nna))
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
		{name: "*int (nil pointer)", typ: reflect.TypeOf((*int)(nil)), want: true},
		{name: "[]byte (nil slice)", typ: reflect.TypeOf([]byte(nil)), want: true},
		{name: "map (nil map)", typ: reflect.TypeOf(map[string]int(nil)), want: true},
		{name: "int (not nullable)", typ: reflect.TypeOf(0), want: false},
		{name: "string (not nullable)", typ: reflect.TypeOf(""), want: false},
		{name: "bool (not nullable)", typ: reflect.TypeOf(false), want: false},
		{name: "sql.NullString (driver.Valuer)", typ: reflect.TypeOf(sql.NullString{}), want: true},
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
	tests := []struct {
		val  any
		want bool
	}{
		{val: time.Time{}, want: true},

		// Not null or zero
		{val: new(int), want: false},

		// TODO more tests
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.val), func(t *testing.T) {
			if got := IsNullOrZero(tt.val); got != tt.want {
				t.Errorf("IsNullOrZero(%#v) = %t, want %t", tt.val, got, tt.want)
			}
		})
	}
}
