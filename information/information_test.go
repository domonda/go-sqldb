package information

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
	"github.com/domonda/go-sqldb/pqconn"
)

const (
	postgresUser     = "testuser"
	postgresPassword = "testpassword"
	postgresHost     = "localhost"
	postgresPort     = "5433"
	dbName           = "testdb"
)

var testCtx context.Context

func TestMain(m *testing.M) {
	err := exec.Command("docker", "compose", "-f", "../pqconn/test/docker-compose.yml", "up", "-d").Run()
	if err != nil {
		log.Fatalf("Failed to start Docker Compose: %v", err)
	}

	ctx := context.Background()
	config := &sqldb.ConnConfig{
		Driver:   pqconn.Driver,
		Host:     postgresHost,
		Port:     5433,
		User:     postgresUser,
		Password: postgresPassword,
		Database: dbName,
		Extra:    map[string]string{"sslmode": "disable"},
	}
	// Retry connecting because docker compose up -d returns
	// before PostgreSQL is ready to accept connections
	var connExt *sqldb.ConnExt
	for range 30 {
		connExt, err = pqconn.ConnectExt(ctx, config, sqldb.NewTaggedStructReflector())
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create a test table for the information schema tests
	err = connExt.Exec(ctx, /*sql*/ `
		DROP TABLE IF EXISTS information_test_child;
		DROP TABLE IF EXISTS information_test;
		CREATE TABLE information_test (
			id    integer PRIMARY KEY,
			name  text NOT NULL,
			value text
		);
		CREATE TABLE information_test_child (
			id        integer PRIMARY KEY,
			parent_id integer NOT NULL REFERENCES information_test(id)
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create test tables: %v", err)
	}

	testCtx = db.ContextWithConn(ctx, connExt)

	m.Run()

	// Cleanup
	cleanupErr := connExt.Exec(ctx, /*sql*/ `
		DROP TABLE IF EXISTS information_test_child;
		DROP TABLE IF EXISTS information_test;
	`)
	if cleanupErr != nil {
		fmt.Fprintf(os.Stderr, "Failed to drop test tables: %v\n", cleanupErr)
	}
}

func TestTableExists(t *testing.T) {
	exists, err := TableExists(testCtx, "information_test")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected information_test to exist")
	}

	exists, err = TableExists(testCtx, "nonexistent_table_xyz")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected nonexistent_table_xyz to not exist")
	}
}

func TestTableExists_WithSchema(t *testing.T) {
	exists, err := TableExists(testCtx, "public.information_test")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected public.information_test to exist")
	}
}

func TestColumnExists(t *testing.T) {
	exists, err := ColumnExists(testCtx, "information_test", "name")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected column 'name' to exist in information_test")
	}

	exists, err = ColumnExists(testCtx, "information_test", "nonexistent_col")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected column 'nonexistent_col' to not exist")
	}
}

func TestColumnExists_WithSchema(t *testing.T) {
	exists, err := ColumnExists(testCtx, "public.information_test", "id")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected column 'id' to exist in public.information_test")
	}
}

func TestGetTable(t *testing.T) {
	table, err := GetTable(testCtx, dbName, "public", "information_test")
	if err != nil {
		t.Fatal(err)
	}
	if table == nil {
		t.Fatal("expected non-nil table")
	}
	if string(table.TableName) != "information_test" {
		t.Errorf("expected table name 'information_test', got %q", table.TableName)
	}
	if string(table.TableSchema) != "public" {
		t.Errorf("expected schema 'public', got %q", table.TableSchema)
	}
	if string(table.TableType) != "BASE TABLE" {
		t.Errorf("expected table type 'BASE TABLE', got %q", table.TableType)
	}
	if !bool(table.IsInsertableInto) {
		t.Error("expected table to be insertable")
	}
}

func TestGetAllTables(t *testing.T) {
	tables, err := GetAllTables(testCtx)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) == 0 {
		t.Fatal("expected at least one table")
	}

	found := false
	for _, table := range tables {
		if string(table.TableName) == "information_test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find information_test in all tables")
	}
}

func TestGetPrimaryKeyColumns(t *testing.T) {
	cols, err := GetPrimaryKeyColumns(testCtx)
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) == 0 {
		t.Fatal("expected at least one primary key column")
	}

	var foundTest, foundChild bool
	for _, col := range cols {
		switch col.Table {
		case "public.information_test":
			if col.Column != "id" {
				t.Errorf("expected primary key column 'id', got %q", col.Column)
			}
			if col.Type != "integer" {
				t.Errorf("expected type 'integer', got %q", col.Type)
			}
			foundTest = true
		case "public.information_test_child":
			if col.Column != "id" {
				t.Errorf("expected primary key column 'id', got %q", col.Column)
			}
			foundChild = true
		}
	}
	if !foundTest {
		t.Error("expected to find primary key for information_test")
	}
	if !foundChild {
		t.Error("expected to find primary key for information_test_child")
	}
}

func TestGetTableRowsWithPrimaryKey(t *testing.T) {
	// Insert a row into the parent table only
	err := db.Conn(testCtx).Exec(testCtx, /*sql*/ `
		DELETE FROM information_test_child;
		DELETE FROM information_test;
		INSERT INTO information_test (id, name, value) VALUES (42, 'test-row', 'test-value');
	`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Conn(testCtx).Exec(testCtx, /*sql*/ `DELETE FROM information_test_child; DELETE FROM information_test`)
	})

	// PK columns for both tables — information_test_child has no row with id=42
	pkCols := []PrimaryKeyColumn{
		{Table: "public.information_test", Column: "id", Type: "integer"},
		{Table: "public.information_test_child", Column: "id", Type: "integer"},
	}

	t.Run("matching row in one table, no rows in other", func(t *testing.T) {
		// pk=42 exists in information_test but not in information_test_child
		// The function should skip information_test_child (sql.ErrNoRows / len < 2 path)
		tableRows, err := GetTableRowsWithPrimaryKey(testCtx, pkCols, 42)
		if err != nil {
			t.Fatal(err)
		}
		if len(tableRows) != 1 {
			t.Fatalf("expected 1 table row, got %d", len(tableRows))
		}
		if tableRows[0].Table != "public.information_test" {
			t.Errorf("expected table public.information_test, got %q", tableRows[0].Table)
		}
		if len(tableRows[0].Header) == 0 {
			t.Error("expected non-empty header")
		}
		if len(tableRows[0].Row) == 0 {
			t.Error("expected non-empty row")
		}
	})

	t.Run("no matching rows in any table", func(t *testing.T) {
		// pk=999 doesn't exist in any table — all tables should be skipped
		tableRows, err := GetTableRowsWithPrimaryKey(testCtx, pkCols, 999)
		if err != nil {
			t.Fatal(err)
		}
		if len(tableRows) != 0 {
			t.Errorf("expected 0 table rows, got %d", len(tableRows))
		}
	})

	t.Run("empty pk columns slice", func(t *testing.T) {
		tableRows, err := GetTableRowsWithPrimaryKey(testCtx, nil, 42)
		if err != nil {
			t.Fatal(err)
		}
		if tableRows != nil {
			t.Errorf("expected nil result, got %v", tableRows)
		}
	})
}

func TestGetPrimaryKeyColumnsOfType(t *testing.T) {
	cols, err := GetPrimaryKeyColumnsOfType(testCtx, "integer")
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, col := range cols {
		if col.Table == "public.information_test" {
			found = true
			if col.Type != "integer" {
				t.Errorf("expected type 'integer', got %q", col.Type)
			}
		}
	}
	if !found {
		t.Error("expected to find integer primary key for information_test")
	}
}
