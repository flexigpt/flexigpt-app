import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { FiAlertTriangle, FiCheckCircle, FiGitPullRequest, FiInfo, FiLoader } from 'react-icons/fi';

import {
	type ApplyUnifiedDiffDiagnostic,
	ApplyUnifiedDiffDiagnosticLevel,
	type ApplyUnifiedDiffFileOut,
	type ApplyUnifiedDiffFileTarget,
	type ApplyUnifiedDiffOut,
	ApplyUnifiedDiffStatus,
} from '@/spec/unified_diff';

import { aggregateAPI } from '@/apis/baseapi';

import { DiffApplyModal } from '@/components/markdown/diff_apply_modal';
import {
	collectOutputDiagnostics,
	collectPatchLevelDiagnostics,
	formatDiagnosticsTitle,
	getDiagnosticSeverityCounts,
	getDiagnosticToneFromCounts,
	getHighestDiagnosticLevel,
	type HeaderButtonTone,
	uniqueDiagnostics,
} from '@/components/markdown/diff_diagnostic';
import {
	buildEditableTargetsFromOutput,
	buildFileStatusCounts,
	buildUnifiedDiffTextForTarget,
	type DiffApplyRunOptions,
	type EditableUnifiedDiffTarget,
	type FileStatusCounts,
	getErrorMessage,
	haveSharedPathIdentity,
	looksLikeUnifiedDiff,
	parseUnifiedDiffForUI,
	summaryLabel,
	uniqueStrings,
} from '@/components/markdown/unified_diff_block';

function joinDiffTextParts(parts: string[]): string {
	return parts.map(part => part.replaceAll(/\n+$/g, '')).join('\n');
}

function isDiffPartHunkComplete(diffPart: string, expectedHunks: number | undefined): boolean {
	if (typeof expectedHunks !== 'number' || expectedHunks <= 0) return true;
	return diffPart.split('\n').filter(line => line.startsWith('@@')).length === expectedHunks;
}

function editableTargetsToFileTargets(targets: EditableUnifiedDiffTarget[]): ApplyUnifiedDiffFileTarget[] {
	return targets
		.map(target => ({
			fileKey: target.fileKey?.trim() || undefined,
			oldPath: target.oldPath?.trim() || undefined,
			newPath: target.newPath?.trim() || undefined,
			targetPath: target.targetPath.trim(),
		}))
		.filter(target => target.targetPath.length > 0);
}

function resolveRequestDiffText(
	fullDiffText: string,
	buildScopedDiffText: (targets: EditableUnifiedDiffTarget[]) => string | undefined,
	targets: EditableUnifiedDiffTarget[],
	options?: DiffApplyRunOptions
): string | undefined {
	if (options?.diffText?.trim()) return options.diffText;
	if (options?.mergeOutput) return buildScopedDiffText(targets);
	return fullDiffText;
}

function getApplicableTargetsForOutput(
	targets: EditableUnifiedDiffTarget[],
	output: ApplyUnifiedDiffOut | undefined
): EditableUnifiedDiffTarget[] {
	if (!output) return [];

	if (output.files && output.files.length > 0) {
		return targets.filter(target => target.ok !== false && target.status === ApplyUnifiedDiffStatus.Applicable);
	}

	if (output.ok && output.status === ApplyUnifiedDiffStatus.Applicable) {
		return targets;
	}

	return [];
}

function buildTitle(state: DiffApplyState, fallbackParsed: ReturnType<typeof parseUnifiedDiffForUI>): string {
	const output = state.output;
	const parts = [
		getButtonTitle(state.status),
		summaryLabel(output, fallbackParsed),
		state.error,
		state.message,
		output?.message,
		...collectOutputDiagnostics(output).slice(0, 8),
	].filter(Boolean);

	return uniqueStrings(parts).join('\n');
}

function mapOutputToControlStatus(output: ApplyUnifiedDiffOut): ControlStatus {
	if (output.ok) {
		switch (output.status) {
			case ApplyUnifiedDiffStatus.Applicable:
				return 'ready';
			case ApplyUnifiedDiffStatus.Applied:
				return 'applied';
			case ApplyUnifiedDiffStatus.AlreadyApplied:
				return 'already-applied';
			default:
				return 'ready';
		}
	}

	switch (output.status) {
		case ApplyUnifiedDiffStatus.NeedsInfo:
			return 'needs-info';

		default:
			return 'blocked';
	}
}

