package sqldb

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestNewTaggedStructReflector(t *testing.T) {
	r := NewTaggedStructReflector()
	assert.Equal(t, "db", r.NameTag)
	assert.Equal(t, "-", r.Ignore)
	assert.Equal(t, "primarykey", r.PrimaryKey)
	assert.Equal(t, "readonly", r.ReadOnly)
	assert.Equal(t, "default", r.Default)
	assert.NotNil(t, r.UntaggedNameFunc)
	assert.Equal(t, "", r.UntaggedNameFunc("AnyField"), "default UntaggedNameFunc should be IgnoreStructField")
	assert.False(t, r.FailOnUnmappedColumns)
	assert.False(t, r.FailOnUnmappedStructFields)
}

func TestTaggedStructReflector_String(t *testing.T) {
	r := NewTaggedStructReflector()
	assert.Equal(t, `NameTag: "db"`, r.String())

	r.NameTag = "col"
	assert.Equal(t, `NameTag: "col"`, r.String())
}

func TestIgnoreStructField(t *testing.T) {
	assert.Equal(t, "", IgnoreStructField(""))
	assert.Equal(t, "", IgnoreStructField("AnyFieldName"))
	assert.Equal(t, "", IgnoreStructField("hello_world"))
}

func TestTaggedStructReflector_TableNameForStruct(t *testing.T) {
	r := NewTaggedStructReflector()

	t.Run("struct with TableName", func(t *testing.T) {
		table, err := r.TableNameForStruct(reflect.TypeFor[reflectTestStruct]())
		require.NoError(t, err)
		assert.Equal(t, "test_table", table)
	})

	t.Run("struct without TableName", func(t *testing.T) {
		_, err := r.TableNameForStruct(reflect.TypeFor[reflectEmbedded]())
		require.Error(t, err)
	})

	t.Run("custom NameTag", func(t *testing.T) {
		type customTagged struct {
			TableName `col:"custom_table"`
		}
		custom := &TaggedStructReflector{
			NameTag:          "col",
			Ignore:           "-",
			PrimaryKey:       "pk",
			ReadOnly:         "readonly",
			Default:          "default",
			UntaggedNameFunc: IgnoreStructField,
		}
		table, err := custom.TableNameForStruct(reflect.TypeFor[customTagged]())
		require.NoError(t, err)
		assert.Equal(t, "custom_table", table)
	})
}

func TestTaggedStructReflector_MapStructField(t *testing.T) {
	naming := &TaggedStructReflector{
		NameTag:          "db",
		Ignore:           "-",
		PrimaryKey:       "pk",
		ReadOnly:         "readonly",
		Default:          "default",
		UntaggedNameFunc: ToSnakeCase,
	}
	type AnonymousEmbedded struct{}
	st := reflect.TypeFor[struct {
		Index          int    "db:\"index,pk\""
		IndexB         int    "db:\"index_b,pk\""
		Str            string "db:\"named_str\""
		ReadOnly       bool   "db:\"read_only,readonly\""
		UntaggedField  bool
		Ignore         bool "db:\"-\""
		PKReadOnly     int  "db:\"pk_read_only,pk,readonly\""
		NoFlag         bool "db:\"no_flag,\""
		MalformedFlags bool "db:\"malformed_flags,x, ,-,readonly,y,  \""
		AnonymousEmbedded
	}]()

	tests := []struct {
		name        string
		structField reflect.StructField
		wantColumn  ColumnInfo
		wantOk      bool
	}{
		{name: "index", structField: st.Field(0), wantColumn: ColumnInfo{Name: "index", Type: "int", PrimaryKey: true}, wantOk: true},
		{name: "index_b", structField: st.Field(1), wantColumn: ColumnInfo{Name: "index_b", Type: "int", PrimaryKey: true}, wantOk: true},
		{name: "named_str", structField: st.Field(2), wantColumn: ColumnInfo{Name: "named_str", Type: "string"}, wantOk: true},
		{name: "read_only", structField: st.Field(3), wantColumn: ColumnInfo{Name: "read_only", Type: "bool", ReadOnly: true}, wantOk: true},
		{name: "untagged_field", structField: st.Field(4), wantColumn: ColumnInfo{Name: "untagged_field", Type: "bool"}, wantOk: true},
		{name: "ignore", structField: st.Field(5), wantColumn: ColumnInfo{}, wantOk: false},
		{name: "pk_read_only", structField: st.Field(6), wantColumn: ColumnInfo{Name: "pk_read_only", Type: "int", PrimaryKey: true, ReadOnly: true}, wantOk: true},
		{name: "no_flag", structField: st.Field(7), wantColumn: ColumnInfo{Name: "no_flag", Type: "bool"}, wantOk: true},
		{name: "malformed_flags", structField: st.Field(8), wantColumn: ColumnInfo{Name: "malformed_flags", Type: "bool", ReadOnly: true}, wantOk: true},
		{name: "Embedded", structField: st.Field(9), wantColumn: ColumnInfo{}, wantOk: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotColumn, gotOk := naming.MapStructField(tt.structField)
			require.Equal(t, tt.wantColumn, gotColumn, "MapStructField(%#v)", tt.structField.Name)
			require.Equal(t, tt.wantOk, gotOk, "MapStructField(%#v)", tt.structField.Name)
		})
	}
}

