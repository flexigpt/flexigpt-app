package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

const (
	RootKind                 artifactstoreSpec.RootKind       = "workspace.root"
	RootDataSchemaID         artifactstoreSpec.SchemaID       = "workspace.root-data.v1"
	AttachmentDataSchemaID   artifactstoreSpec.SchemaID       = "workspace.attachment-data.v1"
	CollectionKind           artifactstoreSpec.CollectionKind = "workspace.collection"
	CollectionDataSchemaID   artifactstoreSpec.SchemaID       = "workspace.collection-data.v1"
	NativeFrontendID         artifactstoreSpec.FrontendID     = "workspace.native"
	CapabilityProfileVersion                                  = "1"
)

const (
	KindWorkspaceDefinition artifactstoreSpec.ArtifactKind = "workspace.definition"
	KindAgentDefinition     artifactstoreSpec.ArtifactKind = "agent.definition"
	KindSkillDefinition     artifactstoreSpec.ArtifactKind = "skill.definition"
	KindModelDefinition     artifactstoreSpec.ArtifactKind = "model.definition"
	KindMCPServerDefinition artifactstoreSpec.ArtifactKind = "mcp.server.definition"
	KindToolDefinition      artifactstoreSpec.ArtifactKind = "tool.definition"
	KindInstructionDocument artifactstoreSpec.ArtifactKind = "instruction.document"
	KindContextDocument     artifactstoreSpec.ArtifactKind = "context.document"
)

const (
	RolePrimary         artifactstoreSpec.AttachmentRole = "primary"
	RoleAttachedPackage artifactstoreSpec.AttachmentRole = "attached-package"
	RoleBuiltIn         artifactstoreSpec.AttachmentRole = "built-in"
	RoleAppLibrary      artifactstoreSpec.AttachmentRole = "app-library"
	RoleOverlay         artifactstoreSpec.AttachmentRole = "overlay"
)

var (
	ErrInvalidWorkspace               = errors.New("workspace: invalid workspace")
	ErrNotWorkspace                   = errors.New("workspace: root is not a workspace")
	ErrAmbiguousWorkspaceDefinition   = errors.New("workspace: multiple workspace definitions")
	ErrWorkspaceDefinitionUnavailable = errors.New("workspace: workspace definition unavailable")
	ErrProjectionUnavailable          = errors.New("workspace: projector unavailable")
	ErrReferenceUnresolved            = errors.New("workspace: reference unresolved")
)

type RootMode string

const (
	RootModeEmpty      RootMode = "empty"
	RootModeFilesystem RootMode = "filesystem"
)

type DiscoveryRoot struct {
	Root            artifactstoreSpec.SourceLocator `json:"root"`
	Recursive       bool                            `json:"recursive"`
	IncludePatterns []string                        `json:"includePatterns,omitempty"`
}

type DiscoveryPreferences struct {
	AdditionalLocators []artifactstoreSpec.SourceLocator `json:"additionalLocators,omitempty"`
	AdditionalRoots    []DiscoveryRoot                   `json:"additionalRoots,omitempty"`
	IncludeReadme      bool                              `json:"includeReadme,omitempty"`
}

type AttachedPackagePreferences struct {
	DiscoverRecursively bool `json:"discoverRecursively,omitempty"`
}

type DisplayPreferences struct {
	DefaultCategory string `json:"defaultCategory,omitempty"`
}

type RootData struct {
	Mode                       RootMode                   `json:"mode"`
	PrimarySourceID            artifactstoreSpec.SourceID `json:"primarySourceID,omitempty"`
	RootTrustReference         string                     `json:"rootTrustReference,omitempty"`
	DiscoveryPreferences       DiscoveryPreferences       `json:"discoveryPreferences"`
	AttachedPackagePreferences AttachedPackagePreferences `json:"attachedPackagePreferences"`
	CapabilityProfileVersion   string                     `json:"capabilityProfileVersion"`
	DisplayPreferences         DisplayPreferences         `json:"displayPreferences"`
}

type AttachmentData struct {
	Recursive     *bool `json:"recursive,omitempty"`
	Authoritative *bool `json:"authoritative,omitempty"`
}

type CollectionData struct {
	ArtifactKind artifactstoreSpec.ArtifactKind `json:"artifactKind"`
}

