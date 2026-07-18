import type { ApplyUnifiedDiffDiagnostic, ApplyUnifiedDiffOut } from '@/spec/unified_diff';
import { ApplyUnifiedDiffDiagnosticLevel, ApplyUnifiedDiffStatus } from '@/spec/unified_diff';

import { uniqueDiagnostics } from '@/components/markdown/diff_diagnostic';

export interface DiffApplyRunOptions {
	diffText?: string;
	mergeOutput?: boolean;
}

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

interface TargetPathInferenceForUI {
	targetPath: string;
	sourcePath: string;
	score: number;
}

interface EditableTargetPathResolutionForUI {
	targetPath: string;
	resolvedPath?: string;
	candidatePaths: string[];
	knownTargetPaths: string[];
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
	isOpenAIPatch: boolean;
	files: ParsedUnifiedDiffFileForUI[];
	hunks: number;
	addedLines: number;
	deletedLines: number;
	diagnostics: ApplyUnifiedDiffDiagnostic[];
}

const DEV_NULL = '/dev/null';
const OPENAI_PATCH_FILE_HEADER_RE = /^\*\*\*\s+(Update|Add|Delete)\s+File:\s*(.+?)\s*$/i;
const OPENAI_PATCH_MOVE_TO_RE = /^\*\*\*\s+Move\s+to:\s*(.+?)\s*$/i;

/**
 * Applied and already-applied are terminal, non-error outcomes. Callers must
 * not let an older applicability `ok` flag override them.
 */
export function isTerminalUnifiedDiffStatus(status: unknown): boolean {
	return status === ApplyUnifiedDiffStatus.Applied || status === ApplyUnifiedDiffStatus.AlreadyApplied;
}

export function isAbsolutePath(value: string | undefined | null): boolean {
	const trimmed = value?.trim();

	return !!(
		trimmed &&
		trimmed !== DEV_NULL &&
		(trimmed.startsWith('/') || /^[A-Za-z]:[\\/]/.test(trimmed) || trimmed.startsWith('\\\\'))
	);
}

export function toAbsolutePath(value: string | undefined | null): string {
	const trimmed = value?.trim() ?? '';
	return isAbsolutePath(trimmed) ? trimmed : '';
}

export function absolutePathStrings(values: Array<string | undefined | null>): string[] {
	return uniqueStrings(
		values.map(v => {
			return toAbsolutePath(v);
		})
	);
}

/**
 * Candidate paths supplied to this component are expected to come from
 * attachments, tools, or another source outside the current diff. Remove
 * exact patch-path echoes defensively so a path copied from the diff cannot
 * become evidence for itself.
 */
export function filterDiffOwnedCandidatePaths(parsed: ParsedUnifiedDiffForUI, candidatePaths: string[]): string[] {
	const patchPaths = parsed.files.flatMap(file => [file.oldPath, file.newPath]);

	return absolutePathStrings(candidatePaths).filter(path => !isPatchPathEcho(path, patchPaths));
}

function isUnifiedDiffLanguage(language: string): boolean {
	const normalized = language.trim().toLowerCase();
	return normalized === 'diff' || normalized === 'patch' || normalized === 'udiff';
}

export function looksLikeUnifiedDiff(value: string, language = ''): boolean {
	const text = value.replaceAll('\r\n', '\n').replaceAll('\r', '\n');

	if (looksLikeOpenAIPatch(text)) {
		return true;
	}
	if (isUnifiedDiffLanguage(language)) {
		return true;
	}
	if (/^diff --git\s+/m.test(text)) {
		return true;
	}
	if (/^Index:\s+/m.test(text)) {
		return true;
	}
	const hasPlainFileHeader = /^---\s+/m.test(text) && /^\+\+\+\s+/m.test(text);
	if (hasPlainFileHeader) {
		return true;
	}

	const hasHunkHeader = /^@@\s*-?\d+(?:,\d+)?\s*\+?\d+(?:,\d+)?\s*@@/m.test(text);
	if (!hasHunkHeader) {
		return false;
	}

	// Let hunk-only LLM diffs through, but avoid showing the apply UI for
	// arbitrary code snippets that merely contain an @@ marker.
	return /^(?:\+[^+\n]|-[^-\n]| [^\n])/m.test(text);
}

