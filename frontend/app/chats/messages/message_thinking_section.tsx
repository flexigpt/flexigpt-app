import { useMemo, useState } from 'react';

import type { ReasoningContent } from '@/spec/inference';

import { ThinkingFence } from '@/components/thinking_fence';

function joinReasoningParts(reasoning: ReasoningContent[] | undefined, key: 'summary' | 'thinking'): string {
	const items = reasoning ?? [];
	const parts: string[] = [];
	for (const rc of items) {
		const arr = rc?.[key];
		if (Array.isArray(arr) && arr.length > 0) parts.push(arr.join('\n'));
	}
	return parts.join('\n\n').trim();
}

export function MessageThinkingSection(props: {
	/** Busy == request in flight */
	isBusy: boolean;
	/** Streaming thinking channel */
	streamedThinking: string;
	/** Final reasoning (if provider sends it at end) */
	reasoningContents?: ReasoningContent[];
}) {
	const { isBusy, streamedThinking, reasoningContents } = props;

	const finalSummary = useMemo(() => joinReasoningParts(reasoningContents, 'summary'), [reasoningContents]);
	const finalThinking = useMemo(() => joinReasoningParts(reasoningContents, 'thinking'), [reasoningContents]);

	// Prefer streamed thinking while busy; otherwise show final thinking.
	const thinkingText = (isBusy ? streamedThinking : finalThinking).trimEnd();

	// Thinking is optional: only show when we actually have something to display.
	const hasSummary = finalSummary.trim().length > 0;
	const hasThinking = thinkingText.trim().length > 0;
	const shouldShow = hasSummary || hasThinking;

	// null means "auto" mode and follows isBusy.
	// Once the user toggles, their explicit choice wins from then on.
	const [manualOpen, setManualOpen] = useState<boolean | null>(null);
	const open = manualOpen ?? isBusy;

	const summaryNode = (
		<div className="flex items-center gap-2">
			<span className="text-xs">Thinking Content</span>
		</div>
	);

	if (!shouldShow) return null;

	return (
		<div className="m-0 p-0">
			{finalSummary ? (
				<ThinkingFence
					detailsSummary={summaryNode}
					text={finalSummary}
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
}
