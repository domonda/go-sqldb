package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConnConfig(t *testing.T) {
	tests := []struct {
		uri                    string
		wantURIWithoutPassword string
		want                   *ConnConfig
		wantErr                bool
	}{
		{
			uri:                    "postgres://user:password@localhost:5432/database?sslmode=disable",
			wantURIWithoutPassword: "postgres://user@localhost:5432/database?sslmode=disable",
			want: &ConnConfig{
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
		t.Run(tt.uri, func(t *testing.T) {
			got, err := ParseConnConfig(tt.uri)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			assert.Equal(t, tt.uri, got.URL().String(), "convertig back to URI should match original")
			assert.Equal(t, tt.wantURIWithoutPassword, got.String(), "convertig back to URI without password")
		})
	}
}
