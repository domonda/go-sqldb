package sqldb

import (
	"log"
	"os"
)

// Logger has a Printf method used for logging
// information that could not be returned by
// any of the package functions directly.
type Logger interface {
	Printf(format string, v ...any)
}

// ErrLogger is a global configuration variable
// that will be used to log errors
// that could not be returned by
// any of the package functions directly.
var ErrLogger Logger = log.New(os.Stderr, "sqldb: ", log.LstdFlags)
