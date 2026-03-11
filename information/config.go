package information

import "github.com/domonda/go-sqldb"

var (
	queryBuilder    sqldb.StdQueryBuilder
	structReflector = sqldb.NewTaggedStructReflector()
)
