import type { ToolOutputUnion, ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

export type ProviderName = string;
export enum ProviderSDKType {
	ProviderSDKTypeAnthropic = 'providerSDKTypeAnthropicMessages',
	ProviderSDKTypeOpenAIChatCompletions = 'providerSDKTypeOpenAIChatCompletions',
	ProviderSDKTypeOpenAIResponses = 'providerSDKTypeOpenAIResponses',
}

export const SDK_DISPLAY_NAME: Record<ProviderSDKType, string> = {
	[ProviderSDKType.ProviderSDKTypeAnthropic]: 'Anthropic Messages API',
	[ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions]: 'OpenAI ChatCompletions API',
	[ProviderSDKType.ProviderSDKTypeOpenAIResponses]: 'OpenAI Responses API',
};

export const SDK_DEFAULTS: Record<
	ProviderSDKType,
	{ chatPath: string; apiKeyHeaderKey: string; defaultHeaders: Record<string, string> }
> = {
	[ProviderSDKType.ProviderSDKTypeAnthropic]: {
		chatPath: '/v1/messages',
		apiKeyHeaderKey: 'x-api-key',
		defaultHeaders: {
			'Content-Type': 'application/json',
			'anthropic-version': '2023-06-01',
		},
	},
	[ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions]: {
		chatPath: '/v1/chat/completions',
		apiKeyHeaderKey: 'Authorization',
		defaultHeaders: {
			'Content-Type': 'application/json',
		},
	},
	[ProviderSDKType.ProviderSDKTypeOpenAIResponses]: {
		chatPath: '/v1/responses',
		apiKeyHeaderKey: 'Authorization',
		defaultHeaders: {
			'Content-Type': 'application/json',
		},
	},
};

export enum RoleEnum {
	System = 'system',
	Developer = 'developer',
	User = 'user',
	Assistant = 'assistant',
	Function = 'function',
	Tool = 'tool',
}

export enum ReasoningType {
	HybridWithTokens = 'hybridWithTokens',
	SingleWithLevels = 'singleWithLevels',
}

export enum ReasoningLevel {
	None = 'none',
	Minimal = 'minimal',
	Low = 'low',
	Medium = 'medium',
	High = 'high',
	XHigh = 'xhigh',
}

export enum Status {
	None = '',
	InProgress = 'inProgress',
	Completed = 'completed',
	Incomplete = 'incomplete',
	Failed = 'failed',
	Cancelled = 'cancelled',
	Queued = 'queued',
	Searching = 'searching',
}

export enum ReasoningSummaryStyle {
	Auto = 'auto',
	Concise = 'concise',
	Detailed = 'detailed',
}

export interface ReasoningParam {
	type: ReasoningType;
	level: ReasoningLevel;
	tokens: number;
	summaryStyle?: ReasoningSummaryStyle;
}

export enum OutputVerbosity {
	Low = 'low',
	Medium = 'medium',
	High = 'high',
	Max = 'max',
}

export enum OutputFormatKind {
	Text = 'text',
	JSONSchema = 'jsonSchema',
}

/**
 * @public
 */
export interface JSONSchemaParam {
	name: string;
	description?: string;
	schema?: Record<string, any>;
	strict?: boolean;
}

/**
 * @public
 */
export interface OutputFormat {
	kind: OutputFormatKind;
	jsonSchemaParam?: JSONSchemaParam;
}
export interface OutputParam {
	format?: OutputFormat;
	verbosity?: OutputVerbosity;
}
export interface ModelParam {
	name: string;
	stream: boolean;
	maxPromptLength: number;
	maxOutputLength: number;
	temperature?: number;
	reasoning?: ReasoningParam;
	systemPrompt: string;
	timeout: number;
	outputParam?: OutputParam;
	stopSequences?: string[];
	additionalParametersRawJSON?: string;
}

export const DefaultModelParams: ModelParam = {
	name: '',
	stream: false,
	maxPromptLength: 2048,
	maxOutputLength: 1024,
	temperature: 0.1,
	outputParam: undefined,
	stopSequences: undefined,
	reasoning: {
		type: ReasoningType.SingleWithLevels,
		level: ReasoningLevel.Medium,
		tokens: 1024,
	},
	systemPrompt: '',
	timeout: 300,
	additionalParametersRawJSON: undefined,
};

export interface InferenceError {
	code: string;
	message: string;
}

export interface InferenceUsage {
	inputTokensTotal: number;
	inputTokensCached: number;
	inputTokensUncached: number;
	outputTokens: number;
	reasoningTokens: number;
}

/**
 * @public
 */
export enum CacheControlKind {
	Ephemeral = 'ephemeral',
}

/**
 * @public
 */
export interface CacheControlEphemeral {
	ttl?: string;
}

export interface CacheControl {
	kind: CacheControlKind;
	cacheControlEphemeral?: CacheControlEphemeral;
}

export enum CitationKind {
	URL = 'urlCitation',
}

/**
 * @public
 */
export interface CitationConfig {
	enabled: boolean;
}

export interface URLCitation {
	url: string;
	title?: string;
	citedText?: string;
	startIndex?: number;
	endIndex?: number;
	encryptedIndex?: string;
}

/**
 * @public
 */
export interface Citation {
	kind: CitationKind;
	urlCitation?: URLCitation;
}

/**
 * @public
 */
export enum ContentItemKind {
	Text = 'text',
	Image = 'image',
	File = 'file',
	Refusal = 'refusal',
}

export interface ContentItemRefusal {
	refusal: string;
}

export interface ContentItemFile {
	id?: string;
	fileName?: string;
	fileMIME?: string;
	fileURL?: string;
	fileData?: string;
	additionalContext?: string;
	citationConfig?: CitationConfig;
}

export enum ImageDetail {
	High = 'high',
	Low = 'low',
	Auto = 'auto',
}

export interface ContentItemImage {
	id?: string;
	detail?: ImageDetail;
	imageName?: string;
	imageMIME?: string;
	imageURL?: string;
	imageData?: string;
}

export interface ContentItemText {
	text: string;
	citations?: Citation[];
}

export interface InputOutputContentItemUnion {
	kind: ContentItemKind;
	textItem?: ContentItemText;
	refusalItem?: ContentItemRefusal;
	imageItem?: ContentItemImage;
	fileItem?: ContentItemFile;
}

export interface InputOutputContent {
	id: string;
	role: RoleEnum;
	status?: Status;
	cacheControl?: CacheControl;
	contents?: InputOutputContentItemUnion[];
}

export interface ReasoningContent {
	id: string;
	role: RoleEnum;
	status?: Status;
	cacheControl?: CacheControl;
	signature?: string;
	summary?: string[];
	thinking?: string[];
	redactedThinking?: string[];
	encryptedContent?: string[];
}

export interface ToolOutputItemUnion {
	kind: ContentItemKind;
	textItem?: ContentItemText;
	imageItem?: ContentItemImage;
	fileItem?: ContentItemFile;
}

/**
 * @public
 */
export enum WebSearchToolCallKind {
	Search = 'search',
	OpenPage = 'openPage',
	Find = 'find',
}

/**
 * @public
 */
export interface WebSearchToolCallFind {
	url: string;
	pattern: string;
}
/**
 * @public
 */
export interface WebSearchToolCallOpenPage {
	url: string;
}
/**
 * @public
 */
export interface WebSearchToolCallSearchSource {
	url: string;
}
/**
 * @public
 */
export interface WebSearchToolCallSearch {
	query: string;
	sources?: WebSearchToolCallSearchSource[];
	input?: Record<string, any>;
}
export interface WebSearchToolCallItemUnion {
	kind: WebSearchToolCallKind;
	searchItem?: WebSearchToolCallSearch;
	openPageItem?: WebSearchToolCallOpenPage;
	findItem?: WebSearchToolCallFind;
}
/**
 * @public
 */
export interface WebSearchToolChoiceItemUserLocation {
	city?: string;
	country?: string;
	region?: string;
	timezone?: string;
}
export interface WebSearchToolChoiceItem {
	maxUses?: number;
	searchContextSize?: string;
	allowedDomains?: string[];
	blockedDomains?: string[];
	userLocation?: WebSearchToolChoiceItemUserLocation;
}

export enum WebSearchToolOutputKind {
	Search = 'search',
	Error = 'error',
}

export interface WebSearchToolOutputError {
	code: string;
}
export interface WebSearchToolOutputSearch {
	url: string;
	title?: string;
	encryptedContent?: string;
	renderedContent?: string;
	page_age?: string;
}
export interface WebSearchToolOutputItemUnion {
	kind: WebSearchToolOutputKind;
	searchItem?: WebSearchToolOutputSearch;
	errorItem?: WebSearchToolOutputError;
}

export enum ToolType {
	Function = 'function',
	Custom = 'custom',
	WebSearch = 'webSearch',
}
export interface ToolCall {
	type: ToolType;
	choiceID: string;
	id: string;
	role: RoleEnum;
	status?: Status;
	cacheControl?: CacheControl;
	callID: string;
	name: string;
	arguments?: string;
	webSearchToolCallItems?: WebSearchToolCallItemUnion[];
}

export interface ToolOutput {
	type: ToolType;
	choiceID: string;
	id: string;
	role: RoleEnum;
	status?: Status;
	cacheControl?: CacheControl;
	callID: string;
	name: string;
	isError: boolean;
	signature?: string;
	contents?: ToolOutputItemUnion[];
	webSearchToolOutputItems?: WebSearchToolOutputItemUnion[];
}

export interface ToolChoice {
	type: ToolType;
	id: string;
	cacheControl?: CacheControl;
	name: string;
	description?: string;
	arguments?: Record<string, any>;
	webSearchArguments?: WebSearchToolChoiceItem;
}

/**
 * @public
 */
export enum InputKind {
	InputMessage = 'inputMessage',
	OutputMessage = 'outputMessage',
	ReasoningMessage = 'reasoningMessage',
	FunctionToolCall = 'functionToolCall',
	FunctionToolOutput = 'functionToolOutput',
	CustomToolCall = 'customToolCall',
	CustomToolOutput = 'customToolOutput',
	WebSearchToolCall = 'webSearchToolCall',
	WebSearchToolOutput = 'webSearchToolOutput',
}

export enum OutputKind {
	OutputMessage = 'outputMessage',
	ReasoningMessage = 'reasoningMessage',
	FunctionToolCall = 'functionToolCall',
	CustomToolCall = 'customToolCall',
	WebSearchToolCall = 'webSearchToolCall',
	WebSearchToolOutput = 'webSearchToolOutput',
}

export interface InputUnion {
	kind: InputKind;
	inputMessage?: InputOutputContent;
	outputMessage?: InputOutputContent;
	reasoningMessage?: ReasoningContent;
	functionToolCall?: ToolCall;
	functionToolOutput?: ToolOutput;
	customToolCall?: ToolCall;
	customToolOutput?: ToolOutput;
	webSearchToolCall?: ToolCall;
	webSearchToolOutput?: ToolOutput;
}
export interface OutputUnion {
	kind: OutputKind;
	outputMessage?: InputOutputContent;
	reasoningMessage?: ReasoningContent;
	functionToolCall?: ToolCall;
	customToolCall?: ToolCall;
	webSearchToolCall?: ToolCall;
	webSearchToolOutput?: ToolOutput;
}

export interface FetchCompletionResponse {
	outputs?: OutputUnion[];
	usage?: InferenceUsage;
	error?: InferenceError;
	debugDetails?: any;
}

export interface CompletionResponseBody {
	inferenceResponse?: FetchCompletionResponse;
	hydratedCurrentInputs?: InputUnion[];
}

/**
 * @public
 * Status for a tool-call chip in the composer/history.
 */
export type UIToolCallStatus = 'pending' | 'running' | 'succeeded' | 'failed' | 'discarded';

/**
 * UI representation of a tool call (for chips).
 * `type` is the inference ToolType (`function` | `custom` | `webSearch`),
 * but kept as string here to avoid a circular import.
 */
export interface UIToolCall {
	id: string;
	callID: string;
	name: string;
	arguments?: string;
	webSearchToolCallItems?: WebSearchToolCallItemUnion[];
	type: ToolStoreChoiceType;
	choiceID: string;
	status: UIToolCallStatus;
	toolStoreChoice: ToolStoreChoice;
	errorMessage?: string;
}

export interface UIToolOutput {
	id: string;
	callID: string;
	name: string;

	type: ToolStoreChoiceType;
	choiceID: string;
	toolStoreChoice: ToolStoreChoice;

	/** Short human-readable label used in chips. */
	summary: string;
	toolOutputs?: ToolOutputUnion[];
	webSearchToolOutputItems?: WebSearchToolOutputItemUnion[];

	isError?: boolean;
	errorMessage?: string;

	arguments?: string;
	webSearchToolCallItems?: WebSearchToolCallItemUnion[];
}
