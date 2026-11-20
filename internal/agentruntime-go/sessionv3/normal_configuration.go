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
	if override.CollectionPolicies != nil {
		output.CollectionPolicies = cloneSessionCollectionPolicies(
			*override.CollectionPolicies,
		)
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
		output.CollectionPolicies.SystemInstructions.Mode = CollectionOverrideReplace
	}
	if override.InstructionArtifacts != nil {
		output.InstructionArtifacts = append(
			[]ArtifactSelection(nil),
			(*override.InstructionArtifacts)...,
		)
		output.CollectionPolicies.InstructionArtifacts.Mode = CollectionOverrideReplace
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
	if override.ToolSelections != nil {
		output.ToolSelections = cloneToolSelections(
			*override.ToolSelections,
		)
		output.CollectionPolicies.ToolSelections.Mode = CollectionOverrideReplace
	}
	if override.Skills != nil {
		output.Skills = append(
			[]ArtifactSelection(nil),
			(*override.Skills)...,
		)
		output.CollectionPolicies.Skills.Mode = CollectionOverrideReplace
	}
	if override.ActiveSkills != nil {
		output.ActiveSkills = append(
			[]ArtifactSelection(nil),
			(*override.ActiveSkills)...,
		)
		output.CollectionPolicies.ActiveSkills.Mode = CollectionOverrideReplace
	}
	if override.MCP != nil {
		output.MCP = cloneSessionDefaults(SessionDefaults{
			MCP: *override.MCP,
		}).MCP
		output.CollectionPolicies.MCP.Mode = CollectionOverrideReplace
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
			output.CollectionPolicies.SystemInstructions.Mode = CollectionOverrideReplace
		case SessionDefaultFieldInstructionArtifacts:
			output.InstructionArtifacts = nil
			output.CollectionPolicies.InstructionArtifacts.Mode = CollectionOverrideReplace
		case SessionDefaultFieldTemperature:
			output.Temperature = nil
		case SessionDefaultFieldMetadata:
			output.Metadata = nil
		case SessionDefaultFieldHistory:
			output.History = HistorySelection{}
		case SessionDefaultFieldToolSelections:
			output.ToolSelections = nil
			output.CollectionPolicies.ToolSelections.Mode = CollectionOverrideReplace
		case SessionDefaultFieldSkills:
			output.Skills = nil
			output.CollectionPolicies.Skills.Mode = CollectionOverrideReplace
		case SessionDefaultFieldActiveSkills:
			output.ActiveSkills = nil
			output.CollectionPolicies.ActiveSkills.Mode = CollectionOverrideReplace
		case SessionDefaultFieldMCP:
			output.MCP = nil
			output.CollectionPolicies.MCP.Mode = CollectionOverrideReplace
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

// EffectiveToolSelections returns the one canonical typed Tool collection.
func EffectiveToolSelections(
	defaults SessionDefaults,
) []ToolSelection {
	return cloneToolSelections(defaults.ToolSelections)
}

func cloneToolSelections(
	values []ToolSelection,
) []ToolSelection {
	if len(values) == 0 {
		return nil
	}
	output := make([]ToolSelection, len(values))
	for index := range values {
		output[index] = CloneToolSelection(values[index])
	}
	return output
}

func CloneToolSelection(
	value ToolSelection,
) ToolSelection {
	output := value
	output.UserArguments = value.UserArguments.Clone()
	if value.AutoExecute != nil {
		autoExecute := *value.AutoExecute
		output.AutoExecute = &autoExecute
	}
	return output
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

func cloneSessionCollectionPolicies(
	value SessionCollectionPolicies,
) SessionCollectionPolicies {
	output := value
	output.SystemInstructions.Removed = append(
		[]string(nil),
		value.SystemInstructions.Removed...,
	)
	output.InstructionArtifacts.Removed = append(
		[]string(nil),
		value.InstructionArtifacts.Removed...,
	)
	output.ToolSelections.Removed = append(
		[]string(nil),
		value.ToolSelections.Removed...,
	)
	output.Skills.Removed = append([]string(nil), value.Skills.Removed...)
	output.ActiveSkills.Removed = append(
		[]string(nil),
		value.ActiveSkills.Removed...,
	)
	output.MCP.Removed = append([]string(nil), value.MCP.Removed...)
	return output
}
