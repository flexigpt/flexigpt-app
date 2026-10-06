package composition

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type resolutionState struct {
	nodes int

	active map[artifactModel.ArtifactRef]struct{}

	directRoot *artifactModel.ArtifactRef

	compositionRootID    rootModel.RootID
	compositionSourceID  sourceModel.SourceID
	hasCompositionSource bool
}

func newResolutionState() resolutionState {
	return resolutionState{
		active: make(map[artifactModel.ArtifactRef]struct{}),
	}
}

func (s *resolutionState) usesCompositionSource(
	rootID rootModel.RootID,
) bool {
	return s != nil &&
		s.hasCompositionSource &&
		s.compositionRootID == rootID
}

type loadedDeclarationArtifact struct {
	record          artifactModel.Artifact
	definition      definitionModel.Definition
	entry           declaration.Entry
	declarationType declaration.Type
}

func (r *Resolver) ResolveText(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeText)
}

func (r *Resolver) ResolveModel(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeModel)
}

func (r *Resolver) ResolveModelProvider(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeModelProvider)
}

func (r *Resolver) ResolveTool(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeTool)
}

func (r *Resolver) ResolveSkill(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeSkill)
}

func (r *Resolver) ResolveMCP(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeMCP)
}

func (r *Resolver) ResolveMCPPolicy(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeMCPPolicy)
}

func (r *Resolver) ResolvePlugin(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypePlugin)
}

func (r *Resolver) ResolveAgent(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeAgent)
}

func (r *Resolver) ResolveTeam(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeTeam)
}

func (r *Resolver) ResolveLoop(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeLoop)
}

func (r *Resolver) ResolveWorkflow(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeWorkflow)
}

func (r *Resolver) ResolveWorkspace(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeWorkspace)
}

func (r *Resolver) ResolveWorkspaceWithCompositionSource(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	compositionSourceID sourceModel.SourceID,
) (*ResolvedEntry, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	if err := compositionSourceID.Validate(); err != nil {
		return nil, err
	}

	state := newResolutionState()
	state.compositionRootID = ref.RootID
	state.compositionSourceID = compositionSourceID
	state.hasCompositionSource = true

	return r.resolveArtifact(
		ctx,
		&state,
		ref,
		declaration.TypeWorkspace,
		"",
		0,
	)
}

// ResolveTerminalArtifact follows only declaration aliases. It does not expand
// relationships and does not synthesize direct capability targets.
func (r *Resolver) ResolveTerminalArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (artifactModel.ArtifactRef, error) {
	if err := ref.Validate(); err != nil {
		return artifactModel.ArtifactRef{}, err
	}

	return r.resolveTerminalArtifact(ctx, ref, "", "")
}

func (r *Resolver) resolveTyped(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expected declaration.Type,
) (*ResolvedEntry, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}

	state := newResolutionState()
	return r.resolveArtifact(ctx, &state, ref, expected, "", 0)
}

