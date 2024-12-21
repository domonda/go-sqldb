package db

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/domonda/go-sqldb"
)

func Test_wrapErrorWithQuery(t *testing.T) {
	type args struct {
		err      error
		query    string
		args     []any
		queryFmt sqldb.QueryFormatter
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
				err:      sql.ErrNoRows,
				query:    `SELECT * FROM table WHERE b = $2 AND a = $1`,
				queryFmt: sqldb.StdQueryFormatter{PlaceholderFmt: "$%d"},
				args:     []any{1, "2"},
			},
			wantError: fmt.Sprintf("%s from query: %s", sql.ErrNoRows, `SELECT * FROM table WHERE b = '2' AND a = 1`),
		},
		{
			name: "multi line",
			args: args{
				err: sql.ErrNoRows,
				query: `
					SELECT *
					FROM table
					WHERE b = $2
						AND a = $1`,
				queryFmt: sqldb.StdQueryFormatter{PlaceholderFmt: "$%d"},
				args:     []any{1, "2"},
			},
			wantError: fmt.Sprintf(
				"%s from query: %s",
				sql.ErrNoRows,
				`SELECT *
FROM table
WHERE b = '2'
	AND a = 1`,
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wrapErrorWithQuery(tt.args.err, tt.args.query, tt.args.args, tt.args.queryFmt)
			if tt.wantError == "" && err != nil || tt.wantError != "" && (err == nil || err.Error() != tt.wantError) {
				t.Errorf("wrapErrorWithQuery() error = \n%s\nwantErr\n%s", err, tt.wantError)
			}
			if !errors.Is(err, tt.args.err) {
				t.Errorf("wrapErrorWithQuery() error = %v does not wrap %v", err, tt.args.err)
			}
		})
	}
}
