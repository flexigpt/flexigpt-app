import { ElementApi, NodeApi, type Path } from 'platejs';
import type { PlateEditor } from 'platejs/react';

import {
	ContentItemKind,
	ImageDetail,
	InputKind,
	type InputUnion,
	OutputKind,
	type OutputUnion,
	type ToolCall,
	type ToolOutputItemUnion,
	type UIToolCall,
} from '@/spec/inference';
import {
	type Tool,
	ToolOutputKind,
	type ToolOutputUnion,
	type ToolStoreChoice,
	ToolStoreChoiceType,
	type UIToolStoreChoice,
	type UIToolUserArgsStatus,
} from '@/spec/tool';

import { getRequiredFromJSONSchema, isJSONObject, type JSONSchema } from '@/lib/jsonschema_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

// Keys for the tools combobox and inline elements
export const KEY_TOOL_SELECTION = 'toolSelection';
// export const KEY_TOOL_PLUS_COMMAND = 'toolPlus';
// const KEY_TOOL_PLUS_INPUT = 'toolPlusInput';

export type ToolSelectionElementNode = {
	type: typeof KEY_TOOL_SELECTION;
	choiceID: string;
	bundleID: string;
	bundleSlug?: string;
	toolSlug: string;
	toolVersion: string;
	selectionID: string;
	toolType: ToolStoreChoiceType;
	autoExecute: boolean;
	userArgSchemaInstance?: string;

	toolSnapshot?: Tool;
	overrides?: {
		displayName?: string;
		description?: string;
		tags?: string[];
	};

	// inline+void node needs a text child
	children: [{ text: '' }];
};

/**
 * Inspect a tool's userArgSchema and a JSON-encoded instance string and
 * compute whether all required keys are populated.
 *
 * We intentionally treat:
 *  - no schema            => satisfied (no args required)
 *  - schema with no "required" keys => satisfied
 *  - invalid / non-object instance  => all required missing
 */
export function computeToolUserArgsStatus(
	schema: JSONSchema | undefined,
	rawInstance?: string | null
): UIToolUserArgsStatus {
	const base: UIToolUserArgsStatus = {
		hasSchema: false,
		requiredKeys: [],
		missingRequired: [],
		isInstancePresent: false,
		isInstanceJSONValid: false,
		isSatisfied: true,
	};

	if (!schema || !isJSONObject(schema)) {
		// No schema at all -> nothing to validate.
		return base;
	}

	const required = getRequiredFromJSONSchema(schema) ?? [];

	const status: UIToolUserArgsStatus = {
		...base,
		hasSchema: true,
		requiredKeys: required,
		isSatisfied: true,
	};

	if (required.length === 0) {
		// Schema exists but does not require anything -> always satisfied.
		return status;
	}

	// From here on, there ARE required keys.
	if (!rawInstance || rawInstance.trim() === '') {
		return {
			...status,
			isInstancePresent: false,
			isInstanceJSONValid: false,
			missingRequired: required,
			isSatisfied: false,
		};
	}

	status.isInstancePresent = true;

	let parsed: unknown;
	try {
		parsed = JSON.parse(rawInstance);
	} catch {
		return {
			...status,
			isInstanceJSONValid: false,
			missingRequired: required,
			isSatisfied: false,
		};
	}

	if (!parsed || typeof parsed !== 'object') {
		return {
			...status,
			isInstanceJSONValid: false,
			missingRequired: required,
			isSatisfied: false,
		};
	}

	status.isInstanceJSONValid = true;

	const obj = parsed as Record<string, unknown>;
	const missing: string[] = [];

	for (const key of required) {
		const v = obj[key];
		if (v === undefined || v === null) {
			missing.push(key);
			continue;
		}
		if (typeof v === 'string' && v.trim() === '') {
			missing.push(key);
			continue;
		}
	}

	status.missingRequired = missing;
	status.isSatisfied = missing.length === 0;
	return status;
}

/**
 * Human-friendly tool name for display.
 * Accepts forms like:
 *   "bundleSlug/toolSlug@version"
 *   "bundleID/toolSlug@version"
 *   "toolSlug"
 */
export function getPrettyToolName(name: string): string {
	if (!name) return 'Tool';
	let base = name;
	if (base.includes('/')) {
		const parts = base.split('/');
		base = parts[parts.length - 1] || base;
	}
	if (base.includes('@')) {
		base = base.split('@')[0] || base;
	}
	return base.replace(/[-_]/g, ' ');
}

/**
 * Best-effort short summary of tool-call arguments for chip labels.
 */
