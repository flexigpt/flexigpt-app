import type { ArtifactRef, MappedTarget, StoreArtifact } from '@/spec/artifact';
import type { CollectionView } from '@/spec/collection';

import type { JSONObject, JSONRawString } from '@/lib/jsonschema_utils';

export type ToolArgsTarget =
	{ kind: 'attached'; selectionID: string } | { kind: 'conversation'; key: string } | { kind: 'webSearch' };

export enum ToolImplType {
	Go = 'go',
	SDK = 'sdk',
}

/**
 * Matches the supported backend SDK tool types.
 * Go tools are advertised to inference as Function.
 */
export enum ToolStoreChoiceType {
	Function = 'function',
	Custom = 'custom',
	WebSearch = 'webSearch',
}

export type ToolJSONSchema = JSONObject | boolean;

type ToolImplementationView =
	| {
			kind: ToolImplType.Go;
			function: string;
	  }
	| {
			kind: ToolImplType.SDK;
			/**
			 * Backend-owned SDK identifier. This is intentionally not limited
			 * to the frontend's currently installed ProviderSDKType values.
			 */
			sdkType: string;
			sdkToolType: ToolStoreChoiceType;
	  };

export interface ToolView {
	artifact: StoreArtifact;
	definitionDigest: string;
	name: string;
	version: string;
	displayName: string;
	description?: string;
	tags?: string[];
	autoExecute: boolean;
	builtIn: boolean;

	inputSchema: ToolJSONSchema;
	userArgSchema?: ToolJSONSchema;
	outputSchema?: ToolJSONSchema;

	implementation: ToolImplementationView;
}

export interface ResolvedToolView {
	tool: ToolView;
	collection: CollectionView;
}

/** Exact aggregate.ToolSelection wire and persistence contract. */
export interface ToolSelection {
	choiceID: string;
	target: MappedTarget;
	autoExecute: boolean;
	userArgSchemaInstance?: JSONRawString;
}

/** UI-only diagnostic. The original selection remains available for inspection. */
export interface ToolSelectionIssue {
	selection: ToolSelection;
	message: string;
}

/**
 * Frontend-only metadata accompanying a backend ToolSelection.
 * Obtain this metadata from the Tool aggregate, never by decoding target.
 */
export interface ToolStoreChoice extends ToolSelection {
	toolType: ToolStoreChoiceType;
	implementationKind: ToolImplType;
	sdkType?: string;

	displayName?: string;
	description?: string;
	toolVersion?: string;

	collectionRef?: ArtifactRef;
	collectionName?: string;
}

export interface UIToolStoreChoice extends ToolStoreChoice {
	selectionID: string;
}

/** An enabled, aggregate-mapped picker entry. */
export interface ToolListItem {
	target: MappedTarget;
	collectionRef: ArtifactRef;
	collectionName: string;
	toolDefinition: ToolView;
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

export interface UIToolUserArgsStatus {
	hasSchema: boolean;
	requiredKeys: string[];
	missingRequired: string[];
	isInstancePresent: boolean;
	isInstanceJSONValid: boolean;
	isSatisfied: boolean;
}
