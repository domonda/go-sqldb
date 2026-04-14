package sqldb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRow_Scan_Scalars(t *testing.T) {
	rows := NewMockRows("id", "name", "active").
		WithRow(int64(1), "Alice", true)
	row := NewRow(rows, NewTaggedStructReflector(), testFormatter, "SELECT * FROM users", nil)

	var id int64
	var name string
	var active bool
	if err := row.Scan(&id, &name, &active); err != nil {
		t.Fatal(err)
	}
	if id != 1 {
		t.Errorf("id: got %d, want 1", id)
	}
	if name != "Alice" {
		t.Errorf("name: got %q, want %q", name, "Alice")
	}
	if !active {
		t.Error("active: got false, want true")
	}
}

func TestRow_Scan_Struct(t *testing.T) {
	type User struct {
		TableName struct{} `db:"users"`
		ID        int64    `db:"id,primarykey"`
		Name      string   `db:"name"`
		Active    bool     `db:"active"`
	}
	rows := NewMockRows("id", "name", "active").
		WithRow(int64(42), "Bob", false)
	row := NewRow(rows, NewTaggedStructReflector(), testFormatter, "SELECT * FROM users", nil)

	var user User
	if err := row.Scan(&user); err != nil {
		t.Fatal(err)
	}
	if user.ID != 42 {
		t.Errorf("ID: got %d, want 42", user.ID)
	}
	if user.Name != "Bob" {
		t.Errorf("Name: got %q, want %q", user.Name, "Bob")
	}
	if user.Active {
		t.Error("Active: got true, want false")
	}
}

// scannerStruct implements sql.Scanner, so Row.Scan should NOT use struct scanning.
type scannerStruct struct {
	scanned bool
}

func (s *scannerStruct) Scan(src any) error {
	s.scanned = true
	return nil
}

func TestRow_Scan_SQLScannerStruct(t *testing.T) {
	rows := NewMockRows("val").
		WithRow("hello")
	row := NewRow(rows, NewTaggedStructReflector(), testFormatter, "SELECT val", nil)

	var dest scannerStruct
	if err := row.Scan(&dest); err != nil {
		t.Fatal(err)
	}
	if !dest.scanned {
		t.Error("expected sql.Scanner.Scan to be called")
	}
}

func TestRow_Scan_NoRows(t *testing.T) {
	rows := NewMockRows("id") // no data rows
	row := NewRow(rows, NewTaggedStructReflector(), testFormatter, "SELECT id", nil)

	var id int64
	err := row.Scan(&id)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got: %v", err)
	}
}

func TestRow_Scan_NoDestinations(t *testing.T) {
	rows := NewMockRows("id").
		WithRow(int64(1))
	row := NewRow(rows, NewTaggedStructReflector(), testFormatter, "SELECT id", nil)

	err := row.Scan()
	if err == nil {
		t.Error("expected error for no destinations")
	}
}

func TestRow_ScanValues(t *testing.T) {
	rows := NewMockRows("id", "name", "data").
		WithRow(int64(1), "Alice", []byte("raw"))
	row := NewRow(rows, NewTaggedStructReflector(), testFormatter, "SELECT id, name, data", nil)

	vals, err := row.ScanValues()
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 3 {
		t.Fatalf("got %d values, want 3", len(vals))
	}
	if vals[0] != int64(1) {
		t.Errorf("vals[0]: got %v, want 1", vals[0])
	}
	if vals[1] != "Alice" {
		t.Errorf("vals[1]: got %v, want Alice", vals[1])
	}
	b, ok := vals[2].([]byte)
	if !ok {
		t.Fatalf("vals[2]: expected []byte, got %T", vals[2])
	}
	if string(b) != "raw" {
		t.Errorf("vals[2]: got %q, want %q", b, "raw")
	}
}

func ExampleRow_ScanMap() {
	// A typical use of Row.ScanMap is to turn a single database row into a
	// JSON object keyed by column name. The mock rows below stand in for a
	// real database query result.
	createdAt := time.Date(2026, 4, 14, 9, 30, 0, 0, time.UTC)
	rows := NewMockRows("id", "name", "data", "created_at", "deleted_at").
		WithRow(int64(42), "Alice", []byte("hello"), createdAt, nil)
	row := NewRow(rows, NewTaggedStructReflector(), testFormatter, "SELECT id, name, data, created_at, deleted_at FROM users WHERE id = $1", []any{42})

	// BytesToStringScanConverter turns []byte columns into plain strings,
	// otherwise json.MarshalIndent would base64-encode them.
	// TimeToStringScanConverter formats time.Time columns with the given
	// layout instead of relying on time.Time's default JSON encoding.
	m, err := row.ScanMap(
		BytesToStringScanConverter(`\x`),
		TimeToStringScanConverter(time.DateTime),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(out))

	// Output:
	// {
	//   "created_at": "2026-04-14 09:30:00",
	//   "data": "hello",
	//   "deleted_at": null,
	//   "id": 42,
	//   "name": "Alice"
	// }
}

func TestRow_ScanMap(t *testing.T) {
	rows := NewMockRows("id", "name", "data", "missing").
		WithRow(int64(1), "Alice", []byte("raw"), nil)
	row := NewRow(rows, NewTaggedStructReflector(), testFormatter, "SELECT id, name, data, missing", nil)

	m, err := row.ScanMap()
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 4 {
		t.Fatalf("got %d entries, want 4", len(m))
	}
	if m["id"] != int64(1) {
		t.Errorf(`m["id"]: got %v, want 1`, m["id"])
	}
	if m["name"] != "Alice" {
		t.Errorf(`m["name"]: got %v, want Alice`, m["name"])
	}
	b, ok := m["data"].([]byte)
	if !ok {
		t.Fatalf(`m["data"]: expected []byte, got %T`, m["data"])
	}
	if string(b) != "raw" {
		t.Errorf(`m["data"]: got %q, want %q`, b, "raw")
	}
	if m["missing"] != nil {
		t.Errorf(`m["missing"]: got %v, want nil`, m["missing"])
	}
}

func TestRow_ScanStrings(t *testing.T) {
	rows := NewMockRows("num", "str", "null_val", "flag", "data").
		WithRow(int64(99), "hello", nil, true, []byte("bytes"))
	row := NewRow(rows, NewTaggedStructReflector(), testFormatter, "SELECT num, str, null_val, flag, data", nil)

	vals, err := row.ScanStrings()
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"99", "hello", "", "true", "bytes"}
	if len(vals) != len(expected) {
		t.Fatalf("got %d values, want %d", len(vals), len(expected))
	}
	for i := range expected {
		if vals[i] != expected[i] {
			t.Errorf("vals[%d]: got %q, want %q", i, vals[i], expected[i])
		}
	}
}
