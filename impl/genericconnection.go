package impl

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"time"

	"github.com/domonda/go-sqldb"
)

// NewGenericConnection returns a generic sqldb.Connection implementation
// for an existing sql.DB connection.
// argFmt is the format string for argument placeholders like "?" or "$%d"
// that will be replaced error messages to format a complete query.
func NewGenericConnection(ctx context.Context, db *sql.DB, config *sqldb.Config, listener sqldb.Listener, structFieldMapper sqldb.StructFieldMapper, validateColumnName func(string) error, converter driver.ValueConverter, argFmt string) sqldb.Connection {
	if listener == nil {
		listener = sqldb.UnsupportedListener()
	}
	return &genericConn{
		ctx:                ctx,
		db:                 db,
		config:             config,
		listener:           listener,
		structFieldMapper:  structFieldMapper,
		validateColumnName: validateColumnName,
		converter:          converter,
		argFmt:             argFmt,
	}
}

type genericConn struct {
	ctx                context.Context
	db                 *sql.DB
	config             *sqldb.Config
	listener           sqldb.Listener
	structFieldMapper  sqldb.StructFieldMapper
	validateColumnName func(string) error
	converter          driver.ValueConverter
	argFmt             string

	tx        *sql.Tx
	txOptions *sql.TxOptions
	txNo      uint64
}

func (conn *genericConn) clone() *genericConn {
	c := *conn
	return &c
}

func (conn *genericConn) Context() context.Context { return conn.ctx }

func (conn *genericConn) WithContext(ctx context.Context) sqldb.Connection {
	if ctx == conn.ctx {
		return conn
	}
	c := conn.clone()
	c.ctx = ctx
	return c
}

func (conn *genericConn) WithStructFieldMapper(mapper sqldb.StructFieldMapper) sqldb.Connection {
	c := conn.clone()
	c.structFieldMapper = mapper
	return c
}

func (conn *genericConn) StructFieldMapper() sqldb.StructFieldMapper {
	return conn.structFieldMapper
}

func (conn *genericConn) ValidateColumnName(name string) error {
	return conn.validateColumnName(name)
}

func (conn *genericConn) Ping(timeout time.Duration) error {
	ctx := conn.ctx
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return conn.db.PingContext(ctx)
}

func (conn *genericConn) Stats() sql.DBStats {
	return conn.db.Stats()
}

func (conn *genericConn) Config() *sqldb.Config {
	return conn.config
}

func (conn *genericConn) Now() (time.Time, error) {
	return QueryNow(conn)
}

func (conn *genericConn) execer() Execer {
	if conn.tx != nil {
		return conn.tx
	}
	return conn.db
}

func (conn *genericConn) queryer() Queryer {
	if conn.tx != nil {
		return conn.tx
	}
	return conn.db
}

func (conn *genericConn) Exec(query string, args ...any) error {
	return Exec(conn.ctx, conn.execer(), query, args, conn.converter, conn.argFmt)
}

func (conn *genericConn) QueryRow(query string, args ...any) sqldb.RowScanner {
	return QueryRow(conn.ctx, conn.queryer(), query, args, conn.converter, conn.argFmt, conn.structFieldMapper)
}

func (conn *genericConn) QueryRows(query string, args ...any) sqldb.RowsScanner {
	return QueryRows(conn.ctx, conn.queryer(), query, args, conn.converter, conn.argFmt, conn.structFieldMapper)
}

func (conn *genericConn) Insert(table string, columValues sqldb.Values) error {
	return Insert(conn.ctx, conn.execer(), table, columValues, conn.converter, conn.argFmt)
}

func (conn *genericConn) InsertUnique(table string, values sqldb.Values, onConflict string) (inserted bool, err error) {
	return InsertUnique(conn.ctx, conn.queryer(), table, values, onConflict, conn.converter, conn.argFmt, conn.structFieldMapper)
}

func (conn *genericConn) InsertReturning(table string, values sqldb.Values, returning string) sqldb.RowScanner {
	return InsertReturning(conn.ctx, conn.queryer(), table, values, returning, conn.converter, conn.argFmt, conn.structFieldMapper)
}

