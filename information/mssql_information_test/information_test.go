// Package mssql_information_test runs the information_schema helpers
// against the dockerized SQL Server instance defined in
// ../../mssqlconn/test/docker-compose.yml.
package mssql_information_test

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

	_ "github.com/microsoft/go-mssqldb"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/information"
	"github.com/domonda/go-sqldb/mssqlconn"
)

var (
	mssqlUser     = cmp.Or(os.Getenv("MSSQL_USER"), "sa")
	mssqlPassword = cmp.Or(os.Getenv("MSSQL_PASSWORD"), "TestPass123!")
	mssqlHost     = cmp.Or(os.Getenv("MSSQL_HOST"), "localhost")
	mssqlPort     = cmp.Or(atoi(os.Getenv("MSSQL_PORT")), 1434)
	dbName        = cmp.Or(os.Getenv("MSSQL_DB"), "testdb")
)

const defaultSchema = "dbo"

var (
	testCtx  context.Context
	testConn sqldb.Connection
)

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

func waitForMSSQL() error {
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
	_, err = db.Exec(fmt.Sprintf(
		/*sql*/ `IF NOT EXISTS (SELECT name FROM sys.databases WHERE name = '%s') CREATE DATABASE [%s]`,
		dbName, dbName,
	))
	return err
}

func TestMain(m *testing.M) {
	if os.Getenv("CI") == "" {
		err := exec.Command("docker", "compose", "-f", "../../mssqlconn/test/docker-compose.yml", "up", "-d").Run()
		if err != nil {
			log.Fatalf("Failed to start Docker Compose: %v", err)
		}
	}

	if err := waitForMSSQL(); err != nil {
		log.Fatalf("Failed waiting for SQL Server: %v", err)
	}
	if err := ensureTestDB(); err != nil {
		log.Fatalf("Failed to create test database: %v", err)
	}

	ctx := context.Background()
	config := &sqldb.Config{
		Driver:   mssqlconn.Driver,
		Host:     mssqlHost,
		Port:     uint16(mssqlPort),
		User:     mssqlUser,
		Password: mssqlPassword,
		Database: dbName,
		Extra:    map[string]string{"encrypt": "disable"},
	}
	var err error
	testConn, err = mssqlconn.Connect(ctx, config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Drop in dependency order, then create. Names are prefixed
	// `info_test_*` to avoid collisions with `mssqlconn/test/`.
	//
	// info_test_profile.user_id is BOTH the primary key AND a foreign
	// key referencing info_test.id — it exercises the EXISTS=true branch
	// in GetPrimaryKeyColumns' foreign-key flag.
	for _, q := range []string{
		/*sql*/ `IF OBJECT_ID('dbo.info_test_profile', 'U') IS NOT NULL DROP TABLE dbo.info_test_profile`,
		/*sql*/ `IF OBJECT_ID('dbo.info_test_child', 'U') IS NOT NULL DROP TABLE dbo.info_test_child`,
		/*sql*/ `IF OBJECT_ID('dbo.info_test', 'U') IS NOT NULL DROP TABLE dbo.info_test`,
		/*sql*/ `
			CREATE TABLE dbo.info_test (
				id    INT NOT NULL PRIMARY KEY,
				name  NVARCHAR(255) NOT NULL,
				value NVARCHAR(255) NULL
			)
		`,
		/*sql*/ `
			CREATE TABLE dbo.info_test_child (
				id        INT NOT NULL PRIMARY KEY,
				parent_id INT NOT NULL,
				CONSTRAINT fk_info_test_child_parent FOREIGN KEY (parent_id) REFERENCES dbo.info_test(id)
			)
		`,
		/*sql*/ `
			CREATE TABLE dbo.info_test_profile (
				user_id INT NOT NULL PRIMARY KEY,
				CONSTRAINT fk_info_test_profile_user FOREIGN KEY (user_id) REFERENCES dbo.info_test(id)
			)
		`,
	} {
		if err := testConn.Exec(ctx, q); err != nil {
			log.Fatalf("Setup failed for %q: %v", q, err)
		}
	}

	testCtx = ctx

	m.Run()

	for _, q := range []string{
		/*sql*/ `IF OBJECT_ID('dbo.info_test_profile', 'U') IS NOT NULL DROP TABLE dbo.info_test_profile`,
		/*sql*/ `IF OBJECT_ID('dbo.info_test_child', 'U') IS NOT NULL DROP TABLE dbo.info_test_child`,
		/*sql*/ `IF OBJECT_ID('dbo.info_test', 'U') IS NOT NULL DROP TABLE dbo.info_test`,
	} {
		if err := testConn.Exec(ctx, q); err != nil {
			fmt.Fprintf(os.Stderr, "Cleanup failed for %q: %v\n", q, err)
		}
	}
}

func TestTableExists(t *testing.T) {
	exists, err := information.TableExists(testCtx, testConn, "info_test")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected info_test to exist")
	}

	exists, err = information.TableExists(testCtx, testConn, "nonexistent_table_xyz")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected nonexistent_table_xyz to not exist")
	}
}

