package mysqlconn

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/domonda/go-sqldb"
)

// Connect creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/go-sql-driver/mysql as driver implementation.
// The connection is pinged with the passed context and only returned
// when there was no error from the ping.
func Connect(ctx context.Context, config *sqldb.ConnConfig) (sqldb.Connection, error) {
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
	return sqldb.NewGenericConn(db, config, sql.LevelRepeatableRead), nil
}

// formatDSN converts a sqldb.ConnConfig to a MySQL DSN string
// using the go-sql-driver/mysql Config.FormatDSN method.
func formatDSN(config *sqldb.ConnConfig) string {
	mysqlCfg := mysqldriver.NewConfig()
	mysqlCfg.User = config.User
	mysqlCfg.Passwd = config.Password
	mysqlCfg.DBName = config.Database
	mysqlCfg.Net = "tcp"
	if config.Port != 0 {
		mysqlCfg.Addr = net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port)))
	} else {
		mysqlCfg.Addr = config.Host
	}
	if len(config.Extra) > 0 {
		if mysqlCfg.Params == nil {
			mysqlCfg.Params = make(map[string]string, len(config.Extra))
		}
		for key, val := range config.Extra {
			mysqlCfg.Params[key] = val
		}
	}
	return mysqlCfg.FormatDSN()
}

// MustConnect creates a new sqldb.Connection using the passed sqldb.Config
// and github.com/go-sql-driver/mysql as driver implementation.
// The connection is pinged with the passed context and only returned
// when there was no error from the ping.
// Errors are panicked.
func MustConnect(ctx context.Context, config *sqldb.ConnConfig) sqldb.Connection {
	conn, err := Connect(ctx, config)
	if err != nil {
		panic(err)
	}
	return conn
}

// ConnectExt establishes a new sqldb.ConnExt using the passed config and structReflector.
// It wraps Connect and returns an extended connection with MySQL-specific components.
func ConnectExt(ctx context.Context, config *sqldb.ConnConfig, structReflector sqldb.StructReflector) (*sqldb.ConnExt, error) {
	conn, err := Connect(ctx, config)
	if err != nil {
		return nil, err
	}
	return NewConnExt(conn, structReflector), nil
}

// MustConnectExt is like ConnectExt but panics on error.
func MustConnectExt(ctx context.Context, config *sqldb.ConnConfig, structReflector sqldb.StructReflector) *sqldb.ConnExt {
	connExt, err := ConnectExt(ctx, config, structReflector)
	if err != nil {
		panic(err)
	}
	return connExt
}

// NewConnExt creates a new sqldb.ConnExt with MySQL-specific components.
// It combines the passed connection and struct reflector with MySQL
// specific QueryFormatter and QueryBuilder.
func NewConnExt(conn sqldb.Connection, structReflector sqldb.StructReflector) *sqldb.ConnExt {
	return sqldb.NewConnExt(
		conn,
		structReflector,
		sqldb.StdQueryFormatter{},
		sqldb.StdQueryBuilder{},
	)
}
