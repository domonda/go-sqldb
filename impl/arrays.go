package impl

import (
	"database/sql"
	"database/sql/driver"
	"reflect"

	"github.com/lib/pq"
)

func WrapForArray(a interface{}) interface {
	driver.Valuer
	sql.Scanner
} {
	return pq.Array(a)
}

func ShouldWrapForArray(v reflect.Value) bool {
	t := v.Type()
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

// func ScanReflectValue(src any, dest reflect.Value) error {
// 	if dest.Kind() == reflect.Interface {
// 		if src != nil {
// 			dest.Set(reflect.ValueOf(src))
// 		} else {
// 			dest.SetZero()
// 		}
// 		return nil
// 	}

// 	if dest.Addr().Type().Implements(typeOfSQLScanner) {
// 		return dest.Addr().Interface().(sql.Scanner).Scan(src)
// 	}

// 	switch x := src.(type) {
// 	case int64:
// 		switch dest.Kind() {
// 		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
// 			dest.SetInt(x)
// 			return nil
// 		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
// 			dest.SetUint(uint64(x))
// 			return nil
// 		case reflect.Float32, reflect.Float64:
// 			dest.SetFloat(float64(x))
// 			return nil
// 		}

// 	case float64:
// 		switch dest.Kind() {
// 		case reflect.Float32, reflect.Float64:
// 			dest.SetFloat(x)
// 			return nil
// 		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
// 			dest.SetInt(int64(x))
// 			return nil
// 		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
// 			dest.SetUint(uint64(x))
// 			return nil
// 		}

// 	case bool:
// 		dest.SetBool(x)
// 		return nil

// 	case []byte:
// 		switch {
// 		case dest.Kind() == reflect.String:
// 			dest.SetString(string(x))
// 			return nil
// 		case dest.Kind() == reflect.Slice && dest.Type().Elem().Kind() == reflect.Uint8:
// 			dest.Set(reflect.ValueOf(x))
// 			return nil
// 		}

// 	case string:
// 		switch {
// 		case dest.Kind() == reflect.String:
// 			dest.SetString(x)
// 			return nil
// 		case dest.Type() == typeOfByteSlice:
// 			dest.Set(reflect.ValueOf([]byte(x)))
// 			return nil
// 		}

// 	case time.Time:
// 		if srcVal := reflect.ValueOf(src); srcVal.Type().AssignableTo(dest.Type()) {
// 			dest.Set(srcVal)
// 			return nil
// 		}

// 	case nil:
// 		switch dest.Kind() {
// 		case reflect.Ptr, reflect.Slice, reflect.Map:
// 			dest.SetZero()
// 			return nil
// 		}
// 	}

// 	return fmt.Errorf("can't scan %#v as %s", src, dest.Type())
// }
