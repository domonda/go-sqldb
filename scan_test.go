package sqldb

import (
	"database/sql"
	"fmt"
	"io"
	"math"
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
		var dest int
		err := ScanDriverValue(&dest, true)
		if err == nil {
			t.Error("expected error scanning bool into int")
		}
	})
}

func TestScanDriverValue_Int64Overflow(t *testing.T) {
	t.Run("overflows int8", func(t *testing.T) {
		var dest int8
		if err := ScanDriverValue(&dest, int64(200)); err == nil {
			t.Error("expected overflow error, got nil")
		}
	})
	t.Run("overflows uint8", func(t *testing.T) {
		var dest uint8
		if err := ScanDriverValue(&dest, int64(256)); err == nil {
			t.Error("expected overflow error, got nil")
		}
	})
	t.Run("negative into unsigned", func(t *testing.T) {
		var dest uint32
		if err := ScanDriverValue(&dest, int64(-1)); err == nil {
			t.Error("expected error for negative into unsigned, got nil")
		}
	})
	t.Run("max int8 fits", func(t *testing.T) {
		var dest int8
		if err := ScanDriverValue(&dest, int64(127)); err != nil {
			t.Fatal(err)
		}
		if dest != 127 {
			t.Errorf("got %d, want 127", dest)
		}
	})
}

func TestScanDriverValue_Float64Loss(t *testing.T) {
	t.Run("fractional into int", func(t *testing.T) {
		var dest int
		if err := ScanDriverValue(&dest, 3.5); err == nil {
			t.Error("expected error scanning fractional float into int, got nil")
		}
	})
	t.Run("fractional into uint", func(t *testing.T) {
		var dest uint
		if err := ScanDriverValue(&dest, 3.5); err == nil {
			t.Error("expected error scanning fractional float into uint, got nil")
		}
	})
	t.Run("overflows int8", func(t *testing.T) {
		var dest int8
		if err := ScanDriverValue(&dest, float64(1000)); err == nil {
			t.Error("expected overflow error, got nil")
		}
	})
	t.Run("negative into uint", func(t *testing.T) {
		var dest uint
		if err := ScanDriverValue(&dest, float64(-1)); err == nil {
			t.Error("expected error for negative float into uint, got nil")
		}
	})
	t.Run("overflows float32", func(t *testing.T) {
		var dest float32
		if err := ScanDriverValue(&dest, 1e300); err == nil {
			t.Error("expected overflow error scanning into float32, got nil")
		}
	})
	t.Run("whole float into int", func(t *testing.T) {
		var dest int
		if err := ScanDriverValue(&dest, float64(42)); err != nil {
			t.Fatal(err)
		}
		if dest != 42 {
			t.Errorf("got %d, want 42", dest)
		}
	})
}

func TestScanDriverValue_InterfaceDest_NonEmpty(t *testing.T) {
	t.Run("not assignable returns error", func(t *testing.T) {
		var dest io.Reader
		err := ScanDriverValue(&dest, int64(1))
		if err == nil {
			t.Error("expected error scanning int64 into io.Reader, got nil")
		}
		if dest != nil {
			t.Errorf("dest should remain nil, got %v", dest)
		}
	})
	t.Run("assignable value is set", func(t *testing.T) {
		var dest fmt.Stringer
		src := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest == nil || dest.String() != src.String() {
			t.Errorf("got %v, want %v", dest, src)
		}
	})
}

func TestScanDriverValue_PointerToPointer(t *testing.T) {
	t.Run("nil pointer allocated and scanned", func(t *testing.T) {
		var dest *string
		if err := ScanDriverValue(&dest, "hello"); err != nil {
			t.Fatal(err)
		}
		if dest == nil || *dest != "hello" {
			t.Errorf("got %v, want \"hello\"", dest)
		}
	})
	t.Run("existing pointer reused", func(t *testing.T) {
		s := "old"
		dest := &s
		if err := ScanDriverValue(&dest, "new"); err != nil {
			t.Fatal(err)
		}
		if dest != &s {
			t.Error("expected existing pointer to be reused")
		}
		if *dest != "new" {
			t.Errorf("got %q, want %q", *dest, "new")
		}
	})
	t.Run("int64 into nil int pointer", func(t *testing.T) {
		var dest *int64
		if err := ScanDriverValue(&dest, int64(7)); err != nil {
			t.Fatal(err)
		}
		if dest == nil || *dest != 7 {
			t.Errorf("got %v, want 7", dest)
		}
	})
	t.Run("incompatible value still errors", func(t *testing.T) {
		var dest *int
		if err := ScanDriverValue(&dest, true); err == nil {
			t.Error("expected error scanning bool into *int, got nil")
		}
	})
}

