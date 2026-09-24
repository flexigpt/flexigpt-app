package consumerapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

type API struct {
	sources          compositionapi.SourceAPI
	discovery        compositionapi.DiscoveryAPI
	artifacts        compositionapi.ArtifactAPI
	managedArtifacts compositionapi.ManagedArtifactAPI
	protection       compositionapi.ProtectionAPI

	builtinRoot root.RootID
	goTools     toolDomain.GoToolLocator
}

func New(
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
	builtinRoot root.RootID,
	goTools toolDomain.GoToolLocator,
) (*API, error) {
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
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
	if !protection.IsProtectedRoot(builtinRoot) {
		return nil, fmt.Errorf(
			"%w: Tool Store builtin Root %q is not protected",
			basespec.ErrInvalid,
			builtinRoot,
		)
	}

	return &API{
		sources:          sources,
		discovery:        discovery,
		artifacts:        artifacts,
		managedArtifacts: managedArtifacts,
		protection:       protection,
		builtinRoot:      builtinRoot,
		goTools:          goTools,
	}, nil
}

func (a *API) ListToolCollections(
	ctx context.Context,
) ([]toolDomain.ToolCollection, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}

	records, err := a.artifacts.ListByRoot(ctx, a.builtinRoot)
	if err != nil {
		return nil, err
	}

	output := make([]toolDomain.ToolCollection, 0)
	for _, record := range records {
		if record.Kind != artifact.ArtifactKind(pluginv1.PluginType) ||
			record.State != artifact.StateAvailable {
			continue
		}

		definitionValue, err := a.artifacts.GetDefinition(
			ctx,
			record.Ref(),
		)
		if err != nil {
			return nil, err
		}
		collection, err := toolDomain.DecodeToolCollection(
			record,
			definitionValue,
		)
		if errors.Is(err, toolDomain.ErrNotToolCollection) {
			continue
		}
		if err != nil {
			return nil, err
		}
		output = append(output, collection)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Artifact.LogicalName !=
			output[right].Artifact.LogicalName {
			return output[left].Artifact.LogicalName <
				output[right].Artifact.LogicalName
		}
		return output[left].Artifact.ID < output[right].Artifact.ID
	})
	return output, nil
}

func (a *API) GetToolCollection(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (toolDomain.ToolCollection, error) {
	if a == nil {
		return toolDomain.ToolCollection{}, basespec.ErrClosed
	}
	if err := a.requireBuiltinRef(ref); err != nil {
		return toolDomain.ToolCollection{}, err
	}

	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return toolDomain.ToolCollection{}, err
	}
	definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return toolDomain.ToolCollection{}, err
	}
	return toolDomain.DecodeToolCollection(record, definitionValue)
}

func (a *API) ListTools(
	ctx context.Context,
	collectionRef artifact.ArtifactRef,
) ([]toolDomain.Tool, error) {
	collection, err := a.GetToolCollection(ctx, collectionRef)
	if err != nil {
		return nil, err
	}

	output := make([]toolDomain.Tool, 0, len(collection.ToolNames))
	for _, name := range collection.ToolNames {
		tool, err := a.toolByName(ctx, name)
		if err != nil {
			return nil, fmt.Errorf(
				"resolve Tool Collection member %q: %w",
				name,
				err,
			)
		}
		output = append(output, tool)
	}
	return output, nil
}

func (a *API) GetTool(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (toolDomain.Tool, error) {
	if a == nil {
		return toolDomain.Tool{}, basespec.ErrClosed
	}
	if err := a.requireBuiltinRef(ref); err != nil {
		return toolDomain.Tool{}, err
	}

	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return toolDomain.Tool{}, err
	}
	definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return toolDomain.Tool{}, err
	}
	tool, err := toolDomain.DecodeTool(record, definitionValue)
	if err != nil {
		return toolDomain.Tool{}, err
	}
	return tool, a.validateGoTool(ctx, tool)
}

