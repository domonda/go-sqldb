package mssqlconn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEscapeIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"users", "users"},
		{"my_table", "my_table"},
		{"order", "[order]"},           // reserved word
		{"select", "[select]"},         // reserved word
		{"MyTable", "[MyTable]"},       // uppercase
		{"My Table", "[My Table]"},     // space
		{"col]name", "[col]]name]"},    // contains ]
		{"dbo", "dbo"},                 // not reserved
		{"int", "[int]"},               // reserved data type
		{"timestamp", "[timestamp]"},   // reserved data type
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, EscapeIdentifier(tt.input))
		})
	}
}

func TestFormatTableName(t *testing.T) {
	var f QueryFormatter

	tests := []struct {
		name     string
		expected string
	}{
		{"users", "users"},
		{"_table", "_table"},
		{"Table123", "[Table123]"},
		{"a", "a"},
		{"tbl_123", "tbl_123"},
		{"order", "[order]"},
		{"My Table", "[My Table]"},
		{"dbo.users", "dbo.users"},
		{"dbo.order", "dbo.[order]"},
		{"DBO.Users", "[DBO].[Users]"},
		{"My Schema.My Table", "[My Schema].[My Table]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := f.FormatTableName(tt.name)
			require.NoErrorf(t, err, "expected %q to be valid", tt.name)
			assert.Equal(t, tt.expected, result)
		})
	}

	invalidNames := []string{
		"",
		".table",    // starts with dot
		"1table",    // starts with digit
		"dbo.1col",  // part starts with digit
	}
	for _, name := range invalidNames {
		t.Run("invalid_"+name, func(t *testing.T) {
			_, err := f.FormatTableName(name)
			assert.Errorf(t, err, "expected %q to be invalid", name)
		})
	}
}

func TestFormatColumnName(t *testing.T) {
	var f QueryFormatter

	tests := []struct {
		name     string
		expected string
	}{
		{"col", "col"},
		{"_col", "_col"},
		{"Col123", "[Col123]"},
		{"a", "a"},
		{"col_123", "col_123"},
		{"order", "[order]"},
		{"My Column", "[My Column]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := f.FormatColumnName(tt.name)
			require.NoErrorf(t, err, "expected %q to be valid", tt.name)
			assert.Equal(t, tt.expected, result)
		})
	}

	invalidNames := []string{
		"",
		"1col",       // starts with digit
		"$col",       // starts with $
		"col.name",   // contains dot
	}
	for _, name := range invalidNames {
		t.Run("invalid_"+name, func(t *testing.T) {
			_, err := f.FormatColumnName(name)
			assert.Errorf(t, err, "expected %q to be invalid", name)
		})
	}
}

func TestFormatPlaceholder(t *testing.T) {
	var f QueryFormatter

	assert.Equal(t, "@p1", f.FormatPlaceholder(0))
	assert.Equal(t, "@p2", f.FormatPlaceholder(1))
	assert.Equal(t, "@p10", f.FormatPlaceholder(9))
	assert.Equal(t, "@p100", f.FormatPlaceholder(99))

	assert.Panics(t, func() { f.FormatPlaceholder(-1) })
}

func TestFormatStringLiteral(t *testing.T) {
	var f QueryFormatter

	assert.Equal(t, "'hello'", f.FormatStringLiteral("hello"))
	assert.Equal(t, "'it''s'", f.FormatStringLiteral("it's"))
	assert.Equal(t, "''", f.FormatStringLiteral(""))
	assert.Equal(t, "'a''b''c'", f.FormatStringLiteral("a'b'c"))
}
