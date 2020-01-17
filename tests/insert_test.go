package tests

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/domonda/go-sqldb/mockimpl"
	"github.com/domonda/go-types/uu"
)

type embed struct {
	Bool bool `db:"bool"`
}

type testRow struct {
	ID  uu.ID `db:"id"`
	Int int   `db:"int"`
	embed
	Str           string `db:"str"`
	Ignore        int    `db:"-"`
	UntaggedField int
	CreatedAt     time.Time `db:"created_at"`
}

func TestInsertStruct(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockimpl.NewConnection(buf)

	row := new(testRow)
	expected := `INSERT INTO public.table("id","int","bool","str","untagged_field","created_at") VALUES($1,$2,$3,$4,$5,$6)` + "\n"

	err := conn.InsertStruct("public.table", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, buf.String())
}

func TestUpsertStruct(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockimpl.NewConnection(buf)

	row := new(testRow)
	expected := `INSERT INTO public.table("id","int","bool","str","untagged_field","created_at") VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT("id") DO UPDATE SET "int"=$2,"bool"=$3,"str"=$4,"untagged_field"=$5,"created_at"=$6` + "\n"

	err := conn.UpsertStruct("public.table", row, "id")
	assert.NoError(t, err)
	assert.Equal(t, expected, buf.String())

	err = conn.UpsertStruct("public.table", row, "xxx")
	assert.Error(t, err, "xxx is not column")
}
