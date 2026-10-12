package skill

import (
	"context"
	"encoding/json"
	"errors"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
)

type ArtifactSessionRequest struct {
	PreviousSessionID string                             `json:"previousSessionID,omitempty"`
	Enabled           []sessionRuntime.ArtifactReference `json:"enabled,omitempty"`
	Active            []sessionRuntime.ArtifactReference `json:"active,omitempty"`
}

type ArtifactSessionResult struct {
	SessionID string                             `json:"sessionID"`
	Active    []sessionRuntime.ArtifactReference `json:"active,omitempty"`
}

type ToolInvocationResult struct {
	Outputs json.RawMessage `json:"outputs,omitempty"`
	Meta    json.RawMessage `json:"meta,omitempty"`
	IsError bool            `json:"isError"`
	Error   string          `json:"error,omitempty"`

	Active      []sessionRuntime.ArtifactReference `json:"active,omitempty"`
	ActiveKnown bool                               `json:"activeKnown,omitempty"`
}

type ArtifactSessionService interface {
	EnsureArtifactSkillSession(
		ctx context.Context,
		request ArtifactSessionRequest,
	) (ArtifactSessionResult, error)

	CloseArtifactSkillSession(
		ctx context.Context,
		sessionID string,
	) error

	InvokeArtifactSkillTool(
		ctx context.Context,
		sessionID string,
		toolName string,
		args json.RawMessage,
	) (ToolInvocationResult, error)

	ListActiveArtifactSkills(
		ctx context.Context,
		sessionID string,
	) ([]sessionRuntime.ArtifactReference, error)
}

type Adapter struct {
	service ArtifactSessionService
}

func NewAdapter(
	service ArtifactSessionService,
) (*Adapter, error) {
	if service == nil {
		return nil, errors.New("session skill adapter requires an artifact skill-session service")
	}

	return &Adapter{service: service}, nil
}

func (a *Adapter) EnsureSkillSession(
	ctx context.Context,
	request sessionRuntime.SkillSessionRequest,
) (sessionRuntime.SkillSessionResult, error) {
	result, err := a.service.EnsureArtifactSkillSession(
		ctx,
		ArtifactSessionRequest{
			PreviousSessionID: request.SessionID,
			Enabled:           request.Enabled,
			Active:            request.Active,
		},
	)
	if err != nil {
		return sessionRuntime.SkillSessionResult{}, err
	}

	return sessionRuntime.SkillSessionResult{
		SessionID: result.SessionID,
		Active:    result.Active,
	}, nil
}

func (a *Adapter) CloseSkillSession(
	ctx context.Context,
	sessionID string,
) error {
	return a.service.CloseArtifactSkillSession(ctx, sessionID)
}

func (a *Adapter) EvaluateTool(
	_ context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolEvaluation, error) {
	if request.SkillSessionID == "" {
		return sessionRuntime.ToolEvaluation{
			Decision: sessionRuntime.ToolEvaluationDenied,
			Reason:   "the Run does not have an active skill session",
			Result: &sessionRuntime.ToolExecutionResult{
				HasResult: true,
				Kind:      sessionRuntime.ToolResultKindError,
				Text:      "Skill tool error: no active skill session exists for this Run.",
				Error:     "no active skill session",
			},
		}, nil
	}

	return sessionRuntime.ToolEvaluation{
		Decision: sessionRuntime.ToolEvaluationAllowed,
	}, nil
}

func (a *Adapter) ExecuteTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolExecutionResult, error) {
	if request.SkillSessionID == "" {
		return sessionRuntime.ToolExecutionResult{}, errors.New(
			"skill tool execution requires a skill session ID",
		)
	}

	result, err := a.service.InvokeArtifactSkillTool(
		ctx,
		request.SkillSessionID,
		request.ToolCall.Name,
		request.ToolCall.Arguments,
	)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}

	text := string(result.Outputs)
	if text == "" {
		text = "Skill tool returned no output."
	}

	output := sessionRuntime.ToolExecutionResult{
		HasResult:  true,
		Kind:       sessionRuntime.ToolResultKindSuccess,
		Text:       text,
		Structured: result.Outputs,
	}
	if result.IsError {
		output.Kind = sessionRuntime.ToolResultKindError
		output.Error = result.Error
		if output.Error == "" {
			output.Error = "skill tool returned an error result"
		}
	}

	if isSkillLifecycleTool(request.ToolCall.Name) {
		active, activeErr := a.service.ListActiveArtifactSkills(
			ctx,
			request.SkillSessionID,
		)
		if activeErr == nil {
			output.SkillSessionActive = active
			output.SkillSessionActiveKnown = true
		}
	}

	return output, nil
}

func (a *Adapter) ResolveToolWait(
	_ context.Context,
	_ sessionRuntime.ToolWaitResolutionRequest,
) (sessionRuntime.ToolWaitResolution, error) {
	return sessionRuntime.ToolWaitResolution{}, errors.New(
		"skill tool execution does not expose an approval wait",
	)
}

func isSkillLifecycleTool(name string) bool {
	switch name {
	case "skills-load", "skills-unload":
		return true
	default:
		return false
	}
}
