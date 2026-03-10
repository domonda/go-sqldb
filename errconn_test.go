package sqldb

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleErrConn_Config() {
	errConn := NewErrConn(errors.New("this is a test error"))
	fmt.Println(errConn.Config().String())
	fmt.Println(errConn.Config().URL().String())

	// Output:
	// ErrConn
	// ErrConn://localhost
}

func TestNewErrConn_PanicsOnNil(t *testing.T) {
	assert.Panics(t, func() {
		NewErrConn(nil)
	})
}

func TestNewErrConnExt_PanicsOnNil(t *testing.T) {
	assert.Panics(t, func() {
		NewErrConnExt(nil)
	})
}

func TestErrConn_AllMethodsReturnErr(t *testing.T) {
	// given
	sentinel := errors.New("sentinel error")
	conn := NewErrConn(sentinel)

	// when / then – every method that can return an error must return the sentinel
	t.Run("Ping", func(t *testing.T) {
		err := conn.Ping(t.Context(), 0)
		require.ErrorIs(t, err, sentinel)
	})

	t.Run("Exec", func(t *testing.T) {
		err := conn.Exec(t.Context(), "SELECT 1")
		require.ErrorIs(t, err, sentinel)
	})

	t.Run("Query returns ErrRows", func(t *testing.T) {
		rows := conn.Query(t.Context(), "SELECT 1")
		require.NotNil(t, rows)
		require.ErrorIs(t, rows.Err(), sentinel)
	})

	t.Run("Prepare", func(t *testing.T) {
		_, err := conn.Prepare(t.Context(), "SELECT 1")
		require.ErrorIs(t, err, sentinel)
	})

	t.Run("Begin", func(t *testing.T) {
		_, err := conn.Begin(t.Context(), 1, nil)
		require.ErrorIs(t, err, sentinel)
	})

	t.Run("Commit", func(t *testing.T) {
		err := conn.Commit()
		require.ErrorIs(t, err, sentinel)
	})

	t.Run("Rollback", func(t *testing.T) {
		err := conn.Rollback()
		require.ErrorIs(t, err, sentinel)
	})

	t.Run("ListenOnChannel", func(t *testing.T) {
		err := conn.ListenOnChannel("ch", nil, nil)
		require.ErrorIs(t, err, sentinel)
	})

	t.Run("UnlistenChannel", func(t *testing.T) {
		err := conn.UnlistenChannel("ch")
		require.ErrorIs(t, err, sentinel)
	})
}

func TestErrConn_CloseReturnsNil(t *testing.T) {
	// given
	conn := NewErrConn(errors.New("some error"))

	// when / then – Close is intentionally a no-op (returns nil)
	assert.NoError(t, conn.Close())
}

func TestErrConn_IsListeningOnChannelReturnsFalse(t *testing.T) {
	// given
	conn := NewErrConn(errors.New("some error"))

	// when / then
	assert.False(t, conn.IsListeningOnChannel("any"))
}

func TestErrConn_StatsReturnsZero(t *testing.T) {
	// given
	conn := NewErrConn(errors.New("some error"))

	// when / then
	assert.Zero(t, conn.Stats())
}

func TestErrConn_TransactionReturnsInactive(t *testing.T) {
	// given
	conn := NewErrConn(errors.New("some error"))

	// when
	tx := conn.Transaction()

	// then
	assert.False(t, tx.Active())
}

func TestErrConn_Config(t *testing.T) {
	// given
	conn := NewErrConn(errors.New("some error"))

	// when
	cfg := conn.Config()

	// then
	require.NotNil(t, cfg)
	assert.Equal(t, "ErrConn", cfg.Driver)
}

func TestErrConn_ImplementsListenerConnection(t *testing.T) {
	// The compile-time assertion in errconn.go already verifies this,
	// but we double-check at runtime with a type assertion.
	conn := NewErrConn(errors.New("e"))
	_, ok := any(conn).(ListenerConnection)
	assert.True(t, ok, "ErrConn must implement ListenerConnection")
}

func TestNewErrConnExt_MethodsDelegateToErr(t *testing.T) {
	// given
	sentinel := errors.New("ext sentinel")
	ext := NewErrConnExt(sentinel)

	t.Run("Ping", func(t *testing.T) {
		err := ext.Ping(t.Context(), time.Second)
		require.ErrorIs(t, err, sentinel)
	})

	t.Run("Exec", func(t *testing.T) {
		err := ext.Exec(t.Context(), "SELECT 1")
		require.ErrorIs(t, err, sentinel)
	})

	t.Run("Query returns ErrRows", func(t *testing.T) {
		rows := ext.Query(t.Context(), "SELECT 1")
		require.NotNil(t, rows)
		require.ErrorIs(t, rows.Err(), sentinel)
	})
}
