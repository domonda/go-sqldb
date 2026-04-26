package db

import (
	"context"

	"github.com/domonda/go-sqldb"
)

// Schemas returns the names of schemas visible to the connected
// user.
//
// On vendors that don't have a separate schema concept inside a
// database (MySQL, MariaDB), this returns the names of databases
// the connected user can see. On PostgreSQL, MSSQL, and Oracle, it
// returns proper schema names within the current database.
func Schemas(ctx context.Context) ([]string, error) {
	return Conn(ctx).Schemas(ctx)
}

// CurrentSchema returns the database's current default schema. Used
// to resolve unqualified identifiers in this package's
// existence-check functions.
//
// Vendor mapping:
//   - PostgreSQL: current_schema(), the first non-empty existing
//     entry in search_path.
//   - MySQL/MariaDB: database(), the connected database.
//   - SQL Server: SCHEMA_NAME(), the user's default schema
//     (typically "dbo").
//   - Oracle: SYS_CONTEXT('USERENV','CURRENT_SCHEMA'), the
//     connected user's schema.
func CurrentSchema(ctx context.Context) (string, error) {
	return Conn(ctx).CurrentSchema(ctx)
}

// Tables returns the schema-qualified names of base tables in the
// connected database, optionally filtered by one or more schemas.
//
// Views are excluded — use [Views] to enumerate those. The result
// includes only objects whose `information_schema.tables.table_type`
// is `BASE TABLE` (or the vendor equivalent), so system tables,
// temporary tables, and foreign tables are also excluded unless the
// vendor reports them as `BASE TABLE`.
//
// Each returned string is `schema.name`. With no schema arguments
// every visible schema is included; with one or more it is
// restricted to those.
func Tables(ctx context.Context, schema ...string) ([]string, error) {
	return Conn(ctx).Tables(ctx, schema...)
}

// TableExists reports whether a base table with the given name
// exists. The argument is either `schema.name` or just `name`, in
// which case the database's default schema is used. Views do not
// count — use [ViewExists] for those.
func TableExists(ctx context.Context, table string) (bool, error) {
	return Conn(ctx).TableExists(ctx, table)
}

// Views returns the schema-qualified names of views in the connected
// database, optionally filtered by one or more schemas.
//
// Each returned string is `schema.name`. Materialized views are
// included only if the vendor reports them in
// `information_schema.views` (PostgreSQL does not; query
// `pg_matviews` separately if you need them).
func Views(ctx context.Context, schema ...string) ([]string, error) {
	return Conn(ctx).Views(ctx, schema...)
}

// ViewExists reports whether a view with the given name exists. The
// argument is either `schema.name` or just `name`, in which case the
// database's default schema is used. Base tables do not count — use
// [TableExists] for those.
func ViewExists(ctx context.Context, view string) (bool, error) {
	return Conn(ctx).ViewExists(ctx, view)
}

// Columns returns the columns of a table or view, in the order
// reported by the catalog (typically declaration order). The
// tableOrView argument is either `schema.name` or just `name`, in
// which case the database's default schema is used.
//
// Returns a wrapped sql.ErrNoRows if no table and no view with the
// given name exist.
//
// Returns `errors.ErrUnsupported` if the connected database does
// not expose column metadata via this interface (e.g. Oracle
// implementations may require a different code path).
func Columns(ctx context.Context, tableOrView string) ([]sqldb.ColumnInfo, error) {
	return Conn(ctx).Columns(ctx, tableOrView)
}

// ColumnExists reports whether a column with the given name exists
// on the named table or view. The tableOrView argument is either
// `schema.name` or just `name`, in which case the database's
// default schema is used. Column matching is case-sensitive on
// databases that fold or quote-preserve case (PostgreSQL); on
// case-insensitive databases (MySQL by default, SQL Server with
// case-insensitive collation) the underlying database determines
// the match.
//
// Returns a wrapped sql.ErrNoRows if no table and no view with the
// given name exist. The bool return is meaningful only when err is
// nil; "column not found on an existing relation" returns
// (false, nil).
func ColumnExists(ctx context.Context, tableOrView, column string) (bool, error) {
	return Conn(ctx).ColumnExists(ctx, tableOrView, column)
}

