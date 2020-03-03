package mockconn

import (
	"bytes"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-types/uu"
)

type embed struct {
	Bool bool `db:"bool"`
}

type testRow struct {
	ID  uu.ID `db:"id,pk"`
	Int int   `db:"int"`
	embed
	Str           string  `db:"str"`
	StrPtr        *string `db:"str_ptr"`
	NilPtr        *byte   `db:"nil_ptr"`
	Ignore        int     `db:"-"`
	UntaggedField int
	CreatedAt     time.Time    `db:"created_at"`
	Bools         pq.BoolArray `db:"bools"`
}

func TestInsertQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)

	rowProvider := &SingleRowProvider{Row: NewRow(struct{ True bool }{true}, sqldb.DefaultStructFieldTagNaming)}
	conn := New(queryOutput, rowProvider)

	str := "Hello World!"
	values := sqldb.Values{
		"id":             uu.IDNil,
		"int":            66,
		"bool":           true,
		"str":            "Hello World!",
		"str_ptr":        &str,
		"nil_ptr":        nil,
		"untagged_field": -1,
		"created_at":     time.Now(),
		"bools":          pq.BoolArray{true, false},
	}

	expected := `INSERT INTO public.table("bool","bools","created_at","id","int","nil_ptr","str","str_ptr","untagged_field") VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	err := conn.Insert("public.table", values)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `INSERT INTO public.table("bool","bools","created_at","id","int","nil_ptr","str","str_ptr","untagged_field") VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (id) DO NOTHING RETURNING TRUE`
	inserted, err := conn.InsertUnique("public.table", values, "(id)")
	assert.NoError(t, err)
	assert.True(t, inserted)
	assert.Equal(t, expected, queryOutput.String())
}

func TestInsertStructQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	conn := New(queryOutput, nil)

	row := new(testRow)

	expected := `INSERT INTO public.table("id","int","bool","str","str_ptr","nil_ptr","untagged_field","created_at","bools") VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	err := conn.InsertStruct("public.table", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `INSERT INTO public.table("id","untagged_field","bools") VALUES($1,$2,$3)`
	err = conn.InsertStruct("public.table", row, "id", "untagged_field", "bools")
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `INSERT INTO public.table("id","int","bool","str","str_ptr","bools") VALUES($1,$2,$3,$4,$5,$6)`
	err = conn.InsertStructIgnoreColums("public.table", row, "nil_ptr", "untagged_field", "created_at")
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())
}

func TestUpdateQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	conn := New(queryOutput, nil)

	str := "Hello World!"
	values := sqldb.Values{
		"int":            66,
		"bool":           true,
		"str":            "Hello World!",
		"str_ptr":        &str,
		"nil_ptr":        nil,
		"untagged_field": -1,
		"created_at":     time.Now(),
		"bools":          pq.BoolArray{true, false},
	}

	expected := `UPDATE public.table SET "bool"=$2,"bools"=$3,"created_at"=$4,"int"=$5,"nil_ptr"=$6,"str"=$7,"str_ptr"=$8,"untagged_field"=$9 WHERE id = $1`
	err := conn.Update("public.table", values, "id = $1", 1)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `UPDATE public.table SET "bool"=$3,"bools"=$4,"created_at"=$5,"int"=$6,"nil_ptr"=$7,"str"=$8,"str_ptr"=$9,"untagged_field"=$10 WHERE a = $1 AND b = $2`
	err = conn.Update("public.table", values, "a = $1 AND b = $2", 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())
}

func TestUpdateReturningQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	conn := New(queryOutput, nil)

	str := "Hello World!"
	values := sqldb.Values{
		"int":            66,
		"bool":           true,
		"str":            "Hello World!",
		"str_ptr":        &str,
		"nil_ptr":        nil,
		"untagged_field": -1,
		"created_at":     time.Now(),
		"bools":          pq.BoolArray{true, false},
	}

	expected := `UPDATE public.table SET "bool"=$2,"bools"=$3,"created_at"=$4,"int"=$5,"nil_ptr"=$6,"str"=$7,"str_ptr"=$8,"untagged_field"=$9 WHERE id = $1 RETURNING *`
	err := conn.UpdateReturningRow("public.table", values, "*", "id = $1", 1).Scan()
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `UPDATE public.table SET "bool"=$2,"bools"=$3,"created_at"=$4,"int"=$5,"nil_ptr"=$6,"str"=$7,"str_ptr"=$8,"untagged_field"=$9 WHERE id = $1 RETURNING created_at,untagged_field`
	err = conn.UpdateReturningRows("public.table", values, "created_at,untagged_field", "id = $1", 1, 2).ScanSlice(nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())
}

func TestUpdateStructQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	conn := New(queryOutput, nil)

	row := new(testRow)

	expected := `UPDATE public.table SET "int"=$2,"bool"=$3,"str"=$4,"str_ptr"=$5,"nil_ptr"=$6,"untagged_field"=$7,"created_at"=$8,"bools"=$9 WHERE "id"=$1`
	err := conn.UpdateStruct("public.table", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `UPDATE public.table SET "bool"=$2,"str"=$3,"created_at"=$4 WHERE "id"=$1`
	err = conn.UpdateStruct("public.table", row, "bool", "str", "created_at")
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `UPDATE public.table SET "int"=$2,"bool"=$3,"str_ptr"=$4,"nil_ptr"=$5,"created_at"=$6 WHERE "id"=$1`
	err = conn.UpdateStructIgnoreColums("public.table", row, "untagged_field", "str", "bools")
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())
}

func TestUpsertStructQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	conn := New(queryOutput, nil)

	row := new(testRow)
	expected := `INSERT INTO public.table("id","int","bool","str","str_ptr","nil_ptr","untagged_field","created_at","bools") VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)` +
		` ON CONFLICT("id") DO UPDATE SET "int"=$2,"bool"=$3,"str"=$4,"str_ptr"=$5,"nil_ptr"=$6,"untagged_field"=$7,"created_at"=$8,"bools"=$9`

	err := conn.UpsertStruct("public.table", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())
}

type multiPrimaryKeyRow struct {
	FirstID  string `db:"first_id,pk"`
	SecondID string `db:"second_id,pk"`
	ThirdID  string `db:"third_id,pk"`

	CreatedAt time.Time `db:"created_at"`
}

func TestUpsertStructMultiPKQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	conn := New(queryOutput, nil)

	row := new(multiPrimaryKeyRow)
	expected := `INSERT INTO public.multi_pk("first_id","second_id","third_id","created_at") VALUES($1,$2,$3,$4) ON CONFLICT("first_id","second_id","third_id") DO UPDATE SET "created_at"=$4`

	err := conn.UpsertStruct("public.multi_pk", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())
}

func TestUpdateStructMultiPKQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	conn := New(queryOutput, nil)

	row := new(multiPrimaryKeyRow)
	expected := `UPDATE public.multi_pk SET "created_at"=$4 WHERE "first_id"=$1 AND "second_id"=$2 AND "third_id"=$3`

	err := conn.UpdateStruct("public.multi_pk", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())
}
