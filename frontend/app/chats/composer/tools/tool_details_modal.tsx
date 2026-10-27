import { FiTool } from 'react-icons/fi';

import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import type { ToolOutputUnion, ToolSelection, ToolStoreChoice } from '@/spec/tool';
import { ToolOutputKind } from '@/spec/tool';

import {
	buildJSONCodeBlock,
	buildJSONOrTextCodeBlock,
	normalizeStructuredDisplayObject,
	normalizeStructuredJSONStringDeep,
} from '@/lib/jsonschema_utils';
import { redactInternalPresentationFields } from '@/lib/user_facing_presentation';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { getPrettyToolName } from '@/apis/tool_management';

import {
	ManagementModalBody,
	ManagementModalDialog,
	ManagementModalPanel,
} from '@/components/managementui/management_modal_layout';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalHeader } from '@/components/modal/modal_header';

import { MessageContentCard } from '@/chats/messages/message_content_card';
import { mediaItemsFromToolOutputItems, MessageMediaContent } from '@/chats/messages/message_media_content';

function buildToolOutputItemLabel(item: ToolOutputUnion, index: number): string {
	switch (item.kind) {
		case ToolOutputKind.Text:
			return `Text item ${index + 1}`;
		case ToolOutputKind.Image:
			return `Image item ${index + 1}`;
		case ToolOutputKind.File:
			return `File item ${index + 1}`;

		default:
			return `Output item ${index + 1}`;
	}
}

