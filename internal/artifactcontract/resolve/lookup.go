package resolve

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
)

func (r *Resolver) resolveEntries(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	values []declaration.Entry,
	from *artifact.Artifact,
	depth int,
	relationshipPath []string,
) ([]*ResolvedEntry, []ResolvedRelationship, error) {
	ordered, err := declaration.SortedMembers(
		"composition members",
		values,
	)
	if err != nil {
		return nil, nil, err
	}

	available := make([]*ResolvedEntry, 0, len(ordered))
	relationships := make([]ResolvedRelationship, 0, len(ordered))
	for _, member := range ordered {
		relationship, err := r.resolveMember(
			ctx,
			state,
			rootID,
			member,
			from,
			depth,
			relationshipPath,
		)
		if err != nil {
			return nil, nil, err
		}
		relationships = append(relationships, relationship)
		if relationship.Resolved != nil {
			available = append(available, relationship.Resolved)
		}
		if relationship.Selector != nil {
			for _, match := range relationship.Selector.Matches {
				if match.Resolved != nil {
					available = append(available, match.Resolved)
				}
			}
		}
	}
	return available, relationships, nil
}

func (r *Resolver) resolveMember(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	member declaration.Entry,
	from *artifact.Artifact,
	depth int,
	relationshipPath []string,
) (ResolvedRelationship, error) {
	var err error
	form, err := member.MemberForm()
	if err != nil {
		return ResolvedRelationship{}, err
	}
	relationshipFields, err := member.Relationship()
	if err != nil {
		return ResolvedRelationship{}, err
	}

	relationship := ResolvedRelationship{
		Declared:  member.Clone(),
		Form:      form,
		Required:  true,
		Scope:     relationshipFields.Scope,
		Overrides: declaration.CloneRawMessageMap(relationshipFields.Overrides),
		Use:       declaration.CloneRawMessageMap(relationshipFields.Use),
	}

	if member.Header().Type == declaration.TypeWorkspace {
		relationship.Status = ResolutionUnavailable
		relationship.Issue = &ResolutionIssue{
			Code:    "artifact.workspace-nested",
			Message: "Workspace cannot be resolved as a nested relationship",
		}
		return relationship, nil
	}

	var resolved *ResolvedEntry
	switch form {
	case declaration.MemberNamed:
		header := member.Header()
		var expectedVersion basespec.LogicalVersion
		expectedVersion, err = memberTextLogicalVersion(member)
		if err != nil {
			return ResolvedRelationship{}, err
		}
		if header.Locator != nil {
			resolved, err = r.resolveLocatedMember(
				ctx,
				state,
				rootID,
				member,
				from,
				expectedVersion,
				depth,
			)
		} else {
			resolved, err = r.resolveNamedMember(
				ctx,
				state,
				rootID,
				header.Type,
				basespec.LogicalName(header.Name),
				expectedVersion,
				relationshipFields.Scope,
				from,
				depth,
			)
		}

	case declaration.MemberContained:
		resolved, err = r.resolveContainedMember(
			ctx,
			state,
			rootID,
			member,
			from,
			relationshipPath,
			depth,
		)

	case declaration.MemberSelector:
		selector, selectorErr := r.expandSelector(
			ctx,
			state,
			rootID,
			member,
			from,
			depth,
		)
		if selectorErr == nil {
			relationship.Status = ResolutionAvailable
			relationship.Selector = &selector
			return relationship, nil
		}
		status, issue, partial := resolutionFailure(selectorErr)
		if !partial {
			return ResolvedRelationship{}, selectorErr
		}
		relationship.Status = status
		relationship.Issue = &issue
		return relationship, nil

	default:
		return ResolvedRelationship{}, fmt.Errorf(
			"%w: unsupported member form %q",
			basespec.ErrInvalid,
			form,
		)
	}

	if err == nil {
		relationship.Status = ResolutionAvailable
		relationship.Resolved = resolved
		return relationship, nil
	}

	status, issue, partial := resolutionFailure(err)
	if !partial {
		return ResolvedRelationship{}, err
	}
	relationship.Status = status
	relationship.Issue = &issue
	return relationship, nil
}

