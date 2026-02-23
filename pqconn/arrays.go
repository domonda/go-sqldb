package pqconn

import (
	"database/sql"
	"database/sql/driver"
	"reflect"

	"github.com/lib/pq"
)

var (
	typeOfSQLScanner   = reflect.TypeFor[sql.Scanner]()
	typeOfDriverValuer = reflect.TypeFor[driver.Valuer]()
	typeOfByte         = reflect.TypeFor[byte]()
)

type valuerScanner interface {
	driver.Valuer
	sql.Scanner
}

func wrapArray(a any) valuerScanner {
	return pq.Array(a)
}

func needsArrayWrappingForScanning(v reflect.Value) bool {
	t := v.Type()
	switch t.Kind() {
	case reflect.Slice:
		// Byte slices are scanned as strings
		return t.Elem() != typeOfByte && !v.Addr().Type().Implements(typeOfSQLScanner)
	case reflect.Array:
		return !v.Addr().Type().Implements(typeOfSQLScanner)
	}
	return false
}

func needsArrayWrappingForArg(arg any) bool {
	if arg == nil {
		return false
	}
	t := reflect.TypeOf(arg)
	switch t.Kind() {
	case reflect.Slice:
		// Byte slices are interpreted as strings
		return t.Elem() != typeOfByte && !t.Implements(typeOfDriverValuer)
	case reflect.Array:
		return !t.Implements(typeOfDriverValuer)
	}
	return false
}

func wrapArrayArgs(args []any) {
	for i, arg := range args {
		if needsArrayWrappingForArg(arg) {
			args[i] = wrapArray(arg)
		}
	}
}

func wrapArrayScanDest(dest []any) {
	for i, d := range dest {
		v := reflect.ValueOf(d)
		if v.Kind() == reflect.Pointer && !v.IsNil() {
			if needsArrayWrappingForScanning(v.Elem()) {
				dest[i] = wrapArray(d)
			}
		}
	}
}
