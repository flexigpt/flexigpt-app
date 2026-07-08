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
import type { ConversationSearchItem, StoreConversation, StoreConversationMessage } from '@/spec/conversation';
import type { CompletionResponseBody, ModelParam, ProviderName } from '@/spec/inference';
import type {
	InvokeMCPToolRequestBody,
	MCPApprovalEvaluation,
	MCPApprovalResolution,
	MCPApprovalToken,
	MCPAuthHealth,
	MCPAuthStatus,
	MCPBundle,
	MCPCompletionResult,
	MCPConversationContext,
	MCPGetPromptResponseBody,
	InvokeMCPToolResponseBody as MCPInvokeToolResponseBody,
	MCPOAuthAuthorization,
	MCPPromptRef,
	MCPReadResourceResponseBody,
	MCPRefType,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPSecretKind,
	MCPServerConfig,
	MCPServerID,
	MCPServerRuntimeSnapshot,
	MCPServerSetupInputValue,
	MCPSettingsView,
	MCPToolCapability,
	PatchMCPServerPolicyPayload,
	PutMCPServerPayload,
	PutMCPServerSecretResponseBody,
} from '@/spec/mcp';
import type {
	ModelPresetID,
	PatchModelPresetPayload,
	PatchProviderPresetPayload,
	PostModelPresetPayload,
	PostProviderPresetPayload,
	ProviderPreset,
} from '@/spec/modelpreset';
import type {
	MessageBlock,
	PromptBundle,
	PromptTemplate,
	PromptTemplateKind,
	PromptTemplateListItem,
	PromptVariable,
} from '@/spec/prompt';
import type { AppTheme, AuthKey, AuthKeyName, AuthKeyType, DebugSettings, SettingsSchema } from '@/spec/setting';
import type {
	InvokeSkillToolResponse,
	ListSkillsRequest,
	RenderSkillResponse,
	RuntimeSkillFilter,
	RuntimeSkillListItem,
	Skill,
	SkillBundle,
	SkillListItem,
	SkillRef,
	SkillSession,
	SkillType,
} from '@/spec/skill';
import type { HTTPToolImpl, Tool, ToolBundle, ToolImplType, ToolListItem, ToolStoreChoice } from '@/spec/tool';
import type { InvokeGoOptions, InvokeHTTPOptions, InvokeToolResponse } from '@/spec/toolruntime';
import type { ApplyUnifiedDiffArgs, ApplyUnifiedDiffOut } from '@/spec/unified_diff';

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

export interface IPromptStoreAPI {
	/** List bundles, optionally filtered by IDs, disabled, and paginated. */
	listPromptBundles(
		bundleIDs?: string[],
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ promptBundles: PromptBundle[]; nextPageToken?: string }>;

	/** Create or update a bundle. */
	putPromptBundle(
		bundleID: string,
		slug: string,
		displayName: string,
		isEnabled: boolean,
		description?: string
	): Promise<void>;

	/** Patch (enable/disable) a bundle. */
	patchPromptBundle(bundleID: string, isEnabled: boolean): Promise<void>;

	/** Delete a bundle. */
	deletePromptBundle(bundleID: string): Promise<void>;

	/** List templates, optionally filtered by bundleIDs, tags, etc. */
	listPromptTemplates(
		bundleIDs?: string[],
		tags?: string[],
		includeDisabled?: boolean,
		kinds?: PromptTemplateKind[],
		onlyResolved?: boolean,
		recommendedPageSize?: number,
		pageToken?: string
	): Promise<{ promptTemplateListItems: PromptTemplateListItem[]; nextPageToken?: string }>;

	/** Create or update a template. */
	putPromptTemplate(
		kind: PromptTemplateKind,
		bundleID: string,
		templateSlug: string,
		displayName: string,
		isEnabled: boolean,
		blocks: MessageBlock[],
		version: string,
		isResolved: boolean,
		description?: string,
		tags?: string[],
		variables?: PromptVariable[]
	): Promise<void>;

	/** Patch (enable/disable) a template version. */
	patchPromptTemplate(bundleID: string, templateSlug: string, version: string, isEnabled: boolean): Promise<void>;

