package mssqlconn

import (
	"database/sql"
	"fmt"
	"log"
	"os/exec"
	"testing"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/mssqlconn"
)

const (
	mssqlUser     = "sa"
	mssqlPassword = "TestPass123!"
	mssqlHost     = "localhost"
	mssqlPort     = 1434
	dbName        = "testdb"
)

func testConfig() *sqldb.ConnConfig {
	return &sqldb.ConnConfig{
		Driver:   mssqlconn.Driver,
		Host:     mssqlHost,
		Port:     uint16(mssqlPort),
		User:     mssqlUser,
		Password: mssqlPassword,
		Database: dbName,
		Extra: map[string]string{
			"encrypt": "disable",
		},
	}
}

func dockerComposeUp() error {
	return exec.Command("docker", "compose", "up", "-d").Run()
}

func waitForMSSQL() error {
	// Connect to master first to check readiness
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=master&encrypt=disable",
		mssqlUser, mssqlPassword, mssqlHost, mssqlPort)
	for range 60 {
		db, err := sql.Open("sqlserver", dsn)
		if err == nil {
			err = db.Ping()
			db.Close()
			if err == nil {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("SQL Server not ready after 60 seconds")
}

func ensureTestDB() error {
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=master&encrypt=disable",
		mssqlUser, mssqlPassword, mssqlHost, mssqlPort)
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create test database if it doesn't exist
	_, err = db.Exec(fmt.Sprintf(
		/*sql*/ `IF NOT EXISTS (SELECT name FROM sys.databases WHERE name = '%s') CREATE DATABASE [%s]`,
		dbName, dbName,
	))
	return err
}

func dropAllTables() error {
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s&encrypt=disable",
		mssqlUser, mssqlPassword, mssqlHost, mssqlPort, dbName)
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	// Drop all user tables
	_, err = db.Exec( /*sql*/ `
		DECLARE @sql NVARCHAR(MAX) = '';
		SELECT @sql += 'DROP TABLE [' + TABLE_SCHEMA + '].[' + TABLE_NAME + '];'
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_TYPE = 'BASE TABLE';
		IF @sql <> '' EXEC sp_executesql @sql;
	`)
	return err
}

func TestMain(m *testing.M) {
	err := dockerComposeUp()
	if err != nil {
		log.Fatalf("Failed to start Docker Compose: %v", err)
	}

	err = waitForMSSQL()
	if err != nil {
		log.Fatalf("Failed waiting for SQL Server: %v", err)
	}

	err = ensureTestDB()
	if err != nil {
		log.Fatalf("Failed to create test database: %v", err)
	}

	err = dropAllTables()
	if err != nil {
		log.Fatalf("Failed to drop all tables before tests: %v", err)
	}

	m.Run()
}

func TestConnect(t *testing.T) {
	conn, err := mssqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	rows := conn.Query(t.Context(), /*sql*/ `SELECT 1`)
	require.True(t, rows.Next())
	var result int
	require.NoError(t, rows.Scan(&result))
	assert.Equal(t, 1, result)
	require.NoError(t, rows.Close())
}

func TestConnectExt(t *testing.T) {
	connExt, err := mssqlconn.ConnectExt(t.Context(), testConfig(), sqldb.NewTaggedStructReflector())
	require.NoError(t, err)
	defer connExt.Close()

	rows := connExt.Query(t.Context(), /*sql*/ `SELECT 1`)
	require.True(t, rows.Next())
	var result int
	require.NoError(t, rows.Scan(&result))
	assert.Equal(t, 1, result)
	require.NoError(t, rows.Close())
}

func TestMustConnectPanics(t *testing.T) {
	badConfig := &sqldb.ConnConfig{
		Driver:   mssqlconn.Driver,
		Host:     "invalid-host-that-does-not-exist",
		Port:     9999,
		User:     "nobody",
		Password: "Nothing123!",
		Database: "nodb",
	}
	assert.Panics(t, func() {
		mssqlconn.MustConnect(t.Context(), badConfig)
	})
}

func TestMustConnectExtPanics(t *testing.T) {
	badConfig := &sqldb.ConnConfig{
		Driver:   mssqlconn.Driver,
		Host:     "invalid-host-that-does-not-exist",
		Port:     9999,
		User:     "nobody",
		Password: "Nothing123!",
		Database: "nodb",
	}
	assert.Panics(t, func() {
		mssqlconn.MustConnectExt(t.Context(), badConfig, sqldb.NewTaggedStructReflector())
	})
}

