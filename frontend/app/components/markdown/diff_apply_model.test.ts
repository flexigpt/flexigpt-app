// oxlint-disable typescript/no-non-null-assertion
import { describe, expect, it } from 'vitest';

import type { ApplyUnifiedDiffOut } from '@/spec/unified_diff';
import { ApplyUnifiedDiffStatus } from '@/spec/unified_diff';

import {
	buildInteractiveDiffRequest,
	inferInteractiveDiffTargets,
	interpretScopedApplyResult,
	normalizeInteractiveTargetPath,
	parseInteractiveDiff,
} from '@/components/markdown/diff_apply_model';

function addFile(path: string, content: string): string {
	return ['--- /dev/null', `+++ b/${path}`, '@@ -0,0 +1,1 @@', `+${content}`].join('\n');
}

function asOutput(value: Partial<ApplyUnifiedDiffOut>): ApplyUnifiedDiffOut {
	return value as ApplyUnifiedDiffOut;
}

describe('parseInteractiveDiff', () => {
	it('keeps multiple new files separate even though each uses /dev/null', () => {
		const parsed = parseInteractiveDiff(
			[addFile('src/a.ts', 'export const a = 1;'), addFile('src/b.ts', 'export const b = 2;')].join('\n'),
			'diff'
		);

		expect(parsed.diagnostics).toEqual([]);
		expect(parsed.files).toHaveLength(2);
		expect(parsed.files.map(file => file.newPath)).toEqual(['src/a.ts', 'src/b.ts']);

		const first = buildInteractiveDiffRequest(parsed.files[0]!, '/repo/src/a.ts');
		const second = buildInteractiveDiffRequest(parsed.files[1]!, '/repo/src/b.ts');

		expect(first).toContain('+++ b/src/a.ts');
		expect(first).not.toContain('+++ b/src/b.ts');
		expect(second).toContain('+++ b/src/b.ts');
		expect(second).not.toContain('+++ b/src/a.ts');
	});

	it('does not merge repeated sections that share the same path', () => {
		const parsed = parseInteractiveDiff(
			[
				'--- a/src/value.ts',
				'+++ b/src/value.ts',
				'@@ -1,1 +1,1 @@',
				'-first',
				'+second',
				'--- a/src/value.ts',
				'+++ b/src/value.ts',
				'@@ -10,1 +10,1 @@',
				'-third',
				'+fourth',
			].join('\n'),
			'diff'
		);

		expect(parsed.files).toHaveLength(2);
		expect(parsed.files.map(file => file.id)).toEqual(['section-1', 'section-2']);
		expect(parsed.files.every(file => file.canApplyWhole)).toBe(true);
	});

	it('treats --- and +++ as payload while a hunk is consuming lines', () => {
		const parsed = parseInteractiveDiff(
			['--- a/example.txt', '+++ b/example.txt', '@@ -1,1 +1,1 @@', '--- old payload', '+++ new payload'].join('\n'),
			'diff'
		);

		const file = parsed.files[0];

		expect(file?.canApplyWhole).toBe(true);
		expect(file?.deletedLines).toBe(1);
		expect(file?.addedLines).toBe(1);
	});

	it('marks incomplete hunks as blocked', () => {
		const parsed = parseInteractiveDiff(
			['--- a/example.txt', '+++ b/example.txt', '@@ -1,2 +1,2 @@', '-old', '+new'].join('\n'),
			'diff'
		);

		const file = parsed.files[0];

		expect(file?.canApplyWhole).toBe(false);
		expect(file?.diagnostics.some(diagnostic => diagnostic.code === 'incomplete_hunk')).toBe(true);
	});

	it('converts multiple OpenAI Add File sections independently', () => {
		const parsed = parseInteractiveDiff(
			[
				'*** Begin Patch',
				'*** Add File: src/first.ts',
				'+export const first = true;',
				'*** Add File: src/second.ts',
				'+export const second = true;',
				'*** End Patch',
			].join('\n')
		);

		expect(parsed.files).toHaveLength(2);
		expect(parsed.files.every(file => file.canApplyWhole)).toBe(true);
		expect(parsed.files.map(file => file.newPath)).toEqual(['src/first.ts', 'src/second.ts']);

		const first = buildInteractiveDiffRequest(parsed.files[0]!, '/repo/src/first.ts');
		const second = buildInteractiveDiffRequest(parsed.files[1]!, '/repo/src/second.ts');

		expect(first).toContain('+++ b/src/first.ts');
		expect(second).toContain('+++ b/src/second.ts');
	});

	it('keeps unsupported OpenAI Update sections isolated from supported Add sections', () => {
		const parsed = parseInteractiveDiff(
			[
				'*** Begin Patch',
				'*** Add File: src/new.ts',
				'+export const value = 1;',
				'*** Update File: src/existing.ts',
				'@@',
				'-old',
				'+new',
				'*** End Patch',
			].join('\n')
		);

		expect(parsed.files).toHaveLength(2);
		expect(parsed.files[0]?.canApplyWhole).toBe(true);
		expect(parsed.files[1]?.canApplyWhole).toBe(false);
	});

	it('builds a valid selected-hunk patch with corrected new-side coordinates', () => {
		const parsed = parseInteractiveDiff(
			[
				'--- a/example.txt',
				'+++ b/example.txt',
				'@@ -2,1 +2,2 @@',
				' existing',
				'+inserted',
				'@@ -10,1 +11,1 @@',
				'-old',
				'+new',
			].join('\n'),
			'diff'
		);

		const file = parsed.files[0];

		expect(file?.canApplyPartial).toBe(true);

		const scoped = buildInteractiveDiffRequest(file!, '/repo/example.txt', [file!.hunks[1]!.id]);

		expect(scoped).toContain('@@ -10 +10 @@');
		expect(scoped).not.toContain('+inserted');

		const verified = parseInteractiveDiff(scoped, 'diff');
		expect(verified.files).toHaveLength(1);
		expect(verified.files[0]?.hunks).toHaveLength(1);
		expect(verified.files[0]?.canApplyWhole).toBe(true);
	});
});

