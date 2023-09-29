package impl

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"reflect"
	"time"
)

var (
	typeOfError        = reflect.TypeOf((*error)(nil)).Elem()
	typeOfByte         = reflect.TypeOf(byte(0))
	typeOfByteSlice    = reflect.TypeOf((*[]byte)(nil)).Elem()
	typeOfContext      = reflect.TypeOf((*context.Context)(nil)).Elem()
	typeOfTime         = reflect.TypeOf(time.Time{})
	typeOfSQLScanner   = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
	typeOfDriverValuer = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
)
