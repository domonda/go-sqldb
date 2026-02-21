# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

- **Run tests for all modules**: `./test-workspace.sh` - This runs tests across all Go workspace modules (main, mssqlconn, mysqlconn, pqconn)
- **Run tests for specific module**: `go test ./...` in the module directory
- **Build all modules**: `go build ./...`
- **Get dependencies**: `go mod tidy` (run in each module directory as needed)

## Go Workspace Structure

This repository uses Go workspaces with multiple modules:
- Main module: `github.com/domonda/go-sqldb` (root)
- Database drivers: `./mssqlconn`, `./mysqlconn`, `./pqconn`
- Command tools: `./cmd/sqldb-dump`
- Examples: `./examples/user_demo`

## Architecture Overview

### Core Components

- **Connection Interface**: Central abstraction for database connections and transactions (`connection.go`)
- **DB Package**: Context-based connection management pattern (`db/` directory)
- **Database Drivers**: Separate modules for PostgreSQL (`pqconn/`), MySQL (`mysqlconn/`), and SQL Server (`mssqlconn/`)
- **Query Building**: Flexible query construction with struct mapping
- **Transaction Management**: Nested transactions with savepoint support

### Key Design Patterns

1. **Context-Based Connection Storage**: Store connections/transactions in context for seamless function composition
2. **Struct-to-SQL Mapping**: Automatic mapping between Go structs and database rows using reflection
3. **Transaction Callbacks**: Execute transactions in callback functions that can be nested
4. **Flexible Query Interface**: Support for both raw SQL and struct-based operations

### Important Packages

- `sqldb` (root): Core interfaces and types
- `db/`: Context-based connection management and transaction utilities
- `information/`: Database schema introspection
- `_mockconn/`: Mock implementations for testing

## Go (golang) Rules

### General Rules
- Use `any` instead of `interface{}`
- Write SQL string literals with backticks and prefix with `/*sql*/`. In case of a multi-line SQL string, begin the string literal in the next line.
- Write HTML string literals with backticks and prefix with `/*html*/`. In case of a multi-line HTML string, begin the string literal in the next line.
- Instead of `for i := 0; i < count; i++ {` use `for i := range count {`

### File Naming Rules
- Use `_test.go` suffix for source files that contain tests
- **NEVER** use underscores `_` or dashes `-` in general Go source file names

### Error Handling
- Use `errs.New` from `github.com/domonda/go-errs` instead of `errors.New`
- Use `errs.Errorf` from `github.com/domonda/go-errs` instead of `fmt.Errorf`
- Use `errs.Errorf` instead of `errs.Wrap` or `errors.Wrap`
- **NEVER** use the exact functions `errs.Wrap` or `errors.Wrap`
- Every exported function returning an error should have error result named `err`
- Add error wrapping defer call: `defer errs.WrapWithFuncParams(&err, arg0, arg1, ...)`
- Ensure empty line between error wrapping and function body

### SQL Rules
- Write SQL string literals with backticks and prefix with `/*sql*/`
- Use numbered parameters (`$1`, `$2`) for PostgreSQL driver
- Avoid SQL code duplication, find existing functions first
- Database lookup functions use `Get`, `Is`, or `Has` prefixes
- Use nullable Go types instead of pointers for NULL columns

### UUID Rules
- Use `github.com/domonda/go-types/uu` package for UUIDs instead of `github.com/google/uuid`
- Single UUID: `uu.ID`
- UUID slice: `uu.IDSlice` (not `[]uu.ID`)
- Zero values: `uu.IDNil` for `uu.ID`, `uu.IDNull` for `uu.NullableID`, `nil` for `uu.IDSlice`
- Use `ID` suffix instead of `UUID` in naming

### Data Validation Rules
- Use `Validate()` method for data validation, `Valid()` for boolean results
- Use `IsNull()` and `IsNotNull()` methods for nullable value checks
- Prefer types from `github.com/domonda/go-types` when appropriate

### Struct Field Mapping
- Default tag: `db:"column_name"`
- Primary key: `db:"id,pk"`
- Ignore field: `db:"-"`

### Building Rules
- Remove build artefacts after testing

### Testing
- Use `t.Context()` instead of `context.Background()` in tests
- Use `github.com/stretchr/testify` for tests
- Import `github.com/stretchr/testify/require` for required conditions
- Import `github.com/stretchr/testify/assert` for assertions where the test should continue even if the condition is not met
- Always use `go clean` after `go build` to not leave binaries behind
- Mock connections available in `_mockconn/` package
- PostgreSQL integration tests use Docker (see `pqconn/test/`)
- Use `db.ContextWithNonConnectionForTest()` for testing without real database
- Helper functions in `db/testhelper.go`

### Commit Message Guidelines
Follow conventional commits format: `<type>(<scope>): <subject>`

Types: `build`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `style`, `test`

- Use imperative, present tense: "change" not "changed"
- Don't capitalize first letter
- No period at the end if single line
- Keep subject line under 100 characters
- Qualify names in Go code with the package name

**Important**: Don't create commits without asking first!

### Markdown Rules
- Pad all markdown tables

## Common Usage Patterns

### Transaction Management
```go
// Simple transaction
err := db.Transaction(ctx, func(ctx context.Context) error {
    // All db.Conn(ctx) calls use the transaction
    return db.Conn(ctx).Exec(/*sql*/ `INSERT ...`)
})

// Serialized transaction (for high concurrency scenarios)
err := db.SerializedTransaction(ctx, func(ctx context.Context) error { ... })

// Transaction with savepoints (nested transactions)
err := db.TransactionSavepoint(ctx, func(ctx context.Context) error { ... })
```

### Struct Operations
```go
// Insert with struct
err := db.InsertStruct(ctx, "table_name", &structInstance)

// Upsert (uses primary key fields)
err := db.UpsertStruct(ctx, "table_name", &structInstance)

// Query into struct
user, err := db.QueryRowValue[User](ctx, 
    /*sql*/ `SELECT * FROM users WHERE id = $1`,
    id, // $1
)
```
