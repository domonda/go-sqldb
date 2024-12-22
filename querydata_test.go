package sqldb

import (
	"reflect"
	"testing"
)

func TestNormalizedQueryData(t *testing.T) {
	tests := []struct {
		query   string
		args    []any
		want    QueryData
		wantErr bool
	}{
		{
			query: `
				SELECT *
				FROM public.user
				WHERE "name" = $1
					and age=$2;
			`,
			args: []any{"Alice", 30},
			want: QueryData{
				Query: `SELECT * FROM public.user WHERE "name" = $1 and age = $2`,
				Args:  []any{"Alice", 30},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got, err := NormalizedQueryData(tt.query, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizedQueryData(%#v) error = %v, wantErr %v", tt.query, err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NormalizedQueryData(%#v) = %#v, want %#v", tt.query, got, tt.want)
			}
		})
	}
}
