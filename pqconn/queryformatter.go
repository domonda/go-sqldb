package pqconn

import (
	"strconv"

	"github.com/domonda/go-sqldb"
)

type QueryFormatter struct {
	sqldb.StdQueryFormatter
}

func (f QueryFormatter) FormatPlaceholder(paramIndex int) string {
	if paramIndex < 0 {
		panic("paramIndex must be greater or equal zero")
	}
	return "$" + strconv.Itoa(paramIndex+1)
}
