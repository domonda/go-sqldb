package pqconn

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

func pqErr(code pqerror.Code) error {
	return &pq.Error{Code: code}
}

func pqErrWithConstraint(code pqerror.Code, constraint string) error {
	return &pq.Error{Code: code, Constraint: constraint}
}

func pqErrWithMessage(code pqerror.Code, message string) error {
	return &pq.Error{Code: code, Message: message}
}

func TestIsConnectionExceptionClass(t *testing.T) {
	codes := []pqerror.Code{
		pqerror.ConnectionException,
		pqerror.ConnectionDoesNotExist,
		pqerror.ConnectionFailure,
		pqerror.SQLClientUnableToEstablishSQLConnection,
		pqerror.ProtocolViolation,
	}
	for _, code := range codes {
		if !IsConnectionExceptionClass(pqErr(code)) {
			t.Errorf("expected true for %s", code)
		}
	}
	if IsConnectionExceptionClass(pqErr(pqerror.UniqueViolation)) {
		t.Error("expected false for class 23")
	}
}

func TestIsDataExceptionClass(t *testing.T) {
	codes := []pqerror.Code{
		pqerror.DataException,
		pqerror.InvalidTextRepresentation,
		pqerror.StringDataRightTruncation,
		pqerror.NullValueNotAllowed,
		pqerror.NumericValueOutOfRange,
		pqerror.DivisionByZero,
	}
	for _, code := range codes {
		if !IsDataExceptionClass(pqErr(code)) {
			t.Errorf("expected true for %s", code)
		}
	}
	if IsDataExceptionClass(pqErr(pqerror.UniqueViolation)) {
		t.Error("expected false for class 23")
	}
}

func TestIsInvalidTextRepresentation(t *testing.T) {
	if !IsInvalidTextRepresentation(pqErr(pqerror.InvalidTextRepresentation)) {
		t.Error("expected true for InvalidTextRepresentation")
	}
	if IsInvalidTextRepresentation(pqErr(pqerror.UniqueViolation)) {
		t.Error("expected false for UniqueViolation")
	}
	if IsInvalidTextRepresentation(errors.New("other")) {
		t.Error("expected false for non-pq error")
	}
}

func TestIsStringDataRightTruncation(t *testing.T) {
	if !IsStringDataRightTruncation(pqErr(pqerror.StringDataRightTruncation)) {
		t.Error("expected true for StringDataRightTruncation")
	}
	if IsStringDataRightTruncation(pqErr(pqerror.InvalidTextRepresentation)) {
		t.Error("expected false for InvalidTextRepresentation")
	}
}

func TestIsIntegrityConstraintViolationClass(t *testing.T) {
	codes := []pqerror.Code{
		pqerror.IntegrityConstraintViolation,
		pqerror.RestrictViolation,
		pqerror.NotNullViolation,
		pqerror.ForeignKeyViolation,
		pqerror.UniqueViolation,
		pqerror.CheckViolation,
		pqerror.ExclusionViolation,
	}
	for _, code := range codes {
		if !IsIntegrityConstraintViolationClass(pqErr(code)) {
			t.Errorf("expected true for %s", code)
		}
	}
	if IsIntegrityConstraintViolationClass(pqErr(pqerror.InsufficientPrivilege)) {
		t.Error("expected false for class 42")
	}
}

func TestIsRestrictViolation(t *testing.T) {
	if !IsRestrictViolation(pqErr(pqerror.RestrictViolation)) {
		t.Error("expected true for RestrictViolation")
	}
	if IsRestrictViolation(pqErr(pqerror.UniqueViolation)) {
		t.Error("expected false for UniqueViolation")
	}
}

func TestIsNotNullViolation(t *testing.T) {
	if !IsNotNullViolation(pqErr(pqerror.NotNullViolation)) {
		t.Error("expected true for NotNullViolation")
	}
	if IsNotNullViolation(pqErr(pqerror.RestrictViolation)) {
		t.Error("expected false for RestrictViolation")
	}
}

func TestIsForeignKeyViolation(t *testing.T) {
	if !IsForeignKeyViolation(pqErr(pqerror.ForeignKeyViolation)) {
		t.Error("expected true for ForeignKeyViolation without constraint filter")
	}
	if !IsForeignKeyViolation(pqErrWithConstraint(pqerror.ForeignKeyViolation, "fk_user"), "fk_user") {
		t.Error("expected true when constraint matches")
	}
	if IsForeignKeyViolation(pqErrWithConstraint(pqerror.ForeignKeyViolation, "fk_user"), "fk_order") {
		t.Error("expected false when constraint does not match")
	}
	if IsForeignKeyViolation(pqErr(pqerror.UniqueViolation)) {
		t.Error("expected false for UniqueViolation")
	}
}

