package impl

import (
	"testing"
	"time"
)

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name    string
		val     interface{}
		want    string
		wantErr bool
	}{
		{name: "nil", val: nil, want: "NULL"},
		{name: "true", val: true, want: "TRUE"},
		{name: "false", val: false, want: "FALSE"},
		// TODO
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
	query1form := `SELECT *
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

	tests := []struct {
		name  string
		query string
		args  []interface{}
		want  string
	}{
		{name: "query1", query: query1, args: []interface{}{createdAt, true, `Erik's Test`}, want: query1form},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatQuery(tt.query, tt.args...); got != tt.want {
				t.Errorf("FormatQuery():\n%q\nWant:\n%q", got, tt.want)
			}
		})
	}
}
