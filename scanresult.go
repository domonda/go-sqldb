package sqldb

// ScanValues returns the values of a row exactly how they are
// passed from the database driver to an sql.Scanner.
// Byte slices will be copied.
func ScanValues(src Row) ([]any, error) {
	cols, err := src.Columns()
	if err != nil {
		return nil, err
	}
	var (
		anys = make([]AnyValue, len(cols))
		vals = make([]any, len(cols))
	)
	for i := range vals {
		vals[i] = &anys[i]
	}
	err = src.Scan(vals...)
	if err != nil {
		return nil, err
	}
	for i := range vals {
		vals[i] = anys[i].Val
	}
	return vals, nil
}

// ScanStrings scans the values of a row as strings.
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with fmt.Sprint.
func ScanStrings(src Row) ([]string, error) {
	cols, err := src.Columns()
	if err != nil {
		return nil, err
	}
	var (
		strs = make([]string, len(cols))
		args = make([]any, len(cols))
	)
	for i := range args {
		args[i] = (*StringScannable)(&strs[i])
	}
	err = src.Scan(args...)
	if err != nil {
		return nil, err
	}
	return strs, nil
}
