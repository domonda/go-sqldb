package sqldb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"reflect"
	"time"
)

var (
	typeOfError        = reflect.TypeFor[error]()
	typeOfContext      = reflect.TypeFor[context.Context]()
	typeOfSQLScanner   = reflect.TypeFor[sql.Scanner]()
	typeOfDriverValuer = reflect.TypeFor[driver.Valuer]()
	typeOfTime         = reflect.TypeFor[time.Time]()
	typeOfByte         = reflect.TypeFor[byte]()
	typeOfByteSlice    = reflect.TypeFor[[]byte]()
)
