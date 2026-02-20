package sqldb

import (
	"bytes"
	"database/sql"
	"testing"
)

func TestTxOptionsString(t *testing.T) {
	tests := []struct {
		name string
		opts *sql.TxOptions
		want string
	}{
		{
			name: "nil options",
			opts: nil,
			want: "",
		},
		{
			name: "default options",
			opts: &sql.TxOptions{},
			want: "",
		},
		{
			name: "read-only default isolation",
			opts: &sql.TxOptions{ReadOnly: true},
			want: "Read-Only",
		},
		{
			name: "read-only with isolation",
			opts: &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelSerializable},
			want: "Read-Only Serializable",
		},
		{
			name: "isolation only",
			opts: &sql.TxOptions{Isolation: sql.LevelRepeatableRead},
			want: "Repeatable Read",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TxOptionsString(tt.opts); got != tt.want {
				t.Errorf("TxOptionsString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFprintTable(t *testing.T) {
	t.Run("basic table", func(t *testing.T) {
		rows := [][]string{
			{"Name", "Age"},
			{"Alice", "30"},
			{"Bob", "7"},
		}
		var buf bytes.Buffer
		err := FprintTable(&buf, rows, "|")
		if err != nil {
			t.Fatal(err)
		}
		want := "Name |Age\nAlice|30 \nBob  |7  \n"
		if buf.String() != want {
			t.Errorf("got:\n%s\nwant:\n%s", buf.String(), want)
		}
	})

	t.Run("unicode padding", func(t *testing.T) {
		rows := [][]string{
			{"Ä", "B"},
			{"CD", "E"},
		}
		var buf bytes.Buffer
		err := FprintTable(&buf, rows, "|")
		if err != nil {
			t.Fatal(err)
		}
		want := "Ä |B\nCD|E\n"
		if buf.String() != want {
			t.Errorf("got:\n%s\nwant:\n%s", buf.String(), want)
		}
	})

	t.Run("uneven row lengths", func(t *testing.T) {
		rows := [][]string{
			{"A", "B", "C"},
			{"D"},
		}
		var buf bytes.Buffer
		err := FprintTable(&buf, rows, "|")
		if err != nil {
			t.Fatal(err)
		}
		// Second row should be padded with empty cells
		want := "A|B|C\nD| | \n"
		if buf.String() != want {
			t.Errorf("got:\n%s\nwant:\n%s", buf.String(), want)
		}
	})

	t.Run("single column", func(t *testing.T) {
		rows := [][]string{
			{"hello"},
			{"world"},
		}
		var buf bytes.Buffer
		err := FprintTable(&buf, rows, "|")
		if err != nil {
			t.Fatal(err)
		}
		want := "hello\nworld\n"
		if buf.String() != want {
			t.Errorf("got:\n%s\nwant:\n%s", buf.String(), want)
		}
	})

	t.Run("empty rows", func(t *testing.T) {
		var buf bytes.Buffer
		err := FprintTable(&buf, nil, "|")
		if err != nil {
			t.Fatal(err)
		}
		if buf.String() != "" {
			t.Errorf("expected empty output, got %q", buf.String())
		}
	})
}
