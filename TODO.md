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

- [ ] **`UpdateRowStruct` signature differs** from `InsertRowStruct`/`UpsertRowStruct` — `UpdateRowStruct` takes `(table string, rowStruct any)` while Insert/Upsert take `StructWithTableName` and derive the table
- [ ] **No `Delete`/`DeleteRowStruct`** — Insert, Update, Upsert exist but Delete is missing from CRUD family



## Testing

- what about testhelper.go ?

### Missing Tests

- [ ] `genericconn.go` / `generictx.go` — `NewGenericConn` and generic transaction types have no unit tests
