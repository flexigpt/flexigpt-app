// LaTeX processing function
const testLatexRegex = /[$\\]/;
const containsLatexRegex = /\\\(.*?\\\)|\\\[.*?\\\]|\$.*?\$|\\begin\{equation\}.*?\\end\{equation\}/;
const inlineLatex = new RegExp(/\\\((.+?)\\\)/, 'g');
const blockLatex = new RegExp(/\\\[(.*?[^\\])\\\]/, 'gs');

function SanitizeLaTeX(content: string) {
	if (!testLatexRegex.test(content)) {
		return content;
	}
	let processedContent = content.replace(/(\$)(?=\s?\d)/g, '\\$');

	if (!containsLatexRegex.test(processedContent)) {
		return processedContent;
	}

	processedContent = processedContent
		.replace(inlineLatex, (match: string, equation: string) => `$${equation}$`)
		.replace(blockLatex, (match: string, equation: string) => `$$${equation}$$`);

	return processedContent;
}

export function SanitizeLaTeXOutsideFences(md: string) {
	if (!testLatexRegex.test(md)) return md;

	const lines = md.split('\n');
	const out: string[] = [];
	let outside: string[] = [];
	let fence: string | null = null; // e.g. ``` or ~~~ (or longer)

	const flushOutside = () => {
		if (outside.length === 0) return;
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
