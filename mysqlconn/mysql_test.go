package mysqlconn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateColumnName(t *testing.T) {
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
		assert.NoErrorf(t, validateColumnName(name), "expected %q to be valid", name)
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
		assert.Errorf(t, validateColumnName(name), "expected %q to be invalid", name)
	}
}
