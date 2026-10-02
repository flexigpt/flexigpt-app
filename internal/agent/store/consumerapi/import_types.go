package consumerapi

import (
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	root "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type AgentImportIssueSeverity string

const (
	AgentImportIssueError        AgentImportIssueSeverity = "error"
	AgentImportIssueConfirmation AgentImportIssueSeverity = "confirmation"
	AgentImportIssueWarning      AgentImportIssueSeverity = "warning"
	AgentImportIssueInformation  AgentImportIssueSeverity = "information"
)

type AgentImportIssue struct {
	Code     string                   `json:"code"`
	Severity AgentImportIssueSeverity `json:"severity"`
	Path     string                   `json:"path,omitempty"`
	Message  string                   `json:"message"`
}

type AgentImportDestination struct {
	RootID          root.RootID          `json:"rootID"`
	RootDisplayName string               `json:"rootDisplayName,omitempty"`
	SourceID        source.SourceID      `json:"sourceID"`
	Collection      artifact.ArtifactRef `json:"collection"`

	CollectionRevision    uint64           `json:"collectionRevision"`
	CollectionName        spec.LogicalName `json:"collectionName"`
	CollectionDisplayName string           `json:"collectionDisplayName"`
	Baseline              bool             `json:"baseline"`
	Enabled               bool             `json:"enabled"`
}

type AgentImportPreviewRequest struct {
	// Path is a transient selected .json, .yaml, or .yml input file path.
	// Its extension selects the backend parser and is never persisted.
	Path string `json:"path"`

	Collection                 artifact.ArtifactRef `json:"collection"`
	ExpectedCollectionRevision uint64               `json:"expectedCollectionRevision"`

	ExpectedSourceDigest cryptoutil.Digest `json:"expectedSourceDigest,omitempty"`
}

type AgentImportArtifactPreview struct {
	OccurrencePath   string              `json:"occurrencePath"`
	Type             declaration.Type    `json:"type"`
	Name             spec.LogicalName    `json:"name"`
	LogicalVersion   spec.LogicalVersion `json:"logicalVersion,omitempty"`
	DefinitionDigest cryptoutil.Digest   `json:"definitionDigest"`
}

type AgentImportRelationship struct {
	Path   string                   `json:"path"`
	Type   declaration.Type         `json:"type"`
	Name   spec.LogicalName         `json:"name"`
	Scope  declaration.LookupScope  `json:"scope,omitempty"`
	Status resolve.ResolutionStatus `json:"status"`

	Artifact *artifact.ArtifactRef `json:"artifact,omitempty"`
	Mapped   *resolve.MappedTarget `json:"mapped,omitempty"`

	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type AgentImportConflict struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type AgentRestoredMembership struct {
	Collection artifact.ArtifactRef `json:"collection"`
	Path       string               `json:"path"`
	Message    string               `json:"message"`
}

type AgentMCPSetupInput struct {
	Name                 string `json:"name"`
	Kind                 string `json:"kind"`
	Label                string `json:"label,omitempty"`
	Description          string `json:"description,omitempty"`
	Required             bool   `json:"required"`
	ClientSecretRequired bool   `json:"clientSecretRequired"`
}

type AgentMCPSetupDescriptor struct {
	OccurrencePath string           `json:"occurrencePath"`
	Name           spec.LogicalName `json:"name"`

	Artifact *artifact.ArtifactRef `json:"artifact,omitempty"`

	Transport string `json:"transport,omitempty"`
	Command   string `json:"command,omitempty"`
	URL       string `json:"url,omitempty"`
	AuthMode  string `json:"authMode,omitempty"`

	Inputs []AgentMCPSetupInput `json:"inputs,omitempty"`
}

type AgentImportPreview struct {
	Prepared            string            `json:"prepared,omitempty"`
	PreparedFingerprint cryptoutil.Digest `json:"preparedFingerprint,omitempty"`
	ExpiresAt           time.Time         `json:"expiresAt,omitzero"`

	SourceDigest     cryptoutil.Digest `json:"sourceDigest,omitempty"`
	DefinitionDigest cryptoutil.Digest `json:"definitionDigest,omitempty"`

	// NormalizedYAML is returned for JSON and YAML input alike. Managed Agent
	// export intentionally remains YAML-only.
	NormalizedYAML string `json:"normalizedYAML,omitempty"`

	Agent               *AgentImportArtifactPreview  `json:"agent,omitempty"`
	Destination         AgentImportDestination       `json:"destination"`
	ProjectedArtifacts  []AgentImportArtifactPreview `json:"projectedArtifacts,omitempty"`
	Relationships       []AgentImportRelationship    `json:"relationships,omitempty"`
	Conflicts           []AgentImportConflict        `json:"conflicts,omitempty"`
	RestoredMemberships []AgentRestoredMembership    `json:"restoredMemberships,omitempty"`
	MCPSetupDescriptors []AgentMCPSetupDescriptor    `json:"mcpSetupDescriptors,omitempty"`

	CanImport                 bool     `json:"canImport"`
	RequiresConfirmation      bool     `json:"requiresConfirmation"`
	RequiredConfirmationCodes []string `json:"requiredConfirmationCodes,omitempty"`

	Issues []AgentImportIssue `json:"issues,omitempty"`
}

type AgentImportCommitRequest struct {
	Prepared            string            `json:"prepared"`
	PreparedFingerprint cryptoutil.Digest `json:"preparedFingerprint"`

	AcceptedConfirmationCodes []string `json:"acceptedConfirmationCodes,omitempty"`
}

type AgentImportCommitResult struct {
	Agent               AgentView                 `json:"agent"`
	Collection          collection.CollectionView `json:"collection"`
	RestoredMemberships []AgentRestoredMembership `json:"restoredMemberships,omitempty"`
	MCPSetupDescriptors []AgentMCPSetupDescriptor `json:"mcpSetupDescriptors,omitempty"`
	PreparedFingerprint cryptoutil.Digest         `json:"preparedFingerprint"`
}

type AgentExportRequest struct {
	Agent artifact.ArtifactRef `json:"agent"`
}

type AgentExportResult struct {
	Type              declaration.Type  `json:"type"`
	Name              spec.LogicalName  `json:"name"`
	MediaType         string            `json:"mediaType"`
	SuggestedFileName string            `json:"suggestedFileName"`
	Content           string            `json:"content"`
	ContentDigest     cryptoutil.Digest `json:"contentDigest"`
	DefinitionDigest  cryptoutil.Digest `json:"definitionDigest"`
	ArtifactRevision  uint64            `json:"artifactRevision"`
	BuiltIn           bool              `json:"builtIn"`
	Managed           bool              `json:"managed"`
}
