import type { ReactNode } from 'react';
import { memo, useMemo } from 'react';

import 'katex/dist/katex.min.css';
import Markdown from 'react-markdown';
import rehypeKatex from 'rehype-katex';
import rehypeRaw from 'rehype-raw';
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize';
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
	...defaultSchema, // keep ancestors / clobber / prefix logic
	attributes: {
		...defaultSchema.attributes,
		// The `language-*` regex is allowed by default.
		code: [['className', /^language-./, /^math-./]],
		input: defaultSchema.attributes?.input.filter(a => a !== 'value' && a !== 'checked'),
	},
};

interface EnhancedMarkdownProps {
	text: string;
	align?: string;
	isBusy?: boolean;
}

interface CodeComponentProps {
	inline?: boolean;
	className?: string;
	children?: ReactNode;
}

interface CustomComponentProps {
	className?: string;
	children?: ReactNode;
}

interface RefComponentProps {
	href?: string;
	className?: string;
	children?: ReactNode;
}

export const EnhancedMarkdown = memo(function EnhancedMarkdown({
	text,
	align = 'left',
	isBusy = false,
}: EnhancedMarkdownProps) {
	const processedText = useMemo(() => {
		// During a stream skip LaTeX sanitisation for speed.
		return isBusy ? text : SanitizeLaTeXOutsideFences(text);
	}, [text, isBusy]);

	const components = useMemo(
		() => ({
			h1: ({ children, className, ...rest }: CustomComponentProps) => (
				<h1 {...rest} className={`my-2 text-xl font-bold ${className ?? ''}`}>
					{children}
				</h1>
			),
			h2: ({ children, className, ...rest }: CustomComponentProps) => (
				<h2 {...rest} className={`my-2 text-lg font-bold ${className ?? ''}`}>
					{children}
				</h2>
			),
			h3: ({ children, className, ...rest }: CustomComponentProps) => (
				<h3 {...rest} className={`my-2 text-base font-bold ${className ?? ''}`}>
					{children}
				</h3>
			),

			ul: ({ children, className, ...rest }: CustomComponentProps) => (
				<ul {...rest} className={`list-disc py-1 pl-4 ${className ?? ''}`}>
					{children}
				</ul>
			),
			ol: ({ children, className, ...rest }: CustomComponentProps) => (
				// Important: keep {...rest} so `start`, `reversed`, `type` are preserved
				<ol {...rest} className={`list-decimal py-1 pl-4 ${className ?? ''}`}>
					{children}
				</ol>
			),
			li: ({ children, className, ...rest }: CustomComponentProps) => (
				<li {...rest} className={`py-1 ${className ?? ''}`}>
					{children}
				</li>
			),

			table: ({ children, className, ...rest }: CustomComponentProps) => (
				<div className="my-2 max-w-full overflow-x-auto" style={{ contain: 'layout paint' }}>
					<table {...rest} className={`min-w-max table-fixed ${className ?? ''}`}>
						{children}
					</table>
				</div>
			),
			thead: ({ children, className, ...rest }: CustomComponentProps) => (
				<thead {...rest} className={`bg-base-300 ${className ?? ''}`}>
					{children}
				</thead>
			),
			tbody: ({ children, className, ...rest }: CustomComponentProps) => (
				<tbody {...rest} className={className ?? ''}>
					{children}
				</tbody>
			),

			tr: ({ children, className, ...rest }: CustomComponentProps) => (
				<tr {...rest} className={`border-t ${className ?? ''}`}>
					{children}
				</tr>
			),
			th: ({ children, className, ...rest }: CustomComponentProps) => (
				<th {...rest} className={`px-4 py-2 text-left ${className ?? ''}`}>
					{children}
				</th>
			),
			td: ({ children, className, ...rest }: CustomComponentProps) => (
				<td {...rest} className={`px-4 py-2 ${className ?? ''}`}>
					{children}
				</td>
			),

			p: ({ className, children, ...rest }: CustomComponentProps) => (
				<p
					{...rest}
					className={`${className ?? ''} my-2 ${align} wrap-break-word`}
					style={{ lineHeight: '1.5', fontSize: '14px' }}
				>
					{children}
				</p>
			),

			blockquote: ({ children, className, ...rest }: CustomComponentProps) => (
				<blockquote {...rest} className={`border-neutral/20 border-l-4 pl-4 italic ${className ?? ''}`}>
					{children}
				</blockquote>
			),

			a: ({ href, children, className, ...rest }: RefComponentProps) => (
				<a
					{...rest}
					href={href}
					target={href?.startsWith('http') ? '_blank' : undefined}
					rel={href?.startsWith('http') ? 'noopener noreferrer' : undefined}
					className={`cursor-pointer text-blue-600 underline hover:text-blue-800 ${className ?? ''}`}
					onClick={e => {
						e.preventDefault();
						if (href) backendAPI.openURL(href);
					}}
				>
					{children}
				</a>
			),

			code: ({ inline, className, children, ...props }: CodeComponentProps) => {
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

				// react-markdown may pass children as string[]; String(array) inserts commas -> breaks Mermaid.
				const raw =
					typeof children === 'string'
						? children
						: Array.isArray(children)
							? children.join('')
							: children == null
								? ''
								: // eslint-disable-next-line @typescript-eslint/no-base-to-string
									String(children);

				// Normalize newlines and trim ONE trailing newline from fenced blocks
				const value = raw.replace(/\r\n/g, '\n').replace(/\n$/, '');

				if (language === (CustomMDLanguage.ThinkingSummary as string)) {
					return <ThinkingFence detailsSummary={<span>Thinking Summary</span>} text={value} />;
				} else if (language === (CustomMDLanguage.Thinking as string)) {
					return <ThinkingFence detailsSummary={<span>Thinking</span>} text={value} />;
				} else {
					return <CodeBlock language={language} value={value} isBusy={isBusy} />;
				}
			},
		}),
		[align, isBusy]
	);

	return (
		<MdErrorBoundary source={processedText}>
			<Markdown
				remarkPlugins={[remarkGfm, remarkMath, supersub, remarkGemoji]}
				rehypePlugins={[rehypeRaw, [rehypeSanitize, { ...strictSchema }], rehypeKatex]}
				components={components}
				skipHtml={false}
			>
				{processedText}
			</Markdown>
		</MdErrorBoundary>
	);
});
