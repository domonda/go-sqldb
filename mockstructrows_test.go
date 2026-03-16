package sqldb

import (
	"testing"
	"time"

	"github.com/domonda/go-types/nullable"
	"github.com/domonda/go-types/uu"
)

type mockSimpleRow struct {
	Name   string  `db:"name"`
	Age    int     `db:"age"`
	Score  float64 `db:"score"`
	Active bool    `db:"active"`
}

type mockTimeRow struct {
	ID        int64     `db:"id"`
	CreatedAt time.Time `db:"created_at"`
}

type mockValuerRow struct {
	ID   uu.ID                  `db:"id"`
	Name nullable.TrimmedString `db:"name"`
}

type mockEmbeddedBase struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

type mockEmbeddedRow struct {
	mockEmbeddedBase
	Extra string `db:"extra"`
}

type mockNoColumnsRow struct {
	Unexported int
	Skipped    int `db:"-"`
}

func TestNewMockStructRows_Columns(t *testing.T) {
	rows := NewMockStructRows[mockSimpleRow](nil)
	cols, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"name", "age", "score", "active"}
	if len(cols) != len(expected) {
		t.Fatalf("expected %d columns, got %d", len(expected), len(cols))
	}
	for i, col := range cols {
		if col != expected[i] {
			t.Errorf("column %d: expected %q, got %q", i, expected[i], col)
		}
	}
}

func TestNewMockStructRows_EmbeddedStruct(t *testing.T) {
	rows := NewMockStructRows[mockEmbeddedRow](nil)
	cols, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"id", "name", "extra"}
	if len(cols) != len(expected) {
		t.Fatalf("expected %d columns, got %d", len(expected), len(cols))
	}
	for i, col := range cols {
		if col != expected[i] {
			t.Errorf("column %d: expected %q, got %q", i, expected[i], col)
		}
	}
}

func TestNewMockStructRows_ZeroRows(t *testing.T) {
	rows := NewMockStructRows[mockSimpleRow](nil)

	if rows.Next() {
		t.Fatal("Next() should return false for zero rows")
	}
	if err := rows.Err(); err != nil {
		t.Fatal("Err() should be nil")
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNewMockStructRows_ScanSimple(t *testing.T) {
	rows := NewMockStructRows(nil,
		mockSimpleRow{Name: "Alice", Age: 30, Score: 9.5, Active: true},
		mockSimpleRow{Name: "Bob", Age: 25, Score: 7.0, Active: false},
	)

	var name string
	var age int64
	var score float64
	var active bool

	// First row
	if !rows.Next() {
		t.Fatal("expected Next() to return true")
	}
	if err := rows.Scan(&name, &age, &score, &active); err != nil {
		t.Fatal(err)
	}
	if name != "Alice" || age != 30 || score != 9.5 || active != true {
		t.Errorf("row 1: got %q, %d, %f, %v", name, age, score, active)
	}

	// Second row
	if !rows.Next() {
		t.Fatal("expected Next() to return true")
	}
	if err := rows.Scan(&name, &age, &score, &active); err != nil {
		t.Fatal(err)
	}
	if name != "Bob" || age != 25 || score != 7.0 || active != false {
		t.Errorf("row 2: got %q, %d, %f, %v", name, age, score, active)
	}

	// No more rows
	if rows.Next() {
		t.Fatal("expected Next() to return false")
	}
}

func TestNewMockStructRows_ScanTime(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	rows := NewMockStructRows(nil, mockTimeRow{ID: 42, CreatedAt: now})

	if !rows.Next() {
		t.Fatal("expected Next() to return true")
	}
	var id int64
	var createdAt time.Time
	if err := rows.Scan(&id, &createdAt); err != nil {
		t.Fatal(err)
	}
	if id != 42 {
		t.Errorf("expected id 42, got %d", id)
	}
	if !createdAt.Equal(now) {
		t.Errorf("expected %v, got %v", now, createdAt)
	}
}

func TestNewMockStructRows_ScanValuer(t *testing.T) {
	id := uu.IDMust("01234567-89ab-cdef-0123-456789abcdef")
	rows := NewMockStructRows(nil, mockValuerRow{ID: id, Name: "  hello  "})

	if !rows.Next() {
		t.Fatal("expected Next() to return true")
	}
	var gotID string
	var gotName string
	if err := rows.Scan(&gotID, &gotName); err != nil {
		t.Fatal(err)
	}
	if gotID != id.String() {
		t.Errorf("expected %q, got %q", id.String(), gotID)
	}
	if gotName != "hello" {
		t.Errorf("expected %q, got %q", "hello", gotName)
	}
}

func TestNewMockStructRows_ScanEmbedded(t *testing.T) {
	rows := NewMockStructRows(nil, mockEmbeddedRow{
		mockEmbeddedBase: mockEmbeddedBase{ID: 1, Name: "test"},
		Extra:            "extra_val",
	})

	if !rows.Next() {
		t.Fatal("expected Next() to return true")
	}
	var id int64
	var name, extra string
	if err := rows.Scan(&id, &name, &extra); err != nil {
		t.Fatal(err)
	}
	if id != 1 || name != "test" || extra != "extra_val" {
		t.Errorf("got %d, %q, %q", id, name, extra)
	}
}

func TestNewMockStructRows_ScanWithoutNext(t *testing.T) {
	rows := NewMockStructRows(nil, mockSimpleRow{Name: "x"})
	var name string
	var age int64
	var score float64
	var active bool
	err := rows.Scan(&name, &age, &score, &active)
	if err == nil {
		t.Fatal("expected error from Scan without Next")
	}
}

func TestNewMockStructRows_ScanAfterClose(t *testing.T) {
	rows := NewMockStructRows(nil, mockSimpleRow{Name: "x"})
	rows.Next()
	rows.Close()
	var name string
	var age int64
	var score float64
	var active bool
	err := rows.Scan(&name, &age, &score, &active)
	if err == nil {
		t.Fatal("expected error from Scan after Close")
	}
}

func TestNewMockStructRows_ScanWrongDestCount(t *testing.T) {
	rows := NewMockStructRows(nil, mockSimpleRow{Name: "x"})
	rows.Next()
	var name string
	err := rows.Scan(&name)
	if err == nil {
		t.Fatal("expected error from Scan with wrong dest count")
	}
}

func TestNewMockStructRows_PanicNoMappedColumns(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for struct with no mapped columns")
		}
	}()
	NewMockStructRows[mockNoColumnsRow](nil)
}

func TestNewMockStructRows_PanicNonStruct(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for non-struct type")
		}
	}()
	NewMockStructRows[int](nil)
}