type Workspace struct {
	Root        artifactstoreSpec.ArtifactRoot
	Data        RootData
	Attachments []artifactstoreSpec.RootSourceAttachment
}

type FilesystemSelectionRequest struct {
	DisplayName         string
	Description         string
	RootPath            string
	FollowSymlinks      bool
	ManagedByApp        bool
	TrustReference      string
	Discovery           DiscoveryPreferences
	DiscoverImmediately bool
}

type EmptyWorkspaceRequest struct {
	DisplayName         string
	Description         string
	TrustReference      string
	Discovery           DiscoveryPreferences
	DiscoverImmediately bool
}

type RefreshResult struct {
	Workspace   Workspace
	Bootstrap   *artifactstoreSpec.ScanResult
	Published   artifactstoreSpec.ScanResult
	Sync        artifactstoreSpec.RecordSyncResult
	Catalog     Catalog
	Diagnostics []artifactstoreSpec.Diagnostic
}

type KindDescriptor struct {
	Kind                  artifactstoreSpec.ArtifactKind
	DefinitionSchemaID    artifactstoreSpec.SchemaID
	CollectionSlug        artifactstoreSpec.CollectionSlug
	CollectionDisplayName string
}

type CatalogResource struct {
	Record         artifactstoreSpec.ArtifactRecord
	Definition     artifactstoreSpec.CanonicalDefinition
	Collection     *artifactstoreSpec.ArtifactCollection
	Occurrence     *artifactstoreSpec.CatalogResource
	CatalogCurrent bool
}

type Catalog struct {
	Workspace  Workspace
	Generation artifactstoreSpec.RootCatalogGeneration
	Resources  []CatalogResource
	Unrecorded []artifactstoreSpec.CatalogResource
}

type Reference struct {
	RecordID *artifactstoreSpec.RecordID
	Selector *artifactstoreSpec.ArtifactSelector
}

type ProjectionInput struct {
	Workspace  Workspace
	Record     artifactstoreSpec.ArtifactRecord
	Definition artifactstoreSpec.CanonicalDefinition
}

type Projection struct {
	Kind        artifactstoreSpec.ArtifactKind
	RecordID    artifactstoreSpec.RecordID
	Value       any
	Diagnostics []artifactstoreSpec.Diagnostic
}

type ProjectionDiagnosticError struct {
	Kind        artifactstoreSpec.ArtifactKind
	Diagnostics []artifactstoreSpec.Diagnostic
}

func (e *ProjectionDiagnosticError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf(
		"workspace projector for %q reported %d diagnostic(s)",
		e.Kind,
		len(e.Diagnostics),
	)
}

func (e *ProjectionDiagnosticError) Unwrap() error {
	return ErrProjectionUnavailable
}

type ProjectedWorkspaceDefinition struct {
	RecordID   artifactstoreSpec.RecordID
	Discovery  DiscoveryPreferences
	Definition json.RawMessage
}

type ProjectedSkillArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type ProjectedSkill struct {
	RecordID         artifactstoreSpec.RecordID
	DefinitionDigest artifactstoreSpec.Digest
	SourceID         artifactstoreSpec.SourceID
	Locator          artifactstoreSpec.SourceLocator
	Name             string
	DisplayName      string
	Description      string
	Insert           string
	Arguments        []ProjectedSkillArgument
	Markdown         string
	Frontmatter      json.RawMessage
}

type ProjectedDocument struct {
	RecordID         artifactstoreSpec.RecordID
	DefinitionDigest artifactstoreSpec.Digest
	Kind             artifactstoreSpec.ArtifactKind
	SourceID         artifactstoreSpec.SourceID
	Locator          artifactstoreSpec.SourceLocator
	Name             string
	Markdown         string
}

// ProjectedStructuredDefinition is the default projection for native Workspace
// artifact kinds whose external domain adapter has not been installed. The
// original canonical JSON remains available without coupling Workspace to a
// legacy persistence model.
type ProjectedStructuredDefinition struct {
	RecordID         artifactstoreSpec.RecordID
	DefinitionDigest artifactstoreSpec.Digest
	Kind             artifactstoreSpec.ArtifactKind
	SourceID         artifactstoreSpec.SourceID
	Locator          artifactstoreSpec.SourceLocator
	Name             string
	Definition       json.RawMessage
}

