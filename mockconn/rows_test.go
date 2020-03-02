package mockconn

import (
	"fmt"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	sqldb "github.com/domonda/go-sqldb"
)

func TestRows(t *testing.T) {
	type Struct struct {
		ID            string `db:"id,pk"`
		Int           int    `db:"int"`
		UntaggedField int
		StrPtr        *string      `db:"str_ptr"`
		NilPtr        *byte        `db:"nil_ptr"`
		Bools         pq.BoolArray `db:"bools"`
	}

	var input []*Struct
	for i := 0; i < 20; i++ {
		str := fmt.Sprintf("Hello World %d", i)
		input = append(input, &Struct{"myID", i, -1, &str, nil, pq.BoolArray{true, false, i%2 == 0}})
	}

	rows := NewRows(input, sqldb.DefaultStructFieldTagNaming)

	cols, err := rows.Columns()
	assert.NoError(t, err)
	assert.Equal(t, []string{"id", "int", "untagged_field", "str_ptr", "nil_ptr", "bools"}, cols)

	assert.NoError(t, rows.Err(), "Err() should not return an error")
	for i := range input {
		var output Struct

		assert.Truef(t, rows.Next(), "Next() for row %d should return true")

		err = rows.Scan(&output.ID, &output.Int, &output.UntaggedField, &output.StrPtr, &output.NilPtr, &output.Bools)
		assert.NoError(t, err)
		assert.Equal(t, *input[i], output)
	}
	assert.NoError(t, rows.Err(), "Err() should not return an error")

	assert.False(t, rows.Next(), "Next() after all rows should return false")
	assert.NoError(t, rows.Err(), "Err() should not return an error")

	assert.NoError(t, rows.Close(), "Close() should not return an error")
	assert.False(t, rows.Next(), "Next() after Close() should return false")

	assert.NoError(t, rows.Err(), "Err() should not return an error")
}
