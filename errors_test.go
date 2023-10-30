package sqldb

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
)

func TestWrapErrorWithQuery(t *testing.T) {
	type args struct {
		err    error
		query  string
		argFmt string
		args   []any
	}
	tests := []struct {
		name      string
		args      args
		wantError string
	}{
		{name: "nil", args: args{err: nil}, wantError: ""},
		{
			name: "select no rows",
			args: args{
				err:    sql.ErrNoRows,
				query:  `SELECT * FROM table WHERE b = $2 and a = $1`,
				argFmt: "$%d",
				args:   []any{1, "2"},
			},
			wantError: fmt.Sprintf("%s from query: %s", sql.ErrNoRows, `SELECT * FROM table WHERE b = '2' and a = 1`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WrapErrorWithQuery(tt.args.err, tt.args.query, tt.args.args, defaultQueryFormatter{})
			if tt.wantError == "" && err != nil || tt.wantError != "" && (err == nil || err.Error() != tt.wantError) {
				t.Errorf("WrapNonNilErrorWithQuery() error = %v, wantErr %v", err, tt.wantError)
			}
			if !errors.Is(err, tt.args.err) {
				t.Errorf("WrapNonNilErrorWithQuery() error = %v does not wrap %v", err, tt.args.err)
			}
		})
	}
}