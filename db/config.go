package db

import (
	"github.com/domonda/go-sqldb"
)

var (
	// Number of retries used for a SerializedTransaction
	// before it fails
	SerializedTransactionRetries = 10

	// TODO set default value
	DefaultStructReflectror sqldb.StructReflector
)

var (
	globalConn sqldb.Connection = sqldb.NewErrConn(
		sqldb.ErrNoDatabaseConnection,
	)
	globalConnCtxKey            int
	serializedTransactionCtxKey int
)
