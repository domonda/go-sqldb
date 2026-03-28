package mysqlconn

import (
	"strings"
	"testing"
)

func TestEscapeIdentifier(t *testing.T) {
	tests := []struct {
		ident string
		want  string
	}{
		// safe: lowercase + underscore only, not a reserved word
		{ident: `my_table`, want: `my_table`},
		{ident: `_private`, want: `_private`},
		// uppercase triggers quoting
		{ident: `MyTable`, want: "`MyTable`"},
		// digit triggers quoting
		{ident: `col1`, want: "`col1`"},
		// dollar sign triggers quoting
		{ident: `$col`, want: "`$col`"},
		{ident: `col_$123`, want: "`col_$123`"},
		// reserved words get quoted
		{ident: `table`, want: "`table`"},
		{ident: `select`, want: "`select`"},
		{ident: `order`, want: "`order`"},
		{ident: `key`, want: "`key`"},
		{ident: `group`, want: "`group`"},
		{ident: `rank`, want: "`rank`"},
		// reserved word matching is case-insensitive
		{ident: `SELECT`, want: "`SELECT`"},
		{ident: `Order`, want: "`Order`"},
		// embedded backtick: escaped and quoted
		{ident: "col`name", want: "`col``name`"},
		// digit-start triggers quoting
		{ident: `1col`, want: "`1col`"},
	}
	for _, tt := range tests {
		t.Run(tt.ident, func(t *testing.T) {
			if got := EscapeIdentifier(tt.ident); got != tt.want {
				t.Errorf("EscapeIdentifier(%#v) = %#v, want %#v", tt.ident, got, tt.want)
			}
		})
	}
}

func TestQueryFormatter_FormatTableName(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		// simple safe name — no quoting
		{name: `my_table`, want: `my_table`},
		// reserved words get quoted
		{name: `table`, want: "`table`"},
		{name: `order`, want: "`order`"},
		{name: `select`, want: "`select`"},
		{name: `key`, want: "`key`"},
		{name: `rank`, want: "`rank`"},
		// schema-qualified
		{name: `mydb.my_table`, want: `mydb.my_table`},
		{name: `mydb.order`, want: "mydb.`order`"},
		{name: `mydb.MyTable`, want: "mydb.`MyTable`"},
		// MySQL-specific: digit-start is valid, gets quoted
		{name: `1table`, want: "`1table`"},
		{name: `mydb.1table`, want: "mydb.`1table`"},
		// dollar sign is valid, gets quoted
		{name: `$table`, want: "`$table`"},
		// uppercase triggers quoting
		{name: `MyTable`, want: "`MyTable`"},
		// max length: 64 chars is valid
		{name: strings.Repeat("a", 64), want: strings.Repeat("a", 64)},
		// invalid: empty, too long, bad chars
		{name: ``, wantErr: true},
		{name: strings.Repeat("a", 65), wantErr: true},
		{name: `my-table`, wantErr: true},
		{name: `my table`, wantErr: true},
		{name: `my.table.extra`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := QueryFormatter{}.FormatTableName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatTableName(%#v) error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FormatTableName(%#v) = %#v, want %#v", tt.name, got, tt.want)
			}
		})
	}
}

func TestQueryFormatter_FormatColumnName(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		// safe: lowercase + underscore, not reserved
		{name: `my_col`, want: `my_col`},
		{name: `_private`, want: `_private`},
		// reserved words get quoted
		{name: `column`, want: "`column`"},
		{name: `key`, want: "`key`"},
		{name: `order`, want: "`order`"},
		{name: `select`, want: "`select`"},
		{name: `rank`, want: "`rank`"},
		// MySQL-specific: digit-start is valid, gets quoted
		{name: `1col`, want: "`1col`"},
		// dollar sign is valid, gets quoted
		{name: `$col`, want: "`$col`"},
		{name: `col_$123`, want: "`col_$123`"},
		// uppercase triggers quoting
		{name: `MyCol`, want: "`MyCol`"},
		// digits in name trigger quoting
		{name: `col1`, want: "`col1`"},
		// max length: 64 chars is valid
		{name: strings.Repeat("a", 64), want: strings.Repeat("a", 64)},
		// invalid: empty, too long, bad chars
		{name: ``, wantErr: true},
		{name: strings.Repeat("a", 65), wantErr: true},
		{name: `col-name`, wantErr: true},
		{name: `col name`, wantErr: true},
		{name: `col.name`, wantErr: true},
		{name: "col`name", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := QueryFormatter{}.FormatColumnName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatColumnName(%#v) error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FormatColumnName(%#v) = %#v, want %#v", tt.name, got, tt.want)
			}
		})
	}
}

func TestQueryFormatter_FormatPlaceholder(t *testing.T) {
	f := QueryFormatter{}
	tests := []struct {
		index int
		want  string
	}{
		{index: 0, want: "?"},
		{index: 1, want: "?"},
		{index: 9, want: "?"},
		{index: 99, want: "?"},
	}
	for _, tt := range tests {
		if got := f.FormatPlaceholder(tt.index); got != tt.want {
			t.Errorf("FormatPlaceholder(%d) = %q, want %q", tt.index, got, tt.want)
		}
	}
}

func TestQueryFormatter_FormatPlaceholder_NegativePanics(t *testing.T) {
	f := QueryFormatter{}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative paramIndex")
		}
	}()
	f.FormatPlaceholder(-1)
}

func TestQueryFormatter_FormatStringLiteral(t *testing.T) {
	f := QueryFormatter{}
	tests := []struct {
		name string
		str  string
		want string
	}{
		{name: "simple", str: "hello", want: "'hello'"},
		{name: "empty", str: "", want: "''"},
		// single quotes are backslash-escaped
		{name: "with single quote", str: "it's", want: `'it\'s'`},
		{name: "two consecutive quotes", str: "it''s", want: `'it\'\'s'`},
		// backslashes are escaped
		{name: "with backslash", str: `path\to`, want: `'path\\to'`},
		{name: "windows path", str: `C:\Users\file`, want: `'C:\\Users\\file'`},
		// backslash immediately before a single quote
		{name: "backslash before quote", str: `it\'s`, want: `'it\\\'s'`},
		// value with outer single quotes
		{name: "value with outer quotes", str: "'hello'", want: `'\'hello\''`},
		// NUL byte
		{name: "NUL byte", str: "a\x00b", want: `'a\0b'`},
		{name: "only NUL", str: "\x00", want: `'\0'`},
		// newline
		{name: "newline", str: "line1\nline2", want: `'line1\nline2'`},
		// carriage return
		{name: "carriage return", str: "line1\rline2", want: `'line1\rline2'`},
		// CRLF
		{name: "CRLF", str: "line1\r\nline2", want: `'line1\r\nline2'`},
		// Ctrl+Z (SUB, 0x1a)
		{name: "Ctrl+Z", str: "a\x1ab", want: `'a\Zb'`},
		// double quote
		{name: "double quote", str: `say "hello"`, want: `'say \"hello\"'`},
		// all special chars combined
		{name: "all special chars", str: "\x00\n\r\x1a'\"\\\x00", want: `'\0\n\r\Z\'\"\\\0'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.FormatStringLiteral(tt.str); got != tt.want {
				t.Errorf("FormatStringLiteral(%q) = %q, want %q", tt.str, got, tt.want)
			}
		})
	}
}
