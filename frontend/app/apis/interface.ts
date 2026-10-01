import type {
	AgentCapabilityPlan,
	AgentExportResult,
	AgentImportCommitRequest,
	AgentImportCommitResult,
	AgentImportDestination,
	AgentImportPreview,
	AgentImportPreviewRequest,
	AgentResolution,
	AgentTextMaterialization,
	AgentView,
	ListAgentsRequest,
} from '@/spec/agent';
import type {
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceID,
	CapabilityPlan,
	MappedTarget,
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
import type {
	AddArtifactMemberRequest,
	AddMemberRequest,
	ArtifactMembershipView,
	CollectionCapabilityPlan,
	CollectionListItem,
	CollectionView,
	CreateCollectionRequest,
	DeleteCollectionRequest,
	RemoveMemberRequest,
	UpdateCollectionRequest,
} from '@/spec/collection';
import type { ConversationSearchItem, StoreConversation, StoreConversationMessage } from '@/spec/conversation';
import type { CompletionResponseBody, ToolChoice } from '@/spec/inference';
import type {
	InvokeMCPToolRequestBody,
	ManagedMCPCreateRequest,
	ManagedMCPCreateResult,
	ManagedMCPPolicyUpsertRequest,
	ManagedMCPPolicyUpsertResult,
	ManagedMCPReplaceRequest,
	ManagedMCPReplaceResult,
	MCPApprovalEvaluation,
	MCPApprovalResolution,
	MCPApprovalResolutionResult,
	MCPAuthSettings,
	MCPCompleteArgumentRequestBody,
	MCPCompletionResult,
	MCPConversationContext,
	MCPDiscoveryPage,
	MCPGetPromptResponseBody,
	MCPManagementPage,
	MCPPolicyListItem,
	MCPPromptRef,
	MCPReadResourceResponseBody,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPRuntimeInvokeToolResponse,
	MCPRuntimeServerID,
	MCPServerAggregateDetails,
	MCPServerData,
	MCPServerListItem,
	MCPServerRuntimeSnapshot,
	MCPServerSecretsView,
	MCPSettings,
	MCPStorePolicyView,
	MCPStoreServerInstallationView,
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
	ModelRequestPatch,
	ModelView,
	ProviderAPIKeyStatus,
	SaveModelSettingsRequest,
	SaveProviderSettingsRequest,
	SetProviderAPIKeyRequest,
} from '@/spec/model';
import type { AppTheme, DebugSettings, SettingsSchema } from '@/spec/setting';
import type {
	ArtifactSkillFilter,
	ArtifactSkillSummary,
	InvokeSkillToolResponse,
	ManagedSkillCreateRequest,
	ManagedSkillCreateResult,
	ManagedSkillReplaceRequest,
	ManagedSkillReplaceResult,
	ResolvedArtifactSkill,
	RuntimeSkillDefinition,
	RuntimeSkillListFilter,
	RuntimeSkillPromptFilter,
	RuntimeSkillRecord,
	RuntimeSkillRenderResult,
	RuntimeSkillSession,
	RuntimeSkillSessionOptions,
	SkillDirectoryRegistration,
	SkillPathRegistration,
	SkillPathRegistrationResult,
	StoreManagedSkillDocument,
	StoreSkillListItem,
} from '@/spec/skill';
import type { ResolvedToolView, ToolSelection, ToolStoreListItem, ToolView } from '@/spec/tool';
import type { InvokeToolResponse } from '@/spec/toolruntime';
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

import type { JSONRawString } from '@/lib/jsonschema_utils';

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

export interface ICompletionAPI {
	fetchCompletion(
		model: ArtifactRef,
		requestPatch: ModelRequestPatch | undefined,
		current: StoreConversationMessage,
		history?: StoreConversationMessage[],
		toolSelections?: ToolSelection[],
		mcpContext?: MCPConversationContext,
		skillSessionID?: string,
		requestID?: string,
		signal?: AbortSignal,
		onStreamTextData?: (text: string) => void,
		onStreamThinkingData?: (thinking: string) => void
	): Promise<CompletionResponseBody | undefined>;

	cancelCompletion(requestID: string): Promise<void>;
}

export interface IToolRuntimeAPI {
	invokeTool(functionName: string, args?: JSONRawString, timeoutMS?: number): Promise<InvokeToolResponse>;
}

export interface IToolStoreAPI {
	listToolCollections(): Promise<CollectionListItem[]>;

	getToolCollection(collection: ArtifactRef): Promise<CollectionView>;

	listCollectionTools(collection: ArtifactRef): Promise<ToolStoreListItem[]>;

	getTool(tool: ArtifactRef): Promise<ToolView>;

	setToolEnabled(tool: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<ToolView>;

	setToolCollectionEnabled(
		collection: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<CollectionView>;
}

export interface IToolTargetResolver {
	resolveMappedTool(target: MappedTarget): Promise<ResolvedToolView>;
}

export interface IToolAggregateAPI extends IToolTargetResolver {
	mapToolTarget(tool: ArtifactRef): Promise<MappedTarget>;

	invokeMappedTool(target: MappedTarget, args?: JSONRawString, timeoutMS?: number): Promise<InvokeToolResponse>;

	hydrateInferenceToolChoice(selection: ToolSelection): Promise<ToolChoice>;
}

export interface IAgentStoreAPI {
	listAgents(request: ListAgentsRequest): Promise<AgentView[]>;

	listAgentsForManagement(): Promise<AgentView[]>;

	getAgent(agent: ArtifactRef): Promise<AgentView>;

	materializeAgentText(text: ArtifactRef): Promise<AgentTextMaterialization>;

	resolveAgent(agent: ArtifactRef): Promise<AgentResolution>;

	resolveAgentCapabilities(agent: ArtifactRef): Promise<AgentCapabilityPlan>;

	setAgentEnabled(agent: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<AgentView>;

	listAgentCollectionMembers(collection: ArtifactRef): Promise<CollectionCapabilityPlan>;

	createAgentCollection(request: CreateCollectionRequest): Promise<CollectionView>;

	getAgentCollection(collection: ArtifactRef): Promise<CollectionView>;

	listAgentCollections(rootID: ArtifactRootID): Promise<CollectionListItem[]>;

	listAgentCollectionsForManagement(): Promise<CollectionListItem[]>;

	updateAgentCollection(request: UpdateCollectionRequest): Promise<CollectionView>;

	setAgentCollectionEnabled(
		collection: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<CollectionView>;

	deleteAgentCollection(collection: ArtifactRef, expectedRevision: number): Promise<void>;

	listAgentImportDestinations(): Promise<AgentImportDestination[]>;

	previewAgentImport(request: AgentImportPreviewRequest): Promise<AgentImportPreview>;

	commitAgentImport(request: AgentImportCommitRequest): Promise<AgentImportCommitResult>;

	exportAgent(agent: ArtifactRef): Promise<AgentExportResult>;

	deleteManagedAgent(agent: ArtifactRef, expectedRevision: number): Promise<void>;
}

export interface ISkillStoreAPI {
	addSkillCollectionMember(request: AddMemberRequest): Promise<CollectionView>;

	addSkillPath(request: SkillPathRegistration): Promise<SkillPathRegistrationResult>;

	attachSkillArtifactToCollection(request: AddArtifactMemberRequest): Promise<CollectionView>;

	createManagedSkill(request: ManagedSkillCreateRequest): Promise<ManagedSkillCreateResult>;

	replaceManagedSkill(request: ManagedSkillReplaceRequest): Promise<ManagedSkillReplaceResult>;

	createSkillCollection(request: CreateCollectionRequest): Promise<CollectionView>;

	deleteSkillCollection(request: DeleteCollectionRequest): Promise<void>;

	getManagedSkillDocument(skill: ArtifactRef): Promise<StoreManagedSkillDocument>;

	getSkill(skill: ArtifactRef): Promise<StoreArtifact>;

	getSkillCollection(collection: ArtifactRef): Promise<CollectionView>;

	listSkillCollectionMemberships(skill: ArtifactRef): Promise<ArtifactMembershipView[]>;

	listSkillCollections(rootID: ArtifactRootID): Promise<CollectionListItem[]>;

	listSkillCollectionsForManagement(): Promise<CollectionListItem[]>;

	listSkills(rootID: ArtifactRootID): Promise<StoreSkillListItem[]>;

	listSkillsForManagement(): Promise<StoreSkillListItem[]>;

	purgeSkill(skill: ArtifactRef, expectedRevision: number): Promise<void>;

	refreshSkillSource(rootID: ArtifactRootID, sourceID: ArtifactSourceID): Promise<void>;

	registerSkillDirectory(request: SkillDirectoryRegistration): Promise<StoreArtifactSourceSummary>;

	removeSkillCollectionMember(request: RemoveMemberRequest): Promise<CollectionView>;

	resolveSkillCapabilities(skill: ArtifactRef): Promise<CapabilityPlan>;

	resolveSkillCollection(collection: ArtifactRef): Promise<CollectionCapabilityPlan>;

	setSkillCollectionEnabled(
		collection: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<CollectionView>;

	setSkillEnabled(skill: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact>;

	updateSkillCollection(request: UpdateCollectionRequest): Promise<CollectionView>;
}

export interface ISkillAggregateAPI {
	resolveArtifactSkill(skill: ArtifactRef): Promise<ResolvedArtifactSkill>;

	resolveArtifactSkills(skills: ArtifactRef[]): Promise<ResolvedArtifactSkill[]>;

	listArtifactSkillRefs(filter: ArtifactSkillFilter): Promise<ArtifactRef[]>;

	describeArtifactSkill(skill: ArtifactRef): Promise<ArtifactSkillSummary>;
}

export interface ISkillRuntimeAPI {
	createSkillSession(options: RuntimeSkillSessionOptions): Promise<RuntimeSkillSession>;

	closeSkillSession(sessionID: string): Promise<void>;

	getSkillsPrompt(filter?: RuntimeSkillPromptFilter): Promise<string>;

	listRuntimeSkills(filter?: RuntimeSkillListFilter): Promise<RuntimeSkillRecord[]>;

	renderSkill(definition: RuntimeSkillDefinition, args?: Record<string, string>): Promise<RuntimeSkillRenderResult>;

	invokeSkillTool(sessionID: string, toolName: string, args?: JSONRawString): Promise<InvokeSkillToolResponse>;
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
	addMCPCollectionMember(request: AddMemberRequest): Promise<CollectionView>;

	removeMCPCollectionMember(request: RemoveMemberRequest): Promise<CollectionView>;

	addMCPServerToCollection(request: AddArtifactMemberRequest): Promise<CollectionView>;

	createMCPCollection(request: CreateCollectionRequest): Promise<CollectionView>;

	deleteMCPCollection(request: DeleteCollectionRequest): Promise<void>;

	getMCPCollection(collection: ArtifactRef): Promise<CollectionView>;
	getMCPPolicy(policy: ArtifactRef): Promise<MCPStorePolicyView>;
	getMCPServerSecrets(server: ArtifactRef): Promise<MCPServerSecretsView>;
	getMCPSettings(): Promise<MCPSettings>;

	listMCPCollectionServers(collection: ArtifactRef): Promise<
		Array<{
			installation: MCPStoreServerInstallationView;
		}>
	>;

	listMCPCollectionMemberships(artifact: ArtifactRef): Promise<ArtifactMembershipView[]>;
	listMCPCollections(rootID: ArtifactRootID): Promise<CollectionListItem[]>;
	listMCPPolicies(rootID: ArtifactRootID): Promise<MCPPolicyListItem[]>;
	listMCPServers(rootID: ArtifactRootID): Promise<MCPServerListItem[]>;
	listMCPCollectionsPage(pageSize: number, pageToken?: string): Promise<MCPManagementPage<CollectionListItem>>;
	listMCPServersPage(pageSize: number, pageToken?: string): Promise<MCPManagementPage<MCPServerListItem>>;

	setMCPCollectionEnabled(collection: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<CollectionView>;
	saveMCPSettings(expectedRevision: number, settings: MCPAuthSettings): Promise<MCPSettings>;
	updateMCPCollection(request: UpdateCollectionRequest): Promise<CollectionView>;
}

export interface IMCPAggregateAPI {
	getMCPServer(server: ArtifactRef): Promise<MCPServerAggregateDetails>;
	getMCPServerForRuntimeServer(server: MCPRuntimeServerID): Promise<MCPServerAggregateDetails>;
	// Management projects this into MCPRuntimeServerView.
	createMCPServer(request: ManagedMCPCreateRequest): Promise<ManagedMCPCreateResult>;
	updateMCPServer(request: ManagedMCPReplaceRequest): Promise<ManagedMCPReplaceResult>;
	deleteMCPServer(server: ArtifactRef, expectedRevision: number): Promise<void>;
	saveMCPPolicy(request: ManagedMCPPolicyUpsertRequest): Promise<ManagedMCPPolicyUpsertResult>;
	deleteMCPPolicy(policy: ArtifactRef, expectedRevision: number): Promise<void>;
	saveMCPServerSettings(
		server: ArtifactRef,
		expectedSettingsRevision: number,
		data: MCPServerData
	): Promise<MCPServerAggregateDetails>;
	setMCPServerSecret(server: ArtifactRef, input: string, secret: string): Promise<MCPServerAggregateDetails>;
	clearMCPServerSecret(server: ArtifactRef, input: string): Promise<MCPServerAggregateDetails>;
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

	invokeMCPTool(server: MCPRuntimeServerID, request: InvokeMCPToolRequestBody): Promise<MCPRuntimeInvokeToolResponse>;

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
	checkMCPToolCall(server: MCPRuntimeServerID, request: InvokeMCPToolRequestBody): Promise<MCPApprovalEvaluation>;
	resolveMCPToolApproval(approvalID: string, resolution: MCPApprovalResolution): Promise<MCPApprovalResolutionResult>;
}
