package pqconn

import (
	"errors"
	"slices"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
	"github.com/lib/pq"
)

func wrapError(err error, query, argFmt string, args []any) error {
	return impl.WrapNonNilErrorWithQuery(WrapKnownErrors(err), query, argFmt, args)
}

func WrapKnownErrors(err error) error {
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
		case "23P01":
			return errors.Join(sqldb.ErrExclusionViolation{Constraint: e.Constraint}, err)
		case "P0001":
			return errors.Join(sqldb.ErrRaisedException{Message: e.Message}, err)
		}
	}
	return err
}

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
