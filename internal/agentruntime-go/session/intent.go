package session

import (
	"encoding/json"
	"strings"
	"time"
)

type ConfigurationOwner string

const (
	ConfigurationOwnerPlatform  ConfigurationOwner = "platform"
	ConfigurationOwnerModel     ConfigurationOwner = "model"
	ConfigurationOwnerAgent     ConfigurationOwner = "agent"
	ConfigurationOwnerWorkspace ConfigurationOwner = "workspace"
	ConfigurationOwnerSession   ConfigurationOwner = "session"
	ConfigurationOwnerRun       ConfigurationOwner = "run"
	ConfigurationOwnerUser      ConfigurationOwner = "user"
)

type ResolutionState string

const (
	ResolutionStateIncluded    ResolutionState = "included"
	ResolutionStateExcluded    ResolutionState = "excluded"
	ResolutionStateUnavailable ResolutionState = "unavailable"
	ResolutionStateChanged     ResolutionState = "changed"
	ResolutionStateBlocked     ResolutionState = "blocked"
	ResolutionStateStale       ResolutionState = "stale"
)

type DiagnosticSeverity string

const (
	DiagnosticSeverityInfo    DiagnosticSeverity = "info"
	DiagnosticSeverityWarning DiagnosticSeverity = "warning"
	DiagnosticSeverityError   DiagnosticSeverity = "error"
)

type ArtifactSelection struct {
	RootID           string   `json:"rootID"`
	ArtifactID       string   `json:"artifactID"`
	ExpectedRevision Revision `json:"expectedRevision,omitempty"`
	ExpectedDigest   string   `json:"expectedDigest,omitempty"`
}

func (s ArtifactSelection) CanonicalKey() string {
	return "artifact:" + s.RootID + ":" + s.ArtifactID
}

func (s ArtifactSelection) Reference() ArtifactReference {
	return ArtifactReference{
		RootID:     s.RootID,
		ArtifactID: s.ArtifactID,
	}
}

type DirectCapabilitySelection struct {
	ProviderIdentity string `json:"providerIdentity"`
	ProviderLocalID  string `json:"providerLocalID"`
	Name             string `json:"name"`
	Type             string `json:"type"`
}

func (s DirectCapabilitySelection) CanonicalKey() string {
	return "direct:" +
		s.ProviderIdentity + ":" +
		s.ProviderLocalID + ":" +
		s.Type + ":" +
		s.Name
}

type CapabilitySelection struct {
	Artifact *ArtifactSelection         `json:"artifact,omitempty"`
	Direct   *DirectCapabilitySelection `json:"direct,omitempty"`
}

func (s CapabilitySelection) CanonicalKey() string {
	switch {
	case s.Artifact != nil:
		return s.Artifact.CanonicalKey()
	case s.Direct != nil:
		return s.Direct.CanonicalKey()
	default:
		return ""
	}
}

type CollectionOverrideMode string

const (
	CollectionOverrideInherit CollectionOverrideMode = "inherit"
	CollectionOverrideAdd     CollectionOverrideMode = "add"
	CollectionOverrideRemove  CollectionOverrideMode = "remove"
	CollectionOverrideReplace CollectionOverrideMode = "replace"
)

type ActivityOverrideMode string

const (
	ActivityOverrideInherit    ActivityOverrideMode = "inherit"
	ActivityOverrideActivate   ActivityOverrideMode = "activate"
	ActivityOverrideDeactivate ActivityOverrideMode = "deactivate"
)

type ModelDefaultField string

