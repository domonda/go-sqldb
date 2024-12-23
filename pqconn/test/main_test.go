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

const (
	postgresUser     = "testuser"
	postgresPassword = "testpassword"
	postgresHost     = "localhost"
	postgresPort     = "5433"
	dbName           = "testdb"
)

func dockerComposeUp() error {
	return exec.Command("docker", "compose", "up", "-d").Run()
}

func dockerComposeDown() error {
	return exec.Command("docker", "compose", "down").Run()
}

func dropSchemaTables() error {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/testdb?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort)
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
	err := dockerComposeUp()
	if err != nil {
		log.Fatalf("Failed to start Docker Compose: %v", err)
	}

	err = dropSchemaTables()
	if err != nil {
		log.Fatalf("Failed drop all user data before tests: %v", err)
	}

	code := m.Run()

	err = dockerComposeDown()
	if err != nil {
		log.Fatalf("Failed to stop Docker Compose: %v", err)
	}

	os.Exit(code)
}

func TestDatabase(t *testing.T) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/testdb?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run your tests using the `db` connection
}
