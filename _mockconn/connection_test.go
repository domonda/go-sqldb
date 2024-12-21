package mockconn

/*
import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
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
	ReadOnly      int     `db:"read_only,readonly"`
	Ignore        int     `db:"-"`
	UntaggedField int
	CreatedAt     time.Time    `db:"created_at,default"`
	Bools         pq.BoolArray `db:"bools"`
}

func TestInsertQuery(t *testing.T) {
	naming := &sqldb.TaggedStructReflector{NameTag: "db", Ignore: "-", UntaggedNameFunc: sqldb.ToSnakeCase}
	queryOutput := bytes.NewBuffer(nil)
	rowProvider := NewSingleRowProvider(NewRow(struct{ True bool }{true}, naming))
	conn := New(context.Background(), queryOutput, rowProvider).WithStructFieldMapper(naming)
	ctx := db.ContextWithConn(context.Background(), conn)

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

	expected := `INSERT INTO public.table("bool","bools","created_at","id","int","nil_ptr","str","str_ptr","untagged_field") VALUES(?1,?2,?3,?4,?5,?6,?7,?8,?9)`
	err := db.Insert(ctx, "public.table", values)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `INSERT INTO public.table("bool","bools","created_at","id","int","nil_ptr","str","str_ptr","untagged_field") VALUES(?1,?2,?3,?4,?5,?6,?7,?8,?9) ON CONFLICT (id) DO NOTHING RETURNING TRUE`
	inserted, err := db.InsertUnique(ctx, "public.table", values, "id")
	assert.NoError(t, err)
	assert.True(t, inserted)
	assert.Equal(t, expected, queryOutput.String())
}

func TestInsertStructQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	naming := &sqldb.TaggedStructReflector{
		NameTag:          "db",
		Ignore:           "-",
		PrimaryKey:       "pk",
		ReadOnly:         "readonly",
		Default:          "default",
		UntaggedNameFunc: sqldb.ToSnakeCase,
	}
	conn := New(context.Background(), queryOutput, nil).WithStructFieldMapper(naming)
	ctx := db.ContextWithConn(context.Background(), conn)

	row := new(testRow)

	expected := `INSERT INTO public.table("id","int","bool","str","str_ptr","nil_ptr","untagged_field","created_at","bools") VALUES(?1,?2,?3,?4,?5,?6,?7,?8,?9)`
	err := db.InsertStruct(ctx, "public.table", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `INSERT INTO public.table("id","untagged_field","bools") VALUES(?1,?2,?3)`
	err = db.InsertStruct(ctx, "public.table", row, sqldb.OnlyColumns("id", "untagged_field", "bools"))
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `INSERT INTO public.table("id","int","bool","str","str_ptr","bools") VALUES(?1,?2,?3,?4,?5,?6)`
	err = db.InsertStruct(ctx, "public.table", row, sqldb.IgnoreColumns("nil_ptr", "untagged_field", "created_at"))
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())
}

func TestInsertUniqueStructQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	naming := &sqldb.TaggedStructReflector{
		NameTag:          "db",
		Ignore:           "-",
		PrimaryKey:       "pk",
		ReadOnly:         "readonly",
		Default:          "default",
		UntaggedNameFunc: sqldb.ToSnakeCase,
	}
	rowProvider := NewSingleRowProvider(NewRow(struct{ True bool }{true}, naming))
	conn := New(context.Background(), queryOutput, rowProvider).WithStructFieldMapper(naming)
	ctx := db.ContextWithConn(context.Background(), conn)

	row := new(testRow)

	expected := `INSERT INTO public.table("id","int","bool","str","str_ptr","nil_ptr","untagged_field","created_at","bools") VALUES(?1,?2,?3,?4,?5,?6,?7,?8,?9) ON CONFLICT (id) DO NOTHING RETURNING TRUE`
	inserted, err := db.InsertUniqueStruct(ctx, "public.table", row, "(id)")
	assert.NoError(t, err)
	assert.True(t, inserted)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `INSERT INTO public.table("id","untagged_field","bools") VALUES(?1,?2,?3) ON CONFLICT (id, untagged_field) DO NOTHING RETURNING TRUE`
	inserted, err = db.InsertUniqueStruct(ctx, "public.table", row, "(id, untagged_field)", sqldb.OnlyColumns("id", "untagged_field", "bools"))
	assert.NoError(t, err)
	assert.True(t, inserted)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `INSERT INTO public.table("id","int","bool","str","str_ptr","bools") VALUES(?1,?2,?3,?4,?5,?6) ON CONFLICT (id) DO NOTHING RETURNING TRUE`
	inserted, err = db.InsertUniqueStruct(ctx, "public.table", row, "(id)", sqldb.IgnoreColumns("nil_ptr", "untagged_field", "created_at"))
	assert.NoError(t, err)
	assert.True(t, inserted)
	assert.Equal(t, expected, queryOutput.String())
}

func TestUpdateQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	naming := &sqldb.TaggedStructReflector{NameTag: "db", Ignore: "-", UntaggedNameFunc: sqldb.ToSnakeCase}
	conn := New(context.Background(), queryOutput, nil).WithStructFieldMapper(naming)
	ctx := db.ContextWithConn(context.Background(), conn)

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

	// Passing one varidic arg as ?1, moves the index of the rest of the args by 1
	expected := `UPDATE public.table SET "bool"=?2, "bools"=?3, "created_at"=?4, "int"=?5, "nil_ptr"=?6, "str"=?7, "str_ptr"=?8, "untagged_field"=?9 WHERE id = ?1`
	err := db.Update(ctx, "public.table", values, "id = ?1", 1)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	// Passing two varidic args as ?1 and ?2, moves the index of the rest of the args by 2
	expected = `UPDATE public.table SET "bool"=?3, "bools"=?4, "created_at"=?5, "int"=?6, "nil_ptr"=?7, "str"=?8, "str_ptr"=?9, "untagged_field"=?10 WHERE a = ?1 AND b = ?2`
	err = db.Update(ctx, "public.table", values, "a = ?1 AND b = ?2", 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())
}

func TestUpdateReturningQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	naming := &sqldb.TaggedStructReflector{NameTag: "db", Ignore: "-", UntaggedNameFunc: sqldb.ToSnakeCase}
	conn := New(context.Background(), queryOutput, nil).WithStructFieldMapper(naming)
	ctx := db.ContextWithConn(context.Background(), conn)

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

	// Passing one varidic arg as ?1, moves the index of the rest of the args by 1
	expected := `UPDATE public.table SET "bool"=?2, "bools"=?3, "created_at"=?4, "int"=?5, "nil_ptr"=?6, "str"=?7, "str_ptr"=?8, "untagged_field"=?9 WHERE id = ?1 RETURNING *`
	err := db.UpdateReturningRow(ctx, "public.table", values, "*", "id = ?1", 1).Scan()
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	// Passing two varidic args as ?1 and ?2, moves the index of the rest of the args by 2
	expected = `UPDATE public.table SET "bool"=?3, "bools"=?4, "created_at"=?5, "int"=?6, "nil_ptr"=?7, "str"=?8, "str_ptr"=?9, "untagged_field"=?10 WHERE id = ?1 RETURNING created_at,untagged_field`
	err = db.UpdateReturningRows(ctx, "public.table", values, "created_at,untagged_field", "id = ?1", 1, 2).ScanSlice(nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())
}

func TestUpdateStructQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	naming := &sqldb.TaggedStructReflector{
		NameTag:          "db",
		Ignore:           "-",
		PrimaryKey:       "pk",
		ReadOnly:         "readonly",
		Default:          "default",
		UntaggedNameFunc: sqldb.ToSnakeCase,
	}
	conn := New(context.Background(), queryOutput, nil).WithStructFieldMapper(naming)
	ctx := db.ContextWithConn(context.Background(), conn)

	row := new(testRow)

	expected := `UPDATE public.table SET "int"=?2, "bool"=?3, "str"=?4, "str_ptr"=?5, "nil_ptr"=?6, "untagged_field"=?7, "created_at"=?8, "bools"=?9 WHERE "id"=?1`
	err := db.UpdateStruct(ctx, "public.table", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `UPDATE public.table SET "bool"=?2, "str"=?3, "created_at"=?4 WHERE "id"=?1`
	err = db.UpdateStruct(ctx, "public.table", row, sqldb.OnlyColumns("id", "bool", "str", "created_at"))
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())

	queryOutput.Reset()
	expected = `UPDATE public.table SET "int"=?2, "bool"=?3, "str_ptr"=?4, "nil_ptr"=?5, "created_at"=?6 WHERE "id"=?1`
	err = db.UpdateStruct(ctx, "public.table", row, sqldb.IgnoreColumns("untagged_field", "str", "bools"))
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())
}

func TestUpsertStructQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	naming := &sqldb.TaggedStructReflector{
		NameTag:          "db",
		Ignore:           "-",
		PrimaryKey:       "pk",
		ReadOnly:         "readonly",
		Default:          "default",
		UntaggedNameFunc: sqldb.ToSnakeCase,
	}
	conn := New(context.Background(), queryOutput, nil).WithStructFieldMapper(naming)
	ctx := db.ContextWithConn(context.Background(), conn)

	row := new(testRow)
	expected := `INSERT INTO public.table("id","int","bool","str","str_ptr","nil_ptr","untagged_field","created_at","bools") VALUES(?1,?2,?3,?4,?5,?6,?7,?8,?9)` +
		` ON CONFLICT("id") DO UPDATE SET "int"=?2, "bool"=?3, "str"=?4, "str_ptr"=?5, "nil_ptr"=?6, "untagged_field"=?7, "created_at"=?8, "bools"=?9`

	err := db.UpsertStruct(ctx, "public.table", row)
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
	naming := &sqldb.TaggedStructReflector{
		NameTag:          "db",
		Ignore:           "-",
		PrimaryKey:       "pk",
		ReadOnly:         "readonly",
		Default:          "default",
		UntaggedNameFunc: sqldb.ToSnakeCase,
	}
	conn := New(context.Background(), queryOutput, nil).WithStructFieldMapper(naming)
	ctx := db.ContextWithConn(context.Background(), conn)

	row := new(multiPrimaryKeyRow)
	expected := `INSERT INTO public.multi_pk("first_id","second_id","third_id","created_at") VALUES(?1,?2,?3,?4) ON CONFLICT("first_id","second_id","third_id") DO UPDATE SET "created_at"=?4`

	err := db.UpsertStruct(ctx, "public.multi_pk", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())
}

func TestUpdateStructMultiPKQuery(t *testing.T) {
	queryOutput := bytes.NewBuffer(nil)
	naming := &sqldb.TaggedStructReflector{
		NameTag:          "db",
		Ignore:           "-",
		PrimaryKey:       "pk",
		ReadOnly:         "readonly",
		Default:          "default",
		UntaggedNameFunc: sqldb.ToSnakeCase,
	}
	conn := New(context.Background(), queryOutput, nil).WithStructFieldMapper(naming)
	ctx := db.ContextWithConn(context.Background(), conn)

	row := new(multiPrimaryKeyRow)
	expected := `UPDATE public.multi_pk SET "created_at"=?4 WHERE "first_id"=?1 AND "second_id"=?2 AND "third_id"=?3`

	err := db.UpdateStruct(ctx, "public.multi_pk", row)
	assert.NoError(t, err)
	assert.Equal(t, expected, queryOutput.String())
}
*/
