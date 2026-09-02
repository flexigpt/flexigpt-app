package aggregate

import (
	"errors"

	"github.com/flexigpt/agentskills-go/document"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
)

var ErrArtifactSkillSelectionRequired = errors.New(
	"artifact skill selection is required",
)

// ArtifactSkillFilter is an internal Artifact Store to Agent Skills bridge
// filter. It is not a Wails transport API.
type ArtifactSkillFilter struct {
	Types          []string               `json:"types,omitempty"`
	Inserts        []document.SkillInsert `json:"inserts,omitempty"`
	NamePrefix     string                 `json:"namePrefix,omitempty"`
	LocationPrefix string                 `json:"locationPrefix,omitempty"`
	AllowArtifacts []artifact.ArtifactRef `json:"allowArtifacts,omitempty"`

	SessionID agentskillsRuntimeSpec.SessionID     `json:"sessionID,omitempty"`
	Activity  agentskillsRuntimeSpec.SkillActivity `json:"activity,omitempty"`
}

// ArtifactSkillSummary is the runtime-derived summary used by durable
// Artifact selection validation. It contains no native path or process-local
// SkillDef identity.
type ArtifactSkillSummary struct {
	Artifact     artifact.ArtifactRef
	IsEnabled    bool
	Insert       document.SkillInsert
	HasArguments bool
	HasResources bool
}
