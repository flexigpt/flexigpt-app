// Package collection temporarily implements editable managed Plugin declarations.
//
// Plugin membership remains declaration content. This package does not
// create Artifact Store ownership, foreign keys, or lifecycle relationships.
package collection

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local/consumerutil"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const (
	ManagedCollectionPackageKind source.PackageKind = "plugin"
)

type API struct {
	sources          local.SourceAPI
	discovery        local.DiscoveryAPI
	artifacts        local.ArtifactAPI
	managedArtifacts local.ManagedArtifactAPI
	domain           *DomainPolicy
	resolver         *resolve.Resolver

	// Immutable Plugin projections keyed by Root-local Definition digest.
	catalogProjections consumerutil.DocumentCache[collectionProjection]
}

func NewWithResolver(
	sources local.SourceAPI,
	discovery local.DiscoveryAPI,
	artifacts local.ArtifactAPI,
	managedArtifacts local.ManagedArtifactAPI,
	resolver *resolve.Resolver,
	domains ...DomainPolicy,
) (*API, error) {
	return newAPI(
		sources,
		discovery,
		artifacts,
		managedArtifacts,
		resolver,
		domains...,
	)
}

func newAPI(
	sources local.SourceAPI,
	discovery local.DiscoveryAPI,
	artifacts local.ArtifactAPI,
	managedArtifacts local.ManagedArtifactAPI,
	resolver *resolve.Resolver,
	domains ...DomainPolicy,
) (*API, error) {
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		managedArtifacts == nil {
		return nil, fmt.Errorf(
			"%w: managed Collection dependencies are incomplete",
			model.ErrInvalid,
		)
	}
	if len(domains) > 1 {
		return nil, fmt.Errorf(
			"%w: Collection API has multiple domain policies",
			model.ErrInvalid,
		)
	}
	output := &API{
		sources:          sources,
		discovery:        discovery,
		artifacts:        artifacts,
		managedArtifacts: managedArtifacts,
		resolver:         resolver,
	}
	if len(domains) == 1 {
		value := domains[0]
		if err := value.Validate(); err != nil {
			return nil, err
		}
		value.AllowedMemberTypes = append(
			[]declaration.Type(nil),
			value.AllowedMemberTypes...,
		)
		value.AllowedMemberForms = append(
			[]declaration.MemberForm(nil),
			value.AllowedMemberForms...,
		)
		output.domain = &value
	}
	return output, nil
}

type CollectionView struct {
	Artifact    artifact.Artifact      `json:"artifact"`
	Name        model.LogicalName      `json:"name"`
	DisplayName string                 `json:"displayName"`
	Description string                 `json:"description,omitempty"`
	Members     []MemberReference      `json:"members"`
	Entries     []CollectionMemberView `json:"entries"`
	Editable    bool                   `json:"editable"`
	Deletable   bool                   `json:"deletable"`
	Baseline    bool                   `json:"baseline"`
}

type CollectionMemberView struct {
	Type      declaration.Type         `json:"type"`
	Name      model.LogicalName        `json:"name,omitempty"`
	Insert    declaration.InsertTarget `json:"insert,omitempty"`
	Locator   *declaration.Locator     `json:"locator,omitempty"`
	Server    model.LogicalName        `json:"server,omitempty"`
	Contained bool                     `json:"contained"`
	Selector  bool                     `json:"selector"`
}

// MemberReference is an external Plugin membership edge. It intentionally
// cannot express a contained declaration.
type MemberReference struct {
	Type    declaration.Type         `json:"type"`
	Name    model.LogicalName        `json:"name"`
	Insert  declaration.InsertTarget `json:"insert,omitempty"`
	Locator *declaration.Locator     `json:"locator,omitempty"`
	Scope   declaration.LookupScope  `json:"scope,omitempty"`
	Server  model.LogicalName        `json:"server,omitempty"`
}

