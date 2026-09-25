import { memo, useMemo, useSyncExternalStore } from 'react';

import { EnhancedMarkdown } from '@/components/markdown/markdown_enhanced';

export interface MessageStreamSource {
	subscribe: (callback: () => void) => () => void;
	getVersionSnapshot: () => number;
	getText: () => string;
	getThinking: () => string;
}

export interface MessageStreamSnapshot {
	version: number;
	text: string;
	thinking: string;
}

const EMPTY_STREAM_SUBSCRIBE = () => () => {};
const EMPTY_STREAM_VERSION = () => 0;
const EMPTY_STREAM_SNAPSHOT: MessageStreamSnapshot = {
	version: 0,
	text: '',
	thinking: '',
};

// oxlint-disable-next-line react/only-export-components
export function useMessageStreamSnapshot(
	source: MessageStreamSource | undefined,
	enabled: boolean
): MessageStreamSnapshot {
	const activeSource = enabled ? source : undefined;
	const version = useSyncExternalStore(
		activeSource?.subscribe ?? EMPTY_STREAM_SUBSCRIBE,
		activeSource?.getVersionSnapshot ?? EMPTY_STREAM_VERSION,
		EMPTY_STREAM_VERSION
	);

	if (!activeSource) {
		return EMPTY_STREAM_SNAPSHOT;
	}

	return {
		version,
		text: activeSource.getText(),
		thinking: activeSource.getThinking(),
	};
}

interface MessageContentCardProps {
	messageID: string;
	// Final text
	content: string;
	isBusy?: boolean;
	align: string;
	renderAsMarkdown?: boolean;
	diffCandidatePaths?: string[];
	streamingText?: string;
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

interface StreamingMarkdownSegment {
	key: string;
	text: string;
}

const STREAMING_MARKDOWN_SEGMENT_TARGET_LENGTH = 4096;

function splitStreamingMarkdown(text: string): {
	segments: StreamingMarkdownSegment[];
	tail: string;
	tailKey: string;
} {
	const segments: StreamingMarkdownSegment[] = [];
	let openFence: string | undefined;
	let segmentStart = 0;
	let offset = 0;
	const lines = text.split('\n');

	for (let index = 0; index < lines.length; index += 1) {
		const line = lines[index] ?? '';
		const hasTrailingNewline = index < lines.length - 1;
		const nextOffset = offset + line.length + (hasTrailingNewline ? 1 : 0);
		const fenceMatch = /^ {0,3}(`{3,}|~{3,})/.exec(line);
		let isSafeBoundary = false;

		if (fenceMatch?.[1]) {
			const marker = fenceMatch[1];
			if (!openFence) {
				openFence = marker;
			} else if (marker.startsWith(openFence)) {
				openFence = undefined;
				isSafeBoundary = true;
			}
		} else if (!openFence && line.trim().length === 0) {
			isSafeBoundary = true;
		}

		if (isSafeBoundary && nextOffset - segmentStart >= STREAMING_MARKDOWN_SEGMENT_TARGET_LENGTH) {
			segments.push({
				key: `${segmentStart}:${nextOffset}`,
				text: text.slice(segmentStart, nextOffset),
			});
			segmentStart = nextOffset;
		}

		offset = nextOffset;
	}

	return {
		segments,
		tail: text.slice(segmentStart),
		tailKey: `tail:${segmentStart}`,
	};
}

function StreamingMarkdownContent(props: {
	text: string;
	align: string;
	diffCandidatePaths?: string[];
	defaultCodeBlockExpanded: boolean;
}) {
	const { segments, tail, tailKey } = useMemo(() => splitStreamingMarkdown(props.text), [props.text]);

	if (props.text.length === 0) {
		return null;
	}

	// Completed safe blocks are memoized independently. Only the active tail
	// reparses for most token callbacks, keeping rich streaming responsive.
	return (
		<div className="p-0">
			{segments.map(segment => (
				<EnhancedMarkdown
					key={segment.key}
					text={segment.text}
					align={props.align}
					isBusy={true}
					diffCandidatePaths={props.diffCandidatePaths}
					defaultCodeBlockExpanded={props.defaultCodeBlockExpanded}
				/>
			))}
			{tail ? (
				<EnhancedMarkdown
					key={tailKey}
					text={tail}
					align={props.align}
					isBusy={true}
					diffCandidatePaths={props.diffCandidatePaths}
					defaultCodeBlockExpanded={props.defaultCodeBlockExpanded}
				/>
			) : null}
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
		prev.streamingText === next.streamingText &&
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
	streamingText,
	defaultCodeBlockExpanded = true,
}: MessageContentCardProps) {
	const textToRender = content;
	const renderBusy = isBusy;

	// The live source owns text rendering while this message has the
	// in-flight request. Streaming Markdown deliberately ignores deferred
	// transcript rendering so the current answer remains formatted.
	if (isBusy && streamingText !== undefined) {
		return (
			<StreamingMarkdownContent
				text={streamingText}
				align={align}
				diffCandidatePaths={diffCandidatePaths}
				defaultCodeBlockExpanded={defaultCodeBlockExpanded}
			/>
		);
	}

	// Streaming state belongs to the message footer. Keeping the body empty
	// avoids disconnected loaders and prevents layout changes when text starts.
	if (!/\S/.test(textToRender)) {
		return null;
	}

	// Deferred transcript paints use the inexpensive plain-text presentation.
	// Live stream text is handled above by the segmented Markdown renderer.
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