func (r *Resolver) resolveNamedMember(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	declarationType declaration.Type,
	name basespec.LogicalName,
	expectedVersion basespec.LogicalVersion,
	scope declaration.LookupScope,
	from *artifact.Artifact,
	depth int,
) (*ResolvedEntry, error) {
	if scope != declaration.LookupScopeBuiltin {
		value, found, err := r.resolveInRoot(
			ctx,
			state,
			rootID,
			declarationType,
			name,
			expectedVersion,
			depth,
		)
		if err != nil {
			return nil, err
		}
		if found {
			return value, nil
		}
	}

	if r.builtinRoot != "" &&
		(scope == declaration.LookupScopeBuiltin || r.builtinRoot != rootID) {
		value, found, err := r.resolveInRoot(
			ctx,
			state,
			r.builtinRoot,
			declarationType,
			name,
			expectedVersion,
			depth,
		)
		if err != nil {
			return nil, err
		}
		if found {
			return value, nil
		}
	}

	return r.resolveFallback(
		ctx,
		state,
		rootID,
		declarationType,
		name,
		expectedVersion,
		scope,
		from,
		depth,
	)
}

func (r *Resolver) resolveInRoot(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	declarationType declaration.Type,
	name basespec.LogicalName,
	expectedVersion basespec.LogicalVersion,
	depth int,
) (*ResolvedEntry, bool, error) {
	records, err := r.artifacts.FindByIdentity(
		ctx,
		rootID,
		artifact.ArtifactKind(declarationType),
		name,
	)
	if err != nil {
		return nil, false, err
	}

	terminals := make(map[artifact.ArtifactRef]struct{}, len(records))
	for _, record := range records {
		if record.State != artifact.StateAvailable {
			continue
		}
		if expectedVersion != "" &&
			record.LogicalVersion != expectedVersion {
			continue
		}

		terminal, err := r.resolveTerminalArtifact(
			ctx,
			record.Ref(),
			declarationType,
			expectedVersion,
		)
		if err != nil {
			status, _, partial := resolutionFailure(err)
			if partial && status == ResolutionUnavailable {
				continue
			}
			return nil, false, err
		}
		terminals[terminal] = struct{}{}
	}

	switch len(terminals) {
	case 0:
		return nil, false, nil
	case 1:
		var terminal artifact.ArtifactRef
		for value := range terminals {
			terminal = value
		}
		resolved, err := r.resolveArtifact(
			ctx,
			state,
			terminal,
			declarationType,
			expectedVersion,
			depth+1,
		)
		if err != nil {
			return nil, false, err
		}
		return resolved, true, nil
	default:
		return nil, false, fmt.Errorf(
			"%w: %s/%s resolves to %d terminal Artifacts in Root %q",
			basespec.ErrIdentityConflict,
			declarationType,
			name,
			len(terminals),
			rootID,
		)
	}
}

