package pqconn

import (
	"context"
	"errors"
	"slices"

	"github.com/domonda/go-sqldb"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

func wrapKnownErrors(err error) error {
	if err == nil {
		return nil
	}
	var e *pq.Error
	if errors.As(err, &e) {
		switch e.Code {
		case pqerror.NullValueNotAllowed:
			return errors.Join(sqldb.ErrNullValueNotAllowed, err)
		case pqerror.IntegrityConstraintViolation:
			return errors.Join(sqldb.ErrIntegrityConstraintViolation{Constraint: e.Constraint}, err)
		case pqerror.RestrictViolation:
			return errors.Join(sqldb.ErrRestrictViolation{Constraint: e.Constraint}, err)
		case pqerror.NotNullViolation:
			return errors.Join(sqldb.ErrNotNullViolation{Constraint: e.Constraint}, err)
		case pqerror.ForeignKeyViolation:
			return errors.Join(sqldb.ErrForeignKeyViolation{Constraint: e.Constraint}, err)
		case pqerror.UniqueViolation:
			return errors.Join(sqldb.ErrUniqueViolation{Constraint: e.Constraint}, err)
		case pqerror.CheckViolation:
			return errors.Join(sqldb.ErrCheckViolation{Constraint: e.Constraint}, err)
		case pqerror.TRDeadlockDetected:
			return errors.Join(sqldb.ErrDeadlock, err)
		case pqerror.QueryCanceled:
			return errors.Join(sqldb.ErrQueryCanceled, err)
		case pqerror.ExclusionViolation:
			return errors.Join(sqldb.ErrExclusionViolation{Constraint: e.Constraint}, err)
		case pqerror.RaiseException:
			return errors.Join(sqldb.ErrRaisedException{Message: e.Message}, err)
		}
	}
	return err
}

// Class 08 — Connection Exception

// IsConnectionExceptionClass indicates if the error belongs to
// the PostgreSQL connection exception class (08xxx).
func IsConnectionExceptionClass(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code.Class() == pqerror.ClassConnectionException
}

// Class 22 — Data Exception

// IsDataExceptionClass indicates if the error belongs to
// the PostgreSQL data exception class (22xxx).
func IsDataExceptionClass(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code.Class() == pqerror.ClassDataException
}

// IsInvalidTextRepresentation indicates if the error was caused by
// an invalid input value for a type, e.g. passing "not-a-uuid" to a uuid column.
func IsInvalidTextRepresentation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.InvalidTextRepresentation
}

// IsStringDataRightTruncation indicates if the error was caused by
// a value being too long for the target column type.
func IsStringDataRightTruncation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.StringDataRightTruncation
}

// Class 23 — Integrity Constraint Violation

// IsIntegrityConstraintViolationClass indicates if the error belongs to
// the PostgreSQL integrity constraint violation class (23xxx).
func IsIntegrityConstraintViolationClass(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code.Class() == pqerror.ClassIntegrityConstraintViolation
}

// IsRestrictViolation indicates if the error was caused by a restrict violation.
func IsRestrictViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.RestrictViolation
}

// IsNotNullViolation indicates if the error was caused by a NOT NULL constraint violation.
func IsNotNullViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.NotNullViolation
}

// IsForeignKeyViolation indicates if the error was caused by a foreign key constraint violation.
// If violatedConstraints are provided, it also checks that the violated constraint name matches one of them.
func IsForeignKeyViolation(err error, violatedConstraints ...string) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.ForeignKeyViolation &&
		(len(violatedConstraints) == 0 || slices.Contains(violatedConstraints, e.Constraint))
}

// IsUniqueViolation indicates if the error was caused by a unique constraint violation.
func IsUniqueViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.UniqueViolation
}

// IsCheckViolation indicates if the error was caused by a check constraint violation.
func IsCheckViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.CheckViolation
}

// IsExclusionViolation indicates if the error was caused by an exclusion constraint violation.
func IsExclusionViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.ExclusionViolation
}

// Class 25 — Invalid Transaction State

// IsInFailedTransaction indicates if the error was caused by
// executing a statement in a transaction that has already failed.
// PostgreSQL rejects all commands in such a transaction until
// it is rolled back.
func IsInFailedTransaction(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.InFailedSQLTransaction
}