export function parseUnifiedDiffForUI(value: string, language = ''): ParsedUnifiedDiffForUI {
	const text = value
		.replaceAll('\r\n', '\n')
		.replaceAll('\r', '\n')
		.replace(/^\uFEFF/, '');
	const lines = text.split('\n');

	const rawFiles: ParsedUnifiedDiffFileForUI[] = [];
	const diagnostics: ApplyUnifiedDiffDiagnostic[] = [];

	let current: WorkingParsedUnifiedDiffFileForUI | null = null;
	let inHunk = false;
	let remainingOldHunkLines: number | undefined;
	let remainingNewHunkLines: number | undefined;
	let sawOpenAIPatchFormat = false;

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
		if (!current) {
			return;
		}
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
		remainingOldHunkLines = undefined;
		remainingNewHunkLines = undefined;
	};

	const ensureCurrent = () => {
		if (!current) {
			current = createWorkingFile();
		}
		return current;
	};

	for (let i = 0; i < lines.length; i += 1) {
		const line = lines[i];

		if (line.trim() === '*** Begin Patch') {
			sawOpenAIPatchFormat = true;
			continue;
		}

		if (line.trim() === '*** End Patch') {
			sawOpenAIPatchFormat = true;
			if (current) {
				current.lines.push(line);
			}
			inHunk = false;
			remainingOldHunkLines = undefined;
			remainingNewHunkLines = undefined;
			continue;
		}

		const openAIHeader = parseOpenAIPatchFileHeader(line);
		if (openAIHeader) {
			sawOpenAIPatchFormat = true;
			pushCurrent();

			const p = openAIHeader.path;
			const oldPath = openAIHeader.kind === 'add' ? DEV_NULL : p;
			const newPath = openAIHeader.kind === 'delete' ? DEV_NULL : p;

			current = {
				...createWorkingFile(),
				oldPath: oldPath || undefined,
				newPath: newPath || undefined,
				candidatePaths: absolutePathStrings([p]),
			};
			current.lines.push(line);
			inHunk = openAIHeader.kind === 'add' || openAIHeader.kind === 'delete';
			continue;
		}

		const openAIMoveTo = parseOpenAIPatchMoveTo(line);
		if (openAIMoveTo && current) {
			sawOpenAIPatchFormat = true;
			current.lines.push(line);
			current.newPath = openAIMoveTo;
			current.candidatePaths = absolutePathStrings([...(current.candidatePaths ?? []), current.oldPath, openAIMoveTo]);
			inHunk = false;
			remainingOldHunkLines = undefined;
			remainingNewHunkLines = undefined;
			continue;
		}

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
				candidatePaths: absolutePathStrings([oldPath, newPath]),
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
				candidatePaths: absolutePathStrings([p]),
			};
			current.lines.push(line);
			inHunk = false;
			continue;
		}

		// Git binary patches and mode-only patches can omit ---/+++ headers.
		// Preserve their operation from the extended git headers.
		if (!inHunk && current && line.startsWith('new file mode ')) {
			current.lines.push(line);
			current.oldPath = DEV_NULL;
			continue;
		}

		if (!inHunk && current && line.startsWith('deleted file mode ')) {
			current.lines.push(line);
			current.newPath = DEV_NULL;
			continue;
		}

		if (!inHunk && current && line.startsWith('rename from ')) {
			current.lines.push(line);
			const oldPath = parseGitExtendedHeaderPath(line.slice('rename from '.length));

			if (oldPath) {
				current.oldPath = oldPath;
				current.candidatePaths = absolutePathStrings([...(current.candidatePaths ?? []), oldPath, current.newPath]);
			}
			continue;
		}

		if (!inHunk && current && line.startsWith('rename to ')) {
			current.lines.push(line);
			const newPath = parseGitExtendedHeaderPath(line.slice('rename to '.length));

			if (newPath) {
				current.newPath = newPath;
				current.candidatePaths = absolutePathStrings([...(current.candidatePaths ?? []), current.oldPath, newPath]);
			}
			continue;
		}

		if (!inHunk && line.startsWith('--- ') && i + 1 < lines.length && lines[i + 1].startsWith('+++ ')) {
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
			file.candidatePaths = absolutePathStrings([...(file.candidatePaths ?? []), oldPath, newPath]);

			i += 1;
			inHunk = false;
			continue;
		}

		if (line.startsWith('@@')) {
			const file = ensureCurrent();
			file.lines.push(line);
			file.hunks += 1;

			const hunkLineCounts = parseUnifiedDiffHunkLineCounts(line);
			remainingOldHunkLines = hunkLineCounts?.oldLines;
			remainingNewHunkLines = hunkLineCounts?.newLines;
			inHunk = !hunkLineCounts || hunkLineCounts.oldLines > 0 || hunkLineCounts.newLines > 0;
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

			if (
				typeof remainingOldHunkLines === 'number' &&
				typeof remainingNewHunkLines === 'number' &&
				!line.startsWith('\\')
			) {
				if (line.startsWith('+')) {
					remainingNewHunkLines -= 1;
				} else if (line.startsWith('-')) {
					remainingOldHunkLines -= 1;
				} else if (line.startsWith(' ')) {
					remainingOldHunkLines -= 1;
					remainingNewHunkLines -= 1;
				}

				if (remainingOldHunkLines <= 0 && remainingNewHunkLines <= 0) {
					inHunk = false;
					remainingOldHunkLines = undefined;
					remainingNewHunkLines = undefined;
				}
			}
		}
	}

	pushCurrent();

	const files = mergeParsedUnifiedDiffFiles(rawFiles);

	if (sawOpenAIPatchFormat) {
		if (
			files.length > 0 &&
			files.every(f => {
				return isParsedOpenAIAddFile(f);
			})
		) {
			diagnostics.push({
				level: ApplyUnifiedDiffDiagnosticLevel.Info,
				code: 'openai_add_format',
				message:
					'Detected OpenAI Add File format. Add-file sections will be converted to standard unified diff before being sent.',
			});
		} else {
			diagnostics.push({
				level: ApplyUnifiedDiffDiagnosticLevel.Warning,
				code: 'openai_patch_format',
				message:
					'Detected OpenAI apply_patch format. Only patches containing Add File sections exclusively can be converted safely by this apply control.',
			});
		}
	}

	return {
		isDiffLike: looksLikeUnifiedDiff(value, language),
		isOpenAIPatch: sawOpenAIPatchFormat,
		files,
		hunks: files.reduce((sum, file) => sum + file.hunks, 0),
		addedLines: files.reduce((sum, file) => sum + file.addedLines, 0),
		deletedLines: files.reduce((sum, file) => sum + file.deletedLines, 0),
		diagnostics,
	};
}

function isParsedOpenAIAddFile(file: ParsedUnifiedDiffFileForUI): boolean {
	return file.oldPath === DEV_NULL && isUsablePatchPath(file.newPath);
}