type CreateRequest struct {
	RootID      root.RootID       `json:"rootID"`
	SourceID    source.SourceID   `json:"sourceID,omitempty"`
	Name        model.LogicalName `json:"name"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
}

type UpdateRequest struct {
	Collection       artifact.ArtifactRef `json:"collection"`
	ExpectedRevision uint64               `json:"expectedRevision"`
	Description      string               `json:"description,omitempty"`
	DisplayName      string               `json:"displayName,omitempty"`
}

type AddMemberRequest struct {
	Collection       artifact.ArtifactRef `json:"collection"`
	ExpectedRevision uint64               `json:"expectedRevision"`
	Member           MemberReference      `json:"member"`
}

type AddEntryRequest struct {
	Collection       artifact.ArtifactRef `json:"collection"`
	ExpectedRevision uint64               `json:"expectedRevision"`
	Entry            declaration.Entry    `json:"entry"`
}

type AddArtifactMemberRequest struct {
	Collection       artifact.ArtifactRef `json:"collection"`
	ExpectedRevision uint64               `json:"expectedRevision"`
	Artifact         artifact.ArtifactRef `json:"artifact"`
}

type RemoveMemberRequest struct {
	Collection       artifact.ArtifactRef `json:"collection"`
	ExpectedRevision uint64               `json:"expectedRevision"`
	Index            int                  `json:"index"`
}

type DeleteRequest struct {
	Collection       artifact.ArtifactRef `json:"collection"`
	ExpectedRevision uint64               `json:"expectedRevision"`
}

// MemberMutationResult is used by managed Skill, MCP, and policy creation.
// Created reports whether this call appended the member rather than finding an
// identical existing membership.
type MemberMutationResult struct {
	Collection CollectionView `json:"collection"`
	Index      int            `json:"index"`
	Created    bool           `json:"created"`
}

type editableCollection struct {
	artifact   artifact.Artifact
	document   pluginv1.PluginDocument
	address    source.ManagedPackageAddress
	generation string
}

func (a *API) Create(
	ctx context.Context,
	request CreateRequest,
) (CollectionView, error) {
	return a.create(ctx, request, false)
}

func (a *API) Get(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (CollectionView, error) {
	if a == nil {
		return CollectionView{}, model.ErrClosed
	}
	value, err := a.loadEditableCollection(ctx, ref, 0)
	if err != nil {
		return CollectionView{}, err
	}
	return a.collectionViewOf(value.artifact, value.document)
}

// SetEnabled changes the generic local enablement metadata of a Collection.
//
// It is intentionally independent of Collection editability, membership,
// deletion eligibility, source ownership, and Root protection. A disabled
// Collection remains readable and editable when it was otherwise editable.
func (a *API) SetEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (CollectionView, error) {
	if a == nil {
		return CollectionView{}, model.ErrClosed
	}
	if expectedRevision == 0 {
		return CollectionView{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			model.ErrInvalid,
		)
	}

	view, err := a.Read(ctx, ref)
	if err != nil {
		return CollectionView{}, err
	}
	if view.Artifact.Revision != expectedRevision {
		return CollectionView{}, model.ErrConflict
	}

	updated, err := a.artifacts.SetEnabled(
		ctx,
		view.Artifact.Ref(),
		expectedRevision,
		enabled,
	)
	if err != nil {
		return CollectionView{}, err
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
) (CollectionView, error) {
	if a == nil {
		return CollectionView{}, model.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return CollectionView{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			model.ErrInvalid,
		)
	}

	value, err := a.loadEditableCollection(
		ctx,
		request.Collection,
		request.ExpectedRevision,
	)
	if err != nil {
		return CollectionView{}, err
	}
	if _, err := a.collectionViewOf(value.artifact, value.document); err != nil {
		return CollectionView{}, err
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
		return CollectionView{}, err
	}
	return a.collectionViewOf(record, value.document)
}

func (a *API) AddMember(
	ctx context.Context,
	request AddMemberRequest,
) (CollectionView, error) {
	result, err := a.mutateMember(ctx, request, false)
	if err != nil {
		return CollectionView{}, err
	}
	return result.Collection, nil
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
) (CollectionView, error) {
	if a == nil {
		return CollectionView{}, model.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return CollectionView{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			model.ErrInvalid,
		)
	}

	value, err := a.loadEditableCollection(
		ctx,
		request.Collection,
		request.ExpectedRevision,
	)
	if err != nil {
		return CollectionView{}, err
	}
	if _, err := a.collectionViewOf(value.artifact, value.document); err != nil {
		return CollectionView{}, err
	}
	if request.Index < 0 || request.Index >= len(value.document.Members) {
		return CollectionView{}, fmt.Errorf(
			"%w: Collection member index %d is out of range",
			model.ErrNotFound,
			request.Index,
		)
	}
	normalized, err := normalizedMemberIndexes(value.document.Members)
	if err != nil {
		return CollectionView{}, err
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
		return CollectionView{}, err
	}
	return a.collectionViewOf(record, value.document)
}

func (a *API) AddEntry(
	ctx context.Context,
	request AddEntryRequest,
) (CollectionView, error) {
	result, err := a.mutateEntry(ctx, request, false)
	if err != nil {
		return CollectionView{}, err
	}
	return result.Collection, nil
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
) (CollectionView, error) {
	if a == nil {
		return CollectionView{}, model.ErrClosed
	}
	if err := request.Collection.Validate(); err != nil {
		return CollectionView{}, err
	}
	if err := request.Artifact.Validate(); err != nil {
		return CollectionView{}, err
	}
	if request.Collection.RootID != request.Artifact.RootID {
		return CollectionView{}, fmt.Errorf(
			"%w: Collection member Artifact belongs to another Root",
			model.ErrInvalid,
		)
	}

	target, err := a.artifacts.Get(ctx, request.Artifact)
	if err != nil {
		return CollectionView{}, err
	}
	declarationType := declaration.Type(target.Kind)
	if err := declarationType.Validate(); err != nil {
		return CollectionView{}, err
	}

	member := MemberReference{
		Type: declarationType,
		Name: target.LogicalName,
	}
	collectionValue, err := a.loadEditableCollection(
		ctx,
		request.Collection,
		request.ExpectedRevision,
	)
	if err != nil {
		return CollectionView{}, err
	}
	if target.Binding.SourceID == collectionValue.artifact.Binding.SourceID {
		locator, err := relativeSourceLocator(
			collectionValue.artifact.Binding.Locator,
			target.Binding.Locator,
		)
		if err != nil {
			return CollectionView{}, err
		}
		member.Locator = &locator
	}

	return a.AddMember(
		ctx,
		AddMemberRequest{
			Collection:       request.Collection,
			ExpectedRevision: request.ExpectedRevision,
			Member:           member,
		},
	)
}

// MemberForCollectionSource builds a location-constrained external reference
// for a declaration that will be published into the same managed Source as
// the Collection.
func (a *API) MemberForCollectionSource(
	ctx context.Context,
	collectionRef artifact.ArtifactRef,
	declarationType declaration.Type,
	name model.LogicalName,
	target model.Locator,
) (MemberReference, error) {
	if a == nil {
		return MemberReference{}, model.ErrClosed
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
		return model.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Collection revision is required",
			model.ErrInvalid,
		)
	}

	value, err := a.loadEditableCollection(
		ctx,
		request.Collection,
		request.ExpectedRevision,
	)
	if err != nil {
		return err
	}
	if a.isBaselineEditableCollection(value) {
		return fmt.Errorf(
			"%w: baseline Collection %q cannot be deleted",
			model.ErrProtected,
			value.artifact.LogicalName,
		)
	}
	if len(value.document.Members) != 0 {
		return fmt.Errorf(
			"%w: Collection %q has %d direct members; detach them before deletion",
			model.ErrConflict,
			value.artifact.LogicalName,
			len(value.document.Members),
		)
	}

	removeRequest := artifact.RemoveArtifactRequest{
		RootID:             value.artifact.RootID,
		SourceID:           value.artifact.Binding.SourceID,
		Package:            value.address,
		ExpectedArtifact:   &request.Collection,
		ExpectedGeneration: value.generation,
	}
	if a.domain != nil {
		locator := value.artifact.Binding.Locator
		removeRequest.PruneDiscoveryLocator = &locator
	}
	if err := a.managedArtifacts.Remove(ctx, removeRequest); err != nil {
		return err
	}

	missing, err := a.artifacts.Get(ctx, request.Collection)
	if err != nil {
		return err
	}
	if missing.State != artifact.StateMissing {
		return fmt.Errorf(
			"%w: removed Collection Artifact is not missing",
			model.ErrConflict,
		)
	}

	return a.artifacts.Purge(
		ctx,
		request.Collection,
		missing.Revision,
	)
}

func (a *API) resolveCollectionRef(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.ArtifactRef, error) {
	if err := ref.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}
	if a == nil || a.resolver == nil {
		return ref, nil
	}
	resolved, err := a.resolver.ResolvePlugin(ctx, ref)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}
	terminal, found := resolved.ArtifactRef()
	if !found {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: plugin did not resolve to a source-backed Artifact",
			model.ErrReferenceUnresolved,
		)
	}
	return terminal, nil
}

func (a *API) create(
	ctx context.Context,
	request CreateRequest,
	allowBaseline bool,
) (CollectionView, error) {
	if a == nil {
		return CollectionView{}, model.ErrClosed
	}
	if err := a.requireDeclarationAuthoring(); err != nil {
		return CollectionView{}, err
	}
	if err := request.RootID.Validate(); err != nil {
		return CollectionView{}, err
	}
	if err := request.Name.Validate(); err != nil {
		return CollectionView{}, err
	}
	if a.domain != nil &&
		request.Name == a.domain.BaselineName &&
		!allowBaseline {
		return CollectionView{}, fmt.Errorf(
			"%w: baseline Collection %q is application-provisioned",
			model.ErrProtected,
			request.Name,
		)
	}

	sourceValue, err := a.managedSource(
		ctx,
		request.RootID,
		request.SourceID,
	)
	if err != nil {
		return CollectionView{}, err
	}
	address, err := a.managedCollectionAddress(request.Name)
	if err != nil {
		return CollectionView{}, err
	}
	locator, err := address.FileLocator(a.managedCollectionDocumentFile())
	if err != nil {
		return CollectionView{}, err
	}

	document := pluginv1.PluginDocument{
		Type:        pluginv1.PluginType,
		Name:        string(request.Name),
		DisplayName: request.DisplayName,
		Description: request.Description,
	}
	_, digest, err := collectionDocumentPayload(document)
	if err != nil {
		return CollectionView{}, err
	}

	existing, err := a.artifacts.FindByOrigin(
		ctx,
		request.RootID,
		artifact.SourceBinding{
			SourceID: sourceValue.ID,
			Locator:  locator,
		},
		artifact.ArtifactKind(pluginv1.PluginType),
	)
	switch {
	case err == nil:
		if existing.State == artifact.StateAvailable &&
			existing.ResolvedDefinition != nil &&
			*existing.ResolvedDefinition == digest {
			return a.Get(ctx, existing.Ref())
		}
		return CollectionView{}, fmt.Errorf(
			"%w: managed Collection %q already exists",
			model.ErrConflict,
			request.Name,
		)

	case errors.Is(err, model.ErrArtifactNotFound),
		errors.Is(err, model.ErrNotFound):
	default:
		return CollectionView{}, err
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
		return CollectionView{}, err
	}
	return a.collectionViewOf(record, document)
}

func (a *API) mutateMember(
	ctx context.Context,
	request AddMemberRequest,
	ensure bool,
) (MemberMutationResult, error) {
	if a == nil {
		return MemberMutationResult{}, model.ErrClosed
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
			Collection:       request.Collection,
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
		return MemberMutationResult{}, model.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return MemberMutationResult{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			model.ErrInvalid,
		)
	}
	if err := a.validateDomainEntry(request.Entry); err != nil {
		return MemberMutationResult{}, err
	}
	member := request.Entry.Clone()
	memberRaw, err := member.CanonicalJSON()
	if err != nil {
		return MemberMutationResult{}, err
	}

	value, err := a.loadEditableCollection(
		ctx,
		request.Collection,
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
		for normalizedIndex, sourceIndex := range normalized {
			current := value.document.Members[sourceIndex]
			currentRaw, err := current.CanonicalJSON()
			if err != nil {
				return MemberMutationResult{}, err
			}
			if bytes.Equal(currentRaw, memberRaw) {
				return MemberMutationResult{
					Collection: view,
					Index:      normalizedIndex,
					Created:    false,
				}, nil
			}
		}
	}

	value.document.Members = append(
		append([]declaration.Entry(nil), value.document.Members...),
		member,
	)
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
		Collection: view,
		Index:      normalizedIndex,
		Created:    true,
	}, nil
}

func (a *API) managedSource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) (source.Summary, error) {
	if a.domain != nil {
		return a.domainManagedSource(ctx, rootID, sourceID)
	}

	value, err := a.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return source.Summary{}, err
	}
	if value.Kind != source.SourceKindManagedDirectory {
		return source.Summary{}, fmt.Errorf(
			"%w: editable Collection Source must have kind %q",
			model.ErrUnsupported,
			source.SourceKindManagedDirectory,
		)
	}
	if !value.Enabled {
		return source.Summary{}, fmt.Errorf(
			"%w: editable Collection Source is disabled",
			model.ErrConflict,
		)
	}
	return value, nil
}

func (a *API) publishDocument(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	address source.ManagedPackageAddress,
	document pluginv1.PluginDocument,
	expectedGeneration string,
	allowPackageReplacement bool,
) (artifact.Artifact, error) {
	documentFile := a.managedCollectionDocumentFile()
	decoderID, err := a.managedCollectionDecoderID()
	if err != nil {
		return artifact.Artifact{}, err
	}

	raw, digest, err := collectionDocumentPayload(document)
	if err != nil {
		return artifact.Artifact{}, err
	}
	locator, err := address.FileLocator(documentFile)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if _, err := a.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceID,
		locator,
		decoderID,
	); err != nil {
		return artifact.Artifact{}, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: rootID,
			Binding: artifact.SourceBinding{
				SourceID: sourceID,
				Locator:  locator,
			},
			ExpectedKind: artifact.ArtifactKind(
				pluginv1.PluginType,
			),
			ExpectedLogicalName: model.LogicalName(document.Name),
			ExpectedDefinition:  digest,
			Package: source.ManagedPackagePublication{
				Address:            address,
				ExpectedGeneration: expectedGeneration,
				Files: []source.ManagedPackageFile{{
					Locator: documentFile,
					Content: raw,
				}},
			},
			AllowPackageReplacement: allowPackageReplacement,
		},
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	return published.Artifact, nil
}

func (a *API) loadEditableCollection(
	ctx context.Context,
	ref artifact.ArtifactRef,
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
	if record.Kind != artifact.ArtifactKind(pluginv1.PluginType) {
		return editableCollection{}, fmt.Errorf(
			"%w: Artifact %q is not a Plugin",
			model.ErrUnsupported,
			record.ID,
		)
	}
	if expectedRevision != 0 && record.Revision != expectedRevision {
		return editableCollection{}, model.ErrConflict
	}
	if record.State != artifact.StateAvailable {
		return editableCollection{}, fmt.Errorf(
			"%w: Collection Artifact %q is unavailable",
			model.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if record.Binding.SubresourceLocator != "" {
		return editableCollection{}, fmt.Errorf(
			"%w: contained Collection declarations are not editable managed Collections",
			model.ErrUnsupported,
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
	if sourceValue.Kind != source.SourceKindManagedDirectory {
		return editableCollection{}, fmt.Errorf(
			"%w: Collection is not backed by a managed Source",
			model.ErrUnsupported,
		)
	}
	if !sourceValue.Enabled {
		return editableCollection{}, fmt.Errorf(
			"%w: Collection Source is disabled",
			model.ErrReferenceUnresolved,
		)
	}
	if a.domain != nil &&
		sourceValue.StorageKey != a.domain.SourceStorageKey {
		return editableCollection{}, fmt.Errorf(
			"%w: Collection belongs to another managed domain Source",
			model.ErrReferenceUnresolved,
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
			"%w: managed Collection package name does not match Artifact identity",
			model.ErrInvalid,
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
			"%w: managed Collection Source requires refresh",
			model.ErrRefreshRequired,
		)
	}

	definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return editableCollection{}, err
	}
	document, err := pluginv1.DecodePluginJSON(
		definitionValue.Body,
	)
	if err != nil {
		return editableCollection{}, err
	}
	if document.Name != string(record.LogicalName) {
		return editableCollection{}, fmt.Errorf(
			"%w: Collection declaration name differs from Artifact identity",
			model.ErrInvalid,
		)
	}
	if document.Locator != nil {
		return editableCollection{}, fmt.Errorf(
			"%w: located Collection aliases are not editable managed Collections",
			model.ErrUnsupported,
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
	name model.LogicalName,
) (source.ManagedPackageAddress, error) {
	return managedCollectionAddressFor(
		a.managedCollectionPackageKind(),
		name,
	)
}

func managedCollectionAddressFor(
	packageKind source.PackageKind,
	name model.LogicalName,
) (source.ManagedPackageAddress, error) {
	return source.NewManagedPackageAddress(
		packageKind,
		name,
		documentTopology.UnversionedPackageVersion(),
	)
}

func (a *API) managedCollectionAddressFromLocator(
	locator model.Locator,
) (source.ManagedPackageAddress, error) {
	return managedCollectionAddressFromLocatorFor(
		a.managedCollectionPackageKind(),
		a.managedCollectionDocumentFile(),
		locator,
	)
}

func (a *API) managedCollectionPackageKind() source.PackageKind {
	if a != nil && a.domain != nil && a.domain.PackageKind != "" {
		return a.domain.PackageKind
	}
	return ManagedCollectionPackageKind
}

func (a *API) managedCollectionDocumentUse() string {
	if a != nil && a.domain != nil && a.domain.DocumentUse != "" {
		return a.domain.DocumentUse
	}
	return documentTopology.DocumentUseManagedCollection
}

func (a *API) managedCollectionDocumentFile() model.Locator {
	return documentTopology.MustDefaultDocumentFile(
		a.managedCollectionDocumentUse(),
	)
}

func (a *API) managedCollectionDecoderID() (model.DecoderID, error) {
	return documentTopology.DefaultDocumentDecoderID(
		a.managedCollectionDocumentUse(),
	)
}

func managedCollectionAddressFromLocator(
	locator model.Locator,
) (source.ManagedPackageAddress, error) {
	return managedCollectionAddressFromLocatorFor(
		ManagedCollectionPackageKind,
		documentTopology.MustDefaultDocumentFile(documentTopology.DocumentUseManagedCollection),
		locator,
	)
}

func managedCollectionAddressFromLocatorFor(
	packageKind source.PackageKind,
	documentFile model.Locator,
	locator model.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if err := packageKind.Validate(); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if err := documentFile.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(documentFile) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: Collection locator %q is not %q",
			model.ErrUnsupported,
			locator,
			documentFile,
		)
	}
	address, err := source.ParseManagedPackageAddressDirectory(
		model.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if address.Kind != packageKind {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Collection package kind must be %q",
			model.ErrUnsupported,
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
	entry, err := declaration.NewEntry(document)
	if err != nil {
		return nil, "", err
	}
	value, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return nil, "", err
	}
	return raw, value.Digest, nil
}

func (a *API) collectionViewOf(
	record artifact.Artifact,
	document pluginv1.PluginDocument,
) (CollectionView, error) {
	baseline := IsBaselineCollectionArtifact(record)
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
			model.ErrUnsupported,
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
			model.ErrUnsupported,
		)
	}

	header := entry.Header()
	output := MemberReference{
		Type: header.Type,
		Name: model.LogicalName(header.Name),
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
		output.Server = model.LogicalName(selector.Server)
		if err := output.Server.Validate(); err != nil {
			return MemberReference{}, err
		}
	}
	return output, nil
}

func relativeSourceLocator(
	from model.Locator,
	target model.Locator,
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
	return 0, fmt.Errorf("%w: Collection member does not exist", model.ErrNotFound)
}
