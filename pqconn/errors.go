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
	if e := pq.As(err); e != nil {
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
		case pqerror.TRSerializationFailure:
			return errors.Join(sqldb.ErrSerializationFailure, err)
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
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassConnectionException
}

// Class 09 — Triggered Action Exception

// IsTriggeredActionExceptionClass indicates if the error belongs to
// the PostgreSQL triggered action exception class (09xxx).
func IsTriggeredActionExceptionClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassTriggeredActionException
}

// Class 0A — Feature Not Supported

// IsFeatureNotSupportedClass indicates if the error belongs to
// the PostgreSQL feature not supported class (0Axxx).
func IsFeatureNotSupportedClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassFeatureNotSupported
}

// Class 0B — Invalid Transaction Initiation

// IsInvalidTransactionInitiationClass indicates if the error belongs to
// the PostgreSQL invalid transaction initiation class (0Bxxx).
func IsInvalidTransactionInitiationClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassInvalidTransactionInitiation
}

// Class 0F — Locator Exception

// IsLocatorExceptionClass indicates if the error belongs to
// the PostgreSQL locator exception class (0Fxxx).
func IsLocatorExceptionClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassLocatorException
}

// Class 0L — Invalid Grantor

// IsInvalidGrantorClass indicates if the error belongs to
// the PostgreSQL invalid grantor class (0Lxxx).
func IsInvalidGrantorClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassInvalidGrantor
}

// Class 0P — Invalid Role Specification

// IsInvalidRoleSpecificationClass indicates if the error belongs to
// the PostgreSQL invalid role specification class (0Pxxx).
func IsInvalidRoleSpecificationClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassInvalidRoleSpecification
}

// Class 0Z — Diagnostics Exception

// IsDiagnosticsExceptionClass indicates if the error belongs to
// the PostgreSQL diagnostics exception class (0Zxxx).
func IsDiagnosticsExceptionClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassDiagnosticsException
}

// Class 20 — Case Not Found

// IsCaseNotFoundClass indicates if the error belongs to
// the PostgreSQL case not found class (20xxx).
func IsCaseNotFoundClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassCaseNotFound
}

// Class 21 — Cardinality Violation

// IsCardinalityViolationClass indicates if the error belongs to
// the PostgreSQL cardinality violation class (21xxx).
func IsCardinalityViolationClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassCardinalityViolation
}

// Class 22 — Data Exception

// IsDataExceptionClass indicates if the error belongs to
// the PostgreSQL data exception class (22xxx).
func IsDataExceptionClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassDataException
}

// IsInvalidTextRepresentation indicates if the error was caused by
// an invalid input value for a type, e.g. passing "not-a-uuid" to a uuid column.
func IsInvalidTextRepresentation(err error) bool {
	return pq.As(err, pqerror.InvalidTextRepresentation) != nil
}

// IsStringDataRightTruncation indicates if the error was caused by
// a value being too long for the target column type.
func IsStringDataRightTruncation(err error) bool {
	return pq.As(err, pqerror.StringDataRightTruncation) != nil
}

// Class 23 — Integrity Constraint Violation

// IsIntegrityConstraintViolationClass indicates if the error belongs to
// the PostgreSQL integrity constraint violation class (23xxx).
func IsIntegrityConstraintViolationClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassIntegrityConstraintViolation
}

// IsRestrictViolation indicates if the error was caused by a restrict violation.
func IsRestrictViolation(err error) bool {
	return pq.As(err, pqerror.RestrictViolation) != nil
}

// IsNotNullViolation indicates if the error was caused by a NOT NULL constraint violation.
func IsNotNullViolation(err error) bool {
	return pq.As(err, pqerror.NotNullViolation) != nil
}

// IsForeignKeyViolation indicates if the error was caused by a foreign key constraint violation.
// If violatedConstraints are provided, it also checks that the violated constraint name matches one of them.
func IsForeignKeyViolation(err error, violatedConstraints ...string) bool {
	e := pq.As(err, pqerror.ForeignKeyViolation)
	return e != nil && (len(violatedConstraints) == 0 || slices.Contains(violatedConstraints, e.Constraint))
}

// IsUniqueViolation indicates if the error was caused by a unique constraint violation.
func IsUniqueViolation(err error) bool {
	return pq.As(err, pqerror.UniqueViolation) != nil
}

// IsCheckViolation indicates if the error was caused by a check constraint violation.
func IsCheckViolation(err error) bool {
	return pq.As(err, pqerror.CheckViolation) != nil
}

// IsExclusionViolation indicates if the error was caused by an exclusion constraint violation.
func IsExclusionViolation(err error) bool {
	return pq.As(err, pqerror.ExclusionViolation) != nil
}

// Class 24 — Invalid Cursor State

// IsInvalidCursorStateClass indicates if the error belongs to
// the PostgreSQL invalid cursor state class (24xxx).
func IsInvalidCursorStateClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassInvalidCursorState
}

// Class 25 — Invalid Transaction State

// IsInvalidTransactionStateClass indicates if the error belongs to
// the PostgreSQL invalid transaction state class (25xxx).
func IsInvalidTransactionStateClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassInvalidTransactionState
}

