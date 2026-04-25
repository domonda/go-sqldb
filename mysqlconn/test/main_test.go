package mysqlconn

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/mysqlconn"
)

var (
	mysqlUser     = cmp.Or(os.Getenv("MYSQL_USER"), "testuser")
	mysqlPassword = cmp.Or(os.Getenv("MYSQL_PASSWORD"), "testpassword")
	mysqlHost     = cmp.Or(os.Getenv("MYSQL_HOST"), "localhost")
	mysqlPort     = cmp.Or(atoi(os.Getenv("MYSQL_PORT")), 3307)
	dbName        = cmp.Or(os.Getenv("MYSQL_DB"), "testdb")

	refl = sqldb.NewTaggedStructReflector()
)

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

func testConfig() *sqldb.Config {
	return &sqldb.Config{
		Driver:   mysqlconn.Driver,
		Host:     mysqlHost,
		Port:     uint16(mysqlPort),
		User:     mysqlUser,
		Password: mysqlPassword,
		Database: dbName,
	}
}

func dockerComposeUp() error {
	return exec.Command("docker", "compose", "up", "-d").Run()
}

func dropAllTables() error {
	ctx := context.Background()
	conn, err := mysqlconn.Connect(ctx, testConfig())
	if err != nil {
		return err
	}
	defer conn.Close()
	return mysqlconn.DropAllTables(ctx, conn)
}

