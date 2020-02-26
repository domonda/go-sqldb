package impl

import (
	"context"
	"fmt"
	"reflect"

	sqldb "github.com/domonda/go-sqldb"
)

var (
	typeOfError   = reflect.TypeOf((*error)(nil)).Elem()
	typeOfContext = reflect.TypeOf((*context.Context)(nil)).Elem()
)

// ForEachRowScanFunc will call the passed callback with scanned values or a struct for every row.
// If the callback function has a single struct or struct pointer argument,
// then RowScanner.ScanStruct will be used per row,
// else RowScanner.Scan will be used for all arguments of the callback.
// If the function has a context.Context as first argument,
// then the passed ctx will be passed on.
// In case of zero rows, no error will be returned.
func ForEachRowScanFunc(ctx context.Context, callback interface{}) (f func(sqldb.RowScanner) error, err error) {
	val := reflect.ValueOf(callback)
	typ := val.Type()
	if typ.Kind() != reflect.Func {
		return nil, fmt.Errorf("ForEachRowScan expected callback function, got %s", typ)
	}
	if typ.IsVariadic() {
		return nil, fmt.Errorf("ForEachRowScan callback function must not be varidic: %s", typ)
	}
	if typ.NumIn() == 0 || (typ.NumIn() == 1 && typ.In(0) == typeOfContext) {
		return nil, fmt.Errorf("ForEachRowScan callback function has no arguments: %s", typ)
	}
	firstArg := 0
	if typ.In(0) == typeOfContext {
		firstArg = 1
	}
	structArg := false
	for i := firstArg; i < typ.NumIn(); i++ {
		if structArg {
			return nil, fmt.Errorf("ForEachRowScan callback function must not have further argument after struct: %s", typ)
		}
		derefArg := typ.In(i)
		for derefArg.Kind() == reflect.Ptr {
			derefArg = derefArg.Elem()
		}
		switch derefArg.Kind() {
		case reflect.Struct:
			structArg = true
		case reflect.Chan, reflect.Func:
			return nil, fmt.Errorf("ForEachRowScan callback function has invalid argument type: %s", typ.In(i))
		}
	}
	if typ.NumOut() > 1 || typ.Out(0) != typeOfError {
		return nil, fmt.Errorf("ForEachRowScan callback function can only have one error result: %s", typ)
	}

	f = func(row sqldb.RowScanner) (err error) {
		// First scan row
		scannedValPtrs := make([]interface{}, typ.NumIn()-firstArg)
		for i := range scannedValPtrs {
			scannedValPtrs[i] = reflect.New(typ.In(firstArg + i)).Interface()
		}
		if structArg {
			err = row.ScanStruct(scannedValPtrs[0])
		} else {
			err = row.Scan(scannedValPtrs...)
		}
		if err != nil {
			return err
		}

		// Then do callback via reflection
		args := make([]reflect.Value, typ.NumIn())
		if firstArg == 1 {
			args[0] = reflect.ValueOf(ctx)
		}
		for i := firstArg; i < len(args); i++ {
			args[i] = reflect.ValueOf(scannedValPtrs[i-firstArg]).Elem()
		}
		res := val.Call(args)
		if !res[0].IsNil() {
			return res[0].Interface().(error)
		}
		return nil
	}
	return f, nil
}
