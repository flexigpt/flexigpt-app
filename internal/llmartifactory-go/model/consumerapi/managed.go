package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	managepackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/model"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/decoder"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/domain"
)

func (a *API) CreateProvider(
	ctx context.Context,
	request ManagedProviderCreateRequest,
) (ManagedProviderCreateResult, error) {
	if err := a.ready(ctx); err != nil {
		return ManagedProviderCreateResult{}, err
	}
	if err := request.RootID.Validate(); err != nil {
		return ManagedProviderCreateResult{}, err
	}
	if a.protection.IsProtectedRoot(request.RootID) {
		return ManagedProviderCreateResult{}, fmt.Errorf(
			"%w: managed Model Provider publication is not allowed in a protected Root",
			spec.ErrProtected,
		)
	}

	definitionValue, raw, err := providerDefinition(request.Document)
	if err != nil {
		return ManagedProviderCreateResult{}, err
	}
	name := request.Document.Name
	address, err := modelDomain.ModelProviderPackageAddress(name)
	if err != nil {
		return ManagedProviderCreateResult{}, err
	}
	locator, err := modelDomain.ModelProviderPackageLocator(address)
	if err != nil {
		return ManagedProviderCreateResult{}, err
	}

	sourceValue, err := a.ensureManagedDeclarationDiscovery(
		ctx,
		request.RootID,
		locator,
		topology.DocumentUseManagedModelProvider,
	)
	if err != nil {
		return ManagedProviderCreateResult{}, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		managepackageModel.PublishRequest{
			RootID: request.RootID,
			Binding: artifactModel.SourceBinding{
				SourceID: sourceValue.ID,
				Locator:  locator,
			},
			ExpectedKind:        modelDomain.ModelProviderArtifactKind,
			ExpectedLogicalName: name,
			ExpectedDefinition:  definitionValue.Digest,
			Package: managedpackageModel.ManagedPackagePublication{
				Address: address,
				Files: []managedpackageModel.ManagedPackageFile{{
					Locator: modelDomain.ModelProviderDocumentFile(),
					Content: raw,
				}},
			},
		},
	)
	if err != nil {
		return ManagedProviderCreateResult{}, err
	}

	record := published.Artifact
	if record.Enabled != request.Enabled {
		record, err = a.artifacts.SetEnabled(
			ctx,
			record.Ref(),
			record.Revision,
			request.Enabled,
		)
		if err != nil {
			return ManagedProviderCreateResult{}, err
		}
	}

	return ManagedProviderCreateResult{
		Artifact: record,
		Address:  record.Address(),
	}, nil
}

