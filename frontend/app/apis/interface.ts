import type {
	AgentCapabilityPlan,
	AgentExportResult,
	AgentImportCommitRequest,
	AgentImportCommitResult,
	AgentImportDestination,
	AgentImportPreview,
	AgentImportPreviewRequest,
	AgentResolution,
	AgentView,
	ListAgentsRequest,
} from '@/spec/agent';
import type {
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceID,
	CapabilityPlan,
	CapabilityTarget,
	StoreArtifact,
	StoreArtifactSourceSummary,
} from '@/spec/artifact';
import type {
	Attachment,
	AttachmentsDroppedPayload,
	DirectoryAttachmentsResult,
	FileFilter,
	PathAttachmentsResult,
} from '@/spec/attachment';
import type { ConversationSearchItem, StoreConversation, StoreConversationMessage } from '@/spec/conversation';
import type {
	ManagedMCPCreateRequest,
	ManagedMCPCreateResult,
	ManagedMCPPolicyUpsertRequest,
	ManagedMCPPolicyUpsertResult,
	ManagedMCPReplaceRequest,
	ManagedMCPReplaceResult,
	MCPAuthSettings,
	MCPCompleteArgumentRequestBody,
	MCPCompletionResult,
	MCPDiscoveryPage,
	MCPGetPromptResponseBody,
	MCPPluginPage,
	MCPPolicyListItem,
	MCPPolicyView,
	MCPPromptRef,
	MCPReadResourceResponseBody,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPRuntimeServerID,
	MCPServerData,
	MCPServerDetails,
	MCPServerListItem,
	MCPServerPage,
	MCPServerRuntimeDetails,
	MCPServerRuntimeSnapshot,
	MCPServerSecretsView,
	MCPSettings,
	MCPToolCapability,
} from '@/spec/mcp';
import type {
	ManagedModelCreateRequest,
	ManagedModelReplaceRequest,
	ManagedModelResult,
	ManagedProviderCreateRequest,
	ManagedProviderReplaceRequest,
	ManagedProviderResult,
	ModelListItem,
	ModelProviderListItem,
	ModelProviderView,
	ModelView,
	ProviderAPIKeyStatus,
	SaveModelSettingsRequest,
	SaveProviderSettingsRequest,
	SetProviderAPIKeyRequest,
} from '@/spec/model';
import type {
	AddPluginArtifactMemberRequest,
	AddPluginMemberRequest,
	ArtifactPluginMembershipView,
	CreatePluginRequest,
	DeletePluginRequest,
	PluginCapabilityPlan,
	PluginDirectMembership,
	PluginListItem,
	PluginView,
	RemovePluginMemberRequest,
	UpdatePluginRequest,
} from '@/spec/plugin';
import type {
	CancelRunCommand,
	CompleteCallAsErrorCommand,
	ContinueRunCommand,
	CreateSessionCommand,
	DeleteSessionCommand,
	EditAndResendCommand,
	ExecuteAppToolCommand,
	ExecuteCallsCommand,
	OpenAppViewCommand,
	PauseRunCommand,
	ReadAppResourceRequest,
	RenameSessionCommand,
	ResolveWaitCommand,
	ResumeRunCommand,
	RetryCallCommand,
	RunID,
	Session,
	SessionAppResourceReadResult,
	SessionCommandReceipt,
	SessionID,
	SessionRunObserver,
	SessionRunSnapshot,
	SessionRuntimeEvent,
	SessionRuntimeListItem,
	SessionSnapshot,
	SessionSnapshotObserver,
	SessionUserMessageSkillTemplateCommand,
	SessionUserMessageSkillTemplateResult,
	SetAutomationHoldCommand,
	SkipCallCommand,
	StageAppContextCommand,
	StartRunCommand,
	UpdateContinuationInputCommand,
	UpdateSessionConfigurationCommand,
} from '@/spec/session';
import type {
	SessionStoreConversationProjection,
	SessionStoreListSessionRunsRequest,
	SessionStoreListSessionsRequest,
	SessionStoreRunListItem,
	SessionStoreSearchSessionsRequest,
	SessionStoreSessionPage,
} from '@/spec/sessionstore';
import type { AppTheme, DebugSettings, SettingsSchema } from '@/spec/setting';
import type {
	ArtifactSkillFilter,
	ArtifactSkillSummary,
	ManagedSkillCreateRequest,
	ManagedSkillCreateResult,
	ManagedSkillReplaceRequest,
	ManagedSkillReplaceResult,
	ResolvedArtifactSkill,
	SkillDirectoryRegistration,
	SkillPathRegistration,
	SkillPathRegistrationResult,
	StoreManagedSkillDocument,
	StoreSkillListItem,
} from '@/spec/skill';
import type { TextMaterialization } from '@/spec/text';
import type { ResolvedToolView, ToolStoreListItem, ToolView } from '@/spec/tool';
import type { ApplyUnifiedDiffArgs, ApplyUnifiedDiffOut } from '@/spec/unified_diff';
import type {
	WorkspaceArtifactView,
	WorkspaceDefaultPolicyView,
	WorkspaceDirectoryRef,
	WorkspaceDirectoryView,
	WorkspaceMCPServerLoadPlan,
	WorkspacePage,
	WorkspacePageRequest,
	WorkspacePromptPlan,
	WorkspaceRuntimePlan,
	WorkspaceRuntimeSelection,
	WorkspaceSkillLoadPlan,
} from '@/spec/workspace';

