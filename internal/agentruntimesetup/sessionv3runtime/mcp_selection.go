package sessionv3runtime

import (
	"errors"
	"fmt"
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// normalizeMCPSelections is the only lowering-time MCP Tool exposure
// normalization boundary. Persisted dispatch plans never contain an empty
// exposure mode.
func normalizeMCPSelections(
	input []sessionv3Runtime.MCPServerSelection,
) ([]sessionv3Runtime.MCPServerSelection, error) {
	if len(input) == 0 {
		return nil, nil
	}

	output := make(
		[]sessionv3Runtime.MCPServerSelection,
		0,
		len(input),
	)
	for index := range input {
		value := jsonutil.CloneJSONInValue(input[index])
		switch value.ToolExposure {
		case "":
			value.ToolExposure = sessionv3Runtime.MCPToolExposureAll

		case sessionv3Runtime.MCPToolExposureAll,
			sessionv3Runtime.MCPToolExposureNone:
			if len(value.SelectedTools) != 0 {
				return nil, fmt.Errorf(
					"mcp selection %d has selected tools with exposure %q",
					index,
					value.ToolExposure,
				)
			}

		case sessionv3Runtime.MCPToolExposureSelected:
			if len(value.SelectedTools) == 0 {
				return nil, fmt.Errorf(
					"mcp selection %d selects no tools",
					index,
				)
			}
			seen := make(map[string]struct{}, len(value.SelectedTools))
			for toolIndex := range value.SelectedTools {
				name := strings.TrimSpace(
					value.SelectedTools[toolIndex].ToolName,
				)
				if name == "" || name != value.SelectedTools[toolIndex].ToolName {
					return nil, errors.New(
						"selected mcp tool name is invalid",
					)
				}
				if _, duplicate := seen[name]; duplicate {
					return nil, fmt.Errorf(
						"selected mcp tool %q is duplicated",
						name,
					)
				}
				seen[name] = struct{}{}
			}

		default:
			return nil, fmt.Errorf(
				"unsupported mcp tool exposure %q",
				value.ToolExposure,
			)
		}
		output = append(output, value)
	}
	return output, nil
}