function summarizeToolCallArguments(args: string): string | undefined {
	if (!args) return undefined;
	try {
		const parsed = JSON.parse(args);
		if (parsed == null || typeof parsed !== 'object') {
			return typeof parsed === 'string' ? parsed : undefined;
		}
		const obj = parsed as Record<string, unknown>;
		const primaryKeys = ['file', 'path', 'url', 'query', 'id', 'name'];
		const parts: string[] = [];

		for (const key of primaryKeys) {
			if (obj[key] != null) {
				// eslint-disable-next-line @typescript-eslint/no-base-to-string
				parts.push(String(obj[key]));
			}
		}

		if (parts.length === 0) {
			const keys = Object.keys(obj);
			for (const key of keys.slice(0, 2)) {
				parts.push(`${key}=${String(obj[key])}`);
			}
		}

		return parts.length ? parts.join(', ') : undefined;
	} catch {
		return undefined;
	}
}

/**
 * Label used for tool-call chips in composer / history.
 */
export function formatToolCallLabel(call: UIToolCall): string {
	const pretty = getPrettyToolName(call.name);
	const argSummary = summarizeToolCallArguments(call.arguments ?? '');
	return argSummary ? `${pretty}: ${argSummary}` : pretty;
}

/**
 * Default summary label for a tool-output chip.
 */
export function formatToolOutputSummary(name: string): string {
	const pretty = getPrettyToolName(name);
	return `Result: ${pretty}`;
}

// Helper: used for summaries / error messages
export function extractPrimaryTextFromToolOutputs(outputs?: ToolOutputUnion[]): string | undefined {
	if (!outputs?.length) return undefined;

	const texts = outputs
		.filter(o => o.kind === ToolOutputKind.Text && o.textItem?.text)
		.map(o => o.textItem?.text.trim())
		.filter(Boolean);

	if (!texts.length) return undefined;

	return texts.join('\n\n');
}

// Convert the editor's attached-tool shape into the persisted ToolStoreChoice shape.
export function editorAttachedToolToToolChoice(att: UIToolStoreChoice): ToolStoreChoice {
	return {
		choiceID: att.choiceID,
		bundleID: att.bundleID,
		toolSlug: att.toolSlug,
		toolVersion: att.toolVersion,
		displayName: att.displayName,
		description: att.description,
		toolID: att.toolID,
		toolType: att.toolType,
		autoExecute: att.autoExecute,
		userArgSchemaInstance: att.userArgSchemaInstance,
	};
}

function toolChoiceIdentityKey(tool: ToolStoreChoice): string {
	return toolIdentityKey(tool.bundleID, undefined, tool.toolSlug, tool.toolVersion);
}

export function dedupeToolChoices(choices: ToolStoreChoice[]): ToolStoreChoice[] {
	const out: ToolStoreChoice[] = [];
	const seen = new Set<string>();

	for (const t of choices ?? []) {
		const key = toolChoiceIdentityKey(t);
		if (seen.has(key)) continue;
		seen.add(key);
		out.push(t);
	}

	return out;
}

// Build a stable identity key for a tool selection (bundle + slug + version).
// Prefer bundleID when present, otherwise fall back to bundleSlug.
export function toolIdentityKey(
	bundleID: string | undefined,
	bundleSlug: string | undefined,
	toolSlug: string,
	toolVersion: string
): string {
	const bundlePart = bundleID ? `id:${bundleID}` : `slug:${bundleSlug ?? ''}`;
	return `${bundlePart}/${toolSlug}@${toolVersion}`;
}

function toolIdentityKeyFromNode(
	n: Pick<ToolSelectionElementNode, 'bundleID' | 'bundleSlug' | 'toolSlug' | 'toolVersion'>
): string {
	return toolIdentityKey(n.bundleID, n.bundleSlug, n.toolSlug, n.toolVersion);
}

function getAttachedToolKeySet(editor: PlateEditor): Set<string> {
	const keys = new Set<string>();
	for (const [el] of NodeApi.elements(editor)) {
		if (ElementApi.isElementType(el, KEY_TOOL_SELECTION)) {
			keys.add(toolIdentityKeyFromNode(el as unknown as ToolSelectionElementNode));
		}
	}
	return keys;
}

