package pqconn

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"testing"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/pqconn"
)

func newTestUUID(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 2
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

var refl = sqldb.NewTaggedStructReflector()

const testArraysSchema = /*sql*/ `
CREATE TABLE IF NOT EXISTS test_arrays (
	id          uuid PRIMARY KEY,
	int_array   integer[]    NOT NULL DEFAULT '{}',
	text_array  text[]       NOT NULL DEFAULT '{}',
	float_array float8[],
	bool_array  boolean[],
	uuid_array  uuid[]
)`

type testArraysRow struct {
	sqldb.TableName `db:"test_arrays"`

	ID         string    `db:"id,primarykey"`
	IntArray   []int64   `db:"int_array"`
	TextArray  []string  `db:"text_array"`
	FloatArray []float64 `db:"float_array"`
	BoolArray  []bool    `db:"bool_array"`
	UUIDArray  []string  `db:"uuid_array"`
}

func testConn(t *testing.T) sqldb.Connection {
	t.Helper()
	ctx := context.Background()

	port, err := strconv.ParseUint(postgresPort, 10, 16)
	if err != nil {
		t.Fatalf("Invalid port %q: %v", postgresPort, err)
	}

	config := &sqldb.Config{
		Driver:   "postgres",
		Host:     postgresHost,
		Port:     uint16(port),
		User:     postgresUser,
		Password: postgresPassword,
		Database: dbName,
		Extra:    map[string]string{"sslmode": "disable"},
	}

	conn, err := pqconn.Connect(ctx, config)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	err = conn.Exec(ctx, testArraysSchema)
	if err != nil {
		t.Fatalf("Failed to create test_arrays table: %v", err)
	}
	t.Cleanup(func() {
		conn.Exec(ctx,
			/*sql*/ `DROP TABLE IF EXISTS test_arrays`,
		)
	})

	return conn
}

func TestArrayStructInsertAndQueryRow(t *testing.T) {
	ctx := context.Background()
	c := testConn(t)

	id1 := newTestUUID(t)
	id2 := newTestUUID(t)
	id3 := newTestUUID(t)

	input := &testArraysRow{
		ID:         newTestUUID(t),
		IntArray:   []int64{10, 20, 30},
		TextArray:  []string{"hello", "world"},
		FloatArray: []float64{1.5, 2.5, 3.5},
		BoolArray:  []bool{true, false, true},
		UUIDArray:  []string{id1, id2, id3},
	}

	err := sqldb.InsertRowStruct(ctx, c, refl, pqconn.QueryBuilder{}, c, input)
	if err != nil {
		t.Fatalf("InsertRowStruct: %v", err)
	}

	var got testArraysRow
	err = sqldb.QueryRow(ctx, c, refl, c,
		/*sql*/ `SELECT * FROM test_arrays WHERE id = $1`,
		input.ID,
	).Scan(&got)
	if err != nil {
		t.Fatalf("QueryRow.Scan: %v", err)
	}

	if got.ID != input.ID {
		t.Errorf("ID = %s, want %s", got.ID, input.ID)
	}
	assertInt64Slice(t, "IntArray", got.IntArray, input.IntArray)
	assertStringSlice(t, "TextArray", got.TextArray, input.TextArray)
	assertFloat64Slice(t, "FloatArray", got.FloatArray, input.FloatArray)
	assertBoolSlice(t, "BoolArray", got.BoolArray, input.BoolArray)
	assertUUIDSlice(t, "UUIDArray", got.UUIDArray, input.UUIDArray)
}

func TestArrayStructQueryRowsAsSlice(t *testing.T) {
	ctx := context.Background()
	c := testConn(t)

	rows := []testArraysRow{
		{
			ID:        newTestUUID(t),
			IntArray:  []int64{1, 2},
			TextArray: []string{"a"},
		},
		{
			ID:        newTestUUID(t),
			IntArray:  []int64{3, 4, 5},
			TextArray: []string{"b", "c"},
		},
	}
	for i := range rows {
		err := sqldb.InsertRowStruct(ctx, c, refl, pqconn.QueryBuilder{}, c, &rows[i])
		if err != nil {
			t.Fatalf("InsertRowStruct[%d]: %v", i, err)
		}
	}

	got, err := sqldb.QueryRowsAsSlice[testArraysRow](ctx, c, refl, c,
		/*sql*/ `SELECT * FROM test_arrays ORDER BY int_array[1]`,
	)
	if err != nil {
		t.Fatalf("QueryRowsAsSlice: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}

	assertInt64Slice(t, "row[0].IntArray", got[0].IntArray, []int64{1, 2})
	assertStringSlice(t, "row[0].TextArray", got[0].TextArray, []string{"a"})
	assertInt64Slice(t, "row[1].IntArray", got[1].IntArray, []int64{3, 4, 5})
	assertStringSlice(t, "row[1].TextArray", got[1].TextArray, []string{"b", "c"})
}

func TestArrayStructNullSlices(t *testing.T) {
	ctx := context.Background()
	c := testConn(t)

	input := &testArraysRow{
		ID:         newTestUUID(t),
		IntArray:   []int64{},
		TextArray:  []string{},
		FloatArray: nil,
		BoolArray:  nil,
		UUIDArray:  nil,
	}

	err := sqldb.InsertRowStruct(ctx, c, refl, pqconn.QueryBuilder{}, c, input)
	if err != nil {
		t.Fatalf("InsertRowStruct: %v", err)
	}

	var got testArraysRow
	err = sqldb.QueryRow(ctx, c, refl, c,
		/*sql*/ `SELECT * FROM test_arrays WHERE id = $1`,
		input.ID,
	).Scan(&got)
	if err != nil {
		t.Fatalf("QueryRow.Scan: %v", err)
	}

	// NOT NULL columns with DEFAULT '{}' get empty array when inserted with empty slice
	if len(got.IntArray) != 0 {
		t.Errorf("IntArray = %v, want empty", got.IntArray)
	}
	if len(got.TextArray) != 0 {
		t.Errorf("TextArray = %v, want empty", got.TextArray)
	}

	// Nullable columns inserted with nil should scan back as nil
	if got.FloatArray != nil {
		t.Errorf("FloatArray = %v, want nil", got.FloatArray)
	}
	if got.BoolArray != nil {
		t.Errorf("BoolArray = %v, want nil", got.BoolArray)
	}
	if got.UUIDArray != nil {
		t.Errorf("UUIDArray = %v, want nil", got.UUIDArray)
	}
}

func TestArrayStructQueryRowByPrimaryKey(t *testing.T) {
	ctx := context.Background()
	c := testConn(t)

	input := &testArraysRow{
		ID:        newTestUUID(t),
		IntArray:  []int64{42},
		TextArray: []string{"pk-test"},
	}

	err := sqldb.InsertRowStruct(ctx, c, refl, pqconn.QueryBuilder{}, c, input)
	if err != nil {
		t.Fatalf("InsertRowStruct: %v", err)
	}

	got, err := sqldb.QueryRowByPrimaryKey[testArraysRow](ctx, c, refl, pqconn.QueryBuilder{}, c, input.ID)
	if err != nil {
		t.Fatalf("QueryRowByPrimaryKey: %v", err)
	}

	assertInt64Slice(t, "IntArray", got.IntArray, []int64{42})
	assertStringSlice(t, "TextArray", got.TextArray, []string{"pk-test"})
}

func TestArrayStructTransaction(t *testing.T) {
	ctx := context.Background()
	c := testConn(t)

	input := &testArraysRow{
		ID:        newTestUUID(t),
		IntArray:  []int64{100, 200},
		TextArray: []string{"tx-test"},
	}

	err := sqldb.Transaction(ctx, c, nil, func(tx sqldb.Connection) error {
		return sqldb.InsertRowStruct(ctx, tx, refl, pqconn.QueryBuilder{}, c, input)
	})
	if err != nil {
		t.Fatalf("Transaction insert: %v", err)
	}

	var got testArraysRow
	err = sqldb.QueryRow(ctx, c, refl, c,
		/*sql*/ `SELECT * FROM test_arrays WHERE id = $1`,
		input.ID,
	).Scan(&got)
	if err != nil {
		t.Fatalf("QueryRow.Scan: %v", err)
	}

	assertInt64Slice(t, "IntArray", got.IntArray, []int64{100, 200})
	assertStringSlice(t, "TextArray", got.TextArray, []string{"tx-test"})
}

func TestArrayStructQueryCallback(t *testing.T) {
	ctx := context.Background()
	c := testConn(t)

	input := &testArraysRow{
		ID:        newTestUUID(t),
		IntArray:  []int64{7, 8, 9},
		TextArray: []string{"callback"},
	}
	err := sqldb.InsertRowStruct(ctx, c, refl, pqconn.QueryBuilder{}, c, input)
	if err != nil {
		t.Fatalf("InsertRowStruct: %v", err)
	}

	var called bool
	err = sqldb.QueryCallback(ctx, c, refl, c,
		func(row testArraysRow) {
			called = true
			assertInt64Slice(t, "IntArray", row.IntArray, []int64{7, 8, 9})
			assertStringSlice(t, "TextArray", row.TextArray, []string{"callback"})
		},
		/*sql*/ `SELECT * FROM test_arrays WHERE id = $1`,
		input.ID,
	)
	if err != nil {
		t.Fatalf("QueryCallback: %v", err)
	}
	if !called {
		t.Error("callback was not called")
	}
}

func TestArrayStructSpecialStrings(t *testing.T) {
	ctx := context.Background()
	c := testConn(t)

	input := &testArraysRow{
		ID:       newTestUUID(t),
		IntArray: []int64{},
		TextArray: []string{
			"with spaces",
			"with\ttabs",
			`with "quotes"`,
			"with,commas",
			"with{braces}",
			"",
		},
	}

	err := sqldb.InsertRowStruct(ctx, c, refl, pqconn.QueryBuilder{}, c, input)
	if err != nil {
		t.Fatalf("InsertRowStruct: %v", err)
	}

	var got testArraysRow
	err = sqldb.QueryRow(ctx, c, refl, c,
		/*sql*/ `SELECT * FROM test_arrays WHERE id = $1`,
		input.ID,
	).Scan(&got)
	if err != nil {
		t.Fatalf("QueryRow.Scan: %v", err)
	}

	assertStringSlice(t, "TextArray", got.TextArray, input.TextArray)
}

func TestArraySliceAsQueryArg(t *testing.T) {
	ctx := context.Background()
	c := testConn(t)

	id1 := newTestUUID(t)
	id2 := newTestUUID(t)
	for _, row := range []*testArraysRow{
		{ID: id1, IntArray: []int64{10, 20, 30}, TextArray: []string{"a"}},
		{ID: id2, IntArray: []int64{40, 50, 60}, TextArray: []string{"b"}},
	} {
		err := sqldb.InsertRowStruct(ctx, c, refl, pqconn.QueryBuilder{}, c, row)
		if err != nil {
			t.Fatalf("InsertRowStruct: %v", err)
		}
	}

	// Use array containment operator with a slice argument
	got, err := sqldb.QueryRowsAsSlice[testArraysRow](ctx, c, refl, c,
		/*sql*/ `SELECT * FROM test_arrays WHERE int_array @> $1`,
		[]int64{10, 20},
	)
	if err != nil {
		t.Fatalf("QueryRowsAsSlice: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ID != id1 {
		t.Errorf("ID = %s, want %s", got[0].ID, id1)
	}
}

func TestArrayPreparedStmt(t *testing.T) {
	ctx := context.Background()
	c := testConn(t)

	input := &testArraysRow{
		ID:        newTestUUID(t),
		IntArray:  []int64{11, 22},
		TextArray: []string{"prepared"},
	}
	err := sqldb.InsertRowStruct(ctx, c, refl, pqconn.QueryBuilder{}, c, input)
	if err != nil {
		t.Fatalf("InsertRowStruct: %v", err)
	}

	// Use prepared statement query that returns rows with arrays
	queryFunc, closeStmt, err := sqldb.QueryRowAsStmt[testArraysRow](ctx, c, refl, c,
		/*sql*/ `SELECT * FROM test_arrays WHERE id = $1`,
	)
	if err != nil {
		t.Fatalf("QueryRowAsStmt: %v", err)
	}
	defer closeStmt()

	got, err := queryFunc(ctx, input.ID)
	if err != nil {
		t.Fatalf("queryFunc: %v", err)
	}

	assertInt64Slice(t, "IntArray", got.IntArray, []int64{11, 22})
	assertStringSlice(t, "TextArray", got.TextArray, []string{"prepared"})
}

// Assertion helpers

func assertInt64Slice(t *testing.T, name string, got, want []int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: len = %d, want %d (got %v)", name, len(got), len(want), got)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %d, want %d", name, i, got[i], want[i])
		}
	}
}

func assertStringSlice(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: len = %d, want %d (got %v)", name, len(got), len(want), got)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %q, want %q", name, i, got[i], want[i])
		}
	}
}

func assertFloat64Slice(t *testing.T, name string, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: len = %d, want %d (got %v)", name, len(got), len(want), got)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %f, want %f", name, i, got[i], want[i])
		}
	}
}

func assertBoolSlice(t *testing.T, name string, got, want []bool) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: len = %d, want %d (got %v)", name, len(got), len(want), got)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %v, want %v", name, i, got[i], want[i])
		}
	}
}

func assertUUIDSlice(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: len = %d, want %d (got %v)", name, len(got), len(want), got)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %s, want %s", name, i, got[i], want[i])
		}
	}
}
