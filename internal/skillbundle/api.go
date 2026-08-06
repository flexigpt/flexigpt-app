package skillbundle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"path"
	"sort"
	"sync/atomic"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
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
	RootID                   basespec.RootID
	CollectionID             basespec.CollectionID
	ManagedSourceID          basespec.SourceID
	DisplayName              string
	Description              string
	Enabled                  bool
	LogicalName              basespec.LogicalName
	LogicalVersion           basespec.LogicalVersion
	Labels                   map[string]string
	PortableDefinitionDigest *cryptoutil.Digest
	Attachments              []AttachmentDraft
}

type UpdateBundleRequest struct {
	Bundle           collection.CollectionRef
	ExpectedRevision uint64
	DisplayName      string
	Description      string
	Enabled          bool
}

type AttachmentDraft struct {
	SourceID              basespec.SourceID
	Role                  basespec.AttachmentRole
	Enabled               bool
	DiscoveryRoot         basespec.Locator
	ExpectedMemberDigests map[basespec.Locator]cryptoutil.Digest
}

type CreateManagedSkillRequest struct {
	Bundle                     collection.CollectionRef
	ExpectedCollectionRevision uint64
	ArtifactID                 basespec.ArtifactID
	SkillName                  string
	SKILLMD                    []byte
	ExpectedArtifactRevision   uint64

	// Document is an optional structured authoring input. Serialization is
	// delegated to agentskills-go. It is mutually exclusive with SKILLMD.
	Document *agentskillsSpec.SkillDocument
	Files    []source.ManagedPackageFile
	Enabled  bool
}

type CreateManagedSkillResponse struct {
	Artifact artifact.Artifact
	Address  artifact.ArtifactAddress
}

// ManagedSkillDocument is the editable projection for a managed Skill.
// It deliberately contains the canonical SKILL.md document only. It never
// exposes Source configuration, a native filesystem path, or package internals.
type ManagedSkillDocument struct {
	Artifact artifact.Artifact
	Document agentskillsSpec.SkillDocument
}

type AdoptSkillRequest struct {
	Bundle                  collection.CollectionRef
	Occurrence              catalog.OccurrenceKey
	ArtifactID              basespec.ArtifactID
	ExpectedCatalogRevision uint64
	Name                    string
	Enabled                 bool
}

type PinSkillRequest struct {
	Bundle                     collection.CollectionRef
	ExpectedCollectionRevision uint64
	ArtifactID                 basespec.ArtifactID
	Binding                    artifact.SourceBinding
	Name                       string
	Enabled                    bool
}

