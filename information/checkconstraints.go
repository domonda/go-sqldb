package information

// CheckConstraints maps a row from information_schema.check_constraints.
type CheckConstraints struct {
	ConstraintCatalog String `db:"constraint_catalog"`
	ConstraintSchema  String `db:"constraint_schema"`
	ConstraintName    String `db:"constraint_name"`
	CheckClause       String `db:"check_clause"`
}
