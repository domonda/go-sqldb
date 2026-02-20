package sqldb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStdQueryFormatter_FormatTableName(t *testing.T) {
	f := StdQueryFormatter{}
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{name: "users", want: "users"},
		{name: "my_table", want: "my_table"},
		{name: "public.users", want: "public.users"},
		{name: "_private", want: "_private"},
		{name: "table123", want: "table123"},
		// Invalid table names
		{name: "", wantErr: true},
		{name: "123start", wantErr: true},
		{name: "has space", wantErr: true},
		{name: "has-dash", wantErr: true},
		{name: "two..dots", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.FormatTableName(tt.name)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tt.name)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStdQueryFormatter_FormatColumnName(t *testing.T) {
	f := StdQueryFormatter{}
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{name: "id", want: "id"},
		{name: "user_name", want: "user_name"},
		{name: "_col", want: "_col"},
		{name: "Col123", want: "Col123"},
		// Invalid column names
		{name: "", wantErr: true},
		{name: "123start", wantErr: true},
		{name: "has space", wantErr: true},
		{name: "has.dot", wantErr: true},
		{name: "has-dash", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.FormatColumnName(tt.name)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tt.name)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStdQueryFormatter_FormatPlaceholder(t *testing.T) {
	t.Run("PostgreSQL style", func(t *testing.T) {
		f := NewQueryFormatter("$")
		tests := []struct {
			index int
			want  string
		}{
			{index: 0, want: "$1"},
			{index: 1, want: "$2"},
			{index: 9, want: "$10"},
		}
		for _, tt := range tests {
			if got := f.FormatPlaceholder(tt.index); got != tt.want {
				t.Errorf("FormatPlaceholder(%d) = %q, want %q", tt.index, got, tt.want)
			}
		}
	})

	t.Run("SQL Server style", func(t *testing.T) {
		f := NewQueryFormatter("@p")
		if got := f.FormatPlaceholder(0); got != "@p1" {
			t.Errorf("got %q, want @p1", got)
		}
		if got := f.FormatPlaceholder(2); got != "@p3" {
			t.Errorf("got %q, want @p3", got)
		}
	})

	t.Run("MySQL style (uniform ?)", func(t *testing.T) {
		f := StdQueryFormatter{} // empty prefix
		if got := f.FormatPlaceholder(0); got != "?" {
			t.Errorf("got %q, want ?", got)
		}
		if got := f.FormatPlaceholder(5); got != "?" {
			t.Errorf("got %q, want ?", got)
		}
	})

	t.Run("negative index panics", func(t *testing.T) {
		f := NewQueryFormatter("$")
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for negative index")
			}
		}()
		f.FormatPlaceholder(-1)
	})
}

func TestStdQueryFormatter_FormatStringLiteral(t *testing.T) {
	f := StdQueryFormatter{}
	tests := []struct {
		name string
		str  string
		want string
	}{
		{name: "simple", str: "hello", want: "'hello'"},
		{name: "empty", str: "", want: "''"},
		{name: "with single quote", str: "it's", want: "'it''s'"},
		{name: "with backslash", str: `path\to`, want: `'path\to'`},
		{name: "with backslash quote", str: `Erik\'s`, want: `'Erik''s'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.FormatStringLiteral(tt.str); got != tt.want {
				t.Errorf("FormatStringLiteral(%q) = %q, want %q", tt.str, got, tt.want)
			}
		})
	}
}

func TestFormatSingleQuoteStringLiteral(t *testing.T) {
	tests := []struct {
		str  string
		want string
	}{
		{str: ``, want: `''`},
		{str: `''`, want: `''`},
		{str: `'''`, want: `''''`},
		{str: `'a'b'c'`, want: `'a''b''c'`},
		{str: `'Hello`, want: `'''Hello'`},
		{str: `World'`, want: `'World'''`},
		{str: `Erik's String`, want: `'Erik''s String'`},
		{str: `'Erik's String'`, want: `'Erik''s String'`},
		{str: `'Erik''s String'`, want: `'Erik''s String'`},
		{str: `Erik\'s String`, want: `'Erik''s String'`},
		{str: `'Erik\'s String'`, want: `'Erik''s String'`},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			got := FormatSingleQuoteStringLiteral(tt.str)
			require.Equal(t, tt.want, got, "FormatSingleQuoteStringLiteral(%#v)", tt.str)
		})
	}
}
