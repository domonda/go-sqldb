# go-sqldb TODO for v1.0

## API Design for v1.0

### Missing Symmetry

- [ ] **`UpdateRowStruct` signature differs** from `InsertRowStruct`/`UpsertRowStruct` — `UpdateRowStruct` takes `(table string, rowStruct any)` while Insert/Upsert take `StructWithTableName` and derive the table

