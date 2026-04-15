package mysqlconn

import (
	"context"
	"errors"

	"github.com/domonda/go-sqldb"
)

// DropAllTables drops all tables in the current database.
// Disables foreign key checks during the operation to allow dropping
// tables in any order regardless of foreign key constraints.
func DropAllTables(ctx context.Context, conn sqldb.Connection) (err error) {
	rows := conn.Query(ctx,
		/*sql*/ `
		SELECT TABLE_NAME
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE'
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

	if err = conn.Exec(ctx, `SET FOREIGN_KEY_CHECKS = 0`); err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, conn.Exec(ctx, `SET FOREIGN_KEY_CHECKS = 1`))
	}()
	for _, table := range tables {
		err = conn.Exec(ctx,
			/*sql*/ `DROP TABLE IF EXISTS `+EscapeIdentifier(table),
		)
		if err != nil {
			return err
		}
	}
	return nil
}
