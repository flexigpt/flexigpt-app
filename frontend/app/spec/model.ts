import type { ArtifactDigest, ArtifactRef, ArtifactState, StoreArtifact, StoreArtifactAddress } from '@/spec/artifact';
import type {
	CacheControl,
	CacheControlKind,
	CacheControlTTL,
	ModelParam,
	OutputFormatKind,
	OutputVerbosity,
	ProviderSDKType,
	ReasoningLevel,
	ReasoningSummaryStyle,
	ReasoningType,
} from '@/spec/inference';

export const PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY = '__conversation__:previous-system-prompt';
export const PREVIOUS_CONVO_SYSTEM_PROMPT_BUNDLEID = '__conversation__';

export type IncludePreviousMessages = number | 'all';

export enum ModelArtifactType {
	Model = 'model',
	Provider = 'model.provider',
}

export enum ModelLookupScope {
	Builtin = 'builtin',
}

export enum ModelAuthenticationMode {
	None = 'none',
	APIKeyHeader = 'apiKeyHeader',
	BearerToken = 'bearerToken',
}

export enum ModelModalityIn {
	Text = 'textIn',
	Image = 'imageIn',
	File = 'fileIn',
	Audio = 'audioIn',
	Video = 'videoIn',
}

export enum ModelModalityOut {
	Text = 'textOut',
	Image = 'imageOut',
	File = 'fileOut',
	Audio = 'audioOut',
	Video = 'videoOut',
}

export enum ModelToolType {
	Function = 'function',
	Custom = 'custom',
	WebSearch = 'webSearch',
}

export enum ModelToolPolicyMode {
	Auto = 'auto',
	Any = 'any',
	Tool = 'tool',
	None = 'none',
}

export enum ModelClientToolOutputFormat {
	String = 'string',
	ContentItemList = 'contentItemList',
}

export enum ModelOutputTokenParameterName {
	MaxCompletionTokens = 'maxCompletionTokens',
	MaxTokens = 'maxTokens',
}

export enum ModelToolChoiceParameterStyle {
	AllowedTools = 'allowedTools',
	RequiredNamed = 'requiredNamed',
}

export enum ModelRequestClearField {
	AdapterParameters = 'adapterParameters',
	CacheControl = 'cacheControl',
	Output = 'output',
	Reasoning = 'reasoning',
	StopSequences = 'stopSequences',
	Temperature = 'temperature',
}

export type ModelReasoningLevel = ReasoningLevel | '';

export type ModelJSONSchema = Record<string, unknown> | boolean;

export type ModelAdapterParameters = Record<string, unknown>;

export interface ModelArtifactNameReference {
	name: string;
	scope?: ModelLookupScope;
}

export interface ModelHeaderPatch {
	set?: Record<string, string>;
	remove?: string[];
}

export interface ModelConnection {
	origin?: string;
	path?: string;
	headers?: ModelHeaderPatch;
}

export interface ModelAuthentication {
	mode: ModelAuthenticationMode;
	headerName?: string;
	prefix?: string;
}

export interface ModelReasoningDefaults {
	type?: ReasoningType;
	level?: ModelReasoningLevel;
	tokens?: number;
	summaryStyle?: ReasoningSummaryStyle;
	context?: string;
	mode?: string;
}

export interface ModelOutputJSONSchema {
	name?: string;
	description?: string;
	schema?: ModelJSONSchema;
	strict?: boolean;
}

export interface ModelOutputFormat {
	kind?: OutputFormatKind;
	jsonSchema?: ModelOutputJSONSchema;
}

export interface ModelOutputDefaults {
	verbosity?: OutputVerbosity;
	format?: ModelOutputFormat;
}

export interface ModelDefaults {
	stream?: boolean;
	maxPromptTokens?: number;
	maxOutputTokens?: number;
	temperature?: number;
	systemPrompt?: string;
	timeoutMS?: number;
	reasoning?: ModelReasoningDefaults;
	cacheControl?: CacheControl;
	output?: ModelOutputDefaults;
	stopSequences?: string[];
	adapterParameters?: ModelAdapterParameters;
}

export interface ModelReasoningTokenBudgetCapabilities {
	minAllowed?: number;
	maxAllowed?: number;
	zeroAllowed?: boolean;
	minusOneAllowed?: boolean;
}

export interface ModelReasoningCapabilities {
	supportsReasoningConfig?: boolean;
	supportedReasoningTypes?: ReasoningType[];
	supportedReasoningLevels?: ReasoningLevel[];
	hybridTokenBudgetCapabilities?: ModelReasoningTokenBudgetCapabilities;
	supportsSummaryStyle?: boolean;
	supportsReasoningContext?: boolean;
	supportsReasoningMode?: boolean;
	supportsEncryptedReasoningInput?: boolean;
	temperatureDisallowedWhenEnabled?: boolean;
}

export interface ModelStopSequenceCapabilities {
	isSupported?: boolean;
	disallowedWithReasoning?: boolean;
	maxSequences?: number;
}

export interface ModelOutputCapabilities {
	supportedOutputFormats?: OutputFormatKind[];
	supportsVerbosity?: boolean;
}

export interface ModelToolCapabilities {
	supportedToolTypes?: ModelToolType[];
	supportedToolPolicyModes?: ModelToolPolicyMode[];
	supportsParallelToolCalls?: boolean;
	maxForcedTools?: number;
	supportedClientToolOutputFormats?: ModelClientToolOutputFormat[];
}

