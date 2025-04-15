package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfigURL(t *testing.T) {
	tests := []struct {
		configURL string
		want      *Config
		wantErr   bool
	}{
		{
			configURL: "postgres://user:password@localhost:5432/database?sslmode=disable",
			want: &Config{
				Driver:   "postgres",
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "password",
				Database: "database",
				Extra:    map[string]string{"sslmode": "disable"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.configURL, func(t *testing.T) {
			got, err := ParseConfigURL(tt.configURL)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			assert.Equal(t, tt.configURL, got.URL().String(), "convertig back to URL should match original")
		})
	}
}
