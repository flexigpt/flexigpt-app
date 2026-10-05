package consumerapi

import (
	"time"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
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
	RootID          rootModel.RootID          `json:"rootID"`
	RootDisplayName string                    `json:"rootDisplayName,omitempty"`
	SourceID        sourceModel.SourceID      `json:"sourceID"`
	Plugin          artifactModel.ArtifactRef `json:"plugin"`

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

	Plugin                     artifactModel.ArtifactRef `json:"plugin"`
	ExpectedCollectionRevision uint64                    `json:"expectedCollectionRevision"`

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
	Path   string                       `json:"path"`
	Type   declaration.Type             `json:"type"`
	Name   spec.LogicalName             `json:"name"`
	Scope  declaration.LookupScope      `json:"scope,omitempty"`
	Status composition.ResolutionStatus `json:"status"`

	Target *composition.CapabilityTarget `json:"target,omitempty"`

	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type AgentImportConflict struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type AgentRestoredMembership struct {
	Plugin  artifactModel.ArtifactRef `json:"plugin"`
	Path    string                    `json:"path"`
	Message string                    `json:"message"`
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

	Artifact *artifactModel.ArtifactRef `json:"artifact,omitempty"`

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
	Plugin              plugin.PluginView         `json:"plugin"`
	RestoredMemberships []AgentRestoredMembership `json:"restoredMemberships,omitempty"`
	MCPSetupDescriptors []AgentMCPSetupDescriptor `json:"mcpSetupDescriptors,omitempty"`
	PreparedFingerprint cryptoutil.Digest         `json:"preparedFingerprint"`
}

type AgentExportRequest struct {
	Agent artifactModel.ArtifactRef `json:"agent"`
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
