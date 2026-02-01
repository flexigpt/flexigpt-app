// src/chats/messages/message_citations_bar.tsx
import { FiExternalLink } from 'react-icons/fi';

import type { URLCitation } from '@/spec/inference';

import { backendAPI } from '@/apis/baseapi';

interface MessageCitationsBarProps {
	citations?: URLCitation[];
}

/**
 * Row of URL citation pills shown under assistant messages, above the
 * attachments/tool bar. No horizontal scroll; pills wrap to multiple rows.
 */
export function MessageCitationsBar({ citations }: MessageCitationsBarProps) {
	const maxTooltipLen = 240;

	if (!citations || citations.length === 0) return null;

	return (
		<div className="flex flex-wrap items-center gap-2 text-xs">
			{citations.map((u, idx) => {
				const display =
					u.title && u.title.trim().length > 0
						? u.title.trim()
						: (() => {
								try {
									const url = new URL(u.url);
									return url.hostname || u.url;
								} catch {
									return u.url;
								}
							})();

				// Build a concise but info-dense tooltip:
				// Title (or display) • "cited text snippet" • Range • URL
				const tooltipParts: string[] = [];

				// Title / display always first
				tooltipParts.push('• ' + display);

				if (u.citedText && u.citedText.trim()) {
					const raw = u.citedText.trim().replace(/\s+/g, ' ');
					const snippet = raw.length > maxTooltipLen ? `${raw.slice(0, maxTooltipLen - 1)}…` : raw;
					tooltipParts.push(`“${snippet}”`);
				}

				if (u.startIndex != null || u.endIndex != null) {
					tooltipParts.push(`Range: ${u.startIndex ?? '?'}–${u.endIndex ?? '?'}`);
				}

				// Always include URL last so full target is inspectable
				tooltipParts.push(u.url);

				const tooltipText = tooltipParts.join('\n• ');

				return (
					<button
						key={`${u.url}-${u.startIndex ?? ''}-${u.endIndex ?? ''}-${idx}`}
						type="button"
						className="btn btn-xs btn-ghost border-base-300 bg-base-200 inline-flex max-w-36 items-center gap-2 rounded-2xl border px-2 py-0 text-left font-normal"
						onClick={() => {
							backendAPI.openURL(u.url);
						}}
						aria-label={display}
						title={tooltipText}
					>
						<span className="truncate">{display}</span>
						<FiExternalLink size={12} className="shrink-0 opacity-80" />
					</button>
				);
			})}
		</div>
	);
}
