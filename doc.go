// Package sqldb provides database-agnostic interfaces and utilities
// for SQL database access with struct-to-row mapping.
//
// # Security model
//
// This package follows the same security model as [database/sql]: the query
// string is trusted SQL source written by the developer, and the separately
// passed arguments are untrusted data sent through driver placeholders.
//
// Several API functions accept raw SQL fragments in addition to the data
// arguments:
//
//   - The query parameter of [QueryRow], [QueryRowAs], [QueryRowAsOr],
//     [QueryRowAsStmt], [QueryRowAsMap], [QueryRowAsStrings],
//     [QueryRowAsStringsWithHeader], [QueryRowsAsMapSlice],
//     [QueryRowsAsSlice], [QueryRowsAsStrings], [QueryCallback],
//     [QueryStructCallback], [Exec], [ExecRowsAffected], [ExecStmt]
//     and [ExecRowsAffectedStmt].
//   - The whereCondition parameter of [Update], [UpdateReturningRow],
//     [UpdateReturningRows], [StdQueryBuilder.Update] and
//     [StdReturningQueryBuilder.UpdateReturning].
//   - The returningColumns parameter of [InsertReturning],
//     [UpdateReturningRow], [UpdateReturningRows],
//     [StdReturningQueryBuilder.InsertReturning] and
//     [StdReturningQueryBuilder.UpdateReturning].
//   - The conflictTarget parameter of [InsertUnique],
//     [InsertUniqueRowStruct] and [UpsertQueryBuilder.InsertUnique]. The
//     name keeps PostgreSQL terminology (ON CONFLICT) but each driver
//     translates the comma-separated column list into its own vendor
//     upsert syntax (PG/SQLite ON CONFLICT, MySQL ON DUPLICATE KEY UPDATE,
//     MSSQL/Oracle MERGE). The argument must NOT include those keywords.
//     The PostgreSQL/SQLite builders embed the column list verbatim; the
//     MySQL, MSSQL and Oracle builders split on commas and validate each
//     name.
//
// All of these are concatenated into the generated SQL without parameterization
// or validation. They MUST be written by the developer as static SQL and MUST
// NOT contain any data that originated outside the program (HTTP request body,
// query string, headers, JSON payload, filename, database content populated
// by external input, etc.). Pass such data through the arguments slice using
// the driver's placeholder syntax ($1, $2, ... for PostgreSQL; ?1, ?2, ... for
// SQLite; ? for MySQL; @p1, @p2, ... for SQL Server; :1, :2, ... for Oracle).
//
// Identifier parameters (table and column names) are validated by the
// [QueryFormatter]. The standard formatter and the PostgreSQL, MySQL, MSSQL
// and Oracle formatters reject names that do not match a conservative
// identifier regex and escape the result using the vendor-specific quoting
// scheme. Struct-based operations derive identifiers from `db:"..."` struct
// tags and the [Values] map keys; those tags and keys must be static strings
// chosen by the developer, not values derived from external input.
//
// The whereCondition parameter of Update and UpdateReturning is the
// boolean expression that follows the WHERE keyword. It must NOT include
// the WHERE keyword itself, which the builder emits.
//
// Example of a SAFE where condition (using PostgreSQL placeholder syntax;
// other drivers use their own — see the placeholder syntax list above):
//
//	err := db.Update(ctx, "public.user",
//	    db.Values{"name": newName},
//	    "id = $1 AND tenant_id = $2",
//	    userID, tenantID,
//	)
//
// Example of an UNSAFE where condition (DO NOT DO THIS):
//
//	// SQL injection: filter is attacker-controlled
//	err := db.Update(ctx, "public.user",
//	    db.Values{"name": newName},
//	    filter,
//	)
package sqldb
