package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/adrg/xdg"
)

const (
	AppTitle = "FlexiGPT"
)

type App struct {
	ctx context.Context

	settingStoreAPI         *SettingStoreWrapper
	conversationStoreAPI    *ConversationCollectionWrapper
	modelPresetStoreAPI     *ModelPresetStoreWrapper
	toolStoreAPI            *ToolStoreWrapper
	toolRuntimeAPI          *ToolRuntimeWrapper
	skillStoreAPI           *SkillStoreWrapper
	skillBuiltInInstaller   builtin.HydrationInstaller
	skillAggregateAPI       *SkillAggregateWrapper
	skillRuntimeAPI         *SkillRuntimeWrapper
	mcpStoreAPI             *MCPStoreWrapper
	mcpRuntimeAPI           *MCPRuntimeWrapper
	mcpAggregateAPI         *MCPAggregateWrapper
	mcpBuiltInInstaller     builtin.HydrationInstaller
	aggregateAPI            *AggregrateWrapper
	assistantPresetStoreAPI *AssistantPresetStoreWrapper
	workspaceStoreAPI       *WorkspaceStoreWrapper
	workspaceRuntimeAPI     *WorkspaceRuntimeWrapper
	workspaceAggregateAPI   *WorkspaceAggregateWrapper

	artifactStoreComposition *compositionapi.Store

	dataBasePath string

	settingsDirPath         string
	conversationsDirPath    string
	modelPresetsDirPath     string
	toolsDirPath            string
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
	app.dataBasePath = filepath.Join(
		xdg.DataHome,
		builtin.ApplicationDataDirectoryName,
	)

	app.settingsDirPath = filepath.Join(
		app.dataBasePath,
		builtin.SettingsDirectoryName,
	)
	app.conversationsDirPath = filepath.Join(
		app.dataBasePath,
		builtin.ConversationsDirectoryName,
	)
	app.modelPresetsDirPath = filepath.Join(
		app.dataBasePath,
		builtin.ModelPresetsDirectoryName,
	)
	app.toolsDirPath = filepath.Join(
		app.dataBasePath,
		builtin.ToolsDirectoryName,
	)
	app.assistantPresetsDirPath = filepath.Join(
		app.dataBasePath,
		builtin.AssistantPresetsDirectoryName,
	)
	app.artifactStoreDirPath = filepath.Join(
		app.dataBasePath,
		builtin.ArtifactStoreDirectoryName,
	)

	if app.settingsDirPath == "" || app.conversationsDirPath == "" ||
		app.modelPresetsDirPath == "" ||
		app.assistantPresetsDirPath == "" || app.toolsDirPath == "" ||
		app.artifactStoreDirPath == "" {
		slog.Error(
			"invalid app path configuration",
			"artifactStoreDirPath", app.artifactStoreDirPath,
			"settingsDirPath", app.settingsDirPath,
			"conversationsDirPath", app.conversationsDirPath,
			"modelPresetsDirPath", app.modelPresetsDirPath,
			"assistantPresetsDirPath", app.assistantPresetsDirPath,
			"toolsDirPath", app.toolsDirPath,
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
	app.skillStoreAPI = &SkillStoreWrapper{}
	app.skillAggregateAPI = &SkillAggregateWrapper{}
	app.skillRuntimeAPI = &SkillRuntimeWrapper{}
	app.mcpStoreAPI = &MCPStoreWrapper{}
	app.mcpRuntimeAPI = &MCPRuntimeWrapper{}
	app.mcpAggregateAPI = &MCPAggregateWrapper{}
	app.toolRuntimeAPI = &ToolRuntimeWrapper{}
	app.aggregateAPI = &AggregrateWrapper{}
	app.workspaceStoreAPI = &WorkspaceStoreWrapper{}
	app.workspaceRuntimeAPI = &WorkspaceRuntimeWrapper{}
	app.workspaceAggregateAPI = &WorkspaceAggregateWrapper{}

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
	return os.MkdirAll(
		location,
		os.FileMode(builtin.ApplicationDirectoryMode),
	)
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

	artifactComposition, err := composeArtifactStore(
		context.Background(),
		a.artifactStoreDirPath,
	)
	if err != nil {
		slog.Error(
			"couldn't compose artifact store",
			"directory", a.artifactStoreDirPath,
			"error", err,
		)
		panic(
			"failed to initialize managers: artifact store composition failed\n" +
				err.Error(),
		)
	}
	a.artifactStoreComposition = artifactComposition

	slog.Info("artifact store initialized", "directory", a.artifactStoreDirPath)

	err = InitSkillStoreWrapper(
		a.skillStoreAPI,
		artifactComposition.Roots,
		artifactComposition.Sources,
		artifactComposition.Discovery,
		artifactComposition.Artifacts,
		artifactComposition.Resources,
		artifactComposition.ManagedArtifacts,
		artifactComposition.Protection,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize Skill Store consumer API",
			"error", err,
		)
		panic("failed to initialize managers: Skill store initialization failed\n" + err.Error())
	}
	slog.Info("skill store consumer API initialized")

	a.skillBuiltInInstaller, err = NewSkillBuiltInInstaller(
		a.skillStoreAPI.api,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize Skill built-in installer",
			"error", err,
		)
		panic("failed to initialize managers: Skill built-in installer initialization failed\n" + err.Error())
	}
	slog.Info("skill built-in installer initialized")

	err = InitSkillAggregateWrapper(
		a.skillAggregateAPI,
		artifactComposition.Artifacts,
		artifactComposition.Resources,
		a.skillRuntimeAPI,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize Skill aggregate and runtime",
			"error", err,
		)
		panic(
			"failed to initialize managers: Skill aggregate initialization failed\n" +
				err.Error(),
		)
	}
	slog.Info("skill aggregate and runtime initialized")

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

	a.mcpBuiltInInstaller, err = InitMCPWrappers(
		context.Background(),
		a.mcpStoreAPI,
		a.mcpRuntimeAPI,
		a.mcpAggregateAPI,
		artifactComposition.Roots,
		artifactComposition.Sources,
		artifactComposition.Discovery,
		artifactComposition.Artifacts,
		artifactComposition.Resources,
		artifactComposition.ManagedArtifacts,
		artifactComposition.Protection,
		a.settingStoreAPI.store,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize mcp host",
			"artifactStoreDirectory", a.artifactStoreDirPath,
			"error", err,
		)
		panic("failed to initialize managers: artifact-backed mcp initialization failed\n" + err.Error())
	}
	slog.Info("artifact-backed mcp host initialized")

	err = InitWorkspaceWrappers(
		a.workspaceStoreAPI,
		a.workspaceRuntimeAPI,
		a.workspaceAggregateAPI,
		artifactComposition.Sources,
		artifactComposition.Discovery,
		artifactComposition.Artifacts,
		artifactComposition.Resources,
		artifactComposition.LocatorResolvers,
		a.mcpStoreAPI.api,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize Workspace APIs",
			"error", err,
		)
		panic("failed to initialize managers: workspace initialization failed\n" + err.Error())
	}
	slog.Info("workspace consumer, runtime engine, and aggregate APIs initialized")

	err = EnsureBuiltinArtifactTopology(
		context.Background(),
		a.artifactStoreComposition.Topology,
		a.skillBuiltInInstaller,
		a.mcpBuiltInInstaller,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize shared built-in topology",
			"error",
			err,
		)
		panic(
			"failed to initialize managers: built-in topology initialization failed\n" +
				err.Error(),
		)
	}
	slog.Info("shared built-in artifact topology initialized")

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
		a.skillAggregateAPI.service,
		a.mcpAggregateAPI.serverResolver,
		a.mcpRuntimeAPI.runtime,
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
		a.skillAggregateAPI.service,
		a.mcpRuntimeAPI.runtime,
		a.workspaceStoreAPI.api,
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
	runtime.WindowShow(a.ctx)
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
	if a.mcpAggregateAPI != nil {
		a.mcpAggregateAPI.close()
	}
	if a.mcpRuntimeAPI != nil {
		a.mcpRuntimeAPI.close()
	}
	if a.mcpStoreAPI != nil {
		a.mcpStoreAPI.close()
	}
	if a.settingStoreAPI != nil {
		a.settingStoreAPI.close()
	}
	if a.skillAggregateAPI != nil {
		a.skillAggregateAPI.close()
	}
	if a.skillRuntimeAPI != nil {
		a.skillRuntimeAPI.close()
	}
	if a.workspaceAggregateAPI != nil {
		a.workspaceAggregateAPI.close()
	}
	if a.workspaceRuntimeAPI != nil {
		a.workspaceRuntimeAPI.close()
	}
	if a.workspaceStoreAPI != nil {
		a.workspaceStoreAPI.close()
	}
	if a.skillStoreAPI != nil {
		a.skillStoreAPI.close()
	}
	a.skillBuiltInInstaller = nil
	a.mcpBuiltInInstaller = nil

	if a.artifactStoreComposition != nil {
		if err := a.artifactStoreComposition.Close(); err != nil {
			slog.Error(
				"close Artifact Store composition",
				"error",
				err,
			)
		}
		a.artifactStoreComposition = nil
	}
	if a.toolStoreAPI != nil {
		a.toolStoreAPI.close()
	}

	if a.conversationStoreAPI != nil {
		a.conversationStoreAPI.close()
	}
}
