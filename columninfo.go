package sqldb

type ColumnInfo struct {
	Name       string
	PrimaryKey bool
	HasDefault bool
	ReadOnly   bool
}

func (c *ColumnInfo) IsEmbeddedField() bool {
	return c.Name == ""
}
