package skillruntime

import (
	"context"
	"errors"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
)

// SessionOperations is the Session-operation preparation capability that Skill
// Runtime needs. The interface keeps the Session core independent from Skill
// Runtime implementation details while keeping Skill execution in this domain.
type SessionOperations interface {
	EnsureSkillSession(
		ctx context.Context,
		request sessionRuntime.SkillSessionRequest,
	) (sessionRuntime.SkillSessionResult, error)

	CloseSkillSession(
		ctx context.Context,
		sessionID string,
	) error

	InvokeSkillTool(
		ctx context.Context,
		request sessionRuntime.ToolExecutionRequest,
	) (sessionRuntime.ToolExecutionResult, error)
}

// SessionToolExecutor adapts a prepared Session operation to the Skill Runtime.
type SessionToolExecutor struct {
	operations SessionOperations
}

func NewSessionToolExecutor(
	operations SessionOperations,
) (*SessionToolExecutor, error) {
	if operations == nil {
		return nil, errors.New(
			"session Skill Tool executor requires Skill session operations",
		)
	}

	return &SessionToolExecutor{operations: operations}, nil
}

func (e *SessionToolExecutor) EnsureSkillSession(
	ctx context.Context,
	request sessionRuntime.SkillSessionRequest,
) (sessionRuntime.SkillSessionResult, error) {
	return e.operations.EnsureSkillSession(ctx, request)
}

func (e *SessionToolExecutor) CloseSkillSession(
	ctx context.Context,
	sessionID string,
) error {
	return e.operations.CloseSkillSession(ctx, sessionID)
}

func (e *SessionToolExecutor) EvaluateTool(
	_ context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolEvaluation, error) {
	if request.SkillSessionID == "" {
		return sessionRuntime.ToolEvaluation{
			Decision: sessionRuntime.ToolEvaluationDenied,
			Reason:   "the Run has no active Skill Session",
			Result: &sessionRuntime.ToolExecutionResult{
				HasResult: true,
				Kind:      sessionRuntime.ToolResultKindError,
				Text:      "Skill Tool error: no active Skill Session exists for this Run.",
				Error:     "no active Skill Session",
			},
		}, nil
	}

	return sessionRuntime.ToolEvaluation{
		Decision: sessionRuntime.ToolEvaluationAllowed,
	}, nil
}

func (e *SessionToolExecutor) ExecuteTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolExecutionResult, error) {
	return e.operations.InvokeSkillTool(ctx, request)
}

func (e *SessionToolExecutor) ResolveToolWait(
	_ context.Context,
	_ sessionRuntime.ToolWaitResolutionRequest,
) (sessionRuntime.ToolWaitResolution, error) {
	return sessionRuntime.ToolWaitResolution{}, errors.New(
		"skill tool execution does not expose an approval wait",
	)
}
