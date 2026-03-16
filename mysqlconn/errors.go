package mysqlconn

import (
	"errors"
	"slices"
	"strings"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/domonda/go-sqldb"
)

// MySQL error numbers for constraint violations.
// https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html
const (
	errBadNullError     = 1048 // Column '%s' cannot be null
	errDupEntry         = 1062 // Duplicate entry '%s' for key '%s'
	errNoReferencedRow  = 1216 // FK child-side insert/update failed (old)
	errRowIsReferenced  = 1217 // FK parent-side delete/update failed (old)
	errDeadlock         = 1213 // Deadlock found when trying to get lock
	errSignal           = 1644 // Unhandled user-defined exception (SIGNAL)
	errRowIsReferenced2 = 1451 // FK parent-side delete/update failed
	errNoReferencedRow2 = 1452 // FK child-side insert/update failed
	errCheckViolated    = 3819 // Check constraint '%s' is violated
)

func wrapKnownErrors(err error) error {
	if err == nil {
		return nil
	}
	var e *mysqldriver.MySQLError
	if !errors.As(err, &e) {
		return err
	}
	msg := e.Message
	switch e.Number {
	case errDeadlock:
		return errors.Join(sqldb.ErrDeadlock, err)
	case errSignal:
		return errors.Join(sqldb.ErrRaisedException{Message: msg}, err)
	case errBadNullError:
		// "Column 'col_name' cannot be null"
		return errors.Join(sqldb.ErrNotNullViolation{Constraint: nthSingleQuoted(msg, 0)}, err)
	case errDupEntry:
		// "Duplicate entry 'value' for key 'table.key_name'"
		key := nthSingleQuoted(msg, 1)
		if dot := strings.LastIndex(key, "."); dot != -1 {
			key = key[dot+1:]
		}
		return errors.Join(sqldb.ErrUniqueViolation{Constraint: key}, err)
	case errNoReferencedRow, errNoReferencedRow2, errRowIsReferenced, errRowIsReferenced2:
		// "... CONSTRAINT `fk_name` FOREIGN KEY ..."
		return errors.Join(sqldb.ErrForeignKeyViolation{Constraint: nthBacktickQuoted(msg, 2)}, err)
	case errCheckViolated:
		// "Check constraint 'chk_name' is violated."
		return errors.Join(sqldb.ErrCheckViolation{Constraint: nthSingleQuoted(msg, 0)}, err)
	}
	return err
}

// nthSingleQuoted returns the content of the nth (0-indexed) single-quoted segment in s.
func nthSingleQuoted(s string, n int) string {
	for i := 0; i <= n; i++ {
		start := strings.Index(s, "'")
		if start == -1 {
			return ""
		}
		end := strings.Index(s[start+1:], "'")
		if end == -1 {
			return ""
		}
		if i == n {
			return s[start+1 : start+1+end]
		}
		s = s[start+1+end+1:]
	}
	return ""
}

// nthBacktickQuoted returns the content of the nth (0-indexed) backtick-quoted segment in s.
func nthBacktickQuoted(s string, n int) string {
	for i := 0; i <= n; i++ {
		start := strings.Index(s, "`")
		if start == -1 {
			return ""
		}
		end := strings.Index(s[start+1:], "`")
		if end == -1 {
			return ""
		}
		if i == n {
			return s[start+1 : start+1+end]
		}
		s = s[start+1+end+1:]
	}
	return ""
}

// IsDeadlockDetected reports whether err was caused by a deadlock
// (MySQL error 1213).
func IsDeadlockDetected(err error) bool {
	var e *mysqldriver.MySQLError
	return errors.As(err, &e) && e.Number == errDeadlock
}

// IsNotNullViolation reports whether err was caused by inserting a NULL
// into a column that does not allow nulls (MySQL error 1048).
func IsNotNullViolation(err error) bool {
	var e *mysqldriver.MySQLError
	return errors.As(err, &e) && e.Number == errBadNullError
}

// IsUniqueViolation reports whether err was caused by a duplicate key
// violating a unique index or primary key (MySQL error 1062).
func IsUniqueViolation(err error) bool {
	var e *mysqldriver.MySQLError
	return errors.As(err, &e) && e.Number == errDupEntry
}

// IsForeignKeyViolation reports whether err was caused by a foreign key
// constraint violation (MySQL errors 1216, 1217, 1451, 1452).
// If violatedConstraints is non-empty, only returns true when the
// violated constraint name matches one of the provided names.
func IsForeignKeyViolation(err error, violatedConstraints ...string) bool {
	var e *mysqldriver.MySQLError
	if !errors.As(err, &e) {
		return false
	}
	switch e.Number {
	case errNoReferencedRow, errNoReferencedRow2, errRowIsReferenced, errRowIsReferenced2:
		if len(violatedConstraints) == 0 {
			return true
		}
		return slices.Contains(violatedConstraints, nthBacktickQuoted(e.Message, 2))
	}
	return false
}

// IsCheckViolation reports whether err was caused by a CHECK constraint
// violation (MySQL error 3819).
func IsCheckViolation(err error) bool {
	var e *mysqldriver.MySQLError
	return errors.As(err, &e) && e.Number == errCheckViolated
}