function formatOpenAIAddPathForUnifiedDiff(path: string): string {
	return /[\s"\\]/.test(path) ? JSON.stringify(path) : path;
}

function convertParsedOpenAIAddFileToUnifiedDiff(file: ParsedUnifiedDiffFileForUI): string | undefined {
	if (!isParsedOpenAIAddFile(file) || !file.diffText || !file.newPath) {
		return undefined;
	}

	const sectionLines = file.diffText.split('\n');
	const headerIndex = sectionLines.findIndex(line => parseOpenAIPatchFileHeader(line)?.kind === 'add');
	if (headerIndex < 0) {
		return undefined;
	}

	const bodyLines: string[] = [];

	for (const line of sectionLines.slice(headerIndex + 1)) {
		if (line.trim() === '*** End Patch') {
			break;
		}

		if (line === '\\ No newline at end of file') {
			bodyLines.push(line);
			continue;
		}

		if (!line.startsWith('+')) {
			return undefined;
		}

		bodyLines.push(line);
	}

	const addedLines = bodyLines.filter(line => line.startsWith('+')).length;
	if (addedLines === 0 && bodyLines.length > 0) {
		return undefined;
	}

	const out = ['--- /dev/null', `+++ ${formatOpenAIAddPathForUnifiedDiff(file.newPath)}`];

	if (addedLines > 0) {
		out.push(`@@ -0,0 +1,${addedLines} @@`, ...bodyLines);
	}

	return out.join('\n');
}

export function prepareUnifiedDiffTextForApply(value: string, language = ''): string {
	const parsed = parseUnifiedDiffForUI(value, language);
	if (!parsed.isOpenAIPatch || parsed.files.length === 0 || !parsed.files.every(isParsedOpenAIAddFile)) {
		return value;
	}

	const convertedParts = parsed.files.map(convertParsedOpenAIAddFileToUnifiedDiff);
	if (convertedParts.some(part => !part)) {
		return value;
	}

	return joinUnifiedDiffTextParts(convertedParts) ?? value;
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

	if (matches.length === 0) {
		return undefined;
	}

	let diffText: string | undefined;
	let expectedHunks = matches.reduce((sum, file) => sum + file.hunks, 0);

	if (
		parsed.isOpenAIPatch &&
		matches.every(f => {
			return isParsedOpenAIAddFile(f);
		})
	) {
		const convertedParts = matches.map(m => {
			return convertParsedOpenAIAddFileToUnifiedDiff(m);
		});
		if (convertedParts.some(part => !part)) {
			return undefined;
		}

		diffText = joinUnifiedDiffTextParts(convertedParts);
		expectedHunks = diffText ? countHunkHeaders(diffText) : 0;
	} else {
		diffText = joinUnifiedDiffTextParts(matches.map(file => file.diffText));
	}

	if (!diffText) {
		return undefined;
	}

	const detectedHunks = countHunkHeaders(diffText);
	const sectionKeyCandidates: string[] = [];

	for (const file of matches) {
		sectionKeyCandidates.push(file.fileKey);

		for (const sectionKey of file.sectionKeys ?? []) {
			sectionKeyCandidates.push(sectionKey);
		}
	}

	return {
		diffText,
		hunks: expectedHunks,
		sectionKeys: uniqueStrings(sectionKeyCandidates),
		verified: expectedHunks === 0 || detectedHunks === expectedHunks,
	};
}

export function buildEditableTargetsFromOutput(
	output: ApplyUnifiedDiffOut | undefined,
	fallback: ParsedUnifiedDiffForUI,
	globalCandidatePaths: string[] = [],
	knownWorkspaceRoots: string[] = []
): EditableUnifiedDiffTarget[] {
	const byKey = new Map<string, EditableUnifiedDiffTarget>();
	const knownSourcePaths = absolutePathStrings(globalCandidatePaths);
	const workspaceRoots = absolutePathStrings(knownWorkspaceRoots);

	const upsert = (target: EditableUnifiedDiffTarget) => {
		upsertEditableTarget(byKey, target);
	};

	for (const file of fallback.files) {
		const resolution = resolveEditableTargetPathsForUI(file, knownSourcePaths, workspaceRoots, [
			...(file.candidatePaths ?? []),
			file.targetPath,
			file.oldPath,
			file.newPath,
		]);

		upsert({
			fileKey: file.fileKey,
			oldPath: file.oldPath,
			newPath: file.newPath,
			targetPath: resolution.targetPath,
			resolvedPath: resolution.resolvedPath,
			candidatePaths: resolution.candidatePaths,
			knownTargetPaths: resolution.knownTargetPaths,
			hunks: file.hunks,
			addedLines: file.addedLines,
			deletedLines: file.deletedLines,
			diffText: file.diffText,
			sectionKeys: file.sectionKeys,
		});
	}

	const outputTargets = output?.fileTargets ?? [];

	for (let index = 0; index < outputTargets.length; index += 1) {
		const target = outputTargets[index];
		if (!target) {
			continue;
		}

		const parsedFile = findParsedFileForOutputSource(fallback.files, target, index, outputTargets.length);
		const effectiveTarget = {
			...target,
			fileKey: target.fileKey || parsedFile?.fileKey,
			oldPath: target.oldPath || parsedFile?.oldPath,
			newPath: target.newPath || parsedFile?.newPath,
		};
		const resolution = resolveEditableTargetPathsForUI(effectiveTarget, knownSourcePaths, workspaceRoots, [
			target.targetPath,
			target.oldPath,
			target.newPath,
			...(parsedFile?.candidatePaths ?? []),
		]);

		upsert({
			fileKey: effectiveTarget.fileKey,
			oldPath: effectiveTarget.oldPath,
			newPath: effectiveTarget.newPath,
			targetPath: resolution.targetPath,
			resolvedPath: resolution.resolvedPath,
			candidatePaths: resolution.candidatePaths,
			knownTargetPaths: resolution.knownTargetPaths,
			diffText: parsedFile?.diffText,
			sectionKeys: parsedFile?.sectionKeys,
			hunks: parsedFile?.hunks,
			addedLines: parsedFile?.addedLines,
			deletedLines: parsedFile?.deletedLines,
		});
	}

	const outputFiles = output?.files ?? [];

	for (let index = 0; index < outputFiles.length; index += 1) {
		const file = outputFiles[index];
		if (!file) {
			continue;
		}

		const parsedFile = findParsedFileForOutputSource(fallback.files, file, index, outputFiles.length);
		const effectiveFile = {
			...file,
			fileKey: file.fileKey || parsedFile?.fileKey,
			oldPath: file.oldPath || parsedFile?.oldPath,
			newPath: file.newPath || parsedFile?.newPath,
		};
		const resolution = resolveEditableTargetPathsForUI(effectiveFile, knownSourcePaths, workspaceRoots, [
			...(file.candidatePaths ?? []),
			file.targetPath,
			file.resolvedPath,
			file.oldPath,
			file.newPath,
			...(parsedFile?.candidatePaths ?? []),
		]);

		upsert({
			fileKey: effectiveFile.fileKey,
			oldPath: effectiveFile.oldPath,
			newPath: effectiveFile.newPath,
			targetPath: resolution.targetPath,
			resolvedPath: resolution.resolvedPath,
			candidatePaths: resolution.candidatePaths,
			knownTargetPaths: resolution.knownTargetPaths,
			ok: file.ok,
			status: file.status,
			message: file.message,
			diagnostics: file.diagnostics ? file.diagnostics.map(d => ({ ...d })) : undefined,
			hunks: file.hunks,
			appliedHunks: file.appliedHunks,
			alreadyAppliedHunks: file.alreadyAppliedHunks,
			addedLines: file.addedLines,
			deletedLines: file.deletedLines,
			diffText: parsedFile?.diffText,
			sectionKeys: parsedFile?.sectionKeys,
		});
	}

	if (byKey.size === 0) {
		byKey.set('file-1', {
			fileKey: 'file-1',
			targetPath: '',
			candidatePaths: [],
			knownTargetPaths: [],
			sectionKeys: ['file-1'],
			hunks: 0,
			addedLines: 0,
			deletedLines: 0,
		});
	}

	return [...byKey.values()];
}

function findParsedFileForOutputSource(
	files: ParsedUnifiedDiffFileForUI[],
	source: {
		fileKey?: string;
		oldPath?: string;
		newPath?: string;
		targetPath?: string;
	},
	index: number,
	sourceCount: number
): ParsedUnifiedDiffFileForUI | undefined {
	const directMatch = files.find(file => parsedFileMatchesTarget(file, source));
	if (directMatch) {
		return directMatch;
	}

	// Some backend implementations return normalized paths but retain file
	// order. Position is used only when both sides have exactly the same count.
	if (sourceCount === files.length) {
		return files[index];
	}

	return undefined;
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

export interface FileStatusCounts {
	total: number;
	applicable: number;
	notApplicable: number;
	blocked: number;
	needsInfo: number;
	conflict: number;
	error: number;
	applied: number;
	alreadyApplied: number;
	unknown: number;
}

export interface EditableUnifiedDiffTarget {
	fileKey?: string;
	oldPath?: string;
	newPath?: string;
	targetPath: string;
	targetPathInput?: string;
	resolvedPath?: string;
	candidatePaths: string[];
	knownTargetPaths?: string[];
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
				candidatePaths: absolutePathStrings(file.candidatePaths),
				sectionKeys: uniqueStrings([file.fileKey, ...(file.sectionKeys ?? [])]),
			};

			byIdentity.set(identity, next);
			out.push(next);
			continue;
		}

		existing.oldPath = existing.oldPath || file.oldPath;
		existing.newPath = existing.newPath || file.newPath;
		existing.targetPath = existing.targetPath || file.targetPath;
		existing.hunks += file.hunks;
		existing.addedLines += file.addedLines;
		existing.deletedLines += file.deletedLines;
		existing.candidatePaths = absolutePathStrings([
			...(existing.candidatePaths ?? []),
			...(file.candidatePaths ?? []),
			existing.targetPath,
			file.targetPath,
		]);
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
	const targetSectionKeys = getEditableTargetSectionKeys(target);

	if (targetSectionKeys.length > 0) {
		for (const [key, existing] of byKey.entries()) {
			const existingSectionKeys = getEditableTargetSectionKeys(existing);

			if (targetSectionKeys.some(sectionKey => existingSectionKeys.includes(sectionKey))) {
				return key;
			}
		}
	}

	const targetPatchPaths = getEditableTargetPatchPaths(target);

	if (targetPatchPaths.length > 0) {
		for (const [key, existing] of byKey.entries()) {
			if (haveSharedPathIdentity(getEditableTargetPatchPaths(existing), targetPatchPaths)) {
				return key;
			}
		}
	}

	const targetResolvedPaths = getEditableTargetResolvedPaths(target);

	if (targetResolvedPaths.length > 0) {
		for (const [key, existing] of byKey.entries()) {
			if (getEditableTargetPatchPaths(existing).length > 0) {
				continue;
			}
			if (haveSharedPathIdentity(getEditableTargetResolvedPaths(existing), targetResolvedPaths)) {
				return key;
			}
		}
	}

	return undefined;
}

