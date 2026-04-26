package pqconn

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/domonda/go-sqldb"
)

// This file implements [sqldb.Information] for the pqconn driver using
// pg_catalog directly. We do NOT route these queries through
// information_schema because pg_catalog is the canonical source of
// truth in PostgreSQL, exposes everything (including PK ordinal order
// and routine signatures) without joins or post-processing, and avoids
// the empty-string columns that information_schema reports for
// PostgreSQL-only metadata.

// splitSchemaName splits "schema.name" into (schema, name). When there
// is no dot, schema is empty so callers can fall back to current_schema().
func splitSchemaName(qualified string) (schema, name string) {
	if before, after, found := strings.Cut(qualified, "."); found {
		return before, after
	}
	return "", qualified
}

// schemas returns user-visible schemas, excluding pg_catalog,
// information_schema, and pg_toast.
func schemas(ctx context.Context, q sqldb.Connection) ([]string, error) {
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
		/*sql*/ `
			SELECT nspname
			FROM pg_catalog.pg_namespace
			WHERE nspname NOT LIKE 'pg_%'
			  AND nspname <> 'information_schema'
			ORDER BY nspname
		`,
	)
}

// currentSchema returns the first existing schema in search_path.
func currentSchema(ctx context.Context, q sqldb.Connection) (string, error) {
	return sqldb.QueryRowAs[string](ctx, q, nil, q, `SELECT current_schema()`)
}

// listRelations enumerates pg_class rows of one or more relkinds,
// optionally restricted to the given schemas.
func listRelations(ctx context.Context, q sqldb.Connection, relkinds string, schemaFilter []string) ([]string, error) {
	// relkinds is a string of one-character codes, e.g. "r" or "vm".
	rkArr := make([]string, 0, len(relkinds))
	for _, r := range relkinds {
		rkArr = append(rkArr, string(r))
	}
	if len(schemaFilter) == 0 {
		return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
			/*sql*/ `
				SELECT n.nspname || '.' || c.relname
				FROM pg_catalog.pg_class c
				JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
				WHERE c.relkind = ANY($1)
				  AND n.nspname NOT LIKE 'pg_%'
				  AND n.nspname <> 'information_schema'
				ORDER BY n.nspname, c.relname
			`,
			pq.Array(rkArr),
		)
	}
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
		/*sql*/ `
			SELECT n.nspname || '.' || c.relname
			FROM pg_catalog.pg_class c
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
			WHERE c.relkind = ANY($1)
			  AND n.nspname = ANY($2)
			ORDER BY n.nspname, c.relname
		`,
		pq.Array(rkArr), pq.Array(schemaFilter),
	)
}

// relationExists checks for a pg_class row of any of the given
// relkinds. relkinds is a string of one-character codes (e.g. "r" for
// just ordinary tables, "rp" for ordinary + partitioned).
func relationExists(ctx context.Context, q sqldb.Connection, relkinds, qualified string) (bool, error) {
	rkArr := make([]string, 0, len(relkinds))
	for _, r := range relkinds {
		rkArr = append(rkArr, string(r))
	}
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT EXISTS (
					SELECT 1
					FROM pg_catalog.pg_class c
					JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
					WHERE c.relkind = ANY($1) AND c.relname = $2 AND n.nspname = current_schema()
				)
			`,
			pq.Array(rkArr), name,
		)
	}
	return sqldb.QueryRowAs[bool](ctx, q, nil, q,
		/*sql*/ `
			SELECT EXISTS (
				SELECT 1
				FROM pg_catalog.pg_class c
				JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
				WHERE c.relkind = ANY($1) AND c.relname = $2 AND n.nspname = $3
			)
		`,
		pq.Array(rkArr), name, schema,
	)
}

// tableOrViewExists checks for any pg_class relkind that exposes
// columns: 'r' base table, 'v' view, 'm' materialized view, 'f'
// foreign table, 'p' partitioned table.
func tableOrViewExists(ctx context.Context, q sqldb.Connection, qualified string) (bool, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT EXISTS (
					SELECT 1
					FROM pg_catalog.pg_class c
					JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
					WHERE c.relname = $1 AND n.nspname = current_schema()
					  AND c.relkind IN ('r', 'v', 'm', 'f', 'p')
				)
			`,
			name,
		)
	}
	return sqldb.QueryRowAs[bool](ctx, q, nil, q,
		/*sql*/ `
			SELECT EXISTS (
				SELECT 1
				FROM pg_catalog.pg_class c
				JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
				WHERE c.relname = $1 AND n.nspname = $2
				  AND c.relkind IN ('r', 'v', 'm', 'f', 'p')
			)
		`,
		name, schema,
	)
}

