package sqldb

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMockRows(t *testing.T) {
	t.Run("creates with columns", func(t *testing.T) {
		rows := NewMockRows("id", "name")
		cols, err := rows.Columns()
		require.NoError(t, err)
		assert.Equal(t, []string{"id", "name"}, cols)
	})

	t.Run("panics with no columns", func(t *testing.T) {
		assert.Panics(t, func() { NewMockRows() })
	})

	t.Run("panics with empty column name", func(t *testing.T) {
		assert.Panics(t, func() { NewMockRows("id", "") })
	})
}

func TestNewMockRowsValue(t *testing.T) {
	rows := NewMockRowsValue("count", int64(42))
	require.True(t, rows.Next())
	var count int64
	require.NoError(t, rows.Scan(&count))
	assert.Equal(t, int64(42), count)
	assert.False(t, rows.Next())

	t.Run("panics with empty column", func(t *testing.T) {
		assert.Panics(t, func() { NewMockRowsValue("", int64(1)) })
	})
}

func TestNewMockRowsValueNull(t *testing.T) {
	rows := NewMockRowsValueNull("val")
	require.True(t, rows.Next())
	var val any
	require.NoError(t, rows.Scan(&val))
	assert.Nil(t, val)

	t.Run("panics with empty column", func(t *testing.T) {
		assert.Panics(t, func() { NewMockRowsValueNull("") })
	})
}

func TestMockRows_Iteration(t *testing.T) {
	t.Run("iterate multiple rows", func(t *testing.T) {
		rows := NewMockRows("name").
			WithRow("Alice").
			WithRow("Bob")

		var names []string
		for rows.Next() {
			var name string
			require.NoError(t, rows.Scan(&name))
			names = append(names, name)
		}
		assert.NoError(t, rows.Err())
		assert.Equal(t, []string{"Alice", "Bob"}, names)
	})

	t.Run("WithRows appends multiple", func(t *testing.T) {
		rows := NewMockRows("id").
			WithRows([][]driver.Value{{int64(1)}, {int64(2)}, {int64(3)}})

		count := 0
		for rows.Next() {
			var id int64
			require.NoError(t, rows.Scan(&id))
			count++
		}
		assert.Equal(t, 3, count)
	})

	t.Run("scan before next errors", func(t *testing.T) {
		rows := NewMockRows("id").WithRow(int64(1))
		var id int64
		err := rows.Scan(&id)
		assert.Error(t, err)
	})

	t.Run("close stops iteration", func(t *testing.T) {
		rows := NewMockRows("id").WithRow(int64(1))
		require.NoError(t, rows.Close())
		assert.False(t, rows.Next())
	})

	t.Run("scan after close errors", func(t *testing.T) {
		rows := NewMockRows("id").WithRow(int64(1))
		require.True(t, rows.Next())
		require.NoError(t, rows.Close())
		var id int64
		assert.Error(t, rows.Scan(&id))
	})
}

func TestMockRows_ScanArgCountMismatch(t *testing.T) {
	rows := NewMockRows("a", "b").WithRow("x", "y")
	require.True(t, rows.Next())
	var a string
	err := rows.Scan(&a) // only 1 dest for 2 columns
	assert.Error(t, err)
}
