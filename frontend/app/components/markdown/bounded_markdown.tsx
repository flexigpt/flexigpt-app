// oxlint-disable unicorn/prefer-code-point
import type { ComponentProps, ReactNode } from 'react';
import { memo, useEffect, useMemo, useRef, useState } from 'react';

import { EnhancedMarkdown } from '@/components/markdown/markdown_enhanced';

const MAX_STREAMING_MARKDOWN_CHARACTERS = 128_000;
const MAX_STREAMING_MARKDOWN_LINES = 16384;
const MAX_STREAMING_MARKDOWN_MARKERS = 8192;
const STREAMING_MARKDOWN_INTERVAL_MS = 32;
const STREAMING_MARKUP_CHARACTERS = '*_~`|<>[]$\\';
const TEXT_CHUNK_CHARACTERS = 8192;

function exceedsStreamingMarkdownBudget(text: string): boolean {
	if (text.length > MAX_STREAMING_MARKDOWN_CHARACTERS) {
		return true;
	}

	let lines = 1;
	let markers = 0;
	for (const character of text) {
		if (character === '\n' && ++lines > MAX_STREAMING_MARKDOWN_LINES) {
			return true;
		}
		if (STREAMING_MARKUP_CHARACTERS.includes(character) && ++markers > MAX_STREAMING_MARKDOWN_MARKERS) {
			return true;
		}
	}
	return false;
}

function useCoalescedMarkdownText(text: string, enabled: boolean): string {
	const [snapshot, setSnapshot] = useState(() => (enabled ? text : ''));
	const latestTextRef = useRef('');
	const timerRef = useRef<number | null>(null);

	useEffect(() => {
		if (!enabled) {
			return;
		}
		latestTextRef.current = text;
		if (timerRef.current !== null || text === snapshot) {
			return;
		}
		timerRef.current = window.setTimeout(() => {
			timerRef.current = null;
			setSnapshot(latestTextRef.current);
		}, STREAMING_MARKDOWN_INTERVAL_MS);
	}, [enabled, snapshot, text]);

	useEffect(
		() => () => {
			if (timerRef.current !== null) {
				window.clearTimeout(timerRef.current);
				timerRef.current = null;
			}
		},
		[enabled]
	);

	// Coalesce append-only streaming updates; replacements and final text
	// must never show a snapshot from a different source.
	return enabled && snapshot !== '' && text.startsWith(snapshot) ? snapshot : text;
}

const TextChunk = memo(function TextChunk({ value }: { value: string }) {
	return <span>{value}</span>;
});

export const PlainMessageText = memo(function PlainMessageText({ text, align = '' }: { text: string; align?: string }) {
	const chunks: ReactNode[] = [];
	for (let start = 0; start < text.length;) {
		let end = Math.min(text.length, start + TEXT_CHUNK_CHARACTERS);
		const last = text.charCodeAt(end - 1);
		const next = text.charCodeAt(end);
		if (last >= 0xd800 && last <= 0xdbff && next >= 0xdc00 && next <= 0xdfff) {
			end -= 1;
		}
		chunks.push(<TextChunk key={start} value={text.slice(start, end)} />);
		start = end;
	}
	return (
		<div className={`${align} whitespace-pre-wrap`} style={{ overflowWrap: 'anywhere', lineHeight: 1.5, fontSize: 14 }}>
			{chunks}
		</div>
	);
});

export const BoundedMarkdown = memo(function BoundedMarkdown(props: ComponentProps<typeof EnhancedMarkdown>) {
	const { text, isBusy = false, align } = props;
	const useSourcePreview = useMemo(() => isBusy && exceedsStreamingMarkdownBudget(text), [isBusy, text]);
	const renderedText = useCoalescedMarkdownText(text, isBusy && !useSourcePreview);

	if (!useSourcePreview) {
		return <EnhancedMarkdown {...props} text={renderedText} />;
	}

	return (
		<div>
			<div className="border-base-300 text-base-content/70 mb-2 flex flex-wrap items-center gap-2 rounded-lg border p-2 text-xs">
				<span>
					Large or complex response: showing source while streaming. Markdown renders automatically when complete.
				</span>
			</div>
			<PlainMessageText text={text} align={align} />
		</div>
	);
});
