package db

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestListenOnChannel(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var listenCount int
		var gotChannel string
		mock.MockListenOnChannel = func(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) error {
			listenCount++
			gotChannel = channel
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := ListenOnChannel(ctx, "my_channel", nil, nil)
		require.NoError(t, err)
		require.Equal(t, 1, listenCount, "MockListenOnChannel call count")
		require.Equal(t, "my_channel", gotChannel)
	})

	t.Run("error", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var listenCount int
		testErr := errors.New("listen failed")
		mock.MockListenOnChannel = func(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) error {
			listenCount++
			return testErr
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := ListenOnChannel(ctx, "my_channel", nil, nil)
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 1, listenCount, "MockListenOnChannel call count")
	})

	t.Run("non-listener connection returns unsupported", func(t *testing.T) {
		// NonConnForTest only implements Connection, not ListenerConnection
		conn := sqldb.NonConnForTest(t)
		config := sqldb.NewConnExt(conn, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := ListenOnChannel(ctx, "my_channel", nil, nil)
		require.ErrorIs(t, err, errors.ErrUnsupported)
	})
}

func TestUnlistenChannel(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var unlistenCount int
		var gotChannel string
		mock.MockUnlistenChannel = func(channel string) error {
			unlistenCount++
			gotChannel = channel
			return nil
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := UnlistenChannel(ctx, "my_channel")
		require.NoError(t, err)
		require.Equal(t, 1, unlistenCount, "MockUnlistenChannel call count")
		require.Equal(t, "my_channel", gotChannel)
	})

	t.Run("error", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var unlistenCount int
		testErr := errors.New("unlisten failed")
		mock.MockUnlistenChannel = func(channel string) error {
			unlistenCount++
			return testErr
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := UnlistenChannel(ctx, "my_channel")
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 1, unlistenCount, "MockUnlistenChannel call count")
	})

	t.Run("non-listener connection returns unsupported", func(t *testing.T) {
		conn := sqldb.NonConnForTest(t)
		config := sqldb.NewConnExt(conn, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		err := UnlistenChannel(ctx, "my_channel")
		require.ErrorIs(t, err, errors.ErrUnsupported)
	})
}

func TestIsListeningOnChannel(t *testing.T) {
	t.Run("listening", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var isListeningCount int
		mock.MockIsListeningOnChannel = func(channel string) bool {
			isListeningCount++
			return true
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		result := IsListeningOnChannel(ctx, "my_channel")
		require.True(t, result)
		require.Equal(t, 1, isListeningCount, "MockIsListeningOnChannel call count")
	})

	t.Run("not listening", func(t *testing.T) {
		mock := sqldb.NewMockConn("$", nil, nil)
		var isListeningCount int
		mock.MockIsListeningOnChannel = func(channel string) bool {
			isListeningCount++
			return false
		}
		config := sqldb.NewConnExt(mock, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		result := IsListeningOnChannel(ctx, "my_channel")
		require.False(t, result)
		require.Equal(t, 1, isListeningCount, "MockIsListeningOnChannel call count")
	})

	t.Run("non-listener connection returns false", func(t *testing.T) {
		conn := sqldb.NonConnForTest(t)
		config := sqldb.NewConnExt(conn, sqldb.NewTaggedStructReflector(), sqldb.NewQueryFormatter("$"), sqldb.StdQueryBuilder{})
		ctx := ContextWithConn(t.Context(), config)

		result := IsListeningOnChannel(ctx, "my_channel")
		require.False(t, result)
	})
}