function getEditableTargetMapKey(target: EditableUnifiedDiffTarget, index: number): string {
	return getEditableTargetIdentity(target) || `target-${index}`;
}

function getEditableTargetSectionKeys(target: { fileKey?: string; sectionKeys?: string[] }): string[] {
	return uniqueStrings([target.fileKey, ...(target.sectionKeys ?? [])]);
}

function getEditableTargetPatchPaths(target: { oldPath?: string; newPath?: string }): string[] {
	return uniqueStrings([target.newPath, target.oldPath].filter(p => isUsablePatchPath(p)));
}

function getEditableTargetResolvedPaths(target: { targetPath?: string; resolvedPath?: string }): string[] {
	return absolutePathStrings([target.resolvedPath, target.targetPath]);
}

export function haveSharedPathIdentity(left: Array<string | undefined>, right: Array<string | undefined>): boolean {
	const leftSet = new Set(left.map(p => getPathIdentity(p)).filter(Boolean));
	if (leftSet.size === 0) {
		return false;
	}
	return right.map(p => getPathIdentity(p)).some(identity => !!identity && leftSet.has(identity));
}

export function buildFileStatusCounts(
	output: ApplyUnifiedDiffOut | undefined,
	fallbackParsed: ReturnType<typeof parseUnifiedDiffForUI>
): FileStatusCounts {
	const files = output?.files ?? [];
	const total = Math.max(output?.summary?.files ?? 0, files.length, fallbackParsed.files.length);
	const counts: FileStatusCounts = {
		total,
		applicable: 0,
		notApplicable: 0,
		blocked: 0,
		needsInfo: 0,
		conflict: 0,
		error: 0,
		applied: 0,
		alreadyApplied: 0,
		unknown: 0,
	};

	if (files.length === 0 && output) {
		switch (output.status) {
			case ApplyUnifiedDiffStatus.Applicable:
				if (output.ok) {
					counts.applicable = total;
				} else {
					counts.notApplicable = total;
					counts.blocked = total;
				}
				break;
			case ApplyUnifiedDiffStatus.Applied:
				counts.applied = total;
				break;
			case ApplyUnifiedDiffStatus.AlreadyApplied:
				counts.alreadyApplied = total;
				break;
			case ApplyUnifiedDiffStatus.NeedsInfo:
				counts.needsInfo = total;
				counts.blocked = total;
				break;
			case ApplyUnifiedDiffStatus.Conflict:
				counts.conflict = total;
				counts.blocked = total;
				break;

			default:
				counts.error = total;
				counts.blocked = total;
				break;
		}
	} else {
		for (const file of files) {
			switch (file.status) {
				case ApplyUnifiedDiffStatus.Applicable:
					if (file.ok) {
						counts.applicable += 1;
					} else {
						counts.notApplicable += 1;
						counts.blocked += 1;
					}
					break;
				case ApplyUnifiedDiffStatus.Applied:
					counts.applied += 1;
					break;
				case ApplyUnifiedDiffStatus.AlreadyApplied:
					counts.alreadyApplied += 1;
					break;
				case ApplyUnifiedDiffStatus.NeedsInfo:
					counts.needsInfo += 1;
					counts.blocked += 1;
					break;
				case ApplyUnifiedDiffStatus.Conflict:
					counts.conflict += 1;
					counts.blocked += 1;
					break;

				default:
					counts.error += 1;
					counts.blocked += 1;
					break;
			}
		}

		counts.unknown = Math.max(0, total - files.length);
	}

	return counts;
}