// Insert a hidden tool selection chip (inline+void) to drive bottom bar UI.
export function insertToolSelectionNode(
	editor: PlateEditor,
	item: {
		bundleID: string;
		bundleSlug?: string;
		toolSlug: string;
		toolVersion: string;
	},
	toolSnapshot?: Tool,
	opts?: {
		toolType?: ToolStoreChoiceType;
		choiceID?: string;
		autoExecute: boolean;
		userArgSchemaInstance?: string;
	}
) {
	const identity = toolIdentityKey(item.bundleID, item.bundleSlug, item.toolSlug, item.toolVersion);
	if (getAttachedToolKeySet(editor).has(identity)) {
		editor.tf.focus();
		return;
	}

	const selectionID = `tool:${item.bundleID}/${item.toolSlug}@${item.toolVersion}:${Date.now().toString(36)}${Math.random()
		.toString(36)
		.slice(2, 8)}`;

	const node: ToolSelectionElementNode = {
		type: KEY_TOOL_SELECTION,
		choiceID: opts?.choiceID ?? getUUIDv7(),

		bundleID: item.bundleID,
		bundleSlug: item.bundleSlug,
		toolSlug: item.toolSlug,
		toolVersion: item.toolVersion,

		toolType: opts?.toolType ?? toolSnapshot?.llmToolType ?? ToolStoreChoiceType.Function,
		autoExecute: opts?.autoExecute ?? toolSnapshot?.autoExecReco ?? false,
		userArgSchemaInstance: opts?.userArgSchemaInstance,

		selectionID,
		toolSnapshot,
		overrides: {},
		children: [{ text: '' }],
	};

	editor.tf.withoutNormalizing(() => {
		// Insert the tool chip (invisible inline) and an empty text leaf after it
		// so the caret has a cheap place to land without forcing block normalization.
		editor.tf.insertNodes([node, { text: '' }], { select: true });
	});
}

// List tool nodes with path in document order.
// By default, returns only the first occurrence for each unique tool identity.
export function getToolNodesWithPath(
	editor: PlateEditor,
	uniqueByIdentity?: boolean
): Array<[ToolSelectionElementNode, Path]> {
	const out: Array<[ToolSelectionElementNode, Path]> = [];
	const unique = uniqueByIdentity ?? true;
	const seen = unique ? new Set<string>() : undefined;
	for (const [el, path] of NodeApi.elements(editor)) {
		if (ElementApi.isElementType(el, KEY_TOOL_SELECTION)) {
			const n = el as unknown as ToolSelectionElementNode;
			if (unique) {
				const key = toolIdentityKeyFromNode(n);
				if ((seen as Set<string>).has(key)) continue;
				(seen as Set<string>).add(key);
			}
			out.push([n, path]);
		}
	}
	return out;
}

// Remove all instances of a tool by identity key (bundle+slug+version).
export function removeToolByKey(editor: PlateEditor, identityKey: string) {
	const paths: Path[] = [];
	for (const [el, path] of NodeApi.elements(editor)) {
		if (ElementApi.isElementType(el, KEY_TOOL_SELECTION)) {
			const n = el as unknown as ToolSelectionElementNode;
			if (toolIdentityKeyFromNode(n) === identityKey) {
				paths.push(path);
			}
		}
	}
	// Remove from last to first to avoid path shift issues.
	for (const p of paths.reverse()) {
		try {
			editor.tf.removeNodes({ at: p });
		} catch {
			// swallow
		}
	}
}

// Build a serializable list of attached tools for submission
export function getAttachedTools(editor: PlateEditor): UIToolStoreChoice[] {
	const items: UIToolStoreChoice[] = [];
	const seen = new Set<string>();

	for (const [el] of NodeApi.elements(editor)) {
		if (ElementApi.isElementType(el, KEY_TOOL_SELECTION)) {
			const n = el as unknown as ToolSelectionElementNode;
			const key = toolIdentityKeyFromNode(n);
			if (seen.has(key)) continue;
			seen.add(key);
			items.push({
				choiceID: n.choiceID,
				selectionID: n.selectionID,
				bundleID: n.bundleID,
				toolSlug: n.toolSlug,
				toolVersion: n.toolVersion,
				displayName: n.overrides?.displayName
					? n.overrides.displayName
					: n.toolSnapshot?.displayName
						? n.toolSnapshot.displayName
						: n.toolSlug,
				description: n.overrides?.description
					? n.overrides.description
					: n.toolSnapshot?.description
						? n.toolSnapshot.description
						: n.toolSlug,
				toolID: n.toolSnapshot?.id,
				toolType: n.toolType,
				autoExecute: n.autoExecute,
				userArgSchemaInstance: n.userArgSchemaInstance,
			});
		}
	}
	return items;
}

function mapImageDetail(detail?: string): ImageDetail | undefined {
	if (!detail) return undefined;
	switch (detail.toLowerCase()) {
		case 'high':
			return ImageDetail.High;
		case 'low':
			return ImageDetail.Low;
		case 'auto':
			return ImageDetail.Auto;
		default:
			return undefined;
	}
}

/**
 * Map inference ToolOutput.contents -> ToolOutputUnion[]
 */