func waitForMariaDB() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", mysqlUser, mysqlPassword, mysqlHost, mysqlPort, dbName)
	for range 30 {
		db, err := sql.Open("mysql", dsn)
		if err == nil {
			err = db.Ping()
			db.Close()
			if err == nil {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("MariaDB not ready after 30 seconds")
}

func TestMain(m *testing.M) {
	if os.Getenv("CI") == "" {
		err := dockerComposeUp()
		if err != nil {
			log.Fatalf("Failed to start Docker Compose: %v", err)
		}
	}

	err := waitForMariaDB()
	if err != nil {
		log.Fatalf("Failed waiting for MariaDB: %v", err)
	}

	err = dropAllTables()
	if err != nil {
		log.Fatalf("Failed to drop all tables before tests: %v", err)
	}

	m.Run()
}

func TestConfig(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	cfg := conn.Config()
	require.NotNil(t, cfg)
	assert.Equal(t, mysqlconn.Driver, cfg.Driver)
	assert.Equal(t, dbName, cfg.Database)
}

func TestPing(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	err = conn.Ping(t.Context(), 5*time.Second)
	assert.NoError(t, err)
}

func TestStats(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	// Stats() should return without panic; exact values depend on pool state
	_ = conn.Stats()
}

func TestDefaultIsolationLevel(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	assert.Equal(t, sql.LevelRepeatableRead, conn.DefaultIsolationLevel())
}

func TestTransactionState(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	t.Run("not in transaction", func(t *testing.T) {
		tx := conn.Transaction()
		assert.False(t, tx.Active())
	})

	t.Run("in transaction", func(t *testing.T) {
		txConn, err := conn.Begin(t.Context(), 1, nil)
		require.NoError(t, err)
		defer txConn.Rollback() //nolint:errcheck

		tx := txConn.Transaction()
		assert.True(t, tx.Active())
	})
}

func TestExecRowsAffected(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE IF NOT EXISTS test_rows_affected (id INT PRIMARY KEY, val TEXT)`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE IF EXISTS test_rows_affected`,
	)

	err = conn.Exec(ctx,
		/*sql*/ `INSERT INTO test_rows_affected (id, val) VALUES (?, ?), (?, ?), (?, ?)`, 1, "a", 2, "b", 3, "c",
	)
	require.NoError(t, err)

	n, err := conn.ExecRowsAffected(ctx,
		/*sql*/ `DELETE FROM test_rows_affected WHERE id IN (?, ?)`, 1, 2,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

func TestPrepare(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE IF NOT EXISTS test_prepare (id INT AUTO_INCREMENT PRIMARY KEY, val TEXT)`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE IF EXISTS test_prepare`,
	)

	stmt, err := conn.Prepare(ctx,
		/*sql*/ `INSERT INTO test_prepare (val) VALUES (?)`,
	)
	require.NoError(t, err)
	defer stmt.Close() //nolint:errcheck

	err = stmt.Exec(ctx, "prepared-value")
	require.NoError(t, err)

	rows := conn.Query(ctx,
		/*sql*/ `SELECT val FROM test_prepare LIMIT 1`,
	)
	require.True(t, rows.Next())
	var val string
	require.NoError(t, rows.Scan(&val))
	assert.Equal(t, "prepared-value", val)
	require.NoError(t, rows.Close())
}

func TestConnect(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	rows := conn.Query(t.Context(),
		/*sql*/ `SELECT 1`,
	)
	require.True(t, rows.Next())
	var result int
	require.NoError(t, rows.Scan(&result))
	assert.Equal(t, 1, result)
	require.NoError(t, rows.Close())
}

func TestMustConnectPanics(t *testing.T) {
	badConfig := &sqldb.Config{
		Driver:   mysqlconn.Driver,
		Host:     "invalid-host-that-does-not-exist",
		Port:     9999,
		User:     "nobody",
		Password: "nothing",
		Database: "nodb",
	}
	assert.Panics(t, func() {
		mysqlconn.MustConnect(t.Context(), badConfig)
	})
}

func TestNewConfig(t *testing.T) {
	config := mysqlconn.NewConfig()
	require.NotNil(t, config)
}

func TestExec(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE IF NOT EXISTS test_exec (id INT PRIMARY KEY, val TEXT)`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE IF EXISTS test_exec`,
	)

	err = conn.Exec(ctx,
		/*sql*/ `INSERT INTO test_exec (id, val) VALUES (?, ?)`, 1, "hello",
	)
	require.NoError(t, err)

	rows := conn.Query(ctx,
		/*sql*/ `SELECT val FROM test_exec WHERE id = ?`, 1,
	)
	require.True(t, rows.Next())
	var val string
	require.NoError(t, rows.Scan(&val))
	assert.Equal(t, "hello", val)
	require.NoError(t, rows.Close())
}

func TestQueryRow(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE IF NOT EXISTS test_queryrow (id INT PRIMARY KEY, val TEXT)`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE IF EXISTS test_queryrow`,
	)

	err = conn.Exec(ctx,
		/*sql*/ `INSERT INTO test_queryrow (id, val) VALUES (?, ?), (?, ?)`, 1, "alpha", 2, "beta",
	)
	require.NoError(t, err)

	rows := conn.Query(ctx,
		/*sql*/ `SELECT val FROM test_queryrow WHERE id = ?`, 2,
	)
	require.True(t, rows.Next())
	var val string
	require.NoError(t, rows.Scan(&val))
	assert.Equal(t, "beta", val)
	assert.False(t, rows.Next())
	require.NoError(t, rows.Close())
}

func TestQueryRows(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE IF NOT EXISTS test_queryrows (id INT PRIMARY KEY, val INT)`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE IF EXISTS test_queryrows`,
	)

	err = conn.Exec(ctx,
		/*sql*/ `INSERT INTO test_queryrows (id, val) VALUES (?, ?), (?, ?), (?, ?)`, 1, 10, 2, 20, 3, 30,
	)
	require.NoError(t, err)

	rows := conn.Query(ctx,
		/*sql*/ `SELECT val FROM test_queryrows WHERE val > ? ORDER BY val`, 15,
	)
	var vals []int
	for rows.Next() {
		var v int
		require.NoError(t, rows.Scan(&v))
		vals = append(vals, v)
	}
	require.NoError(t, rows.Close())
	assert.Equal(t, []int{20, 30}, vals)
}

