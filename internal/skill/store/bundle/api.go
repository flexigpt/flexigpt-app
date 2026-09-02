package bundle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"sort"
	"sync/atomic"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/managedartifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/managed"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	skillArtifact "github.com/flexigpt/flexigpt-app/internal/skill/store/artifact"
)

type skillArtifactPolicy struct {
	ids              artifact.ArtifactIDProvider
	autoAdoptSources map[basespec.SourceID]struct{}
}

type API struct {
	dependencies Dependencies
	closed       atomic.Bool
}

func New(dependencies Dependencies) (*API, error) {
	if err := dependencies.Validate(); err != nil {
		return nil, err
	}
	if !dependencies.HasDecoder(artifactbuiltin.AgentSkillDecoderID) {
		return nil, fmt.Errorf(
			"%w: shared agent skill decoder %q is not registered",
			basespec.ErrDecoderUnavailable,
			artifactbuiltin.AgentSkillDecoderID,
		)
	}
	return &API{dependencies: dependencies}, nil
}

func (a *API) Close() error {
	if a != nil {
		a.closed.Store(true)
	}
	return nil
}

func (a *API) CreateBundle(
	ctx context.Context,
	request CreateBundleRequest,
) (Bundle, error) {
	return a.createBundle(ctx, request, false)
}

func (a *API) ListBundles(
	ctx context.Context,
	rootID basespec.RootID,
) ([]Bundle, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	values, err := a.dependencies.Collections.ListByRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}

	output := make([]Bundle, 0)
	for _, value := range values {
		if value.Kind != artifactbuiltin.SkillCollectionV1Kind {
			continue
		}
		bundle, err := a.GetBundle(ctx, value.Ref())
		if err != nil {
			return nil, err
		}
		output = append(output, bundle)
	}
	return output, nil
}

func (a *API) UpdateBundle(
	ctx context.Context,
	request UpdateBundleRequest,
) (Bundle, error) {
	if err := a.requireBundleMutation(ctx, request.Bundle.RootID, false); err != nil {
		return Bundle{}, err
	}
	current, err := a.GetBundle(ctx, request.Bundle)
	if err != nil {
		return Bundle{}, err
	}
	if request.ExpectedRevision == 0 ||
		current.Collection.Revision != request.ExpectedRevision {
		return Bundle{}, basespec.ErrConflict
	}

	data, err := EncodeCollectionData(current.Data)
	if err != nil {
		return Bundle{}, err
	}
	if _, err := a.dependencies.Collections.Update(
		ctx,
		request.Bundle,
		collection.Update{
			ExpectedRevision: request.ExpectedRevision,
			DisplayName:      request.DisplayName,
			Description:      request.Description,
			Enabled:          request.Enabled,
			Data:             data,
		},
	); err != nil {
		return Bundle{}, err
	}
	return a.GetBundle(ctx, request.Bundle)
}

func (a *API) RetireBundle(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) (collection.Collection, error) {
	if err := a.requireBundleMutation(ctx, ref.RootID, false); err != nil {
		return collection.Collection{}, err
	}
	if _, err := a.GetBundle(ctx, ref); err != nil {
		return collection.Collection{}, err
	}
	skills, err := a.ListSkills(ctx, ref)
	if err != nil {
		return collection.Collection{}, err
	}
	if len(skills) != 0 {
		return collection.Collection{}, fmt.Errorf(
			"%w: remove all Skills before retiring the bundle",
			basespec.ErrConflict,
		)
	}
	return a.dependencies.Collections.Retire(ctx, ref, expectedRevision)
}

func (a *API) PurgeBundle(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) error {
	if err := a.Ready(); err != nil {
		return err
	}
	if err := a.requireBundleMutation(ctx, ref.RootID, false); err != nil {
		return err
	}
	value, err := a.dependencies.Collections.GetRetired(ctx, ref)
	if err != nil {
		return err
	}
	if value.Kind != artifactbuiltin.SkillCollectionV1Kind {
		return fmt.Errorf(
			"%w: collection %q is not a retired skill bundle",
			basespec.ErrNotFound,
			ref.CollectionID,
		)
	}
	data, err := DecodeCollectionData(value.Data)
	if err != nil {
		return err
	}

	var ownedSource source.Summary
	if data.ManagedSourceID != "" {
		ownedSource, err = a.dependencies.Sources.Get(
			ctx,
			ref.RootID,
			data.ManagedSourceID,
		)
		if err != nil {
			return err
		}
		if ownedSource.Kind != artifactbuiltin.ManagedDirectorySourceKind {
			return fmt.Errorf(
				"%w: bundle-owned managed Source %q has kind %q, not %q",
				basespec.ErrInvalid,
				data.ManagedSourceID,
				ownedSource.Kind,
				artifactbuiltin.ManagedDirectorySourceKind,
			)
		}
	}
	if err := a.dependencies.Collections.Purge(ctx, ref, expectedRevision); err != nil {
		return err
	}
	if data.ManagedSourceID != "" {
		if err := a.dependencies.Sources.Discard(
			ctx,
			ref.RootID,
			data.ManagedSourceID,
			ownedSource.Revision,
		); err != nil {
			return fmt.Errorf(
				"bundle was purged but its owned managed Source remains pending cleanup: %w",
				err,
			)
		}
	}
	return nil
}

func (a *API) AttachSource(
	ctx context.Context,
	bundle collection.CollectionRef,
	expectedCollectionRevision uint64,
	draft AttachmentDraft,
) (Bundle, error) {
	if err := a.requireBundleMutation(ctx, bundle.RootID, false); err != nil {
		return Bundle{}, err
	}
	if draft.Role == artifactbuiltin.BuiltInAttachmentRole {
		return Bundle{}, fmt.Errorf(
			"%w: skill bundle built-in attachment role is reserved for bootstrap",
			basespec.ErrInvalid,
		)
	}
	if draft.Role == artifactbuiltin.ManagedAttachmentRole {
		return Bundle{}, fmt.Errorf(
			"%w: managed attachments must be provisioned through managedSourceID when the bundle is created",
			basespec.ErrInvalid,
		)
	}
	if _, err := a.GetBundle(ctx, bundle); err != nil {
		return Bundle{}, err
	}
	if err := a.validateAttachmentDraft(ctx, bundle.RootID, draft); err != nil {
		return Bundle{}, err
	}
	attachmentData, err := NewAttachmentData(
		draft.DiscoveryRoot,
		draft.ExpectedMemberDigests,
	)
	if err != nil {
		return Bundle{}, err
	}
	encodedAttachmentData, err := EncodeAttachmentData(attachmentData)
	if err != nil {
		return Bundle{}, err
	}

	_, _, err = a.dependencies.Collections.Attach(
		ctx,
		bundle,
		expectedCollectionRevision,
		collection.AttachmentDraft{
			SourceID: draft.SourceID,
			Role:     draft.Role,
			Enabled:  draft.Enabled,
			Data:     encodedAttachmentData,
		},
	)
	if err != nil {
		return Bundle{}, err
	}
	return a.GetBundle(ctx, bundle)
}

func (a *API) RefreshBundle(
	ctx context.Context,
	ref collection.CollectionRef,
) (refresh.Result, error) {
	return a.refreshBundle(ctx, ref, false)
}

// RefreshBuiltInBundle is the trusted feature-level refresh path for a
// protected built-in Skill Bundle. It is intentionally unavailable through
// ordinary Skill Bundle mutation APIs.
func (a *API) RefreshBuiltInBundle(
	ctx context.Context,
	ref collection.CollectionRef,
) (refresh.Result, error) {
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return refresh.Result{}, err
	}
	return a.refreshBundle(ctx, ref, true)
}

// EnsureBuiltInBundleCurrent preserves startup convergence without publishing
// a new catalog when the protected bundle is already current. It is reserved
// for the trusted built-in installer and explicit built-in update paths.
func (a *API) EnsureBuiltInBundleCurrent(
	ctx context.Context,
	ref collection.CollectionRef,
) error {
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}
	bundle, err := a.GetBundle(ctx, ref)
	if err != nil {
		return err
	}

	if _, err := a.currentBundleCatalog(ctx, bundle); err == nil {
		return nil
	} else if !errors.Is(err, basespec.ErrCatalogUnavailable) &&
		!errors.Is(err, basespec.ErrCatalogStale) {
		return err
	}

	_, err = a.refreshBundle(ctx, ref, true)
	return err
}

