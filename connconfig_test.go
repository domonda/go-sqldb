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
			assert.Equal(t, tt.uri, got.URL().String(), "converting back to URI should match original")
			assert.Equal(t, tt.wantURIWithoutPassword, got.String(), "converting back to URI without password")
		})
	}
}

func TestParseConnConfig_NoExtra(t *testing.T) {
	got, err := ParseConnConfig("mysql://admin:secret@dbhost:3306/mydb")
	require.NoError(t, err)
	assert.Equal(t, "mysql", got.Driver)
	assert.Equal(t, "dbhost", got.Host)
	assert.Equal(t, uint16(3306), got.Port)
	assert.Equal(t, "admin", got.User)
	assert.Equal(t, "secret", got.Password)
	assert.Equal(t, "mydb", got.Database)
	assert.Nil(t, got.Extra)
}

func TestParseConnConfig_InvalidPort(t *testing.T) {
	_, err := ParseConnConfig("postgres://user:pass@host/db")
	assert.Error(t, err, "missing port should error")
}

func TestConnConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ConnConfig
		wantErr bool
	}{
		{
			name:   "valid",
			config: ConnConfig{Driver: "postgres", Host: "localhost", Database: "mydb"},
		},
		{
			name:    "missing driver",
			config:  ConnConfig{Host: "localhost", Database: "mydb"},
			wantErr: true,
		},
		{
			name:    "missing host",
			config:  ConnConfig{Driver: "postgres", Database: "mydb"},
			wantErr: true,
		},
		{
			name:    "missing database",
			config:  ConnConfig{Driver: "postgres", Host: "localhost"},
			wantErr: true,
		},
		{
			name:    "all empty",
			config:  ConnConfig{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConnConfig_String_DriverOnly(t *testing.T) {
	c := &ConnConfig{Driver: "postgres"}
	assert.Equal(t, "postgres", c.String())
}
