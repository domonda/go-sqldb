package sqldb

// ColumnInfo holds metadata about a database column
// as mapped from a Go struct field.
type ColumnInfo struct {
	Name       string
	PrimaryKey bool
	HasDefault bool
	ReadOnly   bool
}

// IsEmbeddedField reports whether this ColumnInfo represents an embedded struct field,
// indicated by an empty Name.
func (c *ColumnInfo) IsEmbeddedField() bool {
	return c.Name == ""
}
