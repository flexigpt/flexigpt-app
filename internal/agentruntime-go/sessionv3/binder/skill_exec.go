package binder

import (
	"context"
	"sync"

	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
)

type skillSessions struct {
	mu          sync.Mutex
	byExecution map[sessionv3.ExecutionID]string
}

func newSkillSessions() *skillSessions {
	return &skillSessions{byExecution: map[sessionv3.ExecutionID]string{}}
}

func (b *Binder) executeSkillTool(
	ctx context.Context,
	req sessionv3.WorkDispatchRequest,
	payload *sessionv3.ToolWork,
) (sessionv3.WorkExecutionResult, error) {
	sessionID, err := b.ensureSkillSession(ctx, req)
	if err != nil {
		//nolint:nilerr // Explicit return.
		return sessionv3.WorkExecutionResult{
			Error: &sessionv3.SanitizedError{Code: "skill_session_failed", Message: err.Error()},
		}, nil
	}
	resp, err := b.deps.SkillRuntime.InvokeSkillTool(
		ctx,
		&skillRuntime.InvokeSkillToolRequest{
			Body: &skillRuntime.InvokeSkillToolRequestBody{
				SessionID: agentskillsRuntimeSpec.SessionID(sessionID),
				ToolName:  payload.Call.Name,
				Args:      string(payload.Call.Arguments),
			},
		},
	)
	if err != nil {
		//nolint:nilerr // Explicit return.
		return sessionv3.WorkExecutionResult{
			Error: &sessionv3.SanitizedError{Code: "skill_failed", Message: err.Error()},
		}, nil
	}
	if resp == nil || resp.Body == nil {
		return sessionv3.WorkExecutionResult{
			Error: &sessionv3.SanitizedError{Code: "skill_empty", Message: "skill returned no response"},
		}, nil
	}
	text := string(mustMarshal(resp.Body.Outputs))
	kind := sessionv3.ToolOutputKindSuccess
	var execErr *sessionv3.SanitizedError
	if resp.Body.IsError {
		kind = sessionv3.ToolOutputKindError
		execErr = &sessionv3.SanitizedError{Code: "skill_tool_error", Message: resp.Body.ErrorMessage}
	}
	return sessionv3.WorkExecutionResult{
		Output: &sessionv3.WorkItemOutput{
			Tool: &sessionv3.ToolOutput{
				Kind:       kind,
				Text:       text,
				Structured: sessionv3.MustCanonicalValue(resp.Body.Outputs),
			},
		},
		Error:     execErr,
		Confirmed: true,
	}, nil
}

func (b *Binder) ensureSkillSession(ctx context.Context, req sessionv3.WorkDispatchRequest) (string, error) {
	b.skills.mu.Lock()
	existing := b.skills.byExecution[req.Execution.ID]
	b.skills.mu.Unlock()
	if existing != "" {
		return existing, nil
	}
	created, err := b.deps.SkillRuntime.CreateSkillSession(
		ctx,
		&skillRuntime.CreateSkillSessionRequest{Body: &skillRuntime.CreateSkillSessionRequestBody{}},
	)
	if err != nil {
		return "", err
	}
	if created == nil || created.Body == nil || created.Body.SessionID == "" {
		return "", context.Canceled
	}
	id := string(created.Body.SessionID)
	b.skills.mu.Lock()
	b.skills.byExecution[req.Execution.ID] = id
	b.skills.mu.Unlock()
	return id, nil
}
