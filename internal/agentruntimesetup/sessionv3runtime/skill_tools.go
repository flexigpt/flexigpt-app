package sessionv3runtime

import (
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/toolruntime"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
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
		raw, err := jsonutil.DecodeJSONStringRaw(
			jsonutil.JSONRawString(d.schema),
		)
		if err != nil {
			return nil, err
		}
		args, err := toolruntime.InputSchema(raw)
		if err != nil {
			return nil, err
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