func (a *API) CreateManagedSkill(
	ctx context.Context,
	request CreateManagedSkillRequest,
) (CreateManagedSkillResponse, error) {
	return a.createManagedSkill(ctx, request, false)
}

// GetManagedSkillDocument reads the canonical definition for a managed Skill
// package. External and discovered Skills remain source-owned and intentionally
// do not expose an editable document through this API.
func (a *API) GetManagedSkillDocument(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ManagedSkillDocument, error) {
	if err := a.Ready(); err != nil {
		return ManagedSkillDocument{}, err
	}

	value, err := a.GetSkill(ctx, ref)
	if err != nil {
		return ManagedSkillDocument{}, err
	}
	if value.Adoption != artifact.AdoptionPinned {
		return ManagedSkillDocument{}, fmt.Errorf(
			"%w: only managed Skills can be edited",
			basespec.ErrUnsupported,
		)
	}

	bundleRef := collection.CollectionRef{
		RootID:       value.RootID,
		CollectionID: value.CollectionID,
	}
	bundle, err := a.GetBundle(ctx, bundleRef)
	if err != nil {
		return ManagedSkillDocument{}, err
	}
	attachment, _, err := managedAttachmentForRole(bundle, artifactbuiltin.ManagedAttachmentRole)
	if err != nil {
		return ManagedSkillDocument{}, err
	}
	if err := requireBundleOwnedManagedSource(bundle, attachment.SourceID); err != nil {
		return ManagedSkillDocument{}, err
	}
	if attachment.SourceID != value.Binding.SourceID {
		return ManagedSkillDocument{}, fmt.Errorf(
			"%w: Skill is not stored in this bundle's managed source",
			basespec.ErrUnsupported,
		)
	}
	if _, err := managedSkillPackageAddressOf(value.Binding); err != nil {
		return ManagedSkillDocument{}, err
	}
	if value.ResolvedDefinition == nil {
		return ManagedSkillDocument{}, fmt.Errorf(
			"%w: managed Skill has no current definition",
			basespec.ErrReferenceUnresolved,
		)
	}

	definitionValue, err := a.currentDefinitionForArtifact(ctx, value)
	if err != nil {
		return ManagedSkillDocument{}, err
	}
	doc, err := skillArtifact.DocumentFromDefinition(definitionValue)
	if err != nil {
		return ManagedSkillDocument{}, err
	}
	return ManagedSkillDocument{Artifact: value.Clone(), Document: doc}, nil
}

func (a *API) AdoptSkill(
	ctx context.Context,
	request AdoptSkillRequest,
) (artifact.Artifact, error) {
	if err := a.requireBundleMutation(ctx, request.Bundle.RootID, false); err != nil {
		return artifact.Artifact{}, err
	}
	bundle, err := a.GetBundle(ctx, request.Bundle)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if request.Occurrence.CollectionID != request.Bundle.CollectionID {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: skill occurrence belongs to another bundle",
			basespec.ErrInvalid,
		)
	}
	role, err := bundleAttachmentRole(bundle, request.Occurrence.SourceID)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if role == artifactbuiltin.ManagedAttachmentRole || role == artifactbuiltin.BuiltInAttachmentRole {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: managed and built-in Skill occurrences require their dedicated installation flow",
			basespec.ErrUnsupported,
		)
	}
	if request.ExpectedCatalogRevision == 0 {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: expected skill bundle catalog revision is required",
			basespec.ErrInvalid,
		)
	}

	snapshot, err := a.currentBundleCatalog(ctx, bundle)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if snapshot.Revision != request.ExpectedCatalogRevision {
		return artifact.Artifact{}, basespec.ErrConflict
	}
	found := false
	for _, occurrence := range snapshot.Occurrences {
		if occurrence.Key != request.Occurrence {
			continue
		}
		if occurrence.Kind != artifactbuiltin.AgentSkillArtifactKind ||
			occurrence.State != catalog.OccurrenceValid ||
			occurrence.DefinitionDigest == nil {
			return artifact.Artifact{}, fmt.Errorf(
				"%w: requested occurrence is not an adoptable %q Skill",
				basespec.ErrReferenceUnresolved,
				artifactbuiltin.AgentSkillArtifactKind,
			)
		}
		found = true
		break
	}
	if !found {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: requested Skill occurrence is not in the current bundle catalog",
			basespec.ErrReferenceUnresolved,
		)
	}

	if err := basespec.ValidateArtifactID(request.ArtifactID); err != nil {
		return artifact.Artifact{}, err
	}
	return a.dependencies.Artifacts.Adopt(ctx, artifact.AdoptRequest{
		ArtifactID:              request.ArtifactID,
		Collection:              request.Bundle,
		Occurrence:              request.Occurrence,
		ExpectedCatalogRevision: request.ExpectedCatalogRevision,
		Name:                    request.Name,
		Enabled:                 request.Enabled,
		Data:                    emptyArtifactData(),
	})
}

func (a *API) PinSkill(
	ctx context.Context,
	request PinSkillRequest,
) (artifact.Artifact, error) {
	if err := a.requireBundleMutation(ctx, request.Bundle.RootID, false); err != nil {
		return artifact.Artifact{}, err
	}
	bundle, err := a.GetBundle(ctx, request.Bundle)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if request.Binding.ExpectedKind != artifactbuiltin.AgentSkillArtifactKind {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: skill bundle pins only support %q",
			basespec.ErrInvalid,
			artifactbuiltin.AgentSkillArtifactKind,
		)
	}
	role, err := bundleAttachmentRole(bundle, request.Binding.SourceID)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if role == artifactbuiltin.ManagedAttachmentRole ||
		role == artifactbuiltin.BuiltInAttachmentRole {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: managed and built-in attachments require their dedicated pinning flow",
			basespec.ErrUnsupported,
		)
	}
	return a.dependencies.Artifacts.Pin(ctx, artifact.PinRequest{
		ArtifactID:                 request.ArtifactID,
		Collection:                 request.Bundle,
		ExpectedCollectionRevision: request.ExpectedCollectionRevision,
		Binding:                    request.Binding,
		Name:                       request.Name,
		Enabled:                    request.Enabled,
		Data:                       emptyArtifactData(),
	})
}

func (a *API) ListSkills(
	ctx context.Context,
	bundle collection.CollectionRef,
) ([]artifact.Artifact, error) {
	if _, err := a.GetBundle(ctx, bundle); err != nil {
		return nil, err
	}
	values, err := a.dependencies.Artifacts.ListByCollection(ctx, bundle)
	if err != nil {
		return nil, err
	}
	output := make([]artifact.Artifact, 0, len(values))
	for _, value := range values {
		if value.Kind == artifactbuiltin.AgentSkillArtifactKind {
			output = append(output, value)
		}
	}
	return output, nil
}

func (a *API) SetSkillEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if err := a.requireBundleMutation(ctx, ref.RootID, false); err != nil {
		return artifact.Artifact{}, err
	}
	if _, err := a.GetSkill(ctx, ref); err != nil {
		return artifact.Artifact{}, err
	}
	return a.dependencies.Artifacts.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
}

func (a *API) UnadoptSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	suppress bool,
) error {
	if err := a.requireBundleMutation(ctx, ref.RootID, false); err != nil {
		return err
	}
	if _, err := a.GetSkill(ctx, ref); err != nil {
		return err
	}
	return a.dependencies.Artifacts.Unadopt(
		ctx,
		ref,
		expectedRevision,
		suppress,
	)
}

