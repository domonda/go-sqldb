package information

type CheckConstraints struct {
	ConstraintCatalog String `db:"constraint_catalog"`
	ConstraintSchema  String `db:"constraint_schema"`
	ConstraintName    String `db:"constraint_name"`
	CheckClause       String `db:"check_clause"`
}
