import type {
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceID,
	CapabilityPlan,
	MappedTarget,
	StoreArtifact,
	StoreArtifactRoot,
	StoreArtifactRootDraft,
	StoreArtifactSourceSummary,
} from '@/spec/artifact';
import type {
	AssistantPreset,
	AssistantPresetBundle,
	AssistantPresetListItem,
	PutAssistantPresetPayload,
} from '@/spec/assistantpreset';
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
	CollectionView,
	CreateCollectionRequest,
	DeleteCollectionRequest,
	RemoveMemberRequest,
	UpdateCollectionRequest,
} from '@/spec/collection';
import type { ConversationSearchItem, StoreConversation, StoreConversationMessage } from '@/spec/conversation';
import type { CompletionResponseBody, ModelParam, ProviderName } from '@/spec/inference';
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
	MCPAuthHealth,
	MCPAuthSettings,
	MCPCompleteArgumentRequestBody,
	MCPCompletionResult,
	MCPConversationContext,
	MCPDiscoveryPage,
	MCPEffectivePolicy,
	MCPGetPromptResponseBody,
	MCPGlobalSettings,
	MCPManagementPage,
	MCPOAuthAuthorization,
	MCPPromptRef,
	MCPProviderToolMapping,
	MCPReadResourceResponseBody,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPRuntimeInvokeToolResponse,
	MCPRuntimeServerID,
	MCPSecretKind,
	MCPSecretWriteResult,
	MCPServerData,
	MCPServerRuntimeSnapshot,
	MCPStorePolicyView,
	MCPStoreServerInstallationView,
	MCPToolCapability,
} from '@/spec/mcp';
import type {
	ModelPresetID,
	ModelPresetRef,
	PatchModelPresetPayload,
	PatchProviderPresetPayload,
	PostModelPresetPayload,
	PostProviderPresetPayload,
	ProviderPreset,
} from '@/spec/modelpreset';
import type { AppTheme, AuthKey, AuthKeyName, AuthKeyType, DebugSettings, SettingsSchema } from '@/spec/setting';
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
} from '@/spec/skill';
import type { HTTPToolImpl, Tool, ToolBundle, ToolImplType, ToolListItem, ToolRef, ToolStoreChoice } from '@/spec/tool';
import type { InvokeGoOptions, InvokeHTTPOptions, InvokeToolResponse } from '@/spec/toolruntime';
import type { ApplyUnifiedDiffArgs, ApplyUnifiedDiffOut } from '@/spec/unified_diff';
import type {
	FilesystemSourceRegistration,
	WorkspaceArtifactView as StoreWorkspaceArtifactView,
	Workspace,
	WorkspaceLoad,
	WorkspaceMCPServerLoadPlan,
	WorkspacePathRegistration,
	WorkspacePathRegistrationResult,
	WorkspacePromptPlan,
	WorkspaceRefresh,
	WorkspaceRuntimePlan,
	WorkspaceRuntimeSelection,
	WorkspaceSkill,
	WorkspaceSkillLoadPlan,
} from '@/spec/workspace';

import type { JSONRawString, JSONSchema } from '@/lib/jsonschema_utils';

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
}

export interface ISettingStoreAPI {
	setAppTheme: (theme: AppTheme) => Promise<void>;
	setDebugSettings: (settings: DebugSettings) => Promise<void>;
	getAuthKey: (type: AuthKeyType, keyName: AuthKeyName) => Promise<AuthKey>;
	getSettings: (forceFetch?: boolean) => Promise<SettingsSchema>;
}

export interface IModelPresetStoreAPI {
	getDefaultProvider(): Promise<ProviderName>;

	patchDefaultProvider(providerName: ProviderName): Promise<void>;

	patchProviderPreset(providerName: ProviderName, payload: PatchProviderPresetPayload): Promise<void>;

	postModelPreset(
		providerName: ProviderName,
		modelPresetID: ModelPresetID,
		payload: PostModelPresetPayload
	): Promise<void>;

	patchModelPreset(
		providerName: ProviderName,
		modelPresetID: ModelPresetID,
		payload: PatchModelPresetPayload
	): Promise<void>;

	deleteModelPreset(providerName: ProviderName, modelPresetID: ModelPresetID): Promise<void>;

	listProviderPresets(
		names?: ProviderName[],
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ providers: ProviderPreset[]; nextPageToken?: string }>;

	resolveMappedModelTarget(target: MappedTarget): Promise<ModelPresetRef>;
}

export interface IToolStoreAPI {
	/** List tool bundles, optionally filtered by IDs, disabled, and paginated. */
	listToolBundles(
		bundleIDs?: string[],
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ toolBundles: ToolBundle[]; nextPageToken?: string }>;