export function mapToolOutputItemsToToolOutputs(contents?: ToolOutputItemUnion[]): ToolOutputUnion[] | undefined {
	if (!contents?.length) return undefined;

	const outputs: ToolOutputUnion[] = [];

	for (const item of contents) {
		switch (item.kind) {
			case ContentItemKind.Text: {
				const text = item.textItem?.text;
				if (text != null) {
					outputs.push({
						kind: ToolOutputKind.Text,
						textItem: { text },
					});
				}
				break;
			}

			case ContentItemKind.Image: {
				const img = item.imageItem;
				if (img) {
					outputs.push({
						kind: ToolOutputKind.Image,
						imageItem: {
							detail: (img.detail ?? ImageDetail.Auto) as string,
							imageName: img.imageName ?? '',
							imageMIME: img.imageMIME ?? '',
							imageData: img.imageData ?? '',
						},
					});
				}
				break;
			}

			case ContentItemKind.File: {
				const file = item.fileItem;
				if (file) {
					outputs.push({
						kind: ToolOutputKind.File,
						fileItem: {
							fileName: file.fileName ?? '',
							fileMIME: file.fileMIME ?? '',
							fileData: file.fileData ?? '',
						},
					});
				}
				break;
			}

			default:
				// Refusal / other kinds are ignored for tool-store outputs
				break;
		}
	}

	return outputs.length ? outputs : undefined;
}

/**
 * Map ToolOutputUnion[] -> inference ToolOutputItemUnion[]
 */
export function mapToolOutputsToToolOutputItems(outputs?: ToolOutputUnion[]): ToolOutputItemUnion[] | undefined {
	if (!outputs?.length) return undefined;

	const contents: ToolOutputItemUnion[] = [];

	for (const out of outputs) {
		switch (out.kind) {
			case ToolOutputKind.Text: {
				const text = out.textItem?.text;
				if (text != null) {
					contents.push({
						kind: ContentItemKind.Text,
						textItem: { text },
					});
				}
				break;
			}

			case ToolOutputKind.Image: {
				const img = out.imageItem;
				if (img) {
					contents.push({
						kind: ContentItemKind.Image,
						imageItem: {
							detail: mapImageDetail(img.detail),
							imageName: img.imageName,
							imageMIME: img.imageMIME,
							imageData: img.imageData,
						},
					});
				}
				break;
			}

			case ToolOutputKind.File: {
				const file = out.fileItem;
				if (file) {
					contents.push({
						kind: ContentItemKind.File,
						fileItem: {
							fileName: file.fileName,
							fileMIME: file.fileMIME,
							fileData: file.fileData,
						},
					});
				}
				break;
			}

			case ToolOutputKind.None:
			default:
				// ignore
				break;
		}
	}

	return contents.length ? contents : undefined;
}

export function collectToolCallsFromInputs(
	inputs: InputUnion[] | undefined,
	existing?: Map<string, ToolCall>
): Map<string, ToolCall> {
	const map = existing ?? new Map<string, ToolCall>();

	const addCall = (call?: ToolCall) => {
		if (call?.callID) map.set(call.callID, call);
	};

	if (!inputs) return map;

	for (const iu of inputs) {
		switch (iu.kind) {
			case InputKind.FunctionToolCall:
				addCall(iu.functionToolCall);
				break;
			case InputKind.CustomToolCall:
				addCall(iu.customToolCall);
				break;
			case InputKind.WebSearchToolCall:
				addCall(iu.webSearchToolCall);
				break;
			default:
				break;
		}
	}

	return map;
}

export function collectToolCallsFromOutputs(
	outputs: OutputUnion[] | undefined,
	existing?: Map<string, ToolCall>
): Map<string, ToolCall> {
	const map = existing ?? new Map<string, ToolCall>();

	const addCall = (call?: ToolCall) => {
		if (call?.callID) map.set(call.callID, call);
	};

	if (!outputs) return map;

	for (const o of outputs) {
		switch (o.kind) {
			case OutputKind.FunctionToolCall:
				addCall(o.functionToolCall);
				break;
			case OutputKind.CustomToolCall:
				addCall(o.customToolCall);
				break;
			case OutputKind.WebSearchToolCall:
				addCall(o.webSearchToolCall);
				break;
			default:
				break;
		}
	}

	return map;
}

// Update autoExecute for all instances of a tool by identity key (bundle+slug+version).
export function setToolAutoExecuteByKey(editor: PlateEditor, identityKey: string, autoExecute: boolean) {
	const paths: Path[] = [];
	for (const [el, path] of NodeApi.elements(editor)) {
		if (ElementApi.isElementType(el, KEY_TOOL_SELECTION)) {
			const n = el as unknown as ToolSelectionElementNode;
			if (toolIdentityKeyFromNode(n) === identityKey) {
				paths.push(path);
			}
		}
	}

	if (paths.length === 0) return;

	editor.tf.withoutNormalizing(() => {
		for (const p of paths) {
			try {
				// Update the hidden carrier node; chips read from this.
				editor.tf.setNodes<ToolSelectionElementNode>({ autoExecute }, { at: p });
			} catch {
				// swallow
			}
		}
	});
}
