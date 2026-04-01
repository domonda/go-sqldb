package sqldb

import (
	"reflect"
	"strings"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// TestReflectedStructCache
// ---------------------------------------------------------------------------

func TestReflectedStructCache(t *testing.T) {
	reflector := NewTaggedStructReflector()
	typ := reflect.TypeFor[reflectTestStruct]()

	// First call populates cache
	cols1, err := ReflectStructColumns(typ, reflector)
	if err != nil {
		t.Fatal(err)
	}

	// Clear cache
	ClearQueryCaches()

	// Second call repopulates cache, should produce identical results
	cols2, err := ReflectStructColumns(typ, reflector)
	if err != nil {
		t.Fatal(err)
	}

	if len(cols1) != len(cols2) {
		t.Fatalf("column count mismatch after cache clear: %d vs %d", len(cols1), len(cols2))
	}
	for i := range cols1 {
		if cols1[i] != cols2[i] {
			t.Errorf("column %d mismatch after cache clear: %+v vs %+v", i, cols1[i], cols2[i])
		}
	}
}

func TestReflectedStructCacheEmbedded(t *testing.T) {
	reflector := NewTaggedStructReflector()
	typ := reflect.TypeFor[reflectTestEmbedded]()

	// First call
	cols1, err := ReflectStructColumns(typ, reflector)
	if err != nil {
		t.Fatal(err)
	}

	// Clear and re-call
	ClearQueryCaches()
	cols2, err := ReflectStructColumns(typ, reflector)
	if err != nil {
		t.Fatal(err)
	}

	if len(cols1) != len(cols2) {
		t.Fatalf("embedded column count mismatch: %d vs %d", len(cols1), len(cols2))
	}
	for i := range cols1 {
		if cols1[i] != cols2[i] {
			t.Errorf("embedded column %d mismatch: %+v vs %+v", i, cols1[i], cols2[i])
		}
	}
}

// ---------------------------------------------------------------------------
// TestReflectedStructCacheConcurrency
// ---------------------------------------------------------------------------

func TestReflectedStructCacheConcurrency(t *testing.T) {
	reflector := NewTaggedStructReflector()
	types := []reflect.Type{
		reflect.TypeFor[reflectTestStruct](),
		reflect.TypeFor[reflectTestComposite](),
		reflect.TypeFor[reflectTestEmbedded](),
		reflect.TypeFor[reflectTestWithOptions](),
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for _, typ := range types {
				_, _ = ReflectStructColumns(typ, reflector)
				_, _ = PrimaryKeyColumnsOfStruct(reflector, typ)
			}
		}()
	}
	wg.Wait()
}

func TestReflectedStructCacheConcurrencyWithClear(t *testing.T) {
	reflector := NewTaggedStructReflector()
	typ := reflect.TypeFor[reflectTestStruct]()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines + 1)

	// Concurrent readers
	for range goroutines {
		go func() {
			defer wg.Done()
			for range 100 {
				cols, err := ReflectStructColumns(typ, reflector)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if len(cols) != 3 {
					t.Errorf("expected 3 columns, got %d", len(cols))
				}
			}
		}()
	}

	// Concurrent clearer
	go func() {
		defer wg.Done()
		for range 20 {
			ClearQueryCaches()
		}
	}()

	wg.Wait()
}

// ---------------------------------------------------------------------------
// TestGetReflectedStruct
// ---------------------------------------------------------------------------

func TestGetReflectedStruct(t *testing.T) {
	reflector := NewTaggedStructReflector()

	t.Run("flat struct", func(t *testing.T) {
		rs, err := reflectStruct(reflector, reflect.TypeFor[reflectTestStruct]())
		if err != nil {
			t.Fatal(err)
		}
		if len(rs.Fields) != 3 {
			t.Fatalf("expected 3 fields, got %d", len(rs.Fields))
		}
		wantNames := []string{"id", "name", "active"}
		for i, want := range wantNames {
			if rs.Fields[i].Column.Name != want {
				t.Errorf("field %d name = %q, want %q", i, rs.Fields[i].Column.Name, want)
			}
		}
		// Verify ColumnIndex
		for _, f := range rs.Fields {
			idx, ok := rs.ColumnIndex[f.Column.Name]
			if !ok {
				t.Errorf("column %q not in ColumnIndex", f.Column.Name)
			}
			if rs.Fields[idx].Column.Name != f.Column.Name {
				t.Errorf("ColumnIndex[%q] = %d, but Fields[%d].Column.Name = %q", f.Column.Name, idx, idx, rs.Fields[idx].Column.Name)
			}
		}
	})

	t.Run("embedded struct", func(t *testing.T) {
		rs, err := reflectStruct(reflector, reflect.TypeFor[reflectTestEmbedded]())
		if err != nil {
			t.Fatal(err)
		}
		if len(rs.Fields) != 3 {
			t.Fatalf("expected 3 fields, got %d", len(rs.Fields))
		}
		wantNames := []string{"id", "deep_val", "emb_val"}
		for i, want := range wantNames {
			if rs.Fields[i].Column.Name != want {
				t.Errorf("field %d name = %q, want %q", i, rs.Fields[i].Column.Name, want)
			}
		}
		// deep_val should have multi-level FieldIndex
		if len(rs.Fields[1].FieldIndex) < 2 {
			t.Errorf("deep_val FieldIndex %v should have at least 2 levels", rs.Fields[1].FieldIndex)
		}
	})

	t.Run("composite PK", func(t *testing.T) {
		rs, err := reflectStruct(reflector, reflect.TypeFor[reflectTestComposite]())
		if err != nil {
			t.Fatal(err)
		}
		pkCount := 0
		for _, f := range rs.Fields {
			if f.Column.PrimaryKey {
				pkCount++
			}
		}
		if pkCount != 2 {
			t.Errorf("expected 2 PK fields, got %d", pkCount)
		}
	})

	t.Run("with options flags", func(t *testing.T) {
		rs, err := reflectStruct(reflector, reflect.TypeFor[reflectTestWithOptions]())
		if err != nil {
			t.Fatal(err)
		}
		var readOnlyCount, defaultCount int
		for _, f := range rs.Fields {
			if f.Column.ReadOnly {
				readOnlyCount++
			}
			if f.Column.HasDefault {
				defaultCount++
			}
		}
		if readOnlyCount != 1 {
			t.Errorf("expected 1 ReadOnly field, got %d", readOnlyCount)
		}
		if defaultCount != 1 {
			t.Errorf("expected 1 HasDefault field, got %d", defaultCount)
		}
	})
}

