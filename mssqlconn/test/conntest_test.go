package mssqlconn

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/conntest"
	"github.com/domonda/go-sqldb/mssqlconn"
)

func connectMSSQL(t *testing.T) sqldb.Connection {
	t.Helper()
	conn, err := mssqlconn.Connect(t.Context(), testConfig())
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestConnectionSuite(t *testing.T) {
	conntest.RunAll(t, conntest.Config{
		NewConn:      connectMSSQL,
		QueryBuilder: mssqlconn.QueryBuilder{},
		DDL: conntest.DDL{
			CreateSimpleTable: /*sql*/ `CREATE TABLE conntest_simple (
				id  INT PRIMARY KEY,
				val NVARCHAR(255)
			)`,
			CreateUpsertTable: /*sql*/ `CREATE TABLE conntest_upsert (
				id    INT PRIMARY KEY,
				name  NVARCHAR(255) NOT NULL,
				score INT NOT NULL DEFAULT 0
			)`,
			// SQL Server does not support RETURNING clause
		},
		DefaultIsolationLevel:       sql.LevelReadCommitted,
		DriverName:                  mssqlconn.Driver,
		DatabaseName:                dbName,
		SupportsReadOnlyTransaction:  false, // SQL Server does not support read-only transactions
		SupportsCustomIsolationLevel: true,
		ExecAfterClosedTxErrors:      true,
	})
}
