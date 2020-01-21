package mockconn

import "sync"

type boolMap struct {
	m   map[string]bool
	mtx sync.Mutex
}

func newBoolMap() *boolMap {
	return &boolMap{m: make(map[string]bool)}
}

func (b *boolMap) Get(key string) bool {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	return b.m[key]
}

func (b *boolMap) Set(key string, val bool) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	b.m[key] = val
}
