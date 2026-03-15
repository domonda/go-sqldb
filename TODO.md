# go-sqldb TODO for v1.0

## Missing Features

- [x] **Batch insert** — `InsertRowStructs` now uses multi-row `INSERT INTO ... VALUES(...),(...),...` batched by `MaxArgs()` limit

## Dead Code

- [ ] **db/foreachrow.go** — Entire file commented out
- [ ] **db/multirowscanner.go** — Entire file commented out
- [ ] **db/scanresult.go** — Entire file commented out
- [ ] **db/foreachrow_test.go** — Test body entirely commented out
- [x] **db/transaction_test.go** — `TestSerializedTransaction` and `TestTransaction` uncommented, updated to use `sqldb.MockConn`

## API Design for v1.0

### Missing Symmetry

- [ ] **`UpdateRowStruct` signature differs** from `InsertRowStruct`/`UpsertRowStruct` — `UpdateRowStruct` takes `(table string, rowStruct any)` while Insert/Upsert take `StructWithTableName` and derive the table
- [ ] **No `Delete`/`DeleteRowStruct`** — Insert, Update, Upsert exist but Delete is missing from CRUD family



## Testing

- what about testhelper.go ?

