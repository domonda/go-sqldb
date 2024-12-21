package db

import (
	"context"
	"fmt"
	"reflect"
)

// ForEachRowCallFunc will call the passed callback with scanned values or a struct for every row.
// If the callback function has a single struct or struct pointer argument,
// then RowScanner.ScanStruct will be used per row,
// else RowScanner.Scan will be used for all arguments of the callback.
// If the function has a context.Context as first argument,
// then the passed ctx will be passed on.
// The callback can have no result or a single error result value.
// If a non nil error is returned from the callback, then this error
// is returned immediately by this function without scanning further rows.
// In case of zero rows, no error will be returned.
func ForEachRowCallFunc(ctx context.Context, callback any) (f func(*RowScanner) error, err error) {
	val := reflect.ValueOf(callback)
	typ := val.Type()
	if typ.Kind() != reflect.Func {
		return nil, fmt.Errorf("ForEachRowCall expected callback function, got %s", typ)
	}
	if typ.IsVariadic() {
		return nil, fmt.Errorf("ForEachRowCall callback function must not be varidic: %s", typ)
	}
	if typ.NumIn() == 0 || (typ.NumIn() == 1 && typ.In(0) == typeOfContext) {
		return nil, fmt.Errorf("ForEachRowCall callback function has no arguments: %s", typ)
	}
	firstArg := 0
	if typ.In(0) == typeOfContext {
		firstArg = 1
	}
	structArg := false
	for i := firstArg; i < typ.NumIn(); i++ {
		t := typ.In(i)
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t == typeOfTime {
			continue
		}
		switch t.Kind() {
		case reflect.Struct:
			if t.Implements(typeOfSQLScanner) || reflect.PtrTo(t).Implements(typeOfSQLScanner) {
				continue
			}
			if structArg {
				return nil, fmt.Errorf("ForEachRowCall callback function must not have further argument after struct: %s", typ)
			}
			structArg = true
		case reflect.Chan, reflect.Func:
			return nil, fmt.Errorf("ForEachRowCall callback function has invalid argument type: %s", typ.In(i))
		}
	}
	if typ.NumOut() > 1 {
		return nil, fmt.Errorf("ForEachRowCall callback function can only have one result value: %s", typ)
	}
	if typ.NumOut() == 1 && typ.Out(0) != typeOfError {
		return nil, fmt.Errorf("ForEachRowCall callback function result must be of type error: %s", typ)
	}

	f = func(row *RowScanner) (err error) {
		// First scan row
		scannedValPtrs := make([]any, typ.NumIn()-firstArg)
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
		if len(res) > 0 && !res[0].IsNil() {
			return res[0].Interface().(error)
		}
		return nil
	}
	return f, nil
}