func TestTransaction(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE IF NOT EXISTS test_tx (id INT PRIMARY KEY, val TEXT)`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE IF EXISTS test_tx`,
	)

	txConn, err := conn.Begin(ctx, 1, nil)
	require.NoError(t, err)

	err = txConn.Exec(ctx,
		/*sql*/ `INSERT INTO test_tx (id, val) VALUES (?, ?)`, 1, "committed",
	)
	require.NoError(t, err)

	// Verify row visible within transaction
	rows := txConn.Query(ctx,
		/*sql*/ `SELECT val FROM test_tx WHERE id = ?`, 1,
	)
	require.True(t, rows.Next())
	var val string
	require.NoError(t, rows.Scan(&val))
	assert.Equal(t, "committed", val)
	require.NoError(t, rows.Close())

	err = txConn.Commit()
	require.NoError(t, err)

	// Verify row visible after commit
	rows = conn.Query(ctx,
		/*sql*/ `SELECT val FROM test_tx WHERE id = ?`, 1,
	)
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&val))
	assert.Equal(t, "committed", val)
	require.NoError(t, rows.Close())
}

func TestTransactionRollback(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE IF NOT EXISTS test_tx_rollback (id INT PRIMARY KEY, val TEXT)`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE IF EXISTS test_tx_rollback`,
	)

	txConn, err := conn.Begin(ctx, 1, nil)
	require.NoError(t, err)

	err = txConn.Exec(ctx,
		/*sql*/ `INSERT INTO test_tx_rollback (id, val) VALUES (?, ?)`, 1, "rolled-back",
	)
	require.NoError(t, err)

	err = txConn.Rollback()
	require.NoError(t, err)

	// Verify row is absent after rollback
	rows := conn.Query(ctx,
		/*sql*/ `SELECT val FROM test_tx_rollback WHERE id = ?`, 1,
	)
	assert.False(t, rows.Next())
	require.NoError(t, rows.Close())
}

func TestInsertRowStruct(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE IF NOT EXISTS test_insert_struct (id INT PRIMARY KEY, name TEXT, score INT)`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE IF EXISTS test_insert_struct`,
	)

	type Row struct {
		sqldb.TableName `db:"test_insert_struct"`

		ID    int    `db:"id,primarykey"`
		Name  string `db:"name"`
		Score int    `db:"score"`
	}

	row := Row{ID: 1, Name: "alice", Score: 100}
	err = sqldb.InsertRowStruct(ctx, conn, refl, mysqlconn.QueryBuilder{}, conn, &row)
	require.NoError(t, err)

	// Verify the inserted row
	rows := conn.Query(ctx,
		/*sql*/ `SELECT id, name, score FROM test_insert_struct WHERE id = ?`, 1,
	)
	require.True(t, rows.Next())
	var got Row
	require.NoError(t, rows.Scan(&got.ID, &got.Name, &got.Score))
	assert.Equal(t, row, got)
	require.NoError(t, rows.Close())
}

func TestQueryRowScanStruct(t *testing.T) {
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE IF NOT EXISTS test_scan_struct (id INT PRIMARY KEY, label TEXT, amount INT)`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE IF EXISTS test_scan_struct`,
	)

	err = conn.Exec(ctx,
		/*sql*/ `INSERT INTO test_scan_struct (id, label, amount) VALUES (?, ?, ?)`, 42, "widgets", 99,
	)
	require.NoError(t, err)

	type Row struct {
		sqldb.TableName `db:"test_scan_struct"`

		ID     int    `db:"id,primarykey"`
		Label  string `db:"label"`
		Amount int    `db:"amount"`
	}

	var got Row
	err = sqldb.QueryRow(ctx, conn, refl, conn,
		/*sql*/ `SELECT id, label, amount FROM test_scan_struct WHERE id = ?`, 42,
	).Scan(&got)
	require.NoError(t, err)
	assert.Equal(t, Row{ID: 42, Label: "widgets", Amount: 99}, got)
}
