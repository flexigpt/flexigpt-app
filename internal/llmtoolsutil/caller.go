package llmtoolsutil

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/flexigpt/llmtools-go"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

func CallUsingRegistry(
	ctx context.Context,
	reg *llmtools.Registry,
	funcID string,
	args json.RawMessage,
	callOpts ...llmtools.CallOption,
) ([]llmtoolsSpec.ToolOutputUnion, error) {
	if reg == nil {
		return nil, errors.New("nil registry")
	}
	llmtoolsOutputs, err := reg.Call(
		ctx,
		llmtoolsSpec.FuncID(funcID),
		args,
		callOpts...,
	)
	if err != nil {
		return nil, err
	}
	return fromLLMToolsOutputUnions(llmtoolsOutputs)
}

func CallUsingDefaultGoRegistry(
	ctx context.Context,
	funcID string,
	args json.RawMessage,
	callOpts ...llmtools.CallOption,
) ([]llmtoolsSpec.ToolOutputUnion, error) {
	llmtoolsOutputs, err := defaultGoRegistry.Call(
		ctx,
		llmtoolsSpec.FuncID(funcID),
		args,
		callOpts...,
	)
	if err != nil {
		return nil, err
	}
	return fromLLMToolsOutputUnions(llmtoolsOutputs)
}

// fromLLMToolsOutputUnions converts a slice. Cloning and sanitization.
func fromLLMToolsOutputUnions(in []llmtoolsSpec.ToolOutputUnion) ([]llmtoolsSpec.ToolOutputUnion, error) {
	if in == nil {
		return nil, nil
	}

	outs := make([]llmtoolsSpec.ToolOutputUnion, 0)
	for i := range in {
		o, err := fromLLMToolsOutputUnion(in[i])
		if err != nil {
			return nil, err
		}
		outs = append(outs, *o)
	}
	return outs, nil
}

func fromLLMToolsOutputUnion(in llmtoolsSpec.ToolOutputUnion) (*llmtoolsSpec.ToolOutputUnion, error) {
	switch in.Kind {
	case llmtoolsSpec.ToolOutputKindNone:
		return &llmtoolsSpec.ToolOutputUnion{
			Kind: llmtoolsSpec.ToolOutputKindNone,
		}, nil

	case llmtoolsSpec.ToolOutputKindText:
		if in.TextItem != nil {
			return &llmtoolsSpec.ToolOutputUnion{
				Kind:     llmtoolsSpec.ToolOutputKindText,
				TextItem: &llmtoolsSpec.ToolOutputText{Text: in.TextItem.Text},
			}, nil
		} else {
			return nil, errors.New("no text item for output text")
		}
	case llmtoolsSpec.ToolOutputKindImage:
		if in.ImageItem != nil {
			return &llmtoolsSpec.ToolOutputUnion{
				Kind: llmtoolsSpec.ToolOutputKindImage,
				ImageItem: &llmtoolsSpec.ToolOutputImage{
					Detail: llmtoolsSpec.ImageDetail(
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

	case llmtoolsSpec.ToolOutputKindFile:
		if in.FileItem != nil {
			return &llmtoolsSpec.ToolOutputUnion{
				Kind: llmtoolsSpec.ToolOutputKindFile,
				FileItem: &llmtoolsSpec.ToolOutputFile{
					FileName: in.FileItem.FileName,
					FileMIME: in.FileItem.FileMIME,
					FileData: in.FileItem.FileData,
				},
			}, nil
		} else {
			return nil, errors.New("no file item for output file")
		}
	default:
		return nil, errors.New("unknown output kind")
	}
}
