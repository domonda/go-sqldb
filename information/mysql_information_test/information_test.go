// Package mysql_information_test runs the information_schema helpers
// against the dockerized MariaDB instance defined in
// ../../mysqlconn/test/docker-compose.yml.
package mysql_information_test

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

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/information"
	"github.com/domonda/go-sqldb/mysqlconn"
)

var (
	mysqlUser     = cmp.Or(os.Getenv("MYSQL_USER"), "testuser")
	mysqlPassword = cmp.Or(os.Getenv("MYSQL_PASSWORD"), "testpassword")
	mysqlHost     = cmp.Or(os.Getenv("MYSQL_HOST"), "localhost")
	mysqlPort     = cmp.Or(atoi(os.Getenv("MYSQL_PORT")), 3307)
	dbName        = cmp.Or(os.Getenv("MYSQL_DB"), "testdb")
)

var (
	testCtx  context.Context
	testConn sqldb.Connection
)

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

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
		err := exec.Command("docker", "compose", "-f", "../../mysqlconn/test/docker-compose.yml", "up", "-d").Run()
		if err != nil {
			log.Fatalf("Failed to start Docker Compose: %v", err)
		}
	}

	if err := waitForMariaDB(); err != nil {
		log.Fatalf("Failed waiting for MariaDB: %v", err)
	}

	ctx := context.Background()
	config := &sqldb.Config{
		Driver:   mysqlconn.Driver,
		Host:     mysqlHost,
		Port:     uint16(mysqlPort),
		User:     mysqlUser,
		Password: mysqlPassword,
		Database: dbName,
	}
	var err error
	testConn, err = mysqlconn.Connect(ctx, config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create test tables. Names are prefixed `info_test_*` so they don't
	// collide with `mysqlconn/test/`'s `test_*` tables that share the
	// same MariaDB instance.
	//
	// info_test_profile.user_id is BOTH the primary key AND a foreign
	// key referencing info_test.id — it exercises the EXISTS=true branch
	// in GetPrimaryKeyColumns' foreign-key flag.
	for _, q := range []string{
		/*sql*/ `DROP TABLE IF EXISTS info_test_profile`,
		/*sql*/ `DROP TABLE IF EXISTS info_test_child`,
		/*sql*/ `DROP TABLE IF EXISTS info_test`,
	} {
		if err := testConn.Exec(ctx, q); err != nil {
			log.Fatalf("Setup drop failed for %q: %v", q, err)
		}
	}
	err = testConn.Exec(ctx,
		/*sql*/ `
			CREATE TABLE info_test (
				id    INT PRIMARY KEY,
				name  TEXT NOT NULL,
				value TEXT
			)
		`,
	)
	if err != nil {
		log.Fatalf("Failed to create info_test: %v", err)
	}
	err = testConn.Exec(ctx,
		/*sql*/ `
			CREATE TABLE info_test_child (
				id        INT PRIMARY KEY,
				parent_id INT NOT NULL,
				CONSTRAINT fk_info_test_child_parent FOREIGN KEY (parent_id) REFERENCES info_test(id)
			)
		`,
	)
	if err != nil {
		log.Fatalf("Failed to create info_test_child: %v", err)
	}
	err = testConn.Exec(ctx,
		/*sql*/ `
			CREATE TABLE info_test_profile (
				user_id INT PRIMARY KEY,
				CONSTRAINT fk_info_test_profile_user FOREIGN KEY (user_id) REFERENCES info_test(id)
			)
		`,
	)
	if err != nil {
		log.Fatalf("Failed to create info_test_profile: %v", err)
	}

	testCtx = ctx

	m.Run()

	cleanup := []string{
		/*sql*/ `DROP TABLE IF EXISTS info_test_profile`,
		/*sql*/ `DROP TABLE IF EXISTS info_test_child`,
		/*sql*/ `DROP TABLE IF EXISTS info_test`,
	}
	for _, q := range cleanup {
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
	// On MariaDB the schema is the database name.
	exists, err := information.TableExists(testCtx, testConn, dbName+".info_test")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Errorf("expected %s.info_test to exist", dbName)
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
	exists, err := information.ColumnExists(testCtx, testConn, dbName+".info_test", "id")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Errorf("expected column 'id' to exist in %s.info_test", dbName)
	}
}