func (r *Resolver) resolveFallback(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	declarationType declaration.Type,
	name basespec.LogicalName,
	expectedVersion basespec.LogicalVersion,
	scope declaration.LookupScope,
	from *artifact.Artifact,
	depth int,
) (*ResolvedEntry, error) {
	typeResolver, found := r.registry.Resolver(declarationType)
	if !found || typeResolver.FallbackProvider() == nil {
		return nil, fmt.Errorf(
			"%w: %s/%s",
			basespec.ErrReferenceUnresolved,
			declarationType,
			name,
		)
	}

	target, found, err := typeResolver.FallbackProvider().ResolveFallback(
		ctx,
		FallbackRequest{
			RootID: rootID,
			Type:   declarationType,
			Name:   name,
			Scope:  scope,
		},
	)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf(
			"%w: %s/%s",
			basespec.ErrReferenceUnresolved,
			declarationType,
			name,
		)
	}
	if err := target.Validate(); err != nil {
		return nil, err
	}

	if target.Artifact != nil {
		if r.builtinRoot == "" || target.Artifact.RootID != r.builtinRoot {
			return nil, fmt.Errorf(
				"%w: Artifact-backed fallback must target the protected built-in Root",
				basespec.ErrInvalid,
			)
		}
		return r.resolveArtifact(
			ctx,
			state,
			*target.Artifact,
			declarationType,
			expectedVersion,
			depth+1,
		)
	}

	if !typeResolver.SupportsMappedFallbackTargets() {
		return nil, fmt.Errorf(
			"%w: Artifact type %q does not support mapped fallback targets",
			basespec.ErrUnsupported,
			declarationType,
		)
	}
	if target.Mapped.Type != declarationType ||
		target.Mapped.Name != name ||
		!target.Mapped.Builtin {
		return nil, fmt.Errorf(
			"%w: fallback target identity is invalid",
			basespec.ErrInvalid,
		)
	}

	return &ResolvedEntry{
		Type:              declarationType,
		scopeRootID:       rootID,
		DeclarationOrigin: cloneArtifactPointer(from),
		Mapped:            cloneMappedTarget(target.Mapped),
	}, nil
}

func (r *Resolver) resolveLocatedMember(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	member declaration.Entry,
	from *artifact.Artifact,
	expectedVersion basespec.LogicalVersion,
	depth int,
) (*ResolvedEntry, error) {
	if from == nil {
		return nil, fmt.Errorf(
			"%w: located member requires a source-backed declaration origin",
			basespec.ErrLocatorUnresolved,
		)
	}
	if r.locators == nil {
		return nil, fmt.Errorf(
			"%w: declaration locator requires a locator resolver",
			basespec.ErrLocatorUnresolved,
		)
	}

	header := member.Header()
	ref, err := r.locators.ResolveArtifactLocator(
		ctx,
		LocatorRequest{
			RootID:              rootID,
			From:                cloneArtifactPointer(from),
			Entry:               member.Clone(),
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
			"%w: located member resolver returned another Root",
			basespec.ErrInvalid,
		)
	}
	return r.resolveArtifact(
		ctx,
		state,
		ref,
		header.Type,
		expectedVersion,
		depth+1,
	)
}

