package pqconn

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"regexp"

	"github.com/lib/pq"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

const argFmt = "$%d"

var columnNameRegex = regexp.MustCompile(`^[a-zA-Z_][0-9a-zA-Z_]{0,58}$`)

func validateColumnName(name string) error {
	if !columnNameRegex.MatchString(name) {
		return fmt.Errorf("invalid Postgres column name: %q", name)
	}
	return nil
}

type valueConverter struct{}

func (valueConverter) ConvertValue(v any) (driver.Value, error) {
	if valuer, ok := v.(driver.Valuer); ok {
		return valuer.Value()
	}
	if impl.IsSliceOrArray(v) {
		return pq.Array(v).Value()
	}
	return v, nil
}

// New creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/lib/pq as driver implementation.
// The connection is pinged with the passed context
// and only returned when there was no error from the ping.
func New(ctx context.Context, config *sqldb.Config) (sqldb.Connection, error) {
	if config.Driver != "postgres" {
		return nil, fmt.Errorf(`invalid driver %q, pqconn expects "postgres"`, config.Driver)
	}
	config.DefaultIsolationLevel = sql.LevelReadCommitted // postgres default

	db, err := config.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return impl.NewGenericConnection(
		ctx,
		db,
		config,
		Listener,
		sqldb.DefaultStructFieldMapping,
		validateColumnName,
		valueConverter{},
		argFmt,
	), nil
}

// MustNew creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/lib/pq as driver implementation.
// The connection is pinged with the passed context,
// and only returned when there was no error from the ping.
// Errors are paniced.
func MustNew(ctx context.Context, config *sqldb.Config) sqldb.Connection {
	conn, err := New(ctx, config)
	if err != nil {
		panic(err)
	}
	return conn
}
