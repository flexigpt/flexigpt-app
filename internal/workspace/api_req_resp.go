package workspace

import (
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type WorkspaceDiscovery struct {
	AdditionalLocators []artifactstore.Locator  `json:"additionalLocators,omitempty"`
	AdditionalRoots    []WorkspaceDiscoveryRoot `json:"additionalRoots,omitempty"`
	IncludeReadme      bool                     `json:"includeReadme,omitempty"`
}

type WorkspaceDiscoveryRoot struct {
	Root            artifactstore.Locator `json:"root"`
	Recursive       bool                  `json:"recursive"`
	IncludePatterns []string              `json:"includePatterns,omitempty"`
}

type WorkspaceAttachmentSettings struct {
	Recursive     *bool `json:"recursive,omitempty"`
	Authoritative *bool `json:"authoritative,omitempty"`
}

// WorkspaceView is the API-safe representation of a workspace.
//
// It deliberately excludes source configuration, root data, attachment raw
// data, and the trust-reference contents. Local filesystem paths are included
// because the local Workspace management UI intentionally displays them.
type WorkspaceView struct {
	RootID          artifactstore.RootID      `json:"rootID"`
	Revision        uint64                    `json:"revision"`
	DisplayName     string                    `json:"displayName"`
	Description     string                    `json:"description,omitempty"`
	Enabled         bool                      `json:"enabled"`
	Mode            string                    `json:"mode"`
	PrimarySourceID artifactstore.SourceID    `json:"primarySourceID,omitempty"`
	PrimaryPath     string                    `json:"primaryPath,omitempty"`
	Discovery       WorkspaceDiscovery        `json:"discovery"`
	Attachments     []WorkspaceAttachmentView `json:"attachments"`
}

type WorkspaceAttachmentView struct {
	SourceID          artifactstore.SourceID       `json:"sourceID"`
	Revision          uint64                       `json:"revision"`
	Role              artifactstore.AttachmentRole `json:"role"`
	Enabled           bool                         `json:"enabled"`
	SourceDisplayName string                       `json:"sourceDisplayName,omitempty"`
	SourceKind        string                       `json:"sourceKind,omitempty"`
	Path              string                       `json:"path,omitempty"`
	Settings          WorkspaceAttachmentSettings  `json:"settings"`
}

type WorkspaceRecordView struct {
	ID                 artifactstore.RecordID           `json:"id"`
	Revision           uint64                           `json:"revision"`
	Name               string                           `json:"name"`
	Kind               artifactstore.ArtifactKind       `json:"kind"`
	Enabled            bool                             `json:"enabled"`
	State              string                           `json:"state"`
	Mode               string                           `json:"mode"`
	PinnedDefinition   *artifactstore.Digest            `json:"pinnedDefinition,omitempty"`
	ResolvedDefinition *artifactstore.Digest            `json:"resolvedDefinition,omitempty"`
	SourceID           artifactstore.SourceID           `json:"sourceID"`
	Locator            artifactstore.Locator            `json:"locator"`
	SubresourceLocator artifactstore.SubresourceLocator `json:"subresourceLocator,omitempty"`
	RuntimeDisabled    bool                             `json:"runtimeDisabled"`
	Diagnostics        []artifactstore.Diagnostic       `json:"diagnostics,omitempty"`
}

type WorkspaceResourceView struct {
	Record           WorkspaceRecordView        `json:"record"`
	DefinitionDigest artifactstore.Digest       `json:"definitionDigest"`
	SourceID         artifactstore.SourceID     `json:"sourceID"`
	Locator          artifactstore.Locator      `json:"locator"`
	CatalogCurrent   bool                       `json:"catalogCurrent"`
	ProjectionValid  bool                       `json:"projectionValid"`
	Diagnostics      []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
}

type WorkspaceOccurrenceView struct {
	SourceID            artifactstore.SourceID           `json:"sourceID"`
	Locator             artifactstore.Locator            `json:"locator"`
	SubresourceLocator  artifactstore.SubresourceLocator `json:"subresourceLocator,omitempty"`
	Kind                artifactstore.ArtifactKind       `json:"kind,omitempty"`
	LogicalName         artifactstore.LogicalName        `json:"logicalName,omitempty"`
	LogicalVersion      artifactstore.LogicalVersion     `json:"logicalVersion,omitempty"`
	DefinitionDigest    *artifactstore.Digest            `json:"definitionDigest,omitempty"`
	SourceContentDigest *artifactstore.Digest            `json:"sourceContentDigest,omitempty"`
	State               string                           `json:"state"`
	Recorded            bool                             `json:"recorded"`
	RecordID            *artifactstore.RecordID          `json:"recordID,omitempty"`
	Diagnostics         []artifactstore.Diagnostic       `json:"diagnostics,omitempty"`
}

type WorkspaceResourceGroupView struct {
	Kind       artifactstore.ArtifactKind `json:"kind"`
	Resources  []WorkspaceResourceView    `json:"resources"`
	Unrecorded []WorkspaceOccurrenceView  `json:"unrecorded"`
}

type WorkspaceCatalogView struct {
	Workspace             WorkspaceView                `json:"workspace"`
	CatalogRevision       uint64                       `json:"catalogRevision"`
	CatalogCurrent        bool                         `json:"catalogCurrent"`
	Diagnostics           []artifactstore.Diagnostic   `json:"diagnostics,omitempty"`
	Resources             []WorkspaceResourceView      `json:"resources"`
	Groups                []WorkspaceResourceGroupView `json:"groups"`
	Occurrences           []WorkspaceOccurrenceView    `json:"occurrences"`
	ValidOccurrences      []WorkspaceOccurrenceView    `json:"validOccurrences"`
	InvalidOccurrences    []WorkspaceOccurrenceView    `json:"invalidOccurrences"`
	MissingOccurrences    []WorkspaceOccurrenceView    `json:"missingOccurrences"`
	UnrecordedOccurrences []WorkspaceOccurrenceView    `json:"unrecordedOccurrences"`
	UnresolvedRecords     []WorkspaceRecordView        `json:"unresolvedRecords"`
	UnrecordedCount       int                          `json:"unrecordedCount"`
	UnresolvedRecordCount int                          `json:"unresolvedRecordCount"`
}

type WorkspaceRefreshResult struct {
	RootID          artifactstore.RootID       `json:"rootID"`
	CatalogRevision uint64                     `json:"catalogRevision"`
	CreatedRecords  []artifactstore.RecordID   `json:"createdRecords"`
	UpdatedRecords  []artifactstore.RecordID   `json:"updatedRecords"`
	Diagnostics     []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
	Candidates      int                        `json:"candidates"`
}

type WorkspaceContextContribution struct {
	RecordID         artifactstore.RecordID `json:"recordID"`
	DefinitionDigest artifactstore.Digest   `json:"definitionDigest"`
	SourceID         artifactstore.SourceID `json:"sourceID"`
	Locator          artifactstore.Locator  `json:"locator"`
	Name             string                 `json:"name"`
	Role             string                 `json:"role"`
	MediaType        string                 `json:"mediaType"`
	Content          string                 `json:"content"`
	ConventionOrder  int                    `json:"conventionOrder"`
	OriginalBytes    int                    `json:"originalBytes"`
	IncludedBytes    int                    `json:"includedBytes"`
	Truncated        bool                   `json:"truncated"`
}

type WorkspaceContextDecision struct {
	RecordID      artifactstore.RecordID `json:"recordID"`
	Status        string                 `json:"status"`
	Code          string                 `json:"code,omitempty"`
	OriginalBytes int                    `json:"originalBytes"`
	IncludedBytes int                    `json:"includedBytes"`
}

type WorkspaceContextLoadPlan struct {
	RootID          artifactstore.RootID           `json:"rootID"`
	CatalogRevision uint64                         `json:"catalogRevision"`
	Contributions   []WorkspaceContextContribution `json:"contributions"`
	Prompt          string                         `json:"prompt"`
	Diagnostics     []artifactstore.Diagnostic     `json:"diagnostics,omitempty"`
	Decisions       []WorkspaceContextDecision     `json:"decisions"`
	PromptBytes     int                            `json:"promptBytes"`
}

type WorkspaceContextView struct {
	RecordID         artifactstore.RecordID     `json:"recordID"`
	RecordRevision   uint64                     `json:"recordRevision"`
	DefinitionDigest artifactstore.Digest       `json:"definitionDigest"`
	SourceID         artifactstore.SourceID     `json:"sourceID"`
	Locator          artifactstore.Locator      `json:"locator"`
	Name             string                     `json:"name"`
	Role             string                     `json:"role"`
	MediaType        string                     `json:"mediaType"`
	Enabled          bool                       `json:"enabled"`
	State            string                     `json:"state"`
	CatalogCurrent   bool                       `json:"catalogCurrent"`
	RuntimeDisabled  bool                       `json:"runtimeDisabled"`
	Diagnostics      []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
}

type WorkspaceContextInspectionView struct {
	RootID          artifactstore.RootID           `json:"rootID"`
	CatalogRevision uint64                         `json:"catalogRevision"`
	Contributions   []WorkspaceContextContribution `json:"contributions"`
	Diagnostics     []artifactstore.Diagnostic     `json:"diagnostics,omitempty"`
}

type WorkspaceSkillArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type WorkspaceSkillSummary struct {
	SchemaVersion string                   `json:"schemaVersion"`
	ID            artifactstore.RecordID   `json:"id"`
	Slug          string                   `json:"slug"`
	Name          string                   `json:"name"`
	DisplayName   string                   `json:"displayName"`
	Description   string                   `json:"description"`
	Tags          []string                 `json:"tags,omitempty"`
	Insert        string                   `json:"insert"`
	Arguments     []WorkspaceSkillArgument `json:"arguments,omitempty"`
	IsEnabled     bool                     `json:"isEnabled"`
	CreatedAt     time.Time                `json:"createdAt"`
	ModifiedAt    time.Time                `json:"modifiedAt"`
}

type WorkspaceSkillView struct {
	RootID           artifactstore.RootID       `json:"rootID"`
	RecordID         artifactstore.RecordID     `json:"recordID"`
	DefinitionDigest artifactstore.Digest       `json:"definitionDigest"`
	SourceID         artifactstore.SourceID     `json:"sourceID"`
	Locator          artifactstore.Locator      `json:"locator"`
	Skill            WorkspaceSkillSummary      `json:"skill"`
	MarkdownBody     string                     `json:"markdownBody,omitempty"`
	RecordRevision   uint64                     `json:"recordRevision"`
	State            string                     `json:"state"`
	CatalogCurrent   bool                       `json:"catalogCurrent"`
	RuntimeDisabled  bool                       `json:"runtimeDisabled"`
	Diagnostics      []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
}

type WorkspaceSkillLoadView struct {
	RootID          artifactstore.RootID       `json:"rootID"`
	CatalogRevision uint64                     `json:"catalogRevision"`
	Skills          []WorkspaceSkillView       `json:"skills"`
	Diagnostics     []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
}

type CreateFilesystemWorkspaceRequestBody struct {
	DisplayName string             `json:"displayName"           required:"true"`
	Description string             `json:"description,omitempty"`
	RootPath    string             `json:"rootPath"              required:"true"`
	Discovery   WorkspaceDiscovery `json:"discovery"`
}

type CreateFilesystemWorkspaceRequest struct {
	Body *CreateFilesystemWorkspaceRequestBody
}

type CreateFilesystemWorkspaceResponse struct {
	Body *WorkspaceView
}

type CreateEmptyWorkspaceRequestBody struct {
	DisplayName string             `json:"displayName"           required:"true"`
	Description string             `json:"description,omitempty"`
	Discovery   WorkspaceDiscovery `json:"discovery"`
}

type CreateEmptyWorkspaceRequest struct {
	Body *CreateEmptyWorkspaceRequestBody
}

type CreateEmptyWorkspaceResponse struct {
	Body *WorkspaceView
}

type GetWorkspaceRequest struct {
	RootID artifactstore.RootID `path:"rootID" required:"true"`
}

type GetWorkspaceResponse struct {
	Body *WorkspaceView
}

type ListWorkspacesRequest struct{}

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
	RootID artifactstore.RootID `path:"rootID" required:"true"`
	Body   *UpdateWorkspaceRequestBody
}