func (r *Resolver) resolveContainedMember(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	member declaration.Entry,
	from *artifact.Artifact,
	relationshipPath []string,
	depth int,
) (*ResolvedEntry, error) {
	if from == nil {
		return nil, fmt.Errorf(
			"%w: contained declaration requires a source-backed parent",
			basespec.ErrReferenceUnresolved,
		)
	}

	// Parameters are opaque here. ContainedDeclaration only reconstructs the
	// declaration document by combining outer identity with parameters.
	target, err := member.ContainedDeclaration()
	if err != nil {
		return nil, err
	}
	targetRaw, err := target.CanonicalJSON()
	if err != nil {
		return nil, err
	}
	expectedVersion, err := memberTextLogicalVersion(member)
	if err != nil {
		return nil, err
	}
	subresource, err := containedMemberSubresource(
		*from,
		member,
		relationshipPath,
	)
	if err != nil {
		return nil, err
	}

	header := target.Header()
	records, err := r.artifacts.FindByIdentity(
		ctx,
		rootID,
		artifact.ArtifactKind(header.Type),
		basespec.LogicalName(header.Name),
	)
	if err != nil {
		return nil, err
	}

	matches := make([]artifact.Artifact, 0, 1)
	for _, record := range records {
		if record.State != artifact.StateAvailable ||
			record.Binding.SourceID != from.Binding.SourceID ||
			record.Binding.Locator != from.Binding.Locator ||
			record.Binding.SubresourceLocator != subresource {
			continue
		}
		if expectedVersion != "" &&
			record.LogicalVersion != expectedVersion {
			continue
		}
		definitionValue, err := r.artifacts.GetDefinition(ctx, record.Ref())
		if err != nil {
			return nil, err
		}
		if bytes.Equal(definitionValue.Body, targetRaw) {
			matches = append(matches, record)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf(
			"%w: contained declaration %s/%s has no matching source Artifact",
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
			expectedVersion,
			depth+1,
		)
	default:
		return nil, fmt.Errorf(
			"%w: contained declaration %s/%s has %d matching source Artifacts",
			basespec.ErrIdentityConflict,
			header.Type,
			header.Name,
			len(matches),
		)
	}
}

func containedMemberSubresource(
	parent artifact.Artifact,
	member declaration.Entry,
	relationshipPath []string,
) (basespec.SubresourceLocator, error) {
	header := member.Header()
	segments := append([]string(nil), relationshipPath...)
	if !isDirectProgramSlotMember(relationshipPath, header.Type) {
		segments = append(segments, string(header.Type))
	}
	if header.Type == declaration.TypeText {
		insert, err := member.TextInsert()
		if err != nil {
			return "", err
		}
		segments = append(segments, string(insert))
	}
	segments = append(segments, header.Name)

	value := basespec.SubresourceLocator(strings.Join(segments, "/"))
	if parent.Binding.SubresourceLocator != "" {
		value = parent.Binding.SubresourceLocator + "/" + value
	}
	if err := value.Validate(); err != nil {
		return "", err
	}
	return value, nil
}

func isDirectProgramSlotMember(
	relationshipPath []string,
	declarationType declaration.Type,
) bool {
	if len(relationshipPath) == 0 {
		return false
	}

	switch relationshipPath[len(relationshipPath)-1] {
	case loopStr:
		return declarationType == declaration.TypeLoop
	case workflowStr:
		return declarationType == declaration.TypeWorkflow
	default:
		return false
	}
}

func memberTextLogicalVersion(
	member declaration.Entry,
) (basespec.LogicalVersion, error) {
	if member.Header().Type != declaration.TypeText {
		return "", nil
	}
	insert, err := member.TextInsert()
	if err != nil {
		return "", err
	}
	return basespec.LogicalVersion(insert), nil
}

func resolutionFailure(
	err error,
) (ResolutionStatus, ResolutionIssue, bool) {
	issue := func(code string) ResolutionIssue {
		return ResolutionIssue{
			Code:    code,
			Message: diagnostic.BoundedMessage(err.Error()),
		}
	}

	switch {
	case errors.Is(err, basespec.ErrIdentityConflict):
		return ResolutionAmbiguous, issue("artifact.identity-conflict"), true
	case errors.Is(err, basespec.ErrLocatorLimitExceeded):
		return ResolutionUnavailable, issue("artifact.locator-limit-exceeded"), true
	case errors.Is(err, basespec.ErrLocatorUnresolved):
		return ResolutionUnavailable, issue("artifact.locator-unresolved"), true
	case errors.Is(err, basespec.ErrSourceUnavailable):
		return ResolutionUnavailable, issue("artifact.source-unavailable"), true
	case errors.Is(err, basespec.ErrRefreshRequired):
		return ResolutionUnavailable, issue("artifact.refresh-required"), true
	case errors.Is(err, basespec.ErrUnsupported):
		return ResolutionUnavailable, issue("artifact.selector-unavailable"), true
	case errors.Is(err, basespec.ErrReferenceUnresolved),
		errors.Is(err, basespec.ErrArtifactNotFound),
		errors.Is(err, basespec.ErrDefinitionNotFound),
		errors.Is(err, basespec.ErrSourceNotFound),
		errors.Is(err, basespec.ErrNotFound):
		return ResolutionUnavailable, issue("artifact.reference-unresolved"), true
	default:
		return "", ResolutionIssue{}, false
	}
}
