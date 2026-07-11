import { memo, useMemo, useState, useSyncExternalStore } from 'react';

import type { ReasoningContent } from '@/spec/inference';

import { ThinkingFence } from '@/components/markdown/thinking_fence';

import type { MessageStreamSource } from '@/chats/messages/message_content_card';

const EMPTY_STREAM_SUBSCRIBE = () => () => {};
const EMPTY_STREAM_SNAPSHOT = () => 0;

function joinReasoningParts(reasoning: ReasoningContent[] | undefined, key: 'summary' | 'thinking'): string {
	const items = reasoning ?? [];
	const parts: string[] = [];
	for (const rc of items) {
		const arr = rc?.[key];
		if (Array.isArray(arr) && arr.length > 0) {
			parts.push(arr.join('\n'));
		}
	}
	return parts.join('\n\n').trim();
}

export const MessageThinkingSection = memo(function MessageThinkingSection(props: {
	/** Busy == request in flight */
	isBusy: boolean;
	/** Streaming thinking channel */
	streamedThinking: string;
	/** Final reasoning (if provider sends it at end) */
	reasoningContents?: ReasoningContent[];
	streamSource?: MessageStreamSource;
}) {
	const { isBusy, streamedThinking, reasoningContents, streamSource } = props;

	useSyncExternalStore(
		streamSource?.subscribe ?? EMPTY_STREAM_SUBSCRIBE,
		streamSource?.getVersionSnapshot ?? EMPTY_STREAM_SNAPSHOT,
		EMPTY_STREAM_SNAPSHOT
	);

	const finalSummary = useMemo(() => joinReasoningParts(reasoningContents, 'summary'), [reasoningContents]);
	const finalThinking = useMemo(() => joinReasoningParts(reasoningContents, 'thinking'), [reasoningContents]);

	const activeStreamedThinking = isBusy && streamSource ? streamSource.getThinking() : streamedThinking;

	// Prefer streamed thinking while busy; otherwise show final thinking.
	const thinkingText = isBusy ? activeStreamedThinking : finalThinking.trimEnd();

	// Thinking is optional: only show when we actually have something to display.
	const hasSummary = /\S/.test(finalSummary);
	const hasThinking = /\S/.test(thinkingText);
	const shouldShow = hasSummary || hasThinking;

	const [manualOpen, setManualOpen] = useState<boolean | null>(null);
	const open = manualOpen ?? isBusy;

	const summaryNode = (
		<div className="flex items-center gap-2">
			<span className="text-xs">Thinking Content</span>
		</div>
	);

	if (!shouldShow) {
		return null;
	}

	return (
		<div className="m-0 p-0">
			{finalSummary ? (
				<ThinkingFence
					detailsSummary={summaryNode}
					text={finalSummary}
					open={open}
					onOpenChange={value => {
						setManualOpen(value);
					}}
					streaming={isBusy}
					maxRows={10}
					autoScroll={isBusy}
					defaultOpen={isBusy}
				/>
			) : null}

			{hasThinking ? (
				<ThinkingFence
					detailsSummary={summaryNode}
					text={thinkingText}
					open={open}
					onOpenChange={value => {
						setManualOpen(value);
					}}
					streaming={isBusy}
					maxRows={10}
					autoScroll={isBusy}
					defaultOpen={isBusy}
				/>
			) : null}
		</div>
	);
});
