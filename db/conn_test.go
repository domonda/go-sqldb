package db

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestSetConn(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})

		// Save and restore global conn
		saved := globalConn
		defer func() { globalConn = saved }()

		SetConn(config)
		require.Equal(t, mock, globalConn.Connection)
	})

	t.Run("nil panics", func(t *testing.T) {
		require.Panics(t, func() {
			SetConn(nil)
		})
	})
}

func TestConn(t *testing.T) {
	t.Run("from context", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		got := Conn(ctx)
		require.Equal(t, mock, got.Connection)
	})

	t.Run("falls back to global", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})

		// Save and restore global conn
		saved := globalConn
		defer func() { globalConn = saved }()

		SetConn(config)
		got := Conn(t.Context())
		require.Equal(t, mock, got.Connection)
	})
}

func TestContextWithConn(t *testing.T) {
	mock := sqldb.NewMockConn("$", nil, nil)
	config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
	ctx := ContextWithConn(t.Context(), config)

	got := Conn(ctx)
	require.Equal(t, mock, got.Connection)
}

func TestContextWithGlobalConn(t *testing.T) {
	mock := sqldb.NewMockConn("$", nil, nil)
	config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})

	// Save and restore global conn
	saved := globalConn
	defer func() { globalConn = saved }()

	SetConn(config)

	// ContextWithGlobalConn should embed the global conn in context
	ctx := ContextWithGlobalConn(t.Context())
	got := Conn(ctx)
	require.Equal(t, mock, got.Connection)
}

func TestClose(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var closeCount int
		mock.MockClose = func() error {
			closeCount++
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})

		// Save and restore global conn
		saved := globalConn
		defer func() { globalConn = saved }()

		SetConn(config)
		err := Close()
		require.NoError(t, err)
		require.Equal(t, 1, closeCount, "MockClose call count")
	})
}
