package oraconn

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	_ "github.com/sijms/go-ora/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/oraconn"
)

var (
	oracleUser     = envOrDefault("ORACLE_USER", "testuser")
	oraclePassword = envOrDefault("ORACLE_PASSWORD", "TestPass123")
	oracleHost     = envOrDefault("ORACLE_HOST", "localhost")
	oraclePort     = envOrDefaultInt("ORACLE_PORT", 1522)
	oracleService  = envOrDefault("ORACLE_SERVICE", "FREEPDB1")

	refl = sqldb.NewTaggedStructReflector()
)

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envOrDefaultInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func testConfig() *sqldb.ConnConfig {
	return &sqldb.ConnConfig{
		Driver:   oraconn.Driver,
		Host:     oracleHost,
		Port:     uint16(oraclePort),
		User:     oracleUser,
		Password: oraclePassword,
		Database: oracleService,
	}
}

func dockerComposeUp() error {
	return exec.Command("docker", "compose", "up", "-d").Run()
}

func waitForOracle() error {
	dsn := fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		oracleUser, oraclePassword, oracleHost, oraclePort, oracleService)
	for range 120 {
		db, err := sql.Open("oracle", dsn)
		if err == nil {
			err = db.Ping()
			db.Close()
			if err == nil {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("Oracle not ready after 120 seconds")
}

func dropAllTables() error {
	ctx := context.Background()
	conn, err := oraconn.Connect(ctx, testConfig(), true)
	if err != nil {
		return err
	}
	defer conn.Close()
	return oraconn.DropAll(ctx, conn)
}

func TestMain(m *testing.M) {
	if os.Getenv("CI") == "" {
		err := dockerComposeUp()
		if err != nil {
			log.Fatalf("Failed to start Docker Compose: %v", err)
		}
	}

	err := waitForOracle()
	if err != nil {
		log.Fatalf("Failed waiting for Oracle: %v", err)
	}

	err = dropAllTables()
	if err != nil {
		log.Fatalf("Failed to drop all tables before tests: %v", err)
	}

	m.Run()
}

func TestConnect(t *testing.T) {
	conn, err := oraconn.Connect(t.Context(), testConfig(), true)
	require.NoError(t, err)
	defer conn.Close()

	rows := conn.Query(t.Context(),
		/*sql*/ `SELECT 1 FROM DUAL`,
	)
	require.True(t, rows.Next())
	var result int
	require.NoError(t, rows.Scan(&result))
	assert.Equal(t, 1, result)
	require.NoError(t, rows.Close())
}

func TestMustConnectPanics(t *testing.T) {
	badConfig := &sqldb.ConnConfig{
		Driver:   oraconn.Driver,
		Host:     "invalid-host-that-does-not-exist",
		Port:     9999,
		User:     "nobody",
		Password: "Nothing123",
		Database: "NONEXISTENT",
	}
	assert.Panics(t, func() {
		oraconn.MustConnect(t.Context(), badConfig, true)
	})
}

func TestExec(t *testing.T) {
	conn, err := oraconn.Connect(t.Context(), testConfig(), true)
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE test_exec (id NUMBER(10) PRIMARY KEY, val VARCHAR2(255))`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE test_exec`,
	)

	err = conn.Exec(ctx,
		/*sql*/ `INSERT INTO test_exec (id, val) VALUES (:1, :2)`, 1, "hello",
	)
	require.NoError(t, err)

	rows := conn.Query(ctx,
		/*sql*/ `SELECT val FROM test_exec WHERE id = :1`, 1,
	)
	require.True(t, rows.Next())
	var val string
	require.NoError(t, rows.Scan(&val))
	assert.Equal(t, "hello", val)
	require.NoError(t, rows.Close())
}

func TestQueryRow(t *testing.T) {
	conn, err := oraconn.Connect(t.Context(), testConfig(), true)
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE test_queryrow (id NUMBER(10) PRIMARY KEY, val VARCHAR2(255))`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE test_queryrow`,
	)

	err = conn.Exec(ctx,
		/*sql*/ `INSERT INTO test_queryrow (id, val) VALUES (:1, :2)`, 1, "alpha",
	)
	require.NoError(t, err)
	err = conn.Exec(ctx,
		/*sql*/ `INSERT INTO test_queryrow (id, val) VALUES (:1, :2)`, 2, "beta",
	)
	require.NoError(t, err)

	rows := conn.Query(ctx,
		/*sql*/ `SELECT val FROM test_queryrow WHERE id = :1`, 2,
	)
	require.True(t, rows.Next())
	var val string
	require.NoError(t, rows.Scan(&val))
	assert.Equal(t, "beta", val)
	assert.False(t, rows.Next())
	require.NoError(t, rows.Close())
}

