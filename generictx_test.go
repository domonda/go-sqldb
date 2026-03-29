package sqldb

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGenericTx(t *testing.T, opts *sql.TxOptions, id uint64) *genericTx {
	t.Helper()
	parent := &genericConn{
		QueryFormatter:        StdQueryFormatter{PlaceholderPosPrefix: "$"},
		db:                    &sql.DB{},
		config:                &ConnConfig{Driver: "postgres", Host: "localhost", Database: "testdb"},
		defaultIsolationLevel: sql.LevelReadCommitted,
	}
	return &genericTx{
		parent: parent,
		tx:     nil,
		opts:   opts,
		id:     id,
	}
}

func TestGenericTx_Config(t *testing.T) {
	// given
	tx := newTestGenericTx(t, nil, 1)

	// when
	cfg := tx.Config()

	// then
	require.NotNil(t, cfg)
	assert.Equal(t, "postgres", cfg.Driver)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, "testdb", cfg.Database)
}

func TestGenericTx_Stats(t *testing.T) {
	// given
	tx := newTestGenericTx(t, nil, 1)

	// when
	stats := tx.Stats()

	// then - stats from an unconnected sql.DB should be zero-valued
	assert.Equal(t, sql.DBStats{}, stats)
}

func TestGenericTx_DefaultIsolationLevel(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		level sql.IsolationLevel
	}{
		{name: "ReadCommitted", level: sql.LevelReadCommitted},
		{name: "Serializable", level: sql.LevelSerializable},
		{name: "Default", level: sql.LevelDefault},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			parent := &genericConn{
				QueryFormatter:        StdQueryFormatter{},
				db:                    &sql.DB{},
				config:                &ConnConfig{Driver: "postgres"},
				defaultIsolationLevel: scenario.level,
			}
			tx := &genericTx{parent: parent, id: 1}

			// when
			got := tx.DefaultIsolationLevel()

			// then
			assert.Equal(t, scenario.level, got)
		})
	}
}

func TestGenericTx_Transaction(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		id         uint64
		opts       *sql.TxOptions
		wantActive bool
	}{
		{
			name:       "active transaction with opts",
			id:         42,
			opts:       &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: true},
			wantActive: true,
		},
		{
			name:       "active transaction without opts",
			id:         1,
			opts:       nil,
			wantActive: true,
		},
		{
			name:       "zero id is inactive",
			id:         0,
			opts:       nil,
			wantActive: false,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			tx := newTestGenericTx(t, scenario.opts, scenario.id)

			// when
			state := tx.Transaction()

			// then
			assert.Equal(t, scenario.id, state.ID)
			assert.Equal(t, scenario.opts, state.Opts)
			assert.Equal(t, scenario.wantActive, state.Active())
		})
	}
}

func TestGenericTx_FormatTableName(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "simple table", input: "users", want: "users"},
		{name: "schema qualified", input: "public.users", want: "public.users"},
		{name: "invalid name", input: "invalid-name!", wantErr: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			tx := newTestGenericTx(t, nil, 1)

			// when
			got, err := tx.FormatTableName(scenario.input)

			// then
			if scenario.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, scenario.want, got)
			}
		})
	}
}

func TestGenericTx_FormatColumnName(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "simple column", input: "id", want: "id"},
		{name: "underscore column", input: "created_at", want: "created_at"},
		{name: "invalid name", input: "has space", wantErr: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			tx := newTestGenericTx(t, nil, 1)

			// when
			got, err := tx.FormatColumnName(scenario.input)

			// then
			if scenario.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, scenario.want, got)
			}
		})
	}
}

func TestGenericTx_FormatPlaceholder(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		prefix     string
		paramIndex int
		want       string
	}{
		{name: "postgres style index 0", prefix: "$", paramIndex: 0, want: "$1"},
		{name: "postgres style index 3", prefix: "$", paramIndex: 3, want: "$4"},
		{name: "question mark style", prefix: "", paramIndex: 0, want: "?"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			parent := &genericConn{
				QueryFormatter: StdQueryFormatter{PlaceholderPosPrefix: scenario.prefix},
				db:             &sql.DB{},
				config:         &ConnConfig{Driver: "test"},
			}
			tx := &genericTx{parent: parent, id: 1}

			// when
			got := tx.FormatPlaceholder(scenario.paramIndex)

			// then
			assert.Equal(t, scenario.want, got)
		})
	}
}