func TestScanDriverValue_Int64ToBoolStringBytes(t *testing.T) {
	t.Run("0 to bool false", func(t *testing.T) {
		var dest bool
		if err := ScanDriverValue(&dest, int64(0)); err != nil {
			t.Fatal(err)
		}
		if dest {
			t.Error("got true, want false")
		}
	})
	t.Run("1 to bool true", func(t *testing.T) {
		var dest bool
		if err := ScanDriverValue(&dest, int64(1)); err != nil {
			t.Fatal(err)
		}
		if !dest {
			t.Error("got false, want true")
		}
	})
	t.Run("2 to bool errors", func(t *testing.T) {
		var dest bool
		if err := ScanDriverValue(&dest, int64(2)); err == nil {
			t.Error("expected error scanning int64(2) into bool")
		}
	})
	t.Run("to string", func(t *testing.T) {
		var dest string
		if err := ScanDriverValue(&dest, int64(42)); err != nil {
			t.Fatal(err)
		}
		if dest != "42" {
			t.Errorf("got %q, want %q", dest, "42")
		}
	})
	t.Run("to []byte", func(t *testing.T) {
		var dest []byte
		if err := ScanDriverValue(&dest, int64(42)); err != nil {
			t.Fatal(err)
		}
		if string(dest) != "42" {
			t.Errorf("got %q, want %q", dest, "42")
		}
	})
}

func TestScanDriverValue_Float64ToBoolStringBytes(t *testing.T) {
	t.Run("0 to bool false", func(t *testing.T) {
		var dest bool
		if err := ScanDriverValue(&dest, float64(0)); err != nil {
			t.Fatal(err)
		}
		if dest {
			t.Error("got true, want false")
		}
	})
	t.Run("1 to bool true", func(t *testing.T) {
		var dest bool
		if err := ScanDriverValue(&dest, float64(1)); err != nil {
			t.Fatal(err)
		}
		if !dest {
			t.Error("got false, want true")
		}
	})
	t.Run("0.5 to bool errors", func(t *testing.T) {
		var dest bool
		if err := ScanDriverValue(&dest, float64(0.5)); err == nil {
			t.Error("expected error scanning float64(0.5) into bool")
		}
	})
	t.Run("to string", func(t *testing.T) {
		var dest string
		if err := ScanDriverValue(&dest, float64(3.14)); err != nil {
			t.Fatal(err)
		}
		if dest != "3.14" {
			t.Errorf("got %q, want %q", dest, "3.14")
		}
	})
	t.Run("to []byte", func(t *testing.T) {
		var dest []byte
		if err := ScanDriverValue(&dest, float64(3.14)); err != nil {
			t.Fatal(err)
		}
		if string(dest) != "3.14" {
			t.Errorf("got %q, want %q", dest, "3.14")
		}
	})
}

func TestScanDriverValue_BoolToStringBytesNumeric(t *testing.T) {
	t.Run("true to string", func(t *testing.T) {
		var dest string
		if err := ScanDriverValue(&dest, true); err != nil {
			t.Fatal(err)
		}
		if dest != "true" {
			t.Errorf("got %q, want %q", dest, "true")
		}
	})
	t.Run("false to string", func(t *testing.T) {
		var dest string
		if err := ScanDriverValue(&dest, false); err != nil {
			t.Fatal(err)
		}
		if dest != "false" {
			t.Errorf("got %q, want %q", dest, "false")
		}
	})
	t.Run("to []byte", func(t *testing.T) {
		var dest []byte
		if err := ScanDriverValue(&dest, true); err != nil {
			t.Fatal(err)
		}
		if string(dest) != "true" {
			t.Errorf("got %q, want %q", dest, "true")
		}
	})
	t.Run("to int errors", func(t *testing.T) {
		var dest int
		if err := ScanDriverValue(&dest, true); err == nil {
			t.Error("expected error scanning bool into int")
		}
	})
}

