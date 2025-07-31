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

// type ArrayScanner struct {
// 	Dest reflect.Value
// }

// func MustArrayScanner(destPtr any) sql.Scanner {
// 	v := reflect.ValueOf(destPtr).Elem()
// 	if !ShouldWrapForArray(v) {
// 		panic(fmt.Sprintf("expected pointer to slice or array, got %T", destPtr))
// 	}
// 	return &ArrayScanner{Dest: v}
// }

// // Scan implements the sql.Scanner interface.
// func (a *ArrayScanner) Scan(src any) error {
// 	switch src := src.(type) {
// 	case []byte:
// 		return a.scanString(string(src))
// 	case string:
// 		return a.scanString(src)
// 	case nil:
// 		if a.Dest.Kind() != reflect.Slice {
// 			return fmt.Errorf("can't scan NULL as %s", a.Dest.Type())
// 		}
// 		a.Dest.SetZero()
// 		return nil
// 	default:
// 		return fmt.Errorf("can't scan %T as %s", src, a.Dest.Type())
// 	}
// }

// func (a *ArrayScanner) scanString(src string) error {
// 	elems, err := nullable.SplitArray(src)
// 	if err != nil {
// 		return err
// 	}
// 	destIsSlice := a.Dest.Kind() == reflect.Slice
// 	if !destIsSlice && len(elems) != a.Dest.Len() {
// 		return fmt.Errorf("can't scan %d elements into array of length %d", len(elems), a.Dest.Len())
// 	}
// 	if destIsSlice && len(elems) == 0 {
// 		a.Dest.SetZero()
// 		return nil
// 	}
// 	elemType := a.Dest.Type().Elem()
// 	// allocate new slice or array on heap for scanning
// 	// only assign after scaning of all elements was successful
// 	var newDest reflect.Value
// 	if destIsSlice {
// 		newDest = reflect.MakeSlice(elemType, len(elems), len(elems))
// 	} else {
// 		newDest = reflect.New(a.Dest.Type()).Elem()
// 	}
// 	if reflect.PtrTo(elemType).Implements(typeOfSQLScanner) {
// 		for i, elemStr := range elems {
// 			err = newDest.Index(i).Addr().Interface().(sql.Scanner).Scan(elemStr)
// 			if err != nil {
// 				return fmt.Errorf("can't scan %q as element %d of slice %s because of %w", elemStr, i, elemType, err)
// 			}
// 		}
// 	} else {
// 		for i, elemStr := range elems {
// 			// TODO elemStr is a string because we splitted an SQL array string literal, can't scan into an int right now
// 			err = ScanValue(elemStr, newDest.Index(i))
// 			if err != nil {
// 				return fmt.Errorf("can't scan %q as element %d of slice %s because of %w", elemStr, i, elemType, err)
// 			}
// 		}
// 	}
// 	a.Dest.Set(newDest)
// 	return nil
// }