func (a *API) ReplaceProvider(
	ctx context.Context,
	request ManagedProviderReplaceRequest,
) (ManagedProviderReplaceResult, error) {
	if err := a.ready(ctx); err != nil {
		return ManagedProviderReplaceResult{}, err
	}
	if err := request.Provider.Validate(); err != nil {
		return ManagedProviderReplaceResult{}, err
	}
	if err := validateExpectedArtifactRevision(
		request.ExpectedArtifactRevision,
	); err != nil {
		return ManagedProviderReplaceResult{}, err
	}

	current, _, err := a.managedRecord(
		ctx,
		request.Provider,
		modelDomain.ModelProviderArtifactKind,
	)
	if err != nil {
		return ManagedProviderReplaceResult{}, err
	}
	if current.State != artifactModel.StateAvailable {
		return ManagedProviderReplaceResult{}, fmt.Errorf(
			"%w: Model Provider Artifact is unavailable",
			spec.ErrReferenceUnresolved,
		)
	}
	if current.Revision != request.ExpectedArtifactRevision {
		return ManagedProviderReplaceResult{}, spec.ErrConflict
	}
	if current.Binding.SubresourceLocator != "" {
		return ManagedProviderReplaceResult{}, fmt.Errorf(
			"%w: contained Model Provider declarations cannot be replaced as managed packages",
			spec.ErrUnsupported,
		)
	}
	if request.Document.Name != current.LogicalName {
		return ManagedProviderReplaceResult{}, fmt.Errorf(
			"%w: replacement Model Provider logical name must remain %q",
			spec.ErrInvalid,
			current.LogicalName,
		)
	}

	definitionValue, raw, err := providerDefinition(request.Document)
	if err != nil {
		return ManagedProviderReplaceResult{}, err
	}
	currentAddress, err := modelDomain.ModelProviderPackageAddressFromLocator(
		current.Binding.Locator,
	)
	if err != nil {
		return ManagedProviderReplaceResult{}, err
	}
	requestedAddress, err := modelDomain.ModelProviderPackageAddress(request.Document.Name)
	if err != nil {
		return ManagedProviderReplaceResult{}, err
	}
	if requestedAddress != currentAddress {
		return ManagedProviderReplaceResult{}, fmt.Errorf(
			"%w: replacement Model Provider cannot change package identity",
			spec.ErrInvalid,
		)
	}

	if _, err := a.ensureManagedDeclarationDiscovery(
		ctx,
		current.RootID,
		current.Binding.Locator,
		topology.DocumentUseManagedModelProvider,
	); err != nil {
		return ManagedProviderReplaceResult{}, err
	}
	generation, err := a.currentManagedSourceGeneration(
		ctx,
		current.RootID,
		current.Binding.SourceID,
	)
	if err != nil {
		return ManagedProviderReplaceResult{}, err
	}

	refreshedCurrent, err := a.artifacts.Get(ctx, request.Provider)
	if err != nil {
		return ManagedProviderReplaceResult{}, err
	}
	if refreshedCurrent.Revision != request.ExpectedArtifactRevision {
		return ManagedProviderReplaceResult{}, spec.ErrConflict
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		managepackageModel.PublishRequest{
			RootID: current.RootID,
			Binding: artifactModel.SourceBinding{
				SourceID: current.Binding.SourceID,
				Locator:  current.Binding.Locator,
			},
			ExpectedKind:            modelDomain.ModelProviderArtifactKind,
			ExpectedLogicalName:     current.LogicalName,
			ExpectedDefinition:      definitionValue.Digest,
			AllowPackageReplacement: true,
			Package: managedpackageModel.ManagedPackagePublication{
				Address:            currentAddress,
				ExpectedGeneration: generation,
				Files: []managedpackageModel.ManagedPackageFile{{
					Locator: modelDomain.ModelProviderDocumentFile(),
					Content: raw,
				}},
			},
		},
	)
	if err != nil {
		return ManagedProviderReplaceResult{}, err
	}
	if published.Artifact.Ref() != request.Provider {
		return ManagedProviderReplaceResult{}, fmt.Errorf(
			"%w: replacement published another Model Provider Artifact",
			spec.ErrConflict,
		)
	}

	record := published.Artifact
	if record.Enabled != request.Enabled {
		record, err = a.artifacts.SetEnabled(
			ctx,
			record.Ref(),
			record.Revision,
			request.Enabled,
		)
		if err != nil {
			return ManagedProviderReplaceResult{}, err
		}
	}

	return ManagedProviderReplaceResult{
		Artifact: record,
		Address:  record.Address(),
	}, nil
}

// DeleteProvider removes only the Provider package. It intentionally does not
// inspect, mutate, disable, delete, or purge Models that reference it.
func (a *API) DeleteProvider(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedArtifactRevision uint64,
) error {
	if err := a.ready(ctx); err != nil {
		return err
	}
	if err := validateExpectedArtifactRevision(
		expectedArtifactRevision,
	); err != nil {
		return err
	}

	record, _, err := a.managedRecord(
		ctx,
		ref,
		modelDomain.ModelProviderArtifactKind,
	)
	if err != nil {
		return err
	}
	if record.State != artifactModel.StateAvailable {
		return fmt.Errorf(
			"%w: Model Provider Artifact is unavailable",
			spec.ErrReferenceUnresolved,
		)
	}
	if record.Revision != expectedArtifactRevision {
		return spec.ErrConflict
	}

	address, err := modelDomain.ModelProviderPackageAddressFromLocator(
		record.Binding.Locator,
	)
	if err != nil {
		return err
	}
	generation, err := a.currentManagedSourceGeneration(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return err
	}

	locator := record.Binding.Locator
	if err := a.managedArtifacts.Remove(
		ctx,
		managepackageModel.RemoveRequest{
			RootID:                record.RootID,
			SourceID:              record.Binding.SourceID,
			Package:               address,
			ExpectedGeneration:    generation,
			ExpectedArtifact:      &ref,
			PruneDiscoveryLocator: &locator,
		},
	); err != nil {
		return err
	}

	missing, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return err
	}
	if missing.State != artifactModel.StateMissing {
		return fmt.Errorf(
			"%w: removed Model Provider Artifact is not missing",
			spec.ErrConflict,
		)
	}

	// The Model local-state repository removes the mutable Data namespace,
	// protected overlay when applicable, secret bindings, and queues physical
	// secret cleanup through Artifact Store.
	if err := a.purgeProviderLocalState(ctx, ref); err != nil {
		return fmt.Errorf(
			"model provider package was removed but local-state cleanup is pending: %w",
			err,
		)
	}
	return nil
}

