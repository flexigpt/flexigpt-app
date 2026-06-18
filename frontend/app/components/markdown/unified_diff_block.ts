// unified diff parsing and target extraction helpers
import {
	type ApplyUnifiedDiffDiagnostic,
	ApplyUnifiedDiffDiagnosticLevel,
	type ApplyUnifiedDiffFileTarget,
	type ApplyUnifiedDiffOut,
} from '@/spec/unified_diff';

interface ParsedUnifiedDiffFileForUI {
	fileKey: string;
	oldPath?: string;
	newPath?: string;
	hunks: number;
	addedLines: number;
	deletedLines: number;
	candidatePaths: string[];
	targetPath?: string;
	diffText?: string;
	sectionKeys: string[];
}

interface WorkingParsedUnifiedDiffFileForUI extends Omit<ParsedUnifiedDiffFileForUI, 'diffText'> {
	lines: string[];
}

export interface UnifiedDiffTextForTarget {
	diffText: string;
	hunks: number;
	sectionKeys: string[];
	verified: boolean;
}

export interface ParsedUnifiedDiffForUI {
	isDiffLike: boolean;
	files: ParsedUnifiedDiffFileForUI[];
	hunks: number;
	addedLines: number;
	deletedLines: number;
	diagnostics: ApplyUnifiedDiffDiagnostic[];
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
	const hasPlainFileHeader = /^---\s+/m.test(text) && /^\+\+\+\s+/m.test(text);
	if (hasPlainFileHeader) return true;

	const hasHunkHeader = /^@@\s*-?\d+(?:,\d+)?\s*\+?\d+(?:,\d+)?\s*@@/m.test(text);
	if (!hasHunkHeader) return false;

	// Let hunk-only LLM diffs through, but avoid showing the apply UI for
	// arbitrary code snippets that merely contain an @@ marker.
	return /^(?:\+[^+\n]|-[^-\n]| [^\n])/m.test(text);
}

