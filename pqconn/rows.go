package pqconn

import "database/sql"

// rows wraps *sql.Rows so that Scan automatically
// wraps slice/array destination pointers with pq.Array(),
// symmetric to wrapArrayArgs on the input side.
type rows struct {
	*sql.Rows
}

func (r rows) Scan(dest ...any) error {
	wrapArrayScanDest(dest)
	return r.Rows.Scan(dest...)
}
