package oraconn

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/domonda/go-sqldb"
)

// This file implements [sqldb.Information] for the oraconn driver
// using Oracle's ALL_* / USER_* data-dictionary views (Oracle has no
// information_schema). In Oracle a "schema" is a user, so Schemas
// returns user names and CurrentSchema returns the connected user's
// current schema.
//
// Oracle does not support cascading updates; the data dictionary has
// no update_rule column on referential constraints. ForeignKeyInfo
// reports OnUpdate as "NO ACTION" for every Oracle FK.
//
// Oracle placeholders are :1, :2, etc.

// splitSchemaName splits "owner.name". Oracle stores identifiers
// uppercase by default; we pass through whatever the caller wrote and
// rely on Oracle's case-folding for the comparison to work. Callers
// using mixed-case quoted identifiers must pass them as the catalog
// stores them.
func splitSchemaName(qualified string) (schema, name string) {
	if before, after, found := strings.Cut(qualified, "."); found {
		return before, after
	}
	return "", qualified
}

// inPlaceholders returns ":1, :2, ..." for n parameters at startIndex+1.
func inPlaceholders(startIndex, n int) string {
	if n == 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf(":%d", startIndex+i+1)
	}
	return strings.Join(parts, ", ")
}

// schemas excludes Oracle-maintained users (SYS, SYSTEM, GG users,
// and so on). The oracle_maintained='N' filter is the canonical way
// from Oracle 12c onwards.
func schemas(ctx context.Context, q sqldb.Connection) ([]string, error) {
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
		/*sql*/ `
			SELECT username FROM all_users WHERE oracle_maintained = 'N' ORDER BY username
		`,
	)
}

func currentSchema(ctx context.Context, q sqldb.Connection) (string, error) {
	return sqldb.QueryRowAs[string](ctx, q, nil, q,
		/*sql*/ `
			SELECT SYS_CONTEXT('USERENV','CURRENT_SCHEMA') FROM DUAL
		`,
	)
}

func listObjects(ctx context.Context, q sqldb.Connection, view string, schemaFilter []string) ([]string, error) {
	if len(schemaFilter) == 0 {
		query := fmt.Sprintf( /*sql*/ `
			SELECT a.owner || '.' || a.%s
			FROM %s a
			JOIN all_users u ON u.username = a.owner
			WHERE u.oracle_maintained = 'N'
			ORDER BY a.owner, a.%s
		`,
			viewNameColumn(view), view, viewNameColumn(view),
		)
		return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows, query)
	}
	args := make([]any, len(schemaFilter))
	for i, s := range schemaFilter {
		args[i] = s
	}
	query := fmt.Sprintf(
		/*sql*/ `
			SELECT a.owner || '.' || a.%s
			FROM %s a
			WHERE a.owner IN (%s)
			ORDER BY a.owner, a.%s
		`,
		viewNameColumn(view),
		view,
		inPlaceholders(0, len(args)),
		viewNameColumn(view),
	)
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows, query, args...)
}

func viewNameColumn(view string) string {
	switch view {
	case "all_tables":
		return "table_name"
	case "all_views":
		return "view_name"
	}
	return "name"
}

func tableExists(ctx context.Context, q sqldb.Connection, qualified string) (bool, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*) FROM all_tables
				WHERE owner = SYS_CONTEXT('USERENV','CURRENT_SCHEMA') AND table_name = :1
			`,
			name,
		)
	}
	return sqldb.QueryRowAs[bool](ctx, q, nil, q,
		/*sql*/ `
			SELECT COUNT(*) FROM all_tables WHERE owner = :1 AND table_name = :2
		`,
		schema, name,
	)
}

func viewExists(ctx context.Context, q sqldb.Connection, qualified string) (bool, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*) FROM all_views
				WHERE owner = SYS_CONTEXT('USERENV','CURRENT_SCHEMA') AND view_name = :1
			`,
			name,
		)
	}
	return sqldb.QueryRowAs[bool](ctx, q, nil, q,
		/*sql*/ `
			SELECT COUNT(*) FROM all_views WHERE owner = :1 AND view_name = :2
		`,
		schema, name,
	)
}