type UpdateWorkspaceResponse struct {
	Body *WorkspaceView
}

type DeleteWorkspaceRequest struct {
	RootID           artifactstore.RootID `path:"rootID" required:"true"`
	ExpectedRevision uint64               `              required:"true" query:"expectedRevision"`
}

type DeleteWorkspaceResponseBody struct {
	RootID   artifactstore.RootID `json:"rootID"`
	Revision uint64               `json:"revision"`
}

type DeleteWorkspaceResponse struct {
	Body *DeleteWorkspaceResponseBody
}

type AttachWorkspaceSourceRequestBody struct {
	ExpectedRootRevision uint64                       `json:"expectedRootRevision" required:"true"`
	SourceID             artifactstore.SourceID       `json:"sourceID"             required:"true"`
	Role                 artifactstore.AttachmentRole `json:"role"                 required:"true"`
	Enabled              bool                         `json:"enabled"              required:"true"`
	Settings             WorkspaceAttachmentSettings  `json:"settings"`
}

type AttachWorkspaceSourceRequest struct {
	RootID artifactstore.RootID `path:"rootID" required:"true"`
	Body   *AttachWorkspaceSourceRequestBody
}

type AttachWorkspaceSourceResponse struct {
	Body *WorkspaceView
}

type UpdateWorkspaceAttachmentRequestBody struct {
	ExpectedRootRevision       uint64                       `json:"expectedRootRevision"       required:"true"`
	ExpectedAttachmentRevision uint64                       `json:"expectedAttachmentRevision" required:"true"`
	Role                       artifactstore.AttachmentRole `json:"role"                       required:"true"`
	Enabled                    bool                         `json:"enabled"                    required:"true"`
	Settings                   WorkspaceAttachmentSettings  `json:"settings"`
}