const (
	ModelDefaultFieldAdapterParameters ModelDefaultField = "adapterParameters"
	ModelDefaultFieldCacheControl      ModelDefaultField = "cacheControl"
	ModelDefaultFieldMaxOutputTokens   ModelDefaultField = "maxOutputTokens"
	ModelDefaultFieldMaxPromptTokens   ModelDefaultField = "maxPromptTokens"
	ModelDefaultFieldOutput            ModelDefaultField = "output"
	ModelDefaultFieldReasoning         ModelDefaultField = "reasoning"
	ModelDefaultFieldStopSequences     ModelDefaultField = "stopSequences"
	ModelDefaultFieldStream            ModelDefaultField = "stream"
	ModelDefaultFieldSystemPrompt      ModelDefaultField = "systemPrompt"
	ModelDefaultFieldTemperature       ModelDefaultField = "temperature"
	ModelDefaultFieldTimeoutMS         ModelDefaultField = "timeoutMS"
)

type ModelReasoningOverride struct {
	Type         *string `json:"type,omitempty"`
	Level        *string `json:"level,omitempty"`
	Tokens       *int    `json:"tokens,omitempty"`
	SummaryStyle *string `json:"summaryStyle,omitempty"`
	Context      *string `json:"context,omitempty"`
	Mode         *string `json:"mode,omitempty"`
}

type ModelCacheControlOverride struct {
	Kind *string `json:"kind,omitempty"`
	TTL  *string `json:"ttl,omitempty"`
	Key  *string `json:"key,omitempty"`
}

type ModelOutputJSONSchemaOverride struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

type ModelOutputFormatOverride struct {
	Kind       *string                        `json:"kind,omitempty"`
	JSONSchema *ModelOutputJSONSchemaOverride `json:"jsonSchema,omitempty"`
}

type ModelOutputOverride struct {
	Verbosity *string                    `json:"verbosity,omitempty"`
	Format    *ModelOutputFormatOverride `json:"format,omitempty"`
}

type ModelDefaultsOverride struct {
	Stream            *bool                      `json:"stream,omitempty"`
	MaxPromptTokens   *int                       `json:"maxPromptTokens,omitempty"`
	MaxOutputTokens   *int                       `json:"maxOutputTokens,omitempty"`
	Temperature       *float64                   `json:"temperature,omitempty"`
	SystemPrompt      *string                    `json:"systemPrompt,omitempty"`
	TimeoutMS         *int                       `json:"timeoutMS,omitempty"`
	Reasoning         *ModelReasoningOverride    `json:"reasoning,omitempty"`
	CacheControl      *ModelCacheControlOverride `json:"cacheControl,omitempty"`
	Output            *ModelOutputOverride       `json:"output,omitempty"`
	StopSequences     *[]string                  `json:"stopSequences,omitempty"`
	AdapterParameters map[string]json.RawMessage `json:"adapterParameters,omitempty"`
	Clear             []ModelDefaultField        `json:"clear,omitempty"`
}

type InstructionSelection struct {
	Key      string             `json:"key,omitempty"`
	Artifact *ArtifactSelection `json:"artifact,omitempty"`
	Text     string             `json:"text,omitempty"`
	Label    string             `json:"label,omitempty"`
}

func (s InstructionSelection) CanonicalKey() string {
	if strings.TrimSpace(s.Key) != "" {
		return s.Key
	}
	if s.Artifact != nil {
		return s.Artifact.CanonicalKey()
	}
	if strings.TrimSpace(s.Label) != "" {
		return "inline:" + s.Label
	}
	return "inline:" + s.Text
}

type InstructionOverride struct {
	Key   string                 `json:"key"`
	Mode  CollectionOverrideMode `json:"mode"`
	Value *InstructionSelection  `json:"value,omitempty"`
}

type ToolSelectionIntent struct {
	Key           string              `json:"key,omitempty"`
	Capability    CapabilitySelection `json:"capability"`
	AutoExecute   *bool               `json:"autoExecute,omitempty"`
	UserArguments json.RawMessage     `json:"userArguments,omitempty"`
}

func (s ToolSelectionIntent) CanonicalKey() string {
	if strings.TrimSpace(s.Key) != "" {
		return s.Key
	}
	return s.Capability.CanonicalKey()
}

type ToolOverride struct {
	Key   string                 `json:"key"`
	Mode  CollectionOverrideMode `json:"mode"`
	Value *ToolSelectionIntent   `json:"value,omitempty"`
}

