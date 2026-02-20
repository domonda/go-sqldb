# go-sqldb TODO for v1.0

## Bugs

### Critical

- [x] **mockconn.go:291** — `Rollback()` called `MockCommit` instead of `MockRollback`
- [x] **mockconn.go:118** — `Stats()` guarded on `MockPing` instead of `MockStats`
- [x] **information/types.go:29** — `YesNo.Scan` set `true` for `"NO"` (now sets `false`)
- [ ] **information/view.go** — `View` struct is copy-paste of `Schema`; fields map to `information_schema.schemata`, not `information_schema.views`
- [ ] **insert.go:200-225** — `InsertRowStructs` uses outer connection `c`, not the transaction `tx`; prepared statement runs outside the transaction. Same bug in `upsert.go:98` (`UpsertStructs`)
- [ ] **scan.go:67** — `ScanDriverValue` calls `reflect.ValueOf(destPtr).SetBool(src)` on pointer instead of dereferenced value; will panic at runtime. Should be `dest.SetBool(src)`
- [ ] **debug.go:26** — `conn.Query(...).Scan(&t)` called without `Next()` first, rows never closed

### Moderate

- [ ] **insert.go:58-61** — Query cache ignores `QueryOption` parameters; first call's options determine cached query for all subsequent calls with different options
- [ ] **update.go:22** — Error wrapping uses `args` (WHERE args only) instead of `vals` (all query args)
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

- [ ] **db/upsert.go:9** — `UpsertStruct` godoc is `// UpsertStruct TODO`
- [ ] **information/primarykeys.go:149,153** — HTTP handler returns actual `err.Error()` as 500 response

## Testing

- [ ] **Test all `db/` package functions** — Only 7.4% coverage; 6 test files exist but many gaps
- [ ] **pqconn integration tests** — `pqconn/test/` has docker-compose setup but `TestDatabase` is a stub with no assertions
- [x] **Fix `TestQueryCallback_InvalidVariadic`** — Fixed: updated assertion from "varidic" to "variadic"

## Missing Features

- [ ] **Batch insert** — `InsertRowStructs` processes rows one-by-one in a transaction with a prepared statement. Need optimized multi-row INSERT:
  ```go
  func BatchInsert[T any](ctx context.Context, table string, items []T, batchSize int) error
  ```
- [ ] **Struct reflection cache** — Only insert query caching exists (`insert.go:58`). No broader caching of `StructReflector` results for repeated struct types (see commit `090e73d1`)

## Missing Godoc (Exported Symbols)

- [ ] `ConnExt`, `NewConnExt`, `TransactionExt`, `TransactionResult`
- [ ] `TransactionState`, `TransactionState.Active()`
- [ ] `Stmt` interface, `NewStmt`, `NewUnpreparedStmt`
- [ ] `TableNameForStruct`, `ColumnInfo`
- [ ] `StdQueryBuilder` and all its methods
- [ ] `Nullable[T]`, `IsNullable`
- [ ] `AnyValue`, `StringScannable`
- [ ] `db.ContextWithoutTransactions`, `db.IsContextWithoutTransactions`
- [ ] `db.QueryValueStmt`, `db.InsertRowStructStmt`
- [ ] `db.UpsertStruct`, `db.UpsertStructStmt`, `db.UpsertStructs`
- [ ] Most `information/` structs (`Schema`, `View`, `Column`, `Domain`, `CheckConstraints`, `PrimaryKeyColumn`)
