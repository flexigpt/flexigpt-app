// oxlint-disable unicorn/prefer-code-point
import type { ComponentProps, ReactNode } from 'react';
import { memo, useState } from 'react';

import { EnhancedMarkdown } from '@/components/markdown/markdown_enhanced';

const MAX_STREAMING_MARKDOWN_CHARACTERS = 64_000;
const MAX_AUTOMATIC_MARKDOWN_CHARACTERS = 128_000;
const TEXT_CHUNK_CHARACTERS = 8192;

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
	const [approvedSource, setApprovedSource] = useState<string | null>(null);
	const limit = isBusy ? MAX_STREAMING_MARKDOWN_CHARACTERS : MAX_AUTOMATIC_MARKDOWN_CHARACTERS;
	const useSourcePreview = text.length > limit && (isBusy || approvedSource !== text);

	if (!useSourcePreview) {
		return <EnhancedMarkdown {...props} />;
	}

	return (
		<div>
			<div className="border-base-300 text-base-content/70 mb-2 flex flex-wrap items-center gap-2 rounded-lg border p-2 text-xs">
				<span>
					{isBusy
						? 'Large response: showing source while streaming. All text is retained.'
						: 'Large response: source preview avoids expensive automatic formatting.'}
				</span>
				{!isBusy ? (
					<button
						type="button"
						className="btn btn-xs"
						title="Formatting a very large response may temporarily slow the UI."
						onClick={() => {
							setApprovedSource(text);
						}}
					>
						Render formatted Markdown
					</button>
				) : null}
			</div>
			<PlainMessageText text={text} align={align} />
		</div>
	);
});
