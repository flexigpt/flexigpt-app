package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/workspace"
	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/consumerapi"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/consumerapi"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/consumerapi"
	workspaceConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/consumerapi"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/adrg/xdg"
)

const (
	AppTitle                = "FlexiGPT"
	appPrivateDirectoryMode = 0o700
)

type App struct {
	ctx context.Context

	settingStoreAPI       *SettingStoreWrapper
	conversationStoreAPI  *ConversationPluginWrapper
	modelStoreAPI         *ModelStoreWrapper
	modelAggregateAPI     *ModelAggregateWrapper
	toolStoreAPI          *ToolStoreWrapper
	toolRuntimeAPI        *ToolRuntimeWrapper
	toolAggregateAPI      *ToolAggregateWrapper
	toolBuiltInInstaller  installFlow.HydrationInstaller
	modelBuiltInInstaller installFlow.HydrationInstaller
	agentStoreAPI         *AgentStoreWrapper
	agentBuiltInInstaller installFlow.HydrationInstaller
	skillStoreAPI         *SkillStoreWrapper
	skillBuiltInInstaller installFlow.HydrationInstaller
	skillAggregateAPI     *SkillAggregateWrapper
	skillRuntimeAPI       *SkillRuntimeWrapper
	mcpStoreAPI           *MCPStoreWrapper
	mcpRuntimeAPI         *MCPRuntimeWrapper
	mcpAggregateAPI       *MCPAggregateWrapper
	mcpBuiltInInstaller   installFlow.HydrationInstaller
	completionAPI         *CompletionWrapper
	workspaceStoreAPI     *WorkspaceStoreWrapper
	workspaceRuntimeAPI   *WorkspaceRuntimeWrapper

	artifactStoreSetup *artifactsetup.Handle

	dataBasePath string

	settingsDirPath      string
	conversationsDirPath string
	artifactStoreDirPath string

	artifactInitializationError error
}