// errRelationNotFound returns a wrapped sql.ErrNoRows for a missing
// table/view/relation.
func errRelationNotFound(schema, name string) error {
	return fmt.Errorf("relation %q.%q: %w", schema, name, sql.ErrNoRows)
}

// columns returns the columns of the given table or view, with
// PrimaryKey membership filled in from pg_constraint. Returns a
// wrapped sql.ErrNoRows if no relation by that name exists.
func columns(ctx context.Context, q sqldb.Connection, qualified string) ([]sqldb.ColumnInfo, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		s, err := currentSchema(ctx, q)
		if err != nil {
			return nil, err
		}
		schema = s
	}
	// Generated is narrower than ReadOnly: GENERATED ALWAYS AS (expr)
	// columns set Generated; identity-always columns (attidentity='a')
	// only set ReadOnly. attgenerated <> '' matches stored 's' and,
	// on PostgreSQL 18+, virtual 'v' generated columns.
	rows := q.Query(ctx,
		/*sql*/ `
			SELECT
				a.attname,
				pg_catalog.format_type(a.atttypid, a.atttypmod) AS type,
				COALESCE(pk.is_pk, false) AS is_pk,
				a.atthasdef OR a.attidentity <> '' AS has_default,
				a.attgenerated <> '' OR a.attidentity = 'a' AS read_only,
				a.attgenerated <> '' AS is_generated
			FROM pg_catalog.pg_attribute a
			JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
			LEFT JOIN LATERAL (
				SELECT true AS is_pk
				FROM pg_catalog.pg_constraint pkc
				WHERE pkc.conrelid = c.oid AND pkc.contype = 'p' AND a.attnum = ANY(pkc.conkey)
			) pk ON true
			WHERE n.nspname = $1
			  AND c.relname = $2
			  AND a.attnum > 0
			  AND NOT a.attisdropped
			ORDER BY a.attnum
		`,
		schema, name,
	)
	defer rows.Close()
	var out []sqldb.ColumnInfo
	for rows.Next() {
		var ci sqldb.ColumnInfo
		if err := rows.Scan(&ci.Name, &ci.Type, &ci.PrimaryKey, &ci.HasDefault, &ci.ReadOnly, &ci.Generated); err != nil {
			return nil, err
		}
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
				SELECT EXISTS (
					SELECT 1
					FROM pg_catalog.pg_attribute a
					JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
					JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
					WHERE n.nspname = current_schema()
					  AND c.relname = $1
					  AND a.attname = $2
					  AND a.attnum > 0
					  AND NOT a.attisdropped
				)
			`,
			name, column,
		)
	} else {
		found, err = sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT EXISTS (
					SELECT 1
					FROM pg_catalog.pg_attribute a
					JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
					JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
					WHERE n.nspname = $1
					  AND c.relname = $2
					  AND a.attname = $3
					  AND a.attnum > 0
					  AND NOT a.attisdropped
				)
			`,
			schema, name, column,
		)
	}
	if err != nil || found {
		return found, err
	}
	// Disambiguate "no such column" from "no such relation".
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

// primaryKey returns PK columns in constraint ordinal order using
// array_position(conkey, attnum) to preserve the declared order.
func primaryKey(ctx context.Context, q sqldb.Connection, qualified string) ([]string, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		s, err := currentSchema(ctx, q)
		if err != nil {
			return nil, err
		}
		schema = s
	}
	exists, err := relationExists(ctx, q, "rp", schema+"."+name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errRelationNotFound(schema, name)
	}
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
		/*sql*/ `
			SELECT a.attname
			FROM pg_catalog.pg_constraint c
			JOIN pg_catalog.pg_namespace n ON n.oid = c.connamespace
			JOIN pg_catalog.pg_class t ON t.oid = c.conrelid
			JOIN pg_catalog.pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
			WHERE c.contype = 'p' AND n.nspname = $1 AND t.relname = $2
			ORDER BY array_position(c.conkey, a.attnum)
		`,
		schema, name,
	)
}

