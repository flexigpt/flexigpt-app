package resolve

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/codec"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/textv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
)

type resolutionState struct {
	nodes  int
	active map[artifact.ArtifactRef]struct{}
}

func newResolutionState() resolutionState {
	return resolutionState{
		active: make(map[artifact.ArtifactRef]struct{}),
	}
}

type loadedDeclarationArtifact struct {
	record          artifact.Artifact
	definition      definition.Definition
	entry           declaration.Entry
	declarationType declaration.Type
}

func (r *Resolver) ResolveText(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeText)
}

func (r *Resolver) ResolveModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeModel)
}

func (r *Resolver) ResolveTool(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeTool)
}

func (r *Resolver) ResolveSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeSkill)
}

func (r *Resolver) ResolveMCP(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeMCP)
}

func (r *Resolver) ResolveMCPPolicy(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeMCPPolicy)
}

func (r *Resolver) ResolvePlugin(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypePlugin)
}

func (r *Resolver) ResolveAgent(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeAgent)
}

func (r *Resolver) ResolveTeam(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeTeam)
}

func (r *Resolver) ResolveLoop(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeLoop)
}

func (r *Resolver) ResolveWorkflow(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeWorkflow)
}

func (r *Resolver) ResolveWorkspace(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedWorkspace, error) {
	value, err := r.ResolveWorkspaceEntry(ctx, ref)
	if err != nil {
		return nil, err
	}
	return value.Workspace, nil
}

func (r *Resolver) ResolveWorkspaceEntry(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	value, err := r.resolveTyped(ctx, ref, declaration.TypeWorkspace)
	if err != nil {
		return nil, err
	}
	if value.Workspace == nil {
		return nil, fmt.Errorf(
			"%w: Workspace resolver produced no Workspace projection",
			basespec.ErrReferenceUnresolved,
		)
	}
	return value, nil
}

// ResolveTerminalArtifact is a narrow source-alias helper. It follows only
// source-selected MCP and MCP Policy aliases. It does not resolve a generic
// type/name request and does not expand composition.
func (r *Resolver) ResolveTerminalArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.ArtifactRef, error) {
	if r == nil || r.artifacts == nil {
		return artifact.ArtifactRef{}, basespec.ErrClosed
	}
	if err := validateResolutionContext(ctx); err != nil {
		return artifact.ArtifactRef{}, err
	}
	if err := ref.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}
	return r.resolveTerminalArtifact(ctx, ref, "", "")
}

func (r *Resolver) resolveTyped(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expected declaration.Type,
) (*ResolvedEntry, error) {
	if r == nil || r.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	if err := validateResolutionContext(ctx); err != nil {
		return nil, err
	}
	if err := ref.Validate(); err != nil {
		return nil, err
	}

	state := newResolutionState()
	return r.resolveArtifact(ctx, &state, ref, expected, "", 0)
}

