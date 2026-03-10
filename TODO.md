# go-sqldb TODO for v1.0

## Bugs

## Missing Features

- [ ] **Batch insert** — `InsertRowStructs` processes rows one-by-one in a transaction with a prepared statement. Need optimized multi-row INSERT.

## Dead Code

- [ ] **db/foreachrow.go** — Entire file commented out
- [ ] **db/multirowscanner.go** — Entire file commented out
- [ ] **db/scanresult.go** — Entire file commented out
- [ ] **db/foreachrow_test.go** — Test body entirely commented out
- [x] **scanstruct_test.go** — Entire test commented out (now has active `TestScanStruct`)
- [ ] **db/transaction_test.go** — `TestSerializedTransaction` and `TestTransaction` entirely commented out
- [x] **mysqlconn/mysql.go** — `validateColumnName` and `columnNameRegex` defined but never called in production code (file deleted, logic moved into QueryFormatter)
- [ ] **querybuilder.go** — `DefaultQueryBuilder` declared but never referenced
- [ ] **queryformatter.go** — `DefaultQueryFormatter` declared but never referenced

## API Design for v1.0

### Missing Symmetry

- [ ] **No `UpdateRowStruct`** matching `InsertRowStruct`/`UpsertRowStruct` — `UpdateRowStruct` takes `(table string, rowStruct any)` while Insert/Upsert take `StructWithTableName` and derive the table
- [ ] **No `Delete`/`DeleteRowStruct`** — Insert, Update, Upsert exist but Delete is missing from CRUD family

### Patterns

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

- [x] `scanstruct.go` — `scanStruct` has no active tests (now has `TestScanStruct`)
- [ ] `genericconn.go` / `generictx.go` — `NewGenericConn` and generic transaction types have no unit tests
- [x] `query.go` — `QueryRowAsMap`, `QueryRowsAsStrings` have no tests (now covered in `query_test.go`)
- [x] `query.go` — `QueryCallback` has no tests in root package (now has `TestQueryCallback` in both root and db package)
- [x] `update.go` — `UpdateReturningRow` and `UpdateReturningRows` have no tests (now covered in `update_test.go`)
- [x] `db/statement.go` — `Prepare` and `stmtWithErrWrapping` have no tests (now has `TestPrepare_Success` and `TestPrepare_Error`)
- [x] `nonconnfortest.go` — `NonConnForTest` has no direct tests (now has extensive tests in `nonconnfortest_test.go`)
- [ ] `errconn.go` — Only compile-time assertion, no behavioral tests
- [x] `pqconn/test` — `TestDatabase` is a stub with no assertions (now has subtests with assertions)
- [x] `nullable_test.go:151` — `TODO more tests` for `TestIsNullOrZero` (TODO removed, tests added)
- [ ] `strings_test.go:37,40` — Commented-out SQL injection test cases (`admin' #` and `; DROP TABLE users--` not yet detected)
