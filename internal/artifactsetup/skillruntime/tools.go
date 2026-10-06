package skillruntime

import (
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func buildSkillToolChoices(
	includeAll bool,
	includeRunScript bool,
) ([]inferenceSpec.ToolChoice, error) {
	definitions := []struct {
		name string
		tool llmtoolsSpec.Tool
	}{
		{name: "skills-load", tool: agentskillsRuntimeSpec.SkillsLoadTool()},
	}
	if includeAll {
		definitions = append(definitions,
			struct {
				name string
				tool llmtoolsSpec.Tool
			}{name: "skills-unload", tool: agentskillsRuntimeSpec.SkillsUnloadTool()},
			struct {
				name string
				tool llmtoolsSpec.Tool
			}{name: "skills-readresource", tool: agentskillsRuntimeSpec.SkillsReadResourceTool()},
		)
		if includeRunScript {
			definitions = append(definitions, struct {
				name string
				tool llmtoolsSpec.Tool
			}{name: "skills-runscript", tool: agentskillsRuntimeSpec.SkillsRunScriptTool()})
		}
	}

	output := make([]inferenceSpec.ToolChoice, 0, len(definitions))
	for _, definition := range definitions {
		arguments, err := decodeToolArgSchema(
			jsonutil.JSONRawString(definition.tool.ArgSchema),
		)
		if err != nil {
			return nil, err
		}
		output = append(output, inferenceSpec.ToolChoice{
			Type:        inferenceSpec.ToolTypeFunction,
			ID:          "builtin." + definition.name,
			Name:        definition.name,
			Description: definition.tool.Description,
			Arguments:   arguments,
		})
	}
	return output, nil
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

func decodeToolArgSchema(
	raw jsonutil.JSONRawString,
) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{"type": "object"}, nil
	}
	schema, err := jsonutil.DecodeJSONStringRawInto[map[string]any](raw)
	if err != nil {
		return nil, err
	}
	if len(schema) == 0 {
		return map[string]any{"type": "object"}, nil
	}
	return schema, nil
}
