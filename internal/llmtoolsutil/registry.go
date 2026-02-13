package llmtoolsutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/llmtools-go"
	llmtoolsgoSpec "github.com/flexigpt/llmtools-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
)

// defaultGoRegistry is a package-level global registry with a 5s timeout.
// It is created during package initialization and panics on failure.
var defaultGoRegistry *llmtools.Registry

func init() {
	defaultGoRegistry = mustNewGoRegistry(llmtools.WithDefaultCallTimeout(5 * time.Second))
}

type LLMToolMeta struct {
	BundleID     bundleitemutils.BundleID
	AutoExecReco bool
}

// LLMToolsGoBuiltinCatalog is the SINGLE source of truth for how llmtools-go builtins
// are represented in this app.
//
// Key: exact Go FuncID string from llmtools-go (stable in your environment).
// Value: bundle routing + auto-exec recommendation.
//
// When llmtools-go returns a tool that isn't mapped here, injectLLMToolsGoBuiltins fails
// with a clear error telling you to add it.
var LLMToolsGoBuiltinCatalog = map[string]LLMToolMeta{
	// FStool.
	"github.com/flexigpt/llmtools-go/fstool/readfile.ReadFile": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/fstool/searchfiles.SearchFiles": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/fstool/writefile.WriteFile": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/fstool/deletefile.DeleteFile": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/fstool/listdirectory.ListDirectory": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/fstool/statpath.StatPath": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/fstool/mimeforpath.MIMEForPath": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/fstool/mimeforextension.MIMEForExtension": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: true,
	},

	// Imagetool.
	"github.com/flexigpt/llmtools-go/imagetool/readimage.ReadImage": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsImage),
		AutoExecReco: true,
	},

	// Exectool (dangerous by default).
	"github.com/flexigpt/llmtools-go/exectool/shellcommand.ShellCommand": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsExec),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/exectool/runscript.RunScript": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsExec),
		AutoExecReco: false,
	},

	// Texttool.
	"github.com/flexigpt/llmtools-go/texttool/readtextrange.ReadTextRange": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/texttool/findtext.FindText": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/texttool/inserttextlines.InsertTextLines": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/texttool/replacetextlines.ReplaceTextLines": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/texttool/deletetextlines.DeleteTextLines": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: false,
	},
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

// fromLLMToolsOutputUnions converts a slice. Cloning and sanitization.
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
			return nil, errors.New("no file item for output file")
		}
	default:
		return nil, errors.New("unknown output kind")
	}
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
