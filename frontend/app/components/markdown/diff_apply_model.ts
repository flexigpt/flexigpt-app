import type { ApplyUnifiedDiffDiagnostic, ApplyUnifiedDiffOut } from '@/spec/unified_diff';
import { ApplyUnifiedDiffDiagnosticLevel, ApplyUnifiedDiffStatus } from '@/spec/unified_diff';

const DEV_NULL = '/dev/null';

const MAX_INTERACTIVE_DIFF_CHARACTERS = 1_500_000;
const MAX_INTERACTIVE_DIFF_FILES = 100;
export const MAX_INTERACTIVE_HUNK_PROBES = 24;
const MAX_INTERACTIVE_CANDIDATES = 32;
const MAX_INTERACTIVE_DIAGNOSTICS = 32;

type InteractiveDiffFileKind = 'modify' | 'add' | 'delete' | 'rename';

export interface InteractiveDiffHunk {
	id: string;
	header: string;
	suffix: string;
	oldStart: number;
	oldCount: number;
	newStart: number;
	newCount: number;
	lines: string[];
	valid: boolean;
}

export interface InteractiveDiffFile {
	id: string;
	oldPath?: string;
	newPath?: string;
	kind: InteractiveDiffFileKind;
	sourceText: string;
	hunks: InteractiveDiffHunk[];
	addedLines: number;
	deletedLines: number;
	diagnostics: ApplyUnifiedDiffDiagnostic[];
	canApplyWhole: boolean;
	canApplyPartial: boolean;
	needsSyntheticHeaders: boolean;
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
}

interface WorkingDiffSection {
	lines: string[];
	oldPath?: string;
	newPath?: string;
	hunks: InteractiveDiffHunk[];
	diagnostics: ApplyUnifiedDiffDiagnostic[];
	hasHeaders: boolean;
	hasMetadata: boolean;
	binary: boolean;
}

interface ParsedHunk {
	hunk?: InteractiveDiffHunk;
	sourceLines: string[];
	end: number;
	diagnostic?: ApplyUnifiedDiffDiagnostic;
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
				'Additional diagnostics were suppressed to keep the diff UI responsive.'
			)
		);
	}
}

function hasErrorDiagnostics(diagnostics: ApplyUnifiedDiffDiagnostic[]): boolean {
	return diagnostics.some(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Error);
}

function ensureTrailingNewline(value: string): string {
	return value.endsWith('\n') ? value : `${value}\n`;
}

function dedupeInteractiveDiagnostics(
	values: Array<ApplyUnifiedDiffDiagnostic | undefined | null>,
	limit = MAX_INTERACTIVE_DIAGNOSTICS
): ApplyUnifiedDiffDiagnostic[] {
	const output: ApplyUnifiedDiffDiagnostic[] = [];
	const seen = new Set<string>();
	let truncated = false;

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

		if (seen.has(key)) {
			continue;
		}

		seen.add(key);

		if (output.length >= limit) {
			truncated = true;
			continue;
		}

		output.push({
			level,
			code: code || undefined,
			message,
		});
	}

	if (truncated) {
		output.push(
			createDiagnostic(
				ApplyUnifiedDiffDiagnosticLevel.Warning,
				'diagnostics_truncated',
				'Additional diagnostics were suppressed to keep the diff UI responsive.'
			)
		);
	}

	return output;
}

export function normalizeInteractiveTargetPath(value: string | undefined | null): string {
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
	const raw = normalizePatchPath(value);

	if (!raw || raw === DEV_NULL || normalizeInteractiveTargetPath(raw)) {
		return '';
	}

	if (raw.startsWith('/') || /^[A-Za-z]:/.test(raw) || raw.startsWith('//')) {
		return '';
	}

	const parts = raw.split('/').filter(Boolean);

	if (parts.length === 0 || parts.some(part => part === '.' || part === '..')) {
		return '';
	}

	return parts.join('/');
}

function patchPathIdentity(value: string | undefined): string {
	const absolute = targetPathIdentity(value);

	if (absolute) {
		return absolute;
	}

	return normalizePatchPath(value);
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
	return normalizeInteractiveTargetPath(root || '/');
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

	for (let index = 1; index < value.length; index += 1) {
		const character = value[index];

		if (character === '"') {
			return {
				token,
				rest: value.slice(index + 1).trimStart(),
			};
		}

		if (character !== '\\') {
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

			token += String.fromCodePoint(Number.parseInt(octal, 8));
			index = cursor - 1;
			continue;
		}

		const escapes: Record<string, string> = {
			'"': '"',
			'\\': '\\',
			n: '\n',
			r: '\r',
			t: '\t',
		};

		token += escapes[next] ?? next;
		index += 1;
	}

	throw new Error('Unterminated quoted diff path.');
}

