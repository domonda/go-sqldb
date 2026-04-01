package sqldb

import (
	"database/sql"
	"reflect"
	"testing"
	"time"
)

func Test_isNonSQLScannerStruct(t *testing.T) {
	tests := []struct {
		t    reflect.Type
		want bool
	}{
		// Structs that do not implement sql.Scanner
		{t: reflect.TypeFor[struct{ X int }](), want: true},

		// Structs that implement sql.Scanner
		{t: reflect.TypeFor[time.Time](), want: false},
		{t: reflect.TypeFor[sql.NullTime](), want: false},

		// Non struct types
		{t: reflect.TypeFor[int](), want: false},
		{t: reflect.TypeFor[string](), want: false},
		{t: reflect.TypeFor[[]byte](), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.t.String(), func(t *testing.T) {
			if got := isNonSQLScannerStruct(tt.t); got != tt.want {
				t.Errorf("isNonSQLScannerStruct(%s) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}

// Test struct types used across multiple tests.

type reflectDeepEmbedded struct {
	DeepVal string `db:"deep_val"`
}

type reflectEmbedded struct {
	reflectDeepEmbedded
	EmbVal int `db:"emb_val"`
}

type reflectTestStruct struct {
	TableName `db:"test_table"`

	ID      int64  `db:"id,primarykey"`
	Name    string `db:"name"`
	Active  bool   `db:"active"`
	Ignored int    `db:"-"`
	private int    //nolint:unused
}

type reflectTestComposite struct {
	TableName `db:"composite_table"`

	OrgID  int64  `db:"org_id,primarykey"`
	ItemID int64  `db:"item_id,primarykey"`
	Name   string `db:"name"`
}

type reflectTestEmbedded struct {
	TableName `db:"embed_table"`

	ID int64 `db:"id,primarykey"`
	reflectEmbedded
}

type reflectTestWithOptions struct {
	TableName `db:"options_table"`

	ID       int64  `db:"id,primarykey"`
	Name     string `db:"name"`
	ReadOnly string `db:"ro,readonly"`
	HasDef   string `db:"has_def,default"`
}

var reflectTestReflector = NewTaggedStructReflector()

// ---------------------------------------------------------------------------
// PrimaryKeyColumnsOfStruct
// ---------------------------------------------------------------------------

func TestPrimaryKeyColumnsOfStruct(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want []string
	}{
		{
			name: "single PK",
			typ:  reflect.TypeFor[reflectTestStruct](),
			want: []string{"id"},
		},
		{
			name: "composite PK",
			typ:  reflect.TypeFor[reflectTestComposite](),
			want: []string{"org_id", "item_id"},
		},
		{
			name: "no PK columns",
			typ:  reflect.TypeFor[reflectEmbedded](),
			want: nil,
		},
		{
			name: "embedded struct with PK",
			typ:  reflect.TypeFor[reflectTestEmbedded](),
			want: []string{"id"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PrimaryKeyColumnsOfStruct(reflectTestReflector, tt.typ)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ReflectStructColumnsAndValues
// ---------------------------------------------------------------------------

func TestReflectStructColumnsAndValues(t *testing.T) {
	t.Run("flat struct", func(t *testing.T) {
		s := reflectTestStruct{ID: 1, Name: "Alice", Active: true, Ignored: 99}
		cols, vals, err := ReflectStructColumnsAndValues(reflect.ValueOf(s), reflectTestReflector)
		if err != nil {
			t.Fatal(err)
		}
		wantNames := []string{"id", "name", "active"}
		if len(cols) != len(wantNames) {
			t.Fatalf("got %d columns, want %d", len(cols), len(wantNames))
		}
		for i, want := range wantNames {
			if cols[i].Name != want {
				t.Errorf("cols[%d].Name = %q, want %q", i, cols[i].Name, want)
			}
		}
		if vals[0] != int64(1) {
			t.Errorf("vals[0] = %v, want 1", vals[0])
		}
		if vals[1] != "Alice" {
			t.Errorf("vals[1] = %v, want Alice", vals[1])
		}
		if vals[2] != true {
			t.Errorf("vals[2] = %v, want true", vals[2])
		}
	})

	t.Run("embedded struct", func(t *testing.T) {
		s := reflectTestEmbedded{
			ID:              10,
			reflectEmbedded: reflectEmbedded{EmbVal: 42, reflectDeepEmbedded: reflectDeepEmbedded{DeepVal: "deep"}},
		}
		cols, vals, err := ReflectStructColumnsAndValues(reflect.ValueOf(s), reflectTestReflector)
		if err != nil {
			t.Fatal(err)
		}
		wantNames := []string{"id", "deep_val", "emb_val"}
		if len(cols) != len(wantNames) {
			t.Fatalf("got %d columns, want %d", len(cols), len(wantNames))
		}
		for i, want := range wantNames {
			if cols[i].Name != want {
				t.Errorf("cols[%d].Name = %q, want %q", i, cols[i].Name, want)
			}
		}
		if vals[0] != int64(10) {
			t.Errorf("vals[0] = %v, want 10", vals[0])
		}
		if vals[1] != "deep" {
			t.Errorf("vals[1] = %v, want deep", vals[1])
		}
		if vals[2] != 42 {
			t.Errorf("vals[2] = %v, want 42", vals[2])
		}
	})

	t.Run("with IgnoreColumns option", func(t *testing.T) {
		s := reflectTestStruct{ID: 1, Name: "Bob", Active: false}
		cols, vals, err := ReflectStructColumnsAndValues(reflect.ValueOf(s), reflectTestReflector, IgnoreColumns("active"))
		if err != nil {
			t.Fatal(err)
		}
		wantNames := []string{"id", "name"}
		if len(cols) != len(wantNames) {
			t.Fatalf("got %d columns, want %d", len(cols), len(wantNames))
		}
		for i, want := range wantNames {
			if cols[i].Name != want {
				t.Errorf("cols[%d].Name = %q, want %q", i, cols[i].Name, want)
			}
		}
		if len(vals) != 2 {
			t.Fatalf("got %d values, want 2", len(vals))
		}
	})
}

// ---------------------------------------------------------------------------
// ReflectStructColumnsFieldIndicesAndValues
// ---------------------------------------------------------------------------

func TestReflectStructColumnsFieldIndicesAndValues(t *testing.T) {
	t.Run("flat struct", func(t *testing.T) {
		s := reflectTestStruct{ID: 5, Name: "Charlie", Active: true}
		cols, indices, vals, err := ReflectStructColumnsFieldIndicesAndValues(reflect.ValueOf(s), reflectTestReflector)
		if err != nil {
			t.Fatal(err)
		}
		wantNames := []string{"id", "name", "active"}
		if len(cols) != len(wantNames) {
			t.Fatalf("got %d columns, want %d", len(cols), len(wantNames))
		}
		for i, want := range wantNames {
			if cols[i].Name != want {
				t.Errorf("cols[%d].Name = %q, want %q", i, cols[i].Name, want)
			}
		}
		if len(indices) != len(wantNames) {
			t.Fatalf("got %d indices, want %d", len(indices), len(wantNames))
		}
		if len(vals) != len(wantNames) {
			t.Fatalf("got %d values, want %d", len(vals), len(wantNames))
		}
		// Verify field indices point to correct fields
		sType := reflect.TypeFor[reflectTestStruct]()
		for i, idx := range indices {
			field := sType.FieldByIndex(idx)
			col, _ := reflectTestReflector.MapStructField(field)
			if col.Name != wantNames[i] {
				t.Errorf("index %v resolves to %q, want %q", idx, col.Name, wantNames[i])
			}
		}
	})

	t.Run("embedded struct indices", func(t *testing.T) {
		s := reflectTestEmbedded{
			ID:              10,
			reflectEmbedded: reflectEmbedded{EmbVal: 42, reflectDeepEmbedded: reflectDeepEmbedded{DeepVal: "deep"}},
		}
		cols, indices, vals, err := ReflectStructColumnsFieldIndicesAndValues(reflect.ValueOf(s), reflectTestReflector)
		if err != nil {
			t.Fatal(err)
		}
		if len(cols) != 3 {
			t.Fatalf("got %d columns, want 3", len(cols))
		}
		if len(indices) != 3 || len(vals) != 3 {
			t.Fatalf("got %d indices and %d vals, want 3 each", len(indices), len(vals))
		}
		// "deep_val" should have a multi-level index (through reflectEmbedded.reflectDeepEmbedded)
		if len(indices[1]) < 2 {
			t.Errorf("deep_val index %v should have at least 2 levels", indices[1])
		}
	})
}

// ---------------------------------------------------------------------------
// ReflectStructValues
// ---------------------------------------------------------------------------

func TestReflectStructValues(t *testing.T) {
	t.Run("returns only values", func(t *testing.T) {
		s := reflectTestStruct{ID: 7, Name: "Dana", Active: false, Ignored: 100}
		vals, err := ReflectStructValues(reflect.ValueOf(s), reflectTestReflector)
		if err != nil {
			t.Fatal(err)
		}
		if len(vals) != 3 {
			t.Fatalf("got %d values, want 3", len(vals))
		}
		if vals[0] != int64(7) {
			t.Errorf("vals[0] = %v, want 7", vals[0])
		}
		if vals[1] != "Dana" {
			t.Errorf("vals[1] = %v, want Dana", vals[1])
		}
		if vals[2] != false {
			t.Errorf("vals[2] = %v, want false", vals[2])
		}
	})

	t.Run("embedded struct values", func(t *testing.T) {
		s := reflectTestEmbedded{
			ID:              1,
			reflectEmbedded: reflectEmbedded{EmbVal: 99, reflectDeepEmbedded: reflectDeepEmbedded{DeepVal: "x"}},
		}
		vals, err := ReflectStructValues(reflect.ValueOf(s), reflectTestReflector)
		if err != nil {
			t.Fatal(err)
		}
		if len(vals) != 3 {
			t.Fatalf("got %d values, want 3", len(vals))
		}
		if vals[0] != int64(1) {
			t.Errorf("vals[0] = %v, want 1", vals[0])
		}
		if vals[1] != "x" {
			t.Errorf("vals[1] = %v, want x", vals[1])
		}
		if vals[2] != 99 {
			t.Errorf("vals[2] = %v, want 99", vals[2])
		}
	})

	t.Run("with IgnoreReadOnly option", func(t *testing.T) {
		s := reflectTestWithOptions{ID: 1, Name: "Test", ReadOnly: "ro_val", HasDef: "def_val"}
		vals, err := ReflectStructValues(reflect.ValueOf(s), reflectTestReflector, IgnoreReadOnly)
		if err != nil {
			t.Fatal(err)
		}
		// Should skip readonly column, leaving: id, name, has_def
		if len(vals) != 3 {
			t.Fatalf("got %d values, want 3", len(vals))
		}
	})
}

// ---------------------------------------------------------------------------
// ReflectStructColumns
// ---------------------------------------------------------------------------

func TestReflectStructColumns(t *testing.T) {
	t.Run("flat struct", func(t *testing.T) {
		cols, err := ReflectStructColumns(reflect.TypeFor[reflectTestStruct](), reflectTestReflector)
		if err != nil {
			t.Fatal(err)
		}
		wantNames := []string{"id", "name", "active"}
		if len(cols) != len(wantNames) {
			t.Fatalf("got %d columns, want %d", len(cols), len(wantNames))
		}
		for i, want := range wantNames {
			if cols[i].Name != want {
				t.Errorf("cols[%d].Name = %q, want %q", i, cols[i].Name, want)
			}
		}
		// Verify PK flag
		if !cols[0].PrimaryKey {
			t.Error("id column should have PrimaryKey=true")
		}
		if cols[1].PrimaryKey {
			t.Error("name column should have PrimaryKey=false")
		}
	})

	t.Run("embedded struct", func(t *testing.T) {
		cols, err := ReflectStructColumns(reflect.TypeFor[reflectTestEmbedded](), reflectTestReflector)
		if err != nil {
			t.Fatal(err)
		}
		wantNames := []string{"id", "deep_val", "emb_val"}
		if len(cols) != len(wantNames) {
			t.Fatalf("got %d columns, want %d", len(cols), len(wantNames))
		}
		for i, want := range wantNames {
			if cols[i].Name != want {
				t.Errorf("cols[%d].Name = %q, want %q", i, cols[i].Name, want)
			}
		}
	})

	t.Run("with OnlyColumns option", func(t *testing.T) {
		cols, err := ReflectStructColumns(reflect.TypeFor[reflectTestStruct](), reflectTestReflector, OnlyColumns("id", "name"))
		if err != nil {
			t.Fatal(err)
		}
		if len(cols) != 2 {
			t.Fatalf("got %d columns, want 2", len(cols))
		}
		if cols[0].Name != "id" {
			t.Errorf("cols[0].Name = %q, want %q", cols[0].Name, "id")
		}
		if cols[1].Name != "name" {
			t.Errorf("cols[1].Name = %q, want %q", cols[1].Name, "name")
		}
	})

	t.Run("ignored and private fields excluded", func(t *testing.T) {
		cols, err := ReflectStructColumns(reflect.TypeFor[reflectTestStruct](), reflectTestReflector)
		if err != nil {
			t.Fatal(err)
		}
		for _, col := range cols {
			if col.Name == "-" || col.Name == "private" || col.Name == "Ignored" {
				t.Errorf("unexpected column %q should be excluded", col.Name)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// ReflectStructColumnsAndFields
// ---------------------------------------------------------------------------

func TestReflectStructColumnsAndFields(t *testing.T) {
	t.Run("flat struct", func(t *testing.T) {
		s := reflectTestStruct{ID: 1, Name: "Alice", Active: true}
		cols, fields, err := ReflectStructColumnsAndFields(reflect.ValueOf(s), reflectTestReflector)
		if err != nil {
			t.Fatal(err)
		}
		wantNames := []string{"id", "name", "active"}
		if len(cols) != len(wantNames) {
			t.Fatalf("got %d columns, want %d", len(cols), len(wantNames))
		}
		for i, want := range wantNames {
			if cols[i].Name != want {
				t.Errorf("cols[%d].Name = %q, want %q", i, cols[i].Name, want)
			}
		}
		if len(fields) != len(wantNames) {
			t.Fatalf("got %d fields, want %d", len(fields), len(wantNames))
		}
		if fields[0] != reflect.TypeFor[int64]() {
			t.Errorf("fields[0] = %v, want int64", fields[0])
		}
		if fields[1] != reflect.TypeFor[string]() {
			t.Errorf("fields[1] = %v, want string", fields[1])
		}
		if fields[2] != reflect.TypeFor[bool]() {
			t.Errorf("fields[2] = %v, want bool", fields[2])
		}
	})

	t.Run("with IgnoreColumns option", func(t *testing.T) {
		s := reflectTestStruct{ID: 1, Name: "Bob", Active: false}
		cols, fields, err := ReflectStructColumnsAndFields(reflect.ValueOf(s), reflectTestReflector, IgnoreColumns("active"))
		if err != nil {
			t.Fatal(err)
		}
		if len(cols) != 2 || len(fields) != 2 {
			t.Fatalf("got %d cols and %d fields, want 2 each", len(cols), len(fields))
		}
		if cols[0].Name != "id" || cols[1].Name != "name" {
			t.Errorf("unexpected columns: %v, %v", cols[0].Name, cols[1].Name)
		}
	})

	t.Run("embedded struct", func(t *testing.T) {
		s := reflectTestEmbedded{
			ID:              10,
			reflectEmbedded: reflectEmbedded{EmbVal: 42, reflectDeepEmbedded: reflectDeepEmbedded{DeepVal: "deep"}},
		}
		cols, fields, err := ReflectStructColumnsAndFields(reflect.ValueOf(s), reflectTestReflector)
		if err != nil {
			t.Fatal(err)
		}
		if len(cols) != 3 || len(fields) != 3 {
			t.Fatalf("got %d cols and %d fields, want 3 each", len(cols), len(fields))
		}
		// deep_val should be a string field
		if fields[1] != reflect.TypeFor[string]() {
			t.Errorf("fields[1] = %v, want string for deep_val", fields[1])
		}
	})
}

// ---------------------------------------------------------------------------
// TaggedStructReflector.ColumnPointers
// ---------------------------------------------------------------------------

func TestTaggedStructReflector_ColumnPointers(t *testing.T) {
	t.Run("flat struct", func(t *testing.T) {
		s := reflectTestStruct{ID: 1, Name: "Test", Active: true}
		v := reflect.ValueOf(&s).Elem()
		ptrs, err := reflectTestReflector.ColumnPointers(v, []string{"id", "name", "active"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ptrs) != 3 {
			t.Fatalf("got %d pointers, want 3", len(ptrs))
		}
		// Verify pointers point to the actual struct fields
		idPtr, ok := ptrs[0].(*int64)
		if !ok {
			t.Fatalf("ptrs[0] is %T, want *int64", ptrs[0])
		}
		if *idPtr != 1 {
			t.Errorf("*idPtr = %d, want 1", *idPtr)
		}
		namePtr, ok := ptrs[1].(*string)
		if !ok {
			t.Fatalf("ptrs[1] is %T, want *string", ptrs[1])
		}
		if *namePtr != "Test" {
			t.Errorf("*namePtr = %q, want %q", *namePtr, "Test")
		}
	})

	t.Run("embedded struct", func(t *testing.T) {
		s := reflectTestEmbedded{
			ID:              5,
			reflectEmbedded: reflectEmbedded{EmbVal: 77, reflectDeepEmbedded: reflectDeepEmbedded{DeepVal: "d"}},
		}
		v := reflect.ValueOf(&s).Elem()
		ptrs, err := reflectTestReflector.ColumnPointers(v, []string{"id", "emb_val", "deep_val"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ptrs) != 3 {
			t.Fatalf("got %d pointers, want 3", len(ptrs))
		}
		embPtr, ok := ptrs[1].(*int)
		if !ok {
			t.Fatalf("ptrs[1] is %T, want *int", ptrs[1])
		}
		if *embPtr != 77 {
			t.Errorf("*embPtr = %d, want 77", *embPtr)
		}
	})

	t.Run("no columns error", func(t *testing.T) {
		s := reflectTestStruct{}
		v := reflect.ValueOf(&s).Elem()
		_, err := reflectTestReflector.ColumnPointers(v, nil)
		if err == nil {
			t.Error("expected error for no columns")
		}
	})

	t.Run("unmapped column error with FailOnUnmappedColumns", func(t *testing.T) {
		reflector := &TaggedStructReflector{
			NameTag:               "db",
			Ignore:                "-",
			PrimaryKey:            "primarykey",
			ReadOnly:              "readonly",
			Default:               "default",
			UntaggedNameFunc:      IgnoreStructField,
			FailOnUnmappedColumns: true,
		}
		s := reflectTestStruct{}
		v := reflect.ValueOf(&s).Elem()
		_, err := reflector.ColumnPointers(v, []string{"nonexistent"})
		if err == nil {
			t.Error("expected error for unmapped column with FailOnUnmappedColumns")
		}
	})

	t.Run("unmapped column ignored", func(t *testing.T) {
		s := reflectTestStruct{ID: 42}
		v := reflect.ValueOf(&s).Elem()
		ptrs, err := reflectTestReflector.ColumnPointers(v, []string{"id", "nonexistent"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ptrs) != 2 {
			t.Fatalf("got %d pointers, want 2", len(ptrs))
		}
		// First pointer should be the mapped struct field
		idPtr, ok := ptrs[0].(*int64)
		if !ok {
			t.Fatalf("ptrs[0] is %T, want *int64", ptrs[0])
		}
		if *idPtr != 42 {
			t.Errorf("*idPtr = %d, want 42", *idPtr)
		}
		// Second pointer should be a non-nil discard destination
		if ptrs[1] == nil {
			t.Error("ptrs[1] is nil, want non-nil discard pointer")
		}
	})

	t.Run("mutation through pointer", func(t *testing.T) {
		s := reflectTestStruct{ID: 0, Name: ""}
		v := reflect.ValueOf(&s).Elem()
		ptrs, err := reflectTestReflector.ColumnPointers(v, []string{"id", "name"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Mutate through the returned pointers
		*(ptrs[0].(*int64)) = 999
		*(ptrs[1].(*string)) = "mutated"
		if s.ID != 999 {
			t.Errorf("s.ID = %d, want 999", s.ID)
		}
		if s.Name != "mutated" {
			t.Errorf("s.Name = %q, want %q", s.Name, "mutated")
		}
	})
}
