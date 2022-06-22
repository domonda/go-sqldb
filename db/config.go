package db

import (
	"errors"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/reflection"
)

var (
	// Number of retries used for a SerializedTransaction
	// before it fails
	SerializedTransactionRetries = 10

	// DefaultStructFieldMapping provides the default StructFieldTagNaming
	// using "db" as NameTag and IgnoreStructField as UntaggedNameFunc.
	// Implements StructFieldMapper.
	DefaultStructFieldMapping = reflection.NewTaggedStructFieldMapping()
)

var (
	globalConn = sqldb.ConnectionWithError(
		errors.New("database connection not initialized"),
	)
	globalConnCtxKey            int
	serializedTransactionCtxKey int
)