type ArtifactOverride struct {
	Key   string                 `json:"key"`
	Mode  CollectionOverrideMode `json:"mode"`
	Value *ArtifactSelection     `json:"value,omitempty"`
}

type ActiveSkillOverride struct {
	Key   string               `json:"key"`
	Mode  ActivityOverrideMode `json:"mode"`
	Value *ArtifactSelection   `json:"value,omitempty"`
}

type MCPToolExposure string

const (
	MCPToolExposureNone     MCPToolExposure = "none"
	MCPToolExposureAll      MCPToolExposure = "all"
	MCPToolExposureSelected MCPToolExposure = "selected"
)

type MCPToolSelection struct {
	ToolName       string  `json:"toolName"`
	ExpectedDigest string  `json:"expectedDigest,omitempty"`
	ApprovalRule   *string `json:"approvalRule,omitempty"`
	ExecutionMode  *string `json:"executionMode,omitempty"`
}

type MCPResourceSelection struct {
	URI            string `json:"uri"`
	ExpectedDigest string `json:"expectedDigest,omitempty"`
}

type MCPResourceTemplateSelection struct {
	URITemplate    string            `json:"uriTemplate"`
	ExpectedDigest string            `json:"expectedDigest,omitempty"`
	Arguments      map[string]string `json:"arguments,omitempty"`
}

type MCPPromptSelection struct {
	PromptName     string            `json:"promptName"`
	ExpectedDigest string            `json:"expectedDigest,omitempty"`
	Arguments      map[string]string `json:"arguments,omitempty"`
}

type MCPServerSelection struct {
	Key                       string                         `json:"key,omitempty"`
	Server                    ArtifactSelection              `json:"server"`
	IncludeServerInstructions bool                           `json:"includeServerInstructions,omitempty"`
	ToolExposure              MCPToolExposure                `json:"toolExposure"`
	SelectedTools             []MCPToolSelection             `json:"selectedTools,omitempty"`
	Resources                 []MCPResourceSelection         `json:"resources,omitempty"`
	ResourceTemplates         []MCPResourceTemplateSelection `json:"resourceTemplates,omitempty"`
	Prompts                   []MCPPromptSelection           `json:"prompts,omitempty"`
}

func (s MCPServerSelection) CanonicalKey() string {
	if strings.TrimSpace(s.Key) != "" {
		return s.Key
	}
	return s.Server.CanonicalKey()
}

type MCPServerOverride struct {
	Key   string                 `json:"key"`
	Mode  CollectionOverrideMode `json:"mode"`
	Value *MCPServerSelection    `json:"value,omitempty"`
}

type WorkspaceSelectionIntent struct {
	Workspace ArtifactSelection   `json:"workspace"`
	Context   []ArtifactOverride  `json:"context,omitempty"`
	Skills    []ArtifactOverride  `json:"skills,omitempty"`
	MCP       []MCPServerOverride `json:"mcp,omitempty"`
}

type SessionConfiguration struct {
	Agent          *ArtifactSelection        `json:"agent,omitempty"`
	Workspace      *WorkspaceSelectionIntent `json:"workspace,omitempty"`
	Model          *ArtifactSelection        `json:"model,omitempty"`
	ModelOverrides ModelDefaultsOverride     `json:"modelOverrides,omitzero"`

	Instructions []InstructionOverride `json:"instructions,omitempty"`
	Tools        []ToolOverride        `json:"tools,omitempty"`
	Skills       []ArtifactOverride    `json:"skills,omitempty"`
	ActiveSkills []ActiveSkillOverride `json:"activeSkills,omitempty"`
	MCP          []MCPServerOverride   `json:"mcp,omitempty"`

	History  HistorySelection `json:"history,omitzero"`
	Metadata json.RawMessage  `json:"metadata,omitempty"`
}

