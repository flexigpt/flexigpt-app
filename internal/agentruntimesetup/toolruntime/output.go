package toolruntime

import (
	"encoding/json"
	"strings"

	inferenceSpec "github.com/flexigpt/inference-go/spec"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// OutputItemsFromLLMToolOutputs adapts Tool Runtime output into the
// provider-neutral inference Tool-output representation stored by Session.
func OutputItemsFromLLMToolOutputs(
	outputs []llmtoolsSpec.ToolOutputUnion,
) []sessionRuntime.InferenceToolOutputItem {
	result := make(
		[]sessionRuntime.InferenceToolOutputItem,
		0,
		len(outputs),
	)

	for _, output := range outputs {
		switch output.Kind {
		case llmtoolsSpec.ToolOutputKindText:
			if output.TextItem == nil {
				continue
			}
			result = append(result, inferenceSpec.ToolOutputItemUnion{
				Kind: inferenceSpec.ContentItemKindText,
				TextItem: &inferenceSpec.ContentItemText{
					Text: output.TextItem.Text,
				},
			})

		case llmtoolsSpec.ToolOutputKindImage:
			if output.ImageItem == nil {
				continue
			}
			result = append(result, inferenceSpec.ToolOutputItemUnion{
				Kind: inferenceSpec.ContentItemKindImage,
				ImageItem: &inferenceSpec.ContentItemImage{
					ImageName: output.ImageItem.ImageName,
					ImageMIME: output.ImageItem.ImageMIME,
					ImageData: output.ImageItem.ImageData,
				},
			})

		case llmtoolsSpec.ToolOutputKindFile:
			if output.FileItem == nil {
				continue
			}
			result = append(result, inferenceSpec.ToolOutputItemUnion{
				Kind: inferenceSpec.ContentItemKindFile,
				FileItem: &inferenceSpec.ContentItemFile{
					FileName: output.FileItem.FileName,
					FileMIME: output.FileItem.FileMIME,
					FileData: output.FileItem.FileData,
				},
			})
		default:
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// ToolOutputsText produces a durable text projection suitable for runtimes
// that do not retain binary Tool-output content in their public model.
func ToolOutputsText(
	outputs []llmtoolsSpec.ToolOutputUnion,
) string {
	parts := make([]string, 0, len(outputs))

	for _, output := range outputs {
		switch output.Kind {
		case llmtoolsSpec.ToolOutputKindText:
			if output.TextItem != nil &&
				strings.TrimSpace(output.TextItem.Text) != "" {
				parts = append(parts, output.TextItem.Text)
			}

		case llmtoolsSpec.ToolOutputKindImage:
			parts = append(parts, "[tool image omitted]")

		case llmtoolsSpec.ToolOutputKindFile:
			parts = append(parts, "[tool file omitted]")

		default:
			parts = append(parts, "[tool output omitted]")
		}
	}

	return strings.Join(parts, "\n\n")
}

// CanonicalToolOutputs converts Tool Runtime output to canonical JSON before it
// crosses into a durable Session runtime model.
func CanonicalToolOutputs(
	outputs []llmtoolsSpec.ToolOutputUnion,
) (json.RawMessage, error) {
	raw, err := jsonutil.EncodeToJSONRaw(outputs)
	if err != nil {
		return nil, err
	}

	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(canonical), nil
}
