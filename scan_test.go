package sqldb

import (
	"database/sql"
	"testing"
	"time"
)

func TestScanDriverValue_Int64Source(t *testing.T) {
	var src int64 = 42

	t.Run("to int", func(t *testing.T) {
		var dest int
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != 42 {
			t.Errorf("got %d, want 42", dest)
		}
	})
	t.Run("to int8", func(t *testing.T) {
		var dest int8
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != 42 {
			t.Errorf("got %d, want 42", dest)
		}
	})
	t.Run("to int16", func(t *testing.T) {
		var dest int16
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != 42 {
			t.Errorf("got %d, want 42", dest)
		}
	})
	t.Run("to int32", func(t *testing.T) {
		var dest int32
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != 42 {
			t.Errorf("got %d, want 42", dest)
		}
	})
	t.Run("to int64", func(t *testing.T) {
		var dest int64
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != 42 {
			t.Errorf("got %d, want 42", dest)
		}
	})
	t.Run("to uint", func(t *testing.T) {
		var dest uint
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != 42 {
			t.Errorf("got %d, want 42", dest)
		}
	})
	t.Run("to uint8", func(t *testing.T) {
		var dest uint8
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != 42 {
			t.Errorf("got %d, want 42", dest)
		}
	})
	t.Run("to uint16", func(t *testing.T) {
		var dest uint16
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != 42 {
			t.Errorf("got %d, want 42", dest)
		}
	})
	t.Run("to uint32", func(t *testing.T) {
		var dest uint32
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != 42 {
			t.Errorf("got %d, want 42", dest)
		}
	})
	t.Run("to uint64", func(t *testing.T) {
		var dest uint64
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != 42 {
			t.Errorf("got %d, want 42", dest)
		}
	})
	t.Run("to float32", func(t *testing.T) {
		var dest float32
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != 42 {
			t.Errorf("got %f, want 42", dest)
		}
	})
	t.Run("to float64", func(t *testing.T) {
		var dest float64
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != 42 {
			t.Errorf("got %f, want 42", dest)
		}
	})
}

func TestScanDriverValue_Float64Source(t *testing.T) {
	var src float64 = 3.14

	t.Run("to float32", func(t *testing.T) {
		var dest float32
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != float32(3.14) {
			t.Errorf("got %f, want 3.14", dest)
		}
	})
	t.Run("to float64", func(t *testing.T) {
		var dest float64
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != 3.14 {
			t.Errorf("got %f, want 3.14", dest)
		}
	})
	t.Run("to int", func(t *testing.T) {
		var dest int
		if err := ScanDriverValue(&dest, float64(7)); err != nil {
			t.Fatal(err)
		}
		if dest != 7 {
			t.Errorf("got %d, want 7", dest)
		}
	})
	t.Run("to int64", func(t *testing.T) {
		var dest int64
		if err := ScanDriverValue(&dest, float64(99)); err != nil {
			t.Fatal(err)
		}
		if dest != 99 {
			t.Errorf("got %d, want 99", dest)
		}
	})
	t.Run("to uint", func(t *testing.T) {
		var dest uint
		if err := ScanDriverValue(&dest, float64(5)); err != nil {
			t.Fatal(err)
		}
		if dest != 5 {
			t.Errorf("got %d, want 5", dest)
		}
	})
	t.Run("to uint64", func(t *testing.T) {
		var dest uint64
		if err := ScanDriverValue(&dest, float64(100)); err != nil {
			t.Fatal(err)
		}
		if dest != 100 {
			t.Errorf("got %d, want 100", dest)
		}
	})
}

func TestScanDriverValue_BoolSource(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		var dest bool
		if err := ScanDriverValue(&dest, true); err != nil {
			t.Fatal(err)
		}
		if !dest {
			t.Error("got false, want true")
		}
	})
	t.Run("false", func(t *testing.T) {
		var dest bool
		dest = true
		if err := ScanDriverValue(&dest, false); err != nil {
			t.Fatal(err)
		}
		if dest {
			t.Error("got true, want false")
		}
	})
}

