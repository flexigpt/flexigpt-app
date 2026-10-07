package model

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	modelv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/contract/v1"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/domain"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/overlay"
	modelproviderv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/modelprovider/contract/v1"
)

type Service struct {
	protection       root.ProtectionAPI
	artifacts        artifact.API
	cat              catalog.API
	definitions      definition.API
	sources          source.API
	discovery        refreshFlow.API
	managedArtifacts managepackageFlow.API

	overlays    modelOverlay.OverlayRepository
	adapters    AdapterRegistry
	builtinRoot rootModel.RootID
	support     Support
}

func New(
	dependencies Dependencies,
) (*Service, error) {
	if dependencies.Sources == nil ||
		dependencies.Discovery == nil ||
		dependencies.Artifacts == nil ||
		dependencies.ManagedArtifacts == nil ||
		dependencies.Protection == nil ||
		dependencies.Overlays == nil ||
		dependencies.Adapters == nil ||
		dependencies.Cat == nil ||
		dependencies.Definitions == nil {
		return nil, fmt.Errorf(
			"%w: Model Store dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	if err := dependencies.Support.Validate(); err != nil {
		return nil, err
	}
	if !dependencies.Protection.IsProtectedRoot(
		dependencies.Support.BuiltinRoot,
	) {
		return nil, fmt.Errorf(
			"%w: Model Store built-in Root must be protected",
			spec.ErrInvalid,
		)
	}

	output := &Service{
		protection:       dependencies.Protection,
		artifacts:        dependencies.Artifacts,
		cat:              dependencies.Cat,
		definitions:      dependencies.Definitions,
		sources:          dependencies.Sources,
		discovery:        dependencies.Discovery,
		managedArtifacts: dependencies.ManagedArtifacts,
		overlays:         dependencies.Overlays,
		adapters:         dependencies.Adapters,
		builtinRoot:      dependencies.Support.BuiltinRoot,
		support:          dependencies.Support,
	}
	return output, nil
}

func (a *Service) GetProvider(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ProviderView, error) {
	value, err := a.loadProvider(ctx, ref)
	if err != nil {
		return ProviderView{}, err
	}
	return a.providerView(ctx, value)
}

func (a *Service) GetModel(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ModelView, error) {
	value, err := a.loadModel(ctx, ref)
	if err != nil {
		return ModelView{}, err
	}
	return a.modelView(ctx, value)
}

func (a *Service) ListProviders(
	ctx context.Context,
	request ListProvidersRequest,
) ([]ProviderListItem, error) {
	entries, err := a.cat.ListByRoot(
		ctx,
		request.RootID,
		catalogModel.ListOptions{
			Kind:            modelDomain.ModelProviderArtifactKind,
			Enabled:         request.Enabled,
			IncludeDocument: true,
		},
	)
	if err != nil {
		return nil, err
	}

	output := make([]ProviderListItem, 0, len(entries))
	for _, entry := range entries {
		item := ProviderListItem{
			Ref:         entry.Ref(),
			Name:        entry.LogicalName,
			DisplayName: entry.DisplayName,
			State:       entry.State,
			Enabled:     entry.Enabled,
			Revision:    entry.Revision,
			BuiltIn:     a.protection.IsProtectedRoot(entry.RootID),
		}

		if entry.Definition != nil {
			item.Description = entry.Definition.Description
		}
		if entry.Document != nil {
			document, err := modelproviderv1.DecodeModelProviderJSON(
				entry.Document.Body,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"decode listed Model Provider %q: %w",
					entry.ID,
					err,
				)
			}
			item.Adapter = document.Adapter

		}
		output = append(output, item)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Ref.ArtifactID <
			output[right].Ref.ArtifactID
	})
	return output, nil
}

func (a *Service) ListModels(
	ctx context.Context,
	request ListModelsRequest,
) ([]ModelListItem, error) {
	entries, err := a.cat.ListByRoot(
		ctx,
		request.RootID,
		catalogModel.ListOptions{
			Kind:            modelDomain.ModelArtifactKind,
			Enabled:         request.Enabled,
			IncludeDocument: true,
		},
	)
	if err != nil {
		return nil, err
	}

	output := make([]ModelListItem, 0, len(entries))
	for _, entry := range entries {
		item := ModelListItem{
			Ref:         entry.Ref(),
			Name:        entry.LogicalName,
			DisplayName: entry.DisplayName,
			State:       entry.State,
			Enabled:     entry.Enabled,
			Revision:    entry.Revision,
			BuiltIn:     a.protection.IsProtectedRoot(entry.RootID),
		}
		if entry.Definition != nil {
			item.Description = entry.Definition.Description
		}
		if entry.Document != nil {
			document, err := modelv1.DecodeModelJSON(entry.Document.Body)
			if err != nil {
				return nil, fmt.Errorf(
					"decode listed Model %q: %w",
					entry.ID,
					err,
				)
			}
			provider := modelDomain.ArtifactNameReferenceFromDeclaration(
				document.Provider,
			)
			item.Provider = &provider
			item.ProviderModelID = document.ProviderModelID
		}
		output = append(output, item)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Ref.ArtifactID <
			output[right].Ref.ArtifactID
	})
	return output, nil
}