// IsInFailedTransaction indicates if the error was caused by
// executing a statement in a transaction that has already failed.
// PostgreSQL rejects all commands in such a transaction until
// it is rolled back.
func IsInFailedTransaction(err error) bool {
	return pq.As(err, pqerror.InFailedSQLTransaction) != nil
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
	return pq.As(err, pqerror.IdleInTransactionSessionTimeout) != nil
}

// IsTransactionTimeout indicates if the error was caused by
// a transaction exceeding the configured transaction_timeout.
func IsTransactionTimeout(err error) bool {
	return pq.As(err, pqerror.TransactionTimeout) != nil
}

// IsReadOnlySQLTransaction indicates if the error was caused by
// attempting a write operation on a read-only transaction or connection,
// e.g. when connected to a read replica.
func IsReadOnlySQLTransaction(err error) bool {
	return pq.As(err, pqerror.ReadOnlySQLTransaction) != nil
}

// Class 26 — Invalid SQL Statement Name

// IsInvalidSQLStatementNameClass indicates if the error belongs to
// the PostgreSQL invalid SQL statement name class (26xxx).
func IsInvalidSQLStatementNameClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassInvalidSQLStatementName
}

// Class 27 — Triggered Data Change Violation

// IsTriggeredDataChangeViolationClass indicates if the error belongs to
// the PostgreSQL triggered data change violation class (27xxx).
func IsTriggeredDataChangeViolationClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassTriggeredDataChangeViolation
}

// Class 28 — Invalid Authorization Specification

// IsInvalidAuthorizationSpecificationClass indicates if the error belongs to
// the PostgreSQL invalid authorization specification class (28xxx).
func IsInvalidAuthorizationSpecificationClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassInvalidAuthorizationSpecification
}

// Class 2B — Dependent Privilege Descriptors Still Exist

// IsDependentPrivilegeDescriptorsStillExistClass indicates if the error belongs to
// the PostgreSQL dependent privilege descriptors still exist class (2Bxxx).
func IsDependentPrivilegeDescriptorsStillExistClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassDependentPrivilegeDescriptorsStillExist
}

// Class 2D — Invalid Transaction Termination

// IsInvalidTransactionTerminationClass indicates if the error belongs to
// the PostgreSQL invalid transaction termination class (2Dxxx).
func IsInvalidTransactionTerminationClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassInvalidTransactionTermination
}

// Class 2F — SQL Routine Exception

// IsSQLRoutineExceptionClass indicates if the error belongs to
// the PostgreSQL SQL routine exception class (2Fxxx).
func IsSQLRoutineExceptionClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassSQLRoutineException
}

// Class 34 — Invalid Cursor Name

// IsInvalidCursorNameClass indicates if the error belongs to
// the PostgreSQL invalid cursor name class (34xxx).
func IsInvalidCursorNameClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassInvalidCursorName
}

// Class 38 — External Routine Exception

// IsExternalRoutineExceptionClass indicates if the error belongs to
// the PostgreSQL external routine exception class (38xxx).
func IsExternalRoutineExceptionClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassExternalRoutineException
}

// Class 39 — External Routine Invocation Exception

// IsExternalRoutineInvocationExceptionClass indicates if the error belongs to
// the PostgreSQL external routine invocation exception class (39xxx).
func IsExternalRoutineInvocationExceptionClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassExternalRoutineInvocationException
}

// Class 3B — Savepoint Exception

// IsSavepointExceptionClass indicates if the error belongs to
// the PostgreSQL savepoint exception class (3Bxxx).
func IsSavepointExceptionClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassSavepointException
}

// Class 3D — Invalid Catalog Name

// IsInvalidCatalogNameClass indicates if the error belongs to
// the PostgreSQL invalid catalog name class (3Dxxx).
func IsInvalidCatalogNameClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassInvalidCatalogName
}

// Class 3F — Invalid Schema Name

// IsInvalidSchemaNameClass indicates if the error belongs to
// the PostgreSQL invalid schema name class (3Fxxx).
func IsInvalidSchemaNameClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassInvalidSchemaName
}

// Class 40 — Transaction Rollback

// IsTransactionRollbackClass indicates if the error belongs to
// the PostgreSQL transaction rollback class (40xxx).
// This covers serialization failures, deadlocks, and other
// transaction rollback reasons. The caller should typically retry the transaction.
func IsTransactionRollbackClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassTransactionRollback
}

// IsSerializationFailure indicates if the error was caused by
// a transaction serialization failure. This typically occurs when using
// SERIALIZABLE or REPEATABLE READ isolation levels and concurrent
// transactions conflict. The caller should retry the transaction.
func IsSerializationFailure(err error) bool {
	return pq.As(err, pqerror.TRSerializationFailure) != nil
}

// IsDeadlockDetected indicates if the error was caused by
// a deadlock between concurrent transactions.
// The caller should retry the transaction.
func IsDeadlockDetected(err error) bool {
	return pq.As(err, pqerror.TRDeadlockDetected) != nil
}

// Class 42 — Syntax Error or Access Rule Violation

