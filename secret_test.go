package sqldb

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecret(t *testing.T) {
	s := KeepSecret("hunter2")

	t.Run("Secret returns wrapped value", func(t *testing.T) {
		assert.Equal(t, "hunter2", s.Secret())
	})

	t.Run("String is redacted", func(t *testing.T) {
		assert.Equal(t, "***REDACTED***", s.String())
	})

	t.Run("Value passes through wrapped value", func(t *testing.T) {
		v, err := s.Value()
		require.NoError(t, err)
		assert.Equal(t, "hunter2", v)
	})

	t.Run("Value delegates to wrapped driver.Valuer", func(t *testing.T) {
		v, err := KeepSecret(driver.Valuer(AnyValue{Val: int64(42)})).Value()
		require.NoError(t, err)
		assert.Equal(t, int64(42), v)
	})

	t.Run("Scan passes through to wrapped value", func(t *testing.T) {
		var dst string
		require.NoError(t, KeepSecret(&dst).Scan("scanned"))
		assert.Equal(t, "scanned", dst)
	})

	t.Run("FormatValue redacts the secret", func(t *testing.T) {
		got, err := FormatValue(s)
		require.NoError(t, err)
		assert.Equal(t, "'***REDACTED***'", got)
	})

	t.Run("SubstitutePlaceholders redacts the secret", func(t *testing.T) {
		query, err := SubstitutePlaceholders(
			NewQueryFormatter("$"),
			/*sql*/ `SELECT * FROM users WHERE name = $1 AND password = $2`,
			[]any{"erik", KeepSecret("hunter2")},
		)
		require.NoError(t, err)
		assert.Equal(t, `SELECT * FROM users WHERE name = 'erik' AND password = '***REDACTED***'`, query)
	})
}
