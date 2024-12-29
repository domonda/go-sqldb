package db

import (
	"reflect"
	"testing"
	"time"
)

func Test_isNonSQLScannerStruct(t *testing.T) {
	tests := []struct {
		t    reflect.Type
		want bool
	}{
		{t: reflect.TypeFor[struct{ X int }](), want: true},

		{t: reflect.TypeFor[int](), want: false},
		{t: reflect.TypeFor[time.Time](), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.t.String(), func(t *testing.T) {
			if got := isNonSQLScannerStruct(tt.t); got != tt.want {
				t.Errorf("isNonSQLScannerStruct(%s) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}
