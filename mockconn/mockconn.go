package mockconn

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"maps"
	"regexp"
	"strings"
	"time"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/impl"
)

// Conn implements sqldb.Connection
var _ sqldb.Connection = new(Conn)

// NormalizeQueryFunc normalizes a query string before recording or lookup.
type NormalizeQueryFunc func(query string) (string, error)

// QueryRecordings records exec and query calls.
type QueryRecordings struct {
	Execs   []QueryData
	Queries []QueryData
}

// QueryData holds a recorded query and its arguments.
type QueryData struct {
	Query string
	Args  []any
}

// Conn is a mock implementation of sqldb.Connection for testing.
//
// Methods where the corresponding mock function is nil
// return sane defaults and no errors.
//
// All query/exec Connection methods delegate to MockExec and MockQuery.
type Conn struct {
	Ctx              context.Context
	StructFieldNamer sqldb.StructFieldMapper // DefaultStructFieldMapping if nil
	ArgFmt           string                  // e.g. "$%d"

	NormalizeQuery NormalizeQueryFunc
	QueryLog       io.Writer

	TxNo   uint64
	TxOpts *sql.TxOptions

	ListeningOn      map[string]struct{}
	Recordings       QueryRecordings
	MockQueryResults map[string]*MockRows

	// Base mock hooks — ALL query/exec methods delegate to these
	MockExec  func(query string, args ...any) error
	MockQuery func(query string, args ...any) *MockRows

	// Non-query mock hooks
	MockConfig               func() *sqldb.Config
	MockPing                 func(time.Duration) error
	MockStats                func() sql.DBStats
	MockBegin                func(opts *sql.TxOptions, no uint64) (sqldb.Connection, error)
	MockCommit               func() error
	MockRollback             func() error
	MockListenOnChannel      func(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) error
	MockUnlistenChannel      func(channel string) error
	MockIsListeningOnChannel func(channel string) bool
	MockClose                func() error
}

// New creates a new Conn with the given argument format, normalizer, and query log.
func New(argFmt string, normalizeQuery NormalizeQueryFunc, queryLog io.Writer) *Conn {
	return &Conn{
		Ctx:              context.Background(),
		StructFieldNamer: sqldb.DefaultStructFieldMapping,
		ArgFmt:           argFmt,
		NormalizeQuery:   normalizeQuery,
		QueryLog:         queryLog,
	}
}

// Clone returns a shallow copy of the Conn with cloned maps.
func (c *Conn) Clone() *Conn {
	cp := *c
	cp.ListeningOn = maps.Clone(c.ListeningOn)
	cp.MockQueryResults = maps.Clone(c.MockQueryResults)
	return &cp
}

// WithQueryResult returns a clone of the Conn with a pre-configured
// query result for the given query and args.
func (c *Conn) WithQueryResult(columns []string, rows [][]driver.Value, forQuery string, args ...any) *Conn {
	query := forQuery
	if c.NormalizeQuery != nil {
		var err error
		query, err = c.NormalizeQuery(query)
		if err != nil {
			panic(err)
		}
	}
	key := impl.FormatQuery(query, c.ArgFmt, args...)

	cc := c.Clone()
	if cc.MockQueryResults == nil {
		cc.MockQueryResults = make(map[string]*MockRows)
	}
	cc.MockQueryResults[key] = NewMockRows(columns...).WithRows(rows)
	return cc
}

func (c *Conn) ctx() context.Context {
	if c.Ctx != nil {
		return c.Ctx
	}
	return context.Background()
}

func (c *Conn) structFieldMapper() sqldb.StructFieldMapper {
	if c.StructFieldNamer != nil {
		return c.StructFieldNamer
	}
	return sqldb.DefaultStructFieldMapping
}

// Context implements sqldb.Connection.
func (c *Conn) Context() context.Context {
	return c.ctx()
}

// WithContext implements sqldb.Connection.
func (c *Conn) WithContext(ctx context.Context) sqldb.Connection {
	if ctx == c.Ctx {
		return c
	}
	cp := c.Clone()
	cp.Ctx = ctx
	return cp
}

// WithStructFieldMapper implements sqldb.Connection.
func (c *Conn) WithStructFieldMapper(namer sqldb.StructFieldMapper) sqldb.Connection {
	cp := c.Clone()
	cp.StructFieldNamer = namer
	return cp
}

