package information

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYesNo_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    YesNo
		wantErr bool
	}{
		{name: "YES string", input: "YES", want: true},
		{name: "NO string", input: "NO", want: false},
		{name: "true bool", input: true, want: true},
		{name: "false bool", input: false, want: false},
		{name: "invalid string", input: "MAYBE", wantErr: true},
		{name: "invalid type int", input: 42, wantErr: true},
		{name: "invalid type nil", input: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var y YesNo
			err := y.Scan(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, y)
		})
	}
}

func TestString_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    String
		wantErr bool
	}{
		{name: "string value", input: "hello", want: "hello"},
		{name: "empty string", input: "", want: ""},
		{name: "nil is empty", input: nil, want: ""},
		{name: "byte slice", input: []byte("bytes"), want: "bytes"},
		{name: "invalid type int", input: 42, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s String
			err := s.Scan(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, s)
		})
	}
}
