// Package conntest provides a shared integration test suite
// for sqldb.Connection implementations.
// Vendor-specific test packages call conntest.RunAll
// with a Config that provides the connection factory and DDL.
package conntest

import (
	"context"
	"database/sql"
	"net/mail"
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

	// SupportsReadOnlyTransaction indicates whether the driver enforces
	// read-only transactions at the SQL execution level.
	// If false, the ReadOnlyTransactionRejectsWrite test is skipped.
	SupportsReadOnlyTransaction bool

	// SupportsCustomIsolationLevel indicates whether the driver supports
	// setting a non-default isolation level in sql.TxOptions.
	// If false, the TransactionIsolation test uses nil opts.
	SupportsCustomIsolationLevel bool

	// ExecAfterClosedTxErrors indicates whether the driver returns an error
	// when executing on a committed or rolled-back transaction.
	// Drivers wrapping *sql.Tx return sql.ErrTxDone;
	// drivers with custom connection management (e.g., SQLite) may not.
	ExecAfterClosedTxErrors bool

	// Information records which Information interface features are
	// supported by this driver. Used by the Information test group.
	Information InformationFeatures
}

func (c *Config) selectOneQuery() string {
	if c.SelectOneQuery != "" {
		return c.SelectOneQuery
	}
	return /*sql*/ `SELECT 1`
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

	// CreateMailAddressTable creates a table with columns:
	//   id (int PK), email (text).
	// The table name MUST be "conntest_mail_address".
	CreateMailAddressTable string

	// CreateInfoParent creates a parent table for the Information test
	// group with a composite primary key declared in NON-declaration
	// order:
	//   id1 int, id2 int, PRIMARY KEY (id2, id1).
	// The PK ordering test asserts the constraint order is (id2, id1)
	// even though declaration order is (id1, id2). The table name MUST
	// be "conntest_info_parent". May be empty to skip the Information
	// tests for this driver.
	CreateInfoParent string

	// CreateInfoChild creates a child table for the Information test
	// group with one PK column and a composite foreign key referencing
	// conntest_info_parent in (id2, id1) order:
	//   child_id int PK, parent_id1 int, parent_id2 int,
	//   FOREIGN KEY (parent_id2, parent_id1)
	//     REFERENCES conntest_info_parent (id2, id1) ON DELETE CASCADE.
	// The table name MUST be "conntest_info_child". May be empty.
	CreateInfoChild string

	// CreateInfoView creates a view over conntest_info_parent named
	// "conntest_info_view" with columns (id1, id2). May be empty.
	CreateInfoView string

	// CreateInfoGenerated creates a table for the Information test
	// group exercising the [sqldb.ColumnInfo.Generated] flag and the
	// "expression default is NOT generated" boundary. The shape MUST be:
	//   id         <int>         PRIMARY KEY
	//   gen_col    <int>         GENERATED ALWAYS AS (id + 1) [STORED|VIRTUAL]
	//   created_at <timestamp>   DEFAULT <vendor-current-timestamp-expr>
	// The table name MUST be "conntest_info_generated". Each driver
	// supplies its own DDL using the vendor's generated-column and
	// expression-default syntax. created_at exists to verify that
	// columns with non-literal defaults (e.g. MySQL 8.0.13+ writes
	// "DEFAULT_GENERATED" into information_schema.columns.extra) are
	// NOT misclassified as Generated/ReadOnly. May be empty to skip
	// the Generated assertions on a driver that does not support
	// generated columns.
	CreateInfoGenerated string
}