func TestQueryRows(t *testing.T) {
	conn, err := oraconn.Connect(t.Context(), testConfig(), true)
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE test_queryrows (id NUMBER(10) PRIMARY KEY, val NUMBER(10))`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE test_queryrows`,
	)

	err = conn.Exec(ctx,
		/*sql*/ `INSERT INTO test_queryrows (id, val) VALUES (:1, :2)`, 1, 10,
	)
	require.NoError(t, err)
	err = conn.Exec(ctx,
		/*sql*/ `INSERT INTO test_queryrows (id, val) VALUES (:1, :2)`, 2, 20,
	)
	require.NoError(t, err)
	err = conn.Exec(ctx,
		/*sql*/ `INSERT INTO test_queryrows (id, val) VALUES (:1, :2)`, 3, 30,
	)
	require.NoError(t, err)

	rows := conn.Query(ctx,
		/*sql*/ `SELECT val FROM test_queryrows WHERE val > :1 ORDER BY val`, 15,
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
	conn, err := oraconn.Connect(t.Context(), testConfig(), true)
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE test_tx (id NUMBER(10) PRIMARY KEY, val VARCHAR2(255))`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE test_tx`,
	)

	txConn, err := conn.Begin(ctx, 1, nil)
	require.NoError(t, err)

	err = txConn.Exec(ctx,
		/*sql*/ `INSERT INTO test_tx (id, val) VALUES (:1, :2)`, 1, "committed",
	)
	require.NoError(t, err)

	// Verify row visible within transaction
	rows := txConn.Query(ctx,
		/*sql*/ `SELECT val FROM test_tx WHERE id = :1`, 1,
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
		/*sql*/ `SELECT val FROM test_tx WHERE id = :1`, 1,
	)
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&val))
	assert.Equal(t, "committed", val)
	require.NoError(t, rows.Close())
}

func TestTransactionRollback(t *testing.T) {
	conn, err := oraconn.Connect(t.Context(), testConfig(), true)
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE test_tx_rollback (id NUMBER(10) PRIMARY KEY, val VARCHAR2(255))`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE test_tx_rollback`,
	)

	txConn, err := conn.Begin(ctx, 1, nil)
	require.NoError(t, err)

	err = txConn.Exec(ctx,
		/*sql*/ `INSERT INTO test_tx_rollback (id, val) VALUES (:1, :2)`, 1, "rolled-back",
	)
	require.NoError(t, err)

	err = txConn.Rollback()
	require.NoError(t, err)

	// Verify row is absent after rollback
	rows := conn.Query(ctx,
		/*sql*/ `SELECT val FROM test_tx_rollback WHERE id = :1`, 1,
	)
	assert.False(t, rows.Next())
	require.NoError(t, rows.Close())
}

func TestInsertRowStruct(t *testing.T) {
	conn, err := oraconn.Connect(t.Context(), testConfig(), true)
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE test_insert_struct (id NUMBER(10) PRIMARY KEY, name VARCHAR2(255), score NUMBER(10))`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE test_insert_struct`,
	)

	type Row struct {
		sqldb.TableName `db:"test_insert_struct"`

		ID    int    `db:"id,primarykey"`
		Name  string `db:"name"`
		Score int    `db:"score"`
	}

	row := Row{ID: 1, Name: "alice", Score: 100}
	err = sqldb.InsertRowStruct(ctx, conn, refl, oraconn.QueryBuilder{}, conn, &row)
	require.NoError(t, err)

	// Verify the inserted row
	rows := conn.Query(ctx,
		/*sql*/ `SELECT id, name, score FROM test_insert_struct WHERE id = :1`, 1,
	)
	require.True(t, rows.Next())
	var got Row
	require.NoError(t, rows.Scan(&got.ID, &got.Name, &got.Score))
	assert.Equal(t, row, got)
	require.NoError(t, rows.Close())
}

func TestQueryRowScanStruct(t *testing.T) {
	conn, err := oraconn.Connect(t.Context(), testConfig(), true)
	require.NoError(t, err)
	defer conn.Close()

	ctx := t.Context()

	err = conn.Exec(ctx,
		/*sql*/ `CREATE TABLE test_scan_struct (id NUMBER(10) PRIMARY KEY, label VARCHAR2(255), amount NUMBER(10))`,
	)
	require.NoError(t, err)
	defer conn.Exec(ctx, //nolint:errcheck
		/*sql*/ `DROP TABLE test_scan_struct`,
	)

	err = conn.Exec(ctx,
		/*sql*/ `INSERT INTO test_scan_struct (id, label, amount) VALUES (:1, :2, :3)`, 42, "widgets", 99,
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
		/*sql*/ `SELECT id, label, amount FROM test_scan_struct WHERE id = :1`, 42,
	).Scan(&got)
	require.NoError(t, err)
	assert.Equal(t, Row{ID: 42, Label: "widgets", Amount: 99}, got)
}
