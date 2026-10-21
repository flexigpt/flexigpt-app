package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/flexigpt/llmtools-go/fstool"
	"github.com/flexigpt/llmtools-go/texttool"

	attachmentRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/attachment"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
)

func (a *App) ApplyUnifiedDiff(
	req *texttool.ApplyUnifiedDiffArgs,
) (*texttool.ApplyUnifiedDiffOut, error) {
	return withRecoveryResp(func() (*texttool.ApplyUnifiedDiffOut, error) {
		if req == nil {
			return nil, errors.New("invalid arguments: nil request received")
		}
		return llmtoolsutil.ApplyUnifiedDiff(context.Background(), *req)
	})
}

func (a *App) OpenURLAsAttachment(
	rawURL string,
) (att *attachmentRuntime.Attachment, err error) {
	return withRecoveryResp(func() (*attachmentRuntime.Attachment, error) {
		return attachmentRuntime.BuildAttachmentForURL(rawURL)
	})
}

// SaveFile handles saving any content to a file.
func (a *App) SaveFile(
	defaultFilename string,
	contentBase64 string,
	additionalFilters []attachmentRuntime.FileFilter,
) error {
	_, err := withRecoveryResp(func() (struct{}, error) {
		return struct{}{}, a.saveFile(defaultFilename, contentBase64, additionalFilters)
	})
	return err
}

// OpenMultipleFilesAsAttachments opens a native file dialog and returns selected file paths.
// When allowMultiple is true, users can pick multiple files; otherwise at most one path is returned.
func (a *App) OpenMultipleFilesAsAttachments(
	allowMultiple bool,
	additionalFilters []attachmentRuntime.FileFilter,
) (attachments []attachmentRuntime.Attachment, err error) {
	return withRecoveryResp(func() ([]attachmentRuntime.Attachment, error) {
		return a.openMultipleFilesAsAttachments(allowMultiple, additionalFilters)
	})
}

// PickDirectory opens a directory picker without walking the selected tree or
// constructing attachment objects. Workspace setup uses this path-only API.
func (a *App) PickDirectory() (path string, err error) {
	return withRecoveryResp(func() (string, error) {
		return a.pickDirectory()
	})
}

// PickFiles opens a file picker without constructing attachment objects.
// Workspace setup uses it when selecting explicit Context files.
func (a *App) PickFiles(allowMultiple bool) (paths []string, err error) {
	return withRecoveryResp(func() ([]string, error) {
		return a.pickFiles(allowMultiple)
	})
}

// OpenDirectoryAsAttachments opens a single directory pick dialog and then does WalkDirectoryWithFiles for fetching max
// no of files.
func (a *App) OpenDirectoryAsAttachments(maxFiles int) (*attachmentRuntime.DirectoryAttachmentsResult, error) {
	return withRecoveryResp(func() (*attachmentRuntime.DirectoryAttachmentsResult, error) {
		return a.openDirectoryAsAttachments(maxFiles)
	})
}

func (a *App) GetPathsAsAttachments(
	paths []string,
	maxFilesPerDir int,
) (*attachmentRuntime.PathAttachmentsResult, error) {
	return withRecoveryResp(func() (*attachmentRuntime.PathAttachmentsResult, error) {
		return a.getPathsAsAttachments(paths, maxFilesPerDir)
	})
}

