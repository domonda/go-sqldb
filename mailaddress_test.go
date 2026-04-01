package sqldb

import (
	"net/mail"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMailAddressTypeWrapper_WrapAsScanner(t *testing.T) {
	var tw MailAddressTypeWrapper

	t.Run("nil for unsupported type", func(t *testing.T) {
		// given
		val := reflect.ValueOf(new(string)).Elem()

		// when
		scanner := tw.WrapAsScanner(val)

		// then
		assert.Nil(t, scanner)
	})

	t.Run("scan string into mail.Address", func(t *testing.T) {
		// given
		var addr mail.Address
		val := reflect.ValueOf(&addr).Elem()

		// when
		scanner := tw.WrapAsScanner(val)
		require.NotNil(t, scanner)
		err := scanner.Scan(`"Alice" <alice@example.com>`)

		// then
		require.NoError(t, err)
		assert.Equal(t, "Alice", addr.Name)
		assert.Equal(t, "alice@example.com", addr.Address)
	})

	t.Run("scan bytes into mail.Address", func(t *testing.T) {
		// given
		var addr mail.Address
		val := reflect.ValueOf(&addr).Elem()

		// when
		scanner := tw.WrapAsScanner(val)
		require.NotNil(t, scanner)
		err := scanner.Scan([]byte(`<bob@example.com>`))

		// then
		require.NoError(t, err)
		assert.Equal(t, "", addr.Name)
		assert.Equal(t, "bob@example.com", addr.Address)
	})

	t.Run("scan nil into mail.Address sets zero value", func(t *testing.T) {
		// given
		addr := mail.Address{Name: "Alice", Address: "alice@example.com"}
		val := reflect.ValueOf(&addr).Elem()

		// when
		scanner := tw.WrapAsScanner(val)
		require.NotNil(t, scanner)
		err := scanner.Scan(nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, mail.Address{}, addr)
	})

	t.Run("scan string into *mail.Address", func(t *testing.T) {
		// given
		var addr *mail.Address
		val := reflect.ValueOf(&addr).Elem()

		// when
		scanner := tw.WrapAsScanner(val)
		require.NotNil(t, scanner)
		err := scanner.Scan(`"Charlie" <charlie@example.com>`)

		// then
		require.NoError(t, err)
		require.NotNil(t, addr)
		assert.Equal(t, "Charlie", addr.Name)
		assert.Equal(t, "charlie@example.com", addr.Address)
	})

	t.Run("scan nil into *mail.Address sets nil", func(t *testing.T) {
		// given
		addr := &mail.Address{Name: "Alice", Address: "alice@example.com"}
		val := reflect.ValueOf(&addr).Elem()

		// when
		scanner := tw.WrapAsScanner(val)
		require.NotNil(t, scanner)
		err := scanner.Scan(nil)

		// then
		require.NoError(t, err)
		assert.Nil(t, addr)
	})

	t.Run("scan invalid string returns error", func(t *testing.T) {
		// given
		var addr mail.Address
		val := reflect.ValueOf(&addr).Elem()

		// when
		scanner := tw.WrapAsScanner(val)
		require.NotNil(t, scanner)
		err := scanner.Scan("not a valid address")

		// then
		assert.Error(t, err)
	})

	t.Run("scan unsupported type returns error", func(t *testing.T) {
		// given
		var addr mail.Address
		val := reflect.ValueOf(&addr).Elem()

		// when
		scanner := tw.WrapAsScanner(val)
		require.NotNil(t, scanner)
		err := scanner.Scan(42)

		// then
		assert.Error(t, err)
	})
}

func TestMailAddressTypeWrapper_WrapAsValuer(t *testing.T) {
	var tw MailAddressTypeWrapper

	t.Run("nil for unsupported type", func(t *testing.T) {
		// given
		val := reflect.ValueOf(new(string)).Elem()

		// when
		valuer := tw.WrapAsValuer(val)

		// then
		assert.Nil(t, valuer)
	})

	t.Run("value from mail.Address", func(t *testing.T) {
		// given
		addr := mail.Address{Name: "Alice", Address: "alice@example.com"}
		val := reflect.ValueOf(&addr).Elem()

		// when
		valuer := tw.WrapAsValuer(val)
		require.NotNil(t, valuer)
		v, err := valuer.Value()

		// then
		require.NoError(t, err)
		assert.Equal(t, `"Alice" <alice@example.com>`, v)
	})

	t.Run("value from *mail.Address", func(t *testing.T) {
		// given
		addr := &mail.Address{Name: "Bob", Address: "bob@example.com"}
		val := reflect.ValueOf(&addr).Elem()

		// when
		valuer := tw.WrapAsValuer(val)
		require.NotNil(t, valuer)
		v, err := valuer.Value()

		// then
		require.NoError(t, err)
		assert.Equal(t, `"Bob" <bob@example.com>`, v)
	})

	t.Run("value from nil *mail.Address", func(t *testing.T) {
		// given
		var addr *mail.Address
		val := reflect.ValueOf(&addr).Elem()

		// when
		valuer := tw.WrapAsValuer(val)
		require.NotNil(t, valuer)
		v, err := valuer.Value()

		// then
		require.NoError(t, err)
		assert.Nil(t, v)
	})

	t.Run("value from mail.Address without name", func(t *testing.T) {
		// given
		addr := mail.Address{Address: "alice@example.com"}
		val := reflect.ValueOf(&addr).Elem()

		// when
		valuer := tw.WrapAsValuer(val)
		require.NotNil(t, valuer)
		v, err := valuer.Value()

		// then
		require.NoError(t, err)
		assert.Equal(t, `<alice@example.com>`, v)
	})
}