function buildToolOutputItemsMarkdown(items?: ToolOutputUnion[]): string | null {
	if (!items || items.length === 0) {
		return null;
	}

	const nonEmptyItems = items.filter(item => {
		if (item.kind === ToolOutputKind.Text) {
			return (item.textItem?.text?.trim().length ?? 0) > 0;
		}
		return true;
	});

	if (nonEmptyItems.length === 0) {
		return null;
	}

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
	| { kind: 'selection'; selection: ToolSelection; message?: string }
	| { kind: 'call'; call: UIToolCall }
	| { kind: 'output'; output: UIToolOutput }
	| null;

interface ToolDetailsModalProps {
	state: ToolDetailsState;
	onClose: () => void;
}

function getChoiceDisplayInfo(c: ToolStoreChoice) {
	const display = c.displayName || c.target.name || 'Tool';
	return { display };
}

function getToolCallDisplayName(call: UIToolCall): string {
	return call.mcpToolSelection?.toolName || getPrettyToolName(call.name);
}

function formatToolOutputSummary(n: string): string {
	const pretty = getPrettyToolName(n);
	return `Result: ${pretty}`;
}

function buildPayload(state: Exclude<ToolDetailsState, null>): { title: string; payload: unknown } {
	switch (state.kind) {
		case 'selection':
			return {
				title: `Tool selection: ${state.selection.target.name}`,
				payload: redactInternalPresentationFields({ selection: state.selection, resolutionError: state.message }),
			};
		case 'choice': {
			const c = state.choice;
			const { display } = getChoiceDisplayInfo(c);

			return {
				title: `Tool choice • ${display}`,
				payload: redactInternalPresentationFields(normalizeStructuredDisplayObject(c)),
			};
		}
		case 'call': {
			const call = state.call;
			return {
				title: `Tool call • ${getToolCallDisplayName(call)}`,
				payload: redactInternalPresentationFields(normalizeStructuredJSONStringDeep(call)),
			};
		}
		case 'output': {
			const out = state.output;

			return {
				title: `Tool output • ${
					out.summary && out.summary.length > 0 ? out.summary : formatToolOutputSummary(out.name)
				}`,
				payload: redactInternalPresentationFields(normalizeStructuredJSONStringDeep(out)),
			};
		}
	}
}

// Human-oriented "primary" view for a tool choice.
function buildChoicePrimaryContent(choice: ToolStoreChoice): string {
	const { display } = getChoiceDisplayInfo(choice);
	const lines: string[] = [`### Tool: ${display}`];

	if (choice.pluginName?.trim() && choice.pluginName.trim() !== display) {
		lines.push(`### Provided by: ${choice.pluginName.trim()}`);
	}

	if (choice.description) {
		lines.push('', `### Description: ${choice.description}`);
	}

	const userArgsBlock = buildJSONOrTextCodeBlock(choice.userArgSchemaInstance);
	if (userArgsBlock) {
		lines.push('', '### Tool options', '', userArgsBlock);
	}

	return lines.join('\n');
}

// Human-oriented "primary" view for a tool call.
function buildCallPrimaryContent(call: UIToolCall): string {
	const lines: string[] = [`### Tool: ${getToolCallDisplayName(call)}`, `### Status: \`${call.status}\``];

	const errorBlock = buildJSONOrTextCodeBlock(call.errorMessage);
	if (errorBlock) {
		lines.push('', '### Error details', '', errorBlock);
	}

	lines.push('');

	const argumentsBlock = buildJSONOrTextCodeBlock(call.arguments);
	if (argumentsBlock) {
		lines.push('### Arguments', '', argumentsBlock);
	} else {
		lines.push('### Arguments: no arguments provided for this call');
	}

	if (call.webSearchToolCallItems && (call.webSearchToolCallItems as any[]).length > 0) {
		lines.push('', '### Web-search call items', '', buildJSONCodeBlock(call.webSearchToolCallItems));
	}

	return lines.join('\n');
}

// Human-oriented "primary" view for a tool output.
function buildOutputPrimaryContent(output: UIToolOutput): string {
	const lines: string[] = [];

	const titleText = output.summary && output.summary.length > 0 ? output.summary : formatToolOutputSummary(output.name);

	lines.push(`### Summary: ${titleText}`, `### Tool: ${getPrettyToolName(output.name)}`);

	if (output.isSkipped) {
		lines.push('### Status: `skipped`');
	} else if (typeof output.isError === 'boolean') {
		lines.push(`### Status: \`${output.isError ? 'error' : 'ok'}\``);
	}
	const argumentsBlock = buildJSONOrTextCodeBlock(output.arguments);
	if (argumentsBlock) {
		lines.push('', '### Arguments', '', argumentsBlock);
	}
	const errorBlock = buildJSONOrTextCodeBlock(output.errorMessage);
	if (errorBlock) {
		lines.push('', '### Error details', '', errorBlock);
	}
	lines.push('');

	let resultBlock: string | null = null;

	if (output.toolOutputs && output.toolOutputs.length > 0) {
		resultBlock = buildToolOutputItemsMarkdown(output.toolOutputs);
	} else if (output.webSearchToolOutputItems && output.webSearchToolOutputItems.length > 0) {
		resultBlock = buildJSONCodeBlock(normalizeStructuredJSONStringDeep(output.webSearchToolOutputItems));
	}

	if (!resultBlock) {
		lines.push('### Tool result', '', '_Tool returned no output._');
	} else {
		lines.push('### Tool result', '', resultBlock);
	}

	return lines.join('\n');
}

function ToolDetailsModalContent({ state }: ToolDetailsModalProps) {
	const { requestClose } = useModalDialogController();
	if (!state) {
		return null;
	}
	const { title, payload } = buildPayload(state);

	// Raw payload, preferring JSON display with a safe text fallback.
	const rawPayloadMarkdown = `### Additional details\n\n${buildJSONOrTextCodeBlock(payload) ?? '_No additional details available._'}`;

	const outputMedia =
		state.kind === 'output'
			? mediaItemsFromToolOutputItems(state.output.toolOutputs, `tool-output:${state.output.id}`)
			: [];

	// Human-oriented primary content (semantics-first).
	let primaryContent = '';
	let baseMessageId = 'tool-details';

	switch (state.kind) {
		case 'selection': {
			primaryContent = [
				`### Tool: ${state.selection.target.name}`,
				state.message ? `Resolution error: ${state.message}` : '',
			]
				.filter(Boolean)
				.join('\n\n');
			baseMessageId = `tool-selection:${state.selection.choiceID}`;
			break;
		}
		case 'choice': {
			primaryContent = buildChoicePrimaryContent(state.choice);
			const choiceId = state.choice.choiceID;
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

	return (
		<ManagementModalPanel width="wide">
			<ModalHeader
				title={
					<span className="flex items-center gap-2">
						<FiTool size={16} />
						<span>{title}</span>
					</span>
				}
				onClose={() => {
					requestClose();
				}}
			/>
			<ManagementModalBody>
				{/* Primary, human-friendly view (semantics first) */}
				{primaryContent.trim().length > 0 && (
					<div className="mb-4">
						<MessageContentCard
							messageID={`${baseMessageId}:primary`}
							content={primaryContent}
							isBusy={false}
							align="items-start text-left"
							renderAsMarkdown={true}
							defaultCodeBlockExpanded={false}
						/>
					</div>
				)}

				{outputMedia.length > 0 ? (
					<div className="border-base-300 bg-base-100 mb-4 rounded-xl border p-3">
						<div className="mb-2 text-sm font-semibold">Generated images and files</div>
						<MessageMediaContent items={outputMedia} />
					</div>
				) : null}

				{/* Raw payload for full inspection */}
				<div>
					<MessageContentCard
						messageID={`${baseMessageId}:raw`}
						content={rawPayloadMarkdown}
						isBusy={false}
						align="items-start text-left"
						renderAsMarkdown={true}
						defaultCodeBlockExpanded={false}
					/>
				</div>
			</ManagementModalBody>
			<ModalActions>
				<button
					type="button"
					className="btn bg-base-300 rounded-xl"
					onClick={() => {
						requestClose();
					}}
				>
					Close
				</button>
			</ModalActions>
		</ManagementModalPanel>
	);
}

export function ToolDetailsModal(props: ToolDetailsModalProps) {
	if (!props.state) {
		return null;
	}

	return (
		<ManagementModalDialog isOpen onClose={props.onClose}>
			<ToolDetailsModalContent {...props} />
		</ManagementModalDialog>
	);
}
