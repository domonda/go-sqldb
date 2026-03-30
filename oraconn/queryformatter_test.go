package oraconn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEscapeIdentifier(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		ident string
		want  string
	}{
		{
			name:  "lowercase without reserved words is unquoted",
			ident: "mycolumn",
			want:  "mycolumn",
		},
		{
			name:  "underscore only is unquoted",
			ident: "my_col",
			want:  "my_col",
		},
		{
			name:  "reserved word table is double-quoted",
			ident: "table",
			want:  `"table"`,
		},
		{
			name:  "reserved word select is double-quoted",
			ident: "select",
			want:  `"select"`,
		},
		{
			name:  "reserved word column is double-quoted",
			ident: "column",
			want:  `"column"`,
		},
		{
			name:  "reserved word timestamp is double-quoted",
			ident: "timestamp",
			want:  `"timestamp"`,
		},
		{
			name:  "uppercase identifier is double-quoted",
			ident: "MyCol",
			want:  `"MyCol"`,
		},
		{
			name:  "all uppercase identifier is double-quoted",
			ident: "MYCOL",
			want:  `"MYCOL"`,
		},
		{
			name:  "embedded double quotes are escaped",
			ident: `my"col`,
			want:  `"my""col"`,
		},
		{
			name:  "digits in identifier cause quoting",
			ident: "col1",
			want:  `"col1"`,
		},
		{
			name:  "leading underscore lowercase is unquoted",
			ident: "_mycol",
			want:  "_mycol",
		},
		{
			name:  "single lowercase letter is unquoted",
			ident: "x",
			want:  "x",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := EscapeIdentifier(scenario.ident)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}

func TestQueryFormatter_FormatTableName(t *testing.T) {
	var qf QueryFormatter

	for _, scenario := range []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "simple lowercase table name",
			input: "users",
			want:  "users",
		},
		{
			name:  "schema qualified table name",
			input: "myschema.users",
			want:  "myschema.users",
		},
		{
			name:  "reserved word as table name is quoted",
			input: "table",
			want:  `"table"`,
		},
		{
			name:  "reserved word as schema and table name are both quoted",
			input: "select.table",
			want:  `"select"."table"`,
		},
		{
			name:  "mixed case table name is quoted",
			input: "MyTable",
			want:  `"MyTable"`,
		},
		{
			name:  "schema with uppercase is quoted",
			input: "MySchema.mytable",
			want:  `"MySchema".mytable`,
		},
		{
			name:  "table with digits is quoted",
			input: "table1",
			want:  `"table1"`,
		},
		{
			name:  "underscore in names is valid",
			input: "my_schema.my_table",
			want:  "my_schema.my_table",
		},
		{
			name:    "empty string is invalid",
			input:   "",
			wantErr: true,
		},
		{
			name:    "starts with digit is invalid",
			input:   "1table",
			wantErr: true,
		},
		{
			name:    "contains space is invalid",
			input:   "my table",
			wantErr: true,
		},
		{
			name:    "double dot is invalid",
			input:   "schema..table",
			wantErr: true,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result, err := qf.FormatTableName(scenario.input)

			// then
			if scenario.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, scenario.want, result)
		})
	}
}

func TestQueryFormatter_FormatColumnName(t *testing.T) {
	var qf QueryFormatter

	for _, scenario := range []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "simple lowercase column name",
			input: "username",
			want:  "username",
		},
		{
			name:  "underscore column name",
			input: "first_name",
			want:  "first_name",
		},
		{
			name:  "reserved word is quoted",
			input: "column",
			want:  `"column"`,
		},
		{
			name:  "mixed case is quoted",
			input: "FirstName",
			want:  `"FirstName"`,
		},
		{
			name:  "column with digits is quoted",
			input: "col1",
			want:  `"col1"`,
		},
		{
			name:    "empty string is invalid",
			input:   "",
			wantErr: true,
		},
		{
			name:    "starts with digit is invalid",
			input:   "1col",
			wantErr: true,
		},
		{
			name:    "contains space is invalid",
			input:   "my col",
			wantErr: true,
		},
		{
			name:    "schema qualified is invalid for column name",
			input:   "schema.col",
			wantErr: true,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result, err := qf.FormatColumnName(scenario.input)

			// then
			if scenario.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, scenario.want, result)
		})
	}
}

func TestQueryFormatter_FormatPlaceholder(t *testing.T) {
	var qf QueryFormatter

	for _, scenario := range []struct {
		name       string
		paramIndex int
		want       string
	}{
		{
			name:       "index 0 returns :1",
			paramIndex: 0,
			want:       ":1",
		},
		{
			name:       "index 1 returns :2",
			paramIndex: 1,
			want:       ":2",
		},
		{
			name:       "index 9 returns :10",
			paramIndex: 9,
			want:       ":10",
		},
		{
			name:       "index 99 returns :100",
			paramIndex: 99,
			want:       ":100",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := qf.FormatPlaceholder(scenario.paramIndex)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}

func TestQueryFormatter_FormatStringLiteral(t *testing.T) {
	var qf QueryFormatter

	for _, scenario := range []struct {
		name string
		str  string
		want string
	}{
		{
			name: "simple string is single-quoted",
			str:  "hello",
			want: "'hello'",
		},
		{
			name: "empty string",
			str:  "",
			want: "''",
		},
		{
			name: "string with single quote is escaped",
			str:  "it's",
			want: "'it''s'",
		},
		{
			name: "string with multiple single quotes",
			str:  "it's a 'test'",
			want: "'it''s a ''test'''",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			result := qf.FormatStringLiteral(scenario.str)

			// then
			assert.Equal(t, scenario.want, result)
		})
	}
}

func TestQueryFormatter_MaxArgs(t *testing.T) {
	var qf QueryFormatter

	// when
	result := qf.MaxArgs()

	// then
	assert.Equal(t, 65535, result)
}
