package resolve

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
)

func (r *Resolver) resolveEntries(
	ctx context.Context,
	state *resolutionState,
	rootID rootModel.RootID,
	values []declaration.Entry,
	from *artifactModel.Artifact,
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
	rootID rootModel.RootID,
	member declaration.Entry,
	from *artifactModel.Artifact,
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
			Code:    "artifactModel.workspace-nested",
			Message: "Workspace cannot be resolved as a nested relationship",
		}
		return relationship, nil
	}

	var resolved *ResolvedEntry
	switch form {
	case declaration.MemberNamed:
		header := member.Header()
		var expectedVersion spec.LogicalVersion
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
				spec.LogicalName(header.Name),
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
			spec.ErrInvalid,
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
	rootID rootModel.RootID,
	declarationType declaration.Type,
	name spec.LogicalName,
	expectedVersion spec.LogicalVersion,
	scope declaration.LookupScope,
	from *artifactModel.Artifact,
	depth int,
) (*ResolvedEntry, error) {
	if state.usesCompositionSource(rootID) {
		return r.resolveNamedMemberSourceLocal(
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

func (r *Resolver) resolveNamedMemberSourceLocal(
	ctx context.Context,
	state *resolutionState,
	rootID rootModel.RootID,
	declarationType declaration.Type,
	name spec.LogicalName,
	expectedVersion spec.LogicalVersion,
	scope declaration.LookupScope,
	from *artifactModel.Artifact,
	depth int,
) (*ResolvedEntry, error) {
	if scope == declaration.LookupScopeBuiltin {
		if r.builtinRoot != "" {
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

	records, err := r.catalog.FindByIdentity(
		ctx,
		rootID,
		artifactModel.ArtifactKind(declarationType),
		name,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return nil, err
	}
	compositionRecords := make([]catalogModel.Entry, 0)
	for _, record := range records {
		if record.Binding.SourceID != state.compositionSourceID {
			continue
		}
		compositionRecords = append(compositionRecords, record)
	}
	if len(compositionRecords) == 0 {
		if r.builtinRoot != "" && r.builtinRoot != rootID {
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
		return r.resolveFallback(ctx, state, rootID, declarationType, name, expectedVersion, scope, from, depth)
	}
	terminals := make(map[artifactModel.ArtifactRef]struct{})
	for _, record := range compositionRecords {
		if record.State != artifactModel.StateAvailable {
			continue
		}
		if expectedVersion != "" && record.LogicalVersion != expectedVersion {
			continue
		}
		terminal, err := r.resolveTerminalArtifact(ctx, record.Ref(), declarationType, expectedVersion)
		if err != nil {
			status, _, partial := resolutionFailure(err)
			if partial && status == ResolutionUnavailable {
				continue
			}
			return nil, err
		}
		terminals[terminal] = struct{}{}
	}
	switch len(terminals) {
	case 0:
		return nil, fmt.Errorf(
			"%w: %s/%s is unavailable in composition source",
			spec.ErrReferenceUnresolved,
			declarationType,
			name,
		)
	case 1:
		var terminal artifactModel.ArtifactRef
		for value := range terminals {
			terminal = value
		}
		return r.resolveArtifact(ctx, state, terminal, declarationType, expectedVersion, depth+1)
	default:
		return nil, fmt.Errorf(
			"%w: %s/%s resolves to %d terminal Artifacts",
			spec.ErrIdentityConflict,
			declarationType,
			name,
			len(terminals),
		)
	}
}

func (r *Resolver) resolveInRoot(
	ctx context.Context,
	state *resolutionState,
	rootID rootModel.RootID,
	declarationType declaration.Type,
	name spec.LogicalName,
	expectedVersion spec.LogicalVersion,
	depth int,
) (*ResolvedEntry, bool, error) {
	records, err := r.catalog.FindByIdentity(
		ctx,
		rootID,
		artifactModel.ArtifactKind(declarationType),
		name,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return nil, false, err
	}

	terminals := make(map[artifactModel.ArtifactRef]struct{}, len(records))
	for _, record := range records {
		if record.State != artifactModel.StateAvailable {
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
		var terminal artifactModel.ArtifactRef
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
			spec.ErrIdentityConflict,
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
	rootID rootModel.RootID,
	declarationType declaration.Type,
	name spec.LogicalName,
	expectedVersion spec.LogicalVersion,
	scope declaration.LookupScope,
	from *artifactModel.Artifact,
	depth int,
) (*ResolvedEntry, error) {
	typeResolver, found := r.registry.Resolver(declarationType)
	if !found || typeResolver.FallbackProvider() == nil {
		return nil, fmt.Errorf(
			"%w: %s/%s",
			spec.ErrReferenceUnresolved,
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
			spec.ErrReferenceUnresolved,
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
				spec.ErrInvalid,
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
			spec.ErrUnsupported,
			declarationType,
		)
	}
	if target.Mapped.Type != declarationType ||
		target.Mapped.Name != name ||
		!target.Mapped.Builtin {
		return nil, fmt.Errorf(
			"%w: fallback target identity is invalid",
			spec.ErrInvalid,
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
	rootID rootModel.RootID,
	member declaration.Entry,
	from *artifactModel.Artifact,
	expectedVersion spec.LogicalVersion,
	depth int,
) (*ResolvedEntry, error) {
	if from == nil {
		return nil, fmt.Errorf(
			"%w: located member requires a source-backed declaration origin",
			spec.ErrLocatorUnresolved,
		)
	}
	effectiveFrom := from
	if state.usesCompositionSource(rootID) {
		copied := from.Clone()
		copied.Binding.SourceID = state.compositionSourceID
		effectiveFrom = &copied
	}
	if r.locators == nil {
		return nil, fmt.Errorf(
			"%w: declaration locator requires a locator resolver",
			spec.ErrLocatorUnresolved,
		)
	}

	header := member.Header()
	ref, err := r.locators.ResolveArtifactLocator(
		ctx,
		LocatorRequest{
			RootID:              rootID,
			From:                cloneArtifactPointer(effectiveFrom),
			Entry:               member.Clone(),
			Locator:             header.Locator.Clone(),
			ExpectedType:        header.Type,
			ExpectedLogicalName: spec.LogicalName(header.Name),
		},
	)
	if err != nil {
		return nil, err
	}
	if ref.RootID != rootID {
		return nil, fmt.Errorf(
			"%w: located member resolver returned another Root",
			spec.ErrInvalid,
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
	rootID rootModel.RootID,
	member declaration.Entry,
	from *artifactModel.Artifact,
	relationshipPath []string,
	depth int,
) (*ResolvedEntry, error) {
	if from == nil {
		return nil, fmt.Errorf(
			"%w: contained declaration requires a source-backed parent",
			spec.ErrReferenceUnresolved,
		)
	}

	// Parameters are opaque here. ContainedDeclaration only reconstructs the
	// declaration document by combining outer identity with parameters.
	target, err := member.ContainedDeclaration()
	if err != nil {
		return nil, err
	}
	expectedDefinition, err := decoder.DefinitionForEntry(target)
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
	records, err := r.catalog.FindByIdentity(
		ctx,
		rootID,
		artifactModel.ArtifactKind(header.Type),
		spec.LogicalName(header.Name),
		catalogModel.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	matches := make([]artifactModel.ArtifactRef, 0, 1)
	for _, record := range records {
		if record.State != artifactModel.StateAvailable ||
			record.Binding.SourceID != from.Binding.SourceID ||
			record.Binding.Locator != from.Binding.Locator ||
			record.Binding.SubresourceLocator != subresource {
			continue
		}
		if expectedVersion != "" &&
			record.LogicalVersion != expectedVersion {
			continue
		}
		if record.Definition == nil ||
			record.Definition.Digest != expectedDefinition.Digest {
			continue
		}

		matches = append(matches, record.Ref())
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf(
			"%w: contained declaration %s/%s has no matching source Artifact",
			spec.ErrReferenceUnresolved,
			header.Type,
			header.Name,
		)
	case 1:
		return r.resolveArtifact(
			ctx,
			state,
			matches[0],
			header.Type,
			expectedVersion,
			depth+1,
		)
	default:
		return nil, fmt.Errorf(
			"%w: contained declaration %s/%s has %d matching source Artifacts",
			spec.ErrIdentityConflict,
			header.Type,
			header.Name,
			len(matches),
		)
	}
}

func containedMemberSubresource(
	parent artifactModel.Artifact,
	member declaration.Entry,
	relationshipPath []string,
) (spec.SubresourceLocator, error) {
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

	value := spec.SubresourceLocator(strings.Join(segments, "/"))
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
) (spec.LogicalVersion, error) {
	if member.Header().Type != declaration.TypeText {
		return "", nil
	}
	insert, err := member.TextInsert()
	if err != nil {
		return "", err
	}
	return spec.LogicalVersion(insert), nil
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
	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, spec.ErrClosed):
		return "", ResolutionIssue{}, false
	case errors.Is(err, spec.ErrDigestMismatch),
		errors.Is(err, spec.ErrInvalid):
		return ResolutionUnavailable, issue("artifactModel.reference-invalid"), true
	case errors.Is(err, spec.ErrIdentityConflict):
		return ResolutionAmbiguous, issue("artifactModel.identity-conflict"), true
	case errors.Is(err, spec.ErrLocatorLimitExceeded):
		return ResolutionUnavailable, issue("artifactModel.locator-limit-exceeded"), true
	case errors.Is(err, spec.ErrLocatorUnresolved):
		return ResolutionUnavailable, issue("artifactModel.locator-unresolved"), true
	case errors.Is(err, spec.ErrSourceUnavailable):
		return ResolutionUnavailable, issue("artifactModel.source-unavailable"), true
	case errors.Is(err, spec.ErrRefreshRequired):
		return ResolutionUnavailable, issue("artifactModel.refresh-required"), true
	case errors.Is(err, spec.ErrUnsupported):
		return ResolutionUnavailable, issue("artifactModel.selector-unavailable"), true
	case errors.Is(err, spec.ErrReferenceUnresolved),
		errors.Is(err, spec.ErrArtifactNotFound),
		errors.Is(err, spec.ErrRootNotFound),
		errors.Is(err, spec.ErrDefinitionNotFound),
		errors.Is(err, spec.ErrSourceNotFound),
		errors.Is(err, spec.ErrNotFound):
		return ResolutionUnavailable, issue("artifactModel.reference-unresolved"), true
	default:
		return "", ResolutionIssue{}, false
	}
}
