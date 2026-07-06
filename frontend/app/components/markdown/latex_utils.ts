// LaTeX processing function
const testLatexRegex = /[$\\]/;
const containsLatexRegex =
	/\\\([\s\S]*?\\\)|\\\[[\s\S]*?\\\]|\$[\s\S]*?\$|\\begin\{equation\}[\s\S]*?\\end\{equation\}/;
const inlineLatex = /\\\(([\s\S]+?)\\\)/g;
const blockLatex = /\\\[([\s\S]+?)\\\]/g;

// oxlint-disable-next-line typescript/consistent-type-definitions
type MarkdownAstNode = {
	type?: string;
	value?: unknown;
	lang?: unknown;
	position?: unknown;
	children?: MarkdownAstNode[];
	data?: {
		hName?: string;
		hProperties?: {
			className?: string[];
		};
		hChildren?: Array<{ type: 'text'; value: string }>;
	};
};

type RemarkTransformer = (tree: MarkdownAstNode) => void;

interface InlineCodeMathOptions {
	allowPlainVariables?: boolean;
}

const MATH_INLINE_CODE_MAX_LENGTH = 240;

const inlineCodeMathCommands = [
	'alpha',
	'beta',
	'gamma',
	'delta',
	'epsilon',
	'varepsilon',
	'zeta',
	'eta',
	'theta',
	'vartheta',
	'iota',
	'kappa',
	'lambda',
	'mu',
	'nu',
	'xi',
	'pi',
	'varpi',
	'rho',
	'varrho',
	'sigma',
	'varsigma',
	'tau',
	'upsilon',
	'phi',
	'varphi',
	'chi',
	'psi',
	'omega',
	'Gamma',
	'Delta',
	'Theta',
	'Lambda',
	'Xi',
	'Pi',
	'Sigma',
	'Upsilon',
	'Phi',
	'Psi',
	'Omega',
	'nabla',
	'partial',
	'frac',
	'dfrac',
	'tfrac',
	'sqrt',
	'sum',
	'prod',
	'int',
	'oint',
	'lim',
	'sin',
	'cos',
	'tan',
	'cot',
	'sec',
	'csc',
	'log',
	'ln',
	'exp',
	'min',
	'max',
	'arg',
	'cdot',
	'times',
	'div',
	'pm',
	'mp',
	'le',
	'leq',
	'ge',
	'geq',
	'ne',
	'neq',
	'approx',
	'sim',
	'simeq',
	'propto',
	'infty',
	'in',
	'notin',
	'subset',
	'subseteq',
	'supset',
	'supseteq',
	'cup',
	'cap',
	'forall',
	'exists',
	'neg',
	'land',
	'lor',
	'to',
	'rightarrow',
	'leftarrow',
	'leftrightarrow',
	'Rightarrow',
	'Leftarrow',
	'Leftrightarrow',
	'mathbf',
	'mathrm',
	'mathit',
	'mathsf',
	'mathtt',
	'mathcal',
	'mathbb',
	'boldsymbol',
	'vec',
	'hat',
	'bar',
	'tilde',
	'dot',
	'ddot',
	'overline',
	'underline',
	'left',
	'right',
	'begin',
	'end',
	'text',
	'ldots',
	'cdots',
	'dots',
];

const inlineCodeMathCommandRegex = new RegExp(`\\\\(?:${inlineCodeMathCommands.join('|')})\\b`);
const inlineCodePlainMathVariableRegex = /^[A-Za-z](?:(?:[_^]\{?[A-Za-z0-9]+\}?))*$/;
const inlineCodeNonMathRegex =
	/^(?:https?:\/\/|www\.|[A-Za-z]:[\\/]|\.{1,2}[\\/]|\/[\w.-]+|[A-Za-z_$][\w$]*\([^)]*\)|(?:const|let|var|return|import|export|class|function|interface|type|npm|yarn|pnpm|git|curl)\b)/i;
const skipInlineCodeMathChildTypes = new Set(['code', 'math', 'inlineMath']);

function trimMathDelimiterPadding(equation: string): string {
	return equation.trim();
}

function renderDisplayMathFence(equation: string): string {
	const trimmed = trimMathDelimiterPadding(equation);
	return trimmed ? `\n$$\n${trimmed}\n$$\n` : '';
}

function normalizeInlineCodeMath(value: string): string {
	const trimmed = trimMathDelimiterPadding(value);
	const inlineMatch = /^\\\(([\s\S]*)\\\)$/.exec(trimmed);
	const displayMatch = /^\\\[([\s\S]*)\\\]$/.exec(trimmed);
	const dollarMatch = /^\$([\s\S]*)\$$/.exec(trimmed);

	return trimMathDelimiterPadding(inlineMatch?.[1] ?? displayMatch?.[1] ?? dollarMatch?.[1] ?? trimmed);
}

