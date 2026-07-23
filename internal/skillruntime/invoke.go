package skillruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/flexigpt-app/internal/skillruntime/spec"
)

const maxSkillToolArgsBytes = 1 << 20

func (s *SkillRuntime) InvokeSkillTool(
	ctx context.Context,
	req *spec.InvokeSkillToolRequest,
) (*spec.InvokeSkillToolResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", errSkillInvalidRequest)
	}
	sessionID := strings.TrimSpace(string(req.Body.SessionID))
	if sessionID == "" {
		return nil, fmt.Errorf("%w: sessionID required", errSkillInvalidRequest)
	}
	toolName := strings.TrimSpace(req.Body.ToolName)
	if toolName == "" {
		return nil, fmt.Errorf("%w: toolName required", errSkillInvalidRequest)
	}
	arguments := strings.TrimSpace(req.Body.Args)
	if arguments == "" {
		arguments = "{}"
	}
	if len(arguments) > maxSkillToolArgsBytes {
		return nil, fmt.Errorf("%w: args too large", errSkillInvalidRequest)
	}
	if !json.Valid([]byte(arguments)) {
		return nil, fmt.Errorf("%w: args must be valid JSON", errSkillInvalidRequest)
	}
	if arguments[0] != '{' {
		return nil, fmt.Errorf("%w: args must be a JSON object", errSkillInvalidRequest)
	}

	registry, err := s.runtime.NewSessionRegistry(ctx, agentskillsSpec.SessionID(sessionID))
	if err != nil {
		return nil, err
	}
	var functionID string
	switch toolName {
	case "skills-load":
		functionID = string(agentskillsSpec.FuncIDSkillsLoad)
	case "skills-unload":
		functionID = string(agentskillsSpec.FuncIDSkillsUnload)
	case "skills-readresource":
		functionID = string(agentskillsSpec.FuncIDSkillsReadResource)
	case "skills-runscript":
		functionID = string(agentskillsSpec.FuncIDSkillsRunScript)
	default:
		return nil, fmt.Errorf("%w: unknown toolName %q", errSkillInvalidRequest, toolName)
	}

	outputs, callErr := llmtoolsutil.CallUsingRegistry(ctx, registry, functionID, json.RawMessage(arguments))
	response := &spec.InvokeSkillToolResponse{Body: &spec.InvokeSkillToolResponseBody{
		Outputs:   outputs,
		Meta:      map[string]any{"toolName": toolName},
		IsBuiltIn: true,
	}}
	if callErr != nil {
		response.Body.IsError = true
		response.Body.ErrorMessage = callErr.Error()
	}
	return response, nil
}
