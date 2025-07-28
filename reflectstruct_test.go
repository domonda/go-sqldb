package sqldb

import (
	"database/sql"
	"reflect"
	"testing"
	"time"
)

func Test_isNonSQLScannerStruct(t *testing.T) {
	tests := []struct {
		t    reflect.Type
		want bool
	}{
		// Structs that do not implement sql.Scanner
		{t: reflect.TypeFor[struct{ X int }](), want: true},

		// Structs that implement sql.Scanner
		{t: reflect.TypeFor[time.Time](), want: false},
		{t: reflect.TypeFor[sql.NullTime](), want: false},

		// Non struct types
		{t: reflect.TypeFor[int](), want: false},
		{t: reflect.TypeFor[string](), want: false},
		{t: reflect.TypeFor[[]byte](), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.t.String(), func(t *testing.T) {
			if got := isNonSQLScannerStruct(tt.t); got != tt.want {
				t.Errorf("isNonSQLScannerStruct(%s) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}
