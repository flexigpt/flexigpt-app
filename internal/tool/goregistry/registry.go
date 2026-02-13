package goregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/llmtools-go"
	llmtoolsgoSpec "github.com/flexigpt/llmtools-go/spec"
)

// defaultGoRegistry is a package-level global registry with a 5s timeout.
// It is created during package initialization and panics on failure.
var defaultGoRegistry *llmtools.Registry

func init() {
	defaultGoRegistry = mustNewGoRegistry(llmtools.WithDefaultCallTimeout(5 * time.Second))
}

// mustNewGoRegistry panics if NewGoRegistry fails.
// This is useful for package-level initialization.
func mustNewGoRegistry(opts ...llmtools.RegistryOption) *llmtools.Registry {
	r, err := llmtools.NewBuiltinRegistry(opts...)
	if err != nil {
		panic(fmt.Errorf("failed to create default go registry: %w", err))
	}
	return r
}

func RegisterOutputsToolUsingDefaultGoRegistry[T any](
	tool llmtoolsgoSpec.Tool,
	fn func(context.Context, T) ([]llmtoolsgoSpec.ToolOutputUnion, error),
) error {
	return llmtools.RegisterOutputsTool(defaultGoRegistry, tool, fn)
}

func RegisterTypedAsTextToolUsingDefaultGoRegistry[T, R any](
	tool llmtoolsgoSpec.Tool,
	fn func(context.Context, T) (R, error),
) error {
	return llmtools.RegisterTypedAsTextTool(defaultGoRegistry, tool, fn)
}

func CallUsingDefaultGoRegistry(
	ctx context.Context,
	funcID string,
	args json.RawMessage,
	callOpts ...llmtools.CallOption,
) ([]llmtoolsgoSpec.ToolOutputUnion, error) {
	llmtoolsOutputs, err := defaultGoRegistry.Call(
		ctx,
		llmtoolsgoSpec.FuncID(funcID),
		args,
		callOpts...,
	)
	if err != nil {
		return nil, err
	}
	return fromLLMToolsOutputUnions(llmtoolsOutputs)
}

// fromLLMToolsOutputUnions converts a slice.
func fromLLMToolsOutputUnions(in []llmtoolsgoSpec.ToolOutputUnion) ([]llmtoolsgoSpec.ToolOutputUnion, error) {
	if in == nil {
		return nil, nil
	}

	outs := make([]llmtoolsgoSpec.ToolOutputUnion, 0)
	for i := range in {
		o, err := fromLLMToolsOutputUnion(in[i])
		if err != nil {
			return nil, err
		}
		outs = append(outs, *o)
	}
	return outs, nil
}

func fromLLMToolsOutputUnion(in llmtoolsgoSpec.ToolOutputUnion) (*llmtoolsgoSpec.ToolOutputUnion, error) {
	switch in.Kind {
	case llmtoolsgoSpec.ToolOutputKindNone:
		return &llmtoolsgoSpec.ToolOutputUnion{
			Kind: llmtoolsgoSpec.ToolOutputKindNone,
		}, nil

	case llmtoolsgoSpec.ToolOutputKindText:
		if in.TextItem != nil {
			return &llmtoolsgoSpec.ToolOutputUnion{
				Kind:     llmtoolsgoSpec.ToolOutputKindText,
				TextItem: &llmtoolsgoSpec.ToolOutputText{Text: in.TextItem.Text},
			}, nil
		} else {
			return nil, errors.New("no text item for output text")
		}
	case llmtoolsgoSpec.ToolOutputKindImage:
		if in.ImageItem != nil {
			return &llmtoolsgoSpec.ToolOutputUnion{
				Kind: llmtoolsgoSpec.ToolOutputKindImage,
				ImageItem: &llmtoolsgoSpec.ToolOutputImage{
					Detail: llmtoolsgoSpec.ImageDetail(
						string(in.ImageItem.Detail),
					), // robust to new/unknown detail values
					ImageName: in.ImageItem.ImageName,
					ImageMIME: in.ImageItem.ImageMIME,
					ImageData: in.ImageItem.ImageData,
				},
			}, nil
		} else {
			return nil, errors.New("no image item for output image")
		}

	case llmtoolsgoSpec.ToolOutputKindFile:
		if in.FileItem != nil {
			return &llmtoolsgoSpec.ToolOutputUnion{
				Kind: llmtoolsgoSpec.ToolOutputKindFile,
				FileItem: &llmtoolsgoSpec.ToolOutputFile{
					FileName: in.FileItem.FileName,
					FileMIME: in.FileItem.FileMIME,
					FileData: in.FileItem.FileData,
				},
			}, nil
		} else {
			return nil, errors.New("no image item for output image")
		}
	default:
		return nil, errors.New("unknown output kind")
	}
}
