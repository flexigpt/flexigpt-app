// Package plugin owns managed Plugin declarations and direct membership.
//
// Plugin membership remains declaration content. This package does not
// create Artifact Store ownership, foreign keys, or lifecycle relationships.
package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	managepackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/model"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
	pluginDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/domain"
)

const (
	ManagedCollectionPackageKind managedpackageModel.PackageKind = "plugin"
)

type API struct {
	artifacts        artifact.API
	cat              catalog.API
	sources          source.API
	discovery        refreshFlow.API
	definitions      definition.API
	managedArtifacts managepackageFlow.API

	domain   *Profile
	resolver *composition.Resolver

	// Immutable Plugin projections keyed by Root-local Definition digest.
	catalogProjections DocumentCache[collectionProjection]
}

func New(
	artifacts artifact.API,
	cat catalog.API,
	sources source.API,
	discovery refreshFlow.API,
	managedArtifacts managepackageFlow.API,
	definitions definition.API,
	resolver *composition.Resolver,
	profiles ...Profile,
) (*API, error) {
	if artifacts == nil ||
		cat == nil ||
		sources == nil ||
		discovery == nil ||
		definitions == nil ||
		managedArtifacts == nil {
		return nil, fmt.Errorf(
			"%w: managed Plugin dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	if len(profiles) > 1 {
		return nil, fmt.Errorf(
			"%w: Plugin API has multiple domain policies",
			spec.ErrInvalid,
		)
	}
	output := &API{
		artifacts:        artifacts,
		cat:              cat,
		sources:          sources,
		discovery:        discovery,
		managedArtifacts: managedArtifacts,
		definitions:      definitions,
		resolver:         resolver,
	}
	if len(profiles) == 1 {
		value := profiles[0]
		if err := value.Validate(); err != nil {
			return nil, err
		}
		value.MembershipPolicy.AllowedTypes = append([]declaration.Type(nil), value.MembershipPolicy.AllowedTypes...)
		value.MembershipPolicy.AllowedForms = append(
			[]declaration.MemberForm(nil),
			value.MembershipPolicy.AllowedForms...,
		)
		output.domain = &value
	}
	return output, nil
}

type PluginView struct {
	Artifact    artifactModel.Artifact        `json:"artifact"`
	Name        spec.LogicalName              `json:"name"`
	DisplayName string                        `json:"displayName"`
	Description string                        `json:"description,omitempty"`
	Members     []MemberReference             `json:"members"`
	Entries     []PluginMemberView            `json:"entries"`
	Editable    bool                          `json:"editable"`
	Deletable   bool                          `json:"deletable"`
	Baseline    bool                          `json:"baseline"`
	Membership  pluginDomain.MembershipPolicy `json:"membership"`
}

type PluginMemberView struct {
	Type      declaration.Type         `json:"type"`
	Name      spec.LogicalName         `json:"name,omitempty"`
	Insert    declaration.InsertTarget `json:"insert,omitempty"`
	Locator   *declaration.Locator     `json:"locator,omitempty"`
	Server    spec.LogicalName         `json:"server,omitempty"`
	Contained bool                     `json:"contained"`
	Selector  bool                     `json:"selector"`
}

// MemberReference is an external Plugin membership edge. It intentionally
// cannot express a contained declaration.
type MemberReference struct {
	Type    declaration.Type         `json:"type"`
	Name    spec.LogicalName         `json:"name"`
	Insert  declaration.InsertTarget `json:"insert,omitempty"`
	Locator *declaration.Locator     `json:"locator,omitempty"`
	Scope   declaration.LookupScope  `json:"scope,omitempty"`
	Server  spec.LogicalName         `json:"server,omitempty"`
}

type CreateRequest struct {
	RootID      rootModel.RootID     `json:"rootID"`
	SourceID    sourceModel.SourceID `json:"sourceID,omitempty"`
	Name        spec.LogicalName     `json:"name"`
	DisplayName string               `json:"displayName,omitempty"`
	Description string               `json:"description,omitempty"`
}

type UpdateRequest struct {
	Plugin           artifactModel.ArtifactRef `json:"plugin"`
	ExpectedRevision uint64                    `json:"expectedRevision"`
	Description      string                    `json:"description,omitempty"`
	DisplayName      string                    `json:"displayName,omitempty"`
}

type AddMemberRequest struct {
	Plugin           artifactModel.ArtifactRef `json:"plugin"`
	ExpectedRevision uint64                    `json:"expectedRevision"`
	Member           MemberReference           `json:"member"`
}

type AddEntryRequest struct {
	Plugin           artifactModel.ArtifactRef `json:"plugin"`
	ExpectedRevision uint64                    `json:"expectedRevision"`
	Entry            declaration.Entry         `json:"entry"`
}

type AddArtifactMemberRequest struct {
	Plugin           artifactModel.ArtifactRef `json:"plugin"`
	ExpectedRevision uint64                    `json:"expectedRevision"`
	Artifact         artifactModel.ArtifactRef `json:"artifact"`
}

type RemoveMemberRequest struct {
	Plugin           artifactModel.ArtifactRef `json:"plugin"`
	ExpectedRevision uint64                    `json:"expectedRevision"`
	Index            int                       `json:"index"`
}

type DeleteRequest struct {
	Plugin           artifactModel.ArtifactRef `json:"plugin"`
	ExpectedRevision uint64                    `json:"expectedRevision"`
}

// MemberMutationResult is used by managed Skill, MCP, and policy creation.
// Created reports whether this call appended the member rather than finding an
// identical existing membership.
type MemberMutationResult struct {
	Plugin  PluginView `json:"plugin"`
	Index   int        `json:"index"`
	Created bool       `json:"created"`
}

type editableCollection struct {
	artifact   artifactModel.Artifact
	document   pluginv1.PluginDocument
	address    managedpackageModel.ManagedPackageAddress
	generation string
}

func (a *API) Create(
	ctx context.Context,
	request CreateRequest,
) (PluginView, error) {
	return a.create(ctx, request, false)
}

func (a *API) Get(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (PluginView, error) {
	if a == nil {
		return PluginView{}, spec.ErrClosed
	}
	value, err := a.loadEditableCollection(ctx, ref, 0)
	if err != nil {
		return PluginView{}, err
	}
	return a.collectionViewOf(value.artifact, value.document)
}

// SetEnabled changes the generic local enablement metadata of a Plugin.
//
// It is intentionally independent of Plugin editability, membership,
// deletion eligibility, source ownership, and Root protection. A disabled
// Plugin remains readable and editable when it was otherwise editable.
func (a *API) SetEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (PluginView, error) {
	if a == nil {
		return PluginView{}, spec.ErrClosed
	}
	if expectedRevision == 0 {
		return PluginView{}, fmt.Errorf(
			"%w: expected Plugin revision is required",
			spec.ErrInvalid,
		)
	}

	view, err := a.Read(ctx, ref)
	if err != nil {
		return PluginView{}, err
	}
	if view.Artifact.Revision != expectedRevision {
		return PluginView{}, spec.ErrConflict
	}

	updated, err := a.artifacts.SetEnabled(
		ctx,
		view.Artifact.Ref(),
		expectedRevision,
		enabled,
	)
	if err != nil {
		return PluginView{}, err
	}
	view.Artifact = updated.Clone()
	return view, nil
}

func (a *API) List(
	ctx context.Context,
	request ListRequest,
) ([]ListItem, error) {
	return a.listCollections(ctx, request, false)
}

func (a *API) Update(
	ctx context.Context,
	request UpdateRequest,
) (PluginView, error) {
	if a == nil {
		return PluginView{}, spec.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return PluginView{}, fmt.Errorf(
			"%w: expected Plugin revision is required",
			spec.ErrInvalid,
		)
	}

	value, err := a.loadEditableCollection(
		ctx,
		request.Plugin,
		request.ExpectedRevision,
	)
	if err != nil {
		return PluginView{}, err
	}
	if _, err := a.collectionViewOf(value.artifact, value.document); err != nil {
		return PluginView{}, err
	}

	if request.DisplayName != "" {
		value.document.DisplayName = request.DisplayName
	}

	value.document.Description = request.Description
	record, err := a.publishDocument(
		ctx,
		value.artifact.RootID,
		value.artifact.Binding.SourceID,
		value.address,
		value.document,
		value.generation,
		false,
	)
	if err != nil {
		return PluginView{}, err
	}
	return a.collectionViewOf(record, value.document)
}

func (a *API) AddMember(
	ctx context.Context,
	request AddMemberRequest,
) (PluginView, error) {
	result, err := a.mutateMember(ctx, request, false)
	if err != nil {
		return PluginView{}, err
	}
	return result.Plugin, nil
}

func (a *API) EnsureMember(
	ctx context.Context,
	request AddMemberRequest,
) (MemberMutationResult, error) {
	return a.mutateMember(ctx, request, true)
}

func (a *API) RemoveMember(
	ctx context.Context,
	request RemoveMemberRequest,
) (PluginView, error) {
	if a == nil {
		return PluginView{}, spec.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return PluginView{}, fmt.Errorf(
			"%w: expected Plugin revision is required",
			spec.ErrInvalid,
		)
	}

	value, err := a.loadEditableCollection(
		ctx,
		request.Plugin,
		request.ExpectedRevision,
	)
	if err != nil {
		return PluginView{}, err
	}
	if _, err := a.collectionViewOf(value.artifact, value.document); err != nil {
		return PluginView{}, err
	}
	if request.Index < 0 || request.Index >= len(value.document.Members) {
		return PluginView{}, fmt.Errorf(
			"%w: Plugin member index %d is out of range",
			spec.ErrNotFound,
			request.Index,
		)
	}
	normalized, err := normalizedMemberIndexes(value.document.Members)
	if err != nil {
		return PluginView{}, err
	}
	sourceIndex := normalized[request.Index]

	members := append(
		[]declaration.Entry(nil),
		value.document.Members[:sourceIndex]...,
	)
	members = append(
		members,
		value.document.Members[sourceIndex+1:]...,
	)
	value.document.Members = members

	record, err := a.publishDocument(
		ctx,
		value.artifact.RootID,
		value.artifact.Binding.SourceID,
		value.address,
		value.document,
		value.generation,
		false,
	)
	if err != nil {
		return PluginView{}, err
	}
	return a.collectionViewOf(record, value.document)
}

func (a *API) AddEntry(
	ctx context.Context,
	request AddEntryRequest,
) (PluginView, error) {
	result, err := a.mutateEntry(ctx, request, false)
	if err != nil {
		return PluginView{}, err
	}
	return result.Plugin, nil
}

func (a *API) EnsureEntry(
	ctx context.Context,
	request AddEntryRequest,
) (MemberMutationResult, error) {
	return a.mutateEntry(ctx, request, true)
}

func (a *API) AddArtifactMember(
	ctx context.Context,
	request AddArtifactMemberRequest,
) (PluginView, error) {
	if a == nil {
		return PluginView{}, spec.ErrClosed
	}
	if err := request.Plugin.Validate(); err != nil {
		return PluginView{}, err
	}
	if err := request.Artifact.Validate(); err != nil {
		return PluginView{}, err
	}
	if request.Plugin.RootID != request.Artifact.RootID {
		return PluginView{}, fmt.Errorf(
			"%w: Plugin member Artifact belongs to another Root",
			spec.ErrInvalid,
		)
	}

	target, err := a.artifacts.Get(ctx, request.Artifact)
	if err != nil {
		return PluginView{}, err
	}
	declarationType := declaration.Type(target.Kind)
	if err := declarationType.Validate(); err != nil {
		return PluginView{}, err
	}

	member := MemberReference{
		Type: declarationType,
		Name: target.LogicalName,
	}
	collectionValue, err := a.loadEditableCollection(
		ctx,
		request.Plugin,
		request.ExpectedRevision,
	)
	if err != nil {
		return PluginView{}, err
	}
	if target.Binding.SourceID == collectionValue.artifact.Binding.SourceID {
		locator, err := relativeSourceLocator(
			collectionValue.artifact.Binding.Locator,
			target.Binding.Locator,
		)
		if err != nil {
			return PluginView{}, err
		}
		member.Locator = &locator
	}

	return a.AddMember(
		ctx,
		AddMemberRequest{
			Plugin:           request.Plugin,
			ExpectedRevision: request.ExpectedRevision,
			Member:           member,
		},
	)
}

// MemberForCollectionSource builds a location-constrained external reference
// for a declaration that will be published into the same managed Source as
// the Plugin.
func (a *API) MemberForCollectionSource(
	ctx context.Context,
	collectionRef artifactModel.ArtifactRef,
	declarationType declaration.Type,
	name spec.LogicalName,
	target spec.Locator,
) (MemberReference, error) {
	if a == nil {
		return MemberReference{}, spec.ErrClosed
	}
	if err := declarationType.Validate(); err != nil {
		return MemberReference{}, err
	}
	if err := a.validateDomainMember(MemberReference{
		Type: declarationType,
	}); err != nil {
		return MemberReference{}, err
	}
	if err := name.Validate(); err != nil {
		return MemberReference{}, err
	}
	if err := target.Validate(false); err != nil {
		return MemberReference{}, err
	}

	value, err := a.loadEditableCollection(ctx, collectionRef, 0)
	if err != nil {
		return MemberReference{}, err
	}
	locator, err := relativeSourceLocator(
		value.artifact.Binding.Locator,
		target,
	)
	if err != nil {
		return MemberReference{}, err
	}
	return MemberReference{
		Type:    declarationType,
		Name:    name,
		Locator: &locator,
	}, nil
}

func (a *API) Delete(
	ctx context.Context,
	request DeleteRequest,
) error {
	if a == nil {
		return spec.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Plugin revision is required",
			spec.ErrInvalid,
		)
	}

	value, err := a.loadEditableCollection(
		ctx,
		request.Plugin,
		request.ExpectedRevision,
	)
	if err != nil {
		return err
	}
	if a.isBaselineEditableCollection(value) {
		return fmt.Errorf(
			"%w: baseline Plugin %q cannot be deleted",
			spec.ErrProtected,
			value.artifact.LogicalName,
		)
	}
	if len(value.document.Members) != 0 {
		return fmt.Errorf(
			"%w: Plugin %q has %d direct members; detach them before deletion",
			spec.ErrConflict,
			value.artifact.LogicalName,
			len(value.document.Members),
		)
	}

	removeRequest := managepackageModel.RemoveRequest{
		RootID:             value.artifact.RootID,
		SourceID:           value.artifact.Binding.SourceID,
		Package:            value.address,
		ExpectedArtifact:   &request.Plugin,
		ExpectedGeneration: value.generation,
	}
	if a.domain != nil {
		locator := value.artifact.Binding.Locator
		removeRequest.PruneDiscoveryLocator = &locator
	}
	if err := a.managedArtifacts.Remove(ctx, removeRequest); err != nil {
		return err
	}

	missing, err := a.artifacts.Get(ctx, request.Plugin)
	if err != nil {
		return err
	}
	if missing.State != artifactModel.StateMissing {
		return fmt.Errorf(
			"%w: removed Plugin Artifact is not missing",
			spec.ErrConflict,
		)
	}

	return a.artifacts.Purge(
		ctx,
		request.Plugin,
		missing.Revision,
	)
}

func (a *API) resolveCollectionRef(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (artifactModel.ArtifactRef, error) {
	if err := ref.Validate(); err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	if a == nil || a.resolver == nil {
		return ref, nil
	}
	resolved, err := a.resolver.ResolvePlugin(ctx, ref)
	if err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	terminal, found := resolved.ArtifactRef()
	if !found {
		return artifactModel.ArtifactRef{}, fmt.Errorf(
			"%w: plugin did not resolve to a source-backed Artifact",
			spec.ErrReferenceUnresolved,
		)
	}
	return terminal, nil
}

func (a *API) create(
	ctx context.Context,
	request CreateRequest,
	allowBaseline bool,
) (PluginView, error) {
	if a == nil {
		return PluginView{}, spec.ErrClosed
	}
	if err := a.requireDeclarationAuthoring(); err != nil {
		return PluginView{}, err
	}
	if err := request.RootID.Validate(); err != nil {
		return PluginView{}, err
	}
	if err := request.Name.Validate(); err != nil {
		return PluginView{}, err
	}
	if a.domain != nil &&
		request.Name == a.domain.BaselineName &&
		!allowBaseline {
		return PluginView{}, fmt.Errorf(
			"%w: baseline Plugin %q is application-provisioned",
			spec.ErrProtected,
			request.Name,
		)
	}

	sourceValue, err := a.managedSource(
		ctx,
		request.RootID,
		request.SourceID,
	)
	if err != nil {
		return PluginView{}, err
	}
	address, err := a.managedCollectionAddress(request.Name)
	if err != nil {
		return PluginView{}, err
	}
	locator, err := address.FileLocator(a.managedCollectionDocumentFile())
	if err != nil {
		return PluginView{}, err
	}

	document := pluginv1.PluginDocument{
		Type:        pluginv1.PluginType,
		Name:        string(request.Name),
		DisplayName: request.DisplayName,
		Description: request.Description,
	}
	if a.domain != nil {
		document.Metadata, err = pluginDomain.PutMetadata(
			document.Metadata,
			a.domain.MembershipPolicy,
		)
		if err != nil {
			return PluginView{}, err
		}
	}
	_, digest, err := collectionDocumentPayload(document)
	if err != nil {
		return PluginView{}, err
	}

	existing, err := a.artifacts.FindByOrigin(
		ctx,
		request.RootID,
		artifactModel.SourceBinding{
			SourceID: sourceValue.ID,
			Locator:  locator,
		},
		artifactModel.ArtifactKind(pluginv1.PluginType),
	)
	switch {
	case err == nil:
		if existing.State == artifactModel.StateAvailable &&
			existing.ResolvedDefinition != nil &&
			*existing.ResolvedDefinition == digest {
			return a.Get(ctx, existing.Ref())
		}
		return PluginView{}, fmt.Errorf(
			"%w: managed Plugin %q already exists",
			spec.ErrConflict,
			request.Name,
		)

	case errors.Is(err, spec.ErrArtifactNotFound),
		errors.Is(err, spec.ErrNotFound):
	default:
		return PluginView{}, err
	}

	record, err := a.publishDocument(
		ctx,
		request.RootID,
		sourceValue.ID,
		address,
		document,
		"",
		false,
	)
	if err != nil {
		return PluginView{}, err
	}
	return a.collectionViewOf(record, document)
}

func (a *API) mutateMember(
	ctx context.Context,
	request AddMemberRequest,
	ensure bool,
) (MemberMutationResult, error) {
	if a == nil {
		return MemberMutationResult{}, spec.ErrClosed
	}

	if err := a.validateDomainMember(request.Member); err != nil {
		return MemberMutationResult{}, err
	}
	member, err := request.Member.entry()
	if err != nil {
		return MemberMutationResult{}, err
	}
	return a.mutateEntry(
		ctx,
		AddEntryRequest{
			Plugin:           request.Plugin,
			ExpectedRevision: request.ExpectedRevision,
			Entry:            member,
		},
		ensure,
	)
}

func (a *API) mutateEntry(
	ctx context.Context,
	request AddEntryRequest,
	ensure bool,
) (MemberMutationResult, error) {
	if a == nil {
		return MemberMutationResult{}, spec.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return MemberMutationResult{}, fmt.Errorf(
			"%w: expected Plugin revision is required",
			spec.ErrInvalid,
		)
	}
	if err := a.validateDomainEntry(request.Entry); err != nil {
		return MemberMutationResult{}, err
	}
	member := request.Entry.Clone()
	_, err := member.CanonicalJSON()
	if err != nil {
		return MemberMutationResult{}, err
	}

	value, err := a.loadEditableCollection(
		ctx,
		request.Plugin,
		request.ExpectedRevision,
	)
	if err != nil {
		return MemberMutationResult{}, err
	}
	view, err := a.collectionViewOf(value.artifact, value.document)
	if err != nil {
		return MemberMutationResult{}, err
	}

	if ensure {
		normalized, err := normalizedMemberIndexes(
			value.document.Members,
		)
		if err != nil {
			return MemberMutationResult{}, err
		}
		memberIdentity, err := declaration.MemberIdentityJSON(member)
		if err != nil {
			return MemberMutationResult{}, err
		}
		for normalizedIndex, sourceIndex := range normalized {
			current := value.document.Members[sourceIndex]
			currentIdentity, err := declaration.MemberIdentityJSON(current)
			if err != nil {
				return MemberMutationResult{}, err
			}
			if bytes.Equal(currentIdentity, memberIdentity) {
				return MemberMutationResult{
					Plugin:  view,
					Index:   normalizedIndex,
					Created: false,
				}, nil
			}
		}
	}

	value.document.Members = append(
		append([]declaration.Entry(nil), value.document.Members...),
		member,
	)
	policy, err := pluginDomain.FromMetadata(value.document.Metadata)
	if err != nil {
		return MemberMutationResult{}, err
	}
	if err := policy.Allows(member); err != nil {
		return MemberMutationResult{}, err
	}

	normalizedIndex, err := normalizedMemberIndex(
		value.document.Members,
		len(value.document.Members)-1,
	)
	if err != nil {
		return MemberMutationResult{}, err
	}
	record, err := a.publishDocument(
		ctx,
		value.artifact.RootID,
		value.artifact.Binding.SourceID,
		value.address,
		value.document,
		value.generation,
		false,
	)
	if err != nil {
		return MemberMutationResult{}, err
	}
	view, err = a.collectionViewOf(record, value.document)
	if err != nil {
		return MemberMutationResult{}, err
	}
	return MemberMutationResult{
		Plugin:  view,
		Index:   normalizedIndex,
		Created: true,
	}, nil
}

func (a *API) managedSource(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) (sourceModel.Summary, error) {
	if a.domain != nil {
		return a.domainManagedSource(ctx, rootID, sourceID)
	}

	value, err := a.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	if value.Kind != managedfs.Kind {
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: editable Plugin Source must have kind %q",
			spec.ErrUnsupported,
			managedfs.Kind,
		)
	}
	if !value.Enabled {
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: editable Plugin Source is disabled",
			spec.ErrConflict,
		)
	}
	return value, nil
}

