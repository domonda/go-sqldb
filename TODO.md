# go-sqldb v1.0 Release To Dos

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
- **`oraconn/`**: Entire package has zero unit tests (only integration tests in `oraconn/test/`)
- **Connector error wrapping** (`wrapKnownErrors`): sqliteconn tests it indirectly via connection tests; pqconn tests `Is*` helpers but not the `wrapKnownErrors` mapping to `sqldb.Err*` types; mysqlconn, mssqlconn, and oraconn have no error tests at all
- **`generictx.go`**: Every method at 0% — generic `database/sql` transaction wrapper (not currently used by any connector, each has its own implementation)

---

## 5. ADDITIONAL FINDINGS FROM DEEP REVIEW

### ~~5a. pqconn listener race conditions~~ ✓ FIXED
Fixed via stop channel + `sync.Once` in `pqconn/listener.go`. The `listen()` goroutine now exits on `<-l.stop`, `close()` is idempotent via `stopOnce.Do`, and `unlistenChannel` uses `isStopped()` instead of the racy `l.conn == nil` check.

### 5c. `postgres/querybuilder.go:66` — inconsistent SQL spacing
`ON CONFLICT(` missing space before `(`, while line 41 in `InsertUnique` uses `ON CONFLICT (%s)` with a space. Both valid but inconsistent.

### 5d. `db/insert.go:51` — redundant `Conn(ctx)` call
Already assigned `conn := Conn(ctx)` on line 48, then shadows it with another `conn := Conn(ctx)` on line 51.

### 5e. Foreign key cleanup not guaranteed in `DropAllTables`
**`sqliteconn/dropall.go:39-48`**, **`mysqlconn/dropall.go:39-48`** — If `SET FOREIGN_KEY_CHECKS = 1` / `PRAGMA foreign_keys = ON` fails after an error, the database is left with foreign keys disabled. The error from re-enabling is silently discarded.

---

## 6. PRIORITY MATRIX

### Should fix:
7. [ ] Add unit tests for `postgres/querybuilder.go`
8. [ ] Add unit tests for `oraconn` queryformatter

### Nice to have:
13. [x] Improve pqconn listener thread safety with `sync.Once`
14. [ ] Add sqliteconn stmt.go context checks
16. [ ] Improve overall test coverage (especially connectors at 0-35%)