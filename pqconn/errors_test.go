package pqconn

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"
)

func pqErr(code pq.ErrorCode) error {
	return &pq.Error{Code: code}
}

func pqErrWithConstraint(code pq.ErrorCode, constraint string) error {
	return &pq.Error{Code: code, Constraint: constraint}
}

func pqErrWithMessage(code pq.ErrorCode, message string) error {
	return &pq.Error{Code: code, Message: message}
}

func TestIsInvalidTextRepresentation(t *testing.T) {
	if !IsInvalidTextRepresentation(pqErr("22P02")) {
		t.Error("expected true for 22P02")
	}
	if IsInvalidTextRepresentation(pqErr("23505")) {
		t.Error("expected false for 23505")
	}
	if IsInvalidTextRepresentation(errors.New("other")) {
		t.Error("expected false for non-pq error")
	}
}

func TestIsStringDataRightTruncation(t *testing.T) {
	if !IsStringDataRightTruncation(pqErr("22001")) {
		t.Error("expected true for 22001")
	}
	if IsStringDataRightTruncation(pqErr("22P02")) {
		t.Error("expected false for 22P02")
	}
}

func TestIsIntegrityConstraintViolationClass(t *testing.T) {
	codes := []pq.ErrorCode{"23000", "23001", "23502", "23503", "23505", "23514", "23P01"}
	for _, code := range codes {
		if !IsIntegrityConstraintViolationClass(pqErr(code)) {
			t.Errorf("expected true for %s", code)
		}
	}
	if IsIntegrityConstraintViolationClass(pqErr("42501")) {
		t.Error("expected false for class 42")
	}
}

func TestIsRestrictViolation(t *testing.T) {
	if !IsRestrictViolation(pqErr("23001")) {
		t.Error("expected true for 23001")
	}
	if IsRestrictViolation(pqErr("23505")) {
		t.Error("expected false for 23505")
	}
}

func TestIsNotNullViolation(t *testing.T) {
	if !IsNotNullViolation(pqErr("23502")) {
		t.Error("expected true for 23502")
	}
	if IsNotNullViolation(pqErr("23001")) {
		t.Error("expected false for 23001")
	}
}

func TestIsForeignKeyViolation(t *testing.T) {
	if !IsForeignKeyViolation(pqErr("23503")) {
		t.Error("expected true for 23503 without constraint filter")
	}
	if !IsForeignKeyViolation(pqErrWithConstraint("23503", "fk_user"), "fk_user") {
		t.Error("expected true when constraint matches")
	}
	if IsForeignKeyViolation(pqErrWithConstraint("23503", "fk_user"), "fk_order") {
		t.Error("expected false when constraint does not match")
	}
	if IsForeignKeyViolation(pqErr("23505")) {
		t.Error("expected false for 23505")
	}
}

func TestIsUniqueViolation(t *testing.T) {
	if !IsUniqueViolation(pqErr("23505")) {
		t.Error("expected true for 23505")
	}
	if IsUniqueViolation(pqErr("23503")) {
		t.Error("expected false for 23503")
	}
}

func TestIsCheckViolation(t *testing.T) {
	if !IsCheckViolation(pqErr("23514")) {
		t.Error("expected true for 23514")
	}
	if IsCheckViolation(pqErr("23505")) {
		t.Error("expected false for 23505")
	}
}

func TestIsExclusionViolation(t *testing.T) {
	if !IsExclusionViolation(pqErr("23P01")) {
		t.Error("expected true for 23P01")
	}
	if IsExclusionViolation(pqErr("23505")) {
		t.Error("expected false for 23505")
	}
}

func TestIsSerializationFailure(t *testing.T) {
	if !IsSerializationFailure(pqErr("40001")) {
		t.Error("expected true for 40001")
	}
	if IsSerializationFailure(pqErr("40P01")) {
		t.Error("expected false for 40P01")
	}
}

func TestIsDeadlockDetected(t *testing.T) {
	if !IsDeadlockDetected(pqErr("40P01")) {
		t.Error("expected true for 40P01")
	}
	if IsDeadlockDetected(pqErr("40001")) {
		t.Error("expected false for 40001")
	}
}

func TestIsInsufficientPrivilege(t *testing.T) {
	if !IsInsufficientPrivilege(pqErr("42501")) {
		t.Error("expected true for 42501")
	}
	if IsInsufficientPrivilege(pqErr("42000")) {
		t.Error("expected false for 42000")
	}
}

func TestIsLockNotAvailable(t *testing.T) {
	if !IsLockNotAvailable(pqErr("55P03")) {
		t.Error("expected true for 55P03")
	}
	if IsLockNotAvailable(pqErr("55000")) {
		t.Error("expected false for 55000")
	}
}

func TestIsQueryCanceled(t *testing.T) {
	if !IsQueryCanceled(pqErr("57014")) {
		t.Error("expected true for 57014")
	}
	if IsQueryCanceled(pqErr("57000")) {
		t.Error("expected false for 57000")
	}
}

func TestIsPLPGSQLErrorClass(t *testing.T) {
	if !IsPLPGSQLErrorClass(pqErr("P0001")) {
		t.Error("expected true for P0001")
	}
	if !IsPLPGSQLErrorClass(pqErr("P0000")) {
		t.Error("expected true for P0000")
	}
	if IsPLPGSQLErrorClass(pqErr("42501")) {
		t.Error("expected false for 42501")
	}
}

func TestIsRaisedException(t *testing.T) {
	if !IsRaisedException(pqErr("P0001")) {
		t.Error("expected true for P0001")
	}
	if IsRaisedException(pqErr("P0000")) {
		t.Error("expected false for P0000")
	}
}

func TestGetRaisedException(t *testing.T) {
	msg := GetRaisedException(pqErrWithMessage("P0001", "custom error"))
	if msg != "custom error" {
		t.Errorf("got %q, want %q", msg, "custom error")
	}
	if got := GetRaisedException(pqErr("P0000")); got != "" {
		t.Errorf("expected empty for non-P0001, got %q", got)
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
		{"IsInvalidTextRepresentation", IsInvalidTextRepresentation},
		{"IsStringDataRightTruncation", IsStringDataRightTruncation},
		{"IsIntegrityConstraintViolationClass", IsIntegrityConstraintViolationClass},
		{"IsRestrictViolation", IsRestrictViolation},
		{"IsNotNullViolation", IsNotNullViolation},
		{"IsUniqueViolation", IsUniqueViolation},
		{"IsCheckViolation", IsCheckViolation},
		{"IsExclusionViolation", IsExclusionViolation},
		{"IsSerializationFailure", IsSerializationFailure},
		{"IsDeadlockDetected", IsDeadlockDetected},
		{"IsInsufficientPrivilege", IsInsufficientPrivilege},
		{"IsLockNotAvailable", IsLockNotAvailable},
		{"IsQueryCanceled", IsQueryCanceled},
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
	wrapped := fmt.Errorf("wrapped: %w", pqErr("23505"))
	if !IsUniqueViolation(wrapped) {
		t.Error("IsUniqueViolation should detect wrapped pq.Error")
	}
}
