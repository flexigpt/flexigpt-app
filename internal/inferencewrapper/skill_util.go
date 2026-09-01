package inferencewrapper

import (
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

func buildSkillToolChoices(includeAll, includeRunScript bool) ([]inferenceSpec.ToolChoice, error) {
	mk := func(choiceID, toolName string, t llmtoolsSpec.Tool) (inferenceSpec.ToolChoice, error) {
		schema, err := decodeToolArgSchema(jsonutil.JSONRawString(t.ArgSchema))
		if err != nil {
			return inferenceSpec.ToolChoice{}, err
		}
		return inferenceSpec.ToolChoice{
			Type:        inferenceSpec.ToolTypeFunction,
			ID:          choiceID, // choiceID (ToolCall.choiceID)
			Name:        toolName, // ToolCall.name
			Description: t.Description,
			Arguments:   schema,
		}, nil
	}

	var out []inferenceSpec.ToolChoice
	tc, err := mk("builtin.skills-load", "skills-load", agentskillsRuntimeSpec.SkillsLoadTool())
	if err != nil {
		return nil, err
	}
	out = append(out, tc)

	if includeAll {
		if tc, err = mk(
			"builtin.skills-unload",
			"skills-unload",
			agentskillsRuntimeSpec.SkillsUnloadTool(),
		); err != nil {
			return nil, err
		}
		out = append(out, tc)
		if tc, err = mk(
			"builtin.skills-readresource",
			"skills-readresource",
			agentskillsRuntimeSpec.SkillsReadResourceTool(),
		); err != nil {
			return nil, err
		}
		out = append(out, tc)
		if includeRunScript {
			if tc, err = mk(
				"builtin.skills-runscript",
				"skills-runscript",
				agentskillsRuntimeSpec.SkillsRunScriptTool(),
			); err != nil {
				return nil, err
			}
			out = append(out, tc)
		}
	}
	return out, nil
}

func skillsRulesPrompt(includeAll, includeRunScript bool) string {
	if !includeAll {
		return agentskillsRuntimeSpec.SkillsRulesPromptLoadOnly
	}

	if !includeRunScript {
		return agentskillsRuntimeSpec.SkillsRulesPromptWithoutRunScript
	}

	return agentskillsRuntimeSpec.SkillsRulesPromptAll
}