func TestScanDriverValue_StringToBoolNumeric(t *testing.T) {
	t.Run("to bool true", func(t *testing.T) {
		var dest bool
		if err := ScanDriverValue(&dest, "true"); err != nil {
			t.Fatal(err)
		}
		if !dest {
			t.Error("got false, want true")
		}
	})
	t.Run("to bool from 1", func(t *testing.T) {
		var dest bool
		if err := ScanDriverValue(&dest, "1"); err != nil {
			t.Fatal(err)
		}
		if !dest {
			t.Error("got false, want true")
		}
	})
	t.Run("to bool invalid errors", func(t *testing.T) {
		var dest bool
		if err := ScanDriverValue(&dest, "notabool"); err == nil {
			t.Error(`expected error scanning "notabool" into bool`)
		}
	})
	t.Run("to int", func(t *testing.T) {
		var dest int
		if err := ScanDriverValue(&dest, "123"); err != nil {
			t.Fatal(err)
		}
		if dest != 123 {
			t.Errorf("got %d, want 123", dest)
		}
	})
	t.Run("to int fractional errors", func(t *testing.T) {
		var dest int
		if err := ScanDriverValue(&dest, "12.5"); err == nil {
			t.Error(`expected error scanning "12.5" into int`)
		}
	})
	t.Run("to int overflow errors", func(t *testing.T) {
		var dest int8
		if err := ScanDriverValue(&dest, "1000"); err == nil {
			t.Error(`expected overflow error scanning "1000" into int8`)
		}
	})
	t.Run("to uint", func(t *testing.T) {
		var dest uint
		if err := ScanDriverValue(&dest, "123"); err != nil {
			t.Fatal(err)
		}
		if dest != 123 {
			t.Errorf("got %d, want 123", dest)
		}
	})
	t.Run("to uint negative errors", func(t *testing.T) {
		var dest uint
		if err := ScanDriverValue(&dest, "-1"); err == nil {
			t.Error(`expected error scanning "-1" into uint`)
		}
	})
	t.Run("to float", func(t *testing.T) {
		var dest float64
		if err := ScanDriverValue(&dest, "3.14"); err != nil {
			t.Fatal(err)
		}
		if dest != 3.14 {
			t.Errorf("got %g, want 3.14", dest)
		}
	})
	t.Run("to float invalid errors", func(t *testing.T) {
		var dest float64
		if err := ScanDriverValue(&dest, "abc"); err == nil {
			t.Error(`expected error scanning "abc" into float64`)
		}
	})
}

func TestScanDriverValue_ByteSliceToBoolNumeric(t *testing.T) {
	t.Run("to bool", func(t *testing.T) {
		var dest bool
		if err := ScanDriverValue(&dest, []byte("true")); err != nil {
			t.Fatal(err)
		}
		if !dest {
			t.Error("got false, want true")
		}
	})
	t.Run("to int", func(t *testing.T) {
		var dest int
		if err := ScanDriverValue(&dest, []byte("123")); err != nil {
			t.Fatal(err)
		}
		if dest != 123 {
			t.Errorf("got %d, want 123", dest)
		}
	})
	t.Run("to uint", func(t *testing.T) {
		var dest uint64
		if err := ScanDriverValue(&dest, []byte("123")); err != nil {
			t.Fatal(err)
		}
		if dest != 123 {
			t.Errorf("got %d, want 123", dest)
		}
	})
	t.Run("to float", func(t *testing.T) {
		var dest float64
		if err := ScanDriverValue(&dest, []byte("3.14")); err != nil {
			t.Fatal(err)
		}
		if dest != 3.14 {
			t.Errorf("got %g, want 3.14", dest)
		}
	})
	t.Run("to int invalid errors", func(t *testing.T) {
		var dest int
		if err := ScanDriverValue(&dest, []byte("xyz")); err == nil {
			t.Error(`expected error scanning []byte("xyz") into int`)
		}
	})
}

func TestScanDriverValue_TimeToStringBytes(t *testing.T) {
	src := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	want := src.Format(time.RFC3339Nano)

	t.Run("to string", func(t *testing.T) {
		var dest string
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if dest != want {
			t.Errorf("got %q, want %q", dest, want)
		}
	})
	t.Run("to []byte", func(t *testing.T) {
		var dest []byte
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if string(dest) != want {
			t.Errorf("got %q, want %q", dest, want)
		}
	})
	t.Run("to bool errors", func(t *testing.T) {
		var dest bool
		if err := ScanDriverValue(&dest, src); err == nil {
			t.Error("expected error scanning time.Time into bool")
		}
	})
}

