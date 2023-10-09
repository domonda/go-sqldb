package sqldb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"reflect"
)

var (
	typeOfByte         = reflect.TypeOf(byte(0))
	typeOfError        = reflect.TypeOf((*error)(nil)).Elem()
	typeOfByteSlice    = reflect.TypeOf((*[]byte)(nil)).Elem()
	typeOfContext      = reflect.TypeOf((*context.Context)(nil)).Elem()
	typeOfSQLScanner   = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
	typeOfDriverValuer = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
	typeOfTableName    = reflect.TypeOf(TableName{})
	// typeOfTime         = reflect.TypeOf(time.Time{})
)