type ResourceProjector interface {
	Kind() artifactstoreSpec.ArtifactKind
	Project(ctx context.Context, input ProjectionInput) (any, []artifactstoreSpec.Diagnostic)
}

type LoadPlanItem struct {
	Resource   CatalogResource
	Projection Projection
	Dependency artifactstoreSpec.DependencyGraph
}

type LoadPlan struct {
	RootID      artifactstoreSpec.RootID
	Generation  artifactstoreSpec.RootCatalogGeneration
	Items       []LoadPlanItem
	Diagnostics []artifactstoreSpec.Diagnostic
}

type DiscoveryInput struct {
	Workspace             Workspace
	DefinitionPreferences DiscoveryPreferences
	FrontendIDs           []artifactstoreSpec.FrontendID
}

type DiscoveryPlanner interface {
	BuildBootstrapPlan(ctx context.Context, input DiscoveryInput) (artifactstoreSpec.ScanPlan, error)
	BuildExpandedPlan(ctx context.Context, input DiscoveryInput) (artifactstoreSpec.ScanPlan, error)
}

type YAMLDecoder func(ctx context.Context, content []byte) (json.RawMessage, error)

// YAMLDecoder implementations must reject duplicate mapping keys, multiple
// documents, non-string mapping keys, unsafe tags, and alias expansion. They
// must return one bounded JSON object.

type ArtifactStore interface {
	RegisterArtifactFrontend(frontend artifactstoreSpec.ArtifactFrontend) error
	RegisterRootKindHook(hook artifactstoreSpec.RootKindHook) error
	RegisterCollectionKindHook(hook artifactstoreSpec.CollectionKindHook) error

	CreateRoot(ctx context.Context, draft artifactstoreSpec.RootDraft) (artifactstoreSpec.ArtifactRoot, error)
	GetRoot(ctx context.Context, rootID artifactstoreSpec.RootID) (artifactstoreSpec.ArtifactRoot, error)
	ListRoots(ctx context.Context, includeSoftDeleted bool) ([]artifactstoreSpec.ArtifactRoot, error)
	DeleteRoot(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
		expectedModifiedAt time.Time,
	) (artifactstoreSpec.ArtifactRoot, error)

	CreateSource(ctx context.Context, draft artifactstoreSpec.SourceDraft) (artifactstoreSpec.ArtifactSource, error)
	DeleteSource(
		ctx context.Context,
		sourceID artifactstoreSpec.SourceID,
		expectedModifiedAt time.Time,
	) error
	AttachSource(
		ctx context.Context,
		draft artifactstore.RootSourceAttachmentDraft,
	) (artifactstoreSpec.RootSourceAttachment, error)
	ListRootSources(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
	) ([]artifactstoreSpec.RootSourceAttachment, error)

	ScanRoot(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
		plan artifactstoreSpec.ScanPlan,
	) (artifactstoreSpec.ScanResult, error)
	GetRootCatalogGeneration(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
	) (artifactstoreSpec.RootCatalogGeneration, error)
	ListCatalogResourcesForRoot(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
	) ([]artifactstoreSpec.CatalogResource, error)
	GetDefinitionByDigest(
		ctx context.Context,
		digest artifactstoreSpec.Digest,
	) (artifactstoreSpec.CanonicalDefinition, error)

	EnsureBaseCollection(
		ctx context.Context,
		draft artifactstoreSpec.CollectionDraft,
	) (artifactstoreSpec.ArtifactCollection, error)
	ListCollections(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
		includeSoftDeleted bool,
	) ([]artifactstoreSpec.ArtifactCollection, error)
	ListRecords(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
	) ([]artifactstoreSpec.ArtifactRecord, error)
	GetRecord(
		ctx context.Context,
		recordID artifactstoreSpec.RecordID,
	) (artifactstoreSpec.ArtifactRecord, error)
	SyncRecords(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
		policy artifactstoreSpec.RecordSyncPolicy,
	) (artifactstoreSpec.RecordSyncResult, error)
	FindCandidates(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
		selector artifactstoreSpec.ArtifactSelector,
	) ([]artifactstoreSpec.DependencyCandidate, error)
	BuildDependencyGraph(
		ctx context.Context,
		recordID artifactstoreSpec.RecordID,
	) (artifactstoreSpec.DependencyGraph, error)
}

var _ ArtifactStore = (*artifactstore.Store)(nil)