type UpdateWorkspaceAttachmentRequest struct {
	RootID   artifactstore.RootID   `path:"rootID"   required:"true"`
	SourceID artifactstore.SourceID `path:"sourceID" required:"true"`
	Body     *UpdateWorkspaceAttachmentRequestBody
}

type UpdateWorkspaceAttachmentResponse struct {
	Body *WorkspaceView
}

type DetachWorkspaceSourceRequest struct {
	RootID                     artifactstore.RootID   `path:"rootID"   required:"true"`
	SourceID                   artifactstore.SourceID `path:"sourceID" required:"true"`
	ExpectedRootRevision       uint64                 `                required:"true" query:"expectedRootRevision"`
	ExpectedAttachmentRevision uint64                 `                required:"true" query:"expectedAttachmentRevision"`
}

type DetachWorkspaceSourceResponse struct {
	Body *WorkspaceView
}

type RefreshWorkspaceRequest struct {
	RootID artifactstore.RootID `path:"rootID" required:"true"`
}

type RefreshWorkspaceResponse struct {
	Body *WorkspaceRefreshResult
}

type GetWorkspaceCatalogRequest struct {
	RootID artifactstore.RootID `path:"rootID" required:"true"`
}

type GetWorkspaceCatalogResponse struct {
	Body *WorkspaceCatalogView
}

