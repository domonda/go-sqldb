# information

Helpers for querying `information_schema` metadata: tables, views, columns,
key usage, primary keys, domains, check constraints.

The struct tags and SQL queries in this package were written against
PostgreSQL. Despite `information_schema` being part of the SQL standard
(ISO/IEC 9075-11), real-world support across vendors is uneven — the
sections below document where this package works today, where it
silently falls back to empty/null fields, and where it breaks outright.

## Vendor support for `information_schema`

| Vendor                    | `information_schema` available? | Notes                                                                                       |
| ------------------------- | ------------------------------- | ------------------------------------------------------------------------------------------- |
| PostgreSQL 9+             | Yes (full)                      | Reference implementation. All struct fields populated.                                      |
| MySQL 5.7 / 8.0+          | Yes (partial)                   | Different data type spelling; `table_catalog` is always literal `'def'`; no `domains` view. |
| MariaDB 10.5+             | Yes (partial)                   | Adds `domains` view; otherwise behaves like MySQL.                                          |
| SQL Server 2016+ (T-SQL)  | Yes (partial)                   | ISO subset only; many Postgres-specific columns missing. Use `sys.*` for complete metadata. |
| SQLite                    | No                              | Uses `sqlite_schema` and `PRAGMA table_info(...)` instead. Not addressable by this package. |
| Oracle                    | No                              | Uses `USER_TABLES`/`ALL_TABLES`/`DBA_TABLES` data-dictionary views. Different shape.        |

## View / struct compatibility

`✓` = view exists with the queried columns. `partial` = view exists but
some struct fields will be empty/null. `✗` = view does not exist.

| Struct (`information_schema` view)           | PostgreSQL | MySQL 8.0+ | MariaDB 10.5+ | SQL Server | SQLite | Oracle |
| -------------------------------------------- | ---------- | ---------- | ------------- | ---------- | ------ | ------ |
| `Schema` (`schemata`)                        | ✓          | partial    | partial       | partial    | ✗      | ✗      |
| `Table` (`tables`)                           | ✓          | partial    | partial       | partial    | ✗      | ✗      |
| `View` (`views`)                             | ✓          | partial    | partial       | partial    | ✗      | ✗      |
| `Column` (`columns`)                         | ✓          | partial    | partial       | partial    | ✗      | ✗      |
| `KeyColumnUsage` (`key_column_usage`)        | ✓          | ✓          | ✓             | ✓          | ✗      | ✗      |
| `Domain` (`domains`)                         | ✓          | ✗          | ✓             | ✗          | ✗      | ✗      |
| `CheckConstraints` (`check_constraints`)     | ✓          | ✓ (8.0.16+)| ✓             | ✓          | ✗      | ✗      |
| `table_constraints` (used by `GetPrimaryKeyColumns`) | ✓ | ✓        | ✓             | ✓          | ✗      | ✗      |

### Columns that go missing on non-Postgres vendors

The `Column` struct (mapping `information_schema.columns`) is the most
exposed to vendor drift. Selecting `*` against MySQL/MariaDB/SQL Server
will return only the columns each vendor implements; the others scan as
empty `String` or nil `*int`.

| `Column` field          | PostgreSQL | MySQL/MariaDB | SQL Server |
| ----------------------- | ---------- | ------------- | ---------- |
| `TableCatalog`          | real db    | always `'def'`| real db    |
| `IsNullable`            | `'YES'`/`'NO'` | `'YES'`/`'NO'` | `'YES'`/`'NO'` |
| `DataType`              | `integer`, `character varying`, `jsonb`, `timestamp with time zone` | `int`, `varchar`, `json`, `timestamp` | `int`, `nvarchar`, `datetime2`, ... |
| `CharacterSet*`         | populated for char types | populated | not populated |
| `Collation*`            | populated  | populated     | not populated |
| `Domain*`               | populated  | not populated | not populated |
| `UDT*`                  | populated  | not populated | not populated |
| `Scope*`, `MaximumCardinality`, `DTDIdentifier` | populated | not populated | not populated |
| `IsIdentity`, `Identity*` | populated (PG 10+) | populated (MySQL 8.0+) | not populated (use `sys.identity_columns`) |
| `IsGenerated`, `GenerationExpression` | populated | populated (MySQL 5.7+) | not populated |
| `IsSelfReferencing`     | populated  | not populated | not populated |
| `IsUpdatable`           | populated  | populated     | not populated |

### Helper-function compatibility

| Function                       | Works on                  | Caveats |
| ------------------------------ | ------------------------- | ------- |
| `GetTable`                     | PG, MySQL, MariaDB, MSSQL | Returns rows for both base tables AND views (and on some vendors foreign tables, sequences, etc.) — check `Table.TableType` to distinguish. On MySQL/MariaDB pass catalog `"def"`. Many returned struct fields scan empty on non-PG vendors. |
| `TableExists`                  | PG, MySQL, MariaDB, MSSQL | Returns true for tables AND views (per SQL standard). Unqualified name matches across all schemas. |
| `GetAllTables`                 | PG, MySQL, MariaDB, MSSQL | Includes views and any other relation kinds the vendor exposes via `information_schema.tables`. Same struct-field caveats as `GetTable`. |
| `ColumnExists`                 | PG, MySQL, MariaDB, MSSQL | Matches columns of base tables AND views (per SQL standard). Unqualified relation name matches across all schemas. |
| `GetPrimaryKeyColumns`         | PG, MySQL, MariaDB, MSSQL | `Table` is composed in Go as `"schema.table"`. |
| `GetPrimaryKeyColumnsOfType`   | PG, MySQL, MariaDB, MSSQL | `pkType` must match the vendor's `data_type` spelling. |
| `GetTableRowsWithPrimaryKey`   | All vendors               | No `information_schema` reference; identifier and placeholder are formatted per driver. Requires `pkCols` populated for the vendor. |
| `RenderUUIDPrimaryKeyRefsHTML` | PG (de facto)             | Filters `data_type = 'uuid'`; matches PG and MariaDB 10.7+. Other vendors return no rows. |