func (r *Resolver) resolveArtifact(
	ctx context.Context,
	state *resolutionState,
	ref artifactModel.ArtifactRef,
	expectedType declaration.Type,
	expectedVersion spec.LogicalVersion,
	depth int,
) (*ResolvedEntry, error) {
	if err := r.reserve(state, depth); err != nil {
		return nil, err
	}

	loaded, err := r.resolveTerminalDeclaration(
		ctx,
		ref,
		expectedType,
		expectedVersion,
	)
	if err != nil {
		return nil, err
	}
	ref = loaded.record.Ref()

	if _, active := state.active[ref]; active {
		return nil, fmt.Errorf(
			"%w: declaration composition cycle at Artifact %q",
			spec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	state.active[ref] = struct{}{}
	defer delete(state.active, ref)

	target, err := r.targetForLoadedArtifact(ctx, state, loaded)
	if err != nil {
		return nil, err
	}

	node := &ResolvedEntry{
		Type:              loaded.declarationType,
		scopeRootID:       loaded.record.RootID,
		DeclarationOrigin: pointerArtifact(loaded.record),
		Artifact:          pointerArtifact(loaded.record),
		Definition:        pointerDefinition(loaded.definition),
		Target:            pointerTarget(target),
	}

	// Direct Plugin membership resolves the immediate target but intentionally
	// does not recursively expand that target's own graph.
	if state.directRoot != nil && ref != *state.directRoot {
		return node, nil
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

func (r *Resolver) targetForLoadedArtifact(
	ctx context.Context,
	state *resolutionState,
	loaded loadedDeclarationArtifact,
) (CapabilityTarget, error) {
	target := CapabilityTarget{
		Form:       TargetFormArtifact,
		Type:       loaded.declarationType,
		Name:       loaded.record.LogicalName,
		Provenance: targetProvenanceForArtifact(r, state, loaded.record),
		Artifact:   pointerArtifactRef(loaded.record.Ref()),
	}

	projector := r.projectors[loaded.declarationType]
	if projector == nil {
		return target, nil
	}

	projected, handled, err := projector.ProjectArtifactCapability(
		ctx,
		ArtifactCapabilityRequest{
			Artifact: loaded.record.Clone(),
			Type:     loaded.declarationType,
		},
	)
	if err != nil {
		return CapabilityTarget{}, err
	}
	if !handled {
		return target, nil
	}
	if err := projected.Validate(); err != nil {
		return CapabilityTarget{}, err
	}
	if projected.Type != loaded.declarationType ||
		projected.Name != loaded.record.LogicalName {
		return CapabilityTarget{}, fmt.Errorf(
			"%w: projected capability target does not match Artifact identity",
			spec.ErrInvalid,
		)
	}

	switch projected.Form {
	case TargetFormArtifact:
		if projected.Artifact == nil ||
			*projected.Artifact != loaded.record.Ref() {
			return CapabilityTarget{}, fmt.Errorf(
				"%w: projected Artifact capability target changed Artifact identity",
				spec.ErrInvalid,
			)
		}

	case TargetFormDirect:
		if projected.Provenance != TargetProvenanceDirectCapability {
			return CapabilityTarget{}, fmt.Errorf(
				"%w: projected direct capability target has invalid provenance",
				spec.ErrInvalid,
			)
		}

	default:
		return CapabilityTarget{}, fmt.Errorf(
			"%w: projected capability target has unsupported form",
			spec.ErrInvalid,
		)
	}
	return projected.Clone(), nil
}

func (r *Resolver) loadAvailableDeclarationArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedType declaration.Type,
	expectedVersion spec.LogicalVersion,
) (loadedDeclarationArtifact, error) {
	record, err := r.artifacts.Get(ctx, ref)
	if err != nil {
		return loadedDeclarationArtifact{}, err
	}
	if record.Ref() != ref {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact reader returned another Artifact",
			spec.ErrInvalid,
		)
	}
	if record.State != artifactModel.StateAvailable {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q is not available",
			spec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	declarationType := declaration.Type(record.Kind)
	if err := declarationType.Validate(); err != nil {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q has unsupported declaration type: %w",
			spec.ErrReferenceUnresolved,
			record.ID,
			err,
		)
	}
	if expectedType != "" && declarationType != expectedType {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q has type %q, expected %q",
			spec.ErrReferenceUnresolved,
			record.ID,
			declarationType,
			expectedType,
		)
	}
	if expectedVersion != "" &&
		record.LogicalVersion != expectedVersion {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q has logical version %q, expected %q",
			spec.ErrReferenceUnresolved,
			record.ID,
			record.LogicalVersion,
			expectedVersion,
		)
	}
	if record.ResolvedDefinition == nil {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q has no current Definition",
			spec.ErrDefinitionNotFound,
			record.ID,
		)
	}

	definitionValue, err := r.artifacts.GetDefinition(ctx, record.Ref())
	if err != nil {
		return loadedDeclarationArtifact{}, err
	}
	if definitionValue.Digest != *record.ResolvedDefinition {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact Definition changed during resolution",
			spec.ErrRefreshRequired,
		)
	}
	if definitionValue.Kind != record.Kind ||
		definitionValue.LogicalName != record.LogicalName ||
		definitionValue.LogicalVersion != record.LogicalVersion {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact Definition identity differs from Artifact state",
			spec.ErrDigestMismatch,
		)
	}

	entry, err := r.interpretations.EntryFromDefinition(definitionValue)
	if err != nil {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact Definition interpretation: %w",
			spec.ErrDigestMismatch,
			err,
		)
	}
	header := entry.Header()
	if header.Type != declarationType ||
		header.Name != string(record.LogicalName) {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact Definition declaration identity differs from Artifact state",
			spec.ErrDigestMismatch,
		)
	}

	return loadedDeclarationArtifact{
		record:          record.Clone(),
		definition:      definitionValue.Clone(),
		entry:           entry,
		declarationType: declarationType,
	}, nil
}

func (r *Resolver) resolveTerminalArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedType declaration.Type,
	expectedVersion spec.LogicalVersion,
) (artifactModel.ArtifactRef, error) {
	loaded, err := r.resolveTerminalDeclaration(
		ctx,
		ref,
		expectedType,
		expectedVersion,
	)
	if err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	return loaded.record.Ref(), nil
}

