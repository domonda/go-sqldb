# go-sqldb v1.0 Release To Dos

---

## 4. TEST COVERAGE SUMMARY

| Module             | Coverage  | Notes                                                |
|--------------------|-----------|------------------------------------------------------|
| Root (`go-sqldb`)  | **66.6%** | `generictx.go` delegation + error paths tested       |
| `db`               | **68.1%** | Most Result variants at 0%                           |
| `sqliteconn`       | **51.3%** | `scanColumn` at 14.9%, `bindArgs` at 18.8%           |
| `mssqlconn`        | **46.4%** | `wrapKnownErrors` + helpers tested                   |
| `mysqlconn`        | **43.0%** | `wrapKnownErrors` + helpers tested                   |
| `pqconn`           | **28.0%** | `wrapKnownErrors`, `Is*` helpers, formatter tested   |
| `oraconn`          | **22.3%** | Errors, `Is*` helpers, queryformatter tested         |
| `postgres`         | **91.3%** | `InsertUnique` and `Upsert` tested                   |
| `information`      | **14.9%** | All table/primarykey functions at 0%                 |

### Remaining uncovered code:
- **Connector connection/transaction methods**: All connectors still have 0% coverage for `Exec`, `Query`, `Begin`, `Commit` etc. (require live database)
- **`information/`**: All table/primarykey introspection functions at 0% (require live database)
- **`db/` Result variants**: Most `Result*` convenience functions at 0%

---

## 5. ADDITIONAL FINDINGS FROM DEEP REVIEW

### 5c. `postgres/querybuilder.go:66` — inconsistent SQL spacing
`ON CONFLICT(` missing space before `(`, while line 41 in `InsertUnique` uses `ON CONFLICT (%s)` with a space. Both valid but inconsistent.

### 5e. Foreign key cleanup not guaranteed in `DropAllTables`
**`sqliteconn/dropall.go:39-48`**, **`mysqlconn/dropall.go:39-48`** — If `SET FOREIGN_KEY_CHECKS = 1` / `PRAGMA foreign_keys = ON` fails after an error, the database is left with foreign keys disabled. The error from re-enabling is silently discarded.
