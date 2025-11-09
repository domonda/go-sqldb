package sqliteconn

import (
	"context"

	"github.com/domonda/go-sqldb"
	"zombiezen.com/go/sqlite"
)

type statement struct {
	query string
	stmt  *sqlite.Stmt
	conn  *sqlite.Conn
}

func (s *statement) PreparedQuery() string {
	return s.query
}

func (s *statement) Exec(ctx context.Context, args ...any) error {
	// Reset the statement for reuse
	if err := s.stmt.Reset(); err != nil {
		return wrapKnownErrors(err)
	}
	if err := s.stmt.ClearBindings(); err != nil {
		return wrapKnownErrors(err)
	}

	// Bind arguments
	if err := bindArgs(s.stmt, args); err != nil {
		return wrapKnownErrors(err)
	}

	// Execute the statement
	_, err := s.stmt.Step()
	if err != nil {
		return wrapKnownErrors(err)
	}

	return nil
}

func (s *statement) Query(ctx context.Context, args ...any) sqldb.Rows {
	// Reset the statement for reuse
	if err := s.stmt.Reset(); err != nil {
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}
	if err := s.stmt.ClearBindings(); err != nil {
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}

	// Bind arguments
	if err := bindArgs(s.stmt, args); err != nil {
		return sqldb.NewErrRows(wrapKnownErrors(err))
	}

	// Note: We create a pseudo-rows that wraps the statement
	// The statement should not be finalized until the rows are closed
	return &rows{
		stmt:              s.stmt,
		conn:              s.conn,
		hasRow:            false,
		shouldFinalizeStmt: false, // Prepared statement owns the stmt
	}
}

func (s *statement) Close() error {
	return s.stmt.Finalize()
}
