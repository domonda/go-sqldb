package sqldb

// RowsScanner scans the values from multiple rows.
type RowsScanner interface {
	// ScanSlice scans one value per row into one slice element of dest.
	// dest must be a pointer to a slice with a row value compatible element type.
	// In case of zero rows, dest will be set to nil and no error will be returned.
	// In case of an error, dest will not be modified.
	// It is an error to query more than one column.
	ScanSlice(dest interface{}) error

	// ScanStructSlice scans every row into the struct fields of dest slice elements.
	// dest must be a pointer to a slice of structs or struct pointers.
	// In case of zero rows, dest will be set to nil and no error will be returned.
	// In case of an error, dest will not be modified.
	// Every mapped struct field must have a corresponding column in the query results.
	ScanStructSlice(dest interface{}) error

	// ScanStrings scans the values of all rows as strings.
	// Byte slices will be interpreted as strings,
	// nil (SQL NULL) will be converted to an empty string,
	// all other types are converted with fmt.Sprint(src).
	ScanStrings() (rows [][]string, err error)

	// ForEachRow will call the passed callback with a RowScanner for every row.
	// In case of zero rows, no error will be returned.
	ForEachRow(callback func(RowScanner) error) error

	// ForEachRowScan will call the passed callback with scanned values or a struct for every row.
	// If the callback function has a single struct or struct pointer argument,
	// then RowScanner.ScanStruct will be used per row,
	// else RowScanner.Scan will be used for all arguments of the callback.
	// If the function has a context.Context as first argument,
	// then the context of the query call will be passed on.
	// In case of zero rows, no error will be returned.
	ForEachRowScan(callback interface{}) error
}
