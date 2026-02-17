import { memo, useMemo } from 'react';

import { useDebounce } from '@/hooks/use_debounce';

import { EnhancedMarkdown } from '@/components/markdown_enhanced';

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
}

function areEqual(prev: MessageContentCardProps, next: MessageContentCardProps) {
	return (
		prev.content === next.content &&
		prev.streamedText === next.streamedText &&
		prev.isStreaming === next.isStreaming &&
		prev.isBusy === next.isBusy &&
		prev.align === next.align &&
		prev.renderAsMarkdown === next.renderAsMarkdown
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
}: MessageContentCardProps) {
	const liveText = isStreaming ? streamedText : content;
	// Max ~4×/sec.
	const textToRender = useDebounce(liveText, 128);
	const isSettling = liveText !== textToRender;
	const renderBusy = isBusy || isSettling;

	// Compute plain-text nodes unconditionally to keep hook order stable.
	// Work is gated by renderAsMarkdown so we avoid heavy work when Markdown is on.
	const plainTextNodes = useMemo(() => {
		if (renderAsMarkdown) return null;
		return textToRender.split('\n').map((line, idx) => (
			<p
				key={idx}
				className={`${align} wrap-break-word`}
				style={{ whiteSpace: 'pre-wrap', lineHeight: '1.5', fontSize: '14px' }}
			>
				{line === '' ? <br /> : line}
			</p>
		));
	}, [textToRender, align, renderAsMarkdown]);

	// If we truly have nothing:
	// - while busy: show a small loader so non-streaming doesn't look "empty"
	// - otherwise: render nothing
	if (textToRender.trim() === '') {
		if (!isBusy) return null;
		return (
			<div className="flex items-center gap-2 p-0">
				Thinking <span className="loading loading-dots loading-sm ml-2" />
			</div>
		);
	}
	if (!renderAsMarkdown) {
		return (
			<div className="p-0">
				{plainTextNodes}
				{isStreaming && (
					<div className="flex items-center gap-2 p-0">
						Streaming <span className="loading loading-dots loading-sm ml-2" />
					</div>
				)}
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
			/>
			{isStreaming && (
				<div className="flex items-center gap-2 p-0">
					Streaming <span className="loading loading-dots loading-sm ml-2" />
				</div>
			)}
		</div>
	);
}, areEqual);
