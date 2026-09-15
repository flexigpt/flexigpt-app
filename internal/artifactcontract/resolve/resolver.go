package resolve

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/codec"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/collectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/contextv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/instructionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/loopv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/skillv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/teamv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workflowv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workspacev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
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

func (r *Resolver) ResolveArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (Graph, error) {
	if r == nil || r.artifacts == nil {
		return Graph{}, basespec.ErrClosed
	}
	if err := validateResolutionContext(ctx); err != nil {
		return Graph{}, err
	}
	if err := ref.Validate(); err != nil {
		return Graph{}, err
	}

	state := newResolutionState()
	entry, err := r.resolveArtifact(
		ctx,
		&state,
		ref,
		"",
		0,
	)
	if err != nil {
		return Graph{}, err
	}
	return Graph{Root: entry}, nil
}

func (r *Resolver) resolveInlineGraph(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	entry declaration.Entry,
) (Graph, error) {
	if r == nil || r.artifacts == nil {
		return Graph{}, basespec.ErrClosed
	}
	if err := entry.Validate(); err != nil {
		return Graph{}, err
	}
	value, err := r.resolveEntry(
		ctx,
		state,
		rootID,
		entry,
		nil,
		0,
		nil,
	)
	if err != nil {
		return Graph{}, err
	}
	return Graph{Root: value}, nil
}

func (r *Resolver) ResolveReference(
	ctx context.Context,
	rootID root.RootID,
	declarationType declaration.Type,
	name basespec.LogicalName,
) (Graph, error) {
	if r == nil || r.artifacts == nil {
		return Graph{}, basespec.ErrClosed
	}
	if err := validateResolutionContext(ctx); err != nil {
		return Graph{}, err
	}
	if err := rootID.Validate(); err != nil {
		return Graph{}, err
	}
	if err := declarationType.Validate(); err != nil {
		return Graph{}, err
	}
	if err := name.Validate(); err != nil {
		return Graph{}, err
	}

	state := newResolutionState()

	entry, err := r.resolveSymbolic(
		ctx,
		&state,
		rootID,
		declarationType,
		name,
		0,
	)
	if err != nil {
		return Graph{}, err
	}
	return Graph{Root: entry}, nil
}