function getButtonLabel(status: ControlStatus): string {
	switch (status) {
		case 'checking':
			return 'Checking';
		case 'ready':
			return 'Apply';
		case 'needs-info':
			return 'Need info';
		case 'blocked':
			return 'Blocked';
		case 'applying':
			return 'Applying';
		case 'applied':
			return 'Applied';
		case 'already-applied':
			return 'Already applied';
		default:
			return 'Diff';
	}
}

function getButtonTitle(status: ControlStatus): string {
	switch (status) {
		case 'checking':
			return 'Checking unified diff';
		case 'ready':
			return 'Apply unified diff';
		case 'needs-info':
		case 'blocked':
		case 'applied':
		case 'already-applied':
			return 'Open diff details';
		case 'applying':
			return 'Applying unified diff';
		default:
			return 'Preparing unified diff';
	}
}

function getButtonIcon(status: ControlStatus) {
	switch (status) {
		case 'checking':
		case 'applying':
			return <FiLoader size={14} className="animate-spin" />;
		case 'ready':
			return <FiGitPullRequest size={14} />;
		case 'needs-info':
			return <FiInfo size={14} />;
		case 'blocked':
			return <FiAlertTriangle size={14} />;
		case 'applied':
		case 'already-applied':
			return <FiCheckCircle size={14} />;
		default:
			return <FiGitPullRequest size={14} />;
	}
}

function getControlButtonClassName(tone: HeaderButtonTone = 'neutral'): string {
	const toneClassName: Record<HeaderButtonTone, string> = {
		neutral: 'app-text-code',
		success: 'text-success',
		warning: 'text-warning',
		error: 'text-error',
		info: 'text-info',
	};

	return `inline-flex h-6 max-w-full items-center gap-1 overflow-hidden rounded-full border border-base-300 px-2 py-0 text-[11px] font-medium leading-none whitespace-nowrap shadow-none transition-colors hover:border-base-300 hover:opacity-50 disabled:cursor-not-allowed disabled:bg-base-100/40 disabled:opacity-50 ${toneClassName[tone]}`;
}

function formatCompactFileCount(count: number): string {
	return `${count}`;
}

function getHeaderDetailsTone(status: ControlStatus, diagnostics: ApplyUnifiedDiffDiagnostic[]): HeaderButtonTone {
	if (status === 'blocked') return 'error';
	if (status === 'needs-info') return 'warning';

	const highest = getHighestDiagnosticLevel(diagnostics);
	if (highest === ApplyUnifiedDiffDiagnosticLevel.Error) return 'error';
	if (highest === ApplyUnifiedDiffDiagnosticLevel.Warning) return 'warning';

	return 'neutral';
}

function getApplyPatchIdentityPaths(file: { newPath?: string; oldPath?: string }): string[] {
	return uniqueStrings([file.newPath, file.oldPath]).filter(path => path !== '/dev/null');
}

function getApplyResolvedIdentityPaths(file: { resolvedPath?: string; targetPath?: string }): string[] {
	return uniqueStrings([file.resolvedPath, file.targetPath]).filter(path => path !== '/dev/null');
}

function applyOutputFileMatchesTarget(file: ApplyUnifiedDiffFileOut, target: ApplyUnifiedDiffFileTarget): boolean {
	if (file.fileKey && target.fileKey && file.fileKey === target.fileKey) return true;

	const filePatchPaths = getApplyPatchIdentityPaths(file);
	const targetPatchPaths = getApplyPatchIdentityPaths(target);

	if (filePatchPaths.length > 0 && targetPatchPaths.length > 0) {
		return haveSharedPathIdentity(filePatchPaths, targetPatchPaths);
	}

	if (filePatchPaths.length === 0 && targetPatchPaths.length === 0) {
		return haveSharedPathIdentity(getApplyResolvedIdentityPaths(file), getApplyResolvedIdentityPaths(target));
	}

	return false;
}