func TestGenericTx_FormatStringLiteral(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple string", input: "hello", want: "'hello'"},
		{name: "string with quote", input: "it's", want: "'it''s'"},
		{name: "empty string", input: "", want: "''"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// given
			tx := newTestGenericTx(t, nil, 1)

			// when
			got := tx.FormatStringLiteral(scenario.input)

			// then
			assert.Equal(t, scenario.want, got)
		})
	}
}

func TestGenericTx_MaxArgs(t *testing.T) {
	// given
	tx := newTestGenericTx(t, nil, 1)

	// when
	got := tx.MaxArgs()

	// then - StdQueryFormatter returns 65535
	assert.Equal(t, 65535, got)
}

func TestGenericTxWithQueryBuilder_InsertUnique_NotSupported(t *testing.T) {
	// given - StdQueryBuilder does not implement UpsertQueryBuilder
	tx := newTestGenericTx(t, nil, 1)
	conn := &genericTxWithQueryBuilder{
		genericTx:    tx,
		QueryBuilder: StdQueryBuilder{},
	}
	columns := []ColumnInfo{{Name: "id", PrimaryKey: true}}

	// when
	_, err := conn.InsertUnique(tx.parent, "test_table", columns, "ON CONFLICT DO NOTHING")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not implement UpsertQueryBuilder")
}

func TestGenericTxWithQueryBuilder_Upsert_NotSupported(t *testing.T) {
	// given - StdQueryBuilder does not implement UpsertQueryBuilder
	tx := newTestGenericTx(t, nil, 1)
	conn := &genericTxWithQueryBuilder{
		genericTx:    tx,
		QueryBuilder: StdQueryBuilder{},
	}
	columns := []ColumnInfo{{Name: "id", PrimaryKey: true}}

	// when
	_, err := conn.Upsert(tx.parent, "test_table", columns)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not implement UpsertQueryBuilder")
}

func TestGenericTxWithQueryBuilder_InsertReturning_NotSupported(t *testing.T) {
	// given - StdQueryBuilder does not implement ReturningQueryBuilder
	tx := newTestGenericTx(t, nil, 1)
	conn := &genericTxWithQueryBuilder{
		genericTx:    tx,
		QueryBuilder: StdQueryBuilder{},
	}
	columns := []ColumnInfo{{Name: "id", PrimaryKey: true}}

	// when
	_, err := conn.InsertReturning(tx.parent, "test_table", columns, "*")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not implement ReturningQueryBuilder")
}

func TestGenericTxWithQueryBuilder_UpdateReturning_NotSupported(t *testing.T) {
	// given - StdQueryBuilder does not implement ReturningQueryBuilder
	tx := newTestGenericTx(t, nil, 1)
	conn := &genericTxWithQueryBuilder{
		genericTx:    tx,
		QueryBuilder: StdQueryBuilder{},
	}

	// when
	_, _, err := conn.UpdateReturning(tx.parent, "test_table", nil, "*", "id=$1", []any{1})

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not implement ReturningQueryBuilder")
}

func TestGenericTxWithQueryBuilder_DelegatesConfig(t *testing.T) {
	// given
	tx := newTestGenericTx(t, nil, 1)
	conn := &genericTxWithQueryBuilder{
		genericTx:    tx,
		QueryBuilder: StdQueryBuilder{},
	}

	// when
	cfg := conn.Config()

	// then
	require.NotNil(t, cfg)
	assert.Equal(t, "postgres", cfg.Driver)
}

func TestGenericTxWithQueryBuilder_DelegatesTransaction(t *testing.T) {
	// given
	opts := &sql.TxOptions{Isolation: sql.LevelSerializable}
	tx := newTestGenericTx(t, opts, 99)
	conn := &genericTxWithQueryBuilder{
		genericTx:    tx,
		QueryBuilder: StdQueryBuilder{},
	}

	// when
	state := conn.Transaction()

	// then
	assert.True(t, state.Active())
	assert.Equal(t, uint64(99), state.ID)
	assert.Equal(t, opts, state.Opts)
}
