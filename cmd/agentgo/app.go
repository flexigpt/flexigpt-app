package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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

func (a *App) Ping() string {
	return "pong"
}

func (a *App) GetAppVersion() string {
	return Version
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
