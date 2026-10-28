import { memo, useCallback, useMemo, useState, useSyncExternalStore } from 'react';
import { FiArrowDown, FiArrowUp, FiCode, FiEdit2 } from 'react-icons/fi';

import type { InferenceUsage, ReasoningContent } from '@/spec/inference';

import { CopyButton } from '@/components/copy_button';
import { HoverTip } from '@/components/hover_tip';
import { stripCustomMDFences } from '@/components/markdown/custom_md_utils';

import { getDebugDetailsMarkdown } from '@/chats/conversation/completion_helper';
import { MessageDetailsModal } from '@/chats/messages/message_details_modal';
import { MessageStreamingStatus } from '@/chats/messages/message_streaming_status';

export interface AssistantResponseTiming {
	startedAt?: string;
	finishedAt?: string;
}

interface MessageFooterAreaProps {
	messageID: string;
	isUser: boolean;
	cardCopyContent: string;
	onEdit: () => void;
	reasoningContents?: ReasoningContent[];
	isBusy: boolean;
	bodyPresent: boolean;
	streamText?: string;
	streamThinking?: string;
	disableMarkdown: boolean;
	onDisableMarkdownChange: (checked: boolean) => void;
	usage?: InferenceUsage;
	debugDetails?: unknown;
	debugDetailsError?: string;
	errorDetails?: unknown;
	responseTiming?: AssistantResponseTiming;
}

function hasReasoningContent(reasoningContents?: ReasoningContent[]): boolean {
	return (
		reasoningContents?.some(rc => {
			const hasSummary = (rc?.summary ?? []).some(s => /\S/.test(s ?? ''));
			const hasThinking = (rc?.thinking ?? []).some(s => /\S/.test(s ?? ''));
			return hasSummary || hasThinking;
		}) ?? false
	);
}

function timestampMilliseconds(value: string | undefined): number | undefined {
	if (!value) {
		return undefined;
	}

	const parsed = Date.parse(value);
	return Number.isFinite(parsed) ? parsed : undefined;
}

function formatElapsedDuration(milliseconds: number): string {
	const totalSeconds = Math.max(0, Math.floor(milliseconds / 1_000));

	if (totalSeconds === 0) {
		return '<1s';
	}

	if (totalSeconds < 60) {
		return `${totalSeconds}s`;
	}

	const minutes = Math.floor(totalSeconds / 60);
	const seconds = totalSeconds % 60;
	return `${minutes}m ${seconds.toString().padStart(2, '0')}s`;
}

const getInitialElapsedSeconds = () => 0;

function useElapsedSeconds(startedAt: number | undefined, isRunning: boolean): number {
	const subscribe = useCallback(
		(onStoreChange: () => void) => {
			if (!isRunning || startedAt === undefined) {
				return () => {};
			}

			let timer: number | undefined;

			const scheduleNextSecond = () => {
				const elapsed = Math.max(0, Date.now() - startedAt);
				const remainingToNextSecond = 1_000 - (elapsed % 1_000);

				timer = window.setTimeout(
					() => {
						onStoreChange();
						scheduleNextSecond();
					},
					Math.max(25, remainingToNextSecond)
				);
			};

			scheduleNextSecond();

			return () => {
				if (timer !== undefined) {
					window.clearTimeout(timer);
				}
			};
		},
		[isRunning, startedAt]
	);

	const getSnapshot = useCallback(() => {
		if (!isRunning || startedAt === undefined) {
			return 0;
		}

		return Math.floor(Math.max(0, Date.now() - startedAt) / 1_000);
	}, [isRunning, startedAt]);

	return useSyncExternalStore(subscribe, getSnapshot, getInitialElapsedSeconds);
}

function AssistantResponseTimingLabel({ timing, isBusy }: { timing?: AssistantResponseTiming; isBusy: boolean }) {
	const startedAt = timestampMilliseconds(timing?.startedAt);
	const finishedAt = timestampMilliseconds(timing?.finishedAt);
	const elapsedSeconds = useElapsedSeconds(startedAt, isBusy && finishedAt === undefined);

	if (startedAt === undefined) {
		return null;
	}

	const elapsed = Math.max(0, finishedAt === undefined ? elapsedSeconds * 1_000 : finishedAt - startedAt);
	const formatted = formatElapsedDuration(elapsed);

	return (
		<HoverTip content={`Assistant response time: ${formatted}`} placement="top">
			<output
				className="text-base-content/60 shrink-0 text-xs tabular-nums"
				aria-label={`Assistant response time ${formatted}`}
			>
				{formatted}
			</output>
		</HoverTip>
	);
}

