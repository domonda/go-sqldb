package mssqlconn

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestConnectWrongDriver(t *testing.T) {
	config := &sqldb.ConnConfig{
		Driver:   "postgres",
		Host:     "localhost",
		Database: "testdb",
	}
	conn, err := Connect(context.Background(), config)
	require.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "invalid driver")
}

func TestFormatDSN(t *testing.T) {
	config := &sqldb.ConnConfig{
		Driver:   Driver,
		Host:     "localhost",
		Port:     1433,
		User:     "sa",
		Password: "MyPass123",
		Database: "testdb",
	}
	dsn := formatDSN(config)
	assert.Contains(t, dsn, "sqlserver://")
	assert.Contains(t, dsn, "localhost:1433")
	assert.Contains(t, dsn, "database=testdb")
	// Database should be in query params, not in path
	assert.NotContains(t, dsn, "/testdb")
}

func TestFormatDSNWithExtra(t *testing.T) {
	config := &sqldb.ConnConfig{
		Driver:   Driver,
		Host:     "localhost",
		Port:     1433,
		User:     "sa",
		Password: "MyPass123",
		Database: "testdb",
		Extra: map[string]string{
			"encrypt":            "disable",
			"connection timeout": "30",
		},
	}
	dsn := formatDSN(config)
	assert.Contains(t, dsn, "database=testdb")
	assert.Contains(t, dsn, "encrypt=disable")
}
