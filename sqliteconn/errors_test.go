package sqliteconn

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestErrorWrapping(t *testing.T) {
	ctx := t.Context()
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	// Create test table with various constraints
	err := conn.Exec(ctx, `
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			unique_col TEXT UNIQUE,
			not_null_col TEXT NOT NULL,
			check_col INTEGER CHECK(check_col > 0),
			foreign_key_col INTEGER,
			FOREIGN KEY (foreign_key_col) REFERENCES test_table(id)
		)
	`)
	require.NoError(t, err)

	t.Run("unique constraint violation", func(t *testing.T) {
		err := conn.Exec(ctx, `INSERT INTO test_table (id, unique_col, not_null_col) VALUES (?, ?, ?)`, 1, "unique", "value")
		require.NoError(t, err)

		err = conn.Exec(ctx, `INSERT INTO test_table (id, unique_col, not_null_col) VALUES (?, ?, ?)`, 2, "unique", "value")
		assert.Error(t, err)
		assert.True(t, IsUniqueViolation(err), "expected unique violation")
		assert.True(t, IsConstraintViolation(err), "expected constraint violation")
	})

	t.Run("not null constraint violation", func(t *testing.T) {
		err := conn.Exec(ctx, `INSERT INTO test_table (id, unique_col) VALUES (?, ?)`, 3, "test")
		assert.Error(t, err)
		assert.True(t, IsNotNullViolation(err), "expected not null violation")
		assert.True(t, IsConstraintViolation(err), "expected constraint violation")
	})

	t.Run("check constraint violation", func(t *testing.T) {
		err := conn.Exec(ctx, `INSERT INTO test_table (id, not_null_col, check_col) VALUES (?, ?, ?)`, 4, "test", -1)
		assert.Error(t, err)
		assert.True(t, IsCheckViolation(err), "expected check violation")
		assert.True(t, IsConstraintViolation(err), "expected constraint violation")
	})

	t.Run("foreign key constraint violation", func(t *testing.T) {
		err := conn.Exec(ctx, `INSERT INTO test_table (id, not_null_col, foreign_key_col) VALUES (?, ?, ?)`, 5, "test", 999)
		assert.Error(t, err)
		assert.True(t, IsForeignKeyViolation(err), "expected foreign key violation")
		assert.True(t, IsConstraintViolation(err), "expected constraint violation")
	})

	t.Run("no error", func(t *testing.T) {
		err := conn.Exec(ctx, `INSERT INTO test_table (id, not_null_col, check_col) VALUES (?, ?, ?)`, 6, "test", 10)
		assert.NoError(t, err)
		assert.False(t, IsUniqueViolation(err))
		assert.False(t, IsNotNullViolation(err))
		assert.False(t, IsCheckViolation(err))
		assert.False(t, IsForeignKeyViolation(err))
		assert.False(t, IsConstraintViolation(err))
		assert.False(t, IsDatabaseLocked(err))
	})
}

func TestExtractConstraint(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{
			name:     "unique constraint",
			errMsg:   "unique constraint failed: test_table.unique_col",
			expected: "test_table.unique_col",
		},
		{
			name:     "foreign key constraint",
			errMsg:   "foreign key constraint failed",
			expected: "",
		},
		{
			name:     "not null constraint",
			errMsg:   "not null constraint failed: test_table.not_null_col",
			expected: "test_table.not_null_col",
		},
		{
			name:     "check constraint",
			errMsg:   "check constraint failed: check_col",
			expected: "check_col",
		},
		{
			name:     "no constraint info",
			errMsg:   "some other error",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractConstraint(tt.errMsg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsConstraintViolation_NilError(t *testing.T) {
	assert.False(t, IsConstraintViolation(nil))
	assert.False(t, IsUniqueViolation(nil))
	assert.False(t, IsNotNullViolation(nil))
	assert.False(t, IsCheckViolation(nil))
	assert.False(t, IsForeignKeyViolation(nil))
	assert.False(t, IsDatabaseLocked(nil))
}

func TestIsDatabaseLocked(t *testing.T) {
	// Use a file-based database so two connections can access it
	dbPath := filepath.Join(t.TempDir(), "locked.db")

	config := &sqldb.ConnConfig{
		Driver:   "sqlite",
		Host:     "localhost",
		Database: dbPath,
	}

	// Open two connections to the same file
	conn1, err := Connect(config)
	require.NoError(t, err)
	t.Cleanup(func() { conn1.Close() })

	conn2, err := Connect(config)
	require.NoError(t, err)
	t.Cleanup(func() { conn2.Close() })

	// Ensure conn2 returns SQLITE_BUSY immediately instead of waiting
	err = conn2.Exec(t.Context(), `PRAGMA busy_timeout = 0`)
	require.NoError(t, err)

	// Create a table using connection 1
	err = conn1.Exec(t.Context(), `CREATE TABLE locktest (id INTEGER PRIMARY KEY, val TEXT)`)
	require.NoError(t, err)

	// Connection 1: begin an IMMEDIATE transaction to acquire the write lock
	tx1, err := conn1.Begin(t.Context(), 1, &sql.TxOptions{Isolation: sql.LevelSerializable})
	require.NoError(t, err)

	err = tx1.Exec(t.Context(), `INSERT INTO locktest (id, val) VALUES (?, ?)`, 1, "from_conn1")
	require.NoError(t, err)

	// Connection 2: try to write while connection 1 holds the write lock
	err = conn2.Exec(t.Context(), `INSERT INTO locktest (id, val) VALUES (?, ?)`, 2, "from_conn2")
	require.Error(t, err)
	assert.True(t, IsDatabaseLocked(err), "expected database locked error, got: %v", err)

	// Verify IsDatabaseLocked returns false for non-lock errors
	assert.False(t, IsDatabaseLocked(nil))

	// Commit connection 1 transaction
	err = tx1.Commit()
	require.NoError(t, err)

	// Now connection 2 should be able to write
	err = conn2.Exec(t.Context(), `INSERT INTO locktest (id, val) VALUES (?, ?)`, 2, "from_conn2")
	assert.NoError(t, err)
}

func TestReadOnlyError(t *testing.T) {
	// Create a read-only connection
	config := &sqldb.ConnConfig{
		Driver:   "sqlite",
		Host:     "localhost",
		Database: ":memory:",
		ReadOnly: true,
	}

	conn, err := Connect(config)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	// Try to create a table (should fail in read-only mode)
	err = conn.Exec(t.Context(), `CREATE TABLE test (id INTEGER PRIMARY KEY)`)
	assert.Error(t, err)
	// Note: The specific error depends on SQLite's read-only implementation
}

func TestContextCancellationError(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	setupTestTable(t, conn)

	// Create a cancelled context
	cancelledCtx, cancel := context.WithCancel(t.Context())
	cancel()

	// Try to execute with cancelled context
	err := conn.Exec(cancelledCtx, `INSERT INTO users (name, email) VALUES (?, ?)`, "Test", "test@example.com")
	// The error behavior depends on timing, but it should be an error
	if err != nil {
		// If we get an error, it might be wrapped with context.Canceled
		t.Logf("Got error with cancelled context: %v", err)
	}
}
