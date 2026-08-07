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

	settingsDirectoryName         = "settings_v1"
	conversationsDirectoryName    = "conversations_v1"
	modelPresetsDirectoryName     = "model_presets_v1"
	toolsDirectoryName            = "tools_v1"
	mcpDirectoryName              = "mcp_servers_v1"
	assistantPresetsDirectoryName = "assistant_presets_v1"
	// Workspace Collections are stored by the shared artifact store. Do not
	// add a second Workspace-owned persistence directory. This is deliberately
	// a clean namespace: startup must not locate, import, copy, or migrate any
	// legacy Workspace or Artifact Store directory into it.
	artifactStoreDirectoryName = "artifacts_v1"
	appDirectoryMode           = 0o700
)

type App struct {
	ctx context.Context

	settingStoreAPI         *SettingStoreWrapper
	conversationStoreAPI    *ConversationCollectionWrapper
	modelPresetStoreAPI     *ModelPresetStoreWrapper
	toolStoreAPI            *ToolStoreWrapper
	toolRuntimeAPI          *ToolRuntimeWrapper
	skillBundleAPI          *SkillBundleWrapper
	mcpAPI                  *MCPWrapper
	aggregateAPI            *AggregrateWrapper
	assistantPresetStoreAPI *AssistantPresetStoreWrapper
	artifactStoreAPI        *ArtifactStoreWrapper
	workspaceAPI            *WorkspaceWrapper

	dataBasePath string

	settingsDirPath         string
	conversationsDirPath    string
	modelPresetsDirPath     string
	toolsDirPath            string
	mcpsDirPath             string
	assistantPresetsDirPath string
	artifactStoreDirPath    string
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
	app.mcpsDirPath = filepath.Join(app.dataBasePath, mcpDirectoryName)
	app.assistantPresetsDirPath = filepath.Join(app.dataBasePath, assistantPresetsDirectoryName)
	app.artifactStoreDirPath = filepath.Join(app.dataBasePath, artifactStoreDirectoryName)

	if app.settingsDirPath == "" || app.conversationsDirPath == "" ||
		app.modelPresetsDirPath == "" ||
		app.assistantPresetsDirPath == "" || app.toolsDirPath == "" ||
		app.mcpsDirPath == "" || app.artifactStoreDirPath == "" {
		slog.Error(
			"invalid app path configuration",
			"artifactStoreDirPath", app.artifactStoreDirPath,
			"settingsDirPath", app.settingsDirPath,
			"conversationsDirPath", app.conversationsDirPath,
			"modelPresetsDirPath", app.modelPresetsDirPath,
			"assistantPresetsDirPath", app.assistantPresetsDirPath,
			"toolsDirPath", app.toolsDirPath,
			"mcpsDirPath", app.mcpsDirPath,
		)
		panic("failed to initialize app: invalid path configuration")
	}
	if err := ensureAppPrivateDirectory(app.dataBasePath); err != nil {
		slog.Error(
			"failed to create application data directory",
			"path", app.dataBasePath,
			"error", err,
		)
		panic("failed to initialize app: could not create application data directory")
	}

	// Wails needs some instance of a struct to create bindings from its methods.
	// Therefore, the pattern followed is to create a hollow struct in new and then init in startup.
	app.settingStoreAPI = &SettingStoreWrapper{}
	app.conversationStoreAPI = &ConversationCollectionWrapper{}
	app.modelPresetStoreAPI = &ModelPresetStoreWrapper{}
	app.toolStoreAPI = &ToolStoreWrapper{}
	app.skillBundleAPI = &SkillBundleWrapper{}
	app.mcpAPI = &MCPWrapper{}
	app.toolRuntimeAPI = &ToolRuntimeWrapper{}
	app.aggregateAPI = &AggregrateWrapper{}
	app.artifactStoreAPI = &ArtifactStoreWrapper{}
	app.workspaceAPI = &WorkspaceWrapper{}

	app.assistantPresetStoreAPI = &AssistantPresetStoreWrapper{}

	if err := ensureAppPrivateDirectory(app.settingsDirPath); err != nil {
		slog.Error(
			"failed to create settings directory",
			"settings path", app.settingsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create settings directory")
	}
	if err := ensureAppPrivateDirectory(app.conversationsDirPath); err != nil {
		slog.Error(
			"failed to create conversations directory",
			"conversations path", app.conversationsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create conversations directory")
	}
	if err := ensureAppPrivateDirectory(app.modelPresetsDirPath); err != nil {

		slog.Error(
			"failed to create model presets directory",
			"model presets path", app.modelPresetsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create model presets directory")
	}

	if err := ensureAppPrivateDirectory(app.toolsDirPath); err != nil {

		slog.Error(
			"failed to create tools directory",
			"tools path", app.toolsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create tools directory")
	}
	if err := ensureAppPrivateDirectory(app.mcpsDirPath); err != nil {

		slog.Error(
			"failed to create mcp directory",
			"mcps path", app.mcpsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create mcp server directory")

	}
	if err := ensureAppPrivateDirectory(app.assistantPresetsDirPath); err != nil {

		slog.Error(
			"failed to create assistant presets directory",
			"assistant presets path", app.assistantPresetsDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create assistant presets directory")
	}
	if err := ensureAppPrivateDirectory(app.artifactStoreDirPath); err != nil {

		slog.Error(
			"failed to create artifact store directory",
			"artifactStoreDirPath", app.artifactStoreDirPath,
			"error", err,
		)
		panic("failed to initialize app: could not create artifact store directory")
	}

	slog.Info(
		"flexiGPT paths initialized",
		"app data", app.dataBasePath,
		"settingsDirPath", app.settingsDirPath,
		"conversationsDirPath", app.conversationsDirPath,
		"modelPresetsDirPath", app.modelPresetsDirPath,
		"toolsDirPath", app.toolsDirPath,
		"mcpsDirPath", app.mcpsDirPath,
		"assistantPresetsDirPath", app.assistantPresetsDirPath,
		"artifactStoreDirPath", app.artifactStoreDirPath,
	)
	return app
}

func (a *App) Ping() string {
	return "pong"
}

func (a *App) GetAppVersion() string {
	return Version
}

func ensureAppPrivateDirectory(location string) error {
	return os.MkdirAll(location, os.FileMode(appDirectoryMode))
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

	err = InitArtifactStoreWrapper(
		a.artifactStoreAPI,
		a.artifactStoreDirPath,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize artifact store",
			"directory", a.artifactStoreDirPath,
			"error", err,
		)
		panic("failed to initialize managers: artifact store initialization failed\n" + err.Error())
	}
	slog.Info("artifact store initialized", "directory", a.artifactStoreDirPath)

	err = InitWorkspaceWrapper(
		a.workspaceAPI,
		a.artifactStoreAPI.components,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize Workspace",
			"directory", a.artifactStoreDirPath,
			"error", err,
		)
		panic("failed to initialize managers: workspace initialization failed\n" + err.Error())
	}
	slog.Info("workspace initialized")

	err = InitSkillBundleWrapper(
		a.skillBundleAPI,
		a.artifactStoreAPI.components,
		a.workspaceAPI.api.SkillAdapter(),
		a.artifactStoreAPI.builtInTopology,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize artifact-backed Skill bundles",
			"error", err,
		)
		panic("failed to initialize managers: skill bundle initialization failed\n" + err.Error())
	}
	slog.Info("artifact-backed skill bundles initialized")

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
		a.skillBundleAPI.runtime,
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
		a.skillBundleAPI.runtime,
		a.mcpAPI.runtime,
		a.workspaceAPI.api,
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
	if a.workspaceAPI != nil {
		a.workspaceAPI.close()
	}
	if a.skillBundleAPI != nil {
		//nolint:contextcheck // Need separate context in shutdown.
		a.skillBundleAPI.close()
	}
	if a.artifactStoreAPI != nil {
		a.artifactStoreAPI.close()
	}
	if a.toolStoreAPI != nil {
		a.toolStoreAPI.close()
	}

	if a.conversationStoreAPI != nil {
		a.conversationStoreAPI.close()
	}
}
