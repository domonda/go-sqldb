package sqliteconn

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/domonda/go-sqldb"
)

// This file implements [sqldb.Information] for the sqliteconn driver
// using SQLite's native catalog: sqlite_schema for object enumeration
// and PRAGMA functions for column / FK metadata. SQLite does not
// expose information_schema and has no concept of schemas inside a
// database. "Schemas" here means attached databases.
//
// SQLite does not have stored routines. Routines and RoutineExists
// return errors.ErrUnsupported.

// splitSchemaName splits "schema.name". For SQLite "schema" is an
// attached database name (defaults to "main").
func splitSchemaName(qualified string) (schema, name string) {
	if before, after, found := strings.Cut(qualified, "."); found {
		return before, after
	}
	return "", qualified
}

// schemas returns attached databases via PRAGMA database_list. The
// list always contains at least "main"; "temp" is added when the
// connection has temporary tables; further entries appear after
// ATTACH DATABASE.
func schemas(ctx context.Context, q sqldb.Connection) ([]string, error) {
	rows := q.Query(ctx, `PRAGMA database_list`)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var seq int64
		var name, file string
		if err := rows.Scan(&seq, &name, &file); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// currentSchema is always "main" on SQLite. There is no per-connection
// default schema selector — unqualified names always resolve against
// the temp/main databases in that order. ctx and q are unused; the
// signature matches the helper shape used by every other driver in
// this file so it can be called from the same code paths.
func currentSchema(_ context.Context, _ sqldb.Connection) (string, error) {
	return "main", nil
}

// listSchemaObjects lists tables or views from sqlite_schema for the
// given attached database. SQLite stores per-attached-database schemas
// in `<dbname>.sqlite_schema`, so we have to query each one separately.
func listSchemaObjects(ctx context.Context, q sqldb.Connection, objType string, schemaFilter []string) ([]string, error) {
	dbs := schemaFilter
	if len(dbs) == 0 {
		all, err := schemas(ctx, q)
		if err != nil {
			return nil, err
		}
		dbs = all
	}
	var out []string
	for _, db := range dbs {
		// sqlite_schema is a virtual table per attached database.
		// We must interpolate the database name (it's not a value
		// parameter slot). Identifier quoting via double-quotes.
		query := fmt.Sprintf(
			/*sql*/ `
				SELECT name FROM "%s".sqlite_schema
				WHERE type = ? AND name NOT LIKE 'sqlite_%%'
				ORDER BY name
			`,
			strings.ReplaceAll(db, `"`, `""`),
		)
		names, err := sqldb.QueryRowsAsSlice[string](ctx, q, nil, q, sqldb.UnlimitedMaxNumRows, query, objType)
		if err != nil {
			return nil, err
		}
		for _, n := range names {
			out = append(out, db+"."+n)
		}
	}
	return out, nil
}

func objectExists(ctx context.Context, q sqldb.Connection, objType, qualified string) (bool, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		schema = "main"
	}
	query := fmt.Sprintf(
		/*sql*/ `
			SELECT COUNT(*) FROM "%s".sqlite_schema WHERE type = ? AND name = ?
		`,
		strings.ReplaceAll(schema, `"`, `""`))
	return sqldb.QueryRowAs[bool](ctx, q, nil, q, query, objType, name)
}

// tableOrViewExists checks sqlite_schema for either a 'table' or a
// 'view' of the given name in the named (or default) attached
// database.
func tableOrViewExists(ctx context.Context, q sqldb.Connection, qualified string) (bool, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		schema = "main"
	}
	query := fmt.Sprintf(
		/*sql*/ `
			SELECT COUNT(*) FROM "%s".sqlite_schema
			WHERE type IN ('table', 'view') AND name = ?
		`,
		strings.ReplaceAll(schema, `"`, `""`),
	)
	return sqldb.QueryRowAs[bool](ctx, q, nil, q, query, name)
}

// errRelationNotFound returns a wrapped sql.ErrNoRows for a missing
// table/view.
func errRelationNotFound(schema, name string) error {
	return fmt.Errorf("relation %q.%q: %w", schema, name, sql.ErrNoRows)
}

