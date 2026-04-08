import { useEffect, useRef } from 'react';

import { createPortal } from 'react-dom';

import { FiTool, FiX } from 'react-icons/fi';

import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import { ToolOutputKind, type ToolOutputUnion, type ToolStoreChoice } from '@/spec/tool';

import { ModalBackdrop } from '@/components/modal_backdrop';

import { MessageContentCard } from '@/chats/messages/message_content_card';
import { formatToolCallLabel } from '@/tools/lib/tool_call_utils';
import { formatToolOutputSummary } from '@/tools/lib/tool_output_utils';

function buildTextCodeBlock(text: string): string {
	return ['```text', text, '```'].join('\n');
}

function buildJSONCodeBlock(value: unknown): string {
	try {
		return ['```json', JSON.stringify(value, null, 2), '```'].join('\n');
	} catch {
		return buildTextCodeBlock(String(value));
	}
}

function parseStructuredJSONString(raw?: string): unknown {
	if (typeof raw !== 'string') return undefined;

	const trimmed = raw.trim();
	if (!trimmed) return undefined;

	try {
		const parsed = JSON.parse(trimmed);
		if (parsed !== null && typeof parsed === 'object') {
			return parsed;
		}
	} catch {
		// ignore
	}

	return undefined;
}

function buildJSONOrTextCodeBlock(raw?: string): string | null {
	if (typeof raw !== 'string' || raw.trim().length === 0) {
		return null;
	}

	return buildJSONCodeBlock(parseStructuredJSONString(raw) ?? raw);
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function normalizeStructuredJSONStringDeep(value: unknown): unknown {
	const parsed = typeof value === 'string' ? parseStructuredJSONString(value) : undefined;
	if (parsed !== undefined) {
		return normalizeStructuredJSONStringDeep(parsed);
	}

	if (Array.isArray(value)) {
		return value.map(item => normalizeStructuredJSONStringDeep(item));
	}
	if (isRecord(value)) {
		return Object.fromEntries(
			Object.entries(value).map(([key, item]) => [key, normalizeStructuredJSONStringDeep(item)])
		);
	}
	return value;
}

// eslint-disable-next-line @typescript-eslint/no-unnecessary-type-parameters
function normalizeStructuredDisplayObject<T extends object>(value: T): Record<string, unknown> {
	const normalized = normalizeStructuredJSONStringDeep(value);
	return isRecord(normalized) ? normalized : (value as Record<string, unknown>);
}

function buildToolOutputItemLabel(item: ToolOutputUnion, index: number): string {
	switch (item.kind) {
		case ToolOutputKind.Text:
			return `Text item ${index + 1}`;
		case ToolOutputKind.Image:
			return `Image item ${index + 1}`;
		case ToolOutputKind.File:
			return `File item ${index + 1}`;
		case ToolOutputKind.None:
		default:
			return `Output item ${index + 1}`;
	}
}

function buildToolOutputItemsMarkdown(items?: ToolOutputUnion[]): string | null {
	if (!items || items.length === 0) return null;

	const nonEmptyItems = items.filter(item => {
		if (item.kind === ToolOutputKind.Text) {
			return (item.textItem?.text?.trim().length ?? 0) > 0;
		}
		return true;
	});

	if (nonEmptyItems.length === 0) return null;

	if (nonEmptyItems.length === 1 && nonEmptyItems[0].kind === ToolOutputKind.Text) {
		return buildJSONOrTextCodeBlock(nonEmptyItems[0].textItem?.text) ?? null;
	}
	const parts: string[] = [];

	for (const [index, item] of nonEmptyItems.entries()) {
		parts.push(`#### ${buildToolOutputItemLabel(item, index)}`, '');

		if (item.kind === ToolOutputKind.Text) {
			parts.push(buildJSONOrTextCodeBlock(item.textItem?.text) ?? '_Empty text item._');
		} else {
			parts.push(buildJSONCodeBlock(normalizeStructuredJSONStringDeep(item)));
		}

		if (index < nonEmptyItems.length - 1) {
			parts.push('');
		}
	}
	return parts.join('\n');
}

export type ToolDetailsState =
	| { kind: 'choice'; choice: ToolStoreChoice }
	| { kind: 'call'; call: UIToolCall }
	| { kind: 'output'; output: UIToolOutput }
	| null;

interface ToolDetailsModalProps {
	state: ToolDetailsState;
	onClose: () => void;
}

function getChoiceDisplayInfo(c: ToolStoreChoice) {
	const display = (c.displayName && c.displayName.length > 0 ? c.displayName : c.toolSlug) || 'Tool';
	const slug = `${c.bundleID}/${c.toolSlug}@${c.toolVersion}`;
	return { display, slug };
}

function buildPayload(state: Exclude<ToolDetailsState, null>): { title: string; payload: unknown } {
	switch (state.kind) {
		case 'choice': {
			const c = state.choice;
			const { display, slug } = getChoiceDisplayInfo(c);

			return {
				title: `Tool choice • ${display}`,
				payload: {
					...normalizeStructuredDisplayObject(c),
					__meta: {
						identity: slug,
					},
				},
			};
		}
		case 'call': {
			const call = state.call;
			return {
				title: `Tool call • ${formatToolCallLabel(call)}`,
				payload: normalizeStructuredJSONStringDeep(call),
			};
		}
		case 'output': {
			const out = state.output;

			return {
				title: `Tool output • ${
					out.summary && out.summary.length > 0 ? out.summary : formatToolOutputSummary(out.name)
				}`,
				payload: normalizeStructuredJSONStringDeep(out),
			};
		}
	}
}

// Human-oriented "primary" view for a tool choice.
function buildChoicePrimaryContent(choice: ToolStoreChoice): string {
	const { display, slug } = getChoiceDisplayInfo(choice);
	const lines: string[] = [];

	lines.push(`### Tool: ${display}`);
	if (slug) {
		lines.push(`### ID: \`${slug}\``);
	}

	if (choice.description) {
		lines.push('');
		lines.push(`### Description: ${choice.description}`);
	}

	const userArgsBlock = buildJSONOrTextCodeBlock(choice.userArgSchemaInstance);
	if (userArgsBlock) {
		lines.push('');
		lines.push('### Tool options');
		lines.push('');
		lines.push(userArgsBlock);
	}

	return lines.join('\n');
}

// Human-oriented "primary" view for a tool call.
function buildCallPrimaryContent(call: UIToolCall): string {
	const lines: string[] = [];

	lines.push(`### Tool: ${call.name}`);
	lines.push(`### Call ID: \`${call.callID}\``);
	lines.push(`### Status: \`${call.status}\``);
	const errorBlock = buildJSONOrTextCodeBlock(call.errorMessage);
	if (errorBlock) {
		lines.push('');
		lines.push('### Error details');
		lines.push('');
		lines.push(errorBlock);
	}

	lines.push('');

	const argumentsBlock = buildJSONOrTextCodeBlock(call.arguments);
	if (argumentsBlock) {
		lines.push('### Arguments');
		lines.push('');
		lines.push(argumentsBlock);
	} else {
		lines.push('### Arguments: no arguments provided for this call');
	}

	if (call.webSearchToolCallItems && (call.webSearchToolCallItems as any[]).length > 0) {
		lines.push('');
		lines.push('### Web-search call items');
		lines.push('');
		lines.push(buildJSONCodeBlock(call.webSearchToolCallItems));
	}

	return lines.join('\n');
}

// Human-oriented "primary" view for a tool output.
function buildOutputPrimaryContent(output: UIToolOutput): string {
	const lines: string[] = [];

	const titleText = output.summary && output.summary.length > 0 ? output.summary : formatToolOutputSummary(output.name);

	lines.push(`### Summary: ${titleText}`);
	lines.push(`### Tool: ${output.name}`);
	lines.push(`### Call ID: \`${output.callID}\``);
	if (typeof output.isError === 'boolean') {
		lines.push(`### Status: \`${output.isError ? 'error' : 'ok'}\``);
	}
	const argumentsBlock = buildJSONOrTextCodeBlock(output.arguments);
	if (argumentsBlock) {
		lines.push('');
		lines.push('### Arguments');
		lines.push('');
		lines.push(argumentsBlock);
	}
	const errorBlock = buildJSONOrTextCodeBlock(output.errorMessage);
	if (errorBlock) {
		lines.push('');
		lines.push('### Error details');
		lines.push('');
		lines.push(errorBlock);
	}
	lines.push('');

	let resultBlock: string | null = null;

	if (output.toolOutputs && output.toolOutputs.length > 0) {
		resultBlock = buildToolOutputItemsMarkdown(output.toolOutputs);
	} else if (output.webSearchToolOutputItems && output.webSearchToolOutputItems.length > 0) {
		resultBlock = buildJSONCodeBlock(normalizeStructuredJSONStringDeep(output.webSearchToolOutputItems));
	}

	if (!resultBlock) {
		lines.push('### Tool result');
		lines.push('');
		lines.push('_Tool returned no output._');
	} else {
		lines.push('### Tool result');
		lines.push('');
		lines.push(resultBlock);
	}

	return lines.join('\n');
}

export function ToolDetailsModal({ state, onClose }: ToolDetailsModalProps) {
	const dialogRef = useRef<HTMLDialogElement | null>(null);

	useEffect(() => {
		if (!state) return;
		const dialog = dialogRef.current;
		if (!dialog) return;
		if (!dialog.open) dialog.showModal();

		return () => {
			if (dialog.open) dialog.close();
		};
	}, [state]);

	const handleDialogClose = () => {
		onClose();
	};

	if (!state) return null;
	const { title, payload } = buildPayload(state);

	// Raw JSON payload, rendered as a fenced code block for syntax highlighting.
	let rawPayloadMarkdown = '### Raw Payload\n\n';
	try {
		const json = JSON.stringify(payload, null, 2);
		rawPayloadMarkdown += ['```json', json, '```'].join('\n');
	} catch (err) {
		rawPayloadMarkdown +=
			'Error serializing payload:\n\n' +
			((err as Error).message ?? 'Unknown serialization error ' + JSON.stringify(err));
	}

	// Human-oriented primary content (semantics-first).
	let primaryContent = '';
	let baseMessageId = 'tool-details';

	switch (state.kind) {
		case 'choice': {
			primaryContent = buildChoicePrimaryContent(state.choice);
			const choiceId = state.choice.toolSlug;
			baseMessageId = `tool-choice:${choiceId}`;
			break;
		}
		case 'call': {
			primaryContent = buildCallPrimaryContent(state.call);
			baseMessageId = `tool-call:${state.call.id}`;
			break;
		}
		case 'output': {
			primaryContent = buildOutputPrimaryContent(state.output);
			baseMessageId = `tool-output:${state.output.id}`;
			break;
		}
	}

	return createPortal(
		<dialog ref={dialogRef} className="modal" onClose={handleDialogClose}>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-[80vw] overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					{/* header */}
					<div className="mb-4 flex items-center justify-between">
						<h3 className="flex items-center gap-2 text-lg font-bold">
							<FiTool size={16} />
							<span>{title}</span>
						</h3>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={() => dialogRef.current?.close()}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					{/* Primary, human-friendly view (semantics first) */}
					{primaryContent.trim().length > 0 && (
						<div className="mb-4">
							<MessageContentCard
								messageID={`${baseMessageId}:primary`}
								content={primaryContent}
								streamedText=""
								isStreaming={false}
								isBusy={false}
								align="items-start text-left"
								renderAsMarkdown={true}
							/>
						</div>
					)}

					{/* Raw payload for full inspection */}
					<div>
						<MessageContentCard
							messageID={`${baseMessageId}:raw`}
							content={rawPayloadMarkdown}
							streamedText=""
							isStreaming={false}
							isBusy={false}
							align="items-start text-left"
							renderAsMarkdown={true}
						/>
					</div>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>,
		document.body
	);
}
