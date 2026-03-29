package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValues_SortedColumnsAndValues(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		// given
		v := Values{}

		// when
		columns, values := v.SortedColumnsAndValues()

		// then
		assert.Empty(t, columns)
		assert.Empty(t, values)
	})

	t.Run("single entry", func(t *testing.T) {
		// given
		v := Values{"name": "Alice"}

		// when
		columns, values := v.SortedColumnsAndValues()

		// then
		require.Len(t, columns, 1)
		assert.Equal(t, "name", columns[0].Name)
		require.Len(t, values, 1)
		assert.Equal(t, "Alice", values[0])
	})

	t.Run("multiple entries sorted by column name", func(t *testing.T) {
		// given
		v := Values{
			"zebra": 3,
			"alpha": 1,
			"mango": 2,
		}

		// when
		columns, values := v.SortedColumnsAndValues()

		// then
		require.Len(t, columns, 3)
		assert.Equal(t, "alpha", columns[0].Name)
		assert.Equal(t, "mango", columns[1].Name)
		assert.Equal(t, "zebra", columns[2].Name)

		assert.Equal(t, 1, values[0])
		assert.Equal(t, 2, values[1])
		assert.Equal(t, 3, values[2])
	})

	t.Run("nil value is preserved", func(t *testing.T) {
		// given
		v := Values{"col": nil}

		// when
		columns, values := v.SortedColumnsAndValues()

		// then
		require.Len(t, columns, 1)
		assert.Equal(t, "col", columns[0].Name)
		assert.Nil(t, values[0])
	})
}