function mergeEditableTarget(
	existing: EditableUnifiedDiffTarget | undefined,
	update: EditableUnifiedDiffTarget
): EditableUnifiedDiffTarget {
	const updateTargetPath = toAbsolutePath(update.targetPath);
	const updateResolvedPath = toAbsolutePath(update.resolvedPath);
	const updateKnownTargetPaths = absolutePathStrings(update.knownTargetPaths ?? []);
	const updateIsNewFile = isNewUnifiedDiffFile(update);
	const safeUpdateTargetPath =
		updateIsNewFile && updateTargetPath && !pathListIncludes(updateKnownTargetPaths, updateTargetPath)
			? ''
			: updateTargetPath;
	const safeUpdateResolvedPath =
		updateIsNewFile && updateResolvedPath && !pathListIncludes(updateKnownTargetPaths, updateResolvedPath)
			? ''
			: updateResolvedPath;

	if (!existing) {
		return {
			...update,
			targetPath: safeUpdateTargetPath,
			resolvedPath: safeUpdateResolvedPath || undefined,
			candidatePaths: absolutePathStrings([
				...updateKnownTargetPaths,
				...(update.candidatePaths ?? []),
				safeUpdateTargetPath,
			]),
			knownTargetPaths: updateKnownTargetPaths,
			sectionKeys: uniqueStrings([update.fileKey, ...(update.sectionKeys ?? [])]),
		};
	}

	const existingTargetPath = toAbsolutePath(existing.targetPath);
	const existingResolvedPath = toAbsolutePath(existing.resolvedPath);
	const oldPath = update.oldPath || existing.oldPath;
	const newPath = update.newPath || existing.newPath;
	const knownTargetPaths = absolutePathStrings([...(existing.knownTargetPaths ?? []), ...updateKnownTargetPaths]);
	const mergedIsNewFile = isNewUnifiedDiffFile({ oldPath, newPath });
	const targetPath =
		mergedIsNewFile && updateTargetPath && !pathListIncludes(knownTargetPaths, updateTargetPath)
			? existingTargetPath
			: updateTargetPath || existingTargetPath;
	const nextResolvedPath = updateResolvedPath || existingResolvedPath;
	const resolvedPath =
		mergedIsNewFile && nextResolvedPath && !pathListIncludes(knownTargetPaths, nextResolvedPath)
			? pathListIncludes(knownTargetPaths, existingResolvedPath)
				? existingResolvedPath
				: ''
			: nextResolvedPath;
	const mergedCandidatePaths = absolutePathStrings([
		...(existing.candidatePaths ?? []),
		...(update.candidatePaths ?? []),
		...knownTargetPaths,
		existingTargetPath,
		targetPath,
	]);
	const candidatePaths = mergedIsNewFile
		? mergedCandidatePaths.filter(
				path =>
					pathListIncludes(knownTargetPaths, path) || getComparablePathKey(path) === getComparablePathKey(targetPath)
			)
		: mergedCandidatePaths;

	return {
		...existing,
		...update,
		fileKey: existing.fileKey || update.fileKey,
		oldPath,
		newPath,
		targetPath,
		targetPathInput: update.targetPathInput ?? existing.targetPathInput,
		resolvedPath: resolvedPath || undefined,
		candidatePaths,
		knownTargetPaths,
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
		hunks: mergeNumberMax(existing.hunks, update.hunks),
		appliedHunks: mergeNumberMax(existing.appliedHunks, update.appliedHunks),
		alreadyAppliedHunks: mergeNumberMax(existing.alreadyAppliedHunks, update.alreadyAppliedHunks),
		addedLines: mergeNumberMax(existing.addedLines, update.addedLines),
		deletedLines: mergeNumberMax(existing.deletedLines, update.deletedLines),
	};
}

function resolveEditableTargetPathsForUI(
	file: {
		targetPath?: string;
		resolvedPath?: string;
		oldPath?: string;
		newPath?: string;
		candidatePaths?: string[];
	},
	knownCandidatePaths: string[],
	knownWorkspaceRoots: string[],
	localCandidatePaths: Array<string | undefined | null>
): EditableTargetPathResolutionForUI {
	const patchPaths = getEditableTargetPatchPaths(file);
	const externalKnownCandidatePaths = absolutePathStrings(knownCandidatePaths).filter(
		path => !isPatchPathEcho(path, patchPaths)
	);
	const externalWorkspaceRoots = absolutePathStrings(knownWorkspaceRoots).filter(
		path => !isPatchPathEcho(path, patchPaths)
	);
	const workspaceRootKeys = new Set(externalWorkspaceRoots.map(path => getComparablePathKey(path)).filter(Boolean));
	const knownSourcePaths = absolutePathStrings([...externalKnownCandidatePaths, ...externalWorkspaceRoots]);
	const knownInferences = inferTargetPathsForUI(file, knownSourcePaths, workspaceRootKeys);
	const isNewFile = isNewUnifiedDiffFile(file);
	const localSourcePaths = absolutePathStrings([
		...localCandidatePaths,
		file.targetPath,
		file.resolvedPath,
		file.oldPath,
		file.newPath,
	]);
	const localInferences = isNewFile ? [] : inferTargetPathsForUI(file, localSourcePaths);
	const knownTargetPaths = uniqueStrings(knownInferences.map(inference => inference.targetPath));
	const directLocalTargetPaths = absolutePathStrings([file.resolvedPath, file.targetPath]).filter(path =>
		isResolvedPathCompatibleWithPatchPath(path, file)
	);
	const fallbackTargetPaths = uniqueStrings([
		...localInferences.map(inference => inference.targetPath),
		...directLocalTargetPaths,
	]);
	const targetPath = chooseEditableTargetPath(file, knownInferences, localInferences);
	const rawResolvedPath = toAbsolutePath(file.resolvedPath);
	const resolvedPath =
		rawResolvedPath &&
		isResolvedPathCompatibleWithPatchPath(rawResolvedPath, file) &&
		(!isNewFile || pathListIncludes(knownTargetPaths, rawResolvedPath))
			? rawResolvedPath
			: undefined;

	return {
		targetPath,
		resolvedPath,
		candidatePaths: isNewFile
			? absolutePathStrings([...knownTargetPaths, targetPath])
			: absolutePathStrings([...knownTargetPaths, ...fallbackTargetPaths, targetPath, resolvedPath]),
		knownTargetPaths,
	};
}

