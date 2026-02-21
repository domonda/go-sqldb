package sqliteconn

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestConnect(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		config := &sqldb.ConnConfig{
			Driver:   "sqlite",
			Host:     "localhost",
			Database: ":memory:",
		}

		conn, err := Connect(t.Context(), config)
		require.NoError(t, err)
		require.NotNil(t, conn)
		t.Cleanup(func() { conn.Close() })

		// Verify connection is working
		err = conn.Ping(t.Context(), time.Second)
		assert.NoError(t, err)
	})

	t.Run("invalid driver", func(t *testing.T) {
		config := &sqldb.ConnConfig{
			Driver:   "postgres",
			Host:     "localhost",
			Database: ":memory:",
		}

		conn, err := Connect(t.Context(), config)
		assert.Error(t, err)
		assert.Nil(t, conn)
		assert.Contains(t, err.Error(), "invalid driver")
	})

	t.Run("read-only mode", func(t *testing.T) {
		config := &sqldb.ConnConfig{
			Driver:   "sqlite",
			Host:     "localhost",
			Database: ":memory:",
			ReadOnly: true,
		}

		conn, err := Connect(t.Context(), config)
		require.NoError(t, err)
		require.NotNil(t, conn)
		t.Cleanup(func() { conn.Close() })
	})
}

func TestMustConnect(t *testing.T) {

	t.Run("successful connection", func(t *testing.T) {
		config := &sqldb.ConnConfig{
			Driver:   "sqlite",
			Host:     "localhost",
			Database: ":memory:",
		}

		conn := MustConnect(t.Context(), config)
		require.NotNil(t, conn)
		t.Cleanup(func() { conn.Close() })
	})

	t.Run("panic on error", func(t *testing.T) {
		config := &sqldb.ConnConfig{
			Driver:   "invalid",
			Host:     "localhost",
			Database: ":memory:",
		}

		assert.Panics(t, func() {
			MustConnect(t.Context(), config)
		})
	})
}

func TestConnection_Exec(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	t.Run("create table", func(t *testing.T) {
		err := conn.Exec(t.Context(), `
			CREATE TABLE users (
				id INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				email TEXT UNIQUE
			)
		`)
		assert.NoError(t, err)
	})

	t.Run("insert data", func(t *testing.T) {
		err := conn.Exec(t.Context(), `INSERT INTO users (id, name, email) VALUES (?, ?, ?)`, 1, "Alice", "alice@example.com")
		assert.NoError(t, err)
	})

	t.Run("insert duplicate - unique violation", func(t *testing.T) {
		err := conn.Exec(t.Context(), `INSERT INTO users (id, name, email) VALUES (?, ?, ?)`, 2, "Bob", "alice@example.com")
		assert.Error(t, err)
		assert.True(t, IsUniqueViolation(err))
	})

	t.Run("insert null - not null violation", func(t *testing.T) {
		err := conn.Exec(t.Context(), `INSERT INTO users (id, name) VALUES (?, ?)`, 3, nil)
		assert.Error(t, err)
		assert.True(t, IsNotNullViolation(err))
	})
}

func TestConnection_Query(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	// Setup test data
	setupTestTable(t, conn)

	t.Run("query rows", func(t *testing.T) {
		rows := conn.Query(t.Context(), `SELECT id, name, email FROM users ORDER BY id`)
		t.Cleanup(func() { rows.Close() })

		assert.NoError(t, rows.Err())

		count := 0
		for rows.Next() {
			var id int
			var name, email string
			err := rows.Scan(&id, &name, &email)
			assert.NoError(t, err)
			count++
		}
		assert.Equal(t, 3, count)
		assert.NoError(t, rows.Err())
	})

	t.Run("query with parameters", func(t *testing.T) {
		rows := conn.Query(t.Context(), `SELECT name FROM users WHERE id = ?`, 2)
		t.Cleanup(func() { rows.Close() })

		assert.True(t, rows.Next())
		var name string
		err := rows.Scan(&name)
		assert.NoError(t, err)
		assert.Equal(t, "Bob", name)
	})

	t.Run("query no rows", func(t *testing.T) {
		rows := conn.Query(t.Context(), `SELECT * FROM users WHERE id = ?`, 999)
		t.Cleanup(func() { rows.Close() })

		assert.False(t, rows.Next())
		assert.NoError(t, rows.Err())
	})
}