export interface ILogger {
	log(...args: unknown[]): void;
	error(...args: unknown[]): void;
	info(...args: unknown[]): void;
	debug(...args: unknown[]): void;
	warn(...args: unknown[]): void;
}

export interface IBackendAPI {
	appQuit: () => void;
	appWindowMinimise: () => void;
	appWindowToggleMaximise: () => void;
	isAppWindowMaximised: () => Promise<boolean>;

	getAppVersion: () => Promise<string>;
	ping: () => Promise<string>;
	pickDirectoryPath: () => Promise<string | undefined>;
	pickFilePaths: (allowMultiple: boolean) => Promise<string[]>;
	log: (level: string, ...args: unknown[]) => void;

	openURL(url: string): void;
	openURLAsAttachment(rawURL: string): Promise<Attachment | undefined>;
	saveFile(defaultFilename: string, contentBase64: string, additionalFilters?: Array<FileFilter>): Promise<void>;
	openMultipleFilesAsAttachments(allowMultiple: boolean, additionalFilters?: Array<FileFilter>): Promise<Attachment[]>;
	openDirectoryAsAttachments(maxFiles: number): Promise<DirectoryAttachmentsResult>;
	getPathsAsAttachments(paths: string[], maxFilesPerDir: number): Promise<PathAttachmentsResult>;

	applyUnifiedDiff(args: ApplyUnifiedDiffArgs): Promise<ApplyUnifiedDiffOut>;
}

export interface ISettingStoreAPI {
	setAppTheme: (theme: AppTheme) => Promise<void>;
	setDebugSettings: (settings: DebugSettings) => Promise<void>;
	getSettings: (forceFetch?: boolean) => Promise<SettingsSchema>;
}

export interface ISessionRuntimeAPI {
	subscribeRuntimeEvents(listener: (event: SessionRuntimeEvent) => void): () => void;

	prepareUserMessageSkillTemplate(
		command: SessionUserMessageSkillTemplateCommand
	): Promise<SessionUserMessageSkillTemplateResult>;

	openAppView(command: OpenAppViewCommand): Promise<SessionCommandReceipt>;
	executeAppTool(command: ExecuteAppToolCommand): Promise<SessionCommandReceipt>;
	stageAppContext(command: StageAppContextCommand): Promise<SessionCommandReceipt>;
	readAppResource(request: ReadAppResourceRequest): Promise<SessionAppResourceReadResult>;

