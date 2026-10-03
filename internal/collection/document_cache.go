package collection

import (
	"container/list"
	"sync"

	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
)

const (
	documentCacheEntries    = 256
	documentCacheInputBytes = 8 << 20
	documentCacheEntryCost  = 256
)

// DocumentCache holds immutable in-process projections of admitted Definitions.
// Returned values are borrowed. The owning consumer must clone before exposing
// maps, slices, or pointers to callers.
//
// The byte limit accounts for input size, not an exact Go heap measurement.
// The independent entry-count limit also bounds small-document overhead.
type DocumentCache[T any] struct {
	mu     sync.Mutex
	values map[definitionModel.Key]*list.Element
	order  list.List
	bytes  int
}

type documentCacheValue[T any] struct {
	key   definitionModel.Key
	value T
	cost  int
}

func (c *DocumentCache[T]) Get(
	key definitionModel.Key,
) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element := c.values[key]
	if element == nil {
		var zero T
		return zero, false
	}
	c.order.MoveToFront(element)
	//nolint:errcheck,forcetypeassert // Ok.
	return element.Value.(documentCacheValue[T]).value, true
}

func (c *DocumentCache[T]) GetOrLoad(
	key definitionModel.Key,
	inputBytes int,
	load func() (T, error),
) (T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element := c.values[key]; element != nil {
		c.order.MoveToFront(element)
		//nolint:errcheck,forcetypeassert // Ok.
		return element.Value.(documentCacheValue[T]).value, nil
	}

	value, err := load()
	if err != nil {
		return value, err
	}
	if inputBytes < 0 || inputBytes > documentCacheInputBytes-documentCacheEntryCost {
		return value, nil
	}
	cost := inputBytes + documentCacheEntryCost
	if c.values == nil {
		c.values = make(map[definitionModel.Key]*list.Element)
	}
	for len(c.values) >= documentCacheEntries || c.bytes+cost > documentCacheInputBytes {
		element := c.order.Back()
		//nolint:errcheck,forcetypeassert // Ok.
		previous := element.Value.(documentCacheValue[T])
		delete(c.values, previous.key)
		c.order.Remove(element)
		c.bytes -= previous.cost
	}
	c.values[key] = c.order.PushFront(documentCacheValue[T]{
		key:   key,
		value: value,
		cost:  cost,
	})
	c.bytes += cost
	return value, nil
}
