# go-sqldb TODO for v1.0

## Bugs

### Moderate

- [ ] **sqliteconn/transaction.go:87,103,111** — Nested savepoints all use hardcoded name `nested_tx`; multi-level nesting releases/rolls back wrong savepoint
- [ ] **sqliteconn/connection.go** — `ctx context.Context` accepted but never used in `Exec`, `Query`, `Prepare`, `Begin`; context cancellation silently ignored
- [ ] **cmd/sqldb-dump/sqldb-dump.go** — Won't compile: uses `sqldb.Config` (should be `ConnConfig`) and `pqconn.New` (should be `Connect`)

## Dead Code

- [ ] **_mockconn/** — Entire package mostly commented out, won't compile, underscore prefix hides from `go build`. Decide: restore or delete
- [ ] **db/foreachrow.go** — Entire file commented out
- [ ] **db/multirowscanner.go** — Entire file commented out
- [ ] **db/scanresult.go** — Entire file commented out
- [ ] **scanstruct_test.go** — Entire test commented out
- [ ] **pqconn/arrays.go:61-131** — Large block of commented-out code
- [ ] **db/insert.go:21-35,47-61** — Commented-out functions
- [ ] **mysqlconn/mysql.go** — `validateColumnName` and `columnNameRegex` defined but never called

## API Design for v1.0

### Naming Inconsistencies

- [ ] **"RowStruct" vs "Struct"** — Insert uses `InsertRowStruct`, `InsertRowStructs`; Update/Upsert use `UpdateStruct`, `UpsertStruct`. Pick one convention
- [ ] **"Read" vs "Query"** — `QueryRow`, `QueryValue`, `QueryRowsAsSlice` use "Query" prefix; `ReadRowStructWithTableName` uses "Read". Same abstraction level, different prefix
- [ ] **`ReadRowStructWithTableName`** — 29 chars. The `StructWithTableName` constraint already enforces table name. Could be `ReadRow[S]`
- [ ] **Stmt close parameter names** — `closeStmt`, `closeFunc`, `done` across different Stmt-returning functions

### Missing Symmetry

- [ ] **No `UpdateRowStruct`** matching `InsertRowStruct`/`UpsertStruct` — `UpdateStruct` takes `(table string, rowStruct any)` while Insert/Upsert take `StructWithTableName` and derive the table
- [ ] **No `Delete`/`DeleteRowStruct`** — Insert, Update, Upsert exist but Delete is missing from CRUD family

### Coupling

- [ ] **pqconn imports db** — `pqconn/queryformatter.go` imports `db` for `db.StagedTypeMapper` in `NewTypeMapper()`. Driver should not depend on the high-level convenience layer. Move `StagedTypeMapper`/`TypeMapper` to root `sqldb` package
- [ ] **information imports db** — Could accept `*sqldb.ConnExt` directly instead of requiring global connection pattern
- [ ] **ConnExt bundles 3 orthogonal concerns** — Connection (I/O) + StructReflector (Go reflection) + QueryFormatter/QueryBuilder (SQL text). Functions that only need one still carry all three

### Patterns

- [ ] **Stmt functions return `(workFunc, closeFunc, error)` triple** — Inconsistent naming, easy to misuse. Consider returning a struct or `Stmt` value
- [ ] **Two parallel APIs (root `sqldb` vs `db`)** — Every function exists twice. `db` is a thin forwarding layer that must stay in sync
- [ ] **`QueryCallback` uses runtime reflection on `any`** — Could use generics for compile-time safety
- [ ] **Global mutable insert cache** (insert.go:58-61) — Grows without bound, invisible to callers, key doesn't account for `QueryOption`
- [ ] **`anyvalue.go` receiver named `any`** — Shadows the builtin

### Driver Feature Parity

| Feature                           | pqconn | mysqlconn | mssqlconn | sqliteconn |
| --------------------------------- | ------ | --------- | --------- | ---------- |
| Custom error wrapping             | Yes    | No        | No        | Yes        |
| Identifier escaping               | Yes    | No        | No        | No         |
| `Connect` takes `context.Context` | Yes    | Yes       | Yes       | **No**     |
| `driver.Valuer`/`sql.Scanner`     | Yes    | Yes       | Yes       | **No**     |
| LISTEN/NOTIFY                     | Yes    | N/A       | N/A       | N/A        |
| Drop schema queries               | Yes    | No        | No        | No         |
| README                            | Yes    | No        | No        | Yes        |
| Package doc comment               | No     | No        | No        | Yes        |

- [ ] **mssqlconn/queryformatter.go:11** — TODO says "backticks" but MSSQL uses `[brackets]`
- [ ] **mssqlconn** — No identifier escaping; reserved words as table/column names will fail
- [ ] **mssqlconn** — `FormatTableName` doesn't support schema-qualified names (`dbo.table`)
- [ ] **sqliteconn** — `Connect` missing `context.Context` parameter (all other drivers have it)
- [ ] **sqliteconn** — No `driver.Valuer`/`sql.Scanner` support in argument binding or result scanning
- [ ] **sqliteconn/README.md** — Examples show `Connect(ctx, config)` but actual signature is `Connect(config)`

## Other

- [x] **db/upsert.go:9** — `UpsertStruct` godoc is `// UpsertStruct TODO` — Fixed: added proper godoc
- [ ] **information/primarykeys.go:149,153** — HTTP handler returns actual `err.Error()` as 500 response

## Testing

### Critical — Core Logic With Zero Tests

- [ ] **querybuilder.go** — `StdQueryBuilder` has zero tests for all 6 SQL-generating methods: `QueryRowWithPK`, `Insert`, `InsertUnique`, `Upsert`, `Update`, `UpdateColumns`
- [ ] **reflectstruct.go** — All 6 exported `Reflect*` functions untested: `PrimaryKeyColumnsOfStruct`, `ReflectStructColumnsAndValues`, `ReflectStructColumnsFieldIndicesAndValues`, `ReflectStructValues`, `ReflectStructColumns`, `ReflectStructColumnPointers`. Note: `scanstruct_test.go` exists but is entirely commented out
- [ ] **scan.go** — `ScanDriverValue` untested; wide type-switching function covering bool, float64, int64, string, []byte, time.Time
- [ ] **row.go** — `Row.Scan` (struct-detection logic), `Row.ScanValues`, `Row.ScanStrings` all untested

### High Priority — Important Public API

- [ ] **queryformatter.go** — `StdQueryFormatter` methods untested: `FormatTableName`, `FormatColumnName`, `FormatPlaceholder`, `FormatStringLiteral`
- [ ] **pqconn/errors.go** — All 16 error-predicate functions untested (`IsUniqueViolation`, `IsForeignKeyViolation`, `IsSerializationFailure`, etc.). Can be tested with synthetic `*pq.Error` values
- [ ] **connconfig.go** — `ConnConfig.Validate()` untested; `ParseConnConfig` has only one happy-path test case
- [ ] **queryoption.go** — `IgnoreColumns`, `OnlyColumns`, `IgnoreStructFields`, `OnlyStructFields`, `QueryOptionsIgnoreColumn` all untested
- [ ] **format.go** — `QuoteLiteral` untested (backslash `E'...'` path); `FormatQuery` only tests numbered placeholders, not uniform (`?`)
- [ ] **nullable.go** — `Nullable[T].Scan`, `Nullable[T].Value`, `IsNullable` untested
- [ ] **debug.go** — `TxOptionsString` (4-branch function), `FprintTable` (unicode-aware padding) untested

### Root `sqldb` Package — Untested Functions

- [ ] `Insert`, `InsertUnique`, `InsertUniqueRowStruct`, `InsertRowStructs`
- [ ] `Update`, `UpdateStruct`
- [ ] `UpsertStruct`, `UpsertStructStmt`, `UpsertStructs`
- [ ] `QueryRow`, `QueryValue`, `QueryValueOr`, `QueryValueStmt`, `ReadRowStructWithTableName`, `QueryRowAsMap`, `QueryRowsAsSlice`
- [ ] `Exec`, `ExecStmt`
- [ ] `Transaction`, `IsolatedTransaction`
- [ ] `ConnExt.WithConnection`, `TransactionExt`, `TransactionResult`
- [ ] `AnyValue.Scan`, `AnyValue.Value`

### `db/` Package — Untested Functions

- [ ] `Update`, `UpdateStruct`
- [ ] `UpsertStruct`, `UpsertStructs`
- [ ] `InsertUnique`, `InsertUniqueRowStruct`, `InsertRowStructStmt`, `InsertRowStructs`
- [ ] `QueryRow`, `QueryValueStmt`, `ReadRowStructWithTableName`, `ReadRowStructWithTableNameOr`, `QueryRowAsMap`, `QueryRowsAsSlice`
- [ ] `Exec`, `ExecStmt`
- [ ] `ValidateWithinTransaction`, `ValidateNotWithinTransaction`, `IsTransaction`
- [ ] `IsolatedTransaction`, `OptionalTransaction`, `TransactionReadOnly`, `TransactionSavepoint`, `TransactionResult`
- [ ] `ContextWithoutTransactions`, `IsContextWithoutTransactions`
- [ ] `ListenOnChannel`, `UnlistenChannel`, `IsListeningOnChannel`
- [ ] `ReplaceErrNoRows`, `IsOtherThanErrNoRows`
- [ ] `SetConn`, `Conn`, `Close`

### `pqconn/` Package

- [ ] **Error predicates** — All 16 `Is*` functions need unit tests with synthetic `*pq.Error` values
- [ ] **pqconn/test** — `TestDatabase` is a stub with no assertions
- [ ] `QueryFormatter.FormatPlaceholder` — Panic on negative index untested
- [ ] `QueryFormatter.FormatStringLiteral`, `NewTypeMapper`
- [ ] `Connect` — Driver validation path untested

### `mysqlconn/` Package — Zero Tests

- [ ] Entire package has no test files: `Connect`, `MustConnect`, `ConnectExt`, `NewConfig`
- [ ] `validateColumnName` — Internal function with MySQL-specific regex, never tested

### `mssqlconn/` Package — Zero Tests

- [ ] Entire package has no test files: `Connect`, `MustConnect`, `ConnectExt`
- [ ] `QueryFormatter.FormatTableName`, `FormatColumnName`, `FormatPlaceholder` (`@p1` style) untested

### `sqliteconn/` Package — Gaps

- [ ] `IsDatabaseLocked` error predicate untested
- [ ] Nested savepoint transactions not tested (related to hardcoded `nested_tx` name bug)

### `information/` Package — Gaps

- [ ] `GetTableRowsWithPrimaryKey` — Queries multiple tables by PK; `sql.ErrNoRows` skip path untested

## Missing Features

- [ ] **Batch insert** — `InsertRowStructs` processes rows one-by-one in a transaction with a prepared statement. Need optimized multi-row INSERT:
  ```go
  func BatchInsert[T any](ctx context.Context, table string, items []T, batchSize int) error
  ```
- [ ] **Struct reflection cache** — Only insert query caching exists (`insert.go:58`). No broader caching of `StructReflector` results for repeated struct types (see commit `090e73d1`)

## Missing Godoc (Exported Symbols) — Done

- [x] `ConnExt`, `NewConnExt`, `TransactionExt`, `TransactionResult`
- [x] `TransactionState`, `TransactionState.Active()`
- [x] `Stmt` interface, `NewStmt`, `NewUnpreparedStmt`
- [x] `TableNameForStruct`, `ColumnInfo`
- [x] `StdQueryBuilder` and all its methods
- [x] `Nullable[T]`, `IsNullable`
- [ ] `AnyValue`, `StringScannable`
- [x] `db.ContextWithoutTransactions`, `db.IsContextWithoutTransactions`
- [x] `db.QueryValueStmt`, `db.InsertRowStructStmt`
- [x] `db.UpsertStruct`, `db.UpsertStructStmt`, `db.UpsertStructs`
- [x] Most `information/` structs (`Schema`, `View`, `Column`, `Domain`, `CheckConstraints`, `PrimaryKeyColumn`)
