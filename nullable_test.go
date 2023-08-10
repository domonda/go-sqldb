package sqldb

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/domonda/go-types/date"
	"github.com/domonda/go-types/notnull"
	"github.com/domonda/go-types/nullable"
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
