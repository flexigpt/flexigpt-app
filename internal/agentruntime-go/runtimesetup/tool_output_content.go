package runtimesetup

import (
	"encoding/base64"
	"strings"

	inferenceSpec "github.com/flexigpt/inference-go/spec"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
)

// toolOutputItemsFromLLMToolOutputs adapts executor-originated llmtools
// output to the canonical inference-go ToolOutput content representation.
func toolOutputItemsFromLLMToolOutputs(
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

// toolOutputItemsFromMCPContents keeps MCP-specific adaptation in runtime
// setup, while exposing only inference-go content types to Session state.
func toolOutputItemsFromMCPContents(
	contents []mcpServer.MCPContent,
) []sessionRuntime.InferenceToolOutputItem {
	result := make(
		[]sessionRuntime.InferenceToolOutputItem,
		0,
		len(contents),
	)

	for _, content := range contents {
		switch content.Type {
		case mcpServer.MCPContentTypeText:
			result = append(result, inferenceSpec.ToolOutputItemUnion{
				Kind: inferenceSpec.ContentItemKindText,
				TextItem: &inferenceSpec.ContentItemText{
					Text: content.Text,
				},
			})

		case mcpServer.MCPContentTypeImage:
			result = append(result, inferenceSpec.ToolOutputItemUnion{
				Kind: inferenceSpec.ContentItemKindImage,
				ImageItem: &inferenceSpec.ContentItemImage{
					ImageName: content.Name,
					ImageMIME: content.MIMEType,
					ImageData: base64.StdEncoding.EncodeToString(content.Data),
				},
			})

		case mcpServer.MCPContentTypeAudio:
			result = append(result, inferenceSpec.ToolOutputItemUnion{
				Kind: inferenceSpec.ContentItemKindFile,
				FileItem: &inferenceSpec.ContentItemFile{
					FileName: firstToolOutputName(content.Name, "audio"),
					FileMIME: content.MIMEType,
					FileData: base64.StdEncoding.EncodeToString(content.Data),
				},
			})

		case mcpServer.MCPContentTypeResource:
			if content.Resource == nil {
				continue
			}
			if strings.TrimSpace(content.Resource.Text) != "" {
				result = append(result, inferenceSpec.ToolOutputItemUnion{
					Kind: inferenceSpec.ContentItemKindText,
					TextItem: &inferenceSpec.ContentItemText{
						Text: content.Resource.Text,
					},
				})
			}
			if len(content.Resource.Blob) != 0 {
				result = append(result, inferenceSpec.ToolOutputItemUnion{
					Kind: inferenceSpec.ContentItemKindFile,
					FileItem: &inferenceSpec.ContentItemFile{
						FileName: firstToolOutputName(
							content.Name,
							content.Resource.URI,
							"resource",
						),
						FileMIME: content.Resource.MIMEType,
						FileData: base64.StdEncoding.EncodeToString(
							content.Resource.Blob,
						),
					},
				})
			}

		case mcpServer.MCPContentTypeResourceLink:
			text := strings.TrimSpace(content.Name + " " + content.URI)
			if text != "" {
				result = append(result, inferenceSpec.ToolOutputItemUnion{
					Kind: inferenceSpec.ContentItemKindText,
					TextItem: &inferenceSpec.ContentItemText{
						Text: text,
					},
				})
			}
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func firstToolOutputName(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "file"
}
