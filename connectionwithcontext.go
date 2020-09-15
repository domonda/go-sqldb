package sqldb

import (
	"context"
)

// ConnectionWithContext wraps the passed Connection so that all
// non Context methods call their Context counterparts with the
// ctx passed to this function.
// If conn was alread wrapped with ctx, then it's returned unchanged.
func ConnectionWithContext(conn Connection, ctx context.Context) Connection {
	if c, ok := conn.(connWithCtx); ok && c.ctx == ctx {
		return c
	}
	return connWithCtx{conn, ctx}
}

type connWithCtx struct {
	Connection

	ctx context.Context
}

func (c connWithCtx) Exec(query string, args ...interface{}) error {
	return c.ExecContext(c.ctx, query, args...)
}

func (c connWithCtx) Insert(table string, values Values) error {
	return c.InsertContext(c.ctx, table, values)
}

func (c connWithCtx) InsertUnique(table string, values Values, onConflict string) (inserted bool, err error) {
	return c.InsertUniqueContext(c.ctx, table, values, onConflict)
}

func (c connWithCtx) InsertReturning(table string, values Values, returning string) RowScanner {
	return c.InsertReturningContext(c.ctx, table, values, returning)
}

func (c connWithCtx) InsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return c.InsertStructContext(c.ctx, table, rowStruct, restrictToColumns...)
}

func (c connWithCtx) InsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return c.InsertStructIgnoreColumnsContext(c.ctx, table, rowStruct, ignoreColumns...)
}

func (c connWithCtx) InsertUniqueStruct(table string, rowStruct interface{}, onConflict string, restrictToColumns ...string) (inserted bool, err error) {
	return c.InsertUniqueStructContext(c.ctx, table, rowStruct, onConflict, restrictToColumns...)
}

func (c connWithCtx) InsertUniqueStructIgnoreColumns(table string, rowStruct interface{}, onConflict string, ignoreColumns ...string) (inserted bool, err error) {
	return c.InsertUniqueStructIgnoreColumnsContext(c.ctx, table, rowStruct, onConflict, ignoreColumns...)
}

func (c connWithCtx) Update(table string, values Values, where string, args ...interface{}) error {
	return c.UpdateContext(c.ctx, table, values, where, args...)
}

func (c connWithCtx) UpdateReturningRow(table string, values Values, returning, where string, args ...interface{}) RowScanner {
	return c.UpdateReturningRowContext(c.ctx, table, values, returning, where, args...)
}

func (c connWithCtx) UpdateReturningRows(table string, values Values, returning, where string, args ...interface{}) RowsScanner {
	return c.UpdateReturningRowsContext(c.ctx, table, values, returning, where, args...)
}

func (c connWithCtx) UpdateStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return c.UpdateStructContext(c.ctx, table, rowStruct, restrictToColumns...)
}

func (c connWithCtx) UpdateStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return c.UpdateStructIgnoreColumnsContext(c.ctx, table, rowStruct, ignoreColumns...)
}

func (c connWithCtx) UpsertStruct(table string, rowStruct interface{}, restrictToColumns ...string) error {
	return c.UpsertStructContext(c.ctx, table, rowStruct, restrictToColumns...)
}

func (c connWithCtx) UpsertStructIgnoreColumns(table string, rowStruct interface{}, ignoreColumns ...string) error {
	return c.UpsertStructIgnoreColumnsContext(c.ctx, table, rowStruct, ignoreColumns...)
}

func (c connWithCtx) QueryRow(query string, args ...interface{}) RowScanner {
	return c.QueryRowContext(c.ctx, query, args...)
}

func (c connWithCtx) QueryRows(query string, args ...interface{}) RowsScanner {
	return c.QueryRowsContext(c.ctx, query, args...)
}
