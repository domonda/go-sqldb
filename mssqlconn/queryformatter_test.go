package mssqlconn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatTableName(t *testing.T) {
	var f QueryFormatter

	validNames := []string{
		"users",
		"_table",
		"$table",
		"Table123",
		"a",
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", // 64 chars
		"1table",
		"tbl_$123",
	}
	for _, name := range validNames {
		result, err := f.FormatTableName(name)
		require.NoErrorf(t, err, "expected %q to be valid", name)
		assert.Equal(t, name, result)
	}

	invalidNames := []string{
		"",
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", // 65 chars
		"table-name",
		"table name",
		"table.name",
		"table`name",
		"table@name",
	}
	for _, name := range invalidNames {
		_, err := f.FormatTableName(name)
		assert.Errorf(t, err, "expected %q to be invalid", name)
	}
}

func TestFormatColumnName(t *testing.T) {
	var f QueryFormatter

	validNames := []string{
		"col",
		"_col",
		"$col",
		"Col123",
		"a",
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", // 64 chars
		"1col",
		"col_$123",
	}
	for _, name := range validNames {
		result, err := f.FormatColumnName(name)
		require.NoErrorf(t, err, "expected %q to be valid", name)
		assert.Equal(t, name, result)
	}

	invalidNames := []string{
		"",
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", // 65 chars
		"col-name",
		"col name",
		"col.name",
		"col`name",
		"col@name",
	}
	for _, name := range invalidNames {
		_, err := f.FormatColumnName(name)
		assert.Errorf(t, err, "expected %q to be invalid", name)
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