	/** Delete a template version. */
	deletePromptTemplate(bundleID: string, templateSlug: string, version: string): Promise<void>;

	/** Get a template version. */
	getPromptTemplate(bundleID: string, templateSlug: string, version: string): Promise<PromptTemplate | undefined>;
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
		autoExecReco: boolean,
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
}

export interface ISkillStoreAPI {
	/** List skill bundles, optionally filtered by IDs, disabled, and paginated. */
	listSkillBundles(
		bundleIDs?: string[],
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ skillBundles: SkillBundle[]; nextPageToken?: string }>;

	/** Create or update a skill bundle. */
	putSkillBundle(
		bundleID: string,
		slug: string,
		displayName: string,
		isEnabled: boolean,
		description?: string
	): Promise<void>;

	/** Patch (enable/disable) a bundle. */
	patchSkillBundle(bundleID: string, isEnabled: boolean): Promise<void>;

	/** Delete a bundle. */
	deleteSkillBundle(bundleID: string): Promise<void>;

	/** List skills, optionally filtered by bundleIDs/types and paginated. */
	listSkills(req: ListSkillsRequest): Promise<{ skillListItems: SkillListItem[]; nextPageToken?: string }>;

	/** Create or update a skill. */
	putSkill(
		bundleID: string,
		skillSlug: string,
		skillType: SkillType,
		location: string,
		name: string,
		isEnabled: boolean,
		displayName?: string,
		description?: string,
		tags?: string[]
	): Promise<void>;

	/** Patch a skill (enable/disable, location). */
	patchSkill(
		bundleID: string,
		skillSlug: string,
		isEnabled?: boolean,
		location?: string,
		displayName?: string,
		description?: string,
		tags?: string[]
	): Promise<void>;

	/** Delete a skill. */
	deleteSkill(bundleID: string, skillSlug: string): Promise<void>;

	/** Get a skill. */
	getSkill(bundleID: string, skillSlug: string, includeDisabled: boolean): Promise<Skill | undefined>;

	/** Runtime: get a skills prompt. */
	getSkillsPrompt(filter?: RuntimeSkillFilter): Promise<string>;

	/** Runtime: create a skill session. */
	createSkillSession(
		closeSessionID?: string,
		maxActivePerSession?: number,
		allowSkillRefs?: SkillRef[],
		activeSkillRefs?: SkillRef[]
	): Promise<SkillSession>;

	/** Runtime: close a skill session. */
	closeSkillSession(sessionID: string): Promise<void>;

	/** Runtime: list skills from the runtime catalog (optionally filtered). */
	listRuntimeSkills(filter?: RuntimeSkillFilter): Promise<RuntimeSkillListItem[]>;

	invokeSkillTool(sessionID: string, toolName: string, args?: JSONRawString): Promise<InvokeSkillToolResponse>;

	renderSkill(ref: SkillRef, args?: Record<string, string>): Promise<RenderSkillResponse>;
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

/**
 * @public
 *
 * Flattened frontend-facing MCP bridge.
 * Heavy structured payloads stay as objects, while simple requests stay flattened.
 */
export interface IMCPAPI {
	listMCPBundles(
		bundleIDs?: string[],
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ bundles: MCPBundle[]; nextPageToken?: string }>;

	putMCPBundle(
		bundleID: string,
		slug: string,
		displayName: string,
		isEnabled: boolean,
		description?: string
	): Promise<void>;

	patchMCPBundle(bundleID: string, isEnabled: boolean): Promise<void>;

	deleteMCPBundle(bundleID: string): Promise<void>;

	listMCPServers(
		bundleID: string,
		serverIDs?: MCPServerID[],
		enabled?: boolean,
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ servers: MCPServerConfig[]; nextPageToken?: string }>;

	putMCPServer(bundleID: string, serverID: MCPServerID, payload: PutMCPServerPayload): Promise<void>;

	getMCPServer(bundleID: string, serverID: MCPServerID): Promise<MCPServerConfig | undefined>;

