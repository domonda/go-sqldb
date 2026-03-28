package oraconn

import (
	"database/sql"
	"strings"
)

// lowercaseRows wraps *sql.Rows so that Columns returns
// lowercased column names. Oracle returns uppercase
// identifiers by default for unquoted names, but Go
// struct tags and the sqldb convention use lowercase.
//
// Oracle SQL itself is case-insensitive for unquoted identifiers,
// so this only affects the Go-side column name matching
// used by the sqldb struct reflector.
type lowercaseRows struct {
	*sql.Rows
}

func (r lowercaseRows) Columns() ([]string, error) {
	cols, err := r.Rows.Columns()
	if err != nil {
		return nil, err
	}
	for i, col := range cols {
		cols[i] = strings.ToLower(col)
	}
	return cols, nil
}
