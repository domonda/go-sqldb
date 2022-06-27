package reflection

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestTaggedStructFieldMapping_StructFieldName(t *testing.T) {
	naming := &TaggedStructFieldMapping{
		NameTag:          "db",
		Ignore:           "-",
		PrimaryKey:       "pk",
		ReadOnly:         "readonly",
		Default:          "default",
		UntaggedNameFunc: ToSnakeCase,
	}
	type AnonymousEmbedded struct{}
	var s struct {
		Index             int    `db:"index,pk=public.my_table"` // Field(0)
		IndexB            int    `db:"index_b,pk"`               // Field(1)
		Str               string `db:"named_str"`                // Field(2)
		ReadOnly          bool   `db:"read_only,readonly"`       // Field(3)
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
		wantTable   string
		wantColumn  string
		wantFlags   StructFieldFlags
		wantOk      bool
	}{
		{name: "index", structField: st.Field(0), wantTable: "public.my_table", wantColumn: "index", wantFlags: FlagPrimaryKey, wantOk: true},
		{name: "index_b", structField: st.Field(1), wantTable: "", wantColumn: "index_b", wantFlags: FlagPrimaryKey, wantOk: true},
		{name: "named_str", structField: st.Field(2), wantColumn: "named_str", wantFlags: 0, wantOk: true},
		{name: "read_only", structField: st.Field(3), wantColumn: "read_only", wantFlags: FlagReadOnly, wantOk: true},
		{name: "untagged_field", structField: st.Field(4), wantColumn: "untagged_field", wantFlags: 0, wantOk: true},
		{name: "ignore", structField: st.Field(5), wantColumn: "", wantFlags: 0, wantOk: false},
		{name: "pk_read_only", structField: st.Field(6), wantColumn: "pk_read_only", wantFlags: FlagPrimaryKey | FlagReadOnly, wantOk: true},
		{name: "no_flag", structField: st.Field(7), wantColumn: "no_flag", wantFlags: 0, wantOk: true},
		{name: "malformed_flags", structField: st.Field(8), wantColumn: "malformed_flags", wantFlags: FlagReadOnly, wantOk: true},
		{name: "Embedded", structField: st.Field(9), wantColumn: "", wantFlags: 0, wantOk: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTable, gotColumn, gotFlags, gotOk := naming.MapStructField(tt.structField)
			if gotTable != tt.wantTable {
				t.Errorf("TaggedStructFieldMapping.MapStructField(%q) gotTable = %q, want %q", tt.structField.Name, gotTable, tt.wantTable)
			}
			if gotColumn != tt.wantColumn {
				t.Errorf("TaggedStructFieldMapping.MapStructField(%q) gotColumn = %q, want %q", tt.structField.Name, gotColumn, tt.wantColumn)
			}
			if gotFlags != tt.wantFlags {
				t.Errorf("TaggedStructFieldMapping.MapStructField(%q) gotFlags = %v, want %v", tt.structField.Name, gotFlags, tt.wantFlags)
			}
			if gotOk != tt.wantOk {
				t.Errorf("TaggedStructFieldMapping.MapStructField(%q) gotOk = %v, want %v", tt.structField.Name, gotOk, tt.wantOk)
			}
		})
	}
}