func TestGetTable(t *testing.T) {
	// MariaDB requires catalog "def".
	table, err := information.GetTable(testCtx, testConn, "def", dbName, "info_test")
	if err != nil {
		t.Fatal(err)
	}
	if table == nil {
		t.Fatal("expected non-nil table")
	}
	if string(table.TableName) != "info_test" {
		t.Errorf("expected table name 'info_test', got %q", table.TableName)
	}
	if string(table.TableSchema) != dbName {
		t.Errorf("expected schema %q, got %q", dbName, table.TableSchema)
	}
	if string(table.TableType) != "BASE TABLE" {
		t.Errorf("expected table type 'BASE TABLE', got %q", table.TableType)
	}
	// Note: MariaDB's information_schema.tables does NOT have an
	// IS_INSERTABLE_INTO column (it uses MySQL-style extension columns
	// like ENGINE/ROW_FORMAT instead), so IsInsertableInto scans as the
	// YesNo zero value (false). We don't assert on it here.
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
		if string(table.TableName) == "info_test" && string(table.TableSchema) == dbName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find %s.info_test in all tables", dbName)
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

	parent := dbName + ".info_test"
	child := dbName + ".info_test_child"
	var foundParent, foundChild bool
	for _, col := range cols {
		switch col.Table {
		case parent:
			if col.Column != "id" {
				t.Errorf("expected primary key column 'id', got %q", col.Column)
			}
			// MariaDB's information_schema.columns.data_type spelling.
			if col.Type != "int" {
				t.Errorf("expected type 'int' on MariaDB, got %q", col.Type)
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
	// MariaDB reports INT as "int", not "integer".
	cols, err := information.GetPrimaryKeyColumnsOfType(testCtx, testConn, "int")
	if err != nil {
		t.Fatal(err)
	}

	want := dbName + ".info_test"
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
	// info_test_child.id is the PK and is NOT a foreign key
	// (parent_id is the FK, but it isn't a PK column).
	cols, err := information.GetPrimaryKeyColumns(testCtx, testConn)
	if err != nil {
		t.Fatal(err)
	}

	parent := dbName + ".info_test"
	child := dbName + ".info_test_child"
	profile := dbName + ".info_test_profile"
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
	err := testConn.Exec(testCtx,
		/*sql*/ `DELETE FROM info_test_child`,
	)
	if err != nil {
		t.Fatal(err)
	}
	err = testConn.Exec(testCtx,
		/*sql*/ `DELETE FROM info_test`,
	)
	if err != nil {
		t.Fatal(err)
	}
	err = testConn.Exec(testCtx,
		/*sql*/ `INSERT INTO info_test (id, name, value) VALUES (?, ?, ?)`,
		42, "test-row", "test-value",
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testConn.Exec(testCtx, /*sql*/ `DELETE FROM info_test_child`) //nolint:errcheck
		testConn.Exec(testCtx, /*sql*/ `DELETE FROM info_test`)       //nolint:errcheck
	})

	pkCols := []information.PrimaryKeyColumn{
		{Table: dbName + ".info_test", Column: "id", Type: "int"},
		{Table: dbName + ".info_test_child", Column: "id", Type: "int"},
	}

	t.Run("matching row in one table", func(t *testing.T) {
		tableRows, err := information.GetTableRowsWithPrimaryKey(testCtx, testConn, pkCols, 42)
		if err != nil {
			t.Fatal(err)
		}
		if len(tableRows) != 1 {
			t.Fatalf("expected 1 table row, got %d", len(tableRows))
		}
		if tableRows[0].Table != dbName+".info_test" {
			t.Errorf("expected table %s.info_test, got %q", dbName, tableRows[0].Table)
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
	// MariaDB has a UUID type since 10.7, but our test fixtures use INT
	// PKs only, so the handler should find no matching tables and not panic.
	handler := information.RenderUUIDPrimaryKeyRefsHTML(testConn)
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}
