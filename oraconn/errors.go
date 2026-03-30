package oraconn

import (
	"errors"
	"slices"
	"strings"

	"github.com/sijms/go-ora/v2/network"

	"github.com/domonda/go-sqldb"
)

// Oracle error codes for constraint violations and other mapped errors.
// https://docs.oracle.com/en/database/oracle/oracle-database/23/errmg/
const (
	errUniqueViolation      = 1     // ORA-00001: unique constraint violated
	errDeadlock             = 60    // ORA-00060: deadlock detected while waiting for resource
	errCannotInsertNull     = 1400  // ORA-01400: cannot insert NULL
	errQueryCanceled        = 1013  // ORA-01013: user requested cancel of current operation
	errFKParentNotFound     = 2291  // ORA-02291: integrity constraint - parent key not found
	errFKChildRecordFound   = 2292  // ORA-02292: integrity constraint - child record found
	errCheckViolation       = 2290  // ORA-02290: check constraint violated
	errSerializationFailure = 8177  // ORA-08177: can't serialize access for this transaction
	errRaisedUserException  = 20000 // ORA-20000 through ORA-20999: user-defined exceptions
)

func wrapKnownErrors(err error) error {
	if err == nil {
		return nil
	}
	var oraErr *network.OracleError
	if !errors.As(err, &oraErr) {
		return err
	}
	constraint := extractConstraintName(oraErr.ErrMsg)
	switch {
	case oraErr.ErrCode == errUniqueViolation:
		return errors.Join(sqldb.ErrUniqueViolation{Constraint: constraint}, err)
	case oraErr.ErrCode == errCannotInsertNull:
		return errors.Join(sqldb.ErrNotNullViolation{Constraint: constraint}, err)
	case oraErr.ErrCode == errFKParentNotFound || oraErr.ErrCode == errFKChildRecordFound:
		return errors.Join(sqldb.ErrForeignKeyViolation{Constraint: constraint}, err)
	case oraErr.ErrCode == errCheckViolation:
		return errors.Join(sqldb.ErrCheckViolation{Constraint: constraint}, err)
	case oraErr.ErrCode == errDeadlock:
		return errors.Join(sqldb.ErrDeadlock, err)
	case oraErr.ErrCode == errSerializationFailure:
		return errors.Join(sqldb.ErrSerializationFailure, err)
	case oraErr.ErrCode == errQueryCanceled:
		return errors.Join(sqldb.ErrQueryCanceled, err)
	case oraErr.ErrCode >= errRaisedUserException && oraErr.ErrCode <= 20999:
		return errors.Join(sqldb.ErrRaisedException{Message: oraErr.ErrMsg}, err)
	}
	return err
}

// extractConstraintName attempts to extract a constraint name
// from an Oracle error message. Oracle typically includes the
// constraint name in parentheses like "SCHEMA.CONSTRAINT_NAME".
func extractConstraintName(msg string) string {
	// Oracle error messages typically include constraint names in parentheses:
	// "ORA-00001: unique constraint (SCHEMA.UK_NAME) violated"
	start := strings.Index(msg, "(")
	if start == -1 {
		return ""
	}
	end := strings.Index(msg[start:], ")")
	if end == -1 {
		return ""
	}
	return msg[start+1 : start+end]
}

// IsUniqueViolation reports whether err was caused by a unique constraint
// violation (ORA-00001).
func IsUniqueViolation(err error) bool {
	var oraErr *network.OracleError
	return errors.As(err, &oraErr) && oraErr.ErrCode == errUniqueViolation
}

// IsNotNullViolation reports whether err was caused by inserting a NULL
// into a column that does not allow nulls (ORA-01400).
func IsNotNullViolation(err error) bool {
	var oraErr *network.OracleError
	return errors.As(err, &oraErr) && oraErr.ErrCode == errCannotInsertNull
}

// IsForeignKeyViolation reports whether err was caused by a foreign key
// constraint violation (ORA-02291 or ORA-02292).
// If violatedConstraints is non-empty, only returns true when the
// violated constraint name matches one of the provided names.
func IsForeignKeyViolation(err error, violatedConstraints ...string) bool {
	var oraErr *network.OracleError
	if !errors.As(err, &oraErr) {
		return false
	}
	if oraErr.ErrCode != errFKParentNotFound && oraErr.ErrCode != errFKChildRecordFound {
		return false
	}
	if len(violatedConstraints) == 0 {
		return true
	}
	return slices.Contains(violatedConstraints, extractConstraintName(oraErr.ErrMsg))
}

// IsCheckViolation reports whether err was caused by a CHECK constraint
// violation (ORA-02290).
func IsCheckViolation(err error) bool {
	var oraErr *network.OracleError
	return errors.As(err, &oraErr) && oraErr.ErrCode == errCheckViolation
}

// IsDeadlockDetected reports whether err was caused by a deadlock
// (ORA-00060).
func IsDeadlockDetected(err error) bool {
	var oraErr *network.OracleError
	return errors.As(err, &oraErr) && oraErr.ErrCode == errDeadlock
}

// IsQueryCanceled reports whether err was caused by a user cancellation
// of a query (ORA-01013).
func IsQueryCanceled(err error) bool {
	var oraErr *network.OracleError
	return errors.As(err, &oraErr) && oraErr.ErrCode == errQueryCanceled
}
