package tests

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/domonda/go-sqldb/mockconn"
	"github.com/domonda/go-types/uu"
)

type embed struct {
	Bool bool `db:"bool"`
}

type testRow struct {
	ID  uu.ID `db:"id,pk"`
	Int int   `db:"int"`
	embed
	Str           string `db:"str"`
	Ignore        int    `db:"-"`
	UntaggedField int
	CreatedAt     time.Time `db:"created_at"`
}

func TestInsertStruct(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockconn.New(buf, nil)

	row := new(testRow)
	expected := `INSERT INTO public.table("id","int","bool","str","untagged_field","created_at") VALUES($1,$2,$3,$4,$5,$6)` + "\n"

	err := conn.InsertStruct("public.table", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, buf.String())
}

func TestUpdateStruct(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockconn.New(buf, nil)

	row := new(testRow)
	expected := `UPDATE public.table SET "int"=$2,"bool"=$3,"str"=$4,"untagged_field"=$5,"created_at"=$6 WHERE "id"=$1` + "\n"

	err := conn.UpdateStruct("public.table", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, buf.String())
}

func TestUpsertStruct(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockconn.New(buf, nil)

	row := new(testRow)
	expected := `INSERT INTO public.table("id","int","bool","str","untagged_field","created_at") VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT("id") DO UPDATE SET "int"=$2,"bool"=$3,"str"=$4,"untagged_field"=$5,"created_at"=$6` + "\n"

	err := conn.UpsertStruct("public.table", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, buf.String())
}

type multiPrimaryKeyRow struct {
	FirstID  string `db:"first_id,pk"`
	SecondID string `db:"second_id,pk"`
	ThirdID  string `db:"third_id,pk"`

	CreatedAt time.Time `db:"created_at"`
}

func TestUpsertStructMultiPK(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockconn.New(buf, nil)

	row := new(multiPrimaryKeyRow)
	expected := `INSERT INTO public.multi_pk("first_id","second_id","third_id","created_at") VALUES($1,$2,$3,$4) ON CONFLICT("first_id","second_id","third_id") DO UPDATE SET "created_at"=$4` + "\n"

	err := conn.UpsertStruct("public.multi_pk", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, buf.String())
}

func TestUpdateStructMultiPK(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockconn.New(buf, nil)

	row := new(multiPrimaryKeyRow)
	expected := `UPDATE public.multi_pk SET "created_at"=$4 WHERE "first_id"=$1 AND "second_id"=$2 AND "third_id"=$3` + "\n"

	err := conn.UpdateStruct("public.multi_pk", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, buf.String())
}