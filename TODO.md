# go-sqldb v1.0 Release To Dos

---

## 4. TEST COVERAGE SUMMARY

| Module             | Unit      | Integration | Notes                                         |
|--------------------|-----------|-------------|-----------------------------------------------|
| Root (`go-sqldb`)  | **72.6%** | —           | Error types, MockConn/Rows/Stmt, format       |
| `db`               | **86.2%** | —           | All Result variants + transaction helpers      |
| `pqconn`           | **28.0%** | **45.0%**   | Config, Ping, Exec, Prepare, Tx, Listener     |
| `mysqlconn`        | **43.0%** | **52.0%**   | Config, Ping, Exec, Prepare, Tx, RowsAffected |
| `mssqlconn`        | **46.4%** | **59.6%**   | Config, Ping, Exec, Prepare, Tx, RowsAffected |
| `sqliteconn`       | **51.3%** | —           | `scanColumn` at 14.9%, `bindArgs` at 18.8%    |
| `oraconn`          | **22.3%** | —           | Config, Ping, Exec, Prepare, Tx, RowsAffected |
| `postgres`         | **91.3%** | —           | `InsertUnique` and `Upsert` tested             |
| `information`      | **14.9%** | —           | Table/column/PK introspection tested via pqconn|

### Remaining uncovered code:
- **`sqliteconn`**: `scanColumn` and `bindArgs` branches need more cases
- **`information/`**: Domain, View, CheckConstraints lookups untested