func (a *API) CreateModel(
	ctx context.Context,
	request ManagedModelCreateRequest,
) (ManagedModelCreateResult, error) {
	if err := a.ready(ctx); err != nil {
		return ManagedModelCreateResult{}, err
	}
	if err := request.RootID.Validate(); err != nil {
		return ManagedModelCreateResult{}, err
	}
	if a.protection.IsProtectedRoot(request.RootID) {
		return ManagedModelCreateResult{}, fmt.Errorf(
			"%w: managed Model publication is not allowed in a protected Root",
			spec.ErrProtected,
		)
	}

	definitionValue, raw, err := modelDefinition(request.Document)
	if err != nil {
		return ManagedModelCreateResult{}, err
	}
	name := request.Document.Name
	address, err := modelDomain.ModelPackageAddress(name)
	if err != nil {
		return ManagedModelCreateResult{}, err
	}
	locator, err := modelDomain.ModelPackageLocator(address)
	if err != nil {
		return ManagedModelCreateResult{}, err
	}

	sourceValue, err := a.ensureManagedDeclarationDiscovery(
		ctx,
		request.RootID,
		locator,
		topology.DocumentUseManagedModel,
	)
	if err != nil {
		return ManagedModelCreateResult{}, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		managepackageModel.PublishRequest{
			RootID: request.RootID,
			Binding: artifactModel.SourceBinding{
				SourceID: sourceValue.ID,
				Locator:  locator,
			},
			ExpectedKind:        modelDomain.ModelArtifactKind,
			ExpectedLogicalName: name,
			ExpectedDefinition:  definitionValue.Digest,
			Package: managedpackageModel.ManagedPackagePublication{
				Address: address,
				Files: []managedpackageModel.ManagedPackageFile{{
					Locator: modelDomain.ModelDocumentFile(),
					Content: raw,
				}},
			},
		},
	)
	if err != nil {
		return ManagedModelCreateResult{}, err
	}

	record := published.Artifact
	if record.Enabled != request.Enabled {
		record, err = a.artifacts.SetEnabled(
			ctx,
			record.Ref(),
			record.Revision,
			request.Enabled,
		)
		if err != nil {
			return ManagedModelCreateResult{}, err
		}
	}

	return ManagedModelCreateResult{
		Artifact: record,
		Address:  record.Address(),
	}, nil
}

