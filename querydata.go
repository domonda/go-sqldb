package sqldb

import "github.com/DataDog/go-sqllexer"

// QueryData holds an SQL query with its arguments
// (query parameters).
type QueryData struct {
	// The SQL query string
	Query string

	// The arguments for the query
	Args []any
}

var queryNormalizer = sqllexer.NewNormalizer(
	sqllexer.WithCollectCommands(true),
	sqllexer.WithCollectTables(true),
	sqllexer.WithUppercaseKeywords(true),
	sqllexer.WithKeepSQLAlias(true),
	sqllexer.WithKeepIdentifierQuotation(true),
)

func UnchangedQueryData(query string, args []any) (QueryData, error) {
	_, _, err := queryNormalizer.Normalize(query)
	if err != nil {
		return QueryData{}, err
	}
	return QueryData{
		Query: query,
		Args:  args,
	}, nil
}

func NormalizedQueryData(query string, args []any) (QueryData, error) {
	query, _, err := queryNormalizer.Normalize(query)
	if err != nil {
		return QueryData{}, err
	}
	return QueryData{
		Query: query,
		Args:  args,
	}, nil
}
