import { useCallback, useId, useState } from 'react';

import { FiAlertTriangle, FiChevronDown, FiChevronUp } from 'react-icons/fi';

import { useHighlight } from '@/hooks/use_highlight';

import { CopyButton } from '@/components/copy_button';
import { DownloadButton } from '@/components/download_button';
import { DiffApplyControl } from '@/components/markdown/diff_apply_control';
import type { MermaidRenderStatus } from '@/components/markdown/mermaid_diagram_card';
import { MermaidDiagram } from '@/components/markdown/mermaid_diagram_card';

interface CodeProps {
	language: string;
	value: string;
	isBusy: boolean;
	hideMermaidCode: boolean;
	diffCandidatePaths?: string[];
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

export function CodeBlock({ language, value, isBusy, hideMermaidCode, diffCandidatePaths }: CodeProps) {
	const html = useHighlight(value, language);
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
	// - normal code: expanded
	// - valid/rendering Mermaid: collapsed
	// - errored Mermaid: expanded unless the caller requested hidden Mermaid code behavior
	const defaultIsExpanded = isMermaid ? hasMermaidSyntaxError && !hideMermaidCode : true;
	const isExpanded = expansionOverride?.key === codeBlockKey ? expansionOverride.isExpanded : defaultIsExpanded;

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

	const fetchValue = async () => value;

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
			<div className="app-bg-code my-4 overflow-hidden rounded-lg">
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

						{!isBusy ? (
							<DiffApplyControl
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

			{isMermaid && !isBusy && <MermaidDiagram code={value} onRenderStatusChange={handleMermaidRenderStatusChange} />}
		</>
	);
}
