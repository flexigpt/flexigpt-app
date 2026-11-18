package mcpruntime

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// MCPToolOutputSemantics is the common provider-neutral representation of an
// MCP response. Session V2 and V3 retain their own orchestration state but
// share this domain conversion for text, structured output, images, and files.
type MCPToolOutputSemantics struct {
	Text       string
	Contents   []inferenceSpec.ToolOutputItemUnion
	Structured json.RawMessage
}

func AdaptMCPToolOutput(
	contents []mcpServer.MCPContent,
	structuredContent any,
) (MCPToolOutputSemantics, error) {
	raw, err := jsonutil.EncodeToJSONRaw(struct {
		Content           []mcpServer.MCPContent `json:"content"`
		StructuredContent any                    `json:"structuredContent,omitempty"`
	}{
		Content:           contents,
		StructuredContent: structuredContent,
	})
	if err != nil {
		return MCPToolOutputSemantics{}, err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return MCPToolOutputSemantics{}, err
	}

	text := MCPContentsText(contents)
	if strings.TrimSpace(text) == "" {
		text = string(canonical)
	}
	return MCPToolOutputSemantics{
		Text:       text,
		Contents:   ToolOutputItemsFromMCPContents(contents),
		Structured: json.RawMessage(canonical),
	}, nil
}

// ToolOutputItemsFromMCPContents keeps MCP-specific content adaptation in the
// MCP runtime boundary while exposing only provider-neutral inference content.
func ToolOutputItemsFromMCPContents(
	contents []mcpServer.MCPContent,
) []inferenceSpec.ToolOutputItemUnion {
	result := make(
		[]inferenceSpec.ToolOutputItemUnion,
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

func MCPContentsText(contents []mcpServer.MCPContent) string {
	parts := make([]string, 0, len(contents))
	for _, content := range contents {
		switch content.Type {
		case mcpServer.MCPContentTypeText:
			if strings.TrimSpace(content.Text) != "" {
				parts = append(parts, content.Text)
			}
		case mcpServer.MCPContentTypeResource:
			if content.Resource == nil {
				continue
			}
			if strings.TrimSpace(content.Resource.Text) != "" {
				parts = append(parts, content.Resource.Text)
			} else if strings.TrimSpace(content.Resource.URI) != "" {
				parts = append(parts, "[MCP resource omitted: "+content.Resource.URI+"]")
			}
		case mcpServer.MCPContentTypeResourceLink:
			if value := strings.TrimSpace(content.Name + " " + content.URI); value != "" {
				parts = append(parts, value)
			}
		case mcpServer.MCPContentTypeImage:
			parts = append(parts, "[MCP image content]")
		case mcpServer.MCPContentTypeAudio:
			parts = append(parts, "[MCP audio content]")
		default:
		}
	}
	return strings.Join(parts, "\n\n")
}

func firstToolOutputName(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "file"
}
