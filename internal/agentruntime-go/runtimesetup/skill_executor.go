package runtimesetup

import (
	"context"
	"errors"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
)

type skillToolExecutor struct {
	coordinator *coordinator
}

func newSkillToolExecutor(
	coordinator *coordinator,
) (*skillToolExecutor, error) {
	if coordinator == nil {
		return nil, errors.New(
			"session Skill Tool executor requires runtime setup",
		)
	}
	return &skillToolExecutor{
		coordinator: coordinator,
	}, nil
}

func (e *skillToolExecutor) EnsureSkillSession(
	ctx context.Context,
	request sessionRuntime.SkillSessionRequest,
) (sessionRuntime.SkillSessionResult, error) {
	return e.coordinator.ensureSkillSession(ctx, request)
}

func (e *skillToolExecutor) CloseSkillSession(
	ctx context.Context,
	sessionID string,
) error {
	return e.coordinator.closeSkillSession(ctx, sessionID)
}

func (e *skillToolExecutor) EvaluateTool(
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

func (e *skillToolExecutor) ExecuteTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolExecutionResult, error) {
	return e.coordinator.invokeSkillTool(ctx, request)
}

func (e *skillToolExecutor) ResolveToolWait(
	_ context.Context,
	_ sessionRuntime.ToolWaitResolutionRequest,
) (sessionRuntime.ToolWaitResolution, error) {
	return sessionRuntime.ToolWaitResolution{}, errors.New(
		"skill Tool execution does not expose an approval wait",
	)
}