func TestIsUniqueViolation(t *testing.T) {
	if !IsUniqueViolation(pqErr(pqerror.UniqueViolation)) {
		t.Error("expected true for UniqueViolation")
	}
	if IsUniqueViolation(pqErr(pqerror.ForeignKeyViolation)) {
		t.Error("expected false for ForeignKeyViolation")
	}
}

func TestIsCheckViolation(t *testing.T) {
	if !IsCheckViolation(pqErr(pqerror.CheckViolation)) {
		t.Error("expected true for CheckViolation")
	}
	if IsCheckViolation(pqErr(pqerror.UniqueViolation)) {
		t.Error("expected false for UniqueViolation")
	}
}

func TestIsExclusionViolation(t *testing.T) {
	if !IsExclusionViolation(pqErr(pqerror.ExclusionViolation)) {
		t.Error("expected true for ExclusionViolation")
	}
	if IsExclusionViolation(pqErr(pqerror.UniqueViolation)) {
		t.Error("expected false for UniqueViolation")
	}
}

func TestIsIdleInTransactionSessionTimeout(t *testing.T) {
	if !IsIdleInTransactionSessionTimeout(pqErr(pqerror.IdleInTransactionSessionTimeout)) {
		t.Error("expected true for IdleInTransactionSessionTimeout")
	}
	if IsIdleInTransactionSessionTimeout(pqErr(pqerror.InFailedSQLTransaction)) {
		t.Error("expected false for InFailedSQLTransaction")
	}
}

func TestIsReadOnlySQLTransaction(t *testing.T) {
	if !IsReadOnlySQLTransaction(pqErr(pqerror.ReadOnlySQLTransaction)) {
		t.Error("expected true for ReadOnlySQLTransaction")
	}
	if IsReadOnlySQLTransaction(pqErr(pqerror.InFailedSQLTransaction)) {
		t.Error("expected false for InFailedSQLTransaction")
	}
}

func TestIsTransactionRollbackClass(t *testing.T) {
	codes := []pqerror.Code{
		pqerror.TransactionRollback,
		pqerror.TRSerializationFailure,
		pqerror.TRDeadlockDetected,
		pqerror.TRIntegrityConstraintViolation,
		pqerror.TRStatementCompletionUnknown,
	}
	for _, code := range codes {
		if !IsTransactionRollbackClass(pqErr(code)) {
			t.Errorf("expected true for %s", code)
		}
	}
	if IsTransactionRollbackClass(pqErr(pqerror.UniqueViolation)) {
		t.Error("expected false for class 23")
	}
}

func TestIsSerializationFailure(t *testing.T) {
	if !IsSerializationFailure(pqErr(pqerror.TRSerializationFailure)) {
		t.Error("expected true for TRSerializationFailure")
	}
	if IsSerializationFailure(pqErr(pqerror.TRDeadlockDetected)) {
		t.Error("expected false for TRDeadlockDetected")
	}
}

func TestIsDeadlockDetected(t *testing.T) {
	if !IsDeadlockDetected(pqErr(pqerror.TRDeadlockDetected)) {
		t.Error("expected true for TRDeadlockDetected")
	}
	if IsDeadlockDetected(pqErr(pqerror.TRSerializationFailure)) {
		t.Error("expected false for TRSerializationFailure")
	}
}

func TestIsInsufficientPrivilege(t *testing.T) {
	if !IsInsufficientPrivilege(pqErr(pqerror.InsufficientPrivilege)) {
		t.Error("expected true for InsufficientPrivilege")
	}
	if IsInsufficientPrivilege(pqErr(pqerror.SyntaxErrorOrAccessRuleViolation)) {
		t.Error("expected false for SyntaxErrorOrAccessRuleViolation")
	}
}

func TestIsUndefinedTable(t *testing.T) {
	if !IsUndefinedTable(pqErr(pqerror.UndefinedTable)) {
		t.Error("expected true for UndefinedTable")
	}
	if IsUndefinedTable(pqErr(pqerror.UndefinedColumn)) {
		t.Error("expected false for UndefinedColumn")
	}
}

func TestIsUndefinedColumn(t *testing.T) {
	if !IsUndefinedColumn(pqErr(pqerror.UndefinedColumn)) {
		t.Error("expected true for UndefinedColumn")
	}
	if IsUndefinedColumn(pqErr(pqerror.UndefinedTable)) {
		t.Error("expected false for UndefinedTable")
	}
}

func TestIsTooManyConnections(t *testing.T) {
	if !IsTooManyConnections(pqErr(pqerror.TooManyConnections)) {
		t.Error("expected true for TooManyConnections")
	}
	if IsTooManyConnections(pqErr(pqerror.InsufficientResources)) {
		t.Error("expected false for InsufficientResources")
	}
}

