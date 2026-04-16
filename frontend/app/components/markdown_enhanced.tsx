/* eslint-disable @typescript-eslint/no-unused-vars */
import type { AnchorHTMLAttributes, HTMLAttributes, MouseEvent as ReactMouseEvent, ReactNode } from 'react';
import { memo, useMemo } from 'react';

import { FiExternalLink } from 'react-icons/fi';

import 'katex/dist/katex.min.css';
import type { ExtraProps } from 'react-markdown';
import Markdown from 'react-markdown';
import rehypeKatex from 'rehype-katex';
import rehypeRaw from 'rehype-raw';
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize';
import rehypeSlug from 'rehype-slug';
import remarkGemoji from 'remark-gemoji';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import supersub from 'remark-supersub';

import { SanitizeLaTeXOutsideFences } from '@/lib/markdown_utils';
import { CustomMDLanguage } from '@/lib/text_utils';

import { backendAPI } from '@/apis/baseapi';

import { CodeBlock } from '@/components/markdown_code_block';
import { MdErrorBoundary } from '@/components/markdown_error_boundary';
import { ThinkingFence } from '@/components/thinking_fence';

const strictSchema = {
	...defaultSchema,
	attributes: {
		...defaultSchema.attributes,
		code: [['className', /^language-./, /^math-./]],
		input: defaultSchema.attributes?.input.filter(a => a !== 'value' && a !== 'checked'),
	},
};

interface CodeComponentProps extends HTMLAttributes<HTMLElement>, ExtraProps {
	inline?: boolean;
	className?: string;
	children?: ReactNode;
}

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
	onLinkClick?: (href: string, event: ReactMouseEvent<HTMLAnchorElement>) => boolean;
}

const isExternalHref = (href?: string) => !!href && /^(https?:)?\/\/|^mailto:|^tel:/i.test(href);

export const EnhancedMarkdown = memo(function EnhancedMarkdown({
	text,
	align = 'left',
	isBusy = false,
	hideMermaidCode = false,
	hideH1Title = false,
	onLinkClick,
}: EnhancedMarkdownProps) {
	const processedText = useMemo(() => {
		return isBusy ? text : SanitizeLaTeXOutsideFences(text);
	}, [text, isBusy]);

	const components = useMemo(() => {
		const renderHeading =
			(tag: 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6', baseClassName: string, options?: { hide?: boolean }) =>
			({ node, children, className, id, ...rest }: CustomComponentProps) => {
				if (options?.hide) {
					return id ? <div id={id} className="scroll-mt-4" aria-hidden="true" /> : null;
				}

				const HeadingTag = tag;

				return (
					<HeadingTag {...rest} id={id} className={`${baseClassName} scroll-mt-4 ${className ?? ''}`.trim()}>
						{children}
					</HeadingTag>
				);
			};

		return {
			h1: renderHeading('h1', 'my-2 pt-2 text-xl font-bold', { hide: hideH1Title }),
			h2: renderHeading('h2', 'my-2 pt-2 text-lg font-bold'),
			h3: renderHeading('h3', 'my-1 pt-2 text-base font-semibold'),
			h4: renderHeading('h4', 'my-1 pt-1 text-sm font-semibold'),
			h5: renderHeading('h5', 'my-1 pt-1 text-sm font-semibold'),
			h6: renderHeading('h6', 'my-1 pt-1 text-xs font-semibold uppercase tracking-wide'),

			ul: ({ node, children, className, ...rest }: CustomComponentProps) => (
				<ul {...rest} className={`ml-4 list-disc py-1.5 pl-2 ${className ?? ''}`}>
					{children}
				</ul>
			),

			ol: ({ node, children, className, ...rest }: CustomComponentProps) => (
				<ol {...rest} className={`ml-4 list-decimal py-1.5 pl-2 ${className ?? ''}`}>
					{children}
				</ol>
			),

			li: ({ node, children, className, ...rest }: CustomComponentProps) => (
				<li {...rest} className={`p-0.5 ${className ?? ''}`}>
					{children}
				</li>
			),

			table: ({ node, children, className, ...rest }: CustomComponentProps) => (
				<div className="my-2 max-w-full overflow-x-auto p-2" style={{ contain: 'layout paint' }}>
					<table {...rest} className={`min-w-max table-fixed ${className ?? ''}`}>
						{children}
					</table>
				</div>
			),

			thead: ({ node, children, className, ...rest }: CustomComponentProps) => (
				<thead {...rest} className={`bg-base-300 ${className ?? ''}`}>
					{children}
				</thead>
			),

			tbody: ({ node, children, className, ...rest }: CustomComponentProps) => (
				<tbody {...rest} className={className ?? ''}>
					{children}
				</tbody>
			),

			tr: ({ node, children, className, ...rest }: CustomComponentProps) => (
				<tr {...rest} className={`border-t ${className ?? ''}`}>
					{children}
				</tr>
			),

			th: ({ node, children, className, ...rest }: CustomComponentProps) => (
				<th {...rest} className={`px-4 py-2 text-left ${className ?? ''}`}>
					{children}
				</th>
			),

			td: ({ node, children, className, ...rest }: CustomComponentProps) => (
				<td {...rest} className={`px-4 py-2 ${className ?? ''}`}>
					{children}
				</td>
			),

			p: ({ node, className, children, ...rest }: CustomComponentProps) => (
				<p
					{...rest}
					className={`${className ?? ''} my-1 ${align} wrap-break-word`}
					style={{ lineHeight: '2', fontSize: '14px' }}
				>
					{children}
				</p>
			),

			blockquote: ({ node, children, className, ...rest }: CustomComponentProps) => (
				<blockquote {...rest} className={`border-neutral/20 border-l-4 pl-4 italic ${className ?? ''}`}>
					{children}
				</blockquote>
			),

			a: ({ node, href, children, className, ...rest }: RefComponentProps) => {
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

							e.preventDefault();
							backendAPI.openURL(href);
						}}
					>
						{children}
						{isExternal && <FiExternalLink aria-hidden="true" size="0.9em" className="ml-1 inline align-[-0.1em]" />}
					</a>
				);
			},

			code: ({ node, inline, className, children, ...props }: CodeComponentProps) => {
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

				const match = /lang-(\w+)/.exec(className || '') || /language-(\w+)/.exec(className || '');
				const language = match && match[1] ? match[1] : 'text';

				const raw =
					typeof children === 'string'
						? children
						: Array.isArray(children)
							? children.join('')
							: children == null
								? ''
								: // eslint-disable-next-line @typescript-eslint/no-base-to-string
									String(children);

				const value = raw.replace(/\r\n/g, '\n').replace(/\n$/, '');

				if (language === (CustomMDLanguage.ThinkingSummary as string)) {
					return <ThinkingFence detailsSummary={<span>Thinking Summary</span>} text={value} />;
				}

				if (language === (CustomMDLanguage.Thinking as string)) {
					return <ThinkingFence detailsSummary={<span>Thinking</span>} text={value} />;
				}

				return <CodeBlock language={language} value={value} isBusy={isBusy} hideMermaidCode={hideMermaidCode} />;
			},
		};
	}, [align, hideH1Title, hideMermaidCode, isBusy, onLinkClick]);

	return (
		<MdErrorBoundary source={processedText}>
			<Markdown
				remarkPlugins={[remarkGfm, remarkMath, supersub, remarkGemoji]}
				rehypePlugins={[rehypeRaw, [rehypeSanitize, { ...strictSchema }], rehypeSlug, rehypeKatex]}
				components={components}
				skipHtml={false}
			>
				{processedText}
			</Markdown>
		</MdErrorBoundary>
	);
});
