import nlp from 'compromise';
import { removeStopwords } from 'stopword';

/* ─────────────────────────  Constants  ─────────────────────────── */
const MAX_LEN = 48;
const DEFAULT_TITLE = 'New conversation';

export interface TitleCandidate {
	title: string; // already ≤ 48 chars, polished etc.
	score: number; // "how good is this title?" 0 ... 1  (1 = perfect)
}

const DEFAULT_TITLE_CANDIDATE: TitleCandidate = { title: DEFAULT_TITLE, score: 0.05 };

/**
 * Very light-weight "does this line look like code?" heuristic.
 * It is NOT 100 % accurate - it is only meant to be fast and
 * surprisingly good for common cases.
 */
function looksLikeCodeLine(line: string): boolean {
	const t = line.trim();
	if (!t) {
		return false;
	} // blank line

	/* 1.  Quick "certain" matches  */

	// single brace / bracket / semicolon lines:  }, );, ]
	if (/^[{}[\]()];?$/.test(t)) {
		return true;
	}

	// comment markers at line start ( //  #  /*  *  <!-- )
	if (/^\s*(\/\/|#|\/\*|\*|<!--)/.test(t)) {
		return true;
	}

	// import / class / def at beginning of line
	if (/^\s*(import|export|package|namespace|using|class|interface|def|func|lambda|const|async)\b/.test(t)) {
		return true;
	}

	/* 2.  Count weaker indicators  */

	let score = 0;

	if (/[{}[$$;]/.test(t)) {
		score++;
	} // braces / semicolon
	if (/=>|->|::|\+\+|--|&&|\|\||\?\.|\?\?/.test(t)) {
		score++;
	} // multi-char operators
	if (/[=+\-*/%]=?\s*\w/.test(t)) {
		score++;
	} // assignments / operators
	if (/\w+\s*$[^)]*$/.test(t)) {
		score++;
	} // foo(...)  or  bar()
	if (/(['"`]).*\1/.test(t)) {
		score++;
	} // string literal

	// At least two weak indicators -> probably code
	return score >= 2;
}

function isCodeParagraph(p: string): boolean {
	const lines = p.split(/\n/);
	const codeLike = lines.filter(l => looksLikeCodeLine(l)).length;
	return codeLike >= lines.length / 2; // ≥ 50 % lines look like code
}

/* remove fenced, indented and inline code + crude HTML */
function stripCode(txt: string): string {
	return (
		txt
			// ``` ... ```  or  ~~~ ... ~~~
			.replaceAll(/```[\s\S]*?```|~~~[\s\S]*?~~~/g, '')
			// 4-space / tab indented blocks
			.replaceAll(/^(?: {4}|\t).*\n?/gm, '')
			// inline `code`
			.replaceAll(/`[^`\n]*`/g, '')
			// very simple HTML tags
			.replaceAll(/<\/?[a-z][^>]*>/gi, '')
	);
}

/* ───────────────────────  Splitters  ───────────────────────────── */
function splitParagraphs(txt: string): string[] {
	return txt
		.split(/\n{2,}/)
		.map(p => p.trim())
		.filter(Boolean);
}

function splitSentences(txt: string): string[] {
	return txt
		.split(/\n+/) // keep single \n as separator
		.flatMap(p => p.split(/(?<=[.!?])\s+/))
		.map(s => s.trim())
		.filter(Boolean);
}

/* ──────────────────  Candidate paragraph selection  ────────────── */
function selectCandidateParagraphs(paragraphs: string[]): string[] {
	const nonCode = paragraphs.filter(p => !isCodeParagraph(p));

	// ① first two + last two non-code paragraphs if possible
	if (nonCode.length >= 4) {
		return [...nonCode.slice(0, 2), ...nonCode.slice(-2)];
	}

	// ② otherwise take whatever non-code we have
	if (nonCode.length > 0) {
		return nonCode;
	}

	// ③ nothing but code -> fall back to very first paragraph
	return [paragraphs[0]];
}

/* ─────────────────────  Scoring heuristics  ────────────────────── */
function scoreEn(s: string, idx: number, total: number): number {
	const t = s.trim();
	let pts = 0;

	if (t.includes('?')) {
		pts += 3;
	}
	if (/^(how|what|why|when|where|can|does|do|is|are|should|could|would|will|give)\b/i.test(t)) {
		pts += 2;
	}
	if (/\b(error|issue|bug|fail|help)\b/i.test(t)) {
		pts += 1.5;
	}
	if (idx === 0 || idx === total - 1) {
		pts += 1;
	}
	if (/^(\s*[-*]\s+|\s*\d+\.\s+)/.test(t)) {
		pts += 1;
	}
	if (t.length < 60) {
		pts += 1;
	}

	const doc = nlp(t);
	let uniq = new Set([...doc.nouns().out('array'), ...doc.verbs().out('array')]);
	pts += 0.2 * uniq.size;
	uniq = new Set([...doc.adjectives().out('array'), ...doc.adverbs().out('array')]);
	pts += 0.1 * uniq.size;

	const tokens = t.toLowerCase().match(/\b[a-z']{2,}\b/g) || [];
	if (tokens.length > 0) {
		const noStop: string[] = removeStopwords(tokens);
		const stopRatio = 1 - noStop.length / tokens.length;
		// Reward moderate stopword ratio (e.g., 0.2-0.6)
		if (stopRatio > 0.15 && stopRatio < 0.65) {
			pts += 1.5;
		}
		// Penalize very low or very high stopword ratio
		else if (stopRatio <= 0.15 || stopRatio >= 0.65) {
			pts -= 0.5;
		}
	}

	return pts;
}

function normalise(raw: number): number {
	// raw ≈ 0-12  ->  0-1
	return Math.max(0, Math.min(1, raw / 12));
}

/* ───────────────────────  Final polish  ────────────────────────── */

function sentenceCase(str: string) {
	if (!str) {
		return '';
	}
	const match = str.match(/^(\s*)(\S)(.*)$/);
	if (!match) {
		return str;
	}
	const [, leading, first, rest] = match;
	return leading + first.toUpperCase() + rest;
}

function finalise(raw: string): string {
	let t = raw
		// salutations
		.replace(/^(hi|hello|hey|dear|greetings|good (morning|afternoon|evening))[,!:.\s-]+/i, '')
		// emails & phones
		.replaceAll(/[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}/gi, '')
		.replaceAll(/\+?\d[\d\s.-]{7,}/g, '')
		// whitespace
		.replaceAll(/\s+/g, ' ')
		.trim()
		// leading / trailing punctuation
		.replaceAll(/^[\s:;,-]+|[\s:;,.?!-]+$/g, '');

	if (!t) {
		return DEFAULT_TITLE;
	}

	if (t.length > MAX_LEN) {
		const cut = t.slice(0, MAX_LEN);
		const lastSpace = cut.lastIndexOf(' ');
		t = lastSpace > 25 ? cut.slice(0, lastSpace) : cut;
	}

	return sentenceCase(t);
}

/* ──────────────────────────  Public API  ───────────────────────── */
export function generateTitle(firstMessage: string): TitleCandidate {
	if (!firstMessage.trim()) {
		return DEFAULT_TITLE_CANDIDATE;
	}

	const paragraphs = splitParagraphs(firstMessage.trim());
	if (paragraphs.length === 0) {
		return DEFAULT_TITLE_CANDIDATE;
	}

	/* ① choose candidate paragraphs */
	const candidates = selectCandidateParagraphs(paragraphs);
	// console.log('candidates', JSON.stringify(candidates, null, 2));

	/* ② collect sentences & drop code-looking lines */
	const sentences = candidates.flatMap(p => splitSentences(p)).filter(l => !looksLikeCodeLine(l));
	// console.log('sentences', JSON.stringify(sentences, null, 2));
	if (sentences.length === 0) {
		// still nothing - just clean first paragraph
		const t: TitleCandidate = { title: finalise(stripCode(paragraphs[0])), score: 0.2 };
		return t;
	}

	let best = '';
	let bestRaw = -Infinity;
	sentences.forEach((s, i) => {
		const pts = scoreEn(s, i, sentences.length);
		if (pts > bestRaw) {
			bestRaw = pts;
			best = s;
		}
	});
	/* ④ final polish */
	return { title: finalise(best), score: normalise(bestRaw) };
}

const isSingleLetter = (tok: string) => {
	return tok.length === 1 && /^[a-z]$/i.test(tok);
};

// Build a safe FTS5 MATCH string from raw user input.
//  - lower-cases
//  - removes stop-words
//  - deduplicates
//  - removes single chars
export function cleanSearchQuery(input: string): string {
	const raw = input.trim();
	if (raw === '') {
		return '';
	}

	const nlpTokens = (nlp(raw).terms().out('array') as string[]).map(t => t.toLowerCase());
	const nonOneCharTokens = nlpTokens.filter(t => !isSingleLetter(t)); // drop 1-char words
	if (nonOneCharTokens.length === 0) {
		return raw.toLowerCase();
	}

	let noStopTokens: string[] = removeStopwords(nonOneCharTokens);
	if (noStopTokens.length === 0) {
		noStopTokens = nonOneCharTokens;
	}

	return Array.from(new Set(noStopTokens)).join(' ');
}
