package sqldb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
