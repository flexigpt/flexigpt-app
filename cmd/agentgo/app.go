package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/attachment"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/llmtools-go/fstool"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/adrg/xdg"
)

const AppTitle = "FlexiGPT"

type App struct {
	ctx                    context.Context
	settingStoreAPI        *SettingStoreWrapper
	conversationStoreAPI   *ConversationCollectionWrapper
	modelPresetStoreAPI    *ModelPresetStoreWrapper
	promptTemplateStoreAPI *PromptTemplateStoreWrapper
	toolStoreAPI           *ToolStoreWrapper
	toolRuntimeAPI         *ToolRuntimeWrapper
	providerSetAPI         *ProviderSetWrapper
	skillStoreAPI          *SkillStoreWrapper

	dataBasePath string

	settingsDirPath      string
	conversationsDirPath string
	modelPresetsDirPath  string
	promptsDirPath       string
	toolsDirPath         string
	skillsDirPath        string
}

func NewApp() *App {
	if xdg.DataHome == "" {
		slog.Error(
			"could not resolve xdg data paths",
			"xdg data dir", xdg.DataHome,
		)
		panic("failed to initialize app: xdg paths not set")
	}

	app := &App{}
	app.dataBasePath = filepath.Join(xdg.DataHome, strings.ToLower(AppTitle))

	app.settingsDirPath = filepath.Join(app.dataBasePath, "settings")
	app.conversationsDirPath = filepath.Join(app.dataBasePath, "conversationsv1")
	app.modelPresetsDirPath = filepath.Join(app.dataBasePath, "modelpresetsv1")
	app.promptsDirPath = filepath.Join(app.dataBasePath, "prompttemplates")
	app.toolsDirPath = filepath.Join(app.dataBasePath, "toolsv1")
	app.skillsDirPath = filepath.Join(app.dataBasePath, "skills")

	if app.settingsDirPath == "" || app.conversationsDirPath == "" ||
		app.modelPresetsDirPath == "" || app.promptsDirPath == "" || app.toolsDirPath == "" || app.skillsDirPath == "" {
		slog.Error(
			"invalid app path configuration",
			"settingsDirPath", app.settingsDirPath,
			"conversationsDirPath", app.conversationsDirPath,
			"modelPresetsDirPath", app.modelPresetsDirPath,
			"promptsDirPath", app.promptsDirPath,
			"toolsDirPath", app.toolsDirPath,
			"skillsDirPath", app.skillsDirPath,
		)
		panic("failed to initialize app: invalid path configuration")
	}

	// Wails needs some instance of a struct to create bindings from its methods.
	// Therefore, the pattern followed is to create a hollow struct in new and then init in startup.
	app.settingStoreAPI = &SettingStoreWrapper{}
	app.conversationStoreAPI = &ConversationCollectionWrapper{}
	app.providerSetAPI = &ProviderSetWrapper{}
	app.modelPresetStoreAPI = &ModelPresetStoreWrapper{}
	app.promptTemplateStoreAPI = &PromptTemplateStoreWrapper{}
	app.toolStoreAPI = &ToolStoreWrapper{}
	app.skillStoreAPI = &SkillStoreWrapper{}
	app.toolRuntimeAPI = &ToolRuntimeWrapper{}

	if err := os.MkdirAll(app.settingsDirPath, os.FileMode(0o770)); err != nil {
		slog.Error(
			"failed to create settings directory",
			"settings path", app.settingsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create settings directory")
	}
	if err := os.MkdirAll(app.conversationsDirPath, os.FileMode(0o770)); err != nil {
		slog.Error(
			"failed to create conversations directory",
			"conversations path", app.conversationsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create conversations directory")
	}
	if err := os.MkdirAll(app.modelPresetsDirPath, os.FileMode(0o770)); err != nil {
		slog.Error(
			"failed to create model presets directory",
			"model presets path", app.modelPresetsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create model presets directory")
	}
	if err := os.MkdirAll(app.promptsDirPath, os.FileMode(0o770)); err != nil {
		slog.Error(
			"failed to create prompt templates directory",
			"prompt Templates path", app.promptsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create prompt templates directory")
	}
	if err := os.MkdirAll(app.toolsDirPath, os.FileMode(0o770)); err != nil {
		slog.Error(
			"failed to create tools directory",
			"tools path", app.toolsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create tools directory")
	}
	if err := os.MkdirAll(app.skillsDirPath, os.FileMode(0o770)); err != nil {
		slog.Error(
			"failed to create skills directory",
			"skills path", app.skillsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create skills directory")
	}

	slog.Info(
		"flexiGPT paths initialized",
		"app data", app.dataBasePath,
		"settingsDirPath", app.settingsDirPath,
		"conversationsDirPath", app.conversationsDirPath,
		"modelPresetsDirPath", app.modelPresetsDirPath,
		"promptsDirPath", app.promptsDirPath,
		"toolsDirPath", app.toolsDirPath,
		"skillsDirPath", app.skillsDirPath,
	)
	return app
}

// Ping - Greet returns a greeting for the given name.
func (a *App) Ping() string {
	return "pong"
}

func (a *App) GetAppVersion() string {
	return Version
}

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

func (a *App) initManagers() {
	err := InitConversationCollectionWrapper(a.conversationStoreAPI, a.conversationsDirPath)
	if err != nil {
		slog.Error(
			"couldn't initialize conversation store",
			"directory", a.conversationsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: conversation store initialization failed")
	}
	slog.Info("conversation store initialized", "directory", a.conversationsDirPath)

	err = InitPromptTemplateStoreWrapper(a.promptTemplateStoreAPI, a.promptsDirPath)
	if err != nil {
		slog.Error(
			"couldn't initialize prompt template store",
			"directory", a.promptsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: prompt template store initialization failed")
	}
	slog.Info("prompt store initialized", "directory", a.promptsDirPath)

	err = InitToolStoreWrapper(a.toolStoreAPI, a.toolsDirPath)
	if err != nil {
		slog.Error(
			"couldn't initialize tool store",
			"directory", a.toolsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: tool store initialization failed")
	}

	err = InitToolRuntimeWrapper(a.toolRuntimeAPI, a.toolStoreAPI.store)
	if err != nil {
		slog.Error(
			"couldn't initialize tool runtime",
			"error", err,
		)
		panic("failed to initialize managers: tool runtime initialization failed")
	}

	err = InitSkillStoreWrapper(a.skillStoreAPI, a.skillsDirPath)
	if err != nil {
		slog.Error(
			"couldn't initialize skill store",
			"directory", a.skillsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: skill store initialization failed")
	}
	slog.Info("skill store initialized", "directory", a.skillsDirPath)

	err = InitProviderSetWrapper(a.providerSetAPI, a.toolStoreAPI.store)
	if err != nil {
		slog.Error(
			"couldn't initialize provider set",
			"error", err,
		)
		panic("failed to initialize managers: provider set initialization failed")
	}

	err = InitSettingStoreWrapper(a.settingStoreAPI, a.providerSetAPI, a.settingsDirPath)
	if err != nil {
		slog.Error(
			"couldn't initialize settings store",
			"directory", a.settingsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: settings store initialization failed")
	}
	slog.Info("settings store initialized", "directory", a.settingsDirPath)

	err = InitModelPresetStoreWrapper(
		a.modelPresetStoreAPI,
		a.settingStoreAPI,
		a.providerSetAPI,
		a.modelPresetsDirPath,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize model presets store",
			"dir", a.modelPresetsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: model presets store initialization failed")
	}
	slog.Info("model presets store initialized", "dir", a.modelPresetsDirPath)
}

// startup is called at application startup.
func (a *App) startup(ctx context.Context) { //nolint:all
	// Perform setup here.
	a.ctx = ctx
	// Load the frontend.
	runtime.WindowShow(a.ctx) //nolint:contextcheck // Use app context.
}

// domReady is called after front-end resources have been loaded.
func (a *App) domReady(ctx context.Context) { //nolint:all
	// Add action here.
}

// beforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) beforeClose(ctx context.Context) (prevent bool) { //nolint:all
	return false
}

// shutdown is called at application termination.
func (a *App) shutdown(ctx context.Context) { //nolint:all
	// Perform any teardown here.

	// Stop background goroutines + flushes for stores that need it.
	if a.skillStoreAPI != nil {
		a.skillStoreAPI.Close()
	}
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
