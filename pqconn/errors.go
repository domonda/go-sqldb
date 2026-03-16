package pqconn

import (
	"context"
	"errors"
	"slices"

	"github.com/domonda/go-sqldb"
	"github.com/lib/pq"
)

func wrapKnownErrors(err error) error {
	if err == nil {
		return nil
	}
	var e *pq.Error
	if errors.As(err, &e) {
		switch e.Code {
		case "22004":
			return errors.Join(sqldb.ErrNullValueNotAllowed, err)
		case "23000":
			return errors.Join(sqldb.ErrIntegrityConstraintViolation{Constraint: e.Constraint}, err)
		case "23001":
			return errors.Join(sqldb.ErrRestrictViolation{Constraint: e.Constraint}, err)
		case "23502":
			return errors.Join(sqldb.ErrNotNullViolation{Constraint: e.Constraint}, err)
		case "23503":
			return errors.Join(sqldb.ErrForeignKeyViolation{Constraint: e.Constraint}, err)
		case "23505":
			return errors.Join(sqldb.ErrUniqueViolation{Constraint: e.Constraint}, err)
		case "23514":
			return errors.Join(sqldb.ErrCheckViolation{Constraint: e.Constraint}, err)
		case "40P01":
			return errors.Join(sqldb.ErrDeadlock, err)
		case "57014":
			return errors.Join(sqldb.ErrQueryCanceled, err)
		case "23P01":
			return errors.Join(sqldb.ErrExclusionViolation{Constraint: e.Constraint}, err)
		case "P0001":
			return errors.Join(sqldb.ErrRaisedException{Message: e.Message}, err)
		}
	}
	return err
}

// Class 22 — Data Exception

// IsInvalidTextRepresentation indicates if the error was caused by
// an invalid input value for a type, e.g. passing "not-a-uuid" to a uuid column.
func IsInvalidTextRepresentation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "22P02" // invalid_text_representation
}

// IsStringDataRightTruncation indicates if the error was caused by
// a value being too long for the target column type.
func IsStringDataRightTruncation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "22001" // string_data_right_truncation
}

// Class 23 — Integrity Constraint Violation

// IsIntegrityConstraintViolationClass indicates if the error belongs to
// the PostgreSQL integrity constraint violation class (23xxx).
func IsIntegrityConstraintViolationClass(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code.Class() == "23"
}

// IsRestrictViolation indicates if the error was caused by a restrict violation.
func IsRestrictViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "23001" // restrict_violation
}

// IsNotNullViolation indicates if the error was caused by a NOT NULL constraint violation.
func IsNotNullViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "23502" // not_null_violation
}

// IsForeignKeyViolation indicates if the error was caused by a foreign key constraint violation.
// If violatedConstraints are provided, it also checks that the violated constraint name matches one of them.
func IsForeignKeyViolation(err error, violatedConstraints ...string) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "23503" && // foreign_key_violation
		(len(violatedConstraints) == 0 || slices.Contains(violatedConstraints, e.Constraint))
}

// IsUniqueViolation indicates if the error was caused by a unique constraint violation.
func IsUniqueViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "23505" // unique_violation
}

// IsCheckViolation indicates if the error was caused by a check constraint violation.
func IsCheckViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "23514" // check_violation
}

// IsExclusionViolation indicates if the error was caused by an exclusion constraint violation.
func IsExclusionViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "23P01" // exclusion_violation
}

// Class 25 — Invalid Transaction State

// IsInFailedTransaction indicates if the error was caused by
// executing a statement in a transaction that has already failed.
// PostgreSQL rejects all commands in such a transaction until
// it is rolled back.
func IsInFailedTransaction(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "25P02" // in_failed_sql_transaction
}

// IsFailedTransaction returns true if conn is a transaction
// that is in a failed state by executing a dummy query
// and checking for the `in_failed_sql_transaction` error.
func IsFailedTransaction(ctx context.Context, conn sqldb.Connection) bool {
	return conn.Transaction().Active() && IsInFailedTransaction(conn.Exec(ctx, "SELECT 1"))
}

// IsTransactionTimeout indicates if the error was caused by
// a transaction exceeding the configured transaction_timeout.
func IsTransactionTimeout(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "25P04" // transaction_timeout
}

// Class 40 — Transaction Rollback

// IsSerializationFailure indicates if the error was caused by
// a transaction serialization failure. This typically occurs when using
// SERIALIZABLE or REPEATABLE READ isolation levels and concurrent
// transactions conflict. The caller should retry the transaction.
func IsSerializationFailure(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "40001" // serialization_failure
}

// IsDeadlockDetected indicates if the error was caused by
// a deadlock between concurrent transactions.
// The caller should retry the transaction.
func IsDeadlockDetected(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "40P01" // deadlock_detected
}

// Class 42 — Syntax Error or Access Rule Violation

// IsInsufficientPrivilege indicates if the error was caused by
// the current user lacking the required permissions for the operation.
func IsInsufficientPrivilege(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "42501" // insufficient_privilege
}

// Class 55 — Object Not In Prerequisite State

// IsLockNotAvailable indicates if the error was caused by
// a lock that could not be acquired, e.g. from SELECT ... FOR UPDATE NOWAIT.
func IsLockNotAvailable(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "55P03" // lock_not_available
}

// Class 57 - Operator Intervention

// IsQueryCanceled indicates if the passed error
// was caused by a user cancellation of a query.
// The pq error might not unwrap to context.Canceled
// even when it was caused by a context cancellation.
func IsQueryCanceled(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "57014"
}

// Class P0 — PL/pgSQL Error

// IsPLPGSQLErrorClass indicates if the error belongs to the PL/pgSQL error class (P0xxx).
func IsPLPGSQLErrorClass(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code.Class() == "P0"
}

// IsRaisedException indicates if the error was caused by a PL/pgSQL RAISE statement.
func IsRaisedException(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "P0001"
}

// GetRaisedException returns the message
// of a PL/pgSQL exception or an empty string
// if the error is nil or not an exception.
func GetRaisedException(err error) string {
	var e *pq.Error
	if errors.As(err, &e) && e.Code == "P0001" {
		return e.Message
	}
	return ""
}
