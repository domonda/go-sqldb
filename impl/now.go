package impl

import (
	"time"

	"github.com/domonda/go-sqldb"
)

func Now(conn sqldb.Connection) (now time.Time, err error) {
	err = conn.QueryRow(`select now()`).Scan(&now)
	if err != nil {
		return time.Time{}, err
	}
	return now, nil
}
