### API Consistency

**7. `QuoteLiteral` in `format.go:182-203` is PostgreSQL-specific but lives in the root package**

The function comment even references `pq.QuoteLiteral` and the implementation uses PostgreSQL's `E'...'` escape syntax. This could mislead users of MySQL/SQLite/MSSQL drivers. Consider:
- Documenting it as PostgreSQL-specific, or
- Moving it to `pqconn`, or  
- Making it generic (the backslash-escaping with `E'` prefix is not standard SQL)

