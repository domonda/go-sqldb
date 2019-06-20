package sqldb

type Connection interface {
	Exec(query string, args ...interface{}) error
	QueryRow(query string, args ...interface{}) RowScanner
	QueryRows(query string, args ...interface{}) RowsScanner

	Begin() (Connection, error)
	Commit() error
	Rollback() error
	Transaction(txFunc func(tx Connection) error) error

	Close() error
}

type RowScanner interface {
	Scan(dest ...interface{}) error
	ScanStruct(dest interface{}) error
}

type RowsScanner interface {
	ScanSlice(dest interface{}) error
	ScanStructSlice(dest interface{}) error
	ForEachRow(callback func(RowScanner) error) error
}
