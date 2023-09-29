package impl

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/domonda/go-types/nullable"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestShouldWrapForArrayScanning(t *testing.T) {
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
		{v: reflect.ValueOf(WrapForArrayScanning([]int{0, 1})), want: false},

		{v: reflect.ValueOf(new([3]string)).Elem(), want: true},
		{v: reflect.ValueOf(new([]string)).Elem(), want: true},
		{v: reflect.ValueOf(new([]sql.NullString)).Elem(), want: true},
	}
	for _, tt := range tests {
		got := ShouldWrapForArrayScanning(tt.v)
		assert.Equal(t, tt.want, got)
	}
}

func TestIsNonDriverValuerSliceOrArrayType(t *testing.T) {
	tests := []struct {
		t    reflect.Type
		want bool
	}{
		{t: reflect.TypeOf(nil), want: false},
		{t: reflect.TypeOf(0), want: false},
		{t: reflect.TypeOf(new(int)), want: false},
		{t: reflect.TypeOf("string"), want: false},
		{t: reflect.TypeOf([]byte("string")), want: false},
		{t: reflect.TypeOf(new([]byte)), want: false},
		{t: reflect.TypeOf(pq.BoolArray{true}), want: false},
		{t: reflect.TypeOf(new(pq.BoolArray)), want: false},
		{t: reflect.TypeOf(new(*[]int)), want: false}, // pointer to a pointer to a slice
		{t: reflect.TypeOf((*driver.Valuer)(nil)), want: false},
		{t: reflect.TypeOf((*driver.Valuer)(nil)).Elem(), want: false},

		{t: reflect.TypeOf([3]int{1, 2, 3}), want: true},
		{t: reflect.TypeOf((*[3]int)(nil)), want: true},
		{t: reflect.TypeOf([]int{1, 2, 3}), want: true},
		{t: reflect.TypeOf((*[]int)(nil)), want: true},
		{t: reflect.TypeOf((*[][]byte)(nil)), want: true},
	}
	for _, tt := range tests {
		got := IsNonDriverValuerSliceOrArrayType(tt.t)
		assert.Equalf(t, tt.want, got, "IsNonDriverValuerSliceOrArrayType(%s)", tt.t)
	}
}

// func TestWrapArgsForArrays(t *testing.T) {
// 	tests := []struct {
// 		args []any
// 		want []any
// 	}{
// 		{args: nil, want: nil},
// 		{args: []any{}, want: []any{}},
// 		{args: []any{0}, want: []any{0}},
// 		{args: []any{nil}, want: []any{nil}},
// 		{args: []any{new(int)}, want: []any{new(int)}},
// 		{args: []any{0, []int{0, 1}, "string"}, want: []any{0, wrapArgForArray([]int{0, 1}), "string"}},
// 		{args: []any{wrapArgForArray([]int{0, 1})}, want: []any{wrapArgForArray([]int{0, 1})}},
// 		{args: []any{[]byte("don't wrap []byte")}, want: []any{[]byte("don't wrap []byte")}},
// 		{args: []any{pq.BoolArray{true}}, want: []any{pq.BoolArray{true}}},
// 		{args: []any{[3]int{1, 2, 3}}, want: []any{wrapArgForArray([3]int{1, 2, 3})}},
// 		{args: []any{wrapArgForArray([3]int{1, 2, 3})}, want: []any{wrapArgForArray([3]int{1, 2, 3})}},
// 	}
// 	for _, tt := range tests {
// 		got := WrapArgsForArrays(tt.args)
// 		assert.Equal(t, tt.want, got)
// 	}
// }
