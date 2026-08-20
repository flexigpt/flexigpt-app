import { memo, useLayoutEffect, useState, useSyncExternalStore } from 'react';

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

interface StreamingMarkdownState {
	committedLength: number;
	generation: number;
	segments: StreamingMarkdownSegment[];
}

const MAX_STREAMING_MARKDOWN_SEGMENTS = 96;
const STREAMING_MARKDOWN_COMPACT_SEGMENT_COUNT = 24;

function findSafeStreamingMarkdownCutoff(text: string): number {
	let openFence: string | undefined;
	let cutoff = 0;
	let offset = 0;

	for (const line of text.split('\n')) {
		const nextOffset = offset + line.length + 1;
		const fenceMatch = /^ {0,3}(`{3,}|~{3,})/.exec(line);

		if (fenceMatch?.[1]) {
			const marker = fenceMatch[1];
			if (!openFence) {
				openFence = marker;
			} else if (marker.startsWith(openFence)) {
				openFence = undefined;
				cutoff = nextOffset;
			}
		} else if (!openFence && line.trim().length === 0) {
			cutoff = nextOffset;
		}

		offset = nextOffset;
	}

	return Math.min(cutoff, text.length);
}

function appendStreamingMarkdownSegment(
	segments: StreamingMarkdownSegment[],
	text: string,
	generation: number,
	offset: number
): StreamingMarkdownSegment[] {
	const next = [...segments, { key: `${generation}:${offset}`, text }];

	// Never mutate the latest committed segment. EnhancedMarkdown is memoized,
	// so immutable segments avoid reparsing old content for every new chunk.
	if (next.length <= MAX_STREAMING_MARKDOWN_SEGMENTS) {
		return next;
	}

	const compacted = next.slice(0, STREAMING_MARKDOWN_COMPACT_SEGMENT_COUNT);
	return [
		{
			key: `${generation}:compact:${offset}`,
			text: compacted.map(segment => segment.text).join(''),
		},
		...next.slice(STREAMING_MARKDOWN_COMPACT_SEGMENT_COUNT),
	];
}

function useStreamingMarkdownSegments(text: string) {
	const [state, setState] = useState<StreamingMarkdownState>({
		committedLength: 0,
		generation: 0,
		segments: [],
	});

	useLayoutEffect(() => {
		// oxlint-disable-next-line react/set-state-in-effect
		setState(previous => {
			const reset = text.length < previous.committedLength;
			const base: StreamingMarkdownState = reset
				? {
						committedLength: 0,
						generation: previous.generation + 1,
						segments: [],
					}
				: previous;
			const pendingText = text.slice(base.committedLength);
			const cutoff = findSafeStreamingMarkdownCutoff(pendingText);

			if (cutoff === 0) {
				return base;
			}

			const stableText = pendingText.slice(0, cutoff);
			return {
				committedLength: base.committedLength + stableText.length,
				generation: base.generation,
				segments: appendStreamingMarkdownSegment(base.segments, stableText, base.generation, base.committedLength),
			};
		});
	}, [text]);

	const stateMatchesText = text.length >= state.committedLength;
	const committedLength = stateMatchesText ? state.committedLength : 0;
	const tailGeneration = stateMatchesText ? state.generation : state.generation + 1;
	const tailKey = `${tailGeneration}:${committedLength}`;

	return {
		segments: stateMatchesText ? state.segments : [],
		tail: text.slice(committedLength),
		tailKey,
	};
}

function StreamingMarkdownContent(props: {
	text: string;
	align: string;
	diffCandidatePaths?: string[];
	defaultCodeBlockExpanded: boolean;
}) {
	const { segments, tail, tailKey } = useStreamingMarkdownSegments(props.text);

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
