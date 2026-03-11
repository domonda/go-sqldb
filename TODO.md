# go-sqldb TODO for v1.0

## Missing Features

- [ ] **Batch insert** — `InsertRowStructs` processes rows one-by-one in a transaction with a prepared statement. Need optimized multi-row INSERT.

## Dead Code

- [ ] **db/foreachrow.go** — Entire file commented out
- [ ] **db/multirowscanner.go** — Entire file commented out
- [ ] **db/scanresult.go** — Entire file commented out
- [ ] **db/foreachrow_test.go** — Test body entirely commented out
- [ ] **db/transaction_test.go** — `TestSerializedTransaction` and `TestTransaction` entirely commented out

## API Design for v1.0

### Drivers

- Connect returns ConnExt?

### Missing Symmetry

- [ ] **No `UpdateRowStruct`** matching `InsertRowStruct`/`UpsertRowStruct` — `UpdateRowStruct` takes `(table string, rowStruct any)` while Insert/Upsert take `StructWithTableName` and derive the table
- [ ] **No `Delete`/`DeleteRowStruct`** — Insert, Update, Upsert exist but Delete is missing from CRUD family

### Driver Feature Parity

| Feature                           | pqconn | mysqlconn | mssqlconn | sqliteconn |
| --------------------------------- | ------ | --------- | --------- | ---------- |
| Custom error wrapping             | Yes    | Yes       | Yes       | Yes        |
| Identifier escaping               | Yes    | Yes       | Yes       | Yes        |
| `Connect` takes `context.Context` | Yes    | Yes       | Yes       | Yes        |
| `driver.Valuer`/`sql.Scanner`     | Yes    | Yes       | Yes       | Yes        |
| LISTEN/NOTIFY                     | Yes    | N/A       | N/A       | N/A        |
| Drop schema queries               | Yes    | Yes       | Yes       | Yes        |
| README                            | Yes    | Yes       | Yes       | Yes        |
| Package doc comment               | Yes    | Yes       | Yes       | Yes        |


## Testing

- what about testhelper.go ?

### Missing Tests

- [ ] `genericconn.go` / `generictx.go` — `NewGenericConn` and generic transaction types have no unit tests
