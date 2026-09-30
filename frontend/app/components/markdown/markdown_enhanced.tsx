// oxlint-disable unicorn/consistent-function-scoping func-name-matching
import type { AnchorHTMLAttributes, HTMLAttributes, MouseEvent as ReactMouseEvent, ReactNode } from 'react';
import { memo, useContext, useMemo } from 'react';
import { FiExternalLink } from 'react-icons/fi';

// oxlint-disable-next-line import/no-unassigned-import
import 'katex/dist/katex.min.css';

import type { Components, ExtraProps } from 'react-markdown';

import type { PluggableList } from 'unified';
import Markdown from 'react-markdown';
import rehypeKatex from 'rehype-katex';
import rehypeRaw from 'rehype-raw';
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize';
import rehypeSlug from 'rehype-slug';
import remarkGemoji from 'remark-gemoji';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import supersub from 'remark-supersub';

import { backendAPI } from '@/apis/baseapi';

import { remarkInlineCodeMath, sanitizeLaTeXOutsideFences } from '@/components/markdown/latex_utils';
import { MarkdownCodeRenderer, MarkdownPreRenderer } from '@/components/markdown/markdown_code_renderer';
import { MarkdownCodeRendererContext } from '@/components/markdown/markdown_code_renderer_context';
import { MdErrorBoundary } from '@/components/markdown/markdown_error_boundary';
import { MarkdownPresentationContext } from '@/components/markdown/markdown_presentation_context';
import { MarkdownTable } from '@/components/markdown/markdown_table';

const strictSchema = {
	...defaultSchema,
	attributes: {
		...defaultSchema.attributes,
		code: [['className', /^language-./, /^math-./]],
		input: defaultSchema.attributes?.input.filter(a => a !== 'value'),
	},
};

const streamingRemarkPlugins: PluggableList = [remarkGfm, supersub, remarkGemoji];
const richRemarkPlugins: PluggableList = [remarkGfm, remarkMath, remarkInlineCodeMath, supersub, remarkGemoji];

const rehypeKatexOptions = {
	throwOnError: false,
};

const streamingRehypePlugins: PluggableList = [rehypeRaw, [rehypeSanitize, strictSchema], rehypeSlug];
const richRehypePlugins: PluggableList = [
	rehypeRaw,
	[rehypeSanitize, strictSchema],
	rehypeSlug,
	[rehypeKatex, rehypeKatexOptions] as const,
];

interface CustomComponentProps extends HTMLAttributes<HTMLElement>, ExtraProps {
	className?: string;
	children?: ReactNode;
	id?: string;
}

interface RefComponentProps extends AnchorHTMLAttributes<HTMLAnchorElement>, ExtraProps {
	href?: string;
	className?: string;
	children?: ReactNode;
}

interface EnhancedMarkdownProps {
	text: string;
	align?: string;
	isBusy?: boolean;
	hideMermaidCode?: boolean;
	hideH1Title?: boolean;
	diffCandidatePaths?: string[];
	diffWorkspaceRoots?: string[];
	defaultCodeBlockExpanded?: boolean;
	onLinkClick?: (href: string, event: ReactMouseEvent<HTMLAnchorElement>) => boolean;
}

const isExternalHref = (href?: string) => !!href && /^(https?:)?\/\/|^mailto:|^tel:/i.test(href);

const renderHeading = (
	tag: 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6',
	baseClassName: string,
	options?: { hide?: boolean }
) =>
	function MarkdownHeading({ node: _node, children, className, id, ...rest }: CustomComponentProps) {
		const { hideH1Title } = useContext(MarkdownPresentationContext);
		if (options?.hide && hideH1Title) {
			return id ? <div id={id} className="scroll-mt-4" aria-hidden="true" /> : null;
		}

		const HeadingTag = tag;

		return (
			<HeadingTag {...rest} id={id} className={`${baseClassName} scroll-mt-4 ${className ?? ''}`.trim()}>
				{children}
			</HeadingTag>
		);
	};

