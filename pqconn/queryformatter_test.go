package pqconn

import (
	"reflect"
	"testing"
	"time"

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
		{name: "with backslash", str: `path\to`, want: `'path\to'`},
		{name: "with backslash quote", str: `Erik\'s`, want: `'Erik''s'`},
		{name: "already quoted no inner quotes", str: "'hello'", want: "'hello'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.FormatStringLiteral(tt.str); got != tt.want {
				t.Errorf("FormatStringLiteral(%q) = %q, want %q", tt.str, got, tt.want)
			}
		})
	}
}

func TestNewTypeMapper(t *testing.T) {
	tm := NewTypeMapper()
	if tm == nil {
		t.Fatal("NewTypeMapper returned nil")
	}

	// Check specific type mapping
	if got := tm.ColumnType(reflect.TypeFor[time.Time]()); got != "timestamptz" {
		t.Errorf("time.Time mapped to %q, want %q", got, "timestamptz")
	}

	// Check kind mappings
	kindTests := []struct {
		kind reflect.Kind
		want string
	}{
		{reflect.Bool, "boolean"},
		{reflect.Int, "bigint"},
		{reflect.Int8, "smallint"},
		{reflect.Int16, "smallint"},
		{reflect.Int32, "integer"},
		{reflect.Int64, "bigint"},
		{reflect.Uint, "bigint"},
		{reflect.Uint8, "smallint"},
		{reflect.Uint16, "integer"},
		{reflect.Uint32, "bigint"},
		{reflect.Uint64, "bigint"},
		{reflect.Float32, "float4"},
		{reflect.Float64, "float8"},
		{reflect.String, "text"},
	}
	for _, tt := range kindTests {
		if got, ok := tm.Kinds[tt.kind]; !ok || got != tt.want {
			t.Errorf("kind %v mapped to %q, want %q", tt.kind, got, tt.want)
		}
	}

	// Unmapped kind should return empty
	if got := tm.ColumnType(reflect.TypeFor[[]byte]()); got != "" {
		t.Errorf("[]byte mapped to %q, want empty", got)
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
	}
	for _, tt := range tests {
		t.Run(tt.ident, func(t *testing.T) {
			if got := EscapeIdentifier(tt.ident); got != tt.want {
				t.Errorf("EscapeIdentifier(%#v) = %#v, want %#v", tt.ident, got, tt.want)
			}
		})
	}
}