func (r *Resolver) resolveTerminalDeclaration(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedType declaration.Type,
	expectedVersion spec.LogicalVersion,
) (loadedDeclarationArtifact, error) {
	current := ref
	expectedName := spec.LogicalName("")
	seen := make(map[artifactModel.ArtifactRef]struct{})

	for depth := 0; depth <= r.limits.MaxDepth; depth++ {
		if _, duplicate := seen[current]; duplicate {
			return loadedDeclarationArtifact{}, fmt.Errorf(
				"%w: declaration alias cycle at Artifact %q",
				spec.ErrReferenceUnresolved,
				current.ArtifactID,
			)
		}
		if len(seen) >= r.limits.MaxNodes {
			return loadedDeclarationArtifact{}, fmt.Errorf(
				"%w: declaration alias limit exceeded",
				spec.ErrLocatorLimitExceeded,
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
			return loadedDeclarationArtifact{}, err
		}
		if expectedName != "" &&
			loaded.record.LogicalName != expectedName {
			return loadedDeclarationArtifact{}, fmt.Errorf(
				"%w: declaration alias resolved to another logical name",
				spec.ErrReferenceUnresolved,
			)
		}

		header := loaded.entry.Header()
		if !r.interpretations.DeclarationAliasEligible(
			loaded.declarationType,
		) || header.Locator == nil {
			return loaded, nil
		}
		if r.locators == nil {
			return loadedDeclarationArtifact{}, fmt.Errorf(
				"%w: declaration alias requires a locator resolver",
				spec.ErrLocatorUnresolved,
			)
		}

		next, err := r.locators.ResolveArtifactLocator(
			ctx,
			LocatorRequest{
				RootID:              loaded.record.RootID,
				From:                pointerArtifact(loaded.record),
				Entry:               loaded.entry.Clone(),
				Locator:             header.Locator.Clone(),
				ExpectedType:        header.Type,
				ExpectedLogicalName: spec.LogicalName(header.Name),
			},
		)
		if err != nil {
			return loadedDeclarationArtifact{}, err
		}
		if err := next.Validate(); err != nil {
			return loadedDeclarationArtifact{}, err
		}
		if next.RootID != loaded.record.RootID {
			return loadedDeclarationArtifact{}, fmt.Errorf(
				"%w: locator resolver returned an Artifact from another Root",
				spec.ErrInvalid,
			)
		}

		expectedType = header.Type
		expectedName = spec.LogicalName(header.Name)
		current = next
	}

	return loadedDeclarationArtifact{}, fmt.Errorf(
		"%w: declaration alias depth exceeded",
		spec.ErrLocatorLimitExceeded,
	)
}

func (r *Resolver) reserve(
	state *resolutionState,
	depth int,
) error {
	if depth > r.limits.MaxDepth {
		return fmt.Errorf(
			"%w: Artifact composition exceeds depth %d",
			spec.ErrLocatorLimitExceeded,
			r.limits.MaxDepth,
		)
	}
	state.nodes++
	if state.nodes > r.limits.MaxNodes {
		return fmt.Errorf(
			"%w: Artifact composition exceeds %d nodes",
			spec.ErrLocatorLimitExceeded,
			r.limits.MaxNodes,
		)
	}
	return nil
}

func targetProvenanceForArtifact(
	r *Resolver,
	state *resolutionState,
	value artifactModel.Artifact,
) TargetProvenance {
	if state != nil &&
		state.usesCompositionSource(value.RootID) &&
		value.Binding.SourceID == state.compositionSourceID {
		return TargetProvenanceCompositionSource
	}
	if r != nil &&
		r.scope.BuiltinRoot != "" &&
		value.RootID == r.scope.BuiltinRoot {
		return TargetProvenanceBuiltinScope
	}
	return TargetProvenanceCurrentRoot
}

func pointerArtifact(
	value artifactModel.Artifact,
) *artifactModel.Artifact {
	copyValue := value.Clone()
	return &copyValue
}

func pointerArtifactRef(
	value artifactModel.ArtifactRef,
) *artifactModel.ArtifactRef {
	copyValue := value
	return &copyValue
}

func cloneArtifactPointer(
	value *artifactModel.Artifact,
) *artifactModel.Artifact {
	if value == nil {
		return nil
	}
	return pointerArtifact(*value)
}

func pointerDefinition(
	value definitionModel.Definition,
) *definitionModel.Definition {
	copyValue := value.Clone()
	return &copyValue
}