function findRequestTargetForOutputFile(
	file: ApplyUnifiedDiffFileOut,
	targets: ApplyUnifiedDiffFileTarget[],
	usedTargetIndexes: Set<number>
): { target: ApplyUnifiedDiffFileTarget; index: number } | undefined {
	for (let index = 0; index < targets.length; index += 1) {
		if (usedTargetIndexes.has(index)) continue;

		const target = targets[index];
		if (!target) continue;

		if (applyOutputFileMatchesTarget(file, target)) {
			return { target, index };
		}
	}

	return undefined;
}

function hydrateApplyOutputWithRequestTargets(
	output: ApplyUnifiedDiffOut,
	requestTargets: ApplyUnifiedDiffFileTarget[]
): ApplyUnifiedDiffOut {
	if (!output.files?.length || requestTargets.length === 0) return output;

	const outputFiles = output.files;
	const usedTargetIndexes = new Set<number>();
	const canUsePositionFallback = outputFiles.length === requestTargets.length;

	const files = outputFiles.map((file, index) => {
		const directMatch = findRequestTargetForOutputFile(file, requestTargets, usedTargetIndexes);
		let target = directMatch?.target;

		if (directMatch) {
			usedTargetIndexes.add(directMatch.index);
		} else if (canUsePositionFallback && requestTargets[index] && !usedTargetIndexes.has(index)) {
			target = requestTargets[index];
			usedTargetIndexes.add(index);
		} else if (outputFiles.length === 1 && requestTargets.length === 1) {
			target = requestTargets[0];
		}

		if (!target) return file;

		return {
			...file,
			fileKey: target.fileKey || file.fileKey,
			oldPath: file.oldPath || target.oldPath,
			newPath: file.newPath || target.newPath,
			targetPath: target.targetPath || file.targetPath,
		};
	});

	return {
		...output,
		fileTargets: mergeApplyFileTargets(output.fileTargets ?? [], requestTargets),
		files,
	};
}

function mergeApplyUnifiedDiffOutput(
	previous: ApplyUnifiedDiffOut | undefined,
	scoped: ApplyUnifiedDiffOut
): ApplyUnifiedDiffOut {
	if (!previous?.files?.length || !scoped.files?.length) return scoped;

	const files = [...previous.files];

	for (const scopedFile of scoped.files) {
		const index = files.findIndex(file => applyOutputFilesMatch(file, scopedFile));

		if (index >= 0) {
			const previousFile = files[index];

			files[index] = {
				...previousFile,

				...scopedFile,
				diagnostics: uniqueDiagnostics([...(previousFile.diagnostics ?? []), ...(scopedFile.diagnostics ?? [])]),
			};
		} else {
			files.push(scopedFile);
		}
	}

	return {
		...previous,
		dryRun: scoped.dryRun,
		ok: files.every(file => file.ok),
		status: getAggregateStatusFromFiles(files, scoped.status),
		message: scoped.message || previous.message,
		diagnostics: uniqueDiagnostics([...(previous.diagnostics ?? []), ...(scoped.diagnostics ?? [])]),
		summary: summarizeApplyFiles(files),
		fileTargets: mergeApplyFileTargets(previous.fileTargets ?? [], scoped.fileTargets ?? []),
		files,
	};
}

function summarizeApplyFiles(files: ApplyUnifiedDiffFileOut[]): ApplyUnifiedDiffOut['summary'] {
	return {
		files: files.length,
		hunks: files.reduce((sum, file) => sum + file.hunks, 0),
		appliedHunks: files.reduce((sum, file) => sum + file.appliedHunks, 0),
		alreadyAppliedHunks: files.reduce((sum, file) => sum + file.alreadyAppliedHunks, 0),
		addedLines: files.reduce((sum, file) => sum + file.addedLines, 0),
		deletedLines: files.reduce((sum, file) => sum + file.deletedLines, 0),
	};
}

