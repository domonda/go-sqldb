package sqldb

type RowValues interface {
	Columns() []string
	RowValues() ([]any, error)
}

type RowPointers interface {
	Columns() []string
	RowPointers() ([]any, error)
}
