package mockconn

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

// MockStructRows implements the impl.Rows interface for testing purposes.
// It is not safe for concurrent use.
type MockStructRows[S any] struct {
	columns []string
	rows    []S

	current int
	closed  bool
	err     error
}

var _ impl.Rows = new(MockStructRows[struct {
	ID int `db:"id"`
}])

// NewMockStructRows returns a new MockStructRows with column names
// derived from the struct fields of S using the given namer
// (or sqldb.DefaultStructFieldMapping if nil) and the given rows as data.
// Panics if S is not a struct or has no mapped columns.
func NewMockStructRows[S any](namer sqldb.StructFieldMapper, rows ...S) *MockStructRows[S] {
	t := reflect.TypeFor[S]()
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("mockconn.NewMockStructRows: type parameter S must be a struct, got %s", t))
	}
	if namer == nil {
		namer = sqldb.DefaultStructFieldMapping
	}
	var zero S
	columns, _, _ := impl.ReflectStructValues(reflect.ValueOf(&zero).Elem(), namer, nil)
	if len(columns) == 0 {
		panic(fmt.Sprintf("mockconn.NewMockStructRows: struct %s has no mapped columns", t))
	}
	return &MockStructRows[S]{
		columns: columns,
		rows:    rows,
		current: -1,
	}
}

func (m *MockStructRows[S]) Columns() ([]string, error) {
	return m.columns, nil
}

func (m *MockStructRows[S]) Next() bool {
	if m.closed || m.err != nil {
		return false
	}
	m.current++
	return m.current < len(m.rows)
}

func (m *MockStructRows[S]) Scan(dest ...any) error {
	if m.err != nil {
		return m.err
	}
	if m.closed {
		return errors.New("sql: Rows are closed")
	}
	if m.current < 0 {
		return errors.New("sql: Scan called without calling Next")
	}
	if m.current >= len(m.rows) {
		return sql.ErrNoRows
	}
	_, _, values := impl.ReflectStructValues(reflect.ValueOf(&m.rows[m.current]).Elem(), sqldb.DefaultStructFieldMapping, nil)
	if len(dest) != len(values) {
		return fmt.Errorf("sql: expected %d destination arguments in Scan, not %d", len(values), len(dest))
	}
	for i, val := range values {
		driverVal, err := structFieldToDriverValue(val)
		if err != nil {
			return fmt.Errorf("sql: converting column index %d, name %q to driver.Value: %w", i, m.columns[i], err)
		}
		err = convertAssign(dest[i], driverVal)
		if err != nil {
			return fmt.Errorf("sql: Scan error on column index %d, name %q: %w", i, m.columns[i], err)
		}
	}
	return nil
}

func (m *MockStructRows[S]) Err() error {
	return m.err
}

func (m *MockStructRows[S]) Close() error {
	m.closed = true
	return nil
}

// structFieldToDriverValue converts a struct field value to a driver.Value.
func structFieldToDriverValue(val any) (driver.Value, error) {
	if val == nil {
		return nil, nil
	}
	if d, ok := val.(decimalDecompose); ok {
		return d, nil
	}
	return driver.DefaultParameterConverter.ConvertValue(val)
}
