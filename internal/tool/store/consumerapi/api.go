package consumerapi

import (
	"context"
	"fmt"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/composition/local/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	toolBuiltin "github.com/flexigpt/flexigpt-app/internal/tool/store/builtin"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

type API struct {
	discovery        compositionapi.DiscoveryAPI
	artifacts        compositionapi.ArtifactAPI
	managedArtifacts compositionapi.ManagedArtifactAPI
	protection       compositionapi.ProtectionAPI
	collections      *collection.API

	builtinRoot      root.RootID
	builtinSource    source.SourceID
	collectionByTool map[model.LogicalName]model.LogicalName
}

func New(
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
	builtinRoot root.RootID,
) (*API, error) {
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		managedArtifacts == nil ||
		protection == nil {
		return nil, fmt.Errorf(
			"%w: Tool Store dependencies are incomplete",
			model.ErrInvalid,
		)
	}
	if err := builtinRoot.Validate(); err != nil {
		return nil, err
	}
	if builtinRoot != documentTopology.BuiltinRootID() ||
		!protection.IsProtectedRoot(builtinRoot) {
		return nil, fmt.Errorf(
			"%w: Tool Store requires the protected built-in Root",
			model.ErrInvalid,
		)
	}

	builtinSource, err := documentTopology.BuiltinSource(
		documentTopology.BuiltinSourceRolePackages,
	)
	if err != nil {
		return nil, err
	}
	if builtinSource.Kind != source.SourceKindManagedDirectory {
		return nil, fmt.Errorf(
			"%w: built-in Tool Source must be managed",
			model.ErrInvalid,
		)
	}
	collectionByTool, err := toolBuiltin.GeneratedToolCollectionIndex()
	if err != nil {
		return nil, err
	}

	// Built-in Tool Collections have direct named references and no aliases.
	// A graph resolver is deliberately not installed here: the Tool target
	// mapper itself calls this Store to check Collection membership.
	collections, err := collection.NewWithResolver(
		sources,
		discovery,
		artifacts,
		managedArtifacts,
		nil,
		toolCollectionPolicy(),
	)
	if err != nil {
		return nil, err
	}

	return &API{
		discovery:        discovery,
		artifacts:        artifacts,
		managedArtifacts: managedArtifacts,
		protection:       protection,
		collections:      collections,
		builtinRoot:      builtinRoot,
		builtinSource:    builtinSource.ID,
		collectionByTool: collectionByTool,
	}, nil
}

func (a *API) GetTool(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ToolView, error) {
	value, err := a.getTool(ctx, ref)
	if err != nil {
		return ToolView{}, err
	}
	return toolView(value), nil
}

func (a *API) ListTools(
	ctx context.Context,
	collectionRef artifact.ArtifactRef,
) ([]ToolListItem, error) {
	view, err := a.GetToolCollection(ctx, collectionRef)
	if err != nil {
		return nil, err
	}

	return a.listCollectionTools(ctx, view)
}

func (a *API) SetToolEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (ToolView, error) {
	if err := a.ready(ctx); err != nil {
		return ToolView{}, err
	}
	if expectedRevision == 0 {
		return ToolView{}, fmt.Errorf(
			"%w: expected Tool revision is required",
			model.ErrInvalid,
		)
	}

	value, err := a.GetTool(ctx, ref)
	if err != nil {
		return ToolView{}, err
	}
	if value.Artifact.Revision != expectedRevision {
		return ToolView{}, model.ErrConflict
	}

	updated, err := a.artifacts.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
	if err != nil {
		return ToolView{}, err
	}
	value.Artifact = updated.Clone()
	return value, nil
}

func (a *API) getTool(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (toolDomain.Tool, error) {
	if err := a.ready(ctx); err != nil {
		return toolDomain.Tool{}, err
	}
	if err := a.requireBuiltinRef(ref); err != nil {
		return toolDomain.Tool{}, err
	}

	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return toolDomain.Tool{}, err
	}
	if err := a.requireBuiltinArtifact(record); err != nil {
		return toolDomain.Tool{}, err
	}

	definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return toolDomain.Tool{}, err
	}
	value, err := toolDomain.DecodeTool(record, definitionValue)
	if err != nil {
		return toolDomain.Tool{}, err
	}

	actual, err := toolDomain.ToolPackageAddressFromLocator(
		record.Binding.Locator,
	)
	if err != nil {
		return toolDomain.Tool{}, err
	}
	expected, err := toolDomain.ToolPackageAddress(
		record.LogicalName,
		value.Document.Version,
	)
	if err != nil {
		return toolDomain.Tool{}, err
	}
	if actual != expected {
		return toolDomain.Tool{}, fmt.Errorf(
			"%w: Tool package identity differs from its declaration",
			model.ErrReferenceUnresolved,
		)
	}
	return value, nil
}

func (a *API) ready(ctx context.Context) error {
	if a == nil {
		return model.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Tool Store context is nil",
			model.ErrInvalid,
		)
	}
	return ctx.Err()
}

func (a *API) requireBuiltinRef(ref artifact.ArtifactRef) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if ref.RootID != a.builtinRoot ||
		!a.protection.IsProtectedRoot(ref.RootID) {
		return fmt.Errorf(
			"%w: Tool Artifact is not in the protected built-in Root",
			model.ErrReferenceUnresolved,
		)
	}
	return nil
}

func (a *API) requireBuiltinArtifact(record artifact.Artifact) error {
	if err := a.requireBuiltinRef(record.Ref()); err != nil {
		return err
	}
	if record.Binding.SourceID != a.builtinSource ||
		record.Binding.SubresourceLocator != "" {
		return fmt.Errorf(
			"%w: Tool catalog Artifact has an unsupported origin",
			model.ErrReferenceUnresolved,
		)
	}
	return nil
}
