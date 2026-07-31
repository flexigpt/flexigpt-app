package skillbundle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/managed"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
)

type API struct {
	dependencies Dependencies
	closed       atomic.Bool
}

type Bundle struct {
	Collection  collection.Collection
	Data        CollectionData
	Attachments []collection.Attachment
	Sources     []source.Summary
}

type CreateBundleRequest struct {
	RootID         basespec.RootID
	DisplayName    string
	Description    string
	Enabled        bool
	LogicalName    basespec.LogicalName
	LogicalVersion basespec.LogicalVersion
	Labels         map[string]string
	BootstrapKey   string
	Attachments    []AttachmentDraft
}

type UpdateBundleRequest struct {
	Bundle           collection.CollectionRef
	ExpectedRevision uint64
	DisplayName      string
	Description      string
	Enabled          bool
}

type AttachmentDraft struct {
	SourceID basespec.SourceID
	Role     basespec.AttachmentRole
	Enabled  bool
}

type CreateManagedSkillRequest struct {
	Bundle                     collection.CollectionRef
	ExpectedCollectionRevision uint64
	SkillName                  string
	OperationKey               string
	SKILLMD                    []byte
	Files                      []source.ManagedPackageFile
	Enabled                    bool
}

type CreateManagedSkillResponse struct {
	Artifact     artifact.Artifact
	Address      artifact.ArtifactAddress
	OperationKey string
}

type AdoptSkillRequest struct {
	Bundle                  collection.CollectionRef
	Occurrence              catalog.OccurrenceKey
	ExpectedCatalogRevision uint64
	Name                    string
	Enabled                 bool
}

type PinSkillRequest struct {
	Bundle                     collection.CollectionRef
	ExpectedCollectionRevision uint64
	Binding                    artifact.SourceBinding
	Name                       string
	Enabled                    bool
}

type BuiltInSkill struct {
	Name    string
	SKILLMD []byte
	Files   []source.ManagedPackageFile
	Enabled bool
}

type BootstrapBundleRequest struct {
	RootID         basespec.RootID
	BootstrapKey   string
	LogicalName    basespec.LogicalName
	LogicalVersion basespec.LogicalVersion
	Labels         map[string]string
	DisplayName    string
	Description    string
	Skills         []BuiltInSkill
}

