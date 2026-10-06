package source

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Registry struct {
	adapters map[sourceModel.SourceKind]driver.Driver
	kinds    []sourceModel.SourceKind
}

func NewRegistry(adapters ...driver.Driver) (*Registry, error) {
	values := make(map[sourceModel.SourceKind]driver.Driver, len(adapters))
	kinds := make([]sourceModel.SourceKind, 0, len(adapters))

	for _, adapter := range adapters {
		if adapter == nil {
			return nil, fmt.Errorf("%w: source adapter is nil", spec.ErrInvalid)
		}
		kind := adapter.Kind()
		if err := kind.Validate(); err != nil {
			return nil, err
		}
		if _, exists := values[kind]; exists {
			return nil, fmt.Errorf("%w: duplicate source adapter %q", spec.ErrConflict, kind)
		}
		values[kind] = adapter
		kinds = append(kinds, kind)
	}
	slices.Sort(kinds)
	return &Registry{adapters: values, kinds: kinds}, nil
}

// NormalizeConfig is the single Source driver-output boundary. It normalizes
// one caller configuration through its selected driver and takes one owned
// canonical copy for Source persistence.
func (r *Registry) NormalizeConfig(
	ctx context.Context,
	kind sourceModel.SourceKind,
	raw json.RawMessage,
) (json.RawMessage, error) {
	if err := kind.Validate(); err != nil {
		return nil, err
	}
	adapter, exists := r.adapter(kind)
	if !exists {
		return nil, fmt.Errorf("%w: source adapter %q", spec.ErrSourceUnavailable, kind)
	}
	normalized, err := adapter.NormalizeConfig(ctx, append(json.RawMessage(nil), raw...))
	if err != nil {
		return nil, err
	}
	canonical, err := jsonutil.CanonicalizeObject(normalized, spec.MaxConfigBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: normalized Source config: %w", spec.ErrInvalid, err)
	}
	return json.RawMessage(canonical), nil
}

func (r *Registry) Open(
	ctx context.Context,
	value sourceModel.Source,
) (driver.Snapshot, error) {
	adapter, exists := r.adapter(value.Kind)
	if !exists {
		return nil, fmt.Errorf("%w: source adapter %q", spec.ErrSourceUnavailable, value.Kind)
	}
	snapshot, err := adapter.Open(ctx, value.Clone())
	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

func (r *Registry) SupportsLocalPath(kind sourceModel.SourceKind) bool {
	adapter, exists := r.adapter(kind)
	if !exists {
		return false
	}
	_, supported := adapter.(driver.LocalPathResolver)
	return supported
}

func (r *Registry) SupportsManagedPackages(kind sourceModel.SourceKind) bool {
	adapter, exists := r.adapter(kind)
	if !exists {
		return false
	}
	_, supported := adapter.(managedpackage.Writer)
	return supported
}

// ResolveLocalPath resolves a source-relative locator to a native absolute
// filesystem path when, and only when, the selected source adapter explicitly
// supports that capability.
func (r *Registry) ResolveLocalPath(
	ctx context.Context,
	value sourceModel.Source,
	locator spec.Locator,
) (string, error) {
	adapter, exists := r.adapter(value.Kind)
	if !exists {
		return "", fmt.Errorf("%w: source adapter %q", spec.ErrSourceUnavailable, value.Kind)
	}
	resolver, supported := adapter.(driver.LocalPathResolver)
	if !supported {
		return "", fmt.Errorf("%w: source kind %q has no native filesystem path", spec.ErrUnsupported, value.Kind)
	}
	return resolver.ResolveLocalPath(ctx, value.Clone(), locator)
}

func (r *Registry) PublishPackage(
	ctx context.Context,
	value sourceModel.Source,
	publication managedpackageModel.ManagedPackagePublication,
) (string, error) {
	normalized, err := managedpackageModel.NormalizeManagedPackagePublication(publication)
	if err != nil {
		return "", err
	}
	adapter, exists := r.adapter(value.Kind)
	if !exists {
		return "", fmt.Errorf("%w: source adapter %q", spec.ErrSourceUnavailable, value.Kind)
	}
	writer, supported := adapter.(managedpackage.Writer)
	if !supported {
		return "", fmt.Errorf("%w: source kind %q is not writable", spec.ErrUnsupported, value.Kind)
	}
	generation, err := writer.PublishPackage(ctx, value.Clone(), normalized)
	if err != nil {
		return "", err
	}
	if err := spec.ValidateSourceGeneration(generation); err != nil {
		return "", fmt.Errorf("%w: managed Source writer returned an invalid generation: %w", spec.ErrInvalid, err)
	}
	return generation, nil
}

func (r *Registry) RemovePackage(
	ctx context.Context,
	value sourceModel.Source,
	address managedpackageModel.ManagedPackageAddress,
	expectedGeneration string,
) error {
	adapter, exists := r.adapter(value.Kind)
	if !exists {
		return fmt.Errorf("%w: source adapter %q", spec.ErrSourceUnavailable, value.Kind)
	}
	writer, supported := adapter.(managedpackage.Writer)
	if !supported {
		return fmt.Errorf("%w: source kind %q is not writable", spec.ErrUnsupported, value.Kind)
	}
	return writer.RemovePackage(ctx, value.Clone(), address, expectedGeneration)
}

// RemoveManagedRoot removes adapter-owned managed storage for a complete Root.
// It is a trusted topology-maintenance capability and is intentionally not
// exposed by source.Service.
func (r *Registry) RemoveManagedRoot(
	ctx context.Context,
	rootStorageKey spec.StorageKey,
) error {
	if err := rootStorageKey.Validate(); err != nil {
		return err
	}
	for _, kind := range r.kinds {
		remover, supported := r.adapters[kind].(driver.ManagedRootRemover)
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

func (r *Registry) Kinds() []sourceModel.SourceKind {
	if r == nil {
		return nil
	}
	return append([]sourceModel.SourceKind(nil), r.kinds...)
}

func (r *Registry) adapter(kind sourceModel.SourceKind) (driver.Driver, bool) {
	if r == nil {
		return nil, false
	}
	value, exists := r.adapters[kind]
	return value, exists
}
