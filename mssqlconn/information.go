package mssqlconn

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/domonda/go-sqldb"
)

// This file implements [sqldb.Information] for the mssqlconn driver
// using SQL Server's sys.* catalog views, which are richer and faster
// than information_schema on MSSQL (e.g. they expose key_ordinal for
// PK column ordering, is_identity / is_computed for ColumnInfo flags,
// and proper foreign-key referential actions).
//
// SQL Server uses @pN placeholders and bracket identifier escaping;
// this file uses parameterized queries throughout.

func splitSchemaName(qualified string) (schema, name string) {
	if before, after, found := strings.Cut(qualified, "."); found {
		return before, after
	}
	return "", qualified
}

// inPlaceholders returns "@p1, @p2, ..." for n parameters starting at startIndex+1.
func inPlaceholders(startIndex, n int) string {
	if n == 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("@p%d", startIndex+i+1)
	}
	return strings.Join(parts, ", ")
}

// schemas excludes built-in role/admin schemas. principal_id <= 4 are
// the system principals (dbo, guest, INFORMATION_SCHEMA, sys); the
// db_* roles get filtered by name.
func schemas(ctx context.Context, q sqldb.Connection) ([]string, error) {
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
		/*sql*/ `
			SELECT name
			FROM sys.schemas
			WHERE name NOT IN ('sys', 'INFORMATION_SCHEMA', 'guest')
			  AND name NOT LIKE 'db[_]%'
			ORDER BY name
		`,
	)
}

func currentSchema(ctx context.Context, q sqldb.Connection) (string, error) {
	return sqldb.QueryRowAs[string](ctx, q, nil, q,
		/*sql*/ `SELECT SCHEMA_NAME()`)
}

func listObjects(ctx context.Context, q sqldb.Connection, sysView string, schemaFilter []string) ([]string, error) {
	if len(schemaFilter) == 0 {
		query := fmt.Sprintf(
			/*sql*/ `
				SELECT s.name + '.' + o.name
				FROM %s o
				JOIN sys.schemas s ON s.schema_id = o.schema_id
				WHERE s.name NOT IN ('sys', 'INFORMATION_SCHEMA')
				ORDER BY s.name, o.name
			`,
			sysView,
		)
		return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows, query)
	}
	args := make([]any, len(schemaFilter))
	for i, s := range schemaFilter {
		args[i] = s
	}
	query := fmt.Sprintf(
		/*sql*/ `
			SELECT s.name + '.' + o.name
			FROM %s o
			JOIN sys.schemas s ON s.schema_id = o.schema_id
			WHERE s.name IN (%s)
			ORDER BY s.name, o.name
		`,
		sysView, inPlaceholders(0, len(args)),
	)
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows, query, args...)
}

func objectExists(ctx context.Context, q sqldb.Connection, sysView, qualified string) (bool, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		query := fmt.Sprintf(
			/*sql*/ `
				SELECT COUNT(*) FROM %s o
				JOIN sys.schemas s ON s.schema_id = o.schema_id
				WHERE o.name = @p1 AND s.name = SCHEMA_NAME()
			`,
			sysView,
		)
		return sqldb.QueryRowAs[bool](ctx, q, nil, q, query, name)
	}
	query := fmt.Sprintf(
		/*sql*/ `
			SELECT COUNT(*) FROM %s o
			JOIN sys.schemas s ON s.schema_id = o.schema_id
			WHERE o.name = @p1 AND s.name = @p2
		`,
		sysView,
	)
	return sqldb.QueryRowAs[bool](ctx, q, nil, q, query, name, schema)
}

