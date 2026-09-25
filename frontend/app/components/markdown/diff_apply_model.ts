import type { ApplyUnifiedDiffDiagnostic, ApplyUnifiedDiffOut } from '@/spec/unified_diff';
import { ApplyUnifiedDiffDiagnosticLevel, ApplyUnifiedDiffStatus } from '@/spec/unified_diff';

const DEV_NULL = '/dev/null';

/*
 * Important architecture boundary:
 *
 * This module intentionally does not validate unified-diff correctness.
 *
 * Do not add frontend enforcement for:
 * - hunk line counts
 * - line numbers
 * - whitespace/context requirements
 * - missing hunk context
 * - malformed LLM-generated patch bodies
 *
 * The backend fuzzy applier and its dry-run endpoint are the sole authority
 * for applicability. This module only isolates likely file/hunk sections and
 * preserves source text for backend review/apply requests.
 */

export const MAX_INTERACTIVE_HUNK_PROBES = 64;
const MAX_INTERACTIVE_DIFF_CHARACTERS = 1_500_000;
const MAX_INTERACTIVE_DIFF_FILES = 100;
const MAX_INTERACTIVE_CANDIDATES = 100;
const MAX_INTERACTIVE_DIAGNOSTICS = 128;

type InteractiveDiffFileKind = 'modify' | 'add' | 'delete' | 'rename';

export interface InteractiveDiffHunk {
	id: string;
	header: string;
	sourceText: string;
}

export interface InteractiveDiffFile {
	id: string;
	oldPath?: string;
	newPath?: string;
	kind: InteractiveDiffFileKind;

	/**
	 * Original file section used for preview. This is intentionally preserved
	 * even when headers/counts/body shape look malformed.
	 */
	sourceText: string;

	/**
	 * Text sent for a whole-file backend review/apply request.
	 * It equals sourceText. The UI does not repair or convert patch syntax
	 * before asking the backend to review it.
	 */
	requestText: string;

	/**
	 * Original file preamble before the first hunk. It is reused when sending
	 * one selected hunk, without rebuilding or renumbering hunk coordinates.
	 */
	hunkPrefixText: string;

	hunks: InteractiveDiffHunk[];
	diagnostics: ApplyUnifiedDiffDiagnostic[];
}

export interface ParsedInteractiveDiff {
	isDiffLike: boolean;
	isOpenAIPatch: boolean;
	files: InteractiveDiffFile[];
	diagnostics: ApplyUnifiedDiffDiagnostic[];
}

export interface InteractiveDiffTargetSuggestion {
	targetPath: string;
	candidates: string[];
}

export type DiffApplyPhase = 'dry-run' | 'apply';

export type DiffApplyOutcomeStatus = 'ready' | 'needs-info' | 'blocked' | 'applied' | 'already-applied' | 'partial';

export interface DiffApplyOutcome {
	phase: DiffApplyPhase;
	status: DiffApplyOutcomeStatus;
	message: string;
	diagnostics: ApplyUnifiedDiffDiagnostic[];

	/**
	 * Backend-reported target/resolved path. This is presentation data and is
	 * not validated or rewritten by the UI.
	 */
	resolvedTargetPath?: string;
}

interface WorkingDiffSection {
	lines: string[];
	hasGitBoundary: boolean;
	hasHeaders: boolean;
	oldPath?: string;
	newPath?: string;
}

interface HeaderPair {
	oldPath: string;
	newPath: string;
}

function createDiagnostic(
	level: ApplyUnifiedDiffDiagnosticLevel,
	code: string,
	message: string
): ApplyUnifiedDiffDiagnostic {
	return {
		level,
		code,
		message,
	};
}

function appendDiagnostic(
	target: ApplyUnifiedDiffDiagnostic[],
	level: ApplyUnifiedDiffDiagnosticLevel,
	code: string,
	message: string
): void {
	if (target.length < MAX_INTERACTIVE_DIAGNOSTICS) {
		target.push(createDiagnostic(level, code, message));
		return;
	}

	if (!target.some(diagnostic => diagnostic.code === 'diagnostics_truncated')) {
		target.push(
			createDiagnostic(
				ApplyUnifiedDiffDiagnosticLevel.Warning,
				'diagnostics_truncated',
				'Additional parser notes were suppressed to keep the diff UI responsive.'
			)
		);
	}
}

function ensureTrailingNewline(value: string): string {
	if (!value) {
		return '';
	}

	return value.endsWith('\n') ? value : `${value}\n`;
}

function sourceFromLines(lines: string[]): string {
	return ensureTrailingNewline(lines.join('\n'));
}

function limitInteractiveDiagnostics(
	values: Array<ApplyUnifiedDiffDiagnostic | undefined | null>,
	limit = MAX_INTERACTIVE_DIAGNOSTICS
): ApplyUnifiedDiffDiagnostic[] {
	const unique = new Map<string, ApplyUnifiedDiffDiagnostic>();

	for (const value of values) {
		if (!value) {
			continue;
		}

		const message = value.message?.trim() ?? '';
		if (!message) {
			continue;
		}

		const level = value.level ?? ApplyUnifiedDiffDiagnosticLevel.Info;
		const code = value.code?.trim() ?? '';
		const key = `${level}\u0000${code}\u0000${message}`;

		if (!unique.has(key)) {
			unique.set(key, {
				level,
				code: code || undefined,
				message,
			});
		}
	}

	const all = [...unique.values()];
	const ordered = [
		...all.filter(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Error),
		...all.filter(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Warning),
		...all.filter(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Info),
	];

	if (ordered.length <= limit) {
		return ordered;
	}

	return [
		...ordered.slice(0, limit),
		createDiagnostic(
			ApplyUnifiedDiffDiagnosticLevel.Warning,
			'diagnostics_truncated',
			'Additional diagnostics were suppressed to keep the diff UI responsive.'
		),
	];
}

