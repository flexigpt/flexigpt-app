import { useEffect, useMemo, useRef } from 'react';

import { FiAlertTriangle, FiCode, FiPlay, FiTerminal, FiTool, FiX } from 'react-icons/fi';

import { type UIToolCall, type UIToolOutput } from '@/spec/inference';
import { ToolStoreChoiceType } from '@/spec/tool';

import { getPrettyToolName } from '@/tools/lib/tool_identity_utils';

type OrderedToolChipItem =
	| { key: string; kind: 'call'; call: UIToolCall }
	| { key: string; kind: 'output'; output: UIToolOutput };

function buildCurrentChipIds(toolCalls: UIToolCall[], toolOutputs: UIToolOutput[]): string[] {
	const ids: string[] = [];
	const seen = new Set<string>();

	for (const toolCall of toolCalls) {
		if (seen.has(toolCall.id)) continue;
		seen.add(toolCall.id);
		ids.push(toolCall.id);
	}

	for (const output of toolOutputs) {
		if (seen.has(output.id)) continue;
		seen.add(output.id);
		ids.push(output.id);
	}

	return ids;
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
 *   - Tool output chips
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
	const visibleCalls = toolCalls.filter(toolCall => toolCall.status !== 'discarded' && toolCall.status !== 'succeeded');
	const hasAny = visibleCalls.length > 0 || toolOutputs.length > 0;
	const openCallDetails = onOpenCallDetails ?? (() => {});
	// Keep a stable visual ordering so a succeeded tool call can "turn into"
	// its output in-place instead of disappearing from the left and reappearing
	// at the far right. This dramatically reduces horizontal layout churn.
	const orderedChipIdsRef = useRef<string[]>([]);

	const displayOrder = useMemo(() => {
		const currentIds = buildCurrentChipIds(visibleCalls, toolOutputs);
		const currentIdSet = new Set(currentIds);

		// eslint-disable-next-line react-hooks/refs
		const nextOrder = orderedChipIdsRef.current.filter(id => currentIdSet.has(id));
		const seen = new Set(nextOrder);

		for (const id of currentIds) {
			if (seen.has(id)) continue;
			seen.add(id);
			nextOrder.push(id);
		}

		return nextOrder;
	}, [toolOutputs, visibleCalls]);

	useEffect(() => {
		orderedChipIdsRef.current = displayOrder;
	}, [displayOrder]);

	const callById = useMemo(
		() => new Map(visibleCalls.map(toolCall => [toolCall.id, toolCall] as const)),
		[visibleCalls]
	);
	const outputById = useMemo(() => new Map(toolOutputs.map(output => [output.id, output] as const)), [toolOutputs]);

	const orderedItems = useMemo<OrderedToolChipItem[]>(() => {
		const items: OrderedToolChipItem[] = [];

		for (const id of displayOrder) {
			const toolCall = callById.get(id);
			if (toolCall) {
				items.push({ key: id, kind: 'call', call: toolCall });
				continue;
			}

			const output = outputById.get(id);
			if (output) {
				items.push({ key: id, kind: 'output', output });
			}
		}

		return items;
	}, [callById, displayOrder, outputById]);

	if (!hasAny) return null;

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

function isSkillsToolName(name: string | undefined): boolean {
	const n = (name ?? '').trim();
	return n.startsWith('skills.');
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
