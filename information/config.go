package information

import (
	"reflect"
	"strings"

	"github.com/domonda/go-sqldb"
)

// structReflector is the package-private reflector used for scanning
// information_schema rows.
//
// It wraps the standard tagged reflector with case-insensitive column
// matching: MariaDB and MySQL return information_schema column names
// in uppercase (TABLE_NAME, TABLE_SCHEMA, ...) regardless of how they
// are written in the SELECT clause, while the struct tags in this
// package use the ISO-standard lowercase form. SQL Server's behaviour
// depends on the database collation; case-insensitive matching makes
// it work without configuration. PostgreSQL already returns lowercase
// names, so the wrapper is a no-op for it.
var structReflector sqldb.StructReflector = caseInsensitiveColumnReflector{
	StructReflector: sqldb.NewTaggedStructReflector(),
}

type caseInsensitiveColumnReflector struct {
	sqldb.StructReflector
}

func (r caseInsensitiveColumnReflector) ScanableStructFieldsForColumns(structVal reflect.Value, columns []string) ([]any, error) {
	lowered := make([]string, len(columns))
	for i, col := range columns {
		lowered[i] = strings.ToLower(col)
	}
	return r.StructReflector.ScanableStructFieldsForColumns(structVal, lowered)
}