function createUnsplitInteractiveDiffFile(
	text: string,
	diagnostics: ApplyUnifiedDiffDiagnostic[] = []
): InteractiveDiffFile {
	return {
		id: 'unsplit-section',
		kind: 'modify',
		sourceText: text,
		requestText: text,
		hunkPrefixText: '',
		hunks: [],
		diagnostics: limitInteractiveDiagnostics(diagnostics),
	};
}

function normalizeInteractiveTargetPath(value: string | undefined | null): string {
	const raw = value?.trim() ?? '';

	if (!raw || raw === DEV_NULL || /[\u0000-\u001F\u007F]/.test(raw)) {
		return '';
	}

	const withForwardSlashes = raw.replaceAll('\\', '/');
	const isUnc = withForwardSlashes.startsWith('//');
	const normalized = isUnc
		? `//${withForwardSlashes.slice(2).replaceAll(/\/+/g, '/')}`
		: withForwardSlashes.replaceAll(/\/+/g, '/');

	const drive = /^([A-Za-z]):\//.exec(normalized);

	if (drive?.[1]) {
		const parts = normalized.slice(3).split('/').filter(Boolean);

		if (parts.some(part => part === '.' || part === '..')) {
			return '';
		}

		const prefix = `${drive[1].toUpperCase()}:/`;
		return parts.length > 0 ? `${prefix}${parts.join('/')}` : prefix;
	}

	if (isUnc) {
		const parts = normalized.slice(2).split('/').filter(Boolean);

		if (parts.length < 2 || parts.some(part => part === '.' || part === '..')) {
			return '';
		}

		return `//${parts.join('/')}`;
	}

	if (!normalized.startsWith('/')) {
		return '';
	}

	const parts = normalized.slice(1).split('/').filter(Boolean);

	if (parts.some(part => part === '.' || part === '..')) {
		return '';
	}

	return parts.length > 0 ? `/${parts.join('/')}` : '/';
}

export function normalizeInteractiveTargetPaths(values: Array<string | undefined | null>): string[] {
	const output: string[] = [];
	const seen = new Set<string>();

	for (const value of values) {
		const path = normalizeInteractiveTargetPath(value);

		if (!path || seen.has(path)) {
			continue;
		}

		seen.add(path);
		output.push(path);
	}

	return output;
}

function targetPathIdentity(value: string | undefined | null): string {
	const normalized = normalizeInteractiveTargetPath(value);

	if (!normalized) {
		return '';
	}

	return normalized.startsWith('//') || /^[A-Z]:\//.test(normalized) ? normalized.toLowerCase() : normalized;
}

function normalizePatchPath(value: string | undefined | null): string {
	const raw = value?.trim() ?? '';

	if (!raw || raw === DEV_NULL) {
		return raw;
	}

	return raw
		.replaceAll('\\', '/')
		.replaceAll(/\/+/g, '/')
		.replace(/^(?:\.\/)+/, '')
		.replaceAll(/\/+$/g, '');
}

function safeRelativePatchPath(value: string | undefined): string {
	const normalized = normalizePatchPath(value);

	if (!normalized || normalized === DEV_NULL || normalizeInteractiveTargetPath(normalized)) {
		return '';
	}

	if (normalized.startsWith('/') || normalized.startsWith('//') || /^[A-Za-z]:/.test(normalized)) {
		return '';
	}

	const parts = normalized.split('/').filter(Boolean);

	if (parts.length === 0 || parts.some(part => part === '.' || part === '..')) {
		return '';
	}

	return parts.join('/');
}

function patchPathIdentity(value: string | undefined): string {
	return targetPathIdentity(value) || normalizePatchPath(value);
}

function pathEndsWithRelativePath(candidate: string, relativePath: string): boolean {
	const normalizedCandidate = normalizeInteractiveTargetPath(candidate);

	if (!normalizedCandidate || !relativePath) {
		return false;
	}

	return normalizedCandidate.endsWith(`/${relativePath}`);
}

function rootForCandidatePath(candidate: string, relativePath: string): string {
	const normalizedCandidate = normalizeInteractiveTargetPath(candidate);
	const suffix = `/${relativePath}`;

	if (!normalizedCandidate || !relativePath || !normalizedCandidate.endsWith(suffix)) {
		return '';
	}

	const root = normalizedCandidate.slice(0, normalizedCandidate.length - suffix.length);
	const absoluteRoot = /^[A-Za-z]:$/.test(root) ? `${root}/` : root;

	return normalizeInteractiveTargetPath(absoluteRoot || '/');
}

function joinWorkspaceRoot(root: string, relativePath: string): string {
	const normalizedRoot = normalizeInteractiveTargetPath(root);

	if (!normalizedRoot || !relativePath) {
		return '';
	}

	return normalizeInteractiveTargetPath(
		normalizedRoot.endsWith('/') ? `${normalizedRoot}${relativePath}` : `${normalizedRoot}/${relativePath}`
	);
}

