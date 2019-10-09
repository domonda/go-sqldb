package sqldb

import (
	"log"
	"os"
)

type Logger interface {
	Printf(format string, v ...interface{})
}

var ErrLogger Logger = log.New(os.Stderr, "sqldb", log.LstdFlags)
