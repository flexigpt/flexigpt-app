package tool

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/domain"
)

type Service struct {
	protection       root.ProtectionAPI
	artifacts        artifact.API
	managedArtifacts managepackageFlow.API
	discovery        refreshFlow.API
	plugins          *pluginAPI.API
	resolver         *composition.Resolver
	cat              catalog.API
	definitions      definition.API
	builtin          toolDomain.BuiltinCatalog
}

func New(
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	managedArtifacts managepackageFlow.API,
	protection root.ProtectionAPI,
	cat catalog.API,
	definitions definition.API,
	builtin toolDomain.BuiltinCatalog,
	resolver *composition.Resolver,
) (*Service, error) {
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		managedArtifacts == nil ||
		protection == nil || cat == nil || definitions == nil {
		return nil, fmt.Errorf(
			"%w: Tool Store dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	if resolver == nil {
		return nil, fmt.Errorf("%w: Tool composition resolver is nil", spec.ErrInvalid)
	}
	if err := builtin.Validate(); err != nil {
		return nil, err
	}
	if !protection.IsProtectedRoot(builtin.RootID) {
		return nil, fmt.Errorf(
			"%w: Tool Store built-in Root must be protected",
			spec.ErrInvalid,
		)
	}

	ownedBuiltin := builtin.Clone()

	plugins, err := pluginAPI.New(
		artifacts,
		cat,
		sources,
		discovery,
		managedArtifacts,
		definitions,
		resolver,
		ownedBuiltin.PluginProfile,
	)
	if err != nil {
		return nil, err
	}

	return &Service{
		protection:       protection,
		artifacts:        artifacts,
		managedArtifacts: managedArtifacts,
		plugins:          plugins,
		discovery:        discovery,
		resolver:         resolver,
		cat:              cat,
		definitions:      definitions,
		builtin:          ownedBuiltin,
	}, nil
}

func (a *Service) GetTool(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ToolView, error) {
	value, err := a.getTool(ctx, ref)
	if err != nil {
		return ToolView{}, err
	}
	return toolView(value), nil
}

func (a *Service) ListTools(
	ctx context.Context,
	pluginRef artifactModel.ArtifactRef,
) ([]ToolListItem, error) {
	view, err := a.GetToolPlugin(ctx, pluginRef)
	if err != nil {
		return nil, err
	}

	return a.listPluginTools(ctx, view)
}

func (a *Service) SetToolEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (ToolView, error) {
	if expectedRevision == 0 {
		return ToolView{}, fmt.Errorf(
			"%w: expected Tool revision is required",
			spec.ErrInvalid,
		)
	}

	value, err := a.GetTool(ctx, ref)
	if err != nil {
		return ToolView{}, err
	}
	if value.Artifact.Revision != expectedRevision {
		return ToolView{}, spec.ErrConflict
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

// MapToolTarget resolves one enabled Tool into the same source-backed
// CapabilityTarget used by Agent, Plugin, Skill, and Workspace composition.
//
// The Tool family remains Artifact-backed. A direct capability target is not
// actionable through Tool Store because it has no Tool Artifact or Plugin
// state to inspect.
func (a *Service) MapToolTarget(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (composition.CapabilityTarget, error) {
	value, err := a.ResolveEnabledTool(ctx, ref)
	if err != nil {
		return composition.CapabilityTarget{}, err
	}
	if a == nil || a.resolver == nil {
		return composition.CapabilityTarget{}, spec.ErrClosed
	}

	resolved, err := a.resolver.ResolveTool(ctx, ref)
	if err != nil {
		return composition.CapabilityTarget{}, err
	}
	target, found := resolved.TargetValue()
	if !found {
		return composition.CapabilityTarget{}, fmt.Errorf(
			"%w: Tool %q resolved without a capability target",
			spec.ErrReferenceUnresolved,
			value.Tool.Name,
		)
	}
	if err := target.Validate(); err != nil {
		return composition.CapabilityTarget{}, err
	}
	if target.Type != declaration.TypeTool ||
		target.Name != value.Tool.Name {
		return composition.CapabilityTarget{}, fmt.Errorf(
			"%w: Tool capability target identity differs from Tool Artifact",
			spec.ErrDigestMismatch,
		)
	}
	if target.Form != composition.TargetFormArtifact ||
		target.Artifact == nil ||
		*target.Artifact != value.Tool.Artifact.Ref() {
		return composition.CapabilityTarget{}, fmt.Errorf(
			"%w: Tool Store requires an Artifact-backed Tool target",
			spec.ErrUnsupported,
		)
	}
	return target.Clone(), nil
}

// ResolveToolTarget resolves an Artifact-backed Tool CapabilityTarget into
// the Tool view plus the owning Tool Plugin. It verifies that the target still
// identifies the current enabled Tool Artifact before returning it.
//
// Direct Tool capabilities deliberately remain owned by their runtime
// provider. Tool Store must not fabricate an Artifact-backed Tool projection
// for them.
func (a *Service) ResolveToolTarget(
	ctx context.Context,
	target composition.CapabilityTarget,
) (ResolvedToolView, error) {
	if err := target.Validate(); err != nil {
		return ResolvedToolView{}, err
	}
	if target.Type != declaration.TypeTool {
		return ResolvedToolView{}, fmt.Errorf(
			"%w: capability target type is %q, expected tool",
			spec.ErrUnsupported,
			target.Type,
		)
	}
	if target.Form != composition.TargetFormArtifact ||
		target.Artifact == nil {
		return ResolvedToolView{}, fmt.Errorf(
			"%w: direct Tool capability targets require their owning runtime adapter",
			spec.ErrUnsupported,
		)
	}

	value, err := a.ResolveEnabledTool(ctx, *target.Artifact)
	if err != nil {
		return ResolvedToolView{}, err
	}
	if value.Tool.Artifact.Ref() != *target.Artifact ||
		value.Tool.Name != target.Name {
		return ResolvedToolView{}, fmt.Errorf(
			"%w: Tool capability target no longer matches the Tool Artifact",
			spec.ErrReferenceUnresolved,
		)
	}
	return value, nil
}

func (a *Service) getTool(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (toolDomain.Tool, error) {
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

	actual, err := a.builtin.ToolPackage.AddressFromLocator(
		record.Binding.Locator,
	)
	if err != nil {
		return toolDomain.Tool{}, err
	}
	expected, err := a.builtin.ToolPackage.Address(
		record.LogicalName,
		value.Document.Version,
	)
	if err != nil {
		return toolDomain.Tool{}, err
	}
	if actual != expected {
		return toolDomain.Tool{}, fmt.Errorf(
			"%w: Tool package identity differs from its declaration",
			spec.ErrReferenceUnresolved,
		)
	}
	return value, nil
}

func (a *Service) requireBuiltinRef(ref artifactModel.ArtifactRef) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if ref.RootID != a.builtin.RootID ||
		!a.protection.IsProtectedRoot(ref.RootID) {
		return fmt.Errorf(
			"%w: Tool Artifact is not in the protected built-in Root",
			spec.ErrReferenceUnresolved,
		)
	}
	return nil
}

func (a *Service) requireBuiltinArtifact(record artifactModel.Artifact) error {
	if err := a.requireBuiltinRef(record.Ref()); err != nil {
		return err
	}
	if record.Binding.SourceID != a.builtin.SourceID ||
		record.Binding.SubresourceLocator != "" {
		return fmt.Errorf(
			"%w: Tool catalog Artifact has an unsupported origin",
			spec.ErrReferenceUnresolved,
		)
	}
	return nil
}