// tableOrViewExists checks all_objects for either a TABLE or a VIEW
// of the given name.
func tableOrViewExists(ctx context.Context, q sqldb.Connection, qualified string) (bool, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*) FROM all_objects
				WHERE owner = SYS_CONTEXT('USERENV','CURRENT_SCHEMA')
				  AND object_name = :1
				  AND object_type IN ('TABLE', 'VIEW')
			`,
			name,
		)
	}
	return sqldb.QueryRowAs[bool](ctx, q, nil, q,
		/*sql*/ `
			SELECT COUNT(*) FROM all_objects
			WHERE owner = :1 AND object_name = :2 AND object_type IN ('TABLE', 'VIEW')
		`,
		schema, name,
	)
}

// errRelationNotFound returns a wrapped sql.ErrNoRows for a missing
// table/view.
func errRelationNotFound(schema, name string) error {
	return fmt.Errorf("relation %q.%q: %w", schema, name, sql.ErrNoRows)
}

func columns(ctx context.Context, q sqldb.Connection, qualified string) ([]sqldb.ColumnInfo, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		s, err := currentSchema(ctx, q)
		if err != nil {
			return nil, err
		}
		schema = s
	}
	// all_tab_cols (note: COLS, not COLUMNS) exposes virtual_column;
	// hidden_column = 'NO' filters out system-generated and hidden
	// columns to match what all_tab_columns would have returned.
	// Generated is narrower than ReadOnly: virtual (computed)
	// columns set Generated; identity columns (identity_column='YES')
	// only set ReadOnly because they are not "generated" in the
	// SQL-standard sense — Oracle generates them from a sequence,
	// not from an expression on other columns.
	rows := q.Query(ctx,
		/*sql*/ `
			SELECT
				c.column_name,
				c.data_type,
				CASE WHEN pk.column_name IS NOT NULL THEN 1 ELSE 0 END AS is_pk,
				CASE WHEN c.data_default IS NOT NULL OR c.identity_column = 'YES' THEN 1 ELSE 0 END AS has_default,
				CASE WHEN c.virtual_column = 'YES' OR c.identity_column = 'YES' THEN 1 ELSE 0 END AS read_only,
				CASE WHEN c.virtual_column = 'YES' THEN 1 ELSE 0 END AS is_generated
			FROM all_tab_cols c
			LEFT JOIN (
				SELECT cc.owner, cc.table_name, cc.column_name
				FROM all_constraints ac
				JOIN all_cons_columns cc
					ON cc.owner = ac.owner AND cc.constraint_name = ac.constraint_name
				WHERE ac.constraint_type = 'P'
			) pk
				ON pk.owner = c.owner AND pk.table_name = c.table_name AND pk.column_name = c.column_name
			WHERE c.owner = :1 AND c.table_name = :2 AND c.hidden_column = 'NO'
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
				SELECT COUNT(*) FROM all_tab_columns
				WHERE owner = SYS_CONTEXT('USERENV','CURRENT_SCHEMA')
				  AND table_name = :1 AND column_name = :2
			`,
			name, column,
		)
	} else {
		found, err = sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*) FROM all_tab_columns
				WHERE owner = :1 AND table_name = :2 AND column_name = :3
			`,
			schema, name, column,
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

// primaryKey orders by all_cons_columns.position, the constraint
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
	exists, err := tableExists(ctx, q, schema+"."+name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errRelationNotFound(schema, name)
	}
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
		/*sql*/ `
			SELECT cc.column_name
			FROM all_constraints ac
			JOIN all_cons_columns cc
				ON cc.owner = ac.owner AND cc.constraint_name = ac.constraint_name
			WHERE ac.constraint_type = 'P' AND ac.owner = :1 AND ac.table_name = :2
			ORDER BY cc.position
		`,
		schema, name,
	)
}