func TestScanDriverValue_ByteSliceToInterfaceClones(t *testing.T) {
	src := []byte("hello")
	var dest any
	if err := ScanDriverValue(&dest, src); err != nil {
		t.Fatal(err)
	}
	b, ok := dest.([]byte)
	if !ok {
		t.Fatalf("got %T, want []byte", dest)
	}
	if string(b) != "hello" {
		t.Errorf("got %q, want %q", b, "hello")
	}
	// Verify it's a clone by mutating the source.
	src[0] = 'X'
	if b[0] == 'X' {
		t.Error("dest shares memory with src, expected a clone")
	}
}

func TestScanDriverValue_NamedTypes(t *testing.T) {
	type myInt int64
	type myString string
	type myFloat float64

	t.Run("int64 to named int", func(t *testing.T) {
		var dest myInt
		if err := ScanDriverValue(&dest, int64(7)); err != nil {
			t.Fatal(err)
		}
		if dest != 7 {
			t.Errorf("got %d, want 7", dest)
		}
	})
	t.Run("string to named string", func(t *testing.T) {
		var dest myString
		if err := ScanDriverValue(&dest, "hi"); err != nil {
			t.Fatal(err)
		}
		if dest != "hi" {
			t.Errorf("got %q, want %q", dest, "hi")
		}
	})
	t.Run("string to named float", func(t *testing.T) {
		var dest myFloat
		if err := ScanDriverValue(&dest, "2.5"); err != nil {
			t.Fatal(err)
		}
		if dest != 2.5 {
			t.Errorf("got %g, want 2.5", dest)
		}
	})

	type myBytes []byte
	t.Run("[]byte to named bytes", func(t *testing.T) {
		var dest myBytes
		if err := ScanDriverValue(&dest, []byte("hi")); err != nil {
			t.Fatal(err)
		}
		if string(dest) != "hi" {
			t.Errorf("got %q, want %q", dest, "hi")
		}
	})
	t.Run("string to named bytes", func(t *testing.T) {
		var dest myBytes
		if err := ScanDriverValue(&dest, "hi"); err != nil {
			t.Fatal(err)
		}
		if string(dest) != "hi" {
			t.Errorf("got %q, want %q", dest, "hi")
		}
	})

	type myBool bool
	t.Run("bool to named bool", func(t *testing.T) {
		var dest myBool
		if err := ScanDriverValue(&dest, true); err != nil {
			t.Fatal(err)
		}
		if !dest {
			t.Error("got false, want true")
		}
	})
	t.Run("int64 1 to named bool", func(t *testing.T) {
		var dest myBool
		if err := ScanDriverValue(&dest, int64(1)); err != nil {
			t.Fatal(err)
		}
		if !dest {
			t.Error("got false, want true")
		}
	})

	type myTime time.Time
	t.Run("time to named struct", func(t *testing.T) {
		src := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		var dest myTime
		if err := ScanDriverValue(&dest, src); err != nil {
			t.Fatal(err)
		}
		if !time.Time(dest).Equal(src) {
			t.Errorf("got %v, want %v", time.Time(dest), src)
		}
	})
}

func TestScanDriverValue_FloatSpecialValues(t *testing.T) {
	t.Run("NaN to int errors", func(t *testing.T) {
		var dest int
		if err := ScanDriverValue(&dest, math.NaN()); err == nil {
			t.Error("expected error scanning NaN into int")
		}
	})
	t.Run("Inf to int errors", func(t *testing.T) {
		var dest int64
		if err := ScanDriverValue(&dest, math.Inf(1)); err == nil {
			t.Error("expected error scanning +Inf into int64")
		}
	})
	t.Run("NaN to bool errors", func(t *testing.T) {
		var dest bool
		if err := ScanDriverValue(&dest, math.NaN()); err == nil {
			t.Error("expected error scanning NaN into bool")
		}
	})
	t.Run("NaN to float64 stored", func(t *testing.T) {
		var dest float64
		if err := ScanDriverValue(&dest, math.NaN()); err != nil {
			t.Fatal(err)
		}
		if !math.IsNaN(dest) {
			t.Errorf("got %g, want NaN", dest)
		}
	})
	t.Run("NaN to string", func(t *testing.T) {
		var dest string
		if err := ScanDriverValue(&dest, math.NaN()); err != nil {
			t.Fatal(err)
		}
		if dest != "NaN" {
			t.Errorf("got %q, want %q", dest, "NaN")
		}
	})
}