function getAggregateStatusFromFiles(
	files: ApplyUnifiedDiffFileOut[],
	fallbackStatus: ApplyUnifiedDiffStatus
): ApplyUnifiedDiffStatus {
	if (files.some(file => file.status === ApplyUnifiedDiffStatus.Error)) return ApplyUnifiedDiffStatus.Error;
	if (files.some(file => file.status === ApplyUnifiedDiffStatus.Conflict)) return ApplyUnifiedDiffStatus.Conflict;
	if (files.some(file => file.status === ApplyUnifiedDiffStatus.NeedsInfo)) return ApplyUnifiedDiffStatus.NeedsInfo;
	if (files.some(file => file.status === ApplyUnifiedDiffStatus.Applicable)) return ApplyUnifiedDiffStatus.Applicable;
	if (files.every(file => file.status === ApplyUnifiedDiffStatus.AlreadyApplied))
		return ApplyUnifiedDiffStatus.AlreadyApplied;
	if (files.every(file => file.status === ApplyUnifiedDiffStatus.Applied)) return ApplyUnifiedDiffStatus.Applied;
	return fallbackStatus;
}

function mergeApplyFileTargets(
	existingTargets: ApplyUnifiedDiffFileTarget[],
	nextTargets: ApplyUnifiedDiffFileTarget[]
): ApplyUnifiedDiffFileTarget[] {
	if (nextTargets.length === 0) return existingTargets;

	const merged = [...existingTargets];

	for (const target of nextTargets) {
		const index = merged.findIndex(existing => fileTargetsMatch(existing, target));

		if (index >= 0) {
			merged[index] = {
				...merged[index],
				...target,
				targetPath: target.targetPath || merged[index].targetPath,
			};
		} else {
			merged.push(target);
		}
	}

	return merged;
}

function applyOutputFilesMatch(left: ApplyUnifiedDiffFileOut, right: ApplyUnifiedDiffFileOut): boolean {
	if (left.fileKey && right.fileKey && left.fileKey === right.fileKey) return true;

	const leftPatchPaths = getApplyPatchIdentityPaths(left);
	const rightPatchPaths = getApplyPatchIdentityPaths(right);

	if (leftPatchPaths.length > 0 && rightPatchPaths.length > 0) {
		return haveSharedPathIdentity(leftPatchPaths, rightPatchPaths);
	}

	if (leftPatchPaths.length === 0 && rightPatchPaths.length === 0) {
		return haveSharedPathIdentity(getApplyResolvedIdentityPaths(left), getApplyResolvedIdentityPaths(right));
	}

	return false;
}

function fileTargetsMatch(left: ApplyUnifiedDiffFileTarget, right: ApplyUnifiedDiffFileTarget): boolean {
	if (left.fileKey && right.fileKey && left.fileKey === right.fileKey) return true;
	return haveSharedPathIdentity([left.newPath, left.oldPath], [right.newPath, right.oldPath]);
}

function getHeaderStatusSummary(
	counts: FileStatusCounts,
	output: ApplyUnifiedDiffOut | undefined,
	fallbackParsed: ReturnType<typeof parseUnifiedDiffForUI>
): string {
	const parts = [summaryLabel(output, fallbackParsed)];
	const blockedCount = Math.max(0, counts.blocked - counts.needsInfo);

	if (!output) {
		if (counts.unknown > 0) parts.push(`${counts.unknown} pending`);
		return parts.join(' · ');
	}

	if (counts.applicable > 0) parts.push(`${counts.applicable} applicable`);
	if (counts.applied > 0) parts.push(`${counts.applied} applied`);
	if (counts.alreadyApplied > 0) parts.push(`${counts.alreadyApplied} already applied`);
	if (counts.needsInfo > 0) parts.push(`${counts.needsInfo} need info`);
	if (blockedCount > 0) parts.push(`${blockedCount} blocked`);
	if (counts.notApplicable > 0 && blockedCount === 0) parts.push(`${counts.notApplicable} not applicable`);
	if (counts.unknown > 0) parts.push(`${counts.unknown} pending`);

	return parts.join(' · ');
}

interface DiffApplyControlProps {
	language: string;
	diffText: string;
	isBusy: boolean;
	candidatePaths?: string[];
}

type ControlStatus =
	| 'idle'
	| 'checking'
	| 'ready'
	| 'needs-info'
	| 'blocked'
	| 'applying'
	| 'applied'
	| 'already-applied';

interface DiffApplyState {
	status: ControlStatus;
	message?: string;
	output?: ApplyUnifiedDiffOut;
	error?: string;
}

