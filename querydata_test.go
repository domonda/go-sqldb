package sqldb

import (
	"reflect"
	"testing"
)

func TestNewQueryData(t *testing.T) {
	tests := []struct {
		query     string
		args      []any
		normalize NormalizeQueryFunc
		want      QueryData
		wantErr   bool
	}{
		{
			query: `
				SELECT *
				FROM public.user
				WHERE "name" = $1
					and age=$2;
			`,
			args:      []any{"Alice", 30},
			normalize: NewQueryNormalizer(),
			want: QueryData{
				Query: `SELECT * FROM public.user WHERE "name" = $1 and age = $2`,
				Args:  []any{"Alice", 30},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got, err := NewQueryData(tt.query, tt.args, tt.normalize)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewQueryData(%#v) error = %v, wantErr %v", tt.query, err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewQueryData(%#v) = %#v, want %#v", tt.query, got, tt.want)
			}
		})
	}
}
