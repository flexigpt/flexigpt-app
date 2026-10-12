package spec

import (
	"context"

	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
)

// SkillSource is the completion consumer's Skill capability.
// Its implementation owns Artifact resolution, runtime synchronization,
// session filtering, and built-in Skill tool preparation.
type SkillSource interface {
	ResolveSkillSession(
		ctx context.Context,
		request SkillSessionRequest,
	) (SkillSession, error)

	RunScriptsEnabled() bool
}

type SkillSessionRequest struct {
	SessionID agentskillsRuntimeSpec.SessionID
	Artifacts []artifactModel.ArtifactRef

	// This may narrow runtime policy, but must not enable script execution
	// when the runtime itself does not support it.
	IncludeRunScript bool
}

// SkillSession is completion-local hydration output, not persistent state.
// Implementations return a zero result on error.
type SkillSession struct {
	AvailableArtifacts []artifactModel.ArtifactRef
	ActiveArtifacts    []artifactModel.ArtifactRef

	RulesPrompt string
	Prompt      string
	ToolChoices []inferenceSpec.ToolChoice
}
