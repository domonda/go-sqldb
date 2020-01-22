package impl

import (
	"database/sql"
	"testing"

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
