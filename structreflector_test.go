package sqldb

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSnakeCase(t *testing.T) {
	testCases := map[string]string{
		"":                     "",
		"_":                    "_",
		"already_snake_case":   "already_snake_case",
		"_already_snake_case_": "_already_snake_case_",
		"HelloWorld":           "hello_world",
		"DocumentID":           "document_id",
		"HTMLHandler":          "htmlhandler",
		"もしもしWorld":            "もしもし_world",
	}
	for str, expected := range testCases {
		t.Run(str, func(t *testing.T) {
			actual := ToSnakeCase(str)
			assert.Equal(t, expected, actual, "snake case")
		})
	}
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
	var s struct {
		Index             int    `db:"index,pk"`           // Field(0)
		IndexB            int    `db:"index_b,pk"`         // Field(1)
		Str               string `db:"named_str"`          // Field(2)
		ReadOnly          bool   `db:"read_only,readonly"` // Field(3)
		UntaggedField     bool   // Field(4)
		Ignore            bool   `db:"-"`                                   // Field(5)
		PKReadOnly        int    `db:"pk_read_only,pk,readonly"`            // Field(6)
		NoFlag            bool   `db:"no_flag,"`                            // Field(7)
		MalformedFlags    bool   `db:"malformed_flags,x, ,-,readonly,y,  "` // Field(8)
		AnonymousEmbedded        // Field(9)
	}
	st := reflect.TypeOf(s)

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
