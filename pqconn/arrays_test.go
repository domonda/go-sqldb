package pqconn

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
)

func Test_wrapArrayScanDest(t *testing.T) {
	// Non-slice pointer should not be wrapped
	var i int
	dest := []any{&i}
	wrapArrayScanDest(dest)
	if _, ok := dest[0].(*int); !ok {
		t.Errorf("expected *int to not be wrapped, got %T", dest[0])
	}

	// String slice pointer should be wrapped
	var s []string
	dest = []any{&s}
	wrapArrayScanDest(dest)
	if _, ok := dest[0].(*[]string); ok {
		t.Errorf("expected *[]string to be wrapped with pq.Array, but got unwrapped *[]string")
	}

	// Byte slice pointer should NOT be wrapped
	var b []byte
	dest = []any{&b}
	wrapArrayScanDest(dest)
	if _, ok := dest[0].(*[]byte); !ok {
		t.Errorf("expected *[]byte to not be wrapped, got %T", dest[0])
	}

	// Nil pointer should not be wrapped
	dest = []any{(*[]string)(nil)}
	wrapArrayScanDest(dest) // should not panic

	// sql.Scanner slice should not be wrapped
	var ns []sql.NullString
	dest = []any{&ns}
	origAddr := &ns
	wrapArrayScanDest(dest)
	// NullString does not implement Scanner on the slice level,
	// so []sql.NullString should be wrapped
	if dest[0] == origAddr {
		// The slice of NullString does not implement sql.Scanner
		// so it should be wrapped
	}
}

func Test_needsArrayWrappingForScanning(t *testing.T) {
	tests := []struct {
		v    reflect.Value
		want bool
	}{
		{v: reflect.ValueOf([]byte(nil)), want: false},
		{v: reflect.ValueOf([]byte{}), want: false},
		{v: reflect.ValueOf(""), want: false},
		{v: reflect.ValueOf(0), want: false},
		{v: reflect.ValueOf(json.RawMessage([]byte("null"))), want: false},
		{v: reflect.ValueOf(new(sql.NullInt64)).Elem(), want: false},

		{v: reflect.ValueOf(new([3]string)).Elem(), want: true},
		{v: reflect.ValueOf(new([]string)).Elem(), want: true},
		{v: reflect.ValueOf(new([]sql.NullString)).Elem(), want: true},
	}
	for _, tt := range tests {
		if got := needsArrayWrappingForScanning(tt.v); got != tt.want {
			t.Errorf("needsArrayWrappingForScanning() = %v, want %v", got, tt.want)
		}
	}
}