func (a *API) ResolveEnabledTool(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (toolDomain.ResolvedTool, error) {
	tool, err := a.GetTool(ctx, ref)
	if err != nil {
		return toolDomain.ResolvedTool{}, err
	}
	return a.resolveEnabledTool(ctx, tool)
}

func (a *API) SetToolEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (toolDomain.Tool, error) {
	if expectedRevision == 0 {
		return toolDomain.Tool{}, fmt.Errorf(
			"%w: expected Tool revision is required",
			basespec.ErrInvalid,
		)
	}

	tool, err := a.GetTool(ctx, ref)
	if err != nil {
		return toolDomain.Tool{}, err
	}
	if tool.Artifact.Revision != expectedRevision {
		return toolDomain.Tool{}, basespec.ErrConflict
	}

	updated, err := a.artifacts.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
	if err != nil {
		return toolDomain.Tool{}, err
	}
	tool.Artifact = updated.Clone()
	return tool, nil
}

func (a *API) SetToolCollectionEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (toolDomain.ToolCollection, error) {
	if expectedRevision == 0 {
		return toolDomain.ToolCollection{}, fmt.Errorf(
			"%w: expected Tool Collection revision is required",
			basespec.ErrInvalid,
		)
	}

	collection, err := a.GetToolCollection(ctx, ref)
	if err != nil {
		return toolDomain.ToolCollection{}, err
	}
	if collection.Artifact.Revision != expectedRevision {
		return toolDomain.ToolCollection{}, basespec.ErrConflict
	}

	updated, err := a.artifacts.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
	if err != nil {
		return toolDomain.ToolCollection{}, err
	}
	collection.Artifact = updated.Clone()
	return collection, nil
}

func (a *API) resolveEnabledTool(
	ctx context.Context,
	tool toolDomain.Tool,
) (toolDomain.ResolvedTool, error) {
	collection, err := a.collectionForTool(
		ctx,
		tool.Artifact.LogicalName,
	)
	if err != nil {
		return toolDomain.ResolvedTool{}, err
	}

	resolved := toolDomain.ResolvedTool{
		Tool:       tool,
		Collection: collection,
	}
	if !resolved.Enabled() {
		return toolDomain.ResolvedTool{}, fmt.Errorf(
			"%w: Tool %q is disabled",
			basespec.ErrReferenceUnresolved,
			tool.Artifact.LogicalName,
		)
	}
	return resolved, nil
}

func (a *API) toolByName(
	ctx context.Context,
	name basespec.LogicalName,
) (toolDomain.Tool, error) {
	records, err := a.artifacts.FindByIdentity(
		ctx,
		a.builtinRoot,
		toolDomain.ToolArtifactKind,
		name,
	)
	if err != nil {
		return toolDomain.Tool{}, err
	}

	candidates := make([]toolDomain.Tool, 0, len(records))
	for _, record := range records {
		if record.State != artifact.StateAvailable {
			continue
		}
		definitionValue, err := a.artifacts.GetDefinition(
			ctx,
			record.Ref(),
		)
		if err != nil {
			return toolDomain.Tool{}, err
		}
		tool, err := toolDomain.DecodeTool(record, definitionValue)
		if err != nil {
			return toolDomain.Tool{}, err
		}
		if err := a.validateGoTool(ctx, tool); err != nil {
			return toolDomain.Tool{}, err
		}
		candidates = append(candidates, tool)
	}

	switch len(candidates) {
	case 0:
		return toolDomain.Tool{}, fmt.Errorf(
			"%w: built-in Tool %q is unavailable",
			basespec.ErrReferenceUnresolved,
			name,
		)
	case 1:
		return candidates[0], nil
	default:
		return toolDomain.Tool{}, fmt.Errorf(
			"%w: built-in Tool %q has %d matching Artifacts",
			basespec.ErrIdentityConflict,
			name,
			len(candidates),
		)
	}
}

func (a *API) collectionForTool(
	ctx context.Context,
	name basespec.LogicalName,
) (toolDomain.ToolCollection, error) {
	collections, err := a.ListToolCollections(ctx)
	if err != nil {
		return toolDomain.ToolCollection{}, err
	}

	matches := make([]toolDomain.ToolCollection, 0, 1)
	for _, collection := range collections {
		if collection.HasTool(name) {
			matches = append(matches, collection)
		}
	}

	switch len(matches) {
	case 0:
		return toolDomain.ToolCollection{}, fmt.Errorf(
			"%w: Tool %q has no Tool Collection",
			basespec.ErrReferenceUnresolved,
			name,
		)
	case 1:
		return matches[0], nil
	default:
		return toolDomain.ToolCollection{}, fmt.Errorf(
			"%w: Tool %q belongs to %d Tool Collections",
			basespec.ErrIdentityConflict,
			name,
			len(matches),
		)
	}
}

func (a *API) requireBuiltinRef(
	ref artifact.ArtifactRef,
) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if ref.RootID != a.builtinRoot ||
		!a.protection.IsProtectedRoot(ref.RootID) {
		return fmt.Errorf(
			"%w: Tool Artifact is not in the built-in Tool Root",
			basespec.ErrReferenceUnresolved,
		)
	}
	return nil
}

func (a *API) requireBuiltinSource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) error {
	if rootID != a.builtinRoot ||
		!a.protection.IsProtectedRoot(rootID) {
		return fmt.Errorf(
			"%w: Tool package Root is not the protected built-in Root",
			basespec.ErrProtected,
		)
	}
	_, err := a.sources.Get(ctx, rootID, sourceID)
	return err
}

func (a *API) validateGoTool(
	ctx context.Context,
	tool toolDomain.Tool,
) error {
	if tool.Document.Implementation.Kind != toolv1.ImplementationKindGo {
		return nil
	}
	if a.goTools == nil {
		return basespec.ErrClosed
	}

	registered, err := a.goTools.LookupGoTool(
		ctx,
		tool.Document.Implementation.Function,
	)
	if err != nil {
		return err
	}
	if registered.Name != tool.Artifact.LogicalName ||
		registered.Version != tool.Document.Version ||
		registered.Function != tool.Document.Implementation.Function {
		return fmt.Errorf(
			"%w: Go Tool declaration does not match registered Go Tool",
			basespec.ErrReferenceUnresolved,
		)
	}

	declaredSchema, err := jsonutil.Canonicalize(
		tool.Document.InputSchema,
	)
	if err != nil {
		return err
	}
	registeredSchema, err := jsonutil.Canonicalize(
		registered.InputSchema,
	)
	if err != nil {
		return err
	}
	if !bytes.Equal(declaredSchema, registeredSchema) {
		return fmt.Errorf(
			"%w: Go Tool input schema differs from registered Go Tool",
			basespec.ErrDigestMismatch,
		)
	}
	return nil
}
