import { memo, useEffect, useState, useSyncExternalStore } from 'react';

import { BoundedMarkdown, PlainMessageText } from '@/components/markdown/bounded_markdown';

const MAX_INITIAL_RICH_RENDER_DELAY_MS = 150;

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
	deferRichRendering?: boolean;
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

function areEqual(prev: MessageContentCardProps, next: MessageContentCardProps) {
	return (
		prev.messageID === next.messageID &&
		prev.content === next.content &&
		prev.isBusy === next.isBusy &&
		prev.align === next.align &&
		prev.renderAsMarkdown === next.renderAsMarkdown &&
		prev.deferRichRendering === next.deferRichRendering &&
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
	deferRichRendering = false,
	diffCandidatePaths,
	streamingText,
	defaultCodeBlockExpanded = true,
}: MessageContentCardProps) {
	const [richRenderingReady, setRichRenderingReady] = useState(!deferRichRendering);

	useEffect(() => {
		if (richRenderingReady || !renderAsMarkdown) {
			return;
		}
		const timeout = window.setTimeout(
			() => {
				setRichRenderingReady(true);
			},
			deferRichRendering ? MAX_INITIAL_RICH_RENDER_DELAY_MS : 0
		);
		return () => {
			window.clearTimeout(timeout);
		};
	}, [deferRichRendering, renderAsMarkdown, richRenderingReady]);

	const textToRender = isBusy && streamingText !== undefined ? streamingText : content;

	if (!/\S/.test(textToRender)) {
		return null;
	}

	// Deferral is an initial scheduling hint, not a permanent veto on final Markdown.
	if (!renderAsMarkdown || (deferRichRendering && !isBusy && !richRenderingReady)) {
		return <PlainMessageText text={textToRender} align={align} />;
	}

	return (
		<div className="p-0">
			<BoundedMarkdown
				key={messageID}
				text={textToRender}
				align={align}
				isBusy={isBusy}
				diffCandidatePaths={diffCandidatePaths}
				defaultCodeBlockExpanded={defaultCodeBlockExpanded}
			/>
		</div>
	);
}, areEqual);
