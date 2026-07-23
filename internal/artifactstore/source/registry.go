package source

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Registry struct {
	adapters map[artifactstore.SourceKind]Adapter
	kinds    []artifactstore.SourceKind
}

func NewRegistry(adapters ...Adapter) (*Registry, error) {
	values := make(map[artifactstore.SourceKind]Adapter, len(adapters))
	kinds := make([]artifactstore.SourceKind, 0, len(adapters))

	for _, adapter := range adapters {
		if adapter == nil {
			return nil, fmt.Errorf("%w: source adapter is nil", artifactstore.ErrInvalid)
		}
		kind := adapter.Kind()
		if err := artifactstore.ValidateSourceKind(kind); err != nil {
			return nil, err
		}
		if _, exists := values[kind]; exists {
			return nil, fmt.Errorf(
				"%w: duplicate source adapter %q",
				artifactstore.ErrConflict,
				kind,
			)
		}
		values[kind] = adapter
		kinds = append(kinds, kind)
	}
	slices.Sort(kinds)
	return &Registry{adapters: values, kinds: kinds}, nil
}

func (r *Registry) Open(
	ctx context.Context,
	value Source,
) (Snapshot, error) {
	if ctx == nil {
		return nil, artifactstore.ErrInvalid
	}
	if err := value.Validate(); err != nil {
		return nil, err
	}
	adapter, exists := r.adapter(value.Kind)
	if !exists {
		return nil, fmt.Errorf(
			"%w: source adapter %q",
			artifactstore.ErrSourceUnavailable,
			value.Kind,
		)
	}
	snapshot, err := adapter.Open(ctx, value.Clone())
	if err != nil {
		return nil, err
	}
	if err := validateSnapshot(snapshot); err != nil {
		_ = snapshot.Close()
		return nil, err
	}
	return snapshot, nil
}

// ResolveLocalPath resolves a source-relative locator to a native absolute
// filesystem path when, and only when, the selected source adapter explicitly
// supports that capability.
//
// This is intentionally a trusted internal capability. Public source APIs
// continue to expose Summary values only and never reveal source paths.
func (r *Registry) ResolveLocalPath(
	ctx context.Context,
	value Source,
	locator artifactstore.Locator,
) (string, error) {
	if ctx == nil {
		return "", artifactstore.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := value.Validate(); err != nil {
		return "", err
	}
	if err := artifactstore.ValidateLocator(locator, true); err != nil {
		return "", err
	}
	adapter, exists := r.adapter(value.Kind)
	if !exists {
		return "", fmt.Errorf(
			"%w: source adapter %q",
			artifactstore.ErrSourceUnavailable,
			value.Kind,
		)
	}
	resolver, supported := adapter.(LocalPathResolver)
	if !supported {
		return "", fmt.Errorf(
			"%w: source kind %q has no native filesystem path",
			artifactstore.ErrUnsupported,
			value.Kind,
		)
	}
	location, err := resolver.ResolveLocalPath(ctx, value.Clone(), locator)
	if err != nil {
		return "", err
	}
	location = filepath.Clean(location)
	if !filepath.IsAbs(location) {
		return "", fmt.Errorf(
			"%w: source adapter returned a non-absolute local path",
			artifactstore.ErrInvalid,
		)
	}
	return location, nil
}

func (r *Registry) Kinds() []artifactstore.SourceKind {
	if r == nil {
		return nil
	}
	return append([]artifactstore.SourceKind(nil), r.kinds...)
}

func (r *Registry) adapter(
	kind artifactstore.SourceKind,
) (Adapter, bool) {
	if r == nil {
		return nil, false
	}
	value, exists := r.adapters[kind]
	return value, exists
}