function readHeaderPath(input: string): string {
	const value = input.trimStart();

	if (!value) {
		throw new Error('Missing diff header path.');
	}

	if (value.startsWith('"')) {
		return readDiffToken(value).token;
	}

	const tabIndex = value.indexOf('\t');
	const path = tabIndex >= 0 ? value.slice(0, tabIndex) : (value.split(/\s+/, 1)[0] ?? '');

	if (!path) {
		throw new Error('Missing diff header path.');
	}

	return path;
}

function readLoosePath(input: string): string {
	const value = input.trim();

	if (!value) {
		return '';
	}

	return value.startsWith('"') ? readDiffToken(value).token : value;
}

function normalizeHeaderPair(oldPath: string, newPath: string): [string, string] {
	const oldHasGitPrefix = oldPath.startsWith('a/');
	const newHasGitPrefix = newPath.startsWith('b/');

	if ((oldHasGitPrefix && (newHasGitPrefix || newPath === DEV_NULL)) || (oldPath === DEV_NULL && newHasGitPrefix)) {
		return [oldHasGitPrefix ? oldPath.slice(2) : oldPath, newHasGitPrefix ? newPath.slice(2) : newPath];
	}

	return [oldPath, newPath];
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

function hunksDoNotOverlap(hunks: InteractiveDiffHunk[]): boolean {
	let previousEnd = -1;
	let previousInsertion: number | undefined;

	for (const hunk of hunks) {
		const start = hunk.oldCount === 0 ? hunk.oldStart : hunk.oldStart - 1;

		if (start < previousEnd || (hunk.oldCount === 0 && previousInsertion === start)) {
			return false;
		}

		previousEnd = Math.max(previousEnd, start + hunk.oldCount);
		previousInsertion = hunk.oldCount === 0 ? start : undefined;
	}

	return true;
}

function parseHunkAt(lines: string[], start: number, hunkID: string): ParsedHunk {
	const header = lines[start] ?? '';
	const match = /^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$/.exec(header);

	if (!match?.[1] || !match[3]) {
		return {
			sourceLines: [header],
			end: start + 1,
			diagnostic: createDiagnostic(
				ApplyUnifiedDiffDiagnosticLevel.Error,
				'invalid_hunk_header',
				`Invalid hunk header: ${header}`
			),
		};
	}

	const oldStart = Number(match[1]);
	const oldCount = match[2] === undefined ? 1 : Number(match[2]);
	const newStart = Number(match[3]);
	const newCount = match[4] === undefined ? 1 : Number(match[4]);

	if (
		![oldStart, oldCount, newStart, newCount].every(n => {
			return Number.isSafeInteger(n);
		}) ||
		oldStart < 0 ||
		newStart < 0 ||
		oldCount < 0 ||
		newCount < 0 ||
		(oldCount > 0 && oldStart === 0) ||
		(newCount > 0 && newStart === 0) ||
		(oldCount === 0 && newCount === 0)
	) {
		return {
			sourceLines: [header],
			end: start + 1,
			diagnostic: createDiagnostic(
				ApplyUnifiedDiffDiagnosticLevel.Error,
				'invalid_hunk_range',
				`Invalid hunk range: ${header}`
			),
		};
	}

	let oldRemaining = oldCount;
	let newRemaining = newCount;
	let cursor = start + 1;
	let previousWasPayload = false;
	let valid = true;
	const body: string[] = [];

	while (cursor < lines.length) {
		const line = lines[cursor] ?? '';

		if (line === '\\ No newline at end of file') {
			if (!previousWasPayload) {
				valid = false;
			}

			body.push(line);
			previousWasPayload = false;
			cursor += 1;
			continue;
		}

		if (oldRemaining === 0 && newRemaining === 0) {
			break;
		}

		const prefix = line[0];

		if (prefix === ' ' && oldRemaining > 0 && newRemaining > 0) {
			oldRemaining -= 1;
			newRemaining -= 1;
		} else if (prefix === '-' && oldRemaining > 0) {
			oldRemaining -= 1;
		} else if (prefix === '+' && newRemaining > 0) {
			newRemaining -= 1;
		} else {
			break;
		}

		body.push(line);
		previousWasPayload = true;
		cursor += 1;
	}

	if (oldRemaining !== 0 || newRemaining !== 0) {
		valid = false;
	}

	const hunk: InteractiveDiffHunk = {
		id: hunkID,
		header,
		suffix: match[5] ?? '',
		oldStart,
		oldCount,
		newStart,
		newCount,
		lines: body,
		valid,
	};

	return {
		hunk,
		sourceLines: [header, ...body],
		end: cursor,
		diagnostic: valid
			? undefined
			: createDiagnostic(
					ApplyUnifiedDiffDiagnosticLevel.Error,
					'incomplete_hunk',
					`Incomplete or malformed hunk: ${header}`
				),
	};
}

function createWorkingSection(): WorkingDiffSection {
	return {
		lines: [],
		hunks: [],
		diagnostics: [],
		hasHeaders: false,
		hasMetadata: false,
		binary: false,
	};
}

function addOutsideHunkLine(section: WorkingDiffSection, line: string): void {
	section.lines.push(line);

	if (!line.trim() || line.startsWith('index ') || /^=+$/.test(line)) {
		return;
	}

	if (line === 'GIT binary patch' || line.startsWith('Binary files ')) {
		section.binary = true;
		appendDiagnostic(
			section.diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Error,
			'binary_patch',
			'Binary patches are not supported by the interactive text diff UI.'
		);
		return;
	}

	if (line.startsWith('new file mode ')) {
		section.oldPath = DEV_NULL;
		section.hasMetadata = true;
		return;
	}

	if (line.startsWith('deleted file mode ')) {
		section.newPath = DEV_NULL;
		section.hasMetadata = true;
		return;
	}

	if (
		line.startsWith('old mode ') ||
		line.startsWith('new mode ') ||
		line.startsWith('similarity index ') ||
		line.startsWith('dissimilarity index ') ||
		line.startsWith('copy from ') ||
		line.startsWith('copy to ')
	) {
		section.hasMetadata = true;
		return;
	}

	if (line.startsWith('rename from ')) {
		section.hasMetadata = true;

		try {
			section.oldPath = readLoosePath(line.slice('rename from '.length));
		} catch (error) {
			appendDiagnostic(
				section.diagnostics,
				ApplyUnifiedDiffDiagnosticLevel.Error,
				'invalid_path',
				error instanceof Error ? error.message : 'Invalid rename source path.'
			);
		}

		return;
	}

	if (line.startsWith('rename to ')) {
		section.hasMetadata = true;

		try {
			section.newPath = readLoosePath(line.slice('rename to '.length));
		} catch (error) {
			appendDiagnostic(
				section.diagnostics,
				ApplyUnifiedDiffDiagnosticLevel.Error,
				'invalid_path',
				error instanceof Error ? error.message : 'Invalid rename target path.'
			);
		}

		return;
	}

	appendDiagnostic(
		section.diagnostics,
		ApplyUnifiedDiffDiagnosticLevel.Error,
		'unsupported_section_text',
		`Unsupported text outside a hunk: ${line}`
	);
}

function finishWorkingSection(section: WorkingDiffSection, sectionNumber: number): InteractiveDiffFile | undefined {
	if (section.lines.length === 0 && section.hunks.length === 0) {
		return undefined;
	}

	if (section.hunks.length === 0) {
		appendDiagnostic(
			section.diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Error,
			'no_hunks',
			'The file section contains no complete textual hunks.'
		);
	}

	if (!section.hasHeaders && section.hunks.length > 0) {
		appendDiagnostic(
			section.diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Warning,
			'hunk_only_patch',
			'This hunk-only patch will use the selected target path to create synthetic file headers.'
		);
	}

	if (section.hasHeaders && (!section.oldPath || !section.newPath)) {
		appendDiagnostic(
			section.diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Error,
			'missing_paths',
			'The file section is missing one of its --- or +++ paths.'
		);
	}

	if (section.oldPath === DEV_NULL && section.newPath === DEV_NULL) {
		appendDiagnostic(
			section.diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Error,
			'invalid_paths',
			'Both file paths cannot be /dev/null.'
		);
	}

	const id = `section-${sectionNumber}`;
	const hunks = section.hunks.map((hunk, index) => ({
		...hunk,
		id: `${id}:hunk-${index + 1}`,
	}));
	const diagnostics = dedupeInteractiveDiagnostics(section.diagnostics);
	const kind = getFileKind(section.oldPath, section.newPath);
	const validHunks = hunks.length > 0 && hunks.every(hunk => hunk.valid);
	const canApplyWhole = validHunks && !section.binary && !hasErrorDiagnostics(diagnostics);

	return {
		id,
		oldPath: section.oldPath,
		newPath: section.newPath,
		kind,
		sourceText: ensureTrailingNewline(section.lines.join('\n')),
		hunks,
		addedLines: hunks.reduce((sum, hunk) => sum + hunk.lines.filter(line => line.startsWith('+')).length, 0),
		deletedLines: hunks.reduce((sum, hunk) => sum + hunk.lines.filter(line => line.startsWith('-')).length, 0),
		diagnostics,
		canApplyWhole,
		canApplyPartial:
			canApplyWhole && kind === 'modify' && hunks.length > 1 && !section.hasMetadata && hunksDoNotOverlap(hunks),
		needsSyntheticHeaders: !section.hasHeaders,
	};
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

		const file = finishWorkingSection(current, files.length + 1);

		if (file) {
			files.push(file);
		}

		current = undefined;
	};

	for (let index = 0; index < lines.length;) {
		const line = lines[index] ?? '';

		if (line.startsWith('diff --git ')) {
			flush();
			current = createWorkingSection();
			current.lines.push(line);

			try {
				const first = readDiffToken(line.slice('diff --git '.length));
				const second = readDiffToken(first.rest);

				if (!first.token || !second.token) {
					throw new Error('diff --git must include both paths.');
				}

				[current.oldPath, current.newPath] = normalizeHeaderPair(first.token, second.token);
			} catch (error) {
				appendDiagnostic(
					current.diagnostics,
					ApplyUnifiedDiffDiagnosticLevel.Error,
					'invalid_path',
					error instanceof Error ? error.message : 'Invalid diff --git path.'
				);
			}

			index += 1;
			continue;
		}

		if (line.startsWith('Index: ')) {
			flush();
			current = createWorkingSection();
			current.lines.push(line);

			try {
				const path = readLoosePath(line.slice('Index: '.length));
				current.oldPath = path;
				current.newPath = path;
			} catch (error) {
				appendDiagnostic(
					current.diagnostics,
					ApplyUnifiedDiffDiagnosticLevel.Error,
					'invalid_path',
					error instanceof Error ? error.message : 'Invalid Index path.'
				);
			}

			index += 1;
			continue;
		}

		const nextLine = lines[index + 1] ?? '';
		const hasPairedHeaders =
			(line.startsWith('--- ') || line.startsWith('---\t')) &&
			(nextLine.startsWith('+++ ') || nextLine.startsWith('+++\t'));

		if (hasPairedHeaders) {
			if (current?.hasHeaders || (current?.hunks.length ?? 0) > 0) {
				flush();
			}

			current ??= createWorkingSection();
			current.lines.push(line, nextLine);
			current.hasHeaders = true;

			try {
				[current.oldPath, current.newPath] = normalizeHeaderPair(
					readHeaderPath(line.slice(4)),
					readHeaderPath(nextLine.slice(4))
				);
			} catch (error) {
				appendDiagnostic(
					current.diagnostics,
					ApplyUnifiedDiffDiagnosticLevel.Error,
					'invalid_path',
					error instanceof Error ? error.message : 'Invalid unified diff header path.'
				);
			}

			index += 2;
			continue;
		}

		if (line.startsWith('@@')) {
			current ??= createWorkingSection();

			const parsed = parseHunkAt(lines, index, `pending-hunk-${current.hunks.length + 1}`);

			for (const sourceLine of parsed.sourceLines) {
				current.lines.push(sourceLine);
			}

			if (parsed.hunk) {
				current.hunks.push(parsed.hunk);
			}

			if (parsed.diagnostic) {
				appendDiagnostic(
					current.diagnostics,
					parsed.diagnostic.level,
					parsed.diagnostic.code ?? 'invalid_hunk',
					parsed.diagnostic.message
				);
			}

			index = parsed.end;
			continue;
		}

		if (!current) {
			// Ignore mail headers and other patch preamble text. It is not sent
			// with a scoped file request, and should not block valid sections.
			index += 1;
			continue;
		}

		addOutsideHunkLine(current, line);
		index += 1;
	}

	flush();

	return {
		files,
		diagnostics,
	};
}

