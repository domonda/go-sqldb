package mysqlconn

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/domonda/go-sqldb"
)

// This file implements [sqldb.Information] for the mysqlconn driver
// using MySQL/MariaDB-idiomatic SQL: SHOW DATABASES / DATABASE() /
// SHOW KEYS for the simple checks, and information_schema for
// composite metadata that SHOW does not return cleanly.
//
// MySQL has no schemas inside a database; "schema" and "database" are
// synonyms in the catalog. The Information interface contract folds
// "current database" into "current schema" accordingly.

func splitSchemaName(qualified string) (schema, name string) {
	if before, after, found := strings.Cut(qualified, "."); found {
		return before, after
	}
	return "", qualified
}

// schemas excludes MySQL's system schemas. SHOW DATABASES is the
// idiomatic source; information_schema.schemata is exactly equivalent.
func schemas(ctx context.Context, q sqldb.Connection) ([]string, error) {
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
		/*sql*/ `
			SELECT schema_name
			FROM information_schema.schemata
			WHERE schema_name NOT IN ('mysql', 'sys', 'performance_schema', 'information_schema')
			ORDER BY schema_name
		`,
	)
}

func currentSchema(ctx context.Context, q sqldb.Connection) (string, error) {
	return sqldb.QueryRowAs[string](ctx, q, nil, q, `SELECT DATABASE()`)
}

func listTables(ctx context.Context, q sqldb.Connection, tableType string, schemaFilter []string) ([]string, error) {
	if len(schemaFilter) == 0 {
		return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
			/*sql*/ `
				SELECT CONCAT(table_schema, '.', table_name)
				FROM information_schema.tables
				WHERE table_type = ?
				  AND table_schema NOT IN ('mysql', 'sys', 'performance_schema', 'information_schema')
				ORDER BY table_schema, table_name
			`,
			tableType,
		)
	}
	// Build IN clause with N placeholders.
	placeholders := strings.Repeat("?,", len(schemaFilter))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, 0, len(schemaFilter)+1)
	args = append(args, tableType)
	for _, s := range schemaFilter {
		args = append(args, s)
	}
	query := fmt.Sprintf(
		/*sql*/ `
			SELECT CONCAT(table_schema, '.', table_name)
			FROM information_schema.tables
			WHERE table_type = ?
			  AND table_schema IN (%s)
			ORDER BY table_schema, table_name
		`,
		placeholders,
	)
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows, query, args...)
}

func tableTypeExists(ctx context.Context, q sqldb.Connection, tableType, qualified string) (bool, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*)
				FROM information_schema.tables
				WHERE table_type = ? AND table_name = ? AND table_schema = DATABASE()
			`,
			tableType, name,
		)
	}
	return sqldb.QueryRowAs[bool](ctx, q, nil, q,
		/*sql*/ `
			SELECT COUNT(*)
			FROM information_schema.tables
			WHERE table_type = ? AND table_name = ? AND table_schema = ?
		`,
		tableType, name, schema,
	)
}

// tableOrViewExists checks information_schema.tables for any
// user-visible relation kind ('BASE TABLE', 'VIEW', 'SYSTEM VIEW',
// and the MariaDB extras like 'SEQUENCE'). The point is to recognize
// "anything that could appear in information_schema.columns".
func tableOrViewExists(ctx context.Context, q sqldb.Connection, qualified string) (bool, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*)
				FROM information_schema.tables
				WHERE table_name = ? AND table_schema = DATABASE()
			`,
			name,
		)
	}
	return sqldb.QueryRowAs[bool](ctx, q, nil, q,
		/*sql*/ `
			SELECT COUNT(*)
			FROM information_schema.tables
			WHERE table_name = ? AND table_schema = ?
		`,
		name, schema,
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
	// MySQL/MariaDB has no separate identity-always concept
	// (auto_increment is exposed as has_default), so Generated and
	// ReadOnly key off the same predicate here. The regex is anchored
	// to "(STORED|VIRTUAL) GENERATED" — the literal sequence MySQL
	// 5.7+ and MariaDB 10.2+ write to extra for generated columns —
	// rather than the loose "GENERATED|STORED|VIRTUAL" alternation,
	// which would also match MySQL 8.0.13+'s "DEFAULT_GENERATED" tag
	// on columns whose default is an expression (e.g. TIMESTAMP
	// DEFAULT CURRENT_TIMESTAMP).
	rows := q.Query(ctx,
		/*sql*/ `
			SELECT
				column_name,
				column_type,
				column_key = 'PRI'                                          AS is_pk,
				column_default IS NOT NULL OR extra LIKE '%auto_increment%' AS has_default,
				extra REGEXP '(STORED|VIRTUAL) GENERATED'                   AS read_only,
				extra REGEXP '(STORED|VIRTUAL) GENERATED'                   AS is_generated
			FROM information_schema.columns
			WHERE table_schema = ? AND table_name = ?
			ORDER BY ordinal_position
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
				SELECT COUNT(*) FROM information_schema.columns
				WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?
			`,
			name, column,
		)
	} else {
		found, err = sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*) FROM information_schema.columns
				WHERE table_schema = ? AND table_name = ? AND column_name = ?
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

