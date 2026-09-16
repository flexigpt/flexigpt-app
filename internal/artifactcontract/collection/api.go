// Package collection implements editable managed Collection declarations.
//
// Collection membership remains declaration content. This package does not
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
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/collectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const (
	ManagedCollectionPackageKind  source.PackageKind      = "collection"
	ManagedCollectionDocumentFile basespec.Locator        = "collection.json"
	ManagedCollectionVersion      basespec.LogicalVersion = "unversioned"

	UserManagedArtifactSourceStorageKey basespec.StorageKey = "user-artifacts"

	UserManagedArtifactSourceDisplayName = "User-managed artifacts"
)

type API struct {
	sources          compositionapi.SourceAPI
	discovery        compositionapi.DiscoveryAPI
	artifacts        compositionapi.ArtifactAPI
	managedArtifacts compositionapi.ManagedArtifactAPI
	domain           *DomainPolicy
	resolver         *resolve.Resolver
}

func New(
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	domains ...DomainPolicy,
) (*API, error) {
	return newAPI(
		sources,
		discovery,
		artifacts,
		managedArtifacts,
		nil,
		domains...,
	)
}

func NewWithResolver(
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
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
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	resolver *resolve.Resolver,
	domains ...DomainPolicy,
) (*API, error) {
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		managedArtifacts == nil {
		return nil, fmt.Errorf(
			"%w: managed Collection dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	if len(domains) > 1 {
		return nil, fmt.Errorf(
			"%w: Collection API has multiple domain policies",
			basespec.ErrInvalid,
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
		output.domain = &value
	}
	return output, nil
}

type CollectionView struct {
	Artifact    artifact.Artifact       `json:"artifact"`
	Name        basespec.LogicalName    `json:"name"`
	Description string                  `json:"description,omitempty"`
	Version     basespec.LogicalVersion `json:"version,omitempty"`
	Members     []MemberReference       `json:"members"`
	Entries     []CollectionMemberView  `json:"entries"`
	Editable    bool                    `json:"editable"`
	Deletable   bool                    `json:"deletable"`
	Baseline    bool                    `json:"baseline"`
}

type CollectionMemberView struct {
	Type        declaration.Type           `json:"type"`
	Name        basespec.LogicalName       `json:"name"`
	Description string                     `json:"description,omitempty"`
	Locator     *declaration.Locator       `json:"locator,omitempty"`
	Metadata    map[string]json.RawMessage `json:"metadata,omitempty"`
	Server      basespec.LogicalName       `json:"server,omitempty"`
	Contained   bool                       `json:"contained"`
}

// MemberReference is an external Collection membership edge. It intentionally
// cannot express a contained declaration.
type MemberReference struct {
	Type        declaration.Type           `json:"type"`
	Name        basespec.LogicalName       `json:"name"`
	Description string                     `json:"description,omitempty"`
	Locator     *declaration.Locator       `json:"locator,omitempty"`
	Metadata    map[string]json.RawMessage `json:"metadata,omitempty"`
	Server      basespec.LogicalName       `json:"server,omitempty"`
}

type CreateRequest struct {
	RootID      root.RootID          `json:"rootID"`
	SourceID    source.SourceID      `json:"sourceID,omitempty"`
	Name        basespec.LogicalName `json:"name"`
	Description string               `json:"description,omitempty"`
}

type UpdateRequest struct {
	Collection       artifact.ArtifactRef `json:"collection"`
	ExpectedRevision uint64               `json:"expectedRevision"`
	Description      string               `json:"description,omitempty"`
}

type AddMemberRequest struct {
	Collection       artifact.ArtifactRef `json:"collection"`
	ExpectedRevision uint64               `json:"expectedRevision"`
	Member           MemberReference      `json:"member"`
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
	document   collectionv1.CollectionDocument
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
		return CollectionView{}, basespec.ErrClosed
	}
	value, err := a.loadEditableCollection(ctx, ref, 0)
	if err != nil {
		return CollectionView{}, err
	}
	return collectionViewOf(value.artifact, value.document)
}

func (a *API) List(
	ctx context.Context,
	rootID root.RootID,
) ([]CollectionView, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return nil, err
	}

	records, err := a.artifacts.ListByRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}

	output := make([]CollectionView, 0)
	for _, record := range records {
		if record.Kind != artifact.ArtifactKind(collectionv1.CollectionType) ||
			record.State != artifact.StateAvailable ||
			record.Binding.SubresourceLocator != "" {
			continue
		}

		view, err := a.Get(ctx, record.Ref())
		if errors.Is(err, basespec.ErrUnsupported) {
			continue
		}
		if err != nil {
			return nil, err
		}
		output = append(output, view)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Artifact.ID < output[right].Artifact.ID
	})
	return output, nil
}

