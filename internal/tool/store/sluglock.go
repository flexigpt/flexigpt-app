package store

import (
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
)

// slugLocks keeps a RW-mutex per bundle|slug pair so that concurrent access to
// the same (bundle, slug) is serialised while allowing full parallelism for
// different pairs.
type slugLocks struct {
	mu sync.Mutex
	m  map[string]*sync.RWMutex
}

// newSlugLocks constructs an empty slugLocks helper.
func newSlugLocks() *slugLocks {
	return &slugLocks{m: map[string]*sync.RWMutex{}}
}

// LockKey returns (and lazily creates) the mutex for the given bundle|slug.
func (l *slugLocks) LockKey(
	bundleID bundleitemutils.BundleID,
	slug bundleitemutils.ItemSlug,
) *sync.RWMutex {
	key := string(bundleID) + "|" + string(slug)

	l.mu.Lock()
	defer l.mu.Unlock()

	if lk, ok := l.m[key]; ok {
		return lk
	}
	lk := &sync.RWMutex{}
	l.m[key] = lk
	return lk
}
