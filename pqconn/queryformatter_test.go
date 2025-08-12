package pqconn

import (
	"testing"
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