func (conn *genericConn) InsertStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return InsertStruct(conn.ctx, conn.execer(), table, rowStruct, conn.structFieldMapper, ignoreColumns, conn.converter, conn.argFmt)
}

func (conn *genericConn) InsertStructs(table string, rowStructs any, ignoreColumns ...sqldb.ColumnFilter) error {
	// TODO optimized version with single query if possible, split into multiple queries depending or maxArgs for query
	return InsertStructs(conn, table, rowStructs, ignoreColumns...)
}

func (conn *genericConn) InsertUniqueStruct(table string, rowStruct any, onConflict string, ignoreColumns ...sqldb.ColumnFilter) (inserted bool, err error) {
	return InsertUniqueStruct(conn.ctx, conn.queryer(), conn.structFieldMapper, table, rowStruct, onConflict, ignoreColumns, conn.converter, conn.argFmt)
}

func (conn *genericConn) Update(table string, values sqldb.Values, where string, args ...any) error {
	return Update(conn.ctx, conn.execer(), table, values, where, args, conn.converter, conn.argFmt)
}

func (conn *genericConn) UpdateReturningRow(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowScanner {
	return UpdateReturningRow(conn.ctx, conn.queryer(), table, values, returning, where, args, conn.converter, conn.argFmt, conn.structFieldMapper)
}

func (conn *genericConn) UpdateReturningRows(table string, values sqldb.Values, returning, where string, args ...any) sqldb.RowsScanner {
	return UpdateReturningRows(conn.ctx, conn.queryer(), table, values, returning, where, args, conn.converter, conn.argFmt, conn.structFieldMapper)
}

func (conn *genericConn) UpdateStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return UpdateStruct(conn.ctx, conn.execer(), table, rowStruct, conn.structFieldMapper, ignoreColumns, conn.converter, conn.argFmt)
}

func (conn *genericConn) UpsertStruct(table string, rowStruct any, ignoreColumns ...sqldb.ColumnFilter) error {
	return UpsertStruct(conn.ctx, conn.execer(), table, rowStruct, conn.structFieldMapper, conn.argFmt, ignoreColumns)
}

func (conn *genericConn) IsTransaction() bool {
	return conn.tx != nil
}

func (conn *genericConn) TransactionNo() uint64 {
	return conn.txNo
}

func (conn *genericConn) TransactionOptions() (*sql.TxOptions, bool) {
	return conn.txOptions, conn.tx != nil
}

func (conn *genericConn) Begin(opts *sql.TxOptions, no uint64) (sqldb.Connection, error) {
	tx, err := conn.db.BeginTx(conn.ctx, opts)
	if err != nil {
		return nil, err
	}
	txConn := conn.clone()
	txConn.tx = tx
	txConn.txOptions = opts
	txConn.txNo = no
	return txConn, nil
}

func (conn *genericConn) Commit() error {
	if conn.tx == nil {
		return sqldb.ErrNotWithinTransaction
	}
	return conn.tx.Commit()
}

func (conn *genericConn) Rollback() error {
	if conn.tx == nil {
		return sqldb.ErrNotWithinTransaction
	}
	return conn.tx.Rollback()
}

func (conn *genericConn) ListenChannel(channel string, onNotify sqldb.OnNotifyFunc, onUnlisten sqldb.OnUnlistenFunc) (err error) {
	return conn.listener.ListenChannel(conn, channel, onNotify, onUnlisten)
}

func (conn *genericConn) UnlistenChannel(channel string) (err error) {
	return conn.listener.UnlistenChannel(conn, channel)
}

func (conn *genericConn) IsListeningOnChannel(channel string) bool {
	return conn.listener.IsListeningOnChannel(conn, channel)
}

func (conn *genericConn) Close() error {
	err := conn.listener.Close(conn)
	if conn.tx != nil {
		return errors.Join(err, conn.tx.Rollback())
	}
	return errors.Join(err, conn.db.Close())
}
