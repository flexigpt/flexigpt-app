package sessionv3runtime

import (
	"context"
	"sync"

	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
)

type skillSessions struct {
	mu          sync.Mutex
	byExecution map[sessionv3Runtime.ExecutionID]string
}

func newSkillSessions() *skillSessions {
	return &skillSessions{byExecution: map[sessionv3Runtime.ExecutionID]string{}}
}

func (b *Binder) executeSkillTool(
	ctx context.Context,
	req sessionv3Runtime.WorkDispatchRequest,
	payload *sessionv3Runtime.ToolWork,
) (sessionv3Runtime.WorkExecutionResult, error) {
	sessionID, err := b.ensureSkillSession(ctx, req)
	if err != nil {
		//nolint:nilerr // Explicit return.
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{Code: "skill_session_failed", Message: err.Error()},
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
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{Code: "skill_failed", Message: err.Error()},
		}, nil
	}
	if resp == nil || resp.Body == nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{Code: "skill_empty", Message: "skill returned no response"},
		}, nil
	}
	structured, err := canonicalValue(resp.Body.Outputs)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{}, err
	}

	text := string(structured)
	kind := sessionv3Runtime.ToolOutputKindSuccess
	var execErr *sessionv3Runtime.SanitizedError
	if resp.Body.IsError {
		kind = sessionv3Runtime.ToolOutputKindError
		execErr = &sessionv3Runtime.SanitizedError{Code: "skill_tool_error", Message: resp.Body.ErrorMessage}
	}
	return sessionv3Runtime.WorkExecutionResult{
		Output: &sessionv3Runtime.WorkItemOutput{
			Tool: &sessionv3Runtime.ToolOutput{
				Kind:       kind,
				Text:       text,
				Structured: structured,
			},
		},
		Error:     execErr,
		Confirmed: true,
	}, nil
}

func (b *Binder) ensureSkillSession(ctx context.Context, req sessionv3Runtime.WorkDispatchRequest) (string, error) {
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
