import { memo, useSyncExternalStore } from 'react';

import { EnhancedMarkdown } from '@/components/markdown/markdown_enhanced';

export interface MessageStreamSource {
	subscribe: (callback: () => void) => () => void;
	getVersionSnapshot: () => number;
	getText: () => string;
	getThinking: () => string;
}

interface MessageContentCardProps {
	messageID: string;
	// Final text
	content: string;
	isBusy?: boolean;
	align: string;
	renderAsMarkdown?: boolean;
	diffCandidatePaths?: string[];
	streamSource?: MessageStreamSource;
	defaultCodeBlockExpanded?: boolean;
}

function stringArraysEqual(left?: string[], right?: string[]): boolean {
	if (left === right) {
		return true;
	}
	if (!left || !right || left.length !== right.length) {
		return false;
	}

	return left.every((value, index) => value === right[index]);
}

function StreamingMarkdownContent(props: {
	source: MessageStreamSource;
	align: string;
	diffCandidatePaths?: string[];
	defaultCodeBlockExpanded: boolean;
}) {
	// Use text itself as the snapshot. Thinking-only stream notifications do
	// not cause the Markdown tree to reparse when the visible answer is unchanged.
	const text = useSyncExternalStore(props.source.subscribe, props.source.getText, props.source.getText);
	const hasText = /\S/.test(text);

	return (
		<div className="p-0">
			{hasText ? (
				<EnhancedMarkdown
					text={text}
					align={props.align}
					isBusy={true}
					diffCandidatePaths={props.diffCandidatePaths}
					defaultCodeBlockExpanded={props.defaultCodeBlockExpanded}
				/>
			) : null}

			<output className="flex items-center gap-2 p-0" aria-live="polite">
				Thinking <span className="loading loading-dots loading-sm ml-2" />
				<span className="sr-only">Generating response</span>
			</output>
		</div>
	);
}

function areEqual(prev: MessageContentCardProps, next: MessageContentCardProps) {
	return (
		prev.messageID === next.messageID &&
		prev.content === next.content &&
		prev.isBusy === next.isBusy &&
		prev.align === next.align &&
		prev.renderAsMarkdown === next.renderAsMarkdown &&
		stringArraysEqual(prev.diffCandidatePaths, next.diffCandidatePaths) &&
		prev.streamSource === next.streamSource &&
		prev.defaultCodeBlockExpanded === next.defaultCodeBlockExpanded
	);
}

export const MessageContentCard = memo(function MessageContentCard({
	messageID,
	content,
	isBusy = false,
	align,
	renderAsMarkdown = true,
	diffCandidatePaths,
	streamSource,
	defaultCodeBlockExpanded = true,
}: MessageContentCardProps) {
	const textToRender = content;
	const renderBusy = isBusy;

	// The live source owns text rendering while this message has the
	// in-flight request. Streaming Markdown deliberately ignores deferred
	// transcript rendering so the current answer remains formatted.
	if (isBusy && streamSource) {
		return (
			<StreamingMarkdownContent
				source={streamSource}
				align={align}
				diffCandidatePaths={diffCandidatePaths}
				defaultCodeBlockExpanded={defaultCodeBlockExpanded}
			/>
		);
	}

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

	// Deferred transcript paints use the inexpensive plain-text presentation.
	// Live stream text has already returned through AppendOnlyStreamingText.
	if (!renderAsMarkdown) {
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
				defaultCodeBlockExpanded={defaultCodeBlockExpanded}
			/>
		</div>
	);
}, areEqual);