func (a *API) publishDocument(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	address managedpackageModel.ManagedPackageAddress,
	document pluginv1.PluginDocument,
	expectedGeneration string,
	allowPackageReplacement bool,
) (artifactModel.Artifact, error) {
	documentFile := a.managedCollectionDocumentFile()
	decoderID, err := a.managedCollectionDecoderID()
	if err != nil {
		return artifactModel.Artifact{}, err
	}

	raw, digest, err := collectionDocumentPayload(document)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	locator, err := address.FileLocator(documentFile)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if _, err := a.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceID,
		locator,
		decoderID,
	); err != nil {
		return artifactModel.Artifact{}, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		managepackageModel.PublishRequest{
			RootID: rootID,
			Binding: artifactModel.SourceBinding{
				SourceID: sourceID,
				Locator:  locator,
			},
			ExpectedKind: artifactModel.ArtifactKind(
				pluginv1.PluginType,
			),
			ExpectedLogicalName: spec.LogicalName(document.Name),
			ExpectedDefinition:  digest,
			Package: managedpackageModel.ManagedPackagePublication{
				Address:            address,
				ExpectedGeneration: expectedGeneration,
				Files: []managedpackageModel.ManagedPackageFile{{
					Locator: documentFile,
					Content: raw,
				}},
			},
			AllowPackageReplacement: allowPackageReplacement,
		},
	)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	return published.Artifact, nil
}

