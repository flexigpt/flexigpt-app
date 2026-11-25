import { CustomMDLanguage } from '@/components/markdown/custom_md_utils';
import { looksLikeInteractiveDiff } from '@/components/markdown/diff_apply_model';

export interface MessageDiffBlockSource {
	id: string;
	language: string;
	diffText: string;
}

interface MarkdownDiffNode {
	type: string;
	tagName?: string;
	value?: string;
	properties?: {
		className?: string | string[];
	};
	data?: Record<string, unknown>;
	children?: MarkdownDiffNode[];
}

function getNodeText(node: MarkdownDiffNode): string {
	if (node.type === 'text') {
		return node.value ?? '';
	}

	return (node.children ?? []).map(child => getNodeText(child)).join('');
}

/**
 * Runs after the normal Markdown transforms and sanitization.
 *
 * The extraction matches MarkdownPreRenderer:
 * - use the first element child of pre
 * - read its language class
 * - remove the single newline added by react-markdown
 */
export function createMessageDiffInventoryPlugin(
	publish: (sources: MessageDiffBlockSource[]) => void
): () => (tree: MarkdownDiffNode) => void {
	return () => tree => {
		const sources: MessageDiffBlockSource[] = [];

		const visit = (node: MarkdownDiffNode): void => {
			if (node.type === 'element' && node.tagName === 'pre') {
				const child = node.children?.find(candidate => candidate.type === 'element');

				if (child) {
					const classes = child.properties?.className;
					const className = Array.isArray(classes) ? classes.join(' ') : (classes ?? '');
					const language = /(?:language-|lang-)([^\s]+)/.exec(className)?.[1] ?? 'text';
					const diffText = getNodeText(child).replace(/\n$/, '');

					const isThinking =
						language === (CustomMDLanguage.Thinking as string) ||
						language === (CustomMDLanguage.ThinkingSummary as string);

					if (!isThinking && looksLikeInteractiveDiff(diffText, language)) {
						const id = `message-diff-${sources.length + 1}`;
						node.data = { ...node.data, messageDiffID: id };
						sources.push({ id, language, diffText });
					}
				}

				// A preformatted block is one code-block occurrence.
				return;
			}

			for (const child of node.children ?? []) {
				visit(child);
			}
		};

		visit(tree);
		publish(sources);
	};
}
