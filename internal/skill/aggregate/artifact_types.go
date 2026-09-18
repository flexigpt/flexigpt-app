package aggregate

import (
	"errors"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
)

var ErrArtifactSkillSelectionRequired = errors.New(
	"artifact skill selection is required",
)

type ArtifactSkillFilter struct {
	Types          []string               `json:"types,omitempty"`
	Inserts        []document.SkillInsert `json:"inserts,omitempty"`
	NamePrefix     string                 `json:"namePrefix,omitempty"`
	LocationPrefix string                 `json:"locationPrefix,omitempty"`
	AllowArtifacts []artifact.ArtifactRef `json:"allowArtifacts,omitempty"`

	SessionID agentskillsRuntimeSpec.SessionID     `json:"sessionID,omitempty"`
	Activity  agentskillsRuntimeSpec.SkillActivity `json:"activity,omitempty"`
}

type ArtifactSkillSummary struct {
	Artifact     artifact.ArtifactRef
	IsEnabled    bool
	Insert       document.SkillInsert
	HasArguments bool
	HasResources bool
}

type ResolvedArtifactSkill struct {
	Artifact   artifact.ArtifactRef
	Definition provider.SkillDef
	Version    string
	Enabled    bool
}
