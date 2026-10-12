package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

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
	if a.ctx == nil {
		return errors.New("context is not initialized")
	}

	saveDialogOptions := runtime.SaveDialogOptions{
		DefaultFilename: defaultFilename,
		Filters:         dialogFilters(additionalFilters),
	}
	savePath, err := runtime.SaveFileDialog(a.ctx, saveDialogOptions)
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
	if a.ctx == nil {
		return nil, errors.New("context is not initialized")
	}

	opts := runtime.OpenDialogOptions{
		Filters:              dialogFilters(additionalFilters),
		ShowHiddenFiles:      true,
		CanCreateDirectories: false,
	}

	paths := []string{}
	if allowMultiple {
		paths, err = runtime.OpenMultipleFilesDialog(a.ctx, opts)
	} else {
		p, e := runtime.OpenFileDialog(a.ctx, opts)
		if e == nil && p != "" {
			paths = append(paths, p)
		}
		err = e
	}
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
	if a.ctx == nil {
		return "", errors.New("context is not initialized")
	}

	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		ShowHiddenFiles:      true,
		CanCreateDirectories: false,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(path), nil
}

func (a *App) pickFiles(allowMultiple bool) ([]string, error) {
	if a.ctx == nil {
		return nil, errors.New("context is not initialized")
	}

	options := runtime.OpenDialogOptions{
		ShowHiddenFiles:      true,
		CanCreateDirectories: false,
	}

	var (
		paths []string
		err   error
	)
	if allowMultiple {
		paths, err = runtime.OpenMultipleFilesDialog(a.ctx, options)
	} else {
		var path string
		path, err = runtime.OpenFileDialog(a.ctx, options)
		if path != "" {
			paths = []string{path}
		}
	}
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
	if a.ctx == nil {
		return nil, errors.New("context is not initialized")
	}

	dialogOpts := runtime.OpenDialogOptions{
		ShowHiddenFiles:      false,
		CanCreateDirectories: false,
	}

	dirPath, err := runtime.OpenDirectoryDialog(a.ctx, dialogOpts)
	if err != nil {
		return nil, err
	}
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

var defaultRuntimeFilters = func() []runtime.FileFilter {
	runtimeFilters := make([]runtime.FileFilter, 0, len(attachmentRuntime.DefaultFileFilters))
	for idx := range attachmentRuntime.DefaultFileFilters {
		runtimeFilters = append(
			runtimeFilters,
			runtime.FileFilter{
				DisplayName: attachmentRuntime.DefaultFileFilters[idx].DisplayName,
				Pattern:     attachmentRuntime.DefaultFileFilters[idx].Pattern(),
			},
		)
	}
	return runtimeFilters
}()

// dialogFilters returns nil when no explicit filters are provided.
// This is important for macOS: providing filters makes NSOpenPanel/NSSavePanel
// restrict selectable/savable file types, which can hide/disable "unknown"
// extensions (e.g. .bazel/.cmake) and dotfiles depending on type mapping.
func dialogFilters(additionalFilters []attachmentRuntime.FileFilter) []runtime.FileFilter {
	if len(additionalFilters) == 0 {
		return nil
	}
	// Only use the explicit filters requested by the caller.
	return getRuntimeFilters(additionalFilters, false)
}

func getRuntimeFilters(additionalFilters []attachmentRuntime.FileFilter, includeDefault bool) []runtime.FileFilter {
	runtimeFilters := make([]runtime.FileFilter, 0, len(additionalFilters)+len(attachmentRuntime.DefaultFileFilters))

	for idx := range additionalFilters {
		runtimeFilters = append(
			runtimeFilters,
			runtime.FileFilter{
				DisplayName: additionalFilters[idx].DisplayName,
				Pattern:     additionalFilters[idx].Pattern(),
			},
		)
	}
	if includeDefault {
		runtimeFilters = append(runtimeFilters, defaultRuntimeFilters...)
	}
	return runtimeFilters
}
