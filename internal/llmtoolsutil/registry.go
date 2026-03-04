package llmtoolsutil

import (
	"context"
	"fmt"
	"time"

	"github.com/flexigpt/llmtools-go"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

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
	tool llmtoolsSpec.Tool,
	fn func(context.Context, T) ([]llmtoolsSpec.ToolOutputUnion, error),
) error {
	return llmtools.RegisterOutputsTool(defaultGoRegistry, tool, fn)
}

func RegisterTypedAsTextToolUsingDefaultGoRegistry[T, R any](
	tool llmtoolsSpec.Tool,
	fn func(context.Context, T) (R, error),
) error {
	return llmtools.RegisterTypedAsTextTool(defaultGoRegistry, tool, fn)
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
