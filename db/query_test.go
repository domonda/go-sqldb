package db

import (
	"context"
	"database/sql/driver"
	"reflect"
	"testing"
	"time"

	"github.com/domonda/go-sqldb"
	"github.com/stretchr/testify/require"
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

func TestQueryValue(t *testing.T) {
	query := /*sql*/ `SELECT EXISTS (SELECT FROM my_table WHERE id = $1)`
	conn := sqldb.NewMockConn("$", nil, nil).WithQueryResult(
		[]string{"exists"},       // columns
		[][]driver.Value{{true}}, // rows
		query,                    // query
		666,                      // args
	)
	ctx := ContextWithConn(context.Background(), conn)
	value, err := QueryValue[bool](ctx, query, 666)
	require.NoError(t, err)
	require.Equal(t, true, value, "QueryValue[bool] result")
}