func (a *API) loadEditableCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) (editableCollection, error) {
	if err := a.requireDeclarationAuthoring(); err != nil {
		return editableCollection{}, err
	}
	if err := ref.Validate(); err != nil {
		return editableCollection{}, err
	}

	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return editableCollection{}, err
	}
	if record.Kind != artifactModel.ArtifactKind(pluginv1.PluginType) {
		return editableCollection{}, fmt.Errorf(
			"%w: Artifact %q is not a Plugin",
			spec.ErrUnsupported,
			record.ID,
		)
	}
	if expectedRevision != 0 && record.Revision != expectedRevision {
		return editableCollection{}, spec.ErrConflict
	}
	if record.State != artifactModel.StateAvailable {
		return editableCollection{}, fmt.Errorf(
			"%w: Plugin Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if record.Binding.SubresourceLocator != "" {
		return editableCollection{}, fmt.Errorf(
			"%w: contained Plugin declarations are not editable managed Collections",
			spec.ErrUnsupported,
		)
	}

	sourceValue, err := a.sources.Get(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return editableCollection{}, err
	}
	if sourceValue.Kind != managedfs.Kind {
		return editableCollection{}, fmt.Errorf(
			"%w: Plugin is not backed by a managed Source",
			spec.ErrUnsupported,
		)
	}
	if !sourceValue.Enabled {
		return editableCollection{}, fmt.Errorf(
			"%w: Plugin Source is disabled",
			spec.ErrReferenceUnresolved,
		)
	}
	if a.domain != nil &&
		sourceValue.StorageKey != a.domain.SourceStorageKey {
		return editableCollection{}, fmt.Errorf(
			"%w: Plugin belongs to another managed domain Source",
			spec.ErrReferenceUnresolved,
		)
	}

	address, err := a.managedCollectionAddressFromLocator(
		record.Binding.Locator,
	)
	if err != nil {
		return editableCollection{}, err
	}
	if address.Name != record.LogicalName {
		return editableCollection{}, fmt.Errorf(
			"%w: managed Plugin package name does not match Artifact identity",
			spec.ErrInvalid,
		)
	}

	inspection, err := a.discovery.InspectSource(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return editableCollection{}, err
	}
	if !inspection.IsCurrent() {
		return editableCollection{}, fmt.Errorf(
			"%w: managed Plugin Source requires refresh",
			spec.ErrRefreshRequired,
		)
	}

	definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return editableCollection{}, err
	}
	document, err := pluginv1.FromDefinition(
		definitionValue,
	)
	if err != nil {
		return editableCollection{}, err
	}
	if document.Name != string(record.LogicalName) {
		return editableCollection{}, fmt.Errorf(
			"%w: Plugin declaration name differs from Artifact identity",
			spec.ErrInvalid,
		)
	}
	if document.Locator != nil {
		return editableCollection{}, fmt.Errorf(
			"%w: located Plugin aliases are not editable managed Collections",
			spec.ErrUnsupported,
		)
	}
	if err := a.validateEditableDomainDocument(document); err != nil {
		return editableCollection{}, err
	}
	return editableCollection{
		artifact:   record,
		document:   document,
		address:    address,
		generation: inspection.State.SourceGeneration,
	}, nil
}