// StructFieldMapper implements sqldb.Connection.
func (c *Conn) StructFieldMapper() sqldb.StructFieldMapper {
	return c.structFieldMapper()
}

// Placeholder implements sqldb.PlaceholderFormatter.
func (c *Conn) Placeholder(paramIndex int) string {
	return fmt.Sprintf(c.ArgFmt, paramIndex+1)
}

var columnNameRegex = regexp.MustCompile(`^[0-9a-zA-Z_]{1,64}$`)

// ValidateColumnName implements sqldb.Connection.
func (c *Conn) ValidateColumnName(name string) error {
	if !columnNameRegex.MatchString(name) {
		return fmt.Errorf("invalid column name: %q", name)
	}
	return nil
}

// Ping implements sqldb.Connection.
func (c *Conn) Ping(timeout time.Duration) error {
	if c.MockPing != nil {
		return c.MockPing(timeout)
	}
	return nil
}

// Config implements sqldb.Connection.
func (c *Conn) Config() *sqldb.Config {
	if c.MockConfig != nil {
		return c.MockConfig()
	}
	return &sqldb.Config{Driver: "mockconn", Host: "localhost", Database: "mock"}
}

// Stats implements sqldb.Connection.
func (c *Conn) Stats() sql.DBStats {
	if c.MockStats != nil {
		return c.MockStats()
	}
	return sql.DBStats{}
}

// Exec implements sqldb.Connection.
// This is the base exec method — all exec-related methods delegate through here.
func (c *Conn) Exec(query string, args ...any) error {
	c.Recordings.Execs = append(c.Recordings.Execs, QueryData{Query: query, Args: args})

	if c.QueryLog != nil {
		q := query
		if c.NormalizeQuery != nil {
			var err error
			q, err = c.NormalizeQuery(q)
			if err != nil {
				return err
			}
		}
		_, err := fmt.Fprint(c.QueryLog, impl.FormatQuery(q, c.ArgFmt, args...), ";\n")
		if err != nil {
			return err
		}
	}

	if c.MockExec != nil {
		return c.MockExec(query, args...)
	}
	return nil
}

// query is the internal base query method —
// all query-related methods delegate through here.
func (c *Conn) query(query string, args ...any) *MockRows {
	c.Recordings.Queries = append(c.Recordings.Queries, QueryData{Query: query, Args: args})

	if c.QueryLog != nil {
		q := query
		if c.NormalizeQuery != nil {
			q, _ = c.NormalizeQuery(q)
		}
		fmt.Fprint(c.QueryLog, impl.FormatQuery(q, c.ArgFmt, args...), ";\n")
	}

	if c.MockQuery != nil {
		return c.MockQuery(query, args...)
	}

	// Look up in MockQueryResults
	q := query
	if c.NormalizeQuery != nil {
		q, _ = c.NormalizeQuery(q)
	}
	key := impl.FormatQuery(q, c.ArgFmt, args...)
	return c.MockQueryResults[key]
}

// Update implements sqldb.Connection.
func (c *Conn) Update(table string, values sqldb.Values, where string, args ...any) error {
	return impl.Update(c, table, values, where, c.ArgFmt, args)
}

// UpdateStruct implements sqldb.Connection.
func (c *Conn) UpdateStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.UpdateStruct(c, table, rowStruct, c.structFieldMapper(), c.ArgFmt, ignoreColumns)
}

// UpsertStruct implements sqldb.Connection.
func (c *Conn) UpsertStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return impl.UpsertStruct(c, table, rowStruct, c.structFieldMapper(), c.ArgFmt, ignoreColumns)
}