func (a *API) ReplaceModel(
	ctx context.Context,
	request ManagedModelReplaceRequest,
) (ManagedModelReplaceResult, error) {
	if err := a.ready(ctx); err != nil {
		return ManagedModelReplaceResult{}, err
	}
	if err := request.Model.Validate(); err != nil {
		return ManagedModelReplaceResult{}, err
	}
	if err := validateExpectedArtifactRevision(
		request.ExpectedArtifactRevision,
	); err != nil {
		return ManagedModelReplaceResult{}, err
	}

	current, _, err := a.managedRecord(
		ctx,
		request.Model,
		modelDomain.ModelArtifactKind,
	)
	if err != nil {
		return ManagedModelReplaceResult{}, err
	}
	if current.State != artifactModel.StateAvailable {
		return ManagedModelReplaceResult{}, fmt.Errorf(
			"%w: Model Artifact is unavailable",
			spec.ErrReferenceUnresolved,
		)
	}
	if current.Revision != request.ExpectedArtifactRevision {
		return ManagedModelReplaceResult{}, spec.ErrConflict
	}
	if current.Binding.SubresourceLocator != "" {
		return ManagedModelReplaceResult{}, fmt.Errorf(
			"%w: contained Model declarations cannot be replaced as managed packages",
			spec.ErrUnsupported,
		)
	}
	if request.Document.Name != current.LogicalName {
		return ManagedModelReplaceResult{}, fmt.Errorf(
			"%w: replacement Model logical name must remain %q",
			spec.ErrInvalid,
			current.LogicalName,
		)
	}

	definitionValue, raw, err := modelDefinition(request.Document)
	if err != nil {
		return ManagedModelReplaceResult{}, err
	}
	currentAddress, err := modelDomain.ModelPackageAddressFromLocator(
		current.Binding.Locator,
	)
	if err != nil {
		return ManagedModelReplaceResult{}, err
	}
	requestedAddress, err := modelDomain.ModelPackageAddress(request.Document.Name)
	if err != nil {
		return ManagedModelReplaceResult{}, err
	}
	if requestedAddress != currentAddress {
		return ManagedModelReplaceResult{}, fmt.Errorf(
			"%w: replacement Model cannot change package identity",
			spec.ErrInvalid,
		)
	}

	if _, err := a.ensureManagedDeclarationDiscovery(
		ctx,
		current.RootID,
		current.Binding.Locator,
		topology.DocumentUseManagedModel,
	); err != nil {
		return ManagedModelReplaceResult{}, err
	}
	generation, err := a.currentManagedSourceGeneration(
		ctx,
		current.RootID,
		current.Binding.SourceID,
	)
	if err != nil {
		return ManagedModelReplaceResult{}, err
	}

	refreshedCurrent, err := a.artifacts.Get(ctx, request.Model)
	if err != nil {
		return ManagedModelReplaceResult{}, err
	}
	if refreshedCurrent.Revision != request.ExpectedArtifactRevision {
		return ManagedModelReplaceResult{}, spec.ErrConflict
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		managepackageModel.PublishRequest{
			RootID: current.RootID,
			Binding: artifactModel.SourceBinding{
				SourceID: current.Binding.SourceID,
				Locator:  current.Binding.Locator,
			},
			ExpectedKind:            modelDomain.ModelArtifactKind,
			ExpectedLogicalName:     current.LogicalName,
			ExpectedDefinition:      definitionValue.Digest,
			AllowPackageReplacement: true,
			Package: managedpackageModel.ManagedPackagePublication{
				Address:            currentAddress,
				ExpectedGeneration: generation,
				Files: []managedpackageModel.ManagedPackageFile{{
					Locator: modelDomain.ModelDocumentFile(),
					Content: raw,
				}},
			},
		},
	)
	if err != nil {
		return ManagedModelReplaceResult{}, err
	}
	if published.Artifact.Ref() != request.Model {
		return ManagedModelReplaceResult{}, fmt.Errorf(
			"%w: replacement published another Model Artifact",
			spec.ErrConflict,
		)
	}

	record := published.Artifact
	if record.Enabled != request.Enabled {
		record, err = a.artifacts.SetEnabled(
			ctx,
			record.Ref(),
			record.Revision,
			request.Enabled,
		)
		if err != nil {
			return ManagedModelReplaceResult{}, err
		}
	}

	return ManagedModelReplaceResult{
		Artifact: record,
		Address:  record.Address(),
	}, nil
}

func (a *API) DeleteModel(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedArtifactRevision uint64,
) error {
	if err := a.ready(ctx); err != nil {
		return err
	}
	if err := validateExpectedArtifactRevision(
		expectedArtifactRevision,
	); err != nil {
		return err
	}

	record, _, err := a.managedRecord(
		ctx,
		ref,
		modelDomain.ModelArtifactKind,
	)
	if err != nil {
		return err
	}
	if record.State != artifactModel.StateAvailable {
		return fmt.Errorf(
			"%w: Model Artifact is unavailable",
			spec.ErrReferenceUnresolved,
		)
	}
	if record.Revision != expectedArtifactRevision {
		return spec.ErrConflict
	}

	address, err := modelDomain.ModelPackageAddressFromLocator(
		record.Binding.Locator,
	)
	if err != nil {
		return err
	}
	generation, err := a.currentManagedSourceGeneration(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return err
	}

	locator := record.Binding.Locator
	if err := a.managedArtifacts.Remove(
		ctx,
		managepackageModel.RemoveRequest{
			RootID:                record.RootID,
			SourceID:              record.Binding.SourceID,
			Package:               address,
			ExpectedGeneration:    generation,
			ExpectedArtifact:      &ref,
			PruneDiscoveryLocator: &locator,
		},
	); err != nil {
		return err
	}

	missing, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return err
	}
	if missing.State != artifactModel.StateMissing {
		return fmt.Errorf(
			"%w: removed Model Artifact is not missing",
			spec.ErrConflict,
		)
	}

	if err := a.purgeModelLocalState(ctx, ref); err != nil {
		return fmt.Errorf(
			"model package was removed but local-state cleanup is pending: %w",
			err,
		)
	}
	return nil
}

