package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Config for a connection.
// For tips see https://www.alexedwards.net/blog/configuring-sqldb
type Config struct {
	Driver   string            `json:"driver"`
	Host     string            `json:"host"`
	Port     uint16            `json:"port,omitempty"`
	User     string            `json:"user,omitempty"`
	Password string            `json:"password,omitempty"`
	Database string            `json:"database"`
	Extra    map[string]string `json:"misc,omitempty"`

	// ReadOnly sets the database connection to read-only mode
	// if supported by the database connection.
	//
	// The default is false.
	ReadOnly bool `json:"readOnly,omitempty"`

	// MaxOpenConns sets the maximum number of open connections to the database.
	//
	// If MaxIdleConns is greater than 0 and the new MaxOpenConns is less than
	// MaxIdleConns, then MaxIdleConns will be reduced to match the new
	// MaxOpenConns limit.
	//
	// If MaxOpenConns <= 0, then there is no limit on the number of open connections.
	// The default is 0 (unlimited).
	MaxOpenConns int `json:"maxOpenConns,omitempty"`

	// MaxIdleConns sets the maximum number of connections in the idle
	// connection pool.
	//
	// If MaxOpenConns is greater than 0 but less than the new MaxIdleConns,
	// then the new MaxIdleConns will be reduced to match the MaxOpenConns limit.
	//
	// If MaxIdleConns <= 0, no idle connections are retained.
	//
	// The default max idle connections is currently 2. This may change in
	// a future release.
	MaxIdleConns int `json:"maxIdleConns,omitempty"`

	// ConnMaxLifetime sets the maximum amount of time a connection may be reused.
	//
	// Expired connections may be closed lazily before reuse.
	//
	// If ConnMaxLifetime <= 0, connections are not closed due to a connection's age.
	ConnMaxLifetime time.Duration `json:"connMaxLifetime,omitempty"`

	DefaultIsolationLevel sql.IsolationLevel `json:"-"`
	Err                   error              `json:"-"`
}

// ParseConfig parses a connection URI string and returns a Config.
// The URI must be in the format:
//
//	driver://user:password@host:port/database?key=value&key2=value2
//
// For example:
//
//	postgres://user:password@localhost:5432/database?sslmode=disable
//
// See also [Config.String]
func ParseConfig(uri string) (*Config, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	port, err := strconv.ParseUint(parsed.Port(), 10, 16)
	if err != nil {
		return nil, err
	}
	password, _ := parsed.User.Password()
	config := &Config{
		Driver:   parsed.Scheme,
		Host:     parsed.Hostname(),
		Port:     uint16(port),
		User:     parsed.User.Username(),
		Password: password,
		Database: strings.TrimPrefix(parsed.Path, "/"),
	}
	if vals := parsed.Query(); len(vals) > 0 {
		config.Extra = make(map[string]string)
		for key, val := range vals {
			config.Extra[key] = val[0]
		}
	}
	return config, nil
}

// Validate returns Config.Err if it is not nil
// or an error if the Config does not have
// a Driver, Host, or Database.
func (c *Config) Validate() error {
	if c.Err != nil {
		return c.Err
	}
	if c.Driver == "" {
		return fmt.Errorf("missing sqldb.Config.Driver")
	}
	if c.Host == "" {
		return fmt.Errorf("missing sqldb.Config.Host")
	}
	if c.Database == "" {
		return fmt.Errorf("missing sqldb.Config.Database")
	}
	return nil
}

// URL returns a [*url.URL] with the connection parameters
// for connecting to a database based on the Config.
func (c *Config) URL() *url.URL {
	extra := make(url.Values)
	for key, val := range c.Extra {
		extra.Add(key, val)
	}
	u := &url.URL{
		Scheme:   c.Driver,
		Host:     c.Host,
		Path:     "/" + c.Database,
		RawQuery: extra.Encode(),
	}
	if c.Port != 0 {
		u.Host += fmt.Sprintf(":%d", c.Port)
	}
	if c.User != "" {
		u.User = url.UserPassword(c.User, c.Password)
	}
	return u
}

// String returns the connection URI string for the Config
// without the password and implements the [fmt.Stringer] interface.
//
// To get the full connection URI including the password use [Config.URL].
//
// The returned string will not include the following fields:
//   - Password
//   - MaxOpenConns
//   - MaxIdleConns
//   - ConnMaxLifetime
//   - DefaultIsolationLevel
//   - Err
//
// See also [ParseConfig]
func (c *Config) String() string {
	uri := c.URL()
	uri.User = url.User(c.User)
	return uri.String()
}

// Connect opens a new [sql.DB] connection,
// sets all Config values and performs a ping with ctx.
// The [sql.DB] will be returned if the ping was successful.
func (c *Config) Connect(ctx context.Context) (*sql.DB, error) {
	err := c.Validate()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(c.Driver, c.URL().String())
	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %w", err)
	}
	db.SetMaxOpenConns(c.MaxOpenConns)
	db.SetMaxIdleConns(c.MaxIdleConns)
	db.SetConnMaxLifetime(c.ConnMaxLifetime)
	err = db.PingContext(ctx)
	if err != nil {
		if e := db.Close(); e != nil {
			err = fmt.Errorf("%w, then %w", err, e)
		}
		return nil, err
	}
	return db, nil
}
