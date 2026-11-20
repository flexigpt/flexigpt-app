package sessionv3runtime

import (
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

// applySpecificCollections applies Session and invocation collection intent
// after Agent and Workspace inherited contributions were prepared. It keeps
// explicit replacement, empty clearing, and removals durable and separate
// from the effective resulting selections.
func applySpecificCollections(
	inherited *sessionv3Runtime.SessionDefaults,
	specific sessionv3Runtime.SessionDefaults,
) {
	if inherited == nil {
		return
	}

	inherited.InstructionArtifacts = overlaySelections(
		inherited.InstructionArtifacts,
		specific.InstructionArtifacts,
		specific.CollectionPolicies.InstructionArtifacts,
	)
	inherited.ToolSelections = overlayToolSelections(
		inherited.ToolSelections,
		specific.ToolSelections,
		specific.CollectionPolicies.ToolSelections,
	)
	inherited.Skills = overlaySelections(
		inherited.Skills,
		specific.Skills,
		specific.CollectionPolicies.Skills,
	)
	inherited.ActiveSkills = overlaySelections(
		inherited.ActiveSkills,
		specific.ActiveSkills,
		specific.CollectionPolicies.ActiveSkills,
	)
	inherited.MCP = overlayMCPSelections(
		inherited.MCP,
		specific.MCP,
		specific.CollectionPolicies.MCP,
	)
	inherited.SystemInstructions = overlayInstructions(
		inherited.SystemInstructions,
		specific.SystemInstructions,
		specific.CollectionPolicies.SystemInstructions,
	)
	inherited.CollectionPolicies = specific.CollectionPolicies
}

func collectionMode(
	value sessionv3Runtime.CollectionPolicy,
) sessionv3Runtime.CollectionOverrideMode {
	switch value.Mode {
	case sessionv3Runtime.CollectionOverrideReplace:
		return sessionv3Runtime.CollectionOverrideReplace
	case sessionv3Runtime.CollectionOverrideAdd:
		return sessionv3Runtime.CollectionOverrideAdd
	default:
		// Existing documents predate policy metadata. Preserve their historical
		// additive behavior rather than treating an omitted policy as a clear.
		return sessionv3Runtime.CollectionOverrideAdd
	}
}

func removedCollectionKeys(
	value sessionv3Runtime.CollectionPolicy,
) map[string]struct{} {
	output := make(map[string]struct{}, len(value.Removed))
	for _, key := range value.Removed {
		if key = strings.TrimSpace(key); key != "" {
			output[key] = struct{}{}
		}
	}
	return output
}

func overlaySelections(
	inherited []sessionv3Runtime.ArtifactSelection,
	specific []sessionv3Runtime.ArtifactSelection,
	policy sessionv3Runtime.CollectionPolicy,
) []sessionv3Runtime.ArtifactSelection {
	removed := removedCollectionKeys(policy)
	output := make([]sessionv3Runtime.ArtifactSelection, 0)
	if collectionMode(policy) != sessionv3Runtime.CollectionOverrideReplace {
		for _, value := range inherited {
			if _, removedValue := removed[selectionKey(value)]; !removedValue {
				output = append(output, value)
			}
		}
	}
	for _, value := range specific {
		if _, removedValue := removed[selectionKey(value)]; removedValue {
			continue
		}
		output = replaceSelection(output, value)
	}
	return output
}

func overlayToolSelections(
	inherited []sessionv3Runtime.ToolSelection,
	specific []sessionv3Runtime.ToolSelection,
	policy sessionv3Runtime.CollectionPolicy,
) []sessionv3Runtime.ToolSelection {
	removed := removedCollectionKeys(policy)
	output := make([]sessionv3Runtime.ToolSelection, 0)
	if collectionMode(policy) != sessionv3Runtime.CollectionOverrideReplace {
		for _, value := range inherited {
			if _, removedValue := removed[selectionKey(value.Artifact)]; !removedValue {
				output = append(output, sessionv3Runtime.CloneToolSelection(value))
			}
		}
	}
	for _, value := range specific {
		if _, removedValue := removed[selectionKey(value.Artifact)]; removedValue {
			continue
		}
		output = replaceToolSelection(output, value)
	}
	return output
}

func overlayMCPSelections(
	inherited []sessionv3Runtime.MCPServerSelection,
	specific []sessionv3Runtime.MCPServerSelection,
	policy sessionv3Runtime.CollectionPolicy,
) []sessionv3Runtime.MCPServerSelection {
	removed := removedCollectionKeys(policy)
	output := make([]sessionv3Runtime.MCPServerSelection, 0)
	if collectionMode(policy) != sessionv3Runtime.CollectionOverrideReplace {
		for _, value := range inherited {
			if _, removedValue := removed[selectionKey(value.Server)]; !removedValue {
				output = append(output, cloneWorkspaceMCPSelection(value))
			}
		}
	}
	for _, value := range specific {
		if _, removedValue := removed[selectionKey(value.Server)]; removedValue {
			continue
		}
		output = replaceMCPSelection(output, value)
	}
	return output
}

func overlayInstructions(
	inherited []sessionv3Runtime.Instruction,
	specific []sessionv3Runtime.Instruction,
	policy sessionv3Runtime.CollectionPolicy,
) []sessionv3Runtime.Instruction {
	removed := removedCollectionKeys(policy)
	output := make([]sessionv3Runtime.Instruction, 0)
	if collectionMode(policy) != sessionv3Runtime.CollectionOverrideReplace {
		for _, value := range inherited {
			if _, removedValue := removed[instructionKey(value)]; !removedValue {
				output = append(output, value)
			}
		}
	}
	for _, value := range specific {
		if _, removedValue := removed[instructionKey(value)]; removedValue {
			continue
		}
		replaced := false
		for index := range output {
			if instructionKey(output[index]) == instructionKey(value) {
				output[index] = value
				replaced = true
				break
			}
		}
		if !replaced {
			output = append(output, value)
		}
	}
	return output
}

func selectionKey(value sessionv3Runtime.ArtifactSelection) string {
	return value.RootID + "\x00" + value.ArtifactID
}

func instructionKey(value sessionv3Runtime.Instruction) string {
	if strings.TrimSpace(value.ID) != "" {
		return "id:" + value.ID
	}
	return "content:" + value.Content
}

func replaceSelection(
	values []sessionv3Runtime.ArtifactSelection,
	value sessionv3Runtime.ArtifactSelection,
) []sessionv3Runtime.ArtifactSelection {
	for index := range values {
		if selectionKey(values[index]) == selectionKey(value) {
			values[index] = value
			return values
		}
	}
	return append(values, value)
}

func replaceToolSelection(
	values []sessionv3Runtime.ToolSelection,
	value sessionv3Runtime.ToolSelection,
) []sessionv3Runtime.ToolSelection {
	for index := range values {
		if selectionKey(values[index].Artifact) == selectionKey(value.Artifact) {
			values[index] = sessionv3Runtime.CloneToolSelection(value)
			return values
		}
	}
	return append(values, sessionv3Runtime.CloneToolSelection(value))
}

func replaceMCPSelection(
	values []sessionv3Runtime.MCPServerSelection,
	value sessionv3Runtime.MCPServerSelection,
) []sessionv3Runtime.MCPServerSelection {
	for index := range values {
		if selectionKey(values[index].Server) == selectionKey(value.Server) {
			values[index] = cloneWorkspaceMCPSelection(value)
			return values
		}
	}
	return append(values, cloneWorkspaceMCPSelection(value))
}
