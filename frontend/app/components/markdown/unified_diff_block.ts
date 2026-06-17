import type { ApplyUnifiedDiffFileTarget, ApplyUnifiedDiffOut } from '@/spec/unified_diff';

interface ParsedUnifiedDiffFileForUI {
	fileKey: string;
	oldPath?: string;
	newPath?: string;
	hunks: number;
	addedLines: number;
	deletedLines: number;
	candidatePaths: string[];
	targetPath?: string;
}

export interface ParsedUnifiedDiffForUI {
	isDiffLike: boolean;
	files: ParsedUnifiedDiffFileForUI[];
	hunks: number;
	addedLines: number;
	deletedLines: number;
	diagnostics: string[];
}

const DEV_NULL = '/dev/null';

function isUnifiedDiffLanguage(language: string): boolean {
	const normalized = language.trim().toLowerCase();
	return normalized === 'diff' || normalized === 'patch' || normalized === 'udiff';
}

export function looksLikeUnifiedDiff(value: string, language = ''): boolean {
	const text = value.replace(/\r\n/g, '\n').replace(/\r/g, '\n');

	if (isUnifiedDiffLanguage(language)) return true;
	if (/^diff --git\s+/m.test(text)) return true;
	if (/^Index:\s+/m.test(text)) return true;
	if (/^@@\s*-?\d+(?:,\d+)?\s*\+?\d+(?:,\d+)?\s*@@/m.test(text)) return true;

	return /^---\s+/m.test(text) && /^\+\+\+\s+/m.test(text);
}

export function parseUnifiedDiffForUI(value: string, language = ''): ParsedUnifiedDiffForUI {
	const text = value
		.replace(/\r\n/g, '\n')
		.replace(/\r/g, '\n')
		.replace(/^\ufeff/, '');
	const lines = text.split('\n');

	const files: ParsedUnifiedDiffFileForUI[] = [];
	const diagnostics: string[] = [];

	let current: ParsedUnifiedDiffFileForUI | null = null;
	let inHunk = false;

	const pushCurrent = () => {
		if (!current) return;
		if (current.oldPath || current.newPath || current.hunks > 0) {
			current.candidatePaths = uniqueStrings(current.candidatePaths);
			files.push(current);
		}
		current = null;
		inHunk = false;
	};

	const ensureCurrent = () => {
		if (!current) {
			current = {
				fileKey: `file-${files.length + 1}`,
				hunks: 0,
				addedLines: 0,
				deletedLines: 0,
				candidatePaths: [],
			};
		}
		return current;
	};

	for (let i = 0; i < lines.length; i += 1) {
		const line = lines[i];

		if (line.startsWith('diff --git ')) {
			pushCurrent();

			const rest = line.slice('diff --git '.length).trim();
			const first = readDiffPathToken(rest);
			const second = readDiffPathToken(first.rest);

			const oldPath = stripGitPathPrefix(normalizeDiffPathToken(first.token), 'a');
			const newPath = stripGitPathPrefix(normalizeDiffPathToken(second.token), 'b');

			current = {
				fileKey: `file-${files.length + 1}`,
				oldPath: oldPath || undefined,
				newPath: newPath || undefined,
				hunks: 0,
				addedLines: 0,
				deletedLines: 0,
				candidatePaths: uniqueStrings([oldPath, newPath].filter(isUsablePatchPath)),
			};
			inHunk = false;
			continue;
		}

		if (line.startsWith('Index: ')) {
			pushCurrent();

			const p = normalizeDiffPathToken(line.slice('Index: '.length).trim());
			current = {
				fileKey: `file-${files.length + 1}`,
				oldPath: p || undefined,
				newPath: p || undefined,
				hunks: 0,
				addedLines: 0,
				deletedLines: 0,
				candidatePaths: isUsablePatchPath(p) ? [p] : [],
			};
			inHunk = false;
			continue;
		}

		if (line.startsWith('--- ') && i + 1 < lines.length && lines[i + 1].startsWith('+++ ')) {
			if (!current || current.hunks > 0) {
				pushCurrent();
				current = {
					fileKey: `file-${files.length + 1}`,
					hunks: 0,
					addedLines: 0,
					deletedLines: 0,
					candidatePaths: [],
				};
			}

			const oldPathRaw = parseDiffHeaderPath(lines[i].slice(4));
			const newPathRaw = parseDiffHeaderPath(lines[i + 1].slice(4));
			const [oldPath, newPath] = normalizeUnifiedHeaderPair(oldPathRaw, newPathRaw);

			current.oldPath = oldPath || undefined;
			current.newPath = newPath || undefined;
			current.candidatePaths = uniqueStrings(
				[...(current.candidatePaths ?? []), oldPath, newPath].filter(isUsablePatchPath)
			);

			i += 1;
			inHunk = false;
			continue;
		}

		if (line.startsWith('@@')) {
			const file = ensureCurrent();
			file.hunks += 1;
			inHunk = true;

			if (!/^@@\s+-\d+(?:,\d+)?\s+\+\d+(?:,\d+)?\s+@@/.test(line)) {
				diagnostics.push(`Non-standard hunk header detected: ${line}`);
			}
			continue;
		}

		if (inHunk && current) {
			if (line.startsWith('+') && !line.startsWith('+++')) {
				current.addedLines += 1;
			} else if (line.startsWith('-') && !line.startsWith('---')) {
				current.deletedLines += 1;
			}
		}
	}

	pushCurrent();

	return {
		isDiffLike: looksLikeUnifiedDiff(value, language),
		files,
		hunks: files.reduce((sum, file) => sum + file.hunks, 0),
		addedLines: files.reduce((sum, file) => sum + file.addedLines, 0),
		deletedLines: files.reduce((sum, file) => sum + file.deletedLines, 0),
		diagnostics,
	};
}

