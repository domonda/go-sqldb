package conntest

import (
	"net/mail"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runMailAddressTests(t *testing.T, config Config) {
	if config.DDL.CreateMailAddressTable == "" {
		t.Skip("CreateMailAddressTable DDL not provided")
	}

	t.Run("InsertAndQueryWithName", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateMailAddressTable, "conntest_mail_address")
		row := mailAddressRow{
			ID:    1,
			Email: &mail.Address{Name: "Alice", Address: "alice@example.com"},
		}

		// when
		insertMailAddressRow(t, conn, qb, row)
		got := queryMailAddressRow(t, conn, qb, 1)

		// then
		require.NotNil(t, got.Email)
		assert.Equal(t, "Alice", got.Email.Name)
		assert.Equal(t, "alice@example.com", got.Email.Address)
	})

	t.Run("InsertAndQueryWithoutName", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateMailAddressTable, "conntest_mail_address")
		row := mailAddressRow{
			ID:    2,
			Email: &mail.Address{Address: "bob@example.com"},
		}

		// when
		insertMailAddressRow(t, conn, qb, row)
		got := queryMailAddressRow(t, conn, qb, 2)

		// then
		require.NotNil(t, got.Email)
		assert.Equal(t, "", got.Email.Name)
		assert.Equal(t, "bob@example.com", got.Email.Address)
	})

	t.Run("InsertAndQueryNil", func(t *testing.T) {
		// given
		conn := config.NewConn(t)
		qb := config.QueryBuilder
		setupTable(t, conn, config.DDL.CreateMailAddressTable, "conntest_mail_address")
		row := mailAddressRow{
			ID:    3,
			Email: nil,
		}

		// when
		insertMailAddressRow(t, conn, qb, row)
		got := queryMailAddressRow(t, conn, qb, 3)

		// then
		assert.Nil(t, got.Email)
	})
}
