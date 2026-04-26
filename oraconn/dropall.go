package oraconn

import (
	"context"
	"errors"

	"github.com/sijms/go-ora/v2/network"

	"github.com/domonda/go-sqldb"
)

// errTableOrViewNotFound is ORA-00942 — the catalog reports a table or
// view that no longer exists by the time we issue the ALTER/DROP.
// Tolerated during DropAllTables because:
//   - Oracle's recyclebin entries appear in user_tables and user_constraints
//     but are sometimes purged between SELECT and DROP.
//   - A previous test run that crashed mid-cleanup can leave constraints in
//     user_constraints whose owning table is already gone.
const errTableOrViewNotFound = 942

// isTableOrViewNotFound reports whether err is ORA-00942.
func isTableOrViewNotFound(err error) bool {
	var oraErr *network.OracleError
	return errors.As(err, &oraErr) && oraErr.ErrCode == errTableOrViewNotFound
}

// DropAllTables drops all user tables,
// first removing all foreign key constraints to allow dropping
// tables in any order. Statements that target tables which have
// already disappeared (ORA-00942) are silently ignored, so this can
// be called against a database left in a partial state by a previous
// crashed test run.
// Use [DropAll] instead to drop every supported object type in the correct order.
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
		if err := conn.Exec(ctx, stmt); err != nil && !isTableOrViewNotFound(err) {
			return err
		}
	}

	return execEachDropTolerant(ctx, conn,
		/*sql*/ `SELECT table_name FROM user_tables`,
		func(name string) string {
			return /*sql*/ `DROP TABLE ` + EscapeIdentifier(name) + /*sql*/ ` CASCADE CONSTRAINTS`
		},
		isTableOrViewNotFound,
	)
}

// DropAllTypes drops all user-defined types.
// Must be called after [DropAllTables] because Oracle cannot drop
// types that are referenced by existing tables.
// Use [DropAll] instead to drop every supported object type in the correct order.
func DropAllTypes(ctx context.Context, conn sqldb.Connection) error {
	return execEachDrop(ctx, conn,
		/*sql*/ `SELECT type_name FROM user_types`,
		func(name string) string {
			return /*sql*/ `DROP TYPE ` + EscapeIdentifier(name) + /*sql*/ ` FORCE`
		},
	)
}

// DropAllSequences drops all user sequences.
// Sequences are independent of tables, so this can be called in any order
// relative to [DropAllTables].
// Use [DropAll] instead to drop every supported object type in the correct order.
func DropAllSequences(ctx context.Context, conn sqldb.Connection) error {
	return execEachDrop(ctx, conn,
		/*sql*/ `SELECT sequence_name FROM user_sequences`,
		func(name string) string {
			return /*sql*/ `DROP SEQUENCE ` + EscapeIdentifier(name)
		},
	)
}

// DropAllViews drops all user views.
// Views can be dropped before or after the tables they reference; Oracle
// invalidates dependent views on table drops rather than blocking them.
// Use [DropAll] instead to drop every supported object type in the correct order.
func DropAllViews(ctx context.Context, conn sqldb.Connection) error {
	return execEachDrop(ctx, conn,
		/*sql*/ `SELECT view_name FROM user_views`,
		func(name string) string {
			return /*sql*/ `DROP VIEW ` + EscapeIdentifier(name) + /*sql*/ ` CASCADE CONSTRAINTS`
		},
	)
}

// DropAllSynonyms drops all private user synonyms.
// Public synonyms are owned by PUBLIC and are not included.
// Use [DropAll] instead to drop every supported object type in the correct order.
func DropAllSynonyms(ctx context.Context, conn sqldb.Connection) error {
	return execEachDrop(ctx, conn,
		/*sql*/ `SELECT synonym_name FROM user_synonyms`,
		func(name string) string {
			return /*sql*/ `DROP SYNONYM ` + EscapeIdentifier(name)
		},
	)
}

// DropAllProceduresFunctionsPackages drops all user-defined procedures,
// functions, and packages. PACKAGE BODY objects are not queried separately
// because DROP PACKAGE drops both the spec and body.
// Use [DropAll] instead to drop every supported object type in the correct order.
func DropAllProceduresFunctionsPackages(ctx context.Context, conn sqldb.Connection) error {
	rows := conn.Query(ctx,
		/*sql*/ `
		SELECT object_name, object_type
		FROM user_objects
		WHERE object_type IN ('PROCEDURE', 'FUNCTION', 'PACKAGE')
	`)
	type drop struct {
		name string
		kind string
	}
	var drops []drop
	for rows.Next() {
		var d drop
		if err := rows.Scan(&d.name, &d.kind); err != nil {
			_ = rows.Close()
			return err
		}
		drops = append(drops, d)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, d := range drops {
		stmt := /*sql*/ `DROP ` + d.kind + ` ` + EscapeIdentifier(d.name)
		if err := conn.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// DropAll drops every user-owned schema object supported by this package,
// in the correct dependency order:
//  1. Synonyms (independent aliases)
//  2. Views (drop before tables they reference)
//  3. Procedures, functions, packages (drop before tables they reference)
//  4. Tables (with foreign key constraints removed first)
//  5. Types (must be after tables that reference them)
//  6. Sequences (independent)
//
// Objects not currently dropped: public synonyms, materialized views,
// triggers (dropped with their tables), database links, jobs, scheduler
// programs, and Java stored procedures.
func DropAll(ctx context.Context, conn sqldb.Connection) error {
	if err := DropAllSynonyms(ctx, conn); err != nil {
		return err
	}
	if err := DropAllViews(ctx, conn); err != nil {
		return err
	}
	if err := DropAllProceduresFunctionsPackages(ctx, conn); err != nil {
		return err
	}
	if err := DropAllTables(ctx, conn); err != nil {
		return err
	}
	if err := DropAllTypes(ctx, conn); err != nil {
		return err
	}
	return DropAllSequences(ctx, conn)
}

// execEachDropTolerant is like [execEachDrop] but ignores any DROP
// statement error for which tolerate returns true. Used to drop
// tables/constraints that may have disappeared between the SELECT and
// the DROP (ORA-00942).
func execEachDropTolerant(
	ctx context.Context,
	conn sqldb.Connection,
	selectQuery string,
	toDropStmt func(name string) string,
	tolerate func(error) bool,
) error {
	rows := conn.Query(ctx, selectQuery)
	var stmts []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			return err
		}
		stmts = append(stmts, toDropStmt(name))
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, stmt := range stmts {
		if err := conn.Exec(ctx, stmt); err != nil && !tolerate(err) {
			return err
		}
	}
	return nil
}

// execEachDrop runs selectQuery, scans a single string column from each row,
// builds a DROP statement via toDropStmt, and executes them sequentially.
// Rows are fully drained and closed before any DROP runs so the same
// connection is free to execute the statements.
func execEachDrop(
	ctx context.Context,
	conn sqldb.Connection,
	selectQuery string,
	toDropStmt func(name string) string,
) error {
	rows := conn.Query(ctx, selectQuery)
	var stmts []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			return err
		}
		stmts = append(stmts, toDropStmt(name))
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
