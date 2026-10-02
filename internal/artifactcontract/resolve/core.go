package resolve

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/codec"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/textv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
)

type resolutionState struct {
	nodes                int
	active               map[artifact.ArtifactRef]struct{}
	directRoot           *artifact.ArtifactRef
	compositionRootID    root.RootID
	compositionSourceID  source.SourceID
	hasCompositionSource bool
}

func newResolutionState() resolutionState {
	return resolutionState{
		active: make(map[artifact.ArtifactRef]struct{}),
	}
}

func (s *resolutionState) usesCompositionSource(rootID root.RootID) bool {
	return s != nil &&
		s.hasCompositionSource &&
		s.compositionRootID == rootID
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

func (r *Resolver) ResolveModelProvider(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	return r.resolveTyped(ctx, ref, declaration.TypeModelProvider)
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

func (r *Resolver) ResolveWorkspaceWithCompositionSource(
	ctx context.Context,
	ref artifact.ArtifactRef,
	compositionSourceID source.SourceID,
) (*ResolvedEntry, error) {
	if r == nil || r.artifacts == nil {
		return nil, model.ErrClosed
	}
	if err := validateResolutionContext(ctx); err != nil {
		return nil, err
	}
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
	return r.resolveArtifact(ctx, &state, ref, declaration.TypeWorkspace, "", 0)
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
			model.ErrReferenceUnresolved,
		)
	}
	return value, nil
}

