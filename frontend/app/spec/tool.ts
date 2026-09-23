import type { JSONRawString, JSONSchema } from '@/lib/jsonschema_utils';

// Shared type for "what args editor is currently open?"
export type ToolArgsTarget =
	{ kind: 'attached'; selectionID: string } | { kind: 'conversation'; key: string } | { kind: 'webSearch' };

export enum ToolStoreChoiceType {
	Function = 'function',
	Custom = 'custom',
	WebSearch = 'webSearch',
}

export interface ToolRef {
	bundleID: string;
	toolSlug: string;
	toolVersion: string;
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
	SDK = 'sdk',
}

interface GoToolImpl {
	/** Built-in Go registry key. */
	func: string;
}

interface SDKToolImpl {
	sdkType: string;
}

export enum ToolOutputKind {
	None = 'none',
	Text = 'text',
	Image = 'image',
	File = 'file',
}

interface ToolOutputFile {
	fileName: string;
	fileMIME: string;
	fileData: string;
}

interface ToolOutputImage {
	detail: string;
	imageName: string;
	imageMIME: string;
	imageData: string;
}

interface ToolOutputText {
	text: string;
}

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
	autoExecute: boolean;

	argSchema: JSONSchema;
	userArgSchema?: JSONSchema;

	llmToolType: ToolStoreChoiceType;
	type: ToolImplType;
	goImpl?: GoToolImpl;
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
}

export interface ToolListItem {
	bundleID: string;
	bundleSlug: string;
	toolSlug: string;
	toolVersion: string;
	isBuiltIn: boolean;
	toolDefinition: Tool;
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