type BuiltInBundleTopology struct {
	RootID                   basespec.RootID                        `json:"-"`
	CollectionID             basespec.CollectionID                  `json:"-"`
	SourceID                 basespec.SourceID                      `json:"-"`
	LogicalName              basespec.LogicalName                   `json:"-"`
	LogicalVersion           basespec.LogicalVersion                `json:"-"`
	DisplayName              string                                 `json:"-"`
	Description              string                                 `json:"-"`
	Labels                   map[string]string                      `json:"-"`
	Enabled                  bool                                   `json:"-"`
	DiscoveryRoot            basespec.Locator                       `json:"-"`
	ExpectedMemberDigests    map[basespec.Locator]cryptoutil.Digest `json:"-"`
	PortableDefinitionDigest cryptoutil.Digest                      `json:"-"`
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
	if value.Kind != CollectionKind {
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
	if draft.Role == RoleBuiltIn {
		return Bundle{}, fmt.Errorf(
			"%w: skill bundle built-in attachment role is reserved for bootstrap",
			basespec.ErrInvalid,
		)
	}
	current, err := a.GetBundle(ctx, bundle)
	if err != nil {
		return Bundle{}, err
	}
	if draft.Role == RoleManaged {
		for _, attachment := range current.Attachments {
			if attachment.Role == RoleManaged {
				return Bundle{}, fmt.Errorf(
					"%w: skill bundle already has a managed attachment",
					basespec.ErrConflict,
				)
			}
		}
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

// BuildLinkedPortableBundleDefinition is intentionally unavailable.
//
// Previous implementations emitted current Source locators as portable member
// locators. Managed Source locators can contain local Artifact UUIDs and are
// therefore not shareable. A future exporter must construct a package-relative
// closure with logical names and an externally supplied package SHA-256.
func (a *API) BuildLinkedPortableBundleDefinition(
	ctx context.Context,
	ref collection.CollectionRef,
) (definition.CollectionDefinition, error) {
	return definition.CollectionDefinition{}, fmt.Errorf(
		"%w: linked Skill Bundle export is disabled until package-closure export is implemented",
		basespec.ErrUnsupported,
	)
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
	attachment, _, err := managedAttachmentForRole(bundle, RoleManaged)
	if err != nil {
		return ManagedSkillDocument{}, err
	}
	if attachment.SourceID != value.Binding.SourceID {
		return ManagedSkillDocument{}, fmt.Errorf(
			"%w: Skill is not stored in this bundle's managed source",
			basespec.ErrUnsupported,
		)
	}
	if _, err := managedSkillPackageDirectoryOf(value.Binding); err != nil {
		return ManagedSkillDocument{}, err
	}
	if value.ResolvedDefinition == nil {
		return ManagedSkillDocument{}, fmt.Errorf(
			"%w: managed Skill has no current definition",
			basespec.ErrReferenceUnresolved,
		)
	}

	definitionValue, err := definition.ReadCanonical(
		ctx, a.dependencies.Definitions, value.RootID, *value.ResolvedDefinition,
	)
	if err != nil {
		return ManagedSkillDocument{}, err
	}
	document, err := skillartifact.DocumentFromDefinition(definitionValue)
	if err != nil {
		return ManagedSkillDocument{}, err
	}
	return ManagedSkillDocument{Artifact: value.Clone(), Document: document}, nil
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
	if role == RoleManaged || role == RoleBuiltIn {
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
	if request.Binding.ExpectedKind != skillartifact.Kind {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: skill bundle pins only support %q",
			basespec.ErrInvalid,
			skillartifact.Kind,
		)
	}
	role, err := bundleAttachmentRole(bundle, request.Binding.SourceID)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if role == RoleManaged || role == RoleBuiltIn {
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
	bundleRef := bundle.Collection.Ref()
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
			"%w: built-in Skill packages are protected and read-only",
			basespec.ErrProtected,
		)
	case RoleManaged:
		if value.Adoption != artifact.AdoptionPinned {
			return fmt.Errorf(
				"%w: managed Skill Artifact must remain pinned until purged",
				basespec.ErrConflict,
			)
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
		return a.dependencies.Artifacts.Purge(ctx, ref, expectedRevision)
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
	if _, err := a.refreshBundle(ctx, bundleRef, false); err != nil {
		return pendingManagedSkillPurgeError(value.Ref(), fmt.Errorf(
			"skill package and artifact were removed, but bundle refresh failed: %w",
			err,
		))
	}
	return nil
}

func (a *API) EnsureBuiltInBundleTopology(
	ctx context.Context,
	request BuiltInBundleTopology,
) (Bundle, error) {
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return Bundle{}, err
	}
	bundle, err := a.createBundle(ctx, CreateBundleRequest{
		RootID:                   request.RootID,
		CollectionID:             request.CollectionID,
		LogicalName:              request.LogicalName,
		LogicalVersion:           request.LogicalVersion,
		DisplayName:              request.DisplayName,
		Description:              request.Description,
		Labels:                   request.Labels,
		Enabled:                  request.Enabled,
		PortableDefinitionDigest: &request.PortableDefinitionDigest,
		Attachments: []AttachmentDraft{{
			SourceID:              request.SourceID,
			Role:                  RoleBuiltIn,
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
			SchemaVersion:            CollectionSchemaVersion,
			DiscoveryPolicyRevision:  DiscoveryPolicyRevision,
			LogicalName:              request.LogicalName,
			LogicalVersion:           request.LogicalVersion,
			Labels:                   request.Labels,
			PortableDefinitionDigest: &request.PortableDefinitionDigest,
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
			if attachment.Role != RoleBuiltIn || !attachment.Enabled ||
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
						Role:                       RoleBuiltIn,
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
	request CreateManagedSkillRequest,
) (CreateManagedSkillResponse, error) {
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	if err := basespec.ValidateArtifactID(request.ArtifactID); err != nil {
		return CreateManagedSkillResponse{}, err
	}
	return a.createManagedSkill(
		ctx,
		request,
		true,
	)
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

	data, err := EncodeCollectionData(CollectionData{
		SchemaVersion:            CollectionSchemaVersion,
		DiscoveryPolicyRevision:  DiscoveryPolicyRevision,
		LogicalName:              request.LogicalName,
		LogicalVersion:           request.LogicalVersion,
		Labels:                   request.Labels,
		ManagedSourceID:          request.ManagedSourceID,
		PortableDefinitionDigest: cryptoutil.CloneDigest(request.PortableDefinitionDigest),
	})
	if err != nil {
		return Bundle{}, err
	}

	attachments := make([]collection.AttachmentDraft, 0, len(request.Attachments))
	roleCounts := make(map[basespec.AttachmentRole]int)
	for _, draft := range request.Attachments {
		if draft.Role == RoleBuiltIn && !allowBuiltInAttachment {
			return Bundle{}, fmt.Errorf(
				"%w: skill bundle built-in attachment role is reserved for bootstrap",
				basespec.ErrInvalid,
			)
		}
		roleCounts[draft.Role]++
		if draft.Role == RoleManaged && roleCounts[draft.Role] > 1 {
			return Bundle{}, fmt.Errorf(
				"%w: skill bundle can have only one managed attachment",
				basespec.ErrInvalid,
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
		for _, draft := range request.Attachments {
			if draft.Role == RoleManaged {
				return Bundle{}, fmt.Errorf(
					"%w: managedSourceID cannot be combined with an explicit managed attachment",
					basespec.ErrInvalid,
				)
			}
		}

		value, createdNew, err := a.dependencies.Sources.CreateWithStatus(
			ctx,
			request.RootID,
			source.Draft{
				ID:          request.ManagedSourceID,
				Kind:        managed.Kind,
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
			Role:     RoleManaged,
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
			Kind:        CollectionKind,
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
				attachment.Role == RoleManaged {
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
		value.Collection.Kind != CollectionKind {
		return false
	}
	if value.Collection.ID != request.CollectionID ||
		value.Collection.DisplayName != request.DisplayName ||
		value.Collection.Description != request.Description ||
		value.Collection.Enabled != request.Enabled ||
		value.Data.LogicalName != request.LogicalName ||
		value.Data.LogicalVersion != request.LogicalVersion ||
		value.Data.ManagedSourceID != request.ManagedSourceID ||
		!maps.Equal(value.Data.Labels, request.Labels) ||
		!cryptoutil.IsDigestEqual(
			value.Data.PortableDefinitionDigest,
			request.PortableDefinitionDigest,
		) {
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
			Role:          RoleManaged,
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
				sourceValue.Kind == managed.Kind &&
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
	return value.Data.PortableDefinitionDigest != nil &&
		*value.Data.PortableDefinitionDigest ==
			request.PortableDefinitionDigest &&
		attachment.RootID == request.RootID &&
		attachment.CollectionID == request.CollectionID &&
		attachment.SourceID == request.SourceID &&
		attachment.Role == RoleBuiltIn &&
		attachment.Enabled &&
		actualAttachment.DiscoveryRoot ==
			expectedAttachment.DiscoveryRoot &&
		maps.Equal(
			actualAttachment.ExpectedMemberDigests,
			expectedAttachment.ExpectedMemberDigests,
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

	autoAdoptSources := make(map[basespec.SourceID]struct{})
	for _, attachment := range bundle.Attachments {
		switch attachment.Role {
		case RoleExternal, RoleImported, RoleLibrary:
			autoAdoptSources[attachment.SourceID] = struct{}{}
		}
	}
	var policy artifact.Policy = skillArtifactPolicy{
		ids:              a.dependencies.AutoAdoptionIDProvider,
		autoAdoptSources: autoAdoptSources,
	}
	if allowProtected {
		policy = builtInSkillArtifactPolicy{}
	}
	result, err := a.dependencies.Refresh.Refresh(
		ctx,
		ref,
		plan,
		policy,
	)
	if err != nil {
		return refresh.Result{}, err
	}
	return result, nil
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
		document := *request.Document
		if document.Name == "" {
			document.Name = request.SkillName
		}
		if document.Name != request.SkillName {
			return CreateManagedSkillResponse{}, fmt.Errorf(
				"%w: structured Skill name does not match requested skill name",
				basespec.ErrInvalid,
			)
		}
		encoded, err := agentskills.MarshalSkillDocument(document)
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
	definitionValue, _, err := skillartifact.DecodeSkillDocument(skillMD,
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
		request.ArtifactID,
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
		if request.ExpectedCollectionRevision == 0 {
			return CreateManagedSkillResponse{}, fmt.Errorf(
				"%w: expected skill bundle revision is required for a new managed Skill",
				basespec.ErrInvalid,
			)
		}
		if bundle.Collection.Revision != request.ExpectedCollectionRevision {
			return CreateManagedSkillResponse{}, basespec.ErrConflict
		}

		localData, err := encodeManagedSkillArtifactData(
			managedSkillArtifactData{
				PackageSHA256: packageSHA256,
			},
		)
		if err != nil {
			return CreateManagedSkillResponse{}, err
		}

		value, pinErr := a.dependencies.Artifacts.Pin(ctx, artifact.PinRequest{
			ArtifactID:                 request.ArtifactID,
			Collection:                 request.Bundle,
			ExpectedCollectionRevision: request.ExpectedCollectionRevision,
			Binding: artifact.SourceBinding{
				SourceID:     sourceValue.ID,
				Locator:      skillLocator,
				ExpectedKind: skillartifact.Kind,
			},
			Name:    artifactName,
			Enabled: request.Enabled,
			Data:    localData,
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

	if request.ExpectedArtifactRevision == 0 {
		// Zero is both the initial-create token and the replay token for a
		// create that became pending after its Artifact was pinned. Resume only
		// when the persisted package intent is byte-for-byte equivalent.
		intent, intentErr := decodeManagedSkillArtifactData(pinned.Data)
		if intentErr != nil ||
			intent.PackageSHA256 != packageSHA256 ||
			pinned.Name != artifactName {
			return CreateManagedSkillResponse{}, fmt.Errorf(
				"%w: managed Skill Artifact %q already exists with different creation intent",
				basespec.ErrConflict,
				request.ArtifactID,
			)
		}
	} else if pinned.Revision != request.ExpectedArtifactRevision {
		return CreateManagedSkillResponse{}, basespec.ErrConflict
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

	if result, complete := managedSkillCreateResult(
		*pinned,
		sourceValue.ID,
		skillLocator,
		definitionValue.Digest,
		packageSHA256,
	); complete {
		return a.setManagedSkillEnabled(ctx, result, request.Enabled)
	}

	state, generation, err := a.dependencies.GetManagedSourceState(
		ctx,
		request.Bundle.RootID,
		sourceValue.ID,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, pending(err)
	}
	if state.ID != sourceValue.ID ||
		state.RootID != request.Bundle.RootID {
		return CreateManagedSkillResponse{}, pending(fmt.Errorf(
			"%w: managed Source state does not match the selected attachment",
			basespec.ErrInvalid,
		))
	}

	publishPackage := a.dependencies.PublishManagedPackage
	if allowBuiltInAttachment {
		publishPackage = a.dependencies.PublishProtectedManagedPackage
	}
	_, _, err = publishPackage(
		ctx,
		request.Bundle.RootID,
		sourceValue.ID,
		state.Revision,
		source.ManagedPackagePublication{
			Directory:          directory,
			ExpectedGeneration: generation,
			Files:              files,
		},
	)
	if err != nil {
		return CreateManagedSkillResponse{}, pending(err)
	}

	if _, err := a.refreshBundle(
		ctx,
		request.Bundle,
		allowBuiltInAttachment,
	); err != nil {
		return CreateManagedSkillResponse{}, pending(err)
	}

	resolved, err := a.dependencies.Artifacts.Get(ctx, pinned.Ref())
	if err != nil {
		return CreateManagedSkillResponse{}, pending(err)
	}
	if result, complete := managedSkillCreateResult(
		resolved,
		sourceValue.ID,
		skillLocator,
		definitionValue.Digest,
		packageSHA256,
	); complete {
		return a.setManagedSkillEnabled(ctx, result, request.Enabled)
	}

	// Refresh reconciles source-derived fields. Package provenance is
	// collection-local authoring metadata, so it is updated afterwards using
	// the freshly reconciled Artifact revision.
	localData, err := encodeManagedSkillArtifactData(
		managedSkillArtifactData{PackageSHA256: packageSHA256},
	)
	if err != nil {
		return CreateManagedSkillResponse{}, pending(err)
	}
	resolved, err = a.dependencies.Artifacts.UpdateData(
		ctx,
		resolved.Ref(),
		resolved.Revision,
		localData,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, pending(err)
	}
	if result, complete := managedSkillCreateResult(
		resolved,
		sourceValue.ID,
		skillLocator,
		definitionValue.Digest,
		packageSHA256,
	); complete {
		return a.setManagedSkillEnabled(ctx, result, request.Enabled)
	}
	return CreateManagedSkillResponse{}, pending(
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
	expectedPackageSHA256 cryptoutil.Digest,
) (CreateManagedSkillResponse, bool) {
	intent, err := decodeManagedSkillArtifactData(value.Data)
	if err != nil ||
		intent.PackageSHA256 != expectedPackageSHA256 ||
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
		Artifact: value,
		Address:  value.Address(),
	}, true
}

func (a *API) setManagedSkillEnabled(
	ctx context.Context,
	result CreateManagedSkillResponse,
	enabled bool,
) (CreateManagedSkillResponse, error) {
	if result.Artifact.Enabled == enabled {
		return result, nil
	}
	updated, err := a.dependencies.Artifacts.SetEnabled(
		ctx,
		result.Artifact.Ref(),
		result.Artifact.Revision,
		enabled,
	)
	if err != nil {
		return CreateManagedSkillResponse{}, err
	}
	result.Artifact = updated
	result.Address = updated.Address()
	return result, nil
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
				IncludePatterns: []string{skillartifact.DefinitionFileName},
			}},
			DecoderHints: []discovery.DecoderHint{{
				Locator:    attachmentData.DiscoveryRoot,
				Recursive:  true,
				DecoderIDs: []basespec.DecoderID{skillartifact.DecoderID},
			}},
			ExpectedContentDigests: expectedContentDigests,
			AllowedDecoderIDs:      []basespec.DecoderID{skillartifact.DecoderID},
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

type skillArtifactPolicy struct {
	ids              ArtifactIDProvider
	autoAdoptSources map[basespec.SourceID]struct{}
}

func (p skillArtifactPolicy) Derive(
	ctx context.Context,
	_ collection.Collection,
	occurrence catalog.Occurrence,
	value definition.Definition,
) (artifact.Draft, bool, []diagnostic.Diagnostic, error) {
	if occurrence.Kind != skillartifact.Kind {
		return artifact.Draft{}, false, nil, nil
	}
	if _, allowed := p.autoAdoptSources[occurrence.Key.SourceID]; !allowed {
		return artifact.Draft{}, false, nil, nil
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
			Directory: locatorPackage,
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

type managedSkillArtifactData struct {
	PackageSHA256 cryptoutil.Digest `json:"packageSHA256"`
}

func validateManagedSkillArtifactData(
	value managedSkillArtifactData,
) error {
	return cryptoutil.ValidateDigest(value.PackageSHA256)
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

func managedSkillPackageDirectory(
	artifactID basespec.ArtifactID,
	skillName string,
) (basespec.Locator, error) {
	if err := basespec.ValidateArtifactID(artifactID); err != nil {
		return "", err
	}
	return basespec.Locator(
		path.Join("packages", string(artifactID), skillName),
	), nil
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
		value.Kind != skillartifact.Kind ||
		value.Adoption != artifact.AdoptionPinned ||
		value.Binding.SourceID != sourceID ||
		value.Binding.Locator != skillLocator ||
		value.Binding.ExpectedKind != skillartifact.Kind {
		return fmt.Errorf(
			"%w: managed Skill Artifact %q conflicts with its existing creation intent",
			basespec.ErrConflict,
			artifactID)
	}

	return nil
}

func managedSkillPackageDigest(
	files []source.ManagedPackageFile,
) (cryptoutil.Digest, error) {
	normalized, err := source.NormalizeManagedPackagePublication(
		source.ManagedPackagePublication{
			Directory: locatorPackage,
			Files:     files,
		},
	)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(normalized.Files)
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
		"managed Skill purge for Artifact %q may have completed only the source-side step; reload and retry if the Artifact remains: %w",
		ref.ArtifactID,
		cause,
	)
}

func emptyArtifactData() json.RawMessage {
	return json.RawMessage(jsonutil.EmptyObject)
}
