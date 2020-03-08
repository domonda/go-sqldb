package sqldb

import (
	"testing"
	"time"
)

func TestStringScannable_Scan(t *testing.T) {
	testTime := time.Now()

	tests := []struct {
		name     string
		expected StringScannable
		src      interface{}
		wantErr  bool
	}{
		{name: "int64", expected: "-66", src: int64(-66)},
		{name: "float64", expected: "-66.6", src: float64(-66.6)},
		{name: "bool", expected: "true", src: true},
		{name: "[]byte", expected: "Hello World!", src: []byte("Hello World!")},
		{name: "string", expected: "Hello World!", src: "Hello World!"},
		{name: "time.Time", expected: StringScannable(testTime.String()), src: testTime},
		{name: "nil", expected: "", src: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s StringScannable
			if err := s.Scan(tt.src); (err != nil) != tt.wantErr {
				t.Errorf("StringScannable.Scan() error = %v, wantErr %v", err, tt.wantErr)
			}
			if s != tt.expected {
				t.Errorf("StringScannable.Scan() expected = %v, got %v", tt.expected, s)
			}
		})
	}
}

//    int64
