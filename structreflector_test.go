package sqldb

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSnakeCase(t *testing.T) {
	for _, scenario := range []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "underscore", input: "_", expected: "_"},
		{name: "space", input: " ", expected: "_"},
		{name: "two spaces", input: "  ", expected: "__"},
		{name: "tab x newline", input: "\tX\n", expected: "_x_"},
		{name: "already snake case", input: "already_snake_case", expected: "already_snake_case"},
		{name: "already snake case surrounded", input: "_already_snake_case_", expected: "_already_snake_case_"},
		{name: "HelloWorld", input: "HelloWorld", expected: "hello_world"},
		{name: "Hello World", input: "Hello World", expected: "hello_world"},
		{name: "Hello-World", input: "Hello-World", expected: "hello_world"},
		{name: "symbols around", input: "*Hello+World*", expected: "_hello_world_"},
		{name: "Hello.World", input: "Hello.World", expected: "hello_world"},
		{name: "Hello/World", input: "Hello/World", expected: "hello_world"},
		{name: "parentheses and bang", input: "(Hello World!)", expected: "_hello_world__"},
		{name: "consecutive uppercase", input: "DocumentID", expected: "document_id"},
		{name: "all uppercase prefix", input: "HTMLHandler", expected: "htmlhandler"},
		{name: "non-ASCII lower", input: "Straßenadresse", expected: "straßenadresse"},
		{name: "non-ASCII with upper", input: "もしもしWorld", expected: "もしもし_world"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			actual := ToSnakeCase(scenario.input)

			// then
			assert.Equal(t, scenario.expected, actual)
		})
	}
}

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
		{name: "index", structField: st.Field(0), wantColumn: ColumnInfo{Name: "index", PrimaryKey: true}, wantOk: true},
		{name: "index_b", structField: st.Field(1), wantColumn: ColumnInfo{Name: "index_b", PrimaryKey: true}, wantOk: true},
		{name: "named_str", structField: st.Field(2), wantColumn: ColumnInfo{Name: "named_str"}, wantOk: true},
		{name: "read_only", structField: st.Field(3), wantColumn: ColumnInfo{Name: "read_only", ReadOnly: true}, wantOk: true},
		{name: "untagged_field", structField: st.Field(4), wantColumn: ColumnInfo{Name: "untagged_field"}, wantOk: true},
		{name: "ignore", structField: st.Field(5), wantColumn: ColumnInfo{}, wantOk: false},
		{name: "pk_read_only", structField: st.Field(6), wantColumn: ColumnInfo{Name: "pk_read_only", PrimaryKey: true, ReadOnly: true}, wantOk: true},
		{name: "no_flag", structField: st.Field(7), wantColumn: ColumnInfo{Name: "no_flag"}, wantOk: true},
		{name: "malformed_flags", structField: st.Field(8), wantColumn: ColumnInfo{Name: "malformed_flags", ReadOnly: true}, wantOk: true},
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
	assert.True(t, col.PrimaryKey)
	assert.True(t, col.ReadOnly)
	assert.True(t, col.HasDefault)
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

func TestTaggedStructReflector_ColumnPointers_FailOnUnmappedColumns(t *testing.T) {
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
		ptrs, err := failReflector.ColumnPointers(v, []string{"id", "name", "active"})
		require.NoError(t, err)
		assert.Len(t, ptrs, 3)
	})

	t.Run("single unmapped column errors", func(t *testing.T) {
		s := reflectTestStruct{}
		v := reflect.ValueOf(&s).Elem()
		_, err := failReflector.ColumnPointers(v, []string{"id", "unknown"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("multiple unmapped columns lists all in error", func(t *testing.T) {
		s := reflectTestStruct{}
		v := reflect.ValueOf(&s).Elem()
		_, err := failReflector.ColumnPointers(v, []string{"id", "foo", "bar"})
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
		ptrs, err := failReflector.ColumnPointers(v, []string{"id"})
		require.NoError(t, err)
		require.Len(t, ptrs, 1)
		idPtr, ok := ptrs[0].(*int64)
		require.True(t, ok)
		assert.Equal(t, int64(42), *idPtr)
	})
}

func TestTaggedStructReflector_ColumnPointers_FewerColumnsThanFields(t *testing.T) {
	r := NewTaggedStructReflector()

	t.Run("single column from multi-field struct", func(t *testing.T) {
		s := reflectTestStruct{ID: 10, Name: "Bob", Active: true}
		v := reflect.ValueOf(&s).Elem()
		ptrs, err := r.ColumnPointers(v, []string{"name"})
		require.NoError(t, err)
		require.Len(t, ptrs, 1)
		namePtr, ok := ptrs[0].(*string)
		require.True(t, ok)
		assert.Equal(t, "Bob", *namePtr)
	})

	t.Run("subset columns from embedded struct", func(t *testing.T) {
		s := reflectTestEmbedded{
			ID:              5,
			reflectEmbedded: reflectEmbedded{EmbVal: 77, reflectDeepEmbedded: reflectDeepEmbedded{DeepVal: "deep"}},
		}
		v := reflect.ValueOf(&s).Elem()
		ptrs, err := r.ColumnPointers(v, []string{"deep_val"})
		require.NoError(t, err)
		require.Len(t, ptrs, 1)
		deepPtr, ok := ptrs[0].(*string)
		require.True(t, ok)
		assert.Equal(t, "deep", *deepPtr)
	})

	t.Run("unscanned fields left unchanged", func(t *testing.T) {
		s := reflectTestStruct{ID: 99, Name: "", Active: true}
		v := reflect.ValueOf(&s).Elem()
		ptrs, err := r.ColumnPointers(v, []string{"name"})
		require.NoError(t, err)
		// Mutate through the pointer as Scan would
		*(ptrs[0].(*string)) = "scanned"
		assert.Equal(t, "scanned", s.Name)
		// Fields not in the column list retain their pre-existing values
		assert.Equal(t, int64(99), s.ID)
		assert.Equal(t, true, s.Active)
	})
}

func TestTaggedStructReflector_ColumnPointers_EmptyColumns(t *testing.T) {
	r := NewTaggedStructReflector()
	s := reflectTestStruct{}
	v := reflect.ValueOf(&s).Elem()
	_, err := r.ColumnPointers(v, []string{})
	require.Error(t, err, "empty columns slice should return error")
}

func TestTaggedStructReflector_ColumnPointers_ColumnOrder(t *testing.T) {
	r := NewTaggedStructReflector()

	// Columns in different order than struct field declaration
	s := reflectTestStruct{ID: 1, Name: "Test", Active: true}
	v := reflect.ValueOf(&s).Elem()
	ptrs, err := r.ColumnPointers(v, []string{"active", "id", "name"})
	require.NoError(t, err)
	require.Len(t, ptrs, 3)

	// Pointers should match column order, not struct field order
	activePtr, ok := ptrs[0].(*bool)
	require.True(t, ok)
	assert.Equal(t, true, *activePtr)

	idPtr, ok := ptrs[1].(*int64)
	require.True(t, ok)
	assert.Equal(t, int64(1), *idPtr)

	namePtr, ok := ptrs[2].(*string)
	require.True(t, ok)
	assert.Equal(t, "Test", *namePtr)
}