function parseOpenAIPath(value: string): string {
	const trimmed = value.trim();

	if (!trimmed) {
		throw new Error('Missing OpenAI patch path.');
	}

	return trimmed.startsWith('"') ? readDiffToken(trimmed).token : trimmed;
}

function unsupportedOpenAISection(
	id: string,
	kind: 'add' | 'update' | 'delete',
	path: string | undefined,
	sourceText: string,
	message: string
): InteractiveDiffFile {
	return {
		id,
		oldPath: kind === 'add' ? DEV_NULL : path,
		newPath: kind === 'delete' ? DEV_NULL : path,
		kind: kind === 'add' ? 'add' : kind === 'delete' ? 'delete' : 'modify',
		sourceText: ensureTrailingNewline(sourceText),
		hunks: [],
		addedLines: 0,
		deletedLines: 0,
		diagnostics: [createDiagnostic(ApplyUnifiedDiffDiagnosticLevel.Error, 'unsupported_openai_section', message)],
		canApplyWhole: false,
		canApplyPartial: false,
		needsSyntheticHeaders: false,
	};
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
		pathText: string;
	}> = [];
	const diagnostics: ApplyUnifiedDiffDiagnostic[] = [];
	const endIndex = lines.findIndex(line => line.trim() === '*** End Patch');
	const hasBegin = lines.some(line => line.trim() === '*** Begin Patch');

	for (let index = 0; index < lines.length; index += 1) {
		const match = headerPattern.exec(lines[index] ?? '');

		if (!match?.[1] || !match[2]) {
			continue;
		}

		const normalizedKind = match[1].toLowerCase();

		sections.push({
			index,
			kind: normalizedKind === 'add' ? 'add' : normalizedKind === 'delete' ? 'delete' : 'update',
			pathText: match[2],
		});
	}

	if (hasBegin && endIndex < 0) {
		appendDiagnostic(
			diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Error,
			'incomplete_openai_patch',
			'The OpenAI patch is missing its End Patch marker.'
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
		const body = lines.slice(section.index + 1, sectionEnd);
		const id = `section-${files.length + 1}`;
		let path: string | undefined;

		try {
			path = parseOpenAIPath(section.pathText);
		} catch (error) {
			files.push(
				unsupportedOpenAISection(
					id,
					section.kind,
					undefined,
					lines.slice(section.index, sectionEnd).join('\n'),
					error instanceof Error ? error.message : 'Invalid OpenAI patch path.'
				)
			);
			continue;
		}

		const isSupportedAddBody =
			section.kind === 'add' &&
			body.length > 0 &&
			body.every(line => line.startsWith('+') || line === '\\ No newline at end of file');

		if (!isSupportedAddBody) {
			files.push(
				unsupportedOpenAISection(
					id,
					section.kind,
					path,
					lines.slice(section.index, sectionEnd).join('\n'),
					section.kind === 'add'
						? 'This Add File section has an invalid body and cannot be safely converted.'
						: 'Only OpenAI Add File sections can be safely converted by the interactive diff UI.'
				)
			);
			continue;
		}

		const addedLineCount = body.filter(line => line.startsWith('+')).length;

		if (addedLineCount === 0) {
			files.push(
				unsupportedOpenAISection(
					id,
					section.kind,
					path,
					lines.slice(section.index, sectionEnd).join('\n'),
					'Creating an empty OpenAI Add File section is not safely representable as a textual unified diff.'
				)
			);
			continue;
		}

		const headerPath = normalizeInteractiveTargetPath(path) || path.startsWith('b/') ? path : `b/${path}`;
		const convertedText = [
			'--- /dev/null',
			`+++ ${formatHeaderPath(headerPath)}`,
			`@@ -0,0 +1,${addedLineCount} @@`,
			...body,
		].join('\n');

		const converted = parseStandardDiff(convertedText).files[0];

		if (!converted || !converted.canApplyWhole) {
			files.push(
				unsupportedOpenAISection(
					id,
					section.kind,
					path,
					lines.slice(section.index, sectionEnd).join('\n'),
					'The converted Add File section could not be verified as a complete unified diff.'
				)
			);
			continue;
		}

		files.push({
			...converted,
			id,
			hunks: converted.hunks.map((hunk, hunkIndex) => ({
				...hunk,
				id: `${id}:hunk-${hunkIndex + 1}`,
			})),
			diagnostics: dedupeInteractiveDiagnostics([
				...converted.diagnostics,
				createDiagnostic(
					ApplyUnifiedDiffDiagnosticLevel.Info,
					'converted_openai_add_file',
					'This OpenAI Add File section is converted to standard unified diff before checking or applying.'
				),
			]),
		});
	}

	return {
		files,
		diagnostics,
	};
}

