package mssqlconn

import (
	"errors"
	"slices"
	"strings"

	mssql "github.com/microsoft/go-mssqldb"

	"github.com/domonda/go-sqldb"
)

// SQL Server error numbers for constraint violations.
// https://learn.microsoft.com/en-us/sql/relational-databases/errors-events/database-engine-events-and-errors
const (
	errCannotInsertNull   = 515   // Cannot insert NULL into column
	errConstraintConflict = 547   // Statement conflicted with FK or CHECK constraint
	errDeadlock           = 1205  // Deadlock detected
	errDupKeyRow          = 2601  // Cannot insert duplicate key row (unique index)
	errRaisedException    = 50000 // User-defined error from RAISERROR/THROW
	errUniqueConstraint   = 2627  // Violation of UNIQUE KEY or PRIMARY KEY constraint
)

func wrapKnownErrors(err error) error {
	if err == nil {
		return nil
	}
	var e mssql.Error
	if !errors.As(err, &e) {
		return err
	}
	msg := e.Message
	switch e.Number {
	case errCannotInsertNull:
		// "Cannot insert the value NULL into column 'col_name', table '...'"
		return errors.Join(sqldb.ErrNotNullViolation{Constraint: firstSingleQuoted(msg)}, err)
	case errConstraintConflict:
		// "The [op] statement conflicted with the [FOREIGN KEY|CHECK] constraint "name"."
		// SQL Server uses double quotes for the constraint name in this message.
		constraint := firstDoubleQuoted(msg)
		if strings.Contains(msg, "FOREIGN KEY") {
			return errors.Join(sqldb.ErrForeignKeyViolation{Constraint: constraint}, err)
		}
		return errors.Join(sqldb.ErrCheckViolation{Constraint: constraint}, err)
	case errDeadlock:
		return errors.Join(sqldb.ErrDeadlock, err)
	case errRaisedException:
		return errors.Join(sqldb.ErrRaisedException{Message: msg}, err)
	case errDupKeyRow:
		// "Cannot insert duplicate key row in object 'schema.table' with unique index 'ix_name'."
		return errors.Join(sqldb.ErrUniqueViolation{Constraint: nthSingleQuoted(msg, 1)}, err)
	case errUniqueConstraint:
		// "Violation of UNIQUE KEY constraint 'ux_name'. Cannot insert duplicate key in object '...'."
		return errors.Join(sqldb.ErrUniqueViolation{Constraint: firstSingleQuoted(msg)}, err)
	}
	return err
}

func firstSingleQuoted(s string) string {
	return nthSingleQuoted(s, 0)
}

func firstDoubleQuoted(s string) string {
	start := strings.Index(s, `"`)
	if start == -1 {
		return ""
	}
	end := strings.Index(s[start+1:], `"`)
	if end == -1 {
		return ""
	}
	return s[start+1 : start+1+end]
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

// IsNotNullViolation reports whether err was caused by inserting a NULL
// into a column that does not allow nulls (SQL Server error 515).
func IsNotNullViolation(err error) bool {
	var e mssql.Error
	return errors.As(err, &e) && e.Number == errCannotInsertNull
}

// IsUniqueViolation reports whether err was caused by a duplicate key
// violating a unique index or primary key (SQL Server errors 2601, 2627).
func IsUniqueViolation(err error) bool {
	var e mssql.Error
	return errors.As(err, &e) && (e.Number == errDupKeyRow || e.Number == errUniqueConstraint)
}

// IsForeignKeyViolation reports whether err was caused by a foreign key
// constraint violation (SQL Server error 547 with "FOREIGN KEY" in the message).
// If violatedConstraints is non-empty, only returns true when the
// violated constraint name matches one of the provided names.
func IsForeignKeyViolation(err error, violatedConstraints ...string) bool {
	var e mssql.Error
	if !errors.As(err, &e) || e.Number != errConstraintConflict {
		return false
	}
	if !strings.Contains(e.Message, "FOREIGN KEY") {
		return false
	}
	if len(violatedConstraints) == 0 {
		return true
	}
	return slices.Contains(violatedConstraints, firstDoubleQuoted(e.Message))
}

// IsCheckViolation reports whether err was caused by a CHECK constraint
// violation (SQL Server error 547 with "CHECK" in the message).
func IsCheckViolation(err error) bool {
	var e mssql.Error
	return errors.As(err, &e) && e.Number == errConstraintConflict &&
		strings.Contains(e.Message, "CHECK")
}

// IsDeadlockDetected reports whether err was caused by a deadlock
// (SQL Server error 1205).
func IsDeadlockDetected(err error) bool {
	var e mssql.Error
	return errors.As(err, &e) && e.Number == errDeadlock
}
