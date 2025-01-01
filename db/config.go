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
	typeOfError        = reflect.TypeFor[error]()
	typeOfContext      = reflect.TypeFor[context.Context]()
	typeOfSQLScanner   = reflect.TypeFor[sql.Scanner]()
	typeOfDriverValuer = reflect.TypeFor[driver.Valuer]()
	typeOfTime         = reflect.TypeFor[time.Time]()
	typeOfByte         = reflect.TypeFor[byte]()
	typeOfByteSlice    = reflect.TypeFor[[]byte]()
)

var (
	// Number of retries used for a SerializedTransaction
	// before it fails
	SerializedTransactionRetries = 10
)

var (
	defaultStructReflector StructReflector = NewTaggedStructReflector()
	structReflectorCtxKey  int
)

func GetStructReflector(ctx context.Context) StructReflector {
	if r := ctx.Value(&structReflectorCtxKey).(StructReflector); r != nil {
		return r
	}
	return defaultStructReflector
}

func SetStructReflector(reflector StructReflector) {
	if reflector == nil {
		panic("can't set nil StructReflector")
	}
	defaultStructReflector = reflector
}

func ContextWithStructReflector(ctx context.Context, reflector StructReflector) context.Context {
	return context.WithValue(ctx, &structReflectorCtxKey, reflector)
}

var (
	globalConn sqldb.Connection = sqldb.NewErrConn(
		sqldb.ErrNoDatabaseConnection,
	)
	globalConnCtxKey int

	serializedTransactionCtxKey int
)

// SetConn sets the global connection returned by Conn
// if there is no other connection in the context passed to Conn.
func SetConn(c sqldb.Connection) {
	if c == nil {
		panic("can't set nil sqldb.Connection")
	}
	globalConn = c
}

// Conn returns a non nil sqldb.Connection from ctx
// or the global connection set with SetConn.
func Conn(ctx context.Context) sqldb.Connection {
	return ConnOr(ctx, globalConn)
}

// ConnOr returns a non nil sqldb.Connection from ctx
// or the passed defaultConn.
func ConnOr(ctx context.Context, defaultConn sqldb.Connection) sqldb.Connection {
	c := ctx.Value(&globalConnCtxKey).(sqldb.Connection)
	if c == nil {
		return defaultConn
	}
	return c
}

// ContextWithConn returns a new context with the passed sqldb.Connection
// added as value so it can be retrieved again using Conn(ctx).
// Passing a nil connection causes Conn(ctx)
// to return the global connection set with SetConn.
func ContextWithConn(ctx context.Context, conn sqldb.Connection) context.Context {
	return context.WithValue(ctx, &globalConnCtxKey, conn)
}
