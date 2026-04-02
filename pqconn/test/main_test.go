package pqconn

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"

	_ "github.com/lib/pq"

	"github.com/domonda/go-sqldb/pqconn"
)

var (
	postgresUser     = envOrDefault("POSTGRES_USER", "testuser")
	postgresPassword = envOrDefault("POSTGRES_PASSWORD", envOrDefault("PGPASSWORD", "testpassword"))
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
	_, err = db.Exec(pqconn.DropAllInCurrentSchemaQuery)
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