func (a *API) PurgeSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Skill Artifact revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := a.requireBundleMutation(ctx, ref.RootID, false); err != nil {
		return err
	}
	value, err := a.GetSkill(ctx, ref)
	if err != nil {
		return err
	}
	if value.Revision != expectedRevision {
		return basespec.ErrConflict
	}

	bundle, err := a.GetBundle(ctx, collection.CollectionRef{
		RootID:       value.RootID,
		CollectionID: value.CollectionID,
	})
	if err != nil {
		return err
	}
	var role basespec.AttachmentRole
	for _, attachment := range bundle.Attachments {
		if attachment.SourceID == value.Binding.SourceID {
			role = attachment.Role
			break
		}
	}
	switch role {
	case artifactbuiltin.BuiltInAttachmentRole:
		return fmt.Errorf(
			"%w: built-in Skill packages are protected and read-only",
			basespec.ErrProtected,
		)
	case artifactbuiltin.ManagedAttachmentRole:
		if value.Adoption != artifact.AdoptionPinned {
			return fmt.Errorf(
				"%w: managed Skill Artifact must remain pinned until purged",
				basespec.ErrConflict,
			)
		}
		if err := requireBundleOwnedManagedSource(bundle, value.Binding.SourceID); err != nil {
			return err
		}
	default:
		if value.Adoption == artifact.AdoptionObserved {
			return a.dependencies.Artifacts.Unadopt(
				ctx,
				ref,
				expectedRevision,
				true,
			)
		}
		return a.dependencies.Artifacts.PurgeAndSuppress(ctx, ref, expectedRevision)
	}

	packageAddress, err := managedSkillPackageAddressOf(value.Binding)
	if err != nil {
		return err
	}
	plan, err := a.discoveryPlan(bundle)
	if err != nil {
		return err
	}
	if err := a.dependencies.ManagedArtifacts.Remove(
		ctx,
		managedartifact.RemoveRequest{
			Artifact:       value,
			Package:        packageAddress,
			Plan:           plan,
			RefreshPolicy:  a.refreshPolicy(bundle, false),
			AllowProtected: false,
		},
	); err != nil {
		return pendingManagedSkillPurgeError(value.Ref(), err)
	}
	return nil
}

func (a *API) GetSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	if err := a.Ready(); err != nil {
		return artifact.Artifact{}, err
	}
	value, err := a.dependencies.Artifacts.Get(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if value.Kind != artifactbuiltin.AgentSkillArtifactKind {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: artifact %q is not an agent skill",
			basespec.ErrNotFound,
			ref.ArtifactID,
		)
	}
	if _, err := a.GetBundle(ctx, collection.CollectionRef{
		RootID:       value.RootID,
		CollectionID: value.CollectionID,
	}); err != nil {
		return artifact.Artifact{}, err
	}
	return value, nil
}

func (a *API) EnsureBuiltInBundleTopology(
	ctx context.Context,
	request BuiltInBundleTopology,
) (Bundle, error) {
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return Bundle{}, err
	}
	bundle, err := a.createBundle(ctx, CreateBundleRequest{
		RootID:         request.RootID,
		CollectionID:   request.CollectionID,
		LogicalName:    request.LogicalName,
		LogicalVersion: request.LogicalVersion,
		DisplayName:    request.DisplayName,
		Description:    request.Description,
		Labels:         request.Labels,
		Enabled:        request.Enabled,
		Attachments: []AttachmentDraft{{
			SourceID:              request.SourceID,
			Role:                  artifactbuiltin.BuiltInAttachmentRole,
			Enabled:               true,
			DiscoveryRoot:         request.DiscoveryRoot,
			ExpectedMemberDigests: request.ExpectedMemberDigests,
		}},
	}, true)
	if err != nil {
		return Bundle{}, err
	}
	if !builtInBundleTopologyMatches(bundle, request) {
		data, err := EncodeCollectionData(CollectionData{
			SchemaVersion:           artifactbuiltin.SkillCollectionV1SchemaVersion,
			DiscoveryPolicyRevision: DiscoveryPolicyRevision,
			LogicalName:             request.LogicalName,
			LogicalVersion:          request.LogicalVersion,
			Labels:                  request.Labels,
		})
		if err != nil {
			return Bundle{}, err
		}

		if bundle.Collection.DisplayName != request.DisplayName ||
			bundle.Collection.Description != request.Description ||
			bundle.Collection.Enabled != request.Enabled ||
			!bytes.Equal(bundle.Collection.Data, data) {
			if _, err := a.dependencies.Collections.Update(
				ctx,
				bundle.Collection.Ref(),
				collection.Update{
					ExpectedRevision: bundle.Collection.Revision,
					DisplayName:      request.DisplayName,
					Description:      request.Description,
					Enabled:          request.Enabled,
					Data:             data,
				},
			); err != nil {
				return Bundle{}, err
			}
		}

		if len(bundle.Attachments) == 1 {
			attachment := bundle.Attachments[0]
			attachmentData, err := NewAttachmentData(
				request.DiscoveryRoot,
				request.ExpectedMemberDigests,
			)
			if err != nil {
				return Bundle{}, err
			}
			encodedAttachmentData, err := EncodeAttachmentData(attachmentData)
			if err != nil {
				return Bundle{}, err
			}
			if attachment.Role != artifactbuiltin.BuiltInAttachmentRole || !attachment.Enabled ||
				!bytes.Equal(attachment.Data, encodedAttachmentData) {
				currentColl, err := a.dependencies.Collections.Get(ctx, bundle.Collection.Ref())
				if err != nil {
					return Bundle{}, err
				}
				if _, _, err := a.dependencies.Collections.UpdateAttachment(
					ctx,
					bundle.Collection.Ref(),
					request.SourceID,
					collection.AttachmentUpdate{
						ExpectedCollectionRevision: currentColl.Revision,
						ExpectedAttachmentRevision: attachment.Revision,
						Role:                       artifactbuiltin.BuiltInAttachmentRole,
						Enabled:                    true,
						Data:                       encodedAttachmentData,
					},
				); err != nil {
					return Bundle{}, err
				}
			}
		}

		bundle, err = a.GetBundle(ctx, bundle.Collection.Ref())
		if err != nil {
			return Bundle{}, err
		}
		if !builtInBundleTopologyMatches(bundle, request) {
			return Bundle{}, fmt.Errorf(
				"%w: built-in bundle %q differs from the protected registry declaration",
				basespec.ErrConflict,
				request.LogicalName,
			)
		}
	}

	return bundle, nil
}

func (a *API) InstallBuiltInSkill(
	ctx context.Context,
	_ CreateManagedSkillRequest,
) (CreateManagedSkillResponse, error) {
	if err := a.Ready(); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	return CreateManagedSkillResponse{}, fmt.Errorf(
		"%w: built-in Skills must be installed through InstallBuiltInCollection",
		basespec.ErrUnsupported,
	)
}

func (a *API) GetBundle(
	ctx context.Context,
	ref collection.CollectionRef,
) (Bundle, error) {
	if err := a.Ready(); err != nil {
		return Bundle{}, err
	}
	if err := ref.Validate(); err != nil {
		return Bundle{}, err
	}

	value, err := a.dependencies.Collections.Get(ctx, ref)
	if err != nil {
		return Bundle{}, err
	}
	if value.Kind != artifactbuiltin.SkillCollectionV1Kind {
		return Bundle{}, fmt.Errorf(
			"%w: collection %q is not a skill bundle",
			basespec.ErrNotFound,
			ref.CollectionID,
		)
	}

	data, err := DecodeCollectionData(value.Data)
	if err != nil {
		return Bundle{}, err
	}
	attachments, err := a.dependencies.Collections.ListAttachments(ctx, ref)
	if err != nil {
		return Bundle{}, err
	}

	sources := make([]source.Summary, 0, len(attachments))
	for _, attachment := range attachments {
		if err := a.validateAttachment(ctx, ref.RootID, attachment); err != nil {
			return Bundle{}, err
		}
		value, err := a.dependencies.Sources.Get(
			ctx,
			ref.RootID,
			attachment.SourceID,
		)
		if err != nil {
			return Bundle{}, err
		}
		sources = append(sources, value)
	}
	if err := validateBundleAttachmentTopology(data, attachments); err != nil {
		return Bundle{}, err
	}
	sort.Slice(attachments, func(left, right int) bool {
		return attachments[left].SourceID < attachments[right].SourceID
	})
	sort.Slice(sources, func(left, right int) bool {
		return sources[left].ID < sources[right].ID
	})

	return Bundle{
		Collection:  value,
		Data:        data,
		Attachments: attachments,
		Sources:     sources,
	}, nil
}

