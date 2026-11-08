package sessionv3

import "errors"

func resolveSessionInvocationDefaults(
	base SessionDefaults,
	override *SessionDefaultsOverride,
) (SessionDefaults, error) {
	output := cloneSessionDefaults(base)
	if override == nil {
		return output, nil
	}

	if override.Model != nil {
		output.Model = cloneSessionDefaultSelection(override.Model)
	}
	if override.ModelOverrides != nil {
		output.ModelOverrides = MergeModelDefaultsOverride(
			output.ModelOverrides,
			*override.ModelOverrides,
		)
	}
	if override.Agent != nil {
		output.Agent = cloneSessionDefaultSelection(override.Agent)
	}
	if override.Workspace != nil {
		output.Workspace = cloneSessionDefaultSelection(override.Workspace)
	}
	if override.WorkspaceOverrides != nil {
		output.WorkspaceOverrides = *override.WorkspaceOverrides
	}
	if override.SystemInstructions != nil {
		output.SystemInstructions = append(
			[]Instruction(nil),
			(*override.SystemInstructions)...,
		)
	}
	if override.InstructionArtifacts != nil {
		output.InstructionArtifacts = append(
			[]ArtifactSelection(nil),
			(*override.InstructionArtifacts)...,
		)
	}
	if override.Temperature != nil {
		value := *override.Temperature
		output.Temperature = &value
	}
	if override.Metadata != nil {
		output.Metadata = override.Metadata.Clone()
	}
	if override.History != nil {
		value := *override.History
		output.History = value
	}
	if override.Tools != nil {
		output.Tools = append(
			[]ArtifactSelection(nil),
			(*override.Tools)...,
		)
	}
	if override.Skills != nil {
		output.Skills = append(
			[]ArtifactSelection(nil),
			(*override.Skills)...,
		)
	}
	if override.ActiveSkills != nil {
		output.ActiveSkills = append(
			[]ArtifactSelection(nil),
			(*override.ActiveSkills)...,
		)
	}
	if override.MCP != nil {
		output.MCP = cloneSessionDefaults(SessionDefaults{
			MCP: *override.MCP,
		}).MCP
	}

	for _, field := range override.Clear {
		switch field {
		case SessionDefaultFieldModel:
			output.Model = nil
		case SessionDefaultFieldModelOverrides:
			output.ModelOverrides = ModelDefaultsOverride{}
		case SessionDefaultFieldAgent:
			output.Agent = nil
		case SessionDefaultFieldWorkspace:
			output.Workspace = nil
		case SessionDefaultFieldWorkspaceOverrides:
			output.WorkspaceOverrides = WorkspaceSelectionOverrides{}
		case SessionDefaultFieldSystemInstructions:
			output.SystemInstructions = nil
		case SessionDefaultFieldInstructionArtifacts:
			output.InstructionArtifacts = nil
		case SessionDefaultFieldTemperature:
			output.Temperature = nil
		case SessionDefaultFieldMetadata:
			output.Metadata = nil
		case SessionDefaultFieldHistory:
			output.History = HistorySelection{}
		case SessionDefaultFieldTools:
			output.Tools = nil
		case SessionDefaultFieldSkills:
			output.Skills = nil
		case SessionDefaultFieldActiveSkills:
			output.ActiveSkills = nil
		case SessionDefaultFieldMCP:
			output.MCP = nil
		default:
			return SessionDefaults{}, errors.New(
				"session invocation default clear field is invalid",
			)
		}
	}

	if !output.Metadata.Valid() {
		return SessionDefaults{}, errors.New(
			"session invocation metadata is not canonical json",
		)
	}
	if err := validateDefaults(output); err != nil {
		return SessionDefaults{}, err
	}
	return output, nil
}

func cloneSessionDefaultSelection(
	value *ArtifactSelection,
) *ArtifactSelection {
	if value == nil {
		return nil
	}
	output := *value
	return &output
}

func cloneInvocationDefaults(
	value *SessionDefaults,
) *SessionDefaults {
	if value == nil {
		return nil
	}
	output := cloneSessionDefaults(*value)
	return &output
}
