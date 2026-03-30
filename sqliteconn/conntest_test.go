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
		},
		DefaultIsolationLevel: sql.LevelSerializable,
		DriverName:            Driver,
		DatabaseName:          ":memory:",
	})
}
