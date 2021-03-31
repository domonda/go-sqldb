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

type (
	connKeyType         struct{} // unique type for this package
	serializedTxKeyType struct{} // unique type for this package
)

var (
	conn    = sqldb.ConnectionWithError(context.Background(), errors.New("database connection not initialized"))
	connKey connKeyType

	serializedTxKey serializedTxKeyType
)