func TestScanDriverValue_NilToScalarError(t *testing.T) {
	t.Run("nil into int errors", func(t *testing.T) {
		var dest int
		if err := ScanDriverValue(&dest, nil); err == nil {
			t.Error("expected error scanning nil into int")
		}
	})
	t.Run("nil into string errors", func(t *testing.T) {
		var dest string
		if err := ScanDriverValue(&dest, nil); err == nil {
			t.Error("expected error scanning nil into string")
		}
	})
}

// testDecimal implements the database/sql decimalDecompose and decimalCompose
// interfaces used by ScanDriverValue for direct decimal conversion.
type testDecimal struct {
	form byte
	neg  bool
	coef []byte
	exp  int32
}

func (d testDecimal) Decompose(buf []byte) (form byte, negative bool, coefficient []byte, exponent int32) {
	return d.form, d.neg, d.coef, d.exp
}

func (d *testDecimal) Compose(form byte, negative bool, coefficient []byte, exponent int32) error {
	d.form = form
	d.neg = negative
	d.coef = append([]byte(nil), coefficient...)
	d.exp = exponent
	return nil
}

func TestScanDriverValue_Decimal(t *testing.T) {
	src := testDecimal{coef: []byte{1, 2, 3}, exp: -2, neg: true}
	var dest testDecimal
	if err := ScanDriverValue(&dest, src); err != nil {
		t.Fatal(err)
	}
	if dest.exp != src.exp || dest.neg != src.neg || string(dest.coef) != string(src.coef) {
		t.Errorf("got %+v, want %+v", dest, src)
	}
}

func TestScanDriverValue_NilPointerDest(t *testing.T) {
	var dest *int
	if err := ScanDriverValue(dest, int64(1)); err == nil {
		t.Error("expected error scanning into a nil pointer")
	}
}

func TestScanDriverValue_DecimalNilPointer(t *testing.T) {
	var dest *testDecimal // nil pointer implementing decimalCompose
	src := testDecimal{coef: []byte{1, 2, 3}}
	if err := ScanDriverValue(dest, src); err == nil {
		t.Error("expected error scanning into nil *testDecimal, got nil")
	}
}

func TestBytesToStringScanConverter(t *testing.T) {
	c := BytesToStringScanConverter(`\x`)

	t.Run("valid utf8", func(t *testing.T) {
		val, ok := c.ConvertValue([]byte("hello"))
		if !ok {
			t.Fatal("expected ok=true for []byte input")
		}
		if val != "hello" {
			t.Errorf("got %q, want %q", val, "hello")
		}
	})

	t.Run("valid utf8 with multibyte runes", func(t *testing.T) {
		val, ok := c.ConvertValue([]byte("héllo 世界"))
		if !ok {
			t.Fatal("expected ok=true")
		}
		if val != "héllo 世界" {
			t.Errorf("got %q, want %q", val, "héllo 世界")
		}
	})

	t.Run("invalid utf8 hex encoded", func(t *testing.T) {
		val, ok := c.ConvertValue([]byte{0xde, 0xad, 0xbe, 0xef})
		if !ok {
			t.Fatal("expected ok=true")
		}
		if val != `\xDEADBEEF` {
			t.Errorf("got %q, want %q", val, `\xDEADBEEF`)
		}
	})

	t.Run("empty bytes", func(t *testing.T) {
		val, ok := c.ConvertValue([]byte{})
		if !ok {
			t.Fatal("expected ok=true")
		}
		if val != "" {
			t.Errorf("got %q, want empty string", val)
		}
	})

	t.Run("nil bytes", func(t *testing.T) {
		val, ok := c.ConvertValue([]byte(nil))
		if !ok {
			t.Fatal("expected ok=true")
		}
		if val != "" {
			t.Errorf("got %q, want empty string", val)
		}
	})

	t.Run("non-bytes value passes through", func(t *testing.T) {
		_, ok := c.ConvertValue(int64(42))
		if ok {
			t.Error("expected ok=false for non-[]byte input")
		}
	})

	t.Run("custom prefix", func(t *testing.T) {
		c := BytesToStringScanConverter("0x")
		val, ok := c.ConvertValue([]byte{0x01, 0xff})
		if !ok {
			t.Fatal("expected ok=true")
		}
		if val != "0x01FF" {
			t.Errorf("got %q, want %q", val, "0x01FF")
		}
	})

	t.Run("empty prefix", func(t *testing.T) {
		c := BytesToStringScanConverter("")
		val, ok := c.ConvertValue([]byte{0xff})
		if !ok {
			t.Fatal("expected ok=true")
		}
		if val != "FF" {
			t.Errorf("got %q, want %q", val, "FF")
		}
	})
}
