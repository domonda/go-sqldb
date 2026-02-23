# go-sqldb TODO for v1.0

## Missing Features

- [ ] **Batch insert** — `InsertRowStructs` processes rows one-by-one in a transaction with a prepared statement. Need optimized multi-row INSERT:
  ```go
  func BatchInsert[T any](ctx context.Context, table string, items []T, batchSize int) error
  ```
- [ ] **Struct reflection cache** — Only insert query caching exists (`insert.go:58`). No broader caching of `StructReflector` results for repeated struct types (see commit `090e73d1`)

## Dead Code

- [ ] **db/foreachrow.go** — Entire file commented out
- [ ] **db/multirowscanner.go** — Entire file commented out
- [ ] **db/scanresult.go** — Entire file commented out
- [ ] **scanstruct_test.go** — Entire test commented out
- [ ] **mysqlconn/mysql.go** — `validateColumnName` and `columnNameRegex` defined but never called

## API Design for v1.0

### Missing Symmetry

- [ ] **No `UpdateRowStruct`** matching `InsertRowStruct`/`UpsertRowStruct` — `UpdateRowStruct` takes `(table string, rowStruct any)` while Insert/Upsert take `StructWithTableName` and derive the table
- [ ] **No `Delete`/`DeleteRowStruct`** — Insert, Update, Upsert exist but Delete is missing from CRUD family

### Coupling

- [ ] **pqconn imports db** — `pqconn/queryformatter.go` imports `db` for `db.StagedTypeMapper` in `NewTypeMapper()`. Driver should not depend on the high-level convenience layer. Move `StagedTypeMapper`/`TypeMapper` to root `sqldb` package
- [ ] **information imports db** — Could accept `*sqldb.ConnExt` directly instead of requiring global connection pattern
- [ ] **ConnExt bundles 3 orthogonal concerns** — Connection (I/O) + StructReflector (Go reflection) + QueryFormatter/QueryBuilder (SQL text). Functions that only need one still carry all three

### Patterns

- [ ] **Two parallel APIs (root `sqldb` vs `db`)** — Every function exists twice. `db` is a thin forwarding layer that must stay in sync
- [ ] **`QueryCallback` uses runtime reflection on `any`** — Could use generics for compile-time safety

### Driver Feature Parity

| Feature                           | pqconn | mysqlconn | mssqlconn | sqliteconn |
| --------------------------------- | ------ | --------- | --------- | ---------- |
| Custom error wrapping             | Yes    | No        | No        | Yes        |
| Identifier escaping               | Yes    | No        | Yes       | No         |
| `Connect` takes `context.Context` | Yes    | Yes       | Yes       | Yes        |
| `driver.Valuer`/`sql.Scanner`     | Yes    | Yes       | Yes       | Yes        |
| LISTEN/NOTIFY                     | Yes    | N/A       | N/A       | N/A        |
| Drop schema queries               | Yes    | No        | No        | No         |
| README                            | Yes    | No        | No        | Yes        |
| Package doc comment               | No     | No        | No        | Yes        |

- [ ] **Custom error wrapping** — Add to mysqlconn (wrap MySQL error codes) and mssqlconn (wrap MSSQL error numbers)
- [ ] **Identifier escaping** — Add to mysqlconn (backtick escaping) and sqliteconn (double-quote escaping)
- [ ] **Drop schema queries** — Add to mysqlconn, mssqlconn, and sqliteconn
- [ ] **README** — Add to mysqlconn and mssqlconn
- [ ] **Package doc comment** — Add to pqconn, mysqlconn, and mssqlconn


## Testing

### `pqconn/` Package

- [ ] **pqconn/test** — `TestDatabase` is a stub with no assertions
