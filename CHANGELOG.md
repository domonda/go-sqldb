# Changelog

All notable changes to **go-sqldb** are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions follow the [Go module versioning](https://go.dev/ref/mod#versions)
scheme: a `v` prefix followed by three [semantic-versioning](https://semver.org/spec/v2.0.0.html)
numbers (`vMAJOR.MINOR.PATCH`). A `!` after the conventional-commit type
(e.g. `feat(sqldb)!:`) marks a commit that contains a breaking change.

The driver sub-modules (`pqconn`, `mysqlconn`, `mssqlconn`, `sqliteconn`,
`oraconn`) are tagged separately as `<module>/vX.Y.Z` and released in lockstep
with the root module.

## [v1.4.0] - 2026-06-19

Pin a connection to one backend session for session-scoped state like
PostgreSQL `pg_advisory_lock`.

[Diff](https://github.com/domonda/go-sqldb/compare/v1.3.1...v1.4.0)

### Added

- `sqldb.ConnPinner` interface and `sqldb.PinConn(ctx, conn)` helper check out
  one dedicated session from the pool and return a `sqldb.PinnedConnection`
  pinned to it for the lifetime of the value. Every query runs on the same
  `*sql.Conn`, and `database/sql` does not reap checked-out sessions
  (`ConnMaxLifetime` / `ConnMaxIdleTime` don't apply), so it is the right
  primitive for session-scoped state such as `pg_advisory_lock`,
  `SET SESSION ...`, and temporary tables. `Close` returns the session to the
  pool; `Begin` starts a real transaction on the same pinned session. `PinConn`
  on a transaction returns `ErrWithinTransaction`, and wraps
  `errors.ErrUnsupported` when the driver has no pool. (`561b194`)
- `db.PinnedConn` / `db.PinnedConnResult[T]` run a callback with the context
  connection pinned to one session, then return it to the pool even on panic.
  An existing transaction or already-pinned connection is passed through
  unchanged, and a nested `db.Transaction` begins on and inherits the pinned
  session. (`561b194`)
- `sqldb.ConnPinner` implemented by `pqconn`, `mysqlconn`, `mssqlconn`, and
  `oraconn`. `sqliteconn` does not implement it, having no connection pool.
  (`561b194`)

### For contributors

- `test-workspace.sh` passes `-quiet` to gosec, so a clean run produces no
  output. (`1a387da`)

## [v1.3.1] - 2026-06-19

Hardened, standard-library-faithful value scanning.

[Diff](https://github.com/domonda/go-sqldb/compare/v1.3.0...v1.3.1)

### Changed

- `ScanDriverValue` reworked into a best-effort converter that mirrors the
  standard library `database/sql` conversions far more closely. In addition to
  the existing overflow / precision checks it now:
  - rejects a nil pointer destination up front (new `errNilDestPtr` message);
  - composes decimals directly when both destination and value implement the
    `database/sql` decimal `Compose`/`Decompose` interfaces;
  - converts `int64`/`float64` values of `0` or `1` to `bool`;
  - formats `bool`, `int64`, `float64` and `time.Time` into `string` and
    `[]byte` destinations, and parses `string`/`[]byte` into `bool`, integer,
    unsigned and floating-point destinations (via the new `scanStringInto`
    helper, so user-defined types like `type Int int64` scan correctly);
  - formats `time.Time` with `time.RFC3339Nano`, and assigns into named struct
    types whose underlying type is `time.Time`;
  - clones driver `[]byte` values before assigning, because the driver may
    reuse the backing array after the call.
- `mockconn.MockRows` keeps its own verbatim copy of the standard library
  `convertAssign` (renamed sentinel `errNilPtr` → `errNilDestPtr`) instead of
  routing through `ScanDriverValue`, so mock scans stay byte-for-byte faithful
  to `*sql.Rows` even as `ScanDriverValue` diverges with sqldb-specific
  behavior.

## [v1.3.0] - 2026-05-15

Opt-in argument redaction for logs and errors, hardened numeric scanning,
safer multi-process SQLite defaults, plus a small breaking interface addition
for custom `QueryFormatter` implementers.

[Release notes](https://github.com/domonda/go-sqldb/releases/tag/v1.3.0) ·
[Diff](https://github.com/domonda/go-sqldb/compare/v1.2.0...v1.3.0)

### Breaking changes

- `sqldb.QueryFormatter` gains one method,
  `SubstitutePlaceholders(query string, args []any) (string, error)`. Any
  out-of-repo `QueryFormatter` implementer must add it (one line, delegating
  to the shared `sqldb.SubstitutePlaceholders` helper). All five built-in
  driver formatters are already updated; for everyone else this is a drop-in
  upgrade. (`5b87305`)

### Added

- `sqldb.Secret` interface + `sqldb.KeepSecret(val)` constructor for
  per-argument redaction. Secret-wrapped arguments render as
  `'***REDACTED***'` everywhere the query is formatted while the real value
  still reaches the driver (the wrapper satisfies `driver.Valuer` and
  `sql.Scanner`). Shares the interface shape of `errs.Secret` from
  [go-errs](https://github.com/domonda/go-errs) by convention. (`1f0d940`)
- `sqldb.ConnectionWithoutPlaceholderSubstitution(conn)` wraps a `Connection`
  so placeholders stay literal in formatted output; the wrapper overrides
  `Begin` so transactions inherit the no-substitution behavior. (`da8e759`)
- `sqldb.SubstitutePlaceholders(formatter, query, args)` shared helper, plus
  the exported `sqldb.TrimSurroundingWhitespace(query)` helper and a
  `StdQueryFormatter.DisableSubstitutePlaceholders` toggle. (`5b87305`)
- `sqliteconn.DefaultBusyTimeoutMs` (`5000`) plus connect-time
  `PRAGMA busy_timeout`, overridable via `Config.Extra["busy_timeout"]` (`"0"`
  restores SQLite's native fail-fast behavior). The full multi-process
  concurrency model is documented in `sqliteconn/README.md`. (`f9535da`)

### Changed

- `ScanDriverValue` hardened to reject `int64`/`float64` overflow of the
  destination type, reject lossy `float64`→integer conversions, check
  assignability before scanning into non-empty interface destinations, and
  handle pointer-to-pointer destinations. (`714af7d`)
- `FormatValue` redacts `sqldb.Secret` values before the `driver.Valuer`
  branch runs. `FormatQuery` trims surrounding whitespace and, on per-argument
  formatting failure, substitutes a `<FORMATERROR>` sentinel so positional
  alignment survives while the joined error rides along via `errors.Join`.
  (`1f0d940`, `5b87305`)

## [v1.2.0] - 2026-04-26

A vendor-portable schema-introspection surface embedded into `Connection`, a
documented SQL-injection model, SQLite identifier-validation parity, and a
broader `oraconn.DropAll`.

[Release notes](https://github.com/domonda/go-sqldb/releases/tag/v1.2.0) ·
[Diff](https://github.com/domonda/go-sqldb/compare/v1.1.0...v1.2.0)

### Breaking changes

- `sqldb.Connection` now embeds the new `sqldb.Information` interface
  (`Schemas`, `CurrentSchema`, `Tables`, `TableExists`, `Views`, `ViewExists`,
  `Columns`, `ColumnExists`, `PrimaryKey`, `ForeignKeys`, `Routines`,
  `RoutineExists`). External `Connection` implementers must add these twelve
  methods or embed `sqldb.ErrConn`. (`41bd5f3`)
- `sqldb.ColumnInfo.IsEmbeddedField()` removed; test the documented empty-name
  sentinel with `column.Name == ""` instead. (`67f056d`)
- Raw-SQL-fragment parameters renamed for clarity (`where` → `whereCondition`,
  `returning` → `returningColumns`, `onConflict` → `conflictTarget`) across the
  query builders, the `sqldb.*` functions, and the `db.*` wrappers. Go does not
  require named call sites, so no caller code breaks. (`29817b8`)

### Added

- `sqldb.Information` interface for vendor-portable catalog access, implemented
  by every driver against its native catalog (`pg_catalog`,
  `information_schema`, `sys.*`, `sqlite_schema` + `PRAGMA`, Oracle `ALL_*`).
  Methods that don't apply on a vendor return `errors.ErrUnsupported`. The `db`
  package exposes ctx-first wrappers (`db.Tables`, `db.TableExists`,
  `db.PrimaryKey`, …). (`41bd5f3`)
- `sqldb.ColumnInfo.Type` field, populated by
  `TaggedStructReflector.MapStructField`, serving both struct reflection and
  database introspection. (`67f056d`)
- `information` subpackage made vendor-neutral, with a new
  `information/README.md` compatibility matrix and MariaDB / SQL Server
  integration-test modules. (`1d3ffc4`)
- `sqliteconn` `QueryFormatter` validates identifier names with a conservative
  regex, matching the other drivers. (`34a751b`)
- `oraconn.DropAll` extended to cover synonyms, views,
  procedures/functions/packages, tables, types and sequences in dependency
  order, plus new per-kind exported helpers. (`46dac87`)

### Documentation

- New top-level Security Model section in `doc.go` and `README.md` listing
  every raw-SQL-fragment parameter with paired SAFE / UNSAFE examples, plus
  fixes to two buggy README examples and removal of a stale SQLite claim.
  (`29817b8`, `259c3ae`)

## [v1.1.0] - 2026-04-15

[Release notes](https://github.com/domonda/go-sqldb/releases/tag/v1.1.0) ·
[Diff](https://github.com/domonda/go-sqldb/compare/v1.0.13...v1.1.0)

### Breaking changes

- `sqldb.QueryRowsAsSlice`, `sqldb.QueryRowsAsStrings`, and
  `sqldb.QueryRowsAsMapSlice` now require a `maxNumRows int` argument before
  the query string. Pass `sqldb.UnlimitedMaxNumRows` (or any negative integer)
  to keep the previous unlimited behavior. Callers of the `db.QueryRows*`
  wrappers are unaffected.

### Added

- Explicit `maxNumRows` cap with partial-results semantics for every
  slice-returning query function: the new `ErrMaxNumRowsExceeded` sentinel is
  returned (wrapped with the query) when the cap is hit, and the rows scanned
  so far still come back so callers can consume them via `errors.As`.
- `db` wrappers read the cap from the context via
  `ContextWithMaxNumRows(ctx, n)`, keeping the variadic `QueryRowsAs*`
  signatures.
- `db` now aliases the full `ScanConverter` family plus `UnlimitedMaxNumRows`
  and `ErrMaxNumRowsExceeded`, so business code never has to import `sqldb`.
- `BytesToStringScanConverter` emits uppercase hex via `%X` (`\xDEADBEEF`),
  with the JSON-escaping behavior documented.

## [v1.0.13] - 2026-04-14

[Diff](https://github.com/domonda/go-sqldb/compare/v1.0.12...v1.0.13)

### Added

- `QueryRowsAsMapSlice` at the `sqldb` and `db` package level, the multi-row
  counterpart of `Row.ScanMap`. Each row becomes a `map[string]any` keyed by
  column name, with an optional nil-safe `ScanConverter` (combine several via
  `sqldb.ScanConverters`).

## [v1.0.12] - 2026-04-10

[Diff](https://github.com/domonda/go-sqldb/compare/v1.0.5...v1.0.12)

### Changed

- Listener management refactored from global to per-connection (#20).
- `github.com/lib/pq` bumped to v1.12.3 across all modules.

## [v1.0.1] – [v1.0.11] - 2026-03-31 … 2026-04-10

A run of incremental patch releases in the days after v1.0.0. Highlights
across the series:

- Scan converters, `Row.ScanMap`, and `QueryRowAsStrings` variants;
  `TimeToStringScanConverter`.
- Global logger and listener configuration moved into the per-connection
  `Config`; assorted `pqconn` listener fixes (`pq.ErrChannelAlreadyOpen`
  handling, password-carrying listener URL, channel race fixes).
- `pqconn` class-level `Is…Class` predicates for all PostgreSQL error classes.
- `TaggedStructReflector` gains `FailOnUnmappedStructFields` /
  `FailOnUnmappedColumns`; reflect methods moved onto the `StructReflector`
  interface with `TypeWrapper` support.
- `QueryRowByPrimaryKey` renamed to `QueryRowStruct`; `QueryRowAs2`–
  `QueryRowAs5` added; `db.Values` alias added.
- `github.com/lib/pq` v1.12.2.

See the individual [release tags](https://github.com/domonda/go-sqldb/tags)
for the precise per-version boundaries.

## [v1.0.0] - 2026-03-30

First stable release. Preceded by release candidates
[`v1.0.0-rc1`](https://github.com/domonda/go-sqldb/releases/tag/v1.0.0-rc1)
through `v1.0.0-rc4` (2026-03-16 … 2026-03-30).

[Diff](https://github.com/domonda/go-sqldb/compare/v0.1.0...v1.0.0)

### Added

- `oraconn` Oracle database driver built on `go-ora/v2`.
- `conntest` shared integration-test suite run against every `Connection`
  implementation, plus broad unit-test coverage for previously untested
  exported functions.
- `ExecRowsAffected` for all `Connection` and `Stmt` implementations.
- Generic error sentinels and cross-driver mapping: `ErrSerializationFailure`,
  `ErrDeadlock`, `ErrRaisedException`, `ErrQueryCanceled`.

### Changed

- `QueryBuilder` split into `QueryBuilder`, `UpsertQueryBuilder`, and
  `ReturningQueryBuilder`.
- `ConnConfig` renamed to `Config`; `QueryRowByPK` renamed to
  `QueryRowByPrimaryKey`.
- PostgreSQL-specific `QuoteLiteral` replaced with the ANSI
  `sqldb.QuoteStringLiteral`; `ToSnakeCase` implemented locally to drop the
  `go-types` dependency from all modules except the examples.
- UPDATE placeholder numbering reordered for positional drivers; prepared-
  statement and `BeginTx` errors consistently wrapped with `wrapKnownErrors`.

### Removed

- `go-types` dependency from the core modules (kept only in the examples).

## [v0.1.0] - 2026-03-13

Initial tagged release. Core `Connection` abstraction, context-based
connection management (`db` package), struct-to-row mapping, the PostgreSQL
(`pqconn`), MySQL/MariaDB (`mysqlconn`), SQL Server (`mssqlconn`), and SQLite
(`sqliteconn`) drivers, the `mockconn` test driver (`MockRows`,
`MockStructRows`), and the `ErrQueryCanceled` sentinel.

[v1.4.0]: https://github.com/domonda/go-sqldb/releases/tag/v1.4.0
[v1.3.1]: https://github.com/domonda/go-sqldb/releases/tag/v1.3.1
[v1.3.0]: https://github.com/domonda/go-sqldb/releases/tag/v1.3.0
[v1.2.0]: https://github.com/domonda/go-sqldb/releases/tag/v1.2.0
[v1.1.0]: https://github.com/domonda/go-sqldb/releases/tag/v1.1.0
[v1.0.13]: https://github.com/domonda/go-sqldb/releases/tag/v1.0.13
[v1.0.12]: https://github.com/domonda/go-sqldb/releases/tag/v1.0.12
[v1.0.11]: https://github.com/domonda/go-sqldb/releases/tag/v1.0.11
[v1.0.1]: https://github.com/domonda/go-sqldb/releases/tag/v1.0.1
[v1.0.0]: https://github.com/domonda/go-sqldb/releases/tag/v1.0.0
[v0.1.0]: https://github.com/domonda/go-sqldb/releases/tag/v0.1.0
