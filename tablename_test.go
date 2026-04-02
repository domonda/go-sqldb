package sqldb

import (
	"reflect"
	"testing"
)

func TestTableNameForStruct(t *testing.T) {
	tests := []struct {
		name      string
		t         reflect.Type
		tagKey    string
		wantTable string
		wantErr   bool
	}{
		{
			name: "OK",
			t: reflect.TypeFor[struct {
				TableName `db:"table_name"`
			}](),
			tagKey:    "db",
			wantTable: "table_name",
		},
		{
			name: "more struct fields",
			t: reflect.TypeFor[struct {
				ID        int `db:"id"`
				TableName `db:"table_name"`
				Value     string `db:"value"`
			}](),
			tagKey:    "db",
			wantTable: "table_name",
		},
		{
			name: "nested embedded struct",
			t: reflect.TypeFor[struct {
				Inner struct {
					TableName `db:"inner_table"`
				}
			}](),
			tagKey:  "db",
			wantErr: true, // Inner is not anonymous, so TableName should not be found
		},
		{
			name: "anonymous nested embedded struct",
			t: func() reflect.Type {
				type Inner struct {
					TableName `db:"inner_table"`
				}
				type Outer struct {
					Inner
				}
				return reflect.TypeFor[Outer]()
			}(),
			tagKey:    "db",
			wantTable: "inner_table",
		},
		{
			name: "deeply nested anonymous embedded struct",
			t: func() reflect.Type {
				type Base struct {
					TableName `db:"deep_table"`
				}
				type Middle struct {
					Base
				}
				type Outer struct {
					Middle
				}
				return reflect.TypeFor[Outer]()
			}(),
			tagKey:    "db",
			wantTable: "deep_table",
		},
		// Error cases
		{
			name:    "empty",
			t:       reflect.TypeFor[struct{}](),
			tagKey:  "db",
			wantErr: true,
		},
		{
			name: "no tagKey",
			t: reflect.TypeFor[struct {
				TableName
			}](),
			tagKey:  "db",
			wantErr: true,
		},
		{
			name: "wrong tagKey",
			t: reflect.TypeFor[struct {
				TableName `json:"table_name"`
			}](),
			tagKey:  "db",
			wantErr: true,
		},
		{
			name: "named field",
			t: reflect.TypeFor[struct {
				Table TableName `db:"table_name"`
			}](),
			tagKey:  "db",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTable, err := TableNameForStruct(tt.t, tt.tagKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("TableForStruct(%s, %#v) error = %v, wantErr %v", tt.t, tt.tagKey, err, tt.wantErr)
				return
			}
			if gotTable != tt.wantTable {
				t.Errorf("TableForStruct(%s, %#v) = %v, want %v", tt.t, tt.tagKey, gotTable, tt.wantTable)
			}
		})
	}
}