describe('inferInteractiveDiffTargets', () => {
	it('selects a new-file target under one explicit workspace root', () => {
		const parsed = parseInteractiveDiff(addFile('src/new.ts', 'export const value = 1;'));
		const file = parsed.files[0]!;
		const suggestions = inferInteractiveDiffTargets(parsed.files, [], ['/repo']);

		expect(suggestions.get(file.id)?.targetPath).toBe('/repo/src/new.ts');
	});

	it('requires explicit user choice when multiple roots are possible', () => {
		const parsed = parseInteractiveDiff(addFile('src/new.ts', 'export const value = 1;'));
		const file = parsed.files[0]!;
		const suggestions = inferInteractiveDiffTargets(parsed.files, [], ['/one', '/two']);

		expect(suggestions.get(file.id)?.targetPath).toBe('');
		expect(suggestions.get(file.id)?.candidates).toEqual(['/one/src/new.ts', '/two/src/new.ts']);
	});

	it('derives a workspace root from an external sibling candidate', () => {
		const parsed = parseInteractiveDiff(
			[
				addFile('src/new.ts', 'export const next = 1;'),
				'--- a/src/existing.ts',
				'+++ b/src/existing.ts',
				'@@ -1,1 +1,1 @@',
				'-old',
				'+new',
			].join('\n')
		);

		const suggestion = inferInteractiveDiffTargets(parsed.files, ['/repo/src/existing.ts'], []);

		expect(suggestion.get(parsed.files[0]!.id)?.targetPath).toBe('/repo/src/new.ts');
	});

	it('rejects dot-segment target paths before a request can be sent', () => {
		expect(normalizeInteractiveTargetPath('/repo/../outside.ts')).toBe('');
		expect(normalizeInteractiveTargetPath('C:\\repo\\..\\outside.ts')).toBe('');
		expect(normalizeInteractiveTargetPath('/repo/src/file.ts')).toBe('/repo/src/file.ts');
		expect(normalizeInteractiveTargetPath('C:\\repo\\src\\file.ts')).toBe('C:/repo/src/file.ts');
	});
});

describe('interpretScopedApplyResult', () => {
	it('maps an applicable one-file dry run to ready', () => {
		const outcome = interpretScopedApplyResult(
			asOutput({
				ok: true,
				dryRun: true,
				status: ApplyUnifiedDiffStatus.Applicable,
				files: [
					{
						ok: true,
						status: ApplyUnifiedDiffStatus.Applicable,
						targetPath: '/repo/src/a.ts',
						fileKey: '',
						hunks: 0,
						appliedHunks: 0,
						alreadyAppliedHunks: 0,
						addedLines: 0,
						deletedLines: 0,
					},
				],
			}),
			'dry-run',
			'/repo/src/a.ts'
		);

		expect(outcome.status).toBe('ready');
	});

	it('rejects a response that names another absolute target path', () => {
		const outcome = interpretScopedApplyResult(
			asOutput({
				ok: true,
				dryRun: true,
				status: ApplyUnifiedDiffStatus.Applicable,
				files: [
					{
						ok: true,
						status: ApplyUnifiedDiffStatus.Applicable,
						targetPath: '/repo/src/other.ts',
						fileKey: '',
						hunks: 0,
						appliedHunks: 0,
						alreadyAppliedHunks: 0,
						addedLines: 0,
						deletedLines: 0,
					},
				],
			}),
			'dry-run',
			'/repo/src/a.ts'
		);

		expect(outcome.status).toBe('blocked');
		expect(outcome.message).toContain('different target path');
	});

	it('rejects a backend response containing more than one file result', () => {
		const outcome = interpretScopedApplyResult(
			asOutput({
				ok: true,
				dryRun: true,
				status: ApplyUnifiedDiffStatus.Applicable,
				files: [
					{
						ok: true,
						status: ApplyUnifiedDiffStatus.Applicable,
						fileKey: '',
						hunks: 0,
						appliedHunks: 0,
						alreadyAppliedHunks: 0,
						addedLines: 0,
						deletedLines: 0,
					},
					{
						ok: true,
						status: ApplyUnifiedDiffStatus.Applicable,
						fileKey: '',
						hunks: 0,
						appliedHunks: 0,
						alreadyAppliedHunks: 0,
						addedLines: 0,
						deletedLines: 0,
					},
				],
			}),
			'dry-run',
			'/repo/src/a.ts'
		);

		expect(outcome.status).toBe('blocked');
		expect(outcome.message).toContain('multiple file results');
	});

	it('accepts AlreadyApplied without treating it as a failed dry run', () => {
		const outcome = interpretScopedApplyResult(
			asOutput({
				ok: true,
				dryRun: true,
				status: ApplyUnifiedDiffStatus.AlreadyApplied,
				files: [
					{
						ok: true,
						status: ApplyUnifiedDiffStatus.AlreadyApplied,
						fileKey: '',
						hunks: 0,
						appliedHunks: 0,
						alreadyAppliedHunks: 0,
						addedLines: 0,
						deletedLines: 0,
					},
				],
			}),
			'dry-run',
			'/repo/src/a.ts'
		);

		expect(outcome.status).toBe('already-applied');
	});
});
