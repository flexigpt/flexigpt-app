package internal

import (
	"context"
	"fmt"
	"slices"

	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type Registry struct {
	adapters map[source.SourceKind]Adapter
	kinds    []source.SourceKind
}

func NewRegistry(adapters ...Adapter) (*Registry, error) {
	values := make(map[source.SourceKind]Adapter, len(adapters))
	kinds := make([]source.SourceKind, 0, len(adapters))

	for _, adapter := range adapters {
		if adapter == nil {
			return nil, fmt.Errorf("%w: source adapter is nil", spec.ErrInvalid)
		}
		kind := adapter.Kind()
		if err := kind.Validate(); err != nil {
			return nil, err
		}
		if _, exists := values[kind]; exists {
			return nil, fmt.Errorf(
				"%w: duplicate source adapter %q",
				spec.ErrConflict,
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
	value source.Source,
) (Snapshot, error) {
	if ctx == nil {
		return nil, spec.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := value.Validate(); err != nil {
		return nil, err
	}
	adapter, exists := r.adapter(value.Kind)
	if !exists {
		return nil, fmt.Errorf(
			"%w: source adapter %q",
			spec.ErrSourceUnavailable,
			value.Kind,
		)
	}
	snapshot, err := adapter.Open(ctx, value.Clone())
	if err != nil {
		return nil, err
	}
	if err := validateSnapshot(snapshot); err != nil {
		if snapshot != nil {
			_ = snapshot.Close()
		}
		return nil, err
	}
	return snapshot, nil
}

func (r *Registry) SupportsLocalPath(
	kind source.SourceKind,
) bool {
	adapter, exists := r.adapter(kind)
	if !exists {
		return false
	}
	_, supported := adapter.(LocalPathResolver)
	return supported
}

func (r *Registry) SupportsManagedPackages(
	kind source.SourceKind,
) bool {
	adapter, exists := r.adapter(kind)
	if !exists {
		return false
	}
	_, supported := adapter.(ManagedPackageWriter)
	return supported
}

// ResolveLocalPath resolves a source-relative locator to a native absolute
// filesystem path when, and only when, the selected source adapter explicitly
// supports that capability.
//
// This is intentionally a trusted internal capability. Public source APIs
// continue to expose Summary values only and never reveal source paths.
func (r *Registry) ResolveLocalPath(
	ctx context.Context,
	value source.Source,
	locator spec.Locator,
) (string, error) {
	if ctx == nil {
		return "", spec.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := value.Validate(); err != nil {
		return "", err
	}
	if err := locator.Validate(true); err != nil {
		return "", err
	}
	adapter, exists := r.adapter(value.Kind)
	if !exists {
		return "", fmt.Errorf(
			"%w: source adapter %q",
			spec.ErrSourceUnavailable,
			value.Kind,
		)
	}
	resolver, supported := adapter.(LocalPathResolver)
	if !supported {
		return "", fmt.Errorf(
			"%w: source kind %q has no native filesystem path",
			spec.ErrUnsupported,
			value.Kind,
		)
	}
	location, err := resolver.ResolveLocalPath(
		ctx,
		value.Clone(),
		locator,
	)
	if err != nil {
		return "", err
	}
	return location, nil
}

func (r *Registry) PublishPackage(
	ctx context.Context,
	value source.Source,
	publication source.ManagedPackagePublication,
) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf(
			"%w: managed Source publication context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := value.Validate(); err != nil {
		return "", err
	}
	normalized, err := source.NormalizeManagedPackagePublication(publication)
	if err != nil {
		return "", err
	}

	adapter, exists := r.adapter(value.Kind)
	if !exists {
		return "", fmt.Errorf(
			"%w: source adapter %q",
			spec.ErrSourceUnavailable,
			value.Kind,
		)
	}
	writer, supported := adapter.(ManagedPackageWriter)
	if !supported {
		return "", fmt.Errorf(
			"%w: source kind %q is not writable",
			spec.ErrUnsupported,
			value.Kind,
		)
	}
	generation, err := writer.PublishPackage(
		ctx,
		value.Clone(),
		normalized,
	)
	if err != nil {
		return "", err
	}
	if err := spec.ValidateSourceGeneration(generation); err != nil {
		return "", fmt.Errorf(
			"%w: managed Source writer returned an invalid generation: %w",
			spec.ErrInvalid,
			err,
		)
	}
	return generation, nil
}

func (r *Registry) RemovePackage(
	ctx context.Context,
	value source.Source,
	address source.ManagedPackageAddress,
	expectedGeneration string,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: managed Source removal context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if err := address.Validate(); err != nil {
		return err
	}
	if err := spec.ValidateSourceGeneration(
		expectedGeneration,
	); err != nil {
		return err
	}

	adapter, exists := r.adapter(value.Kind)
	if !exists {
		return fmt.Errorf(
			"%w: source adapter %q",
			spec.ErrSourceUnavailable,
			value.Kind,
		)
	}
	writer, supported := adapter.(ManagedPackageWriter)
	if !supported {
		return fmt.Errorf(
			"%w: source kind %q is not writable",
			spec.ErrUnsupported,
			value.Kind,
		)
	}
	return writer.RemovePackage(
		ctx,
		value.Clone(),
		address,
		expectedGeneration,
	)
}

// RemoveManagedRoot removes adapter-owned managed storage for a complete Root.
// It is a trusted topology-maintenance capability and is intentionally not
// exposed by source.Service.
func (r *Registry) RemoveManagedRoot(
	ctx context.Context,
	rootStorageKey spec.StorageKey,
) error {
	if r == nil {
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: managed root removal context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := rootStorageKey.Validate(); err != nil {
		return err
	}

	for _, kind := range r.kinds {
		adapter := r.adapters[kind]
		remover, supported := adapter.(ManagedRootRemover)
		if !supported {
			continue
		}
		if err := remover.RemoveManagedRoot(ctx, rootStorageKey); err != nil {
			return fmt.Errorf(
				"remove managed storage for root storage key %q through adapter %q: %w",
				rootStorageKey,
				kind,
				err,
			)
		}
	}
	return nil
}

func (r *Registry) Kinds() []source.SourceKind {
	if r == nil {
		return nil
	}
	return append([]source.SourceKind(nil), r.kinds...)
}

func (r *Registry) adapter(
	kind source.SourceKind,
) (Adapter, bool) {
	if r == nil {
		return nil, false
	}
	value, exists := r.adapters[kind]
	return value, exists
}
