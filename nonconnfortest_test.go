package sqldb

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNonConnForTest_Config(t *testing.T) {
	// given
	conn := NonConnForTest(t)

	// when
	cfg := conn.Config()

	// then
	require.NotNil(t, cfg)
	assert.Equal(t, "nonConnForTest", cfg.Driver)
}

func TestNonConnForTest_DefaultIsolationLevel(t *testing.T) {
	// given
	conn := NonConnForTest(t)

	// when / then
	assert.Equal(t, sql.LevelDefault, conn.DefaultIsolationLevel())
}

func TestNonConnForTest_Transaction_NotInTx(t *testing.T) {
	// given
	conn := NonConnForTest(t)

	// when
	tx := conn.Transaction()

	// then – the base conn is not in a transaction
	assert.False(t, tx.Active())
	assert.Equal(t, uint64(0), tx.ID)
}

func TestNonConnForTest_Commit_NotInTx(t *testing.T) {
	// given
	conn := NonConnForTest(t)

	// when
	err := conn.Commit()

	// then
	assert.ErrorIs(t, err, ErrNotWithinTransaction)
}

func TestNonConnForTest_Rollback_NotInTx(t *testing.T) {
	// given
	conn := NonConnForTest(t)

	// when
	err := conn.Rollback()

	// then
	assert.ErrorIs(t, err, ErrNotWithinTransaction)
}

func TestNonConnForTest_Begin_ZeroIDError(t *testing.T) {
	// given
	conn := NonConnForTest(t)

	// when
	_, err := conn.Begin(t.Context(), 0, nil)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transaction ID must not be zero")
}

func TestNonConnForTest_Begin_ProducesTransaction(t *testing.T) {
	// given
	conn := NonConnForTest(t)

	// when
	tx, err := conn.Begin(t.Context(), 42, nil)

	// then
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.True(t, tx.Transaction().Active())
	assert.Equal(t, uint64(42), tx.Transaction().ID)
}

func TestNonConnForTest_Begin_AlreadyInTxError(t *testing.T) {
	// given
	conn := NonConnForTest(t)
	tx, err := conn.Begin(t.Context(), 1, nil)
	require.NoError(t, err)

	// when – begin again on the already-started transaction
	_, err = tx.Begin(t.Context(), 2, nil)

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWithinTransaction)
}

func TestNonConnForTest_Transaction_InTx(t *testing.T) {
	// given
	conn := NonConnForTest(t)
	txOpts := &sql.TxOptions{ReadOnly: true}
	tx, err := conn.Begin(t.Context(), 99, txOpts)
	require.NoError(t, err)

	// when
	state := tx.Transaction()

	// then
	assert.True(t, state.Active())
	assert.Equal(t, uint64(99), state.ID)
	assert.Equal(t, txOpts, state.Opts)
}

func TestNonConnForTest_Commit_InTx(t *testing.T) {
	// given
	conn := NonConnForTest(t)
	tx, err := conn.Begin(t.Context(), 1, nil)
	require.NoError(t, err)

	// when
	err = tx.Commit()

	// then – commit inside a simulated transaction succeeds
	require.NoError(t, err)
}

func TestNonConnForTest_Rollback_InTx(t *testing.T) {
	// given
	conn := NonConnForTest(t)
	tx, err := conn.Begin(t.Context(), 1, nil)
	require.NoError(t, err)

	// when
	err = tx.Rollback()

	// then – rollback inside a simulated transaction succeeds
	require.NoError(t, err)
}

func TestNonConnForTest_Close(t *testing.T) {
	// given
	conn := NonConnForTest(t)

	// when / then
	assert.NoError(t, conn.Close())
}
