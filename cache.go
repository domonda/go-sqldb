package sqldb

import (
	"reflect"
	"sync"
)

type queryCache struct {
	query              string
	structFieldIndices [][]int
}

var (
	insertRowStructQueryCache    = make(map[reflect.Type]map[StructReflector]map[QueryBuilder]queryCache)
	insertRowStructQueryCacheMtx sync.RWMutex
)

// ClearQueryCaches clears all internal query caches.
// This is useful for testing and debugging to ensure
// that queries are rebuilt from scratch.
func ClearQueryCaches() {
	insertRowStructQueryCacheMtx.Lock()
	clear(insertRowStructQueryCache)
	insertRowStructQueryCacheMtx.Unlock()
}