// IsSyntaxErrorOrAccessRuleViolationClass indicates if the error belongs to
// the PostgreSQL syntax error or access rule violation class (42xxx).
func IsSyntaxErrorOrAccessRuleViolationClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassSyntaxErrorOrAccessRuleViolation
}

// IsInsufficientPrivilege indicates if the error was caused by
// the current user lacking the required permissions for the operation.
func IsInsufficientPrivilege(err error) bool {
	return pq.As(err, pqerror.InsufficientPrivilege) != nil
}

// IsUndefinedTable indicates if the error was caused by
// referencing a table that does not exist.
func IsUndefinedTable(err error) bool {
	return pq.As(err, pqerror.UndefinedTable) != nil
}

// IsUndefinedColumn indicates if the error was caused by
// referencing a column that does not exist.
func IsUndefinedColumn(err error) bool {
	return pq.As(err, pqerror.UndefinedColumn) != nil
}

// Class 44 — WITH CHECK OPTION Violation

// IsWithCheckOptionViolationClass indicates if the error belongs to
// the PostgreSQL WITH CHECK OPTION violation class (44xxx).
func IsWithCheckOptionViolationClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassWithCheckOptionViolation
}

// Class 53 — Insufficient Resources

// IsInsufficientResourcesClass indicates if the error belongs to
// the PostgreSQL insufficient resources class (53xxx).
// This covers disk full, out of memory, too many connections,
// and configuration limit exceeded.
func IsInsufficientResourcesClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassInsufficientResources
}

// IsTooManyConnections indicates if the error was caused by
// exceeding the maximum number of allowed connections.
func IsTooManyConnections(err error) bool {
	return pq.As(err, pqerror.TooManyConnections) != nil
}

// Class 54 — Program Limit Exceeded

// IsProgramLimitExceededClass indicates if the error belongs to
// the PostgreSQL program limit exceeded class (54xxx).
// This covers statement too complex, too many columns, and too many arguments.
func IsProgramLimitExceededClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassProgramLimitExceeded
}

// Class 55 — Object Not In Prerequisite State

// IsObjectNotInPrerequisiteStateClass indicates if the error belongs to
// the PostgreSQL object not in prerequisite state class (55xxx).
func IsObjectNotInPrerequisiteStateClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassObjectNotInPrerequisiteState
}

// IsLockNotAvailable indicates if the error was caused by
// a lock that could not be acquired, e.g. from SELECT ... FOR UPDATE NOWAIT.
func IsLockNotAvailable(err error) bool {
	return pq.As(err, pqerror.LockNotAvailable) != nil
}

// Class 57 — Operator Intervention

// IsOperatorInterventionClass indicates if the error belongs to
// the PostgreSQL operator intervention class (57xxx).
// This covers query cancellation, admin shutdown, crash shutdown,
// inability to connect, database dropped, and idle session timeout.
func IsOperatorInterventionClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassOperatorIntervention
}

// IsQueryCanceled indicates if the passed error
// was caused by a user cancellation of a query.
// The pq error might not unwrap to context.Canceled
// even when it was caused by a context cancellation.
func IsQueryCanceled(err error) bool {
	return pq.As(err, pqerror.QueryCanceled) != nil
}

// IsAdminShutdown indicates if the error was caused by
// the database server shutting down, e.g. during a restart or maintenance.
func IsAdminShutdown(err error) bool {
	return pq.As(err, pqerror.AdminShutdown) != nil
}

// Class 58 — System Error

// IsSystemErrorClass indicates if the error belongs to
// the PostgreSQL system error class (58xxx).
// These are errors external to PostgreSQL itself,
// such as I/O errors or file system problems.
func IsSystemErrorClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassSystemError
}

// Class F0 — Configuration File Error

// IsConfigFileErrorClass indicates if the error belongs to
// the PostgreSQL configuration file error class (F0xxx).
func IsConfigFileErrorClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassConfigFileError
}

// Class HV — Foreign Data Wrapper Error

// IsFDWErrorClass indicates if the error belongs to
// the PostgreSQL foreign data wrapper error class (HVxxx).
func IsFDWErrorClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassFDWError
}

// Class P0 — PL/pgSQL Error

// IsPLPGSQLErrorClass indicates if the error belongs to the PL/pgSQL error class (P0xxx).
func IsPLPGSQLErrorClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassPLpgSQLError
}

// IsRaisedException indicates if the error was caused by a PL/pgSQL RAISE statement.
func IsRaisedException(err error) bool {
	return pq.As(err, pqerror.RaiseException) != nil
}

// GetRaisedException returns the message
// of a PL/pgSQL exception or an empty string
// if the error is nil or not an exception.
func GetRaisedException(err error) string {
	if e := pq.As(err, pqerror.RaiseException); e != nil {
		return e.Message
	}
	return ""
}

// Class XX — Internal Error

// IsInternalErrorClass indicates if the error belongs to
// the PostgreSQL internal error class (XXxxx).
// This covers data corruption and index corruption.
func IsInternalErrorClass(err error) bool {
	e := pq.As(err)
	return e != nil && e.Code.Class() == pqerror.ClassInternalError
}
