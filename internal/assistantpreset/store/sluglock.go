package store

import (
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
)

// slugLocks manages a RW-mutex per bundleID|slug.
// This map may grow over time, which is acceptable for the expected bounded
// cardinality of bundle|slug combinations.
type slugLocks struct {
	mu sync.Mutex
	m  map[string]*sync.RWMutex
}

func newSlugLocks() *slugLocks {
	return &slugLocks{m: map[string]*sync.RWMutex{}}
}

func (l *slugLocks) lockKey(
	bundleID bundleitemutils.BundleID,
	slug bundleitemutils.ItemSlug,
) *sync.RWMutex {
	k := string(bundleID) + "|" + string(slug)

	l.mu.Lock()
	defer l.mu.Unlock()

	if lk, ok := l.m[k]; ok {
		return lk
	}

	lk := &sync.RWMutex{}
	l.m[k] = lk
	return lk
}