func TestTableExists_WithSchema(t *testing.T) {
	exists, err := information.TableExists(testCtx, testConn, defaultSchema+".info_test")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Errorf("expected %s.info_test to exist", defaultSchema)
	}
}

func TestColumnExists(t *testing.T) {
	exists, err := information.ColumnExists(testCtx, testConn, "info_test", "name")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected column 'name' to exist in info_test")
	}

	exists, err = information.ColumnExists(testCtx, testConn, "info_test", "nonexistent_col")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected column 'nonexistent_col' to not exist")
	}
}

func TestColumnExists_WithSchema(t *testing.T) {
	exists, err := information.ColumnExists(testCtx, testConn, defaultSchema+".info_test", "id")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Errorf("expected column 'id' to exist in %s.info_test", defaultSchema)
	}
}

func TestGetTable(t *testing.T) {
	// SQL Server: catalog is the database name, schema is dbo by default.
	table, err := information.GetTable(testCtx, testConn, dbName, defaultSchema, "info_test")
	if err != nil {
		t.Fatal(err)
	}
	if table == nil {
		t.Fatal("expected non-nil table")
	}
	if string(table.TableName) != "info_test" {
		t.Errorf("expected table name 'info_test', got %q", table.TableName)
	}
	if string(table.TableSchema) != defaultSchema {
		t.Errorf("expected schema %q, got %q", defaultSchema, table.TableSchema)
	}
	if string(table.TableType) != "BASE TABLE" {
		t.Errorf("expected table type 'BASE TABLE', got %q", table.TableType)
	}
	// Note: SQL Server does not populate IsInsertableInto in
	// INFORMATION_SCHEMA.TABLES, so the field scans as the YesNo
	// zero value (false). We don't assert on it here.
}

func TestGetAllTables(t *testing.T) {
	tables, err := information.GetAllTables(testCtx, testConn)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) == 0 {
		t.Fatal("expected at least one table")
	}

	found := false
	for _, table := range tables {
		if string(table.TableName) == "info_test" && string(table.TableSchema) == defaultSchema {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find %s.info_test in all tables", defaultSchema)
	}
}

func TestGetPrimaryKeyColumns(t *testing.T) {
	cols, err := information.GetPrimaryKeyColumns(testCtx, testConn)
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) == 0 {
		t.Fatal("expected at least one primary key column")
	}

	parent := defaultSchema + ".info_test"
	child := defaultSchema + ".info_test_child"
	var foundParent, foundChild bool
	for _, col := range cols {
		switch col.Table {
		case parent:
			if col.Column != "id" {
				t.Errorf("expected primary key column 'id', got %q", col.Column)
			}
			// SQL Server reports INT as "int".
			if col.Type != "int" {
				t.Errorf("expected type 'int' on SQL Server, got %q", col.Type)
			}
			foundParent = true
		case child:
			if col.Column != "id" {
				t.Errorf("expected primary key column 'id', got %q", col.Column)
			}
			foundChild = true
		}
	}
	if !foundParent {
		t.Errorf("expected to find primary key for %s", parent)
	}
	if !foundChild {
		t.Errorf("expected to find primary key for %s", child)
	}
}