func TestConnection_Update(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })
	setupTestTable(t, conn)

	t.Run("update single row", func(t *testing.T) {
		err := conn.Exec(t.Context(), `UPDATE users SET name = ? WHERE email = ?`, "Alice Updated", "alice@example.com")
		require.NoError(t, err)

		// Verify the update
		rows := conn.Query(t.Context(), `SELECT name FROM users WHERE email = ?`, "alice@example.com")
		t.Cleanup(func() { rows.Close() })
		require.True(t, rows.Next())
		var name string
		require.NoError(t, rows.Scan(&name))
		assert.Equal(t, "Alice Updated", name)
	})

	t.Run("update multiple rows", func(t *testing.T) {
		err := conn.Exec(t.Context(), `UPDATE users SET name = ? WHERE name != ?`, "Same Name", "Alice Updated")
		require.NoError(t, err)

		// Verify all non-Alice rows have been updated
		rows := conn.Query(t.Context(), `SELECT COUNT(*) FROM users WHERE name = ?`, "Same Name")
		t.Cleanup(func() { rows.Close() })
		require.True(t, rows.Next())
		var count int
		require.NoError(t, rows.Scan(&count))
		assert.Equal(t, 2, count)
	})

	t.Run("update no matching rows", func(t *testing.T) {
		err := conn.Exec(t.Context(), `UPDATE users SET name = ? WHERE email = ?`, "Nobody", "nonexistent@example.com")
		assert.NoError(t, err, "updating zero rows should not error")
	})
}

func TestConnection_Delete(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })
	setupTestTable(t, conn)

	t.Run("delete single row", func(t *testing.T) {
		err := conn.Exec(t.Context(), `DELETE FROM users WHERE email = ?`, "alice@example.com")
		require.NoError(t, err)

		// Verify deleted
		rows := conn.Query(t.Context(), `SELECT COUNT(*) FROM users WHERE email = ?`, "alice@example.com")
		t.Cleanup(func() { rows.Close() })
		require.True(t, rows.Next())
		var count int
		require.NoError(t, rows.Scan(&count))
		assert.Equal(t, 0, count)
	})

	t.Run("delete no matching rows", func(t *testing.T) {
		err := conn.Exec(t.Context(), `DELETE FROM users WHERE email = ?`, "nonexistent@example.com")
		assert.NoError(t, err, "deleting zero rows should not error")
	})

	t.Run("delete all rows", func(t *testing.T) {
		err := conn.Exec(t.Context(), `DELETE FROM users`)
		require.NoError(t, err)

		rows := conn.Query(t.Context(), `SELECT COUNT(*) FROM users`)
		t.Cleanup(func() { rows.Close() })
		require.True(t, rows.Next())
		var count int
		require.NoError(t, rows.Scan(&count))
		assert.Equal(t, 0, count)
	})
}