	createSession(command: CreateSessionCommand): Promise<SessionCommandReceipt>;
	updateSessionConfiguration(command: UpdateSessionConfigurationCommand): Promise<SessionCommandReceipt>;
	renameSession(command: RenameSessionCommand): Promise<SessionCommandReceipt>;
	deleteSession(command: DeleteSessionCommand): Promise<SessionCommandReceipt>;

	startRun(command: StartRunCommand): Promise<SessionCommandReceipt>;
	updateContinuationInput(command: UpdateContinuationInputCommand): Promise<SessionCommandReceipt>;
	continueRun(command: ContinueRunCommand): Promise<SessionCommandReceipt>;
	executeCalls(command: ExecuteCallsCommand): Promise<SessionCommandReceipt>;
	retryCall(command: RetryCallCommand): Promise<SessionCommandReceipt>;
	skipCall(command: SkipCallCommand): Promise<SessionCommandReceipt>;
	completeCallAsError(command: CompleteCallAsErrorCommand): Promise<SessionCommandReceipt>;
	resolveWait(command: ResolveWaitCommand): Promise<SessionCommandReceipt>;
	setAutomationHold(command: SetAutomationHoldCommand): Promise<SessionCommandReceipt>;
	pauseRun(command: PauseRunCommand): Promise<SessionCommandReceipt>;
	resumeRun(command: ResumeRunCommand): Promise<SessionCommandReceipt>;
	cancelRun(command: CancelRunCommand): Promise<SessionCommandReceipt>;
	editAndResend(command: EditAndResendCommand): Promise<SessionCommandReceipt>;

	getSessionSnapshot(sessionID: SessionID): Promise<SessionSnapshot>;
	getRunSnapshot(runID: RunID): Promise<SessionRunSnapshot>;
	listRuntimeSessions(): Promise<SessionRuntimeListItem[]>;
}

export interface ISessionStoreAPI {
	readSession(sessionID: SessionID, forceFetch?: boolean): Promise<Session>;
	readConversation(sessionID: SessionID, forceFetch?: boolean): Promise<SessionStoreConversationProjection>;
	readRun(sessionID: SessionID, runID: RunID, forceFetch?: boolean): Promise<SessionRunSnapshot>;
	listSessions(request?: SessionStoreListSessionsRequest): Promise<SessionStoreSessionPage>;
	searchSessions(request: SessionStoreSearchSessionsRequest): Promise<SessionStoreSessionPage>;
	listSessionRuns(request: SessionStoreListSessionRunsRequest): Promise<SessionStoreRunListItem[]>;
	exportSession(sessionID: SessionID): Promise<string>;
}

export interface ISessionManagementAPI extends ISessionRuntimeAPI, ISessionStoreAPI {
	observeSession(sessionID: SessionID, listener: SessionSnapshotObserver): () => void;

	observeRun(sessionID: SessionID, runID: RunID, listener: SessionRunObserver): () => void;

	dispose(): void;
}

export interface IModelStoreAPI {
	listProviders(rootID?: ArtifactRootID): Promise<ModelProviderListItem[]>;

	listModels(rootID?: ArtifactRootID): Promise<ModelListItem[]>;

	getProvider(ref: ArtifactRef): Promise<ModelProviderView>;

	getModel(ref: ArtifactRef): Promise<ModelView>;

	getProviderAPIKeyStatus(ref: ArtifactRef): Promise<ProviderAPIKeyStatus>;

	createModel(request: ManagedModelCreateRequest): Promise<ManagedModelResult>;

	updateModel(request: ManagedModelReplaceRequest): Promise<ManagedModelResult>;

	deleteModel(ref: ArtifactRef, expectedRevision: number): Promise<void>;

	saveModelSettings(request: SaveModelSettingsRequest): Promise<ModelView>;

