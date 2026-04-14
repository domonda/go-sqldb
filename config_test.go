package sqldb

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		uri                    string
		wantURIWithoutPassword string
		want                   *Config
		wantErr                bool
	}{
		{
			uri:                    "postgres://user:password@localhost:5432/database?sslmode=disable",
			wantURIWithoutPassword: "postgres://user@localhost:5432/database?sslmode=disable",
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
		t.Run(tt.uri, func(t *testing.T) {
			got, err := ParseConfig(tt.uri)
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

func TestParseConfig_NoExtra(t *testing.T) {
	got, err := ParseConfig("mysql://admin:secret@dbhost:3306/mydb")
	require.NoError(t, err)
	assert.Equal(t, "mysql", got.Driver)
	assert.Equal(t, "dbhost", got.Host)
	assert.Equal(t, uint16(3306), got.Port)
	assert.Equal(t, "admin", got.User)
	assert.Equal(t, "secret", got.Password)
	assert.Equal(t, "mydb", got.Database)
	assert.Nil(t, got.Extra)
}

func TestParseConfig_MissingPort(t *testing.T) {
	got, err := ParseConfig("postgres://user:pass@host/db")
	assert.NoError(t, err, "missing port should be accepted as zero")
	assert.Equal(t, uint16(0), got.Port)
}

func TestParseConfig_InvalidPort(t *testing.T) {
	_, err := ParseConfig("postgres://user:pass@host:abc/db")
	assert.Error(t, err, "non-numeric port should error")
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:   "valid",
			config: Config{Driver: "postgres", Host: "localhost", Database: "mydb"},
		},
		{
			name:    "missing driver",
			config:  Config{Host: "localhost", Database: "mydb"},
			wantErr: true,
		},
		{
			name:   "missing host",
			config: Config{Driver: "postgres", Database: "mydb"},
		},
		{
			name:   "sqlite without host",
			config: Config{Driver: "sqlite", Database: "/tmp/test.db"},
		},
		{
			name:    "missing database",
			config:  Config{Driver: "postgres", Host: "localhost"},
			wantErr: true,
		},
		{
			name:    "all empty",
			config:  Config{},
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

func TestConfig_String_DriverOnly(t *testing.T) {
	c := &Config{Driver: "postgres"}
	assert.Equal(t, "postgres", c.String())
}

func TestConfig_URL(t *testing.T) {
	t.Run("full config", func(t *testing.T) {
		// given
		c := &Config{
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
		c := &Config{Driver: "postgres", Database: "mydb"}
		u := c.URL()
		assert.Equal(t, "localhost", u.Host)
	})

	t.Run("defaults to UNKNOWN_DRIVER when driver empty", func(t *testing.T) {
		c := &Config{Database: "mydb"}
		u := c.URL()
		assert.Equal(t, "UNKNOWN_DRIVER", u.Scheme)
	})

	t.Run("no port omits port in host", func(t *testing.T) {
		c := &Config{Driver: "sqlite", Host: "myhost", Database: "test.db"}
		u := c.URL()
		assert.Equal(t, "myhost", u.Host)
	})

	t.Run("no user omits userinfo", func(t *testing.T) {
		c := &Config{Driver: "postgres", Database: "mydb"}
		u := c.URL()
		assert.Nil(t, u.User)
	})

	t.Run("no extra params", func(t *testing.T) {
		c := &Config{Driver: "postgres", Host: "localhost", Port: 5432, Database: "mydb"}
		u := c.URL()
		assert.Empty(t, u.RawQuery)
	})
}

// mockLogger records Printf calls for testing.
type mockLogger struct {
	messages []string
}

func (m *mockLogger) Printf(format string, v ...any) {
	m.messages = append(m.messages, fmt.Sprintf(format, v...))
}

func TestConfig_ErrLogger_NilByDefault(t *testing.T) {
	var c Config
	assert.Nil(t, c.ErrLogger, "ErrLogger should be nil by default")
}

func TestConfig_ErrLogger_Custom(t *testing.T) {
	logger := &mockLogger{}
	c := Config{
		ErrLogger: logger,
	}

	require.NotNil(t, c.ErrLogger)
	c.ErrLogger.Printf("test error %d", 42)
	require.Len(t, logger.messages, 1)
	assert.Equal(t, "test error 42", logger.messages[0])
}

func TestConfig_ListenerEventLogger_NilByDefault(t *testing.T) {
	var c Config
	assert.Nil(t, c.ListenerEventLogger, "ListenerEventLogger should be nil by default")
}

func TestConfig_ListenerEventLogger_Custom(t *testing.T) {
	logger := &mockLogger{}
	c := Config{
		ListenerEventLogger: logger,
	}

	require.NotNil(t, c.ListenerEventLogger)
	c.ListenerEventLogger.Printf("event %s", "connected")
	require.Len(t, logger.messages, 1)
	assert.Equal(t, "event connected", logger.messages[0])
}

func TestParseConfig_LoggersNotSet(t *testing.T) {
	c, err := ParseConfig("postgres://user:pass@localhost:5432/mydb")
	require.NoError(t, err)
	assert.Nil(t, c.ErrLogger, "ParseConfig should not set ErrLogger")
	assert.Nil(t, c.ListenerEventLogger, "ParseConfig should not set ListenerEventLogger")
}
