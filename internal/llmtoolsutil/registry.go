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
	defaultGoRegistry = mustNewGoRegistry(llmtools.WithDefaultCallTimeout(300 * time.Second))
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

	// Web.
	"github.com/flexigpt/llmtools-go/webtool/fetchurl.FetchURL": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsWeb),
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
	"github.com/flexigpt/llmtools-go/texttool/inserttext.InsertText": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/texttool/replacetext.ReplaceText": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/texttool/deletetext.DeleteText": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/texttool/applyunifieddiff.ApplyUnifiedDiff": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: false,
	},

	// Git.
	"github.com/flexigpt/llmtools-go/gittool/status.Status": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/gittool/diff.Diff": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/gittool/log.Log": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/gittool/show.Show": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/gittool/branches.Branches": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/gittool/tags.Tags": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/gittool/createtag.CreateTag": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/gittool/deletetag.DeleteTag": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/gittool/changedfiles.ChangedFiles": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/gittool/listtree.ListTree": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/gittool/readblob.ReadBlob": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/gittool/findrepos.FindRepos": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/gittool/add.Add": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/gittool/reset.Reset": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/gittool/commit.Commit": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/gittool/createbranch.CreateBranch": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/gittool/checkout.Checkout": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/gittool/init.Init": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/gittool/blame.Blame": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/gittool/filehistory.FileHistory": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/gittool/repoinfo.RepoInfo": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/gittool/grep.Grep": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsGit),
		AutoExecReco: true,
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