	setModelEnabled(ref: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact>;

	resetModelSettings(
		ref: ArtifactRef,
		expectedModelRevision: number,
		expectedSettingsRevision: number
	): Promise<ModelView>;
}

export interface IModelAggregateAPI {
	getDefaultProvider(): Promise<ArtifactRef | undefined>;

	setDefaultProvider(provider: ArtifactRef): Promise<void>;

	clearDefaultProvider(): Promise<void>;

	createProvider(request: ManagedProviderCreateRequest): Promise<ManagedProviderResult>;

	updateProvider(request: ManagedProviderReplaceRequest): Promise<ManagedProviderResult>;

	deleteProvider(ref: ArtifactRef, expectedProviderRevision: number): Promise<void>;

	setProviderEnabled(ref: ArtifactRef, expectedProviderRevision: number, enabled: boolean): Promise<StoreArtifact>;

	saveProviderSettings(request: SaveProviderSettingsRequest): Promise<ModelProviderView>;

	resetProviderSettings(
		ref: ArtifactRef,
		expectedProviderRevision: number,
		expectedSettingsRevision: number
	): Promise<ModelProviderView>;

	setProviderAPIKey(request: SetProviderAPIKeyRequest): Promise<ProviderAPIKeyStatus>;

	clearProviderAPIKey(
		ref: ArtifactRef,
		expectedProviderRevision: number,
		expectedAPIKeyRevision: number
	): Promise<ProviderAPIKeyStatus>;
}

export interface IToolStoreAPI {
	listToolPlugins(): Promise<PluginListItem[]>;

	getToolPlugin(plugin: ArtifactRef): Promise<PluginView>;

	listPluginTools(plugin: ArtifactRef): Promise<ToolStoreListItem[]>;

	getTool(tool: ArtifactRef): Promise<ToolView>;

