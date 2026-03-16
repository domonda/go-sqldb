package mysqlconn

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-sqldb"
)

func TestConnectWrongDriver(t *testing.T) {
	config := &sqldb.ConnConfig{
		Driver:   "postgres",
		Host:     "localhost",
		Database: "testdb",
	}
	conn, err := Connect(context.Background(), config)
	require.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "invalid driver")
}