// PrimaryKey returns the primary key column names of the given
// table, in the order they appear in the constraint definition
// (NOT in column declaration order). The table argument is
// `schema.name` or just `name`, in which case the database's
// default schema is used.
//
// Why ordering matters: a `PRIMARY KEY (tenant_id, user_id)` is a
// different index from `PRIMARY KEY (user_id, tenant_id)`. The
// difference is invisible if you derive PK columns by filtering
// [Columns] for `ColumnInfo.PrimaryKey == true`, because that path
// returns columns in their declaration order, not the constraint's
// ordinal order. PrimaryKey preserves the constraint's order, which
// is what callers need for:
//   - Building WHERE clauses that hit the PK index in the right
//     order on databases sensitive to predicate ordering.
//   - Generating positional upsert keys that match the actual
//     constraint definition.
//   - Reproducing or comparing constraint definitions across
//     environments.
//
// Returns an empty slice and no error if the table exists but has
// no primary key. Returns a wrapped sql.ErrNoRows if the named
// relation does not exist OR exists but is a view (views have no
// primary key); use [Columns] if you need the columns of a view.
func PrimaryKey(ctx context.Context, table string) ([]string, error) {
	return Conn(ctx).PrimaryKey(ctx, table)
}

// ForeignKeys returns the foreign key constraints declared on the
// given table. The table argument is `schema.name` or just `name`,
// in which case the database's default schema is used.
//
// Each [sqldb.ForeignKeyInfo] entry describes one constraint. Composite
// foreign keys produce a single entry whose Columns and
// ReferencedColumns slices have matching length and ordering — the
// i-th local column references the i-th remote column. Ordering
// follows the constraint declaration.
//
// Returns an empty slice and no error if the table exists but has
// no foreign keys. Returns a wrapped sql.ErrNoRows if the named
// relation does not exist OR exists but is a view (views cannot
// declare foreign keys).
func ForeignKeys(ctx context.Context, table string) ([]sqldb.ForeignKeyInfo, error) {
	return Conn(ctx).ForeignKeys(ctx, table)
}

// Routines returns the names of stored functions and procedures,
// optionally filtered by one or more schemas.
//
// Each returned string is a schema-qualified routine signature in the
// form `schema.name(argtype, argtype, ...)`, with explicit empty
// parentheses `()` for zero-argument routines. Argument type names use
// the vendor's `information_schema.parameters.data_type` spelling
// (e.g. PostgreSQL `integer` vs MySQL `int`).
//
// PostgreSQL allows overloading routines on argument signature; each
// overload is returned as a separate entry. Other supported vendors do
// not allow overloading, so a name will appear at most once.
//
// Each returned string is directly usable in DROP FUNCTION / DROP
// PROCEDURE statements and as the argument to [RoutineExists].
func Routines(ctx context.Context, schema ...string) ([]string, error) {
	return Conn(ctx).Routines(ctx, schema...)
}

// RoutineExists reports whether a routine matching the given identifier
// exists. The presence of parentheses in the argument selects between
// two matching modes:
//
// Signature match — the argument contains `(` and `)`:
// it must be in the same `schema.name(argtype, ...)` form returned by
// [Routines] (use `schema.name()` for zero-argument routines). The
// function returns true only if a routine with that exact signature
// exists. Use this form to disambiguate PostgreSQL overloads or to
// pre-check before DROP FUNCTION / DROP PROCEDURE.
//
// Name match — the argument has no parentheses:
// it is either `schema.name` or just `name`. An unqualified name
// resolves against the database's default schema (PostgreSQL
// search_path, MySQL/MariaDB current database, SQL Server default
// schema). The function returns true if at least one routine with
// that name exists in the resolved schema. On PostgreSQL multiple
// routines can match due to overloading; the function still returns
// true and does not report how many matched. Use [Routines] if you need
// the individual signatures.
func RoutineExists(ctx context.Context, routine string) (bool, error) {
	return Conn(ctx).RoutineExists(ctx, routine)
}
