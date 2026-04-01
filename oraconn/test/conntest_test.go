package oraconn

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/conntest"
	"github.com/domonda/go-sqldb/oraconn"
)

func connectOracle(t *testing.T) sqldb.Connection {
	t.Helper()
	conn, err := oraconn.Connect(t.Context(), testConfig(), true)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestConnectionSuite(t *testing.T) {
	conntest.RunAll(t, conntest.Config{
		NewConn:      connectOracle,
		QueryBuilder: oraconn.QueryBuilder{},
		DDL: conntest.DDL{
			CreateSimpleTable: /*sql*/ `CREATE TABLE conntest_simple (
				id  NUMBER(10) PRIMARY KEY,
				val VARCHAR2(255)
			)`,
			CreateUpsertTable: /*sql*/ `CREATE TABLE conntest_upsert (
				id    NUMBER(10) PRIMARY KEY,
				name  VARCHAR2(255) NOT NULL,
				score NUMBER(10) DEFAULT 0 NOT NULL
			)`,
			// Oracle does not support RETURNING via QueryBuilder
			CreateMailAddressTable: /*sql*/ `CREATE TABLE conntest_mail_address (
				id    NUMBER(10) PRIMARY KEY,
				email VARCHAR2(255)
			)`,
		},
		DefaultIsolationLevel:       sql.LevelReadCommitted,
		DriverName:                  oraconn.Driver,
		DatabaseName:                oracleService,
		SelectOneQuery:              "SELECT 1 FROM DUAL",
		SupportsReadOnlyTransaction: false, // Oracle read-only transactions need ALTER SESSION
		ExecAfterClosedTxErrors:     true,
	})
}
