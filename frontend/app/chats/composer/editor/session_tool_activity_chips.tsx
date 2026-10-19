import { FiPlay, FiRefreshCw, FiTerminal, FiTool, FiX } from 'react-icons/fi';

import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import { UIToolCallStatus } from '@/spec/inference';

import { getPrettyToolName } from '@/apis/tool_management';

interface SessionToolActivityChipsProps {
	runtimeCalls: UIToolCall[];
	runtimeOutputs: UIToolOutput[];
	draftOutputs: UIToolOutput[];
	onRunCall: (call: UIToolCall) => void;
	onRetryCall: (call: UIToolCall) => void;
	onSkipCall: (call: UIToolCall) => void;
	onInspectCall: (call: UIToolCall) => void;
	onInspectOutput: (output: UIToolOutput) => void;
	onRemoveDraftOutput: (output: UIToolOutput) => void;
}

function callMatchesOutput(call: UIToolCall, output: UIToolOutput): boolean {
	return output.callID === call.id || output.callID === call.callID;
}

function toolCallLabel(call: UIToolCall): string {
	return call.mcpToolSelection?.toolName || getPrettyToolName(call.name);
}

function RuntimeToolCallChip({
	call,
	onRun,
	onRetry,
	onSkip,
	onInspect,
}: {
	call: UIToolCall;
	onRun: () => void;
	onRetry: () => void;
	onSkip: () => void;
	onInspect: () => void;
}) {
	const label = toolCallLabel(call);
	const isPending = call.status === UIToolCallStatus.Pending;
	const isRunning = call.status === UIToolCallStatus.Running;
	const isFailed = call.status === UIToolCallStatus.Failed;

	return (
		<div
			className="bg-base-200 hover:bg-base-300/80 text-base-content flex h-7 shrink-0 items-center gap-1 rounded-2xl px-2 text-xs"
			data-editor-session-tool-call
		>
			<button
				type="button"
				className="flex min-w-0 items-center gap-2 rounded-lg text-left outline-offset-2"
				onClick={onInspect}
				aria-label={`Inspect Tool call: ${label}`}
				title={call.errorMessage || `Tool call: ${label}`}
			>
				{isRunning ? (
					<span className="loading loading-spinner loading-xs text-info shrink-0" aria-label="Running Tool call" />
				) : (
					<FiTerminal size={14} className={isFailed ? 'text-error shrink-0' : 'shrink-0'} />
				)}
				<span className="max-w-64 truncate">{label}</span>
			</button>
			<span className={`px-1 text-[11px] ${isFailed ? 'text-error' : 'text-base-content/60'}`}>{call.status}</span>
			{isPending ? (
				<button
					type="button"
					className="btn btn-ghost btn-xs text-primary gap-1 rounded-lg px-1 shadow-none"
					onClick={onRun}
					aria-label={`Run Tool call: ${label}`}
				>
					<FiPlay size={13} />
					Run
				</button>
			) : null}
			{isFailed ? (
				<button
					type="button"
					className="btn btn-ghost btn-xs text-warning gap-1 rounded-lg px-1 shadow-none"
					onClick={onRetry}
					aria-label={`Retry Tool call: ${label}`}
				>
					<FiRefreshCw size={13} />
					Retry
				</button>
			) : null}
			{isPending || isFailed ? (
				<button
					type="button"
					className="btn btn-ghost btn-xs text-error gap-1 rounded-lg px-1 shadow-none"
					onClick={onSkip}
					title={isFailed ? 'Submit failure as an error result' : 'Skip Tool call'}
					aria-label={`${isFailed ? 'Submit failure' : 'Skip Tool call'}: ${label}`}
				>
					<FiX size={13} />
					{isFailed ? 'Use error' : 'Skip'}
				</button>
			) : null}
		</div>
	);
}

function ToolOutputChip({
	output,
	runtimeOwned,
	onInspect,
	onRemove,
}: {
	output: UIToolOutput;
	runtimeOwned: boolean;
	onInspect: () => void;
	onRemove?: () => void;
}) {
	const label = output.summary || `Result: ${getPrettyToolName(output.name)}`;
	const status = output.isSkipped ? 'Skipped' : output.isError ? 'Error result' : 'Result';

	return (
		<div
			className="bg-base-200 hover:bg-base-300/80 text-base-content flex h-7 shrink-0 items-center gap-1 rounded-2xl px-2 text-xs"
			data-editor-tool-output={runtimeOwned ? 'session' : 'draft'}
		>
			<button
				type="button"
				className="flex min-w-0 items-center gap-2 rounded-lg text-left outline-offset-2"
				onClick={onInspect}
				aria-label={`Inspect Tool result: ${label}`}
				title={output.errorMessage || label}
			>
				<FiTool
					size={14}
					className={`shrink-0 ${
						output.isSkipped ? 'text-warning' : output.isError ? 'text-error' : 'text-base-content/70'
					}`}
				/>
				<span className="max-w-64 truncate">{label}</span>
			</button>
			<span className="text-base-content/60 px-1 text-[11px]">{status}</span>
			{onRemove ? (
				<button
					type="button"
					className="btn btn-ghost btn-xs text-error shrink-0 rounded-lg px-1 shadow-none"
					onClick={onRemove}
					title="Remove carried Tool result"
					aria-label={`Remove carried Tool result: ${label}`}
				>
					<FiX size={14} />
				</button>
			) : null}
		</div>
	);
}

/**
 * Active Session Tool controls and Tool-result inspection live in the
 * Composer. Message rows remain historical/read-only inspection surfaces.
 */
export function SessionToolActivityChips({
	runtimeCalls,
	runtimeOutputs,
	draftOutputs,
	onRunCall,
	onRetryCall,
	onSkipCall,
	onInspectCall,
	onInspectOutput,
	onRemoveDraftOutput,
}: SessionToolActivityChipsProps) {
	const visibleRuntimeCalls = runtimeCalls.filter(call => {
		if (call.status !== UIToolCallStatus.Succeeded) {
			return true;
		}

		return !runtimeOutputs.some(output => callMatchesOutput(call, output));
	});
	// A carried result and the same selected runtime result must not appear
	// twice during a snapshot/edit transition.
	const runtimeOutputIDs = new Set(runtimeOutputs.map(output => output.id));
	const visibleDraftOutputs = draftOutputs.filter(output => !runtimeOutputIDs.has(output.id));

	if (visibleRuntimeCalls.length === 0 && runtimeOutputs.length === 0 && visibleDraftOutputs.length === 0) {
		return null;
	}

	return (
		<>
			{visibleRuntimeCalls.map(call => (
				<RuntimeToolCallChip
					key={`session-call:${call.id}`}
					call={call}
					onRun={() => {
						onRunCall(call);
					}}
					onRetry={() => {
						onRetryCall(call);
					}}
					onSkip={() => {
						onSkipCall(call);
					}}
					onInspect={() => {
						onInspectCall(call);
					}}
				/>
			))}

			{runtimeOutputs.map(output => (
				<ToolOutputChip
					key={`session-output:${output.id}`}
					output={output}
					runtimeOwned={true}
					onInspect={() => {
						onInspectOutput(output);
					}}
				/>
			))}

			{visibleDraftOutputs.map(output => (
				<ToolOutputChip
					key={`draft-output:${output.id}`}
					output={output}
					runtimeOwned={false}
					onInspect={() => {
						onInspectOutput(output);
					}}
					onRemove={() => {
						onRemoveDraftOutput(output);
					}}
				/>
			))}
		</>
	);
}