export interface ModelCacheControlCapabilities {
	supportsTTL?: boolean;
	supportedKinds?: CacheControlKind[];
	supportedTTLs?: CacheControlTTL[];
	supportsKey?: boolean;
}

export interface ModelCacheCapabilities {
	supportsAutomaticCaching?: boolean;
	topLevel?: ModelCacheControlCapabilities;
	inputOutputContent?: ModelCacheControlCapabilities;
	reasoningContent?: ModelCacheControlCapabilities;
	toolChoice?: ModelCacheControlCapabilities;
	toolCall?: ModelCacheControlCapabilities;
	toolOutput?: ModelCacheControlCapabilities;
}

export interface ModelParameterDialect {
	maxOutputTokensParamName?: ModelOutputTokenParameterName;
	toolChoiceParamStyle?: ModelToolChoiceParameterStyle;
}

export interface ModelCapabilities {
	modalitiesIn?: ModelModalityIn[];
	modalitiesOut?: ModelModalityOut[];
	reasoningCapabilities?: ModelReasoningCapabilities;
	stopSequenceCapabilities?: ModelStopSequenceCapabilities;
	outputCapabilities?: ModelOutputCapabilities;
	toolCapabilities?: ModelToolCapabilities;
	cacheCapabilities?: ModelCacheCapabilities;
	paramDialect?: ModelParameterDialect;
}

export interface ModelProviderDocument {
	type: ModelArtifactType.Provider;
	name: string;
	displayName?: string;
	description?: string;
	labels?: Record<string, string>;
	adapter: string;
	connection?: ModelConnection;
	authentication?: ModelAuthentication;
	defaultModel?: ModelArtifactNameReference;
	defaults?: ModelDefaults;
	capabilities?: ModelCapabilities;
	adapterParameters?: ModelAdapterParameters;
}

export interface ModelDocument {
	type: ModelArtifactType.Model;
	name: string;
	displayName?: string;
	description?: string;
	labels?: Record<string, string>;
	provider: ModelArtifactNameReference;
	providerModelID: string;
	defaults?: ModelDefaults;
	capabilities?: ModelCapabilities;
	adapterParameters?: ModelAdapterParameters;
}

export interface ModelProviderListItem {
	ref: ArtifactRef;
	name: string;
	displayName: string;
	description?: string;
	adapter?: string;
	defaultModel?: ModelArtifactNameReference;
	state: ArtifactState;
	enabled: boolean;
	revision: number;
	definitionDigest?: ArtifactDigest;
	builtIn: boolean;
	credentialConfigured?: boolean;
	runtimeOverlayRevision?: number;
}

export interface ModelListItem {
	ref: ArtifactRef;
	name: string;
	displayName: string;
	description?: string;
	provider?: ModelArtifactNameReference;
	providerModelID?: string;
	state: ArtifactState;
	enabled: boolean;
	revision: number;
	definitionDigest?: ArtifactDigest;
	builtIn: boolean;
}

export interface ModelProviderView {
	artifact: StoreArtifact;
	definitionDigest: ArtifactDigest;
	document: ModelProviderDocument;
	builtIn: boolean;
}

export interface ModelView {
	artifact: StoreArtifact;
	definitionDigest: ArtifactDigest;
	document: ModelDocument;
	builtIn: boolean;
}

export interface ManagedProviderCreateRequest {
	rootID: string;
	document: ModelProviderDocument;
	enabled: boolean;
}

export interface ManagedProviderReplaceRequest {
	provider: ArtifactRef;
	expectedArtifactRevision: number;
	document: ModelProviderDocument;
	enabled: boolean;
}

export interface ManagedModelCreateRequest {
	rootID: string;
	document: ModelDocument;
	enabled: boolean;
}

export interface ManagedModelReplaceRequest {
	model: ArtifactRef;
	expectedArtifactRevision: number;
	document: ModelDocument;
	enabled: boolean;
}

export interface ManagedProviderResult {
	artifact: StoreArtifact;
	address: StoreArtifactAddress;
}

export interface ManagedModelResult {
	artifact: StoreArtifact;
	address: StoreArtifactAddress;
}

export interface ModelProviderRuntimeOverlayView {
	revision: number;
	credentialConfigured: boolean;
}

export interface ModelRequestPatch {
	defaults?: ModelDefaults;
	clear?: ModelRequestClearField[];
}

export interface UIModelOption extends ModelParam {
	model?: ArtifactRef;
	provider?: ArtifactRef;

	logicalName: string;
	providerName: string;
	providerAdapter: string;
	providerSDKType: ProviderSDKType;
	providerDisplayName: string;
	modelDisplayName: string;

	includePreviousMessages: number | 'all';

	capabilities?: ModelCapabilities;

	/*
	 * This is UI-only source baseline used to produce the final request patch.
	 * It is not persisted into a Conversation.
	 */
	sourceModelParam?: ModelParam;
}

export const DefaultUIModelOption: UIModelOption = {
	name: '',
	stream: false,
	maxPromptLength: 2048,
	maxOutputLength: 1024,
	temperature: 0.1,
	systemPrompt: '',
	timeout: 300,

	logicalName: 'no-model',
	providerName: 'no-provider',
	providerAdapter: 'openai.responses',
	providerSDKType: 'providerSDKTypeOpenAIResponses' as ProviderSDKType,
	providerDisplayName: 'No Provider',
	modelDisplayName: 'No Model Configured',
	includePreviousMessages: 'all',
};
