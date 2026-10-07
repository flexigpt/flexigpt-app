package model

import (
	"context"
	"errors"
	"fmt"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/domain"
)

// SelectDefaultProvider reads Artifact metadata only. Built-in Providers are
// the fallback pool; selecting a user Provider does not require listing its
// Root or resolving any Models.
func (a *Service) SelectDefaultProvider(
	ctx context.Context,
	preferred *artifactModel.ArtifactRef,
	baseName spec.LogicalName,
) (*artifactModel.ArtifactRef, error) {
	if preferred != nil {
		record, err := a.artifacts.Get(ctx, *preferred)
		if err != nil {
			if !errors.Is(err, spec.ErrArtifactNotFound) &&
				!errors.Is(err, spec.ErrRootNotFound) &&
				!errors.Is(err, spec.ErrRetired) {
				return nil, err
			}
		} else if record.Kind == modelDomain.ModelProviderArtifactKind &&
			record.State == artifactModel.StateAvailable &&
			record.Enabled {
			ref := record.Ref()
			return &ref, nil
		}
	}

	enabled := true
	options := catalogModel.ListOptions{
		Kind:    modelDomain.ModelProviderArtifactKind,
		Enabled: &enabled,
	}
	entries, err := a.cat.FindByIdentity(
		ctx,
		a.builtinRoot,
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
	entries, err = a.cat.ListByRoot(
		ctx,
		a.builtinRoot,
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
func (a *Service) RequireSettableDefaultProvider(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	if err := ref.Validate(); err != nil {
		return err
	}

	record, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelProviderArtifactKind,
	)
	if err != nil {
		return err
	}
	if record.State != artifactModel.StateAvailable || !record.Enabled {
		return fmt.Errorf(
			"%w: Model Provider %q must be enabled before it can be selected as default",
			spec.ErrReferenceUnresolved,
			record.LogicalName,
		)
	}

	credential, found, err := a.overlays.GetProviderCredential(
		ctx,
		ref,
	)
	if err != nil {
		return err
	}
	if !found || !credential.Active() {
		return fmt.Errorf(
			"%w: Model Provider %q has no configured API key",
			spec.ErrReferenceUnresolved,
			record.LogicalName,
		)
	}
	return nil
}

func firstAvailableProviderRef(
	entries []catalogModel.Entry,
) *artifactModel.ArtifactRef {
	var first *catalogModel.Entry
	for index := range entries {
		entry := &entries[index]
		if entry.Kind != modelDomain.ModelProviderArtifactKind ||
			entry.State != artifactModel.StateAvailable ||
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
