package impl

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/domonda/go-types/nullable"
)

func TestShouldWrapForArray(t *testing.T) {
	tests := []struct {
		v    reflect.Value
		want bool
	}{
		{v: reflect.ValueOf([]byte(nil)), want: false},
		{v: reflect.ValueOf([]byte{}), want: false},
		{v: reflect.ValueOf(""), want: false},
		{v: reflect.ValueOf(0), want: false},
		{v: reflect.ValueOf(json.RawMessage([]byte("null"))), want: false},
		{v: reflect.ValueOf(nullable.JSON([]byte("null"))), want: false},
		{v: reflect.ValueOf(new(sql.NullInt64)).Elem(), want: false},

		{v: reflect.ValueOf(new([3]string)).Elem(), want: true},
		{v: reflect.ValueOf(new([]string)).Elem(), want: true},
		{v: reflect.ValueOf(new([]sql.NullString)).Elem(), want: true},
	}
	for _, tt := range tests {
		if got := ShouldWrapForArray(tt.v); got != tt.want {
			t.Errorf("shouldWrapArray() = %v, want %v", got, tt.want)
		}
	}
}
