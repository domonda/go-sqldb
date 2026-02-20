package sqliteconn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSavepoint_CommitNested(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })
	setupTestTable(t, conn)

	// Begin outer transaction
	tx1, err := conn.Begin(t.Context(), 1, nil)
	require.NoError(t, err)

	// Insert in outer transaction
	err = tx1.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "Outer", "outer@example.com")
	require.NoError(t, err)

	// Begin nested savepoint
	tx2, err := tx1.Begin(t.Context(), 2, nil)
	require.NoError(t, err)

	// Insert in nested savepoint
	err = tx2.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "Inner", "inner@example.com")
	require.NoError(t, err)

	// Commit nested savepoint
	err = tx2.Commit()
	require.NoError(t, err)

	// Commit outer transaction
	err = tx1.Commit()
	require.NoError(t, err)

	// Both rows should be visible
	rows := conn.Query(t.Context(), `SELECT COUNT(*) FROM users WHERE email IN (?, ?)`, "outer@example.com", "inner@example.com")
	require.True(t, rows.Next())
	var count int
	require.NoError(t, rows.Scan(&count))
	rows.Close()
	assert.Equal(t, 2, count)
}

func TestSavepoint_RollbackNested(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })
	setupTestTable(t, conn)

	// Begin outer transaction
	tx1, err := conn.Begin(t.Context(), 1, nil)
	require.NoError(t, err)

	// Insert in outer transaction
	err = tx1.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "Outer", "outer@example.com")
	require.NoError(t, err)

	// Begin nested savepoint
	tx2, err := tx1.Begin(t.Context(), 2, nil)
	require.NoError(t, err)

	// Insert in nested savepoint
	err = tx2.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "Inner", "inner@example.com")
	require.NoError(t, err)

	// Rollback nested savepoint
	err = tx2.Rollback()
	require.NoError(t, err)

	// Commit outer transaction
	err = tx1.Commit()
	require.NoError(t, err)

	// Only the outer row should be visible, inner was rolled back
	rows := conn.Query(t.Context(), `SELECT COUNT(*) FROM users WHERE email = ?`, "outer@example.com")
	require.True(t, rows.Next())
	var outerCount int
	require.NoError(t, rows.Scan(&outerCount))
	rows.Close()
	assert.Equal(t, 1, outerCount)

	rows = conn.Query(t.Context(), `SELECT COUNT(*) FROM users WHERE email = ?`, "inner@example.com")
	require.True(t, rows.Next())
	var innerCount int
	require.NoError(t, rows.Scan(&innerCount))
	rows.Close()
	assert.Equal(t, 0, innerCount)
}

func TestSavepoint_RollbackOuterIncludesNested(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })
	setupTestTable(t, conn)

	// Begin outer transaction
	tx1, err := conn.Begin(t.Context(), 1, nil)
	require.NoError(t, err)

	// Begin nested savepoint and commit it
	tx2, err := tx1.Begin(t.Context(), 2, nil)
	require.NoError(t, err)

	err = tx2.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "Inner", "inner@example.com")
	require.NoError(t, err)

	err = tx2.Commit()
	require.NoError(t, err)

	// Rollback outer transaction — should undo everything including the committed savepoint
	err = tx1.Rollback()
	require.NoError(t, err)

	rows := conn.Query(t.Context(), `SELECT COUNT(*) FROM users WHERE email = ?`, "inner@example.com")
	require.True(t, rows.Next())
	var count int
	require.NoError(t, rows.Scan(&count))
	rows.Close()
	assert.Equal(t, 0, count, "committed savepoint should be undone by outer rollback")
}

func TestSavepoint_ThreeLevelsDeep(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })
	setupTestTable(t, conn)

	// Level 0: outer transaction
	tx0, err := conn.Begin(t.Context(), 10, nil)
	require.NoError(t, err)

	err = tx0.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "L0", "l0@example.com")
	require.NoError(t, err)

	// Level 1: first savepoint
	tx1, err := tx0.Begin(t.Context(), 11, nil)
	require.NoError(t, err)

	err = tx1.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "L1", "l1@example.com")
	require.NoError(t, err)

	// Level 2: second savepoint (nested inside first)
	tx2, err := tx1.Begin(t.Context(), 12, nil)
	require.NoError(t, err)

	err = tx2.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "L2", "l2@example.com")
	require.NoError(t, err)

	// Commit level 2
	err = tx2.Commit()
	require.NoError(t, err)

	// Commit level 1
	err = tx1.Commit()
	require.NoError(t, err)

	// Commit level 0
	err = tx0.Commit()
	require.NoError(t, err)

	// All three rows should be visible
	rows := conn.Query(t.Context(), `SELECT COUNT(*) FROM users WHERE email IN (?, ?, ?)`, "l0@example.com", "l1@example.com", "l2@example.com")
	require.True(t, rows.Next())
	var count int
	require.NoError(t, rows.Scan(&count))
	rows.Close()
	assert.Equal(t, 3, count)
}

