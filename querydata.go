package sqldb

import "github.com/DataDog/go-sqllexer"

type NormalizeQueryFunc func(query string) (string, error)

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
