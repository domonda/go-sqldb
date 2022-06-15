package db

import (
	"context"
	"errors"

	"github.com/domonda/go-sqldb"
)

var (
	// Number of retries used for a SerializedTransaction
	// before it fails
	SerializedTransactionRetries = 10
)

var (
	globalConn = sqldb.ConnectionWithError(
		context.Background(),
		errors.New("database connection not initialized"),
	)
	globalConnCtxKey            int
	serializedTransactionCtxKey int
)
