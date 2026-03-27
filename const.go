package sqldb

import (
	"context"
	"database/sql"
	"reflect"
	"time"
)

var (
	typeOfError      = reflect.TypeFor[error]()
	typeOfContext    = reflect.TypeFor[context.Context]()
	typeOfSQLScanner = reflect.TypeFor[sql.Scanner]()
	typeOfTime       = reflect.TypeFor[time.Time]()
)