function chooseEditableTargetPath(
	file: {
		targetPath?: string;
		resolvedPath?: string;
		oldPath?: string;
		newPath?: string;
	},
	knownInferences: TargetPathInferenceForUI[],
	localInferences: TargetPathInferenceForUI[]
): string {
	for (const selectedPath of absolutePathStrings([file.targetPath, file.resolvedPath])) {
		const knownMatch = knownInferences.find(inference => pathListIncludes([inference.targetPath], selectedPath));
		if (knownMatch) {
			return knownMatch.targetPath;
		}
	}

	const bestKnownTargets = getBestInferredTargetPaths(knownInferences);
	if (bestKnownTargets.length > 0) {
		return bestKnownTargets.length === 1 ? (bestKnownTargets[0] ?? '') : '';
	}

	// A new file has no existing filesystem object to validate. Patch paths,
	// backend guesses, and root-relative absolute paths are therefore never
	// automatic fallbacks.
	if (isNewUnifiedDiffFile(file)) {
		return '';
	}

	for (const selectedPath of absolutePathStrings([file.resolvedPath, file.targetPath])) {
		if (isResolvedPathCompatibleWithPatchPath(selectedPath, file)) {
			return selectedPath;
		}
	}

	const bestLocalTargets = getBestInferredTargetPaths(localInferences);
	if (bestLocalTargets.length > 0) {
		return bestLocalTargets.length === 1 ? (bestLocalTargets[0] ?? '') : '';
	}

	return (
		getPatchPathsForTargetInference(file)
			.map(f => {
				return toAbsolutePath(f);
			})
			.find(Boolean) ?? ''
	);
}

function isResolvedPathCompatibleWithPatchPath(
	resolvedPath: string,
	file: {
		oldPath?: string;
		newPath?: string;
	}
): boolean {
	const resolvedKey = normalizePathKey(resolvedPath);
	if (!resolvedKey) {
		return false;
	}

	const patchPaths = getEditableTargetPatchPaths(file);
	if (patchPaths.length === 0) {
		return true;
	}

	return patchPaths.some(patchPath => pathHasSuffixPath(resolvedKey, patchPath));
}

export function isNewUnifiedDiffFile(file: { oldPath?: string; newPath?: string }): boolean {
	return file.oldPath?.trim() === DEV_NULL && isUsablePatchPath(file.newPath);
}

function isDeletedUnifiedDiffFile(file: { oldPath?: string; newPath?: string }): boolean {
	return file.newPath?.trim() === DEV_NULL && isUsablePatchPath(file.oldPath);
}

function getPatchPathsForTargetInference(file: { oldPath?: string; newPath?: string }): string[] {
	const oldPath = isUsablePatchPath(file.oldPath) ? file.oldPath.trim() : '';
	const newPath = isUsablePatchPath(file.newPath) ? file.newPath.trim() : '';

	if (isNewUnifiedDiffFile(file)) {
		return uniqueStrings([newPath]);
	}
	if (isDeletedUnifiedDiffFile(file)) {
		return uniqueStrings([oldPath]);
	}

	// For a rename, the old path identifies the existing local object that
	// the patch must locate. For a normal update the paths are identical.
	if (oldPath && newPath && getComparablePathKey(oldPath) !== getComparablePathKey(newPath)) {
		return [oldPath];
	}

	return uniqueStrings([newPath, oldPath]);
}

function inferTargetPathsForUI(
	file: {
		oldPath?: string;
		newPath?: string;
		candidatePaths?: string[];
	},
	candidatePaths: string[],
	explicitWorkspaceRootKeys: ReadonlySet<string> = new Set<string>()
): TargetPathInferenceForUI[] {
	const inferences: TargetPathInferenceForUI[] = [];
	const patchPaths = getPatchPathsForTargetInference(file);
	const sources = absolutePathStrings([...(file.candidatePaths ?? []), ...candidatePaths]);

	for (let patchIndex = 0; patchIndex < patchPaths.length; patchIndex += 1) {
		const patchPathRaw = patchPaths[patchIndex];
		const patchPath = normalizePathKey(patchPathRaw);
		if (!patchPath) {
			continue;
		}

		const relativePatchPath = toSafeRootRelativePatchPath(patchPathRaw);
		const patchDir = dirnamePathKey(relativePatchPath);
		const patchDirParts = getPathParts(patchDir);
		const patchPriority = Math.max(0, 100 - patchIndex);

		for (const candidateRaw of sources) {
			const absoluteCandidate = toAbsolutePath(candidateRaw);
			const candidate = normalizePathKey(absoluteCandidate);
			if (!candidate) {
				continue;
			}

			const candidateKey = getComparablePathKey(candidate);
			const isExplicitWorkspaceRoot = explicitWorkspaceRootKeys.has(candidateKey);
			const candidateLooksDirectory = isExplicitWorkspaceRoot || looksLikeDirectoryPath(candidateRaw);

			if (!candidateLooksDirectory && getComparablePathKey(candidate) === getComparablePathKey(patchPath)) {
				inferences.push({
					targetPath: absoluteCandidate,
					sourcePath: candidateRaw,
					score: 100_000 + patchPriority,
				});
			}

			if (!relativePatchPath) {
				continue;
			}

			if (!candidateLooksDirectory && pathHasSuffixPath(candidate, relativePatchPath)) {
				inferences.push({
					targetPath: absoluteCandidate,
					sourcePath: candidateRaw,
					score: 90_000 + getPathParts(relativePatchPath).length * 100 + patchPriority,
				});
			}

			const candidateDir = candidateLooksDirectory ? trimTrailingSlashes(candidate) : dirnamePathKey(candidate);
			let foundDirectoryAlignment = false;

			if (candidateDir && patchDirParts.length > 0) {
				const candidateDirParts = getPathParts(candidateDir);
				const maxPrefix = Math.min(patchDirParts.length, candidateDirParts.length);

				for (let prefixLength = maxPrefix; prefixLength >= 1; prefixLength -= 1) {
					const prefix = patchDirParts.slice(0, prefixLength).join('/');
					const root = trimPathSuffix(candidateDir, prefix);

					if (root === undefined) {
						continue;
					}

					foundDirectoryAlignment = true;
					const targetPath = toAbsolutePath(joinPathKey(root, relativePatchPath));
					if (targetPath) {
						inferences.push({
							targetPath,
							sourcePath: candidateRaw,
							score: (prefixLength === patchDirParts.length ? 80_000 : 70_000) + prefixLength * 100 + patchPriority,
						});
					}
					continue;
				}
			}

			// A path explicitly supplied as a workspace root may always be
			// joined with the safe patch-relative path. A trailing-slash folder
			// is treated as a root only when no stronger directory alignment
			// exists.
			if (candidateLooksDirectory && (!foundDirectoryAlignment || isExplicitWorkspaceRoot)) {
				const targetPath = toAbsolutePath(joinPathKey(candidate, relativePatchPath));
				if (targetPath) {
					inferences.push({
						targetPath,
						sourcePath: candidateRaw,
						score: 60_000 + patchPriority,
					});
				}
			}
		}
	}

	return sortAndDedupeTargetInferences(inferences);
}

