package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	assistantpresetSpec "github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
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
	workspaceDefinitionJSONLocator = ".flexigpt/workspace.json"
	workspaceDefinitionYAMLLocator = ".flexigpt/workspace.yaml"
	workspaceDefinitionYMLLocator  = ".flexigpt/workspace.yml"
	workspaceMCPDotJSONLocator     = ".mcp.json"
	workspaceMCPDotsJSONLocator    = ".mcps.json"
	workspaceMCPJSONLocator        = "mcp.json"
	workspaceMCPsJSONLocator       = "mcps.json"
	workspaceAgentsLocator         = "AGENTS.md"
	workspaceReadmeLocator         = "README.md"
	workspaceAgentsDirectory       = ".flexigpt/agents/"
	workspaceModelsDirectory       = ".flexigpt/models/"
	workspaceMCPDirectory          = ".flexigpt/mcp/"
	workspaceToolsDirectory        = ".flexigpt/tools/"
	workspaceSkillsDirectory       = ".skills"
	workspaceSkillMarkdownFileName = "skill.md"
	workspaceDefinitionSchemaV1    = "1"
	workspacePrimaryPriority       = artifactstoreSpec.MaxAttachmentPriority
	workspaceRecordNameHashLength  = 12
	workspaceRecordNameFallback    = "artifact"
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

type AttachedSource struct {
	Attachment artifactstoreSpec.RootSourceAttachment `json:"attachment"`
	Source     artifactstoreSpec.ArtifactSource       `json:"source"`
}

type Workspace struct {
	Root        artifactstoreSpec.ArtifactRoot           `json:"root"`
	Data        RootData                                 `json:"data"`
	Attachments []artifactstoreSpec.RootSourceAttachment `json:"attachments"`
	Sources     []AttachedSource                         `json:"sources"`
}

type FilesystemSelectionRequest struct {
	DisplayName         string               `json:"displayName"`
	Description         string               `json:"description,omitempty"`
	RootPath            string               `json:"rootPath"`
	FollowSymlinks      bool                 `json:"followSymlinks"`
	ManagedByApp        bool                 `json:"managedByApp"`
	TrustReference      string               `json:"trustReference,omitempty"`
	Discovery           DiscoveryPreferences `json:"discovery"`
	DiscoverImmediately bool                 `json:"discoverImmediately"`
}

type EmptyWorkspaceRequest struct {
	DisplayName         string               `json:"displayName"`
	Description         string               `json:"description,omitempty"`
	TrustReference      string               `json:"trustReference,omitempty"`
	Discovery           DiscoveryPreferences `json:"discovery"`
	DiscoverImmediately bool                 `json:"discoverImmediately"`
}

type UpdateWorkspaceRequest struct {
	RootID              artifactstoreSpec.RootID    `json:"rootID"`
	ExpectedModifiedAt  time.Time                   `json:"expectedModifiedAt"`
	DisplayName         *string                     `json:"displayName,omitempty"`
	Description         *string                     `json:"description,omitempty"`
	Enabled             *bool                       `json:"enabled,omitempty"`
	TrustReference      *string                     `json:"trustReference,omitempty"`
	Discovery           *DiscoveryPreferences       `json:"discovery,omitempty"`
	AttachedPackages    *AttachedPackagePreferences `json:"attachedPackages,omitempty"`
	DisplayPreferences  *DisplayPreferences         `json:"displayPreferences,omitempty"`
	DiscoverImmediately bool                        `json:"discoverImmediately"`
}

type DeleteWorkspaceRequest struct {
	RootID             artifactstoreSpec.RootID `json:"rootID"`
	ExpectedModifiedAt time.Time                `json:"expectedModifiedAt"`
}

// AttachSourceRequest attaches an existing Artifact Store source to a
// Workspace. Source lifecycle remains owned by Artifact Store.
type AttachSourceRequest struct {
	RootID              artifactstoreSpec.RootID         `json:"rootID"`
	SourceID            artifactstoreSpec.SourceID       `json:"sourceID"`
	Role                artifactstoreSpec.AttachmentRole `json:"role"`
	Priority            int                              `json:"priority"`
	AttachmentData      AttachmentData                   `json:"attachmentData"`
	DiscoverImmediately bool                             `json:"discoverImmediately"`
}