func TestNewConnExt(t *testing.T) {
	conn, err := mssqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	connExt := mssqlconn.NewConnExt(conn, sqldb.NewTaggedStructReflector())
	require.NotNil(t, connExt)

	rows := connExt.Query(t.Context(), /*sql*/ `SELECT 1`)
	require.True(t, rows.Next())
	var result int
	require.NoError(t, rows.Scan(&result))
	assert.Equal(t, 1, result)
	require.NoError(t, rows.Close())
}

func TestExec(t *testing.T) {
	conn, err := mssqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx, /*sql*/ `
		IF OBJECT_ID('test_exec', 'U') IS NOT NULL DROP TABLE test_exec;
		CREATE TABLE test_exec (id INT PRIMARY KEY, val NVARCHAR(255))
	`)
	require.NoError(t, err)
	defer conn.Exec(ctx, /*sql*/ `DROP TABLE IF EXISTS test_exec`) //nolint:errcheck

	err = conn.Exec(ctx, /*sql*/ `INSERT INTO test_exec (id, val) VALUES (@p1, @p2)`, 1, "hello")
	require.NoError(t, err)

	rows := conn.Query(ctx, /*sql*/ `SELECT val FROM test_exec WHERE id = @p1`, 1)
	require.True(t, rows.Next())
	var val string
	require.NoError(t, rows.Scan(&val))
	assert.Equal(t, "hello", val)
	require.NoError(t, rows.Close())
}

func TestQueryRow(t *testing.T) {
	conn, err := mssqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx, /*sql*/ `
		IF OBJECT_ID('test_queryrow', 'U') IS NOT NULL DROP TABLE test_queryrow;
		CREATE TABLE test_queryrow (id INT PRIMARY KEY, val NVARCHAR(255))
	`)
	require.NoError(t, err)
	defer conn.Exec(ctx, /*sql*/ `DROP TABLE IF EXISTS test_queryrow`) //nolint:errcheck

	err = conn.Exec(ctx, /*sql*/ `INSERT INTO test_queryrow (id, val) VALUES (@p1, @p2), (@p3, @p4)`, 1, "alpha", 2, "beta")
	require.NoError(t, err)

	rows := conn.Query(ctx, /*sql*/ `SELECT val FROM test_queryrow WHERE id = @p1`, 2)
	require.True(t, rows.Next())
	var val string
	require.NoError(t, rows.Scan(&val))
	assert.Equal(t, "beta", val)
	assert.False(t, rows.Next())
	require.NoError(t, rows.Close())
}

func TestQueryRows(t *testing.T) {
	conn, err := mssqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx, /*sql*/ `
		IF OBJECT_ID('test_queryrows', 'U') IS NOT NULL DROP TABLE test_queryrows;
		CREATE TABLE test_queryrows (id INT PRIMARY KEY, val INT)
	`)
	require.NoError(t, err)
	defer conn.Exec(ctx, /*sql*/ `DROP TABLE IF EXISTS test_queryrows`) //nolint:errcheck

	err = conn.Exec(ctx, /*sql*/ `INSERT INTO test_queryrows (id, val) VALUES (@p1, @p2), (@p3, @p4), (@p5, @p6)`, 1, 10, 2, 20, 3, 30)
	require.NoError(t, err)

	rows := conn.Query(ctx, /*sql*/ `SELECT val FROM test_queryrows WHERE val > @p1 ORDER BY val`, 15)
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
	conn, err := mssqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx, /*sql*/ `
		IF OBJECT_ID('test_tx', 'U') IS NOT NULL DROP TABLE test_tx;
		CREATE TABLE test_tx (id INT PRIMARY KEY, val NVARCHAR(255))
	`)
	require.NoError(t, err)
	defer conn.Exec(ctx, /*sql*/ `DROP TABLE IF EXISTS test_tx`) //nolint:errcheck

	txConn, err := conn.Begin(ctx, 1, nil)
	require.NoError(t, err)

	err = txConn.Exec(ctx, /*sql*/ `INSERT INTO test_tx (id, val) VALUES (@p1, @p2)`, 1, "committed")
	require.NoError(t, err)

	// Verify row visible within transaction
	rows := txConn.Query(ctx, /*sql*/ `SELECT val FROM test_tx WHERE id = @p1`, 1)
	require.True(t, rows.Next())
	var val string
	require.NoError(t, rows.Scan(&val))
	assert.Equal(t, "committed", val)
	require.NoError(t, rows.Close())

	err = txConn.Commit()
	require.NoError(t, err)

	// Verify row visible after commit
	rows = conn.Query(ctx, /*sql*/ `SELECT val FROM test_tx WHERE id = @p1`, 1)
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&val))
	assert.Equal(t, "committed", val)
	require.NoError(t, rows.Close())
}

