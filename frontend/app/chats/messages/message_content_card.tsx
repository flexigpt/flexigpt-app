import { memo } from 'react';

import { EnhancedMarkdown } from '@/components/markdown/markdown_enhanced';

interface MessageContentCardProps {
	messageID: string;
	// Final text
	content: string;
	// Partial text while streaming.
	streamedText?: string;
	isStreaming?: boolean;
	isBusy?: boolean;
	align: string;
	renderAsMarkdown?: boolean;
	diffCandidatePaths?: string[];
}

function areEqual(prev: MessageContentCardProps, next: MessageContentCardProps) {
	return (
		prev.content === next.content &&
		prev.streamedText === next.streamedText &&
		prev.isStreaming === next.isStreaming &&
		prev.isBusy === next.isBusy &&
		prev.align === next.align &&
		prev.renderAsMarkdown === next.renderAsMarkdown &&
		prev.diffCandidatePaths === next.diffCandidatePaths
	);
}

export const MessageContentCard = memo(function MessageContentCard({
	messageID,
	content,
	streamedText = '',
	isStreaming = false,
	isBusy = false,
	align,
	renderAsMarkdown = true,
	diffCandidatePaths,
}: MessageContentCardProps) {
	const liveText = isStreaming ? streamedText : content;
	// Backend already throttles few ms.
	const textToRender = liveText;
	const renderBusy = isBusy;

	// If we truly have nothing:
	// - while busy: show a small loader so non-streaming doesn't look "empty"
	// - otherwise: render nothing
	if (!/\S/.test(textToRender)) {
		if (!isBusy) {
			return null;
		}
		return (
			<div className="flex items-center gap-2 p-0">
				Thinking <span className="loading loading-dots loading-sm ml-2" />
			</div>
		);
	}

	// Parsing the complete accumulated response as Markdown for every token is
	// the dominant streaming cost. Keep streaming and intentionally deferred
	// transcript paints on one cheap text node, then render rich Markdown once
	// the content has settled.
	if (isStreaming || !renderAsMarkdown) {
		return (
			<div
				className={`${align} wrap-break-word whitespace-pre-wrap`}
				style={{ lineHeight: 1.5, fontSize: 14, contain: 'paint' }}
			>
				{textToRender}
			</div>
		);
	}

	return (
		<div className="p-0">
			<EnhancedMarkdown
				key={`${messageID}:${renderBusy ? 'live' : 'done'}`}
				text={textToRender}
				align={align}
				isBusy={renderBusy}
				diffCandidatePaths={diffCandidatePaths}
			/>
		</div>
	);
}, areEqual);