	setToolEnabled(tool: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<ToolView>;

	setToolPluginEnabled(plugin: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<PluginView>;

	mapToolTarget(tool: ArtifactRef): Promise<CapabilityTarget>;

	resolveToolTarget(target: CapabilityTarget): Promise<ResolvedToolView>;
}

export interface IAgentStoreAPI {
	listAgents(request: ListAgentsRequest): Promise<AgentView[]>;

	listAgentsForManagement(): Promise<AgentView[]>;

	getAgent(agent: ArtifactRef): Promise<AgentView>;

	resolveAgent(agent: ArtifactRef): Promise<AgentResolution>;

	resolveAgentCapabilities(agent: ArtifactRef): Promise<AgentCapabilityPlan>;

	setAgentEnabled(agent: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<AgentView>;

	addAgentPluginMember(request: AddPluginMemberRequest): Promise<PluginView>;

	addAgentPluginArtifactMember(request: AddPluginArtifactMemberRequest): Promise<PluginView>;

	removeAgentPluginMember(request: RemovePluginMemberRequest): Promise<PluginView>;

	resolveAgentPluginCapabilities(plugin: ArtifactRef): Promise<PluginCapabilityPlan>;

	listAgentPluginMembers(plugin: ArtifactRef): Promise<PluginDirectMembership>;

	createAgentPlugin(request: CreatePluginRequest): Promise<PluginView>;

	getAgentPlugin(plugin: ArtifactRef): Promise<PluginView>;

	listAgentPlugins(rootID: ArtifactRootID): Promise<PluginListItem[]>;

	listAgentPluginsForManagement(): Promise<PluginListItem[]>;

	updateAgentPlugin(request: UpdatePluginRequest): Promise<PluginView>;

	setAgentPluginEnabled(plugin: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<PluginView>;

	deleteAgentPlugin(plugin: ArtifactRef, expectedRevision: number): Promise<void>;

	listAgentImportDestinations(): Promise<AgentImportDestination[]>;

	previewAgentImport(request: AgentImportPreviewRequest): Promise<AgentImportPreview>;

	commitAgentImport(request: AgentImportCommitRequest): Promise<AgentImportCommitResult>;

	exportAgent(agent: ArtifactRef): Promise<AgentExportResult>;

	deleteManagedAgent(agent: ArtifactRef, expectedRevision: number): Promise<void>;
}

export interface ITextStoreAPI {
	materializeText(text: ArtifactRef): Promise<TextMaterialization>;
}

export interface IToolTargetResolver {
	resolveToolTarget(target: CapabilityTarget): Promise<ResolvedToolView>;
}

export interface ISkillStoreAPI {
	addSkillPluginMember(request: AddPluginMemberRequest): Promise<PluginView>;

	addSkillPath(request: SkillPathRegistration): Promise<SkillPathRegistrationResult>;

	attachSkillArtifactToPlugin(request: AddPluginArtifactMemberRequest): Promise<PluginView>;

	createManagedSkill(request: ManagedSkillCreateRequest): Promise<ManagedSkillCreateResult>;

	replaceManagedSkill(request: ManagedSkillReplaceRequest): Promise<ManagedSkillReplaceResult>;

	createSkillPlugin(request: CreatePluginRequest): Promise<PluginView>;

	deleteSkillPlugin(request: DeletePluginRequest): Promise<void>;

	getManagedSkillDocument(skill: ArtifactRef): Promise<StoreManagedSkillDocument>;

	getSkill(skill: ArtifactRef): Promise<StoreArtifact>;

	getSkillPlugin(plugin: ArtifactRef): Promise<PluginView>;

	listSkillPluginMemberships(skill: ArtifactRef): Promise<ArtifactPluginMembershipView[]>;

	listSkillPlugins(rootID: ArtifactRootID): Promise<PluginListItem[]>;

	listSkillPluginsForManagement(): Promise<PluginListItem[]>;

	listSkills(rootID: ArtifactRootID): Promise<StoreSkillListItem[]>;

	listSkillsForManagement(): Promise<StoreSkillListItem[]>;

	purgeSkill(skill: ArtifactRef, expectedRevision: number): Promise<void>;

	refreshSkillSource(rootID: ArtifactRootID, sourceID: ArtifactSourceID): Promise<void>;

	registerSkillDirectory(request: SkillDirectoryRegistration): Promise<StoreArtifactSourceSummary>;

	removeSkillPluginMember(request: RemovePluginMemberRequest): Promise<PluginView>;

	resolveSkillCapabilities(skill: ArtifactRef): Promise<CapabilityPlan>;

	resolveSkillPlugin(plugin: ArtifactRef): Promise<PluginCapabilityPlan>;

	setSkillPluginEnabled(plugin: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<PluginView>;

	setSkillEnabled(skill: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact>;

	updateSkillPlugin(request: UpdatePluginRequest): Promise<PluginView>;
}

export interface ISkillAggregateAPI {
	resolveArtifactSkill(skill: ArtifactRef): Promise<ResolvedArtifactSkill>;

	resolveArtifactSkills(skills: ArtifactRef[]): Promise<ResolvedArtifactSkill[]>;

	listArtifactSkillRefs(filter: ArtifactSkillFilter): Promise<ArtifactRef[]>;

	describeArtifactSkill(skill: ArtifactRef): Promise<ArtifactSkillSummary>;
	getArtifactSkillsPrompt(filter: ArtifactSkillFilter): Promise<string>;
}

export interface IWorkspaceStoreAPI {
	getWorkspaceDefaultPolicy(): Promise<WorkspaceDefaultPolicyView>;

	getWorkspaceDirectory(directory: WorkspaceDirectoryRef): Promise<WorkspaceDirectoryView>;

	listWorkspaceDirectories(request: WorkspacePageRequest): Promise<WorkspacePage>;

	listWorkspaceDirectoryArtifacts(directory: WorkspaceDirectoryRef): Promise<WorkspaceArtifactView[]>;

	refreshWorkspaceDirectory(directory: WorkspaceDirectoryRef): Promise<WorkspaceDirectoryView>;

	registerWorkspaceDirectory(path: string): Promise<WorkspaceDirectoryView>;

	removeWorkspaceDirectory(directory: WorkspaceDirectoryRef, expectedRevision: number): Promise<void>;

	setWorkspaceDirectoryArtifactEnabled(
		directory: WorkspaceDirectoryRef,
		artifact: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<WorkspaceArtifactView>;

	setWorkspaceDirectoryEnabled(
		directory: WorkspaceDirectoryRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<WorkspaceDirectoryView>;
}

export interface IWorkspaceRuntimeAPI {
	composeWorkspacePrompt(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspacePromptPlan>;

	loadWorkspaceMCPServers(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspaceMCPServerLoadPlan>;

	loadWorkspaceSkills(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspaceSkillLoadPlan>;

	resolveWorkspaceRuntimePlan(
		workspace: ArtifactRef,
		selection: WorkspaceRuntimeSelection
	): Promise<WorkspaceRuntimePlan>;
}

export interface IWorkspaceManagementAPI {
	getWorkspaceDefaultPolicy(): Promise<WorkspaceDefaultPolicyView>;

	getWorkspaceDirectory(directory: WorkspaceDirectoryRef): Promise<WorkspaceDirectoryView>;

	listWorkspaceDirectories(request: WorkspacePageRequest): Promise<WorkspacePage>;

	listWorkspaceDirectoryArtifacts(directory: WorkspaceDirectoryRef): Promise<WorkspaceArtifactView[]>;

	refreshWorkspaceDirectory(directory: WorkspaceDirectoryRef): Promise<WorkspaceDirectoryView>;

	registerWorkspaceDirectory(path: string): Promise<WorkspaceDirectoryView>;

	removeWorkspaceDirectory(directory: WorkspaceDirectoryRef, expectedRevision: number): Promise<void>;

	setWorkspaceDirectoryArtifactEnabled(
		directory: WorkspaceDirectoryRef,
		artifact: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<WorkspaceArtifactView>;

	setWorkspaceDirectoryEnabled(
		directory: WorkspaceDirectoryRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<WorkspaceDirectoryView>;

	composeWorkspacePrompt(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspacePromptPlan>;

	loadWorkspaceMCPServers(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspaceMCPServerLoadPlan>;

	loadWorkspaceSkills(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspaceSkillLoadPlan>;

	resolveWorkspaceRuntimePlan(
		workspace: ArtifactRef,
		selection: WorkspaceRuntimeSelection
	): Promise<WorkspaceRuntimePlan>;
}

export interface IConversationStoreAPI {
	putConversation: (conversation: StoreConversation) => Promise<void>;
	putMessagesToConversation(id: string, title: string, messages: StoreConversationMessage[]): Promise<void>;
	deleteConversation: (id: string, title: string) => Promise<void>;
	getConversation: (id: string, title: string, forceFetch?: boolean) => Promise<StoreConversation | null>;
	listConversations: (
		token?: string,
		pageSize?: number
	) => Promise<{ conversations: ConversationSearchItem[]; nextToken?: string }>;
	searchConversations: (
		query: string,
		token?: string,
		pageSize?: number
	) => Promise<{ conversations: ConversationSearchItem[]; nextToken?: string }>;
}

export interface IAttachmentsDropAPI {
	/**
	 * Must be idempotent. Registers the underlying platform event listener. returns cleanup func.
	 */
	startListener(): () => void;

	/**
	 * Sets the current active target (e.g. the chat composer).
	 * Returns an unregister function.
	 */
	registerDropTarget(fn: (payload: AttachmentsDroppedPayload) => void): () => void;

	/**
	 * Called when a drop happens but there is no active target yet.
	 * Useful to navigate to /chats and let pending drops flush.
	 */
	setNoTargetHandler(fn: ((payload: AttachmentsDroppedPayload) => void) | null): void;
}

export interface IMCPStoreAPI {
	addMCPPluginMember(request: AddPluginMemberRequest): Promise<PluginView>;

	removeMCPPluginMember(request: RemovePluginMemberRequest): Promise<PluginView>;

	addMCPServerToPlugin(request: AddPluginArtifactMemberRequest): Promise<PluginView>;

	createMCPPlugin(request: CreatePluginRequest): Promise<PluginView>;

	deleteMCPPlugin(request: DeletePluginRequest): Promise<void>;

	getMCPPlugin(plugin: ArtifactRef): Promise<PluginView>;
	getMCPPolicy(policy: ArtifactRef): Promise<MCPPolicyView>;
	getMCPServerSecrets(server: ArtifactRef): Promise<MCPServerSecretsView>;
	getMCPSettings(): Promise<MCPSettings>;

	listMCPPluginMemberships(artifact: ArtifactRef): Promise<ArtifactPluginMembershipView[]>;
	listMCPPlugins(rootID: ArtifactRootID): Promise<PluginListItem[]>;
	listMCPPolicies(rootID: ArtifactRootID): Promise<MCPPolicyListItem[]>;
	listMCPServers(rootID: ArtifactRootID): Promise<MCPServerListItem[]>;
	listMCPPluginsPage(pageSize: number, pageToken?: string): Promise<MCPPluginPage>;
	listMCPServersPage(pageSize: number, pageToken?: string): Promise<MCPServerPage>;

	setMCPPluginEnabled(plugin: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<PluginView>;
	saveMCPSettings(expectedRevision: number, settings: MCPAuthSettings): Promise<MCPSettings>;
	updateMCPPlugin(request: UpdatePluginRequest): Promise<PluginView>;
}

export interface IMCPAggregateAPI {
	listMCPPluginServers(plugin: ArtifactRef): Promise<MCPServerDetails[]>;

	getMCPServersForRuntimeServers(servers: MCPRuntimeServerID[]): Promise<MCPServerRuntimeDetails[]>;

	getMCPServer(server: ArtifactRef): Promise<MCPServerDetails>;
	createMCPServer(request: ManagedMCPCreateRequest): Promise<ManagedMCPCreateResult>;
	updateMCPServer(request: ManagedMCPReplaceRequest): Promise<ManagedMCPReplaceResult>;
	deleteMCPServer(server: ArtifactRef, expectedRevision: number): Promise<void>;
	saveMCPPolicy(request: ManagedMCPPolicyUpsertRequest): Promise<ManagedMCPPolicyUpsertResult>;
	deleteMCPPolicy(policy: ArtifactRef, expectedRevision: number): Promise<void>;
	saveMCPServerSettings(
		server: ArtifactRef,
		expectedSettingsRevision: number,
		data: MCPServerData
	): Promise<MCPServerDetails>;
	setMCPServerSecret(server: ArtifactRef, input: string, secret: string): Promise<MCPServerDetails>;
	clearMCPServerSecret(server: ArtifactRef, input: string): Promise<MCPServerDetails>;
}

export interface IMCPRuntimeAPI {
	cancelMCPServerAuthorization(server: MCPRuntimeServerID): Promise<boolean>;
	completeMCPArgument(
		server: MCPRuntimeServerID,
		request: MCPCompleteArgumentRequestBody
	): Promise<MCPCompletionResult>;

	connectMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot>;

	disconnectMCPServer(server: MCPRuntimeServerID): Promise<void>;

	getMCPPrompt(
		server: MCPRuntimeServerID,
		promptName: string,
		promptArguments: Record<string, string>
	): Promise<MCPGetPromptResponseBody>;

	listMCPServerPrompts(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPPromptRef>>;

	listMCPServerResources(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPResourceRef>>;

	listMCPServerResourceTemplates(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPResourceTemplateRef>>;

	listMCPServerTools(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPToolCapability>>;

	readMCPResource(server: MCPRuntimeServerID, uri: string): Promise<MCPReadResourceResponseBody>;
	refreshMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot>;
}
