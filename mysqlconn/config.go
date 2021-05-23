package mysqlconn

import "github.com/go-sql-driver/mysql"

const argFmt = "?"

type Config = mysql.Config

// NewConfig creates a new Config and sets default values.
func NewConfig() *Config {
	return mysql.NewConfig()
}
