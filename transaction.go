package sqldb

import (
	"database/sql"

	"github.com/domonda/errors"
	"github.com/domonda/zerolog/log"
)

func Transaction(conn Connection, txFunc func(tx Connection) error) (err error) {
	tx, e := conn.Begin()
	if e != nil {
		return e
	}

	defer func() {
		if r := recover(); r != nil {

			e := tx.Rollback()
			if e != nil {
				if errors.Cause(e) == sql.ErrTxDone {
					// No hard error, should we still debug log it?
				} else {
					// Double error situation, log e so it doesn't get lost
					log.Error().Err(e).Msgf("error from transaction rollback after panic: %+v", r)
				}
			}
			panic(r) // re-throw panic after Rollback

		} else if err != nil {

			e := tx.Rollback()
			if e != nil {
				if errors.Cause(e) == sql.ErrTxDone {
					// No hard error, should we still debug log it?
				} else {
					// Double error situation, log e so it doesn't get lost
					log.Error().Err(e).Msgf("error from transaction rollback after error: %+v", err)
				}
			}

		} else {

			e := tx.Commit()
			if e != nil {
				// TODO clarify: Commit should never get called twice
				// so sql.ErrTxDone should not get ignored.
				// Save for error cases above?
				// Why do we have those sql.ErrTxDone cases?

				// if errors.Cause(e) == sql.ErrTxDone {
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
