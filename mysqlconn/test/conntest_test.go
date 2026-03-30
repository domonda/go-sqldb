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
		},
		DefaultIsolationLevel:       sql.LevelRepeatableRead,
		DriverName:                  mysqlconn.Driver,
		DatabaseName:                dbName,
		SupportsReadOnlyTransaction:  true,
		SupportsCustomIsolationLevel: true,
		ExecAfterClosedTxErrors:      true,
	})
}
