package db

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestInsertRowStruct(t *testing.T) {
	type Struct1 struct {
		sqldb.TableName `db:"my_table"`
		ID              int    `db:"id"`
		Name            string `db:"name"`
	}

	tests := []struct {
		name      string
		rowStruct sqldb.StructWithTableName
		options   []sqldb.QueryOption
		config    *sqldb.ConnExt
		want      sqldb.QueryRecordings
		wantErr   bool
	}{
		{
			name: "simple",
			rowStruct: &Struct1{
				ID:   1,
				Name: "test",
			},
			config: sqldb.NewConnExt(
				sqldb.NewMockConn("$", nil, os.Stdout),
				sqldb.NewTaggedStructReflector(),
				sqldb.NewQueryFormatter("$"),
				sqldb.StdQueryBuilder{},
			),
			want: sqldb.QueryRecordings{
				Execs: []sqldb.QueryData{
					{Query: "INSERT INTO my_table(id,name) VALUES($1,$2)", Args: []any{1, "test"}},
				},
			},
		},
		// Error cases
		{
			name: "TableName without name tag",
			rowStruct: struct {
				sqldb.TableName
				ID   int    `db:"id"`
				Name string `db:"name"`
			}{},
			config: sqldb.NewConnExt(
				sqldb.NewMockConn("$", nil, os.Stdout),
				sqldb.NewTaggedStructReflector(),
				sqldb.NewQueryFormatter("$"),
				sqldb.StdQueryBuilder{},
			),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ContextWithConn(t.Context(), tt.config)
			err := InsertRowStruct(ctx, tt.rowStruct, tt.options...)
			if tt.wantErr {
				require.Error(t, err, "error from InsertStruct")
				return
			}
			require.NoError(t, err, "error from InsertStruct")
			require.Equal(t, tt.want, tt.config.Connection.(*sqldb.MockConn).Recordings, "MockConn.Recordings")
		})
	}
}

func TestInsertRowStruct_CacheWithoutOptions(t *testing.T) {
	// Use a unique struct type so the global cache doesn't
	// interfere with other tests or get influenced by them.
	type CacheTestStruct struct {
		sqldb.TableName `db:"cache_table"`
		ID              int    `db:"id"`
		Name            string `db:"name"`
		Extra           string `db:"extra"`
	}

	mock := sqldb.NewMockConn("$", nil, nil)
	config := sqldb.NewConnExt(
		mock,
		sqldb.NewTaggedStructReflector(),
		sqldb.NewQueryFormatter("$"),
		sqldb.StdQueryBuilder{},
	)
	ctx := ContextWithConn(t.Context(), config)

	// First call populates the cache
	err := InsertRowStruct(ctx, &CacheTestStruct{ID: 1, Name: "first", Extra: "a"})
	require.NoError(t, err)

	// Second call should use the cached query with same columns
	err = InsertRowStruct(ctx, &CacheTestStruct{ID: 2, Name: "second", Extra: "b"})
	require.NoError(t, err)

	require.Len(t, mock.Recordings.Execs, 2)
	// Both calls must produce the same query
	require.Equal(t, mock.Recordings.Execs[0].Query, mock.Recordings.Execs[1].Query)
	// But different args
	require.Equal(t, []any{1, "first", "a"}, mock.Recordings.Execs[0].Args)
	require.Equal(t, []any{2, "second", "b"}, mock.Recordings.Execs[1].Args)
}

func TestInsertRowStruct_CacheBypassedWithOptions(t *testing.T) {
	// Use a unique struct type so the global cache doesn't
	// interfere with other tests or get influenced by them.
	type OptionsCacheTestStruct struct {
		sqldb.TableName `db:"options_cache_table"`
		ID              int    `db:"id"`
		Name            string `db:"name"`
		Extra           string `db:"extra"`
	}

	mock := sqldb.NewMockConn("$", nil, nil)
	config := sqldb.NewConnExt(
		mock,
		sqldb.NewTaggedStructReflector(),
		sqldb.NewQueryFormatter("$"),
		sqldb.StdQueryBuilder{},
	)
	ctx := ContextWithConn(t.Context(), config)

	// First call without options — all columns
	err := InsertRowStruct(ctx, &OptionsCacheTestStruct{ID: 1, Name: "first", Extra: "a"})
	require.NoError(t, err)

	// Second call with IgnoreColumns("extra") — should produce a different query
	err = InsertRowStruct(ctx, &OptionsCacheTestStruct{ID: 2, Name: "second"}, sqldb.IgnoreColumns("extra"))
	require.NoError(t, err)

	require.Len(t, mock.Recordings.Execs, 2)
	// First query has all 3 columns
	require.Equal(t,
		"INSERT INTO options_cache_table(id,name,extra) VALUES($1,$2,$3)",
		mock.Recordings.Execs[0].Query,
	)
	// Second query has only 2 columns because "extra" was ignored
	require.Equal(t,
		"INSERT INTO options_cache_table(id,name) VALUES($1,$2)",
		mock.Recordings.Execs[1].Query,
	)
}

