package pqconn

import (
	"database/sql"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/conntest"
	"github.com/domonda/go-sqldb/pqconn"
)

func connectPQ(t *testing.T) sqldb.Connection {
	t.Helper()
	port, err := strconv.ParseUint(postgresPort, 10, 16)
	require.NoError(t, err)
	config := &sqldb.ConnConfig{
		Driver:   "postgres",
		Host:     postgresHost,
		Port:     uint16(port),
		User:     postgresUser,
		Password: postgresPassword,
		Database: dbName,
		Extra:    map[string]string{"sslmode": "disable"},
	}
	conn, err := pqconn.Connect(t.Context(), config)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestConnectionSuite(t *testing.T) {
	conntest.RunAll(t, conntest.Config{
		NewConn:      connectPQ,
		QueryBuilder: pqconn.QueryBuilder{},
		DDL: conntest.DDL{
			CreateSimpleTable: /*sql*/ `CREATE TABLE conntest_simple (
				id  INTEGER PRIMARY KEY,
				val TEXT
			)`,
			CreateUpsertTable: /*sql*/ `CREATE TABLE conntest_upsert (
				id    INTEGER PRIMARY KEY,
				name  TEXT NOT NULL,
				score INTEGER NOT NULL DEFAULT 0
			)`,
			CreateReturningTable: /*sql*/ `CREATE TABLE conntest_returning (
				id    SERIAL PRIMARY KEY,
				name  TEXT NOT NULL,
				score INTEGER NOT NULL DEFAULT 0
			)`,
		},
		DefaultIsolationLevel:       sql.LevelReadCommitted,
		DriverName:                  pqconn.Driver,
		DatabaseName:                dbName,
		SupportsReadOnlyTransaction:  true,
		SupportsCustomIsolationLevel: true,
		ExecAfterClosedTxErrors:      true,
	})
}
