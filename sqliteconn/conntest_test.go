package sqliteconn

import (
	"database/sql"
	"testing"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/conntest"
)

func TestConnectionSuite(t *testing.T) {
	conntest.RunAll(t, conntest.Config{
		NewConn: func(t *testing.T) sqldb.Connection {
			t.Helper()
			return testConnection(t)
		},
		QueryBuilder: QueryBuilder{},
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
				id    INTEGER PRIMARY KEY AUTOINCREMENT,
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
				created_at TEXT DEFAULT (datetime('now'))
			)`,
		},
		DefaultIsolationLevel: sql.LevelSerializable,
		DriverName:            Driver,
		DatabaseName:          ":memory:",
		Information: conntest.InformationFeatures{
			SupportsRoutines:   false,
			SchemaIsAttachedDB: true,
		},
	})
}
