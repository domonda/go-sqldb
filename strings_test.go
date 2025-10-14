package sqldb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeString(t *testing.T) {
	tests := map[string]string{
		"":                                 "",
		"Hello World!":                     "Hello World!",
		"\u0000,\u0009,\u007f":             ",,",
		"\a,\b,\v":                         ",,",
		"\xc3\x22":                         "\"",
		"\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98": "= ⌘",
		string([]byte{71, 101, 115, 99, 104, 195, 164, 102, 116, 115, 118, 111, 114, 102, 195}): "Geschäftsvorf",
	}
	for str, want := range tests {
		t.Run(str, func(t *testing.T) {
			if got := SanitizeString(str); got != want {
				t.Errorf("SanitizeString(%q) = %q, want %q", str, got, want)
			}
		})
	}
}

func TestIsSQLInjection(t *testing.T) {
	tests := []struct {
		str  string
		want bool
	}{
		// SQL injection attempts
		{"' OR '1'='1", true},
		{"1' OR '1' = '1", true},
		{"admin'--", true},
		// {"admin' #", true}, // TODO
		{"admin'/*", true},
		{"' OR 1=1--", true},
		// {"; DROP TABLE users--", true}, // TODO
		{"'; DROP TABLE users; --", true},
		{"1; DELETE FROM users", true},
		{"' UNION SELECT NULL--", true},
		{"' UNION SELECT * FROM users--", true},
		{"1' UNION ALL SELECT NULL,NULL,NULL--", true},
		{"' AND 1=0 UNION ALL SELECT 'admin', '81dc9bdb52d04dc20036dbd8313ed055'", true},
		{"admin' AND '1'='1", true},
		{"' OR 'x'='x", true},
		{"1' AND 1=1--", true},
		{"1' AND SLEEP(5)--", true},
		{"1' WAITFOR DELAY '00:00:05'--", true},
		{"'; EXEC xp_cmdshell('dir')--", true},
		{"1' AND (SELECT * FROM users) = 1--", true},

		// Valid non-injection strings
		{"john.doe@example.com", false},
		{"O'Brien", false},
		{"It's a beautiful day", false},
		{"user@domain.com", false},
		{"John-Smith", false},
		{"123456", false},
		{"normal text", false},
		{"2024-01-15", false},
		{"value_with_underscore", false},
		{"CamelCaseText", false},
		{"text with spaces", false},
		{"price: $19.99", false},
		{"50%", false},
		{"(555) 123-4567", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			got, info := IsSQLInjection(tt.str)
			require.Equalf(t, tt.want, got, "IsSQLInjection(%#v) returned: %#v, %#v", tt.str, got, info)
		})
	}
}
