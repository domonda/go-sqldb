package pqconn

import (
	"errors"
	"slices"

	"github.com/lib/pq"
)

// Class 23 — Integrity Constraint Violation

func IsIntegrityConstraintViolation(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "23000" // integrity_constraint_violation
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

// Class P0 — PL/pgSQL Error

func IsPlpgsqlError(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "P0000" // plpgsql_error
}

func IsRaiseException(err error) bool {
	var e *pq.Error
	return errors.As(err, &e) && e.Code == "P0001" // raise_exception
}
