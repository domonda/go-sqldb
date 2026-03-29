package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColumnInfo_IsEmbeddedField(t *testing.T) {
	for _, scenario := range []struct {
		name string
		col  ColumnInfo
		want bool
	}{
		{
			name: "empty name is embedded",
			col:  ColumnInfo{Name: ""},
			want: true,
		},
		{
			name: "named column is not embedded",
			col:  ColumnInfo{Name: "id"},
			want: false,
		},
		{
			name: "primary key with name is not embedded",
			col:  ColumnInfo{Name: "id", PrimaryKey: true},
			want: false,
		},
		{
			name: "zero value is embedded",
			col:  ColumnInfo{},
			want: true,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			assert.Equal(t, scenario.want, scenario.col.IsEmbeddedField())
		})
	}
}