func TestInsertRowStruct_CacheNotPollutedByOptions(t *testing.T) {
	// Use a unique struct type so the global cache doesn't
	// interfere with other tests or get influenced by them.
	type PollutionTestStruct struct {
		sqldb.TableName `db:"pollution_table"`
		ID              int    `db:"id"`
		Name            string `db:"name"`
		Extra           string `db:"extra"`
	}

	mock := sqldb.NewMockConn("$", nil, nil)
	config := sqldb.NewConnExt(
		mock,
		sqldb.NewTaggedStructReflector(),
		sqldb.NewQueryFormatter("$"),
		sqldb.StdQueryBuilder{},
	)
	ctx := ContextWithConn(t.Context(), config)

	// First call WITH options — should NOT populate the cache
	err := InsertRowStruct(ctx, &PollutionTestStruct{ID: 1, Name: "first"}, sqldb.IgnoreColumns("extra"))
	require.NoError(t, err)

	// Second call WITHOUT options — should generate query with all columns,
	// not reuse the filtered query from the first call
	err = InsertRowStruct(ctx, &PollutionTestStruct{ID: 2, Name: "second", Extra: "b"})
	require.NoError(t, err)

	require.Len(t, mock.Recordings.Execs, 2)
	// First query has 2 columns (extra was ignored)
	require.Equal(t,
		"INSERT INTO pollution_table(id,name) VALUES($1,$2)",
		mock.Recordings.Execs[0].Query,
	)
	// Second query has all 3 columns — cache was not polluted
	require.Equal(t,
		"INSERT INTO pollution_table(id,name,extra) VALUES($1,$2,$3)",
		mock.Recordings.Execs[1].Query,
	)
}

func TestInsert(t *testing.T) {
	timestamp := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		table   string
		values  sqldb.Values
		config  *sqldb.ConnExt
		want    sqldb.QueryRecordings
		wantErr bool
	}{
		{
			name:  "basic",
			table: "public.my_table",
			values: sqldb.Values{
				"id":         1,
				"name":       "Test",
				"created_at": timestamp,
				"updated_at": sql.NullTime{},
			},
			config: sqldb.NewConnExt(
				sqldb.NewMockConn("$", nil, os.Stdout),
				sqldb.NewTaggedStructReflector(),
				sqldb.NewQueryFormatter("$"),
				sqldb.StdQueryBuilder{},
			),
			want: sqldb.QueryRecordings{
				Execs: []sqldb.QueryData{
					{
						Query: `INSERT INTO public.my_table(created_at,id,name,updated_at) VALUES($1,$2,$3,$4)`,
						Args:  []any{timestamp, 1, "Test", sql.NullTime{}},
					},
				},
			},
		},

		// Error cases
		{
			name:   "no values",
			table:  "public.my_table",
			values: sqldb.Values{},
			config: sqldb.NewConnExt(
				sqldb.NewMockConn("$", nil, os.Stdout),
				sqldb.NewTaggedStructReflector(),
				sqldb.NewQueryFormatter("$"),
				sqldb.StdQueryBuilder{},
			),

			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ContextWithConn(t.Context(), tt.config)
			err := Insert(ctx, tt.table, tt.values)
			if tt.wantErr {
				require.Error(t, err, "error from Insert")
				return
			}
			require.NoError(t, err, "error from Insert")
			require.Equal(t, tt.want, tt.config.Connection.(*sqldb.MockConn).Recordings, "MockConn.Recordings")
		})
	}
}
