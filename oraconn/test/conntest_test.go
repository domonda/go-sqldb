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
	conn, err := connectWithRetry(t.Context())
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
			CreateInfoParent: /*sql*/ `CREATE TABLE conntest_info_parent (
				id1 NUMBER(10) NOT NULL,
				id2 NUMBER(10) NOT NULL,
				CONSTRAINT pk_conntest_info_parent PRIMARY KEY (id2, id1)
			)`,
			CreateInfoChild: /*sql*/ `CREATE TABLE conntest_info_child (
				child_id   NUMBER(10) PRIMARY KEY,
				parent_id1 NUMBER(10) NOT NULL,
				parent_id2 NUMBER(10) NOT NULL,
				CONSTRAINT fk_conntest_info_child FOREIGN KEY (parent_id2, parent_id1)
					REFERENCES conntest_info_parent (id2, id1) ON DELETE CASCADE
			)`,
			CreateInfoView: /*sql*/ `CREATE VIEW conntest_info_view AS
				SELECT id1, id2 FROM conntest_info_parent`,
			CreateInfoGenerated: /*sql*/ `CREATE TABLE conntest_info_generated (
				id         NUMBER(10) PRIMARY KEY,
				gen_col    NUMBER(10) GENERATED ALWAYS AS (id + 1) VIRTUAL,
				created_at TIMESTAMP DEFAULT SYSTIMESTAMP
			)`,
		},
		DefaultIsolationLevel:       sql.LevelReadCommitted,
		DriverName:                  oraconn.Driver,
		DatabaseName:                oracleService,
		SelectOneQuery:              "SELECT 1 FROM DUAL",
		SupportsReadOnlyTransaction: false, // Oracle read-only transactions need ALTER SESSION
		ExecAfterClosedTxErrors:     true,
		Information: conntest.InformationFeatures{
			SupportsRoutines: true,
			CaseFoldsToUpper: true,
		},
	})
}
