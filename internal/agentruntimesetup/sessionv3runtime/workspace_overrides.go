package sessionv3runtime

import (
	"errors"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

func workspaceArtifactSelections(
	plan composition.CapabilityPlan,
	kind declaration.Type,
) ([]sessionv3Runtime.ArtifactSelection, error) {
	refs := v3ArtifactRefsForCapabilities(plan, kind)
	output := make(
		[]sessionv3Runtime.ArtifactSelection,
		0,
		len(refs),
	)
	for _, ref := range refs {
		output = append(output, sessionv3Runtime.ArtifactSelection{
			RootID:     string(ref.RootID),
			ArtifactID: string(ref.ArtifactID),
		})
	}
	return output, nil
}

func applyWorkspaceArtifactOverrides(
	baseline []sessionv3Runtime.ArtifactSelection,
	overrides []sessionv3Runtime.ArtifactOverride,
) ([]sessionv3Runtime.ArtifactSelection, error) {
	allowed := make(
		map[string]struct{},
		len(baseline),
	)
	for _, selection := range baseline {
		allowed[workspaceArtifactSelectionKey(selection)] = struct{}{}
	}

	output := append(
		[]sessionv3Runtime.ArtifactSelection(nil),
		baseline...,
	)
	replaced := false
	for _, override := range overrides {
		switch override.Mode {
		case sessionv3Runtime.CollectionOverrideInherit:
			continue

		case sessionv3Runtime.CollectionOverrideRemove:
			output = removeWorkspaceArtifactSelection(
				output,
				override.Key,
			)

		case sessionv3Runtime.CollectionOverrideReplace:
			if !replaced {
				output = nil
				replaced = true
			}
			fallthrough

		case sessionv3Runtime.CollectionOverrideAdd:
			if override.Value == nil {
				return nil, errors.New(
					"workspace artifact override has no value",
				)
			}
			if !containsWorkspaceArtifactSelection(
				output,
				*override.Value,
			) {
				output = append(output, *override.Value)
			}

		default:
			return nil, fmt.Errorf(
				"workspace artifact override mode %q is unavailable",
				override.Mode,
			)
		}
	}

	for _, selection := range output {
		if _, found := allowed[workspaceArtifactSelectionKey(selection)]; !found {
			return nil, fmt.Errorf(
				"workspace artifact %q is not an available workspace capability",
				selection.ArtifactID,
			)
		}
	}
	return output, nil
}

func workspaceMCPSelections(
	plan composition.CapabilityPlan,
) ([]sessionv3Runtime.MCPServerSelection, error) {
	refs := v3ArtifactRefsForCapabilities(plan, declaration.TypeMCP)
	output := make(
		[]sessionv3Runtime.MCPServerSelection,
		0,
		len(refs),
	)
	for _, ref := range refs {
		output = append(output, sessionv3Runtime.MCPServerSelection{
			Server: sessionv3Runtime.ArtifactSelection{
				RootID:     string(ref.RootID),
				ArtifactID: string(ref.ArtifactID),
			},
			ToolExposure: sessionv3Runtime.MCPToolExposureAll,
		})
	}
	return output, nil
}

func applyWorkspaceMCPOverrides(
	baseline []sessionv3Runtime.MCPServerSelection,
	overrides []sessionv3Runtime.MCPServerOverride,
) ([]sessionv3Runtime.MCPServerSelection, error) {
	allowed := make(
		map[string]struct{},
		len(baseline),
	)
	for _, selection := range baseline {
		allowed[workspaceMCPSelectionKey(selection)] = struct{}{}
	}

	byKey := make(
		map[string]sessionv3Runtime.MCPServerSelection,
		len(baseline),
	)
	order := make([]string, 0, len(baseline))
	for _, selection := range baseline {
		key := workspaceMCPSelectionKey(selection)
		byKey[key] = cloneWorkspaceMCPSelection(selection)
		order = append(order, key)
	}

	replaced := false
	for _, override := range overrides {
		switch override.Mode {
		case sessionv3Runtime.CollectionOverrideInherit:
			continue

		case sessionv3Runtime.CollectionOverrideRemove:
			delete(byKey, override.Key)
			order = removeWorkspaceSelectionKey(order, override.Key)

		case sessionv3Runtime.CollectionOverrideReplace:
			if !replaced {
				clear(byKey)
				order = nil
				replaced = true
			}
			fallthrough

		case sessionv3Runtime.CollectionOverrideAdd:
			if override.Value == nil {
				return nil, errors.New(
					"workspace mcp override has no value",
				)
			}
			key := workspaceMCPSelectionKey(*override.Value)
			if _, found := byKey[key]; !found {
				order = append(order, key)
			}
			byKey[key] = cloneWorkspaceMCPSelection(*override.Value)

		default:
			return nil, fmt.Errorf(
				"workspace mcp override mode %q is unavailable",
				override.Mode,
			)
		}
	}

	output := make(
		[]sessionv3Runtime.MCPServerSelection,
		0,
		len(order),
	)
	for _, key := range order {
		selection, found := byKey[key]
		if !found {
			continue
		}
		if _, allowedSelection := allowed[key]; !allowedSelection {
			return nil, fmt.Errorf(
				"workspace mcp server %q is not an available workspace capability",
				selection.Server.ArtifactID,
			)
		}
		output = append(output, selection)
	}
	return output, nil
}

func workspaceArtifactRefs(
	values []sessionv3Runtime.ArtifactSelection,
) []artifactModel.ArtifactRef {
	output := make([]artifactModel.ArtifactRef, 0, len(values))
	for _, selection := range values {
		output = append(output, artifactRefForSelection(selection))
	}
	return output
}

func workspaceMCPArtifactRefs(
	values []sessionv3Runtime.MCPServerSelection,
) []artifactModel.ArtifactRef {
	output := make([]artifactModel.ArtifactRef, 0, len(values))
	for _, selection := range values {
		output = append(
			output,
			artifactRefForSelection(selection.Server),
		)
	}
	return output
}

func workspaceArtifactSelectionKey(
	value sessionv3Runtime.ArtifactSelection,
) string {
	return value.RootID + "\x00" + value.ArtifactID
}

func workspaceMCPSelectionKey(
	value sessionv3Runtime.MCPServerSelection,
) string {
	return workspaceArtifactSelectionKey(value.Server)
}

func containsWorkspaceArtifactSelection(
	values []sessionv3Runtime.ArtifactSelection,
	want sessionv3Runtime.ArtifactSelection,
) bool {
	for _, value := range values {
		if workspaceArtifactSelectionKey(value) ==
			workspaceArtifactSelectionKey(want) {
			return true
		}
	}
	return false
}

func removeWorkspaceArtifactSelection(
	values []sessionv3Runtime.ArtifactSelection,
	key string,
) []sessionv3Runtime.ArtifactSelection {
	output := make(
		[]sessionv3Runtime.ArtifactSelection,
		0,
		len(values),
	)
	for _, value := range values {
		if workspaceArtifactSelectionKey(value) != key {
			output = append(output, value)
		}
	}
	return output
}

func removeWorkspaceSelectionKey(
	values []string,
	want string,
) []string {
	output := make([]string, 0, len(values))
	for _, value := range values {
		if value != want {
			output = append(output, value)
		}
	}
	return output
}

func cloneWorkspaceMCPSelection(
	value sessionv3Runtime.MCPServerSelection,
) sessionv3Runtime.MCPServerSelection {
	output := value
	output.SelectedTools = append(
		[]sessionv3Runtime.MCPToolSelection(nil),
		value.SelectedTools...,
	)
	output.Resources = append(
		[]sessionv3Runtime.MCPResourceSelection(nil),
		value.Resources...,
	)
	output.ResourceTemplates = append(
		[]sessionv3Runtime.MCPResourceTemplateSelection(nil),
		value.ResourceTemplates...,
	)
	output.Prompts = append(
		[]sessionv3Runtime.MCPPromptSelection(nil),
		value.Prompts...,
	)
	return output
}
