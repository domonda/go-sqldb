package sqldb

import (
	"context"
	"errors"
)

// Information stubs for the generic / mock / error / test connection
// types. Real driver-backed connections override these in their own
// packages. genericTxWithQueryBuilder inherits from genericTx through
// embedding.
//
// Each stub returns errors.ErrUnsupported (or the wrapped error for
// ErrConn) so that callers can detect "this connection has no catalog
// support" via errors.Is.

// --- genericConn ---

func (*genericConn) Schemas(ctx context.Context) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*genericConn) CurrentSchema(ctx context.Context) (string, error) {
	return "", errors.ErrUnsupported
}

func (*genericConn) Tables(ctx context.Context, schema ...string) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*genericConn) TableExists(ctx context.Context, table string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (*genericConn) Views(ctx context.Context, schema ...string) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*genericConn) ViewExists(ctx context.Context, view string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (*genericConn) Columns(ctx context.Context, tableOrView string) ([]ColumnInfo, error) {
	return nil, errors.ErrUnsupported
}

func (*genericConn) ColumnExists(ctx context.Context, tableOrView, column string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (*genericConn) PrimaryKey(ctx context.Context, table string) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*genericConn) ForeignKeys(ctx context.Context, table string) ([]ForeignKeyInfo, error) {
	return nil, errors.ErrUnsupported
}

func (*genericConn) Routines(ctx context.Context, schema ...string) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*genericConn) RoutineExists(ctx context.Context, routine string) (bool, error) {
	return false, errors.ErrUnsupported
}

// --- genericTx ---

func (*genericTx) Schemas(ctx context.Context) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*genericTx) CurrentSchema(ctx context.Context) (string, error) {
	return "", errors.ErrUnsupported
}

func (*genericTx) Tables(ctx context.Context, schema ...string) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*genericTx) TableExists(ctx context.Context, table string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (*genericTx) Views(ctx context.Context, schema ...string) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*genericTx) ViewExists(ctx context.Context, view string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (*genericTx) Columns(ctx context.Context, tableOrView string) ([]ColumnInfo, error) {
	return nil, errors.ErrUnsupported
}

func (*genericTx) ColumnExists(ctx context.Context, tableOrView, column string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (*genericTx) PrimaryKey(ctx context.Context, table string) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*genericTx) ForeignKeys(ctx context.Context, table string) ([]ForeignKeyInfo, error) {
	return nil, errors.ErrUnsupported
}

func (*genericTx) Routines(ctx context.Context, schema ...string) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*genericTx) RoutineExists(ctx context.Context, routine string) (bool, error) {
	return false, errors.ErrUnsupported
}

// --- ErrConn ---

func (e ErrConn) Schemas(ctx context.Context) ([]string, error)         { return nil, e.Err }
func (e ErrConn) CurrentSchema(ctx context.Context) (string, error)     { return "", e.Err }
func (e ErrConn) Tables(ctx context.Context, _ ...string) ([]string, error) {
	return nil, e.Err
}
func (e ErrConn) TableExists(ctx context.Context, _ string) (bool, error) {
	return false, e.Err
}
func (e ErrConn) Views(ctx context.Context, _ ...string) ([]string, error) {
	return nil, e.Err
}
func (e ErrConn) ViewExists(ctx context.Context, _ string) (bool, error) {
	return false, e.Err
}
func (e ErrConn) Columns(ctx context.Context, _ string) ([]ColumnInfo, error) {
	return nil, e.Err
}
func (e ErrConn) ColumnExists(ctx context.Context, _, _ string) (bool, error) {
	return false, e.Err
}
func (e ErrConn) PrimaryKey(ctx context.Context, _ string) ([]string, error) {
	return nil, e.Err
}
func (e ErrConn) ForeignKeys(ctx context.Context, _ string) ([]ForeignKeyInfo, error) {
	return nil, e.Err
}
func (e ErrConn) Routines(ctx context.Context, _ ...string) ([]string, error) {
	return nil, e.Err
}
func (e ErrConn) RoutineExists(ctx context.Context, _ string) (bool, error) {
	return false, e.Err
}

// --- MockConn ---
//
// Each method records the call in c.Recordings.Information (under the
// MockConn mutex) and then delegates to the corresponding MockXxx
// field if set. When the field is nil the method returns
// errors.ErrUnsupported, matching every other stub Information
// implementation in this package, so tests can detect "no
// introspection support" via errors.Is(err, errors.ErrUnsupported).

func (c *MockConn) recordInformationCall(method string, args ...any) {
	c.mtx.Lock()
	c.Recordings.Information = append(c.Recordings.Information, InformationCall{
		Method: method,
		Args:   args,
	})
	c.mtx.Unlock()
}