// UpdateReturningRow implements sqldb.Connection.
func (c *Conn) UpdateReturningRow(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowScanner {
	return impl.UpdateReturningRow(c, table, values, returning, where, args...)
}

// UpdateReturningRows implements sqldb.Connection.
func (c *Conn) UpdateReturningRows(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowsScanner {
	return impl.UpdateReturningRows(c, table, values, returning, where, args...)
}

// QueryRow implements sqldb.Connection.
// If the query method returns nil (MockQuery returned nil
// or no matching entry in MockQueryResults), it returns a RowScanner
// with a joined error of sql.ErrNoRows and the context error (if any).
func (c *Conn) QueryRow(query string, args ...any) sqldb.RowScanner {
	mockRows := c.query(query, args...)
	if mockRows == nil {
		return sqldb.RowScannerWithError(errors.Join(fmt.Errorf("mock %w", sql.ErrNoRows), c.ctx().Err()))
	}
	return impl.NewRowScanner(mockRows, c.structFieldMapper(), query, c.ArgFmt, args)
}

// QueryRows implements sqldb.Connection.
// If the query method returns nil (MockQuery returned nil
// or no matching entry in MockQueryResults), it returns a RowsScanner
// with a joined error of sql.ErrNoRows and the context error (if any).
func (c *Conn) QueryRows(query string, args ...any) sqldb.RowsScanner {
	mockRows := c.query(query, args...)
	if mockRows == nil {
		return sqldb.RowsScannerWithError(errors.Join(fmt.Errorf("mock %w", sql.ErrNoRows), c.ctx().Err()))
	}
	return impl.NewRowsScanner(c.ctx(), mockRows, c.structFieldMapper(), query, c.ArgFmt, args)
}

// IsTransaction implements sqldb.Connection.
func (c *Conn) IsTransaction() bool {
	return c.TxNo != 0
}

// TransactionNo implements sqldb.Connection.
func (c *Conn) TransactionNo() uint64 {
	return c.TxNo
}

// TransactionOptions implements sqldb.Connection.
func (c *Conn) TransactionOptions() (*sql.TxOptions, bool) {
	return c.TxOpts, c.TxNo != 0
}

// Begin implements sqldb.Connection.
func (c *Conn) Begin(opts *sql.TxOptions, no uint64) (sqldb.Connection, error) {
	if no == 0 {
		return nil, errors.New("transaction number must not be zero")
	}

	if c.QueryLog != nil {
		query := "BEGIN"
		if opts != nil {
			if opts.Isolation != sql.LevelDefault {
				query += " ISOLATION LEVEL " + strings.ToUpper(opts.Isolation.String())
			}
			if opts.ReadOnly {
				query += " READ ONLY"
			}
		}
		_, err := fmt.Fprint(c.QueryLog, query+";\n")
		if err != nil {
			return nil, err
		}
	}

	if c.MockBegin != nil {
		return c.MockBegin(opts, no)
	}

	tx := c.Clone()
	tx.TxNo = no
	tx.TxOpts = opts
	return tx, nil
}

// Commit implements sqldb.Connection.
func (c *Conn) Commit() error {
	if c.QueryLog != nil {
		_, err := fmt.Fprint(c.QueryLog, "COMMIT;\n")
		if err != nil {
			return err
		}
	}

	if c.MockCommit != nil {
		return c.MockCommit()
	}
	return nil
}

// Rollback implements sqldb.Connection.
func (c *Conn) Rollback() error {
	if c.QueryLog != nil {
		_, err := fmt.Fprint(c.QueryLog, "ROLLBACK;\n")
		if err != nil {
			return err
		}
	}

	if c.MockRollback != nil {
		return c.MockRollback()
	}
	return nil
}

// ListenOnChannel implements sqldb.Connection.
func (c *Conn) ListenOnChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) error {
	if c.ListeningOn == nil {
		c.ListeningOn = make(map[string]struct{})
	}
	c.ListeningOn[channel] = struct{}{}

	if c.QueryLog != nil {
		_, err := fmt.Fprintf(c.QueryLog, "LISTEN %s;\n", channel)
		if err != nil {
			return err
		}
	}

	if c.MockListenOnChannel != nil {
		return c.MockListenOnChannel(channel, onNotify, onUnlisten)
	}
	return nil
}

// UnlistenChannel implements sqldb.Connection.
func (c *Conn) UnlistenChannel(channel string) error {
	delete(c.ListeningOn, channel)

	if c.QueryLog != nil {
		_, err := fmt.Fprintf(c.QueryLog, "UNLISTEN %s;\n", channel)
		if err != nil {
			return err
		}
	}

	if c.MockUnlistenChannel != nil {
		return c.MockUnlistenChannel(channel)
	}
	return nil
}

// IsListeningOnChannel implements sqldb.Connection.
func (c *Conn) IsListeningOnChannel(channel string) bool {
	if c.MockIsListeningOnChannel != nil {
		return c.MockIsListeningOnChannel(channel)
	}
	_, ok := c.ListeningOn[channel]
	return ok
}

// Close implements sqldb.Connection.
func (c *Conn) Close() error {
	if c.MockClose != nil {
		return c.MockClose()
	}
	return nil
}