func (r *Resolver) resolveArtifact(
	ctx context.Context,
	state *resolutionState,
	ref artifact.ArtifactRef,
	expectedType declaration.Type,
	expectedVersion basespec.LogicalVersion,
	depth int,
) (*ResolvedEntry, error) {
	if err := r.reserve(state, depth); err != nil {
		return nil, err
	}

	terminal, err := r.resolveTerminalArtifact(
		ctx,
		ref,
		expectedType,
		expectedVersion,
	)
	if err != nil {
		return nil, err
	}
	if terminal != ref {
		return r.resolveArtifact(
			ctx,
			state,
			terminal,
			expectedType,
			expectedVersion,
			depth+1,
		)
	}

	if _, active := state.active[ref]; active {
		return nil, fmt.Errorf(
			"%w: declaration composition cycle at Artifact %q",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	state.active[ref] = struct{}{}
	defer delete(state.active, ref)

	loaded, err := r.loadAvailableDeclarationArtifact(
		ctx,
		ref,
		expectedType,
		expectedVersion,
	)
	if err != nil {
		return nil, err
	}

	node := &ResolvedEntry{
		Type:              loaded.declarationType,
		scopeRootID:       loaded.record.RootID,
		DeclarationOrigin: pointerArtifact(loaded.record),
		Artifact:          pointerArtifact(loaded.record),
		Definition:        pointerDefinition(loaded.definition),
	}
	if err := r.resolveStructure(
		ctx,
		state,
		node,
		loaded.entry,
		depth,
	); err != nil {
		return nil, err
	}
	return node, nil
}

func (r *Resolver) loadAvailableDeclarationArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedType declaration.Type,
	expectedVersion basespec.LogicalVersion,
) (loadedDeclarationArtifact, error) {
	record, err := r.artifacts.Get(ctx, ref)
	if err != nil {
		return loadedDeclarationArtifact{}, err
	}
	if record.Ref() != ref {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: ArtifactReader returned another Artifact",
			basespec.ErrInvalid,
		)
	}
	if record.State != artifact.StateAvailable {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q is not available",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	declarationType := declaration.Type(record.Kind)
	if err := declarationType.Validate(); err != nil {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q has unsupported declaration type: %w",
			basespec.ErrReferenceUnresolved,
			record.ID,
			err,
		)
	}
	if expectedType != "" && declarationType != expectedType {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q has type %q, expected %q",
			basespec.ErrReferenceUnresolved,
			record.ID,
			declarationType,
			expectedType,
		)
	}
	if expectedVersion != "" && record.LogicalVersion != expectedVersion {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q has Text insertion identity %q, expected %q",
			basespec.ErrReferenceUnresolved,
			record.ID,
			record.LogicalVersion,
			expectedVersion,
		)
	}

	definitionValue, err := r.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return loadedDeclarationArtifact{}, err
	}
	if err := validateDefinitionContract(definitionValue, declarationType); err != nil {
		return loadedDeclarationArtifact{}, err
	}
	if definitionValue.Kind != record.Kind ||
		definitionValue.LogicalName != record.LogicalName ||
		definitionValue.LogicalVersion != record.LogicalVersion {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact Definition identity differs from Artifact state",
			basespec.ErrDigestMismatch,
		)
	}

	entry, err := declaration.DecodeCanonicalEntryJSON(definitionValue.Body)
	if err != nil {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact Definition body is not a canonical declaration: %w",
			basespec.ErrReferenceUnresolved,
			err,
		)
	}
	header := entry.Header()
	if header.Type != declarationType ||
		header.Name != string(definitionValue.LogicalName) {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact Definition declaration identity differs from Artifact state",
			basespec.ErrDigestMismatch,
		)
	}
	if declarationType == declaration.TypeText {
		text, err := textv1.DecodeTextEntry(entry)
		if err != nil {
			return loadedDeclarationArtifact{}, err
		}
		if record.LogicalVersion != basespec.LogicalVersion(text.Insert) {
			return loadedDeclarationArtifact{}, fmt.Errorf(
				"%w: Text Artifact identity does not match insert target",
				basespec.ErrDigestMismatch,
			)
		}
	}

	return loadedDeclarationArtifact{
		record:          record,
		definition:      definitionValue,
		entry:           entry,
		declarationType: declarationType,
	}, nil
}

