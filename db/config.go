package db

var (
	// Number of retries used for a SerializedTransaction
	// before it fails
	SerializedTransactionRetries = 10

	serializedTransactionCtxKey byte
)
