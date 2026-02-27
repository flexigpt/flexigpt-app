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
	ModelPresetID,
	ProviderPreset,
	PutModelPresetPayload,
	PutProviderPresetPayload,
} from '@/spec/modelpreset';
import type { MessageBlock, PromptBundle, PromptTemplate, PromptTemplateListItem, PromptVariable } from '@/spec/prompt';
import type { AppTheme, AuthKey, AuthKeyName, AuthKeyType, SettingsSchema } from '@/spec/setting';
import type {
	RuntimeSkillFilter,
	Skill,
	SkillBundle,
	SkillDef,
	SkillListItem,
	SkillRecord,
	SkillSession,
	SkillType,
} from '@/spec/skill';
import type { HTTPToolImpl, Tool, ToolBundle, ToolImplType, ToolListItem, ToolStoreChoice } from '@/spec/tool';
import type { InvokeGoOptions, InvokeHTTPOptions, InvokeToolResponse } from '@/spec/toolruntime';

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
	getAuthKey: (type: AuthKeyType, keyName: AuthKeyName) => Promise<AuthKey>;
	deleteAuthKey: (type: AuthKeyType, keyName: AuthKeyName) => Promise<void>;
	setAuthKey: (type: AuthKeyType, keyName: AuthKeyName, secret: string) => Promise<void>;
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
		recommendedPageSize?: number,
		pageToken?: string
	): Promise<{ promptTemplateListItems: PromptTemplateListItem[]; nextPageToken?: string }>;

	/** Create or update a template. */
	putPromptTemplate(
		bundleID: string,
		templateSlug: string,
		displayName: string,
		isEnabled: boolean,
		blocks: MessageBlock[],
		version: string,
		description?: string,
		tags?: string[],
		variables?: PromptVariable[]
	): Promise<void>;

	/** Patch (enable/disable) a template version. */
	patchPromptTemplate(bundleID: string, templateSlug: string, version: string, isEnabled: boolean): Promise<void>;

	/** Delete a template version. */
	deletePromptTemplate(bundleID: string, templateSlug: string, version: string): Promise<void>;

	/** Get a template version (or latest if version omitted). */
	getPromptTemplate(bundleID: string, templateSlug: string, version: string): Promise<PromptTemplate | undefined>;
}

export interface IModelPresetStoreAPI {
	getDefaultProvider(): Promise<ProviderName>;

	patchDefaultProvider(providerName: ProviderName): Promise<void>;

	putProviderPreset(providerName: ProviderName, payload: PutProviderPresetPayload): Promise<void>;

	patchProviderPreset(
		providerName: ProviderName,
		isEnabled?: boolean,
		defaultModelPresetID?: ModelPresetID
	): Promise<void>;

	deleteProviderPreset(providerName: ProviderName): Promise<void>;

	putModelPreset(
		providerName: ProviderName,
		modelPresetID: ModelPresetID,
		payload: PutModelPresetPayload
	): Promise<void>;

	patchModelPreset(providerName: ProviderName, modelPresetID: ModelPresetID, isEnabled: boolean): Promise<void>;

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
	listSkills(
		bundleIDs?: string[],
		types?: SkillType[],
		includeDisabled?: boolean,
		includeMissing?: boolean,
		recommendedPageSize?: number,
		pageToken?: string
	): Promise<{ skillListItems: SkillListItem[]; nextPageToken?: string }>;

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
	patchSkill(bundleID: string, skillSlug: string, isEnabled?: boolean, location?: string): Promise<void>;

	/** Delete a skill. */
	deleteSkill(bundleID: string, skillSlug: string): Promise<void>;

	/** Get a skill. */
	getSkill(bundleID: string, skillSlug: string): Promise<Skill | undefined>;

	/** Runtime: get a skills prompt. */
	getSkillsPromptXML(filter?: RuntimeSkillFilter): Promise<string>;

	/** Runtime: create a skill session. */
	createSkillSession(maxActivePerSession?: number, activeSkills?: SkillDef[]): Promise<SkillSession>;

	/** Runtime: close a skill session. */
	closeSkillSession(sessionID: string): Promise<void>;

	/** Runtime: list skills from the runtime catalog (optionally filtered). */
	listRuntimeSkills(filter?: RuntimeSkillFilter): Promise<SkillRecord[]>;
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

export interface IProviderSetAPI {
	fetchCompletion(
		provider: ProviderName,
		modelParams: ModelParam,
		current: StoreConversationMessage,
		history?: StoreConversationMessage[],
		toolStoreChoices?: ToolStoreChoice[],
		requestId?: string,
		signal?: AbortSignal,
		onStreamTextData?: (textData: string) => void,
		onStreamThinkingData?: (thinkingData: string) => void
	): Promise<CompletionResponseBody | undefined>;

	cancelCompletion(requestId: string): Promise<void>;
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