func New(dependencies Dependencies) (*API, error) {
	if err := dependencies.Validate(); err != nil {
		return nil, err
	}
	if !dependencies.HasDecoder(skillartifact.DecoderID) {
		return nil, fmt.Errorf(
			"%w: shared agent skill decoder %q is not registered",
			basespec.ErrDecoderUnavailable,
			skillartifact.DecoderID,
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

func (a *API) Ready() error {
	if a == nil || a.closed.Load() {
		return basespec.ErrClosed
	}
	return a.dependencies.Validate()
}

func (a *API) CreateBundle(
	ctx context.Context,
	request CreateBundleRequest,
) (Bundle, error) {
	return a.createBundle(ctx, request, false)
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
	if value.Kind != CollectionKind {
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

func (a *API) ListBundles(
	ctx context.Context,
	rootID basespec.RootID,
) ([]Bundle, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if _, err := a.EnsureEmbeddedBuiltInsForRoot(ctx, rootID); err != nil {
		return nil, err
	}

	values, err := a.dependencies.Collections.ListByRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}

	output := make([]Bundle, 0)
	for _, value := range values {
		if value.Kind != CollectionKind {
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

// SkillBundleRefs returns every active Skill Bundle Collection. Runtime startup
// uses this to reconcile derived Agent Skills state without treating a runtime
// ref shape as a source-of-truth ownership discriminator.
func (a *API) SkillBundleRefs(
	ctx context.Context,
) ([]collection.CollectionRef, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}

	roots, err := a.dependencies.Roots.List(ctx)
	if err != nil {
		return nil, err
	}

	refs := make([]collection.CollectionRef, 0)
	for _, rootValue := range roots {
		bundles, err := a.ListBundles(ctx, rootValue.ID)
		if err != nil {
			return nil, err
		}
		for _, bundle := range bundles {
			refs = append(refs, bundle.Collection.Ref())
		}
	}
	sort.Slice(refs, func(left, right int) bool {
		if refs[left].RootID != refs[right].RootID {
			return refs[left].RootID < refs[right].RootID
		}
		return refs[left].CollectionID < refs[right].CollectionID
	})
	return refs, nil
}

func (a *API) UpdateBundle(
	ctx context.Context,
	request UpdateBundleRequest,
) (Bundle, error) {
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
	if _, err := a.GetBundle(ctx, ref); err != nil {
		return collection.Collection{}, err
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
	value, err := a.dependencies.Collections.GetRetired(ctx, ref)
	if err != nil {
		return err
	}
	if value.Kind != CollectionKind {
		return fmt.Errorf(
			"%w: collection %q is not a retired skill bundle",
			basespec.ErrNotFound,
			ref.CollectionID,
		)
	}
	return a.dependencies.Collections.Purge(ctx, ref, expectedRevision)
}

func (a *API) AttachSource(
	ctx context.Context,
	bundle collection.CollectionRef,
	expectedCollectionRevision uint64,
	draft AttachmentDraft,
) (Bundle, error) {
	if draft.Role == RoleBuiltIn {
		return Bundle{}, fmt.Errorf(
			"%w: skill bundle built-in attachment role is reserved for bootstrap",
			basespec.ErrInvalid,
		)
	}
	if _, err := a.GetBundle(ctx, bundle); err != nil {
		return Bundle{}, err
	}
	if err := a.validateAttachmentDraft(ctx, bundle.RootID, draft); err != nil {
		return Bundle{}, err
	}

	_, _, err := a.dependencies.Collections.Attach(
		ctx,
		bundle,
		expectedCollectionRevision,
		collection.AttachmentDraft{
			SourceID: draft.SourceID,
			Role:     draft.Role,
			Enabled:  draft.Enabled,
			Data:     json.RawMessage(`{}`),
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
	result, err := a.dependencies.Refresh.Refresh(
		ctx,
		ref,
		plan,
		skillArtifactPolicy{},
	)
	if err != nil {
		return refresh.Result{}, err
	}
	return result, nil
}

// BuildLinkedPortableBundleDefinition returns a canonical shareable JSON
// descriptor. It intentionally does not capture packages or acquire content.
// A multi-source bundle requires future closure packaging and is rejected.
func (a *API) BuildLinkedPortableBundleDefinition(
	ctx context.Context,
	ref collection.CollectionRef,
) (definition.CollectionDefinition, error) {
	bundle, err := a.GetBundle(ctx, ref)
	if err != nil {
		return definition.CollectionDefinition{}, err
	}

	snapshot, err := catalog.ReadCurrent(ctx, a.dependencies.Catalogs, ref)
	if err != nil {
		return definition.CollectionDefinition{}, err
	}
	records, err := a.dependencies.Artifacts.ListByCollection(ctx, ref)
	if err != nil {
		return definition.CollectionDefinition{}, err
	}

	var sourceID basespec.SourceID
	members := make([]definition.ContentRef, 0)
	for _, record := range records {
		if record.Kind != skillartifact.Kind {
			continue
		}
		if record.State != artifact.StateAvailable ||
			record.ResolvedDefinition == nil {
			return definition.CollectionDefinition{}, fmt.Errorf(
				"%w: Skill Artifact %q is not exportable from the current catalog",
				basespec.ErrReferenceUnresolved,
				record.ID,
			)
		}
		if sourceID == "" {
			sourceID = record.Binding.SourceID
		} else if sourceID != record.Binding.SourceID {
			return definition.CollectionDefinition{}, fmt.Errorf(
				"%w: linked Skill Bundle export requires one Source; use future package closure export",
				basespec.ErrUnsupported,
			)
		}

		var occurrence *catalog.Occurrence
		for index := range snapshot.Occurrences {
			candidate := &snapshot.Occurrences[index]
			if candidate.Key.SourceID == record.Binding.SourceID &&
				candidate.Key.Locator == record.Binding.Locator &&
				candidate.Key.SubresourceLocator == record.Binding.SubresourceLocator {
				occurrence = candidate
				break
			}
		}
		if occurrence == nil ||
			occurrence.State != catalog.OccurrenceValid ||
			occurrence.DefinitionDigest == nil ||
			occurrence.SourceContentDigest == nil ||
			*occurrence.DefinitionDigest != *record.ResolvedDefinition {
			return definition.CollectionDefinition{}, fmt.Errorf(
				"%w: Skill Artifact %q does not match the current catalog",
				basespec.ErrCatalogStale,
				record.ID,
			)
		}

		digest := *occurrence.SourceContentDigest
		members = append(members, definition.ContentRef{
			Locator:   record.Binding.Locator,
			Digest:    &digest,
			MediaType: portableSkillMediaType,
			Role:      string(skillartifact.Kind),
		})
	}

	return NewPortableBundleDefinition(PortableBundleMetadata{
		LogicalName:    bundle.Data.LogicalName,
		LogicalVersion: bundle.Data.LogicalVersion,
		DisplayName:    bundle.Collection.DisplayName,
		Description:    bundle.Collection.Description,
		Labels:         bundle.Data.Labels,
	}, members)
}

// BuildLinkedPortableBundleJSON returns the canonical shareable linked bundle
// descriptor. It deliberately does not claim to contain a package closure.
func (a *API) BuildLinkedPortableBundleJSON(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]byte, error) {
	value, err := a.BuildLinkedPortableBundleDefinition(ctx, ref)
	if err != nil {
		return nil, err
	}
	return definition.MarshalCollectionDefinition(value)
}

func (a *API) CreateManagedSkill(
	ctx context.Context,
	request CreateManagedSkillRequest,
) (CreateManagedSkillResponse, error) {
	return a.createManagedSkill(ctx, request, false)
}

func (a *API) AdoptSkill(
	ctx context.Context,
	request AdoptSkillRequest,
) (artifact.Artifact, error) {
	if _, err := a.GetBundle(ctx, request.Bundle); err != nil {
		return artifact.Artifact{}, err
	}
	if request.Occurrence.CollectionID != request.Bundle.CollectionID {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: skill occurrence belongs to another bundle",
			basespec.ErrInvalid,
		)
	}
	if request.ExpectedCatalogRevision == 0 {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: expected skill bundle catalog revision is required",
			basespec.ErrInvalid,
		)
	}

	snapshot, err := catalog.ReadCurrent(
		ctx,
		a.dependencies.Catalogs,
		request.Bundle,
	)
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
		if occurrence.Kind != skillartifact.Kind ||
			occurrence.State != catalog.OccurrenceValid ||
			occurrence.DefinitionDigest == nil {
			return artifact.Artifact{}, fmt.Errorf(
				"%w: requested occurrence is not an adoptable %q Skill",
				basespec.ErrReferenceUnresolved,
				skillartifact.Kind,
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

	return a.dependencies.Artifacts.Adopt(ctx, artifact.AdoptRequest{
		Collection:              request.Bundle,
		Occurrence:              request.Occurrence,
		ExpectedCatalogRevision: request.ExpectedCatalogRevision,
		Name:                    request.Name,
		Enabled:                 request.Enabled,
		Data:                    EmptyArtifactData(),
	})
}

func (a *API) PinSkill(
	ctx context.Context,
	request PinSkillRequest,
) (artifact.Artifact, error) {
	if _, err := a.GetBundle(ctx, request.Bundle); err != nil {
		return artifact.Artifact{}, err
	}
	if request.Binding.ExpectedKind != skillartifact.Kind {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: skill bundle pins only support %q",
			basespec.ErrInvalid,
			skillartifact.Kind,
		)
	}
	return a.dependencies.Artifacts.Pin(ctx, artifact.PinRequest{
		Collection:                 request.Bundle,
		ExpectedCollectionRevision: request.ExpectedCollectionRevision,
		Binding:                    request.Binding,
		Name:                       request.Name,
		Enabled:                    request.Enabled,
		Data:                       EmptyArtifactData(),
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
		if value.Kind == skillartifact.Kind {
			output = append(output, value)
		}
	}
	return output, nil
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
	if value.Kind != skillartifact.Kind {
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

func (a *API) SetSkillEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
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
	value, err := a.GetSkill(ctx, ref)
	if err != nil {
		return err
	}
	if value.Revision != expectedRevision {
		return basespec.ErrConflict
	}

	if !strings.HasPrefix(
		value.IdempotencyKey,
		managedSkillIdempotencyKeyPrefix,
	) {
		return a.dependencies.Artifacts.Purge(ctx, ref, expectedRevision)
	}
	if value.Adoption != artifact.AdoptionPinned {
		return fmt.Errorf(
			"%w: managed Skill Artifact must remain pinned until purged",
			basespec.ErrConflict,
		)
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
	case RoleBuiltIn:
		return fmt.Errorf(
			"%w: built-in Skill packages are read-only",
			basespec.ErrConflict,
		)
	case RoleManaged:
	default:
		return fmt.Errorf(
			"%w: managed Skill Artifact source is not a managed attachment",
			basespec.ErrConflict,
		)
	}

	directory, err := managedSkillPackageDirectoryOf(value.Binding)
	if err != nil {
		return err
	}
	state, generation, err := a.dependencies.GetManagedSourceState(
		ctx,
		value.RootID,
		value.Binding.SourceID,
	)
	if err != nil {
		return err
	}
	if _, _, err := a.dependencies.RemoveManagedPackage(
		ctx,
		value.RootID,
		value.Binding.SourceID,
		state.Revision,
		directory,
		generation,
	); err != nil {
		return pendingManagedSkillPurgeError(value.Ref(), err)
	}
	if err := a.dependencies.Artifacts.Purge(
		ctx,
		ref,
		expectedRevision,
	); err != nil {
		return pendingManagedSkillPurgeError(value.Ref(), err)
	}
	return nil
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
	if err := basespec.ValidateRootID(request.RootID); err != nil {
		return Bundle{}, err
	}
	if request.BootstrapKey != "" {
		if err := basespec.ValidateIdentifier(
			"skill bundle bootstrap key",
			request.BootstrapKey,
			basespec.MaxKindBytes,
		); err != nil {
			return Bundle{}, err
		}
	}

	data, err := EncodeCollectionData(CollectionData{
		SchemaVersion:           CollectionSchemaVersion,
		DiscoveryPolicyRevision: DiscoveryPolicyRevision,
		LogicalName:             request.LogicalName,
		LogicalVersion:          request.LogicalVersion,
		Labels:                  request.Labels,
	})
	if err != nil {
		return Bundle{}, err
	}

	attachments := make([]collection.AttachmentDraft, 0, len(request.Attachments))
	for _, draft := range request.Attachments {
		if draft.Role == RoleBuiltIn && !allowBuiltInAttachment {
			return Bundle{}, fmt.Errorf(
				"%w: skill bundle built-in attachment role is reserved for bootstrap",
				basespec.ErrInvalid,
			)
		}
		if err := a.validateAttachmentDraft(ctx, request.RootID, draft); err != nil {
			return Bundle{}, err
		}
		attachments = append(attachments, collection.AttachmentDraft{
			SourceID: draft.SourceID,
			Role:     draft.Role,
			Enabled:  draft.Enabled,
			Data:     json.RawMessage(`{}`),
		})
	}

	created, _, err := a.dependencies.Collections.Create(
		ctx,
		request.RootID,
		collection.Draft{
			Kind:           CollectionKind,
			DisplayName:    request.DisplayName,
			Description:    request.Description,
			Enabled:        request.Enabled,
			Data:           data,
			IdempotencyKey: request.BootstrapKey,
		},
		attachments,
	)
	if err != nil {
		return Bundle{}, err
	}
	return a.GetBundle(ctx, created.Ref())
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
	if request.ExpectedCollectionRevision == 0 {
		return CreateManagedSkillResponse{}, fmt.Errorf(
			"%w: expected skill bundle revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateLogicalName(
		basespec.LogicalName(request.SkillName),
	); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if err := validateManagedSkillOperationKey(
		request.OperationKey,
	); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	artifactIdempotencyKey, err := managedSkillArtifactIdempotencyKey(
		request.OperationKey,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}

	files, skillMD, err := normalizeManagedSkillFiles(
		request.SKILLMD,
		request.Files,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}
	definitionValue, _, err := skillartifact.DecodeSkillDocument(skillMD,
		request.SkillName,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, err
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
	if bundle.Collection.Revision != request.ExpectedCollectionRevision {
		return CreateManagedSkillResponse{}, basespec.ErrConflict
	}

	targetRole := RoleManaged
	if allowBuiltInAttachment {
		targetRole = RoleBuiltIn
	}
	attachment, sourceValue, err := managedAttachmentForRole(
		bundle,
		targetRole,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if !attachment.Enabled || !sourceValue.Enabled {
		return CreateManagedSkillResponse{}, fmt.Errorf(
			"%w: managed Skill source is disabled",
			basespec.ErrConflict,
		)
	}
	directory, err := managedSkillPackageDirectory(
		request.OperationKey,
		request.SkillName,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}
	skillLocator := basespec.Locator(
		path.Join(string(directory), skillartifact.DefinitionFileName),
	)
	if err := source.ValidateManagedPackageDirectory(directory); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if err := basespec.ValidatePortableLocator(skillLocator, false); err != nil {
		return CreateManagedSkillResponse{}, err
	}

	artifactName := definitionValue.DisplayName
	if artifactName == "" {
		artifactName = request.SkillName
	}

	pinned, err := a.findManagedSkillByOperation(
		ctx,
		request.Bundle,
		request.OperationKey,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if pinned == nil {
		state, generation, err := a.dependencies.GetManagedSourceState(
			ctx,
			request.Bundle.RootID,
			sourceValue.ID,
		)
		if err != nil {
			return CreateManagedSkillResponse{}, err
		}
		if state.ID != sourceValue.ID ||
			state.RootID != request.Bundle.RootID {
			return CreateManagedSkillResponse{}, fmt.Errorf(
				"%w: managed Source state does not match the selected attachment",
				basespec.ErrInvalid,
			)
		}

		localData, err := encodeManagedSkillArtifactData(
			managedSkillArtifactData{
				ExpectedSourceRevision: state.Revision,
				ExpectedGeneration:     generation,
			},
		)
		if err != nil {
			return CreateManagedSkillResponse{}, err
		}

		value, pinErr := a.dependencies.Artifacts.Pin(ctx, artifact.PinRequest{
			Collection:                 request.Bundle,
			ExpectedCollectionRevision: request.ExpectedCollectionRevision,
			Binding: artifact.SourceBinding{
				SourceID:     sourceValue.ID,
				Locator:      skillLocator,
				ExpectedKind: skillartifact.Kind,
			},
			Name:           artifactName,
			Enabled:        request.Enabled,
			Data:           localData,
			IdempotencyKey: artifactIdempotencyKey,
		})
		switch {
		case pinErr == nil:
			pinned = &value

		case errors.Is(pinErr, basespec.ErrConflict):
			// Another caller may have completed the same operation between
			// our lookup and this insert. Re-read durable Artifact Store state
			// instead of treating that normal race as a failed operation.
			pinned, err = a.findManagedSkillByOperation(
				ctx,
				request.Bundle,
				request.OperationKey,
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
		request.OperationKey,
		sourceValue.ID,
		skillLocator,
		artifactName,
		request.Enabled,
	); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if result, complete, completionErr := a.completeManagedSkillCreate(
		ctx,
		*pinned,
		sourceValue.ID,
		skillLocator,
		definitionValue.Digest,
		request.OperationKey,
	); completionErr != nil {
		return CreateManagedSkillResponse{}, pendingManagedSkillOperationError(
			request.OperationKey,
			completionErr,
		)
	} else if complete {
		return result, nil
	}
	intent, err := managedSkillPublicationIntentOf(pinned.Data)
	if err != nil {
		return CreateManagedSkillResponse{}, pendingManagedSkillOperationError(
			request.OperationKey,
			err,
		)
	}

	state, _, err := a.dependencies.GetManagedSourceState(
		ctx,
		request.Bundle.RootID,
		sourceValue.ID,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, pendingManagedSkillOperationError(request.OperationKey, err)
	}
	if state.ID != sourceValue.ID ||
		state.RootID != request.Bundle.RootID {
		return CreateManagedSkillResponse{}, pendingManagedSkillOperationError(
			request.OperationKey,
			fmt.Errorf(
				"%w: managed Source state does not match the selected attachment",
				basespec.ErrInvalid,
			),
		)
	}
	if state.Revision != intent.ExpectedSourceRevision {
		// A prior attempt may have completed source publication and metadata
		// acknowledgement before returning an error. Refresh first rather than
		// republishing from a newer Source revision.
		if _, err := a.RefreshBundle(ctx, request.Bundle); err != nil {
			return CreateManagedSkillResponse{}, pendingManagedSkillOperationError(
				request.OperationKey,
				err,
			)
		}
		resolved, err := a.dependencies.Artifacts.Get(ctx, pinned.Ref())
		if err != nil {
			return CreateManagedSkillResponse{}, pendingManagedSkillOperationError(
				request.OperationKey,
				err,
			)
		}
		if result, complete, completionErr := a.completeManagedSkillCreate(
			ctx,
			resolved,
			sourceValue.ID,
			skillLocator,
			definitionValue.Digest,
			request.OperationKey,
		); completionErr != nil {
			return CreateManagedSkillResponse{}, pendingManagedSkillOperationError(
				request.OperationKey,
				completionErr,
			)
		} else if complete {
			return result, nil
		}
		return CreateManagedSkillResponse{}, pendingManagedSkillOperationError(
			request.OperationKey,
			fmt.Errorf(
				"%w: managed Source changed before this pending operation could be completed",
				basespec.ErrConflict,
			),
		)
	}

	_, _, err = a.dependencies.PublishManagedPackage(
		ctx,
		request.Bundle.RootID,
		sourceValue.ID,
		intent.ExpectedSourceRevision,
		source.ManagedPackagePublication{
			Directory:          directory,
			ExpectedGeneration: intent.ExpectedGeneration,
			Files:              files,
		},
	)
	if err != nil {
		return CreateManagedSkillResponse{}, pendingManagedSkillOperationError(request.OperationKey, err)
	}

	if _, err := a.RefreshBundle(ctx, request.Bundle); err != nil {
		return CreateManagedSkillResponse{}, pendingManagedSkillOperationError(request.OperationKey, err)
	}

	resolved, err := a.dependencies.Artifacts.Get(ctx, pinned.Ref())
	if err != nil {
		return CreateManagedSkillResponse{}, pendingManagedSkillOperationError(request.OperationKey, err)
	}
	if result, complete, completionErr := a.completeManagedSkillCreate(
		ctx,
		resolved,
		sourceValue.ID,
		skillLocator,
		definitionValue.Digest,
		request.OperationKey,
	); completionErr != nil {
		return CreateManagedSkillResponse{}, pendingManagedSkillOperationError(
			request.OperationKey,
			completionErr,
		)
	} else if complete {
		return result, nil
	}
	return CreateManagedSkillResponse{}, pendingManagedSkillOperationError(
		request.OperationKey,
		fmt.Errorf(
			"%w: published managed Skill did not resolve to its pinned Artifact",
			basespec.ErrReferenceUnresolved,
		),
	)
}

func managedSkillCreateResult(
	value artifact.Artifact,
	sourceID basespec.SourceID,
	skillLocator basespec.Locator,
	expectedDefinition cryptoutil.Digest,
	operationKey string,
) (CreateManagedSkillResponse, bool) {
	idempotencyKey, err := managedSkillArtifactIdempotencyKey(operationKey)
	if err != nil ||
		value.IdempotencyKey != idempotencyKey ||
		value.Adoption != artifact.AdoptionPinned ||
		value.Binding.SourceID != sourceID ||
		value.Binding.Locator != skillLocator ||
		value.Binding.ExpectedKind != skillartifact.Kind ||
		value.State != artifact.StateAvailable ||
		value.ResolvedDefinition == nil ||
		*value.ResolvedDefinition != expectedDefinition {
		return CreateManagedSkillResponse{}, false
	}
	return CreateManagedSkillResponse{
		Artifact:     value,
		Address:      value.Address(),
		OperationKey: operationKey,
	}, true
}

// completeManagedSkillCreate removes source-observation preconditions after
// the pending Artifact has resolved. The Artifact Store idempotency key
// remains the only durable operation identity.
func (a *API) completeManagedSkillCreate(
	ctx context.Context,
	value artifact.Artifact,
	sourceID basespec.SourceID,
	skillLocator basespec.Locator,
	expectedDefinition cryptoutil.Digest,
	operationKey string,
) (CreateManagedSkillResponse, bool, error) {
	result, complete := managedSkillCreateResult(
		value,
		sourceID,
		skillLocator,
		expectedDefinition,
		operationKey,
	)
	if !complete {
		return CreateManagedSkillResponse{}, false, nil
	}

	intent, err := decodeManagedSkillArtifactData(value.Data)
	if err != nil {
		return CreateManagedSkillResponse{}, false, err
	}
	if intent.ExpectedSourceRevision == 0 {
		return result, true, nil
	}

	updated, err := a.dependencies.Artifacts.UpdateData(
		ctx,
		value.Ref(),
		value.Revision,
		EmptyArtifactData(),
	)
	if err == nil {
		result.Artifact = updated
		result.Address = updated.Address()
		return result, true, nil
	}
	if !errors.Is(err, basespec.ErrConflict) {
		return CreateManagedSkillResponse{}, false, err
	}

	current, err := a.dependencies.Artifacts.Get(ctx, value.Ref())
	if err != nil {
		return CreateManagedSkillResponse{}, false, err
	}
	currentIntent, err := decodeManagedSkillArtifactData(current.Data)
	if err != nil {
		return CreateManagedSkillResponse{}, false, err
	}
	currentResult, currentComplete := managedSkillCreateResult(
		current,
		sourceID,
		skillLocator,
		expectedDefinition,
		operationKey,
	)
	if !currentComplete || currentIntent.ExpectedSourceRevision != 0 {
		return CreateManagedSkillResponse{}, false, fmt.Errorf(
			"%w: managed Skill completion changed concurrently",
			basespec.ErrConflict,
		)
	}
	return currentResult, true, nil
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
	case RoleManaged, RoleBuiltIn, RoleExternal, RoleImported, RoleLibrary:
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
	case RoleManaged, RoleBuiltIn:
		if kind != managed.Kind {
			return fmt.Errorf(
				"%w: skill bundle role %q requires source kind %q",
				basespec.ErrInvalid,
				role,
				managed.Kind,
			)
		}
	case RoleExternal, RoleImported, RoleLibrary:
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
		sourceValue, found := sources[attachment.SourceID]
		if !found || !sourceValue.Enabled {
			continue
		}
		plans = append(plans, discovery.SourcePlan{
			SourceID: attachment.SourceID,
			DirectoryRoots: []discovery.DirectoryRoot{{
				Root:            ".",
				Recursive:       true,
				IncludePatterns: []string{skillartifact.DefinitionFileName},
			}},
			DecoderHints: []discovery.DecoderHint{{
				Locator:    ".",
				Recursive:  true,
				DecoderIDs: []basespec.DecoderID{skillartifact.DecoderID},
			}},
			AllowedDecoderIDs: []basespec.DecoderID{skillartifact.DecoderID},
			Authoritative:     true,
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
	case RoleManaged, RoleBuiltIn:
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

type skillArtifactPolicy struct{}

func (skillArtifactPolicy) Derive(
	_ context.Context,
	_ collection.Collection,
	occurrence catalog.Occurrence,
	value definition.Definition,
) (artifact.Draft, bool, []diagnostic.Diagnostic) {
	if occurrence.Kind != skillartifact.Kind {
		return artifact.Draft{}, false, nil
	}
	if err := skillartifact.ValidateDefinition(value); err != nil {
		return artifact.Draft{}, false, []diagnostic.Diagnostic{{
			Severity: diagnostic.DiagnosticError,
			Code:     "skill.bundle.definition-invalid",
			Message:  diagnostic.BoundedDiagnosticMessage(err.Error()),
			Location: &diagnostic.DiagnosticLocation{
				Locator:            occurrence.Key.Locator,
				SubresourceLocator: occurrence.Key.SubresourceLocator,
			},
		}}
	}
	return artifact.Draft{
		Name:    value.DisplayName,
		Enabled: true,
		Data:    EmptyArtifactData(),
	}, true, nil
}

func managedSkillPackageDirectoryOf(
	binding artifact.SourceBinding,
) (basespec.Locator, error) {
	if binding.SubresourceLocator != "" ||
		path.Base(string(binding.Locator)) != skillartifact.DefinitionFileName {
		return "", fmt.Errorf(
			"%w: managed Skill binding does not identify a package %q",
			basespec.ErrInvalid,
			skillartifact.DefinitionFileName,
		)
	}
	directory := basespec.Locator(path.Dir(string(binding.Locator)))
	if directory == "." {
		return "", fmt.Errorf(
			"%w: managed Skill package cannot be the Source root",
			basespec.ErrInvalid,
		)
	}
	return directory, source.ValidateManagedPackageDirectory(directory)
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
			Locator: basespec.Locator(skillartifact.DefinitionFileName),
			Content: append([]byte(nil), skillMD...),
		}}, append([]byte(nil), skillMD...), nil
	}

	normalized, err := source.NormalizeManagedPackagePublication(
		source.ManagedPackagePublication{
			Directory: "package",
			Files:     input,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	var found []byte
	for _, file := range normalized.Files {
		if file.Locator != skillartifact.DefinitionFileName {
			continue
		}
		found = append([]byte(nil), file.Content...)
		break
	}
	if len(found) == 0 {
		return nil, nil, fmt.Errorf(
			"%w: managed skill package must contain %q",
			basespec.ErrInvalid,
			skillartifact.DefinitionFileName,
		)
	}
	if len(skillMD) != 0 && !bytes.Equal(skillMD, found) {
		return nil, nil, fmt.Errorf(
			"%w: request SKILL.md differs from package SKILL.md",
			basespec.ErrInvalid,
		)
	}
	return normalized.Files, found, nil
}

const managedSkillIdempotencyKeyPrefix = "managed-skill:"

type managedSkillArtifactData struct {
	ExpectedSourceRevision uint64 `json:"expectedSourceRevision,omitempty"`
	ExpectedGeneration     string `json:"expectedGeneration,omitempty"`
}

func validateManagedSkillOperationKey(value string) error {
	return basespec.ValidateRequiredText(
		"managed Skill operation key",
		value,
		basespec.MaxLogicalNameBytes,
	)
}

func managedSkillArtifactIdempotencyKey(
	operationKey string,
) (string, error) {
	if err := validateManagedSkillOperationKey(operationKey); err != nil {
		return "", err
	}
	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(operationKey))),
		cryptoutil.DigestSHA256Prefix,
	)
	return managedSkillIdempotencyKeyPrefix + digest, nil
}

func validateManagedSkillArtifactData(
	value managedSkillArtifactData,
) error {
	switch {
	case value.ExpectedSourceRevision == 0 && value.ExpectedGeneration == "":
		return nil
	case value.ExpectedSourceRevision == 0 || value.ExpectedGeneration == "":
		return fmt.Errorf(
			"%w: managed Skill source revision and generation must be supplied together",
			basespec.ErrInvalid,
		)
	}
	return basespec.ValidateSourceGeneration(value.ExpectedGeneration)
}

func validateManagedSkillPublicationIntent(
	value managedSkillArtifactData,
) error {
	if err := validateManagedSkillArtifactData(value); err != nil {
		return err
	}
	if value.ExpectedSourceRevision == 0 {
		return fmt.Errorf(
			"%w: managed Skill expected Source revision is required",
			basespec.ErrInvalid,
		)
	}
	return nil
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

func managedSkillPublicationIntentOf(
	raw json.RawMessage,
) (managedSkillArtifactData, error) {
	value, err := decodeManagedSkillArtifactData(raw)
	if err != nil {
		return managedSkillArtifactData{}, err
	}
	if err := validateManagedSkillPublicationIntent(value); err != nil {
		return managedSkillArtifactData{}, err
	}
	return value, nil
}

func managedSkillPackageDirectory(
	operationKey string,
	skillName string,
) (basespec.Locator, error) {
	if err := validateManagedSkillOperationKey(operationKey); err != nil {
		return "", err
	}
	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(operationKey))),
		cryptoutil.DigestSHA256Prefix,
	)
	return basespec.Locator(path.Join("packages", digest, skillName)), nil
}

func validateManagedSkillOperationIntent(
	value artifact.Artifact,
	operationKey string,
	sourceID basespec.SourceID,
	skillLocator basespec.Locator,
	artifactName string,
	enabled bool,
) error {
	idempotencyKey, err := managedSkillArtifactIdempotencyKey(operationKey)
	if err != nil {
		return err
	}
	if value.IdempotencyKey != idempotencyKey ||
		value.Kind != skillartifact.Kind ||
		value.Adoption != artifact.AdoptionPinned ||
		value.Binding.SourceID != sourceID ||
		value.Binding.Locator != skillLocator ||
		value.Binding.ExpectedKind != skillartifact.Kind ||
		value.Name != artifactName ||
		value.Enabled != enabled {
		return fmt.Errorf(
			"%w: managed Skill operation %q conflicts with its existing Artifact intent",
			basespec.ErrConflict,
			operationKey,
		)
	}
	return nil
}

func (a *API) findManagedSkillByOperation(
	ctx context.Context,
	bundle collection.CollectionRef,
	operationKey string,
) (*artifact.Artifact, error) {
	idempotencyKey, err := managedSkillArtifactIdempotencyKey(operationKey)
	if err != nil {
		return nil, err
	}
	values, err := a.dependencies.Artifacts.ListByCollection(ctx, bundle)
	if err != nil {
		return nil, err
	}
	var found *artifact.Artifact
	for _, value := range values {
		if value.Kind != skillartifact.Kind ||
			value.IdempotencyKey != idempotencyKey {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf(
				"%w: managed Skill operation %q has multiple Artifacts",
				basespec.ErrConflict,
				operationKey,
			)
		}
		copyValue := value.Clone()
		found = &copyValue
	}
	return found, nil
}

func pendingManagedSkillOperationError(operationKey string, cause error) error {
	return fmt.Errorf(
		"managed Skill operation %q remains pending; retry with the same operationKey: %w",
		operationKey,
		cause,
	)
}

func pendingManagedSkillPurgeError(ref artifact.ArtifactRef, cause error) error {
	return fmt.Errorf(
		"managed Skill purge for Artifact %q may have completed only the source-side step; reload and retry if the Artifact remains: %w",
		ref.ArtifactID,
		cause,
	)
}