func (r *Resolver) resolveTerminalArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedType declaration.Type,
	expectedVersion basespec.LogicalVersion,
) (artifact.ArtifactRef, error) {
	current := ref
	expectedName := basespec.LogicalName("")
	seen := make(map[artifact.ArtifactRef]struct{})

	for depth := 0; depth <= r.limits.MaxDepth; depth++ {
		if _, duplicate := seen[current]; duplicate {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: source-selected declaration alias cycle at Artifact %q",
				basespec.ErrReferenceUnresolved,
				current.ArtifactID,
			)
		}
		if len(seen) >= r.limits.MaxNodes {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: source-selected declaration alias limit exceeded",
				basespec.ErrLocatorLimitExceeded,
			)
		}
		seen[current] = struct{}{}

		loaded, err := r.loadAvailableDeclarationArtifact(
			ctx,
			current,
			expectedType,
			expectedVersion,
		)
		if err != nil {
			return artifact.ArtifactRef{}, err
		}
		if expectedName != "" &&
			loaded.record.LogicalName != expectedName {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: source-selected declaration resolved to another name",
				basespec.ErrReferenceUnresolved,
			)
		}

		alias, err := sourceSelectedAlias(loaded.entry)
		if err != nil {
			return artifact.ArtifactRef{}, err
		}
		if !alias {
			return current, nil
		}
		if r.locators == nil {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: source-selected declaration requires a locator resolver",
				basespec.ErrLocatorUnresolved,
			)
		}

		header := loaded.entry.Header()
		next, err := r.locators.ResolveArtifactLocator(
			ctx,
			LocatorRequest{
				RootID:              loaded.record.RootID,
				From:                pointerArtifact(loaded.record),
				Entry:               loaded.entry.Clone(),
				Locator:             header.Locator.Clone(),
				ExpectedType:        header.Type,
				ExpectedLogicalName: basespec.LogicalName(header.Name),
			},
		)
		if err != nil {
			return artifact.ArtifactRef{}, err
		}
		if err := next.Validate(); err != nil {
			return artifact.ArtifactRef{}, err
		}
		if next.RootID != loaded.record.RootID {
			return artifact.ArtifactRef{}, fmt.Errorf(
				"%w: locator resolver returned an Artifact from another Root",
				basespec.ErrInvalid,
			)
		}
		expectedType = header.Type
		expectedName = basespec.LogicalName(header.Name)
		current = next
	}

	return artifact.ArtifactRef{}, fmt.Errorf(
		"%w: source-selected declaration alias depth exceeded",
		basespec.ErrLocatorLimitExceeded,
	)
}

func sourceSelectedAlias(entry declaration.Entry) (bool, error) {
	header := entry.Header()
	if header.Locator == nil {
		return false, nil
	}
	switch header.Type {
	case declaration.TypeMCP:
		value, err := mcpv1.DecodeMCPEntry(entry)
		if err != nil {
			return false, err
		}
		return value.Locator != nil, nil
	case declaration.TypeMCPPolicy:
		value, err := mcppolicyv1.DecodeMCPPolicyEntry(entry)
		if err != nil {
			return false, err
		}
		return value.Locator != nil, nil
	default:
		return false, nil
	}
}

func (r *Resolver) reserve(
	state *resolutionState,
	depth int,
) error {
	if depth > r.limits.MaxDepth {
		return fmt.Errorf(
			"%w: Artifact resolution exceeds depth %d",
			basespec.ErrLocatorLimitExceeded,
			r.limits.MaxDepth,
		)
	}
	state.nodes++
	if state.nodes > r.limits.MaxNodes {
		return fmt.Errorf(
			"%w: Artifact resolution exceeds %d nodes",
			basespec.ErrLocatorLimitExceeded,
			r.limits.MaxNodes,
		)
	}
	return nil
}

func validateDefinitionContract(
	value definition.Definition,
	declarationType declaration.Type,
) error {
	key, found := codec.SchemaKeyForType(declarationType)
	if !found {
		return fmt.Errorf(
			"%w: no schema is registered for Artifact type %q",
			basespec.ErrUnsupported,
			declarationType,
		)
	}
	if value.SchemaID != key.SchemaID ||
		value.SchemaVersion != key.SchemaVersion {
		return fmt.Errorf(
			"%w: Artifact Definition schema does not match Artifact type %q",
			basespec.ErrDigestMismatch,
			declarationType,
		)
	}
	return nil
}

func pointerArtifact(value artifact.Artifact) *artifact.Artifact {
	copyValue := value.Clone()
	return &copyValue
}

func cloneArtifactPointer(value *artifact.Artifact) *artifact.Artifact {
	if value == nil {
		return nil
	}
	return pointerArtifact(*value)
}

func pointerDefinition(value definition.Definition) *definition.Definition {
	copyValue := value.Clone()
	return &copyValue
}

func pointerRelationship(
	value ResolvedRelationship,
) *ResolvedRelationship {
	copyValue := value
	return &copyValue
}

func cloneOutputMatch(
	value *declaration.OutputMatch,
) *declaration.OutputMatch {
	if value == nil {
		return nil
	}
	copyValue := value.Clone()
	return &copyValue
}

func cloneMappedTarget(value *MappedTarget) *MappedTarget {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