type GetWorkspaceRecordRequest struct {
	RootID   artifactstore.RootID   `path:"rootID"   required:"true"`
	RecordID artifactstore.RecordID `path:"recordID" required:"true"`
}

type GetWorkspaceRecordResponse struct {
	Body *WorkspaceRecordView
}

type ListWorkspaceContextsRequest struct {
	RootID artifactstore.RootID `path:"rootID" required:"true"`
}

type ListWorkspaceContextsResponseBody struct {
	Contexts []WorkspaceContextView `json:"contexts"`
}

type ListWorkspaceContextsResponse struct {
	Body *ListWorkspaceContextsResponseBody
}

type LoadWorkspaceContextsRequestBody struct {
	RecordIDs []artifactstore.RecordID `json:"recordIDs,omitempty"`
}

type LoadWorkspaceContextsRequest struct {
	RootID artifactstore.RootID `path:"rootID" required:"true"`
	Body   *LoadWorkspaceContextsRequestBody
}

type LoadWorkspaceContextsResponse struct {
	Body *WorkspaceContextInspectionView
}

type ComposeWorkspaceContextRequestBody struct {
	RecordIDs []artifactstore.RecordID `json:"recordIDs,omitempty"`
}

type ComposeWorkspaceContextRequest struct {
	RootID artifactstore.RootID `path:"rootID" required:"true"`
	Body   *ComposeWorkspaceContextRequestBody
}