func (a *API) Update(
	ctx context.Context,
	request UpdateRequest,
) (CollectionView, error) {
	if a == nil {
		return CollectionView{}, basespec.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return CollectionView{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			basespec.ErrInvalid,
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
	if _, err := collectionViewOf(value.artifact, value.document); err != nil {
		return CollectionView{}, err
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
	return collectionViewOf(record, value.document)
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
		return CollectionView{}, basespec.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return CollectionView{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			basespec.ErrInvalid,
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
	if _, err := collectionViewOf(value.artifact, value.document); err != nil {
		return CollectionView{}, err
	}
	if request.Index < 0 || request.Index >= len(value.document.Members) {
		return CollectionView{}, fmt.Errorf(
			"%w: Collection member index %d is out of range",
			basespec.ErrNotFound,
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
	return collectionViewOf(record, value.document)
}

func (a *API) AddArtifactMember(
	ctx context.Context,
	request AddArtifactMemberRequest,
) (CollectionView, error) {
	if a == nil {
		return CollectionView{}, basespec.ErrClosed
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
			basespec.ErrInvalid,
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
	name basespec.LogicalName,
	target basespec.Locator,
) (MemberReference, error) {
	if a == nil {
		return MemberReference{}, basespec.ErrClosed
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
		return basespec.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Collection revision is required",
			basespec.ErrInvalid,
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
			basespec.ErrProtected,
			value.artifact.LogicalName,
		)
	}
	if len(value.document.Members) != 0 {
		return fmt.Errorf(
			"%w: Collection %q has %d direct members; detach them before deletion",
			basespec.ErrConflict,
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
			basespec.ErrConflict,
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
	return a.resolver.ResolveDeclarationArtifact(ctx, ref)
}

func (a *API) create(
	ctx context.Context,
	request CreateRequest,
	allowBaseline bool,
) (CollectionView, error) {
	if a == nil {
		return CollectionView{}, basespec.ErrClosed
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
			basespec.ErrProtected,
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
	address, err := managedCollectionAddress(request.Name)
	if err != nil {
		return CollectionView{}, err
	}
	locator, err := address.FileLocator(ManagedCollectionDocumentFile)
	if err != nil {
		return CollectionView{}, err
	}

	document := collectionv1.CollectionDocument{
		APIVersion:  collectionv1.CollectionSchemaVersion,
		Type:        collectionv1.CollectionType,
		Name:        string(request.Name),
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
		artifact.ArtifactKind(collectionv1.CollectionType),
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
			basespec.ErrConflict,
			request.Name,
		)

	case errors.Is(err, basespec.ErrArtifactNotFound),
		errors.Is(err, basespec.ErrNotFound):
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
	return collectionViewOf(record, document)
}

func (a *API) mutateMember(
	ctx context.Context,
	request AddMemberRequest,
	ensure bool,
) (MemberMutationResult, error) {
	if a == nil {
		return MemberMutationResult{}, basespec.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return MemberMutationResult{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			basespec.ErrInvalid,
		)
	}

	if err := a.validateDomainMember(request.Member); err != nil {
		return MemberMutationResult{}, err
	}
	member, err := request.Member.entry()
	if err != nil {
		return MemberMutationResult{}, err
	}
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
	view, err := collectionViewOf(value.artifact, value.document)
	if err != nil {
		return MemberMutationResult{}, err
	}

	if ensure {
		for index, current := range value.document.Members {
			currentRaw, err := current.CanonicalJSON()
			if err != nil {
				return MemberMutationResult{}, err
			}
			normalizedIndex, err := normalizedMemberIndex(
				value.document.Members,
				index,
			)
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
	view, err = collectionViewOf(record, value.document)
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
			basespec.ErrUnsupported,
			source.SourceKindManagedDirectory,
		)
	}
	if !value.Enabled {
		return source.Summary{}, fmt.Errorf(
			"%w: editable Collection Source is disabled",
			basespec.ErrConflict,
		)
	}
	return value, nil
}

func (a *API) publishDocument(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	address source.ManagedPackageAddress,
	document collectionv1.CollectionDocument,
	expectedGeneration string,
	allowPackageReplacement bool,
) (artifact.Artifact, error) {
	raw, digest, err := collectionDocumentPayload(document)
	if err != nil {
		return artifact.Artifact{}, err
	}
	locator, err := address.FileLocator(ManagedCollectionDocumentFile)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if _, err := a.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceID,
		locator,
		decoder.JSONDecoderID,
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
				collectionv1.CollectionType,
			),
			ExpectedLogicalName: basespec.LogicalName(document.Name),
			ExpectedDefinition:  digest,
			Package: source.ManagedPackagePublication{
				Address:            address,
				ExpectedGeneration: expectedGeneration,
				Files: []source.ManagedPackageFile{{
					Locator: ManagedCollectionDocumentFile,
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
	if err := ref.Validate(); err != nil {
		return editableCollection{}, err
	}

	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return editableCollection{}, err
	}
	if record.Kind != artifact.ArtifactKind(collectionv1.CollectionType) {
		return editableCollection{}, fmt.Errorf(
			"%w: Artifact %q is not a Collection",
			basespec.ErrUnsupported,
			record.ID,
		)
	}
	if expectedRevision != 0 && record.Revision != expectedRevision {
		return editableCollection{}, basespec.ErrConflict
	}
	if record.State != artifact.StateAvailable {
		return editableCollection{}, fmt.Errorf(
			"%w: Collection Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if record.Binding.SubresourceLocator != "" {
		return editableCollection{}, fmt.Errorf(
			"%w: contained Collection declarations are not editable managed Collections",
			basespec.ErrUnsupported,
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
			basespec.ErrUnsupported,
		)
	}
	if !sourceValue.Enabled {
		return editableCollection{}, fmt.Errorf(
			"%w: Collection Source is disabled",
			basespec.ErrReferenceUnresolved,
		)
	}
	if a.domain != nil &&
		sourceValue.StorageKey != a.domain.SourceStorageKey {
		return editableCollection{}, fmt.Errorf(
			"%w: Collection belongs to another managed domain Source",
			basespec.ErrReferenceUnresolved,
		)
	}

	address, err := managedCollectionAddressFromLocator(
		record.Binding.Locator,
	)
	if err != nil {
		return editableCollection{}, err
	}
	if address.Name != record.LogicalName {
		return editableCollection{}, fmt.Errorf(
			"%w: managed Collection package name does not match Artifact identity",
			basespec.ErrInvalid,
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
			basespec.ErrRefreshRequired,
		)
	}

	definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return editableCollection{}, err
	}
	document, err := collectionv1.DecodeCollectionJSON(
		definitionValue.Body,
	)
	if err != nil {
		return editableCollection{}, err
	}
	if document.Name != string(record.LogicalName) {
		return editableCollection{}, fmt.Errorf(
			"%w: Collection declaration name differs from Artifact identity",
			basespec.ErrInvalid,
		)
	}
	if document.Locator != nil {
		return editableCollection{}, fmt.Errorf(
			"%w: located Collection aliases are not editable managed Collections",
			basespec.ErrUnsupported,
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

func managedCollectionAddress(
	name basespec.LogicalName,
) (source.ManagedPackageAddress, error) {
	return source.NewManagedPackageAddress(
		ManagedCollectionPackageKind,
		name,
		ManagedCollectionVersion,
	)
}

func managedCollectionAddressFromLocator(
	locator basespec.Locator,
) (source.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(ManagedCollectionDocumentFile) {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: Collection locator %q is not %q",
			basespec.ErrUnsupported,
			locator,
			ManagedCollectionDocumentFile,
		)
	}
	address, err := source.ParseManagedPackageAddressDirectory(
		basespec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return source.ManagedPackageAddress{}, err
	}
	if address.Kind != ManagedCollectionPackageKind {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Collection package kind must be %q",
			basespec.ErrUnsupported,
			ManagedCollectionPackageKind,
		)
	}
	return address, nil
}

func collectionDocumentPayload(
	document collectionv1.CollectionDocument,
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

func collectionViewOf(
	record artifact.Artifact,
	document collectionv1.CollectionDocument,
) (CollectionView, error) {
	baseline := IsBaselineCollectionArtifact(record)
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

	header := declaration.Header{
		Type:        m.Type,
		Name:        string(m.Name),
		Description: m.Description,
		Metadata:    declaration.CloneRawMessageMap(m.Metadata),
	}
	if m.Locator != nil {
		locator := m.Locator.Clone()
		header.Locator = &locator
	}
	entry, err := declaration.NewEntry(struct {
		declaration.Header

		Server string `json:"server,omitempty"`
	}{
		Header: header,
		Server: string(m.Server),
	})
	if err != nil {
		return declaration.Entry{}, err
	}
	if err := entry.ValidateCompositionReference(); err != nil {
		return declaration.Entry{}, err
	}
	return entry, nil
}

func memberReferenceFromEntry(
	entry declaration.Entry,
) (MemberReference, error) {
	form, err := entry.CompositionForm()
	if err != nil {
		return MemberReference{}, err
	}
	if form != declaration.CompositionEntryReference {
		return MemberReference{}, fmt.Errorf(
			"%w: editable Collection members must be external references",
			basespec.ErrUnsupported,
		)
	}

	header := entry.Header()
	output := MemberReference{
		Type:        header.Type,
		Name:        basespec.LogicalName(header.Name),
		Description: header.Description,
		Metadata:    declaration.CloneRawMessageMap(header.Metadata),
	}
	if header.Locator != nil {
		locator := header.Locator.Clone()
		output.Locator = &locator
	}

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
		output.Server = basespec.LogicalName(selector.Server)
		if err := output.Server.Validate(); err != nil {
			return MemberReference{}, err
		}
	}
	return output, nil
}

func relativeSourceLocator(
	from basespec.Locator,
	target basespec.Locator,
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
	return 0, fmt.Errorf("%w: Collection member does not exist", basespec.ErrNotFound)
}
