package db

type serializedTransactionCtxKey struct{}

// Number of retries used for a SerializedTransaction
// before it fails.
var SerializedTransactionRetries = 10
