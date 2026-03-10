package sqldb

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenericConn_Config(t *testing.T) {
	// given
	cfg := &ConnConfig{
		Driver:   "postgres",
		Host:     "localhost",
		Database: "testdb",
	}
	conn := NewGenericConn(&sql.DB{}, cfg, sql.LevelDefault)

	// when
	got := conn.Config()

	// then
	require.NotNil(t, got)
	assert.Equal(t, "postgres", got.Driver)
	assert.Equal(t, "testdb", got.Database)
}

func TestNewGenericConn_DefaultIsolationLevel(t *testing.T) {
	for _, lvl := range []sql.IsolationLevel{
		sql.LevelDefault,
		sql.LevelReadCommitted,
		sql.LevelSerializable,
	} {
		t.Run(lvl.String(), func(t *testing.T) {
			// given
			conn := NewGenericConn(&sql.DB{}, &ConnConfig{Driver: "postgres"}, lvl)

			// when / then
			assert.Equal(t, lvl, conn.DefaultIsolationLevel())
		})
	}
}

func TestNewGenericConn_Transaction(t *testing.T) {
	// given
	conn := NewGenericConn(&sql.DB{}, &ConnConfig{Driver: "postgres"}, sql.LevelDefault)

	// when
	tx := conn.Transaction()

	// then – a non-transaction connection must report no active transaction
	assert.False(t, tx.Active())
	assert.Equal(t, uint64(0), tx.ID)
}

func TestNewGenericConn_Commit_NotWithinTransaction(t *testing.T) {
	// given
	conn := NewGenericConn(&sql.DB{}, &ConnConfig{Driver: "postgres"}, sql.LevelDefault)

	// when
	err := conn.Commit()

	// then
	assert.ErrorIs(t, err, ErrNotWithinTransaction)
}

func TestNewGenericConn_Rollback_NotWithinTransaction(t *testing.T) {
	// given
	conn := NewGenericConn(&sql.DB{}, &ConnConfig{Driver: "postgres"}, sql.LevelDefault)

	// when
	err := conn.Rollback()

	// then
	assert.ErrorIs(t, err, ErrNotWithinTransaction)
}

func TestNewGenericConn_Begin_ZeroIDError(t *testing.T) {
	// given
	conn := NewGenericConn(&sql.DB{}, &ConnConfig{Driver: "postgres"}, sql.LevelDefault)

	// when
	_, err := conn.Begin(t.Context(), 0, nil)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transaction ID must not be zero")
}
