package sqliteconn

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// DropAllTables drops all user-defined tables in the database.
// Internal SQLite tables (names prefixed with "sqlite_") are excluded.
// Foreign key enforcement is disabled during the operation to allow
// dropping tables in any order regardless of foreign key constraints.
func DropAllTables(ctx context.Context, conn sqldb.Connection) error {
	rows := conn.Query(ctx,
		/*sql*/ `
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
	`)
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			return err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	if len(tables) == 0 {
		return nil
	}

	if err := conn.Exec(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	for _, table := range tables {
		if err := conn.Exec(ctx, "DROP TABLE IF EXISTS "+EscapeIdentifier(table)); err != nil {
			_ = conn.Exec(ctx, `PRAGMA foreign_keys = ON`)
			return err
		}
	}
	return conn.Exec(ctx, `PRAGMA foreign_keys = ON`)
}
