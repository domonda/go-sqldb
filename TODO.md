# go-sqldb TODO for v1.0

## Bugs

## Missing Features

- [ ] **Batch insert** — `InsertRowStructs` processes rows one-by-one in a transaction with a prepared statement. Need optimized multi-row INSERT.

## Dead Code

- [ ] **db/foreachrow.go** — Entire file commented out
- [ ] **db/multirowscanner.go** — Entire file commented out
- [ ] **db/scanresult.go** — Entire file commented out
- [ ] **db/foreachrow_test.go** — Test body entirely commented out
- [ ] **db/transaction_test.go** — `TestSerializedTransaction` and `TestTransaction` entirely commented out

## API Design for v1.0

### Missing Symmetry

- [ ] **No `UpdateRowStruct`** matching `InsertRowStruct`/`UpsertRowStruct` — `UpdateRowStruct` takes `(table string, rowStruct any)` while Insert/Upsert take `StructWithTableName` and derive the table
- [ ] **No `Delete`/`DeleteRowStruct`** — Insert, Update, Upsert exist but Delete is missing from CRUD family

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
- [ ] **Drop schema queries** — Add to mysqlconn, mssqlconn, and sqliteconn
- [ ] **README** — Add to mysqlconn and mssqlconn
- [ ] **Package doc comment** — Add to pqconn, mysqlconn, and mssqlconn

## Testing

### Missing Tests

- [ ] `genericconn.go` / `generictx.go` — `NewGenericConn` and generic transaction types have no unit tests
- [ ] `errconn.go` — Only compile-time assertion, no behavioral tests
- [ ] `strings_test.go:37,40` — Commented-out SQL injection test cases (`admin' #` and `; DROP TABLE users--` not yet detected)