func (a *API) managedCollectionAddress(
	name spec.LogicalName,
) (managedpackageModel.ManagedPackageAddress, error) {
	return managedCollectionAddressFor(
		a.managedCollectionPackageKind(),
		name,
	)
}

func managedCollectionAddressFor(
	packageKind managedpackageModel.PackageKind,
	name spec.LogicalName,
) (managedpackageModel.ManagedPackageAddress, error) {
	return managedpackageModel.NewManagedPackageAddress(
		packageKind,
		name,
		topology.UnversionedPackageVersion(),
	)
}

func (a *API) managedCollectionAddressFromLocator(
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return managedCollectionAddressFromLocatorFor(
		a.managedCollectionPackageKind(),
		a.managedCollectionDocumentFile(),
		locator,
	)
}

func (a *API) managedCollectionPackageKind() managedpackageModel.PackageKind {
	if a != nil && a.domain != nil && a.domain.PackageKind != "" {
		return a.domain.PackageKind
	}
	return ManagedCollectionPackageKind
}

func (a *API) managedCollectionDocumentUse() string {
	if a != nil && a.domain != nil && a.domain.DocumentUse != "" {
		return a.domain.DocumentUse
	}
	return topology.DocumentUseManagedPlugin
}

func (a *API) managedCollectionDocumentFile() spec.Locator {
	return topology.MustDefaultDocumentFile(
		a.managedCollectionDocumentUse(),
	)
}

