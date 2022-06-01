package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"
)

// Config for a connection.
// For tips see https://www.alexedwards.net/blog/configuring-sqldb
type Config struct {
	Driver                string             `json:"driver"`
	Host                  string             `json:"host"`
	Port                  uint16             `json:"port,omitempty"`
	User                  string             `json:"user,omitempty"`
	Password              string             `json:"password,omitempty"`
	Database              string             `json:"database"`
	Extra                 map[string]string  `json:"misc,omitempty"`
	MaxOpenConns          int                `json:"maxOpenConns,omitempty"`
	MaxIdleConns          int                `json:"maxIdleConns,omitempty"`
	ConnMaxLifetime       time.Duration      `json:"connMaxLifetime,omitempty"`
	DefaultIsolationLevel sql.IsolationLevel `json:"-"`
	Err                   error              `json:"-"`
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

// ConnectURL for connecting to a database
func (c *Config) ConnectURL() string {
	extra := make(url.Values)
	for key, val := range c.Extra {
		extra.Add(key, val)
	}
	u := url.URL{
		Scheme:   c.Driver,
		Host:     c.Host,
		Path:     c.Database,
		RawQuery: extra.Encode(),
	}
	if c.Port != 0 {
		u.Host = fmt.Sprintf("%s:%d", c.Host, c.Port)
	}
	if c.User != "" {
		u.User = url.UserPassword(c.User, c.Password)
	}
	return u.String()
}

// Connect opens a new sql.DB connection,
// sets all Config values and performs a ping with ctx.
// The sql.DB will be returned if the ping was successful.
func (c *Config) Connect(ctx context.Context) (*sql.DB, error) {
	err := c.Validate()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(c.Driver, c.ConnectURL())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(c.MaxOpenConns)
	db.SetMaxIdleConns(c.MaxIdleConns)
	db.SetConnMaxLifetime(c.ConnMaxLifetime)
	err = db.PingContext(ctx)
	if err != nil {
		e := db.Close()
		if e != nil {
			err = fmt.Errorf("%w, then %s", err, e)
		}
		return nil, err
	}
	return db, nil
}
