import type { JSONRawString, JSONSchema } from '@/lib/jsonschema_utils';

export enum ToolStoreChoiceType {
	Function = 'function',
	Custom = 'custom',
	WebSearch = 'webSearch',
}

export interface ToolStoreChoice {
	choiceID: string;
	bundleID: string;
	bundleSlug?: string;

	toolID?: string;
	toolSlug: string;
	toolVersion: string;
	toolType: ToolStoreChoiceType;
	displayName?: string;
	description?: string;

	autoExecute: boolean;
	userArgSchemaInstance?: JSONRawString;
}

export enum ToolImplType {
	Go = 'go',
	HTTP = 'http',
	SDK = 'sdk',
}

/**
 * @public
 */
export interface GoToolImpl {
	/** Fully-qualified registration key, e.g. "github.com/acme/flexigpt/tools.Weather" */
	func: string;
}

export interface SDKToolImpl {
	// SDKType can be ProviderSDKType.
	sdkType: string;
}

/**
 * @public
 */
export interface HTTPAuth {
	type: string;
	in?: string; // "header" | "query" (apiKey only)
	name?: string;
	valueTemplate: string; // may contain ${SECRET}
}

/**
 * @public
 */
export interface HTTPRequest {
	method?: string; // default "GET"
	urlTemplate: string; // http(s)://… may contain ${var}
	query?: Record<string, string>; // k:${var}
	headers?: Record<string, string>; // k:${var}
	body?: string; // raw or template
	auth?: HTTPAuth;
	timeoutMS?: number; // default 10_000
}

/**
 * @public
 */
export enum HTTPBodyOutputMode {
	Auto = 'auto',
	Text = 'text',
	File = 'file',
	Image = 'image',
}

/**
 * @public
 */
export interface HTTPResponse {
	successCodes?: number[]; // default: 2xx
	errorMode?: string; // "fail"(dflt) | "empty"
	bodyOutputMode?: HTTPBodyOutputMode;
}

/**
 * @public
 */
export interface HTTPToolImpl {
	request: HTTPRequest;
	response: HTTPResponse;
}

/**
 * @public
 */
export enum ToolOutputKind {
	None = 'none',
	Text = 'text',
	Image = 'image',
	File = 'file',
}

/**
 * @public
 */
export interface ToolOutputFile {
	fileName: string;
	fileMIME: string;
	fileData: string;
}

/**
 * @public
 */
export interface ToolOutputImage {
	detail: string;
	imageName: string;
	imageMIME: string;
	imageData: string;
}

/**
 * @public
 */
export interface ToolOutputText {
	text: string;
}

/**
 * @public
 */
export interface ToolOutputUnion {
	kind: ToolOutputKind;
	textItem?: ToolOutputText;
	imageItem?: ToolOutputImage;
	fileItem?: ToolOutputFile;
}

export interface Tool {
	schemaVersion: string;

	id: string;
	slug: string;
	version: string;

	displayName: string;
	description?: string;
	tags?: string[];

	userCallable: boolean;
	llmCallable: boolean;
	autoExecReco: boolean;

	argSchema: JSONSchema;
	userArgSchema?: JSONSchema;

	llmToolType: ToolStoreChoiceType;
	type: ToolImplType;
	goImpl?: GoToolImpl;
	httpImpl?: HTTPToolImpl;
	sdkImpl?: SDKToolImpl;

	isEnabled: boolean;
	isBuiltIn: boolean;
	createdAt: string;
	modifiedAt: string;
}

export interface ToolBundle {
	schemaVersion: string;

	id: string;
	slug: string;

	displayName?: string;
	description?: string;
	isEnabled: boolean;
	isBuiltIn: boolean;
	createdAt: string;
	modifiedAt: string;
	softDeletedAt?: string;
}

export interface ToolListItem {
	bundleID: string;
	bundleSlug: string;
	toolSlug: string;
	toolVersion: string;
	isBuiltIn: boolean;
	toolDefinition: Tool;
}

export interface InvokeHTTPOptions {
	timeoutMS?: number;
	extraHeaders?: Record<string, string>;
	secrets?: Record<string, string>;
}

export interface InvokeGoOptions {
	timeoutMS?: number;
}

export interface InvokeToolResponse {
	outputs?: ToolOutputUnion[];
	meta?: Record<string, any>;
	isBuiltIn: boolean;
	isError?: boolean;
	errorMessage?: string;
}

export interface UIToolStoreChoice extends ToolStoreChoice {
	selectionID: string;
}

//  UI-only status of a tool's user-arguments instance vs its JSON schema.
// Used to:
//  - show "Args: OK / N missing" badges on chips
//  - block send when required args are missing
//
export interface UIToolUserArgsStatus {
	/** Tool defines a userArgSchema at all */
	hasSchema: boolean;

	/** All required keys from the schema (if any) */
	requiredKeys: string[];

	/** Subset of requiredKeys that are missing/empty in the instance */
	missingRequired: string[];

	/** Instance string is present (non-empty) */
	isInstancePresent: boolean;

	/** Instance parses as JSON and is an object */
	isInstanceJSONValid: boolean;

	/** True when there is a schema and all required keys are satisfied */
	isSatisfied: boolean;
}