func TestTransactionRollback(t *testing.T) {
	conn, err := mssqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx, /*sql*/ `
		IF OBJECT_ID('test_tx_rollback', 'U') IS NOT NULL DROP TABLE test_tx_rollback;
		CREATE TABLE test_tx_rollback (id INT PRIMARY KEY, val NVARCHAR(255))
	`)
	require.NoError(t, err)
	defer conn.Exec(ctx, /*sql*/ `DROP TABLE IF EXISTS test_tx_rollback`) //nolint:errcheck

	txConn, err := conn.Begin(ctx, 1, nil)
	require.NoError(t, err)

	err = txConn.Exec(ctx, /*sql*/ `INSERT INTO test_tx_rollback (id, val) VALUES (@p1, @p2)`, 1, "rolled-back")
	require.NoError(t, err)

	err = txConn.Rollback()
	require.NoError(t, err)

	// Verify row is absent after rollback
	rows := conn.Query(ctx, /*sql*/ `SELECT val FROM test_tx_rollback WHERE id = @p1`, 1)
	assert.False(t, rows.Next())
	require.NoError(t, rows.Close())
}

func TestInsertRowStruct(t *testing.T) {
	connExt, err := mssqlconn.ConnectExt(t.Context(), testConfig(), sqldb.NewTaggedStructReflector())
	require.NoError(t, err)
	defer connExt.Close()

	ctx := t.Context()

	err = connExt.Exec(ctx, /*sql*/ `
		IF OBJECT_ID('test_insert_struct', 'U') IS NOT NULL DROP TABLE test_insert_struct;
		CREATE TABLE test_insert_struct (id INT PRIMARY KEY, name NVARCHAR(255), score INT)
	`)
	require.NoError(t, err)
	defer connExt.Exec(ctx, /*sql*/ `DROP TABLE IF EXISTS test_insert_struct`) //nolint:errcheck

	type Row struct {
		sqldb.TableName `db:"test_insert_struct"`

		ID    int    `db:"id,primarykey"`
		Name  string `db:"name"`
		Score int    `db:"score"`
	}

	row := Row{ID: 1, Name: "alice", Score: 100}
	err = sqldb.InsertRowStruct(ctx, connExt, &row)
	require.NoError(t, err)

	// Verify the inserted row
	rows := connExt.Query(ctx, /*sql*/ `SELECT id, name, score FROM test_insert_struct WHERE id = @p1`, 1)
	require.True(t, rows.Next())
	var got Row
	require.NoError(t, rows.Scan(&got.ID, &got.Name, &got.Score))
	assert.Equal(t, row, got)
	require.NoError(t, rows.Close())
}

func TestQueryRowScanStruct(t *testing.T) {
	connExt, err := mssqlconn.ConnectExt(t.Context(), testConfig(), sqldb.NewTaggedStructReflector())
	require.NoError(t, err)
	defer connExt.Close()

	ctx := t.Context()

	err = connExt.Exec(ctx, /*sql*/ `
		IF OBJECT_ID('test_scan_struct', 'U') IS NOT NULL DROP TABLE test_scan_struct;
		CREATE TABLE test_scan_struct (id INT PRIMARY KEY, label NVARCHAR(255), amount INT)
	`)
	require.NoError(t, err)
	defer connExt.Exec(ctx, /*sql*/ `DROP TABLE IF EXISTS test_scan_struct`) //nolint:errcheck

	err = connExt.Exec(ctx, /*sql*/ `INSERT INTO test_scan_struct (id, label, amount) VALUES (@p1, @p2, @p3)`, 42, "widgets", 99)
	require.NoError(t, err)

	type Row struct {
		sqldb.TableName `db:"test_scan_struct"`

		ID     int    `db:"id,primarykey"`
		Label  string `db:"label"`
		Amount int    `db:"amount"`
	}

	var got Row
	err = sqldb.QueryRow(ctx, connExt, /*sql*/ `SELECT id, label, amount FROM test_scan_struct WHERE id = @p1`, 42).Scan(&got)
	require.NoError(t, err)
	assert.Equal(t, Row{ID: 42, Label: "widgets", Amount: 99}, got)
}