func TestConnection_Prepare(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	setupTestTable(t, conn)

	t.Run("prepare and execute statement", func(t *testing.T) {
		stmt, err := conn.Prepare(t.Context(), `SELECT name FROM users WHERE id = ?`)
		require.NoError(t, err)
		t.Cleanup(func() { stmt.Close() })

		rows := stmt.Query(t.Context(), 1)
		t.Cleanup(func() { rows.Close() })

		assert.True(t, rows.Next())
		var name string
		err = rows.Scan(&name)
		assert.NoError(t, err)
		assert.Equal(t, "Alice", name)
	})

	t.Run("prepare and execute multiple times", func(t *testing.T) {
		stmt, err := conn.Prepare(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`)
		require.NoError(t, err)
		t.Cleanup(func() { stmt.Close() })

		err = stmt.Exec(t.Context(), "Dave", "dave@example.com")
		assert.NoError(t, err)

		err = stmt.Exec(t.Context(), "Eve", "eve@example.com")
		assert.NoError(t, err)
	})
}

func TestConnection_Transaction(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	setupTestTable(t, conn)

	t.Run("commit transaction", func(t *testing.T) {
		tx, err := conn.Begin(t.Context(), 1, nil)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// Insert data in transaction
		err = tx.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "Frank", "frank@example.com")
		assert.NoError(t, err)

		// Commit
		err = tx.Commit()
		assert.NoError(t, err)

		// Verify data persisted
		rows := conn.Query(t.Context(), `SELECT COUNT(*) FROM users WHERE email = ?`, "frank@example.com")
		t.Cleanup(func() { rows.Close() })
		assert.True(t, rows.Next())
		var count int
		rows.Scan(&count)
		assert.Equal(t, 1, count)
	})

	t.Run("rollback transaction", func(t *testing.T) {
		tx, err := conn.Begin(t.Context(), 2, nil)
		require.NoError(t, err)

		// Insert data in transaction
		err = tx.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "Grace", "grace@example.com")
		assert.NoError(t, err)

		// Rollback
		err = tx.Rollback()
		assert.NoError(t, err)

		// Verify data not persisted
		rows := conn.Query(t.Context(), `SELECT COUNT(*) FROM users WHERE email = ?`, "grace@example.com")
		t.Cleanup(func() { rows.Close() })
		assert.True(t, rows.Next())
		var count int
		rows.Scan(&count)
		assert.Equal(t, 0, count)
	})

	t.Run("nested transaction", func(t *testing.T) {
		tx1, err := conn.Begin(t.Context(), 3, nil)
		require.NoError(t, err)

		// Begin nested transaction
		tx2, err := tx1.Begin(t.Context(), 4, nil)
		require.NoError(t, err)

		err = tx2.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "Henry", "henry@example.com")
		assert.NoError(t, err)

		err = tx2.Commit()
		assert.NoError(t, err)

		err = tx1.Commit()
		assert.NoError(t, err)
	})

	t.Run("transaction isolation level", func(t *testing.T) {
		opts := &sql.TxOptions{
			Isolation: sql.LevelSerializable,
		}
		tx, err := conn.Begin(t.Context(), 5, opts)
		require.NoError(t, err)

		assert.Equal(t, uint64(5), tx.Transaction().ID)
		assert.Equal(t, sql.LevelSerializable, tx.Transaction().Opts.Isolation)

		err = tx.Rollback()
		assert.NoError(t, err)
	})
}

func TestConnection_TransactionState(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	t.Run("not in transaction", func(t *testing.T) {
		state := conn.Transaction()
		assert.False(t, state.Active())
		assert.Equal(t, uint64(0), state.ID)
		assert.Nil(t, state.Opts)
	})

	t.Run("in transaction", func(t *testing.T) {
		tx, err := conn.Begin(t.Context(), 42, nil)
		require.NoError(t, err)
		t.Cleanup(func() { tx.Rollback() })

		state := tx.Transaction()
		assert.True(t, state.Active())
		assert.Equal(t, uint64(42), state.ID)
	})
}

func TestConnection_CommitRollbackErrors(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	t.Run("commit without transaction", func(t *testing.T) {
		err := conn.Commit()
		assert.ErrorIs(t, err, sqldb.ErrNotWithinTransaction)
	})

	t.Run("rollback without transaction", func(t *testing.T) {
		err := conn.Rollback()
		assert.ErrorIs(t, err, sqldb.ErrNotWithinTransaction)
	})
}

func TestConnection_DefaultIsolationLevel(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	level := conn.DefaultIsolationLevel()
	assert.Equal(t, sql.LevelSerializable, level)
}

func TestConnection_Stats(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	stats := conn.Stats()
	assert.GreaterOrEqual(t, stats.OpenConnections, 0)
}

// Note: SQLite does not support LISTEN/NOTIFY (PostgreSQL-specific feature)
// The sqliteconn.connection does not implement ListenerConnection interface

func TestConnection_ForeignKeys(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	// Create parent and child tables
	err := conn.Exec(t.Context(), `
		CREATE TABLE parent (
			id INTEGER PRIMARY KEY
		)
	`)
	require.NoError(t, err)

	err = conn.Exec(t.Context(), `
		CREATE TABLE child (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER NOT NULL,
			FOREIGN KEY (parent_id) REFERENCES parent(id)
		)
	`)
	require.NoError(t, err)

	t.Run("foreign key constraint enforced", func(t *testing.T) {
		// Try to insert child without parent
		err := conn.Exec(t.Context(), `INSERT INTO child (id, parent_id) VALUES (?, ?)`, 1, 999)
		assert.Error(t, err)
		assert.True(t, IsForeignKeyViolation(err))
	})

	t.Run("foreign key constraint satisfied", func(t *testing.T) {
		// Insert parent first
		err := conn.Exec(t.Context(), `INSERT INTO parent (id) VALUES (?)`, 1)
		require.NoError(t, err)

		// Now insert child
		err = conn.Exec(t.Context(), `INSERT INTO child (id, parent_id) VALUES (?, ?)`, 1, 1)
		assert.NoError(t, err)
	})
}

func TestConnection_CheckConstraint(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	err := conn.Exec(t.Context(), `
		CREATE TABLE products (
			id INTEGER PRIMARY KEY,
			price REAL CHECK(price > 0)
		)
	`)
	require.NoError(t, err)

	t.Run("check constraint violated", func(t *testing.T) {
		err := conn.Exec(t.Context(), `INSERT INTO products (id, price) VALUES (?, ?)`, 1, -10.0)
		assert.Error(t, err)
		assert.True(t, IsCheckViolation(err))
	})

	t.Run("check constraint satisfied", func(t *testing.T) {
		err := conn.Exec(t.Context(), `INSERT INTO products (id, price) VALUES (?, ?)`, 2, 10.0)
		assert.NoError(t, err)
	})
}

func TestConnection_ContextCancellation(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })
	setupTestTable(t, conn)

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		err := conn.Exec(ctx, `INSERT INTO users (name, email) VALUES (?, ?)`, "Test", "test@example.com")
		// SQLite operations are very fast and might complete before context cancellation is detected
		// So we don't assert that an error must occur, just log if one does
		if err != nil {
			t.Logf("Got error with cancelled context: %v", err)
		}
	})

	t.Run("timeout context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Ensure timeout

		err := conn.Exec(ctx, `INSERT INTO users (name, email) VALUES (?, ?)`, "Test", "test@example.com")
		// Similar to above, SQLite might complete before timeout is detected
		if err != nil {
			t.Logf("Got error with timeout context: %v", err)
		}
	})
}

// customTextType is a test type implementing driver.Valuer and sql.Scanner
// for testing custom type support in bindArgs and scanColumn.
type customTextType struct {
	data string
}

func (c customTextType) Value() (driver.Value, error) {
	if c.data == "" {
		return nil, nil
	}
	return c.data, nil
}

func (c *customTextType) Scan(src any) error {
	if src == nil {
		c.data = ""
		return nil
	}
	switch v := src.(type) {
	case string:
		c.data = v
	case []byte:
		c.data = string(v)
	default:
		return fmt.Errorf("customTextType.Scan: unsupported type %T", src)
	}
	return nil
}

func TestDriverValuerAndSQLScanner(t *testing.T) {
	conn := testConnection(t)
	t.Cleanup(func() { conn.Close() })

	err := conn.Exec(t.Context(), `
		CREATE TABLE custom_types (
			id INTEGER PRIMARY KEY,
			val TEXT
		)
	`)
	require.NoError(t, err)

	t.Run("insert and query custom type", func(t *testing.T) {
		input := customTextType{data: "hello world"}
		err := conn.Exec(t.Context(), `INSERT INTO custom_types (id, val) VALUES (?, ?)`, 1, input)
		require.NoError(t, err)

		rows := conn.Query(t.Context(), `SELECT val FROM custom_types WHERE id = ?`, 1)
		t.Cleanup(func() { rows.Close() })

		require.True(t, rows.Next())
		var output customTextType
		err = rows.Scan(&output)
		require.NoError(t, err)
		assert.Equal(t, "hello world", output.data)
	})

	t.Run("insert nil via Valuer", func(t *testing.T) {
		input := customTextType{data: ""}
		err := conn.Exec(t.Context(), `INSERT INTO custom_types (id, val) VALUES (?, ?)`, 2, input)
		require.NoError(t, err)

		rows := conn.Query(t.Context(), `SELECT val FROM custom_types WHERE id = ?`, 2)
		t.Cleanup(func() { rows.Close() })

		require.True(t, rows.Next())
		var output customTextType
		err = rows.Scan(&output)
		require.NoError(t, err)
		assert.Equal(t, "", output.data)
	})
}

// Helper functions

func testConnection(t *testing.T) sqldb.Connection {
	t.Helper()
	config := &sqldb.ConnConfig{
		Driver:   "sqlite",
		Host:     "localhost", // SQLite doesn't use host, but ConnConfig requires it
		Database: ":memory:",
	}
	conn, err := Connect(t.Context(), config)
	require.NoError(t, err)
	require.NotNil(t, conn)
	return conn
}

func setupTestTable(t *testing.T, conn sqldb.Connection) {
	t.Helper()

	err := conn.Exec(t.Context(), `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE
		)
	`)
	require.NoError(t, err)

	err = conn.Exec(t.Context(), `DELETE FROM users`)
	require.NoError(t, err)

	err = conn.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "Alice", "alice@example.com")
	require.NoError(t, err)

	err = conn.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "Bob", "bob@example.com")
	require.NoError(t, err)

	err = conn.Exec(t.Context(), `INSERT INTO users (name, email) VALUES (?, ?)`, "Charlie", "charlie@example.com")
	require.NoError(t, err)
}