func (a *App) getPathsAsAttachments(
	paths []string,
	inMaxFilesPerDir int,
) (*attachmentRuntime.PathAttachmentsResult, error) {
	if len(paths) == 0 {
		return nil, errors.New("empty paths received")
	}
	maxFilesPerDir := inMaxFilesPerDir
	if maxFilesPerDir < 1 {
		maxFilesPerDir = 1024
	}

	// Dedupe incoming paths.
	seen := make(map[string]struct{}, len(paths))
	clean := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		clean = append(clean, p)
	}

	out := attachmentRuntime.PathAttachmentsResult{
		FileAttachments: []attachmentRuntime.Attachment{},
		DirAttachments:  []attachmentRuntime.DirectoryAttachmentsResult{},
		Errors:          []string{},
	}

	for _, p := range clean {
		// Use your existing stat utility (consistent behavior with dialogs).
		info, err := llmtoolsutil.StatPath(context.Background(), fstool.StatPathArgs{Path: p})
		if err != nil || info == nil || !info.Exists {
			out.Errors = append(out.Errors, "Cannot access: "+p)
			continue
		}

		if info.IsDir {
			dirRes, derr := a.buildDirectoryAttachments(info.Path, maxFilesPerDir)
			if derr != nil || dirRes == nil {
				out.Errors = append(out.Errors, "Failed to attach folder: "+info.Path)
				continue
			}
			out.DirAttachments = append(out.DirAttachments, *dirRes)
			continue
		}

		att, aerr := attachmentRuntime.BuildAttachmentForFile(context.Background(), &attachmentRuntime.PathInfo{
			Path:    info.Path,
			Name:    info.Name,
			Exists:  info.Exists,
			IsDir:   info.IsDir,
			Size:    info.SizeBytes,
			ModTime: info.ModTime,
		})
		if aerr != nil || att == nil {
			out.Errors = append(out.Errors, "Failed to attach file: "+info.Path)
			continue
		}
		out.FileAttachments = append(out.FileAttachments, *att)
	}
	return &out, nil
}

func (a *App) buildDirectoryAttachments(
	dirPath string,
	maxFiles int,
) (*attachmentRuntime.DirectoryAttachmentsResult, error) {
	walkRes, err := attachmentRuntime.WalkDirectoryWithFiles(a.ctx, dirPath, maxFiles)
	if err != nil {
		return nil, err
	}

	out := &attachmentRuntime.DirectoryAttachmentsResult{
		DirPath:      walkRes.DirPath,
		Attachments:  make([]attachmentRuntime.Attachment, 0, len(walkRes.Files)),
		OverflowDirs: walkRes.OverflowDirs,
		MaxFiles:     walkRes.MaxFiles,
		TotalSize:    walkRes.TotalSize,
		HasMore:      walkRes.HasMore,
	}

	for _, pi := range walkRes.Files {
		att, buildErr := attachmentRuntime.BuildAttachmentForFile(context.Background(), &pi)
		if buildErr != nil || att == nil {
			continue
		}
		out.Attachments = append(out.Attachments, *att)
	}

	return out, nil
}

func (a *App) saveFile(
	defaultFilename string,
	contentBase64 string,
	additionalFilters []attachmentRuntime.FileFilter,
) error {
	if a.desktop == nil || a.window == nil {
		return errors.New("wails application is not initialized")
	}

	dialog := a.desktop.Dialog.SaveFile().
		AttachToWindow(a.window).
		SetFilename(defaultFilename)
	for _, filter := range additionalFilters {
		dialog.AddFilter(filter.DisplayName, filter.Pattern())
	}
	savePath, err := dialog.PromptForSingleSelection()
	if err != nil {
		return err
	}
	if savePath == "" {
		// User cancelled the dialog.
		return nil
	}

	// Decode base64 content.
	contentBytes, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return err
	}

	// Write the content to the file.
	return os.WriteFile(savePath, contentBytes, 0o600)
}

