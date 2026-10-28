import type { MCPAppRenderTarget, MCPToolAppRenderInfo, MCPToolSelection } from '@/spec/session_mcp';
import type { ToolOutputUnion, ToolSelection, ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

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

export function isReasoningType(value: unknown): value is ReasoningType {
	return typeof value === 'string' && (Object.values(ReasoningType) as string[]).includes(value);
}

export enum ReasoningLevel {
	None = 'none',
	Minimal = 'minimal',
	Low = 'low',
	Medium = 'medium',
	High = 'high',
	XHigh = 'xhigh',
	Max = 'max',
}

export const orderedReasoningLevels: ReasoningLevel[] = [
	ReasoningLevel.None,
	ReasoningLevel.Minimal,
	ReasoningLevel.Low,
	ReasoningLevel.Medium,
	ReasoningLevel.High,
	ReasoningLevel.XHigh,
	ReasoningLevel.Max,
];

export function isReasoningLevel(value: unknown): value is ReasoningLevel {
	return typeof value === 'string' && (Object.values(ReasoningLevel) as string[]).includes(value);
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
	Omitted = 'omitted',
}

export function isReasoningSummaryStyle(value: unknown): value is ReasoningSummaryStyle {
	return typeof value === 'string' && (Object.values(ReasoningSummaryStyle) as string[]).includes(value);
}

enum ReasoningContext {
	Auto = 'auto',
	CurrentTurn = 'current_turn',
	AllTurns = 'all_turns',
}

enum ReasoningMode {
	Standard = 'standard',
	Pro = 'pro',
}

export interface ReasoningParam {
	type: ReasoningType;
	level: ReasoningLevel;
	tokens: number;
	summaryStyle?: ReasoningSummaryStyle;
	context?: ReasoningContext;
	mode?: ReasoningMode;
}

export enum OutputVerbosity {
	Low = 'low',
	Medium = 'medium',
	High = 'high',
	XHigh = 'xhigh',
	Max = 'max',
}

export function isOutputVerbosity(value: unknown): value is OutputVerbosity {
	return typeof value === 'string' && (Object.values(OutputVerbosity) as string[]).includes(value);
}

export enum OutputFormatKind {
	Text = 'text',
	JSONSchema = 'jsonSchema',
}

export function isOutputFormatKind(value: unknown): value is OutputFormatKind {
	return typeof value === 'string' && (Object.values(OutputFormatKind) as string[]).includes(value);
}

export interface JSONSchemaParam {
	name: string;
	description?: string;
	schema?: Record<string, any>;
	strict?: boolean;
}

interface OutputFormat {
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
	cacheControl?: CacheControl;
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
	reasoning: {
		type: ReasoningType.SingleWithLevels,
		level: ReasoningLevel.Medium,
		tokens: 1024,
	},
	systemPrompt: '',
	timeout: 300,
	cacheControl: undefined,
	outputParam: undefined,
	stopSequences: undefined,
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

export enum CacheControlKind {
	Ephemeral = 'ephemeral',
}

export enum CacheControlTTL {
	TTL5m = '5m',
	TTL1h = '1h',
	TTL24h = '24h',
	TTLInMemory = 'in-memory',
}

export const CACHE_CONTROL_TTL_PROVIDER_DEFAULT = '__provider_default__' as const;

export type CacheControlTTLSelection = CacheControlTTL | typeof CACHE_CONTROL_TTL_PROVIDER_DEFAULT;

export interface CacheControl {
	kind: CacheControlKind;
	ttl?: CacheControlTTL;
	key?: string;
}

export enum CitationKind {
	URL = 'urlCitation',
}

interface CitationConfig {
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

interface Citation {
	kind: CitationKind;
	urlCitation?: URLCitation;
}

export enum ContentItemKind {
	Text = 'text',
	Image = 'image',
	File = 'file',
	Refusal = 'refusal',
}

interface ContentItemRefusal {
	refusal: string;
}

interface ContentItemFile {
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

interface ContentItemImage {
	id?: string;
	detail?: ImageDetail;
	imageName?: string;
	imageMIME?: string;
	imageURL?: string;
	imageData?: string;
}

interface ContentItemText {
	text: string;
	citations?: Citation[];
}

interface InputOutputContentItemUnion {
	kind: ContentItemKind;
	textItem?: ContentItemText;
	refusalItem?: ContentItemRefusal;
	imageItem?: ContentItemImage;
	fileItem?: ContentItemFile;
}

interface InputOutputContent {
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
	continuationFingerprint?: string;
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

enum WebSearchToolCallKind {
	Search = 'search',
	OpenPage = 'openPage',
	Find = 'find',
}

interface WebSearchToolCallFind {
	url: string;
	pattern: string;
}

interface WebSearchToolCallOpenPage {
	url: string;
}

interface WebSearchToolCallSearchSource {
	url: string;
}

interface WebSearchToolCallSearch {
	query: string;
	sources?: WebSearchToolCallSearchSource[];
	input?: Record<string, any>;
}
interface WebSearchToolCallItemUnion {
	kind: WebSearchToolCallKind;
	searchItem?: WebSearchToolCallSearch;
	openPageItem?: WebSearchToolCallOpenPage;
	findItem?: WebSearchToolCallFind;
}

interface WebSearchToolChoiceItemUserLocation {
	city?: string;
	country?: string;
	region?: string;
	timezone?: string;
}
interface WebSearchToolChoiceItem {
	maxUses?: number;
	searchContextSize?: string;
	allowedDomains?: string[];
	blockedDomains?: string[];
	userLocation?: WebSearchToolChoiceItemUserLocation;
}

enum WebSearchToolOutputKind {
	Search = 'search',
	Error = 'error',
}

interface WebSearchToolOutputError {
	code: string;
}
interface WebSearchToolOutputSearch {
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

enum ToolType {
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

enum InputKind {
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

export enum UIToolCallStatus {
	Pending = 'pending',
	Running = 'running',
	Succeeded = 'succeeded',
	Failed = 'failed',
	Discarded = 'discarded',
}

/**
 * UI representation of a tool call (for chips).
 * `type` is the inference ToolType (`function` | `custom` | `webSearch`),
 * but kept as string here to avoid a circular import.
 */
export interface UIToolCall {
	id: string;
	callID: string;
	name: string;
	displayName?: string;
	description?: string;
	arguments?: string;
	webSearchToolCallItems?: WebSearchToolCallItemUnion[];
	type: ToolStoreChoiceType;
	choiceID: string;
	status: UIToolCallStatus;
	/**
	 * Optional. Some tool calls are injected at runtime (e.g. skills-*) and do not
	 * correspond to a stored ToolSelection.
	 */
	/**
	 * Session Runtime-owned execution preference for this proposed call.
	 * This is distinct from frontend-only tool picker state and is projected
	 * directly from SessionToolCall.automatic.
	 */
	automatic?: boolean;

	toolStoreChoice?: ToolStoreChoice;

	/** Original selection, including when its current Tool metadata is unavailable. */
	toolSelection?: ToolSelection;

	/**
	 * Optional MCP identity for dynamically hydrated MCP tool calls.
	 * Static FlexiGPT tools use `toolStoreChoice`; MCP tools use this.
	 */
	mcpToolSelection?: MCPToolSelection;

	/**
	 * UI-only hydration flag.
	 * Restored tool calls from persisted history should be shown and remain
	 * manually runnable, but must not enter the auto-exec / auto-submit flow
	 * just because the underlying tool choice has autoExecute=true.
	 */
	suppressAutoExecute?: boolean;

	errorMessage?: string;
}

export interface UIToolOutput {
	id: string;
	callID: string;
	name: string;
	displayName?: string;
	description?: string;

	/**
	 * Runtime-only MCP App protocol tool name.
	 *
	 * This field is not user-facing. Normal chat UI must use `displayName`
	 * and must never render this value.
	 */
	appToolName?: string;

	type: ToolStoreChoiceType;
	choiceID: string;

	/**
	 * Optional for the same reason as UIToolCall.toolStoreChoice.
	 * For regular tool-store tools we typically hydrate this.
	 */
	toolStoreChoice?: ToolStoreChoice;
	/** Original selection retained independently of current metadata availability. */
	toolSelection?: ToolSelection;
	/**
	 * Optional MCP identity for dynamically executed MCP tool outputs.
	 */
	mcpToolSelection?: MCPToolSelection;
	/**
	 * Optional MCP Apps render info for tool calls that produced a UI resource.
	 */
	mcpApp?: MCPToolAppRenderInfo;

	/**
	 * Session-scoped route for an MCP App created by this Tool result.
	 * Unlike mcpApp metadata, this identifies the authoritative Session Runtime
	 * command boundary for resource reads, app tools, and staged context.
	 */
	sessionApp?: MCPAppRenderTarget;

	/** Short human-readable label used in chips. */
	summary: string;
	toolOutputs?: ToolOutputUnion[];
	webSearchToolOutputItems?: WebSearchToolOutputItemUnion[];

	isError?: boolean;
	isSkipped?: boolean;
	errorMessage?: string;

	/**
	 * UI/runtime metadata returned by built-in Skill orchestration.
	 * This is not provider tool content and must not be presented as model-authored output.
	 */
	skillRuntimeMeta?: Record<string, any>;

	arguments?: string;
	webSearchToolCallItems?: WebSearchToolCallItemUnion[];
}