// IsFailedTransaction returns true if conn is a transaction
// that is in a failed state by executing a dummy query
// and checking for the `in_failed_sql_transaction` error.
func IsFailedTransaction(ctx context.Context, conn sqldb.Connection) bool {
	return conn.Transaction().Active() && IsInFailedTransaction(conn.Exec(ctx, "SELECT 1"))
}

// IsIdleInTransactionSessionTimeout indicates if the error was caused by
// a transaction being idle longer than the configured idle_in_transaction_session_timeout.
func IsIdleInTransactionSessionTimeout(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.IdleInTransactionSessionTimeout
}

// IsTransactionTimeout indicates if the error was caused by
// a transaction exceeding the configured transaction_timeout.
func IsTransactionTimeout(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.TransactionTimeout
}

// IsReadOnlySQLTransaction indicates if the error was caused by
// attempting a write operation on a read-only transaction or connection,
// e.g. when connected to a read replica.
func IsReadOnlySQLTransaction(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.ReadOnlySQLTransaction
}

// Class 40 — Transaction Rollback

// IsTransactionRollbackClass indicates if the error belongs to
// the PostgreSQL transaction rollback class (40xxx).
// This covers serialization failures, deadlocks, and other
// transaction rollback reasons. The caller should typically retry the transaction.
func IsTransactionRollbackClass(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code.Class() == pqerror.ClassTransactionRollback
}

// IsSerializationFailure indicates if the error was caused by
// a transaction serialization failure. This typically occurs when using
// SERIALIZABLE or REPEATABLE READ isolation levels and concurrent
// transactions conflict. The caller should retry the transaction.
func IsSerializationFailure(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.TRSerializationFailure
}

// IsDeadlockDetected indicates if the error was caused by
// a deadlock between concurrent transactions.
// The caller should retry the transaction.
func IsDeadlockDetected(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.TRDeadlockDetected
}

// Class 42 — Syntax Error or Access Rule Violation

// IsInsufficientPrivilege indicates if the error was caused by
// the current user lacking the required permissions for the operation.
func IsInsufficientPrivilege(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.InsufficientPrivilege
}

// IsUndefinedTable indicates if the error was caused by
// referencing a table that does not exist.
func IsUndefinedTable(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.UndefinedTable
}

// IsUndefinedColumn indicates if the error was caused by
// referencing a column that does not exist.
func IsUndefinedColumn(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.UndefinedColumn
}

// Class 53 — Insufficient Resources

// IsTooManyConnections indicates if the error was caused by
// exceeding the maximum number of allowed connections.
func IsTooManyConnections(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.TooManyConnections
}

// Class 55 — Object Not In Prerequisite State

// IsLockNotAvailable indicates if the error was caused by
// a lock that could not be acquired, e.g. from SELECT ... FOR UPDATE NOWAIT.
func IsLockNotAvailable(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.LockNotAvailable
}

// Class 57 - Operator Intervention

// IsQueryCanceled indicates if the passed error
// was caused by a user cancellation of a query.
// The pq error might not unwrap to context.Canceled
// even when it was caused by a context cancellation.
func IsQueryCanceled(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.QueryCanceled
}

// IsAdminShutdown indicates if the error was caused by
// the database server shutting down, e.g. during a restart or maintenance.
func IsAdminShutdown(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.AdminShutdown
}

// Class P0 — PL/pgSQL Error

// IsPLPGSQLErrorClass indicates if the error belongs to the PL/pgSQL error class (P0xxx).
func IsPLPGSQLErrorClass(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code.Class() == pqerror.ClassPLpgSQLError
}

// IsRaisedException indicates if the error was caused by a PL/pgSQL RAISE statement.
func IsRaisedException(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == pqerror.RaiseException
}

// GetRaisedException returns the message
// of a PL/pgSQL exception or an empty string
// if the error is nil or not an exception.
func GetRaisedException(err error) string {
	var e *pq.Error
	if errors.As(err, &e) && e.Code == pqerror.RaiseException {
		return e.Message
	}
	return ""
}
