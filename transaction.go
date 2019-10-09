package sqldb

import (
	"database/sql"
	"errors"
	"fmt"
)

// Transaction executes txFunc within a transaction that is passed in as tx Connection.
// The transaction will be rolled back if txFunc returns an error or panics.
func Transaction(conn Connection, txFunc func(tx Connection) error) (err error) {
	tx, e := conn.Begin()
	if e != nil {
		return e
	}

	defer func() {
		if r := recover(); r != nil {

			e := tx.Rollback()
			if e != nil {
				if errors.Is(e, sql.ErrTxDone) {
					// No hard error, should we still debug log it?
				} else {
					// Double error situation, log e so it doesn't get lost
					ErrLogger.Printf("error %s from transaction rollback after panic: %+v", e, r)
				}
			}
			panic(r) // re-throw panic after Rollback

		} else if err != nil {

			e := tx.Rollback()
			if e != nil {
				if errors.Is(e, sql.ErrTxDone) {
					// No hard error, should we still debug log it?
				} else {
					// Double error situation, wrap err with e so it doesn't get lost
					err = fmt.Errorf("error %s from transaction rollback after error: %w", e, err)
				}
			}

		} else {

			e := tx.Commit()
			if e != nil {
				// TODO clarify: Commit should never get called twice
				// so sql.ErrTxDone should not get ignored.
				// Save for error cases above?
				// Why do we have those sql.ErrTxDone cases?

				// if errors.Is(e, sql.ErrTxDone) {
				// 	// No hard error, should we still debug log it?
				// } else {
				// 	// Commit failed, set error as function return value
				// 	err = e
				// }

				// Set Commit error as function return value
				err = e
			}

		}
	}()

	return txFunc(tx)
}