// columns uses PRAGMA table_info, which works for both tables and
// views. dflt_value being non-NULL means a default was specified;
// pk > 0 indicates PK membership and the value gives the ordinal.
// SQLite does not surface "computed column" through table_info, so
// ReadOnly is conservatively false. Generated columns appear via
// table_xinfo — we use the broader form to also pick those up,
// marking them ReadOnly when hidden=2 (STORED) or hidden=3 (VIRTUAL).
// Returns a wrapped sql.ErrNoRows if no relation by that name exists.
func columns(ctx context.Context, q sqldb.Connection, qualified string) ([]sqldb.ColumnInfo, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		schema = "main"
	}
	// PRAGMA table_xinfo returns: cid, name, type, notnull, dflt_value, pk, hidden.
	query := fmt.Sprintf(`PRAGMA "%s".table_xinfo("%s")`,
		strings.ReplaceAll(schema, `"`, `""`),
		strings.ReplaceAll(name, `"`, `""`),
	)
	rows := q.Query(ctx, query)
	defer rows.Close()
	var out []sqldb.ColumnInfo
	for rows.Next() {
		var (
			cid, notnull, pk, hidden int64
			colName, colType, dflt   sql.NullString
		)
		if err := rows.Scan(&cid, &colName, &colType, &notnull, &dflt, &pk, &hidden); err != nil {
			return nil, err
		}
		// SQLite has no separate identity-always concept (rowid /
		// AUTOINCREMENT show up as has_default), so Generated and
		// ReadOnly key off the same predicate: hidden=2 (STORED) or
		// hidden=3 (VIRTUAL).
		generated := hidden == 2 || hidden == 3
		out = append(out, sqldb.ColumnInfo{
			Name:       colName.String,
			Type:       colType.String,
			PrimaryKey: pk > 0,
			HasDefault: dflt.Valid,
			ReadOnly:   generated,
			Generated:  generated,
		})
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
	cols, err := columns(ctx, q, tableOrView)
	if err != nil {
		return false, err
	}
	for _, c := range cols {
		if c.Name == column {
			return true, nil
		}
	}
	return false, nil
}

