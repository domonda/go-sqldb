package sqldb

import (
	"database/sql"
	"reflect"
)

// func WrapForArray(a any) interface {
// 	driver.Valuer
// 	sql.Scanner
// } {
// 	return pq.Array(a)
// }

type ArrayHandler interface {
	AsArrayScanner(dest any) sql.Scanner
}

func MakeArrayScannable(dest []any, arrayHandler ArrayHandler) []any {
	if arrayHandler == nil {
		return dest
	}
	var wrappedDest []any
	for i, d := range dest {
		if ShouldWrapForArrayScanning(reflect.ValueOf(d).Elem()) {
			if wrappedDest == nil {
				// Allocate new slice for wrapped element
				wrappedDest = make([]any, len(dest))
				// Copy previous elements
				for h := 0; h < i; h++ {
					wrappedDest[h] = dest[h]
				}
			}
			wrappedDest[i] = arrayHandler.AsArrayScanner(d)
		} else if wrappedDest != nil {
			wrappedDest[i] = d
		}
	}
	if wrappedDest != nil {
		return wrappedDest
	}
	return dest
}

func ShouldWrapForArrayScanning(v reflect.Value) bool {
	t := v.Type()
	if t.Implements(typeOfSQLScanner) {
		return false
	}
	if t.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
		t = v.Type()
	}
	switch t.Kind() {
	case reflect.Slice:
		if t.Elem() == typeOfByte {
			return false // Byte slices are scanned as strings
		}
		return !v.Addr().Type().Implements(typeOfSQLScanner)
	case reflect.Array:
		return !v.Addr().Type().Implements(typeOfSQLScanner)
	}
	return false
}

// IsSliceOrArray returns true if passed value is a slice or array,
// or a pointer to a slice or array and in case of a slice
// not of type []byte.
func IsSliceOrArray(value any) bool {
	if value == nil {
		return false
	}
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	t := v.Type()
	k := t.Kind()
	return k == reflect.Slice && t != typeOfByteSlice || k == reflect.Array
}

// IsNonDriverValuerSliceOrArrayType returns true if passed type
// does not implement driver.Valuer and is a slice or array,
// or a pointer to a slice or array and in case of a slice
// not of type []byte.
func IsNonDriverValuerSliceOrArrayType(t reflect.Type) bool {
	if t == nil || t.Implements(typeOfDriverValuer) {
		return false
	}
	k := t.Kind()
	if k == reflect.Ptr {
		t = t.Elem()
		k = t.Kind()
	}
	return k == reflect.Slice && t != typeOfByteSlice || k == reflect.Array
}

// func FormatArrays(args []any) []any {
// 	var wrappedArgs []any
// 	for i, arg := range args {
// 		if ShouldFormatArray(arg) {
// 			if wrappedArgs == nil {
// 				// Allocate new slice for wrapped element
// 				wrappedArgs = make([]any, len(args))
// 				// Copy previous elements
// 				for h := 0; h < i; h++ {
// 					wrappedArgs[h] = args[h]
// 				}
// 			}
// 			wrappedArgs[i], _ = pq.Array(arg).Value()
// 		} else if wrappedArgs != nil {
// 			wrappedArgs[i] = arg
// 		}
// 	}
// 	if wrappedArgs != nil {
// 		return wrappedArgs
// 	}
// 	return args
// }

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