func (a *Service) ListModelsByProvider(
	ctx context.Context,
	request ListModelsByProviderRequest,
) ([]ModelListItem, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	values, err := a.ListModels(ctx, ListModelsRequest{
		RootID:  request.RootID,
		Enabled: request.Enabled,
	})
	if err != nil {
		return nil, err
	}

	output := make([]ModelListItem, 0, len(values))
	for _, value := range values {
		if value.Provider == nil ||
			value.Provider.Name != request.Provider.Name ||
			string(value.Provider.Scope) != string(request.Provider.Scope) {
			continue
		}
		output = append(output, value)
	}
	return output, nil
}

// SetProviderEnabled wraps universal Artifact enablement. It is valid for
// protected built-ins and mutable user-owned Provider Artifacts alike.
func (a *Service) SetProviderEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifactModel.Artifact, error) {
	if err := validateExpectedArtifactRevision(expectedRevision); err != nil {
		return artifactModel.Artifact{}, err
	}
	if _, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelProviderArtifactKind,
	); err != nil {
		return artifactModel.Artifact{}, err
	}
	return a.artifacts.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
}

// SetModelEnabled wraps universal Artifact enablement. It is independent from
// Provider enablement and never changes any linked Provider state.
func (a *Service) SetModelEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifactModel.Artifact, error) {
	if err := validateExpectedArtifactRevision(expectedRevision); err != nil {
		return artifactModel.Artifact{}, err
	}
	if _, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelArtifactKind,
	); err != nil {
		return artifactModel.Artifact{}, err
	}
	return a.artifacts.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
}

func (a *Service) requireKind(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	kind artifactModel.ArtifactKind,
) (artifactModel.Artifact, error) {
	if err := ref.Validate(); err != nil {
		return artifactModel.Artifact{}, err
	}
	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if record.Kind != kind {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: Artifact %q has kind %q, expected %q",
			spec.ErrUnsupported,
			record.ID,
			record.Kind,
			kind,
		)
	}
	return record, nil
}

func (a *Service) loadProvider(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (modelDomain.Provider, error) {
	record, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelProviderArtifactKind,
	)
	if err != nil {
		return modelDomain.Provider{}, err
	}
	definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return modelDomain.Provider{}, err
	}
	return modelDomain.DecodeProvider(record, definitionValue)
}

func (a *Service) loadModel(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (modelDomain.Model, error) {
	record, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelArtifactKind,
	)
	if err != nil {
		return modelDomain.Model{}, err
	}
	definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return modelDomain.Model{}, err
	}
	return modelDomain.DecodeModel(record, definitionValue)
}

func (a *Service) providerView(
	ctx context.Context,
	value modelDomain.Provider,
) (ProviderView, error) {
	document, err := modelDomain.ProviderDocumentFromDeclaration(
		value.Document,
	)
	if err != nil {
		return ProviderView{}, err
	}
	settings, err := a.providerSettings(ctx, value.Artifact.Ref())
	if err != nil {
		return ProviderView{}, err
	}

	defaultModel := cloneOptionalModelReference(document.DefaultModel)
	if settings.DefaultModel != nil {
		defaultModel = cloneOptionalModelReference(settings.DefaultModel)
	}

	return ProviderView{
		Artifact:         value.Artifact.Clone(),
		DefinitionDigest: value.Definition.Digest,
		Document:         document,
		Settings:         settings,
		DefaultModel:     defaultModel,
		BuiltIn: a.protection.IsProtectedRoot(
			value.Artifact.RootID,
		),
	}, nil
}

func (a *Service) modelView(
	ctx context.Context,
	value modelDomain.Model,
) (ModelView, error) {
	document, err := modelDomain.ModelDocumentFromDeclaration(
		value.Document,
	)
	if err != nil {
		return ModelView{}, err
	}
	settings, err := a.modelSettings(ctx, value.Artifact.Ref())
	if err != nil {
		return ModelView{}, err
	}

	return ModelView{
		Artifact:         value.Artifact.Clone(),
		DefinitionDigest: value.Definition.Digest,
		Document:         document,
		Settings:         settings,
		BuiltIn: a.protection.IsProtectedRoot(
			value.Artifact.RootID,
		),
	}, nil
}
