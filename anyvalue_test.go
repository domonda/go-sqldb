package sqldb

import (
	"database/sql/driver"
	"testing"
)

func TestAnyValue_Scan(t *testing.T) {
	t.Run("non-nil value", func(t *testing.T) {
		var av AnyValue
		err := av.Scan("hello")
		if err != nil {
			t.Fatal(err)
		}
		if av.Val != "hello" {
			t.Errorf("Val = %v, want %q", av.Val, "hello")
		}
	})

	t.Run("nil value", func(t *testing.T) {
		av := AnyValue{Val: "old"}
		err := av.Scan(nil)
		if err != nil {
			t.Fatal(err)
		}
		if av.Val != nil {
			t.Errorf("Val = %v, want nil", av.Val)
		}
	})

	t.Run("byte slice is copied", func(t *testing.T) {
		var av AnyValue
		src := []byte("original")
		err := av.Scan(src)
		if err != nil {
			t.Fatal(err)
		}
		// Modify source to verify copy
		src[0] = 'X'
		b, ok := av.Val.([]byte)
		if !ok {
			t.Fatalf("Val is %T, want []byte", av.Val)
		}
		if string(b) != "original" {
			t.Errorf("Val = %q, want %q (byte slice was not copied)", b, "original")
		}
	})

	t.Run("int64 value", func(t *testing.T) {
		var av AnyValue
		err := av.Scan(int64(42))
		if err != nil {
			t.Fatal(err)
		}
		if av.Val != int64(42) {
			t.Errorf("Val = %v, want 42", av.Val)
		}
	})
}

func TestAnyValue_Value(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		av := AnyValue{Val: "test"}
		v, err := av.Value()
		if err != nil {
			t.Fatal(err)
		}
		if v != "test" {
			t.Errorf("Value() = %v, want %q", v, "test")
		}
	})

	t.Run("nil value", func(t *testing.T) {
		av := AnyValue{Val: nil}
		v, err := av.Value()
		if err != nil {
			t.Fatal(err)
		}
		if v != nil {
			t.Errorf("Value() = %v, want nil", v)
		}
	})

	t.Run("implements driver.Valuer", func(t *testing.T) {
		var _ driver.Valuer = AnyValue{}
	})
}

func TestAnyValue_String(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string
	}{
		{name: "string", val: "hello", want: "hello"},
		{name: "int64", val: int64(42), want: "42"},
		{name: "nil", val: nil, want: "<nil>"},
		{name: "valid utf8 bytes", val: []byte("utf8"), want: "utf8"},
		{name: "bool", val: true, want: "true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			av := AnyValue{Val: tt.val}
			if got := av.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAnyValue_GoString(t *testing.T) {
	t.Run("valid utf8 bytes", func(t *testing.T) {
		av := AnyValue{Val: []byte("hello")}
		got := av.GoString()
		want := `[]byte("hello")`
		if got != want {
			t.Errorf("GoString() = %q, want %q", got, want)
		}
	})

	t.Run("string value", func(t *testing.T) {
		av := AnyValue{Val: "test"}
		got := av.GoString()
		want := `"test"`
		if got != want {
			t.Errorf("GoString() = %q, want %q", got, want)
		}
	})
}
