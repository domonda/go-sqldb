package pqconn

import (
	"strings"
	"testing"

	"github.com/domonda/go-sqldb"
)

func TestQueryFormatter_FormatTableName(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{name: `table`, want: `"table"`},
		{name: `public.table`, want: `public."table"`},
		{name: `public.my_table`, want: `public.my_table`},
		// reserved words get quoted — original set
		{name: `select`, want: `"select"`},
		{name: `public.select`, want: `public."select"`},
		{name: `order`, want: `"order"`},
		// reserved words get quoted — newly added keywords
		{name: `is`, want: `"is"`},
		{name: `like`, want: `"like"`},
		{name: `ilike`, want: `"ilike"`},
		{name: `join`, want: `"join"`},
		{name: `inner`, want: `"inner"`},
		{name: `outer`, want: `"outer"`},
		{name: `left`, want: `"left"`},
		{name: `right`, want: `"right"`},
		{name: `full`, want: `"full"`},
		{name: `cross`, want: `"cross"`},
		{name: `natural`, want: `"natural"`},
		{name: `similar`, want: `"similar"`},
		{name: `between`, want: `"between"`},
		{name: `overlaps`, want: `"overlaps"`},
		{name: `isnull`, want: `"isnull"`},
		{name: `notnull`, want: `"notnull"`},
		{name: `tablesample`, want: `"tablesample"`},
		{name: `freeze`, want: `"freeze"`},
		{name: `concurrently`, want: `"concurrently"`},
		{name: `authorization`, want: `"authorization"`},
		{name: `current_schema`, want: `"current_schema"`},
		// mixed case gets quoted
		{name: `MyTable`, want: `"MyTable"`},
		{name: `public.MyTable`, want: `public."MyTable"`},
		// max length: 63 chars is valid
		{name: strings.Repeat("a", 63), want: strings.Repeat("a", 63)},
		// invalid: too long, empty, digit-start, hyphen
		{name: strings.Repeat("a", 64), wantErr: true},
		{name: ``, wantErr: true},
		{name: `1table`, wantErr: true},
		{name: `my-table`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := QueryFormatter{}.FormatTableName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryFormatter.FormatTableName(%#v) error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("QueryFormatter.FormatTableName(%#v) = %#v, want %#v", tt.name, got, tt.want)
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
		{name: `column`, want: `"column"`},
		{name: `Hello_World`, want: `"Hello_World"`},
		{name: `public.my_table`, wantErr: true},
		// reserved words get quoted — original set
		{name: `select`, want: `"select"`},
		{name: `user`, want: `"user"`},
		{name: `order`, want: `"order"`},
		// reserved words get quoted — newly added keywords
		{name: `is`, want: `"is"`},
		{name: `like`, want: `"like"`},
		{name: `ilike`, want: `"ilike"`},
		{name: `join`, want: `"join"`},
		{name: `inner`, want: `"inner"`},
		{name: `outer`, want: `"outer"`},
		{name: `left`, want: `"left"`},
		{name: `right`, want: `"right"`},
		{name: `full`, want: `"full"`},
		{name: `cross`, want: `"cross"`},
		{name: `natural`, want: `"natural"`},
		{name: `similar`, want: `"similar"`},
		{name: `between`, want: `"between"`},
		{name: `overlaps`, want: `"overlaps"`},
		// plain lowercase not reserved — no quotes
		{name: `my_column`, want: `my_column`},
		// underscore-prefixed, all lowercase — no quotes
		{name: `_private`, want: `_private`},
		// digits in name trigger quoting (not lowercase/underscore)
		{name: `col1`, want: `"col1"`},
		// max length: 63 chars is valid
		{name: strings.Repeat("a", 63), want: strings.Repeat("a", 63)},
		// invalid: too long, empty, digit-start, dot
		{name: strings.Repeat("a", 64), wantErr: true},
		{name: ``, wantErr: true},
		{name: `1column`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := QueryFormatter{}.FormatColumnName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryFormatter.FormatColumnName(%#v) error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("QueryFormatter.FormatColumnName(%#v) = %#v, want %#v", tt.name, got, tt.want)
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
		{index: 0, want: "$1"},
		{index: 1, want: "$2"},
		{index: 9, want: "$10"},
		{index: 99, want: "$100"},
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
		{name: "with single quote", str: "it's", want: "'it''s'"},
		// two consecutive single-quote chars in input — each must be escaped independently
		{name: "two consecutive quotes", str: "it''s", want: "'it''''s'"},
		// backslashes are literal with standard_conforming_strings=on (PostgreSQL default)
		{name: "with backslash", str: `path\to`, want: `'path\to'`},
		{name: "with backslash before quote", str: `Erik\'s`, want: `'Erik\''s'`},
		// a raw value that starts/ends with single quotes must have those quotes escaped too
		{name: "value with outer quotes", str: "'hello'", want: "'''hello'''"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.FormatStringLiteral(tt.str); got != tt.want {
				t.Errorf("FormatStringLiteral(%q) = %q, want %q", tt.str, got, tt.want)
			}
		})
	}
}

func TestConnect_InvalidDriver(t *testing.T) {
	config := &sqldb.ConnConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
	}
	_, err := Connect(t.Context(), config)
	if err == nil {
		t.Fatal("expected error for invalid driver")
	}
	// Should mention the expected driver
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}

func TestEscapeIdentifier(t *testing.T) {
	tests := []struct {
		ident string
		want  string
	}{
		{ident: `table`, want: `"table"`},
		{ident: `public.table`, want: `"public.table"`},
		{ident: `my_column`, want: `my_column`},
		// reserved words get quoted
		{ident: `select`, want: `"select"`},
		{ident: `user`, want: `"user"`},
		// uppercase triggers quoting
		{ident: `MyCol`, want: `"MyCol"`},
		// digits trigger quoting
		{ident: `col1`, want: `"col1"`},
		// embedded double quote: escaped and quoted
		{ident: `col"name`, want: `"col""name"`},
		// underscore-prefixed, all lowercase — no quotes
		{ident: `_private`, want: `_private`},
	}
	for _, tt := range tests {
		t.Run(tt.ident, func(t *testing.T) {
			if got := EscapeIdentifier(tt.ident); got != tt.want {
				t.Errorf("EscapeIdentifier(%#v) = %#v, want %#v", tt.ident, got, tt.want)
			}
		})
	}
}
