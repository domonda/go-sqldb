# go-sqldb v1.0 Release To Dos

## 1. BUGS & TEST FAILURES

### 1a. ~~pqconn `FormatStringLiteral` test failure~~ FIXED
Updated test expectations to match `pq.QuoteLiteral()` E-string syntax (` E'path\\to'`).

### 1b. ~~oraconn `EscapeIdentifier` — wrong comment~~ FIXED
Fixed comment to say "non-lowercase/non-underscore" and removed incorrect claim about lowercase being quoted.

### 1c. ~~`querybuilder.go:244` — wrong error message~~ FIXED
Changed error message from `"DeleteColumns requires at least one column"` to `"Delete requires at least one column"`.

### 1d. `connconfig.go:76-77` — port parse error silently discarded
`ParseConnConfig` uses `_ , _ = strconv.Atoi(parsed.Port())`. If the port is non-numeric, it silently becomes 0.

### 1e. Resource leak in `DropAllTables`/`DropAllTypes` (mssqlconn, oraconn)
**`mssqlconn/dropall.go:34-35,64-65,100-101`** and **`oraconn/dropall.go:32-33,57-58,88-89`** — when `rows.Err()` returns an error, rows are not closed before returning.

### 1f. mysqlconn `VALUES()` syntax is deprecated
**`mysqlconn/querybuilder.go:44,78`** — Uses `VALUES(col)` in `ON DUPLICATE KEY UPDATE`, deprecated since MySQL 8.0.20 and will be removed in MySQL 9.0. Should switch to row alias syntax (`AS new ... new.col`).

---

## 2. CONSISTENCY / MISSING ERROR WRAPPING

### 2a. db package uses `fmt.Errorf`/`errors.New` instead of `errs.Errorf`/`errs.New`
- `db/insert.go:31,52,113`
- `db/upsert.go:21,45,67`
- `db/transaction.go:198,214,319`

### 2b. mysqlconn missing error wrapping in Prepare/Begin
**`mysqlconn/connection.go:134-140`**, **`mysqlconn/transaction.go:67-73,86-95`** — `Prepare()` and `Begin()` don't call `wrapKnownErrors()`, unlike Exec/Query/ExecRowsAffected.

### 2c. ~~mysqlconn `queryformatter.go:137` — comment typo~~ FIXED
Entire `FormatStringLiteral` rewritten to match `go-sql-driver/mysql` `escapeStringBackslash`.

---

## 3. TAG-RELEASE & CI SCRIPT ISSUES

### 3a. `tag-release.sh`
1. **Lines 4-5**: Missing shell quoting — `$(dirname -- "$0")` and `cd $SCRIPT_DIR` need double-quotes
2. **Line 40**: Stale example `cmd/sqldb-dump/v0.99.1` — path doesn't exist
3. **No `set -e`** — script continues after errors
4. **No pre-flight checks** — doesn't verify clean tree, correct branch, or tests passing

### 3b. CI workflow `.github/workflows/go.yml`
1. **Lines 44-50**: MSSQL service has **no health check** options — tests may start before MSSQL is ready
2. **Line 6**: `slim-conn` branch trigger may be stale
3. **Line 58**: `go-version: '1.24'` doesn't match `go.work`'s `1.24.6`

### 3c. `test-workspace.sh:13`
`grep -v /cmd/` filter doesn't match any module path — examples live under `examples/`, not `cmd/`

---

## 4. TEST COVERAGE SUMMARY

| Module | Coverage | Notes |
|--------|----------|-------|
| Root (`go-sqldb`) | **65.6%** | `generictx.go`, `stmt.go`, `errors.go` types at 0% |
| `db` | **68.1%** | Most Result variants at 0% |
| `sqliteconn` | **51.3%** | `scanColumn` at 14.9%, `bindArgs` at 18.8% |
| `mssqlconn` | **35.1%** | All connection/transaction/error methods at 0% |
| `mysqlconn` | **24.1%** | All connection/transaction/error methods at 0% |
| `pqconn` | **23.4%** | Only formatter/builder tested |
| `information` | **14.9%** | All table/primarykey functions at 0% |
| `oraconn` | **0.0%** | No unit tests at all |
| `postgres` | **0.0%** | No unit tests for new QueryBuilder |

### Critical uncovered code:
- **`postgres/querybuilder.go`**: New package with 0% coverage — `InsertUnique` and `Upsert` untested
- **`oraconn/`**: Entire package has zero unit tests
- **All connector error wrapping** (`wrapKnownErrors`): Only sqliteconn has unit tests for error mapping
- **`generictx.go`**: Every method at 0% — this wraps `database/sql` for pqconn/mysqlconn/mssqlconn

---

## 5. ADDITIONAL FINDINGS FROM DEEP REVIEW

### 5a. pqconn listener race conditions
**`pqconn/listener.go:61,77-98`** — The `listen()` goroutine and `close()` can race. Both can call `l.close()` simultaneously. Consider `sync.Once` for the close operation. Also, external `Connection.Close()` may not properly signal the listen goroutine to exit.

### 5b. sqliteconn `stmt.go` missing context checks
**`sqliteconn/stmt.go:20,43,66`** — `Exec()`, `ExecRowsAffected()`, `Query()` accept `ctx` but never check `ctx.Err()`, unlike all other methods in connection.go/transaction.go.

### 5c. `postgres/querybuilder.go:66` — inconsistent SQL spacing
`ON CONFLICT(` missing space before `(`, while line 41 in `InsertUnique` uses `ON CONFLICT (%s)` with a space. Both valid but inconsistent.

### 5d. `db/insert.go:51` — redundant `Conn(ctx)` call
Already assigned `conn := Conn(ctx)` on line 48, then shadows it with another `conn := Conn(ctx)` on line 51.

### 5e. Foreign key cleanup not guaranteed in `DropAllTables`
**`sqliteconn/dropall.go:39-48`**, **`mysqlconn/dropall.go:39-48`** — If `SET FOREIGN_KEY_CHECKS = 1` / `PRAGMA foreign_keys = ON` fails after an error, the database is left with foreign keys disabled. The error from re-enabling is silently discarded.

---

## 6. PRIORITY MATRIX

### Must fix before v1.0:
1. [x] ~~Fix pqconn `FormatStringLiteral` (test is failing)~~
2. [x] ~~Fix oraconn `EscapeIdentifier` comment~~
3. [x] ~~Fix `querybuilder.go:244` error message typo~~
4. Fix `tag-release.sh` quoting and stale example
5. Add MSSQL health check to CI
6. Fix resource leaks in mssqlconn/oraconn `DropAllTables`

### Should fix:
7. Add unit tests for `postgres/querybuilder.go`
8. Add unit tests for `oraconn` queryformatter
9. Add `set -e` to `tag-release.sh`
10. Address mysqlconn `VALUES()` deprecation (or document as known limitation)
11. [x] ~~Fix mysqlconn comment typo~~
12. Fix `db/` package error constructors to use `errs.New`/`errs.Errorf`

### Nice to have:
13. Improve pqconn listener thread safety with `sync.Once`
14. Add sqliteconn stmt.go context checks
15. Add mysqlconn Prepare/Begin error wrapping
16. Improve overall test coverage (especially connectors at 0-35%)