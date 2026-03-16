package sqldb

import "github.com/DataDog/go-sqllexer"

// NormalizeQueryFunc is a function type that normalizes an SQL query string.
type NormalizeQueryFunc func(query string) (string, error)

// NoChangeNormalizeQuery is a NormalizeQueryFunc that returns the query unchanged.
func NoChangeNormalizeQuery(query string) (string, error) {
	return query, nil
}

// NewQueryNormalizer returns a NormalizeQueryFunc that normalizes SQL queries
// using the sqllexer package with sensible defaults.
func NewQueryNormalizer() NormalizeQueryFunc {
	normalizer := sqllexer.NewNormalizer(
		sqllexer.WithCollectCommands(true),
		sqllexer.WithCollectTables(true),
		sqllexer.WithKeepSQLAlias(true),
		sqllexer.WithRemoveSpaceBetweenParentheses(true),
		sqllexer.WithKeepIdentifierQuotation(true),
	)
	return func(query string) (string, error) {
		query, _, err := normalizer.Normalize(query)
		return query, err
	}
}

// QueryData holds an SQL query with its arguments
// (query parameters).
type QueryData struct {
	// The SQL query string
	Query string

	// The arguments for the query
	Args []any
}

// NewQueryData returns a QueryData with the query optionally normalized by the given function.
func NewQueryData(query string, args []any, normalize NormalizeQueryFunc) (QueryData, error) {
	var err error
	if normalize != nil {
		query, err = normalize(query)
	}
	return QueryData{
		Query: query,
		Args:  args,
	}, err
}

// Format returns the query with its arguments substituted using the given formatter.
func (q *QueryData) Format(formatter QueryFormatter) string {
	return FormatQuery(formatter, q.Query, q.Args...)
}
