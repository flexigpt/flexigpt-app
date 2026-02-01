import { useEffect, useRef } from 'react';

import { createPortal } from 'react-dom';

import { FiTool, FiX } from 'react-icons/fi';

import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import type { ToolStoreChoice } from '@/spec/tool';

import { ModalBackdrop } from '@/components/modal_backdrop';

import { MessageContentCard } from '@/chats/messages/message_content_card';
import {
	extractPrimaryTextFromToolStoreOutputs,
	formatToolCallLabel,
	formatToolOutputSummary,
} from '@/chats/tools/tool_editor_utils';

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
					...c,
					__meta: {
						identity: slug,
					},
				},
			};
		}
		case 'call': {
			const call = state.call;
			let args: unknown = call.arguments;
			if (call.arguments) {
				try {
					args = JSON.parse(call.arguments);
				} catch {
					args = call.arguments;
				}
			}
			return {
				title: `Tool call • ${formatToolCallLabel(call)}`,
				payload: {
					...call,
					arguments: args,
				},
			};
		}
		case 'output': {
			const out = state.output;
			return {
				title: `Tool output • ${
					out.summary && out.summary.length > 0 ? out.summary : formatToolOutputSummary(out.name)
				}`,
				payload: out,
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

	return lines.join('\n');
}

// Human-oriented "primary" view for a tool call.
function buildCallPrimaryContent(call: UIToolCall): string {
	const lines: string[] = [];

	lines.push(`### Tool: ${call.name}`);
	lines.push(`### Call ID: \`${call.callID}\``);
	lines.push(`### Status: \`${call.status}\``);
	if (call.errorMessage) {
		lines.push(`Error: ${call.errorMessage}`);
	}

	lines.push('');

	if (call.arguments && call.arguments.trim().length > 0) {
		lines.push('### Arguments');
		lines.push('');

		const raw = call.arguments.trim();
		try {
			const parsed = JSON.parse(raw);
			lines.push('```json');
			lines.push(JSON.stringify(parsed, null, 2));
			lines.push('```');
		} catch {
			lines.push('```text');
			lines.push(raw);
			lines.push('```');
		}
	} else {
		lines.push('### Arguments: no arguments provided for this call');
	}

	if (call.webSearchToolCallItems && (call.webSearchToolCallItems as any[]).length > 0) {
		lines.push('');
		lines.push('### Web-search call items');
		lines.push('');
		lines.push('```json');
		lines.push(JSON.stringify(call.webSearchToolCallItems, null, 2));
		lines.push('```');
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
		lines.push(`## Status: \`${output.isError ? 'error' : 'ok'}\``);
	}
	if (output.errorMessage) {
		lines.push(`### Error message: ${output.errorMessage}`);
	}

	lines.push('');

	let bodyBlock: string | null = null;

	const outputs = output.toolStoreOutputs;

	if (outputs && outputs.length > 0) {
		const primaryText = extractPrimaryTextFromToolStoreOutputs(outputs);

		if (primaryText) {
			const raw = primaryText.trim();
			bodyBlock = ['```text', raw, '```'].join('\n');
		} else {
			try {
				const formatted = JSON.stringify(outputs, null, 2);
				bodyBlock = ['```json', formatted, '```'].join('\n');
			} catch {
				bodyBlock = 'Unable to render tool output.';
			}
		}
	} else if (output.webSearchToolOutputItems && output.webSearchToolOutputItems.length > 0) {
		try {
			const formatted = JSON.stringify(output.webSearchToolOutputItems, null, 2);
			bodyBlock = ['```json', formatted, '```'].join('\n');
		} catch {
			bodyBlock = 'Unable to render web-search tool output.';
		}
	}

	if (!bodyBlock) {
		bodyBlock = '### Body block: _Tool returned no output._';
		lines.push(bodyBlock);
	} else {
		lines.push('### Body block');
		lines.push(bodyBlock);
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
