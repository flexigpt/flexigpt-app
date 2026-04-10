import { FiAlertTriangle, FiCode, FiPlay, FiTerminal, FiTool, FiX } from 'react-icons/fi';

import { type UIToolCall, type UIToolOutput } from '@/spec/inference';
import { ToolStoreChoiceType } from '@/spec/tool';

import { isSkillsToolName } from '@/skills/lib/skill_identity_utils';
import { getPrettyToolName } from '@/tools/lib/tool_identity_utils';

type OrderedToolChipItem =
	| { key: string; kind: 'call'; call: UIToolCall }
	| { key: string; kind: 'output'; output: UIToolOutput };

const noopOpenCallDetails = (_call: UIToolCall) => {};

function buildOrderedToolChipItems(toolCalls: UIToolCall[], toolOutputs: UIToolOutput[]): OrderedToolChipItem[] {
	const items: OrderedToolChipItem[] = [];

	// Match outputs to calls by callID.
	const outputByCallID = new Map<string, UIToolOutput>();
	for (const output of toolOutputs) {
		if (!outputByCallID.has(output.callID)) {
			outputByCallID.set(output.callID, output);
		}
	}

	const usedOutputIds = new Set<string>();
	const seenNonDiscardedCallIDs = new Set<string>();

	for (const toolCall of toolCalls) {
		if (toolCall.status === 'discarded') continue;

		seenNonDiscardedCallIDs.add(toolCall.callID);

		if (toolCall.status === 'succeeded') {
			const output = outputByCallID.get(toolCall.callID);

			if (output) {
				items.push({
					key: toolCall.id,
					kind: 'output',
					output,
				});
				usedOutputIds.add(output.id);
			}

			continue;
		}

		items.push({
			key: toolCall.id,
			kind: 'call',
			call: toolCall,
		});
	}

	// Append outputs that do not correspond to any still-present call.
	for (const output of toolOutputs) {
		if (usedOutputIds.has(output.id)) continue;
		if (seenNonDiscardedCallIDs.has(output.callID)) continue;

		items.push({
			key: output.id,
			kind: 'output',
			output,
		});
	}

	return items;
}

interface ToolChipsComposerRowProps {
	toolCalls: UIToolCall[];
	toolOutputs: UIToolOutput[];
	isBusy: boolean;
	onRunToolCall: (id: string) => void | Promise<void>;
	onDiscardToolCall: (id: string) => void;
	onOpenOutput: (output: UIToolOutput) => void;
	onRemoveOutput: (id: string) => void;
	onRetryErroredOutput: (output: UIToolOutput) => void;
	onOpenCallDetails?: (call: UIToolCall) => void;
}

/**
 * Row of interactive tool-call and tool-output chips used in the composer.
 *
 * Order (left → right):
 *   - Pending / running / failed tool calls
 *   - Succeeded calls replaced in-place by their output
 *   - Orphan outputs appended at the end
 */
export function ToolChipsComposerRow({
	toolCalls,
	toolOutputs,
	isBusy,
	onRunToolCall,
	onDiscardToolCall,
	onOpenOutput,
	onRemoveOutput,
	onRetryErroredOutput,
	onOpenCallDetails,
}: ToolChipsComposerRowProps) {
	const openCallDetails = onOpenCallDetails ?? noopOpenCallDetails;
	const orderedItems = buildOrderedToolChipItems(toolCalls, toolOutputs);

	if (orderedItems.length === 0) return null;

	return (
		<div className="flex shrink-0 items-center gap-1">
			{orderedItems.map(item =>
				item.kind === 'call' ? (
					<ToolCallComposerChipView
						key={item.key}
						toolCall={item.call}
						isBusy={isBusy}
						onRun={() => {
							void onRunToolCall(item.call.id);
						}}
						onDiscard={() => {
							onDiscardToolCall(item.call.id);
						}}
						onDetails={() => {
							openCallDetails(item.call);
						}}
					/>
				) : (
					<ToolOutputComposerChipView
						key={item.key}
						output={item.output}
						onOpen={() => {
							onOpenOutput(item.output);
						}}
						onRemove={() => {
							onRemoveOutput(item.output.id);
						}}
						onRetry={() => {
							onRetryErroredOutput(item.output);
						}}
					/>
				)
			)}
		</div>
	);
}
interface ToolCallComposerChipViewProps {
	toolCall: UIToolCall;
	isBusy: boolean;
	onRun: () => void;
	onDiscard: () => void;
	onDetails: () => void;
}

/**
 * Interactive chip for a single pending / running / failed tool call.
 * - "Run" button executes the tool once.
 * - "×" discards the suggestion from the composer only.
 */
