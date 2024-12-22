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
	dockerComposeFile = "docker-compose.yml"
	postgresUser      = "testuser"
	postgresPassword  = "testpassword"
	postgresHost      = "localhost"
	postgresPort      = "5433"
	dbName            = "testdb"
)

func startDockerCompose() error {
	return exec.Command("docker", "compose", "-f", dockerComposeFile, "up", "-d").Run()
}

func stopDockerCompose() error {
	return exec.Command("docker", "compose", "-f", dockerComposeFile, "down").Run()
}

func recreateDatabase() error {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/testdb?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("DROP DATABASE IF EXISTS testdb")
	if err != nil {
		return err
	}
	_, err = db.Exec("CREATE DATABASE testdb")
	return err
}

func TestMain(m *testing.M) {
	err := startDockerCompose()
	if err != nil {
		log.Fatalf("Failed to start Docker Compose: %v", err)
	}

	code := m.Run()

	err = stopDockerCompose()
	if err != nil {
		log.Fatalf("Failed to stop Docker Compose: %v", err)
	}

	os.Exit(code)
}

func TestDatabase(t *testing.T) {
	err := recreateDatabase()
	if err != nil {
		t.Fatalf("Failed to recreate database: %v", err)
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/testdb?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run your tests using the `db` connection
}