func (a *API) managedCollectionDecoderID() (spec.DecoderID, error) {
	return topology.DefaultDocumentDecoderID(
		a.managedCollectionDocumentUse(),
	)
}

func managedCollectionAddressFromLocator(
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return managedCollectionAddressFromLocatorFor(
		ManagedCollectionPackageKind,
		topology.MustDefaultDocumentFile(topology.DocumentUseManagedPlugin),
		locator,
	)
}

func managedCollectionAddressFromLocatorFor(
	packageKind managedpackageModel.PackageKind,
	documentFile spec.Locator,
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if err := packageKind.Validate(); err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if err := documentFile.ValidatePortable(false); err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(documentFile) {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: Plugin locator %q is not %q",
			spec.ErrUnsupported,
			locator,
			documentFile,
		)
	}
	address, err := managedpackageModel.ParseManagedPackageAddressDirectory(
		spec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if address.Kind != packageKind {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Plugin package kind must be %q",
			spec.ErrUnsupported,
			packageKind,
		)
	}
	return address, nil
}

func collectionDocumentPayload(
	document pluginv1.PluginDocument,
) ([]byte, cryptoutil.Digest, error) {
	raw, err := document.CanonicalJSON()
	if err != nil {
		return nil, "", err
	}
	value, err := pluginv1.DefinitionForDocument(document)
	if err != nil {
		return nil, "", err
	}
	return raw, value.Digest, nil
}