function ToolCallComposerChipView({ toolCall, isBusy, onRun, onDiscard, onDetails }: ToolCallComposerChipViewProps) {
	const label = getPrettyToolName(toolCall.name);
	const truncatedLabel = label.length > 64 ? `${label.slice(0, 61)}…` : label;

	const isRunning = toolCall.status === 'running';
	const isPending = toolCall.status === 'pending';
	const isFailed = toolCall.status === 'failed';

	const isRunnableType = toolCall.type === ToolStoreChoiceType.Function || toolCall.type === ToolStoreChoiceType.Custom;

	const canRun = isRunnableType && (isPending || isFailed) && !isBusy;

	const errorClasses = isFailed ? 'border-error/70 bg-error/5 text-error' : '';

	const titleLines: string[] = [`Suggested: ${label}`];
	if (toolCall.errorMessage && isFailed) {
		titleLines.push(`Error: ${toolCall.errorMessage}`);
	}
	if (toolCall.toolStoreChoice?.autoExecute) {
		titleLines.push('Auto-execute: enabled');
	}
	const title = titleLines.join('\n');

	return (
		<div
			className={`bg-base-200 text-base-content hover:bg-base-300/80 flex min-w-48 shrink-0 items-center gap-2 rounded-2xl border border-transparent px-2 py-0 ${errorClasses}`}
			title={title}
			data-attachment-chip="tool-call"
		>
			<FiTerminal size={14} className={isFailed ? 'text-error' : ''} />
			{toolCall.toolStoreChoice?.autoExecute ? <span className="badge badge-primary badge-xs">Auto</span> : null}
			<span className="max-w-64 truncate">{truncatedLabel}</span>

			<div className="ml-auto flex items-center gap-2 p-0">
				{isRunnableType &&
					(isRunning ? (
						<span className="loading loading-spinner loading-xs" aria-label="Running tool call" />
					) : (
						<button
							type="button"
							className={`btn btn-ghost btn-xs gap-0 p-0 shadow-none ${!canRun ? 'btn-disabled' : ''}`}
							onClick={onRun}
							disabled={!canRun}
							title={isFailed ? 'Retry this tool call' : 'Run this tool call'}
							aria-label={isFailed ? 'Retry tool call' : 'Run tool call'}
						>
							<FiPlay size={12} />
							<span className="ml-1 text-xs">Run</span>
						</button>
					))}

				<button
					type="button"
					className="btn btn-ghost btn-xs text-base-content/60 gap-0 px-1 py-0 shadow-none"
					onClick={onDetails}
					title="Show call details"
					aria-label="Show call details"
				>
					<FiCode size={12} />
				</button>

				{isFailed && (
					<FiAlertTriangle
						size={12}
						className="text-error"
						title={toolCall.errorMessage}
						aria-label="Tool call failed"
					/>
				)}

				<button
					type="button"
					className="btn btn-ghost btn-xs text-error p-0 shadow-none"
					onClick={onDiscard}
					title="Discard this tool call"
					aria-label="Discard tool call"
				>
					<FiX size={12} />
				</button>
			</div>
		</div>
	);
}

interface ToolOutputComposerChipViewProps {
	output: UIToolOutput;
	onOpen: () => void;
	onRemove: () => void;
	onRetry: () => void;
}

/**
 * Interactive chip for a single tool output in the composer.
 * - Click opens the full JSON/text in a modal.
 * - "×" discards the output from the next turn.
 * - If `isError` is true and we have enough info, show a "Retry" button.
 */
function ToolOutputComposerChipView({ output, onOpen, onRemove, onRetry }: ToolOutputComposerChipViewProps) {
	const label = getPrettyToolName(output.name);
	const truncatedLabel = label.length > 64 ? `${label.slice(0, 61)}…` : label;

	const isError = !!output.isError;

	const hasResolvableStoredTool =
		!!output.toolStoreChoice?.bundleID && !!output.toolStoreChoice?.toolSlug && !!output.toolStoreChoice?.toolVersion;
	const canRetry = isError && !!output.arguments && (isSkillsToolName(output.name) || hasResolvableStoredTool);

	const titleLines = [
		isError ? `Errored result from: ${label}` : label,
		`Tool: ${output.name}`,
		`Call ID: ${output.callID}`,
	];
	if (isError && output.errorMessage) {
		titleLines.push(`Error: ${output.errorMessage}`);
	}
	if (isError) {
		titleLines.push('This is a tool error result. It can still be sent to the model as tool output.');
	}
	const title = titleLines.join('\n');

	return (
		<div
			className={`flex min-w-48 shrink-0 cursor-pointer items-center gap-2 rounded-2xl px-2 py-0 transition-colors ${isError ? 'border-error/70 bg-error/5 text-error border' : 'bg-base-200 text-base-content hover:bg-base-300/80'}`}
			title={title}
			role="button"
			tabIndex={0}
			onClick={onOpen}
			onKeyDown={e => {
				if (e.key === 'Enter' || e.key === ' ') {
					e.preventDefault();
					onOpen();
				}
			}}
			data-attachment-chip="tool-output"
		>
			<FiTool size={14} className={isError ? 'text-error' : ''} />
			<span className={`text-[10px] uppercase ${isError ? 'text-error' : 'text-base-content/60'}`}>
				{isError ? 'Tool error' : 'Result'}
			</span>
			<span className="max-w-64 truncate">{truncatedLabel}</span>

			<div className="ml-auto flex items-center gap-1">
				<span className="text-base-content/60">
					<FiCode size={12} />
				</span>

				{canRetry && (
					<button
						type="button"
						className="btn btn-ghost btn-xs gap-0 px-1 py-0 shadow-none"
						onClick={e => {
							e.stopPropagation();
							onRetry();
						}}
						title="Retry this tool"
						aria-label="Retry this tool"
					>
						<FiPlay size={12} />
						<span className="ml-1 text-xs">Retry</span>
					</button>
				)}

				<button
					type="button"
					className="btn btn-ghost btn-xs text-error px-1 py-0 shadow-none"
					onClick={e => {
						e.stopPropagation();
						onRemove();
					}}
					title="Discard this tool output"
					aria-label="Discard tool output"
				>
					<FiX size={12} />
				</button>
			</div>
		</div>
	);
}
