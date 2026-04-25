/*
Package information contains structs and functions to query the
information_schema views defined by ISO/IEC 9075-11.

The struct shapes mirror the standard view definitions, and the queries
emit placeholders and identifiers via the connection's
[sqldb.QueryFormatter] so they adapt to each driver's syntax.
information_schema itself is only fully implemented by PostgreSQL; on
MySQL/MariaDB and SQL Server many ISO extension columns scan as
empty/nil, and SQLite and Oracle do not expose information_schema at
all. Per-type and per-function godoc records the relevant caveats, and
the package README has the full compatibility matrix.

See https://en.wikipedia.org/wiki/Information_schema
and https://www.postgresql.org/docs/current/information-schema.html
*/
package information
