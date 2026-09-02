package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntime "github.com/flexigpt/agentskills-go/runtime"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
)

func (s *Service) CreateSkillSession(
	ctx context.Context,
	req *CreateSkillSessionRequest,
) (*CreateSkillSessionResponse, error) {
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", ErrInvalidRequest)
	}

	options := make([]agentskillsRuntime.SessionOption, 0, 3)
	if req.Body.MaxActivePerSession > 0 {
		options = append(
			options,
			agentskillsRuntime.WithSessionMaxActivePerSession(
				req.Body.MaxActivePerSession,
			),
		)
	}

	// Nil means the caller omitted the policy. In that case no allowlist is
	// created and agentskills-go retains its unrestricted fast path.
	if req.Body.AllowedSkills != nil {
		options = append(
			options,
			agentskillsRuntime.WithSessionAllowedSkills(
				req.Body.AllowedSkills,
			),
		)
	}
	if len(req.Body.ActiveSkills) != 0 {
		options = append(
			options,
			agentskillsRuntime.WithSessionActiveSkills(
				req.Body.ActiveSkills,
			),
		)
	}

	sessionID, active, err := s.NewSession(ctx, options...)
	if err != nil {
		return nil, err
	}

	if req.Body.CloseSessionID != "" {
		_ = s.CloseSession(
			context.WithoutCancel(ctx),
			req.Body.CloseSessionID,
		)
	}

	return &CreateSkillSessionResponse{
		Body: &CreateSkillSessionResponseBody{
			SessionID:    sessionID,
			ActiveSkills: active,
		},
	}, nil
}

func (s *Service) CloseSkillSession(
	ctx context.Context,
	req *CloseSkillSessionRequest,
) (*CloseSkillSessionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: missing request", ErrInvalidRequest)
	}
	if err := s.CloseSession(ctx, req.SessionID); err != nil {
		return nil, err
	}
	return &CloseSkillSessionResponse{}, nil
}

func (s *Service) GetSkillsPrompt(
	ctx context.Context,
	req *GetSkillsPromptRequest,
) (*GetSkillsPromptResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: missing request", ErrInvalidRequest)
	}

	var filter *agentskillsRuntime.SkillFilter
	if req.Body != nil && req.Body.Filter != nil {
		input := req.Body.Filter
		filter = &agentskillsRuntime.SkillFilter{
			Types:          append([]string(nil), input.Types...),
			NamePrefix:     input.NamePrefix,
			LocationPrefix: input.LocationPrefix,
			AllowSkills:    append([]provider.SkillDef(nil), input.AllowSkills...),
			SessionID:      input.SessionID,
			Activity:       input.Activity,
		}
	}

	prompt, err := s.SkillsPrompt(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &GetSkillsPromptResponse{
		Body: &GetSkillsPromptResponseBody{Prompt: prompt},
	}, nil
}

func (s *Service) ListSkills(
	ctx context.Context,
	req *ListSkillsRequest,
) (*ListSkillsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: missing request", ErrInvalidRequest)
	}

	var filter *agentskillsRuntime.SkillListFilter
	if req.Body != nil && req.Body.Filter != nil {
		input := req.Body.Filter
		filter = &agentskillsRuntime.SkillListFilter{
			Types:          append([]string(nil), input.Types...),
			NamePrefix:     input.NamePrefix,
			LocationPrefix: input.LocationPrefix,
			AllowSkills:    append([]provider.SkillDef(nil), input.AllowSkills...),
			Inserts:        append([]document.SkillInsert(nil), input.Inserts...),
			SessionID:      input.SessionID,
			Activity:       input.Activity,
		}
	}

	records, err := s.ListAgentSkills(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &ListSkillsResponse{
		Body: &ListSkillsResponseBody{Skills: records},
	}, nil
}

func (s *Service) RenderSkill(
	ctx context.Context,
	req *RenderSkillRequest,
) (*RenderSkillResponse, error) {
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", ErrInvalidRequest)
	}

	output, err := s.RenderAgentSkill(
		ctx,
		agentskillsRuntime.RenderSkillParams{
			Def:       req.Body.Definition,
			Arguments: req.Body.Arguments,
		},
	)
	if err != nil {
		return nil, err
	}
	return &RenderSkillResponse{Body: &output}, nil
}

func (s *Service) InvokeSkillTool(
	ctx context.Context,
	req *InvokeSkillToolRequest,
) (*InvokeSkillToolResponse, error) {
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", ErrInvalidRequest)
	}

	sessionID := strings.TrimSpace(string(req.Body.SessionID))
	if sessionID == "" {
		return nil, fmt.Errorf("%w: sessionID required", ErrInvalidRequest)
	}

	toolName := strings.TrimSpace(req.Body.ToolName)
	if toolName == "" {
		return nil, fmt.Errorf("%w: toolName required", ErrInvalidRequest)
	}

	arguments := strings.TrimSpace(req.Body.Args)
	if arguments == "" {
		arguments = "{}"
	}
	if len(arguments) > maxSkillToolArgsBytes {
		return nil, fmt.Errorf("%w: args too large", ErrInvalidRequest)
	}
	if !json.Valid([]byte(arguments)) || arguments[0] != '{' {
		return nil, fmt.Errorf(
			"%w: args must be a JSON object",
			ErrInvalidRequest,
		)
	}

	registry, err := s.NewSessionRegistry(
		ctx,
		agentskillsRuntimeSpec.SessionID(sessionID),
	)
	if err != nil {
		return nil, err
	}

	var functionID string
	switch toolName {
	case "skills-load":
		functionID = string(agentskillsRuntimeSpec.FuncIDSkillsLoad)
	case "skills-unload":
		functionID = string(agentskillsRuntimeSpec.FuncIDSkillsUnload)
	case "skills-readresource":
		functionID = string(agentskillsRuntimeSpec.FuncIDSkillsReadResource)
	case "skills-runscript":
		if !s.SupportsRunScript() {
			return nil, fmt.Errorf(
				"%w: skills-runscript is disabled by runtime policy",
				ErrInvalidRequest,
			)
		}
		functionID = string(agentskillsRuntimeSpec.FuncIDSkillsRunScript)
	default:
		return nil, fmt.Errorf(
			"%w: unknown toolName %q",
			ErrInvalidRequest,
			toolName,
		)
	}

	outputs, callErr := llmtoolsutil.CallUsingRegistry(
		ctx,
		registry,
		functionID,
		json.RawMessage(arguments),
	)
	response := &InvokeSkillToolResponse{
		Body: &InvokeSkillToolResponseBody{
			Outputs:   outputs,
			Meta:      map[string]any{"toolName": toolName},
			IsBuiltIn: true,
		},
	}
	if callErr != nil {
		response.Body.IsError = true
		response.Body.ErrorMessage = callErr.Error()
	}
	return response, nil
}
