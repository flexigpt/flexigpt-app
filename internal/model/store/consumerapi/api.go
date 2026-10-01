package consumerapi

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelproviderv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/model/store/domain"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/model/store/overlay"
)

type API struct {
	sources          compositionapi.SourceAPI
	discovery        compositionapi.DiscoveryAPI
	artifacts        compositionapi.ArtifactAPI
	managedArtifacts compositionapi.ManagedArtifactAPI
	protection       compositionapi.ProtectionAPI

	overlays    modelOverlay.OverlayRepository
	adapters    AdapterRegistry
	builtinRoot root.RootID
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
		dependencies.Adapters == nil {
		return nil, fmt.Errorf(
			"%w: Model Store dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	if err := dependencies.BuiltinRoot.Validate(); err != nil {
		return nil, err
	}
	if dependencies.BuiltinRoot != documentTopology.BuiltinRootID() ||
		!dependencies.Protection.IsProtectedRoot(
			dependencies.BuiltinRoot,
		) {
		return nil, fmt.Errorf(
			"%w: Model Store requires the protected built-in Root",
			basespec.ErrInvalid,
		)
	}

	return &API{
		sources:          dependencies.Sources,
		discovery:        dependencies.Discovery,
		artifacts:        dependencies.Artifacts,
		managedArtifacts: dependencies.ManagedArtifacts,
		protection:       dependencies.Protection,
		overlays:         dependencies.Overlays,
		adapters:         dependencies.Adapters,
		builtinRoot:      dependencies.BuiltinRoot,
	}, nil
}

func (a *API) GetProvider(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ProviderView, error) {
	value, err := a.loadProvider(ctx, ref)
	if err != nil {
		return ProviderView{}, err
	}
	return a.providerView(ctx, value)
}

func (a *API) GetModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
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

	entries, err := a.artifacts.ListByRoot(
		ctx,
		request.RootID,
		catalog.ListOptions{
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

	entries, err := a.artifacts.ListByRoot(
		ctx,
		request.RootID,
		catalog.ListOptions{
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
			provider := document.Provider.Clone()
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
			value.Provider.Scope != request.Provider.Scope {
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
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if err := a.ready(ctx); err != nil {
		return artifact.Artifact{}, err
	}
	if err := validateExpectedArtifactRevision(expectedRevision); err != nil {
		return artifact.Artifact{}, err
	}
	if _, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelProviderArtifactKind,
	); err != nil {
		return artifact.Artifact{}, err
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
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if err := a.ready(ctx); err != nil {
		return artifact.Artifact{}, err
	}
	if err := validateExpectedArtifactRevision(expectedRevision); err != nil {
		return artifact.Artifact{}, err
	}
	if _, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelArtifactKind,
	); err != nil {
		return artifact.Artifact{}, err
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
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Model Store context is nil",
			basespec.ErrInvalid,
		)
	}
	return ctx.Err()
}

func (a *API) requireKind(
	ctx context.Context,
	ref artifact.ArtifactRef,
	kind artifact.ArtifactKind,
) (artifact.Artifact, error) {
	if err := ref.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if record.Kind != kind {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: Artifact %q has kind %q, expected %q",
			basespec.ErrUnsupported,
			record.ID,
			record.Kind,
			kind,
		)
	}
	return record, nil
}

func (a *API) loadProvider(
	ctx context.Context,
	ref artifact.ArtifactRef,
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
	ref artifact.ArtifactRef,
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
	document, err := value.Document.Clone()
	if err != nil {
		return ProviderView{}, err
	}
	settings, err := a.providerSettings(ctx, value.Artifact.Ref())
	if err != nil {
		return ProviderView{}, err
	}

	defaultModel := cloneOptionalReference(document.DefaultModel)
	if settings.DefaultModel != nil {
		defaultModel = cloneOptionalReference(settings.DefaultModel)
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
	document, err := value.Document.Clone()
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
