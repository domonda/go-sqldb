package sqldb

// ColumnInfo holds metadata about a database column
// as mapped from a Go struct field.
type ColumnInfo struct {
	Name       string
	PrimaryKey bool
	HasDefault bool
	ReadOnly   bool
}

func (c *ColumnInfo) IsEmbeddedField() bool {
	return c.Name == ""
}