func (a *API) collectionViewOf(
	record artifactModel.Artifact,
	document pluginv1.PluginDocument,
) (PluginView, error) {
	baseline := IsBaselinePluginArtifact(record)
	if a != nil && a.domain != nil {
		address, err := a.managedCollectionAddressFromLocator(
			record.Binding.Locator,
		)
		if err == nil &&
			record.LogicalName == a.domain.BaselineName &&
			address.Name == a.domain.BaselineName {
			baseline = true
		}
	}
	return readCollectionViewOf(
		record,
		document,
		true,
		!baseline,
		baseline,
	)
}

func (m MemberReference) entry() (declaration.Entry, error) {
	if err := m.Type.Validate(); err != nil {
		return declaration.Entry{}, err
	}
	if err := m.Name.Validate(); err != nil {
		return declaration.Entry{}, err
	}
	if m.Locator != nil {
		locator := m.Locator.Clone()
		m.Locator = &locator
	}
	entry, err := declaration.NewEntry(struct {
		declaration.Header

		Insert declaration.InsertTarget `json:"insert,omitempty"`
		Scope  declaration.LookupScope  `json:"scope,omitempty"`
		Server string                   `json:"server,omitempty"`
	}{
		Type:    m.Type,
		Name:    string(m.Name),
		Locator: m.Locator,

		Insert: m.Insert,
		Scope:  m.Scope,
		Server: string(m.Server),
	})
	if err != nil {
		return declaration.Entry{}, err
	}
	form, err := entry.MemberForm()
	if err != nil {
		return declaration.Entry{}, err
	}
	if form != declaration.MemberNamed {
		return declaration.Entry{}, fmt.Errorf(
			"%w: managed Plugin member must be a named external member",
			spec.ErrUnsupported,
		)
	}
	return entry, nil
}

