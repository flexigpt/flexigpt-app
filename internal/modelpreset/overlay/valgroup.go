package overlay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// TypedGroup provides type-safe access to a specific group in the store.
type TypedGroup[K Key, ValT any] struct {
	store   *Store
	groupID GroupID
}

// NewTypedGroup creates a typed group accessor - this is the correct way since
// Go doesn't allow type parameters on struct methods.
func NewTypedGroup[K Key, ValT any](
	ctx context.Context,
	store *Store,
) (*TypedGroup[K, ValT], error) {
	if store == nil {
		return nil, errors.New("overlay: cannot have nil store")
	}
	var k K
	group := k.Group()
	if err := store.ensureRegistered(group); err != nil {
		return nil, fmt.Errorf("overlay: group %q is not registered", group)
	}

	return &TypedGroup[K, ValT]{
		store:   store,
		groupID: k.Group(),
	}, nil
}

// GetFlag retrieves a typed value from the group.
func (g *TypedGroup[K, ValT]) GetFlag(ctx context.Context, k K) (TypedFlag[ValT], bool, error) {
	var zero TypedFlag[ValT]

	flag, exists, err := g.store.GetFlag(ctx, k)
	if err != nil {
		return zero, false, err
	}
	if !exists {
		return zero, false, nil
	}

	var value ValT
	if err := json.Unmarshal(flag.Value, &value); err != nil {
		return zero, false, err
	}

	return TypedFlag[ValT]{
		Value:      value,
		CreatedAt:  flag.CreatedAt,
		ModifiedAt: flag.ModifiedAt,
	}, true, nil
}

// SetFlag stores a typed value in the group.
func (g *TypedGroup[K, ValT]) SetFlag(
	ctx context.Context,
	k K,
	value ValT,
) (TypedFlag[ValT], error) {
	var zero TypedFlag[ValT]
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return zero, err
	}

	f, err := g.store.SetFlag(ctx, k, jsonValue)
	if err != nil {
		return zero, err
	}
	return TypedFlag[ValT]{
		Value:      value,
		CreatedAt:  f.CreatedAt,
		ModifiedAt: f.ModifiedAt,
	}, nil
}

// DeleteKey removes an entry from the group.
func (g *TypedGroup[K, ValT]) DeleteKey(ctx context.Context, k K) error {
	return g.store.DeleteKey(ctx, k)
}