// tableOrViewExists checks sys.objects for both user tables ('U') and
// views ('V').
func tableOrViewExists(ctx context.Context, q sqldb.Connection, qualified string) (bool, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*) FROM sys.objects o
				JOIN sys.schemas s ON s.schema_id = o.schema_id
				WHERE o.name = @p1 AND s.name = SCHEMA_NAME() AND o.type IN ('U', 'V')
			`,
			name,
		)
	}
	return sqldb.QueryRowAs[bool](ctx, q, nil, q,
		/*sql*/ `
			SELECT COUNT(*) FROM sys.objects o
			JOIN sys.schemas s ON s.schema_id = o.schema_id
			WHERE o.name = @p1 AND s.name = @p2 AND o.type IN ('U', 'V')
		`,
		name, schema,
	)
}

// errRelationNotFound returns a wrapped sql.ErrNoRows for a missing
// table/view.
func errRelationNotFound(schema, name string) error {
	return fmt.Errorf("relation %q.%q: %w", schema, name, sql.ErrNoRows)
}

// columns reads sys.columns + sys.types for type name. PK membership
// joins sys.indexes (is_primary_key=1) + sys.index_columns. Identity
// and default columns set HasDefault; computed columns set ReadOnly.
// Returns a wrapped sql.ErrNoRows if no relation by that name exists.
func columns(ctx context.Context, q sqldb.Connection, qualified string) ([]sqldb.ColumnInfo, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		s, err := currentSchema(ctx, q)
		if err != nil {
			return nil, err
		}
		schema = s
	}
	// SQL Server has IDENTITY columns (recorded in is_identity / set
	// in HasDefault) and computed columns (is_computed). Computed
	// columns are the only "generated" kind here, so Generated and
	// ReadOnly key off the same predicate.
	rows := q.Query(ctx,
		/*sql*/ `
			SELECT
				c.name,
				t.name,
				CASE WHEN ic.column_id IS NOT NULL THEN 1 ELSE 0 END AS is_pk,
				CASE WHEN c.default_object_id <> 0 OR c.is_identity = 1 THEN 1 ELSE 0 END AS has_default,
				CASE WHEN c.is_computed = 1 THEN 1 ELSE 0 END AS read_only,
				CASE WHEN c.is_computed = 1 THEN 1 ELSE 0 END AS is_generated
			FROM sys.objects o
			JOIN sys.schemas s   ON s.schema_id = o.schema_id
			JOIN sys.columns c   ON c.object_id = o.object_id
			JOIN sys.types   t   ON t.user_type_id = c.user_type_id
			LEFT JOIN sys.indexes pk
				ON pk.object_id = o.object_id AND pk.is_primary_key = 1
			LEFT JOIN sys.index_columns ic
				ON ic.object_id = pk.object_id AND ic.index_id = pk.index_id AND ic.column_id = c.column_id
			WHERE s.name = @p1 AND o.name = @p2
			ORDER BY c.column_id
		`,
		schema, name,
	)
	defer rows.Close()
	var out []sqldb.ColumnInfo
	for rows.Next() {
		var ci sqldb.ColumnInfo
		var isPK, hasDef, readOnly, isGen int64
		if err := rows.Scan(&ci.Name, &ci.Type, &isPK, &hasDef, &readOnly, &isGen); err != nil {
			return nil, err
		}
		ci.PrimaryKey = isPK != 0
		ci.HasDefault = hasDef != 0
		ci.ReadOnly = readOnly != 0
		ci.Generated = isGen != 0
		out = append(out, ci)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		exists, err := tableOrViewExists(ctx, q, schema+"."+name)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errRelationNotFound(schema, name)
		}
	}
	return out, nil
}

func columnExists(ctx context.Context, q sqldb.Connection, tableOrView, column string) (bool, error) {
	schema, name := splitSchemaName(tableOrView)
	var (
		found bool
		err   error
	)
	if schema == "" {
		found, err = sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*)
				FROM sys.columns c
				JOIN sys.objects o ON o.object_id = c.object_id
				JOIN sys.schemas s ON s.schema_id = o.schema_id
				WHERE o.name = @p1 AND c.name = @p2 AND s.name = SCHEMA_NAME()
			`,
			name, column,
		)
	} else {
		found, err = sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*)
				FROM sys.columns c
				JOIN sys.objects o ON o.object_id = c.object_id
				JOIN sys.schemas s ON s.schema_id = o.schema_id
				WHERE o.name = @p1 AND c.name = @p2 AND s.name = @p3
			`,
			name, column, schema,
		)
	}
	if err != nil || found {
		return found, err
	}
	exists, err := tableOrViewExists(ctx, q, tableOrView)
	if err != nil {
		return false, err
	}
	if !exists {
		s := schema
		if s == "" {
			cs, cerr := currentSchema(ctx, q)
			if cerr != nil {
				return false, cerr
			}
			s = cs
		}
		return false, errRelationNotFound(s, name)
	}
	return false, nil
}

// primaryKey orders by sys.index_columns.key_ordinal, the constraint
// declaration order.
func primaryKey(ctx context.Context, q sqldb.Connection, qualified string) ([]string, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		s, err := currentSchema(ctx, q)
		if err != nil {
			return nil, err
		}
		schema = s
	}
	exists, err := objectExists(ctx, q, "sys.tables", schema+"."+name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errRelationNotFound(schema, name)
	}
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
		/*sql*/ `
			SELECT c.name
			FROM sys.indexes i
			JOIN sys.tables  t  ON t.object_id = i.object_id
			JOIN sys.schemas s  ON s.schema_id = t.schema_id
			JOIN sys.index_columns ic
				ON ic.object_id = i.object_id AND ic.index_id = i.index_id
			JOIN sys.columns c  ON c.object_id = ic.object_id AND c.column_id = ic.column_id
			WHERE i.is_primary_key = 1 AND s.name = @p1 AND t.name = @p2
			ORDER BY ic.key_ordinal
		`,
		schema, name,
	)
}

// foreignKeys returns one row per FK column and aggregates in Go,
// avoiding STRING_AGG's silent truncation past 8000/4000 bytes and
// the comma-in-identifier hazard of splitting a joined string.
// Returns a wrapped sql.ErrNoRows if the named relation is not a base
// table (missing, or a view).
func foreignKeys(ctx context.Context, q sqldb.Connection, qualified string) ([]sqldb.ForeignKeyInfo, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		s, err := currentSchema(ctx, q)
		if err != nil {
			return nil, err
		}
		schema = s
	}
	exists, err := objectExists(ctx, q, "sys.tables", schema+"."+name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errRelationNotFound(schema, name)
	}
	rows := q.Query(ctx,
		/*sql*/ `
			SELECT
				fk.name,
				lc.name AS local_col,
				rs.name + '.' + rt.name AS ref_table,
				rc.name AS ref_col,
				fk.delete_referential_action_desc,
				fk.update_referential_action_desc
			FROM sys.foreign_keys fk
			JOIN sys.tables  t  ON t.object_id = fk.parent_object_id
			JOIN sys.schemas s  ON s.schema_id = t.schema_id
			JOIN sys.tables  rt ON rt.object_id = fk.referenced_object_id
			JOIN sys.schemas rs ON rs.schema_id = rt.schema_id
			JOIN sys.foreign_key_columns fkc ON fkc.constraint_object_id = fk.object_id
			JOIN sys.columns lc ON lc.object_id = fkc.parent_object_id AND lc.column_id = fkc.parent_column_id
			JOIN sys.columns rc ON rc.object_id = fkc.referenced_object_id AND rc.column_id = fkc.referenced_column_id
			WHERE s.name = @p1 AND t.name = @p2
			ORDER BY fk.name, fkc.constraint_column_id
		`,
		schema, name,
	)
	defer rows.Close()
	byName := map[string]*sqldb.ForeignKeyInfo{}
	var order []string
	for rows.Next() {
		var fkName, localCol, refTable, refCol, delDesc, updDesc string
		if err := rows.Scan(&fkName, &localCol, &refTable, &refCol, &delDesc, &updDesc); err != nil {
			return nil, err
		}
		fk, ok := byName[fkName]
		if !ok {
			fk = &sqldb.ForeignKeyInfo{
				Name:            fkName,
				ReferencedTable: refTable,
				OnDelete:        mssqlFKAction(delDesc),
				OnUpdate:        mssqlFKAction(updDesc),
			}
			byName[fkName] = fk
			order = append(order, fkName)
		}
		fk.Columns = append(fk.Columns, localCol)
		fk.ReferencedColumns = append(fk.ReferencedColumns, refCol)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]sqldb.ForeignKeyInfo, 0, len(order))
	for _, n := range order {
		out = append(out, *byName[n])
	}
	return out, nil
}

// mssqlFKAction maps SQL Server's underscore-named action descriptions
// to the ISO action vocabulary.
func mssqlFKAction(desc string) string {
	switch desc {
	case "NO_ACTION":
		return "NO ACTION"
	case "CASCADE":
		return "CASCADE"
	case "SET_NULL":
		return "SET NULL"
	case "SET_DEFAULT":
		return "SET DEFAULT"
	}
	return desc
}

// routines: SQL Server's procedure/function objects in sys.objects.
// Argument types come from sys.parameters joined to sys.types. SQL
// Server has no overloading so each (schema, name) is unique.
//
// type codes: P=procedure, FN=scalar function, IF=inline TVF,
// TF=multi-statement TVF.
func routines(ctx context.Context, q sqldb.Connection, schemaFilter []string) ([]string, error) {
	base := /*sql*/ `
		SELECT
			s.name + '.' + o.name + '(' +
				COALESCE(STRING_AGG(t.name, ', ') WITHIN GROUP (ORDER BY p.parameter_id), '') +
			')'
		FROM sys.objects o
		JOIN sys.schemas s ON s.schema_id = o.schema_id
		LEFT JOIN sys.parameters p ON p.object_id = o.object_id AND p.parameter_id > 0
		LEFT JOIN sys.types t      ON t.user_type_id = p.user_type_id
		WHERE o.type IN ('P', 'FN', 'IF', 'TF')
	`
	if len(schemaFilter) == 0 {
		return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
			base+ /*sql*/ `
				AND s.name NOT IN ('sys', 'INFORMATION_SCHEMA')
				GROUP BY s.name, o.name, o.object_id
				ORDER BY s.name, o.name
			`,
		)
	}
	args := make([]any, len(schemaFilter))
	for i, sch := range schemaFilter {
		args[i] = sch
	}
	query := base + fmt.Sprintf(
		/*sql*/ `
			AND s.name IN (%s)
			GROUP BY s.name, o.name, o.object_id
			ORDER BY s.name, o.name
		`,
		inPlaceholders(0, len(args)),
	)
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows, query, args...)
}

func routineExists(ctx context.Context, q sqldb.Connection, routine string) (bool, error) {
	if head, _, ok := strings.Cut(routine, "("); ok {
		schema, name := splitSchemaName(head)
		if schema == "" {
			s, err := currentSchema(ctx, q)
			if err != nil {
				return false, err
			}
			schema = s
		}
		// Build the canonical signature and compare.
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*) FROM (
					SELECT
						s.name + '.' + o.name + '(' +
							COALESCE(STRING_AGG(t.name, ', ') WITHIN GROUP (ORDER BY p.parameter_id), '') +
						')' AS sig
					FROM sys.objects o
					JOIN sys.schemas s ON s.schema_id = o.schema_id
					LEFT JOIN sys.parameters p ON p.object_id = o.object_id AND p.parameter_id > 0
					LEFT JOIN sys.types t      ON t.user_type_id = p.user_type_id
					WHERE o.type IN ('P', 'FN', 'IF', 'TF')
					  AND s.name = @p1 AND o.name = @p2
					GROUP BY s.name, o.name, o.object_id
				) x WHERE x.sig = @p3
			`,
			schema, name, routine,
		)
	}
	schema, name := splitSchemaName(routine)
	if schema == "" {
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*) FROM sys.objects o
				JOIN sys.schemas s ON s.schema_id = o.schema_id
				WHERE o.type IN ('P', 'FN', 'IF', 'TF') AND o.name = @p1 AND s.name = SCHEMA_NAME()
			`,
			name,
		)
	}
	return sqldb.QueryRowAs[bool](ctx, q, nil, q,
		/*sql*/ `
			SELECT COUNT(*) FROM sys.objects o
			JOIN sys.schemas s ON s.schema_id = o.schema_id
			WHERE o.type IN ('P', 'FN', 'IF', 'TF') AND o.name = @p1 AND s.name = @p2
		`,
		name, schema,
	)
}

// --- connection methods ---

func (conn *connection) Schemas(ctx context.Context) ([]string, error) {
	return schemas(ctx, conn)
}
func (conn *connection) CurrentSchema(ctx context.Context) (string, error) {
	return currentSchema(ctx, conn)
}
func (conn *connection) Tables(ctx context.Context, schema ...string) ([]string, error) {
	return listObjects(ctx, conn, "sys.tables", schema)
}
func (conn *connection) TableExists(ctx context.Context, table string) (bool, error) {
	return objectExists(ctx, conn, "sys.tables", table)
}
func (conn *connection) Views(ctx context.Context, schema ...string) ([]string, error) {
	return listObjects(ctx, conn, "sys.views", schema)
}
func (conn *connection) ViewExists(ctx context.Context, view string) (bool, error) {
	return objectExists(ctx, conn, "sys.views", view)
}
func (conn *connection) Columns(ctx context.Context, tableOrView string) ([]sqldb.ColumnInfo, error) {
	return columns(ctx, conn, tableOrView)
}
func (conn *connection) ColumnExists(ctx context.Context, tableOrView, column string) (bool, error) {
	return columnExists(ctx, conn, tableOrView, column)
}
func (conn *connection) PrimaryKey(ctx context.Context, table string) ([]string, error) {
	return primaryKey(ctx, conn, table)
}
func (conn *connection) ForeignKeys(ctx context.Context, table string) ([]sqldb.ForeignKeyInfo, error) {
	return foreignKeys(ctx, conn, table)
}
func (conn *connection) Routines(ctx context.Context, schema ...string) ([]string, error) {
	return routines(ctx, conn, schema)
}
func (conn *connection) RoutineExists(ctx context.Context, routine string) (bool, error) {
	return routineExists(ctx, conn, routine)
}

// --- transaction methods ---

func (conn *transaction) Schemas(ctx context.Context) ([]string, error) {
	return schemas(ctx, conn)
}
func (conn *transaction) CurrentSchema(ctx context.Context) (string, error) {
	return currentSchema(ctx, conn)
}
func (conn *transaction) Tables(ctx context.Context, schema ...string) ([]string, error) {
	return listObjects(ctx, conn, "sys.tables", schema)
}
func (conn *transaction) TableExists(ctx context.Context, table string) (bool, error) {
	return objectExists(ctx, conn, "sys.tables", table)
}
func (conn *transaction) Views(ctx context.Context, schema ...string) ([]string, error) {
	return listObjects(ctx, conn, "sys.views", schema)
}
func (conn *transaction) ViewExists(ctx context.Context, view string) (bool, error) {
	return objectExists(ctx, conn, "sys.views", view)
}
func (conn *transaction) Columns(ctx context.Context, tableOrView string) ([]sqldb.ColumnInfo, error) {
	return columns(ctx, conn, tableOrView)
}
func (conn *transaction) ColumnExists(ctx context.Context, tableOrView, column string) (bool, error) {
	return columnExists(ctx, conn, tableOrView, column)
}
func (conn *transaction) PrimaryKey(ctx context.Context, table string) ([]string, error) {
	return primaryKey(ctx, conn, table)
}
func (conn *transaction) ForeignKeys(ctx context.Context, table string) ([]sqldb.ForeignKeyInfo, error) {
	return foreignKeys(ctx, conn, table)
}
func (conn *transaction) Routines(ctx context.Context, schema ...string) ([]string, error) {
	return routines(ctx, conn, schema)
}
func (conn *transaction) RoutineExists(ctx context.Context, routine string) (bool, error) {
	return routineExists(ctx, conn, routine)
}
