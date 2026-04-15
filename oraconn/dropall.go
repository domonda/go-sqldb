package oraconn

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// DropAllTables drops all user tables,
// first removing all foreign key constraints to allow dropping
// tables in any order.
// Use [DropAll] instead to drop tables and types in the correct order.
func DropAllTables(ctx context.Context, conn sqldb.Connection) error {
	// Collect all foreign key constraints to drop first
	fkRows := conn.Query(ctx,
		/*sql*/ `
		SELECT constraint_name, table_name
		FROM user_constraints
		WHERE constraint_type = 'R'
	`)
	var fkStmts []string
	for fkRows.Next() {
		var constraint, table string
		if err := fkRows.Scan(&constraint, &table); err != nil {
			_ = fkRows.Close()
			return err
		}
		fkStmts = append(fkStmts,
			/*sql*/ `ALTER TABLE `+EscapeIdentifier(table)+
				/*sql*/ ` DROP CONSTRAINT `+EscapeIdentifier(constraint))
	}
	if err := fkRows.Err(); err != nil {
		_ = fkRows.Close()
		return err
	}
	if err := fkRows.Close(); err != nil {
		return err
	}
	for _, stmt := range fkStmts {
		if err := conn.Exec(ctx, stmt); err != nil {
			return err
		}
	}

	// Collect all user tables to drop
	tableRows := conn.Query(ctx,
		/*sql*/ `SELECT table_name FROM user_tables`)
	var tableStmts []string
	for tableRows.Next() {
		var table string
		if err := tableRows.Scan(&table); err != nil {
			_ = tableRows.Close()
			return err
		}
		tableStmts = append(tableStmts,
			/*sql*/ `DROP TABLE `+EscapeIdentifier(table)+ /*sql*/ ` CASCADE CONSTRAINTS`)
	}
	if err := tableRows.Err(); err != nil {
		_ = tableRows.Close()
		return err
	}
	if err := tableRows.Close(); err != nil {
		return err
	}
	for _, stmt := range tableStmts {
		if err := conn.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// DropAllTypes drops all user-defined types.
// Must be called after [DropAllTables] because Oracle cannot drop
// types that are referenced by existing tables.
// Use [DropAll] instead to drop tables and types in the correct order.
func DropAllTypes(ctx context.Context, conn sqldb.Connection) error {
	rows := conn.Query(ctx,
		/*sql*/ `SELECT type_name FROM user_types`)
	var stmts []string
	for rows.Next() {
		var typeName string
		if err := rows.Scan(&typeName); err != nil {
			_ = rows.Close()
			return err
		}
		stmts = append(stmts,
			/*sql*/ `DROP TYPE `+EscapeIdentifier(typeName)+ /*sql*/ ` FORCE`)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, stmt := range stmts {
		if err := conn.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// DropAll drops all user tables first (including foreign key constraints),
// then all user-defined types.
func DropAll(ctx context.Context, conn sqldb.Connection) error {
	if err := DropAllTables(ctx, conn); err != nil {
		return err
	}
	return DropAllTypes(ctx, conn)
}
