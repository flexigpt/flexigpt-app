package sessionv3runtime

import (
	"encoding/json"

	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func (b *Binder) skillLifecycleChoices() ([]preparedChoice, error) {
	if b.deps.SkillRuntime == nil {
		return nil, nil
	}
	defs := []struct {
		name   string
		desc   string
		schema string
	}{
		{
			name:   "skills-load",
			desc:   "Load an available skill",
			schema: string(agentskillsRuntimeSpec.SkillsLoadTool().ArgSchema),
		},
		{
			name:   "skills-unload",
			desc:   "Unload an active skill",
			schema: string(agentskillsRuntimeSpec.SkillsUnloadTool().ArgSchema),
		},
		{
			name:   "skills-readresource",
			desc:   "Read a skill resource",
			schema: string(agentskillsRuntimeSpec.SkillsReadResourceTool().ArgSchema),
		},
	}
	if b.deps.SkillRuntime.SupportsRunScript() {
		defs = append(defs, struct {
			name   string
			desc   string
			schema string
		}{name: "skills-runscript", desc: "Run a skill script", schema: string(agentskillsRuntimeSpec.SkillsRunScriptTool().ArgSchema)})
	}
	out := make([]preparedChoice, 0, len(defs))
	for _, d := range defs {
		args := getEmptySchema()
		if d.schema != "" {
			var decoded map[string]any
			if err := json.Unmarshal([]byte(d.schema), &decoded); err == nil && len(decoded) != 0 {
				args = decoded
			}
		}
		choice := inferenceSpec.ToolChoice{
			Type:        inferenceSpec.ToolTypeFunction,
			ID:          "builtin." + d.name,
			Name:        d.name,
			Description: d.desc,
			Arguments:   args,
		}
		call := sessionv3Runtime.ToolCall{
			Name:        d.name,
			ChoiceID:    "builtin." + d.name,
			DisplayName: d.name,
			Description: d.desc,
			Automatic:   true,
			Binding: sessionv3Runtime.ToolBinding{
				Kind:  sessionv3Runtime.ToolBindingKindSkill,
				Skill: &sessionv3Runtime.SkillToolBinding{Name: d.name},
			},
		}
		out = append(out, preparedChoice{Choice: choice, Call: call})
	}
	return out, nil
}
