package llmtoolsutil

import (
	"context"

	"github.com/flexigpt/llmtools-go/exectool"
	"github.com/flexigpt/llmtools-go/fstool"
	"github.com/flexigpt/llmtools-go/gittool"
	"github.com/flexigpt/llmtools-go/imagetool"
	"github.com/flexigpt/llmtools-go/texttool"
	"github.com/flexigpt/llmtools-go/webtool"

	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

func ReadFile(ctx context.Context, args fstool.ReadFileArgs) ([]llmtoolsSpec.ToolOutputUnion, error) {
	t, err := fstool.NewFSTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.ReadFile(ctx, args)
}

func SearchFiles(ctx context.Context, args fstool.SearchFilesArgs) (*fstool.SearchFilesOut, error) {
	t, err := fstool.NewFSTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.SearchFiles(ctx, args)
}

func WriteFile(ctx context.Context, args fstool.WriteFileArgs) (*fstool.WriteFileOut, error) {
	t, err := fstool.NewFSTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.WriteFile(ctx, args)
}

func DeleteFile(ctx context.Context, args fstool.DeleteFileArgs) (*fstool.DeleteFileOut, error) {
	t, err := fstool.NewFSTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.DeleteFile(ctx, args)
}

func ListDirectory(ctx context.Context, args fstool.ListDirectoryArgs) (*fstool.ListDirectoryOut, error) {
	t, err := fstool.NewFSTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.ListDirectory(ctx, args)
}

func StatPath(ctx context.Context, args fstool.StatPathArgs) (*fstool.StatPathOut, error) {
	t, err := fstool.NewFSTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.StatPath(ctx, args)
}

func MIMEForPath(ctx context.Context, args fstool.MIMEForPathArgs) (*fstool.MIMEForPathOut, error) {
	t, err := fstool.NewFSTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.MIMEForPath(ctx, args)
}

func MIMEForExtension(ctx context.Context, args fstool.MIMEForExtensionArgs) (*fstool.MIMEForExtensionOut, error) {
	t, err := fstool.NewFSTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.MIMEForExtension(ctx, args)
}

func ReadImage(ctx context.Context, args imagetool.ReadImageArgs) (*imagetool.ReadImageOut, error) {
	t, err := imagetool.NewImageTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.ReadImage(ctx, args)
}

func FetchURL(ctx context.Context, args webtool.FetchURLArgs) ([]llmtoolsSpec.ToolOutputUnion, error) {
	t, err := webtool.NewWebTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.FetchURL(ctx, args)
}

func ShellCommand(ctx context.Context, args exectool.ShellCommandArgs) (*exectool.ShellCommandOut, error) {
	t, err := exectool.NewExecTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.ShellCommand(ctx, args)
}

func RunScript(ctx context.Context, args exectool.RunScriptArgs) (*exectool.RunScriptOut, error) {
	t, err := exectool.NewExecTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.RunScript(ctx, args)
}

func ReadTextRange(ctx context.Context, args texttool.ReadTextRangeArgs) (*texttool.ReadTextRangeOut, error) {
	t, err := texttool.NewTextTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.ReadTextRange(ctx, args)
}

func FindText(ctx context.Context, args texttool.FindTextArgs) (*texttool.FindTextOut, error) {
	t, err := texttool.NewTextTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.FindText(ctx, args)
}

func InsertText(ctx context.Context, args texttool.InsertTextArgs) (*texttool.InsertTextOut, error) {
	t, err := texttool.NewTextTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.InsertText(ctx, args)
}

func ReplaceText(ctx context.Context, args texttool.ReplaceTextArgs) (*texttool.ReplaceTextOut, error) {
	t, err := texttool.NewTextTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.ReplaceText(ctx, args)
}

func DeleteText(ctx context.Context, args texttool.DeleteTextArgs) (*texttool.DeleteTextOut, error) {
	t, err := texttool.NewTextTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.DeleteText(ctx, args)
}

func ApplyUnifiedDiff(ctx context.Context, args texttool.ApplyUnifiedDiffArgs) (*texttool.ApplyUnifiedDiffOut, error) {
	t, err := texttool.NewTextTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.ApplyUnifiedDiff(ctx, args)
}

func Status(ctx context.Context, args gittool.StatusArgs) (*gittool.StatusOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.Status(ctx, args)
}

func Diff(ctx context.Context, args gittool.DiffArgs) (*gittool.DiffOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.Diff(ctx, args)
}

func Log(ctx context.Context, args gittool.LogArgs) (*gittool.LogOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.Log(ctx, args)
}

func Show(ctx context.Context, args gittool.ShowArgs) (*gittool.ShowOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.Show(ctx, args)
}

func Branches(ctx context.Context, args gittool.BranchesArgs) (*gittool.BranchesOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.Branches(ctx, args)
}

func Tags(ctx context.Context, args gittool.TagsArgs) (*gittool.TagsOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.Tags(ctx, args)
}

func CreateTag(ctx context.Context, args gittool.CreateTagArgs) (*gittool.CreateTagOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.CreateTag(ctx, args)
}

func DeleteTag(ctx context.Context, args gittool.DeleteTagArgs) (*gittool.DeleteTagOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.DeleteTag(ctx, args)
}

func ChangedFiles(ctx context.Context, args gittool.ChangedFilesArgs) (*gittool.ChangedFilesOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.ChangedFiles(ctx, args)
}

func ListTree(ctx context.Context, args gittool.ListTreeArgs) (*gittool.ListTreeOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.ListTree(ctx, args)
}

func ReadBlob(ctx context.Context, args gittool.ReadBlobArgs) (*gittool.ReadBlobOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.ReadBlob(ctx, args)
}

func FindRepos(ctx context.Context, args gittool.FindReposArgs) (*gittool.FindReposOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.FindRepos(ctx, args)
}

func Add(ctx context.Context, args gittool.AddArgs) (*gittool.AddOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.Add(ctx, args)
}

func Reset(ctx context.Context, args gittool.ResetArgs) (*gittool.ResetOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.Reset(ctx, args)
}

func Commit(ctx context.Context, args gittool.CommitArgs) (*gittool.CommitOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.Commit(ctx, args)
}

func CreateBranch(ctx context.Context, args gittool.CreateBranchArgs) (*gittool.CreateBranchOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.CreateBranch(ctx, args)
}

func Checkout(ctx context.Context, args gittool.CheckoutArgs) (*gittool.CheckoutOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.Checkout(ctx, args)
}

func Init(ctx context.Context, args gittool.InitArgs) (*gittool.InitOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.Init(ctx, args)
}

func Blame(ctx context.Context, args gittool.BlameArgs) (*gittool.BlameOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.Blame(ctx, args)
}

func FileHistory(ctx context.Context, args gittool.FileHistoryArgs) (*gittool.FileHistoryOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.FileHistory(ctx, args)
}

func RepoInfo(ctx context.Context, args gittool.RepoInfoArgs) (*gittool.RepoInfoOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.RepoInfo(ctx, args)
}

func Grep(ctx context.Context, args gittool.GrepArgs) (*gittool.GrepOut, error) {
	t, err := gittool.NewGitTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.Grep(ctx, args)
}
