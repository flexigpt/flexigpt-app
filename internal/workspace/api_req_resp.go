package workspace

import (
	"encoding/json"
	"time"

	"github.com/flexigpt/agentskills-go/document"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type WorkspaceRef = collection.CollectionRef

type WorkspaceDiscoveryRoot struct {
	Root            basespec.Locator `json:"root"`
	Recursive       bool             `json:"recursive"`
	IncludePatterns []string         `json:"includePatterns,omitempty"`
}
type WorkspaceDiscovery struct {
	AdditionalLocators []basespec.Locator       `json:"additionalLocators,omitempty"`
	AdditionalRoots    []WorkspaceDiscoveryRoot `json:"additionalRoots,omitempty"`
	IncludeReadme      bool                     `json:"includeReadme,omitempty"`
}

type WorkspaceAttachmentSettings struct {
	Recursive     *bool `json:"recursive,omitempty"`
	Authoritative *bool `json:"authoritative,omitempty"`
}

type WorkspaceArtifactSettings struct {
	RuntimeDisabled bool `json:"runtimeDisabled"`
}

type WorkspaceOccurrenceRef struct {
	SourceID           basespec.SourceID           `json:"sourceID"`
	Locator            basespec.Locator            `json:"locator"`
	SubresourceLocator basespec.SubresourceLocator `json:"subresourceLocator,omitempty"`
}

type WorkspaceAttachmentView struct {
	SourceID          basespec.SourceID           `json:"sourceID"`
	Revision          uint64                      `json:"revision"`
	Role              basespec.AttachmentRole     `json:"role"`
	Enabled           bool                        `json:"enabled"`
	SourceDisplayName string                      `json:"sourceDisplayName,omitempty"`
	SourceKind        string                      `json:"sourceKind,omitempty"`
	Path              string                      `json:"path,omitempty"`
	Settings          WorkspaceAttachmentSettings `json:"settings"`
	Diagnostics       []diagnostic.Diagnostic     `json:"diagnostics,omitempty"`
}

// WorkspaceView is the API-safe representation of a workspace.
//
// It deliberately excludes source configuration, root data, attachment raw
// data, and the trust-reference contents. Local filesystem paths are included
// because the local Workspace management UI intentionally displays them.
type WorkspaceView struct {
	Workspace       WorkspaceRef              `json:"workspace"`
	Revision        uint64                    `json:"revision"`
	DisplayName     string                    `json:"displayName"`
	Description     string                    `json:"description,omitempty"`
	Enabled         bool                      `json:"enabled"`
	Mode            spec.Mode                 `json:"mode"`
	PrimarySourceID basespec.SourceID         `json:"primarySourceID,omitempty"`
	PrimaryPath     string                    `json:"primaryPath,omitempty"`
	Discovery       WorkspaceDiscovery        `json:"discovery"`
	Attachments     []WorkspaceAttachmentView `json:"attachments"`
}

type WorkspaceArtifactView struct {
	Artifact           artifact.ArtifactRef        `json:"artifact"`
	Revision           uint64                      `json:"revision"`
	Name               string                      `json:"name"`
	Kind               basespec.ArtifactKind       `json:"kind"`
	Enabled            bool                        `json:"enabled"`
	State              artifact.State              `json:"state"`
	Adoption           artifact.AdoptionMode       `json:"adoption"`
	ResolvedDefinition *cryptoutil.Digest          `json:"resolvedDefinition,omitempty"`
	SourceID           basespec.SourceID           `json:"sourceID"`
	Locator            basespec.Locator            `json:"locator"`
	SubresourceLocator basespec.SubresourceLocator `json:"subresourceLocator,omitempty"`
	RuntimeDisabled    bool                        `json:"runtimeDisabled"`
	Diagnostics        []diagnostic.Diagnostic     `json:"diagnostics,omitempty"`
}

type WorkspaceSuppressionView struct {
	Workspace  WorkspaceRef           `json:"workspace"`
	Binding    artifact.SourceBinding `json:"binding"`
	Revision   uint64                 `json:"revision"`
	CreatedAt  time.Time              `json:"createdAt"`
	ModifiedAt time.Time              `json:"modifiedAt"`
}

type WorkspaceResourceView struct {
	Artifact         WorkspaceArtifactView   `json:"artifact"`
	DefinitionDigest cryptoutil.Digest       `json:"definitionDigest"`
	SourceID         basespec.SourceID       `json:"sourceID"`
	Locator          basespec.Locator        `json:"locator"`
	CatalogCurrent   bool                    `json:"catalogCurrent"`
	ProjectionValid  bool                    `json:"projectionValid"`
	Diagnostics      []diagnostic.Diagnostic `json:"diagnostics,omitempty"`
}

type WorkspaceOccurrenceView struct {
	SourceID            basespec.SourceID           `json:"sourceID"`
	Locator             basespec.Locator            `json:"locator"`
	SubresourceLocator  basespec.SubresourceLocator `json:"subresourceLocator,omitempty"`
	Kind                basespec.ArtifactKind       `json:"kind,omitempty"`
	LogicalName         basespec.LogicalName        `json:"logicalName,omitempty"`
	LogicalVersion      basespec.LogicalVersion     `json:"logicalVersion,omitempty"`
	DefinitionDigest    *cryptoutil.Digest          `json:"definitionDigest,omitempty"`
	SourceContentDigest *cryptoutil.Digest          `json:"sourceContentDigest,omitempty"`
	State               string                      `json:"state"`
	Recorded            bool                        `json:"recorded"`
	Artifact            *artifact.ArtifactRef       `json:"artifact,omitempty"`
	Diagnostics         []diagnostic.Diagnostic     `json:"diagnostics,omitempty"`
}

type WorkspaceResourceGroupView struct {
	Kind       basespec.ArtifactKind     `json:"kind"`
	Resources  []WorkspaceResourceView   `json:"resources"`
	Unrecorded []WorkspaceOccurrenceView `json:"unrecorded"`
}

type WorkspaceDefinitionView struct {
	Digest         cryptoutil.Digest       `json:"digest"`
	Kind           basespec.ArtifactKind   `json:"kind"`
	SchemaID       basespec.SchemaID       `json:"schemaID"`
	SchemaVersion  string                  `json:"schemaVersion"`
	LogicalName    basespec.LogicalName    `json:"logicalName"`
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                  `json:"displayName,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Labels         map[string]string       `json:"labels,omitempty"`
	Body           json.RawMessage         `json:"body"`
	Dependencies   []definition.Selector   `json:"dependencies,omitempty"`
}

type WorkspaceLoadPlanItemView struct {
	Artifact         WorkspaceArtifactView   `json:"artifact"`
	Definition       WorkspaceDefinitionView `json:"definition"`
	DefinitionDigest cryptoutil.Digest       `json:"definitionDigest"`
	SourceID         basespec.SourceID       `json:"sourceID"`
	SourceKind       basespec.SourceKind     `json:"sourceKind"`
	Locator          basespec.Locator        `json:"locator"`
	CatalogCurrent   bool                    `json:"catalogCurrent"`
}

type WorkspaceLoadPlanView struct {
	Workspace       WorkspaceRef                `json:"workspace"`
	CatalogRevision uint64                      `json:"catalogRevision"`
	Items           []WorkspaceLoadPlanItemView `json:"items"`
	Diagnostics     []diagnostic.Diagnostic     `json:"diagnostics,omitempty"`
}

type ResolveWorkspaceResourceResponseBody struct {
	Resource   WorkspaceResourceView   `json:"resource"`
	Definition WorkspaceDefinitionView `json:"definition"`
}

type WorkspaceCatalogView struct {
	Workspace               WorkspaceView                `json:"workspace"`
	CatalogRevision         uint64                       `json:"catalogRevision"`
	CatalogCurrent          bool                         `json:"catalogCurrent"`
	Diagnostics             []diagnostic.Diagnostic      `json:"diagnostics,omitempty"`
	Resources               []WorkspaceResourceView      `json:"resources"`
	Groups                  []WorkspaceResourceGroupView `json:"groups"`
	Occurrences             []WorkspaceOccurrenceView    `json:"occurrences"`
	ValidOccurrences        []WorkspaceOccurrenceView    `json:"validOccurrences"`
	InvalidOccurrences      []WorkspaceOccurrenceView    `json:"invalidOccurrences"`
	MissingOccurrences      []WorkspaceOccurrenceView    `json:"missingOccurrences"`
	UnrecordedOccurrences   []WorkspaceOccurrenceView    `json:"unrecordedOccurrences"`
	UnresolvedArtifacts     []WorkspaceArtifactView      `json:"unresolvedArtifacts"`
	UnrecordedCount         int                          `json:"unrecordedCount"`
	UnresolvedArtifactCount int                          `json:"unresolvedArtifactCount"`
}

type WorkspaceRefreshResult struct {
	Workspace        WorkspaceRef            `json:"workspace"`
	CatalogRevision  uint64                  `json:"catalogRevision"`
	CreatedArtifacts []artifact.ArtifactRef  `json:"createdArtifacts"`
	UpdatedArtifacts []artifact.ArtifactRef  `json:"updatedArtifacts"`
	Diagnostics      []diagnostic.Diagnostic `json:"diagnostics,omitempty"`
	Candidates       int                     `json:"candidates"`
}

type WorkspaceContextContribution struct {
	Artifact         artifact.ArtifactRef                      `json:"artifact"`
	RecordRevision   uint64                                    `json:"recordRevision"`
	DefinitionDigest cryptoutil.Digest                         `json:"definitionDigest"`
	SourceID         basespec.SourceID                         `json:"sourceID"`
	Locator          basespec.Locator                          `json:"locator"`
	Name             string                                    `json:"name"`
	Role             artifactbuiltin.WorkspaceContextRole      `json:"role"`
	MediaType        artifactbuiltin.WorkspaceContextMediaType `json:"mediaType"`
	Content          string                                    `json:"content"`
	ConventionOrder  int                                       `json:"conventionOrder"`
	OriginalBytes    int                                       `json:"originalBytes"`
	IncludedBytes    int                                       `json:"includedBytes"`
	Truncated        bool                                      `json:"truncated"`
}

type WorkspaceContextDecision struct {
	Artifact      artifact.ArtifactRef             `json:"artifact"`
	Status        contextadapter.CompositionStatus `json:"status"`
	Code          string                           `json:"code,omitempty"`
	OriginalBytes int                              `json:"originalBytes"`
	IncludedBytes int                              `json:"includedBytes"`
}

type WorkspaceContextLoadPlan struct {
	Workspace       WorkspaceRef                   `json:"workspace"`
	CatalogRevision uint64                         `json:"catalogRevision"`
	Contributions   []WorkspaceContextContribution `json:"contributions"`
	Prompt          string                         `json:"prompt"`
	Diagnostics     []diagnostic.Diagnostic        `json:"diagnostics,omitempty"`
	Decisions       []WorkspaceContextDecision     `json:"decisions"`
	PromptBytes     int                            `json:"promptBytes"`
}

type WorkspaceContextView struct {
	Artifact         artifact.ArtifactRef                      `json:"artifact"`
	RecordRevision   uint64                                    `json:"recordRevision"`
	DefinitionDigest cryptoutil.Digest                         `json:"definitionDigest"`
	SourceID         basespec.SourceID                         `json:"sourceID"`
	Locator          basespec.Locator                          `json:"locator"`
	Name             string                                    `json:"name"`
	Role             artifactbuiltin.WorkspaceContextRole      `json:"role"`
	MediaType        artifactbuiltin.WorkspaceContextMediaType `json:"mediaType"`
	Enabled          bool                                      `json:"enabled"`
	State            artifact.State                            `json:"state"`
	CatalogCurrent   bool                                      `json:"catalogCurrent"`
	ProjectionValid  bool                                      `json:"projectionValid"`
	RuntimeDisabled  bool                                      `json:"runtimeDisabled"`
	Diagnostics      []diagnostic.Diagnostic                   `json:"diagnostics,omitempty"`
}

type WorkspaceContextInspectionView struct {
	Workspace       WorkspaceRef                   `json:"workspace"`
	CatalogRevision uint64                         `json:"catalogRevision"`
	Contributions   []WorkspaceContextContribution `json:"contributions"`
	Diagnostics     []diagnostic.Diagnostic        `json:"diagnostics,omitempty"`
}

type WorkspaceSkillArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type WorkspaceSkillSummary struct {
	SchemaVersion string                   `json:"schemaVersion"`
	ID            basespec.ArtifactID      `json:"id"`
	Slug          string                   `json:"slug"`
	Name          string                   `json:"name"`
	DisplayName   string                   `json:"displayName"`
	Description   string                   `json:"description"`
	Tags          []string                 `json:"tags,omitempty"`
	Insert        document.SkillInsert     `json:"insert"`
	Arguments     []WorkspaceSkillArgument `json:"arguments,omitempty"`
	IsEnabled     bool                     `json:"isEnabled"`
	CreatedAt     time.Time                `json:"createdAt"`
	ModifiedAt    time.Time                `json:"modifiedAt"`
}

type WorkspaceSkillView struct {
	Workspace        WorkspaceRef            `json:"workspace"`
	Artifact         artifact.ArtifactRef    `json:"artifact"`
	DefinitionDigest cryptoutil.Digest       `json:"definitionDigest"`
	SourceID         basespec.SourceID       `json:"sourceID"`
	Locator          basespec.Locator        `json:"locator"`
	Skill            WorkspaceSkillSummary   `json:"skill"`
	MarkdownBody     string                  `json:"markdownBody,omitempty"`
	RecordRevision   uint64                  `json:"recordRevision"`
	State            artifact.State          `json:"state"`
	ProjectionValid  bool                    `json:"projectionValid"`
	CatalogCurrent   bool                    `json:"catalogCurrent"`
	RuntimeDisabled  bool                    `json:"runtimeDisabled"`
	Diagnostics      []diagnostic.Diagnostic `json:"diagnostics,omitempty"`
}

type WorkspaceSkillLoadView struct {
	Workspace       WorkspaceRef            `json:"workspace"`
	CatalogRevision uint64                  `json:"catalogRevision"`
	Skills          []WorkspaceSkillView    `json:"skills"`
	Diagnostics     []diagnostic.Diagnostic `json:"diagnostics,omitempty"`
}

type CreateFilesystemWorkspaceRequestBody struct {
	WorkspaceID      basespec.CollectionID `json:"workspaceID"           required:"true"`
	SourceID         basespec.SourceID     `json:"sourceID"              required:"true"`
	SourceStorageKey basespec.StorageKey   `json:"sourceStorageKey"      required:"true"`
	DisplayName      string                `json:"displayName"           required:"true"`
	Description      string                `json:"description,omitempty"`
	RootPath         string                `json:"rootPath"              required:"true"`
	Discovery        WorkspaceDiscovery    `json:"discovery"`
}

type CreateFilesystemWorkspaceRequest struct {
	// RootID is retained for transport compatibility. Workspace creation uses
	// the single Root configured by application composition.
	RootID basespec.RootID `json:"rootID,omitempty"`
	Body   *CreateFilesystemWorkspaceRequestBody
}

type CreateFilesystemWorkspaceResponse struct {
	Body *WorkspaceView
}

type CreateEmptyWorkspaceRequestBody struct {
	WorkspaceID basespec.CollectionID `json:"workspaceID"           required:"true"`
	DisplayName string                `json:"displayName"           required:"true"`
	Description string                `json:"description,omitempty"`
	Discovery   WorkspaceDiscovery    `json:"discovery"`
}

type CreateEmptyWorkspaceRequest struct {
	// RootID is retained for transport compatibility. Workspace creation uses
	// the single Root configured by application composition.
	RootID basespec.RootID `json:"rootID,omitempty"`
	Body   *CreateEmptyWorkspaceRequestBody
}

type CreateEmptyWorkspaceResponse struct {
	Body *WorkspaceView
}

type GetWorkspaceRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
}

type GetWorkspaceResponse struct {
	Body *WorkspaceView
}

type ListWorkspacesRequest struct {
	// RootID is retained for transport compatibility. Listing uses the
	// application-configured Workspace Root.
	RootID basespec.RootID `json:"rootID,omitempty"`
}

type ListWorkspacesResponseBody struct {
	Workspaces []WorkspaceView `json:"workspaces"`
}

type ListWorkspacesResponse struct {
	Body *ListWorkspacesResponseBody
}

type UpdateWorkspaceRequestBody struct {
	ExpectedRevision uint64             `json:"expectedRevision"      required:"true"`
	DisplayName      string             `json:"displayName"           required:"true"`
	Description      string             `json:"description,omitempty"`
	Enabled          bool               `json:"enabled"               required:"true"`
	Discovery        WorkspaceDiscovery `json:"discovery"`
}

type UpdateWorkspaceRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
	Body      *UpdateWorkspaceRequestBody
}

type UpdateWorkspaceResponse struct {
	Body *WorkspaceView
}

type ReplaceWorkspacePrimarySourceRequestBody struct {
	ExpectedCollectionRevision         uint64            `json:"expectedCollectionRevision"         required:"true"`
	PreviousSourceID                   basespec.SourceID `json:"previousSourceID"                   required:"true"`
	ExpectedPreviousAttachmentRevision uint64            `json:"expectedPreviousAttachmentRevision" required:"true"`
	SourceID                           basespec.SourceID `json:"sourceID"                           required:"true"`
}

type ReplaceWorkspacePrimarySourceRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
	Body      *ReplaceWorkspacePrimarySourceRequestBody
}

type ReplaceWorkspacePrimarySourceResponse struct {
	Body *WorkspaceView
}

type SetWorkspacePrimarySourceRequestBody struct {
	ExpectedCollectionRevision         uint64            `json:"expectedCollectionRevision"                   required:"true"`
	PreviousSourceID                   basespec.SourceID `json:"previousSourceID,omitempty"`
	ExpectedPreviousAttachmentRevision uint64            `json:"expectedPreviousAttachmentRevision,omitempty"`
	SourceID                           basespec.SourceID `json:"sourceID,omitempty"`
	Clear                              bool              `json:"clear,omitempty"`
}

type SetWorkspacePrimarySourceRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
	Body      *SetWorkspacePrimarySourceRequestBody
}

type SetWorkspacePrimarySourceResponse struct {
	Body *WorkspaceView
}

type RetireWorkspaceRequest struct {
	Workspace        WorkspaceRef `json:"workspace"        required:"true"`
	ExpectedRevision uint64       `json:"expectedRevision" required:"true"`
}

type RetireWorkspaceResponseBody struct {
	Workspace WorkspaceRef `json:"workspace"`
	Revision  uint64       `json:"revision"`
}

type RetireWorkspaceResponse struct {
	Body *RetireWorkspaceResponseBody
}

type PurgeWorkspaceRequest struct {
	Workspace        WorkspaceRef `json:"workspace"        required:"true"`
	ExpectedRevision uint64       `json:"expectedRevision" required:"true"`
}

type PurgeWorkspaceResponseBody struct {
	Workspace WorkspaceRef `json:"workspace"`
}

type PurgeWorkspaceResponse struct {
	Body *PurgeWorkspaceResponseBody
}

type AttachWorkspaceSourceRequestBody struct {
	ExpectedCollectionRevision uint64                      `json:"expectedCollectionRevision" required:"true"`
	SourceID                   basespec.SourceID           `json:"sourceID"                   required:"true"`
	Role                       basespec.AttachmentRole     `json:"role"                       required:"true"`
	Enabled                    bool                        `json:"enabled"                    required:"true"`
	Settings                   WorkspaceAttachmentSettings `json:"settings"`
}

type AttachWorkspaceSourceRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
	Body      *AttachWorkspaceSourceRequestBody
}

type AttachWorkspaceSourceResponse struct {
	Body *WorkspaceView
}

type UpdateWorkspaceAttachmentRequestBody struct {
	ExpectedCollectionRevision uint64                      `json:"expectedCollectionRevision" required:"true"`
	ExpectedAttachmentRevision uint64                      `json:"expectedAttachmentRevision" required:"true"`
	Role                       basespec.AttachmentRole     `json:"role"                       required:"true"`
	Enabled                    bool                        `json:"enabled"                    required:"true"`
	Settings                   WorkspaceAttachmentSettings `json:"settings"`
}

type UpdateWorkspaceAttachmentRequest struct {
	Workspace WorkspaceRef      `json:"workspace" required:"true"`
	SourceID  basespec.SourceID `json:"sourceID"  required:"true"`
	Body      *UpdateWorkspaceAttachmentRequestBody
}

type UpdateWorkspaceAttachmentResponse struct {
	Body *WorkspaceView
}

type DetachWorkspaceSourceRequest struct {
	Workspace                  WorkspaceRef      `json:"workspace"                  required:"true"`
	SourceID                   basespec.SourceID `json:"sourceID"                   required:"true"`
	ExpectedCollectionRevision uint64            `json:"expectedCollectionRevision" required:"true"`
	ExpectedAttachmentRevision uint64            `json:"expectedAttachmentRevision" required:"true"`
}

type DetachWorkspaceSourceResponse struct {
	Body *WorkspaceView
}

type RefreshWorkspaceRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
}

type RefreshWorkspaceResponse struct {
	Body *WorkspaceRefreshResult
}

type GetWorkspaceCatalogRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
}

type GetWorkspaceCatalogResponse struct {
	Body *WorkspaceCatalogView
}

type ComposeWorkspaceLoadPlanRequestBody struct {
	Artifacts []artifact.ArtifactRef `json:"artifacts"`
}

type ComposeWorkspaceLoadPlanRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
	Body      *ComposeWorkspaceLoadPlanRequestBody
}

type ComposeWorkspaceLoadPlanResponse struct {
	Body *WorkspaceLoadPlanView
}

type ResolveWorkspaceResourceRequestBody struct {
	Artifact *artifact.ArtifactRef `json:"artifact,omitempty"`
	Selector *definition.Selector  `json:"selector,omitempty"`
}

type ResolveWorkspaceResourceRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
	Body      *ResolveWorkspaceResourceRequestBody
}

type ResolveWorkspaceResourceResponse struct {
	Body *ResolveWorkspaceResourceResponseBody
}

type GetWorkspaceArtifactRequest struct {
	Workspace WorkspaceRef         `json:"workspace" required:"true"`
	Artifact  artifact.ArtifactRef `json:"artifact"  required:"true"`
}

type GetWorkspaceArtifactResponse struct {
	Body *WorkspaceArtifactView
}

type ListWorkspaceArtifactsRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
}

type ListWorkspaceArtifactsResponseBody struct {
	Artifacts []WorkspaceArtifactView `json:"artifacts"`
}

type ListWorkspaceArtifactsResponse struct {
	Body *ListWorkspaceArtifactsResponseBody
}

type AdoptWorkspaceOccurrenceRequestBody struct {
	ExpectedCatalogRevision uint64                    `json:"expectedCatalogRevision" required:"true"`
	Occurrence              WorkspaceOccurrenceRef    `json:"occurrence"              required:"true"`
	ArtifactID              basespec.ArtifactID       `json:"artifactID"              required:"true"`
	Name                    string                    `json:"name,omitempty"`
	Enabled                 bool                      `json:"enabled"                 required:"true"`
	Settings                WorkspaceArtifactSettings `json:"settings"`
}

type AdoptWorkspaceOccurrenceRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
	Body      *AdoptWorkspaceOccurrenceRequestBody
}

type AdoptWorkspaceOccurrenceResponse struct {
	Body *WorkspaceArtifactView
}

type PinWorkspaceArtifactRequestBody struct {
	ExpectedCollectionRevision uint64                    `json:"expectedCollectionRevision" required:"true"`
	Binding                    artifact.SourceBinding    `json:"binding"                    required:"true"`
	ArtifactID                 basespec.ArtifactID       `json:"artifactID"                 required:"true"`
	Name                       string                    `json:"name"                       required:"true"`
	Enabled                    bool                      `json:"enabled"                    required:"true"`
	Settings                   WorkspaceArtifactSettings `json:"settings"`
}

type PinWorkspaceArtifactRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
	Body      *PinWorkspaceArtifactRequestBody
}

type PinWorkspaceArtifactResponse struct {
	Body *WorkspaceArtifactView
}

type ListWorkspaceSuppressionsRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
}

type ListWorkspaceSuppressionsResponseBody struct {
	Suppressions []WorkspaceSuppressionView `json:"suppressions"`
}

type ListWorkspaceSuppressionsResponse struct {
	Body *ListWorkspaceSuppressionsResponseBody
}

type SuppressWorkspaceBindingRequestBody struct {
	ExpectedCollectionRevision uint64                 `json:"expectedCollectionRevision" required:"true"`
	Binding                    artifact.SourceBinding `json:"binding"                    required:"true"`
}

type SuppressWorkspaceBindingRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
	Body      *SuppressWorkspaceBindingRequestBody
}

type SuppressWorkspaceBindingResponse struct {
	Body *WorkspaceSuppressionView
}

type ListWorkspaceContextsRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
}

type ListWorkspaceContextsResponseBody struct {
	Contexts []WorkspaceContextView `json:"contexts"`
}

type ListWorkspaceContextsResponse struct {
	Body *ListWorkspaceContextsResponseBody
}

type LoadWorkspaceContextsRequestBody struct {
	Artifacts []artifact.ArtifactRef `json:"artifacts,omitempty"`
}

type LoadWorkspaceContextsRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
	Body      *LoadWorkspaceContextsRequestBody
}

type LoadWorkspaceContextsResponse struct {
	Body *WorkspaceContextInspectionView
}

type ComposeWorkspaceContextRequestBody struct {
	Artifacts []artifact.ArtifactRef `json:"artifacts,omitempty"`
}

type ComposeWorkspaceContextRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
	Body      *ComposeWorkspaceContextRequestBody
}

type ComposeWorkspaceContextResponse struct {
	Body *WorkspaceContextLoadPlan
}

type ListWorkspaceSkillsRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
}

type ListWorkspaceSkillsResponseBody struct {
	Skills []WorkspaceSkillView `json:"skills"`
}

type ListWorkspaceSkillsResponse struct {
	Body *ListWorkspaceSkillsResponseBody
}

type LoadWorkspaceSkillsRequestBody struct {
	Artifacts []artifact.ArtifactRef `json:"artifacts"`
}

type LoadWorkspaceSkillsRequest struct {
	Workspace WorkspaceRef `json:"workspace" required:"true"`
	Body      *LoadWorkspaceSkillsRequestBody
}

type LoadWorkspaceSkillsResponse struct {
	Body *WorkspaceSkillLoadView
}

type SetWorkspaceArtifactEnabledRequestBody struct {
	ExpectedRevision uint64 `json:"expectedRevision" required:"true"`
	Enabled          bool   `json:"enabled"          required:"true"`
}

type SetWorkspaceArtifactEnabledRequest struct {
	Workspace WorkspaceRef         `json:"workspace" required:"true"`
	Artifact  artifact.ArtifactRef `json:"artifact"  required:"true"`
	Body      *SetWorkspaceArtifactEnabledRequestBody
}

type SetWorkspaceArtifactEnabledResponse struct {
	Body *WorkspaceArtifactView
}

type UnadoptWorkspaceArtifactRequest struct {
	Workspace        WorkspaceRef         `json:"workspace"        required:"true"`
	Artifact         artifact.ArtifactRef `json:"artifact"         required:"true"`
	ExpectedRevision uint64               `json:"expectedRevision" required:"true"`
	Suppress         bool                 `json:"suppress"`
}

type UnadoptWorkspaceArtifactResponseBody struct {
	Artifact artifact.ArtifactRef `json:"artifact"`
}

type UnadoptWorkspaceArtifactResponse struct {
	Body *UnadoptWorkspaceArtifactResponseBody
}

type PurgeWorkspaceArtifactRequest struct {
	Workspace        WorkspaceRef         `json:"workspace"        required:"true"`
	Artifact         artifact.ArtifactRef `json:"artifact"         required:"true"`
	ExpectedRevision uint64               `json:"expectedRevision" required:"true"`
}

type PurgeWorkspaceArtifactResponseBody struct {
	Artifact artifact.ArtifactRef `json:"artifact"`
}

type PurgeWorkspaceArtifactResponse struct {
	Body *PurgeWorkspaceArtifactResponseBody
}

type UnsuppressWorkspaceBindingRequest struct {
	Workspace        WorkspaceRef           `json:"workspace"        required:"true"`
	Binding          artifact.SourceBinding `json:"binding"          required:"true"`
	ExpectedRevision uint64                 `json:"expectedRevision" required:"true"`
}

type UnsuppressWorkspaceBindingResponseBody struct {
	Workspace WorkspaceRef           `json:"workspace"`
	Binding   artifact.SourceBinding `json:"binding"`
}

type UnsuppressWorkspaceBindingResponse struct {
	Body *UnsuppressWorkspaceBindingResponseBody
}

type SetWorkspaceArtifactRuntimeDisabledRequestBody struct {
	ExpectedRevision uint64 `json:"expectedRevision" required:"true"`
	RuntimeDisabled  bool   `json:"runtimeDisabled"  required:"true"`
}

type SetWorkspaceArtifactRuntimeDisabledRequest struct {
	Workspace WorkspaceRef         `json:"workspace" required:"true"`
	Artifact  artifact.ArtifactRef `json:"artifact"  required:"true"`
	Body      *SetWorkspaceArtifactRuntimeDisabledRequestBody
}

type SetWorkspaceArtifactRuntimeDisabledResponse struct {
	Body *WorkspaceArtifactView
}