function getBestInferredTargetPaths(inferences: TargetPathInferenceForUI[]): string[] {
	if (inferences.length === 0) {
		return [];
	}

	const bestScore = inferences[0].score;
	return uniqueStrings(
		inferences.filter(candidate => candidate.score === bestScore).map(candidate => candidate.targetPath)
	);
}

function sortAndDedupeTargetInferences(inferences: TargetPathInferenceForUI[]): TargetPathInferenceForUI[] {
	const byTarget = new Map<string, TargetPathInferenceForUI>();

	for (const inference of inferences) {
		const targetPath = toAbsolutePath(inference.targetPath);
		const key = normalizePathKey(targetPath);
		if (!targetPath || !key) {
			continue;
		}

		const existing = byTarget.get(key);
		if (!existing || inference.score > existing.score) {
			byTarget.set(key, { ...inference, targetPath });
		}
	}

	return [...byTarget.values()].toSorted((left, right) => {
		if (left.score !== right.score) {
			return right.score - left.score;
		}
		return normalizePathKey(left.targetPath).localeCompare(normalizePathKey(right.targetPath));
	});
}

function isPatchPathEcho(candidatePath: string, patchPaths: Array<string | undefined>): boolean {
	const candidateKey = getComparablePathKey(candidatePath);
	const candidateRelative = toSafeRootRelativePatchPath(candidatePath);

	return patchPaths.some(patchPath => {
		const patchKey = getComparablePathKey(patchPath);
		if (!patchKey) {
			return false;
		}
		if (candidateKey === patchKey) {
			return true;
		}

		const patchRelative = toSafeRootRelativePatchPath(patchPath);
		return !!candidateRelative && !!patchRelative && candidateRelative === patchRelative;
	});
}

function pathListIncludes(paths: Array<string | undefined>, value: string | undefined): boolean {
	const valueKey = getComparablePathKey(value);
	return !!valueKey && paths.some(path => getComparablePathKey(path) === valueKey);
}

function getComparablePathKey(value: string | undefined): string {
	return trimTrailingSlashes(normalizePathKey(value));
}

function toSafeRootRelativePatchPath(value: string | undefined): string {
	const trimmed = value?.trim() ?? '';
	if (trimmed.startsWith('\\\\') || trimmed.startsWith('//')) {
		return '';
	}

	let normalized = normalizePathKey(trimmed);
	if (/^[A-Za-z]:\//.test(normalized)) {
		normalized = normalized.slice(3);
	} else {
		normalized = normalized.replace(/^\/+/, '');
	}

	const parts = normalized.split('/').filter(Boolean);
	if (parts.length === 0 || parts.some(part => part === '.' || part === '..')) {
		return '';
	}

	return parts.join('/');
}

function looksLikeDirectoryPath(value: string): boolean {
	const trimmed = value.trim();
	return trimmed.endsWith('/') || trimmed.endsWith('\\');
}

function dirnamePathKey(value: string): string {
	const normalized = normalizePathKey(value);
	if (!normalized) {
		return '';
	}
	const index = normalized.lastIndexOf('/');
	if (index < 0) {
		return '';
	}
	if (index === 0) {
		return '/';
	}
	return normalized.slice(0, index);
}

function getPathParts(value: string): string[] {
	return trimTrailingSlashes(value).replace(/^\/+/, '').split('/').filter(Boolean);
}

function trimPathSuffix(value: string, suffix: string): string | undefined {
	const normalizedValue = trimTrailingSlashes(normalizePathKey(value));
	const normalizedSuffix = trimTrailingSlashes(normalizePathKey(suffix)).replace(/^\/+/, '');
	if (!normalizedValue || !normalizedSuffix) {
		return undefined;
	}
	if (normalizedValue === normalizedSuffix) {
		return '';
	}
	const marker = `/${normalizedSuffix}`;
	if (!normalizedValue.endsWith(marker)) {
		return undefined;
	}
	return normalizedValue.slice(0, normalizedValue.length - marker.length);
}

function pathHasSuffixPath(value: string, suffix: string): boolean {
	const normalizedValue = trimTrailingSlashes(normalizePathKey(value));
	const normalizedSuffix = trimTrailingSlashes(normalizePathKey(suffix)).replace(/^\/+/, '');
	if (!normalizedValue || !normalizedSuffix) {
		return false;
	}
	return normalizedValue === normalizedSuffix || normalizedValue.endsWith(`/${normalizedSuffix}`);
}

function joinPathKey(root: string, rel: string): string {
	const cleanRoot = trimTrailingSlashes(normalizePathKey(root));
	const cleanRel = normalizePathKey(rel).replace(/^\/+/, '');
	if (!cleanRoot || !cleanRel) {
		return '';
	}
	if (cleanRoot === '/') {
		return `/${cleanRel}`;
	}
	return `${cleanRoot}/${cleanRel}`;
}

function trimTrailingSlashes(value: string): string {
	if (value === '/') {
		return value;
	}
	return value.replaceAll(/\/+$/g, '');
}