func TestTaggedStructReflector_MapStructField_EmbeddedIgnored(t *testing.T) {
	r := NewTaggedStructReflector()

	type Ignored struct{}
	st := reflect.TypeFor[struct {
		Ignored `db:"-"`
	}]()
	_, use := r.MapStructField(st.Field(0))
	assert.False(t, use, "embedded struct tagged db:\"-\" should be ignored")
}

func TestTaggedStructReflector_MapStructField_DefaultFlag(t *testing.T) {
	r := NewTaggedStructReflector()

	st := reflect.TypeFor[struct {
		Col string `db:"col,default"`
	}]()
	col, use := r.MapStructField(st.Field(0))
	require.True(t, use)
	assert.Equal(t, "col", col.Name)
	assert.True(t, col.HasDefault)
	assert.False(t, col.PrimaryKey)
	assert.False(t, col.ReadOnly)
}

func TestTaggedStructReflector_MapStructField_AllFlags(t *testing.T) {
	r := NewTaggedStructReflector()

	st := reflect.TypeFor[struct {
		Col string `db:"col,primarykey,readonly,default"`
	}]()
	col, use := r.MapStructField(st.Field(0))
	require.True(t, use)
	assert.Equal(t, "col", col.Name)
	assert.Equal(t, "string", col.Type)
	assert.True(t, col.PrimaryKey)
	assert.True(t, col.ReadOnly)
	assert.True(t, col.HasDefault)
}

// MyID exists at package scope so reflect.Type.String() reports its
// fully-qualified name and not "struct{...}.MyID".
type myID int64

func TestTaggedStructReflector_MapStructField_TypeReflection(t *testing.T) {
	r := NewTaggedStructReflector()

	st := reflect.TypeFor[struct {
		Plain   string    `db:"plain"`
		Ptr     *string   `db:"ptr"`
		Slice   []byte    `db:"slice"`
		Time    time.Time `db:"created_at"`
		PtrTime *time.Time `db:"updated_at"`
		Named   myID      `db:"id"`
	}]()

	tests := []struct {
		fieldIndex int
		wantType   string
	}{
		{0, "string"},
		{1, "*string"},
		{2, "[]uint8"},
		{3, "time.Time"},
		{4, "*time.Time"},
		{5, "sqldb.myID"},
	}
	for _, tt := range tests {
		t.Run(st.Field(tt.fieldIndex).Name, func(t *testing.T) {
			col, use := r.MapStructField(st.Field(tt.fieldIndex))
			require.True(t, use)
			assert.Equal(t, tt.wantType, col.Type,
				"Type should be the reflect.Type.String() of the Go field type")
		})
	}
}

func TestTaggedStructReflector_MapStructField_UntaggedWithToSnakeCase(t *testing.T) {
	r := &TaggedStructReflector{
		NameTag:          "db",
		Ignore:           "-",
		PrimaryKey:       "primarykey",
		ReadOnly:         "readonly",
		Default:          "default",
		UntaggedNameFunc: ToSnakeCase,
	}

	st := reflect.TypeFor[struct {
		MyFieldName string
	}]()
	col, use := r.MapStructField(st.Field(0))
	require.True(t, use)
	assert.Equal(t, "my_field_name", col.Name)
}