export function buildEditableTargetsFromOutput(
	output: ApplyUnifiedDiffOut | undefined,
	fallback: ParsedUnifiedDiffForUI
): EditableUnifiedDiffTarget[] {
	const byKey = new Map<string, EditableUnifiedDiffTarget>();

	for (const target of output?.fileTargets ?? []) {
		const key = target.fileKey || `${target.oldPath ?? ''}\u0000${target.newPath ?? ''}\u0000${target.targetPath}`;
		if (!key) continue;

		byKey.set(key, {
			fileKey: target.fileKey,
			oldPath: target.oldPath,
			newPath: target.newPath,
			targetPath: target.targetPath,
			candidatePaths: uniqueStrings([target.targetPath]),
		});
	}

	for (const file of output?.files ?? []) {
		const key = file.fileKey || `${file.oldPath ?? ''}\u0000${file.newPath ?? ''}`;

		const existing = byKey.get(key);
		const candidatePaths = uniqueStrings(
			[
				...(existing?.candidatePaths ?? []),
				...(file.candidatePaths ?? []),
				file.targetPath,
				file.oldPath,
				file.newPath,
			].filter(isUsablePatchPath)
		);

		byKey.set(key, {
			fileKey: file.fileKey,
			oldPath: file.oldPath,
			newPath: file.newPath,
			targetPath: existing?.targetPath || file.targetPath || '',
			candidatePaths,
			status: file.status,
			message: file.message,
			diagnostics: file.diagnostics,
			hunks: file.hunks,
			addedLines: file.addedLines,
			deletedLines: file.deletedLines,
		});
	}

	if (byKey.size === 0) {
		for (const file of fallback.files) {
			byKey.set(file.fileKey, {
				fileKey: file.fileKey,
				oldPath: file.oldPath,
				newPath: file.newPath,
				targetPath: file.targetPath || '',
				candidatePaths: uniqueStrings(file.candidatePaths),
				hunks: file.hunks,
				addedLines: file.addedLines,
				deletedLines: file.deletedLines,
			});
		}
	}

	if (byKey.size === 0) {
		byKey.set('file-1', {
			fileKey: 'file-1',
			targetPath: '',
			candidatePaths: [],
			hunks: 0,
			addedLines: 0,
			deletedLines: 0,
		});
	}

	return Array.from(byKey.values());
}

export function editableTargetsToFileTargets(targets: EditableUnifiedDiffTarget[]): ApplyUnifiedDiffFileTarget[] {
	return targets
		.map(target => ({
			fileKey: target.fileKey?.trim() || undefined,
			oldPath: target.oldPath?.trim() || undefined,
			newPath: target.newPath?.trim() || undefined,
			targetPath: target.targetPath.trim(),
		}))
		.filter(target => target.targetPath.length > 0);
}