function looksLikeInteractiveDiff(value: string, language = ''): boolean {
	const normalizedLanguage = language.trim().toLowerCase();

	if (normalizedLanguage === 'diff' || normalizedLanguage === 'patch' || normalizedLanguage === 'udiff') {
		return true;
	}

	return (
		/^diff --git\s+/m.test(value) ||
		/^Index:\s+/m.test(value) ||
		(/^---[ \t]/m.test(value) && /^\+\+\+[ \t]/m.test(value)) ||
		/^@@ -\d+(?:,\d+)? \+\d+(?:,\d+)? @@/m.test(value) ||
		/^\*\*\*\s+(?:Begin Patch|(?:Add|Update|Delete)\s+File:)/m.test(value)
	);
}

export function parseInteractiveDiff(value: string, language = ''): ParsedInteractiveDiff {
	const text = value
		.replaceAll('\r\n', '\n')
		.replaceAll('\r', '\n')
		.replace(/^\uFEFF/, '');
	const isOpenAIPatch = /^\*\*\*\s+(?:Begin Patch|(?:Add|Update|Delete)\s+File:)/m.test(text);
	const isDiffLike = looksLikeInteractiveDiff(text, language);

	if (text.length > MAX_INTERACTIVE_DIFF_CHARACTERS) {
		return {
			isDiffLike,
			isOpenAIPatch,
			files: [],
			diagnostics: [
				createDiagnostic(
					ApplyUnifiedDiffDiagnosticLevel.Error,
					'diff_too_large',
					`This diff is larger than ${MAX_INTERACTIVE_DIFF_CHARACTERS.toLocaleString()} characters and is not opened in the interactive apply UI.`
				),
			],
		};
	}

	const parsed = isOpenAIPatch ? parseOpenAIPatch(text) : parseStandardDiff(text);
	let files = parsed.files;
	const diagnostics = [...parsed.diagnostics];

	if (files.length === 0) {
		appendDiagnostic(
			diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Error,
			'no_file_sections',
			'No complete file sections could be extracted from this patch.'
		);
	}

	if (files.length > MAX_INTERACTIVE_DIFF_FILES) {
		files = files.slice(0, MAX_INTERACTIVE_DIFF_FILES);
		appendDiagnostic(
			diagnostics,
			ApplyUnifiedDiffDiagnosticLevel.Error,
			'too_many_file_sections',
			`This patch contains more than ${MAX_INTERACTIVE_DIFF_FILES} file sections and is not opened in the interactive apply UI.`
		);
	}

	return {
		isDiffLike,
		isOpenAIPatch,
		files,
		diagnostics: dedupeInteractiveDiagnostics(diagnostics),
	};
}