func (a *API) Ready() error {
	if a == nil || a.closed.Load() {
		return basespec.ErrClosed
	}
	return a.dependencies.Validate()
}

func (a *API) currentBundleCatalog(
	ctx context.Context,
	bundle Bundle,
) (catalog.Snapshot, error) {
	snapshot, err := catalog.ReadCurrent(
		ctx,
		a.dependencies.Catalogs,
		bundle.Collection.Ref(),
	)
	if err != nil {
		return catalog.Snapshot{}, err
	}

	plan, err := a.discoveryPlan(bundle)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	planFingerprint, err := plan.Fingerprint()
	if err != nil {
		return catalog.Snapshot{}, err
	}
	decoderFingerprint, err := a.dependencies.DecoderFingerprint()
	if err != nil {
		return catalog.Snapshot{}, err
	}
	if snapshot.PlanFingerprint != planFingerprint ||
		snapshot.DecoderFingerprint != decoderFingerprint {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: Skill Bundle catalog capability inputs changed",
			basespec.ErrCatalogStale,
		)
	}
	return snapshot, nil
}

func (a *API) currentDefinitionForArtifact(
	ctx context.Context,
	record artifact.Artifact,
) (definition.Definition, error) {
	if record.ResolvedDefinition == nil {
		return definition.Definition{}, fmt.Errorf(
			"%w: Skill Artifact %q has no current definition",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	snapshot, err := catalog.ReadCurrent(
		ctx,
		a.dependencies.Catalogs,
		collection.CollectionRef{
			RootID:       record.RootID,
			CollectionID: record.CollectionID,
		},
	)
	if err != nil {
		return definition.Definition{}, err
	}

	return definitionForArtifact(snapshot, record)
}

// createBundle keeps the built-in attachment role inside trusted bootstrap
// composition. Public bundle creation must not mint built-in provenance.
func (a *API) createBundle(
	ctx context.Context,
	request CreateBundleRequest,
	allowBuiltInAttachment bool,
) (Bundle, error) {
	if err := a.Ready(); err != nil {
		return Bundle{}, err
	}
	if err := basespec.ValidateCollectionID(request.CollectionID); err != nil {
		return Bundle{}, err
	}
	if err := basespec.ValidateRootID(request.RootID); err != nil {
		return Bundle{}, err
	}
	if err := a.requireBundleMutation(
		ctx,
		request.RootID,
		allowBuiltInAttachment,
	); err != nil {
		return Bundle{}, err
	}

	if request.ManagedSourceID != "" {
		if err := basespec.ValidateStorageKey(request.ManagedSourceStorageKey); err != nil {
			return Bundle{}, err
		}
	}

	data, err := EncodeCollectionData(CollectionData{
		SchemaVersion:           artifactbuiltin.SkillCollectionV1SchemaVersion,
		DiscoveryPolicyRevision: DiscoveryPolicyRevision,
		LogicalName:             request.LogicalName,
		LogicalVersion:          request.LogicalVersion,
		Labels:                  request.Labels,
		ManagedSourceID:         request.ManagedSourceID,
	})
	if err != nil {
		return Bundle{}, err
	}

	attachments := make([]collection.AttachmentDraft, 0, len(request.Attachments))
	roleCounts := make(map[basespec.AttachmentRole]int)
	for _, draft := range request.Attachments {
		if draft.Role == artifactbuiltin.BuiltInAttachmentRole &&
			!allowBuiltInAttachment {
			return Bundle{}, fmt.Errorf(
				"%w: skill bundle built-in attachment role is reserved for bootstrap",
				basespec.ErrInvalid,
			)
		}
		if draft.Role == artifactbuiltin.ManagedAttachmentRole {
			return Bundle{}, fmt.Errorf(
				"%w: managed attachments must be provisioned through managedSourceID",
				basespec.ErrInvalid,
			)
		}
		if request.ManagedSourceID != "" &&
			draft.SourceID == request.ManagedSourceID {
			return Bundle{}, fmt.Errorf(
				"%w: managedSourceID is provisioned and attached by bundle creation and must not also be an explicit attachment",
				basespec.ErrInvalid,
			)
		}
		roleCounts[draft.Role]++
		if (draft.Role == artifactbuiltin.ManagedAttachmentRole ||
			draft.Role == artifactbuiltin.BuiltInAttachmentRole) &&
			roleCounts[draft.Role] > 1 {
			return Bundle{}, fmt.Errorf(
				"%w: skill bundle can have only one %q attachment",
				basespec.ErrInvalid,
				draft.Role,
			)
		}
		if err := a.validateAttachmentDraft(ctx, request.RootID, draft); err != nil {
			return Bundle{}, err
		}
		attachmentData, err := NewAttachmentData(
			draft.DiscoveryRoot,
			draft.ExpectedMemberDigests,
		)
		if err != nil {
			return Bundle{}, err
		}
		encodedAttachmentData, err := EncodeAttachmentData(attachmentData)
		if err != nil {
			return Bundle{}, err
		}
		attachments = append(attachments, collection.AttachmentDraft{
			SourceID: draft.SourceID,
			Role:     draft.Role,
			Enabled:  draft.Enabled,
			Data:     encodedAttachmentData,
		})
	}

	var provisionedSource *source.Summary
	if request.ManagedSourceID != "" {
		if allowBuiltInAttachment {
			return Bundle{}, fmt.Errorf(
				"%w: protected bundle topology must declare its attachment explicitly",
				basespec.ErrInvalid,
			)
		}

		value, createdNew, err := a.dependencies.Sources.CreateWithStatus(
			ctx,
			request.RootID,
			source.Draft{
				ID:          request.ManagedSourceID,
				StorageKey:  request.ManagedSourceStorageKey,
				Kind:        artifactbuiltin.ManagedDirectorySourceKind,
				DisplayName: request.DisplayName,
				Enabled:     true,
				Config:      json.RawMessage(jsonutil.EmptyObject),
			},
		)
		if err != nil {
			return Bundle{}, err
		}
		if createdNew {
			provisionedSource = &value
		}

		attachmentData, err := NewAttachmentData(".", nil)
		if err != nil {
			return Bundle{}, err
		}
		encodedAttachmentData, err := EncodeAttachmentData(attachmentData)
		if err != nil {
			return Bundle{}, err
		}
		attachments = append(attachments, collection.AttachmentDraft{
			SourceID: request.ManagedSourceID,
			Role:     artifactbuiltin.ManagedAttachmentRole,
			Enabled:  true,
			Data:     encodedAttachmentData,
		})
	}

	cleanupProvisionedSource := func(cause error) error {
		if provisionedSource == nil {
			return cause
		}
		cleanupErr := a.dependencies.Sources.Discard(
			context.WithoutCancel(ctx),
			request.RootID,
			provisionedSource.ID,
			provisionedSource.Revision,
		)
		return errors.Join(cause, cleanupErr)
	}

	created, _, err := a.dependencies.Collections.Create(
		ctx,
		request.RootID,
		collection.Draft{
			ID:          request.CollectionID,
			Kind:        artifactbuiltin.SkillCollectionV1Kind,
			DisplayName: request.DisplayName,
			Description: request.Description,
			Enabled:     request.Enabled,
			Data:        data,
		},
		attachments,
	)
	if err != nil {
		return Bundle{}, cleanupProvisionedSource(err)
	}
	bundle, err := a.GetBundle(ctx, created.Ref())
	if err != nil {
		return Bundle{}, err
	}
	if provisionedSource != nil {
		attached := false
		for _, attachment := range bundle.Attachments {
			if attachment.SourceID == provisionedSource.ID &&
				attachment.Role == artifactbuiltin.ManagedAttachmentRole {
				attached = true
				break
			}
		}
		if !attached {
			return Bundle{}, cleanupProvisionedSource(fmt.Errorf(
				"%w: provisioned managed Source was not attached to the bundle",
				basespec.ErrConflict,
			))
		}
	}
	if !bundleCreationIntentMatches(bundle, request) {
		return Bundle{}, cleanupProvisionedSource(fmt.Errorf(
			"%w: skill bundle %q creation intent differs",
			basespec.ErrConflict,
			request.CollectionID,
		))
	}
	return bundle, nil
}

func bundleCreationIntentMatches(
	value Bundle,
	request CreateBundleRequest,
) bool {
	if value.Collection.RootID != request.RootID ||
		value.Collection.ID != request.CollectionID ||
		value.Collection.Kind != artifactbuiltin.SkillCollectionV1Kind {
		return false
	}
	if value.Collection.ID != request.CollectionID ||
		value.Collection.DisplayName != request.DisplayName ||
		value.Collection.Description != request.Description ||
		value.Collection.Enabled != request.Enabled ||
		value.Data.LogicalName != request.LogicalName ||
		value.Data.LogicalVersion != request.LogicalVersion ||
		value.Data.ManagedSourceID != request.ManagedSourceID ||
		!maps.Equal(value.Data.Labels, request.Labels) {
		return false
	}

	expected := make(map[basespec.SourceID]AttachmentDraft)
	for _, draft := range request.Attachments {
		if _, duplicate := expected[draft.SourceID]; duplicate {
			return false
		}
		expected[draft.SourceID] = draft
	}
	if request.ManagedSourceID != "" {
		if _, duplicate := expected[request.ManagedSourceID]; duplicate {
			return false
		}
		expected[request.ManagedSourceID] = AttachmentDraft{
			SourceID:      request.ManagedSourceID,
			Role:          artifactbuiltin.ManagedAttachmentRole,
			Enabled:       true,
			DiscoveryRoot: ".",
		}
	}
	if len(expected) != len(value.Attachments) {
		return false
	}

	for _, attachment := range value.Attachments {
		draft, found := expected[attachment.SourceID]
		if !found ||
			attachment.Role != draft.Role ||
			attachment.Enabled != draft.Enabled {
			return false
		}
		actualData, err := DecodeAttachmentData(attachment.Data)
		if err != nil {
			return false
		}
		expectedData, err := NewAttachmentData(
			draft.DiscoveryRoot,
			draft.ExpectedMemberDigests,
		)
		if err != nil ||
			actualData.DiscoveryRoot != expectedData.DiscoveryRoot ||
			!maps.Equal(
				actualData.ExpectedMemberDigests,
				expectedData.ExpectedMemberDigests,
			) {
			return false
		}
	}

	if request.ManagedSourceID != "" {
		found := false
		for _, sourceValue := range value.Sources {
			if sourceValue.ID != request.ManagedSourceID {
				continue
			}
			found = sourceValue.RootID == request.RootID &&
				sourceValue.Kind == artifactbuiltin.ManagedDirectorySourceKind &&
				sourceValue.DisplayName == request.DisplayName &&
				sourceValue.Enabled
			break
		}
		if !found {
			return false
		}
	}
	return true
}

func builtInBundleTopologyMatches(
	value Bundle,
	request BuiltInBundleTopology,
) bool {
	if value.Collection.RootID != request.RootID ||
		value.Collection.ID != request.CollectionID ||
		value.Collection.DisplayName != request.DisplayName ||
		value.Collection.Description != request.Description ||
		value.Collection.Enabled != request.Enabled ||
		value.Data.LogicalName != request.LogicalName ||
		value.Data.LogicalVersion != request.LogicalVersion ||
		value.Data.ManagedSourceID != "" ||
		!maps.Equal(value.Data.Labels, request.Labels) ||
		len(value.Attachments) != 1 {
		return false
	}

	attachment := value.Attachments[0]
	expectedAttachment, err := NewAttachmentData(
		request.DiscoveryRoot,
		request.ExpectedMemberDigests,
	)
	if err != nil {
		return false
	}
	actualAttachment, err := DecodeAttachmentData(attachment.Data)
	if err != nil {
		return false
	}
	return attachment.RootID == request.RootID &&
		attachment.CollectionID == request.CollectionID &&
		attachment.SourceID == request.SourceID &&
		attachment.Role == artifactbuiltin.BuiltInAttachmentRole &&
		attachment.Enabled &&
		actualAttachment.DiscoveryRoot ==
			expectedAttachment.DiscoveryRoot &&
		maps.Equal(
			actualAttachment.ExpectedMemberDigests,
			expectedAttachment.ExpectedMemberDigests,
		)
}

func (a *API) refreshBundle(
	ctx context.Context,
	ref collection.CollectionRef,
	allowProtected bool,
) (refresh.Result, error) {
	if err := a.requireBundleMutation(
		ctx,
		ref.RootID,
		allowProtected,
	); err != nil {
		return refresh.Result{}, err
	}
	bundle, err := a.GetBundle(ctx, ref)
	if err != nil {
		return refresh.Result{}, err
	}
	if !bundle.Collection.Enabled {
		return refresh.Result{}, fmt.Errorf(
			"%w: skill bundle %q is disabled",
			basespec.ErrConflict,
			ref.CollectionID,
		)
	}

	plan, err := a.discoveryPlan(bundle)
	if err != nil {
		return refresh.Result{}, err
	}
	policy := a.refreshPolicy(bundle, allowProtected)
	return a.dependencies.Refresh.Refresh(ctx, ref, plan, policy)
}

// createManagedSkill performs the managed package publication workflow.
//
// Built-in package bootstrap is the only caller permitted to publish through
// a RoleBuiltIn attachment. User-facing creation always uses RoleManaged.
func (a *API) createManagedSkill(
	ctx context.Context,
	request CreateManagedSkillRequest,
	allowBuiltInAttachment bool,
) (CreateManagedSkillResponse, error) {
	if err := a.Ready(); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if err := request.Bundle.Validate(); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if err := a.requireBundleMutation(
		ctx,
		request.Bundle.RootID,
		allowBuiltInAttachment,
	); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if err := basespec.ValidateArtifactID(request.ArtifactID); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if err := basespec.ValidateLogicalName(basespec.LogicalName(request.SkillName)); err != nil {
		return CreateManagedSkillResponse{}, err
	}

	skillMDInput := append([]byte(nil), request.SKILLMD...)
	if request.Document != nil {
		if len(skillMDInput) != 0 {
			return CreateManagedSkillResponse{}, fmt.Errorf(
				"%w: exactly one of SKILLMD or Document may be supplied",
				basespec.ErrInvalid,
			)
		}
		doc := *request.Document
		if doc.Name == "" {
			doc.Name = request.SkillName
		}
		if doc.Name != request.SkillName {
			return CreateManagedSkillResponse{}, fmt.Errorf(
				"%w: structured Skill name does not match requested skill name",
				basespec.ErrInvalid,
			)
		}
		encoded, err := document.MarshalSkillDocument(doc)
		if err != nil {
			return CreateManagedSkillResponse{}, err
		}
		skillMDInput = append([]byte(nil), encoded...)
	}
	files, skillMD, err := normalizeManagedSkillFiles(
		skillMDInput,
		request.Files,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}
	definitionValue, _, err := skillArtifact.DecodeSkillDocument(skillMD,
		request.SkillName,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}
	packageSHA256, err := managedSkillPackageDigest(files)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}
	pending := func(cause error) error {
		return pendingManagedSkillCreateError(request.ArtifactID, cause)
	}

	if definitionValue.LogicalName != basespec.LogicalName(request.SkillName) {
		return CreateManagedSkillResponse{}, fmt.Errorf(
			"%w: SKILL.md name does not match requested skill name",
			basespec.ErrInvalid,
		)
	}

	bundle, err := a.GetBundle(ctx, request.Bundle)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if request.ExpectedCollectionRevision == 0 {
		return CreateManagedSkillResponse{}, fmt.Errorf(
			"%w: expected Skill Bundle revision is required",
			basespec.ErrInvalid,
		)
	}
	if bundle.Collection.Revision != request.ExpectedCollectionRevision {
		return CreateManagedSkillResponse{}, basespec.ErrConflict
	}
	if !bundle.Collection.Enabled {
		return CreateManagedSkillResponse{}, fmt.Errorf(
			"%w: Skill Bundle %q is disabled",
			basespec.ErrConflict,
			request.Bundle.CollectionID,
		)
	}

	targetRole := artifactbuiltin.ManagedAttachmentRole
	if allowBuiltInAttachment {
		targetRole = artifactbuiltin.BuiltInAttachmentRole
	}
	attachment, sourceValue, err := managedAttachmentForRole(
		bundle,
		targetRole,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if targetRole == artifactbuiltin.ManagedAttachmentRole {
		if err := requireBundleOwnedManagedSource(bundle, sourceValue.ID); err != nil {
			return CreateManagedSkillResponse{}, err
		}
	}
	if !attachment.Enabled || !sourceValue.Enabled {
		return CreateManagedSkillResponse{}, fmt.Errorf(
			"%w: managed Skill source is disabled",
			basespec.ErrConflict,
		)
	}
	packageAddress, err := skillArtifact.ManagedPackageAddressForSkill(
		definitionValue.LogicalName,
		definitionValue.LogicalVersion,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}
	skillLocator, err := skillArtifact.ManagedPackageLocatorForSkill(packageAddress)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if err := basespec.ValidatePortableLocator(skillLocator, false); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if err := managed.ValidatePackagePublication(
		source.ManagedPackagePublication{
			Address: packageAddress,
			Files:   files,
		},
	); err != nil {
		return CreateManagedSkillResponse{}, err
	}

	artifactName := definitionValue.DisplayName
	if artifactName == "" {
		artifactName = request.SkillName
	}
	packageData, err := encodeManagedSkillArtifactData(
		newManagedSkillArtifactData(packageSHA256, request.Enabled),
	)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}

	pinned, err := a.managedSkillByID(
		ctx,
		request.Bundle.RootID,
		request.ArtifactID,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if pinned == nil {
		if request.ExpectedArtifactRevision != 0 {
			return CreateManagedSkillResponse{}, basespec.ErrConflict
		}

		value, pinErr := a.dependencies.Artifacts.Pin(ctx, artifact.PinRequest{
			ArtifactID:                 request.ArtifactID,
			Collection:                 request.Bundle,
			ExpectedCollectionRevision: request.ExpectedCollectionRevision,
			Binding: artifact.SourceBinding{
				SourceID:     sourceValue.ID,
				Locator:      skillLocator,
				ExpectedKind: artifactbuiltin.AgentSkillArtifactKind,
			},
			Name:    artifactName,
			Enabled: request.Enabled,
			Data:    packageData,
		})
		switch {
		case pinErr == nil:
			pinned = &value

		case errors.Is(pinErr, basespec.ErrConflict):
			// Another caller may have created the same caller-supplied Artifact
			// ID between the read and the pin attempt.
			pinned, err = a.managedSkillByID(
				ctx,
				request.Bundle.RootID,
				request.ArtifactID,
			)
			if err != nil {
				return CreateManagedSkillResponse{}, err
			}
			if pinned == nil {
				return CreateManagedSkillResponse{}, pinErr
			}

		default:
			return CreateManagedSkillResponse{}, pinErr
		}
	}

	if err := validateManagedSkillOperationIntent(
		*pinned,
		request.Bundle,
		request.ArtifactID,
		sourceValue.ID,
		skillLocator,
	); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if request.ExpectedArtifactRevision != 0 &&
		pinned.Revision != request.ExpectedArtifactRevision {
		return CreateManagedSkillResponse{}, fmt.Errorf(
			"%w: managed Skill Artifact %q changed since it was read",
			basespec.ErrConflict,
			request.ArtifactID,
		)
	}

	intent, intentErr := decodeManagedSkillArtifactData(pinned.Data)
	samePackageIntent := intentErr == nil &&
		intent.PackageSHA256 == packageSHA256 &&
		*intent.Enabled == request.Enabled

	if request.ExpectedArtifactRevision == 0 {
		// Zero is the initial-create token and an idempotent replay token.
		// The persisted package data is the durable operation marker.
		if !samePackageIntent ||
			pinned.Name != artifactName {
			return CreateManagedSkillResponse{}, fmt.Errorf(
				"%w: managed Skill Artifact %q already exists with different creation intent",
				basespec.ErrConflict,
				request.ArtifactID,
			)
		}
	} else if !samePackageIntent {

		// Persist full package intent before touching source-side storage.
		// If publication or refresh later fails, this marker permits a retry
		// using the original Artifact revision.
		updated, err := a.dependencies.Artifacts.UpdateData(
			ctx,
			pinned.Ref(),
			pinned.Revision,
			packageData,
		)
		if err != nil {
			return CreateManagedSkillResponse{}, err
		}
		pinned = &updated
	}

	if pinned.Name != artifactName {
		updated, err := a.dependencies.Artifacts.SetName(
			ctx,
			pinned.Ref(),
			pinned.Revision,
			artifactName,
		)
		if err != nil {
			return CreateManagedSkillResponse{}, err
		}
		pinned = &updated
	}

	if pinned.Enabled != request.Enabled {
		updated, err := a.dependencies.Artifacts.SetEnabled(
			ctx,
			pinned.Ref(),
			pinned.Revision,
			request.Enabled,
		)
		if err != nil {
			return CreateManagedSkillResponse{}, err
		}
		pinned = &updated
	}

	plan, err := a.discoveryPlan(bundle)
	if err != nil {
		return CreateManagedSkillResponse{}, pending(err)
	}
	published, err := a.dependencies.ManagedArtifacts.Publish(
		ctx,
		managedartifact.PublishRequest{
			Artifact:           *pinned,
			ExpectedDefinition: definitionValue.Digest,
			Package: source.ManagedPackagePublication{
				Address: packageAddress,
				Files:   files,
			},
			Plan:           plan,
			RefreshPolicy:  a.refreshPolicy(bundle, allowBuiltInAttachment),
			AllowProtected: allowBuiltInAttachment,
		},
	)
	if err != nil {
		return CreateManagedSkillResponse{}, pending(err)
	}
	if result, complete := managedSkillCreateResult(
		published.Artifact,
		sourceValue.ID,
		skillLocator,
		definitionValue.Digest,
		packageSHA256,
	); complete {
		if published.Artifact.Name != artifactName ||
			published.Artifact.Enabled != request.Enabled {
			return CreateManagedSkillResponse{}, pending(fmt.Errorf(
				"%w: managed Skill Artifact changed during package publication",
				basespec.ErrConflict,
			))
		}
		return result, nil
	}
	return CreateManagedSkillResponse{}, pending(
		fmt.Errorf(
			"%w: managed package did not resolve to its pinned Artifact",
			basespec.ErrReferenceUnresolved,
		),
	)
}

func (a *API) requireBundleMutation(
	ctx context.Context,
	rootID basespec.RootID,
	allowProtected bool,
) error {
	if err := a.Ready(); err != nil {
		return err
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
		return err
	}
	if !a.dependencies.RootMutationPolicy.IsProtectedRoot(rootID) {
		return nil
	}
	if !allowProtected {
		return fmt.Errorf(
			"%w: protected Root %q may only be changed through a trusted built-in installer or update path",
			basespec.ErrProtected,
			rootID,
		)
	}
	return protection.RequirePrivilegedInstaller(ctx)
}

func (a *API) refreshPolicy(
	bundle Bundle,
	allowProtected bool,
) artifact.Policy {
	if allowProtected {
		return builtInSkillArtifactPolicy{}
	}
	autoAdoptSources := make(map[basespec.SourceID]struct{})
	for _, attachment := range bundle.Attachments {
		switch attachment.Role {
		case RoleExternal, RoleLibrary:
			autoAdoptSources[attachment.SourceID] = struct{}{}
		}
	}
	return skillArtifactPolicy{
		ids:              a.dependencies.AutoAdoptionIDProvider,
		autoAdoptSources: autoAdoptSources,
	}
}

func managedSkillCreateResult(
	value artifact.Artifact,
	sourceID basespec.SourceID,
	skillLocator basespec.Locator,
	expectedDefinition cryptoutil.Digest,
	expectedPackageSHA256 cryptoutil.Digest,
) (CreateManagedSkillResponse, bool) {
	intent, err := decodeManagedSkillArtifactData(value.Data)
	if err != nil ||
		intent.PackageSHA256 != expectedPackageSHA256 ||
		value.Adoption != artifact.AdoptionPinned ||
		value.Binding.SourceID != sourceID ||
		value.Binding.Locator != skillLocator ||
		value.Binding.ExpectedKind != artifactbuiltin.AgentSkillArtifactKind ||
		value.State != artifact.StateAvailable ||
		value.ResolvedDefinition == nil ||
		*value.ResolvedDefinition != expectedDefinition {
		return CreateManagedSkillResponse{}, false
	}
	return CreateManagedSkillResponse{
		Artifact: value,
		Address:  value.Address(),
	}, true
}

func validateBundleAttachmentTopology(
	data CollectionData,
	attachments []collection.Attachment,
) error {
	var (
		managedAttachmentCount int
		managedAttachmentID    basespec.SourceID
		builtInAttachmentCount int
	)
	for _, attachment := range attachments {
		switch attachment.Role {
		case artifactbuiltin.ManagedAttachmentRole:
			managedAttachmentCount++
			managedAttachmentID = attachment.SourceID
		case artifactbuiltin.BuiltInAttachmentRole:
			builtInAttachmentCount++
		}
	}
	if managedAttachmentCount > 1 {
		return fmt.Errorf(
			"%w: skill bundle has multiple managed attachments",
			basespec.ErrInvalid,
		)
	}
	if builtInAttachmentCount > 1 {
		return fmt.Errorf(
			"%w: skill bundle has multiple built-in attachments",
			basespec.ErrInvalid,
		)
	}
	if data.ManagedSourceID == "" {
		// Legacy bundles can still be read. They cannot be used for new
		// managed package writes until an explicit ownership repair exists.
		return nil
	}
	if managedAttachmentCount != 1 ||
		managedAttachmentID != data.ManagedSourceID {
		return fmt.Errorf(
			"%w: bundle-owned managed Source %q is not its sole managed attachment",
			basespec.ErrInvalid,
			data.ManagedSourceID,
		)
	}
	return nil
}

func requireBundleOwnedManagedSource(
	bundle Bundle,
	sourceID basespec.SourceID,
) error {
	if bundle.Data.ManagedSourceID == "" ||
		bundle.Data.ManagedSourceID != sourceID {
		return fmt.Errorf(
			"%w: managed Source %q is not owned by Skill Bundle %q",
			basespec.ErrConflict,
			sourceID,
			bundle.Collection.ID,
		)
	}
	return nil
}

func (a *API) validateAttachmentDraft(
	ctx context.Context,
	rootID basespec.RootID,
	draft AttachmentDraft,
) error {
	if err := basespec.ValidateSourceID(draft.SourceID); err != nil {
		return err
	}
	if err := validateRole(draft.Role); err != nil {
		return err
	}
	if _, err := NewAttachmentData(
		draft.DiscoveryRoot,
		draft.ExpectedMemberDigests,
	); err != nil {
		return err
	}
	value, err := a.dependencies.Sources.Get(ctx, rootID, draft.SourceID)
	if err != nil {
		return err
	}
	return validateRoleSourceKind(draft.Role, value.Kind)
}

func (a *API) validateAttachment(
	ctx context.Context,
	rootID basespec.RootID,
	value collection.Attachment,
) error {
	if err := validateRole(value.Role); err != nil {
		return err
	}
	if _, err := DecodeAttachmentData(value.Data); err != nil {
		return err
	}
	sourceValue, err := a.dependencies.Sources.Get(
		ctx,
		rootID,
		value.SourceID,
	)
	if err != nil {
		return err
	}
	return validateRoleSourceKind(value.Role, sourceValue.Kind)
}

func validateRole(role basespec.AttachmentRole) error {
	switch role {
	case artifactbuiltin.ManagedAttachmentRole, artifactbuiltin.BuiltInAttachmentRole, RoleExternal, RoleLibrary:
		return nil
	default:
		return fmt.Errorf(
			"%w: unsupported skill bundle attachment role %q",
			basespec.ErrInvalid,
			role,
		)
	}
}

func validateRoleSourceKind(
	role basespec.AttachmentRole,
	kind basespec.SourceKind,
) error {
	switch role {
	case artifactbuiltin.ManagedAttachmentRole, artifactbuiltin.BuiltInAttachmentRole:
		if kind != artifactbuiltin.ManagedDirectorySourceKind {
			return fmt.Errorf(
				"%w: skill bundle role %q requires source kind %q",
				basespec.ErrInvalid,
				role,
				artifactbuiltin.ManagedDirectorySourceKind,
			)
		}
	case RoleExternal, RoleLibrary:
		if kind != fsdir.Kind {
			return fmt.Errorf(
				"%w: skill bundle role %q requires source kind %q",
				basespec.ErrInvalid,
				role,
				fsdir.Kind,
			)
		}
	}
	return nil
}

func (a *API) discoveryPlan(value Bundle) (discovery.Plan, error) {
	plans := make([]discovery.SourcePlan, 0, len(value.Attachments))
	sources := make(map[basespec.SourceID]source.Summary, len(value.Sources))
	for _, sourceValue := range value.Sources {
		sources[sourceValue.ID] = sourceValue
	}

	for _, attachment := range value.Attachments {
		if !attachment.Enabled {
			continue
		}
		attachmentData, err := DecodeAttachmentData(attachment.Data)
		if err != nil {
			return discovery.Plan{}, err
		}
		expectedContentDigests, err := attachmentData.SourceExpectedContentDigests()
		if err != nil {
			return discovery.Plan{}, err
		}
		sourceValue, found := sources[attachment.SourceID]
		if !found || !sourceValue.Enabled {
			continue
		}
		plans = append(plans, discovery.SourcePlan{
			SourceID: attachment.SourceID,
			DirectoryRoots: []discovery.DirectoryRoot{{
				Root:            attachmentData.DiscoveryRoot,
				Recursive:       true,
				IncludePatterns: []string{string(artifactbuiltin.AgentSkillDefinitionFileName)},
			}},
			DecoderHints: []discovery.DecoderHint{{
				Locator:    attachmentData.DiscoveryRoot,
				Recursive:  true,
				DecoderIDs: []basespec.DecoderID{artifactbuiltin.AgentSkillDecoderID},
			}},
			ExpectedContentDigests: expectedContentDigests,
			AllowedDecoderIDs:      []basespec.DecoderID{artifactbuiltin.AgentSkillDecoderID},
			Authoritative:          true,
		}.Normalized())
	}

	plan := discovery.Plan{
		Revision: DiscoveryPolicyRevision,
		Sources:  plans,
	}
	if err := plan.Validate(); err != nil {
		return discovery.Plan{}, err
	}
	return plan, nil
}

func managedAttachmentForRole(
	value Bundle,
	role basespec.AttachmentRole,
) (collection.Attachment, source.Summary, error) {
	sources := make(map[basespec.SourceID]source.Summary, len(value.Sources))
	for _, sourceValue := range value.Sources {
		sources[sourceValue.ID] = sourceValue
	}

	var (
		attachment  collection.Attachment
		sourceValue source.Summary
		found       bool
	)
	switch role {
	case artifactbuiltin.ManagedAttachmentRole, artifactbuiltin.BuiltInAttachmentRole:
	default:
		return collection.Attachment{}, source.Summary{}, fmt.Errorf(
			"%w: unsupported managed Skill attachment role %q",
			basespec.ErrInvalid,
			role,
		)
	}
	for _, candidate := range value.Attachments {
		if candidate.Role != role {
			continue
		}
		currentSource, exists := sources[candidate.SourceID]
		if !exists {
			return collection.Attachment{}, source.Summary{}, fmt.Errorf(
				"%w: managed attachment source is unavailable",
				basespec.ErrAttachmentNotFound,
			)
		}
		if found {
			return collection.Attachment{}, source.Summary{}, fmt.Errorf(
				"%w: skill bundle has multiple %q attachments",
				basespec.ErrConflict,
				role,
			)
		}
		attachment = candidate
		sourceValue = currentSource
		found = true
	}
	if !found {
		return collection.Attachment{}, source.Summary{}, fmt.Errorf(
			"%w: skill bundle has no %q attachment",
			basespec.ErrAttachmentNotFound,
			role,
		)
	}
	return attachment, sourceValue, nil
}

func bundleAttachmentRole(
	value Bundle,
	sourceID basespec.SourceID,
) (basespec.AttachmentRole, error) {
	for _, attachment := range value.Attachments {
		if attachment.SourceID == sourceID {
			return attachment.Role, nil
		}
	}
	return "", fmt.Errorf(
		"%w: source %q is not attached to bundle %q",
		basespec.ErrAttachmentNotFound,
		sourceID,
		value.Collection.ID,
	)
}

func (p skillArtifactPolicy) Derive(
	ctx context.Context,
	_ collection.Collection,
	occurrence catalog.Occurrence,
	value definition.Definition,
) (artifact.Draft, bool, []diagnostic.Diagnostic, error) {
	if occurrence.Kind != artifactbuiltin.AgentSkillArtifactKind {
		return artifact.Draft{}, false, nil, nil
	}
	if _, allowed := p.autoAdoptSources[occurrence.Key.SourceID]; !allowed {
		return artifact.Draft{}, false, nil, nil
	}
	if err := skillArtifact.ValidateDefinition(value); err != nil {
		return artifact.Draft{}, false, []diagnostic.Diagnostic{{
			Severity: diagnostic.DiagnosticError,
			Code:     "skill.bundle.definition-invalid",
			Message:  diagnostic.BoundedDiagnosticMessage(err.Error()),
			Location: &diagnostic.DiagnosticLocation{
				Locator:            occurrence.Key.Locator,
				SubresourceLocator: occurrence.Key.SubresourceLocator,
			},
		}}, nil
	}
	if p.ids == nil {
		return artifact.Draft{}, false, nil, fmt.Errorf(
			"%w: skill bundle automatic adoption ID provider is unavailable",
			basespec.ErrInvalid,
		)
	}
	id, err := p.ids.NewArtifactID(ctx)
	if err != nil {
		return artifact.Draft{}, false, nil, err
	}
	if err := basespec.ValidateArtifactID(id); err != nil {
		return artifact.Draft{}, false, nil, err
	}
	name := value.DisplayName
	if name == "" {
		name = string(value.LogicalName)
	}
	return artifact.Draft{
		ID:      id,
		Name:    name,
		Enabled: true,
		Data:    emptyArtifactData(),
	}, true, nil, nil
}

func managedSkillPackageAddressOf(
	binding artifact.SourceBinding,
) (source.ManagedPackageAddress, error) {
	if binding.SubresourceLocator != "" {
		return source.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Skill cannot target a subresource",
			basespec.ErrInvalid,
		)
	}
	return skillArtifact.ManagedPackageAddressFromSkillLocator(binding.Locator)
}

func normalizeManagedSkillFiles(
	skillMD []byte,
	input []source.ManagedPackageFile,
) ([]source.ManagedPackageFile, []byte, error) {
	if len(input) == 0 {
		if len(skillMD) == 0 {
			return nil, nil, fmt.Errorf(
				"%w: SKILL.md content is required",
				basespec.ErrInvalid,
			)
		}
		return []source.ManagedPackageFile{{
			Locator: artifactbuiltin.AgentSkillDefinitionFileName,
			Content: append([]byte(nil), skillMD...),
		}}, append([]byte(nil), skillMD...), nil
	}

	normalized, err := source.NormalizeManagedPackageFiles(input)
	if err != nil {
		return nil, nil, err
	}

	var found []byte
	for _, file := range normalized {
		if file.Locator != artifactbuiltin.AgentSkillDefinitionFileName {
			continue
		}
		found = append([]byte(nil), file.Content...)
		break
	}
	if len(found) == 0 {
		return nil, nil, fmt.Errorf(
			"%w: managed skill package must contain %q",
			basespec.ErrInvalid,
			artifactbuiltin.AgentSkillDefinitionFileName,
		)
	}
	if len(skillMD) != 0 && !bytes.Equal(skillMD, found) {
		return nil, nil, fmt.Errorf(
			"%w: request SKILL.md differs from package SKILL.md",
			basespec.ErrInvalid,
		)
	}
	return normalized, found, nil
}

type managedSkillArtifactData struct {
	PackageSHA256 cryptoutil.Digest `json:"packageSHA256"`

	// Enabled is required durable operation intent. The pointer exists only to
	// detect a missing JSON field during strict decoding.
	Enabled *bool `json:"enabled,omitempty"`
}

func newManagedSkillArtifactData(
	packageSHA256 cryptoutil.Digest,
	enabled bool,
) managedSkillArtifactData {
	requestedEnabled := enabled
	return managedSkillArtifactData{
		PackageSHA256: packageSHA256,
		Enabled:       &requestedEnabled,
	}
}

func encodeManagedSkillArtifactData(
	value managedSkillArtifactData,
) (json.RawMessage, error) {
	if err := validateManagedSkillArtifactData(value); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func decodeManagedSkillArtifactData(
	raw json.RawMessage,
) (managedSkillArtifactData, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return managedSkillArtifactData{}, err
	}
	var value managedSkillArtifactData
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return managedSkillArtifactData{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("managed Skill Artifact data has trailing JSON")
		}
		return managedSkillArtifactData{}, err
	}
	if err := validateManagedSkillArtifactData(value); err != nil {
		return managedSkillArtifactData{}, err
	}
	return value, nil
}