func TestSavepoint_ThreeLevelsRollbackMiddle(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })
	setupTestTable(t, conn)

	// Level 0: outer transaction
	tx0, err := conn.Begin(t.Context(), 20, nil)
	require.NoError(t, err)

	err = tx0.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "L0", "l0@example.com")
	require.NoError(t, err)

	// Level 1: first savepoint
	tx1, err := tx0.Begin(t.Context(), 21, nil)
	require.NoError(t, err)

	err = tx1.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "L1", "l1@example.com")
	require.NoError(t, err)

	// Level 2: second savepoint
	tx2, err := tx1.Begin(t.Context(), 22, nil)
	require.NoError(t, err)

	err = tx2.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "L2", "l2@example.com")
	require.NoError(t, err)

	// Commit level 2
	err = tx2.Commit()
	require.NoError(t, err)

	// Rollback level 1 — should undo both L1 and L2
	err = tx1.Rollback()
	require.NoError(t, err)

	// Commit level 0
	err = tx0.Commit()
	require.NoError(t, err)

	// Only L0 should be visible
	for _, tc := range []struct {
		email string
		want  int
	}{
		{"l0@example.com", 1},
		{"l1@example.com", 0},
		{"l2@example.com", 0},
	} {
		rows := conn.Query(t.Context(), `SELECT COUNT(*) FROM users WHERE email = ?`, tc.email)
		require.True(t, rows.Next(), "expected row for %s", tc.email)
		var count int
		require.NoError(t, rows.Scan(&count))
		rows.Close()
		assert.Equal(t, tc.want, count, "email %s", tc.email)
	}
}

func TestSavepoint_ThreeLevelsRollbackDeepest(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })
	setupTestTable(t, conn)

	// Level 0: outer transaction
	tx0, err := conn.Begin(t.Context(), 30, nil)
	require.NoError(t, err)

	err = tx0.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "L0", "l0@example.com")
	require.NoError(t, err)

	// Level 1: first savepoint
	tx1, err := tx0.Begin(t.Context(), 31, nil)
	require.NoError(t, err)

	err = tx1.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "L1", "l1@example.com")
	require.NoError(t, err)

	// Level 2: second savepoint
	tx2, err := tx1.Begin(t.Context(), 32, nil)
	require.NoError(t, err)

	err = tx2.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "L2", "l2@example.com")
	require.NoError(t, err)

	// Rollback only level 2
	err = tx2.Rollback()
	require.NoError(t, err)

	// Commit level 1 and level 0
	err = tx1.Commit()
	require.NoError(t, err)
	err = tx0.Commit()
	require.NoError(t, err)

	// L0 and L1 should be visible, L2 should not
	for _, tc := range []struct {
		email string
		want  int
	}{
		{"l0@example.com", 1},
		{"l1@example.com", 1},
		{"l2@example.com", 0},
	} {
		rows := conn.Query(t.Context(), `SELECT COUNT(*) FROM users WHERE email = ?`, tc.email)
		require.True(t, rows.Next(), "expected row for %s", tc.email)
		var count int
		require.NoError(t, rows.Scan(&count))
		rows.Close()
		assert.Equal(t, tc.want, count, "email %s", tc.email)
	}
}

func TestSavepoint_SiblingsSavepoints(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })
	setupTestTable(t, conn)

	// Outer transaction
	tx0, err := conn.Begin(t.Context(), 40, nil)
	require.NoError(t, err)

	// First sibling savepoint — commit
	sp1, err := tx0.Begin(t.Context(), 41, nil)
	require.NoError(t, err)

	err = sp1.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "SP1", "sp1@example.com")
	require.NoError(t, err)

	err = sp1.Commit()
	require.NoError(t, err)

	// Second sibling savepoint — rollback
	sp2, err := tx0.Begin(t.Context(), 42, nil)
	require.NoError(t, err)

	err = sp2.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "SP2", "sp2@example.com")
	require.NoError(t, err)

	err = sp2.Rollback()
	require.NoError(t, err)

	// Third sibling savepoint — commit
	sp3, err := tx0.Begin(t.Context(), 43, nil)
	require.NoError(t, err)

	err = sp3.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "SP3", "sp3@example.com")
	require.NoError(t, err)

	err = sp3.Commit()
	require.NoError(t, err)

	// Commit outer
	err = tx0.Commit()
	require.NoError(t, err)

	// SP1 and SP3 committed, SP2 rolled back
	for _, tc := range []struct {
		email string
		want  int
	}{
		{"sp1@example.com", 1},
		{"sp2@example.com", 0},
		{"sp3@example.com", 1},
	} {
		rows := conn.Query(t.Context(), `SELECT COUNT(*) FROM users WHERE email = ?`, tc.email)
		require.True(t, rows.Next(), "expected row for %s", tc.email)
		var count int
		require.NoError(t, rows.Scan(&count))
		rows.Close()
		assert.Equal(t, tc.want, count, "email %s", tc.email)
	}
}

func TestSavepoint_TransactionState(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	tx, err := conn.Begin(t.Context(), 50, nil)
	require.NoError(t, err)
	t.Cleanup(func() { tx.Rollback() })

	// Outer transaction state
	state := tx.Transaction()
	assert.True(t, state.Active())
	assert.Equal(t, uint64(50), state.ID)

	// Nested savepoint state
	sp, err := tx.Begin(t.Context(), 51, nil)
	require.NoError(t, err)

	spState := sp.Transaction()
	assert.True(t, spState.Active())
	assert.Equal(t, uint64(51), spState.ID)

	sp.Rollback()
}

func TestSavepoint_CloseRollsBack(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })
	setupTestTable(t, conn)

	tx, err := conn.Begin(t.Context(), 60, nil)
	require.NoError(t, err)

	sp, err := tx.Begin(t.Context(), 61, nil)
	require.NoError(t, err)

	err = sp.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "Closed", "closed@example.com")
	require.NoError(t, err)

	// Close the savepoint (should rollback)
	err = sp.Close()
	require.NoError(t, err)

	// Commit outer
	err = tx.Commit()
	require.NoError(t, err)

	// Row should not be visible because Close rolled back the savepoint
	rows := conn.Query(t.Context(), `SELECT COUNT(*) FROM users WHERE email = ?`, "closed@example.com")
	require.True(t, rows.Next())
	var count int
	require.NoError(t, rows.Scan(&count))
	rows.Close()
	assert.Equal(t, 0, count)
}