// primaryKey reads PRAGMA table_info and returns columns with pk > 0
// in pk-ordinal order.
func primaryKey(ctx context.Context, q sqldb.Connection, qualified string) ([]string, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		schema = "main"
	}
	exists, err := objectExists(ctx, q, "table", schema+"."+name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errRelationNotFound(schema, name)
	}
	query := fmt.Sprintf( /*sql*/ `PRAGMA "%s".table_info("%s")`,
		strings.ReplaceAll(schema, `"`, `""`),
		strings.ReplaceAll(name, `"`, `""`),
	)
	rows := q.Query(ctx, query)
	defer rows.Close()
	type pkCol struct {
		name string
		pk   int64
	}
	var pks []pkCol
	for rows.Next() {
		var cid, notnull, pk int64
		var colName, colType, dflt sql.NullString
		if err := rows.Scan(&cid, &colName, &colType, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		if pk > 0 {
			pks = append(pks, pkCol{name: colName.String, pk: pk})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Sort by pk ordinal.
	for i := 1; i < len(pks); i++ {
		for j := i; j > 0 && pks[j-1].pk > pks[j].pk; j-- {
			pks[j-1], pks[j] = pks[j], pks[j-1]
		}
	}
	out := make([]string, len(pks))
	for i, c := range pks {
		out[i] = c.name
	}
	return out, nil
}

// foreignKeys uses PRAGMA foreign_key_list. Each row is one column of
// one constraint (id groups them). Composite FKs are aggregated by id
// in Go. Returns a wrapped sql.ErrNoRows if the named relation is not
// a base table (missing, or a view).
func foreignKeys(ctx context.Context, q sqldb.Connection, qualified string) ([]sqldb.ForeignKeyInfo, error) {
	schema, name := splitSchemaName(qualified)
	if schema == "" {
		schema = "main"
	}
	exists, err := objectExists(ctx, q, "table", schema+"."+name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errRelationNotFound(schema, name)
	}
	query := fmt.Sprintf(`PRAGMA "%s".foreign_key_list("%s")`,
		strings.ReplaceAll(schema, `"`, `""`),
		strings.ReplaceAll(name, `"`, `""`),
	)
	rows := q.Query(ctx, query)
	defer rows.Close()
	type fkRow struct {
		id, seq                                       int64
		table, from, to, onUpdate, onDelete, matchOpt string
	}
	var raw []fkRow
	for rows.Next() {
		var r fkRow
		if err := rows.Scan(&r.id, &r.seq, &r.table, &r.from, &r.to, &r.onUpdate, &r.onDelete, &r.matchOpt); err != nil {
			return nil, err
		}
		raw = append(raw, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Sort by (id, seq) so we can simply append within each group.
	// This is defensive: SQLite is documented to return rows in this
	// order anyway, but the indexed-write-via-pad approach this
	// replaces would silently produce empty-string columns if a future
	// SQLite version ever returned sparse seq values.
	slices.SortStableFunc(raw, func(a, b fkRow) int {
		if a.id != b.id {
			return cmp.Compare(a.id, b.id)
		}
		return cmp.Compare(a.seq, b.seq)
	})
	byID := map[int64]*sqldb.ForeignKeyInfo{}
	var order []int64
	for _, r := range raw {
		fk, ok := byID[r.id]
		if !ok {
			fk = &sqldb.ForeignKeyInfo{
				Name:            fmt.Sprintf("fk_%d", r.id), // SQLite has no FK constraint names
				ReferencedTable: schema + "." + r.table,
				OnDelete:        normalizeFKAction(r.onDelete),
				OnUpdate:        normalizeFKAction(r.onUpdate),
			}
			byID[r.id] = fk
			order = append(order, r.id)
		}
		fk.Columns = append(fk.Columns, r.from)
		fk.ReferencedColumns = append(fk.ReferencedColumns, r.to)
	}
	out := make([]sqldb.ForeignKeyInfo, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out, nil
}

func normalizeFKAction(s string) string {
	switch s {
	case "NO ACTION", "":
		return "NO ACTION"
	case "RESTRICT", "CASCADE", "SET NULL", "SET DEFAULT":
		return s
	}
	return s
}

// --- connection methods ---

func (conn *connection) Schemas(ctx context.Context) ([]string, error) {
	return schemas(ctx, conn)
}
func (conn *connection) CurrentSchema(ctx context.Context) (string, error) {
	return currentSchema(ctx, conn)
}
func (conn *connection) Tables(ctx context.Context, schema ...string) ([]string, error) {
	return listSchemaObjects(ctx, conn, "table", schema)
}
func (conn *connection) TableExists(ctx context.Context, table string) (bool, error) {
	return objectExists(ctx, conn, "table", table)
}
func (conn *connection) Views(ctx context.Context, schema ...string) ([]string, error) {
	return listSchemaObjects(ctx, conn, "view", schema)
}
func (conn *connection) ViewExists(ctx context.Context, view string) (bool, error) {
	return objectExists(ctx, conn, "view", view)
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
func (*connection) Routines(ctx context.Context, schema ...string) ([]string, error) {
	return nil, errors.ErrUnsupported
}
func (*connection) RoutineExists(ctx context.Context, routine string) (bool, error) {
	return false, errors.ErrUnsupported
}

// --- transaction methods ---

func (conn *transaction) Schemas(ctx context.Context) ([]string, error) {
	return schemas(ctx, conn)
}
func (conn *transaction) CurrentSchema(ctx context.Context) (string, error) {
	return currentSchema(ctx, conn)
}
func (conn *transaction) Tables(ctx context.Context, schema ...string) ([]string, error) {
	return listSchemaObjects(ctx, conn, "table", schema)
}
func (conn *transaction) TableExists(ctx context.Context, table string) (bool, error) {
	return objectExists(ctx, conn, "table", table)
}
func (conn *transaction) Views(ctx context.Context, schema ...string) ([]string, error) {
	return listSchemaObjects(ctx, conn, "view", schema)
}
func (conn *transaction) ViewExists(ctx context.Context, view string) (bool, error) {
	return objectExists(ctx, conn, "view", view)
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
func (*transaction) Routines(ctx context.Context, schema ...string) ([]string, error) {
	return nil, errors.ErrUnsupported
}
func (*transaction) RoutineExists(ctx context.Context, routine string) (bool, error) {
	return false, errors.ErrUnsupported
}
