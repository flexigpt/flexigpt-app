package toolruntime

import (
	"encoding/json"
	"strings"

	inferenceSpec "github.com/flexigpt/inference-go/spec"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// ToolOutputSemantics is the shared provider-neutral durable representation
// of native Go and Skill Tool output. Session V2 and V3 can project it into
// their own orchestration records without independently adapting rich output.
type ToolOutputSemantics struct {
	Text       string
	Contents   []inferenceSpec.ToolOutputItemUnion
	Structured json.RawMessage
}

func AdaptToolOutputs(
	outputs []llmtoolsSpec.ToolOutputUnion,
) (ToolOutputSemantics, error) {
	structured, err := CanonicalToolOutputs(outputs)
	if err != nil {
		return ToolOutputSemantics{}, err
	}
	return ToolOutputSemantics{
		Text:       ToolOutputsText(outputs),
		Contents:   OutputItemsFromLLMToolOutputs(outputs),
		Structured: structured,
	}, nil
}

// OutputItemsFromLLMToolOutputs adapts Tool Runtime output into the
// provider-neutral inference Tool-output representation stored by Session.
func OutputItemsFromLLMToolOutputs(
	outputs []llmtoolsSpec.ToolOutputUnion,
) []inferenceSpec.ToolOutputItemUnion {
	result := make(
		[]inferenceSpec.ToolOutputItemUnion,
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
