# go-sqldb v1.0 Release To Dos

---

## 3. TAG-RELEASE & CI SCRIPT ISSUES

### 3a. ~~`tag-release.sh`~~ FIXED
Added `set -e`, fixed shell quoting on lines 4-5, and updated stale `cmd/sqldb-dump` example to match current `MODULE_PATHS`.

### 3b. ~~CI workflow `.github/workflows/go.yml`~~ FIXED
Added MSSQL health check, removed stale `slim-conn` branch trigger, updated `go-version` to `1.24.6`,
and fixed `grep -v /cmd/` to `grep -v /examples/` for gosec filter.

### 3c. ~~`test-workspace.sh:13`~~ FIXED
Changed `grep -v /cmd/` to `grep -v /examples/` to correctly filter example modules from gosec.

---

## 4. TEST COVERAGE SUMMARY

| Module             | Coverage  | Notes                                                |
|--------------------|-----------|------------------------------------------------------|
| Root (`go-sqldb`)  | **65.6%** | `generictx.go`, `stmt.go`, `errors.go` types at 0%  |
| `db`               | **68.1%** | Most Result variants at 0%                           |
| `sqliteconn`       | **51.3%** | `scanColumn` at 14.9%, `bindArgs` at 18.8%           |
| `mssqlconn`        | **35.1%** | All connection/transaction/error methods at 0%       |
| `mysqlconn`        | **24.1%** | All connection/transaction/error methods at 0%       |
| `pqconn`           | **23.4%** | Only formatter/builder tested                        |
| `information`      | **14.9%** | All table/primarykey functions at 0%                 |
| `oraconn`          | **0.0%**  | No unit tests at all                                 |
| `postgres`         | **0.0%**  | No unit tests for new QueryBuilder                   |

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
4. [x] ~~Fix `tag-release.sh` quoting and stale example~~
5. [x] ~~Add MSSQL health check to CI~~
6. [x] ~~Fix resource leaks in mssqlconn/oraconn `DropAllTables`~~

### Should fix:
7. [ ] Add unit tests for `postgres/querybuilder.go`
8. [ ] Add unit tests for `oraconn` queryformatter
9. [x] ~~Add `set -e` to `tag-release.sh`~~
10. [x] ~~Fix mysqlconn `Upsert` MariaDB compatibility~~
11. [x] ~~Fix mysqlconn comment typo~~
12. [ ] Fix `db/` package error constructors to use `errs.New`/`errs.Errorf`

### Nice to have:
13. [ ] Improve pqconn listener thread safety with `sync.Once`
14. [ ] Add sqliteconn stmt.go context checks
15. [x] ~~Add mysqlconn Prepare/Begin error wrapping~~
16. [ ] Improve overall test coverage (especially connectors at 0-35%)