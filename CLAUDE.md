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

## Code Conventions

### SQL Queries
- Write SQL string literals with backticks and prefix with `/*sql*/` comment
- Use numbered parameters (`$1`, `$2`) for PostgreSQL driver

### Error Handling  
- Use `github.com/domonda/go-errs` instead of standard `errors` package
- Use `errs.New()` instead of `errors.New()`
- Use `errs.Errorf()` instead of `fmt.Errorf()`

### UUID Types
- Use `github.com/domonda/go-types/uu` package for UUIDs
- Single UUID: `uu.ID`
- UUID slice: `uu.IDSlice` (not `[]uu.ID`)
- Zero values: `uu.IDNil` for `uu.ID`, `uu.IDNull` for `uu.NullableID`, `nil` for `uu.IDSlice`

### General Go Rules
- Use `any` instead of `interface{}`
- In HTTP handlers, use `http.Request.Context()` for context
- Never return actual error strings as HTTP 500 responses - use abstract descriptions

### Struct Field Mapping
- Default tag: `db:"column_name"`
- Primary key: `db:"id,pk"`
- Ignore field: `db:"-"`

## Testing
- Mock connections available in `_mockconn/` package
- PostgreSQL integration tests use Docker (see `pqconn/test/`)
- Use `db.ContextWithNonConnectionForTest()` for testing without real database
- Helper functions in `db/testhelper.go`

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
user, err := db.QueryRowValue[User](ctx, /*sql*/ `SELECT * FROM users WHERE id = $1`, id)
```