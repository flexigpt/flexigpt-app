import { describe, expect, it } from 'vitest';

import { codeMessage } from '@/lib/__tests__/sample_code_message';
import type { TitleCandidate } from '@/lib/title_utils';
import { generateTitle } from '@/lib/title_utils';

const DEFAULT = 'New conversation';
const MAX_PREVIEW = 120; // truncate long log lines

/* ─── debug printer ──────────────────────────────────────────────── */
function debugPrint(label: string, input: string, cand: TitleCandidate) {
	const crop = (t: string) => (t.length > MAX_PREVIEW ? t.slice(0, MAX_PREVIEW) + ' ...' : t);
	console.log(
		`\n${label}\n` +
			`  input (${input.length}):  ${crop(input)}\n` +
			`  title (${cand.title.length}, score=${cand.score.toFixed(2)}):  ${cand.title}\n`
	);
}

/* ─── base behaviour ─────────────────────────────────────────────── */
describe('generateTitle - base cases', () => {
	it('falls back to default on empty input', () => {
		const cand = generateTitle('');
		debugPrint('Empty string', '', cand);
		expect(cand.title).toBe(DEFAULT);
	});

	it('keeps short single-sentence prompts (sentence-cased)', () => {
		const txt = 'how to install react?';
		const cand = generateTitle(txt);
		debugPrint('Short question', txt, cand);
		expect(cand.title).toBe('How to install react');
	});

	it('simple one', () => {
		const txt = 'Give a good fibonnaci code';
		const cand = generateTitle(txt);
		debugPrint('Short request', txt, cand);
		expect(cand.title).toBe('Give a good fibonnaci code');
	});
});

/* ─── rich English examples ──────────────────────────────────────── */
describe('generateTitle - rich English path', () => {
	it('picks the best question in a paragraph block', () => {
		const msg = `
Hello!
I keep getting "Cannot set headers after they are sent".
How can I resolve this error?
`;
		const cand = generateTitle(msg);
		debugPrint('Error paragraph', msg, cand);
		expect(cand.title).toBe('How can I resolve this error');
	});

	it('ignores fenced/inline code while scoring', () => {
		const msg = `
\`\`\`python
raise KeyError('id')
\`\`\`
Why does this KeyError happen?
`;
		const cand = generateTitle(msg);
		debugPrint('Code + question', msg, cand);
		expect(cand.title).toBe('Why does this KeyError happen');
	});
});

/* ─── non-English fallback ───────────────────────────────────────── */
describe('generateTitle - non-English / generic path', () => {
	it('still returns ≤48 chars for Spanish input', () => {
		const esp = 'Hola, ¿cómo puedo centrar un div en CSS?';
		const cand = generateTitle(esp);
		debugPrint('Spanish question', esp, cand);
		expect(cand.title.length).toBeLessThanOrEqual(48);
	});
});

/* ─── hard limits & casing ───────────────────────────────────────── */
describe('generateTitle - hard limits', () => {
	it('never exceeds 48 characters', () => {
		const long = 'Hello! '.repeat(100);
		const cand = generateTitle(long);
		debugPrint('Long repeated input', long, cand);
		expect(cand.title.length).toBeLessThanOrEqual(48);
	});

	it('output is sentence-case', () => {
		const msg = 'WHY AM I SEEING ECONNREFUSED WHEN I DEPLOY?';
		const cand = generateTitle(msg);
		debugPrint('All-caps input', msg, cand);
		const { title } = cand;
		expect(title[0]).toBe(title[0].toUpperCase());
		expect(title.slice(1)).toBe(title.slice(1).toUpperCase());
	});
});

/* ─── code sample -> rectify request ─────────────────────────────── */
describe('generateTitle - code sample + rectify request', () => {
	it('extracts the final "how do I fix" question', () => {
		const msg = `
Hi team,

Below code crashes:

\`\`\`python
data = {'a': 1}
print(data['b'])   # KeyError here
\`\`\`

It raises KeyError: 'b'.

How can I fix the code so it prints 0 when the key is missing, without using try/except?
`;
		const cand = generateTitle(msg);
		debugPrint('KeyError rectify', msg, cand);
		expect(cand.title.length).toBeLessThanOrEqual(48);
		expect(cand.title.toLowerCase()).toMatch(/fix|avoid|keyerror/);
		expect(cand.title).not.toBe(DEFAULT);
	});
});

/* ─── generic factual Q&A (fruit) ───────────────────────────────── */
describe('generateTitle - generic factual Q&A (fruit)', () => {
	it('produces a concise, topic-rich title', () => {
		const msg = `
Hey!

I've been reading about different fruits.

Could you explain why bananas have so much potassium and how that
benefits human health?  A short answer is fine.
`;
		const cand = generateTitle(msg);
		debugPrint('Banana question', msg, cand);
		expect(cand.title.toLowerCase().includes('bananas')).toBe(true);
	});
});

/* ─── paragraph-aware edge cases ─────────────────────────────────── */
it('ignores plain code lines inside a paragraph', () => {
	const msg = `
analyze the code below
processData()
return result
please list any obvious bugs?
`;
	const cand = generateTitle(msg);
	debugPrint('Plain code lines', msg, cand);
	expect(cand.title).toBe('Please list any obvious bugs');
});

it('multi-line short prompts keep both lines', () => {
	const msg = 'give me a quote\ngive author details too';
	const cand = generateTitle(msg);
	debugPrint('Short multi-line', msg, cand);
	expect(cand.title).toBe('Give author details too');
});

it('prefers the last paragraph question over earlier notes', () => {
	const msg = `
Random note at top.

someFunction();
anotherCall();

Finally, how can I optimise this without recursion?
`;
	const cand = generateTitle(msg);
	debugPrint('Last paragraph wins', msg, cand);
	expect(cand.title.toLowerCase()).toMatch(/optimise|optimize/);
});

it('min actual lines around code', () => {
	const cand = generateTitle(codeMessage);
	debugPrint('Plain code lines', codeMessage, cand);
	expect(cand.title).toBe('Analyze and give a review of below code');
});
