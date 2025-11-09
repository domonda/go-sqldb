package sqliteconn

import (
	"context"
	"errors"
	"strings"

	"github.com/domonda/go-sqldb"
	"zombiezen.com/go/sqlite"
)

func wrapKnownErrors(err error) error {
	if err == nil {
		return nil
	}

	// Handle context cancellation
	if errors.Is(err, context.Canceled) {
		return err
	}

	// Get the SQLite error code
	code := sqlite.ErrCode(err)
	msg := strings.ToLower(err.Error())

	// Check for constraint violations
	primary := code.ToPrimary()

	if primary == sqlite.ResultConstraint {
		// Parse constraint violation types from error message
		if strings.Contains(msg, "foreign key") {
			return errors.Join(sqldb.ErrForeignKeyViolation{Constraint: extractConstraint(msg)}, err)
		}
		if strings.Contains(msg, "unique") {
			return errors.Join(sqldb.ErrUniqueViolation{Constraint: extractConstraint(msg)}, err)
		}
		if strings.Contains(msg, "not null") {
			return errors.Join(sqldb.ErrNotNullViolation{Constraint: extractConstraint(msg)}, err)
		}
		if strings.Contains(msg, "check") {
			return errors.Join(sqldb.ErrCheckViolation{Constraint: extractConstraint(msg)}, err)
		}
		// Generic constraint violation
		return errors.Join(sqldb.ErrIntegrityConstraintViolation{Constraint: extractConstraint(msg)}, err)
	}

	// SQLITE_INTERRUPT
	if primary == sqlite.ResultInterrupt {
		return errors.Join(context.Canceled, err)
	}

	// SQLITE_LOCKED or SQLITE_BUSY
	if primary == sqlite.ResultLocked || primary == sqlite.ResultBusy {
		return err
	}

	// SQLITE_READONLY
	if primary == sqlite.ResultReadOnly {
		return errors.Join(errors.New("database is read-only"), err)
	}

	return err
}

// extractConstraint attempts to extract the constraint name from an SQLite error message.
// SQLite error messages typically include the constraint name in the format:
// "UNIQUE constraint failed: table.column" or "FOREIGN KEY constraint failed"
func extractConstraint(msg string) string {
	// Try to extract constraint name from common patterns
	if idx := strings.Index(msg, "constraint failed:"); idx != -1 {
		constraint := strings.TrimSpace(msg[idx+len("constraint failed:"):])
		// Remove any trailing text after the constraint (but not dots within table.column)
		// Split on whitespace, comma, semicolon, or other separators
		if endIdx := strings.IndexAny(constraint, " ,;"); endIdx != -1 {
			constraint = constraint[:endIdx]
		}
		return constraint
	}
	return ""
}

// IsConstraintViolation checks if the error is a constraint violation.
func IsConstraintViolation(err error) bool {
	if err == nil {
		return false
	}
	code := sqlite.ErrCode(err)
	return code.ToPrimary() == sqlite.ResultConstraint
}

// IsForeignKeyViolation checks if the error is a foreign key constraint violation.
func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return IsConstraintViolation(err) && strings.Contains(msg, "foreign key")
}

// IsUniqueViolation checks if the error is a unique constraint violation.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return IsConstraintViolation(err) && strings.Contains(msg, "unique")
}

// IsNotNullViolation checks if the error is a not null constraint violation.
func IsNotNullViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return IsConstraintViolation(err) && strings.Contains(msg, "not null")
}

// IsCheckViolation checks if the error is a check constraint violation.
func IsCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return IsConstraintViolation(err) && strings.Contains(msg, "check")
}

// IsDatabaseLocked checks if the error indicates the database is locked.
func IsDatabaseLocked(err error) bool {
	if err == nil {
		return false
	}
	code := sqlite.ErrCode(err).ToPrimary()
	return code == sqlite.ResultLocked || code == sqlite.ResultBusy
}
