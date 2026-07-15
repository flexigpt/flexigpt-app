package artifactstore

import "sync"

type storeOperationLease struct {
	mu     sync.Mutex
	store  *Store
	refs   int
	active bool
}

func (l *storeOperationLease) retain() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.active {
		return false
	}
	l.refs++
	return true
}

func (l *storeOperationLease) release() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.active || l.refs <= 0 {
		return false
	}
	l.refs--
	if l.refs != 0 {
		return false
	}
	l.active = false
	return true
}
