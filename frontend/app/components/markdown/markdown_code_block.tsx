import { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react';

import { FiAlertTriangle, FiChevronDown, FiChevronUp } from 'react-icons/fi';

import { useHighlight } from '@/hooks/use_highlight';

import { CopyButton } from '@/components/copy_button';
import { DownloadButton } from '@/components/download_button';
import { DiffApplyControl } from '@/components/markdown/diff_apply_control';
import type { MermaidRenderStatus } from '@/components/markdown/mermaid_diagram_card';
import { MermaidDiagram } from '@/components/markdown/mermaid_diagram_card';
import { looksLikeUnifiedDiff } from '@/components/markdown/unified_diff_block';

interface CodeProps {
	language: string;
	value: string;
	isBusy: boolean;
	hideMermaidCode: boolean;
	diffCandidatePaths?: string[];
	defaultExpanded?: boolean;
}

interface MermaidResultState {
	key: string;
	status: Extract<MermaidRenderStatus, 'rendered' | 'error'>;
	message?: string;
}

interface ExpansionOverrideState {
	key: string;
	isExpanded: boolean;
}

const getCodeBlockKey = (language: string, value: string) => `${language.toLowerCase()}\u0000${value}`;

function useNearViewport(enabled: boolean) {
	const elementRef = useRef<HTMLDivElement | null>(null);
	const [activated, setActivated] = useState(false);

	useEffect(() => {
		if (!enabled || activated) {
			return;
		}

		const element = elementRef.current;
		if (!element) {
			return;
		}

		if (typeof IntersectionObserver === 'undefined') {
			const frame = window.requestAnimationFrame(() => {
				setActivated(true);
			});
			return () => {
				window.cancelAnimationFrame(frame);
			};
		}

		const observer = new IntersectionObserver(
			entries => {
				if (entries.some(entry => entry.isIntersecting)) {
					setActivated(true);
					observer.disconnect();
				}
			},
			{ rootMargin: '800px 0px' }
		);

		observer.observe(element);
		return () => {
			observer.disconnect();
		};
	}, [activated, enabled]);

	return { elementRef, activated };
}

export function CodeBlock({
	language,
	value,
	isBusy,
	hideMermaidCode,
	diffCandidatePaths,
	defaultExpanded = true,
}: CodeProps) {
	const codeBodyId = useId();

	const normalizedLanguage = language.toLowerCase();
	const isMermaid = normalizedLanguage === 'mermaid';
	const codeBlockKey = getCodeBlockKey(language, value);

	const [mermaidResult, setMermaidResult] = useState<MermaidResultState | null>(null);
	const [expansionOverride, setExpansionOverride] = useState<ExpansionOverrideState | null>(null);

	const currentMermaidResult = isMermaid && mermaidResult?.key === codeBlockKey ? mermaidResult : null;

	const mermaidRenderStatus: MermaidRenderStatus =
		!isMermaid || isBusy || !value.trim()
			? 'idle'
			: currentMermaidResult?.status === 'error'
				? 'error'
				: currentMermaidResult?.status === 'rendered'
					? 'rendered'
					: 'rendering';

	const mermaidRenderError = currentMermaidResult?.status === 'error' ? currentMermaidResult.message : null;
	const hasMermaidSyntaxError = isMermaid && mermaidRenderStatus === 'error';

	// Default behavior:
	// - normal code: caller-controlled (expanded unless explicitly collapsed)
	// - valid/rendering Mermaid: collapsed
	// - errored Mermaid: expanded unless the caller requested hidden Mermaid code behavior
	const defaultIsExpanded = isMermaid ? hasMermaidSyntaxError && !hideMermaidCode : defaultExpanded;
	const isExpanded = expansionOverride?.key === codeBlockKey ? expansionOverride.isExpanded : defaultIsExpanded;

	const { elementRef, activated: richCodeWorkActivated } = useNearViewport(!isBusy);
	const html = useHighlight(value, language, richCodeWorkActivated && isExpanded);
	const isDiffLike = useMemo(
		() => richCodeWorkActivated && !isBusy && looksLikeUnifiedDiff(value, language),
		[isBusy, language, richCodeWorkActivated, value]
	);

	const highlightedHtml = html ?? '';
	const showFallback = !value.trim() || html === null || html === '';

	const headerLabel = hasMermaidSyntaxError
		? 'Mermaid syntax error'
		: isMermaid
			? language + ' Code'
			: language || 'text';
	const headerTitle = hasMermaidSyntaxError ? (mermaidRenderError ?? 'Mermaid syntax error') : undefined;

	const fallback = (
		<pre className="app-text-code overflow-auto rounded-sm bg-transparent p-2 text-sm">
			<code>{value}</code>
		</pre>
	);

	const fetchValue = useCallback(async () => value, [value]);

	const handleToggleExpanded = () => {
		setExpansionOverride({
			key: codeBlockKey,
			isExpanded: !isExpanded,
		});
	};

	const handleMermaidRenderStatusChange = useCallback(
		(status: MermaidRenderStatus, message?: string) => {
			if (status === 'rendered' || status === 'error') {
				setMermaidResult({
					key: codeBlockKey,
					status,
					message,
				});
			}
		},
		[codeBlockKey]
	);

	return (
		<>
			<div ref={elementRef} className="app-bg-code my-4 overflow-hidden rounded-lg">
				<div className="app-bg-code-header flex min-w-0 items-center justify-between gap-2 px-2 py-0.5">
					<div className="flex min-w-0 flex-1 items-center gap-2 overflow-hidden text-xs" title={headerTitle}>
						<span
							className={`inline-flex max-w-48 min-w-0 shrink-0 items-center gap-1 leading-none ${
								hasMermaidSyntaxError ? 'text-error' : 'app-text-code capitalize'
							}`}
						>
							{hasMermaidSyntaxError ? (
								<FiAlertTriangle aria-hidden="true" size={14} className="block shrink-0" />
							) : null}
							<span className="truncate leading-none">{headerLabel}</span>
						</span>

						{isDiffLike ? (
							<DiffApplyControl
								key={codeBlockKey}
								language={language}
								diffText={value}
								isBusy={isBusy}
								candidatePaths={diffCandidatePaths}
							/>
						) : null}
					</div>
					<div className="flex shrink-0 items-center gap-1">
						<DownloadButton
							language={language}
							valueFetcher={fetchValue}
							size={16}
							className="btn btn-sm app-text-code flex items-center border-none bg-transparent shadow-none hover:opacity-60"
						/>

						<CopyButton
							value={value}
							className="btn btn-sm app-text-code flex items-center border-none bg-transparent shadow-none hover:opacity-60"
							size={16}
						/>
						<button
							type="button"
							className="btn btn-sm app-text-code flex items-center border-none bg-transparent shadow-none hover:opacity-60"
							onClick={handleToggleExpanded}
							aria-expanded={isExpanded}
							aria-controls={codeBodyId}
							title={isExpanded ? 'Collapse code' : 'Expand code'}
						>
							{isExpanded ? (
								<FiChevronUp aria-hidden="true" size={16} />
							) : (
								<FiChevronDown aria-hidden="true" size={16} />
							)}
							<span className="sr-only">{isExpanded ? 'Collapse code' : 'Expand code'}</span>
						</button>
					</div>
				</div>

				{isExpanded && (
					<div id={codeBodyId} className="app-text-code p-1" style={{ fontSize: 14, lineHeight: 1.5 }}>
						{showFallback ? (
							fallback
						) : (
							<div
								className="app-shiki-container max-w-full overflow-x-auto"
								// oxlint-disable-next-line react/no-danger
								dangerouslySetInnerHTML={{ __html: highlightedHtml }}
							/>
						)}
					</div>
				)}
			</div>

			{isMermaid && !isBusy && richCodeWorkActivated ? (
				<MermaidDiagram code={value} onRenderStatusChange={handleMermaidRenderStatusChange} />
			) : null}
		</>
	);
}
