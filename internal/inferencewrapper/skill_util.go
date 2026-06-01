package inferencewrapper

import (
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

func buildSkillToolChoices(includeAll, includeRunScript bool) ([]inferenceSpec.ToolChoice, error) {
	mk := func(choiceID, toolName string, t llmtoolsSpec.Tool) (inferenceSpec.ToolChoice, error) {
		schema, err := decodeToolArgSchema(toolSpec.JSONRawString(t.ArgSchema))
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
	tc, err := mk("builtin.skills-load", "skills-load", agentskillsSpec.SkillsLoadTool())
	if err != nil {
		return nil, err
	}
	out = append(out, tc)

	if includeAll {
		if tc, err = mk("builtin.skills-unload", "skills-unload", agentskillsSpec.SkillsUnloadTool()); err != nil {
			return nil, err
		}
		out = append(out, tc)
		if tc, err = mk(
			"builtin.skills-readresource",
			"skills-readresource",
			agentskillsSpec.SkillsReadResourceTool(),
		); err != nil {
			return nil, err
		}
		out = append(out, tc)
		if includeRunScript {
			if tc, err = mk(
				"builtin.skills-runscript",
				"skills-runscript",
				agentskillsSpec.SkillsRunScriptTool(),
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
		return agentskillsSpec.SkillsRulesPromptLoadOnly
	}

	if !includeRunScript {
		return agentskillsSpec.SkillsRulesPromptWithoutRunScript
	}

	return agentskillsSpec.SkillsRulesPromptAll
}