export function getInteractiveDiffTargetPath(file: InteractiveDiffFile): string {
	if (file.kind === 'add') {
		return file.newPath ?? '';
	}

	if (file.kind === 'delete' || file.kind === 'rename') {
		return file.oldPath ?? '';
	}

	return file.oldPath ?? file.newPath ?? '';
}

export function isNewInteractiveDiffFile(file: InteractiveDiffFile): boolean {
	return file.kind === 'add' && file.oldPath === DEV_NULL && !!file.newPath;
}

export function inferInteractiveDiffTargets(
	files: InteractiveDiffFile[],
	candidatePaths: string[] = [],
	workspaceRoots: string[] = []
): Map<string, InteractiveDiffTargetSuggestion> {
	const externalCandidates = normalizeInteractiveTargetPaths(candidatePaths);
	const explicitRoots = normalizeInteractiveTargetPaths(workspaceRoots);
	const inferredRoots: string[] = [];

	for (const file of files) {
		const relativePath = safeRelativePatchPath(getInteractiveDiffTargetPath(file));

		if (!relativePath) {
			continue;
		}

		for (const candidate of externalCandidates) {
			const root = rootForCandidatePath(candidate, relativePath);

			if (root) {
				inferredRoots.push(root);
			}
		}
	}

	const roots = normalizeInteractiveTargetPaths([...explicitRoots, ...inferredRoots]);

	const suggestions = new Map<string, InteractiveDiffTargetSuggestion>();

	for (const file of files) {
		const patchPath = getInteractiveDiffTargetPath(file);
		const absolutePatchPath = normalizeInteractiveTargetPath(patchPath);
		const relativePatchPath = safeRelativePatchPath(patchPath);

		const directCandidates = absolutePatchPath
			? externalCandidates.filter(candidate => targetPathIdentity(candidate) === targetPathIdentity(absolutePatchPath))
			: relativePatchPath
				? externalCandidates.filter(candidate => pathEndsWithRelativePath(candidate, relativePatchPath))
				: [];

		const rootedCandidates = absolutePatchPath
			? roots.filter(root => isPathWithinRoot(absolutePatchPath, root)).map(() => absolutePatchPath)
			: relativePatchPath
				? roots.map(root => joinWorkspaceRoot(root, relativePatchPath)).filter(Boolean)
				: [];

		const allCandidates = normalizeInteractiveTargetPaths([...directCandidates, ...rootedCandidates]);

		const targetPath =
			directCandidates.length === 1
				? (directCandidates[0] ?? '')
				: allCandidates.length === 1
					? (allCandidates[0] ?? '')
					: '';

		const visibleCandidates = allCandidates.slice(0, MAX_INTERACTIVE_CANDIDATES);

		if (targetPath && !visibleCandidates.includes(targetPath)) {
			visibleCandidates.unshift(targetPath);
			visibleCandidates.splice(MAX_INTERACTIVE_CANDIDATES);
		}

		suggestions.set(file.id, {
			targetPath,
			candidates: visibleCandidates,
		});
	}

	return suggestions;
}

