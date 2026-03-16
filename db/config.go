package db

type serializedTransactionCtxKey struct{}

// SerializedTransactionRetries is the number of retries
// for a SerializedTransaction before it fails.
var SerializedTransactionRetries = 10