	/** Create or update a tool bundle. */
	putToolBundle(
		bundleID: string,
		slug: string,
		displayName: string,
		isEnabled: boolean,
		description?: string
	): Promise<void>;

	/** Patch (enable/disable) a tool bundle. */
	patchToolBundle(bundleID: string, isEnabled: boolean): Promise<void>;

	/** Delete a tool bundle. */
	deleteToolBundle(bundleID: string): Promise<void>;

	/** List tools, optionally filtered by bundleIDs, tags, etc. */
	listTools(
		bundleIDs?: string[],
		tags?: string[],
		includeDisabled?: boolean,
		recommendedPageSize?: number,
		pageToken?: string
	): Promise<{ toolListItems: ToolListItem[]; nextPageToken?: string }>;

	/** Create or update a tool. */
	putTool(
		bundleID: string,
		toolSlug: string,
		version: string,
		displayName: string,
		isEnabled: boolean,
		userCallable: boolean,
		llmCallable: boolean,
		autoExecute: boolean,
		argSchema: JSONSchema,
		type: ToolImplType,
		httpImpl?: HTTPToolImpl,
		description?: string,
		tags?: string[]
	): Promise<void>;

	/** Patch (enable/disable) a tool version. */
	patchTool(bundleID: string, toolSlug: string, version: string, isEnabled: boolean): Promise<void>;

	/** Delete a tool version. */
	deleteTool(bundleID: string, toolSlug: string, version: string): Promise<void>;

	/** Get a tool version. */
	getTool(bundleID: string, toolSlug: string, version: string): Promise<Tool | undefined>;

	resolveMappedToolTarget(target: MappedTarget): Promise<ToolRef>;
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

	listSkillCollections(rootID: ArtifactRootID): Promise<CollectionView[]>;

	listSkillCollectionsForManagement(): Promise<CollectionView[]>;

	listSkills(rootID: ArtifactRootID): Promise<StoreArtifact[]>;

	listSkillsForManagement(): Promise<StoreArtifact[]>;

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
	addWorkspacePath(request: WorkspacePathRegistration): Promise<WorkspacePathRegistrationResult>;

	createWorkspaceRoot(request: StoreArtifactRootDraft): Promise<StoreArtifactRoot>;

	getWorkspace(workspace: ArtifactRef): Promise<Workspace>;

	listWorkspaceArtifacts(workspace: ArtifactRef): Promise<StoreArtifact[]>;

	listWorkspaceRoots(): Promise<StoreArtifactRoot[]>;

	listWorkspaces(rootID: ArtifactRootID): Promise<Workspace[]>;

	loadWorkspace(workspace: ArtifactRef): Promise<WorkspaceLoad>;

	refreshWorkspace(workspace: ArtifactRef): Promise<WorkspaceRefresh>;

	registerFilesystemWorkspaceSource(request: FilesystemSourceRegistration): Promise<StoreArtifactSourceSummary>;

	resolveWorkspaceArtifactCapabilities(artifact: ArtifactRef): Promise<CapabilityPlan>;

	resolveWorkspaceCapabilities(workspace: ArtifactRef): Promise<CapabilityPlan>;