func memberReferenceFromEntry(
	entry declaration.Entry,
) (MemberReference, error) {
	form, err := entry.MemberForm()
	if err != nil {
		return MemberReference{}, err
	}
	if form != declaration.MemberNamed {
		return MemberReference{}, fmt.Errorf(
			"%w: editable Plugin members must be named external references",
			spec.ErrUnsupported,
		)
	}

	header := entry.Header()
	output := MemberReference{
		Type: header.Type,
		Name: spec.LogicalName(header.Name),
	}
	if header.Locator != nil {
		locator := header.Locator.Clone()
		output.Locator = &locator
	}
	if header.Type == declaration.TypeText {
		insert, err := entry.TextInsert()
		if err != nil {
			return MemberReference{}, err
		}
		output.Insert = insert
	}
	relationship, err := entry.Relationship()
	if err != nil {
		return MemberReference{}, err
	}
	output.Scope = relationship.Scope

	raw, err := entry.CanonicalJSON()
	if err != nil {
		return MemberReference{}, err
	}
	var selector struct {
		Server string `json:"server"`
	}
	if err := json.Unmarshal(raw, &selector); err != nil {
		return MemberReference{}, err
	}
	if selector.Server != "" {
		output.Server = spec.LogicalName(selector.Server)
		if err := output.Server.Validate(); err != nil {
			return MemberReference{}, err
		}
	}
	return output, nil
}

