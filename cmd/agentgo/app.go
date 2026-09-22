package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/agent/store/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
	workspaceConsumerAPI "github.com/flexigpt/flexigpt-app/internal/workspace/store/consumerapi"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/adrg/xdg"
)

const (
	AppTitle = "FlexiGPT"
)

type App struct {
	ctx context.Context

	settingStoreAPI       *SettingStoreWrapper
	conversationStoreAPI  *ConversationCollectionWrapper
	modelPresetStoreAPI   *ModelPresetStoreWrapper
	toolStoreAPI          *ToolStoreWrapper
	toolRuntimeAPI        *ToolRuntimeWrapper
	agentStoreAPI         *AgentStoreWrapper
	agentBuiltInInstaller builtin.HydrationInstaller
	skillStoreAPI         *SkillStoreWrapper
	skillBuiltInInstaller builtin.HydrationInstaller
	skillAggregateAPI     *SkillAggregateWrapper
	skillRuntimeAPI       *SkillRuntimeWrapper
	mcpStoreAPI           *MCPStoreWrapper
	mcpRuntimeAPI         *MCPRuntimeWrapper
	mcpAggregateAPI       *MCPAggregateWrapper
	mcpBuiltInInstaller   builtin.HydrationInstaller
	aggregateAPI          *AggregrateWrapper
	workspaceStoreAPI     *WorkspaceStoreWrapper
	workspaceRuntimeAPI   *WorkspaceRuntimeWrapper

	artifactStoreComposition *compositionapi.Store

	dataBasePath string

	settingsDirPath      string
	conversationsDirPath string
	modelPresetsDirPath  string
	toolsDirPath         string
	artifactStoreDirPath string
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
	storageName := documentTopology.MustApplicationStorageName
	app.dataBasePath = filepath.Join(
		xdg.DataHome,
		storageName(documentTopology.ApplicationStorageDataDirectory),
	)

	app.settingsDirPath = filepath.Join(
		app.dataBasePath,
		storageName(documentTopology.ApplicationStorageSettingsDirectory),
	)
	app.conversationsDirPath = filepath.Join(
		app.dataBasePath,
		storageName(documentTopology.ApplicationStorageConversationsDirectory),
	)
	app.modelPresetsDirPath = filepath.Join(
		app.dataBasePath,
		storageName(documentTopology.ApplicationStorageModelPresetsDirectory),
	)
	app.toolsDirPath = filepath.Join(
		app.dataBasePath,
		storageName(documentTopology.ApplicationStorageToolsDirectory),
	)
	app.artifactStoreDirPath = filepath.Join(
		app.dataBasePath,
		storageName(
			documentTopology.ApplicationStorageArtifactStoreDirectory,
		),
	)

	if app.settingsDirPath == "" || app.conversationsDirPath == "" ||
		app.modelPresetsDirPath == "" ||
		app.toolsDirPath == "" ||
		app.artifactStoreDirPath == "" {
		slog.Error(
			"invalid app path configuration",
			"artifactStoreDirPath", app.artifactStoreDirPath,
			"settingsDirPath", app.settingsDirPath,
			"conversationsDirPath", app.conversationsDirPath,
			"modelPresetsDirPath", app.modelPresetsDirPath,
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
	app.agentStoreAPI = &AgentStoreWrapper{}
	app.skillRuntimeAPI = &SkillRuntimeWrapper{}
	app.mcpStoreAPI = &MCPStoreWrapper{}
	app.mcpRuntimeAPI = &MCPRuntimeWrapper{}
	app.mcpAggregateAPI = &MCPAggregateWrapper{}
	app.toolRuntimeAPI = &ToolRuntimeWrapper{}
	app.aggregateAPI = &AggregrateWrapper{}
	app.workspaceStoreAPI = &WorkspaceStoreWrapper{}
	app.workspaceRuntimeAPI = &WorkspaceRuntimeWrapper{}

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
		os.FileMode(basespec.ApplicationDirectoryMode),
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

	err = InitModelPresetStoreWrapper(
		a.modelPresetStoreAPI,
		a.modelPresetsDirPath,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize model presets store",
			"dir",
			a.modelPresetsDirPath,
			"error",
			err,
		)
		panic("failed to initialize managers: model presets store initialization failed\n" + err.Error())
	}
	slog.Info("model presets store initialized", "dir", a.modelPresetsDirPath)

	fallbackProviders, err := artifactFallbackProviders(
		a.toolStoreAPI,
		a.modelPresetStoreAPI,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize Artifact fallback providers",
			"error",
			err,
		)
		panic(
			"failed to initialize managers: Artifact fallback providers failed\n" +
				err.Error(),
		)
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
		fallbackProviders,
		artifactComposition.LocatorResolvers...,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize Skill Store consumer API",
			"error", err,
		)
		panic("failed to initialize managers: Skill store initialization failed\n" + err.Error())
	}
	slog.Info("skill store consumer API initialized")

	skillBuiltinStore, err := skillConsumerAPI.NewBuiltinStore(
		a.skillStoreAPI.api,
	)
	if err != nil {
		panic("failed to initialize Skill built-in port: " + err.Error())
	}
	skillBaselineEnsurer, err := skillConsumerAPI.NewBaselineEnsurer(
		a.skillStoreAPI.api,
	)
	if err != nil {
		panic("failed to initialize Skill baseline port: " + err.Error())
	}

	a.skillBuiltInInstaller, err = NewSkillBuiltInInstaller(
		skillBuiltinStore,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize Skill built-in installer",
			"error", err,
		)
		panic("failed to initialize managers: Skill built-in installer initialization failed\n" + err.Error())
	}
	slog.Info("skill built-in installer initialized")

	err = InitAgentStoreWrapper(
		a.agentStoreAPI,
		artifactComposition.Roots,
		artifactComposition.Sources,
		artifactComposition.Discovery,
		artifactComposition.Artifacts,
		artifactComposition.Resources,
		artifactComposition.ManagedArtifacts,
		artifactComposition.Protection,
		fallbackProviders,
		artifactComposition.LocatorResolvers...,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize agent Store consumer API",
			"error",
			err,
		)
		panic("failed to initialize managers: agent Store initialization failed\n" + err.Error())
	}
	slog.Info("agent store consumer API initialized")

	agentBuiltinStore, err := agentConsumerAPI.NewBuiltinStore(
		a.agentStoreAPI.api,
	)
	if err != nil {
		panic("failed to initialize Agent built-in port: " + err.Error())
	}
	agentBaselineEnsurer, err := agentConsumerAPI.NewBaselineEnsurer(
		a.agentStoreAPI.api,
	)
	if err != nil {
		panic("failed to initialize Agent baseline port: " + err.Error())
	}

	a.agentBuiltInInstaller, err = NewAgentBuiltInInstaller(
		agentBuiltinStore,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize agent built-in installer",
			"error",
			err,
		)
		panic(
			"failed to initialize managers: agent built-in installer initialization failed\n" +
				err.Error(),
		)
	}
	slog.Info("agent built-in installer initialized")

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
		artifactComposition.LocatorResolvers,
		fallbackProviders,
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

	mcpBaselineEnsurer, err := mcpConsumerAPI.NewBaselineEnsurer(
		a.mcpStoreAPI.api,
	)
	if err != nil {
		panic("failed to initialize MCP baseline port: " + err.Error())
	}
	mcpWorkspaceResolver, err := mcpConsumerAPI.NewWorkspaceServerResolver(
		a.mcpStoreAPI.api,
	)
	if err != nil {
		panic("failed to initialize Workspace MCP resolver: " + err.Error())
	}

	err = InitWorkspaceWrappers(
		a.workspaceStoreAPI,
		a.workspaceRuntimeAPI,
		artifactComposition.Roots,
		artifactComposition.Sources,
		artifactComposition.Discovery,
		artifactComposition.Artifacts,
		artifactComposition.Resources,
		artifactComposition.LocatorResolvers,
		fallbackProviders,
		mcpWorkspaceResolver,
		func(ctx context.Context, rootID root.RootID) error {
			return EnsureUserArtifactBaselineCollectionsForRoot(
				ctx,
				rootID,
				skillBaselineEnsurer,
				mcpBaselineEnsurer,
				agentBaselineEnsurer,
			)
		},
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
		a.agentBuiltInInstaller,
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

	err = EnsureUserArtifactBaselineCollections(
		context.Background(),
		artifactComposition.Roots,
		artifactComposition.Protection,
		skillBaselineEnsurer,
		mcpBaselineEnsurer,
		agentBaselineEnsurer,
	)
	if err != nil {
		slog.Error(
			"couldn't provision user Artifact baseline Collections",
			"error",
			err,
		)
		panic(
			"failed to initialize managers: user Artifact baseline provisioning failed\n" +
				err.Error(),
		)
	}
	slog.Info("user Artifact baseline Collections initialized")

	workspaceConversationSource, err := workspaceConsumerAPI.NewConversationSource(
		a.workspaceStoreAPI.api,
	)
	if err != nil {
		panic(
			"failed to initialize managers: Workspace conversation source failed\n" +
				err.Error(),
		)
	}

	err = InitAggregrateWrapper(
		a.aggregateAPI,
		a.modelPresetStoreAPI.store,
		a.settingStoreAPI.store,
		a.toolStoreAPI.store,
		a.skillAggregateAPI.service,
		a.mcpRuntimeAPI.runtime,
		workspaceConversationSource,
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

	SetWrappedProviderAppContext(a.aggregateAPI, a.ctx)

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
	if a.agentStoreAPI != nil {
		a.agentStoreAPI.close()
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
	a.agentBuiltInInstaller = nil
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
	if a.modelPresetStoreAPI != nil {
		a.modelPresetStoreAPI.close()
	}
	if a.toolStoreAPI != nil {
		a.toolStoreAPI.close()
	}

	if a.conversationStoreAPI != nil {
		a.conversationStoreAPI.close()
	}
}