func newApp() *App {
	if xdg.DataHome == "" {
		slog.Error(
			"could not resolve xdg data paths",
			"xdg data dir", xdg.DataHome,
		)
		panic("failed to initialize app: xdg paths not set")
	}

	app := &App{}
	storageName := topology.MustApplicationStorageName
	app.dataBasePath = filepath.Join(
		xdg.DataHome,
		storageName(topology.ApplicationStorageDataDirectory),
	)

	app.settingsDirPath = filepath.Join(
		app.dataBasePath,
		storageName(topology.ApplicationStorageSettingsDirectory),
	)
	app.conversationsDirPath = filepath.Join(
		app.dataBasePath,
		storageName(topology.ApplicationStorageConversationsDirectory),
	)
	app.artifactStoreDirPath = filepath.Join(
		app.dataBasePath,
		storageName(
			topology.ApplicationStorageArtifactStoreDirectory,
		),
	)

	if app.settingsDirPath == "" || app.conversationsDirPath == "" || app.artifactStoreDirPath == "" {
		slog.Error(
			"invalid app path configuration",
			"artifactStoreDirPath", app.artifactStoreDirPath,
			"settingsDirPath", app.settingsDirPath,
			"conversationsDirPath", app.conversationsDirPath,
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
	app.conversationStoreAPI = &ConversationPluginWrapper{}
	app.modelStoreAPI = &ModelStoreWrapper{}
	app.modelAggregateAPI = &ModelAggregateWrapper{}
	app.toolStoreAPI = &ToolStoreWrapper{}
	app.toolRuntimeAPI = &ToolRuntimeWrapper{}
	app.toolAggregateAPI = &ToolAggregateWrapper{}
	app.skillStoreAPI = &SkillStoreWrapper{}
	app.skillAggregateAPI = &SkillAggregateWrapper{}
	app.agentStoreAPI = &AgentStoreWrapper{}
	app.skillRuntimeAPI = &SkillRuntimeWrapper{}
	app.mcpStoreAPI = &MCPStoreWrapper{}
	app.mcpRuntimeAPI = &MCPRuntimeWrapper{}
	app.mcpAggregateAPI = &MCPAggregateWrapper{}
	app.completionAPI = &CompletionWrapper{}
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

func (a *App) GetArtifactInitializationError() string {
	if a == nil || a.artifactInitializationError == nil {
		return ""
	}
	return a.artifactInitializationError.Error()
}

func ensureAppPrivateDirectory(location string) error {
	return os.MkdirAll(
		location,
		os.FileMode(appPrivateDirectoryMode),
	)
}

func (a *App) initManagers() {
	err := InitConversationPluginWrapper(a.conversationStoreAPI, a.conversationsDirPath)
	if err != nil {
		slog.Error(
			"couldn't initialize conversation store",
			"directory", a.conversationsDirPath,
			"error", err,
		)
		panic("failed to initialize managers: conversation store initialization failed\n" + err.Error())
	}
	slog.Info("conversation store initialized", "directory", a.conversationsDirPath)

	artifactSetup, err := artifactsetup.OpenArtifactStore(
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
	if artifactSetup.Store == nil || artifactSetup.LLM == nil {
		_ = artifactSetup.Close()
		panic(
			"failed to initialize managers: Artifact Store setup returned an incomplete handle",
		)
	}

	a.artifactStoreSetup = artifactSetup
	artifactComposition := artifactSetup.Store

	artifactCompositionResolver := artifactSetup.LLM.Composition()
	artifactInterpretations := artifactSetup.LLM.Interpretations()
	if artifactCompositionResolver == nil {
		panic(
			"failed to initialize managers: LLM Artifactory composition resolver is unavailable",
		)
	}
	if artifactInterpretations == nil {
		panic(
			"failed to initialize managers: LLM Artifactory declaration interpretation registry is unavailable",
		)
	}
	slog.Info("artifact store initialized", "directory", a.artifactStoreDirPath)

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

	err = InitToolRuntimeWrapper(a.toolRuntimeAPI)
	if err != nil {
		slog.Error(
			"couldn't initialize artifact-backed Tool runtime",
			"error",
			err,
		)
		panic(
			"failed to initialize managers: Tool runtime initialization failed\n" +
				err.Error(),
		)
	}

	toolBuiltinCatalog, err := ToolBuiltinCatalog()
	if err != nil {
		slog.Error(
			"couldn't load generated Tool built-in catalog",
			"error",
			err,
		)
		panic(
			"failed to initialize managers: Tool built-in catalog initialization failed\n" +
				err.Error(),
		)
	}
	err = InitToolStoreWrapper(
		a.toolStoreAPI,
		artifactComposition.Sources,
		artifactComposition.Refresh,
		artifactComposition.Artifacts,
		artifactComposition.ManagedPackages,
		artifactComposition.Protection,
		artifactComposition.Catalog,
		artifactComposition.Definitions,
		toolBuiltinCatalog,
		artifactCompositionResolver,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize artifact-backed Tool Store",
			"error",
			err,
		)
		panic(
			"failed to initialize managers: Tool Store initialization failed\n" +
				err.Error(),
		)
	}

	err = InitToolAggregateWrapper(
		a.toolAggregateAPI,
		a.toolStoreAPI,
		a.toolRuntimeAPI,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize artifact-backed Tool aggregate",
			"error",
			err,
		)
		panic(
			"failed to initialize managers: Tool aggregate initialization failed\n" +
				err.Error(),
		)
	}

	a.toolBuiltInInstaller, err = NewToolBuiltInInstaller(
		artifactComposition.Topology,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize Tool built-in installer",
			"error",
			err,
		)
		panic(
			"failed to initialize managers: Tool built-in installer initialization failed\n" +
				err.Error(),
		)
	}
	slog.Info("artifact-backed Tool store, runtime, and aggregate initialized")

	a.modelBuiltInInstaller, err = initModelWrappers(
		context.Background(),
		a.modelStoreAPI,
		a.modelAggregateAPI,
		artifactComposition.Sources,
		artifactComposition.Refresh,
		artifactComposition.Artifacts,
		artifactComposition.Roots,
		artifactComposition.ManagedPackages,
		artifactComposition.Protection,
		artifactComposition.ProtectedOverlays,
		artifactComposition.SecretBindings,
		artifactComposition.SecretRuntime,
		artifactComposition.ArtifactCleanup,
		artifactComposition.StoreOverlays,
		artifactComposition.Catalog,
		artifactComposition.Definitions,
		artifactComposition.Topology,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize artifact-backed Model Store",
			"error",
			err,
		)
		panic(
			"failed to initialize managers: Model Store initialization failed\n" +
				err.Error(),
		)
	}
	slog.Info("artifact-backed Model Store and aggregate initialized")

	err = InitSkillStoreWrapper(
		a.skillStoreAPI,
		artifactComposition.Roots,
		artifactComposition.Sources,
		artifactComposition.Refresh,
		artifactComposition.Artifacts,
		artifactComposition.Catalog,
		artifactComposition.Resources,
		artifactComposition.ManagedPackages,
		artifactComposition.Protection,
		artifactComposition.Definitions,
		artifactCompositionResolver,
	)
	if err != nil {
		slog.Error(
			"couldn't initialize Skill Store consumer API",
			"error", err,
		)
		panic("failed to initialize managers: Skill store initialization failed\n" + err.Error())
	}
	slog.Info("skill store consumer API initialized")

	skillBaselineEnsurer, err := skillConsumerAPI.NewBaselineEnsurer(
		a.skillStoreAPI.api,
	)
	if err != nil {
		panic("failed to initialize Skill baseline port: " + err.Error())
	}

	a.skillBuiltInInstaller, err = NewSkillBuiltInInstaller(
		artifactComposition.Topology,
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
		artifactComposition.Catalog,
		artifactComposition.Sources,
		artifactComposition.Refresh,
		artifactComposition.Artifacts,
		artifactComposition.Resources,
		artifactComposition.ManagedPackages,
		artifactComposition.Protection,
		artifactComposition.Definitions,
		artifactCompositionResolver,
		artifactInterpretations,
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

	agentBaselineEnsurer, err := agentConsumerAPI.NewBaselineEnsurer(
		a.agentStoreAPI.api,
	)
	if err != nil {
		panic("failed to initialize Agent baseline port: " + err.Error())
	}

	a.agentBuiltInInstaller, err = NewAgentBuiltInInstaller(
		artifactComposition.Topology,
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
		artifactComposition.Catalog,
		artifactComposition.Resources,
		artifactComposition.TrustedNativeResources,
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

	a.mcpBuiltInInstaller, err = initMCPWrappers(
		context.Background(),
		a.mcpStoreAPI,
		a.mcpRuntimeAPI,
		a.mcpAggregateAPI,
		artifactComposition.Roots,
		artifactComposition.Sources,
		artifactComposition.Refresh,
		artifactComposition.Artifacts,
		artifactComposition.Catalog,
		artifactComposition.Definitions,
		artifactComposition.Resources,
		artifactComposition.ManagedPackages,
		artifactComposition.Protection,
		artifactComposition.ProtectedOverlays,
		artifactComposition.StoreOverlays,
		artifactComposition.SecretBindings,
		artifactComposition.SecretRuntime,
		artifactComposition.ArtifactCleanup,
		artifactComposition.Topology,
		artifactCompositionResolver,
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

	workspaceConfig, err := workspace.DefaultWorkspaceConfig()
	if err != nil {
		panic(
			"failed to initialize managers: Workspace default policy setup failed\n" +
				err.Error(),
		)
	}

	err = InitWorkspaceWrappers(
		a.workspaceStoreAPI,
		a.workspaceRuntimeAPI,
		artifactComposition.Roots,
		artifactComposition.Sources,
		artifactComposition.Refresh,
		artifactComposition.Artifacts,
		artifactComposition.Catalog,
		artifactComposition.Resources,
		artifactComposition.TrustedNativeResources,
		artifactCompositionResolver,
		workspaceConfig,
		mcpWorkspaceResolver,
		func(ctx context.Context, rootID rootModel.RootID) error {
			return artifactsetup.EnsureRootBaselines(
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

	err = artifactsetup.EnsureBuiltInTopology(
		context.Background(),
		artifactComposition.Topology,
		a.toolBuiltInInstaller,
		a.modelBuiltInInstaller,
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
		a.artifactInitializationError = errors.Join(
			a.artifactInitializationError,
			err,
		)
	} else {
		slog.Info("shared built-in artifact topology initialized")
	}

	err = artifactsetup.EnsureMutableRootBaselines(
		context.Background(),
		artifactComposition.Roots,
		artifactComposition.Protection,
		skillBaselineEnsurer,
		mcpBaselineEnsurer,
		agentBaselineEnsurer,
	)
	if err != nil {
		slog.Error(
			"couldn't provision user Artifact baseline Plugins",
			"error",
			err,
		)
		a.artifactInitializationError = errors.Join(
			a.artifactInitializationError,
			err,
		)
	} else {
		slog.Info("user Artifact baseline Plugins initialized")
	}

	workspaceConversationSource, err := workspaceConsumerAPI.NewConversationSource(
		a.workspaceStoreAPI.api,
	)
	if err != nil {
		panic(
			"failed to initialize managers: Workspace conversation source failed\n" +
				err.Error(),
		)
	}

	err = InitCompletionWrapper(
		a.completionAPI,
		a.modelAggregateAPI.service,
		a.settingStoreAPI.store,
		a.toolAggregateAPI.service,
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

	providerPublisher, err := newInferenceProviderRuntimePublisher(
		a.completionAPI.providersetAPI,
	)
	if err != nil {
		panic(
			"failed to initialize managers: Model Provider runtime publisher failed\n" +
				err.Error(),
		)
	}
	if err := a.modelAggregateAPI.setProviderRuntimePublisher(
		providerPublisher,
	); err != nil {
		panic(
			"failed to initialize managers: Model Provider runtime publisher binding failed\n" +
				err.Error(),
		)
	}

	// Hydration and publisher binding must finish before restoring Providers.
	err = initModelProviderRuntime(
		context.Background(),
		a.modelStoreAPI,
		a.modelAggregateAPI,
	)
	if err != nil {
		slog.Error("couldn't initialize Model Provider runtime", "error", err)
		a.artifactInitializationError = errors.Join(
			a.artifactInitializationError,
			err,
		)
	} else {
		slog.Info("model Provider runtime initialized")
	}
}

// startup is called at application startup.
func (a *App) startup(ctx context.Context) { //nolint:all
	a.ctx = ctx

	SetWrappedProviderAppContext(a.completionAPI, a.ctx)

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
	if a.toolAggregateAPI != nil {
		a.toolAggregateAPI.close()
	}
	if a.toolStoreAPI != nil {
		a.toolStoreAPI.close()
	}
	if a.toolRuntimeAPI != nil {
		a.toolRuntimeAPI.close()
	}
	if a.modelAggregateAPI != nil {
		a.modelAggregateAPI.close()
	}
	if a.modelStoreAPI != nil {
		a.modelStoreAPI.close()
	}
	a.skillBuiltInInstaller = nil
	a.mcpBuiltInInstaller = nil
	a.agentBuiltInInstaller = nil
	a.modelBuiltInInstaller = nil
	a.toolBuiltInInstaller = nil
	if a.artifactStoreSetup != nil {
		if err := a.artifactStoreSetup.Close(); err != nil {
			slog.Error(
				"close artifact store setup",
				"error",
				err,
			)
		}
		a.artifactStoreSetup = nil
	}

	if a.conversationStoreAPI != nil {
		a.conversationStoreAPI.close()
	}
}