export function mergeNumberMax(left: number | undefined, right: number | undefined): number | undefined {
	if (typeof left === 'number' && typeof right === 'number') {
		return Math.max(left, right);
	}
	if (typeof right === 'number') {
		return right;
	}
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
	if (targetIdentity && targetIdentity === getPatchFileIdentity(file)) {
		return true;
	}

	const filePaths = uniqueStrings([file.targetPath, file.newPath, file.oldPath, ...file.candidatePaths]);
	const targetPaths = uniqueStrings([target.newPath, target.oldPath]);
	if (targetPaths.length === 0 && target.targetPath?.trim()) {
		targetPaths.push(target.targetPath.trim());
	}

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
	const patchPath = getEditableTargetPatchPaths(file)[0];
	if (patchPath) {
		return `path:${normalizePathKey(patchPath)}`;
	}

	const targetPath = normalizePathKey(toAbsolutePath(file.targetPath));
	if (targetPath) {
		return `path:${targetPath}`;
	}

	const resolvedPath = normalizePathKey(toAbsolutePath(file.resolvedPath));
	if (resolvedPath) {
		return `path:${resolvedPath}`;
	}

	return '';
}

function getEditableTargetIdentity(file: {
	fileKey?: string;
	sectionKeys?: string[];
	targetPath?: string;
	resolvedPath?: string;
	oldPath?: string;
	newPath?: string;
}): string {
	const fileKey = file.fileKey?.trim();
	if (fileKey) {
		return `file:${fileKey}`;
	}

	const sectionKeys = uniqueStrings(file.sectionKeys ?? []);
	if (sectionKeys.length > 0) {
		return `section:${sectionKeys.join('|')}`;
	}

	const patchPath = getEditableTargetPatchPaths(file)[0];
	if (patchPath) {
		return `patch:${normalizePathKey(patchPath)}`;
	}

	const resolvedPath = getEditableTargetResolvedPaths(file)[0];
	if (resolvedPath) {
		return `path:${normalizePathKey(resolvedPath)}`;
	}

	return '';
}

function joinUnifiedDiffTextParts(parts: Array<string | undefined>): string | undefined {
	const normalized = parts.map(part => part?.replaceAll(/\n+$/g, '')).filter((part): part is string => !!part);
	if (normalized.length === 0) {
		return undefined;
	}
	return normalized.join('\n');
}

function countHunkHeaders(diffText: string): number {
	return diffText.split('\n').filter(line => line.startsWith('@@')).length;
}

function parseUnifiedDiffHunkLineCounts(line: string): { oldLines: number; newLines: number } | undefined {
	const match = /^@@\s+-(\d+)(?:,(\d+))?\s+\+(\d+)(?:,(\d+))?\s+@@/.exec(line);
	if (!match) {
		return undefined;
	}

	const oldLines = match[2] === undefined ? 1 : Number(match[2]);
	const newLines = match[4] === undefined ? 1 : Number(match[4]);

	if (!Number.isSafeInteger(oldLines) || !Number.isSafeInteger(newLines)) {
		return undefined;
	}

	return { oldLines, newLines };
}

function normalizePathKey(value: string | undefined): string {
	if (!isUsablePatchPath(value)) {
		return '';
	}
	return value
		.trim()
		.replaceAll('\\', '/')
		.replaceAll(/\/+/g, '/')
		.replace(/^(?:\.\/)+/, '');
}

function parseDiffHeaderPath(input: string): string {
	return normalizeDiffPathToken(input);
}

function looksLikeOpenAIPatch(value: string): boolean {
	return /^\*\*\*\s+Begin\s+Patch\s*$/im.test(value) || /^\*\*\*\s+(?:Update|Add|Delete)\s+File:\s+/im.test(value);
}

function parseOpenAIPatchFileHeader(line: string): { kind: 'update' | 'add' | 'delete'; path: string } | undefined {
	const match = OPENAI_PATCH_FILE_HEADER_RE.exec(line);
	if (!match?.[1] || !match[2]) {
		return undefined;
	}

	const kind = match[1].trim().toLowerCase();
	const path = normalizeOpenAIPatchPath(match[2]);
	if (!path) {
		return undefined;
	}

	return {
		kind: kind === 'add' ? 'add' : kind === 'delete' ? 'delete' : 'update',
		path,
	};
}

function parseOpenAIPatchMoveTo(line: string): string | undefined {
	const match = OPENAI_PATCH_MOVE_TO_RE.exec(line);
	return match?.[1] ? normalizeOpenAIPatchPath(match[1]) || undefined : undefined;
}

function parseGitExtendedHeaderPath(input: string): string {
	const value = input.trim();
	if (!value) {
		return '';
	}

	return value.startsWith('"') ? readDiffPathToken(value).token.trim() : value;
}

function normalizeOpenAIPatchPath(input: string): string {
	const value = input.trim();
	if (!value) {
		return '';
	}
	return value.startsWith('"') ? readDiffPathToken(value).token.trim() : value;
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
	if (!value) {
		return { token: '', rest: '' };
	}

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
	if (!value) {
		return '';
	}

	if (value === DEV_NULL) {
		return DEV_NULL;
	}

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
	if (value.startsWith(marker)) {
		return value.slice(marker.length);
	}
	return value;
}

function isUsablePatchPath(value: unknown): value is string {
	return typeof value === 'string' && value.trim().length > 0 && value.trim() !== DEV_NULL;
}

export function uniqueStrings(values: Array<string | undefined | null>): string[] {
	const out: string[] = [];
	const seen = new Set<string>();

	for (const value of values) {
		const trimmed = value?.trim();
		if (!trimmed) {
			continue;
		}

		const key = trimmed.replaceAll('\\', '/').replaceAll(/\/+/g, '/');

		if (seen.has(key)) {
			continue;
		}

		seen.add(key);
		out.push(trimmed);
	}

	return out;
}

export function getPathIdentity(value: string | undefined | null): string {
	const trimmed = value?.trim();
	if (!trimmed || trimmed === '/dev/null') {
		return '';
	}

	return trimmed
		.replaceAll('\\', '/')
		.replaceAll(/\/+/g, '/')
		.replace(/^(?:\.\/)+/, '');
}

export function getErrorMessage(error: unknown): string {
	if (error instanceof Error && error.message.trim()) {
		return error.message;
	}
	return 'Unexpected error while checking or applying unified diff.';
}