// foreignKeys returns one row per FK column and aggregates in Go,
// avoiding LISTAGG's ORA-01489 (4000-byte ceiling) on long composite
// keys and the comma-in-identifier hazard of splitting a joined
// string. Local and referenced columns align by all_cons_columns.position,
// which is the constraint declaration order on both sides. Oracle has
// no on-update; OnUpdate is set to "NO ACTION" by the Go side.
// Returns a wrapped sql.ErrNoRows if the named relation is not a
// base table (missing, or a view).
func foreignKeys(ctx context.Context, q sqldb.Connection, qualified string) ([]sqldb.ForeignKeyInfo, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		s, err := currentSchema(ctx, q)
		if err != nil {
			return nil, err
		}
		schema = s
	}
	exists, err := tableExists(ctx, q, schema+"."+name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errRelationNotFound(schema, name)
	}
	rows := q.Query(ctx,
		/*sql*/ `
			SELECT
				ac.constraint_name,
				lc.column_name,
				ac.r_owner || '.' || rac.table_name AS ref_table,
				rc.column_name,
				ac.delete_rule
			FROM all_constraints ac
			JOIN all_constraints rac
				ON rac.owner = ac.r_owner AND rac.constraint_name = ac.r_constraint_name
			JOIN all_cons_columns lc
				ON lc.owner = ac.owner AND lc.constraint_name = ac.constraint_name
			JOIN all_cons_columns rc
				ON rc.owner = ac.r_owner AND rc.constraint_name = ac.r_constraint_name
			   AND rc.position = lc.position
			WHERE ac.constraint_type = 'R' AND ac.owner = :1 AND ac.table_name = :2
			ORDER BY ac.constraint_name, lc.position
		`,
		schema, name,
	)
	defer rows.Close()
	byName := map[string]*sqldb.ForeignKeyInfo{}
	var order []string
	for rows.Next() {
		var fkName, localCol, refTable, refCol, delRule string
		if err := rows.Scan(&fkName, &localCol, &refTable, &refCol, &delRule); err != nil {
			return nil, err
		}
		fk, ok := byName[fkName]
		if !ok {
			fk = &sqldb.ForeignKeyInfo{
				Name:            fkName,
				ReferencedTable: refTable,
				OnDelete:        oraFKAction(delRule),
				OnUpdate:        "NO ACTION",
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

// oraFKAction normalizes Oracle's delete_rule values. Oracle supports
// only NO ACTION (the default), CASCADE, and SET NULL.
func oraFKAction(s string) string {
	switch s {
	case "", "NO ACTION":
		return "NO ACTION"
	}
	return s
}

// routines returns top-level standalone procedures and functions.
// Routines defined inside PACKAGE bodies are intentionally excluded
// because their fully-qualified name is `schema.package.routine`,
// which doesn't fit the `schema.name(args)` shape this interface
// promises.
func routines(ctx context.Context, q sqldb.Connection, schemaFilter []string) ([]string, error) {
	base := /*sql*/ `
		SELECT
			o.owner || '.' || o.object_name || '(' ||
				NVL((
					SELECT LISTAGG(a.data_type, ', ') WITHIN GROUP (ORDER BY a.position)
					FROM all_arguments a
					WHERE a.owner = o.owner
					  AND a.object_name = o.object_name
					  AND a.package_name IS NULL
					  AND a.position > 0
				), '') ||
			')'
		FROM all_objects o
		WHERE o.object_type IN ('PROCEDURE', 'FUNCTION')`

	if len(schemaFilter) == 0 {
		return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
			base+ /*sql*/ `
				AND o.owner IN (SELECT username FROM all_users WHERE oracle_maintained = 'N')
				ORDER BY o.owner, o.object_name
			`,
		)
	}
	args := make([]any, len(schemaFilter))
	for i, s := range schemaFilter {
		args[i] = s
	}
	query := base + fmt.Sprintf(
		/*sql*/ `
			AND o.owner IN (%s)
			ORDER BY o.owner, o.object_name
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
		// Build canonical signature server-side and compare.
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*) FROM (
					SELECT
						o.owner || '.' || o.object_name || '(' ||
							NVL((
								SELECT LISTAGG(a.data_type, ', ') WITHIN GROUP (ORDER BY a.position)
								FROM all_arguments a
								WHERE a.owner = o.owner
								  AND a.object_name = o.object_name
								  AND a.package_name IS NULL
								  AND a.position > 0
							), '') ||
						')' AS sig
					FROM all_objects o
					WHERE o.object_type IN ('PROCEDURE', 'FUNCTION')
					  AND o.owner = :1 AND o.object_name = :2
				) WHERE sig = :3
			`,
			schema, name, routine,
		)
	}
	schema, name := splitSchemaName(routine)
	if schema == "" {
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*) FROM all_objects
				WHERE object_type IN ('PROCEDURE', 'FUNCTION')
				  AND owner = SYS_CONTEXT('USERENV','CURRENT_SCHEMA')
				  AND object_name = :1
			`,
			name,
		)
	}
	return sqldb.QueryRowAs[bool](ctx, q, nil, q,
		/*sql*/ `
			SELECT COUNT(*) FROM all_objects
			WHERE object_type IN ('PROCEDURE', 'FUNCTION') AND owner = :1 AND object_name = :2
		`,
		schema, name,
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
	return listObjects(ctx, conn, "all_tables", schema)
}
func (conn *connection) TableExists(ctx context.Context, table string) (bool, error) {
	return tableExists(ctx, conn, table)
}
func (conn *connection) Views(ctx context.Context, schema ...string) ([]string, error) {
	return listObjects(ctx, conn, "all_views", schema)
}
func (conn *connection) ViewExists(ctx context.Context, view string) (bool, error) {
	return viewExists(ctx, conn, view)
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
	return listObjects(ctx, conn, "all_tables", schema)
}
func (conn *transaction) TableExists(ctx context.Context, table string) (bool, error) {
	return tableExists(ctx, conn, table)
}
func (conn *transaction) Views(ctx context.Context, schema ...string) ([]string, error) {
	return listObjects(ctx, conn, "all_views", schema)
}
func (conn *transaction) ViewExists(ctx context.Context, view string) (bool, error) {
	return viewExists(ctx, conn, view)
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