export function DiffApplyControl({ language, diffText, isBusy, candidatePaths }: DiffApplyControlProps) {
	const isDiffLike = useMemo(() => looksLikeUnifiedDiff(diffText, language), [diffText, language]);
	const fallbackParsed = useMemo(() => parseUnifiedDiffForUI(diffText, language), [diffText, language]);
	const normalizedCandidatePaths = useMemo(() => uniqueStrings(candidatePaths ?? []), [candidatePaths]);

	const [state, setState] = useState<DiffApplyState>({ status: 'idle' });
	const [fileTargets, setFileTargets] = useState<ApplyUnifiedDiffFileTarget[]>([]);
	const [strict, setStrict] = useState(false);
	const [isDetailsOpen, setIsDetailsOpen] = useState(false);

	const stateRef = useRef<DiffApplyState>(state);
	const requestSeqRef = useRef(0);

	const setControlState = useCallback((next: DiffApplyState | ((previous: DiffApplyState) => DiffApplyState)) => {
		setState(previous => {
			const value = typeof next === 'function' ? next(previous) : next;
			stateRef.current = value;
			return value;
		});
	}, []);

	const blockKey = useMemo(
		() => `${language}\u0000${diffText}\u0000${normalizedCandidatePaths.join('\u0000')}`,
		[diffText, language, normalizedCandidatePaths]
	);

	useEffect(() => {
		stateRef.current = state;
	}, [state]);

	useEffect(() => {
		return () => {
			requestSeqRef.current += 1;
		};
	}, []);

	const deriveAndStoreTargets = useCallback(
		(output: ApplyUnifiedDiffOut | undefined, previousTargets?: ApplyUnifiedDiffFileTarget[]) => {
			const editable = buildEditableTargetsFromOutput(output, fallbackParsed, normalizedCandidatePaths);
			const next = editableTargetsToFileTargets(editable);
			const nextWithPrevious =
				previousTargets && previousTargets.length > 0 ? mergeApplyFileTargets(next, previousTargets) : next;

			if (nextWithPrevious.length > 0) {
				setFileTargets(nextWithPrevious);
				return nextWithPrevious;
			}

			if (output?.fileTargets && output.fileTargets.length > 0) {
				const outputTargets =
					previousTargets && previousTargets.length > 0
						? mergeApplyFileTargets(output.fileTargets, previousTargets)
						: output.fileTargets;
				setFileTargets(outputTargets);
				return outputTargets;
			}

			if (previousTargets) {
				setFileTargets(previousTargets);
				return previousTargets;
			}

			return [];
		},
		[fallbackParsed, normalizedCandidatePaths]
	);

	const buildDiffTextForEditableTargets = useCallback(
		(targets: EditableUnifiedDiffTarget[]): string | undefined => {
			if (targets.length === 0) return diffText;

			const parts: string[] = [];
			const seen = new Set<string>();

			for (const target of targets) {
				const split = buildUnifiedDiffTextForTarget(diffText, language, target);
				const part = split?.diffText ?? target.diffText;

				if (!part?.trim()) {
					if (targets.length === 1 && fallbackParsed.files.length <= 1) {
						return diffText;
					}

					return undefined;
				}

				if (split && !split.verified) return undefined;
				if (!isDiffPartHunkComplete(part, target.hunks)) return undefined;

				const key = part.replaceAll(/\n+$/g, '');
				if (seen.has(key)) continue;

				seen.add(key);
				parts.push(part);
			}

			if (parts.length === 0) return targets.length <= 1 ? diffText : undefined;
			return joinDiffTextParts(parts);
		},
		[diffText, fallbackParsed.files.length, language]
	);

	const runDryRun = useCallback(
		async (targets: ApplyUnifiedDiffFileTarget[], nextStrict: boolean, options?: DiffApplyRunOptions) => {
			const seq = ++requestSeqRef.current;
			const requestDiffText = options?.diffText ?? diffText;

			setControlState(previous => ({
				...previous,
				status: 'checking',
				message: 'Checking whether this diff can be applied.',
				error: undefined,
			}));

			try {
				const rawOutput = await aggregateAPI.applyUnifiedDiff({
					diffText: requestDiffText,
					dryRun: true,
					strict: nextStrict,
					fileTargets: targets.length > 0 ? targets : undefined,
					candidatePaths: normalizedCandidatePaths.length > 0 ? normalizedCandidatePaths : undefined,
				});

				if (seq !== requestSeqRef.current) return undefined;

				const output = hydrateApplyOutputWithRequestTargets(rawOutput, targets);
				const viewOutput = options?.mergeOutput ? mergeApplyUnifiedDiffOutput(stateRef.current.output, output) : output;

				deriveAndStoreTargets(viewOutput, targets);

				setControlState({
					status: mapOutputToControlStatus(viewOutput),
					message: viewOutput.message,
					output: viewOutput,
				});

				return output;
			} catch (error) {
				if (seq !== requestSeqRef.current) return undefined;

				setControlState(previous => ({
					...previous,
					status: 'blocked',
					error: getErrorMessage(error),
				}));
				return undefined;
			}
		},
		[deriveAndStoreTargets, diffText, normalizedCandidatePaths, setControlState]
	);

	useEffect(() => {
		requestSeqRef.current += 1;

		if (!isDiffLike || isBusy) {
			setControlState({ status: 'idle' });
			// eslint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
			setFileTargets([]);
			return;
		}
		// eslint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setStrict(false);
		// eslint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setFileTargets([]);

		void runDryRun([], false);
	}, [blockKey, isBusy, isDiffLike, runDryRun, setControlState]);

	const runApply = async (
		targets: ApplyUnifiedDiffFileTarget[],
		nextStrict: boolean,
		options?: DiffApplyRunOptions
	) => {
		const requestDiffText = options?.diffText ?? diffText;

		setControlState(previous => ({
			...previous,
			status: 'checking',
			message: 'Rechecking diff before applying.',
			error: undefined,
		}));

		const dryRunOutput = await runDryRun(targets, nextStrict, {
			...options,
			diffText: requestDiffText,
		});
		if (!dryRunOutput) return;

		const dryRunViewOutput = options?.mergeOutput ? (stateRef.current.output ?? dryRunOutput) : dryRunOutput;

		if (!dryRunOutput.ok) {
			setControlState({
				status: mapOutputToControlStatus(dryRunViewOutput),
				message: dryRunOutput.message,
				output: dryRunViewOutput,
			});
			return;
		}

		if (dryRunOutput.status === ApplyUnifiedDiffStatus.AlreadyApplied) {
			setControlState({
				status: mapOutputToControlStatus(dryRunViewOutput),
				message: dryRunOutput.message || 'Unified diff is already applied.',
				output: dryRunViewOutput,
			});
			return;
		}

		const applyTargets = deriveAndStoreTargets(dryRunOutput, targets);

		setControlState(previous => ({
			...previous,
			status: 'applying',
			message: 'Applying diff.',
			output: options?.mergeOutput ? previous.output : dryRunViewOutput,
			error: undefined,
		}));
		const applySeq = ++requestSeqRef.current;

		try {
			const rawOutput = await aggregateAPI.applyUnifiedDiff({
				diffText: requestDiffText,
				dryRun: false,
				strict: nextStrict,
				fileTargets: applyTargets.length > 0 ? applyTargets : undefined,
				candidatePaths: normalizedCandidatePaths.length > 0 ? normalizedCandidatePaths : undefined,
			});
			if (applySeq !== requestSeqRef.current) return;

			const output = hydrateApplyOutputWithRequestTargets(rawOutput, applyTargets);

			const viewOutput = options?.mergeOutput ? mergeApplyUnifiedDiffOutput(stateRef.current.output, output) : output;

			deriveAndStoreTargets(viewOutput, applyTargets);

			setControlState({
				status: mapOutputToControlStatus(viewOutput),
				message: viewOutput.message,
				output: viewOutput,
			});
		} catch (error) {
			if (applySeq !== requestSeqRef.current) return;

			setControlState(previous => ({
				...previous,
				status: 'blocked',
				error: getErrorMessage(error),
			}));
		}
	};

	if (!isDiffLike || isBusy) return null;

	const title = buildTitle(state, fallbackParsed);
	const buttonTitle = getButtonTitle(state.status);
	const label = getButtonLabel(state.status);
	const icon = getButtonIcon(state.status);
	const isRequestBusy = state.status === 'checking' || state.status === 'applying';
	const hasDryRunResult = !!state.output;

	const statusCounts = buildFileStatusCounts(state.output, fallbackParsed);
	const patchDiagnostics = uniqueDiagnostics([
		...(fallbackParsed.diagnostics ?? []),
		...collectPatchLevelDiagnostics(state.output),
	]);
	const patchDiagnosticCounts = getDiagnosticSeverityCounts(patchDiagnostics);
	const headerStatusSummary = getHeaderStatusSummary(statusCounts, state.output, fallbackParsed);
	const detailTitle = uniqueStrings([headerStatusSummary, title, formatDiagnosticsTitle(patchDiagnostics)]).join('\n');
	const blockedCount = Math.max(0, statusCounts.blocked - statusCounts.needsInfo);

	const editableTargetsForHeader = buildEditableTargetsFromOutput(
		state.output,
		fallbackParsed,
		normalizedCandidatePaths
	);
	const applicableTargets = getApplicableTargetsForOutput(editableTargetsForHeader, state.output);
	const applicableTargetsHaveMissingPaths =
		applicableTargets.length > 0 && editableTargetsToFileTargets(applicableTargets).length !== applicableTargets.length;
	const canHeaderApply = applicableTargets.length > 0 && !isRequestBusy && !applicableTargetsHaveMissingPaths;

	const handleHeaderApply = () => {
		if (!canHeaderApply) {
			setIsDetailsOpen(true);
			return;
		}

		const requestDiffText = buildDiffTextForEditableTargets(applicableTargets);
		if (!requestDiffText) {
			setControlState(previous => ({
				...previous,
				status: 'blocked',
				error: 'Could not isolate the applicable file patches. Open details and retry per file.',
			}));
			setIsDetailsOpen(true);
			return;
		}

		const normalizedTargets = editableTargetsToFileTargets(applicableTargets);
		setFileTargets(previous => mergeApplyFileTargets(previous, normalizedTargets));

		void runApply(normalizedTargets, strict, {
			diffText: requestDiffText,
			mergeOutput: true,
		});
	};

	return (
		<>
			<div className="app-text-code flex max-w-full min-w-0 items-center gap-1 overflow-hidden">
				<button
					type="button"
					className={getControlButtonClassName()}
					onClick={() => {
						setIsDetailsOpen(true);
					}}
					title={detailTitle || title}
					aria-label="Open diff details"
				>
					<FiInfo size={12} className="mb-0.5 shrink-0" />
					<span className="leading-none">Files</span>
					{statusCounts.total > 0 ? (
						<span className="leading-none">{formatCompactFileCount(statusCounts.total)}</span>
					) : null}
				</button>

				{isRequestBusy ? (
					<button type="button" className={getControlButtonClassName('info')} disabled title={title}>
						<div className="mb-0.5 shrink-0">{icon}</div>
						<span>{label}</span>
					</button>
				) : hasDryRunResult ? (
					<>
						{statusCounts.applicable > 0 ? (
							<button
								type="button"
								className={getControlButtonClassName('success')}
								disabled={!canHeaderApply}
								onClick={handleHeaderApply}
								title={`${statusCounts.applicable} applicable file patch${statusCounts.applicable === 1 ? '' : 'es'}`}
							>
								<FiGitPullRequest size={12} className="mb-0.5 shrink-0" />
								Apply {statusCounts.applicable}
							</button>
						) : null}

						{statusCounts.needsInfo > 0 ? (
							<button
								type="button"
								className={getControlButtonClassName('warning')}
								onClick={() => {
									setIsDetailsOpen(true);
								}}
								title={`${statusCounts.needsInfo} file patch${statusCounts.needsInfo === 1 ? '' : 'es'} need more info`}
							>
								<FiInfo size={12} className="mb-0.5 shrink-0" />
								Need info {statusCounts.needsInfo}
							</button>
						) : null}

						{blockedCount > 0 ? (
							<button
								type="button"
								className={getControlButtonClassName('error')}
								onClick={() => {
									setIsDetailsOpen(true);
								}}
								title={`${blockedCount} blocked file patch${blockedCount === 1 ? '' : 'es'}`}
							>
								<FiAlertTriangle size={12} className="mb-0.5 shrink-0" />
								Blocked {blockedCount}
							</button>
						) : null}

						{patchDiagnosticCounts.total > 0 ? (
							<button
								type="button"
								className={getControlButtonClassName(getDiagnosticToneFromCounts(patchDiagnosticCounts))}
								onClick={() => {
									setIsDetailsOpen(true);
								}}
								title={formatDiagnosticsTitle(patchDiagnostics)}
							>
								{patchDiagnosticCounts.error > 0 ? (
									<FiAlertTriangle size={12} className="mb-0.5 shrink-0" />
								) : (
									<FiInfo size={12} className="mb-0.5 shrink-0" />
								)}
								Diag {patchDiagnosticCounts.total}
							</button>
						) : null}

						{statusCounts.applied > 0 ? (
							<button
								type="button"
								className={`${getControlButtonClassName('success')} hidden sm:inline-flex`}
								onClick={() => {
									setIsDetailsOpen(true);
								}}
								title={`${statusCounts.applied} applied file patch${statusCounts.applied === 1 ? '' : 'es'}`}
							>
								<FiCheckCircle size={12} className="mb-0.5 shrink-0" />
								Applied {statusCounts.applied}
							</button>
						) : null}

						{statusCounts.alreadyApplied > 0 ? (
							<button
								type="button"
								className={`${getControlButtonClassName('info')} hidden sm:inline-flex`}
								onClick={() => {
									setIsDetailsOpen(true);
								}}
								title={`${statusCounts.alreadyApplied} already applied file patch${
									statusCounts.alreadyApplied === 1 ? '' : 'es'
								}`}
							>
								<FiCheckCircle size={12} className="mb-0.5 shrink-0" />
								Already {statusCounts.alreadyApplied}
							</button>
						) : null}

						{statusCounts.unknown > 0 ? (
							<button
								type="button"
								className={`${getControlButtonClassName()} hidden sm:inline-flex`}
								onClick={() => {
									setIsDetailsOpen(true);
								}}
								title={`${statusCounts.unknown} file patch${statusCounts.unknown === 1 ? '' : 'es'} pending`}
							>
								Pending {statusCounts.unknown}
							</button>
						) : null}
					</>
				) : (
					<button
						type="button"
						className={getControlButtonClassName(getHeaderDetailsTone(state.status, patchDiagnostics))}
						disabled={state.status === 'idle'}
						onClick={() => {
							setIsDetailsOpen(true);
						}}
						title={`${buttonTitle}\n${detailTitle || title}`}
					>
						<div className="mb-0.5 shrink-0">{icon}</div>
						<span>{label}</span>
					</button>
				)}
			</div>

			<DiffApplyModal
				isOpen={isDetailsOpen}
				onClose={() => {
					setIsDetailsOpen(false);
				}}
				fallbackParsed={fallbackParsed}
				output={state.output}
				error={state.error}
				candidatePaths={normalizedCandidatePaths}
				fileTargets={fileTargets}
				strict={strict}
				onStrictChange={setStrict}
				onDryRun={async (nextTargets, nextStrict, options) => {
					const normalized = editableTargetsToFileTargets(nextTargets);
					setFileTargets(previous => (options?.mergeOutput ? mergeApplyFileTargets(previous, normalized) : normalized));

					const requestDiffText = resolveRequestDiffText(
						diffText,
						buildDiffTextForEditableTargets,
						nextTargets,
						options
					);

					if (!requestDiffText) {
						setControlState(previous => ({
							...previous,
							status: 'blocked',
							error: 'Could not isolate this file patch from the unified diff. The full diff was not sent.',
						}));
						return;
					}

					await runDryRun(normalized, nextStrict, { ...options, diffText: requestDiffText });
				}}
				onApply={async (nextTargets, nextStrict, options) => {
					const normalized = editableTargetsToFileTargets(nextTargets);
					setFileTargets(previous => (options?.mergeOutput ? mergeApplyFileTargets(previous, normalized) : normalized));

					const requestDiffText = resolveRequestDiffText(
						diffText,
						buildDiffTextForEditableTargets,
						nextTargets,
						options
					);

					if (!requestDiffText) {
						setControlState(previous => ({
							...previous,
							status: 'blocked',
							error: 'Could not isolate this file patch from the unified diff. The full diff was not sent.',
						}));
						return;
					}

					await runApply(normalized, nextStrict, { ...options, diffText: requestDiffText });
				}}
			/>
		</>
	);
}
