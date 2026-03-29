package mssqlconn

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// DropAllTables drops all user tables in all user schemas,
// first removing all foreign key constraints to allow dropping
// tables in any order.
// Use [DropAll] instead to drop tables and types in the correct order.
func DropAllTables(ctx context.Context, conn sqldb.Connection) error {
	// Collect all foreign key constraints to drop first
	fkRows := conn.Query(ctx,
		/*sql*/ `
		SELECT s.name, t.name, fk.name
		FROM sys.foreign_keys fk
		JOIN sys.tables t ON fk.parent_object_id = t.object_id
		JOIN sys.schemas s ON t.schema_id = s.schema_id
		WHERE t.is_ms_shipped = 0
	`)
	var fkStmts []string
	for fkRows.Next() {
		var schema, table, constraint string
		if err := fkRows.Scan(&schema, &table, &constraint); err != nil {
			_ = fkRows.Close()
			return err
		}
		fkStmts = append(fkStmts,
			"ALTER TABLE "+EscapeIdentifier(schema)+"."+EscapeIdentifier(table)+
				" DROP CONSTRAINT "+EscapeIdentifier(constraint))
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
		/*sql*/ `
		SELECT s.name, t.name
		FROM sys.tables t
		JOIN sys.schemas s ON t.schema_id = s.schema_id
		WHERE t.is_ms_shipped = 0
	`)
	var tableStmts []string
	for tableRows.Next() {
		var schema, table string
		if err := tableRows.Scan(&schema, &table); err != nil {
			_ = tableRows.Close()
			return err
		}
		tableStmts = append(tableStmts,
			"DROP TABLE "+EscapeIdentifier(schema)+"."+EscapeIdentifier(table))
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

// DropAllTypes drops all user-defined types in all user schemas.
// Must be called after [DropAllTables] because SQL Server cannot drop
// types that are referenced by existing tables.
// Use [DropAll] instead to drop tables and types in the correct order.
func DropAllTypes(ctx context.Context, conn sqldb.Connection) error {
	rows := conn.Query(ctx,
		/*sql*/ `
		SELECT s.name, t.name
		FROM sys.types t
		JOIN sys.schemas s ON t.schema_id = s.schema_id
		WHERE t.is_user_defined = 1
	`)
	var stmts []string
	for rows.Next() {
		var schema, typName string
		if err := rows.Scan(&schema, &typName); err != nil {
			_ = rows.Close()
			return err
		}
		stmts = append(stmts,
			"DROP TYPE "+EscapeIdentifier(schema)+"."+EscapeIdentifier(typName))
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
// then all user-defined types in all user schemas.
func DropAll(ctx context.Context, conn sqldb.Connection) error {
	if err := DropAllTables(ctx, conn); err != nil {
		return err
	}
	return DropAllTypes(ctx, conn)
}
