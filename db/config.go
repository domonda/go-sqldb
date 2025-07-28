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
	defaultStructReflector sqldb.StructReflector = sqldb.NewTaggedStructReflector()
	structReflectorCtxKey  int
)

func GetStructReflector(ctx context.Context) sqldb.StructReflector {
	if r, ok := ctx.Value(&structReflectorCtxKey).(sqldb.StructReflector); ok {
		return r
	}
	return defaultStructReflector
}

func SetStructReflector(reflector sqldb.StructReflector) {
	if reflector == nil {
		panic("can't set nil StructReflector")
	}
	defaultStructReflector = reflector
}

func ContextWithStructReflector(ctx context.Context, reflector sqldb.StructReflector) context.Context {
	return context.WithValue(ctx, &structReflectorCtxKey, reflector)
}

var (
	globalConn       sqldb.Connection = sqldb.NewErrConn(sqldb.ErrNoDatabaseConnection)
	globalConnCtxKey int

	queryBuilderFunc       sqldb.QueryBuilderFunc = sqldb.DefaultQueryBuilder
	queryBuilderFuncCtxKey int

	serializedTransactionCtxKey int
)

// SetConn sets the global connection that will be returned by [Conn]
// if there is no other connection in the context passed to [Conn].
func SetConn(c sqldb.Connection) {
	if c == nil {
		panic("can't set nil sqldb.Connection") // Prefer to panic early
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
	if c, _ := ctx.Value(&globalConnCtxKey).(sqldb.Connection); c != nil {
		return c
	}
	return defaultConn
}

// ContextWithConn returns a new context with the passed sqldb.Connection
// added as value so it can be retrieved again using [Conn].
// Passing a nil connection causes [Conn] to return the global connection
// configured with [SetConn].
func ContextWithConn(ctx context.Context, conn sqldb.Connection) context.Context {
	return context.WithValue(ctx, &globalConnCtxKey, conn)
}

// ContextWithGlobalConn returns a new context with the global connection
// added as value so it can be retrieved again using Conn(ctx).
func ContextWithGlobalConn(ctx context.Context) context.Context {
	return ContextWithConn(ctx, globalConn)
}

// Close the global connection that was configured with [SetConn].
func Close() error {
	return globalConn.Close()
}

func SetQueryBuilderFunc(f sqldb.QueryBuilderFunc) {
	if f == nil {
		panic("can't set nil sqldb.QueryBuilderFunc") // Prefer to panic early
	}
	queryBuilderFunc = f
}

func ContextWithQueryBuilderFunc(ctx context.Context, f sqldb.QueryBuilderFunc) context.Context {
	return context.WithValue(ctx, &queryBuilderFuncCtxKey, f)
}

func QueryBuilderFuncFromContext(ctx context.Context) sqldb.QueryBuilderFunc {
	if f, _ := ctx.Value(&queryBuilderFuncCtxKey).(sqldb.QueryBuilderFunc); f != nil {
		return f
	}
	return queryBuilderFunc
}
