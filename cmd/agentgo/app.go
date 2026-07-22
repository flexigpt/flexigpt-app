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

const (
	AppTitle = "FlexiGPT"

	settingsDirectoryName           = "settings"
	conversationsDirectoryName      = "conversationsv1"
	modelPresetsDirectoryName       = "modelpresetsv1"
	toolsDirectoryName              = "toolsv1"
	skillsDirectoryName             = "skillsv1"
	mcpDirectoryName                = "mcpserversv1"
	assistantPresetsDirectoryName   = "assistantpresetsv1"
	workspaceArtifactsDirectoryName = "workspace-artifacts"
	appDirectoryMode                = 0o770
)

type App struct {
	ctx context.Context

	settingStoreAPI         *SettingStoreWrapper
	conversationStoreAPI    *ConversationCollectionWrapper
	modelPresetStoreAPI     *ModelPresetStoreWrapper
	toolStoreAPI            *ToolStoreWrapper
	toolRuntimeAPI          *ToolRuntimeWrapper
	skillStoreAPI           *SkillStoreWrapper
	mcpAPI                  *MCPWrapper
	aggregateAPI            *AggregrateWrapper
	assistantPresetStoreAPI *AssistantPresetStoreWrapper
	workspaceAPI            *WorkspaceWrapper

	dataBasePath string

	settingsDirPath           string
	conversationsDirPath      string
	modelPresetsDirPath       string
	toolsDirPath              string
	skillsDirPath             string
	mcpsDirPath               string
	assistantPresetsDirPath   string
	workspaceArtifactsDirPath string
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

	app.settingsDirPath = filepath.Join(app.dataBasePath, settingsDirectoryName)
	app.conversationsDirPath = filepath.Join(app.dataBasePath, conversationsDirectoryName)
	app.modelPresetsDirPath = filepath.Join(app.dataBasePath, modelPresetsDirectoryName)
	app.toolsDirPath = filepath.Join(app.dataBasePath, toolsDirectoryName)
	app.skillsDirPath = filepath.Join(app.dataBasePath, skillsDirectoryName)
	app.mcpsDirPath = filepath.Join(app.dataBasePath, mcpDirectoryName)
	app.assistantPresetsDirPath = filepath.Join(app.dataBasePath, assistantPresetsDirectoryName)
	app.workspaceArtifactsDirPath = filepath.Join(app.dataBasePath, workspaceArtifactsDirectoryName)

	if app.settingsDirPath == "" || app.conversationsDirPath == "" ||
		app.modelPresetsDirPath == "" ||
		app.assistantPresetsDirPath == "" || app.toolsDirPath == "" ||
		app.skillsDirPath == "" || app.mcpsDirPath == "" ||
		app.workspaceArtifactsDirPath == "" {
		slog.Error(
			"invalid app path configuration",
			"workspaceArtifactsDirPath", app.workspaceArtifactsDirPath,
			"settingsDirPath", app.settingsDirPath,
			"conversationsDirPath", app.conversationsDirPath,
			"modelPresetsDirPath", app.modelPresetsDirPath,
			"assistantPresetsDirPath", app.assistantPresetsDirPath,
			"toolsDirPath", app.toolsDirPath,
			"skillsDirPath", app.skillsDirPath,
			"mcpsDirPath", app.mcpsDirPath,
		)
		panic("failed to initialize app: invalid path configuration")
	}

	// Wails needs some instance of a struct to create bindings from its methods.
	// Therefore, the pattern followed is to create a hollow struct in new and then init in startup.
	app.settingStoreAPI = &SettingStoreWrapper{}
	app.conversationStoreAPI = &ConversationCollectionWrapper{}
	app.modelPresetStoreAPI = &ModelPresetStoreWrapper{}
	app.toolStoreAPI = &ToolStoreWrapper{}
	app.skillStoreAPI = &SkillStoreWrapper{}
	app.mcpAPI = &MCPWrapper{}
	app.toolRuntimeAPI = &ToolRuntimeWrapper{}
	app.aggregateAPI = &AggregrateWrapper{}
	app.workspaceAPI = &WorkspaceWrapper{}

	app.assistantPresetStoreAPI = &AssistantPresetStoreWrapper{}

	if err := os.MkdirAll(app.settingsDirPath, os.FileMode(appDirectoryMode)); err != nil {
		slog.Error(
			"failed to create settings directory",
			"settings path", app.settingsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create settings directory")
	}
	if err := os.MkdirAll(app.conversationsDirPath, os.FileMode(appDirectoryMode)); err != nil {
		slog.Error(
			"failed to create conversations directory",
			"conversations path", app.conversationsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create conversations directory")
	}
	if err := os.MkdirAll(app.modelPresetsDirPath, os.FileMode(appDirectoryMode)); err != nil {
		slog.Error(
			"failed to create model presets directory",
			"model presets path", app.modelPresetsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create model presets directory")
	}

	if err := os.MkdirAll(app.toolsDirPath, os.FileMode(appDirectoryMode)); err != nil {
		slog.Error(
			"failed to create tools directory",
			"tools path", app.toolsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create tools directory")
	}
	if err := os.MkdirAll(app.skillsDirPath, os.FileMode(appDirectoryMode)); err != nil {
		slog.Error(
			"failed to create skills directory",
			"skills path", app.skillsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create skills directory")
	}
	if err := os.MkdirAll(app.mcpsDirPath, os.FileMode(appDirectoryMode)); err != nil {
		slog.Error(
			"failed to create mcp directory",
			"mcps path", app.mcpsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create mcp server directory")

	}
	if err := os.MkdirAll(app.assistantPresetsDirPath, os.FileMode(appDirectoryMode)); err != nil {
		slog.Error(
			"failed to create assistant presets directory",
			"assistant presets path", app.assistantPresetsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create assistant presets directory")
	}
	if err := os.MkdirAll(app.workspaceArtifactsDirPath, os.FileMode(appDirectoryMode)); err != nil {
		slog.Error(
			"failed to create Workspace artifact directory",
			"workspaceArtifactsDirPath", app.workspaceArtifactsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create Workspace artifact directory")
	}

	slog.Info(
		"flexiGPT paths initialized",
		"app data", app.dataBasePath,
		"settingsDirPath", app.settingsDirPath,
		"conversationsDirPath", app.conversationsDirPath,
		"modelPresetsDirPath", app.modelPresetsDirPath,
		"toolsDirPath", app.toolsDirPath,
		"skillsDirPath", app.skillsDirPath,
		"mcpsDirPath", app.mcpsDirPath,
		"assistantPresetsDirPath", app.assistantPresetsDirPath,
		"workspaceArtifactsDirPath", app.workspaceArtifactsDirPath,
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
		panic("failed to initialize managers: conversation store initialization failed\n" + err.Error())
	}
	slog.Info("conversation store initialized", "directory", a.conversationsDirPath)

	err = InitToolStoreWrapper(a.toolStoreAPI, a.toolsDirPath)
	if err != nil {
		slog.Error(
			"couldn't initialize tool store",
			"directory", a.toolsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: tool store initialization failed\n" + err.Error())
	}

	err = InitToolRuntimeWrapper(a.toolRuntimeAPI, a.toolStoreAPI.store)
	if err != nil {
		slog.Error(
			"couldn't initialize tool runtime",
			"error", err,
		)
		panic("failed to initialize managers: tool runtime initialization failed\n" + err.Error())
	}

	err = InitSkillStoreWrapper(a.skillStoreAPI, a.skillsDirPath)
	if err != nil {
		slog.Error(
			"couldn't initialize skill store",
			"directory", a.skillsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: skill store initialization failed\n" + err.Error())
	}
	slog.Info("skill store initialized", "directory", a.skillsDirPath)

	err = InitWorkspaceWrapper(
		a.workspaceAPI,
		a.workspaceArtifactsDirPath,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize Workspace",
			"directory", a.workspaceArtifactsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: Workspace initialization failed\n" + err.Error())
	}
	slog.Info("workspace initialized", "directory", a.workspaceArtifactsDirPath)

	err = InitAggregateSkillProvider(
		a.skillStoreAPI,
		a.workspaceAPI.api.SkillProvider(),
	)
	if err != nil {
		slog.Error(
			"couldn't initialize aggregate Skill provider",
			"error", err,
		)
		panic("failed to initialize managers: aggregate Skill provider initialization failed\n" + err.Error())
	}
	slog.Info("aggregate Skill provider initialized")

	err = InitSettingStoreWrapper(a.settingStoreAPI, a.settingsDirPath)
	if err != nil {
		slog.Error(
			"couldn't initialize settings store",
			"directory", a.settingsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: settings store initialization failed\n" + err.Error())
	}
	slog.Info("settings store initialized", "directory", a.settingsDirPath)

	err = InitMCPWrapper(
		context.Background(),
		a.mcpAPI,
		a.mcpsDirPath,
		newSettingSecretResolver(a.settingStoreAPI.store),
	)
	if err != nil {
		slog.Error(
			"couldn't initialize mcp host",
			"directory", a.mcpsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: mcp store initialization failed\n" + err.Error())
	}
	slog.Info("mcp host initialized", "directory", a.mcpsDirPath)

	err = InitModelPresetStoreWrapper(
		a.modelPresetStoreAPI,
		a.modelPresetsDirPath,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize model presets store",
			"dir", a.modelPresetsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: model presets store initialization failed\n" + err.Error())
	}
	slog.Info("model presets store initialized", "dir", a.modelPresetsDirPath)

	err = InitAssistantPresetStoreWrapper(
		a.assistantPresetStoreAPI,
		a.assistantPresetsDirPath,
		a.modelPresetStoreAPI.store,
		a.toolStoreAPI.store,
		a.skillStoreAPI.store,
		a.mcpAPI.store,
		a.mcpAPI.runtime,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize assistant preset store",
			"dir", a.assistantPresetsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: assistant preset store initialization failed\n" + err.Error())
	}
	slog.Info(
		"assistant preset store initialized",
		"dir", a.assistantPresetsDirPath,
	)

	err = InitAggregrateWrapper(
		a.aggregateAPI,
		a.modelPresetStoreAPI.store,
		a.settingStoreAPI.store,
		a.toolStoreAPI.store,
		a.skillStoreAPI.store,
		a.mcpAPI.runtime,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize aggregate",
			"error", err,
		)
		panic("failed to initialize managers: aggregate initialization failed\n" + err.Error())
	}

	slog.Info("aggregate initialized", "dir", a.modelPresetsDirPath)
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

	if a.assistantPresetStoreAPI != nil {
		a.assistantPresetStoreAPI.close()
	}
	if a.modelPresetStoreAPI != nil {
		a.modelPresetStoreAPI.close()
	}
	if a.mcpAPI != nil {
		//nolint:contextcheck // Need separate context in shutdown.
		a.mcpAPI.close()
	}
	if a.settingStoreAPI != nil {
		a.settingStoreAPI.close()
	}
	if a.skillStoreAPI != nil {
		a.skillStoreAPI.close()
	}
	if a.workspaceAPI != nil {
		a.workspaceAPI.close()
	}
	if a.toolStoreAPI != nil {
		a.toolStoreAPI.close()
	}

	if a.conversationStoreAPI != nil {
		a.conversationStoreAPI.close()
	}
}