function formatRange(start: number, count: number): string {
	return count === 1 ? `${start}` : `${start},${count}`;
}

function renderSyntheticPatch(
	file: InteractiveDiffFile,
	targetPath: string,
	hunks: InteractiveDiffHunk[],
	adjustNewPositions: boolean
): string {
	const oldPath = file.kind === 'add' ? DEV_NULL : (file.oldPath ?? targetPath);
	const newPath = file.kind === 'delete' ? DEV_NULL : (file.newPath ?? targetPath);
	let selectedDelta = 0;

	const renderedHunks = hunks.map(hunk => {
		let newStart = hunk.newStart;

		if (adjustNewPositions) {
			newStart = hunk.oldStart + selectedDelta;

			if (hunk.oldCount === 0) {
				newStart += 1;
			}

			if (hunk.newCount === 0) {
				newStart -= 1;
			}

			selectedDelta += hunk.newCount - hunk.oldCount;
		}

		const header = adjustNewPositions
			? `@@ -${formatRange(hunk.oldStart, hunk.oldCount)} +${formatRange(newStart, hunk.newCount)} @@${hunk.suffix}`
			: hunk.header;

		return [header, ...hunk.lines].join('\n');
	});

	return ensureTrailingNewline(
		[`--- ${formatHeaderPath(oldPath)}`, `+++ ${formatHeaderPath(newPath)}`, ...renderedHunks].join('\n')
	);
}

