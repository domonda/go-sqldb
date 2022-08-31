package sqldb

import "testing"

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