export function summaryLabel(output: ApplyUnifiedDiffOut | undefined, fallback: ParsedUnifiedDiffForUI): string {
	const summary = output?.summary;
	if (summary) {
		return `${summary.files} file${summary.files === 1 ? '' : 's'}, ${summary.hunks} hunk${
			summary.hunks === 1 ? '' : 's'
		}, +${summary.addedLines}/-${summary.deletedLines}`;
	}

	return `${fallback.files.length} file${fallback.files.length === 1 ? '' : 's'}, ${fallback.hunks} hunk${
		fallback.hunks === 1 ? '' : 's'
	}, +${fallback.addedLines}/-${fallback.deletedLines}`;
}

export function collectOutputDiagnostics(output?: ApplyUnifiedDiffOut): string[] {
	if (!output) return [];

	return uniqueStrings([
		...(output.diagnostics ?? []),
		...(output.files ?? []).flatMap(file => file.diagnostics ?? []),
	]);
}

export interface EditableUnifiedDiffTarget {
	fileKey?: string;
	oldPath?: string;
	newPath?: string;
	targetPath: string;
	candidatePaths: string[];

	status?: string;
	message?: string;
	diagnostics?: string[];
	hunks?: number;
	addedLines?: number;
	deletedLines?: number;
}

function parseDiffHeaderPath(input: string): string {
	return normalizeDiffPathToken(input);
}

function normalizeUnifiedHeaderPair(oldPath: string, newPath: string): [string, string] {
	const oldHasGitPrefix = oldPath.startsWith('a/');
	const newHasGitPrefix = newPath.startsWith('b/');

	if ((oldHasGitPrefix && (newHasGitPrefix || newPath === DEV_NULL)) || (newHasGitPrefix && oldPath === DEV_NULL)) {
		return [stripGitPathPrefix(oldPath, 'a'), stripGitPathPrefix(newPath, 'b')];
	}

	return [oldPath, newPath];
}

function readDiffPathToken(input: string): { token: string; rest: string } {
	const value = input.trim();
	if (!value) return { token: '', rest: '' };

	if (!value.startsWith('"')) {
		const match = /^(\S+)(?:\s+([\s\S]*))?$/.exec(value);
		return {
			token: match?.[1] ?? '',
			rest: match?.[2] ?? '',
		};
	}

	let token = '';
	let escaped = false;

	for (let i = 1; i < value.length; i += 1) {
		const ch = value[i];

		if (escaped) {
			token += ch;
			escaped = false;
			continue;
		}

		if (ch === '\\') {
			escaped = true;
			continue;
		}

		if (ch === '"') {
			return {
				token,
				rest: value.slice(i + 1).trim(),
			};
		}

		token += ch;
	}

	return { token, rest: '' };
}

function normalizeDiffPathToken(input: string): string {
	let value = input.trim();
	if (!value) return '';

	if (value === DEV_NULL) return DEV_NULL;

	if (value.startsWith('"')) {
		value = readDiffPathToken(value).token;
	} else {
		const tabIndex = value.indexOf('\t');
		if (tabIndex >= 0) {
			value = value.slice(0, tabIndex);
		} else {
			value = value.split(/\s+/)[0] ?? '';
		}
	}

	return value.trim();
}

function stripGitPathPrefix(value: string, prefix: string): string {
	const marker = `${prefix}/`;
	if (value.startsWith(marker)) return value.slice(marker.length);
	return value;
}

function isUsablePatchPath(value: unknown): value is string {
	return typeof value === 'string' && value.trim().length > 0 && value.trim() !== DEV_NULL;
}

function uniqueStrings(values: Array<string | undefined | null>): string[] {
	const out: string[] = [];
	const seen = new Set<string>();

	for (const value of values) {
		const trimmed = value?.trim();
		if (!trimmed) continue;

		const key = trimmed.replaceAll('\\', '/').replace(/\/+/g, '/').toLowerCase();
		if (seen.has(key)) continue;

		seen.add(key);
		out.push(trimmed);
	}

	return out;
}
