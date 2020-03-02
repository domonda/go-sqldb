package mockconn

import (
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	sqldb "github.com/domonda/go-sqldb"
)

func TestRow(t *testing.T) {
	type Struct struct {
		ID            string `db:"id,pk"`
		Int           int    `db:"int"`
		UntaggedField int
		StrPtr        *string      `db:"str_ptr"`
		NilPtr        *byte        `db:"nil_ptr"`
		Bools         pq.BoolArray `db:"bools"`
	}

	str := "Hello World!"
	input := Struct{"myID", 66, -1, &str, nil, pq.BoolArray{true, false}}
	row := NewRow(input, sqldb.DefaultStructFieldTagNaming)

	cols, err := row.Columns()
	assert.NoError(t, err)
	assert.Equal(t, []string{"id", "int", "untagged_field", "str_ptr", "nil_ptr", "bools"}, cols)

	var output Struct
	err = row.Scan(&output.ID, &output.Int, &output.UntaggedField, &output.StrPtr, &output.NilPtr, &output.Bools)
	assert.NoError(t, err)
	assert.Equal(t, input, output)
}
