package mockconn

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"sync"

	sqldb "github.com/domonda/go-sqldb"
)

type OneTimeRowsProvider struct {
	rowScanners  map[string]sqldb.RowScanner
	rowsScanners map[string]sqldb.RowsScanner
	mtx          sync.Mutex
}

func NewOneTimeRowsProvider() *OneTimeRowsProvider {
	return &OneTimeRowsProvider{
		rowScanners:  make(map[string]sqldb.RowScanner),
		rowsScanners: make(map[string]sqldb.RowsScanner),
	}
}

func (p *OneTimeRowsProvider) AddRowScannerQuery(scanner sqldb.RowScanner, query string, args ...any) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	key := uniqueQueryString(query, args)
	if _, exists := p.rowScanners[key]; exists {
		panic("query already added: " + key)
	}
	p.rowScanners[key] = scanner
}

func (p *OneTimeRowsProvider) AddRowsScannerQuery(scanner sqldb.RowsScanner, query string, args ...any) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	key := uniqueQueryString(query, args)
	if _, exists := p.rowScanners[key]; exists {
		panic("query already added: " + key)
	}
	p.rowsScanners[key] = scanner
}

func (p *OneTimeRowsProvider) QueryRow(structFieldNamer sqldb.StructFieldMapper, query string, args ...any) sqldb.RowScanner {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	key := uniqueQueryString(query, args)
	scanner := p.rowScanners[key]
	delete(p.rowScanners, key)
	return scanner
}

func (p *OneTimeRowsProvider) QueryRows(structFieldNamer sqldb.StructFieldMapper, query string, args ...any) sqldb.RowsScanner {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	key := uniqueQueryString(query, args)
	scanner := p.rowsScanners[key]
	delete(p.rowScanners, key)
	return scanner
}

func uniqueQueryString(query string, args []any) string {
	var b strings.Builder
	b.WriteString(query)
	for _, arg := range args {
		if valuer, ok := arg.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				panic(err)
			}
			arg = val
		}
		b.WriteString(fmt.Sprint(arg))
	}
	return b.String()
}
