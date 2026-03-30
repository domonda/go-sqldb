package conntest

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func runTransactionTests(t *testing.T, config Config) {
	t.Run("NotInTransaction", func(t *testing.T) {
		// given
		conn := config.NewConn(t)

		// when
		tx := conn.Transaction()

		// then
		assert.False(t, tx.Active())
		assert.Equal(t, uint64(0), tx.ID)
	})

	t.Run("InTransaction", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		id := sqldb.NextTransactionID()

		// when
		txConn, err := conn.Begin(ctx, id, nil)
		require.NoError(t, err)
		defer txConn.Rollback() //nolint:errcheck

		// then
		tx := txConn.Transaction()
		assert.True(t, tx.Active())
		assert.Equal(t, id, tx.ID)
	})

	t.Run("Commit", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		txConn, err := conn.Begin(ctx, sqldb.NextTransactionID(), nil)
		require.NoError(t, err)

		// when
		err = sqldb.InsertRowStruct(ctx, txConn, refl, qb, txConn, &simpleRow{ID: 1, Val: "committed"})
		require.NoError(t, err)
		err = txConn.Commit()
		require.NoError(t, err)

		// then — row visible from original conn
		got := querySimpleRow(t, conn, qb, 1)
		assert.Equal(t, "committed", got.Val)
	})

	t.Run("Rollback", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		txConn, err := conn.Begin(ctx, sqldb.NextTransactionID(), nil)
		require.NoError(t, err)

		// when
		err = sqldb.InsertRowStruct(ctx, txConn, refl, qb, txConn, &simpleRow{ID: 1, Val: "rolled-back"})
		require.NoError(t, err)
		err = txConn.Rollback()
		require.NoError(t, err)

		// then — row NOT visible from original conn
		_, err = sqldb.QueryRowByPK[simpleRow](ctx, conn, refl, qb, conn, 1)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
	})

	t.Run("CommitWithoutTransaction", func(t *testing.T) {
		// given
		conn := config.NewConn(t)

		// when
		err := conn.Commit()

		// then
		assert.ErrorIs(t, err, sqldb.ErrNotWithinTransaction)
	})

	t.Run("RollbackWithoutTransaction", func(t *testing.T) {
		// given
		conn := config.NewConn(t)

		// when
		err := conn.Rollback()

		// then
		assert.ErrorIs(t, err, sqldb.ErrNotWithinTransaction)
	})

	t.Run("BeginFromTransaction", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()

		id1 := sqldb.NextTransactionID()
		tx1, err := conn.Begin(ctx, id1, nil)
		require.NoError(t, err)
		defer tx1.Rollback() //nolint:errcheck

		// when — begin a new transaction from the existing one
		id2 := sqldb.NextTransactionID()
		tx2, err := tx1.Begin(ctx, id2, nil)
		require.NoError(t, err)
		defer tx2.Rollback() //nolint:errcheck

		// then — different transaction IDs
		assert.NotEqual(t, tx1.Transaction().ID, tx2.Transaction().ID)
		assert.Equal(t, id1, tx1.Transaction().ID)
		assert.Equal(t, id2, tx2.Transaction().ID)
	})

	t.Run("TransactionIsolation", func(t *testing.T) {
		if !config.SupportsReadOnlyTransaction && !config.SupportsCustomIsolationLevel {
			t.Skip("driver does not support custom transaction options")
		}
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		var opts *sql.TxOptions
		if config.SupportsReadOnlyTransaction {
			opts = &sql.TxOptions{ReadOnly: true}
		} else {
			opts = &sql.TxOptions{Isolation: sql.LevelReadCommitted}
		}

		// when
		txConn, err := conn.Begin(ctx, sqldb.NextTransactionID(), opts)
		require.NoError(t, err)
		defer txConn.Rollback() //nolint:errcheck

		// then
		tx := txConn.Transaction()
		require.NotNil(t, tx.Opts)
		if config.SupportsReadOnlyTransaction {
			assert.True(t, tx.Opts.ReadOnly)
		} else {
			assert.Equal(t, sql.LevelReadCommitted, tx.Opts.Isolation)
		}
	})

	t.Run("CloseRollsBack", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		txConn, err := conn.Begin(ctx, sqldb.NextTransactionID(), nil)
		require.NoError(t, err)

		err = sqldb.InsertRowStruct(ctx, txConn, refl, qb, txConn, &simpleRow{ID: 1, Val: "close-rollback"})
		require.NoError(t, err)

		// when — close the transaction connection instead of commit/rollback
		err = txConn.Close()
		require.NoError(t, err)

		// then — row NOT visible from original conn
		_, err = sqldb.QueryRowByPK[simpleRow](ctx, conn, refl, qb, conn, 1)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
	})

	t.Run("TransactionHelper", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		// when
		err := sqldb.Transaction(ctx, conn, nil, func(tx sqldb.Connection) error {
			return sqldb.InsertRowStruct(ctx, tx, refl, qb, tx, &simpleRow{ID: 1, Val: "tx-helper"})
		})

		// then
		require.NoError(t, err)
		got := querySimpleRow(t, conn, qb, 1)
		assert.Equal(t, "tx-helper", got.Val)
	})

	t.Run("TransactionHelperRollback", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		// when
		testErr := errors.New("test error")
		err := sqldb.Transaction(ctx, conn, nil, func(tx sqldb.Connection) error {
			insertErr := sqldb.InsertRowStruct(ctx, tx, refl, qb, tx, &simpleRow{ID: 1, Val: "should-rollback"})
			if insertErr != nil {
				return insertErr
			}
			return testErr
		})

		// then
		assert.ErrorIs(t, err, testErr)
		_, err = sqldb.QueryRowByPK[simpleRow](ctx, conn, refl, qb, conn, 1)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
	})

	t.Run("DoubleCommit", func(t *testing.T) {
		conn := config.NewConn(t)
		ctx := t.Context()

		txConn, err := conn.Begin(ctx, sqldb.NextTransactionID(), nil)
		require.NoError(t, err)

		err = txConn.Commit()
		require.NoError(t, err)

		// Second commit should error
		err = txConn.Commit()
		assert.Error(t, err)
	})

	t.Run("DoubleRollback", func(t *testing.T) {
		conn := config.NewConn(t)
		ctx := t.Context()

		txConn, err := conn.Begin(ctx, sqldb.NextTransactionID(), nil)
		require.NoError(t, err)

		err = txConn.Rollback()
		require.NoError(t, err)

		// Second rollback should error
		err = txConn.Rollback()
		assert.Error(t, err)
	})

	t.Run("CommitThenRollback", func(t *testing.T) {
		conn := config.NewConn(t)
		ctx := t.Context()

		txConn, err := conn.Begin(ctx, sqldb.NextTransactionID(), nil)
		require.NoError(t, err)

		err = txConn.Commit()
		require.NoError(t, err)

		// Rollback after commit should error
		err = txConn.Rollback()
		assert.Error(t, err)
	})

	t.Run("RollbackThenCommit", func(t *testing.T) {
		conn := config.NewConn(t)
		ctx := t.Context()

		txConn, err := conn.Begin(ctx, sqldb.NextTransactionID(), nil)
		require.NoError(t, err)

		err = txConn.Rollback()
		require.NoError(t, err)

		// Commit after rollback should error
		err = txConn.Commit()
		assert.Error(t, err)
	})

	t.Run("ExecAfterCommit", func(t *testing.T) {
		if !config.ExecAfterClosedTxErrors {
			t.Skip("driver does not error on exec after closed transaction")
		}
		conn := config.NewConn(t)
		ctx := t.Context()

		txConn, err := conn.Begin(ctx, sqldb.NextTransactionID(), nil)
		require.NoError(t, err)

		err = txConn.Commit()
		require.NoError(t, err)

		// Exec after commit should error
		err = txConn.Exec(ctx, config.selectOneQuery())
		assert.Error(t, err)
	})

	t.Run("ExecAfterRollback", func(t *testing.T) {
		if !config.ExecAfterClosedTxErrors {
			t.Skip("driver does not error on exec after closed transaction")
		}
		conn := config.NewConn(t)
		ctx := t.Context()

		txConn, err := conn.Begin(ctx, sqldb.NextTransactionID(), nil)
		require.NoError(t, err)

		err = txConn.Rollback()
		require.NoError(t, err)

		// Exec after rollback should error
		err = txConn.Exec(ctx, config.selectOneQuery())
		assert.Error(t, err)
	})

	t.Run("TransactionHelperPanic", func(t *testing.T) {
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		// Panic in txFunc should auto-rollback and re-panic
		assert.PanicsWithValue(t, "test panic", func() {
			_ = sqldb.Transaction(ctx, conn, nil, func(tx sqldb.Connection) error { //#nosec G104 -- error irrelevant, testing panic rollback
				err := sqldb.InsertRowStruct(ctx, tx, refl, qb, tx, &simpleRow{ID: 1, Val: "should-rollback"})
				if err != nil {
					return err
				}
				panic("test panic")
			})
		})

		// Data should not be visible after panic rollback
		_, err := sqldb.QueryRowByPK[simpleRow](ctx, conn, refl, qb, conn, 1)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
	})

	t.Run("IsolatedTransactionCommit", func(t *testing.T) {
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		err := sqldb.IsolatedTransaction(ctx, conn, nil, func(tx sqldb.Connection) error {
			return sqldb.InsertRowStruct(ctx, tx, refl, qb, tx, &simpleRow{ID: 1, Val: "isolated"})
		})
		require.NoError(t, err)

		got := querySimpleRow(t, conn, qb, 1)
		assert.Equal(t, "isolated", got.Val)
	})

	t.Run("IsolatedTransactionRollback", func(t *testing.T) {
		conn := config.NewConn(t)
		ctx := t.Context()
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		testErr := errors.New("isolated error")
		err := sqldb.IsolatedTransaction(ctx, conn, nil, func(tx sqldb.Connection) error {
			insertErr := sqldb.InsertRowStruct(ctx, tx, refl, qb, tx, &simpleRow{ID: 1, Val: "should-rollback"})
			if insertErr != nil {
				return insertErr
			}
			return testErr
		})
		assert.ErrorIs(t, err, testErr)

		_, err = sqldb.QueryRowByPK[simpleRow](ctx, conn, refl, qb, conn, 1)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
	})

	t.Run("ReadOnlyTransactionRejectsWrite", func(t *testing.T) {
		if !config.SupportsReadOnlyTransaction {
			t.Skip("driver does not enforce read-only transactions")
		}
		conn := config.NewConn(t)
		ctx := t.Context()
		setupTable(t, conn, config.DDL.CreateSimpleTable, "conntest_simple")

		txConn, err := conn.Begin(ctx, sqldb.NextTransactionID(), &sql.TxOptions{ReadOnly: true})
		require.NoError(t, err)
		defer txConn.Rollback() //nolint:errcheck

		// INSERT inside read-only transaction should be rejected by the database
		err = txConn.Exec(ctx /*sql*/, `INSERT INTO conntest_simple (id, val) VALUES (`+txConn.FormatPlaceholder(0)+`, `+txConn.FormatPlaceholder(1)+`)`, 1, "should-fail")
		assert.Error(t, err, "INSERT in read-only transaction should error")
	})

	t.Run("DefaultIsolationLevel", func(t *testing.T) {
		conn := config.NewConn(t)
		assert.Equal(t, config.DefaultIsolationLevel, conn.DefaultIsolationLevel())
	})
}
