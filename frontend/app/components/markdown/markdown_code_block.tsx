import type { ReactNode } from 'react';
import { createElement, useCallback, useEffect, useId, useMemo, useRef, useState } from 'react';
import { FiAlertTriangle, FiChevronDown, FiChevronUp } from 'react-icons/fi';

import { useHighlight } from '@/hooks/use_highlight';

import type { MermaidRenderStatus } from '@/components/markdown/mermaid_diagram_card';
import { CopyButton } from '@/components/copy_button';
import { DownloadButton } from '@/components/download_button';
import { DiffApplyControl } from '@/components/markdown/diff_apply_control';
import { looksLikeInteractiveDiff } from '@/components/markdown/diff_apply_model';
import { MermaidDiagram } from '@/components/markdown/mermaid_diagram_card';

interface CodeProps {
	language: string;
	value: string;
	isBusy: boolean;
	hideMermaidCode: boolean;
	diffCandidatePaths?: string[];
	diffWorkspaceRoots?: string[];
	defaultExpanded?: boolean;
	disableControls?: boolean;
	autoReviewEpoch?: number;
}

interface MermaidResultState {
	key: string;
	status: Extract<MermaidRenderStatus, 'error'>;
	message?: string;
}

interface ExpansionOverrideState {
	key: string;
	isExpanded: boolean;
}

const getCodeBlockKey = (language: string, value: string) => `${language.toLowerCase()}\u0000${value}`;

const MAX_HIGHLIGHT_CHARACTERS = 100_000;
const shikiAllowedTags = new Set(['code', 'pre', 'span']);

function renderShikiNode(node: Node, key: string): ReactNode {
	if (node.nodeType === Node.TEXT_NODE) {
		return node.textContent;
	}

	if (node.nodeType !== Node.ELEMENT_NODE) {
		return null;
	}

	const element = node as HTMLElement;
	const children = [...element.childNodes].map((child, index) => renderShikiNode(child, `${key}-${index}`));
	const tagName = element.tagName.toLowerCase();

	if (!shikiAllowedTags.has(tagName)) {
		return children;
	}

	return createElement(
		tagName,
		{
			key,
			className: element.className || undefined,
			style: {
				backgroundColor: element.style.backgroundColor || undefined,
				color: element.style.color || undefined,
				fontStyle: element.style.fontStyle || undefined,
				fontWeight: element.style.fontWeight || undefined,
			},
		},
		children
	);
}

function ShikiHighlightedCode({ html }: { html: string }) {
	const content = useMemo(() => {
		if (typeof DOMParser === 'undefined') {
			return null;
		}

		const document = new DOMParser().parseFromString(html, 'text/html');
		return [...document.body.childNodes].map((node, index) => renderShikiNode(node, String(index)));
	}, [html]);

	// oxlint-disable-next-line react/jsx-no-useless-fragment
	return <>{content}</>;
}

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
	diffWorkspaceRoots,
	defaultExpanded = true,
	disableControls = false,
	autoReviewEpoch = 0,
}: CodeProps) {
	const codeBodyId = useId();

	const normalizedLanguage = language.toLowerCase();
	const isMermaid = normalizedLanguage === 'mermaid';
	const codeBlockKey = getCodeBlockKey(language, value);

	const [mermaidResult, setMermaidResult] = useState<MermaidResultState | null>(null);
	const [expansionOverride, setExpansionOverride] = useState<ExpansionOverrideState | null>(null);

	const currentMermaidResult = isMermaid && mermaidResult?.key === codeBlockKey ? mermaidResult : null;
	// Streaming values are not stable enough for diff application controls.
	// They become available once the code block is no longer busy.
	const diffControlsDisabled = disableControls || isBusy;

	const mermaidRenderStatus: MermaidRenderStatus =
		!isMermaid || isBusy || !value.trim() ? 'idle' : currentMermaidResult?.status === 'error' ? 'error' : 'rendering';

	const mermaidRenderError = currentMermaidResult?.status === 'error' ? currentMermaidResult.message : null;
	const hasMermaidSyntaxError = isMermaid && mermaidRenderStatus === 'error';

	// Default behavior:
	// - normal code: caller-controlled (expanded unless explicitly collapsed)
	// - valid/rendering Mermaid: collapsed
	// - errored Mermaid: expanded unless the caller requested hidden Mermaid code behavior
	const defaultIsExpanded = isMermaid ? hasMermaidSyntaxError && !hideMermaidCode : defaultExpanded;
	const isExpanded = expansionOverride?.key === codeBlockKey ? expansionOverride.isExpanded : defaultIsExpanded;

	const { elementRef, activated: richCodeWorkActivated } = useNearViewport(!isMermaid || !isBusy);
	// Shiki replaces the complete code subtree whenever a result arrives.
	// Deferring does not coalesce token updates, so keep its input stable and
	// render the current raw value until the stream has settled.
	const withinHighlightBudget = value.length <= MAX_HIGHLIGHT_CHARACTERS;
	const shouldHighlight = !isBusy && richCodeWorkActivated && isExpanded && withinHighlightBudget;
	const valueForHighlight = isBusy || !withinHighlightBudget ? '' : value;
	const html = useHighlight(valueForHighlight, language, shouldHighlight);

	const isDiffLike = useMemo(
		() => !diffControlsDisabled && looksLikeInteractiveDiff(value, language),
		[diffControlsDisabled, language, value]
	);

	const highlightedHtml = html ?? '';
	const showFallback = !withinHighlightBudget || isBusy || !value.trim() || html === null || html === '';

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
			// A successful render does not change CodeBlock presentation.
			// Avoid rebuilding this subtree merely to store "rendered".
			if (status !== 'error') {
				return;
			}

			setMermaidResult({
				key: codeBlockKey,
				status,
				message,
			});
		},
		[codeBlockKey]
	);

	return (
		<>
			<div ref={elementRef} className="app-bg-code my-4 overflow-hidden rounded-lg">
				<div className="app-bg-code-header flex min-h-9 min-w-0 items-center justify-between gap-2 px-2 py-0.5">
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
								language={language}
								diffText={value}
								isBusy={diffControlsDisabled}
								candidatePaths={diffCandidatePaths}
								workspaceRoots={diffWorkspaceRoots}
								autoReviewEpoch={autoReviewEpoch}
							/>
						) : null}
					</div>
					{!disableControls ? (
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
					) : null}
				</div>

				{isExpanded && (
					<div id={codeBodyId} className="app-text-code p-1" style={{ fontSize: 14, lineHeight: 1.5 }}>
						{showFallback ? (
							fallback
						) : (
							<div className="app-shiki-container max-w-full overflow-x-auto">
								<ShikiHighlightedCode html={highlightedHtml} />
							</div>
						)}
					</div>
				)}
			</div>

			{isMermaid && !isBusy && !disableControls && richCodeWorkActivated ? (
				<MermaidDiagram code={value} onRenderStatusChange={handleMermaidRenderStatusChange} />
			) : null}
		</>
	);
}