func validateManagedSkillArtifactData(
	value managedSkillArtifactData,
) error {
	if err := cryptoutil.ValidateDigest(value.PackageSHA256); err != nil {
		return err
	}
	if value.Enabled == nil {
		return fmt.Errorf(
			"%w: managed Skill Artifact data requires enabled",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func validateManagedSkillOperationIntent(
	value artifact.Artifact,
	bundle collection.CollectionRef,
	artifactID basespec.ArtifactID,
	sourceID basespec.SourceID,
	skillLocator basespec.Locator,
) error {
	if value.ID != artifactID ||
		value.RootID != bundle.RootID ||
		value.CollectionID != bundle.CollectionID ||
		value.Kind != artifactbuiltin.AgentSkillArtifactKind ||
		value.Adoption != artifact.AdoptionPinned ||
		value.Binding.SourceID != sourceID ||
		value.Binding.Locator != skillLocator ||
		value.Binding.ExpectedKind != artifactbuiltin.AgentSkillArtifactKind {
		return fmt.Errorf(
			"%w: managed Skill Artifact %q conflicts with its existing creation intent",
			basespec.ErrConflict,
			artifactID,
		)
	}

	return nil
}

func managedSkillPackageDigest(
	files []source.ManagedPackageFile,
) (cryptoutil.Digest, error) {
	normalized, err := source.NormalizeManagedPackageFiles(files)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(canonical), nil
}

func (a *API) managedSkillByID(
	ctx context.Context,
	rootID basespec.RootID,
	artifactID basespec.ArtifactID,
) (*artifact.Artifact, error) {
	value, err := a.dependencies.Artifacts.Get(ctx, artifact.ArtifactRef{
		RootID:     rootID,
		ArtifactID: artifactID,
	})
	if errors.Is(err, basespec.ErrArtifactNotFound) {
		//nolint:nilnil // Explicit.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	output := value.Clone()
	return &output, nil
}

func pendingManagedSkillCreateError(
	artifactID basespec.ArtifactID,
	cause error,
) error {
	return fmt.Errorf(
		"managed Skill create for Artifact %q remains pending; retry with the same artifactID: %w",
		artifactID,
		cause,
	)
}

func pendingManagedSkillPurgeError(ref artifact.ArtifactRef, cause error) error {
	return fmt.Errorf(
		"managed Skill purge for Artifact %q remains pending; reload its current revision before retrying with the same artifactID: %w",
		ref.ArtifactID,
		cause,
	)
}

func emptyArtifactData() json.RawMessage {
	return json.RawMessage(jsonutil.EmptyObject)
}
