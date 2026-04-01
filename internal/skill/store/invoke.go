package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

// Defense-in-depth: cap JSON args size for skills tools.
const maxSkillToolArgsBytes = 1 << 20 // 1 MiB

func (s *SkillStore) InvokeSkillTool(
	ctx context.Context,
	req *spec.InvokeSkillToolRequest,
) (*spec.InvokeSkillToolResponse, error) {
	if s.runtime == nil {
		return nil, fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", spec.ErrSkillInvalidRequest)
	}

	sid := strings.TrimSpace(string(req.Body.SessionID))
	if sid == "" {
		return nil, fmt.Errorf("%w: sessionID required", spec.ErrSkillInvalidRequest)
	}

	toolName := strings.TrimSpace(req.Body.ToolName)
	if toolName == "" {
		return nil, fmt.Errorf("%w: toolName required", spec.ErrSkillInvalidRequest)
	}

	argsStr := strings.TrimSpace(req.Body.Args)
	if argsStr == "" {
		// Be forgiving: models (and manual retries) sometimes omit "{}".
		argsStr = "{}"
	}
	if len(argsStr) > maxSkillToolArgsBytes {
		return nil, fmt.Errorf("%w: args too large", spec.ErrSkillInvalidRequest)
	}
	if !json.Valid([]byte(argsStr)) {
		return nil, fmt.Errorf("%w: args must be valid JSON", spec.ErrSkillInvalidRequest)
	}
	trim := strings.TrimSpace(argsStr)
	if trim != "" && trim[0] != '{' {
		return nil, fmt.Errorf("%w: args must be a JSON object", spec.ErrSkillInvalidRequest)
	}

	reg, err := s.runtime.NewSessionRegistry(ctx, agentskillsSpec.SessionID(sid))
	if err != nil {
		return nil, err
	}

	var funcID string
	switch toolName {
	case "skills-load":
		funcID = string(agentskillsSpec.FuncIDSkillsLoad)
	case "skills-unload":
		funcID = string(agentskillsSpec.FuncIDSkillsUnload)
	case "skills-readresource":
		funcID = string(agentskillsSpec.FuncIDSkillsReadResource)
	case "skills-runscript":
		funcID = string(agentskillsSpec.FuncIDSkillsRunScript)
	default:
		return nil, fmt.Errorf("%w: unknown toolName %q", spec.ErrSkillInvalidRequest, toolName)
	}

	outs, callErr := llmtoolsutil.CallUsingRegistry(ctx, reg, funcID, json.RawMessage([]byte(argsStr)))
	isErr := callErr != nil
	errMsg := ""
	if callErr != nil {
		errMsg = callErr.Error()
	}

	return &spec.InvokeSkillToolResponse{
		Body: &spec.InvokeSkillToolResponseBody{
			Outputs:      outs,
			Meta:         map[string]any{"toolName": toolName},
			IsBuiltIn:    true,
			IsError:      isErr,
			ErrorMessage: errMsg,
		},
	}, nil
}
