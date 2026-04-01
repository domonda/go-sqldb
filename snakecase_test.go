package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToSnakeCase(t *testing.T) {
	for _, scenario := range []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "underscore", input: "_", expected: "_"},
		{name: "space", input: " ", expected: "_"},
		{name: "two spaces", input: "  ", expected: "__"},
		{name: "tab x newline", input: "\tX\n", expected: "_x_"},
		{name: "already snake case", input: "already_snake_case", expected: "already_snake_case"},
		{name: "already snake case surrounded", input: "_already_snake_case_", expected: "_already_snake_case_"},
		{name: "HelloWorld", input: "HelloWorld", expected: "hello_world"},
		{name: "Hello World", input: "Hello World", expected: "hello_world"},
		{name: "Hello-World", input: "Hello-World", expected: "hello_world"},
		{name: "symbols around", input: "*Hello+World*", expected: "_hello_world_"},
		{name: "Hello.World", input: "Hello.World", expected: "hello_world"},
		{name: "Hello/World", input: "Hello/World", expected: "hello_world"},
		{name: "parentheses and bang", input: "(Hello World!)", expected: "_hello_world__"},
		{name: "consecutive uppercase", input: "DocumentID", expected: "document_id"},
		{name: "all uppercase prefix", input: "HTMLHandler", expected: "htmlhandler"},
		{name: "non-ASCII lower", input: "Straßenadresse", expected: "straßenadresse"},
		{name: "non-ASCII with upper", input: "もしもしWorld", expected: "もしもし_world"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// when
			actual := ToSnakeCase(scenario.input)

			// then
			assert.Equal(t, scenario.expected, actual)
		})
	}
}