func TestTaggedStructReflector_MapStructField_UnexportedField(t *testing.T) {
	r := NewTaggedStructReflector()

	st := reflect.TypeFor[struct {
		exported   int //nolint:unused
		unexported int //nolint:unused
	}]()
	// Both fields are unexported and not anonymous
	for i := range st.NumField() {
		_, use := r.MapStructField(st.Field(i))
		assert.False(t, use, "unexported non-embedded field %q should not be used", st.Field(i).Name)
	}
}

func TestTaggedStructReflector_ScanableStructFieldsForColumns_FailOnUnmappedColumns(t *testing.T) {
	failReflector := &TaggedStructReflector{
		NameTag:               "db",
		Ignore:                "-",
		PrimaryKey:            "primarykey",
		ReadOnly:              "readonly",
		Default:               "default",
		UntaggedNameFunc:      IgnoreStructField,
		FailOnUnmappedColumns: true,
	}

	t.Run("all columns mapped succeeds", func(t *testing.T) {
		s := reflectTestStruct{ID: 1, Name: "Test", Active: true}
		v := reflect.ValueOf(&s).Elem()
		scanable, err := failReflector.ScanableStructFieldsForColumns(v, []string{"id", "name", "active"})
		require.NoError(t, err)
		assert.Len(t, scanable, 3)
	})

	t.Run("single unmapped column errors", func(t *testing.T) {
		s := reflectTestStruct{}
		v := reflect.ValueOf(&s).Elem()
		_, err := failReflector.ScanableStructFieldsForColumns(v, []string{"id", "unknown"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("multiple unmapped columns lists all in error", func(t *testing.T) {
		s := reflectTestStruct{}
		v := reflect.ValueOf(&s).Elem()
		_, err := failReflector.ScanableStructFieldsForColumns(v, []string{"id", "foo", "bar"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "foo")
		assert.Contains(t, err.Error(), "bar")
		// The mapped column "id" should not appear in the error
		assert.NotContains(t, err.Error(), "column=id")
	})

	t.Run("subset of struct columns succeeds", func(t *testing.T) {
		// Query returns fewer columns than struct has fields — should work fine
		s := reflectTestStruct{ID: 42, Name: "Alice", Active: true}
		v := reflect.ValueOf(&s).Elem()
		scanable, err := failReflector.ScanableStructFieldsForColumns(v, []string{"id"})
		require.NoError(t, err)
		require.Len(t, scanable, 1)
		idPtr, ok := scanable[0].(*int64)
		require.True(t, ok)
		assert.Equal(t, int64(42), *idPtr)
	})
}

func TestTaggedStructReflector_ScanableStructFieldsForColumns_FewerColumnsThanFields(t *testing.T) {
	r := NewTaggedStructReflector()

	t.Run("single column from multi-field struct", func(t *testing.T) {
		s := reflectTestStruct{ID: 10, Name: "Bob", Active: true}
		v := reflect.ValueOf(&s).Elem()
		scanable, err := r.ScanableStructFieldsForColumns(v, []string{"name"})
		require.NoError(t, err)
		require.Len(t, scanable, 1)
		namePtr, ok := scanable[0].(*string)
		require.True(t, ok)
		assert.Equal(t, "Bob", *namePtr)
	})

	t.Run("subset columns from embedded struct", func(t *testing.T) {
		s := reflectTestEmbedded{
			ID:              5,
			reflectEmbedded: reflectEmbedded{EmbVal: 77, reflectDeepEmbedded: reflectDeepEmbedded{DeepVal: "deep"}},
		}
		v := reflect.ValueOf(&s).Elem()
		scanable, err := r.ScanableStructFieldsForColumns(v, []string{"deep_val"})
		require.NoError(t, err)
		require.Len(t, scanable, 1)
		deepPtr, ok := scanable[0].(*string)
		require.True(t, ok)
		assert.Equal(t, "deep", *deepPtr)
	})

	t.Run("unscanned fields left unchanged", func(t *testing.T) {
		s := reflectTestStruct{ID: 99, Name: "", Active: true}
		v := reflect.ValueOf(&s).Elem()
		scanable, err := r.ScanableStructFieldsForColumns(v, []string{"name"})
		require.NoError(t, err)
		// Mutate through the scanable value as Scan would
		*(scanable[0].(*string)) = "scanned"
		assert.Equal(t, "scanned", s.Name)
		// Fields not in the column list retain their pre-existing values
		assert.Equal(t, int64(99), s.ID)
		assert.Equal(t, true, s.Active)
	})
}

func TestTaggedStructReflector_ScanableStructFieldsForColumns_EmptyColumns(t *testing.T) {
	r := NewTaggedStructReflector()
	s := reflectTestStruct{}
	v := reflect.ValueOf(&s).Elem()
	_, err := r.ScanableStructFieldsForColumns(v, []string{})
	require.Error(t, err, "empty columns slice should return error")
}

func TestTaggedStructReflector_ScanableStructFieldsForColumns_ColumnOrder(t *testing.T) {
	r := NewTaggedStructReflector()

	// Columns in different order than struct field declaration
	s := reflectTestStruct{ID: 1, Name: "Test", Active: true}
	v := reflect.ValueOf(&s).Elem()
	scanable, err := r.ScanableStructFieldsForColumns(v, []string{"active", "id", "name"})
	require.NoError(t, err)
	require.Len(t, scanable, 3)

	// Scanable values should match column order, not struct field order
	activePtr, ok := scanable[0].(*bool)
	require.True(t, ok)
	assert.Equal(t, true, *activePtr)

	idPtr, ok := scanable[1].(*int64)
	require.True(t, ok)
	assert.Equal(t, int64(1), *idPtr)

	namePtr, ok := scanable[2].(*string)
	require.True(t, ok)
	assert.Equal(t, "Test", *namePtr)
}

func TestTaggedStructReflector_ScanableStructFieldsForColumns_FailOnUnmappedStructFields(t *testing.T) {
	failReflector := &TaggedStructReflector{
		NameTag:                    "db",
		Ignore:                     "-",
		PrimaryKey:                 "primarykey",
		ReadOnly:                   "readonly",
		Default:                    "default",
		UntaggedNameFunc:           IgnoreStructField,
		FailOnUnmappedStructFields: true,
	}

	t.Run("all struct fields covered succeeds", func(t *testing.T) {
		s := reflectTestStruct{ID: 1, Name: "Test", Active: true}
		v := reflect.ValueOf(&s).Elem()
		scanable, err := failReflector.ScanableStructFieldsForColumns(v, []string{"id", "name", "active"})
		require.NoError(t, err)
		assert.Len(t, scanable, 3)
	})

	t.Run("single unmapped struct field errors", func(t *testing.T) {
		s := reflectTestStruct{}
		v := reflect.ValueOf(&s).Elem()
		// Query omits "active"
		_, err := failReflector.ScanableStructFieldsForColumns(v, []string{"id", "name"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "active")
	})

	t.Run("multiple unmapped struct fields lists all in error", func(t *testing.T) {
		s := reflectTestStruct{}
		v := reflect.ValueOf(&s).Elem()
		// Query only has "id", missing "name" and "active"
		_, err := failReflector.ScanableStructFieldsForColumns(v, []string{"id"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name")
		assert.Contains(t, err.Error(), "active")
		// The covered field "id" should not appear in the error
		assert.NotContains(t, err.Error(), "id")
	})

	t.Run("extra query columns don't trigger error", func(t *testing.T) {
		// All struct fields are covered, extra query column is fine for this flag
		s := reflectTestStruct{ID: 1, Name: "Test", Active: true}
		v := reflect.ValueOf(&s).Elem()
		scanable, err := failReflector.ScanableStructFieldsForColumns(v, []string{"id", "name", "active", "extra"})
		require.NoError(t, err)
		assert.Len(t, scanable, 4)
		// The extra column should get a discard destination (FailOnUnmappedColumns is false)
		assert.NotNil(t, scanable[3])
	})

	t.Run("both flags enabled unmapped column errors first", func(t *testing.T) {
		bothReflector := &TaggedStructReflector{
			NameTag:                    "db",
			Ignore:                     "-",
			PrimaryKey:                 "primarykey",
			ReadOnly:                   "readonly",
			Default:                    "default",
			UntaggedNameFunc:           IgnoreStructField,
			FailOnUnmappedColumns:      true,
			FailOnUnmappedStructFields: true,
		}
		s := reflectTestStruct{}
		v := reflect.ValueOf(&s).Elem()
		// "extra" is unmapped column, "active" is unmapped struct field
		_, err := bothReflector.ScanableStructFieldsForColumns(v, []string{"id", "name", "extra"})
		require.Error(t, err)
		// FailOnUnmappedColumns check runs first
		assert.Contains(t, err.Error(), "extra")
	})

	t.Run("both flags enabled all matched succeeds", func(t *testing.T) {
		bothReflector := &TaggedStructReflector{
			NameTag:                    "db",
			Ignore:                     "-",
			PrimaryKey:                 "primarykey",
			ReadOnly:                   "readonly",
			Default:                    "default",
			UntaggedNameFunc:           IgnoreStructField,
			FailOnUnmappedColumns:      true,
			FailOnUnmappedStructFields: true,
		}
		s := reflectTestStruct{ID: 1, Name: "Test", Active: true}
		v := reflect.ValueOf(&s).Elem()
		scanable, err := bothReflector.ScanableStructFieldsForColumns(v, []string{"id", "name", "active"})
		require.NoError(t, err)
		assert.Len(t, scanable, 3)
	})
}

func TestTaggedStructReflector_PrimaryKeyColumnsOfStruct(t *testing.T) {
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
			got, err := reflectTestReflector.PrimaryKeyColumnsOfStruct(tt.typ)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaggedStructReflector_ReflectStructColumnsAndValues(t *testing.T) {
	t.Run("flat struct", func(t *testing.T) {
		s := reflectTestStruct{ID: 1, Name: "Alice", Active: true, Ignored: 99}
		cols, vals, err := reflectTestReflector.ReflectStructColumnsAndValues(reflect.ValueOf(s))
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
		cols, vals, err := reflectTestReflector.ReflectStructColumnsAndValues(reflect.ValueOf(s))
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
		cols, vals, err := reflectTestReflector.ReflectStructColumnsAndValues(reflect.ValueOf(s), IgnoreColumns("active"))
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

func TestTaggedStructReflector_ReflectStructColumnsFieldIndicesAndValues(t *testing.T) {
	t.Run("flat struct", func(t *testing.T) {
		s := reflectTestStruct{ID: 5, Name: "Charlie", Active: true}
		cols, indices, vals, err := reflectTestReflector.ReflectStructColumnsFieldIndicesAndValues(reflect.ValueOf(s))
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
		cols, indices, vals, err := reflectTestReflector.ReflectStructColumnsFieldIndicesAndValues(reflect.ValueOf(s))
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

func TestTaggedStructReflector_ReflectStructValues(t *testing.T) {
	t.Run("returns only values", func(t *testing.T) {
		s := reflectTestStruct{ID: 7, Name: "Dana", Active: false, Ignored: 100}
		vals, err := reflectTestReflector.ReflectStructValues(reflect.ValueOf(s))
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
		vals, err := reflectTestReflector.ReflectStructValues(reflect.ValueOf(s))
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
		vals, err := reflectTestReflector.ReflectStructValues(reflect.ValueOf(s), IgnoreReadOnly)
		if err != nil {
			t.Fatal(err)
		}
		// Should skip readonly column, leaving: id, name, has_def
		if len(vals) != 3 {
			t.Fatalf("got %d values, want 3", len(vals))
		}
	})
}

func TestTaggedStructReflector_ReflectStructColumns(t *testing.T) {
	t.Run("flat struct", func(t *testing.T) {
		cols, err := reflectTestReflector.ReflectStructColumns(reflect.TypeFor[reflectTestStruct]())
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
		cols, err := reflectTestReflector.ReflectStructColumns(reflect.TypeFor[reflectTestEmbedded]())
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
		cols, err := reflectTestReflector.ReflectStructColumns(reflect.TypeFor[reflectTestStruct](), OnlyColumns("id", "name"))
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
		cols, err := reflectTestReflector.ReflectStructColumns(reflect.TypeFor[reflectTestStruct]())
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

func TestTaggedStructReflector_ReflectStructColumnsAndFields(t *testing.T) {
	t.Run("flat struct", func(t *testing.T) {
		s := reflectTestStruct{ID: 1, Name: "Alice", Active: true}
		cols, fields, err := reflectTestReflector.ReflectStructColumnsAndFields(reflect.ValueOf(s))
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
		cols, fields, err := reflectTestReflector.ReflectStructColumnsAndFields(reflect.ValueOf(s), IgnoreColumns("active"))
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
		cols, fields, err := reflectTestReflector.ReflectStructColumnsAndFields(reflect.ValueOf(s))
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

func TestTaggedStructReflector_ScanableStructFieldsForColumns_Basic(t *testing.T) {
	t.Run("flat struct", func(t *testing.T) {
		s := reflectTestStruct{ID: 1, Name: "Test", Active: true}
		v := reflect.ValueOf(&s).Elem()
		scanable, err := reflectTestReflector.ScanableStructFieldsForColumns(v, []string{"id", "name", "active"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(scanable) != 3 {
			t.Fatalf("got %d scanable values, want 3", len(scanable))
		}
		// Verify scanable values point to the actual struct fields
		idPtr, ok := scanable[0].(*int64)
		if !ok {
			t.Fatalf("scanable[0] is %T, want *int64", scanable[0])
		}
		if *idPtr != 1 {
			t.Errorf("*idPtr = %d, want 1", *idPtr)
		}
		namePtr, ok := scanable[1].(*string)
		if !ok {
			t.Fatalf("scanable[1] is %T, want *string", scanable[1])
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
		scanable, err := reflectTestReflector.ScanableStructFieldsForColumns(v, []string{"id", "emb_val", "deep_val"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(scanable) != 3 {
			t.Fatalf("got %d scanable values, want 3", len(scanable))
		}
		embPtr, ok := scanable[1].(*int)
		if !ok {
			t.Fatalf("scanable[1] is %T, want *int", scanable[1])
		}
		if *embPtr != 77 {
			t.Errorf("*embPtr = %d, want 77", *embPtr)
		}
	})

	t.Run("no columns error", func(t *testing.T) {
		s := reflectTestStruct{}
		v := reflect.ValueOf(&s).Elem()
		_, err := reflectTestReflector.ScanableStructFieldsForColumns(v, nil)
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
		_, err := reflector.ScanableStructFieldsForColumns(v, []string{"nonexistent"})
		if err == nil {
			t.Error("expected error for unmapped column with FailOnUnmappedColumns")
		}
	})

	t.Run("unmapped column ignored", func(t *testing.T) {
		s := reflectTestStruct{ID: 42}
		v := reflect.ValueOf(&s).Elem()
		scanable, err := reflectTestReflector.ScanableStructFieldsForColumns(v, []string{"id", "nonexistent"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(scanable) != 2 {
			t.Fatalf("got %d scanable values, want 2", len(scanable))
		}
		// First value should be the mapped struct field
		idPtr, ok := scanable[0].(*int64)
		if !ok {
			t.Fatalf("scanable[0] is %T, want *int64", scanable[0])
		}
		if *idPtr != 42 {
			t.Errorf("*idPtr = %d, want 42", *idPtr)
		}
		// Second value should be a non-nil discard destination
		if scanable[1] == nil {
			t.Error("scanable[1] is nil, want non-nil discard value")
		}
	})

	t.Run("mutation through pointer", func(t *testing.T) {
		s := reflectTestStruct{ID: 0, Name: ""}
		v := reflect.ValueOf(&s).Elem()
		scanable, err := reflectTestReflector.ScanableStructFieldsForColumns(v, []string{"id", "name"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Mutate through the returned scanable values
		*(scanable[0].(*int64)) = 999
		*(scanable[1].(*string)) = "mutated"
		if s.ID != 999 {
			t.Errorf("s.ID = %d, want 999", s.ID)
		}
		if s.Name != "mutated" {
			t.Errorf("s.Name = %q, want %q", s.Name, "mutated")
		}
	})
}