// InformationFeatures records which Information methods are
// implemented for this driver. Tests skip the corresponding
// assertions when a feature is reported unsupported.
type InformationFeatures struct {
	// SupportsRoutines is false on drivers that have no concept of
	// stored routines (SQLite). When false, Routines and RoutineExists
	// are expected to return errors.ErrUnsupported.
	SupportsRoutines bool

	// CaseFoldsToUpper is true on drivers that fold unquoted
	// identifiers to uppercase in the catalog (Oracle). Tests
	// uppercase their expected names when comparing on these drivers.
	CaseFoldsToUpper bool

	// SchemaIsAttachedDB is true on drivers where "schema" really
	// means an attached database (SQLite). Tests skip schema-name
	// assertions that don't apply.
	SchemaIsAttachedDB bool
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

type mailAddressRow struct {
	sqldb.TableName `db:"conntest_mail_address"`

	ID    int           `db:"id,primarykey"`
	Email *mail.Address `db:"email"`
}

var refl = sqldb.NewTaggedStructReflector()
var mailAddressRefl = sqldb.NewTaggedStructReflector(sqldb.MailAddressTypeWrapper{})

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
	t.Run("QueryCallback", func(t *testing.T) { runQueryCallbackTests(t, config) })
	t.Run("Batch", func(t *testing.T) { runBatchTests(t, config) })
	t.Run("MailAddress", func(t *testing.T) { runMailAddressTests(t, config) })
	t.Run("Information", func(t *testing.T) { runInformationTests(t, config) })
}

// setupTable drops the table if it exists, creates it using the given DDL,
// and registers cleanup to drop it again. The cleanup runs against a
// context.WithoutCancel-derived context so the DROP still executes
// even after the test's own context has been cancelled.
func setupTable(t *testing.T, conn sqldb.Connection, createDDL, tableName string) {
	t.Helper()
	ctx := t.Context()
	_ = conn.Exec(ctx,
		/*sql*/ `DROP TABLE IF EXISTS `+tableName,
	)
	err := conn.Exec(ctx, createDDL)
	require.NoError(t, err, "creating table %s", tableName)
	cleanupCtx := context.WithoutCancel(ctx)
	t.Cleanup(func() {
		_ = conn.Exec(cleanupCtx,
			/*sql*/ `DROP TABLE IF EXISTS `+tableName,
		)
	})
}

// insertSimpleRow inserts a simpleRow using sqldb.InsertRowStruct.
func insertSimpleRow(t *testing.T, conn sqldb.Connection, qb sqldb.QueryBuilder, row simpleRow) {
	t.Helper()
	err := sqldb.InsertRowStruct(t.Context(), conn, refl, qb, conn, &row)
	require.NoError(t, err)
}

// querySimpleRow queries a simpleRow by PK using sqldb.QueryRowStruct.
func querySimpleRow(t *testing.T, conn sqldb.Connection, qb sqldb.QueryBuilder, id int) simpleRow {
	t.Helper()
	row, err := sqldb.QueryRowStruct[simpleRow](t.Context(), conn, refl, qb, conn, id)
	require.NoError(t, err)
	return row
}

// insertUpsertRow inserts an upsertRow using sqldb.InsertRowStruct.
func insertUpsertRow(t *testing.T, conn sqldb.Connection, qb sqldb.QueryBuilder, row upsertRow) {
	t.Helper()
	err := sqldb.InsertRowStruct(t.Context(), conn, refl, qb, conn, &row)
	require.NoError(t, err)
}

// queryUpsertRow queries an upsertRow by PK using sqldb.QueryRowStruct.
func queryUpsertRow(t *testing.T, conn sqldb.Connection, qb sqldb.QueryBuilder, id int) upsertRow {
	t.Helper()
	row, err := sqldb.QueryRowStruct[upsertRow](t.Context(), conn, refl, qb, conn, id)
	require.NoError(t, err)
	return row
}

// insertMailAddressRow inserts a mailAddressRow using sqldb.InsertRowStruct.
func insertMailAddressRow(t *testing.T, conn sqldb.Connection, qb sqldb.QueryBuilder, row mailAddressRow) {
	t.Helper()
	err := sqldb.InsertRowStruct(t.Context(), conn, mailAddressRefl, qb, conn, &row)
	require.NoError(t, err)
}

// queryMailAddressRow queries a mailAddressRow by PK using sqldb.QueryRowStruct.
func queryMailAddressRow(t *testing.T, conn sqldb.Connection, qb sqldb.QueryBuilder, id int) mailAddressRow {
	t.Helper()
	row, err := sqldb.QueryRowStruct[mailAddressRow](t.Context(), conn, mailAddressRefl, qb, conn, id)
	require.NoError(t, err)
	return row
}
