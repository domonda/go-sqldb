package sqldb

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/mail"
	"reflect"
)

var (
	mailAddressType    = reflect.TypeFor[mail.Address]()
	mailAddressPtrType = reflect.TypeFor[*mail.Address]()
)

// Ensure that MailAddressTypeWrapper implements TypeWrapper
var _ TypeWrapper = MailAddressTypeWrapper{}

// MailAddressTypeWrapper implements TypeWrapper for mail.Address and *mail.Address.
// It uses mail.ParseAddress to scan strings and byte arrays,
// and mail.Address.String() for the driver.Valuer implementation.
// Scanning NULL into mail.Address sets the zero value,
// scanning NULL into *mail.Address sets nil.
type MailAddressTypeWrapper struct{}

func (MailAddressTypeWrapper) WrapAsScanner(val reflect.Value) sql.Scanner {
	switch val.Type() {
	case mailAddressType, mailAddressPtrType:
		return &mailAddressScanner{ptr: val.Addr()}
	}
	return nil
}

func (MailAddressTypeWrapper) WrapAsValuer(val reflect.Value) driver.Valuer {
	switch val.Type() {
	case mailAddressType:
		return mailAddressValuer{addr: val.Addr().Interface().(*mail.Address)}
	case mailAddressPtrType:
		return mailAddressValuer{addr: val.Interface().(*mail.Address)}
	}
	return nil
}

type mailAddressScanner struct {
	ptr reflect.Value // pointer to mail.Address or *mail.Address
}

func (s *mailAddressScanner) Scan(src any) error {
	var str string
	switch src := src.(type) {
	case nil:
		s.ptr.Elem().Set(reflect.Zero(s.ptr.Elem().Type()))
		return nil
	case string:
		str = src
	case []byte:
		str = string(src)
	default:
		return fmt.Errorf("mailAddressScanner.Scan: unsupported type %T", src)
	}
	addr, err := mail.ParseAddress(str)
	if err != nil {
		return fmt.Errorf("mailAddressScanner.Scan: %w", err)
	}
	if s.ptr.Elem().Kind() == reflect.Pointer {
		s.ptr.Elem().Set(reflect.ValueOf(addr))
	} else {
		s.ptr.Elem().Set(reflect.ValueOf(*addr))
	}
	return nil
}

type mailAddressValuer struct {
	addr *mail.Address
}

func (v mailAddressValuer) Value() (driver.Value, error) {
	if v.addr == nil {
		return nil, nil
	}
	return v.addr.String(), nil
}