func (c *MockConn) Schemas(ctx context.Context) ([]string, error) {
	c.recordInformationCall("Schemas")
	if c.MockSchemas != nil {
		return c.MockSchemas(ctx)
	}
	return nil, errors.ErrUnsupported
}

func (c *MockConn) CurrentSchema(ctx context.Context) (string, error) {
	c.recordInformationCall("CurrentSchema")
	if c.MockCurrentSchema != nil {
		return c.MockCurrentSchema(ctx)
	}
	return "", errors.ErrUnsupported
}

func (c *MockConn) Tables(ctx context.Context, schema ...string) ([]string, error) {
	c.recordInformationCall("Tables", schema)
	if c.MockTables != nil {
		return c.MockTables(ctx, schema...)
	}
	return nil, errors.ErrUnsupported
}

func (c *MockConn) TableExists(ctx context.Context, table string) (bool, error) {
	c.recordInformationCall("TableExists", table)
	if c.MockTableExists != nil {
		return c.MockTableExists(ctx, table)
	}
	return false, errors.ErrUnsupported
}

func (c *MockConn) Views(ctx context.Context, schema ...string) ([]string, error) {
	c.recordInformationCall("Views", schema)
	if c.MockViews != nil {
		return c.MockViews(ctx, schema...)
	}
	return nil, errors.ErrUnsupported
}

func (c *MockConn) ViewExists(ctx context.Context, view string) (bool, error) {
	c.recordInformationCall("ViewExists", view)
	if c.MockViewExists != nil {
		return c.MockViewExists(ctx, view)
	}
	return false, errors.ErrUnsupported
}

func (c *MockConn) Columns(ctx context.Context, tableOrView string) ([]ColumnInfo, error) {
	c.recordInformationCall("Columns", tableOrView)
	if c.MockColumns != nil {
		return c.MockColumns(ctx, tableOrView)
	}
	return nil, errors.ErrUnsupported
}

func (c *MockConn) ColumnExists(ctx context.Context, tableOrView, column string) (bool, error) {
	c.recordInformationCall("ColumnExists", tableOrView, column)
	if c.MockColumnExists != nil {
		return c.MockColumnExists(ctx, tableOrView, column)
	}
	return false, errors.ErrUnsupported
}

func (c *MockConn) PrimaryKey(ctx context.Context, table string) ([]string, error) {
	c.recordInformationCall("PrimaryKey", table)
	if c.MockPrimaryKey != nil {
		return c.MockPrimaryKey(ctx, table)
	}
	return nil, errors.ErrUnsupported
}

func (c *MockConn) ForeignKeys(ctx context.Context, table string) ([]ForeignKeyInfo, error) {
	c.recordInformationCall("ForeignKeys", table)
	if c.MockForeignKeys != nil {
		return c.MockForeignKeys(ctx, table)
	}
	return nil, errors.ErrUnsupported
}

func (c *MockConn) Routines(ctx context.Context, schema ...string) ([]string, error) {
	c.recordInformationCall("Routines", schema)
	if c.MockRoutines != nil {
		return c.MockRoutines(ctx, schema...)
	}
	return nil, errors.ErrUnsupported
}

func (c *MockConn) RoutineExists(ctx context.Context, routine string) (bool, error) {
	c.recordInformationCall("RoutineExists", routine)
	if c.MockRoutineExists != nil {
		return c.MockRoutineExists(ctx, routine)
	}
	return false, errors.ErrUnsupported
}

// --- nonConnForTest ---

func (*nonConnForTest) Schemas(ctx context.Context) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*nonConnForTest) CurrentSchema(ctx context.Context) (string, error) {
	return "", errors.ErrUnsupported
}

func (*nonConnForTest) Tables(ctx context.Context, schema ...string) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*nonConnForTest) TableExists(ctx context.Context, table string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (*nonConnForTest) Views(ctx context.Context, schema ...string) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*nonConnForTest) ViewExists(ctx context.Context, view string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (*nonConnForTest) Columns(ctx context.Context, tableOrView string) ([]ColumnInfo, error) {
	return nil, errors.ErrUnsupported
}

func (*nonConnForTest) ColumnExists(ctx context.Context, tableOrView, column string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (*nonConnForTest) PrimaryKey(ctx context.Context, table string) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*nonConnForTest) ForeignKeys(ctx context.Context, table string) ([]ForeignKeyInfo, error) {
	return nil, errors.ErrUnsupported
}

func (*nonConnForTest) Routines(ctx context.Context, schema ...string) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (*nonConnForTest) RoutineExists(ctx context.Context, routine string) (bool, error) {
	return false, errors.ErrUnsupported
}