func TestScanDriverValue_ByteSliceSource(t *testing.T) {
	src := []byte("hello")

	t.Run("to string", func(t *testing.T) {
		var dest string
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != "hello" {
			t.Errorf("got %q, want %q", dest, "hello")
		}
	})
	t.Run("to []byte copy", func(t *testing.T) {
		var dest []byte
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if string(dest) != "hello" {
			t.Errorf("got %q, want %q", dest, "hello")
		}
		// Verify it's a copy by mutating the source
		src[0] = 'X'
		if dest[0] == 'X' {
			t.Error("dest shares memory with src, expected a copy")
		}
	})
}

func TestScanDriverValue_StringSource(t *testing.T) {
	t.Run("to string", func(t *testing.T) {
		var dest string
		if err := ScanDriverValue(&dest, "world"); err != nil {
			t.Fatal(err)
		}
		if dest != "world" {
			t.Errorf("got %q, want %q", dest, "world")
		}
	})
	t.Run("to []byte", func(t *testing.T) {
		var dest []byte
		if err := ScanDriverValue(&dest, "world"); err != nil {
			t.Fatal(err)
		}
		if string(dest) != "world" {
			t.Errorf("got %q, want %q", dest, "world")
		}
	})
}

func TestScanDriverValue_TimeSource(t *testing.T) {
	src := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	var dest time.Time
	if err := ScanDriverValue(&dest, src); err != nil {
		t.Fatal(err)
	}
	if !dest.Equal(src) {
		t.Errorf("got %v, want %v", dest, src)
	}
}

func TestScanDriverValue_NilSource(t *testing.T) {
	t.Run("pointer set to nil", func(t *testing.T) {
		s := "not nil"
		dest := &s
		if err := ScanDriverValue(&dest, nil); err != nil {
			t.Fatal(err)
		}
		if dest != nil {
			t.Error("expected nil pointer")
		}
	})
	t.Run("slice set to nil", func(t *testing.T) {
		dest := []byte("not nil")
		if err := ScanDriverValue(&dest, nil); err != nil {
			t.Fatal(err)
		}
		if dest != nil {
			t.Error("expected nil slice")
		}
	})
	t.Run("map set to nil", func(t *testing.T) {
		dest := map[string]int{"a": 1}
		if err := ScanDriverValue(&dest, nil); err != nil {
			t.Fatal(err)
		}
		if dest != nil {
			t.Error("expected nil map")
		}
	})
	t.Run("SetNull interface", func(t *testing.T) {
		var dest nullSetter
		if err := ScanDriverValue(&dest, nil); err != nil {
			t.Fatal(err)
		}
		if !dest.wasSetNull {
			t.Error("expected SetNull to be called")
		}
	})
}

type nullSetter struct {
	wasSetNull bool
}

func (n *nullSetter) SetNull() {
	n.wasSetNull = true
}

func TestScanDriverValue_InterfaceDest(t *testing.T) {
	t.Run("non-nil value", func(t *testing.T) {
		var dest any
		if err := ScanDriverValue(&dest, int64(123)); err != nil {
			t.Fatal(err)
		}
		if dest != int64(123) {
			t.Errorf("got %v, want 123", dest)
		}
	})
	t.Run("nil value sets zero", func(t *testing.T) {
		var dest any = "something"
		if err := ScanDriverValue(&dest, nil); err != nil {
			t.Fatal(err)
		}
		if dest != nil {
			t.Errorf("got %v, want nil", dest)
		}
	})
}

func TestScanDriverValue_SQLScanner(t *testing.T) {
	var dest sql.NullString
	if err := ScanDriverValue(&dest, "hello"); err != nil {
		t.Fatal(err)
	}
	if !dest.Valid || dest.String != "hello" {
		t.Errorf("got %+v, want {String:hello Valid:true}", dest)
	}
}

func TestScanDriverValue_Errors(t *testing.T) {
	t.Run("nil destPtr", func(t *testing.T) {
		err := ScanDriverValue(nil, int64(1))
		if err == nil {
			t.Error("expected error for nil destPtr")
		}
	})
	t.Run("non-pointer destPtr", func(t *testing.T) {
		err := ScanDriverValue(42, int64(1))
		if err == nil {
			t.Error("expected error for non-pointer")
		}
	})
	t.Run("incompatible types", func(t *testing.T) {
		var dest bool
		err := ScanDriverValue(&dest, int64(1))
		if err == nil {
			t.Error("expected error for incompatible types")
		}
	})
}
