# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Version Control

- **NEVER** create a git commit without permission
- **NEVER** git push

## Architecture Overview

### Core Components

- **Connection Interface**: Central abstraction for database connections and transactions (`connection.go`)
- **DB Package**: Context-based connection management pattern (`db/` directory)
- **Database Drivers**: Separate modules for PostgreSQL (`pqconn/`), MySQL (`mysqlconn/`), SQL Server (`mssqlconn/`), SQLite (`sqliteconn/`), and Oracle (`oraconn/`)
- **Query Building**: Flexible query construction with struct mapping
- **Transaction Management**: Nested transactions with savepoint support
- **Schema Introspection**: Two layers. The high-level `sqldb.Information` interface (`information.go`) is embedded into `Connection` and provides vendor-portable methods (`Schemas`, `Tables`, `Views`, `Columns`, `PrimaryKey`, `ForeignKeys`, `Routines`, plus `*Exists` variants). Each driver implements it against its native catalog (`pg_catalog`, `information_schema`, `sys.*`, `sqlite_schema` + `PRAGMA`, or Oracle's `ALL_*` views). The lower-level `information/` subpackage queries ISO `information_schema` views directly with typed structs — use it when you need raw catalog rows on PG/MySQL/MariaDB/MSSQL.

### Key Design Patterns

1. **Context-Based Connection Storage**: Store connections/transactions in context for seamless function composition
2. **Struct-to-SQL Mapping**: Automatic mapping between Go structs and database rows using reflection
3. **Transaction Callbacks**: Execute transactions in callback functions that can be nested
4. **Flexible Query Interface**: Support for both raw SQL and struct-based operations

### Context Key Convention

Use unexported empty struct types as context keys, with no separate variable.
The type name is the key name, and `keyName{}` is used directly as the value:

```go
type myCtxKey struct{}

func ContextWithMyValue(ctx context.Context, val string) context.Context {
    return context.WithValue(ctx, myCtxKey{}, val)
}

func MyValueFromContext(ctx context.Context) string {
    val, _ := ctx.Value(myCtxKey{}).(string)
    return val
}
```

## Go (golang) Rules

### General Rules
- Use `any` instead of `interface{}`
- Write SQL string literals with backticks and prefix with `/*sql*/`. In case of a multi-line SQL string, begin the string literal in the next line.
- Write HTML string literals with backticks and prefix with `/*html*/`. In case of a multi-line HTML string, begin the string literal in the next line.
- Instead of `for i := 0; i < count; i++ {` use `for i := range count {`

### File Naming Rules
- Use `_test.go` suffix for source files that contain tests
- **NEVER** use underscores `_` or dashes `-` in general Go source file names

### SQL Rules
- Write SQL string literals with backticks and prefix with `/*sql*/`
  and start such SQL string literals in a new line if not the first argument
- Check for the position of `/*sql*/` after running gofmt, move to next line if placement was moved the previous argument
- Use numbered parameters (`$1`, `$2`) for PostgreSQL driver
- Avoid SQL code duplication, find existing functions first
- Database lookup functions use `Get`, `Is`, or `Has` prefixes
- Use nullable Go types instead of pointers for NULL columns

### Struct Field Mapping
- Default tag: `db:"column_name"`
- Primary key: `db:"id,primarykey"`
- Ignore field: `db:"-"`

### Building Rules
- Remove build artefacts after testing with `go clean ./...`

### Testing
- **Acceptance tests for workspace**: `./test-workspace.sh` 
- Use `t.Context()` instead of `context.Background()` in tests
- Use `github.com/stretchr/testify` for tests
- Import `github.com/stretchr/testify/require` for required conditions
- Import `github.com/stretchr/testify/assert` for assertions where the test should continue even if the condition is not met
- Always use `go clean` after `go build` to not leave binaries behind
- PostgreSQL integration tests use dockerized PostgreSQL 17 on port 5433 (see `pqconn/test/docker-compose.yml`)
- Run `./pqconn/test/reset-postgres-data.sh` after changing the PostgreSQL version
- MariaDB integration tests use dockerized MariaDB 11.7 on port 3307 (see `mysqlconn/test/docker-compose.yml`)
- Run `./mysqlconn/test/reset-mariadb-data.sh` after changing the MariaDB version
- SQL Server integration tests use dockerized SQL Server 2022 on port 1434 (see `mssqlconn/test/docker-compose.yml`)
- Run `./mssqlconn/test/reset-mssql-data.sh` after changing the SQL Server version
- Do **not** use `os.Exit(m.Run())` in `TestMain` — just call `m.Run()` directly

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
// Insert with struct (struct must implement sqldb.StructWithTableName)
err := db.InsertRowStruct(ctx, &structInstance)

// Upsert (uses primary key fields)
err := db.UpsertRowStruct(ctx, &structInstance)

// Query into value/struct
user, err := db.QueryRowAs[User](ctx,
    /*sql*/ `SELECT * FROM users WHERE id = $1`,
    id, // $1
)
```

### Schema Introspection
```go
// Vendor-portable catalog access via the sqldb.Information interface
// embedded into every Connection. The db package wraps these methods
// as top-level db.* functions following the same ctx-first style.
// Methods that don't apply on a vendor (e.g. Routines on SQLite)
// return errors.ErrUnsupported.
exists, err := db.TableExists(ctx, "public.user")
cols, err := db.Columns(ctx, "public.user")
pk, err := db.PrimaryKey(ctx, "public.user") // constraint order, not declaration order
fks, err := db.ForeignKeys(ctx, "public.order")
```

Per-driver caveats live in each driver's README (`pqconn/`, `mysqlconn/`,
`mssqlconn/`, `sqliteconn/`, `oraconn/`). The lower-level
`information/` subpackage offers typed `information_schema` row structs
for PG/MySQL/MariaDB/MSSQL when raw catalog rows are needed.
