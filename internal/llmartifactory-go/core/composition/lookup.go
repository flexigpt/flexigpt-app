package composition

import (
	"context"
	"errors"
	"fmt"
	"strings"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func (r *Resolver) resolveRelationship(
	ctx context.Context,
	state *resolutionState,
	rootID rootModel.RootID,
	fact interpretation.Relationship,
	from *artifactModel.Artifact,
	depth int,
) (ResolvedRelationship, error) {
	member := fact.Declared.Clone()
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
		Path:      append([]string(nil), fact.Path...),
		Required:  fact.Required,
		Scope:     relationshipFields.Scope,
		Overrides: declaration.CloneRawMessageMap(relationshipFields.Overrides),
		Use:       declaration.CloneRawMessageMap(relationshipFields.Use),
	}

	if member.Header().Type == declaration.TypeWorkspace {
		relationship.Status = ResolutionUnavailable
		relationship.Issue = &ResolutionIssue{
			Code:    "artifact.reference.workspace-nested",
			Message: "Workspace cannot be resolved as a nested relationship",
		}
		return relationship, nil
	}

	var resolved *ResolvedEntry
	var errNew error
	switch form {
	case declaration.MemberNamed:
		header := member.Header()
		expectedVersion, err := memberTextLogicalVersion(member)
		if err != nil {
			return ResolvedRelationship{}, err
		}
		if header.Locator != nil {
			resolved, errNew = r.resolveLocatedMember(
				ctx,
				state,
				rootID,
				member,
				from,
				expectedVersion,
				depth,
			)
		} else {
			resolved, errNew = r.resolveNamedMember(
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
		resolved, errNew = r.resolveContainedMember(
			ctx,
			state,
			rootID,
			fact,
			from,
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

	if errNew == nil {
		relationship.Status = ResolutionAvailable
		relationship.Resolved = resolved
		return relationship, nil
	}

	status, issue, partial := resolutionFailure(errNew)
	if !partial {
		return ResolvedRelationship{}, errNew
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

	if r.scope.BuiltinRoot != "" &&
		(scope == declaration.LookupScopeBuiltin ||
			r.scope.BuiltinRoot != rootID) {
		value, found, err := r.resolveInRoot(
			ctx,
			state,
			r.scope.BuiltinRoot,
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

	return r.resolveDirectCapability(
		ctx,
		rootID,
		declarationType,
		name,
		scope,
		from,
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
		if r.scope.BuiltinRoot != "" {
			value, found, err := r.resolveInRoot(
				ctx,
				state,
				r.scope.BuiltinRoot,
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
		return r.resolveDirectCapability(
			ctx,
			rootID,
			declarationType,
			name,
			scope,
			from,
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

	terminals := make(map[artifactModel.ArtifactRef]struct{})
	for _, record := range records {
		if record.Binding.SourceID != state.compositionSourceID ||
			record.State != artifactModel.StateAvailable {
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
			return nil, err
		}
		terminals[terminal] = struct{}{}
	}

	switch len(terminals) {
	case 1:
		var terminal artifactModel.ArtifactRef
		for value := range terminals {
			terminal = value
		}
		return r.resolveArtifact(
			ctx,
			state,
			terminal,
			declarationType,
			expectedVersion,
			depth+1,
		)

	case 0:
		if r.scope.BuiltinRoot != "" &&
			r.scope.BuiltinRoot != rootID {
			value, found, err := r.resolveInRoot(
				ctx,
				state,
				r.scope.BuiltinRoot,
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
		return r.resolveDirectCapability(
			ctx,
			rootID,
			declarationType,
			name,
			scope,
			from,
		)

	default:
		return nil, fmt.Errorf(
			"%w: %s/%s resolves to %d terminal Artifacts in the composition Source",
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

func (r *Resolver) resolveDirectCapability(
	ctx context.Context,
	rootID rootModel.RootID,
	declarationType declaration.Type,
	name spec.LogicalName,
	scope declaration.LookupScope,
	from *artifactModel.Artifact,
) (*ResolvedEntry, error) {
	for _, provider := range r.directCapabilities {
		target, found, err := provider.ResolveDirectCapability(
			ctx,
			DirectCapabilityRequest{
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
			continue
		}
		if err := target.Validate(); err != nil {
			return nil, err
		}
		if target.Form != TargetFormDirect ||
			target.Type != declarationType ||
			target.Name != name ||
			target.Provenance != TargetProvenanceDirectCapability ||
			target.ProviderIdentity != provider.ProviderIdentity() {
			return nil, fmt.Errorf(
				"%w: direct capability target identity is invalid",
				spec.ErrInvalid,
			)
		}
		return &ResolvedEntry{
			Type:              declarationType,
			scopeRootID:       rootID,
			DeclarationOrigin: cloneArtifactPointer(from),
			Target:            pointerTarget(target),
		}, nil
	}

	return nil, fmt.Errorf(
		"%w: %s/%s",
		spec.ErrReferenceUnresolved,
		declarationType,
		name,
	)
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
	if r.locators == nil {
		return nil, fmt.Errorf(
			"%w: declaration locator requires a locator resolver",
			spec.ErrLocatorUnresolved,
		)
	}

	header := member.Header()
	if header.Locator == nil {
		return nil, fmt.Errorf(
			"%w: located member has no locator",
			spec.ErrInvalid,
		)
	}

	effectiveFrom := from
	if state.usesCompositionSource(rootID) {
		copyValue := from.Clone()
		copyValue.Binding.SourceID = state.compositionSourceID
		effectiveFrom = &copyValue
	}

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
			"%w: located member resolver returned an Artifact from another Root",
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
	fact interpretation.Relationship,
	from *artifactModel.Artifact,
	depth int,
) (*ResolvedEntry, error) {
	if from == nil {
		return nil, fmt.Errorf(
			"%w: contained declaration requires a source-backed parent",
			spec.ErrReferenceUnresolved,
		)
	}

	target, err := fact.Declared.ContainedDeclaration()
	if err != nil {
		return nil, err
	}
	expectedDefinition, err := r.interpretations.DefinitionForEntry(target)
	if err != nil {
		return nil, err
	}
	expectedVersion, err := memberTextLogicalVersion(fact.Declared)
	if err != nil {
		return nil, err
	}

	subresource, err := containedMemberSubresource(*from, fact)
	if err != nil {
		return nil, err
	}
	sourceID := from.Binding.SourceID
	if state.usesCompositionSource(rootID) {
		sourceID = state.compositionSourceID
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
			record.Binding.SourceID != sourceID ||
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
	fact interpretation.Relationship,
) (spec.SubresourceLocator, error) {
	value := spec.SubresourceLocator(
		strings.Join(fact.ContainedPath, "/"),
	)
	if parent.Binding.SubresourceLocator != "" {
		value = parent.Binding.SubresourceLocator + "/" + value
	}
	if err := value.Validate(); err != nil {
		return "", err
	}
	return value, nil
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
		return ResolutionUnavailable, issue("artifact.reference-invalid"), true

	case errors.Is(err, spec.ErrIdentityConflict):
		return ResolutionAmbiguous, issue("artifact.identity-conflict"), true

	case errors.Is(err, spec.ErrLocatorLimitExceeded):
		return ResolutionUnavailable, issue("artifact.locator-limit-exceeded"), true

	case errors.Is(err, spec.ErrLocatorUnresolved):
		return ResolutionUnavailable, issue("artifact.locator-unresolved"), true

	case errors.Is(err, spec.ErrSourceUnavailable):
		return ResolutionUnavailable, issue("artifact.source-unavailable"), true

	case errors.Is(err, spec.ErrRefreshRequired):
		return ResolutionUnavailable, issue("artifact.refresh-required"), true

	case errors.Is(err, spec.ErrUnsupported):
		return ResolutionUnavailable, issue("artifact.reference-unsupported"), true

	case errors.Is(err, spec.ErrReferenceUnresolved),
		errors.Is(err, spec.ErrArtifactNotFound),
		errors.Is(err, spec.ErrRootNotFound),
		errors.Is(err, spec.ErrDefinitionNotFound),
		errors.Is(err, spec.ErrSourceNotFound),
		errors.Is(err, spec.ErrNotFound):
		return ResolutionUnavailable, issue("artifact.reference-unresolved"), true

	default:
		return "", ResolutionIssue{}, false
	}
}