// primaryKey orders columns by SEQ_IN_INDEX, which is the constraint
// ordinal position. SHOW KEYS would return the same order but
// information_schema.statistics is portable to non-interactive callers.
func primaryKey(ctx context.Context, q sqldb.Connection, qualified string) ([]string, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		s, err := currentSchema(ctx, q)
		if err != nil {
			return nil, err
		}
		schema = s
	}
	exists, err := tableTypeExists(ctx, q, "BASE TABLE", schema+"."+name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errRelationNotFound(schema, name)
	}
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
		/*sql*/ `
			SELECT column_name
			FROM information_schema.statistics
			WHERE table_schema = ? AND table_name = ? AND index_name = 'PRIMARY'
			ORDER BY seq_in_index
		`,
		schema, name,
	)
}

// foreignKeys returns one row per FK column and aggregates in Go,
// avoiding GROUP_CONCAT's silent truncation at group_concat_max_len
// (1024 bytes by default) and the comma-in-identifier hazard of
// splitting a joined string. Returns a wrapped sql.ErrNoRows if the
// named relation is not a base table (missing, or a view).
func foreignKeys(ctx context.Context, q sqldb.Connection, qualified string) ([]sqldb.ForeignKeyInfo, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		s, err := currentSchema(ctx, q)
		if err != nil {
			return nil, err
		}
		schema = s
	}
	exists, err := tableTypeExists(ctx, q, "BASE TABLE", schema+"."+name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errRelationNotFound(schema, name)
	}
	rows := q.Query(ctx,
		/*sql*/ `
			SELECT
				rc.constraint_name,
				kcu.column_name,
				CONCAT(rc.unique_constraint_schema, '.', rc.referenced_table_name) AS ref_table,
				kcu.referenced_column_name,
				rc.delete_rule,
				rc.update_rule
			FROM information_schema.referential_constraints rc
			JOIN information_schema.key_column_usage kcu
				ON kcu.constraint_schema = rc.constraint_schema
			   AND kcu.constraint_name   = rc.constraint_name
			   AND kcu.table_schema      = rc.constraint_schema
			   AND kcu.table_name        = rc.table_name
			WHERE rc.constraint_schema = ? AND rc.table_name = ?
			ORDER BY rc.constraint_name, kcu.ordinal_position
		`,
		schema, name,
	)
	defer rows.Close()
	byName := map[string]*sqldb.ForeignKeyInfo{}
	var order []string
	for rows.Next() {
		var fkName, localCol, refTable, refCol, delRule, updRule string
		if err := rows.Scan(&fkName, &localCol, &refTable, &refCol, &delRule, &updRule); err != nil {
			return nil, err
		}
		fk, ok := byName[fkName]
		if !ok {
			fk = &sqldb.ForeignKeyInfo{
				Name:            fkName,
				ReferencedTable: refTable,
				OnDelete:        delRule,
				OnUpdate:        updRule,
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

// routines: information_schema.routines + parameters joined on
// (specific_schema, specific_name). MySQL has no overloading so each
// (schema, name) is unique.
func routines(ctx context.Context, q sqldb.Connection, schemaFilter []string) ([]string, error) {
	// Aggregate parameters per routine, ordered by ordinal_position.
	// parameters has parameter_mode = NULL for the synthetic return
	// row of FUNCTIONs and ordinal_position = 0; filter that out.
	base := /*sql*/ `
		SELECT
			CONCAT(
				r.routine_schema, '.', r.routine_name, '(',
				COALESCE(GROUP_CONCAT(p.data_type ORDER BY p.ordinal_position SEPARATOR ', '), ''),
				')'
			)
		FROM information_schema.routines r
		LEFT JOIN information_schema.parameters p
			ON p.specific_schema = r.routine_schema
			AND p.specific_name   = r.specific_name
			AND p.ordinal_position > 0
		WHERE r.routine_type IN ('FUNCTION', 'PROCEDURE')`

	if len(schemaFilter) == 0 {
		return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
			base+ /*sql*/ `
				AND r.routine_schema NOT IN ('mysql', 'sys', 'performance_schema', 'information_schema')
				GROUP BY r.routine_schema, r.routine_name, r.specific_name
				ORDER BY r.routine_schema, r.routine_name
			`,
		)
	}
	placeholders := strings.Repeat("?,", len(schemaFilter))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, 0, len(schemaFilter))
	for _, s := range schemaFilter {
		args = append(args, s)
	}
	query := base + fmt.Sprintf(
		/*sql*/ `
			AND r.routine_schema IN (%s)
			GROUP BY r.routine_schema, r.routine_name, r.specific_name
			ORDER BY r.routine_schema, r.routine_name
		`,
		placeholders,
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
		// Build the canonical signature for the candidate(s) and
		// compare strings. MySQL has no overloading so the routine
		// is unique by (schema, name); we still build the signature
		// in SQL to ensure formatting matches Routines exactly.
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*) FROM (
					SELECT
						CONCAT(
							r.routine_schema, '.', r.routine_name, '(',
							COALESCE(GROUP_CONCAT(p.data_type ORDER BY p.ordinal_position SEPARATOR ', '), ''),
							')'
						) AS sig
					FROM information_schema.routines r
					LEFT JOIN information_schema.parameters p
						ON p.specific_schema = r.routine_schema
					   AND p.specific_name   = r.specific_name
					   AND p.ordinal_position > 0
					WHERE r.routine_type IN ('FUNCTION', 'PROCEDURE')
					  AND r.routine_schema = ? AND r.routine_name = ?
					GROUP BY r.routine_schema, r.routine_name, r.specific_name
				) s WHERE s.sig = ?
			`,
			schema, name, routine,
		)
	}
	schema, name := splitSchemaName(routine)
	if schema == "" {
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT COUNT(*) FROM information_schema.routines
				WHERE routine_type IN ('FUNCTION', 'PROCEDURE')
				  AND routine_schema = DATABASE() AND routine_name = ?
			`,
			name,
		)
	}
	return sqldb.QueryRowAs[bool](ctx, q, nil, q,
		/*sql*/ `
			SELECT COUNT(*) FROM information_schema.routines
			WHERE routine_type IN ('FUNCTION', 'PROCEDURE')
			  AND routine_schema = ? AND routine_name = ?
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
	return listTables(ctx, conn, "BASE TABLE", schema)
}
func (conn *connection) TableExists(ctx context.Context, table string) (bool, error) {
	return tableTypeExists(ctx, conn, "BASE TABLE", table)
}
func (conn *connection) Views(ctx context.Context, schema ...string) ([]string, error) {
	return listTables(ctx, conn, "VIEW", schema)
}
func (conn *connection) ViewExists(ctx context.Context, view string) (bool, error) {
	return tableTypeExists(ctx, conn, "VIEW", view)
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
	return listTables(ctx, conn, "BASE TABLE", schema)
}
func (conn *transaction) TableExists(ctx context.Context, table string) (bool, error) {
	return tableTypeExists(ctx, conn, "BASE TABLE", table)
}
func (conn *transaction) Views(ctx context.Context, schema ...string) ([]string, error) {
	return listTables(ctx, conn, "VIEW", schema)
}
func (conn *transaction) ViewExists(ctx context.Context, view string) (bool, error) {
	return tableTypeExists(ctx, conn, "VIEW", view)
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