func (r *Resolver) ResolveWorkspace(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedWorkspace, error) {
	graph, err := r.ResolveArtifact(ctx, ref)
	if err != nil {
		return nil, err
	}
	if graph.Root == nil ||
		graph.Root.Type != declaration.TypeWorkspace ||
		graph.Root.Workspace == nil {
		return nil, fmt.Errorf(
			"%w: Artifact %q is not a Workspace",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	return graph.Root.Workspace, nil
}

func (r *Resolver) resolveArtifact(
	ctx context.Context,
	state *resolutionState,
	ref artifact.ArtifactRef,
	expected declaration.Type,
	depth int,
) (*ResolvedEntry, error) {
	if err := r.reserve(state, depth); err != nil {
		return nil, err
	}
	if _, resolving := state.active[ref]; resolving {
		return nil, fmt.Errorf(
			"%w: declaration composition cycle at Artifact %q",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	state.active[ref] = struct{}{}
	defer delete(state.active, ref)

	record, err := r.artifacts.Get(ctx, ref)
	if err != nil {
		return nil, err
	}
	if record.State != artifact.StateAvailable {
		return nil, fmt.Errorf(
			"%w: Artifact %q is not available",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if !r.options.IncludeDisabled && !record.Enabled {
		return nil, fmt.Errorf(
			"%w: Artifact %q is disabled",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	declarationType := declaration.Type(record.Kind)
	if err := declarationType.Validate(); err != nil {
		return nil, fmt.Errorf(
			"%w: Artifact %q has unsupported declaration type: %w",
			basespec.ErrReferenceUnresolved,
			record.ID,
			err,
		)
	}
	if expected != "" && declarationType != expected {
		return nil, fmt.Errorf(
			"%w: Artifact %q has type %q, expected %q",
			basespec.ErrReferenceUnresolved,
			record.ID,
			declarationType,
			expected,
		)
	}

	definitionValue, err := r.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return nil, err
	}
	if err := validateDefinitionContract(definitionValue, declarationType); err != nil {
		return nil, err
	}
	if definitionValue.Kind != record.Kind {
		return nil, fmt.Errorf(
			"%w: Artifact Definition kind does not match Artifact kind",
			basespec.ErrDigestMismatch,
		)
	}
	if definitionValue.LogicalName != record.LogicalName ||
		definitionValue.LogicalVersion != record.LogicalVersion {
		return nil, fmt.Errorf(
			"%w: Artifact Definition identity does not match Artifact state",
			basespec.ErrDigestMismatch,
		)
	}
	entry, err := declaration.DecodeCanonicalEntryJSON(definitionValue.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: Artifact Definition body is not a canonical declaration: %w",
			basespec.ErrReferenceUnresolved,
			err,
		)
	}
	if entry.Header().Type != declarationType {
		return nil, fmt.Errorf(
			"%w: Artifact Definition declaration type differs from Artifact kind",
			basespec.ErrDigestMismatch,
		)
	}
	if entry.Header().Name != string(definitionValue.LogicalName) {
		return nil, fmt.Errorf(
			"%w: Artifact Definition declaration name differs from logical name",
			basespec.ErrDigestMismatch,
		)
	}

	if shouldResolveDeclarationLocator(
		entry,
		entry.Header().Locator,
	) {
		return r.resolveDeclarationLocator(
			ctx,
			state,
			record.RootID,
			pointerArtifact(record),
			entry,
			depth,
		)
	}

	node := &ResolvedEntry{
		Type:              declarationType,
		scopeRootID:       record.RootID,
		DeclarationOrigin: pointerArtifact(record),
		Artifact:          pointerArtifact(record),
		Definition:        pointerDefinition(definitionValue),
	}

	if err := r.resolveStructure(
		ctx,
		state,
		node,
		entry,
		depth,
		nil,
	); err != nil {
		return nil, err
	}
	return node, nil
}

func (r *Resolver) resolveSymbolic(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	declarationType declaration.Type,
	name basespec.LogicalName,
	depth int,
) (*ResolvedEntry, error) {
	records, err := r.artifacts.FindByIdentity(
		ctx,
		rootID,
		artifact.ArtifactKind(declarationType),
		name,
	)
	if err != nil {
		return nil, err
	}

	candidates := make([]artifact.Artifact, 0, len(records))
	for _, record := range records {
		if record.State != artifact.StateAvailable {
			continue
		}
		if !r.options.IncludeDisabled && !record.Enabled {
			continue
		}
		candidates = append(candidates, record)
	}
	switch len(candidates) {
	case 0:
		return nil, fmt.Errorf(
			"%w: %s/%s",
			basespec.ErrReferenceUnresolved,
			declarationType,
			name,
		)
	case 1:
		return r.resolveArtifact(
			ctx,
			state,
			candidates[0].Ref(),
			declarationType,
			depth,
		)
	default:
		return nil, fmt.Errorf(
			"%w: %s/%s has %d available Artifacts in Root %q",
			basespec.ErrIdentityConflict,
			declarationType,
			name,
			len(candidates),
			rootID,
		)
	}
}

func (r *Resolver) resolveEntry(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	entry declaration.Entry,
	from *artifact.Artifact,
	depth int,
	implicitLoopOwner *ResolvedEntry,
) (*ResolvedEntry, error) {
	if err := entry.Validate(); err != nil {
		return nil, err
	}

	header := entry.Header()
	if entry.IsSymbolic() {
		return r.resolveSymbolic(
			ctx,
			state,
			rootID,
			header.Type,
			basespec.LogicalName(header.Name),
			depth,
		)
	}
	if shouldResolveDeclarationLocator(entry, header.Locator) {
		return r.resolveDeclarationLocator(
			ctx,
			state,
			rootID,
			from,
			entry,
			depth,
		)
	}
	bodylessImplicitLoop, err := isBodylessImplicitLoop(
		entry,
		implicitLoopOwner,
	)
	if err != nil {
		return nil, err
	}
	if from != nil &&
		header.Name != "" &&
		!bodylessImplicitLoop {
		return r.resolveNamedInlineArtifact(
			ctx,
			state,
			rootID,
			from,
			entry,
			depth,
		)
	}

	if err := r.reserve(state, depth); err != nil {
		return nil, err
	}
	node := &ResolvedEntry{
		Type:              header.Type,
		scopeRootID:       rootID,
		DeclarationOrigin: cloneArtifactPointer(from),
		Inline:            pointerEntry(entry),
	}
	if err := r.resolveStructure(
		ctx,
		state,
		node,
		entry,
		depth,
		implicitLoopOwner,
	); err != nil {
		return nil, err
	}
	return node, nil
}

func (r *Resolver) resolveNamedInlineArtifact(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	from *artifact.Artifact,
	entry declaration.Entry,
	depth int,
) (*ResolvedEntry, error) {
	header := entry.Header()
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return nil, err
	}
	records, err := r.artifacts.FindByIdentity(
		ctx,
		rootID,
		artifact.ArtifactKind(header.Type),
		basespec.LogicalName(header.Name),
	)
	if err != nil {
		return nil, err
	}

	matches := make([]artifact.Artifact, 0, len(records))
	for _, record := range records {
		if record.State != artifact.StateAvailable ||
			(!r.options.IncludeDisabled && !record.Enabled) {
			continue
		}
		if from != nil &&
			!isNestedDeclarationOrigin(record, *from) {
			continue
		}
		definitionValue, err := r.artifacts.GetDefinition(
			ctx,
			record.Ref(),
		)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(raw, definitionValue.Body) {
			matches = append(matches, record)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf(
			"%w: named inline declaration %s/%s has no matching Root Artifact",
			basespec.ErrReferenceUnresolved,
			header.Type,
			header.Name,
		)
	case 1:
		return r.resolveArtifact(
			ctx,
			state,
			matches[0].Ref(),
			header.Type,
			depth,
		)
	default:
		return nil, fmt.Errorf(
			"%w: named inline declaration %s/%s matches %d Artifacts",
			basespec.ErrIdentityConflict,
			header.Type,
			header.Name,
			len(matches),
		)
	}
}

func isNestedDeclarationOrigin(
	record artifact.Artifact,
	parent artifact.Artifact,
) bool {
	if record.Binding.SourceID != parent.Binding.SourceID ||
		record.Binding.Locator != parent.Binding.Locator {
		return false
	}

	child := string(record.Binding.SubresourceLocator)
	ancestor := string(parent.Binding.SubresourceLocator)
	if ancestor == "" {
		return child != ""
	}
	return strings.HasPrefix(
		child,
		ancestor+"/",
	)
}

func (r *Resolver) resolveStructure(
	ctx context.Context,
	state *resolutionState,
	node *ResolvedEntry,
	entry declaration.Entry,
	depth int,
	implicitLoopOwner *ResolvedEntry,
) error {
	rootID, _ := node.RootID()
	if rootID == "" {
		return fmt.Errorf(
			"%w: inline declaration requires a Root resolution scope",
			basespec.ErrInvalid,
		)
	}

	var from *artifact.Artifact
	if node.Artifact != nil {
		from = node.Artifact
	} else if node.DeclarationOrigin != nil {
		from = node.DeclarationOrigin
	}

	switch node.Type {
	case declaration.TypeInstruction:
		_, err := instructionv1.DecodeInstructionEntry(entry)
		return err

	case declaration.TypeContext:
		_, err := contextv1.DecodeContextEntry(entry)
		return err

	case declaration.TypeTool:
		_, err := toolv1.DecodeToolEntry(entry)
		return err

	case declaration.TypeModel:
		_, err := modelv1.DecodeModelEntry(entry)
		return err

	case declaration.TypeSkill:
		value, err := skillv1.DecodeSkillEntry(entry)
		if err != nil {
			return err
		}
		node.AllowedTools, err = r.resolveEntries(
			ctx,
			state,
			rootID,
			value.AllowedTools,
			from,
			depth+1,
			nil,
		)
		return err

	case declaration.TypeMCP:
		_, err := mcpv1.DecodeMCPEntry(entry)
		return err

	case declaration.TypeMCPPolicy:
		_, err := mcppolicyv1.DecodeMCPPolicyEntry(entry)
		return err

	case declaration.TypeCollection:
		value, err := collectionv1.DecodeCollectionEntry(entry)
		if err != nil {
			return err
		}
		node.Members, err = r.resolveEntries(
			ctx,
			state,
			rootID,
			value.Members,
			from,
			depth+1,
			nil,
		)
		return err

	case declaration.TypeAgent:
		value, err := agentv1.DecodeAgentEntry(entry)
		if err != nil {
			return err
		}
		node.Members, err = r.resolveEntries(
			ctx,
			state,
			rootID,
			value.Members,
			from,
			depth+1,
			nil,
		)
		if err != nil {
			return err
		}
		if value.Program != nil {
			node.Program, err = r.resolveEntry(
				ctx,
				state,
				rootID,
				*value.Program,
				from,
				depth+1,
				node,
			)
		}
		return err

	case declaration.TypeTeam:
		value, err := teamv1.DecodeTeamEntry(entry)
		if err != nil {
			return err
		}
		node.Members, err = r.resolveEntries(
			ctx,
			state,
			rootID,
			value.Members,
			from,
			depth+1,
			nil,
		)
		if err != nil {
			return err
		}
		if value.Program != nil {
			node.Program, err = r.resolveEntry(
				ctx,
				state,
				rootID,
				*value.Program,
				from,
				depth+1,
				node,
			)
		}
		return err

	case declaration.TypeLoop:
		value, err := loopv1.DecodeLoopEntry(
			entry,
			implicitLoopOwner != nil,
		)
		if err != nil {
			return err
		}
		loop := &ResolvedLoop{
			MaxIterations: value.MaxIterations,
		}
		if value.Until != nil {
			copyValue := value.Until.Clone()
			loop.Until = &copyValue
		}
		node.Loop = loop
		if value.Body != nil {
			loop.Body, err = r.resolveEntry(
				ctx,
				state,
				rootID,
				*value.Body,
				from,
				depth+1,
				nil,
			)
			return err
		}
		loop.Body = implicitLoopOwner
		return nil

	case declaration.TypeWorkflow:
		value, err := workflowv1.DecodeWorkflowEntry(entry)
		if err != nil {
			return err
		}
		workflow := &ResolvedWorkflow{
			Start: append([]string(nil), value.Start...),
			Nodes: make(
				[]ResolvedWorkflowNode,
				0,
				len(value.Nodes),
			),
			Edges: make(
				[]ResolvedWorkflowEdge,
				0,
				len(value.Edges),
			),
		}
		node.Workflow = workflow
		for _, workflowNode := range value.Nodes {
			target, err := r.resolveEntry(
				ctx,
				state,
				rootID,
				workflowNode.Target,
				from,
				depth+1,
				nil,
			)
			if err != nil {
				return err
			}
			join := workflowNode.Join
			if join == "" {
				join = workflowv1.JoinAll
			}
			workflow.Nodes = append(
				workflow.Nodes,
				ResolvedWorkflowNode{
					ID:     workflowNode.ID,
					Join:   string(join),
					Target: target,
				},
			)
		}
		for _, edge := range value.Edges {
			output := ResolvedWorkflowEdge{
				From: edge.From,
				To:   edge.To,
			}
			if edge.Match != nil {
				copyValue := edge.Match.Clone()
				output.Match = &copyValue
			}
			workflow.Edges = append(workflow.Edges, output)
		}
		return nil

	case declaration.TypeWorkspace:
		value, err := workspacev1.DecodeWorkspaceEntry(entry)
		if err != nil {
			return err
		}
		workspace := &ResolvedWorkspace{
			Roots: make([]*ResolvedEntry, 0, len(value.Roots)),
		}
		node.Workspace = workspace
		for _, rootEntry := range value.Roots {
			resolved, err := r.resolveEntry(
				ctx,
				state,
				rootID,
				rootEntry,
				from,
				depth+1,
				nil,
			)
			if err != nil {
				return err
			}
			workspace.Roots = append(workspace.Roots, resolved)
		}
		return nil

	default:
		return fmt.Errorf(
			"%w: unsupported declaration type %q",
			basespec.ErrUnsupported,
			node.Type,
		)
	}
}

func (r *Resolver) resolveEntries(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	values []declaration.Entry,
	from *artifact.Artifact,
	depth int,
	implicitLoopOwner *ResolvedEntry,
) ([]*ResolvedEntry, error) {
	output := make([]*ResolvedEntry, 0, len(values))
	for _, value := range values {
		resolved, err := r.resolveEntry(
			ctx,
			state,
			rootID,
			value,
			from,
			depth,
			implicitLoopOwner,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, resolved)
	}
	return output, nil
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

func (r *Resolver) resolveDeclarationLocator(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	from *artifact.Artifact,
	entry declaration.Entry,
	depth int,
) (*ResolvedEntry, error) {
	if r.locators == nil {
		return nil, fmt.Errorf(
			"%w: declaration locator requires a locator resolver",
			basespec.ErrLocatorUnresolved,
		)
	}
	header := entry.Header()
	ref, err := r.locators.ResolveArtifactLocator(
		ctx,
		LocatorRequest{
			RootID:              rootID,
			From:                cloneArtifactPointer(from),
			Entry:               entry.Clone(),
			Locator:             header.Locator.Clone(),
			ExpectedType:        header.Type,
			ExpectedLogicalName: basespec.LogicalName(header.Name),
		},
	)
	if err != nil {
		return nil, err
	}
	if ref.RootID != rootID {
		return nil, fmt.Errorf(
			"%w: locator resolver returned an Artifact from another Root",
			basespec.ErrInvalid,
		)
	}
	resolved, err := r.resolveArtifact(
		ctx,
		state,
		ref,
		header.Type,
		depth+1,
	)
	if err != nil {
		return nil, err
	}
	if header.Name != "" &&
		(resolved.Artifact == nil ||
			resolved.Artifact.LogicalName != basespec.LogicalName(header.Name)) {
		return nil, fmt.Errorf(
			"%w: located declaration %s/%s resolved to another logical name",
			basespec.ErrReferenceUnresolved,
			header.Type,
			header.Name,
		)
	}
	return resolved, nil
}

func shouldResolveDeclarationLocator(
	entry declaration.Entry,
	locator *declaration.Locator,
) bool {
	return locator != nil &&
		entry.IsDeclarationLocatorReference()
}

func isBodylessImplicitLoop(
	entry declaration.Entry,
	owner *ResolvedEntry,
) (bool, error) {
	if owner == nil ||
		entry.Header().Type != declaration.TypeLoop {
		return false, nil
	}

	raw, err := entry.CanonicalJSON()
	if err != nil {
		return false, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return false, err
	}
	_, hasBody := fields["body"]
	return !hasBody, nil
}

func validateDefinitionContract(
	value definition.Definition,
	declarationType declaration.Type,
) error {
	key, found := codec.SchemaKeyForType(declarationType)
	if !found {
		return fmt.Errorf(
			"%w: no canonical schema is registered for declaration type %q",
			basespec.ErrUnsupported,
			declarationType,
		)
	}
	if value.SchemaID != key.SchemaID ||
		value.SchemaVersion != key.SchemaVersion {
		return fmt.Errorf(
			"%w: Artifact Definition schema does not match declaration type %q",
			basespec.ErrDigestMismatch,
			declarationType,
		)
	}
	return nil
}

func pointerArtifact(
	value artifact.Artifact,
) *artifact.Artifact {
	copyValue := value.Clone()
	return &copyValue
}

func cloneArtifactPointer(
	value *artifact.Artifact,
) *artifact.Artifact {
	if value == nil {
		return nil
	}
	return pointerArtifact(*value)
}

func pointerDefinition(
	value definition.Definition,
) *definition.Definition {
	copyValue := value.Clone()
	return &copyValue
}

func pointerEntry(
	value declaration.Entry,
) *declaration.Entry {
	copyValue := value.Clone()
	return &copyValue
}
