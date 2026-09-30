// oxlint-disable react/no-react-children
import type { HTMLAttributes, ReactNode } from 'react';
import type { ExtraProps } from 'react-markdown';
import { Children, isValidElement, useContext } from 'react';

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
	inline: _inline,
	className,
	children,
	...props
}: MarkdownCodeRendererProps) {
	return (
		<code {...props} className={`bg-base-200 inline text-wrap wrap-break-word whitespace-pre-wrap ${className ?? ''}`}>
			{children}
		</code>
	);
}

function getTextContent(children: ReactNode): string {
	const parts: string[] = [];
	Children.forEach(children, child => {
		if (typeof child === 'string' || typeof child === 'number') {
			parts.push(String(child));
		} else if (isValidElement<{ children?: ReactNode }>(child)) {
			parts.push(getTextContent(child.props.children));
		}
	});
	return parts.join('');
}

export function MarkdownPreRenderer({ node: _node, children, ...props }: HTMLAttributes<HTMLPreElement> & ExtraProps) {
	const settings = useContext(MarkdownCodeRendererContext);
	const child = Children.toArray(children).find(value => isValidElement(value));

	if (!isValidElement<{ className?: string; children?: ReactNode }>(child)) {
		return <pre {...props}>{children}</pre>;
	}

	const className = child.props.className ?? '';
	const match = /(?:language-|lang-)([^\s]+)/.exec(className);
	const language = match?.[1] ?? 'text';
	// react-markdown adds one terminal newline to a fenced code node.
	// Do not otherwise rewrite patch line endings or whitespace here.
	const value = getTextContent(child.props.children).replace(/\n$/, '');

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
		/>
	);
}
