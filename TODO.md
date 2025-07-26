# TODO - Unfinished Work in go-sqldb

## Major TODOs from README.md

- [ ] Test all pkg db functions
- [ ] Rethink Config
- [ ] pkg information completion
- [ ] Test pqconn with dockerized Postgres
- [ ] Cache struct types (see commit 090e73d1d9db8534d2950dd7236d7ebe192cd512)
- [ ] Std SQL driver for mocks
- [ ] Smooth out listener for Postgres
- [ ] SQLite integration https://github.com/zombiezen/go-sqlite

## Code-Level TODOs and Missing Implementations

### Performance Optimizations

- [ ] **db/insert.go:203** - `InsertRowStructs` missing optimized batch insert (currently processes one-by-one in transaction)
- [ ] **db/insert.go:76** - Commented code for RETURNING clause needs error wrapping
- [ ] **pqconn/arrays.go:128** - Array element scanning needs type conversion improvement for different element types

### Function Implementations

- [ ] **db/insert.go:152** - Complete commented out `InsertStructStmt` function with TODO placeholder
- [ ] **mssqlconn/queryformatter.go:11** - Allow spaces and other characters with backtick escaping

### API Design Questions

- [ ] **db/scanresult.go:3** - Consider moving ScanResult to RowScanner interface
- [ ] **db/multirowscanner.go:15,97** - Resolve API design questions about single vs multi-column scanning
- [ ] **db/reflectstruct.go:168** - Clean up Connection implementation detail

## Missing Patterns

### 1. Batch Operations
- Current `InsertRowStructs` processes items individually in a transaction
- Need optimized batch INSERT statements that combine multiple structs
- Consider maxArgs parameter limitations

### 2. RETURNING Clause Support
- Commented implementation exists in insert.go:76
- Need proper error wrapping for query execution
- Should integrate with existing query building patterns

### 3. Error Handling Standardization
- Some query error wrapping is incomplete
- Need consistent pattern across all database operations

### 4. Type Conversion Enhancement
- Array scanning needs improvement for different element types
- String-to-type conversion challenges in pqconn/arrays.go:128

## Key Areas for Completion

### High Priority
1. **Performance Optimization**: Implement batch insert operations
2. **Testing**: Comprehensive test coverage for db package functions
3. **Configuration**: Rethink and improve Config structure

### Medium Priority
4. **Database Support**: Complete SQLite integration
5. **Error Handling**: Standardize query error wrapping patterns
6. **API Consistency**: Resolve design questions in multirowscanner

### Low Priority
7. **Code Organization**: Move ScanResult and clean up implementation details
8. **Features**: Enhanced array type support and RETURNING clause functionality

## Implementation Notes

- The UpsertStruct function (db/upsert.go:14) is marked "TODO" but appears fully implemented
- Mock connection patterns are well established in _mockconn/ package
- Go workspace structure supports multiple database drivers effectively
- Context-based connection management pattern is consistently implemented