type ComposeWorkspaceContextResponse struct {
	Body *WorkspaceContextLoadPlan
}

type ListWorkspaceSkillsRequest struct {
	RootID artifactstore.RootID `path:"rootID" required:"true"`
}

type ListWorkspaceSkillsResponseBody struct {
	Skills []WorkspaceSkillView `json:"skills"`
}

type ListWorkspaceSkillsResponse struct {
	Body *ListWorkspaceSkillsResponseBody
}

type LoadWorkspaceSkillsRequestBody struct {
	RecordIDs []artifactstore.RecordID `json:"recordIDs"`
}

type LoadWorkspaceSkillsRequest struct {
	RootID artifactstore.RootID `path:"rootID" required:"true"`
	Body   *LoadWorkspaceSkillsRequestBody
}

type LoadWorkspaceSkillsResponse struct {
	Body *WorkspaceSkillLoadView
}

type SetWorkspaceRecordEnabledRequestBody struct {
	ExpectedRevision uint64 `json:"expectedRevision" required:"true"`
	Enabled          bool   `json:"enabled"          required:"true"`
}

type SetWorkspaceRecordEnabledRequest struct {
	RootID   artifactstore.RootID   `path:"rootID"   required:"true"`
	RecordID artifactstore.RecordID `path:"recordID" required:"true"`
	Body     *SetWorkspaceRecordEnabledRequestBody
}

type SetWorkspaceRecordEnabledResponse struct {
	Body *WorkspaceRecordView
}

type PinWorkspaceRecordRequestBody struct {
	ExpectedRevision uint64               `json:"expectedRevision" required:"true"`
	DefinitionDigest artifactstore.Digest `json:"definitionDigest" required:"true"`
}

type PinWorkspaceRecordRequest struct {
	RootID   artifactstore.RootID   `path:"rootID"   required:"true"`
	RecordID artifactstore.RecordID `path:"recordID" required:"true"`
	Body     *PinWorkspaceRecordRequestBody
}

type PinWorkspaceRecordResponse struct {
	Body *WorkspaceRecordView
}

type FollowWorkspaceRecordRequestBody struct {
	ExpectedRevision uint64 `json:"expectedRevision" required:"true"`
}

type FollowWorkspaceRecordRequest struct {
	RootID   artifactstore.RootID   `path:"rootID"   required:"true"`
	RecordID artifactstore.RecordID `path:"recordID" required:"true"`
	Body     *FollowWorkspaceRecordRequestBody
}

type FollowWorkspaceRecordResponse struct {
	Body *WorkspaceRecordView
}

type DeleteWorkspaceRecordRequest struct {
	RootID           artifactstore.RootID   `path:"rootID"   required:"true"`
	RecordID         artifactstore.RecordID `path:"recordID" required:"true"`
	ExpectedRevision uint64                 `                required:"true" query:"expectedRevision"`
}

type DeleteWorkspaceRecordResponseBody struct {
	RecordID artifactstore.RecordID `json:"recordID"`
}

type DeleteWorkspaceRecordResponse struct {
	Body *DeleteWorkspaceRecordResponseBody
}

type SetWorkspaceRecordRuntimeDisabledRequestBody struct {
	ExpectedRevision uint64 `json:"expectedRevision" required:"true"`
	RuntimeDisabled  bool   `json:"runtimeDisabled"  required:"true"`
}

type SetWorkspaceRecordRuntimeDisabledRequest struct {
	RootID   artifactstore.RootID   `path:"rootID"   required:"true"`
	RecordID artifactstore.RecordID `path:"recordID" required:"true"`
	Body     *SetWorkspaceRecordRuntimeDisabledRequestBody
}

type SetWorkspaceRecordRuntimeDisabledResponse struct {
	Body *WorkspaceRecordView
}
