package information

// CheckConstraints maps a row from information_schema.check_constraints.
//
// Vendor support:
//   - PostgreSQL: all fields populated.
//   - MySQL 8.0.16+, MariaDB 10.2+: all fields populated.
//   - SQL Server: all fields populated.
//   - SQLite, Oracle: information_schema is not implemented.
type CheckConstraints struct {
	ConstraintCatalog String `db:"constraint_catalog"`
	ConstraintSchema  String `db:"constraint_schema"`
	ConstraintName    String `db:"constraint_name"`
	CheckClause       String `db:"check_clause"`
}
