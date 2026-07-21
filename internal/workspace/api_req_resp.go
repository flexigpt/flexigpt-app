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
// data, source filesystem paths, and the trust reference itself.
type WorkspaceView struct {
	RootID            artifactstore.RootID      `json:"rootID"`
	Revision          uint64                    `json:"revision"`
	DisplayName       string                    `json:"displayName"`
	Description       string                    `json:"description,omitempty"`
	Enabled           bool                      `json:"enabled"`
	Mode              string                    `json:"mode"`
	PrimarySourceID   artifactstore.SourceID    `json:"primarySourceID,omitempty"`
	HasTrustReference bool                      `json:"hasTrustReference"`
	Discovery         WorkspaceDiscovery        `json:"discovery"`
	Attachments       []WorkspaceAttachmentView `json:"attachments"`
}

type WorkspaceAttachmentView struct {
	SourceID artifactstore.SourceID       `json:"sourceID"`
	Revision uint64                       `json:"revision"`
	Role     artifactstore.AttachmentRole `json:"role"`
	Priority int                          `json:"priority"`
	Enabled  bool                         `json:"enabled"`
	Settings WorkspaceAttachmentSettings  `json:"settings"`
}

type WorkspaceRecordView struct {
	ID                 artifactstore.RecordID     `json:"id"`
	Revision           uint64                     `json:"revision"`
	Name               string                     `json:"name"`
	Kind               artifactstore.ArtifactKind `json:"kind"`
	Enabled            bool                       `json:"enabled"`
	State              string                     `json:"state"`
	ResolvedDefinition *artifactstore.Digest      `json:"resolvedDefinition,omitempty"`
}

type WorkspaceResourceView struct {
	Record           WorkspaceRecordView    `json:"record"`
	DefinitionDigest artifactstore.Digest   `json:"definitionDigest"`
	SourceID         artifactstore.SourceID `json:"sourceID"`
	Locator          artifactstore.Locator  `json:"locator"`
	CatalogCurrent   bool                   `json:"catalogCurrent"`
}

type WorkspaceCatalogView struct {
	Workspace             WorkspaceView           `json:"workspace"`
	CatalogRevision       uint64                  `json:"catalogRevision"`
	Resources             []WorkspaceResourceView `json:"resources"`
	UnrecordedCount       int                     `json:"unrecordedCount"`
	UnresolvedRecordCount int                     `json:"unresolvedRecordCount"`
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
	Priority         int                    `json:"priority"`
	Name             string                 `json:"name"`
	Role             string                 `json:"role"`
	MediaType        string                 `json:"mediaType"`
	Content          string                 `json:"content"`
}

type WorkspaceContextLoadPlan struct {
	RootID          artifactstore.RootID           `json:"rootID"`
	CatalogRevision uint64                         `json:"catalogRevision"`
	Contributions   []WorkspaceContextContribution `json:"contributions"`
	Prompt          string                         `json:"prompt"`
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
	Type          string                   `json:"type"`
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
	RootID           artifactstore.RootID   `json:"rootID"`
	RecordID         artifactstore.RecordID `json:"recordID"`
	DefinitionDigest artifactstore.Digest   `json:"definitionDigest"`
	SourceID         artifactstore.SourceID `json:"sourceID"`
	Locator          artifactstore.Locator  `json:"locator"`
	Skill            WorkspaceSkillSummary  `json:"skill"`
	MarkdownBody     string                 `json:"markdownBody,omitempty"`
}

type WorkspaceSkillLoadView struct {
	RootID          artifactstore.RootID       `json:"rootID"`
	CatalogRevision uint64                     `json:"catalogRevision"`
	Skills          []WorkspaceSkillView       `json:"skills"`
	Diagnostics     []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
}

type CreateFilesystemWorkspaceRequestBody struct {
	DisplayName    string             `json:"displayName"              required:"true"`
	Description    string             `json:"description,omitempty"`
	RootPath       string             `json:"rootPath"                 required:"true"`
	TrustReference string             `json:"trustReference,omitempty"`
	Discovery      WorkspaceDiscovery `json:"discovery"`
}

type CreateFilesystemWorkspaceRequest struct {
	Body *CreateFilesystemWorkspaceRequestBody
}

type CreateFilesystemWorkspaceResponse struct {
	Body *WorkspaceView
}

type CreateEmptyWorkspaceRequestBody struct {
	DisplayName    string             `json:"displayName"              required:"true"`
	Description    string             `json:"description,omitempty"`
	TrustReference string             `json:"trustReference,omitempty"`
	Discovery      WorkspaceDiscovery `json:"discovery"`
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
	ExpectedRevision uint64             `json:"expectedRevision"         required:"true"`
	DisplayName      string             `json:"displayName"              required:"true"`
	Description      string             `json:"description,omitempty"`
	Enabled          bool               `json:"enabled"                  required:"true"`
	TrustReference   string             `json:"trustReference,omitempty"`
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
	Priority             int                          `json:"priority"`
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
	Priority                   int                          `json:"priority"`
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
