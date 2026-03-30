package conntest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func runBasicTests(t *testing.T, config Config) {
	t.Run("Config", func(t *testing.T) {
		// given
		conn := config.NewConn(t)

		// when
		cfg := conn.Config()

		// then
		require.NotNil(t, cfg)
		assert.Equal(t, config.DriverName, cfg.Driver)
		assert.Equal(t, config.DatabaseName, cfg.Database)
	})

	t.Run("Ping", func(t *testing.T) {
		// given
		conn := config.NewConn(t)

		// when
		err := conn.Ping(t.Context(), 5*time.Second)

		// then
		assert.NoError(t, err)
	})

	t.Run("Stats", func(t *testing.T) {
		// given
		conn := config.NewConn(t)

		// when / then — should not panic
		_ = conn.Stats()
	})

	t.Run("DefaultIsolationLevel", func(t *testing.T) {
		// given
		conn := config.NewConn(t)

		// when
		level := conn.DefaultIsolationLevel()

		// then
		assert.Equal(t, config.DefaultIsolationLevel, level)
	})

	t.Run("SelectOne", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()

		// when
		val, err := sqldb.QueryRowAs[int](ctx, conn, refl, conn, config.selectOneQuery())

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, val)
	})
}