export function buildInteractiveDiffRequest(
	file: InteractiveDiffFile,
	targetPath: string,
	selectedHunkIDs?: string[]
): string {
	const normalizedTargetPath = normalizeInteractiveTargetPath(targetPath);

	if (!normalizedTargetPath) {
		throw new Error('Choose an absolute target file path before checking or applying this section.');
	}

	if (!file.canApplyWhole) {
		throw new Error('This file section has structural errors and cannot be applied.');
	}

	let selectedHunks = file.hunks;
	let shouldAdjustNewPositions = false;

	if (selectedHunkIDs !== undefined) {
		if (!file.canApplyPartial) {
			throw new Error('Partial application is available only for ordinary, non-overlapping text modifications.');
		}

		const selected = new Set(selectedHunkIDs);
		selectedHunks = file.hunks.filter(hunk => selected.has(hunk.id));

		if (selectedHunks.length === 0 || selectedHunks.length !== selected.size) {
			throw new Error('The selected hunks do not belong to this file section.');
		}

		shouldAdjustNewPositions = true;
	}

	const requestText =
		selectedHunkIDs === undefined && !file.needsSyntheticHeaders
			? file.sourceText
			: renderSyntheticPatch(file, normalizedTargetPath, selectedHunks, shouldAdjustNewPositions);

	const verification = parseInteractiveDiff(requestText, 'diff');
	const verifiedFile = verification.files[0];
	const expectedHunkCount = selectedHunks.length;

	if (
		hasErrorDiagnostics(verification.diagnostics) ||
		verification.files.length !== 1 ||
		!verifiedFile?.canApplyWhole ||
		verifiedFile.hunks.length !== expectedHunkCount
	) {
		throw new Error('The scoped request could not be verified as exactly one complete file patch.');
	}

	return requestText;
}

