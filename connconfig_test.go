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

func TestParseConnConfig_MissingPort(t *testing.T) {
	got, err := ParseConnConfig("postgres://user:pass@host/db")
	assert.NoError(t, err, "missing port should be accepted as zero")
	assert.Equal(t, uint16(0), got.Port)
}

func TestParseConnConfig_InvalidPort(t *testing.T) {
	_, err := ParseConnConfig("postgres://user:pass@host:abc/db")
	assert.Error(t, err, "non-numeric port should error")
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
			name:   "missing host",
			config: ConnConfig{Driver: "postgres", Database: "mydb"},
		},
		{
			name:   "sqlite without host",
			config: ConnConfig{Driver: "sqlite", Database: "/tmp/test.db"},
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

func TestConnConfig_URL(t *testing.T) {
	t.Run("full config", func(t *testing.T) {
		// given
		c := &ConnConfig{
			Driver:   "postgres",
			Host:     "dbhost",
			Port:     5432,
			User:     "admin",
			Password: "secret",
			Database: "mydb",
			Extra:    map[string]string{"sslmode": "disable"},
		}

		// when
		u := c.URL()

		// then
		assert.Equal(t, "postgres", u.Scheme)
		assert.Equal(t, "dbhost:5432", u.Host)
		assert.Equal(t, "mydb", u.Path)
		assert.Equal(t, "admin", u.User.Username())
		pw, _ := u.User.Password()
		assert.Equal(t, "secret", pw)
		assert.Equal(t, "disable", u.Query().Get("sslmode"))
	})

	t.Run("defaults to localhost when host empty", func(t *testing.T) {
		c := &ConnConfig{Driver: "postgres", Database: "mydb"}
		u := c.URL()
		assert.Equal(t, "localhost", u.Host)
	})

	t.Run("defaults to UNKNOWN_DRIVER when driver empty", func(t *testing.T) {
		c := &ConnConfig{Database: "mydb"}
		u := c.URL()
		assert.Equal(t, "UNKNOWN_DRIVER", u.Scheme)
	})

	t.Run("no port omits port in host", func(t *testing.T) {
		c := &ConnConfig{Driver: "sqlite", Host: "myhost", Database: "test.db"}
		u := c.URL()
		assert.Equal(t, "myhost", u.Host)
	})

	t.Run("no user omits userinfo", func(t *testing.T) {
		c := &ConnConfig{Driver: "postgres", Database: "mydb"}
		u := c.URL()
		assert.Nil(t, u.User)
	})

	t.Run("no extra params", func(t *testing.T) {
		c := &ConnConfig{Driver: "postgres", Host: "localhost", Port: 5432, Database: "mydb"}
		u := c.URL()
		assert.Empty(t, u.RawQuery)
	})
}