func TestIsLockNotAvailable(t *testing.T) {
	if !IsLockNotAvailable(pqErr(pqerror.LockNotAvailable)) {
		t.Error("expected true for LockNotAvailable")
	}
	if IsLockNotAvailable(pqErr(pqerror.ObjectNotInPrerequisiteState)) {
		t.Error("expected false for ObjectNotInPrerequisiteState")
	}
}

func TestIsQueryCanceled(t *testing.T) {
	if !IsQueryCanceled(pqErr(pqerror.QueryCanceled)) {
		t.Error("expected true for QueryCanceled")
	}
	if IsQueryCanceled(pqErr(pqerror.OperatorIntervention)) {
		t.Error("expected false for OperatorIntervention")
	}
}

func TestIsAdminShutdown(t *testing.T) {
	if !IsAdminShutdown(pqErr(pqerror.AdminShutdown)) {
		t.Error("expected true for AdminShutdown")
	}
	if IsAdminShutdown(pqErr(pqerror.QueryCanceled)) {
		t.Error("expected false for QueryCanceled")
	}
}

func TestIsPLPGSQLErrorClass(t *testing.T) {
	if !IsPLPGSQLErrorClass(pqErr(pqerror.RaiseException)) {
		t.Error("expected true for RaiseException")
	}
	if !IsPLPGSQLErrorClass(pqErr(pqerror.PLpgSQLError)) {
		t.Error("expected true for PLpgSQLError")
	}
	if IsPLPGSQLErrorClass(pqErr(pqerror.InsufficientPrivilege)) {
		t.Error("expected false for InsufficientPrivilege")
	}
}

func TestIsRaisedException(t *testing.T) {
	if !IsRaisedException(pqErr(pqerror.RaiseException)) {
		t.Error("expected true for RaiseException")
	}
	if IsRaisedException(pqErr(pqerror.PLpgSQLError)) {
		t.Error("expected false for PLpgSQLError")
	}
}

func TestGetRaisedException(t *testing.T) {
	msg := GetRaisedException(pqErrWithMessage(pqerror.RaiseException, "custom error"))
	if msg != "custom error" {
		t.Errorf("got %q, want %q", msg, "custom error")
	}
	if got := GetRaisedException(pqErr(pqerror.PLpgSQLError)); got != "" {
		t.Errorf("expected empty for PLpgSQLError, got %q", got)
	}
	if got := GetRaisedException(nil); got != "" {
		t.Errorf("expected empty for nil, got %q", got)
	}
	if got := GetRaisedException(fmt.Errorf("plain")); got != "" {
		t.Errorf("expected empty for non-pq error, got %q", got)
	}
}

func TestErrorPredicates_NonPqError(t *testing.T) {
	err := errors.New("not a pq error")
	predicates := []struct {
		name string
		fn   func(error) bool
	}{
		{"IsConnectionExceptionClass", IsConnectionExceptionClass},
		{"IsDataExceptionClass", IsDataExceptionClass},
		{"IsInvalidTextRepresentation", IsInvalidTextRepresentation},
		{"IsStringDataRightTruncation", IsStringDataRightTruncation},
		{"IsIntegrityConstraintViolationClass", IsIntegrityConstraintViolationClass},
		{"IsRestrictViolation", IsRestrictViolation},
		{"IsNotNullViolation", IsNotNullViolation},
		{"IsUniqueViolation", IsUniqueViolation},
		{"IsCheckViolation", IsCheckViolation},
		{"IsExclusionViolation", IsExclusionViolation},
		{"IsIdleInTransactionSessionTimeout", IsIdleInTransactionSessionTimeout},
		{"IsReadOnlySQLTransaction", IsReadOnlySQLTransaction},
		{"IsTransactionRollbackClass", IsTransactionRollbackClass},
		{"IsSerializationFailure", IsSerializationFailure},
		{"IsDeadlockDetected", IsDeadlockDetected},
		{"IsInsufficientPrivilege", IsInsufficientPrivilege},
		{"IsUndefinedTable", IsUndefinedTable},
		{"IsUndefinedColumn", IsUndefinedColumn},
		{"IsTooManyConnections", IsTooManyConnections},
		{"IsLockNotAvailable", IsLockNotAvailable},
		{"IsQueryCanceled", IsQueryCanceled},
		{"IsAdminShutdown", IsAdminShutdown},
		{"IsPLPGSQLErrorClass", IsPLPGSQLErrorClass},
		{"IsRaisedException", IsRaisedException},
	}
	for _, p := range predicates {
		t.Run(p.name, func(t *testing.T) {
			if p.fn(err) {
				t.Errorf("%s should return false for non-pq error", p.name)
			}
		})
	}
}

func TestErrorPredicates_WrappedError(t *testing.T) {
	wrapped := fmt.Errorf("wrapped: %w", pqErr(pqerror.UniqueViolation))
	if !IsUniqueViolation(wrapped) {
		t.Error("IsUniqueViolation should detect wrapped pq.Error")
	}
}