function createMarkdownComponents(): Components {
	return {
		h1: renderHeading('h1', 'my-2 pt-2 text-xl font-bold', { hide: true }),
		h2: renderHeading('h2', 'my-2 pt-2 text-lg font-bold'),
		h3: renderHeading('h3', 'my-1 pt-2 text-base font-semibold'),
		h4: renderHeading('h4', 'my-1 pt-1 text-sm font-semibold'),
		h5: renderHeading('h5', 'my-1 pt-1 text-sm font-semibold'),
		h6: renderHeading('h6', 'my-1 pt-1 text-xs font-semibold uppercase tracking-wide'),

		ul: ({ node: _node, children, className, ...rest }: CustomComponentProps) => (
			<ul {...rest} className={`ml-4 list-disc py-1.5 pl-2 ${className ?? ''}`} style={{ fontSize: 14 }}>
				{children}
			</ul>
		),

		ol: ({ node: _node, children, className, ...rest }: CustomComponentProps) => (
			<ol {...rest} className={`ml-4 list-decimal py-1.5 pl-2 ${className ?? ''}`} style={{ fontSize: 14 }}>
				{children}
			</ol>
		),

		li: ({ node: _node, children, className, ...rest }: CustomComponentProps) => (
			<li {...rest} className={`p-0.5 ${className ?? ''}`} style={{ fontSize: 14 }}>
				{children}
			</li>
		),

		table: MarkdownTable,

		thead: ({ node: _node, children, className, ...rest }: CustomComponentProps) => (
			<thead {...rest} className={`bg-base-300 ${className ?? ''}`}>
				{children}
			</thead>
		),

		tbody: ({ node: _node, children, className, ...rest }: CustomComponentProps) => (
			<tbody {...rest} className={className ?? ''}>
				{children}
			</tbody>
		),

		tr: ({ node: _node, children, className, ...rest }: CustomComponentProps) => (
			<tr {...rest} className={`border-t ${className ?? ''}`}>
				{children}
			</tr>
		),

		th: ({ node: _node, children, className, ...rest }: CustomComponentProps) => (
			<th {...rest} className={`px-3 py-2 text-left align-top ${className ?? ''}`}>
				{children}
			</th>
		),

		td: ({ node: _node, children, className, ...rest }: CustomComponentProps) => (
			<td {...rest} className={`px-3 py-2 align-top ${className ?? ''}`}>
				{children}
			</td>
		),

		p: function MarkdownParagraph({ node: _node, className, children, ...rest }: CustomComponentProps) {
			const { align } = useContext(MarkdownPresentationContext);
			return (
				<p
					{...rest}
					className={`${className ?? ''} p-1 ${align} wrap-break-word`}
					style={{ lineHeight: 1.5, fontSize: 14 }}
				>
					{children}
				</p>
			);
		},

		blockquote: ({ node: _node, children, className, ...rest }: CustomComponentProps) => (
			<blockquote {...rest} className={`border-neutral/20 border-l-4 pl-4 italic ${className ?? ''}`}>
				{children}
			</blockquote>
		),

		a: function MarkdownLink({ node: _node, href, children, className, ...rest }: RefComponentProps) {
			const { onLinkClick } = useContext(MarkdownPresentationContext);
			const isExternal = isExternalHref(href);

			return (
				<a
					{...rest}
					href={href}
					target={isExternal ? '_blank' : undefined}
					rel={isExternal ? 'noopener noreferrer' : undefined}
					className={`cursor-pointer text-blue-600 hover:text-blue-800 ${className ?? ''}`}
					onClick={e => {
						if (!href) {
							e.preventDefault();
							return;
						}

						const handled = onLinkClick?.(href, e);
						if (handled) {
							e.preventDefault();
							return;
						}

						if (href.startsWith('#')) {
							return;
						}
						e.preventDefault();
						backendAPI.openURL(href);
					}}
				>
					{children}
					{isExternal && <FiExternalLink aria-hidden="true" size="0.9em" className="ml-1 inline align-[-0.1em]" />}
				</a>
			);
		},

		code: MarkdownCodeRenderer,
		pre: MarkdownPreRenderer,
	};
}

const markdownComponents = createMarkdownComponents();

const MarkdownDocument = memo(function MarkdownDocument({ text, isBusy }: { text: string; isBusy: boolean }) {
	return (
		<Markdown
			remarkPlugins={isBusy ? streamingRemarkPlugins : richRemarkPlugins}
			rehypePlugins={isBusy ? streamingRehypePlugins : richRehypePlugins}
			components={markdownComponents}
			skipHtml={false}
		>
			{text}
		</Markdown>
	);
});

export const EnhancedMarkdown = memo(function EnhancedMarkdown({
	text,
	align = 'left',
	isBusy = false,
	hideMermaidCode = false,
	hideH1Title = false,
	diffCandidatePaths,
	diffWorkspaceRoots,
	defaultCodeBlockExpanded = true,
	onLinkClick,
}: EnhancedMarkdownProps) {
	const processedText = useMemo(() => (isBusy ? text : sanitizeLaTeXOutsideFences(text)), [isBusy, text]);
	const presentation = useMemo(() => ({ align, hideH1Title, onLinkClick }), [align, hideH1Title, onLinkClick]);
	const codeRendererSettings = useMemo(
		() => ({ isBusy, hideMermaidCode, diffCandidatePaths, diffWorkspaceRoots, defaultCodeBlockExpanded }),
		[defaultCodeBlockExpanded, diffCandidatePaths, diffWorkspaceRoots, hideMermaidCode, isBusy]
	);

	return (
		<MdErrorBoundary source={processedText}>
			<MarkdownPresentationContext.Provider value={presentation}>
				<MarkdownCodeRendererContext.Provider value={codeRendererSettings}>
					<MarkdownDocument text={processedText} isBusy={isBusy} />
				</MarkdownCodeRendererContext.Provider>
			</MarkdownPresentationContext.Provider>
		</MdErrorBoundary>
	);
});
