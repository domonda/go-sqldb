# go-sqldb TODO for v1.0

## Bugs

## Missing Features

- [x] Query formatter tests with name escaping
- [ ] **Batch insert** — `InsertRowStructs` processes rows one-by-one in a transaction with a prepared statement. Need optimized multi-row INSERT:
  ```go
  func BatchInsert[T any](ctx context.Context, table string, items []T, batchSize int) error
  ```

## Dead Code

- [ ] **db/foreachrow.go** — Entire file commented out
- [ ] **db/multirowscanner.go** — Entire file commented out
- [ ] **db/scanresult.go** — Entire file commented out
- [ ] **db/foreachrow_test.go** — Test body entirely commented out
- [ ] **scanstruct_test.go** — Entire test commented out
- [ ] **db/transaction_test.go** — `TestSerializedTransaction` and `TestTransaction` entirely commented out
- [x] **mysqlconn/mysql.go** — `validateColumnName` and `columnNameRegex` defined but never called in production code (file deleted, logic moved into QueryFormatter)
- [ ] **querybuilder.go** — `DefaultQueryBuilder` declared but never referenced
- [ ] **queryformatter.go** — `DefaultQueryFormatter` declared but never referenced

## Oversights

- [ ] `transaction.go:16` — Comment references nonexistent `Connection.TransactionNo()`, should be `Connection.Transaction().ID`
- [ ] `transaction.go:56` — Comment typo: "paniced" should be "panicked"
- [ ] `db/listen.go:19,30,39` — `ListenerConnection` type assertion on `Conn(ctx)` always succeeds because `connExtImpl` satisfies the interface; the `ok == false` branch is dead code
- [ ] `connext.go:29-35` — `NewConnExt` and `NewConnExtWithConn` don't validate nil arguments
- [ ] `query.go:21,31,82,122,133` — Missing space after comma: `conn,refl` should be `conn, refl`
- [ ] `information/primarykeys.go:56,91` — Odd casing in SQL: `ordinal_positiON` (works but looks wrong)
- [ ] `information/primarykeys.go:163,290,322` — Odd casing in HTML/CSS: `buttON`, `captiON` (works but looks wrong)
- [ ] `errconn.go:10` — `var _` assertion checks `Connection` but comment says `ListenerConnection`
- [ ] `db/conn.go:11,19,28` — `globalConn` read/written without synchronization (race if `SetConn` concurrent with `Conn`)

## API Design for v1.0

### Missing Symmetry

- [ ] **No `UpdateRowStruct`** matching `InsertRowStruct`/`UpsertRowStruct` — `UpdateRowStruct` takes `(table string, rowStruct any)` while Insert/Upsert take `StructWithTableName` and derive the table
- [ ] **No `Delete`/`DeleteRowStruct`** — Insert, Update, Upsert exist but Delete is missing from CRUD family

### Coupling

- [ ] **pqconn imports db** — `pqconn/queryformatter.go` imports `db` for `db.StagedTypeMapper` in `NewTypeMapper()`. Driver should not depend on the high-level convenience layer. Move `StagedTypeMapper`/`TypeMapper` to root `sqldb` package
- [ ] **information imports db** — `information/table.go`, `information/column.go`, `information/primarykeys.go` import `db`. Could accept `sqldb.ConnExt` directly instead of requiring global connection pattern

### Patterns

- [ ] **Two parallel APIs (root `sqldb` vs `db`)** — Every function exists twice. `db` is a thin forwarding layer that must stay in sync
- [ ] **`QueryCallback` uses runtime reflection on `any`** — Could use generics for compile-time safety

### Driver Feature Parity

| Feature                           | pqconn | mysqlconn | mssqlconn | sqliteconn |
| --------------------------------- | ------ | --------- | --------- | ---------- |
| Custom error wrapping             | Yes    | No        | No        | Yes        |
| Identifier escaping               | Yes    | Yes       | Yes       | No         |
| `Connect` takes `context.Context` | Yes    | Yes       | Yes       | Yes        |
| `driver.Valuer`/`sql.Scanner`     | Yes    | Yes       | Yes       | Yes        |
| LISTEN/NOTIFY                     | Yes    | N/A       | N/A       | N/A        |
| Drop schema queries               | Yes    | No        | No        | No         |
| README                            | Yes    | No        | No        | Yes        |
| Package doc comment               | No     | No        | No        | Yes        |

- [ ] **Custom error wrapping** — Add to mysqlconn (wrap MySQL error codes) and mssqlconn (wrap MSSQL error numbers)
- [x] **Identifier escaping** — Added to mysqlconn (backtick escaping); sqliteconn (double-quote escaping) still missing
- [ ] **Drop schema queries** — Add to mysqlconn, mssqlconn, and sqliteconn
- [ ] **README** — Add to mysqlconn and mssqlconn
- [ ] **Package doc comment** — Add to pqconn, mysqlconn, and mssqlconn

## Testing

### Missing Tests

- [ ] `scanstruct.go` — `scanStruct` has no active tests (test file entirely commented out)
- [ ] `genericconn.go` / `generictx.go` — `NewGenericConn` and generic transaction types have no unit tests
- [ ] `query.go` — `QueryRowAsMap`, `QueryRowsAsStrings` have no tests
- [ ] `query.go` — `QueryCallback` has no tests in root package (db package has coverage)
- [ ] `update.go` — `UpdateReturningRow` and `UpdateReturningRows` have no tests
- [ ] `db/statement.go` — `Prepare` and `stmtWithErrWrapping` have no tests
- [ ] `nonconnfortest.go` — `NonConnForTest` has no direct tests
- [ ] `errconn.go` — Only compile-time assertion, no behavioral tests
- [ ] `pqconn/test` — `TestDatabase` is a stub with no assertions
- [ ] `nullable_test.go:151` — `TODO more tests` for `TestIsNullOrZero`
- [ ] `strings_test.go:37,40` — Commented-out SQL injection test cases
