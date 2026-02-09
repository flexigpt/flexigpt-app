package llmtoolsutil

import (
	"context"

	"github.com/flexigpt/llmtools-go/exectool"
	"github.com/flexigpt/llmtools-go/fstool"
	"github.com/flexigpt/llmtools-go/imagetool"
	"github.com/flexigpt/llmtools-go/texttool"

	llmtoolsgoSpec "github.com/flexigpt/llmtools-go/spec"
)

func ReadFile(ctx context.Context, args fstool.ReadFileArgs) ([]llmtoolsgoSpec.ToolStoreOutputUnion, error) {
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

func InsertTextLines(ctx context.Context, args texttool.InsertTextLinesArgs) (*texttool.InsertTextLinesOut, error) {
	t, err := texttool.NewTextTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.InsertTextLines(ctx, args)
}

func ReplaceTextLines(ctx context.Context, args texttool.ReplaceTextLinesArgs) (*texttool.ReplaceTextLinesOut, error) {
	t, err := texttool.NewTextTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.ReplaceTextLines(ctx, args)
}

func DeleteTextLines(ctx context.Context, args texttool.DeleteTextLinesArgs) (*texttool.DeleteTextLinesOut, error) {
	t, err := texttool.NewTextTool()
	if err != nil || t == nil {
		return nil, err
	}
	return t.DeleteTextLines(ctx, args)
}
