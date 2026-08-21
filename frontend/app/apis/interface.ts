import type {
	ArtifactCollection,
	ArtifactCollectionRef,
	ArtifactDefinitionSelector,
	ArtifactRecord,
	ArtifactRef,
	ArtifactRoot,
	ArtifactRootID,
	ArtifactSourceBinding,
	ArtifactSourceDraft,
	ArtifactSourceID,
	ArtifactSourceKind,
	ArtifactSourceSummary,
	CreateArtifactRootBody,
	ManagedSourcePackageResult,
	ManagedSourceState,
	PublishManagedSourcePackageBody,
	PurgeArtifactRootResult,
	PurgeArtifactSourceResult,
	RemoveManagedSourcePackageBody,
	UpdateArtifactRootBody,
	UpdateArtifactSourceBody,
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
import type { ConversationSearchItem, StoreConversation, StoreConversationMessage } from '@/spec/conversation';
import type { CompletionResponseBody, ModelParam, ProviderName } from '@/spec/inference';
import type {
	InvokeMCPToolRequestBody,
	MCPApprovalEvaluation,
	MCPApprovalResolution,
	MCPApprovalResolutionResult,
	MCPAuthHealth,
	MCPBundle,
	MCPBundleDocument,
	MCPBundleInstallation,
	MCPCompletionRefType,
	MCPCompletionResult,
	MCPConversationContext,
	MCPCreateBundleInput,
	MCPGetPromptResponseBody,
	MCPGlobalSettings,
	InvokeMCPToolResponseBody as MCPInvokeToolResponseBody,
	MCPOAuthAuthorization,
	MCPPolicyView,
	MCPPromptRef,
	MCPProviderToolMapping,
	MCPReadResourceResponseBody,
	MCPReplaceBundleDocumentInput,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPSecretKind,
	MCPSecretWriteResult,
	MCPServerData,
	MCPServerInstallation,
	MCPServerResolved,
	MCPServerRuntimeSnapshot,
	MCPToolCapability,
} from '@/spec/mcp_artifact';
import type {
	ModelPresetID,
	PatchModelPresetPayload,
	PatchProviderPresetPayload,
	PostModelPresetPayload,
	PostProviderPresetPayload,
	ProviderPreset,
} from '@/spec/modelpreset';
import type { AppTheme, AuthKey, AuthKeyName, AuthKeyType, DebugSettings, SettingsSchema } from '@/spec/setting';
import type {
	AdoptSkillBody,
	AttachSkillBundleSourceBody,
	CreateManagedSkillBody,
	CreateManagedSkillResult,
	CreateSkillBundleBody,
	CreateSkillSessionOptions,
	InvokeSkillToolResponse,
	ManagedSkillDocumentView,
	PinSkillBody,
	RenderSkillResponse,
	RetireSkillBundleResult,
	RuntimeSkillFilter,
	RuntimeSkillListItem,
	SetSkillEnabledBody,
	SkillArtifactView,
	SkillBundleRef,
	SkillBundleView,
	SkillSession,
	UpdateSkillBundleBody,
} from '@/spec/skill';
import type { HTTPToolImpl, Tool, ToolBundle, ToolImplType, ToolListItem, ToolStoreChoice } from '@/spec/tool';
import type { InvokeGoOptions, InvokeHTTPOptions, InvokeToolResponse } from '@/spec/toolruntime';
import type { ApplyUnifiedDiffArgs, ApplyUnifiedDiffOut } from '@/spec/unified_diff';
import type {
	AdoptWorkspaceOccurrenceBody,
	AttachWorkspaceSourceBody,
	CreateEmptyWorkspaceBody,
	CreateFilesystemWorkspaceBody,
	DetachWorkspaceSourceBody,
	PinWorkspaceArtifactBody,
	ReplaceWorkspacePrimarySourceBody,
	ResolveWorkspaceResourceResult,
	RetireWorkspaceResult,
	SetWorkspaceArtifactEnabledBody,
	SetWorkspaceArtifactRuntimeDisabledBody,
	SetWorkspacePrimarySourceBody,
	SuppressWorkspaceBindingBody,
	UnadoptWorkspaceArtifactBody,
	UnadoptWorkspaceArtifactResult,
	UnsuppressWorkspaceBindingResult,
	UpdateWorkspaceAttachmentBody,
	UpdateWorkspaceBody,
	WorkspaceArtifactView,
	WorkspaceCatalogView,
	WorkspaceContextInspectionView,
	WorkspaceContextLoadPlan,
	WorkspaceContextView,
	WorkspaceLoadPlanView,
	WorkspaceRef,
	WorkspaceRefreshResult,
	WorkspaceSkillLoadView,
	WorkspaceSkillView,
	WorkspaceSuppressionView,
	WorkspaceView,
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

export interface ISkillBundleAPI {
	createSkillBundle(rootID: ArtifactRootID, body: CreateSkillBundleBody): Promise<SkillBundleView>;

	getSkillBundle(bundle: SkillBundleRef): Promise<SkillBundleView>;

	listSkillBundles(rootID: ArtifactRootID): Promise<SkillBundleView[]>;

	updateSkillBundle(bundle: SkillBundleRef, body: UpdateSkillBundleBody): Promise<SkillBundleView>;

	retireSkillBundle(bundle: SkillBundleRef, expectedRevision: number): Promise<RetireSkillBundleResult>;

	purgeSkillBundle(bundle: SkillBundleRef, expectedRevision: number): Promise<SkillBundleRef>;

	attachSkillBundleSource(bundle: SkillBundleRef, body: AttachSkillBundleSourceBody): Promise<SkillBundleView>;

	refreshSkillBundle(bundle: SkillBundleRef): Promise<void>;

	listSkillBundleArtifacts(bundle: SkillBundleRef): Promise<SkillArtifactView[]>;

	createManagedSkill(bundle: SkillBundleRef, body: CreateManagedSkillBody): Promise<CreateManagedSkillResult>;

	getManagedSkillDocument(artifact: ArtifactRef): Promise<ManagedSkillDocumentView>;

	adoptSkill(bundle: SkillBundleRef, body: AdoptSkillBody): Promise<SkillArtifactView>;

	pinSkill(bundle: SkillBundleRef, body: PinSkillBody): Promise<SkillArtifactView>;

	setSkillEnabled(artifact: ArtifactRef, body: SetSkillEnabledBody): Promise<SkillArtifactView>;

	unadoptSkill(artifact: ArtifactRef, expectedRevision: number, suppress: boolean): Promise<ArtifactRef>;

	purgeSkill(artifact: ArtifactRef, expectedRevision: number): Promise<ArtifactRef>;

	getSkillsPrompt(filter: RuntimeSkillFilter): Promise<string>;

	createSkillSession(options: CreateSkillSessionOptions): Promise<SkillSession>;

	closeSkillSession(sessionID: string): Promise<void>;

	listRuntimeSkills(filter: RuntimeSkillFilter): Promise<RuntimeSkillListItem[]>;

	invokeSkillTool(sessionID: string, toolName: string, args?: JSONRawString): Promise<InvokeSkillToolResponse>;

	renderSkill(artifact: ArtifactRef, args?: Record<string, string>): Promise<RenderSkillResponse>;
}

export interface IArtifactStoreAPI {
	createArtifactRoot(body: CreateArtifactRootBody): Promise<ArtifactRoot>;

	getArtifactRoot(rootID: ArtifactRootID): Promise<ArtifactRoot>;

	listArtifactRoots(): Promise<ArtifactRoot[]>;

	updateArtifactRoot(rootID: ArtifactRootID, body: UpdateArtifactRootBody): Promise<ArtifactRoot>;

	retireArtifactRoot(rootID: ArtifactRootID, expectedRevision: number): Promise<ArtifactRoot>;

	purgeArtifactRoot(rootID: ArtifactRootID, expectedRevision: number): Promise<PurgeArtifactRootResult>;

	createArtifactSource(rootID: ArtifactRootID, body: ArtifactSourceDraft): Promise<ArtifactSourceSummary>;

	getArtifactSource(rootID: ArtifactRootID, sourceID: ArtifactSourceID): Promise<ArtifactSourceSummary>;

	listArtifactSources(rootID: ArtifactRootID): Promise<ArtifactSourceSummary[]>;

	updateArtifactSource(
		rootID: ArtifactRootID,
		sourceID: ArtifactSourceID,
		body: UpdateArtifactSourceBody
	): Promise<ArtifactSourceSummary>;

	retireArtifactSource(
		rootID: ArtifactRootID,
		sourceID: ArtifactSourceID,
		expectedRevision: number
	): Promise<ArtifactSourceSummary>;

	purgeArtifactSource(
		rootID: ArtifactRootID,
		sourceID: ArtifactSourceID,
		expectedRevision: number
	): Promise<PurgeArtifactSourceResult>;

	listArtifactSourceKinds(): Promise<ArtifactSourceKind[]>;

	getManagedSourceState(rootID: ArtifactRootID, sourceID: ArtifactSourceID): Promise<ManagedSourceState>;

	publishManagedSourcePackage(
		rootID: ArtifactRootID,
		sourceID: ArtifactSourceID,
		body: PublishManagedSourcePackageBody
	): Promise<ManagedSourcePackageResult>;

	removeManagedSourcePackage(
		rootID: ArtifactRootID,
		sourceID: ArtifactSourceID,
		body: RemoveManagedSourcePackageBody
	): Promise<ManagedSourcePackageResult>;
}

export interface IWorkspaceAPI {
	createFilesystemWorkspace(rootID: ArtifactRootID, body: CreateFilesystemWorkspaceBody): Promise<WorkspaceView>;

	createEmptyWorkspace(rootID: ArtifactRootID, body: CreateEmptyWorkspaceBody): Promise<WorkspaceView>;

	getWorkspace(workspace: WorkspaceRef): Promise<WorkspaceView>;

	listWorkspaces(rootID: ArtifactRootID): Promise<WorkspaceView[]>;

	updateWorkspace(workspace: WorkspaceRef, body: UpdateWorkspaceBody): Promise<WorkspaceView>;

	replaceWorkspacePrimarySource(
		workspace: WorkspaceRef,
		body: ReplaceWorkspacePrimarySourceBody
	): Promise<WorkspaceView>;

	setWorkspacePrimarySource(workspace: WorkspaceRef, body: SetWorkspacePrimarySourceBody): Promise<WorkspaceView>;

	retireWorkspace(workspace: WorkspaceRef, expectedRevision: number): Promise<RetireWorkspaceResult>;

	purgeWorkspace(workspace: WorkspaceRef, expectedRevision: number): Promise<WorkspaceRef>;

	attachWorkspaceSource(workspace: WorkspaceRef, body: AttachWorkspaceSourceBody): Promise<WorkspaceView>;

	updateWorkspaceAttachment(
		workspace: WorkspaceRef,
		sourceID: ArtifactSourceID,
		body: UpdateWorkspaceAttachmentBody
	): Promise<WorkspaceView>;

	detachWorkspaceSource(
		workspace: WorkspaceRef,
		sourceID: ArtifactSourceID,
		body: DetachWorkspaceSourceBody
	): Promise<WorkspaceView>;

	refreshWorkspace(workspace: WorkspaceRef): Promise<WorkspaceRefreshResult>;

	getWorkspaceCatalog(workspace: WorkspaceRef): Promise<WorkspaceCatalogView>;

	composeWorkspaceLoadPlan(workspace: WorkspaceRef, artifacts: ArtifactRef[]): Promise<WorkspaceLoadPlanView>;

	resolveWorkspaceResource(
		workspace: WorkspaceRef,
		artifact?: ArtifactRef,
		selector?: ArtifactDefinitionSelector
	): Promise<ResolveWorkspaceResourceResult>;

	getWorkspaceArtifact(workspace: WorkspaceRef, artifact: ArtifactRef): Promise<WorkspaceArtifactView>;

	listWorkspaceArtifacts(workspace: WorkspaceRef): Promise<WorkspaceArtifactView[]>;

	adoptWorkspaceOccurrence(workspace: WorkspaceRef, body: AdoptWorkspaceOccurrenceBody): Promise<WorkspaceArtifactView>;

	pinWorkspaceArtifact(workspace: WorkspaceRef, body: PinWorkspaceArtifactBody): Promise<WorkspaceArtifactView>;

	listWorkspaceSuppressions(workspace: WorkspaceRef): Promise<WorkspaceSuppressionView[]>;

	suppressWorkspaceBinding(
		workspace: WorkspaceRef,
		body: SuppressWorkspaceBindingBody
	): Promise<WorkspaceSuppressionView>;

	unsuppressWorkspaceBinding(
		workspace: WorkspaceRef,
		binding: ArtifactSourceBinding,
		expectedRevision: number
	): Promise<UnsuppressWorkspaceBindingResult>;

	listWorkspaceContexts(workspace: WorkspaceRef): Promise<WorkspaceContextView[]>;

	loadWorkspaceContexts(workspace: WorkspaceRef, artifacts?: ArtifactRef[]): Promise<WorkspaceContextInspectionView>;

	composeWorkspaceContext(workspace: WorkspaceRef, artifacts?: ArtifactRef[]): Promise<WorkspaceContextLoadPlan>;

	listWorkspaceSkills(workspace: WorkspaceRef): Promise<WorkspaceSkillView[]>;

	loadWorkspaceSkills(workspace: WorkspaceRef, artifacts: ArtifactRef[]): Promise<WorkspaceSkillLoadView>;

	setWorkspaceArtifactEnabled(
		workspace: WorkspaceRef,
		artifact: ArtifactRef,
		body: SetWorkspaceArtifactEnabledBody
	): Promise<WorkspaceArtifactView>;

	unadoptWorkspaceArtifact(
		workspace: WorkspaceRef,
		artifact: ArtifactRef,
		body: UnadoptWorkspaceArtifactBody
	): Promise<UnadoptWorkspaceArtifactResult>;

	purgeWorkspaceArtifact(
		workspace: WorkspaceRef,
		artifact: ArtifactRef,
		expectedRevision: number
	): Promise<ArtifactRef>;

	setWorkspaceArtifactRuntimeDisabled(
		workspace: WorkspaceRef,
		artifact: ArtifactRef,
		body: SetWorkspaceArtifactRuntimeDisabledBody
	): Promise<WorkspaceArtifactView>;
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

export interface IMCPAPI {
	createMCPBundle(input: MCPCreateBundleInput): Promise<MCPBundle>;
	getMCPBundle(bundle: ArtifactCollectionRef): Promise<MCPBundle>;
	listMCPBundles(rootID: ArtifactRootID): Promise<MCPBundle[]>;
	getMCPBundleDocument(bundle: ArtifactCollectionRef): Promise<MCPBundleDocument>;
	listMCPBundleServers(bundle: ArtifactCollectionRef): Promise<ArtifactRecord[]>;
	listMCPBundlePolicies(bundle: ArtifactCollectionRef): Promise<ArtifactRecord[]>;
	getMCPBundleInstallation(bundle: ArtifactCollectionRef): Promise<MCPBundleInstallation>;
	replaceMCPBundleDocument(input: MCPReplaceBundleDocumentInput): Promise<MCPBundle>;
	refreshMCPBundle(bundle: ArtifactCollectionRef): Promise<MCPBundle>;
	updateMCPBundleEnabled(bundle: ArtifactCollectionRef, expectedRevision: number, enabled: boolean): Promise<MCPBundle>;
	updateProtectedMCPBundleInstallation(
		bundle: ArtifactCollectionRef,
		expectedOverlayRevision: number,
		runtimeEnabled: boolean
	): Promise<void>;
	retireMCPBundle(bundle: ArtifactCollectionRef, expectedRevision: number): Promise<ArtifactCollection>;
	purgeMCPBundle(bundle: ArtifactCollectionRef, expectedRevision: number): Promise<void>;

	getMCPServerInstallation(server: ArtifactRef): Promise<MCPServerInstallation>;
	inspectMCPServer(server: ArtifactRef): Promise<MCPServerResolved>;
	inspectMCPPolicy(policy: ArtifactRef): Promise<MCPPolicyView>;
	updateMCPServerInstallation(
		server: ArtifactRef,
		expectedArtifactRevision: number,
		data: MCPServerData
	): Promise<ArtifactRecord>;
	updateProtectedMCPServerInstallation(
		server: ArtifactRef,
		expectedOverlayRevision: number,
		runtimeEnabled: boolean,
		data: MCPServerData
	): Promise<void>;

	connectMCPServer(server: ArtifactRef): Promise<MCPServerRuntimeSnapshot>;
	disconnectMCPServer(server: ArtifactRef): Promise<void>;
	refreshMCPServer(server: ArtifactRef): Promise<MCPServerRuntimeSnapshot>;
	getMCPServerStatus(server: ArtifactRef): Promise<MCPServerRuntimeSnapshot>;
	listMCPServerTools(server: ArtifactRef): Promise<MCPToolCapability[]>;
	listMCPServerResources(server: ArtifactRef): Promise<MCPResourceRef[]>;
	listMCPServerResourceTemplates(server: ArtifactRef): Promise<MCPResourceTemplateRef[]>;
	listMCPServerPrompts(server: ArtifactRef): Promise<MCPPromptRef[]>;

	readMCPResource(server: ArtifactRef, uri: string): Promise<MCPReadResourceResponseBody>;
	getMCPPrompt(
		server: ArtifactRef,
		promptName: string,
		promptArguments?: Record<string, string>
	): Promise<MCPGetPromptResponseBody>;
	completeMCPArgument(
		server: ArtifactRef,
		refType: MCPCompletionRefType,
		name: string,
		argumentName: string,
		argumentValue?: string,
		context?: Record<string, string>
	): Promise<MCPCompletionResult>;

	evaluateMCPToolCall(server: ArtifactRef, request: InvokeMCPToolRequestBody): Promise<MCPApprovalEvaluation>;
	evaluateMappedMCPToolCall(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<MCPApprovalEvaluation>;
	invokeMCPTool(server: ArtifactRef, request: InvokeMCPToolRequestBody): Promise<MCPInvokeToolResponseBody>;
	invokeMappedMCPTool(
		mapping: MCPProviderToolMapping,
		request: InvokeMCPToolRequestBody
	): Promise<MCPInvokeToolResponseBody>;
	resolveMCPApproval(approvalID: string, resolution: MCPApprovalResolution): Promise<MCPApprovalResolutionResult>;

	getMCPServerAuthHealth(server: ArtifactRef): Promise<MCPAuthHealth>;
	listPendingMCPOAuthAuthorizations(): Promise<MCPOAuthAuthorization[]>;
	cancelPendingMCPOAuthAuthorization(server: ArtifactRef): Promise<boolean>;
	putMCPServerSecret(
		server: ArtifactRef,
		kind: MCPSecretKind,
		slot: string,
		secret: string
	): Promise<MCPSecretWriteResult>;
	deleteMCPServerSecret(server: ArtifactRef, kind: MCPSecretKind, slot: string): Promise<void>;

	getMCPGlobalSettings(): Promise<MCPGlobalSettings>;
	updateMCPGlobalSettings(expectedRevision: number, oauthLoopbackListenAddr?: string): Promise<number>;
}
