package sqldb

import "fmt"

type ParamPlaceholderFormatter interface {
	// ParamPlaceholder returns a parameter value placeholder
	// for the parameter with the passed zero based index
	// specific to the database type of the connection.
	ParamPlaceholder(index int) string
}

func NewParamPlaceholderFormatter(format string, indexOffset int) ParamPlaceholderFormatter {
	return &paramPlaceholderFormatter{format, indexOffset}
}

type paramPlaceholderFormatter struct {
	format      string
	indexOffset int
}

func (f *paramPlaceholderFormatter) ParamPlaceholder(index int) string {
	return fmt.Sprintf(f.format, index+f.indexOffset)
}