	patchMCPServerEnabled(bundleID: string, serverID: MCPServerID, enabled: boolean): Promise<void>;

	patchMCPServerPolicy(bundleID: string, serverID: MCPServerID, payload: PatchMCPServerPolicyPayload): Promise<void>;

	patchMCPServerSetup(
		bundleID: string,
		serverID: MCPServerID,
		inputValues: Record<string, MCPServerSetupInputValue>,
		reset?: boolean
	): Promise<MCPServerConfig | undefined>;

	patchMCPSettings(oauthLoopbackListenAddr?: string): Promise<MCPSettingsView | undefined>;

	getMCPSettings(): Promise<MCPSettingsView | undefined>;

	deleteMCPServer(bundleID: string, serverID: MCPServerID): Promise<void>;

	connectMCPServer(bundleID: string, serverID: MCPServerID): Promise<MCPServerRuntimeSnapshot | undefined>;

	disconnectMCPServer(bundleID: string, serverID: MCPServerID): Promise<void>;

	refreshMCPServer(bundleID: string, serverID: MCPServerID): Promise<MCPServerRuntimeSnapshot | undefined>;

	getMCPServerStatus(bundleID: string, serverID: MCPServerID): Promise<MCPServerRuntimeSnapshot | undefined>;

	listMCPServerTools(
		bundleID: string,
		serverID: MCPServerID,
		pageSize?: number,
		pageToken?: string
	): Promise<{ tools: MCPToolCapability[]; nextPageToken?: string }>;

	listMCPServerResources(
		bundleID: string,
		serverID: MCPServerID,
		pageSize?: number,
		pageToken?: string
	): Promise<{ resources: MCPResourceRef[]; nextPageToken?: string }>;

	listMCPServerResourceTemplates(
		bundleID: string,
		serverID: MCPServerID,
		pageSize?: number,
		pageToken?: string
	): Promise<{ resourceTemplates: MCPResourceTemplateRef[]; nextPageToken?: string }>;

	listMCPServerPrompts(
		bundleID: string,
		serverID: MCPServerID,
		pageSize?: number,
		pageToken?: string
	): Promise<{ prompts: MCPPromptRef[]; nextPageToken?: string }>;

	readMCPResource(
		bundleID: string,
		serverID: MCPServerID,
		uri: string
	): Promise<MCPReadResourceResponseBody | undefined>;

	getMCPPrompt(
		bundleID: string,
		serverID: MCPServerID,
		promptName: string,
		promptArguments?: Record<string, string>
	): Promise<MCPGetPromptResponseBody | undefined>;

	completeMCPArgument(
		bundleID: string,
		serverID: MCPServerID,
		refType: MCPRefType,
		name: string,
		argumentName: string,
		argumentValue?: string,
		context?: Record<string, string>
	): Promise<MCPCompletionResult>;

	evaluateMCPToolCall(bundleID: string, request: InvokeMCPToolRequestBody): Promise<MCPApprovalEvaluation | undefined>;

	invokeMCPTool(bundleID: string, request: InvokeMCPToolRequestBody): Promise<MCPInvokeToolResponseBody | undefined>;

	resolveMCPApproval(approvalID: string, resolution: MCPApprovalResolution): Promise<MCPApprovalToken | undefined>;

	listPendingMCPOAuthAuthorizations(): Promise<MCPOAuthAuthorization[]>;

	cancelPendingMCPOAuthAuthorization(bundleID: string, serverID: MCPServerID): Promise<void>;

	getMCPServerAuthStatus(bundleID: string, serverID: MCPServerID): Promise<MCPAuthStatus | undefined>;

	getMCPServerAuthHealth(bundleID: string, serverID: MCPServerID): Promise<MCPAuthHealth | undefined>;

	putMCPServerSecret(
		bundleID: string,
		serverID: MCPServerID,
		kind: MCPSecretKind,
		slot: string,
		secret: string
	): Promise<PutMCPServerSecretResponseBody | undefined>;

	deleteMCPServerSecret(bundleID: string, serverID: MCPServerID, kind: MCPSecretKind, slot: string): Promise<void>;
}
