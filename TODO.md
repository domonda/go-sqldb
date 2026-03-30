# go-sqldb v1.0 Release To Dos

  - pqconn and sqliteconn have custom stmt types that already call wrapKnownErrors — no changes needed
  - mssqlconn, oraconn, mysqlconn use sqldb.NewStmt which returns a wrappedStmt that does NOT wrap errors — this is the bug
  - genericconn / generictx also use NewStmt but have no driver-specific error wrapping