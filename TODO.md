### TODO

- Add Oracle support using github.com/sijms/go-ora/v2 for new package oraconn, use pqconn as template

Support Matrix

 ┌────────────┬───────────��─────────────┬──────────────────────────────────────────┬─────────────────���─────┐
 │   Vendor   │   UpsertQueryBuilder    │              InsertUnique                │ ReturningQueryBuilder │
 ├────────���───┼───────────────���─────────┼──────────────────────────────────────────┼───────────────────────┤
 │ pqconn     │ ON CONFLICT DO UPDATE   │ ON CONFLICT DO NOTHING                  │ RETURNING clause      │
 ├────────────┼───���─────────────────────┼──────────────────────────────────────────┼─────��─────────────────┤
 │ sqliteconn │ ON CONFLICT DO UPDATE   │ ON CONFLICT DO NOTHING                  │ RETURNING clause      │
 ├────────────┼─────────────────────────┼──────────────────���───────────────────────┼────────���──────────────┤
 │ mysqlconn  │ ON DUPLICATE KEY UPDATE │ ON DUPLICATE KEY UPDATE col = col        │ Not supported         │
 ├────────────┼─────────────���───────────┼───────────���──────────────────────────────┼───────────────────────┤
 │ mssqlconn  │ MERGE ... WHEN MATCHED  │ MERGE ... WHEN NOT MATCHED THEN INSERT  │ Not supported         │
 └───────���────┴───────────────���─────────┴──────────────────────��───────────────────┴───────────────────────┘


**7. `QuoteLiteral` in `format.go:182-203` is PostgreSQL-specific but lives in the root package**

The function comment even references `pq.QuoteLiteral` and the implementation uses PostgreSQL's `E'...'` escape syntax. This could mislead users of MySQL/SQLite/MSSQL drivers. Consider:
- Documenting it as PostgreSQL-specific, or
- Moving it to `pqconn`, or
- Making it generic (the backslash-escaping with `E'` prefix is not standard SQL)
