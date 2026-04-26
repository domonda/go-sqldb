# oraconn

[![Go Reference](https://pkg.go.dev/badge/github.com/domonda/go-sqldb/oraconn.svg)](https://pkg.go.dev/github.com/domonda/go-sqldb/oraconn)

Oracle Database driver for [go-sqldb](https://github.com/domonda/go-sqldb) using [github.com/sijms/go-ora/v2](https://github.com/sijms/go-ora).

## Connecting

```go
config := &sqldb.Config{
    Driver:   oraconn.Driver, // "oracle"
    Host:     "localhost",
    Port:     1521,
    User:     "myuser",
    Password: "secret",
    Database: "FREEPDB1", // service name
}

conn, err := oraconn.Connect(ctx, config, true)
```

The `Database` field is used as the Oracle service name.
Extra connection parameters can be passed via the `Extra` map.

The third parameter `lowercaseColumns` controls whether column names returned
by `Rows.Columns()` are lowercased. Oracle returns uppercase names for unquoted
identifiers (e.g. `SELECT id` returns column `ID`), but Go struct `db` tags
conventionally use lowercase. Set to `true` when using struct scanning with
lowercase tags. Oracle SQL itself is case-insensitive for unquoted identifiers,
so this only affects the Go-side column name matching used by the sqldb struct
reflector.

## Query formatting

- **Placeholders**: `:1`, `:2`, ...
- **Identifier quoting**: `"double quotes"` (applied automatically for reserved words)
- **String literals**: standard single-quote doubling

## Query builders

`oraconn.QueryBuilder` implements `sqldb.QueryBuilder` and `sqldb.UpsertQueryBuilder`:

- Standard SQL operations via embedded `sqldb.StdQueryBuilder` (with `Update` overridden to reorder arguments for Oracle's positional `:N` binding)
- **Upsert** via Oracle `MERGE INTO ... USING (SELECT ... FROM DUAL) ...`
- **InsertUnique** via MERGE with only `WHEN NOT MATCHED`

`ReturningQueryBuilder` is not supported because Oracle's `RETURNING ... INTO` syntax
is incompatible with the row-returning interface.

## Error inspection

Oracle errors are mapped to generic `sqldb` error types:

| Oracle Error | sqldb Error | Helper Function |
|---|---|---|
| ORA-00001 | `ErrUniqueViolation` | `IsUniqueViolation` |
| ORA-01400 | `ErrNotNullViolation` | `IsNotNullViolation` |
| ORA-02291, ORA-02292 | `ErrForeignKeyViolation` | `IsForeignKeyViolation` |
| ORA-02290 | `ErrCheckViolation` | `IsCheckViolation` |
| ORA-00060 | `ErrDeadlock` | `IsDeadlockDetected` |
| ORA-08177 | `ErrSerializationFailure` | `IsSerializationFailure` |
| ORA-01013 | `ErrQueryCanceled` | `IsQueryCanceled` |
| ORA-20000–20999 | `ErrRaisedException` | — |

## Schema introspection

`oraconn` implements `sqldb.Information` using Oracle's `ALL_*` data-dictionary views (Oracle does not expose `information_schema`). In Oracle a "schema" is a user, so `Schemas` returns user names rather than schema names.

| Method            | Source                                                                                                |
| ----------------- | ----------------------------------------------------------------------------------------------------- |
| `Schemas`         | `ALL_USERS WHERE oracle_maintained = 'N'` — excludes `SYS`, `SYSTEM`, GoldenGate users, and other Oracle-maintained accounts |
| `CurrentSchema`   | `SYS_CONTEXT('USERENV','CURRENT_SCHEMA')`                                                              |
| `Tables`/`Views`  | `ALL_TABLES` / `ALL_VIEWS`, restricted to non-Oracle-maintained owners                                 |
| `Columns`         | `ALL_TAB_COLUMNS` joined with `ALL_CONSTRAINTS` for primary-key flags                                  |
| `PrimaryKey`      | `ALL_CONSTRAINTS` + `ALL_CONS_COLUMNS`, ordered by `position` (constraint-declaration order)           |
| `ForeignKeys`     | `ALL_CONSTRAINTS` + `ALL_CONS_COLUMNS` with composite-FK column ordering preserved                     |
| `Routines`        | `ALL_OBJECTS` filtered to top-level `PROCEDURE` and `FUNCTION` objects                                  |
| `RoutineExists`   | Signature-match when the argument contains `(`, otherwise name-match in the resolved schema           |

**Caveats specific to Oracle:**

- **Case folding:** Oracle stores unquoted identifiers as uppercase in the catalog (e.g. `CREATE TABLE foo` is stored as `FOO`). Compare names case-insensitively or pass identifiers in uppercase. `conntest.InformationFeatures.CaseFoldsToUpper = true` flips the conntest assertions for this.
- **`OnUpdate` is always `"NO ACTION"`:** Oracle does not support cascading updates. The catalog only records the `ON DELETE` action; the driver populates `ForeignKeyInfo.OnUpdate` with `"NO ACTION"` for every FK so the field has a consistent value.
- **Routines exclude package bodies:** `Routines` returns top-level standalone procedures and functions only. Routines defined inside `PACKAGE` bodies are intentionally excluded because their fully qualified name (`schema.package.routine`) does not fit the `schema.name(args)` shape the interface contracts.

## Drop queries

For resetting test databases:

- `DropAllTables(ctx, conn)` — drops all foreign keys, then all user tables
- `DropAllTypes(ctx, conn)` — drops all user-defined types
- `DropAll(ctx, conn)` — both in correct order

## Integration tests

Start the test Oracle instance:

```bash
cd test
docker compose up -d
```

This uses [`gvenzl/oracle-free:23-slim-faststart`](https://hub.docker.com/r/gvenzl/oracle-free) on port 1522.
The `slim-faststart` tag ships a pre-initialized database baked into the image,
eliminating the database creation step on first start.

The container is **ephemeral** — no volume is mounted, so `docker compose down` + `up`
gives a completely fresh database. To keep the container running between test runs
(faster re-runs), just leave it up — `TestMain` calls `DropAll` before each test run.

Tunings applied for test workloads:
- `shm_size: 1g` — Oracle needs more shared memory than Docker's default 64MB
- `healthcheck.sh` — built-in readiness check
- `initdb/00-tune-for-tests.sql` — disables recyclebin (faster DDL) and reduces process limit

Run tests:

```bash
go test -v -count=1 -timeout 120s ./test/...
```