type AppContextContent struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	URI         string `json:"uri,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
	Data        []byte `json:"data,omitempty"`
}

type AppContextUpdate struct {
	AppInstanceID     AppInstanceID       `json:"appInstanceID"`
	ViewID            AppViewID           `json:"viewID"`
	ServerID          string              `json:"serverID"`
	ResourceURI       string              `json:"resourceURI,omitempty"`
	Content           []AppContextContent `json:"content,omitempty"`
	StructuredContent json.RawMessage     `json:"structuredContent,omitempty"`
	UpdatedAt         time.Time           `json:"updatedAt"`
}

type ResolutionDiagnostic struct {
	Severity DiagnosticSeverity `json:"severity"`
	Code     string             `json:"code"`
	Message  string             `json:"message"`
	Key      string             `json:"key,omitempty"`
}

type ResolvedArtifact struct {
	RootID     string   `json:"rootID"`
	ArtifactID string   `json:"artifactID"`
	Revision   Revision `json:"revision,omitempty"`
	Digest     string   `json:"digest,omitempty"`
}

type ResolutionItem struct {
	Key      string             `json:"key"`
	Kind     string             `json:"kind"`
	Owner    ConfigurationOwner `json:"owner"`
	State    ResolutionState    `json:"state"`
	Selected *ArtifactSelection `json:"selected,omitempty"`
	Used     *ResolvedArtifact  `json:"used,omitempty"`
	Reason   string             `json:"reason,omitempty"`
}

type OperationResolutionSummary struct {
	Version      uint32                 `json:"version"`
	Agent        *ResolutionItem        `json:"agent,omitempty"`
	Workspace    *ResolutionItem        `json:"workspace,omitempty"`
	Model        *ResolutionItem        `json:"model,omitempty"`
	Instructions []ResolutionItem       `json:"instructions,omitempty"`
	Tools        []ResolutionItem       `json:"tools,omitempty"`
	Skills       []ResolutionItem       `json:"skills,omitempty"`
	MCP          []ResolutionItem       `json:"mcp,omitempty"`
	Attachments  []ResolutionItem       `json:"attachments,omitempty"`
	Diagnostics  []ResolutionDiagnostic `json:"diagnostics,omitempty"`
}

type PreparationState string

const (
	PreparationStatePending  PreparationState = "pending"
	PreparationStatePrepared PreparationState = "prepared"
	PreparationStateFailed   PreparationState = "failed"
)

type OperationPreparation struct {
	State           PreparationState           `json:"state"`
	Fingerprint     string                     `json:"fingerprint,omitempty"`
	PreparedAt      time.Time                  `json:"preparedAt,omitzero"`
	Resolution      OperationResolutionSummary `json:"resolution"`
	EffectiveSkills []ArtifactReference        `json:"effectiveSkills,omitempty"`
	ActiveSkills    []ArtifactReference        `json:"activeSkills,omitempty"`

	// Private is an immutable in-memory provider-ready operation payload.
	// It is never serialized, checkpointed, or exposed through snapshots.
	Private any `json:"-"`
}

type ToolCallSource string

const (
	ToolCallSourceModel ToolCallSource = "model"
	ToolCallSourceApp   ToolCallSource = "app"
)

type AppRoute struct {
	ServerID    string      `json:"serverID"`
	ResourceURI string      `json:"resourceURI,omitempty"`
	Binding     ToolBinding `json:"-"`
}

type AppInstance struct {
	ID                AppInstanceID `json:"id"`
	SessionID         SessionID     `json:"sessionID"`
	RunID             RunID         `json:"runID"`
	OriginOperationID OperationID   `json:"originOperationID"`
	OriginToolCallID  ToolCallID    `json:"originToolCallID"`
	ServerID          string        `json:"serverID"`
	ResourceURI       string        `json:"resourceURI,omitempty"`
	CreatedAt         time.Time     `json:"createdAt"`
	Views             []AppViewID   `json:"views,omitempty"`
	Route             *AppRoute     `json:"-"`
	UnavailableReason string        `json:"unavailableReason,omitempty"`
}
