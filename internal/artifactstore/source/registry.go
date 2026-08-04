package source

import (
	"context"
	"fmt"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type Registry struct {
	adapters map[basespec.SourceKind]Adapter
	kinds    []basespec.SourceKind
}

func NewRegistry(adapters ...Adapter) (*Registry, error) {
	values := make(map[basespec.SourceKind]Adapter, len(adapters))
	kinds := make([]basespec.SourceKind, 0, len(adapters))

	for _, adapter := range adapters {
		if adapter == nil {
			return nil, fmt.Errorf("%w: source adapter is nil", basespec.ErrInvalid)
		}
		kind := adapter.Kind()
		if err := basespec.ValidateSourceKind(kind); err != nil {
			return nil, err
		}
		if _, exists := values[kind]; exists {
			return nil, fmt.Errorf(
				"%w: duplicate source adapter %q",
				basespec.ErrConflict,
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
		return nil, basespec.ErrInvalid
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
			basespec.ErrSourceUnavailable,
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
	kind basespec.SourceKind,
) bool {
	adapter, exists := r.adapter(kind)
	if !exists {
		return false
	}
	_, supported := adapter.(LocalPathResolver)
	return supported
}

func (r *Registry) SupportsManagedPackages(
	kind basespec.SourceKind,
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
	value Source,
	locator basespec.Locator,
) (string, error) {
	if ctx == nil {
		return "", basespec.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := value.Validate(); err != nil {
		return "", err
	}
	if err := basespec.ValidateLocator(locator, true); err != nil {
		return "", err
	}
	adapter, exists := r.adapter(value.Kind)
	if !exists {
		return "", fmt.Errorf(
			"%w: source adapter %q",
			basespec.ErrSourceUnavailable,
			value.Kind,
		)
	}
	resolver, supported := adapter.(LocalPathResolver)
	if !supported {
		return "", fmt.Errorf(
			"%w: source kind %q has no native filesystem path",
			basespec.ErrUnsupported,
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
	value Source,
	publication ManagedPackagePublication,
) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf(
			"%w: managed Source publication context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := value.Validate(); err != nil {
		return "", err
	}
	normalized, err := NormalizeManagedPackagePublication(publication)
	if err != nil {
		return "", err
	}

	adapter, exists := r.adapter(value.Kind)
	if !exists {
		return "", fmt.Errorf(
			"%w: source adapter %q",
			basespec.ErrSourceUnavailable,
			value.Kind,
		)
	}
	writer, supported := adapter.(ManagedPackageWriter)
	if !supported {
		return "", fmt.Errorf(
			"%w: source kind %q is not writable",
			basespec.ErrUnsupported,
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
	if err := basespec.ValidateSourceGeneration(generation); err != nil {
		return "", fmt.Errorf(
			"%w: managed Source writer returned an invalid generation: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return generation, nil
}

func (r *Registry) RemovePackage(
	ctx context.Context,
	value Source,
	directory basespec.Locator,
	expectedGeneration string,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: managed Source removal context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if err := ValidateManagedPackageDirectory(directory); err != nil {
		return err
	}
	if err := basespec.ValidateSourceGeneration(
		expectedGeneration,
	); err != nil {
		return err
	}

	adapter, exists := r.adapter(value.Kind)
	if !exists {
		return fmt.Errorf(
			"%w: source adapter %q",
			basespec.ErrSourceUnavailable,
			value.Kind,
		)
	}
	writer, supported := adapter.(ManagedPackageWriter)
	if !supported {
		return fmt.Errorf(
			"%w: source kind %q is not writable",
			basespec.ErrUnsupported,
			value.Kind,
		)
	}
	return writer.RemovePackage(
		ctx,
		value.Clone(),
		directory,
		expectedGeneration,
	)
}

func (r *Registry) Kinds() []basespec.SourceKind {
	if r == nil {
		return nil
	}
	return append([]basespec.SourceKind(nil), r.kinds...)
}

func (r *Registry) adapter(
	kind basespec.SourceKind,
) (Adapter, bool) {
	if r == nil {
		return nil, false
	}
	value, exists := r.adapters[kind]
	return value, exists
}