func relativeSourceLocator(
	from spec.Locator,
	target spec.Locator,
) (declaration.Locator, error) {
	if err := from.Validate(false); err != nil {
		return declaration.Locator{}, err
	}
	if err := target.Validate(false); err != nil {
		return declaration.Locator{}, err
	}

	fromDirectory := path.Dir(string(from))
	fromParts := locatorParts(fromDirectory)
	targetParts := locatorParts(string(target))

	common := 0
	for common < len(fromParts) &&
		common < len(targetParts) &&
		fromParts[common] == targetParts[common] {
		common++
	}

	relativeParts := make(
		[]string,
		0,
		len(fromParts)-common+len(targetParts)-common,
	)
	for index := common; index < len(fromParts); index++ {
		relativeParts = append(relativeParts, "..")
	}
	relativeParts = append(relativeParts, targetParts[common:]...)
	if len(relativeParts) == 0 {
		relativeParts = []string{path.Base(string(target))}
	}

	value := declaration.PathLocator(strings.Join(relativeParts, "/"))
	if err := value.Validate(); err != nil {
		return declaration.Locator{}, err
	}
	return value, nil
}

func locatorParts(value string) []string {
	if value == "" || value == "." {
		return nil
	}
	return strings.Split(value, "/")
}

func (a *API) isBaselineEditableCollection(
	value editableCollection,
) bool {
	return a != nil &&
		a.domain != nil &&
		value.artifact.LogicalName == a.domain.BaselineName &&
		value.address.Name == a.domain.BaselineName
}

func normalizedMemberIndexes(
	values []declaration.Entry,
) ([]int, error) {
	type member struct {
		index int
		raw   []byte
	}

	ordered := make([]member, 0, len(values))
	for index, value := range values {
		raw, err := value.CanonicalJSON()
		if err != nil {
			return nil, err
		}
		ordered = append(ordered, member{
			index: index,
			raw:   raw,
		})
	}
	sort.SliceStable(ordered, func(left, right int) bool {
		return bytes.Compare(ordered[left].raw, ordered[right].raw) < 0
	})

	output := make([]int, len(ordered))
	for index, value := range ordered {
		output[index] = value.index
	}
	return output, nil
}

func normalizedMemberIndex(
	values []declaration.Entry,
	sourceIndex int,
) (int, error) {
	ordered, err := normalizedMemberIndexes(values)
	if err != nil {
		return 0, err
	}
	for index, value := range ordered {
		if value == sourceIndex {
			return index, nil
		}
	}
	return 0, fmt.Errorf("%w: Plugin member does not exist", spec.ErrNotFound)
}
