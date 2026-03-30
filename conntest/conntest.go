// Package conntest provides a shared integration test suite
// for sqldb.Connection implementations.
// Vendor-specific test packages call conntest.RunAll
// with a Config that provides the connection factory and DDL.
package conntest

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

// Config configures the shared integration test suite.
type Config struct {
	// NewConn creates a fresh Connection for a test.
	// The returned connection will be closed via t.Cleanup.
	NewConn func(t *testing.T) sqldb.Connection

	// QueryBuilder returns the vendor-specific QueryBuilder.
	QueryBuilder sqldb.QueryBuilder

	// DDL provides vendor-specific CREATE TABLE statements.
	DDL DDL

	// DefaultIsolationLevel is the expected default isolation level
	// for this vendor (e.g., sql.LevelReadCommitted for PostgreSQL).
	DefaultIsolationLevel sql.IsolationLevel

	// DriverName is the expected driver name from Config().Driver.
	DriverName string

	// DatabaseName is the expected database name from Config().Database.
	DatabaseName string

	// SelectOneQuery is the SQL to select the literal 1.
	// Defaults to "SELECT 1" if empty.
	// Oracle needs "SELECT 1 FROM DUAL".
	SelectOneQuery string
}

func (c *Config) selectOneQuery() string {
	if c.SelectOneQuery != "" {
		return c.SelectOneQuery
	}
	return "SELECT 1"
}

// DDL holds vendor-specific CREATE TABLE statements for the test suite.
type DDL struct {
	// CreateSimpleTable creates a table with columns: id (int PK), val (text).
	// The table name MUST be "conntest_simple".
	CreateSimpleTable string

	// CreateUpsertTable creates a table with columns:
	//   id (int PK), name (text NOT NULL), score (int NOT NULL DEFAULT 0).
	// The table name MUST be "conntest_upsert".
	CreateUpsertTable string

	// CreateReturningTable creates a table with columns:
	//   id (auto-increment int PK), name (text NOT NULL), score (int NOT NULL DEFAULT 0).
	// The table name MUST be "conntest_returning".
	// May be empty if the vendor does not support ReturningQueryBuilder.
	CreateReturningTable string
}

// Shared struct types for test tables.
type simpleRow struct {
	sqldb.TableName `db:"conntest_simple"`

	ID  int    `db:"id,primarykey"`
	Val string `db:"val"`
}

type upsertRow struct {
	sqldb.TableName `db:"conntest_upsert"`

	ID    int    `db:"id,primarykey"`
	Name  string `db:"name"`
	Score int    `db:"score"`
}

var refl = sqldb.NewTaggedStructReflector()

// RunAll runs the full shared Connection integration test suite.
func RunAll(t *testing.T, config Config) {
	t.Helper()

	t.Run("Basic", func(t *testing.T) { runBasicTests(t, config) })
	t.Run("Exec", func(t *testing.T) { runExecTests(t, config) })
	t.Run("Query", func(t *testing.T) { runQueryTests(t, config) })
	t.Run("Prepare", func(t *testing.T) { runPrepareTests(t, config) })
	t.Run("Transaction", func(t *testing.T) { runTransactionTests(t, config) })
	t.Run("QueryBuilder", func(t *testing.T) { runQueryBuilderTests(t, config) })
	t.Run("Upsert", func(t *testing.T) { runUpsertTests(t, config) })
	t.Run("Returning", func(t *testing.T) { runReturningTests(t, config) })
}

// setupTable drops the table if it exists, creates it using the given DDL,
// and registers cleanup to drop it again.
func setupTable(t *testing.T, conn sqldb.Connection, createDDL, tableName string) {
	t.Helper()
	ctx := t.Context()
	_ = conn.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
	err := conn.Exec(ctx, createDDL)
	require.NoError(t, err, "creating table %s", tableName)
	t.Cleanup(func() {
		_ = conn.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
	})
}

// insertSimpleRow inserts a simpleRow using sqldb.InsertRowStruct.
func insertSimpleRow(t *testing.T, conn sqldb.Connection, qb sqldb.QueryBuilder, row simpleRow) {
	t.Helper()
	err := sqldb.InsertRowStruct(t.Context(), conn, refl, qb, conn, &row)
	require.NoError(t, err)
}

// querySimpleRow queries a simpleRow by PK using sqldb.QueryRowByPK.
func querySimpleRow(t *testing.T, conn sqldb.Connection, qb sqldb.QueryBuilder, id int) simpleRow {
	t.Helper()
	row, err := sqldb.QueryRowByPK[simpleRow](t.Context(), conn, refl, qb, conn, id)
	require.NoError(t, err)
	return row
}

// insertUpsertRow inserts an upsertRow using sqldb.InsertRowStruct.
func insertUpsertRow(t *testing.T, conn sqldb.Connection, qb sqldb.QueryBuilder, row upsertRow) {
	t.Helper()
	err := sqldb.InsertRowStruct(t.Context(), conn, refl, qb, conn, &row)
	require.NoError(t, err)
}

// queryUpsertRow queries an upsertRow by PK using sqldb.QueryRowByPK.
func queryUpsertRow(t *testing.T, conn sqldb.Connection, qb sqldb.QueryBuilder, id int) upsertRow {
	t.Helper()
	row, err := sqldb.QueryRowByPK[upsertRow](t.Context(), conn, refl, qb, conn, id)
	require.NoError(t, err)
	return row
}