// ---------------------------------------------------------------------------
// TestFieldByIndex consistency
// ---------------------------------------------------------------------------

func TestFieldByIndexConsistency(t *testing.T) {
	reflector := NewTaggedStructReflector()
	s := reflectTestEmbedded{
		ID:              10,
		reflectEmbedded: reflectEmbedded{EmbVal: 42, reflectDeepEmbedded: reflectDeepEmbedded{DeepVal: "deep"}},
	}
	v := reflect.ValueOf(s)
	rs, err := reflectStruct(reflector, v.Type())
	if err != nil {
		t.Fatal(err)
	}

	wantValues := []any{int64(10), "deep", 42}
	for i, f := range rs.Fields {
		got := v.FieldByIndex(f.FieldIndex).Interface()
		if got != wantValues[i] {
			t.Errorf("field %q: FieldByIndex(%v) = %v, want %v", f.Column.Name, f.FieldIndex, got, wantValues[i])
		}
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkReflectStructColumns(b *testing.B) {
	reflector := NewTaggedStructReflector()
	typ := reflect.TypeFor[reflectTestStruct]()

	b.Run("flat", func(b *testing.B) {
		for range b.N {
			_, _ = ReflectStructColumns(typ, reflector)
		}
	})
	b.Run("embedded", func(b *testing.B) {
		typ := reflect.TypeFor[reflectTestEmbedded]()
		for range b.N {
			_, _ = ReflectStructColumns(typ, reflector)
		}
	})
}

func BenchmarkReflectStructColumnsAndValues(b *testing.B) {
	reflector := NewTaggedStructReflector()

	b.Run("flat", func(b *testing.B) {
		s := reflectTestStruct{ID: 1, Name: "Alice", Active: true}
		v := reflect.ValueOf(s)
		for range b.N {
			_, _, _ = ReflectStructColumnsAndValues(v, reflector)
		}
	})
	b.Run("embedded", func(b *testing.B) {
		s := reflectTestEmbedded{
			ID:              10,
			reflectEmbedded: reflectEmbedded{EmbVal: 42, reflectDeepEmbedded: reflectDeepEmbedded{DeepVal: "deep"}},
		}
		v := reflect.ValueOf(s)
		for range b.N {
			_, _, _ = ReflectStructColumnsAndValues(v, reflector)
		}
	})
}

func BenchmarkColumnPointers(b *testing.B) {
	reflector := NewTaggedStructReflector()

	b.Run("flat", func(b *testing.B) {
		s := reflectTestStruct{ID: 1, Name: "Test", Active: true}
		v := reflect.ValueOf(&s).Elem()
		columns := []string{"id", "name", "active"}
		for range b.N {
			_, _ = reflector.ColumnPointers(v, columns)
		}
	})
	b.Run("embedded", func(b *testing.B) {
		s := reflectTestEmbedded{
			ID:              5,
			reflectEmbedded: reflectEmbedded{EmbVal: 77, reflectDeepEmbedded: reflectDeepEmbedded{DeepVal: "d"}},
		}
		v := reflect.ValueOf(&s).Elem()
		columns := []string{"id", "emb_val", "deep_val"}
		for range b.N {
			_, _ = reflector.ColumnPointers(v, columns)
		}
	})
}

func BenchmarkPrimaryKeyColumnsOfStruct(b *testing.B) {
	reflector := NewTaggedStructReflector()

	b.Run("single_pk", func(b *testing.B) {
		typ := reflect.TypeFor[reflectTestStruct]()
		for range b.N {
			_, _ = PrimaryKeyColumnsOfStruct(reflector, typ)
		}
	})
	b.Run("composite_pk", func(b *testing.B) {
		typ := reflect.TypeFor[reflectTestComposite]()
		for range b.N {
			_, _ = PrimaryKeyColumnsOfStruct(reflector, typ)
		}
	})
	b.Run("embedded", func(b *testing.B) {
		typ := reflect.TypeFor[reflectTestEmbedded]()
		for range b.N {
			_, _ = PrimaryKeyColumnsOfStruct(reflector, typ)
		}
	})
}

func BenchmarkReflectStructColumnsFieldIndicesAndValues(b *testing.B) {
	reflector := NewTaggedStructReflector()

	b.Run("flat", func(b *testing.B) {
		s := reflectTestStruct{ID: 5, Name: "Charlie", Active: true}
		v := reflect.ValueOf(s)
		for range b.N {
			_, _, _, _ = ReflectStructColumnsFieldIndicesAndValues(v, reflector)
		}
	})
	b.Run("embedded", func(b *testing.B) {
		s := reflectTestEmbedded{
			ID:              10,
			reflectEmbedded: reflectEmbedded{EmbVal: 42, reflectDeepEmbedded: reflectDeepEmbedded{DeepVal: "deep"}},
		}
		v := reflect.ValueOf(s)
		for range b.N {
			_, _, _, _ = ReflectStructColumnsFieldIndicesAndValues(v, reflector)
		}
	})
}

func BenchmarkReflectStructValues(b *testing.B) {
	reflector := NewTaggedStructReflector()

	b.Run("flat", func(b *testing.B) {
		s := reflectTestStruct{ID: 7, Name: "Dana", Active: false}
		v := reflect.ValueOf(s)
		for range b.N {
			_, _ = ReflectStructValues(v, reflector)
		}
	})
	b.Run("embedded", func(b *testing.B) {
		s := reflectTestEmbedded{
			ID:              1,
			reflectEmbedded: reflectEmbedded{EmbVal: 99, reflectDeepEmbedded: reflectDeepEmbedded{DeepVal: "x"}},
		}
		v := reflect.ValueOf(s)
		for range b.N {
			_, _ = ReflectStructValues(v, reflector)
		}
	})
}

// ---------------------------------------------------------------------------
// Duplicate column error tests
// ---------------------------------------------------------------------------

type reflectDuplicateColumnStruct struct {
	TableName `db:"dup_table"`

	Name1 string `db:"name"`
	Name2 string `db:"name"`
}

type reflectDuplicateEmbeddedInner struct {
	Val string `db:"val"`
}

type reflectDuplicateEmbeddedStruct struct {
	TableName `db:"dup_emb_table"`

	Val string `db:"val"`
	reflectDuplicateEmbeddedInner
}

func TestReflectStructDuplicateColumnError(t *testing.T) {
	reflector := NewTaggedStructReflector()

	t.Run("duplicate column in flat struct", func(t *testing.T) {
		_, err := reflectStruct(reflector, reflect.TypeFor[reflectDuplicateColumnStruct]())
		if err == nil {
			t.Fatal("expected error for duplicate column")
		}
		if !strings.Contains(err.Error(), `duplicate column "name"`) {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("duplicate column via embedding", func(t *testing.T) {
		_, err := reflectStruct(reflector, reflect.TypeFor[reflectDuplicateEmbeddedStruct]())
		if err == nil {
			t.Fatal("expected error for duplicate column via embedding")
		}
		if !strings.Contains(err.Error(), `duplicate column "val"`) {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestReflectStructDuplicateColumnErrorPropagation(t *testing.T) {
	reflector := NewTaggedStructReflector()
	dupType := reflect.TypeFor[reflectDuplicateColumnStruct]()
	dupVal := reflect.ValueOf(reflectDuplicateColumnStruct{Name1: "a", Name2: "b"})

	t.Run("PrimaryKeyColumnsOfStruct", func(t *testing.T) {
		ClearQueryCaches()
		_, err := PrimaryKeyColumnsOfStruct(reflector, dupType)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ReflectStructColumns", func(t *testing.T) {
		ClearQueryCaches()
		_, err := ReflectStructColumns(dupType, reflector)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ReflectStructColumnsAndValues", func(t *testing.T) {
		ClearQueryCaches()
		_, _, err := ReflectStructColumnsAndValues(dupVal, reflector)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ReflectStructColumnsFieldIndicesAndValues", func(t *testing.T) {
		ClearQueryCaches()
		_, _, _, err := ReflectStructColumnsFieldIndicesAndValues(dupVal, reflector)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ReflectStructValues", func(t *testing.T) {
		ClearQueryCaches()
		_, err := ReflectStructValues(dupVal, reflector)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ReflectStructColumnsAndFields", func(t *testing.T) {
		ClearQueryCaches()
		_, _, err := ReflectStructColumnsAndFields(dupVal, reflector)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ColumnPointers", func(t *testing.T) {
		ClearQueryCaches()
		s := reflectDuplicateColumnStruct{Name1: "a", Name2: "b"}
		v := reflect.ValueOf(&s).Elem()
		_, err := reflector.ColumnPointers(v, []string{"name"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
