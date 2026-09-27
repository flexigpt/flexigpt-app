package consumerapi

import (
	"bytes"
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

type API struct {
	sources          compositionapi.SourceAPI
	discovery        compositionapi.DiscoveryAPI
	artifacts        compositionapi.ArtifactAPI
	resources        compositionapi.ResourceAPI
	managedArtifacts compositionapi.ManagedArtifactAPI
	protection       compositionapi.ProtectionAPI
	collections      *collection.API

	builtinRoot   root.RootID
	builtinSource source.SourceID
	goTools       toolDomain.GoToolLocator
}

func New(
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
	builtinRoot root.RootID,
	goTools toolDomain.GoToolLocator,
) (*API, error) {
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		resources == nil ||
		managedArtifacts == nil ||
		protection == nil ||
		goTools == nil {
		return nil, fmt.Errorf(
			"%w: Tool Store dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	if err := builtinRoot.Validate(); err != nil {
		return nil, err
	}
	if builtinRoot != documentTopology.BuiltinRootID() ||
		!protection.IsProtectedRoot(builtinRoot) {
		return nil, fmt.Errorf(
			"%w: Tool Store requires the protected built-in Root",
			basespec.ErrInvalid,
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
			basespec.ErrInvalid,
		)
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
		sources:          sources,
		discovery:        discovery,
		artifacts:        artifacts,
		resources:        resources,
		managedArtifacts: managedArtifacts,
		protection:       protection,
		collections:      collections,
		builtinRoot:      builtinRoot,
		builtinSource:    builtinSource.ID,
		goTools:          goTools,
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
			basespec.ErrInvalid,
		)
	}

	value, err := a.GetTool(ctx, ref)
	if err != nil {
		return ToolView{}, err
	}
	if value.Artifact.Revision != expectedRevision {
		return ToolView{}, basespec.ErrConflict
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
			basespec.ErrReferenceUnresolved,
		)
	}

	if err := a.validateGoTool(ctx, value); err != nil {
		return toolDomain.Tool{}, err
	}
	return value, nil
}

func (a *API) validateGoTool(
	ctx context.Context,
	value toolDomain.Tool,
) error {
	if value.Document.Implementation.Kind != toolv1.ImplementationKindGo {
		return nil
	}

	registered, err := a.goTools.LookupGoTool(
		ctx,
		value.Document.Implementation.Function,
	)
	if err != nil {
		return err
	}
	if registered.Name != value.Artifact.LogicalName ||
		registered.Version != value.Document.Version ||
		registered.Function != value.Document.Implementation.Function {
		return fmt.Errorf(
			"%w: Go Tool declaration does not match the registered Go Tool",
			basespec.ErrReferenceUnresolved,
		)
	}

	declaredSchema, err := jsonutil.Canonicalize(value.Document.InputSchema)
	if err != nil {
		return err
	}
	registeredSchema, err := jsonutil.Canonicalize(registered.InputSchema)
	if err != nil {
		return err
	}
	if !bytes.Equal(declaredSchema, registeredSchema) {
		return fmt.Errorf(
			"%w: Go Tool input schema differs from the registered Go Tool",
			basespec.ErrDigestMismatch,
		)
	}
	return nil
}

func (a *API) ready(ctx context.Context) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Tool Store context is nil",
			basespec.ErrInvalid,
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
			basespec.ErrReferenceUnresolved,
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
			basespec.ErrReferenceUnresolved,
		)
	}
	return nil
}