export const MessageFooterArea = memo(function MessageFooterArea({
	messageID,
	isUser,
	cardCopyContent,
	onEdit,
	reasoningContents,
	isBusy,
	bodyPresent,
	streamText,
	streamThinking,
	disableMarkdown,
	onDisableMarkdownChange,
	usage,
	debugDetails,
	debugDetailsError,
	errorDetails,
	responseTiming,
}: MessageFooterAreaProps) {
	const [isDetailsOpen, setIsDetailsOpen] = useState(false);

	const copyContent = useMemo(() => (isBusy ? '' : stripCustomMDFences(cardCopyContent)), [cardCopyContent, isBusy]);
	const hasDebugDetailsError = typeof debugDetailsError === 'string' && /\S/.test(debugDetailsError);
	const hasDebugDetails = debugDetails !== undefined || hasDebugDetailsError || errorDetails !== undefined;
	const hasInlineHiddenReasoning = !bodyPresent && hasReasoningContent(reasoningContents);
	const hasDetails = hasDebugDetails || hasInlineHiddenReasoning;
	const hasContent = !!cardCopyContent;
	const hasUsage = Boolean(
		usage &&
		(usage.inputTokensTotal > 0 ||
			usage.inputTokensCached > 0 ||
			usage.inputTokensUncached > 0 ||
			usage.outputTokens > 0 ||
			usage.reasoningTokens > 0)
	);

	const renderedMessageDetails = useMemo(() => {
		return isDetailsOpen ? (getDebugDetailsMarkdown(debugDetails, errorDetails, debugDetailsError) ?? '') : '';
	}, [debugDetails, debugDetailsError, errorDetails, isDetailsOpen]);

	const toggleDetailsModal = useCallback(() => {
		if (!hasDetails || isBusy) {
			return;
		}
		setIsDetailsOpen(prev => !prev);
	}, [hasDetails, isBusy]);

	const usageTooltip =
		hasUsage && usage
			? (() => {
					const parts: string[] = [];

					if (usage.inputTokensTotal > 0) {
						parts.push(`Total input tokens: ${usage.inputTokensTotal}`);
					}
					if (usage.inputTokensCached > 0) {
						parts.push(`Input cached tokens: ${usage.inputTokensCached}`);
						if (usage.inputTokensUncached > 0) {
							parts.push(`Input uncached tokens: ${usage.inputTokensUncached}`);
						}
					}
					if (usage.outputTokens > 0) {
						parts.push(`Total output tokens: ${usage.outputTokens}`);
					}
					if (usage.reasoningTokens > 0) {
						parts.push(`Reasoning tokens: ${usage.reasoningTokens}`);
					}

					return parts.join('\n');
				})()
			: '';

	return (
		<div className={bodyPresent ? 'grow' : ''}>
			<div
				className={`flex items-center gap-3 overflow-x-auto ${bodyPresent ? 'justify-between' : isUser ? 'justify-start' : 'justify-end'}`}
			>
				<div className="flex min-w-0 items-center gap-3">
					{isBusy ? (
						<MessageStreamingStatus streamText={streamText} streamThinking={streamThinking} />
					) : hasUsage && usage ? (
						<HoverTip content={usageTooltip} placement="top" wrapperElement="div">
							<div className="flex items-center bg-transparent p-0 text-xs">
								<div className="flex items-center">
									<FiArrowUp size={14} />
									<span className="shrink-0">{usage.inputTokensTotal}&nbsp;&nbsp;-&nbsp;&nbsp;</span>
									<FiArrowDown size={14} /> <span>{usage.outputTokens}</span>
								</div>
							</div>
						</HoverTip>
					) : null}

					{!isUser ? <AssistantResponseTimingLabel timing={responseTiming} isBusy={isBusy} /> : null}
				</div>

				<div className={`flex items-center justify-end space-x-6 ${!isBusy && !hasUsage ? 'w-full' : ''}`}>
					{hasContent && !isBusy && (
						<label className="ml-1 flex h-full items-center space-x-2 truncate p-1" title="Disable Markdown">
							<input
								type="checkbox"
								checked={disableMarkdown}
								onChange={e => {
									onDisableMarkdownChange(e.target.checked);
								}}
								className="checkbox checkbox-xs ml-1 rounded-full"
								spellCheck="false"
							/>
							<span className="text-base-content text-xs text-nowrap">Disable Markdown</span>
						</label>
					)}

					{hasDetails && !isBusy && (
						<HoverTip content="Show message details" placement="top">
							<button
								type="button"
								className="btn btn-sm flex items-center border-none bg-transparent! p-0 shadow-none"
								onClick={toggleDetailsModal}
								aria-label="Show message details"
							>
								<FiCode size={16} />
							</button>
						</HoverTip>
					)}
					{isUser && !isBusy && (
						<HoverTip content="Edit and resend from this message" placement="top">
							<button
								type="button"
								className="btn btn-sm flex items-center border-none bg-transparent! p-0 shadow-none"
								onClick={onEdit}
								aria-label="Edit and resend from this message"
							>
								<FiEdit2 size={16} />
							</button>
						</HoverTip>
					)}

					{cardCopyContent !== '' && !isBusy && (
						<HoverTip content="Copy message text" placement="top">
							<CopyButton
								value={copyContent}
								className="btn btn-sm flex items-center border-none bg-transparent! p-0 shadow-none"
								size={16}
							/>
						</HoverTip>
					)}
				</div>
			</div>

			{/* Details modal (works for both user & assistant messages) */}
			{isDetailsOpen && hasDetails ? (
				<MessageDetailsModal
					isOpen={true}
					onClose={() => {
						setIsDetailsOpen(false);
					}}
					messageID={messageID}
					title={isUser ? 'User message details' : 'Assistant message details'}
					content={renderedMessageDetails}
					isBusy={isBusy}
					reasoningContents={reasoningContents}
					showReasoningAtTop={!bodyPresent}
				/>
			) : null}
		</div>
	);
});
