package db

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"time"
	"unicode/utf8"

	"github.com/domonda/go-sqldb"
)

// ReplaceErrNoRows returns the passed replacement error
// if errors.Is(err, sql.ErrNoRows),
// else err is returned unchanged.
func ReplaceErrNoRows(err, replacement error) error {
	return sqldb.ReplaceErrNoRows(err, replacement)
}

// Now returns the result of the SQL now() function
// using the sqldb.Connection from the passed context.
// This is useful to get the timestamp of a
// SQL transaction for use in Go code.
func Now(ctx context.Context) (time.Time, error) {
	var now time.Time
	err := Conn(ctx).QueryRow(`select now()`).Scan(&now)
	if err != nil {
		return time.Time{}, err
	}
	return now, nil
}

// DebugPrintConn prints a line to stderr using the passed args
// and appending the transaction state of the connection
// and the current time of the database using `select now()`
// or an error if the time could not be queried.
func DebugPrintConn(ctx context.Context, args ...interface{}) {
	opts, isTx := Conn(ctx).TransactionOptions()
	if isTx {
		args = append(args, "SQL-Transaction")
		if optsStr := TxOptionsString(opts); optsStr != "" {
			args = append(args, "Isolation", optsStr)
		}
	}
	now, err := Now(ctx)
	if err == nil {
		args = append(args, "NOW():", now)
	} else {
		args = append(args, "ERROR:", err)
	}
	fmt.Fprintln(os.Stderr, args...)
}

// TxOptionsString returns a string representing the
// passed TxOptions wich will be empty for the default options.
func TxOptionsString(opts *sql.TxOptions) string {
	switch {
	case opts == nil:
		return ""
	case opts.ReadOnly && opts.Isolation == sql.LevelDefault:
		return "Read-Only"
	case opts.ReadOnly && opts.Isolation != sql.LevelDefault:
		return "Read-Only " + opts.Isolation.String()
	case opts.Isolation != sql.LevelDefault:
		return opts.Isolation.String()
	default:
		return ""
	}
}

func PrintlnTable(rows [][]string, err error) error {
	if err != nil {
		_, e := fmt.Println(err)
		return e
	}
	return FprintTable(os.Stdout, rows, "|")
}

func FprintTable(w io.Writer, rows [][]string, columnDelimiter string) error {
	// Collect column widths
	var colRuneCount []int
	for row := range rows {
		for col, str := range rows[row] {
			count := utf8.RuneCountInString(str)
			if col >= len(colRuneCount) {
				colRuneCount = append(colRuneCount, count)
			} else if count > colRuneCount[col] {
				colRuneCount[col] = count
			}
		}
	}
	// Print with padded cell widths and columnDelimiter
	line := make([]byte, 0, 1024)
	for row := range rows {
		// Append cells of row to line
		for col, str := range rows[row] {
			if col > 0 {
				line = append(line, columnDelimiter...)
			}
			line = append(line, str...)
			if pad := colRuneCount[col] - utf8.RuneCountInString(str); pad > 0 {
				line = append(line, bytes.Repeat([]byte{' '}, pad)...)
			}
		}
		// In case not all rows have the same number of cells
		// pad line with empty cells
		for col := len(rows[row]); col < len(colRuneCount); col++ {
			if col > 0 {
				line = append(line, columnDelimiter...)
			}
			line = append(line, bytes.Repeat([]byte{' '}, colRuneCount[col])...)
		}
		line = append(line, '\n')
		_, err := w.Write(line)
		if err != nil {
			return err
		}
		line = line[:0]
	}
	return nil
}