func TestGetPrimaryKeyColumnsOfType(t *testing.T) {
	// SQL Server's data_type spelling for INT is "int".
	cols, err := information.GetPrimaryKeyColumnsOfType(testCtx, testConn, "int")
	if err != nil {
		t.Fatal(err)
	}

	want := defaultSchema + ".info_test"
	found := false
	for _, col := range cols {
		if col.Table == want {
			found = true
			if col.Type != "int" {
				t.Errorf("expected type 'int', got %q", col.Type)
			}
		}
	}
	if !found {
		t.Errorf("expected to find int primary key for %s", want)
	}
}

func TestGetPrimaryKeyColumns_ForeignKey(t *testing.T) {
	cols, err := information.GetPrimaryKeyColumns(testCtx, testConn)
	if err != nil {
		t.Fatal(err)
	}

	parent := defaultSchema + ".info_test"
	child := defaultSchema + ".info_test_child"
	profile := defaultSchema + ".info_test_profile"
	var foundProfile bool
	for _, col := range cols {
		switch col.Table {
		case parent:
			if col.ForeignKey {
				t.Error("info_test.id should not be a foreign key")
			}
		case child:
			if col.Column == "id" && col.ForeignKey {
				t.Error("info_test_child.id PK should not be a foreign key")
			}
		case profile:
			if col.Column != "user_id" {
				t.Errorf("expected primary key column 'user_id', got %q", col.Column)
			}
			if !col.ForeignKey {
				t.Error("info_test_profile.user_id PK should also be a foreign key")
			}
			foundProfile = true
		}
	}
	if !foundProfile {
		t.Errorf("expected to find primary key for %s", profile)
	}
}

func TestGetTableRowsWithPrimaryKey(t *testing.T) {
	for _, q := range []string{
		/*sql*/ `DELETE FROM dbo.info_test_child`,
		/*sql*/ `DELETE FROM dbo.info_test`,
	} {
		if err := testConn.Exec(testCtx, q); err != nil {
			t.Fatal(err)
		}
	}
	err := testConn.Exec(testCtx,
		/*sql*/ `INSERT INTO dbo.info_test (id, name, value) VALUES (@p1, @p2, @p3)`,
		42, "test-row", "test-value",
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testConn.Exec(testCtx, /*sql*/ `DELETE FROM dbo.info_test_child`) //nolint:errcheck
		testConn.Exec(testCtx, /*sql*/ `DELETE FROM dbo.info_test`)       //nolint:errcheck
	})

	pkCols := []information.PrimaryKeyColumn{
		{Table: defaultSchema + ".info_test", Column: "id", Type: "int"},
		{Table: defaultSchema + ".info_test_child", Column: "id", Type: "int"},
	}

	t.Run("matching row in one table", func(t *testing.T) {
		tableRows, err := information.GetTableRowsWithPrimaryKey(testCtx, testConn, pkCols, 42)
		if err != nil {
			t.Fatal(err)
		}
		if len(tableRows) != 1 {
			t.Fatalf("expected 1 table row, got %d", len(tableRows))
		}
		if tableRows[0].Table != defaultSchema+".info_test" {
			t.Errorf("expected table %s.info_test, got %q", defaultSchema, tableRows[0].Table)
		}
	})

	t.Run("no matching rows", func(t *testing.T) {
		tableRows, err := information.GetTableRowsWithPrimaryKey(testCtx, testConn, pkCols, 999)
		if err != nil {
			t.Fatal(err)
		}
		if len(tableRows) != 0 {
			t.Errorf("expected 0 table rows, got %d", len(tableRows))
		}
	})
}

func TestRenderUUIDPrimaryKeyRefsHTML(t *testing.T) {
	// SQL Server's UUID type is reported as "uniqueidentifier", not "uuid",
	// so this handler finds no matching tables. We only verify it doesn't
	// panic when constructed.
	handler := information.RenderUUIDPrimaryKeyRefsHTML(testConn)
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}