// EmbeddedSourceAttachmentRequest creates an app-local embedded filesystem
// source and attaches it to an existing Workspace.
type EmbeddedSourceAttachmentRequest struct {
	RootID              artifactstoreSpec.RootID         `json:"rootID"`
	DisplayName         string                           `json:"displayName"`
	ProviderKey         string                           `json:"providerKey"`
	RootLocator         artifactstoreSpec.SourceLocator  `json:"rootLocator"`
	Role                artifactstoreSpec.AttachmentRole `json:"role"`
	Priority            int                              `json:"priority"`
	AttachmentData      AttachmentData                   `json:"attachmentData"`
	DiscoverImmediately bool                             `json:"discoverImmediately"`
}

type CatalogResource struct {
	Record         artifactstoreSpec.ArtifactRecord
	Definition     artifactstoreSpec.CanonicalDefinition
	Collection     *artifactstoreSpec.ArtifactCollection
	Occurrence     *artifactstoreSpec.CatalogResource
	CatalogCurrent bool
}

type Catalog struct {
	Workspace         Workspace
	Generation        artifactstoreSpec.RootCatalogGeneration
	Resources         []CatalogResource
	Unrecorded        []artifactstoreSpec.CatalogResource
	UnresolvedRecords []artifactstoreSpec.ArtifactRecord
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
	Assets           []artifactstoreSpec.AssetManifestEntry

	Skill    skillSpec.Skill
	SkillRef skillSpec.SkillRef
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

type ProjectedTool struct {
	Tool    toolSpec.Tool
	ToolRef toolSpec.ToolRef
}

type ProjectedMCPServer struct {
	Server mcpSpec.MCPServerConfig
}

type ProjectedModel struct {
	Provider modelpresetSpec.ProviderPreset
	Model    modelpresetSpec.ModelPreset
	Ref      modelpresetSpec.ModelPresetRef
}

type AgentToolSelection struct {
	Selector        artifactstoreSpec.ArtifactSelector `json:"selector"`
	ToolChoicePatch *toolSpec.ToolChoicePatch          `json:"toolChoicePatch,omitempty"`
}

type AgentSkillSelection struct {
	Selector          artifactstoreSpec.ArtifactSelector `json:"selector"`
	PreLoadAsActive   bool                               `json:"preLoadAsActive,omitempty"`
	UseAsInstructions bool                               `json:"useAsInstructions,omitempty"`
}

type AgentMCPToolSelection struct {
	ToolName       string                    `json:"toolName"`
	ApprovalRule   *mcpSpec.MCPApprovalRule  `json:"approvalRule,omitempty"`
	ExecutionMode  *mcpSpec.MCPExecutionMode `json:"executionMode,omitempty"`
	AppResourceURI string                    `json:"appResourceURI,omitempty"`
	Visibility     []string                  `json:"visibility,omitempty"`
}

type AgentMCPServerSelection struct {
	Selector                  artifactstoreSpec.ArtifactSelector `json:"selector"`
	ToolExposure              mcpSpec.MCPToolExposure            `json:"toolExposure,omitempty"`
	SelectedTools             []AgentMCPToolSelection            `json:"selectedTools,omitempty"`
	IncludeServerInstructions bool                               `json:"includeServerInstructions,omitempty"`
}

// AgentDefinitionDocument is the portable agent data stored in a canonical
// definition. References use selectors and therefore contain no app-local
// bundle, item, source, or record IDs.
type AgentDefinitionDocument struct {
	StartingText                     string                              `json:"startingText,omitempty"`
	StartingIncludeModelSystemPrompt *bool                               `json:"startingIncludeModelSystemPrompt,omitempty"`
	StartingModel                    *artifactstoreSpec.ArtifactSelector `json:"startingModel,omitempty"`
	StartingTools                    []AgentToolSelection                `json:"startingTools,omitempty"`
	StartingSkills                   []AgentSkillSelection               `json:"startingSkills,omitempty"`
	StartingMCPServers               []AgentMCPServerSelection           `json:"startingMCPServers,omitempty"`
}

type ProjectedAgent struct {
	BundleID           bundleitemutils.BundleID
	Preset             assistantpresetSpec.AssistantPreset
	StartingModel      *artifactstoreSpec.ArtifactSelector
	StartingTools      []AgentToolSelection
	StartingSkills     []AgentSkillSelection
	StartingMCPServers []AgentMCPServerSelection
	Definition         json.RawMessage
}

type TransferResult struct {
	Record  artifactstoreSpec.ArtifactRecord
	Refresh *RefreshResult
}

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
// must return one canonical, bounded JSON object.

type ArtifactStore interface {
	RegisterArtifactFrontend(frontend artifactstoreSpec.ArtifactFrontend) error
	RegisterRootKindHook(hook artifactstoreSpec.RootKindHook) error
	RegisterCollectionKindHook(hook artifactstoreSpec.CollectionKindHook) error
	RegisterDependencyResolver(resolver artifactstoreSpec.DependencyResolver) error
	RegisterArtifactVersionMatcher(matcher artifactstoreSpec.ArtifactVersionMatcher) error

	CreateRoot(ctx context.Context, draft artifactstoreSpec.RootDraft) (artifactstoreSpec.ArtifactRoot, error)
	GetRoot(ctx context.Context, rootID artifactstoreSpec.RootID) (artifactstoreSpec.ArtifactRoot, error)
	ListRoots(ctx context.Context, includeSoftDeleted bool) ([]artifactstoreSpec.ArtifactRoot, error)
	DeleteRoot(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
		expectedModifiedAt time.Time,
	) (artifactstoreSpec.ArtifactRoot, error)
	UpdateRoot(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
		update artifactstore.RootUpdate,
	) (artifactstoreSpec.ArtifactRoot, error)

	CreateSource(ctx context.Context, draft artifactstoreSpec.SourceDraft) (artifactstoreSpec.ArtifactSource, error)
	GetSource(ctx context.Context, sourceID artifactstoreSpec.SourceID) (artifactstoreSpec.ArtifactSource, error)
	DeleteSource(
		ctx context.Context,
		sourceID artifactstoreSpec.SourceID,
		expectedModifiedAt time.Time,
	) error
	AttachSource(
		ctx context.Context,
		draft artifactstore.RootSourceAttachmentDraft,
	) (artifactstoreSpec.RootSourceAttachment, error)
	GetRootSourceAttachment(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
		sourceID artifactstoreSpec.SourceID,
	) (artifactstoreSpec.RootSourceAttachment, error)
	ListRootSources(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
	) ([]artifactstoreSpec.RootSourceAttachment, error)
	DetachSource(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
		sourceID artifactstoreSpec.SourceID,
		expectedModifiedAt time.Time,
	) error

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
	ExplainDependencyResolution(
		ctx context.Context,
		rootID artifactstoreSpec.RootID,
		selector artifactstoreSpec.ArtifactSelector,
	) (artifactstoreSpec.DependencyExplanation, error)
	BuildDependencyGraph(
		ctx context.Context,
		recordID artifactstoreSpec.RecordID,
	) (artifactstoreSpec.DependencyGraph, error)

	ExportRecord(
		ctx context.Context,
		recordID artifactstoreSpec.RecordID,
	) (artifactstoreSpec.ExportedRecord, error)
	ImportDefinition(
		ctx context.Context,
		request artifactstoreSpec.ImportDefinitionRequest,
	) (artifactstoreSpec.ArtifactRecord, error)
	CaptureRecord(
		ctx context.Context,
		request artifactstoreSpec.CaptureRecordRequest,
	) (artifactstoreSpec.ArtifactRecord, error)
	ForkRecord(
		ctx context.Context,
		request artifactstoreSpec.ForkRecordRequest,
	) (artifactstoreSpec.ArtifactRecord, error)
}

var _ ArtifactStore = (*artifactstore.Store)(nil)
