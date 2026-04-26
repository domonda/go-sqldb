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

func pqConnect(t *testing.T) sqldb.Connection {
	t.Helper()
	port, err := strconv.ParseUint(postgresPort, 10, 16)
	require.NoError(t, err)
	config := &sqldb.Config{
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
		NewConn:      pqConnect,
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
			CreateMailAddressTable: /*sql*/ `CREATE TABLE conntest_mail_address (
				id    INTEGER PRIMARY KEY,
				email TEXT
			)`,
			CreateInfoParent: /*sql*/ `CREATE TABLE conntest_info_parent (
				id1 INTEGER NOT NULL,
				id2 INTEGER NOT NULL,
				PRIMARY KEY (id2, id1)
			)`,
			CreateInfoChild: /*sql*/ `CREATE TABLE conntest_info_child (
				child_id   INTEGER PRIMARY KEY,
				parent_id1 INTEGER NOT NULL,
				parent_id2 INTEGER NOT NULL,
				FOREIGN KEY (parent_id2, parent_id1)
					REFERENCES conntest_info_parent (id2, id1) ON DELETE CASCADE
			)`,
			CreateInfoView: /*sql*/ `CREATE VIEW conntest_info_view AS
				SELECT id1, id2 FROM conntest_info_parent`,
			CreateInfoGenerated: /*sql*/ `CREATE TABLE conntest_info_generated (
				id         INTEGER PRIMARY KEY,
				gen_col    INTEGER GENERATED ALWAYS AS (id + 1) STORED,
				created_at TIMESTAMPTZ DEFAULT now()
			)`,
		},
		DefaultIsolationLevel:        sql.LevelReadCommitted,
		DriverName:                   pqconn.Driver,
		DatabaseName:                 dbName,
		SupportsReadOnlyTransaction:  true,
		SupportsCustomIsolationLevel: true,
		ExecAfterClosedTxErrors:      true,
		Information: conntest.InformationFeatures{
			SupportsRoutines: true,
		},
	})
}
