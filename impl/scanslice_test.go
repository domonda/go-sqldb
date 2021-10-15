package impl

import (
	"reflect"
	"testing"
)

func TestSplitArray(t *testing.T) {
	tests := []struct {
		name       string
		array      string
		wantFields []string
		wantErr    bool
	}{
		{
			name:    "empty",
			array:   ``,
			wantErr: true,
		},
		{
			name:       "empty[]",
			array:      `[]`,
			wantFields: nil,
		},
		{
			name:       "empty{}",
			array:      `{}`,
			wantFields: nil,
		},
		{
			name:       `[a]`,
			array:      `[a]`,
			wantFields: []string{`a`},
		},
		{
			name:       `[a,b]`,
			array:      `[a,b]`,
			wantFields: []string{`a`, `b`},
		},
		{
			name:       `[a, b]`,
			array:      `[a, b]`,
			wantFields: []string{`a`, `b`},
		},
		{
			name:       `["[quoted", "{", "comma,string", "}"]`,
			array:      `["[quoted", "{", "comma,string", "}"]`,
			wantFields: []string{`"[quoted"`, `"{"`, `"comma,string"`, `"}"`},
		},
		{
			name:       `[[1,2,3], {"key": "comma,string"}, null]`,
			array:      `[[1,2,3], {"key": "comma,string"}, null]`,
			wantFields: []string{`[1,2,3]`, `{"key": "comma,string"}`, `null`},
		},
		{
			name:       `{{1,2,3},{4,5,6},{7,8,9}}`,
			array:      `{{1,2,3},{4,5,6},{7,8,9}}`,
			wantFields: []string{`{1,2,3}`, `{4,5,6}`, `{7,8,9}`},
		},
		{
			name:       `{{"meeting", "lunch"}, {"training", "presentation"}}`,
			array:      `{{"meeting", "lunch"}, {"training", "presentation"}}`,
			wantFields: []string{`{"meeting", "lunch"}`, `{"training", "presentation"}`},
		},
		{
			name:       `[['meeting', 'lunch'], ['training', 'presentation']]`,
			array:      `[['meeting', 'lunch'], ['training', 'presentation']]`,
			wantFields: []string{`['meeting', 'lunch']`, `['training', 'presentation']`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFields, err := SplitArray(tt.array)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitArray() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotFields, tt.wantFields) {
				t.Errorf("SplitArray() = %#v, want %#v", gotFields, tt.wantFields)
			}
		})
	}
}