// foreignKeys returns FK constraints with composite columns aggregated
// in matching order using unnest WITH ORDINALITY. Returns a wrapped
// sql.ErrNoRows if the named relation is not a base table (missing,
// or a view).
func foreignKeys(ctx context.Context, q sqldb.Connection, qualified string) ([]sqldb.ForeignKeyInfo, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		s, err := currentSchema(ctx, q)
		if err != nil {
			return nil, err
		}
		schema = s
	}
	exists, err := relationExists(ctx, q, "rp", schema+"."+name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errRelationNotFound(schema, name)
	}
	rows := q.Query(ctx,
		/*sql*/ `
			WITH fk AS (
				SELECT c.oid, c.conname, c.confdeltype, c.confupdtype, c.confrelid,
					ord.ordinality AS ord, ord.local_attnum, ord.ref_attnum,
					rn.nspname AS ref_schema, rt.relname AS ref_table
				FROM pg_catalog.pg_constraint c
				JOIN pg_catalog.pg_namespace n ON n.oid = c.connamespace
				JOIN pg_catalog.pg_class t ON t.oid = c.conrelid
				JOIN pg_catalog.pg_class rt ON rt.oid = c.confrelid
				JOIN pg_catalog.pg_namespace rn ON rn.oid = rt.relnamespace
				JOIN LATERAL unnest(c.conkey, c.confkey)
					WITH ORDINALITY AS ord(local_attnum, ref_attnum, ordinality) ON true
				WHERE c.contype = 'f' AND n.nspname = $1 AND t.relname = $2
			)
			SELECT
				fk.conname,
				array_agg(la.attname ORDER BY fk.ord) AS local_cols,
				fk.ref_schema || '.' || fk.ref_table AS ref_table,
				array_agg(ra.attname ORDER BY fk.ord) AS ref_cols,
				fk.confdeltype,
				fk.confupdtype
			FROM fk
			JOIN pg_catalog.pg_attribute la ON la.attrelid = (SELECT conrelid FROM pg_catalog.pg_constraint WHERE oid = fk.oid) AND la.attnum = fk.local_attnum
			JOIN pg_catalog.pg_attribute ra ON ra.attrelid = fk.confrelid AND ra.attnum = fk.ref_attnum
			GROUP BY fk.oid, fk.conname, fk.ref_schema, fk.ref_table, fk.confdeltype, fk.confupdtype
			ORDER BY fk.conname
		`,
		schema, name,
	)
	defer rows.Close()
	var out []sqldb.ForeignKeyInfo
	for rows.Next() {
		var (
			fk                 sqldb.ForeignKeyInfo
			localCols, refCols pq.StringArray
			delCode, updCode   string
		)
		if err := rows.Scan(&fk.Name, &localCols, &fk.ReferencedTable, &refCols, &delCode, &updCode); err != nil {
			return nil, err
		}
		fk.Columns = []string(localCols)
		fk.ReferencedColumns = []string(refCols)
		fk.OnDelete = pgFKAction(delCode)
		fk.OnUpdate = pgFKAction(updCode)
		out = append(out, fk)
	}
	return out, rows.Err()
}

// pgFKAction maps PostgreSQL's single-character action codes to the
// ISO action vocabulary used in [sqldb.ForeignKeyInfo].
func pgFKAction(code string) string {
	switch code {
	case "a":
		return "NO ACTION"
	case "r":
		return "RESTRICT"
	case "c":
		return "CASCADE"
	case "n":
		return "SET NULL"
	case "d":
		return "SET DEFAULT"
	}
	return ""
}

