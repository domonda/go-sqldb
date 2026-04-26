package mysqlconn

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/conntest"
	"github.com/domonda/go-sqldb/mysqlconn"
)

func connectMySQL(t *testing.T) sqldb.Connection {
	t.Helper()
	conn, err := mysqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestConnectionSuite(t *testing.T) {
	conntest.RunAll(t, conntest.Config{
		NewConn:      connectMySQL,
		QueryBuilder: mysqlconn.QueryBuilder{},
		DDL: conntest.DDL{
			CreateSimpleTable: /*sql*/ `CREATE TABLE conntest_simple (
				id  INT PRIMARY KEY,
				val TEXT
			)`,
			CreateUpsertTable: /*sql*/ `CREATE TABLE conntest_upsert (
				id    INT PRIMARY KEY,
				name  TEXT NOT NULL,
				score INT NOT NULL DEFAULT 0
			)`,
			// MySQL does not support RETURNING clause
			CreateMailAddressTable: /*sql*/ `CREATE TABLE conntest_mail_address (
				id    INT PRIMARY KEY,
				email TEXT
			)`,
			CreateInfoParent: /*sql*/ `CREATE TABLE conntest_info_parent (
				id1 INT NOT NULL,
				id2 INT NOT NULL,
				PRIMARY KEY (id2, id1)
			)`,
			CreateInfoChild: /*sql*/ `CREATE TABLE conntest_info_child (
				child_id   INT PRIMARY KEY,
				parent_id1 INT NOT NULL,
				parent_id2 INT NOT NULL,
				FOREIGN KEY (parent_id2, parent_id1)
					REFERENCES conntest_info_parent (id2, id1) ON DELETE CASCADE
			)`,
			CreateInfoView: /*sql*/ `CREATE VIEW conntest_info_view AS
				SELECT id1, id2 FROM conntest_info_parent`,
			CreateInfoGenerated: /*sql*/ `CREATE TABLE conntest_info_generated (
				id         INT PRIMARY KEY,
				gen_col    INT GENERATED ALWAYS AS (id + 1) STORED,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		DefaultIsolationLevel:        sql.LevelRepeatableRead,
		DriverName:                   mysqlconn.Driver,
		DatabaseName:                 dbName,
		SupportsReadOnlyTransaction:  true,
		SupportsCustomIsolationLevel: true,
		ExecAfterClosedTxErrors:      true,
		Information: conntest.InformationFeatures{
			SupportsRoutines: true,
		},
	})
}