function isPathWithinRoot(path: string, root: string): boolean {
	const pathKey = targetPathIdentity(path);
	const rootKey = targetPathIdentity(root);

	if (!pathKey || !rootKey) {
		return false;
	}

	if (rootKey === '/') {
		return pathKey.startsWith('/');
	}

	return pathKey === rootKey || pathKey.startsWith(rootKey.endsWith('/') ? rootKey : `${rootKey}/`);
}

function readDiffToken(input: string): { token: string; rest: string } {
	const value = input.trimStart();

	if (!value) {
		return {
			token: '',
			rest: '',
		};
	}

	if (!value.startsWith('"')) {
		const match = /^(\S+)(?:\s+([\s\S]*))?$/.exec(value);

		return {
			token: match?.[1] ?? '',
			rest: match?.[2] ?? '',
		};
	}

	let token = '';
	const bytes: number[] = [];

	const flushBytes = () => {
		if (bytes.length > 0) {
			token += new TextDecoder('utf-8', { fatal: true }).decode(new Uint8Array(bytes));
			bytes.length = 0;
		}
	};

	for (let index = 1; index < value.length; index += 1) {
		const character = value[index];

		if (character === '"') {
			flushBytes();

			return {
				token,
				rest: value.slice(index + 1).trimStart(),
			};
		}

		if (character !== '\\') {
			flushBytes();
			token += character;
			continue;
		}

		const next = value[index + 1];

		if (!next) {
			throw new Error('Unterminated escape in quoted diff path.');
		}

		if (/[0-7]/.test(next)) {
			let octal = next;
			let cursor = index + 2;

			while (octal.length < 3 && /[0-7]/.test(value[cursor] ?? '')) {
				octal += value[cursor];
				cursor += 1;
			}

			const byte = Number.parseInt(octal, 8);

			if (byte > 255) {
				throw new Error('Invalid octal byte in quoted diff path.');
			}

			bytes.push(byte);
			index = cursor - 1;
			continue;
		}

		flushBytes();

		const escapes: Record<string, string> = {
			'"': '"',
			'\\': '\\',
			n: '\n',
			r: '\r',
			t: '\t',
		};

		if (!(next in escapes)) {
			throw new Error('Unsupported escape in quoted diff path.');
		}

		token += escapes[next];
		index += 1;
	}

	throw new Error('Unterminated quoted diff path.');
}

function readHeaderPath(input: string): string {
	const value = input.trimStart();

	if (!value) {
		return '';
	}

	if (value.startsWith('"')) {
		try {
			return readDiffToken(value).token;
		} catch {
			return '';
		}
	}

	const tabIndex = value.indexOf('\t');

	if (tabIndex >= 0) {
		return value.slice(0, tabIndex).trim();
	}

	// Standard unified diffs typically use a tab before timestamps. If an LLM
	// omitted it, keep a likely spaced filename rather than validating/rejecting.
	return value.replace(/\s+\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:[\s\S]*$/, '').trim();
}

function normalizeHeaderPair(oldPath: string, newPath: string): HeaderPair {
	const oldHasGitPrefix = oldPath.startsWith('a/');
	const newHasGitPrefix = newPath.startsWith('b/');

	if ((oldHasGitPrefix && (newHasGitPrefix || newPath === DEV_NULL)) || (oldPath === DEV_NULL && newHasGitPrefix)) {
		return {
			oldPath: oldHasGitPrefix ? oldPath.slice(2) : oldPath,
			newPath: newHasGitPrefix ? newPath.slice(2) : newPath,
		};
	}

	return {
		oldPath,
		newPath,
	};
}