// ResolveTerminalArtifact follows source-selected aliases for every
// alias-capable declaration type. Text and Skill locators remain resource
// locators, while Tool and Model locator behavior remains outside this
// helper. It does not resolve a generic type/name request or expand
// composition.
func (r *Resolver) ResolveTerminalArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.ArtifactRef, error) {
	if r == nil || r.artifacts == nil {
		return artifact.ArtifactRef{}, model.ErrClosed
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
		return nil, model.ErrClosed
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
	expectedVersion model.LogicalVersion,
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
			model.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	state.active[ref] = struct{}{}
	defer delete(state.active, ref)

	// Direct Collection membership resolution must resolve the Collection
	// relationships, but must not recursively expand the selected member.
	// The selected Artifact is still loaded and type-checked above.
	if state.directRoot != nil && ref != *state.directRoot {
		return &ResolvedEntry{
			Type:              loaded.declarationType,
			scopeRootID:       loaded.record.RootID,
			DeclarationOrigin: pointerArtifact(loaded.record),
			Artifact:          pointerArtifact(loaded.record),
			Definition:        pointerDefinition(loaded.definition),
		}, nil
	}

	if mapper := r.targetMappers[loaded.declarationType]; mapper != nil {
		mapped, handled, err := mapper.MapArtifactTarget(
			ctx,
			ArtifactTargetRequest{
				Artifact:   loaded.record.Clone(),
				Definition: loaded.definition.Clone(),
				Type:       loaded.declarationType,
			},
		)
		if err != nil {
			return nil, err
		}
		if handled {
			if err := mapped.Validate(); err != nil {
				return nil, err
			}
			if mapped.Type != loaded.declarationType ||
				mapped.Name != loaded.record.LogicalName {
				return nil, fmt.Errorf(
					"%w: mapped target does not match Artifact identity",
					model.ErrInvalid,
				)
			}
			return &ResolvedEntry{
				Type:              loaded.declarationType,
				scopeRootID:       loaded.record.RootID,
				DeclarationOrigin: pointerArtifact(loaded.record),
				Mapped:            cloneMappedTarget(&mapped),
			}, nil
		}
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
	expectedVersion model.LogicalVersion,
) (loadedDeclarationArtifact, error) {
	record, err := r.artifacts.Get(ctx, ref)
	if err != nil {
		return loadedDeclarationArtifact{}, err
	}
	if record.Ref() != ref {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: ArtifactReader returned another Artifact",
			model.ErrInvalid,
		)
	}
	if record.State != artifact.StateAvailable {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q is not available",
			model.ErrReferenceUnresolved,
			record.ID,
		)
	}

	declarationType := declaration.Type(record.Kind)
	if err := declarationType.Validate(); err != nil {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q has unsupported declaration type: %w",
			model.ErrReferenceUnresolved,
			record.ID,
			err,
		)
	}
	if expectedType != "" && declarationType != expectedType {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q has type %q, expected %q",
			model.ErrReferenceUnresolved,
			record.ID,
			declarationType,
			expectedType,
		)
	}
	if expectedVersion != "" && record.LogicalVersion != expectedVersion {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q has Text insertion identity %q, expected %q",
			model.ErrReferenceUnresolved,
			record.ID,
			record.LogicalVersion,
			expectedVersion,
		)
	}

	if record.ResolvedDefinition == nil {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact %q has no current Definition",
			model.ErrDefinitionNotFound,
			record.ID,
		)
	}
	definitions, err := r.artifacts.GetDefinitions(
		ctx,
		[]definition.Key{{
			RootID: record.RootID,
			Digest: *record.ResolvedDefinition,
		}},
	)
	if err != nil {
		return loadedDeclarationArtifact{}, err
	}
	if len(definitions) != 1 ||
		definitions[0].Digest != *record.ResolvedDefinition {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact Definition changed during resolution",
			model.ErrRefreshRequired,
		)
	}
	definitionValue := definitions[0]

	if err := validateDefinitionContract(definitionValue, declarationType); err != nil {
		return loadedDeclarationArtifact{}, err
	}
	if definitionValue.Kind != record.Kind ||
		definitionValue.LogicalName != record.LogicalName ||
		definitionValue.LogicalVersion != record.LogicalVersion {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact Definition identity differs from Artifact state",
			model.ErrDigestMismatch,
		)
	}

	entry, err := declaration.DecodeCanonicalEntryJSON(definitionValue.Body)
	if err != nil {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact Definition body is not a canonical declaration: %w",
			model.ErrReferenceUnresolved,
			err,
		)
	}
	header := entry.Header()
	if header.Type != declarationType ||
		header.Name != string(definitionValue.LogicalName) {
		return loadedDeclarationArtifact{}, fmt.Errorf(
			"%w: Artifact Definition declaration identity differs from Artifact state",
			model.ErrDigestMismatch,
		)
	}
	if declarationType == declaration.TypeText {
		text, err := textv1.DecodeTextEntry(entry)
		if err != nil {
			return loadedDeclarationArtifact{}, err
		}
		if record.LogicalVersion != model.LogicalVersion(text.Insert) {
			return loadedDeclarationArtifact{}, fmt.Errorf(
				"%w: Text Artifact identity does not match insert target",
				model.ErrDigestMismatch,
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
	expectedVersion model.LogicalVersion,
) (artifact.ArtifactRef, error) {
	loaded, err := r.resolveTerminalDeclaration(ctx, ref, expectedType, expectedVersion)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}
	return loaded.record.Ref(), nil
}

func (r *Resolver) resolveTerminalDeclaration(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedType declaration.Type,
	expectedVersion model.LogicalVersion,
) (loadedDeclarationArtifact, error) {
	current := ref
	expectedName := model.LogicalName("")
	seen := make(map[artifact.ArtifactRef]struct{})

	for depth := 0; depth <= r.limits.MaxDepth; depth++ {
		if _, duplicate := seen[current]; duplicate {
			return loadedDeclarationArtifact{}, fmt.Errorf(
				"%w: source-selected declaration alias cycle at Artifact %q",
				model.ErrReferenceUnresolved,
				current.ArtifactID,
			)
		}
		if len(seen) >= r.limits.MaxNodes {
			return loadedDeclarationArtifact{}, fmt.Errorf(
				"%w: source-selected declaration alias limit exceeded",
				model.ErrLocatorLimitExceeded,
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
				"%w: source-selected declaration resolved to another name",
				model.ErrReferenceUnresolved,
			)
		}

		alias, err := sourceSelectedAlias(loaded.entry)
		if err != nil {
			return loadedDeclarationArtifact{}, err
		}
		if !alias {
			return loaded, nil
		}
		if r.locators == nil {
			return loadedDeclarationArtifact{}, fmt.Errorf(
				"%w: source-selected declaration requires a locator resolver",
				model.ErrLocatorUnresolved,
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
				ExpectedLogicalName: model.LogicalName(header.Name),
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
				model.ErrInvalid,
			)
		}
		expectedType = header.Type
		expectedName = model.LogicalName(header.Name)
		current = next
	}

	return loadedDeclarationArtifact{}, fmt.Errorf(
		"%w: source-selected declaration alias depth exceeded",
		model.ErrLocatorLimitExceeded,
	)
}

func sourceSelectedAlias(entry declaration.Entry) (bool, error) {
	header := entry.Header()
	if header.Locator == nil {
		return false, nil
	}
	switch header.Type {
	case declaration.TypePlugin,
		declaration.TypeAgent,
		declaration.TypeTeam,
		declaration.TypeLoop,
		declaration.TypeWorkflow,
		declaration.TypeWorkspace,
		declaration.TypeMCP,
		declaration.TypeMCPPolicy:
		return true, nil
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
			model.ErrLocatorLimitExceeded,
			r.limits.MaxDepth,
		)
	}
	state.nodes++
	if state.nodes > r.limits.MaxNodes {
		return fmt.Errorf(
			"%w: Artifact resolution exceeds %d nodes",
			model.ErrLocatorLimitExceeded,
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
			model.ErrUnsupported,
			declarationType,
		)
	}
	if value.SchemaID != key.SchemaID ||
		value.SchemaVersion != key.SchemaVersion {
		return fmt.Errorf(
			"%w: Artifact Definition schema does not match Artifact type %q",
			model.ErrDigestMismatch,
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
