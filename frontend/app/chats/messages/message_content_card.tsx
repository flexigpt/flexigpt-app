import { memo, useLayoutEffect, useRef } from 'react';

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
	// Partial text while streaming.
	streamedText?: string;
	isStreaming?: boolean;
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

function AppendOnlyStreamingText(props: { source: MessageStreamSource; align: string }) {
	const textElementRef = useRef<HTMLDivElement | null>(null);
	const loaderElementRef = useRef<HTMLDivElement | null>(null);
	const renderedLengthRef = useRef(0);
	const hasVisibleTextRef = useRef(false);

	useLayoutEffect(() => {
		const textElement = textElementRef.current;
		const loaderElement = loaderElementRef.current;
		if (!textElement || !loaderElement) {
			return;
		}

		renderedLengthRef.current = 0;
		hasVisibleTextRef.current = false;
		textElement.textContent = '';

		const syncText = () => {
			const nextText = props.source.getText();
			const previousLength = renderedLengthRef.current;

			if (nextText.length < previousLength) {
				textElement.textContent = nextText;
				renderedLengthRef.current = nextText.length;
				hasVisibleTextRef.current = /\S/.test(nextText);
			} else if (nextText.length > previousLength) {
				const appendedText = nextText.slice(previousLength);
				const currentTextNode = textElement.firstChild;

				if (previousLength > 0 && currentTextNode instanceof Text && currentTextNode.nextSibling === null) {
					currentTextNode.appendData(appendedText);
				} else {
					textElement.textContent = nextText;
				}

				renderedLengthRef.current = nextText.length;

				if (!hasVisibleTextRef.current && /\S/.test(appendedText)) {
					hasVisibleTextRef.current = true;
				}
			}

			const hasVisibleText = hasVisibleTextRef.current;
			textElement.style.display = hasVisibleText ? '' : 'none';
			loaderElement.style.display = hasVisibleText ? 'none' : '';
		};

		syncText();
		return props.source.subscribe(syncText);
	}, [props.source]);

	return (
		<>
			<div
				ref={textElementRef}
				className={`${props.align} wrap-break-word whitespace-pre-wrap`}
				style={{
					display: 'none',
					lineHeight: 1.5,
					fontSize: 14,
					contain: 'paint',
				}}
			/>
			<div ref={loaderElementRef} className="flex items-center gap-2 p-0">
				Thinking <span className="loading loading-dots loading-sm ml-2" />
			</div>
		</>
	);
}

function areEqual(prev: MessageContentCardProps, next: MessageContentCardProps) {
	return (
		prev.messageID === next.messageID &&
		prev.content === next.content &&
		prev.streamedText === next.streamedText &&
		prev.isStreaming === next.isStreaming &&
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
	streamedText = '',
	isStreaming = false,
	isBusy = false,
	align,
	renderAsMarkdown = true,
	diffCandidatePaths,
	streamSource,
	defaultCodeBlockExpanded = true,
}: MessageContentCardProps) {
	const liveText = isStreaming ? streamedText : content;
	const textToRender = liveText;
	const renderBusy = isBusy;

	if (isStreaming && streamSource) {
		return <AppendOnlyStreamingText source={streamSource} align={align} />;
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
				defaultCodeBlockExpanded={defaultCodeBlockExpanded}
			/>
		</div>
	);
}, areEqual);
