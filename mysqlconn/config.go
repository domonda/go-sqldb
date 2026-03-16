package mysqlconn

import "github.com/go-sql-driver/mysql"

// Driver is the database/sql driver name used for MySQL/MariaDB connections.
const (
	Driver = "mysql"
)

// Config is an alias for the go-sql-driver/mysql [mysql.Config] type.
type Config = mysql.Config

// NewConfig creates a new Config and sets default values.
func NewConfig() *Config {
	return mysql.NewConfig()
}