func (a *App) openMultipleFilesAsAttachments(
	allowMultiple bool,
	additionalFilters []attachmentRuntime.FileFilter,
) (attachments []attachmentRuntime.Attachment, err error) {
	paths, err := a.promptOpenPaths(
		allowMultiple,
		false,
		true,
		additionalFilters,
	)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return []attachmentRuntime.Attachment{}, nil
	}

	attachments = make([]attachmentRuntime.Attachment, 0, len(paths))
	for _, p := range paths {
		path := strings.TrimSpace(p)
		if path == "" {
			slog.Debug("got empty path")
			continue
		}
		// Basic sanity + existence checks.
		pathInfo, err := llmtoolsutil.StatPath(context.Background(), fstool.StatPathArgs{
			Path: path,
		})

		if err != nil || pathInfo == nil {
			slog.Debug("failed to build attachment for file", "path", p, "error", "stat failed")
			continue
		}

		att, attErr := attachmentRuntime.BuildAttachmentForFile(context.Background(), &attachmentRuntime.PathInfo{
			Path:    pathInfo.Path,
			Name:    pathInfo.Name,
			Exists:  pathInfo.Exists,
			IsDir:   pathInfo.IsDir,
			Size:    pathInfo.SizeBytes,
			ModTime: pathInfo.ModTime,
		})
		if attErr != nil || att == nil {
			slog.Debug("failed to build attachment for file", "path", p, "error", attErr)
			continue
		}
		attachments = append(attachments, *att)
	}

	return attachments, nil
}

func (a *App) pickDirectory() (string, error) {
	paths, err := a.promptOpenPaths(false, true, true, nil)
	if err != nil {
		return "", err
	}

	if len(paths) == 0 {
		return "", nil
	}
	return strings.TrimSpace(paths[0]), nil
}

func (a *App) pickFiles(allowMultiple bool) ([]string, error) {
	paths, err := a.promptOpenPaths(allowMultiple, false, true, nil)
	if err != nil {
		return nil, err
	}

	output := make([]string, 0, len(paths))
	for _, path := range paths {
		if path = strings.TrimSpace(path); path != "" {
			output = append(output, path)
		}
	}
	return output, nil
}

func (a *App) openDirectoryAsAttachments(maxFiles int) (*attachmentRuntime.DirectoryAttachmentsResult, error) {
	paths, err := a.promptOpenPaths(false, true, false, nil)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		// User cancelled; do not walk an empty path.
		return &attachmentRuntime.DirectoryAttachmentsResult{}, nil
	}
	dirPath := paths[0]
	walkRes, err := attachmentRuntime.WalkDirectoryWithFiles(a.ctx, dirPath, maxFiles)
	if err != nil {
		return nil, err
	}
	out := &attachmentRuntime.DirectoryAttachmentsResult{
		DirPath:      walkRes.DirPath,
		Attachments:  make([]attachmentRuntime.Attachment, 0, len(walkRes.Files)),
		OverflowDirs: walkRes.OverflowDirs,
		MaxFiles:     walkRes.MaxFiles,
		TotalSize:    walkRes.TotalSize,
		HasMore:      walkRes.HasMore,
	}
	for _, pi := range walkRes.Files {
		att, buildErr := attachmentRuntime.BuildAttachmentForFile(context.Background(), &pi)
		if buildErr != nil || att == nil {
			slog.Debug("failed to build attachment for directory file",
				"path", pi.Path,
				"error", buildErr,
			)
			continue
		}
		out.Attachments = append(out.Attachments, *att)
	}
	return out, nil
}

// promptOpenPaths returns paths only; attachment construction stays separate.
// Do not add default filters: macOS must allow unknown extensions and dotfiles.
func (a *App) promptOpenPaths(
	allowMultiple bool,
	chooseDirectories bool,
	showHiddenFiles bool,
	filters []attachmentRuntime.FileFilter,
) ([]string, error) {
	if a.desktop == nil || a.window == nil {
		return nil, errors.New("wails application is not initialized")
	}

	dialog := a.desktop.Dialog.OpenFile().
		AttachToWindow(a.window).
		CanChooseFiles(!chooseDirectories).
		CanChooseDirectories(chooseDirectories).
		CanCreateDirectories(false).
		ShowHiddenFiles(showHiddenFiles)
	for _, filter := range filters {
		dialog.AddFilter(filter.DisplayName, filter.Pattern())
	}

	if allowMultiple {
		paths, err := dialog.PromptForMultipleSelection()
		if err != nil {
			return nil, err
		}
		if paths == nil {
			return []string{}, nil
		}
		return paths, nil
	}

	path, err := dialog.PromptForSingleSelection()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return []string{}, nil
	}
	return []string{path}, nil
}
