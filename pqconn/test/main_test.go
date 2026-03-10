package pqconn

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"

	_ "github.com/lib/pq"
)

var (
	postgresUser     = envOrDefault("POSTGRES_USER", "testuser")
	postgresPassword = envOrDefault("POSTGRES_PASSWORD", "testpassword")
	postgresHost     = envOrDefault("POSTGRES_HOST", "localhost")
	postgresPort     = envOrDefault("POSTGRES_PORT", "5433")
	dbName           = envOrDefault("POSTGRES_DB", "testdb")
)

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func dockerComposeUp() error {
	return exec.Command("docker", "compose", "up", "-d").Run()
}

func dropSchemaTables() error {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort, dbName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec( /*sql*/ `
		DO $$
		DECLARE
			r RECORD;
		BEGIN
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP
				EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$`,
	)
	return err
}

func TestMain(m *testing.M) {
	if os.Getenv("CI") == "" {
		err := dockerComposeUp()
		if err != nil {
			log.Fatalf("Failed to start Docker Compose: %v", err)
		}
	}

	err := dropSchemaTables()
	if err != nil {
		log.Fatalf("Failed to drop all user data before tests: %v", err)
	}

	m.Run()
}

func TestDatabase(t *testing.T) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort, dbName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	t.Run("ping succeeds", func(t *testing.T) {
		if err := db.PingContext(t.Context()); err != nil {
			t.Fatalf("Ping failed: %v", err)
		}
	})

	t.Run("simple scalar query", func(t *testing.T) {
		var result int
		if err := db.QueryRowContext(t.Context(), "SELECT 1").Scan(&result); err != nil {
			t.Fatalf("QueryRow failed: %v", err)
		}
		if result != 1 {
			t.Errorf("SELECT 1 returned %d, want 1", result)
		}
	})

	t.Run("create table insert and select", func(t *testing.T) {
		createSQL := /*sql*/ `
			CREATE TABLE IF NOT EXISTS _test_db (
				id   SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			)`
		_, err := db.ExecContext(t.Context(), createSQL)
		if err != nil {
			t.Fatalf("CREATE TABLE failed: %v", err)
		}
		t.Cleanup(func() {
			dropSQL := /*sql*/ `DROP TABLE IF EXISTS _test_db`
			db.ExecContext(t.Context(), dropSQL) //nolint:errcheck
		})

		insertSQL := /*sql*/ `INSERT INTO _test_db (name) VALUES ($1)`
		_, err = db.ExecContext(t.Context(), insertSQL, "hello")
		if err != nil {
			t.Fatalf("INSERT failed: %v", err)
		}

		selectSQL := /*sql*/ `SELECT name FROM _test_db LIMIT 1`
		var name string
		if err := db.QueryRowContext(t.Context(), selectSQL).Scan(&name); err != nil {
			t.Fatalf("SELECT failed: %v", err)
		}
		if name != "hello" {
			t.Errorf("name = %q, want %q", name, "hello")
		}
	})

	t.Run("transaction commit", func(t *testing.T) {
		tx, err := db.BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		var n int
		if err := tx.QueryRowContext(t.Context(), "SELECT 42").Scan(&n); err != nil {
			tx.Rollback()
			t.Fatalf("QueryRow in tx failed: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}
		if n != 42 {
			t.Errorf("SELECT 42 returned %d, want 42", n)
		}
	})

	t.Run("transaction rollback", func(t *testing.T) {
		tx, err := db.BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		if err := tx.Rollback(); err != nil {
			t.Fatalf("Rollback failed: %v", err)
		}
	})
}
