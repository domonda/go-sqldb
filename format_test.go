package sqldb

import (
	"database/sql/driver"
	"testing"
	"time"
)

type driverValuer struct{}

func (v driverValuer) Value() (driver.Value, error) {
	return "A driver.Valuer", nil
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name    string
		val     any
		want    string
		wantErr bool
	}{
		{name: "nil", val: nil, want: `NULL`},
		{name: "nil string", val: (*string)(nil), want: `NULL`},
		{name: "nil driver.Valuer", val: (driver.Valuer)(nil), want: `NULL`},
		{name: "nil driver.Valuer impl", val: (*driverValuer)(nil), want: `NULL`},
		{name: "driver.Valuer", val: driverValuer{}, want: `'A driver.Valuer'`},
		{name: "driver.Valuer ptr", val: &driverValuer{}, want: `'A driver.Valuer'`},
		{name: "true", val: true, want: `TRUE`},
		{name: "false", val: false, want: `FALSE`},
		{name: "string", val: "Hello World!", want: `'Hello World!'`},
		{name: "string pointer", val: new(string), want: `''`},
		{name: "byte string", val: []byte(`Hello World!`), want: `'Hello World!'`},
		{name: "byte array", val: []byte(`[1,2,3]`), want: `'[1,2,3]'`},
		{name: "string array", val: `[1,2,3]`, want: `'[1,2,3]'`},
		{name: "object array", val: []byte(`[{"a":"foo"},{"b":"bar"}]`), want: `'[{"a":"foo"},{"b":"bar"}]'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatValue(tt.val)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FormatValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatQuery(t *testing.T) {
	query1 := `

	  SELECT *
	  FROM public.user


	  WHERE
	  	name = $3
	  	AND
	  	active = $2
	  	AND
	  	created_at >= $1
	`
	query1formatted := `SELECT *
FROM public.user
WHERE
	name = 'Erik''s Test'
	AND
	active = TRUE
	AND
	created_at >= '2006-01-02 15:04:05.999999+07:00:00'`
	createdAt, err := time.Parse(timeFormat, "'2006-01-02 15:04:05.999999999+07:00:00'")
	if err != nil {
		panic(err)
	}

	query2 := `UPDATE table SET "v1"=$1,v2=$2 ,"v3" = $3`
	query2formatted := `UPDATE table SET "v1"='',v2=2 ,"v3" = '3'`

	tests := []struct {
		name   string
		query  string
		argFmt ParamPlaceholderFormatter
		args   []any
		want   string
	}{
		{name: "query1", query: query1, argFmt: NewParamPlaceholderFormatter("$%d", 1), args: []any{createdAt, true, `Erik's Test`}, want: query1formatted},
		{name: "query2", query: query2, argFmt: NewParamPlaceholderFormatter("$%d", 1), args: []any{"", 2, "3"}, want: query2formatted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatQuery(tt.query, tt.argFmt, tt.args...); got != tt.want {
				t.Errorf("FormatQuery():\n%q\nWant:\n%q", got, tt.want)
			}
		})
	}
}
