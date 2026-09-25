import type { HTMLAttributes, ReactNode } from 'react';
import type { ExtraProps } from 'react-markdown';
import { useContext } from 'react';

import { CustomMDLanguage } from '@/components/markdown/custom_md_utils';
import { CodeBlock } from '@/components/markdown/markdown_code_block';
import { MarkdownCodeRendererContext } from '@/components/markdown/markdown_code_renderer_context';
import { ThinkingFence } from '@/components/markdown/thinking_fence';

interface MarkdownCodeRendererProps extends HTMLAttributes<HTMLElement>, ExtraProps {
	inline?: boolean;
	className?: string;
	children?: ReactNode;
}

export function MarkdownCodeRenderer({
	node: _node,
	inline,
	className,
	children,
	...props
}: MarkdownCodeRendererProps) {
	const settings = useContext(MarkdownCodeRendererContext);

	if (inline || !className) {
		return (
			<code
				{...props}
				className={`bg-base-200 inline text-wrap wrap-break-word whitespace-pre-wrap ${className ?? ''}`}
			>
				{children}
			</code>
		);
	}

	const match = /lang-(\w+)/.exec(className) || /language-(\w+)/.exec(className);
	const language = match?.[1] ?? 'text';

	const raw =
		typeof children === 'string'
			? children
			: Array.isArray(children)
				? children.join('')
				: children === null
					? ''
					: // oxlint-disable-next-line typescript/no-base-to-string
						String(children);

	const value = raw.replaceAll('\r\n', '\n').replace(/\n$/, '');

	if (language === (CustomMDLanguage.ThinkingSummary as string)) {
		return (
			<ThinkingFence
				detailsSummary={<span>Thinking Summary</span>}
				text={value}
				defaultOpen={settings.isBusy}
				streaming={settings.isBusy}
			/>
		);
	}

	if (language === (CustomMDLanguage.Thinking as string)) {
		return (
			<ThinkingFence
				detailsSummary={<span>Thinking</span>}
				text={value}
				defaultOpen={settings.isBusy}
				streaming={settings.isBusy}
			/>
		);
	}

	return (
		<CodeBlock
			language={language}
			value={value}
			isBusy={settings.isBusy}
			hideMermaidCode={settings.hideMermaidCode}
			diffCandidatePaths={settings.diffCandidatePaths}
			diffWorkspaceRoots={settings.diffWorkspaceRoots}
			defaultExpanded={settings.defaultCodeBlockExpanded}
			disableControls={settings.isBusy}
			autoReviewEpoch={settings.autoReviewEpoch}
		/>
	);
}
