package consumerapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/model/store/domain"
)

// SelectDefaultProvider reads Artifact metadata only. Built-in Providers are
// the fallback pool; selecting a user Provider does not require listing its
// Root or resolving any Models.
func (s *ManagementStoreFacade) SelectDefaultProvider(
	ctx context.Context,
	preferred *artifact.ArtifactRef,
	baseName basespec.LogicalName,
) (*artifact.ArtifactRef, error) {
	if s == nil || s.api == nil {
		return nil, basespec.ErrClosed
	}
	if err := s.api.ready(ctx); err != nil {
		return nil, err
	}

	if preferred != nil {
		record, err := s.api.artifacts.Get(ctx, *preferred)
		if err != nil {
			if !errors.Is(err, basespec.ErrArtifactNotFound) &&
				!errors.Is(err, basespec.ErrRootNotFound) &&
				!errors.Is(err, basespec.ErrRetired) {
				return nil, err
			}
		} else if record.Kind == modelDomain.ModelProviderArtifactKind &&
			record.State == artifact.StateAvailable &&
			record.Enabled {
			ref := record.Ref()
			return &ref, nil
		}
	}

	enabled := true
	options := catalog.ListOptions{
		Kind:    modelDomain.ModelProviderArtifactKind,
		Enabled: &enabled,
	}
	entries, err := s.api.artifacts.FindByIdentity(
		ctx,
		s.api.builtinRoot,
		modelDomain.ModelProviderArtifactKind,
		baseName,
		options,
	)
	if err != nil {
		return nil, err
	}
	if ref := firstAvailableProviderRef(entries); ref != nil {
		return ref, nil
	}

	// The baseline Provider may be disabled. Respect that choice and inspect
	// only built-in Provider metadata, not documents, overlays, or Models.
	entries, err = s.api.artifacts.ListByRoot(
		ctx,
		s.api.builtinRoot,
		options,
	)
	if err != nil {
		return nil, err
	}
	return firstAvailableProviderRef(entries), nil
}

// RequireSettableDefaultProvider verifies the current state required for an
// explicit user default-provider selection. It intentionally does not resolve
// a Model or decrypt a credential secret.
//
// A later Provider disablement or credential removal does not alter the saved
// preference. Default-provider reads retain their normal fallback behavior.
func (s *ManagementStoreFacade) RequireSettableDefaultProvider(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if s == nil || s.api == nil {
		return basespec.ErrClosed
	}
	if err := s.api.ready(ctx); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}

	record, err := s.api.requireKind(
		ctx,
		ref,
		modelDomain.ModelProviderArtifactKind,
	)
	if err != nil {
		return err
	}
	if record.State != artifact.StateAvailable || !record.Enabled {
		return fmt.Errorf(
			"%w: Model Provider %q must be enabled before it can be selected as default",
			basespec.ErrReferenceUnresolved,
			record.LogicalName,
		)
	}

	credential, found, err := s.api.overlays.GetProviderCredential(
		ctx,
		ref,
	)
	if err != nil {
		return err
	}
	if !found || !credential.Active() {
		return fmt.Errorf(
			"%w: Model Provider %q has no configured API key",
			basespec.ErrReferenceUnresolved,
			record.LogicalName,
		)
	}
	return nil
}

func firstAvailableProviderRef(
	entries []catalog.Entry,
) *artifact.ArtifactRef {
	var first *catalog.Entry
	for index := range entries {
		entry := &entries[index]
		if entry.Kind != modelDomain.ModelProviderArtifactKind ||
			entry.State != artifact.StateAvailable ||
			!entry.Enabled {
			continue
		}
		if first == nil ||
			entry.LogicalName < first.LogicalName ||
			(entry.LogicalName == first.LogicalName && entry.ID < first.ID) {
			first = entry
		}
	}
	if first == nil {
		return nil
	}
	ref := first.Ref()
	return &ref
}
