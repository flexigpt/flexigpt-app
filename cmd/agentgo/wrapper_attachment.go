package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/attachment"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/llmtools-go/fstool"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) OpenURLAsAttachment(
	rawURL string,
) (att *attachment.Attachment, err error) {
	return middleware.WithRecoveryResp(func() (*attachment.Attachment, error) {
		return attachment.BuildAttachmentForURL(rawURL)
	})
}

// SaveFile handles saving any content to a file.
func (a *App) SaveFile(
	defaultFilename string,
	contentBase64 string,
	additionalFilters []attachment.FileFilter,
) error {
	_, err := middleware.WithRecoveryResp(func() (struct{}, error) {
		return struct{}{}, a.saveFile(defaultFilename, contentBase64, additionalFilters)
	})
	return err
}

// OpenMultipleFilesAsAttachments opens a native file dialog and returns selected file paths.
// When allowMultiple is true, users can pick multiple files; otherwise at most one path is returned.
func (a *App) OpenMultipleFilesAsAttachments(
	allowMultiple bool,
	additionalFilters []attachment.FileFilter,
) (attachments []attachment.Attachment, err error) {
	return middleware.WithRecoveryResp(func() ([]attachment.Attachment, error) {
		return a.openMultipleFilesAsAttachments(allowMultiple, additionalFilters)
	})
}

// OpenDirectoryAsAttachments opens a single directory pick dialog and then does WalkDirectoryWithFiles for fetching max
// no of files.
func (a *App) OpenDirectoryAsAttachments(maxFiles int) (*attachment.DirectoryAttachmentsResult, error) {
	return middleware.WithRecoveryResp(func() (*attachment.DirectoryAttachmentsResult, error) {
		return a.openDirectoryAsAttachments(maxFiles)
	})
}

func (a *App) GetPathsAsAttachments(paths []string, maxFilesPerDir int) (*attachment.PathAttachmentsResult, error) {
	return middleware.WithRecoveryResp(func() (*attachment.PathAttachmentsResult, error) {
		return a.getPathsAsAttachments(paths, maxFilesPerDir)
	})
}

func (a *App) getPathsAsAttachments(paths []string, inMaxFilesPerDir int) (*attachment.PathAttachmentsResult, error) {
	if len(paths) == 0 {
		return nil, errors.New("empty paths received")
	}
	maxFilesPerDir := 256
	if inMaxFilesPerDir > 0 && inMaxFilesPerDir <= 256 {
		maxFilesPerDir = inMaxFilesPerDir
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

	out := attachment.PathAttachmentsResult{
		FileAttachments: []attachment.Attachment{},
		DirAttachments:  []attachment.DirectoryAttachmentsResult{},
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

		att, aerr := attachment.BuildAttachmentForFile(context.Background(), &attachment.PathInfo{
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

func (a *App) buildDirectoryAttachments(dirPath string, maxFiles int) (*attachment.DirectoryAttachmentsResult, error) {
	walkRes, err := attachment.WalkDirectoryWithFiles(a.ctx, dirPath, maxFiles)
	if err != nil {
		return nil, err
	}

	out := &attachment.DirectoryAttachmentsResult{
		DirPath:      walkRes.DirPath,
		Attachments:  make([]attachment.Attachment, 0, len(walkRes.Files)),
		OverflowDirs: walkRes.OverflowDirs,
		MaxFiles:     walkRes.MaxFiles,
		TotalSize:    walkRes.TotalSize,
		HasMore:      walkRes.HasMore,
	}

	for _, pi := range walkRes.Files {
		att, buildErr := attachment.BuildAttachmentForFile(context.Background(), &pi)
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
	additionalFilters []attachment.FileFilter,
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
	additionalFilters []attachment.FileFilter,
) (attachments []attachment.Attachment, err error) {
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
		return []attachment.Attachment{}, nil
	}

	attachments = make([]attachment.Attachment, 0, len(paths))
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

		att, attErr := attachment.BuildAttachmentForFile(context.Background(), &attachment.PathInfo{
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

func (a *App) openDirectoryAsAttachments(maxFiles int) (*attachment.DirectoryAttachmentsResult, error) {
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
	walkRes, err := attachment.WalkDirectoryWithFiles(a.ctx, dirPath, maxFiles)
	if err != nil {
		return nil, err
	}
	out := &attachment.DirectoryAttachmentsResult{
		DirPath:      walkRes.DirPath,
		Attachments:  make([]attachment.Attachment, 0, len(walkRes.Files)),
		OverflowDirs: walkRes.OverflowDirs,
		MaxFiles:     walkRes.MaxFiles,
		TotalSize:    walkRes.TotalSize,
		HasMore:      walkRes.HasMore,
	}
	for _, pi := range walkRes.Files {
		att, buildErr := attachment.BuildAttachmentForFile(context.Background(), &pi)
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
	runtimeFilters := make([]runtime.FileFilter, 0, len(attachment.DefaultFileFilters))
	for idx := range attachment.DefaultFileFilters {
		runtimeFilters = append(
			runtimeFilters,
			runtime.FileFilter{
				DisplayName: attachment.DefaultFileFilters[idx].DisplayName,
				Pattern:     attachment.DefaultFileFilters[idx].Pattern(),
			},
		)
	}
	return runtimeFilters
}()

// dialogFilters returns nil when no explicit filters are provided.
// This is important for macOS: providing filters makes NSOpenPanel/NSSavePanel
// restrict selectable/savable file types, which can hide/disable "unknown"
// extensions (e.g. .bazel/.cmake) and dotfiles depending on type mapping.
func dialogFilters(additionalFilters []attachment.FileFilter) []runtime.FileFilter {
	if len(additionalFilters) == 0 {
		return nil
	}
	// Only use the explicit filters requested by the caller.
	return getRuntimeFilters(additionalFilters, false)
}

func getRuntimeFilters(additionalFilters []attachment.FileFilter, includeDefault bool) []runtime.FileFilter {
	runtimeFilters := make([]runtime.FileFilter, 0, len(additionalFilters)+len(attachment.DefaultFileFilters))

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
