import { FiAlertTriangle, FiCode, FiPlay, FiTerminal, FiTool, FiX } from 'react-icons/fi';

import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import { UIToolCallStatus } from '@/spec/inference';
import { MCPExecutionMode } from '@/spec/mcp_artifact';

import { isSkillsToolName } from '@/skills/lib/skill_identity_utils';
import { isRunnableComposerToolCall } from '@/tools/lib/tool_call_utils';
import { getPrettyToolName } from '@/tools/lib/tool_identity_utils';

type OrderedToolChipItem =
	{ key: string; kind: 'call'; call: UIToolCall } | { key: string; kind: 'output'; output: UIToolOutput };

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
		if (toolCall.status === UIToolCallStatus.Discarded) {
			continue;
		}
		seenNonDiscardedCallIDs.add(toolCall.callID);
		if (toolCall.status === UIToolCallStatus.Succeeded) {
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
		if (usedOutputIds.has(output.id)) {
			continue;
		}
		if (seenNonDiscardedCallIDs.has(output.callID)) {
			continue;
		}

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
	onRetryErroredOutput,
	onOpenCallDetails,
}: ToolChipsComposerRowProps) {
	const openCallDetails = onOpenCallDetails ?? noopOpenCallDetails;
	const orderedItems = buildOrderedToolChipItems(toolCalls, toolOutputs);

	if (orderedItems.length === 0) {
		return null;
	}

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
	const label = toolCall.mcpToolSelection?.toolName
		? toolCall.mcpToolSelection.toolName
		: getPrettyToolName(toolCall.name);
	const truncatedLabel = label.length > 64 ? `${label.slice(0, 61)}…` : label;

	const isRunning = toolCall.status === UIToolCallStatus.Running;
	const isPending = toolCall.status === UIToolCallStatus.Pending;
	const isFailed = toolCall.status === UIToolCallStatus.Failed;

	const isRunnableType = isRunnableComposerToolCall(toolCall);
	const isAutoExecute =
		Boolean(toolCall.toolStoreChoice?.autoExecute) ||
		toolCall.mcpToolSelection?.executionMode === MCPExecutionMode.Auto;

	const canRun = isRunnableType && (isPending || isFailed) && !isBusy;

	const errorClasses = isFailed ? 'border-error/70 bg-error/5 text-error' : '';

	const titleLines: string[] = [`Suggested: ${label}`];
	if (toolCall.errorMessage && isFailed) {
		titleLines.push(`Error: ${toolCall.errorMessage}`);
	}
	if (isAutoExecute) {
		titleLines.push('Auto-execute: enabled');
	}
	if (toolCall.mcpToolSelection) {
		titleLines.push(`MCP: ${toolCall.mcpToolSelection.server}/${toolCall.mcpToolSelection.toolName}`);
	}
	const title = titleLines.join('\n');

	return (
		<div
			className={`bg-base-200 text-base-content hover:bg-base-300/80 flex min-w-48 shrink-0 items-center gap-2 rounded-2xl border border-transparent px-2 py-0 ${errorClasses}`}
			title={title}
			data-attachment-chip="tool-call"
		>
			<FiTerminal size={14} className={isFailed ? 'text-error' : ''} />
			{isAutoExecute ? <span className="badge badge-primary badge-xs">Auto</span> : null}
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
							<span className="ml-1 text-xs">{isFailed ? 'Retry' : 'Run'}</span>
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

				{!isRunning ? (
					<button
						type="button"
						className="btn btn-ghost btn-xs text-error gap-1 px-1 py-0 shadow-none"
						onClick={onDiscard}
						title="Submit an error result without running this call"
						aria-label="Submit tool error result"
					>
						<FiX size={12} />
						<span className="text-xs">Submit error</span>
					</button>
				) : null}
			</div>
		</div>
	);
}

interface ToolOutputComposerChipViewProps {
	output: UIToolOutput;
	onOpen: () => void;
	onRetry: () => void;
}

/**
 * Interactive chip for a single tool output in the composer.
 * - Click opens the full JSON/text in a modal.
 * - "×" discards the output from the next turn.
 * - If `isError` is true and we have enough info, show a "Retry" button.
 */
function ToolOutputComposerChipView({ output, onOpen, onRetry }: ToolOutputComposerChipViewProps) {
	const label = getPrettyToolName(output.name);
	const truncatedLabel = label.length > 64 ? `${label.slice(0, 61)}…` : label;

	const isError = !!output.isError;

	const hasResolvableStoredTool =
		!!output.toolStoreChoice?.bundleID && !!output.toolStoreChoice?.toolSlug && !!output.toolStoreChoice?.toolVersion;
	const canRetry = isError && (isSkillsToolName(output.name) || hasResolvableStoredTool || !!output.mcpToolSelection);
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
		// Keep each action as a sibling native button. Nesting buttons produces
		// invalid HTML and can cause the server and client DOM trees to differ.
		<div
			className={`flex min-w-48 shrink-0 cursor-pointer items-center gap-2 rounded-2xl px-2 py-0 transition-colors ${isError ? 'border-warning/70 bg-warning/10 text-warning-content border' : 'bg-base-200 text-base-content hover:bg-base-300/80'}`}
			title={title}
			data-attachment-chip="tool-output"
		>
			<button
				type="button"
				className="flex min-w-0 flex-1 cursor-pointer items-center gap-2 text-left"
				onClick={onOpen}
				aria-label={`Open tool output for ${label}`}
			>
				<FiTool size={14} className="shrink-0" />
				<span className={`shrink-0 text-[10px] uppercase ${isError ? 'font-semibold' : 'text-base-content/60'}`}>
					{isError ? 'Error result' : 'Result'}
				</span>
				<span className="max-w-64 truncate">{truncatedLabel}</span>

				<span className="text-base-content/60 ml-auto shrink-0">
					<FiCode size={12} />
				</span>
			</button>

			<div className="flex shrink-0 items-center gap-1">
				{canRetry && (
					<button
						type="button"
						className="btn btn-ghost btn-xs gap-0 px-1 py-0 shadow-none"
						onClick={onRetry}
						title="Retry this tool"
						aria-label="Retry this tool"
					>
						<FiPlay size={12} />
						<span className="ml-1 text-xs">Retry</span>
					</button>
				)}
			</div>
		</div>
	);
}