### Tables vs. views

`information_schema.tables` and `information_schema.columns` are defined by
ISO/IEC 9075-11 to include rows for both base tables and views. Every
vendor in the matrix above honors this. If you need to restrict to base
tables only, filter on `Table.TableType` (or query
`information_schema.tables.table_type` directly). Per-vendor enumerations
verified against live servers:

| Vendor      | `table_type` values                                                     |
| ----------- | ----------------------------------------------------------------------- |
| PostgreSQL  | `BASE TABLE`, `VIEW`, `FOREIGN`, `LOCAL TEMPORARY`                      |
| MySQL       | `BASE TABLE`, `VIEW`, `SYSTEM VIEW`                                     |
| MariaDB     | `BASE TABLE`, `VIEW`, `SYSTEM VIEW`, `SEQUENCE`, `TEMPORARY`, `SYSTEM VERSIONED` |
| SQL Server  | `BASE TABLE`, `VIEW`                                                    |

SQLite and Oracle are not supported by any helper that touches
`information_schema`, because neither vendor exposes those views.

## Tests

Each supported vendor has its own integration-test module that runs the
helpers against a live dockerized server:

| Module                                                       | Vendor      | Compose file                          |
| ------------------------------------------------------------ | ----------- | ------------------------------------- |
| `information/postgres_information_test`                      | PostgreSQL  | `pqconn/test/docker-compose.yml`      |
| `information/mysql_information_test`                         | MariaDB     | `mysqlconn/test/docker-compose.yml`   |
| `information/mssql_information_test`                         | SQL Server  | `mssqlconn/test/docker-compose.yml`   |

`./test-workspace.sh` from the repo root runs all three. The MariaDB and
SQL Server tests share the docker-compose files (and the running
containers) used by `mysqlconn/test/` and `mssqlconn/test/` and use
table names prefixed `info_test_*` to avoid collisions.

## Portability mechanics

The package now routes its queries through dialect-neutral primitives so
the same Go calls produce vendor-correct SQL. The underlying constraints
that motivated each choice:

1. **Placeholders** — every query builds its placeholders via
   `conn.FormatPlaceholder(i)`, so the emitted SQL uses `$1`/`$2`/... on
   PostgreSQL, `?` on MySQL/SQLite, `@p1`/`@p2`/... on SQL Server, and
   `:1`/`:2`/... on Oracle.
2. **String concatenation** — `||` has been removed from
   `primarykeys.go`. The schema and table name are returned as separate
   columns and composed in Go (`Schema + "." + TableName`), which sidesteps
   the `||` (PG/Oracle/SQLite) vs `CONCAT()` (MySQL/MSSQL) split entirely.
3. **EXISTS in projection** — `TableExists`, `ColumnExists` and the
   foreign-key flag in `GetPrimaryKeyColumns` use the portable
   `CASE WHEN EXISTS(...) THEN 1 ELSE 0 END` form (SQL Server forbids
   bare `EXISTS` in a SELECT list). The result is scanned as `int` and
   converted to `bool` in Go because not every driver maps `0/1` to a Go
   `bool` on Scan.
4. **No "public" default** — `TableExists` and `ColumnExists` no longer
   substitute a Postgres-flavored default schema when the caller passes
   an unqualified name. Unqualified means "any schema" — useful for a
   quick existence check, but be aware that on multi-schema databases it
   matches a same-named table in any schema.
5. **Identifier quoting** — `GetTableRowsWithPrimaryKey` runs
   `col.Table` through `conn.FormatTableName` and `col.Column` through
   `conn.FormatColumnName`, so each driver applies its own quoting rules
   (`"id"` for PG/SQLite/standard SQL, `` `id` `` for MySQL,
   `[id]` for SQL Server).
6. **Type literals** — `data_type` values used in queries
   (`GetPrimaryKeyColumnsOfType`, the UUID HTML handler) are still
   vendor-specific strings: `"uuid"` for PG, `"uniqueidentifier"` for
   SQL Server, etc. Callers must pass a value that matches the target
   vendor's `information_schema.columns.data_type` spelling.

## What still won't work cross-vendor

1. **SQLite and Oracle** — neither exposes `information_schema`, so
   every query in this package fails on those drivers. SQLite uses
   `sqlite_schema` and `PRAGMA`; Oracle uses `USER_TABLES` /
   `ALL_TABLES`. Supporting them requires per-driver implementations
   behind a small interface (the `sqldb.Information` surface in
   `information.go`, embedded into `sqldb.Connection`) — not a SQL
   rewrite.
2. **Sparsely populated structs** — `Table`, `Column`, `View`, `Schema`,
   and `Domain` keep the full ISO column list. On MySQL/MariaDB and SQL
   Server many of those columns are absent from the underlying view, so
   the corresponding struct fields scan as empty/nil. The per-type godoc
   spells out which fields actually populate where.
3. **Vendor-specific data type spellings** — `GetPrimaryKeyColumnsOfType`
   takes a literal that must match `information_schema.columns.data_type`
   on the target vendor. Callers carry that knowledge.
4. **`RenderUUIDPrimaryKeyRefsHTML`** — calls
   `GetPrimaryKeyColumnsOfType(ctx, conn, "uuid")`, hard-coding the
   Postgres spelling. Effectively PostgreSQL-only; documented on the
   function.