function createInlineMathNodeFromInlineCode(node: MarkdownAstNode): MarkdownAstNode | undefined {
	if (typeof node.value !== 'string') {
		return undefined;
	}

	const value = normalizeInlineCodeMath(node.value);
	return {
		type: 'inlineMath',
		value,
		position: node.position,
		data: getInlineMathHastData(value),
	};
}

function looksLikeNonMathInlineCode(value: string): boolean {
	return inlineCodeNonMathRegex.test(value) || value.includes('://') || value.includes('`') || value.startsWith('\\\\');
}

function isLikelyInlineCodeMath(value: string, options?: InlineCodeMathOptions): boolean {
	const trimmed = value.trim();
	if (!trimmed || trimmed.length > MATH_INLINE_CODE_MAX_LENGTH || trimmed.includes('\n')) {
		return false;
	}

	const normalized = normalizeInlineCodeMath(trimmed);
	if (!normalized || looksLikeNonMathInlineCode(trimmed) || looksLikeNonMathInlineCode(normalized)) {
		return false;
	}

	if (inlineCodeMathCommandRegex.test(normalized)) {
		return true;
	}

	return options?.allowPlainVariables === true && inlineCodePlainMathVariableRegex.test(normalized);
}

function isMathContextNode(node: MarkdownAstNode): boolean {
	if (node.type === 'math' || node.type === 'inlineMath') {
		return true;
	}

	if (node.type === 'code' && typeof node.lang === 'string' && node.lang.trim().toLowerCase() === 'math') {
		return true;
	}

	return node.type === 'inlineCode' && typeof node.value === 'string'
		? isLikelyInlineCodeMath(node.value, { allowPlainVariables: false })
		: false;
}

function hasMathContext(node: MarkdownAstNode): boolean {
	if (isMathContextNode(node)) {
		return true;
	}

	return node.children?.some(child => hasMathContext(child)) ?? false;
}

function transformInlineCodeMath(node: MarkdownAstNode, allowPlainVariables: boolean): void {
	const children = node.children;
	if (!children) {
		return;
	}

	for (let index = 0; index < children.length; index += 1) {
		const child = children[index];

		if (child.type === 'inlineCode' && typeof child.value === 'string') {
			if (isLikelyInlineCodeMath(child.value, { allowPlainVariables })) {
				const inlineMathNode = createInlineMathNodeFromInlineCode(child);
				if (inlineMathNode) {
					children[index] = inlineMathNode;
				}
			}
			continue;
		}

		if (!skipInlineCodeMathChildTypes.has(child.type ?? '')) {
			transformInlineCodeMath(child, allowPlainVariables);
		}
	}
}

export function remarkInlineCodeMath(): RemarkTransformer {
	return tree => {
		const allowPlainVariables = hasMathContext(tree);
		transformInlineCodeMath(tree, allowPlainVariables);
	};
}

function getInlineMathHastData(value: string): NonNullable<MarkdownAstNode['data']> {
	return {
		hName: 'code',
		hProperties: {
			className: ['language-math', 'math-inline'],
		},
		hChildren: [{ type: 'text', value }],
	};
}

function SanitizeLaTeX(content: string) {
	if (!testLatexRegex.test(content)) {
		return content;
	}
	let processedContent = content.replaceAll(/(\$)(?=\s?\d)/g, '\\$');

	if (!containsLatexRegex.test(processedContent)) {
		return processedContent;
	}

	processedContent = processedContent
		.replace(inlineLatex, (match: string, equation: string) => {
			const trimmed = trimMathDelimiterPadding(equation);
			return trimmed ? `$${trimmed}$` : match;
		})
		.replace(blockLatex, (match: string, equation: string) => renderDisplayMathFence(equation) || match);

	return processedContent;
}

export function SanitizeLaTeXOutsideFences(md: string) {
	if (!testLatexRegex.test(md)) {
		return md;
	}

	const lines = md.split('\n');
	const out: string[] = [];
	let outside: string[] = [];
	let fence: string | null = null; // e.g. ``` or ~~~ (or longer)

	const flushOutside = () => {
		if (outside.length === 0) {
			return;
		}
		out.push(SanitizeLaTeX(outside.join('\n')));
		outside = [];
	};

	for (const line of lines) {
		const m = line.match(/^([`~]{3,})/);
		if (!fence) {
			if (m) {
				flushOutside();
				fence = m[1];
				out.push(line);
			} else {
				outside.push(line);
			}
		} else {
			out.push(line);
			if (line.startsWith(fence)) {
				fence = null;
			}
		}
	}
	flushOutside();
	return out.join('\n');
}
