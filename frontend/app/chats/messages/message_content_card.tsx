import { memo, useLayoutEffect, useState, useSyncExternalStore } from 'react';

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

interface StreamingMarkdownSegment {
	key: string;
	text: string;
}

interface StreamingMarkdownState {
	committedLength: number;
	generation: number;
	segments: StreamingMarkdownSegment[];
}

const STREAMING_MARKDOWN_SEGMENT_TARGET_LENGTH = 4_096;

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
	const previous = segments.at(-1);
	if (previous && previous.text.length + text.length <= STREAMING_MARKDOWN_SEGMENT_TARGET_LENGTH) {
		return [...segments.slice(0, -1), { ...previous, text: `${previous.text}${text}` }];
	}

	return [...segments, { key: `${generation}:${offset}`, text }];
}

function useStreamingMarkdownSegments(text: string) {
	const [state, setState] = useState<StreamingMarkdownState>({
		committedLength: 0,
		generation: 0,
		segments: [],
	});

	useLayoutEffect(() => {
		// oxlint-disable-next-line jsreact-hooks/set-state-in-effect
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

	return {
		segments: stateMatchesText ? state.segments : [],
		tail: text.slice(committedLength),
	};
}

function StreamingMarkdownContent(props: {
	source: MessageStreamSource;
	align: string;
	diffCandidatePaths?: string[];
	defaultCodeBlockExpanded: boolean;
}) {
	const text = useSyncExternalStore(props.source.subscribe, props.source.getText, props.source.getText);
	const { segments, tail } = useStreamingMarkdownSegments(text);

	if (text.length === 0) {
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
