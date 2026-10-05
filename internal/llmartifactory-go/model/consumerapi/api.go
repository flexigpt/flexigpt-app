package consumerapi

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
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/modelproviderv1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/modelv1"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/domain"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/overlay"
)

type API struct {
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
}

func New(
	dependencies Dependencies,
) (*API, error) {
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
	if err := dependencies.BuiltinRoot.Validate(); err != nil {
		return nil, err
	}
	if dependencies.BuiltinRoot != topology.BuiltinRootID() ||
		!dependencies.Protection.IsProtectedRoot(
			dependencies.BuiltinRoot,
		) {
		return nil, fmt.Errorf(
			"%w: Model Store requires the protected built-in Root",
			spec.ErrInvalid,
		)
	}

	return &API{
		protection:       dependencies.Protection,
		artifacts:        dependencies.Artifacts,
		cat:              dependencies.Cat,
		definitions:      dependencies.Definitions,
		sources:          dependencies.Sources,
		discovery:        dependencies.Discovery,
		managedArtifacts: dependencies.ManagedArtifacts,
		overlays:         dependencies.Overlays,
		adapters:         dependencies.Adapters,
		builtinRoot:      dependencies.BuiltinRoot,
	}, nil
}

func (a *API) GetProvider(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ProviderView, error) {
	value, err := a.loadProvider(ctx, ref)
	if err != nil {
		return ProviderView{}, err
	}
	return a.providerView(ctx, value)
}

func (a *API) GetModel(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ModelView, error) {
	value, err := a.loadModel(ctx, ref)
	if err != nil {
		return ModelView{}, err
	}
	return a.modelView(ctx, value)
}

func (a *API) ListProviders(
	ctx context.Context,
	request ListProvidersRequest,
) ([]ProviderListItem, error) {
	if err := a.ready(ctx); err != nil {
		return nil, err
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}

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
		if entry.Kind != modelDomain.ModelProviderArtifactKind {
			continue
		}

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

func (a *API) ListModels(
	ctx context.Context,
	request ListModelsRequest,
) ([]ModelListItem, error) {
	if err := a.ready(ctx); err != nil {
		return nil, err
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}

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
		if entry.Kind != modelDomain.ModelArtifactKind {
			continue
		}

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

func (a *API) ListModelsByProvider(
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
func (a *API) SetProviderEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifactModel.Artifact, error) {
	if err := a.ready(ctx); err != nil {
		return artifactModel.Artifact{}, err
	}
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
func (a *API) SetModelEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifactModel.Artifact, error) {
	if err := a.ready(ctx); err != nil {
		return artifactModel.Artifact{}, err
	}
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

func (a *API) ready(ctx context.Context) error {
	if a == nil ||
		a.sources == nil ||
		a.discovery == nil ||
		a.artifacts == nil ||
		a.managedArtifacts == nil ||
		a.protection == nil ||
		a.overlays == nil ||
		a.adapters == nil ||
		a.builtinRoot == "" {
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Model Store context is nil",
			spec.ErrInvalid,
		)
	}
	return ctx.Err()
}

func (a *API) requireKind(
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

func (a *API) loadProvider(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (modelDomain.Provider, error) {
	if err := a.ready(ctx); err != nil {
		return modelDomain.Provider{}, err
	}

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

func (a *API) loadModel(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (modelDomain.Model, error) {
	if err := a.ready(ctx); err != nil {
		return modelDomain.Model{}, err
	}

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

func (a *API) providerView(
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

func (a *API) modelView(
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