export function parseUnifiedDiffForUI(value: string, language = ''): ParsedUnifiedDiffForUI {
	const text = value
		.replace(/\r\n/g, '\n')
		.replace(/\r/g, '\n')
		.replace(/^\ufeff/, '');
	const lines = text.split('\n');

	const rawFiles: ParsedUnifiedDiffFileForUI[] = [];
	const diagnostics: ApplyUnifiedDiffDiagnostic[] = [];

	let current: WorkingParsedUnifiedDiffFileForUI | null = null;
	let inHunk = false;

	const createWorkingFile = (): WorkingParsedUnifiedDiffFileForUI => ({
		fileKey: `file-${rawFiles.length + 1}`,
		hunks: 0,
		addedLines: 0,
		deletedLines: 0,
		candidatePaths: [],
		sectionKeys: [],
		lines: [],
	});

	const pushCurrent = () => {
		if (!current) return;
		if (current.oldPath || current.newPath || current.hunks > 0) {
			const { lines: fileLines, ...file } = current;

			rawFiles.push({
				...file,
				candidatePaths: uniqueStrings(file.candidatePaths),
				diffText: fileLines.join('\n'),
				sectionKeys: [file.fileKey],
			});
		}
		current = null;
		inHunk = false;
	};

	const ensureCurrent = () => {
		if (!current) {
			current = createWorkingFile();
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
				...createWorkingFile(),
				oldPath: oldPath || undefined,
				newPath: newPath || undefined,
				candidatePaths: uniqueStrings([oldPath, newPath].filter(isUsablePatchPath)),
			};
			current.lines.push(line);
			inHunk = false;
			continue;
		}

		if (line.startsWith('Index: ')) {
			pushCurrent();

			const p = normalizeDiffPathToken(line.slice('Index: '.length).trim());
			current = {
				...createWorkingFile(),
				oldPath: p || undefined,
				newPath: p || undefined,
				candidatePaths: isUsablePatchPath(p) ? [p] : [],
			};
			current.lines.push(line);
			inHunk = false;
			continue;
		}

		if (line.startsWith('--- ') && i + 1 < lines.length && lines[i + 1].startsWith('+++ ')) {
			if (!current || current.hunks > 0) {
				pushCurrent();
				current = createWorkingFile();
			}

			const file = ensureCurrent();
			file.lines.push(line, lines[i + 1]);

			const oldPathRaw = parseDiffHeaderPath(lines[i].slice(4));
			const newPathRaw = parseDiffHeaderPath(lines[i + 1].slice(4));
			const [oldPath, newPath] = normalizeUnifiedHeaderPair(oldPathRaw, newPathRaw);

			file.oldPath = oldPath || undefined;
			file.newPath = newPath || undefined;
			file.candidatePaths = uniqueStrings([...(file.candidatePaths ?? []), oldPath, newPath].filter(isUsablePatchPath));

			i += 1;
			inHunk = false;
			continue;
		}

		if (line.startsWith('@@')) {
			const file = ensureCurrent();
			file.lines.push(line);
			file.hunks += 1;
			inHunk = true;

			if (!/^@@\s+-\d+(?:,\d+)?\s+\+\d+(?:,\d+)?\s+@@/.test(line)) {
				const d: ApplyUnifiedDiffDiagnostic = {
					level: ApplyUnifiedDiffDiagnosticLevel.Error,
					message: `Non-standard hunk header detected: ${line}`,
				};
				diagnostics.push(d);
			}
			continue;
		}

		if (current) {
			current.lines.push(line);
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

	const files = mergeParsedUnifiedDiffFiles(rawFiles);

	return {
		isDiffLike: looksLikeUnifiedDiff(value, language),
		files,
		hunks: files.reduce((sum, file) => sum + file.hunks, 0),
		addedLines: files.reduce((sum, file) => sum + file.addedLines, 0),
		deletedLines: files.reduce((sum, file) => sum + file.deletedLines, 0),
		diagnostics,
	};
}

export function buildUnifiedDiffTextForTarget(
	value: string,
	language: string,
	target: {
		fileKey?: string;
		oldPath?: string;
		newPath?: string;
		targetPath?: string;
		sectionKeys?: string[];
	}
): UnifiedDiffTextForTarget | undefined {
	const parsed = parseUnifiedDiffForUI(value, language);
	let matches = parsed.files.filter(file => parsedFileMatchesTarget(file, target));

	if (matches.length === 0 && parsed.files.length === 1) {
		matches = parsed.files;
	}

	if (matches.length === 0) return undefined;

	const diffText = joinUnifiedDiffTextParts(matches.map(file => file.diffText));
	if (!diffText) return undefined;

	const hunks = matches.reduce((sum, file) => sum + file.hunks, 0);
	const detectedHunks = countHunkHeaders(diffText);

	return {
		diffText,
		hunks,
		sectionKeys: uniqueStrings(matches.flatMap(file => [file.fileKey, ...(file.sectionKeys ?? [])])),
		verified: hunks === 0 || detectedHunks === hunks,
	};
}

export function buildEditableTargetsFromOutput(
	output: ApplyUnifiedDiffOut | undefined,
	fallback: ParsedUnifiedDiffForUI
): EditableUnifiedDiffTarget[] {
	const byKey = new Map<string, EditableUnifiedDiffTarget>();

	const upsert = (target: EditableUnifiedDiffTarget) => {
		upsertEditableTarget(byKey, target);
	};

	for (const file of fallback.files) {
		upsert({
			fileKey: file.fileKey,
			oldPath: file.oldPath,
			newPath: file.newPath,
			targetPath: file.targetPath || '',
			candidatePaths: uniqueStrings(file.candidatePaths),
			hunks: file.hunks,
			addedLines: file.addedLines,
			deletedLines: file.deletedLines,
			diffText: file.diffText,
			sectionKeys: file.sectionKeys,
		});
	}

	for (const target of output?.fileTargets ?? []) {
		upsert({
			fileKey: target.fileKey,
			oldPath: target.oldPath,
			newPath: target.newPath,
			targetPath: target.targetPath,
			candidatePaths: uniqueStrings([target.targetPath, target.newPath, target.oldPath].filter(isUsablePatchPath)),
		});
	}

	for (const file of output?.files ?? []) {
		upsert({
			fileKey: file.fileKey,
			oldPath: file.oldPath,
			newPath: file.newPath,
			targetPath: file.targetPath || file.resolvedPath || '',
			resolvedPath: file.resolvedPath,
			candidatePaths: uniqueStrings(
				[...(file.candidatePaths ?? []), file.targetPath, file.resolvedPath, file.oldPath, file.newPath].filter(
					isUsablePatchPath
				)
			),
			ok: file.ok,
			status: file.status,
			message: file.message,
			diagnostics: file.diagnostics ? file.diagnostics.map(d => ({ ...d })) : undefined,
			hunks: file.hunks,
			appliedHunks: file.appliedHunks,
			alreadyAppliedHunks: file.alreadyAppliedHunks,
			addedLines: file.addedLines,
			deletedLines: file.deletedLines,
		});
	}

	if (byKey.size === 0) {
		byKey.set('file-1', {
			fileKey: 'file-1',
			targetPath: '',
			candidatePaths: [],
			sectionKeys: ['file-1'],
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

export function collectPatchLevelDiagnostics(output?: ApplyUnifiedDiffOut): ApplyUnifiedDiffDiagnostic[] {
	return uniqueDiagnostics(output?.diagnostics ?? []);
}

export function collectFileLevelDiagnostics(output?: ApplyUnifiedDiffOut): ApplyUnifiedDiffDiagnostic[] {
	return uniqueDiagnostics((output?.files ?? []).flatMap(file => file.diagnostics ?? []));
}

export function collectOutputDiagnostics(output?: ApplyUnifiedDiffOut): string[] {
	return uniqueStringsFromDiagnostics([
		...collectPatchLevelDiagnostics(output),
		...collectFileLevelDiagnostics(output),
	]);
}

export interface EditableUnifiedDiffTarget {
	fileKey?: string;
	oldPath?: string;
	newPath?: string;
	targetPath: string;
	resolvedPath?: string;
	candidatePaths: string[];
	diffText?: string;
	sectionKeys?: string[];

	ok?: boolean;
	status?: string;
	message?: string;
	diagnostics?: ApplyUnifiedDiffDiagnostic[];
	hunks?: number;
	appliedHunks?: number;
	alreadyAppliedHunks?: number;
	addedLines?: number;
	deletedLines?: number;
}

function mergeParsedUnifiedDiffFiles(files: ParsedUnifiedDiffFileForUI[]): ParsedUnifiedDiffFileForUI[] {
	const out: ParsedUnifiedDiffFileForUI[] = [];
	const byIdentity = new Map<string, ParsedUnifiedDiffFileForUI>();

	for (const file of files) {
		const identity = getPatchFileIdentity(file) || `section:${file.fileKey}`;
		const existing = byIdentity.get(identity);

		if (!existing) {
			const next: ParsedUnifiedDiffFileForUI = {
				...file,
				fileKey: `file-${out.length + 1}`,
				candidatePaths: uniqueStrings(file.candidatePaths),
				sectionKeys: uniqueStrings([file.fileKey, ...(file.sectionKeys ?? [])]),
			};

			byIdentity.set(identity, next);
			out.push(next);
			continue;
		}

		existing.oldPath = existing.oldPath || file.oldPath;
		existing.newPath = existing.newPath || file.newPath;
		existing.hunks += file.hunks;
		existing.addedLines += file.addedLines;
		existing.deletedLines += file.deletedLines;
		existing.candidatePaths = uniqueStrings([...(existing.candidatePaths ?? []), ...(file.candidatePaths ?? [])]);
		existing.sectionKeys = uniqueStrings([...(existing.sectionKeys ?? []), file.fileKey, ...(file.sectionKeys ?? [])]);
		existing.diffText = joinUnifiedDiffTextParts([existing.diffText, file.diffText]);
	}

	return out;
}

function upsertEditableTarget(byKey: Map<string, EditableUnifiedDiffTarget>, target: EditableUnifiedDiffTarget) {
	const existingKey = findEditableTargetMapKey(byKey, target);
	const key = existingKey ?? getEditableTargetMapKey(target, byKey.size);
	const existing = existingKey ? byKey.get(existingKey) : undefined;

	byKey.set(key, mergeEditableTarget(existing, target));
}

function findEditableTargetMapKey(
	byKey: Map<string, EditableUnifiedDiffTarget>,
	target: EditableUnifiedDiffTarget
): string | undefined {
	const fileKey = target.fileKey?.trim();

	if (fileKey) {
		for (const [key, existing] of byKey.entries()) {
			if (existing.fileKey === fileKey || existing.sectionKeys?.includes(fileKey)) {
				return key;
			}
		}
	}

	const identity = getPatchFileIdentity(target);
	if (!identity) return undefined;

	for (const [key, existing] of byKey.entries()) {
		if (getPatchFileIdentity(existing) === identity) return key;
	}

	return undefined;
}

function getEditableTargetMapKey(target: EditableUnifiedDiffTarget, index: number): string {
	return getPatchFileIdentity(target) || target.fileKey || `target-${index}`;
}

function mergeEditableTarget(
	existing: EditableUnifiedDiffTarget | undefined,
	update: EditableUnifiedDiffTarget
): EditableUnifiedDiffTarget {
	if (!existing) {
		return {
			...update,
			targetPath: update.targetPath || '',
			candidatePaths: uniqueStrings([
				...(update.candidatePaths ?? []),
				update.targetPath,
				update.resolvedPath,
				update.newPath,
				update.oldPath,
			]),
			sectionKeys: uniqueStrings([update.fileKey, ...(update.sectionKeys ?? [])]),
		};
	}

	return {
		...existing,
		...update,
		fileKey: existing.fileKey || update.fileKey,
		oldPath: update.oldPath || existing.oldPath,
		newPath: update.newPath || existing.newPath,
		targetPath: update.targetPath || existing.targetPath || '',
		resolvedPath: update.resolvedPath || existing.resolvedPath,
		candidatePaths: uniqueStrings([
			...(existing.candidatePaths ?? []),
			...(update.candidatePaths ?? []),
			existing.targetPath,
			update.targetPath,
			existing.resolvedPath,
			update.resolvedPath,
			update.newPath,
			update.oldPath,
			existing.newPath,
			existing.oldPath,
		]),
		diffText: update.diffText || existing.diffText,
		sectionKeys: uniqueStrings([
			...(existing.sectionKeys ?? []),
			existing.fileKey,
			update.fileKey,
			...(update.sectionKeys ?? []),
		]),
		ok: update.ok ?? existing.ok,
		status: update.status ?? existing.status,
		message: update.message ?? existing.message,
		diagnostics: uniqueDiagnostics([...(existing.diagnostics ?? []), ...(update.diagnostics ?? [])]),
		hunks: mergeNumericValue(existing.hunks, update.hunks),
		appliedHunks: mergeNumericValue(existing.appliedHunks, update.appliedHunks),
		alreadyAppliedHunks: mergeNumericValue(existing.alreadyAppliedHunks, update.alreadyAppliedHunks),
		addedLines: mergeNumericValue(existing.addedLines, update.addedLines),
		deletedLines: mergeNumericValue(existing.deletedLines, update.deletedLines),
	};
}

export function uniqueDiagnostics(
	values: Array<ApplyUnifiedDiffDiagnostic | undefined | null>
): ApplyUnifiedDiffDiagnostic[] {
	const out: ApplyUnifiedDiffDiagnostic[] = [];
	const seen = new Set<string>();

	for (const value of values) {
		if (!value) continue;

		const message = value.message.trim();
		if (!message) continue;

		const level = value.level ?? ApplyUnifiedDiffDiagnosticLevel.Info;
		const code = value.code?.trim() ?? '';
		const key = `${level}\u0000${code}\u0000${message.replaceAll('\\', '/').replace(/\/+/g, '/')}`;

		if (seen.has(key)) continue;

		seen.add(key);
		out.push({
			level,
			code: code || undefined,
			message,
		});
	}

	return out;
}

function mergeNumericValue(left: number | undefined, right: number | undefined): number | undefined {
	if (typeof left === 'number' && typeof right === 'number') return Math.max(left, right);
	if (typeof right === 'number') return right;
	return left;
}

function parsedFileMatchesTarget(
	file: ParsedUnifiedDiffFileForUI,
	target: {
		fileKey?: string;
		oldPath?: string;
		newPath?: string;
		targetPath?: string;
		sectionKeys?: string[];
	}
): boolean {
	const targetFileKey = target.fileKey?.trim();

	if (targetFileKey && (file.fileKey === targetFileKey || file.sectionKeys.includes(targetFileKey))) {
		return true;
	}

	const targetSectionKeys = uniqueStrings([targetFileKey, ...(target.sectionKeys ?? [])]);
	if (targetSectionKeys.length > 0) {
		const fileSectionKeys = uniqueStrings([file.fileKey, ...(file.sectionKeys ?? [])]);
		if (targetSectionKeys.some(key => fileSectionKeys.includes(key))) {
			return true;
		}
	}

	const targetIdentity = getPatchFileIdentity(target);
	if (targetIdentity && targetIdentity === getPatchFileIdentity(file)) return true;

	const filePaths = uniqueStrings([file.targetPath, file.newPath, file.oldPath, ...file.candidatePaths]);
	const targetPaths = uniqueStrings([target.targetPath, target.newPath, target.oldPath]);

	return filePaths.some(filePath =>
		targetPaths.some(targetPath => normalizePathKey(filePath) === normalizePathKey(targetPath))
	);
}

function getPatchFileIdentity(file: {
	targetPath?: string;
	resolvedPath?: string;
	oldPath?: string;
	newPath?: string;
}): string {
	const targetPath = normalizePathKey(file.targetPath);
	if (targetPath) return `path:${targetPath}`;

	const resolvedPath = normalizePathKey(file.resolvedPath);
	if (resolvedPath) return `path:${resolvedPath}`;

	const newPath = normalizePathKey(file.newPath);
	if (newPath) return `path:${newPath}`;

	const oldPath = normalizePathKey(file.oldPath);
	if (oldPath) return `path:${oldPath}`;

	return '';
}

function joinUnifiedDiffTextParts(parts: Array<string | undefined>): string | undefined {
	const normalized = parts.map(part => part?.replace(/\n+$/g, '')).filter((part): part is string => !!part);
	if (normalized.length === 0) return undefined;
	return normalized.join('\n');
}

function countHunkHeaders(diffText: string): number {
	return diffText.split('\n').filter(line => line.startsWith('@@')).length;
}

function normalizePathKey(value: string | undefined): string {
	if (!isUsablePatchPath(value)) return '';
	return value
		.trim()
		.replaceAll('\\', '/')
		.replace(/\/+/g, '/')
		.replace(/^(?:\.\/)+/, '');
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

function uniqueStringsFromDiagnostics(values: Array<ApplyUnifiedDiffDiagnostic | undefined | null>): string[] {
	const out: string[] = [];
	const seen = new Set<string>();

	for (const value of values) {
		const trimmed = value?.message.trim();
		if (!trimmed) continue;

		const key = trimmed.replaceAll('\\', '/').replace(/\/+/g, '/');

		if (seen.has(key)) continue;

		seen.add(key);
		out.push(trimmed);
	}

	return out;
}

function uniqueStrings(values: Array<string | undefined | null>): string[] {
	const out: string[] = [];
	const seen = new Set<string>();

	for (const value of values) {
		const trimmed = value?.trim();
		if (!trimmed) continue;

		const key = trimmed.replaceAll('\\', '/').replace(/\/+/g, '/');

		if (seen.has(key)) continue;

		seen.add(key);
		out.push(trimmed);
	}

	return out;
}
