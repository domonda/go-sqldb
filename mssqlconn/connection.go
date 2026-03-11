package mssqlconn

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/microsoft/go-mssqldb"

	"github.com/domonda/go-sqldb"
)

const Driver = "sqlserver"

// Connect establishes a new [sqldb.Connection] using the passed config
// and github.com/microsoft/go-mssqldb as driver implementation.
// The connection is pinged with the passed context and only returned
// when there was no error from the ping.
func Connect(ctx context.Context, config *sqldb.ConnConfig) (sqldb.ConnectionQueryFormatter, error) {
	if config.Driver != Driver {
		return nil, fmt.Errorf(`invalid driver %q, expected %q`, config.Driver, Driver)
	}
	err := config.Validate()
	if err != nil {
		return nil, err
	}

	dsn := formatDSN(config)
	db, err := sql.Open(Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %w", err)
	}
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	err = db.PingContext(ctx)
	if err != nil {
		if e := db.Close(); e != nil {
			err = fmt.Errorf("%w, then %w", err, e)
		}
		return nil, err
	}
	return sqldb.NewGenericConn(
		db,
		config,
		sql.LevelReadCommitted,
		QueryFormatter{},
		wrapKnownErrors,
	), nil
}

// formatDSN converts a sqldb.ConnConfig to a SQL Server connection URL.
// The go-mssqldb driver expects the database as a query parameter,
// not in the URL path (which is used for instance names).
func formatDSN(config *sqldb.ConnConfig) string {
	query := make(url.Values)
	query.Set("database", config.Database)
	for key, val := range config.Extra {
		query.Set(key, val)
	}
	u := &url.URL{
		Scheme:   Driver,
		Host:     config.Host,
		RawQuery: query.Encode(),
	}
	if config.Port != 0 {
		u.Host = fmt.Sprintf("%s:%d", config.Host, config.Port)
	}
	if config.User != "" {
		u.User = url.UserPassword(config.User, config.Password)
	}
	return u.String()
}

// MustConnect is like Connect but panics on error.
func MustConnect(ctx context.Context, config *sqldb.ConnConfig) sqldb.ConnectionQueryFormatter {
	conn, err := Connect(ctx, config)
	if err != nil {
		panic(err)
	}
	return conn
}
