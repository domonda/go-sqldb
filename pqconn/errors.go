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
		case "57014":
			return errors.Join(context.Canceled, err)
		case "23P01":
			return errors.Join(sqldb.ErrExclusionViolation{Constraint: e.Constraint}, err)
		case "P0001":
			return errors.Join(sqldb.ErrRaisedException{Message: e.Message}, err)
		}
	}
	return err
}

// Class 23 — Integrity Constraint Violation

func IsIntegrityConstraintViolationClass(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code.Class() == "23"
}

func IsRestrictViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "23001" // restrict_violation
}

func IsNotNullViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "23502" // not_null_violation
}

func IsForeignKeyViolation(err error, violatedConstraints ...string) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "23503" && // foreign_key_violation
		(len(violatedConstraints) == 0 || slices.Contains(violatedConstraints, e.Constraint))
}

func IsUniqueViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "23505" // unique_violation
}

func IsCheckViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "23514" // check_violation
}

func IsExclusionViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "23P01" // exclusion_violation
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

func IsPLPGSQLErrorClass(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code.Class() == "P0"
}

func IsRaisedException(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "P0001"
}

// GetRaisedException returns the message
// of an PL/pgSQL exception or and empty string
// if the error is nil or not an exception.
func GetRaisedException(err error) string {
	var e *pq.Error
	if errors.As(err, &e) && e.Code == "P0001" {
		return e.Message
	}
	return ""
}
