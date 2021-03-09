package sqldb

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

func TestStructFieldTagNaming_StructFieldName(t *testing.T) {
	naming := StructFieldTagNaming{
		NameTag:          "db",
		IgnoreName:       "-",
		UntaggedNameFunc: ToSnakeCase,
	}
	type AnonymousEmbedded struct{}
	var s struct {
		Index          int    `db:"index,pk"`
		Str            string `db:"named_str"`
		ReadOnly       bool   `db:"read_only,readonly"`
		UntaggedField  bool
		Ignore         bool `db:"-"`
		PKReadOnly     int  `db:"pk_read_only,pk,readonly"`
		NoFlag         bool `db:"no_flag,"`
		MalformedFlags bool `db:"malformed_flags,x, ,-,readonly,y,  "`
		AnonymousEmbedded
	}
	st := reflect.TypeOf(s)

	tests := []struct {
		name        string
		structField reflect.StructField
		wantName    string
		wantFlags   FieldFlag
		wantOk      bool
	}{
		{name: "index", structField: st.Field(0), wantName: "index", wantFlags: FieldFlagPrimaryKey, wantOk: true},
		{name: "named_str", structField: st.Field(1), wantName: "named_str", wantFlags: 0, wantOk: true},
		{name: "read_only", structField: st.Field(2), wantName: "read_only", wantFlags: FieldFlagReadOnly, wantOk: true},
		{name: "untagged_field", structField: st.Field(3), wantName: "untagged_field", wantFlags: 0, wantOk: true},
		{name: "ignore", structField: st.Field(4), wantName: "", wantFlags: 0, wantOk: false},
		{name: "pk_read_only", structField: st.Field(5), wantName: "pk_read_only", wantFlags: FieldFlagPrimaryKey + FieldFlagReadOnly, wantOk: true},
		{name: "no_flag", structField: st.Field(6), wantName: "no_flag", wantFlags: 0, wantOk: true},
		{name: "malformed_flags", structField: st.Field(7), wantName: "malformed_flags", wantFlags: FieldFlagReadOnly, wantOk: true},
		{name: "Embedded", structField: st.Field(8), wantName: "", wantFlags: 0, wantOk: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotFlags, gotOk := naming.StructFieldName(tt.structField)
			if gotName != tt.wantName {
				t.Errorf("StructFieldTagNaming.StructFieldName() gotName = %v, want %v", gotName, tt.wantName)
			}
			if gotFlags != tt.wantFlags {
				t.Errorf("StructFieldTagNaming.StructFieldName() gotFlags = %v, want %v", gotFlags, tt.wantFlags)
			}
			if gotOk != tt.wantOk {
				t.Errorf("StructFieldTagNaming.StructFieldName() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}
