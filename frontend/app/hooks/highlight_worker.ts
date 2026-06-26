/// <reference lib="webworker" />
import { bundledLanguages, createHighlighter } from 'shiki';
import type { Highlighter } from 'shiki';

interface Request {
	id: number;
	code: string;
	lang: string;
}
interface Response {
	id: number;
	html?: string;
	error?: string;
}

export const baseLangs = [
	'bash',
	'c',
	'c#',
	'c++',
	'cpp',
	'css',
	'dart',
	'go',
	'html',
	'java',
	'javascript',
	'js',
	'jsx',
	'json',
	'jsonc',
	'kotlin',
	'lua',
	'markdown',
	'mermaid',
	'php',
	'powershell',
	'py',
	'python',
	'r',
	'ruby',
	'rust',
	'scala',
	'shell',
	'sql',
	'swift',
	'text',
	'ts',
	'tsx',
	'typescript',
	'vue',
	'xml',
	'yaml',
	'yml',
];

// Singleton highlighter -- store the PROMISE so we never create two.
let highlighterPromise: Promise<Highlighter> | undefined;

async function init(): Promise<Highlighter> {
	if (!highlighterPromise) {
		highlighterPromise = createHighlighter({
			themes: ['monokai'],
			langs: baseLangs,
		});
	}
	return highlighterPromise;
}

// Kick it off right now so grammars & wasm load while the UI idles.
// oxlint-disable-next-line unicorn/prefer-top-level-await
init().catch(console.error);

// Message handler.
self.onmessage = async (evt: MessageEvent<Request>) => {
	const { id, code, lang } = evt.data;
	const msg: Response = { id };

	try {
		const hl = await init();

		const known = lang in bundledLanguages || hl.getLoadedLanguages().includes(lang as any);

		if (!known) {
			msg.html = hl.codeToHtml(code, { lang: 'text', theme: 'monokai' });
		} else {
			// Load on demand - Shiki caches internally.
			if (!hl.getLoadedLanguages().includes(lang as any)) {
				await hl.loadLanguage(lang as any);
			}

			msg.html = hl.codeToHtml(code, { lang, theme: 'monokai' });
		}
	} catch (e: unknown) {
		msg.error = String(e);
	}

	postMessage(msg);
};
