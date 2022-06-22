package mockconn

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"sync"

	sqldb "github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/reflection"
)

type OneTimeRowsProvider struct {
	rowScanners  map[string]sqldb.Row
	rowsScanners map[string]sqldb.Rows
	mtx          sync.Mutex
}

func NewOneTimeRowsProvider() *OneTimeRowsProvider {
	return &OneTimeRowsProvider{
		rowScanners:  make(map[string]sqldb.Row),
		rowsScanners: make(map[string]sqldb.Rows),
	}
}

func (p *OneTimeRowsProvider) AddRowQuery(scanner sqldb.Row, query string, args ...any) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	key := uniqueQueryString(query, args)
	if _, exists := p.rowScanners[key]; exists {
		panic("query already added: " + key)
	}
	p.rowScanners[key] = scanner
}

func (p *OneTimeRowsProvider) AddRowsQuery(scanner sqldb.Rows, query string, args ...any) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	key := uniqueQueryString(query, args)
	if _, exists := p.rowScanners[key]; exists {
		panic("query already added: " + key)
	}
	p.rowsScanners[key] = scanner
}

func (p *OneTimeRowsProvider) QueryRow(structFieldMapper reflection.StructFieldMapper, query string, args ...any) sqldb.Row {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	key := uniqueQueryString(query, args)
	scanner := p.rowScanners[key]
	delete(p.rowScanners, key)
	return scanner
}

func (p *OneTimeRowsProvider) QueryRows(structFieldMapper reflection.StructFieldMapper, query string, args ...any) sqldb.Rows {
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