func (a *API) ensureManagedSource(
	ctx context.Context,
	rootID rootModel.RootID,
) (sourceModel.Summary, error) {
	if err := rootID.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if a.protection.IsProtectedRoot(rootID) {
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: managed Model Source is not allowed in a protected Root",
			spec.ErrProtected,
		)
	}

	draft := modelDomain.ManagedSourceDraft()
	summary, _, err := a.sources.Ensure(ctx, rootID, draft)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	if summary.ID != draft.ID ||
		summary.StorageKey != draft.StorageKey ||
		summary.Kind != managedfs.Kind {
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: Root %q has an incompatible managed Model Source",
			spec.ErrConflict,
			rootID,
		)
	}
	if summary.RetiredAt != nil {
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: managed Model Source is retired",
			spec.ErrRetired,
		)
	}
	return summary, nil
}

func (a *API) ensureManagedDeclarationDiscovery(
	ctx context.Context,
	rootID rootModel.RootID,
	locator spec.Locator,
	documentUse string,
) (sourceModel.Summary, error) {
	summary, err := a.ensureManagedSource(ctx, rootID)
	if err != nil {
		return sourceModel.Summary{}, err
	}

	required, err := topology.DiscoverySpecForLocatorForUse(
		documentUse,
		locator,
	)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	desired := source.MergeDiscoveryScopes(
		summary.Discovery,
		required,
	)
	if summary.Enabled && summary.Discovery.Equal(desired) {
		return summary, nil
	}

	updated, err := a.sources.Update(
		ctx,
		rootID,
		summary.ID,
		sourceModel.Update{
			ExpectedRevision: summary.Revision,
			DisplayName:      summary.DisplayName,
			Enabled:          true,
			Discovery:        &desired,
		},
	)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	return updated, nil
}

func (a *API) currentManagedSourceGeneration(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) (string, error) {
	if err := refreshFlow.EnsureSourceCurrent(
		ctx,
		a.discovery,
		rootID,
		sourceID,
	); err != nil {
		return "", err
	}

	inspection, err := a.discovery.InspectSource(
		ctx,
		rootID,
		sourceID,
	)
	if err != nil {
		return "", err
	}
	if !inspection.IsCurrent() {
		return "", fmt.Errorf(
			"%w: managed Model Source requires refresh",
			spec.ErrRefreshRequired,
		)
	}
	return inspection.State.SourceGeneration, nil
}

func (a *API) managedRecord(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	kind artifactModel.ArtifactKind,
) (artifactModel.Artifact, sourceModel.Summary, error) {
	record, err := a.requireKind(ctx, ref, kind)
	if err != nil {
		return artifactModel.Artifact{}, sourceModel.Summary{}, err
	}
	if a.protection.IsProtectedRoot(record.RootID) {
		return artifactModel.Artifact{}, sourceModel.Summary{}, fmt.Errorf(
			"%w: protected Model Artifacts cannot be mutated through managed authoring",
			spec.ErrProtected,
		)
	}
	if record.Binding.SourceID != modelDomain.ManagedSourceID {
		return artifactModel.Artifact{}, sourceModel.Summary{}, fmt.Errorf(
			"%w: Model Artifact is not owned by the managed Model Source",
			spec.ErrUnsupported,
		)
	}

	summary, err := a.sources.Get(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return artifactModel.Artifact{}, sourceModel.Summary{}, err
	}
	if summary.Kind != managedfs.Kind ||
		summary.StorageKey != modelDomain.ManagedSourceStorageKey {
		return artifactModel.Artifact{}, sourceModel.Summary{}, fmt.Errorf(
			"%w: Model Artifact is not backed by the expected managed Source",
			spec.ErrUnsupported,
		)
	}
	return record, summary, nil
}

func providerDefinition(
	document modelDomain.ProviderDocument,
) (definitionModel.Definition, []byte, error) {
	declarationDocument, err := document.ToDeclaration()
	if err != nil {
		return definitionModel.Definition{}, nil, err
	}
	raw, err := declarationDocument.CanonicalJSON()
	if err != nil {
		return definitionModel.Definition{}, nil, err
	}
	entry, err := declaration.NewEntry(declarationDocument)
	if err != nil {
		return definitionModel.Definition{}, nil, err
	}
	value, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return definitionModel.Definition{}, nil, err
	}
	return value, raw, nil
}

func modelDefinition(
	document modelDomain.ModelDocument,
) (definitionModel.Definition, []byte, error) {
	declarationDocument, err := document.ToDeclaration()
	if err != nil {
		return definitionModel.Definition{}, nil, err
	}
	raw, err := declarationDocument.CanonicalJSON()
	if err != nil {
		return definitionModel.Definition{}, nil, err
	}
	entry, err := declaration.NewEntry(declarationDocument)
	if err != nil {
		return definitionModel.Definition{}, nil, err
	}
	value, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return definitionModel.Definition{}, nil, err
	}
	return value, raw, nil
}
