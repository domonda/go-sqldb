package db

import "github.com/domonda/go-sqldb"

// TableName is an alias for [sqldb.TableName].
// Embed it in structs to specify the table name via a struct tag.
//
// Example:
//
//	type MyTable struct {
//	    db.TableName `db:"my_table"`
//	}
type TableName = sqldb.TableName
