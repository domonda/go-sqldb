package sqldb

import (
	"database/sql/driver"
	"fmt"
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

func TestNormalizeAndFormatQuery(t *testing.T) {
	t.Run("nil normalize and formatter uses defaults", func(t *testing.T) {
		result, err := NormalizeAndFormatQuery(nil, nil, "SELECT * FROM t WHERE id = $1", 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "SELECT * FROM t WHERE id = 42" {
			t.Errorf("got %q, want %q", result, "SELECT * FROM t WHERE id = 42")
		}
	})

	t.Run("with custom formatter", func(t *testing.T) {
		formatter := NewQueryFormatter("$")
		result, err := NormalizeAndFormatQuery(nil, formatter, "SELECT $1, $2", "a", "b")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "SELECT 'a', 'b'" {
			t.Errorf("got %q, want %q", result, "SELECT 'a', 'b'")
		}
	})

	t.Run("normalize error propagates", func(t *testing.T) {
		badNormalize := func(query string) (string, error) {
			return "", fmt.Errorf("normalize failed")
		}
		_, err := NormalizeAndFormatQuery(badNormalize, nil, "SELECT 1")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestMustNormalizeAndFormatQuery(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		result := MustNormalizeAndFormatQuery(nil, nil, "SELECT $1", 42)
		if result != "SELECT 42" {
			t.Errorf("got %q, want %q", result, "SELECT 42")
		}
	})

	t.Run("panics on error", func(t *testing.T) {
		badNormalize := func(query string) (string, error) {
			return "", fmt.Errorf("normalize failed")
		}
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic")
			}
		}()
		MustNormalizeAndFormatQuery(badNormalize, nil, "SELECT 1")
	})
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
	created_at >= '2006-01-02 15:04:05.999999+07:00'`
	createdAt, err := time.Parse(timeFormat, "'2006-01-02 15:04:05.999999999+07:00'")
	if err != nil {
		panic(err)
	}

	query2 := `UPDATE table SET "v1"=$1,v2=$2 ,"v3" = $3`
	query2formatted := `UPDATE table SET "v1"='',v2=2 ,"v3" = '3'`

	tests := []struct {
		name   string
		argFmt QueryFormatter
		query  string
		args   []any
		want   string
	}{
		{name: "query1", query: query1, argFmt: NewQueryFormatter("$"), args: []any{createdAt, true, `Erik's Test`}, want: query1formatted},
		{name: "query2", query: query2, argFmt: NewQueryFormatter("$"), args: []any{"", 2, "3"}, want: query2formatted},
		{name: "uniform placeholders", query: "SELECT * FROM t WHERE a = ? AND b = ?", argFmt: StdQueryFormatter{}, args: []any{"x", 42}, want: "SELECT * FROM t WHERE a = 'x' AND b = 42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatQuery(tt.argFmt, tt.query, tt.args...); got != tt.want {
				t.Errorf("FormatQuery():\n%q\nWant:\n%q", got, tt.want)
			}
		})
	}
}
