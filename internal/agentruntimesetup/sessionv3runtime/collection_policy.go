package sessionv3runtime

import (
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

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