	setWorkspaceArtifactEnabled(
		workspace: ArtifactRef,
		artifact: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<StoreWorkspaceArtifactView>;
}

export interface IWorkspaceRuntimeAPI {
	composeWorkspacePrompt(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspacePromptPlan>;

	listWorkspaceSkills(workspace: ArtifactRef): Promise<WorkspaceSkill[]>;

	loadWorkspaceMCPServers(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspaceMCPServerLoadPlan>;

	loadWorkspaceSkills(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspaceSkillLoadPlan>;

	resolveWorkspaceRuntimePlan(
		workspace: ArtifactRef,
		selection: WorkspaceRuntimeSelection
	): Promise<WorkspaceRuntimePlan>;
}

export interface IWorkspaceAggregateAPI {
	setWorkspaceArtifactRuntimeDisabled(
		workspace: ArtifactRef,
		artifact: ArtifactRef,
		expectedRevision: number,
		runtimeDisabled: boolean
	): Promise<StoreWorkspaceArtifactView>;
}

export interface IToolRuntimeAPI {
	/** Invoke a tool version. */
	invokeTool(
		bundleID: string,
		toolSlug: string,
		version: string,
		args?: JSONRawString,
		httpOptions?: InvokeHTTPOptions,
		goOptions?: InvokeGoOptions
	): Promise<InvokeToolResponse>;
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

export interface IAggregateAPI {
	applyUnifiedDiff(args: ApplyUnifiedDiffArgs): Promise<ApplyUnifiedDiffOut>;

	postProviderPreset(providerName: ProviderName, payload: PostProviderPresetPayload): Promise<void>;
	deleteProviderPreset(providerName: ProviderName): Promise<void>;

	deleteAuthKey: (type: AuthKeyType, keyName: AuthKeyName) => Promise<void>;
	setAuthKey: (type: AuthKeyType, keyName: AuthKeyName, secret: string) => Promise<void>;

	fetchCompletion(
		provider: ProviderName,
		modelPresetID: ModelPresetID,
		modelParams: ModelParam,
		current: StoreConversationMessage,
		history?: StoreConversationMessage[],
		toolStoreChoices?: ToolStoreChoice[],
		mcpContext?: MCPConversationContext,
		skillSessionID?: string,
		requestId?: string,
		signal?: AbortSignal,
		onStreamTextData?: (textData: string) => void,
		onStreamThinkingData?: (thinkingData: string) => void
	): Promise<CompletionResponseBody | undefined>;

	cancelCompletion(requestId: string): Promise<void>;
}

export interface IAssistantPresetStoreAPI {
	/** List assistant preset bundles, optionally filtered by IDs, disabled, and paginated. */
	listAssistantPresetBundles(
		bundleIDs?: string[],
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ assistantPresetBundles: AssistantPresetBundle[]; nextPageToken?: string }>;

	/** Create or update an assistant preset bundle. */
	putAssistantPresetBundle(
		bundleID: string,
		slug: string,
		displayName: string,
		isEnabled: boolean,
		description?: string
	): Promise<void>;

	/** Patch (enable/disable) an assistant preset bundle. */
	patchAssistantPresetBundle(bundleID: string, isEnabled: boolean): Promise<void>;

	/** Delete an assistant preset bundle. */
	deleteAssistantPresetBundle(bundleID: string): Promise<void>;

	/** List assistant presets, optionally filtered by bundle IDs and paginated. */
	listAssistantPresets(
		bundleIDs?: string[],
		includeDisabled?: boolean,
		recommendedPageSize?: number,
		pageToken?: string
	): Promise<{ assistantPresetListItems: AssistantPresetListItem[]; nextPageToken?: string }>;

	/** Create or update an assistant preset version. */
	putAssistantPreset(
		bundleID: string,
		assistantPresetSlug: string,
		version: string,
		payload: PutAssistantPresetPayload
	): Promise<void>;

	/** Patch (enable/disable) an assistant preset version. */
	patchAssistantPreset(
		bundleID: string,
		assistantPresetSlug: string,
		version: string,
		isEnabled: boolean
	): Promise<void>;

	/** Delete an assistant preset version. */
	deleteAssistantPreset(bundleID: string, assistantPresetSlug: string, version: string): Promise<void>;

	/** Get an assistant preset version. */
	getAssistantPreset(
		bundleID: string,
		assistantPresetSlug: string,
		version: string
	): Promise<AssistantPreset | undefined>;
}

export interface IMCPStoreAPI {
	addMCPCollectionMember(request: AddMemberRequest): Promise<CollectionView>;

	removeMCPCollectionMember(request: RemoveMemberRequest): Promise<CollectionView>;

	attachMCPArtifactToCollection(request: AddArtifactMemberRequest): Promise<CollectionView>;

	createMCPCollection(request: CreateCollectionRequest): Promise<CollectionView>;

	deleteMCPCollection(request: DeleteCollectionRequest): Promise<void>;

	getMCPCollection(collection: ArtifactRef): Promise<CollectionView>;

	getMCPPolicy(policy: ArtifactRef): Promise<MCPStorePolicyView>;

	getMCPServerInstallation(server: ArtifactRef): Promise<MCPStoreServerInstallationView>;

	listMCPCollectionMemberships(artifact: ArtifactRef): Promise<ArtifactMembershipView[]>;

	listMCPCollections(rootID: ArtifactRootID): Promise<CollectionView[]>;

	listMCPPolicies(rootID: ArtifactRootID): Promise<StoreArtifact[]>;

	listMCPServers(rootID: ArtifactRootID): Promise<StoreArtifact[]>;

	listMCPCollectionsPage(pageSize: number, pageToken?: string): Promise<MCPManagementPage<CollectionView>>;

	listMCPServersPage(pageSize: number, pageToken?: string): Promise<MCPManagementPage<StoreArtifact>>;

	resolveMCPArtifactCapabilities(artifact: ArtifactRef): Promise<CapabilityPlan>;

	resolveMCPCollection(collection: ArtifactRef): Promise<CollectionCapabilityPlan>;

	setMCPCollectionEnabled(collection: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<CollectionView>;

	setMCPPolicyEnabled(policy: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact>;

	setMCPServerEnabled(server: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact>;

	updateMCPCollection(request: UpdateCollectionRequest): Promise<CollectionView>;
}

export interface IMCPAggregateAPI {
	getMCPEffectivePolicy(server: ArtifactRef): Promise<MCPEffectivePolicy>;

	createManagedMCP(request: ManagedMCPCreateRequest): Promise<ManagedMCPCreateResult>;

	replaceManagedMCP(request: ManagedMCPReplaceRequest): Promise<ManagedMCPReplaceResult>;

	purgeManagedMCP(server: ArtifactRef, expectedRevision: number): Promise<void>;

	upsertManagedMCPPolicy(request: ManagedMCPPolicyUpsertRequest): Promise<ManagedMCPPolicyUpsertResult>;

	purgeManagedMCPPolicy(policy: ArtifactRef, expectedRevision: number): Promise<void>;

	artifactRefForRuntimeServerID(server: MCPRuntimeServerID): Promise<ArtifactRef>;

	deleteMCPServerSecret(server: ArtifactRef, kind: MCPSecretKind, slot: string): Promise<void>;

	getMCPServerAuthHealth(server: ArtifactRef): Promise<MCPAuthHealth>;

	putMCPServerSecret(
		server: ArtifactRef,
		kind: MCPSecretKind,
		slot: string,
		secret: string
	): Promise<MCPSecretWriteResult>;

	rootIDForRuntimeCatalogID(catalogID: string): Promise<ArtifactRootID>;

	runtimeServerIDForArtifact(artifact: ArtifactRef): Promise<MCPRuntimeServerID>;

	updateMCPServerInstallation(
		server: ArtifactRef,
		expectedArtifactRevision: number,
		data: MCPServerData
	): Promise<StoreArtifact>;

	updateProtectedMCPServerInstallation(
		server: ArtifactRef,
		expectedOverlayRevision: number,

		data: MCPServerData
	): Promise<void>;
}

export interface IMCPRuntimeAPI {
	cancelPendingMCPOAuthAuthorization(server: MCPRuntimeServerID): Promise<boolean>;

	completeMCPArgument(
		server: MCPRuntimeServerID,
		request: MCPCompleteArgumentRequestBody
	): Promise<MCPCompletionResult>;

	connectMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot>;

	disconnectMCPServer(server: MCPRuntimeServerID): Promise<void>;

	evaluateMappedMCPToolCall(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<MCPApprovalEvaluation>;

	evaluateMCPToolCall(server: MCPRuntimeServerID, request: InvokeMCPToolRequestBody): Promise<MCPApprovalEvaluation>;

	getMCPGlobalSettings(): Promise<MCPGlobalSettings>;

	getMCPPrompt(
		server: MCPRuntimeServerID,
		promptName: string,

		promptArguments: Record<string, string>
	): Promise<MCPGetPromptResponseBody>;

	getMCPServerStatus(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot>;

	invokeMappedMCPTool(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<MCPRuntimeInvokeToolResponse>;

	invokeMCPTool(server: MCPRuntimeServerID, request: InvokeMCPToolRequestBody): Promise<MCPRuntimeInvokeToolResponse>;

	listMCPServerPrompts(server: MCPRuntimeServerID): Promise<MCPPromptRef[]>;

	listMCPServerPromptsPage(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPPromptRef>>;

	listMCPServerResources(server: MCPRuntimeServerID): Promise<MCPResourceRef[]>;

	listMCPServerResourcesPage(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPResourceRef>>;

	listMCPServerResourceTemplates(server: MCPRuntimeServerID): Promise<MCPResourceTemplateRef[]>;

	listMCPServerResourceTemplatesPage(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPResourceTemplateRef>>;

	listMCPServerTools(server: MCPRuntimeServerID): Promise<MCPToolCapability[]>;

	listMCPServerToolsPage(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken?: string
	): Promise<MCPDiscoveryPage<MCPToolCapability>>;

	listPendingMCPOAuthAuthorizations(): Promise<MCPOAuthAuthorization[]>;

	readMCPResource(server: MCPRuntimeServerID, uri: string): Promise<MCPReadResourceResponseBody>;

	refreshMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot>;

	resolveMCPApproval(approvalID: string, resolution: MCPApprovalResolution): Promise<MCPApprovalResolutionResult>;

	startMCPServerConnect(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot>;

	updateMCPGlobalSettings(expectedRevision: number, settings: MCPAuthSettings): Promise<number>;
}
