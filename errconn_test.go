package sqldb

import (
	"errors"
	"fmt"
)

func ExampleErrConn_Config() {
	errConn := NewErrConn(errors.New("this is a test error"))
	fmt.Println(errConn.Config().String())
	fmt.Println(errConn.Config().URL().String())

	// Output:
	// ErrConn
	// ErrConn://localhost
}