// routines returns every function and procedure across user schemas
// formatted as `schema.name(argtypes)`. Each PG overload is a separate
// entry because pg_get_function_identity_arguments uniquely identifies
// the overload.
func routines(ctx context.Context, q sqldb.Connection, schemaFilter []string) ([]string, error) {
	if len(schemaFilter) == 0 {
		return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
			/*sql*/ `
				SELECT n.nspname || '.' || p.proname || '(' || pg_catalog.pg_get_function_identity_arguments(p.oid) || ')'
				FROM pg_catalog.pg_proc p
				JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
				WHERE p.prokind IN ('f', 'p')
				  AND n.nspname NOT LIKE 'pg_%'
				  AND n.nspname <> 'information_schema'
				ORDER BY n.nspname, p.proname, p.oid
			`,
		)
	}
	return sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows,
		/*sql*/ `
			SELECT n.nspname || '.' || p.proname || '(' || pg_catalog.pg_get_function_identity_arguments(p.oid) || ')'
			FROM pg_catalog.pg_proc p
			JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
			WHERE p.prokind IN ('f', 'p')
			AND n.nspname = ANY($1)
			ORDER BY n.nspname, p.proname, p.oid
		`,
		pq.Array(schemaFilter),
	)
}

// routineExists implements both signature-match (parens present) and
// name-match (no parens) modes.
func routineExists(ctx context.Context, q sqldb.Connection, routine string) (bool, error) {
	if head, _, ok := strings.Cut(routine, "("); ok {
		// Signature match: parse "schema.name(args)" and recompute
		// the canonical signature for each candidate. We compare full
		// strings rather than try to parse arg type spellings because
		// pg_get_function_identity_arguments may normalize whitespace
		// and casts in ways that differ from caller input.
		schema, name := splitSchemaName(head)
		if schema == "" {
			s, err := currentSchema(ctx, q)
			if err != nil {
				return false, err
			}
			schema = s
		}
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT EXISTS (
					SELECT 1
					FROM pg_catalog.pg_proc p
					JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
					WHERE p.prokind IN ('f', 'p')
					  AND n.nspname = $1
					  AND p.proname = $2
					  AND ($3 = n.nspname || '.' || p.proname || '(' || pg_catalog.pg_get_function_identity_arguments(p.oid) || ')')
				)
			`,
			schema, name, routine,
		)
	}
	// Name match: any overload.
	schema, name := splitSchemaName(routine)
	if schema == "" {
		return sqldb.QueryRowAs[bool](ctx, q, nil, q,
			/*sql*/ `
				SELECT EXISTS (
					SELECT 1
					FROM pg_catalog.pg_proc p
					JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
					WHERE p.prokind IN ('f', 'p')
					  AND n.nspname = current_schema()
					  AND p.proname = $1
				)
			`,
			name,
		)
	}
	return sqldb.QueryRowAs[bool](ctx, q, nil, q,
		/*sql*/ `
			SELECT EXISTS (
				SELECT 1
				FROM pg_catalog.pg_proc p
				JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
				WHERE p.prokind IN ('f', 'p')
				  AND n.nspname = $1
				  AND p.proname = $2
			)
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
	// 'r' = ordinary table, 'p' = partitioned table parent. Both are
	// "base tables" from a caller's perspective; child partitions of a
	// partitioned table appear as 'r' alongside ordinary tables.
	return listRelations(ctx, conn, "rp", schema)
}
func (conn *connection) TableExists(ctx context.Context, table string) (bool, error) {
	return relationExists(ctx, conn, "rp", table)
}
func (conn *connection) Views(ctx context.Context, schema ...string) ([]string, error) {
	return listRelations(ctx, conn, "v", schema)
}
func (conn *connection) ViewExists(ctx context.Context, view string) (bool, error) {
	return relationExists(ctx, conn, "v", view)
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
	// 'r' = ordinary table, 'p' = partitioned table parent. Both are
	// "base tables" from a caller's perspective; child partitions of a
	// partitioned table appear as 'r' alongside ordinary tables.
	return listRelations(ctx, conn, "rp", schema)
}
func (conn *transaction) TableExists(ctx context.Context, table string) (bool, error) {
	return relationExists(ctx, conn, "rp", table)
}
func (conn *transaction) Views(ctx context.Context, schema ...string) ([]string, error) {
	return listRelations(ctx, conn, "v", schema)
}
func (conn *transaction) ViewExists(ctx context.Context, view string) (bool, error) {
	return relationExists(ctx, conn, "v", view)
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
