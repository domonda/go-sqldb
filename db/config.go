package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"reflect"
	"time"

	"github.com/domonda/go-sqldb"
)

var (
	// Number of retries used for a SerializedTransaction
	// before it fails
	SerializedTransactionRetries = 10

	// TODO set default value
	DefaultStructReflectror StructReflector
)

var (
	globalConn sqldb.Connection = sqldb.NewErrConn(
		sqldb.ErrNoDatabaseConnection,
	)
	globalConnCtxKey            int
	serializedTransactionCtxKey int
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