function formatHeaderPath(path: string): string {
	if (path === DEV_NULL) {
		return DEV_NULL;
	}

	return /[\s"\\]/.test(path) ? JSON.stringify(path) : path;
}

function getFileKind(oldPath: string | undefined, newPath: string | undefined): InteractiveDiffFileKind {
	if (oldPath === DEV_NULL && newPath && newPath !== DEV_NULL) {
		return 'add';
	}

	if (newPath === DEV_NULL && oldPath && oldPath !== DEV_NULL) {
		return 'delete';
	}

	if (oldPath && newPath && patchPathIdentity(oldPath) !== patchPathIdentity(newPath)) {
		return 'rename';
	}

	return 'modify';
}

function isHeaderLine(line: string, prefix: '---' | '+++'): boolean {
	return line.startsWith(`${prefix} `) || line.startsWith(`${prefix}\t`);
}

function readHeaderPair(lines: string[], index: number): HeaderPair | undefined {
	const oldLine = lines[index] ?? '';
	const newLine = lines[index + 1] ?? '';

	if (!isHeaderLine(oldLine, '---') || !isHeaderLine(newLine, '+++')) {
		return undefined;
	}

	const oldPath = readHeaderPath(oldLine.slice(4));
	const newPath = readHeaderPath(newLine.slice(4));

	if (!oldPath || !newPath) {
		return undefined;
	}

	return normalizeHeaderPair(oldPath, newPath);
}

function likelyNewBareFileSection(pair: HeaderPair): boolean {
	if (pair.oldPath === DEV_NULL || pair.newPath === DEV_NULL) {
		return true;
	}

	if (pair.oldPath === pair.newPath) {
		return true;
	}

	return /[/.\\]/.test(pair.oldPath) && /[/.\\]/.test(pair.newPath);
}

function extractHunks(
	lines: string[],
	sectionID: string
): {
	hunks: InteractiveDiffHunk[];
	hunkPrefixText: string;
} {
	const hunkIndexes: number[] = [];

	for (let index = 0; index < lines.length; index += 1) {
		if ((lines[index] ?? '').startsWith('@@')) {
			hunkIndexes.push(index);
		}
	}

	if (hunkIndexes.length === 0) {
		return {
			hunks: [],
			hunkPrefixText: sourceFromLines(lines),
		};
	}

	const firstHunkIndex = hunkIndexes[0] ?? 0;
	const hunks = hunkIndexes.map((start, index) => {
		const nextStart = hunkIndexes[index + 1] ?? lines.length;
		const header = lines[start] ?? '';

		return {
			id: `${sectionID}:hunk-${index + 1}`,
			header,
			sourceText: sourceFromLines(lines.slice(start, nextStart)),
		};
	});

	return {
		hunks,
		hunkPrefixText: sourceFromLines(lines.slice(0, firstHunkIndex)),
	};
}

function finalizeStandardSection(section: WorkingDiffSection, sectionNumber: number): InteractiveDiffFile | undefined {
	if (section.lines.length === 0) {
		return undefined;
	}

	const id = `section-${sectionNumber}`;
	const sourceText = sourceFromLines(section.lines);
	const extracted = extractHunks(section.lines, id);
	const diagnostics: ApplyUnifiedDiffDiagnostic[] = [];

	if (!section.hasHeaders) {
		appendDiagnostic(
			diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Info,
			'missing_file_headers',
			'No paired ---/+++ file headers were found. The backend will resolve this section during review.'
		);
	}

	if (extracted.hunks.length === 0) {
		appendDiagnostic(
			diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Info,
			'no_hunk_markers',
			'No @@ hunk markers were found. The file section is still sent unchanged to the backend.'
		);
	}

	return {
		id,
		oldPath: section.oldPath,
		newPath: section.newPath,
		kind: getFileKind(section.oldPath, section.newPath),
		sourceText,
		requestText: sourceText,
		hunkPrefixText: extracted.hunkPrefixText,
		hunks: extracted.hunks,
		diagnostics: limitInteractiveDiagnostics(diagnostics),
	};
}

function parseGitBoundaryPaths(line: string): HeaderPair | undefined {
	try {
		const first = readDiffToken(line.slice('diff --git '.length));
		const second = readDiffToken(first.rest);

		if (!first.token || !second.token) {
			return undefined;
		}

		return normalizeHeaderPair(first.token, second.token);
	} catch {
		return undefined;
	}
}

function parseStandardDiff(text: string): {
	files: InteractiveDiffFile[];
	diagnostics: ApplyUnifiedDiffDiagnostic[];
} {
	const lines = text.split('\n');
	const files: InteractiveDiffFile[] = [];
	const diagnostics: ApplyUnifiedDiffDiagnostic[] = [];
	let current: WorkingDiffSection | undefined;

	const flush = () => {
		if (!current) {
			return;
		}

		const file = finalizeStandardSection(current, files.length + 1);

		if (file) {
			files.push(file);
		}

		current = undefined;
	};

	for (let index = 0; index < lines.length;) {
		const line = lines[index] ?? '';

		if (line.startsWith('diff --git ')) {
			flush();

			current = {
				lines: [line],
				hasGitBoundary: true,
				hasHeaders: false,
			};

			const pair = parseGitBoundaryPaths(line);

			if (pair) {
				current.oldPath = pair.oldPath;
				current.newPath = pair.newPath;
			}

			index += 1;
			continue;
		}

		if (line.startsWith('Index: ')) {
			flush();

			const path = line.slice('Index: '.length).trim();

			current = {
				lines: [line],
				hasGitBoundary: true,
				hasHeaders: false,
				oldPath: path || undefined,
				newPath: path || undefined,
			};

			index += 1;
			continue;
		}

		const pair = readHeaderPair(lines, index);

		if (pair) {
			if (!current) {
				current = {
					lines: [],
					hasGitBoundary: false,
					hasHeaders: false,
				};
			} else if (current.hasHeaders && !current.hasGitBoundary && likelyNewBareFileSection(pair)) {
				flush();
				current = {
					lines: [],
					hasGitBoundary: false,
					hasHeaders: false,
				};
			}

			if (!current.hasHeaders) {
				current.oldPath = pair.oldPath;
				current.newPath = pair.newPath;
				current.hasHeaders = true;
				current.lines.push(line, lines[index + 1] ?? '');
				index += 2;
				continue;
			}

			/*
			 * The current section already has headers. This can be a hunk
			 * payload such as "--- old" followed by "+++ new". Preserve it
			 * exactly instead of treating it as another file boundary.
			 */
			current.lines.push(line);
			index += 1;
			continue;
		}

		if (!current && line.startsWith('@@')) {
			current = {
				lines: [],
				hasGitBoundary: false,
				hasHeaders: false,
			};
		}

		if (current) {
			current.lines.push(line);
		}

		// Ignore mail headers and patch preambles outside known file sections.
		index += 1;
	}

	flush();

	if (files.length === 0) {
		appendDiagnostic(
			diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Warning,
			'no_file_sections',
			'No file sections could be confidently separated.'
		);

		if (text.trim()) {
			files.push(
				createUnsplitInteractiveDiffFile(text, [
					createDiagnostic(
						ApplyUnifiedDiffDiagnosticLevel.Info,
						'unsplit_diff',
						'The UI could not split this diff. The original text will be sent unchanged to the backend.'
					),
				])
			);
		}
	}

	return {
		files,
		diagnostics,
	};
}

function parseOpenAIPath(value: string): string {
	const trimmed = value.trim();

	if (!trimmed) {
		return '';
	}

	if (!trimmed.startsWith('"')) {
		return trimmed;
	}

	try {
		return readDiffToken(trimmed).token;
	} catch {
		return '';
	}
}

function parseOpenAIPatch(text: string): {
	files: InteractiveDiffFile[];
	diagnostics: ApplyUnifiedDiffDiagnostic[];
} {
	const lines = text.split('\n');
	const headerPattern = /^\*\*\*\s+(Add|Update|Delete)\s+File:\s*(.+?)\s*$/i;
	const sections: Array<{
		index: number;
		kind: 'add' | 'update' | 'delete';
		path: string;
	}> = [];
	const diagnostics: ApplyUnifiedDiffDiagnostic[] = [];
	const endIndex = lines.findIndex(line => line.trim() === '*** End Patch');
	const hasBegin = lines.some(line => line.trim() === '*** Begin Patch');

	for (let index = 0; index < lines.length; index += 1) {
		const match = headerPattern.exec(lines[index] ?? '');

		if (!match?.[1] || !match[2]) {
			continue;
		}

		const kind = match[1].toLowerCase();
		const path = parseOpenAIPath(match[2]);

		sections.push({
			index,
			kind: kind === 'add' ? 'add' : kind === 'delete' ? 'delete' : 'update',
			path,
		});
	}

	if (hasBegin && endIndex < 0) {
		appendDiagnostic(
			diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Warning,
			'openai_patch_missing_end',
			'The OpenAI patch has no End Patch marker. Sections are still preserved for backend review.'
		);
	}

	if (endIndex >= 0 && lines.slice(endIndex + 1).some(line => line.trim())) {
		appendDiagnostic(
			diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Warning,
			'openai_patch_trailing_text',
			'Unexpected text follows End Patch. It was not included in a scoped file request.'
		);
	}

	const files: InteractiveDiffFile[] = [];

	for (let index = 0; index < sections.length; index += 1) {
		const section = sections[index];

		if (!section) {
			continue;
		}

		const nextSection = sections[index + 1];
		const sectionEnd =
			endIndex >= section.index
				? Math.min(nextSection?.index ?? lines.length, endIndex)
				: (nextSection?.index ?? lines.length);
		const rawLines = lines.slice(section.index, sectionEnd);
		const rawSourceText = sourceFromLines(rawLines);
		const id = `section-${files.length + 1}`;
		const oldPath = section.kind === 'add' ? DEV_NULL : section.path || undefined;
		const newPath = section.kind === 'delete' ? DEV_NULL : section.path || undefined;
		const diagnosticsForFile: ApplyUnifiedDiffDiagnostic[] = [];
		const extracted = extractHunks(rawLines, id);

		appendDiagnostic(
			diagnosticsForFile,
			ApplyUnifiedDiffDiagnosticLevel.Info,
			'openai_section_preserved',
			'This OpenAI patch section is sent unchanged to the backend for fuzzy review.'
		);

		files.push({
			id,
			oldPath,
			newPath,
			kind: section.kind === 'add' ? 'add' : section.kind === 'delete' ? 'delete' : 'modify',
			sourceText: rawSourceText,
			requestText: rawSourceText,
			hunkPrefixText: extracted.hunkPrefixText,
			hunks: extracted.hunks,
			diagnostics: limitInteractiveDiagnostics(diagnosticsForFile),
		});
	}

	if (files.length === 0) {
		appendDiagnostic(
			diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Warning,
			'no_openai_file_sections',
			'No OpenAI file sections could be extracted.'
		);

		if (text.trim()) {
			files.push(
				createUnsplitInteractiveDiffFile(text, [
					createDiagnostic(
						ApplyUnifiedDiffDiagnosticLevel.Info,
						'unsplit_openai_patch',
						'The UI could not split this OpenAI patch. The original text will be sent unchanged to the backend.'
					),
				])
			);
		}
	}

	return {
		files,
		diagnostics,
	};
}

export function looksLikeInteractiveDiff(value: string, language = ''): boolean {
	const normalizedLanguage = language.trim().toLowerCase();

	if (normalizedLanguage === 'diff' || normalizedLanguage === 'patch' || normalizedLanguage === 'udiff') {
		return true;
	}

	return (
		/^diff --git\s+/m.test(value) ||
		/^Index:\s+/m.test(value) ||
		(/^---[ \t]/m.test(value) && /^\+\+\+[ \t]/m.test(value)) ||
		/^@@/m.test(value) ||
		/^\*\*\*\s+(?:Begin Patch|(?:Add|Update|Delete)\s+File:)/m.test(value)
	);
}

export function parseInteractiveDiff(value: string, language = ''): ParsedInteractiveDiff {
	/*
	 * Preserve line bodies as supplied. In particular, do not normalize bare
	 * carriage returns, whitespace, hunk counts, or line numbers. The backend
	 * fuzzy applier owns applicability semantics.
	 */
	const text = value.replace(/^\uFEFF/, '');
	const isDiffLike = looksLikeInteractiveDiff(text, language);
	const isOpenAIPatch = /^\*\*\*\s+(?:Begin Patch|(?:Add|Update|Delete)\s+File:)/m.test(text);

	if (text.length > MAX_INTERACTIVE_DIFF_CHARACTERS) {
		const tooLargeDiagnostic = createDiagnostic(
			ApplyUnifiedDiffDiagnosticLevel.Warning,
			'diff_too_large',
			`This diff exceeds the interactive UI budget of ${MAX_INTERACTIVE_DIFF_CHARACTERS.toLocaleString()} characters. It is sent unchanged to the backend as one section.`
		);

		return {
			isDiffLike,
			isOpenAIPatch,
			files: [createUnsplitInteractiveDiffFile(text, [tooLargeDiagnostic])],
			diagnostics: [tooLargeDiagnostic],
		};
	}

	const parsed = isOpenAIPatch ? parseOpenAIPatch(text) : parseStandardDiff(text);

	if (parsed.files.length > MAX_INTERACTIVE_DIFF_FILES) {
		const tooManySectionsDiagnostic = createDiagnostic(
			ApplyUnifiedDiffDiagnosticLevel.Warning,
			'too_many_file_sections',
			`This diff has more than ${MAX_INTERACTIVE_DIFF_FILES} file sections. It is sent unchanged to the backend as one section.`
		);

		return {
			isDiffLike,
			isOpenAIPatch,
			files: [createUnsplitInteractiveDiffFile(text, [tooManySectionsDiagnostic])],
			diagnostics: limitInteractiveDiagnostics([...parsed.diagnostics, tooManySectionsDiagnostic]),
		};
	}

	return {
		isDiffLike,
		isOpenAIPatch,
		files: parsed.files,
		diagnostics: limitInteractiveDiagnostics(parsed.diagnostics),
	};
}

export function getInteractiveDiffTargetPath(file: InteractiveDiffFile): string {
	if (file.kind === 'add') {
		return file.newPath ?? '';
	}

	if (file.kind === 'delete' || file.kind === 'rename') {
		return file.oldPath ?? '';
	}

	return file.newPath ?? file.oldPath ?? '';
}

export function isNewInteractiveDiffFile(file: InteractiveDiffFile): boolean {
	return file.kind === 'add' && file.oldPath === DEV_NULL && !!file.newPath;
}

interface AbsoluteInteractiveTargetPath {
	root: string;
	segments: string[];
	caseInsensitive: boolean;
}

interface RankedInteractiveTargetPath {
	path: string;
	rank: number;
	order: number;
}

interface DirectoryAnchor {
	candidateIndex: number;
	targetIndex: number;
	length: number;
}

function splitAbsoluteInteractiveTargetPath(value: string): AbsoluteInteractiveTargetPath | undefined {
	const path = normalizeInteractiveTargetPath(value);

	if (!path) {
		return undefined;
	}

	if (/^[A-Z]:\//.test(path)) {
		return {
			root: path.slice(0, 3),
			segments: path.slice(3).split('/').filter(Boolean),
			caseInsensitive: true,
		};
	}

	if (path.startsWith('//')) {
		const parts = path.slice(2).split('/').filter(Boolean);

		if (parts.length < 2) {
			return undefined;
		}

		return {
			root: `//${parts[0]}/${parts[1]}`,
			segments: parts.slice(2),
			caseInsensitive: true,
		};
	}

	return {
		root: '/',
		segments: path.slice(1).split('/').filter(Boolean),
		caseInsensitive: false,
	};
}

function joinInteractiveAbsolutePath(parts: AbsoluteInteractiveTargetPath, segments: string[]): string {
	const suffix = segments.filter(Boolean).join('/');

	if (parts.root === '/') {
		return normalizeInteractiveTargetPath(suffix ? `/${suffix}` : '/');
	}

	if (parts.root.endsWith('/')) {
		return normalizeInteractiveTargetPath(suffix ? `${parts.root}${suffix}` : parts.root);
	}

	return normalizeInteractiveTargetPath(suffix ? `${parts.root}/${suffix}` : parts.root);
}

function sameInteractivePathSegment(left: string, right: string, caseInsensitive: boolean): boolean {
	return caseInsensitive ? left.toLowerCase() === right.toLowerCase() : left === right;
}

function findBestDirectoryAnchor(
	candidateSegments: string[],
	targetSegments: string[],
	caseInsensitive: boolean
): DirectoryAnchor | undefined {
	const maxLength = Math.min(candidateSegments.length, targetSegments.length);

	for (let length = maxLength; length > 0; length -= 1) {
		for (let targetIndex = 0; targetIndex <= targetSegments.length - length; targetIndex += 1) {
			for (let candidateIndex = 0; candidateIndex <= candidateSegments.length - length; candidateIndex += 1) {
				const matches = targetSegments
					.slice(targetIndex, targetIndex + length)
					.every((segment, offset) =>
						sameInteractivePathSegment(segment, candidateSegments[candidateIndex + offset] ?? '', caseInsensitive)
					);

				if (matches) {
					return {
						candidateIndex,
						targetIndex,
						length,
					};
				}
			}
		}
	}

	return undefined;
}

function addRankedInteractiveTarget(
	targets: Map<string, RankedInteractiveTargetPath>,
	value: string,
	rank: number,
	order: number
): void {
	const path = normalizeInteractiveTargetPath(value);
	const key = targetPathIdentity(path);

	if (!path || !key) {
		return;
	}

	const previous = targets.get(key);

	if (!previous || rank < previous.rank || (rank === previous.rank && order < previous.order)) {
		targets.set(key, {
			path,
			rank,
			order,
		});
	}
}

export function inferInteractiveDiffTargets(
	files: InteractiveDiffFile[],
	candidatePaths: string[] = [],
	workspaceRoots: string[] = []
): Map<string, InteractiveDiffTargetSuggestion> {
	const externalCandidates = normalizeInteractiveTargetPaths(candidatePaths);
	const explicitRoots = normalizeInteractiveTargetPaths(workspaceRoots);
	const inferredAttachmentRoots: string[] = [];

	for (const file of files) {
		const relativePath = safeRelativePatchPath(getInteractiveDiffTargetPath(file));

		if (!relativePath) {
			continue;
		}

		for (const candidate of externalCandidates) {
			const root = rootForCandidatePath(candidate, relativePath);

			if (root) {
				inferredAttachmentRoots.push(root);
			}
		}
	}

	const attachmentRoots = normalizeInteractiveTargetPaths(inferredAttachmentRoots);

	const suggestions = new Map<string, InteractiveDiffTargetSuggestion>();

	for (const file of files) {
		const patchPath = getInteractiveDiffTargetPath(file);
		const absolutePatchPath = normalizeInteractiveTargetPath(patchPath);
		const relativePatchPath = safeRelativePatchPath(patchPath);
		const relativeSegments = relativePatchPath ? relativePatchPath.split('/').filter(Boolean) : [];
		const targetFileName = relativeSegments.at(-1) ?? '';
		const targetDirectorySegments = relativeSegments.slice(0, -1);
		const rankedTargets = new Map<string, RankedInteractiveTargetPath>();
		let order = 0;

		const addTarget = (path: string, rank: number) => {
			addRankedInteractiveTarget(rankedTargets, path, rank, order);
			order += 1;
		};

		/*
		 * An absolute path written in the patch is useful context, but an exact
		 * attached-file match always ranks above it.
		 */
		if (absolutePatchPath) {
			addTarget(absolutePatchPath, 1);
		}

		for (const candidate of externalCandidates) {
			const isExactFileMatch = absolutePatchPath
				? targetPathIdentity(candidate) === targetPathIdentity(absolutePatchPath)
				: relativePatchPath
					? pathEndsWithRelativePath(candidate, relativePatchPath)
					: false;

			if (isExactFileMatch) {
				addTarget(candidate, 0);
			}

			/*
			 * Candidate paths are existing files. Remove the filename, locate
			 * the closest directory anchor from the path in the patch text, and
			 * build a possible destination from that directory.
			 *
			 * This intentionally also runs for added files. New destinations
			 * normally do not exist in candidatePaths, so directory matching is
			 * the useful source of suggestions.
			 */
			if (!relativePatchPath || !targetFileName) {
				continue;
			}

			const candidateParts = splitAbsoluteInteractiveTargetPath(candidate);

			if (!candidateParts) {
				continue;
			}

			const candidateDirectorySegments = candidateParts.segments.slice(0, -1);

			if (targetDirectorySegments.length === 0) {
				addTarget(joinInteractiveAbsolutePath(candidateParts, [...candidateDirectorySegments, targetFileName]), 50);
				continue;
			}

			const anchor = findBestDirectoryAnchor(
				candidateDirectorySegments,
				targetDirectorySegments,
				candidateParts.caseInsensitive
			);

			if (anchor) {
				const destination = joinInteractiveAbsolutePath(candidateParts, [
					...candidateDirectorySegments.slice(0, anchor.candidateIndex),
					...targetDirectorySegments.slice(anchor.targetIndex),
					targetFileName,
				]);
				const isExactDirectoryMatch =
					anchor.targetIndex === 0 &&
					anchor.length === targetDirectorySegments.length &&
					anchor.candidateIndex + anchor.length === candidateDirectorySegments.length;

				addTarget(destination, isExactDirectoryMatch ? 10 : 20 + targetDirectorySegments.length - anchor.length);
				continue;
			}

			/*
			 * Keep a low-ranked fallback rooted at attached files. This is less
			 * precise than a directory anchor, but gives the user a useful path
			 * when attachments are the only workspace information available.
			 */
			addTarget(joinInteractiveAbsolutePath(candidateParts, [...candidateDirectorySegments, ...relativeSegments]), 50);
		}

		for (const root of attachmentRoots) {
			if (absolutePatchPath) {
				if (isPathWithinRoot(absolutePatchPath, root)) {
					addTarget(absolutePatchPath, 60);
				}
			} else if (relativePatchPath) {
				addTarget(joinWorkspaceRoot(root, relativePatchPath), 60);
			}
		}

		for (const root of explicitRoots) {
			if (absolutePatchPath) {
				if (isPathWithinRoot(absolutePatchPath, root)) {
					addTarget(absolutePatchPath, 100);
				}
			} else if (relativePatchPath) {
				addTarget(joinWorkspaceRoot(root, relativePatchPath), 100);
			}
		}

		const candidates = [...rankedTargets.values()]
			.toSorted((left, right) => left.rank - right.rank || left.order - right.order)
			.map(target => target.path);

		suggestions.set(file.id, {
			/*
			 * Ranking is only presentation. The UI does not automatically turn
			 * the first candidate into an explicit backend target.
			 */
			targetPath: candidates.length === 1 ? (candidates[0] ?? '') : '',
			candidates: candidates.slice(0, MAX_INTERACTIVE_CANDIDATES),
		});
	}

	return suggestions;
}

function concatenatePatchParts(parts: string[]): string {
	let output = '';

	for (const part of parts) {
		if (!part) {
			continue;
		}

		if (output && !output.endsWith('\n')) {
			output += '\n';
		}

		output += part;
	}

	return ensureTrailingNewline(output);
}

function syntheticHunkPrefix(file: InteractiveDiffFile, targetPath: string): string {
	if (!targetPath) {
		return '';
	}

	const oldPath = file.kind === 'add' ? DEV_NULL : (file.oldPath ?? targetPath);
	const newPath = file.kind === 'delete' ? DEV_NULL : (file.newPath ?? targetPath);

	return sourceFromLines([`--- ${formatHeaderPath(oldPath)}`, `+++ ${formatHeaderPath(newPath)}`]);
}

/**
 * Builds a backend request without validating or rewriting patch semantics.
 *
 * Whole-file requests use the original file section. Hunk requests preserve
 * the original hunk headers and bodies exactly. Do not renumber hunk ranges,
 * repair line counts, or reject fuzzy/LLM-shaped patch content here.
 */
export function buildInteractiveDiffRequest(
	file: InteractiveDiffFile,
	targetPath = '',
	selectedHunkIDs?: string[]
): string {
	if (selectedHunkIDs === undefined) {
		return file.requestText || file.sourceText;
	}

	const selected = new Set(selectedHunkIDs);
	const hunks = file.hunks.filter(hunk => selected.has(hunk.id));

	if (hunks.length === 0 || hunks.length !== selected.size) {
		throw new Error('The selected hunks do not belong to this file section.');
	}

	const prefix = file.hunkPrefixText || syntheticHunkPrefix(file, targetPath);

	return concatenatePatchParts([prefix, ...hunks.map(hunk => hunk.sourceText)]);
}

export function createDiffApplyOutcome(
	phase: DiffApplyPhase,
	status: DiffApplyOutcomeStatus,
	message: string,
	diagnostics: ApplyUnifiedDiffDiagnostic[] = [],
	resolvedTargetPath?: string
): DiffApplyOutcome {
	return {
		phase,
		status,
		message,
		diagnostics: limitInteractiveDiagnostics(diagnostics),
		resolvedTargetPath: resolvedTargetPath?.trim() || undefined,
	};
}

function backendMessage(output: ApplyUnifiedDiffOut, fileMessage: string | undefined, fallback: string): string {
	return fileMessage || output.message || fallback;
}

function responseTargetPath(output: ApplyUnifiedDiffOut): string {
	const file = output.files?.length === 1 ? output.files[0] : undefined;

	return file?.targetPath?.trim() || file?.resolvedPath?.trim() || '';
}

/**
 * Maps backend results to UI states. Backend status is authoritative.
 *
 * Diagnostics are displayed, but do not independently override a successful
 * fuzzy dry run or apply result.
 */
export function interpretScopedApplyResult(output: ApplyUnifiedDiffOut, phase: DiffApplyPhase): DiffApplyOutcome {
	const files = output.files ?? [];
	const file = files.length === 1 ? files[0] : undefined;
	const status = file?.status ?? output.status;
	const diagnostics = limitInteractiveDiagnostics([
		...(output.diagnostics ?? []),
		...(file?.diagnostics ?? []),
		...(files.length > 1
			? [
					createDiagnostic(
						ApplyUnifiedDiffDiagnosticLevel.Warning,
						'unexpected_multiple_file_response',
						'The backend returned multiple file results for a scoped request. The aggregate backend status is shown.'
					),
				]
			: []),
	]);
	const targetPath = responseTargetPath(output);
	const message = backendMessage(output, file?.message, 'The backend returned no message.');

	if (typeof output.dryRun === 'boolean' && output.dryRun !== (phase === 'dry-run')) {
		return createDiffApplyOutcome(
			phase,
			'blocked',
			'The backend response phase did not match the requested review/apply operation.',
			diagnostics,
			targetPath
		);
	}

	if (status === ApplyUnifiedDiffStatus.NeedsInfo) {
		return createDiffApplyOutcome(phase, 'needs-info', message, diagnostics, targetPath);
	}

	if (status === ApplyUnifiedDiffStatus.Error || status === ApplyUnifiedDiffStatus.Conflict) {
		return createDiffApplyOutcome(phase, 'blocked', message, diagnostics, targetPath);
	}

	/*
	 * Terminal backend statuses remain terminal even when an older backend
	 * implementation uses ok:false for AlreadyApplied. The status is the
	 * backend's semantic authority.
	 */
	if (status === ApplyUnifiedDiffStatus.Applied) {
		return createDiffApplyOutcome(phase, 'applied', message, diagnostics, targetPath);
	}

	if (status === ApplyUnifiedDiffStatus.AlreadyApplied) {
		return createDiffApplyOutcome(phase, 'already-applied', message, diagnostics, targetPath);
	}

	// Backend dry-run status is authoritative. A normal successful response
	// is { ok: true, status: Applicable }; do not invert this condition.
	// A normal backend review response is:
	// { ok: true, status: Applicable }.
	//
	// The backend owns fuzzy applicability. Do not invert this check or make
	// a successful backend review look blocked in the UI.
	if (status === ApplyUnifiedDiffStatus.Applicable && output.ok && file?.ok !== false) {
		if (phase === 'dry-run') {
			return createDiffApplyOutcome(phase, 'ready', message, diagnostics, targetPath);
		}
		return createDiffApplyOutcome(
			phase,
			'blocked',
			'The backend reported applicability instead of an applied write result.',
			diagnostics,
			targetPath
		);
	}

	if (!output.ok || file?.ok === false) {
		return createDiffApplyOutcome(phase, 'blocked', message, diagnostics, targetPath);
	}

	return createDiffApplyOutcome(
		phase,
		'blocked',
		message || 'The backend did not report an applicable or applied result.',
		diagnostics,
		targetPath
	);
}
