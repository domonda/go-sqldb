package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Config for a connection.
// For tips see https://www.alexedwards.net/blog/configuring-sqldb
type Config struct {
	Driver          string            `json:"driver"`
	Host            string            `json:"host"`
	Port            uint16            `json:"port,omitempty"`
	User            string            `json:"user,omitempty"`
	Password        string            `json:"password,omitempty"`
	Database        string            `json:"database"`
	Extra           map[string]string `json:"misc,omitempty"`
	MaxOpenConns    int               `json:"maxOpenConns,omitempty"`
	MaxIdleConns    int               `json:"maxIdleConns,omitempty"`
	ConnMaxLifetime time.Duration     `json:"connMaxLifetime,omitempty"`

	// ValidateColumnName returns an error
	// if the passed name is not valid for a
	// column of the connection's database.
	ValidateColumnName func(name string) error `json:"-"`

	// ParamPlaceholder returns a parameter value placeholder
	// for the parameter with the passed zero based index
	// specific to the database type of the connection.
	ParamPlaceholderFormatter `json:"-"`

	DefaultIsolationLevel sql.IsolationLevel `json:"-"`

	// Err will be returned from Connection.Err()
	Err error `json:"-"`
}

// func (c *DBConnection) ValidateColumnName(name string) error {
// 	if name == "" {
// 		return errors.New("empty column name")
// 	}
// 	return nil
// }

// func (c *DBConnection) ParamPlaceholder(index int) string {
// 	return fmt.Sprintf(":%d", index+1)
// }

// Validate returns Config.Err if it is not nil
// or an error if the Config does not have
// a Driver, Host, or Database.
func (c *Config) Validate() error {
	if c.Err != nil {
		return c.Err
	}
	if c.ValidateColumnName == nil {
		return errors.New("missing sqldb.Config.ValidateColumnName")
	}
	if c.ParamPlaceholderFormatter == nil {
		return errors.New("missing sqldb.Config.ParamPlaceholderFormatter")
	}
	if c.Driver == "" {
		return errors.New("missing sqldb.Config.Driver")
	}
	if c.Host == "" {
		return errors.New("missing sqldb.Config.Host")
	}
	if c.Database == "" {
		return errors.New("missing sqldb.Config.Database")
	}
	return nil
}

// ConnectURL returns a connection URL for the Config
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