export function createDiffApplyOutcome(
	phase: DiffApplyPhase,
	status: DiffApplyOutcomeStatus,
	message: string,
	diagnostics: ApplyUnifiedDiffDiagnostic[] = []
): DiffApplyOutcome {
	return {
		phase,
		status,
		message,
		diagnostics: dedupeInteractiveDiagnostics(diagnostics),
	};
}

function terminalStatus(status: ApplyUnifiedDiffStatus | undefined): boolean {
	return status === ApplyUnifiedDiffStatus.Applied || status === ApplyUnifiedDiffStatus.AlreadyApplied;
}

function backendMessage(
	output: ApplyUnifiedDiffOut,
	file: ApplyUnifiedDiffOut['files'] extends Array<infer T> | undefined ? T | undefined : never,
	fallback: string
): string {
	return file?.message || output.message || fallback;
}

export function interpretScopedApplyResult(
	output: ApplyUnifiedDiffOut,
	phase: DiffApplyPhase,
	targetPath: string
): DiffApplyOutcome {
	const files = output.files ?? [];
	const file = files[0];
	const diagnostics = dedupeInteractiveDiagnostics([...(output.diagnostics ?? []), ...(file?.diagnostics ?? [])]);
	const status = file?.status ?? output.status;
	const requestedDryRun = phase === 'dry-run';
	const returnedTargetPath = normalizeInteractiveTargetPath(file?.targetPath);

	const blocked = (message: string, needsInfo = false) =>
		createDiffApplyOutcome(phase, needsInfo ? 'needs-info' : 'blocked', message, diagnostics);

	if (files.length > 1 || (output.summary?.files ?? 0) > 1) {
		return blocked('The backend returned multiple file results for a one-file request.');
	}

	if (typeof output.dryRun === 'boolean' && output.dryRun !== requestedDryRun) {
		return blocked('The backend response does not match the requested dry-run/write phase.');
	}

	if (returnedTargetPath && targetPathIdentity(returnedTargetPath) !== targetPathIdentity(targetPath)) {
		return blocked('The backend reported a different target path. No subsequent write was submitted.');
	}

	if (hasErrorDiagnostics(diagnostics)) {
		return blocked(backendMessage(output, file, 'The backend reported an error for this file section.'));
	}

	if (!file?.ok) {
		return blocked(backendMessage(output, file, 'The backend rejected this file section.'));
	}

	if (output.status === ApplyUnifiedDiffStatus.Error || output.status === ApplyUnifiedDiffStatus.Conflict) {
		return blocked(backendMessage(output, file, 'The backend reported a blocked file section.'));
	}

	if (output.status === ApplyUnifiedDiffStatus.NeedsInfo || status === ApplyUnifiedDiffStatus.NeedsInfo) {
		return blocked(backendMessage(output, file, 'The backend needs more information for this file section.'), true);
	}

	if (!output.ok && !terminalStatus(status)) {
		return blocked(backendMessage(output, file, 'The backend did not report a successful result.'));
	}

	if (requestedDryRun) {
		if (status === ApplyUnifiedDiffStatus.Applicable) {
			return createDiffApplyOutcome(
				phase,
				'ready',
				backendMessage(output, file, 'Dry run succeeded for this target.'),
				diagnostics
			);
		}

		if (status === ApplyUnifiedDiffStatus.AlreadyApplied) {
			return createDiffApplyOutcome(
				phase,
				'already-applied',
				backendMessage(output, file, 'These changes are already present.'),
				diagnostics
			);
		}

		return blocked(backendMessage(output, file, 'The file section is not currently applicable.'));
	}

	if (status === ApplyUnifiedDiffStatus.Applied) {
		return createDiffApplyOutcome(
			phase,
			'applied',
			backendMessage(output, file, 'The file section was applied.'),
			diagnostics
		);
	}

	if (status === ApplyUnifiedDiffStatus.AlreadyApplied) {
		return createDiffApplyOutcome(
			phase,
			'already-applied',
			backendMessage(output, file, 'These changes are already present.'),
			diagnostics
		);
	}

	return blocked(
		backendMessage(output, file, 'The backend did not report a successful write. Recheck the file before retrying.')
	);
}
