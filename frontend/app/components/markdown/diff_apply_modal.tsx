import { useEffect, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiChevronRight, FiGitPullRequest, FiX } from 'react-icons/fi';

import type { ApplyUnifiedDiffDiagnostic, ApplyUnifiedDiffFileTarget, ApplyUnifiedDiffOut } from '@/spec/unified_diff';
import { ApplyUnifiedDiffDiagnosticLevel, ApplyUnifiedDiffStatus } from '@/spec/unified_diff';

import type { DiagnosticSeverityCounts, HeaderButtonTone } from '@/components/markdown/diff_diagnostic';
import {
	collectFileLevelDiagnostics,
	collectPatchLevelDiagnostics,
	formatDiagnosticsTitle,
	getDiagnosticSeverityCounts,
	getDiagnosticToneFromCounts,
	renderDiagnosticSeveritySummary,
	renderDiagnosticsPanel,
	uniqueDiagnostics,
} from '@/components/markdown/diff_diagnostic';
import type {
	DiffApplyRunOptions,
	EditableUnifiedDiffTarget,
	parseUnifiedDiffForUI,
} from '@/components/markdown/unified_diff_block';
import {
	buildEditableTargetsFromOutput,
	buildFileStatusCounts,
	getPathIdentity,
	haveSharedPathIdentity,
	mergeNumberMax,
	summaryLabel,
	uniqueStrings,
} from '@/components/markdown/unified_diff_block';
import { ModalBackdrop } from '@/components/modal_backdrop';

interface ModalRunningAction {
	key: string;
	kind: 'dry-run' | 'apply';
}
type TargetVisualState = 'neutral' | 'info' | 'success' | 'warning' | 'error';

function getTargetCardClassName(visualState: TargetVisualState): string {
	switch (visualState) {
		default:
			return 'border-base-300 bg-base-100';
	}
}

function getTargetStatusBadgeClassName(visualState: TargetVisualState): string {
	switch (visualState) {
		case 'error':
			return 'badge-error';
		case 'warning':
			return 'badge-warning';
		case 'success':
			return 'badge-success';
		case 'info':
			return 'badge-info';

		default:
			return 'badge-ghost';
	}
}

function getTargetStatusLabel(target: EditableUnifiedDiffTarget, missing: boolean): string {
	if (missing) {
		return 'needs path';
	}
	if (target.status) {
		return target.status.replaceAll('_', ' ');
	}
	if (target.ok === false) {
		return 'blocked';
	}
	return 'pending';
}

function getTargetDisplayPath(target: EditableUnifiedDiffTarget): string {
	return (
		target.targetPath.trim() ||
		target.resolvedPath ||
		target.newPath ||
		target.oldPath ||
		target.fileKey ||
		'Unresolved file'
	);
}
function getBadgeToneClassName(tone: HeaderButtonTone): string {
	switch (tone) {
		case 'success':
			return 'badge-success';
		case 'warning':
			return 'badge-warning';
		case 'error':
			return 'badge-error';
		case 'info':
			return 'badge-info';

		default:
			return 'badge-ghost';
	}
}

function getDiagnosticSummaryBadgeClassName(counts: DiagnosticSeverityCounts): string {
	return getBadgeToneClassName(getDiagnosticToneFromCounts(counts));
}

function getModalTargetSectionKeys(target: EditableUnifiedDiffTarget): string[] {
	return uniqueStrings([target.fileKey, ...(target.sectionKeys ?? [])]);
}

function getModalTargetPatchPaths(target: EditableUnifiedDiffTarget): string[] {
	return uniqueStrings([target.newPath, target.oldPath]).filter(path => path !== '/dev/null');
}

function getModalTargetResolvedPaths(target: EditableUnifiedDiffTarget): string[] {
	return uniqueStrings([target.resolvedPath, target.targetPath]).filter(path => path !== '/dev/null');
}

function editableTargetsMatch(left: EditableUnifiedDiffTarget, right: EditableUnifiedDiffTarget): boolean {
	const leftKeys = getModalTargetSectionKeys(left);
	const rightKeys = getModalTargetSectionKeys(right);

	if (leftKeys.some(key => rightKeys.includes(key))) {
		return true;
	}

	const leftPatchPaths = getModalTargetPatchPaths(left);
	const rightPatchPaths = getModalTargetPatchPaths(right);

	if (leftPatchPaths.length > 0 && rightPatchPaths.length > 0) {
		return haveSharedPathIdentity(leftPatchPaths, rightPatchPaths);
	}

	if (leftPatchPaths.length === 0 && rightPatchPaths.length === 0) {
		return haveSharedPathIdentity(getModalTargetResolvedPaths(left), getModalTargetResolvedPaths(right));
	}

	return false;
}

function mergeEditableTargetForModal(
	existing: EditableUnifiedDiffTarget | undefined,
	target: EditableUnifiedDiffTarget
): EditableUnifiedDiffTarget {
	if (!existing) {
		return {
			...target,
			targetPath: target.targetPath || '',
			candidatePaths: uniqueStrings([
				...(target.candidatePaths ?? []),
				target.targetPath,
				target.resolvedPath,
				target.newPath,
				target.oldPath,
			]),
			sectionKeys: uniqueStrings([target.fileKey, ...(target.sectionKeys ?? [])]),
		};
	}

	return {
		...existing,
		...target,
		fileKey: existing.fileKey || target.fileKey,
		oldPath: target.oldPath || existing.oldPath,
		newPath: target.newPath || existing.newPath,
		targetPath: target.targetPath || existing.targetPath || '',
		resolvedPath: target.resolvedPath || existing.resolvedPath,
		candidatePaths: uniqueStrings([
			...(existing.candidatePaths ?? []),
			...(target.candidatePaths ?? []),
			existing.targetPath,
			target.targetPath,
			existing.resolvedPath,
			target.resolvedPath,
			existing.newPath,
			target.newPath,
			existing.oldPath,
			target.oldPath,
		]),
		diffText: target.diffText || existing.diffText,
		sectionKeys: uniqueStrings([
			...(existing.sectionKeys ?? []),
			existing.fileKey,
			target.fileKey,
			...(target.sectionKeys ?? []),
		]),
		ok: target.ok ?? existing.ok,
		status: target.status ?? existing.status,
		message: target.message ?? existing.message,
		diagnostics: uniqueDiagnostics([...(existing.diagnostics ?? []), ...(target.diagnostics ?? [])]),
		hunks: mergeNumberMax(existing.hunks, target.hunks),
		appliedHunks: mergeNumberMax(existing.appliedHunks, target.appliedHunks),
		alreadyAppliedHunks: mergeNumberMax(existing.alreadyAppliedHunks, target.alreadyAppliedHunks),
		addedLines: mergeNumberMax(existing.addedLines, target.addedLines),
		deletedLines: mergeNumberMax(existing.deletedLines, target.deletedLines),
	};
}

function upsertEditableTargetForModal(
	byKey: Map<string, EditableUnifiedDiffTarget>,
	target: EditableUnifiedDiffTarget
) {
	const existingEntry = [...byKey.entries()].find(([, existing]) => editableTargetsMatch(existing, target));
	const key = existingEntry?.[0] ?? getLocalTargetKey(target, byKey.size);
	const existing = existingEntry?.[1];

	byKey.set(key, mergeEditableTargetForModal(existing, target));
}

function mergeEditableTargetPreservingLocalPath(
	base: EditableUnifiedDiffTarget,
	local: EditableUnifiedDiffTarget
): EditableUnifiedDiffTarget {
	const merged = mergeEditableTargetForModal(local, base);
	const localTargetPath = local.targetPath.trim();

	if (!localTargetPath) {
		return merged;
	}

	return {
		...merged,
		targetPath: local.targetPath,
		candidatePaths: uniqueStrings([
			local.targetPath,
			...(local.candidatePaths ?? []),
			...(merged.candidatePaths ?? []),
		]),
	};
}

function mergeModalTargetsPreservingLocalEdits(
	nextTargets: EditableUnifiedDiffTarget[],
	previousTargets: EditableUnifiedDiffTarget[]
): EditableUnifiedDiffTarget[] {
	if (previousTargets.length === 0) {
		return nextTargets;
	}

	const matchedPreviousIndexes = new Set<number>();

	return nextTargets.map(next => {
		const previousIndex = previousTargets.findIndex(
			(previous, index) => !matchedPreviousIndexes.has(index) && editableTargetsMatch(previous, next)
		);

		if (previousIndex < 0) {
			return next;
		}

		const previous = previousTargets[previousIndex];
		if (!previous) {
			return next;
		}

		matchedPreviousIndexes.add(previousIndex);
		return mergeEditableTargetPreservingLocalPath(next, previous);
	});
}

function isTargetProblem(target: EditableUnifiedDiffTarget, missing: boolean): boolean {
	return (
		missing ||
		target.ok === false ||
		target.status === ApplyUnifiedDiffStatus.NeedsInfo ||
		target.status === ApplyUnifiedDiffStatus.Conflict ||
		target.status === ApplyUnifiedDiffStatus.Error
	);
}
function getLocalTargetKey(target: EditableUnifiedDiffTarget, index: number): string {
	return (
		target.fileKey?.trim() ||
		target.sectionKeys?.join('|') ||
		getPathIdentity(target.resolvedPath) ||
		getPathIdentity(target.newPath) ||
		getPathIdentity(target.oldPath) ||
		`target-${index}`
	);
}

function renderStatusCountBadge(label: string, count: number, tone: HeaderButtonTone) {
	if (count <= 0) {
		return null;
	}

	return (
		<span className={`badge badge-outline badge-sm ${getBadgeToneClassName(tone)}`}>
			{count} {label}
		</span>
	);
}

function renderPathMeta(label: string, path: string) {
	return (
		<div className="flex min-w-0 items-center gap-2">
			<span className="text-base-content/40 w-8 shrink-0 uppercase">{label}</span>
			<span className="min-w-0 truncate font-mono" title={path}>
				{path}
			</span>
		</div>
	);
}

function getTargetVisualState(
	target: EditableUnifiedDiffTarget,
	missing: boolean,
	diagnostics: ApplyUnifiedDiffDiagnostic[] = []
): TargetVisualState {
	if (
		target.status === ApplyUnifiedDiffStatus.Conflict ||
		target.status === ApplyUnifiedDiffStatus.Error ||
		target.ok === false ||
		diagnostics.some(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Error)
	) {
		return 'error';
	}

	if (
		missing ||
		target.status === ApplyUnifiedDiffStatus.NeedsInfo ||
		diagnostics.some(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Warning)
	) {
		return 'warning';
	}

	if (target.status === ApplyUnifiedDiffStatus.Applicable) {
		return 'success';
	}
	if (target.status === ApplyUnifiedDiffStatus.Applied || target.status === ApplyUnifiedDiffStatus.AlreadyApplied) {
		return 'info';
	}
	if (diagnostics.some(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Info)) {
		return 'info';
	}
	return 'neutral';
}

function getTargetMessageClassName(visualState: TargetVisualState): string {
	switch (visualState) {
		default:
			return 'border border-base-300 bg-base-200/60 text-base-content/75';
	}
}

interface DiffApplyModalProps {
	isOpen: boolean;
	onClose: () => void;
	fallbackParsed: ReturnType<typeof parseUnifiedDiffForUI>;
	output?: ApplyUnifiedDiffOut;
	error?: string;
	candidatePaths: string[];
	fileTargets: ApplyUnifiedDiffFileTarget[];
	strict: boolean;
	onStrictChange: (strict: boolean) => void;
	onDryRun: (targets: EditableUnifiedDiffTarget[], strict: boolean, options?: DiffApplyRunOptions) => Promise<void>;
	onApply: (targets: EditableUnifiedDiffTarget[], strict: boolean, options?: DiffApplyRunOptions) => Promise<void>;
}

export function DiffApplyModal({
	isOpen,
	onClose,
	fallbackParsed,
	output,
	error,
	candidatePaths,
	fileTargets,
	strict,
	onStrictChange,
	onDryRun,
	onApply,
}: DiffApplyModalProps) {
	const dialogRef = useRef<HTMLDialogElement | null>(null);

	const [localTargets, setLocalTargets] = useState<EditableUnifiedDiffTarget[]>([]);
	const displayTargets = localTargets;

	const [localStrict, setLocalStrict] = useState(strict);
	const [runningAction, setRunningAction] = useState<ModalRunningAction | null>(null);
	const isRunning = runningAction !== null;
	const patchDiagnostics = uniqueDiagnostics([
		...(fallbackParsed.diagnostics ?? []),
		...collectPatchLevelDiagnostics(output),
	]);

	useEffect(() => {
		if (!isOpen) {
			return;
		}

		const fromOutput = buildEditableTargetsFromOutput(output, fallbackParsed, candidatePaths);
		const byKey = new Map<string, EditableUnifiedDiffTarget>();

		for (const target of fromOutput) {
			upsertEditableTargetForModal(byKey, target);
		}

		for (const target of fileTargets) {
			upsertEditableTargetForModal(byKey, {
				fileKey: target.fileKey,
				oldPath: target.oldPath,
				newPath: target.newPath,
				targetPath: target.targetPath,
				candidatePaths: uniqueStrings([target.targetPath, target.newPath, target.oldPath]),
			});
		}

		// eslint-disable-next-line react-hooks/set-state-in-effect, react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setLocalTargets(previous => mergeModalTargetsPreservingLocalEdits([...byKey.values()], previous));
		// eslint-disable-next-line react-you-might-not-need-an-effect/no-derived-state
		setLocalStrict(strict);
	}, [candidatePaths, fallbackParsed, fileTargets, isOpen, output, strict]);

	useEffect(() => {
		if (!isOpen) {
			return;
		}

		const dialog = dialogRef.current;
		if (!dialog) {
			return;
		}

		if (!dialog.open) {
			dialog.showModal();
		}

		return () => {
			if (dialog.open) {
				dialog.close();
			}
		};
	}, [isOpen]);

	if (!isOpen || typeof document === 'undefined') {
		return null;
	}

	const fileDiagnostics = collectFileLevelDiagnostics(output);
	const summary = summaryLabel(output, fallbackParsed);
	const counts = buildFileStatusCounts(output, fallbackParsed);
	const missingCount = displayTargets.filter(target => !target.targetPath.trim()).length;
	const hasAnyTargets = displayTargets.length > 0;
	const canApplyFromModal = hasAnyTargets && missingCount === 0 && !isRunning;
	const patchDiagnosticCounts = getDiagnosticSeverityCounts(patchDiagnostics);
	const blockedFileCount = Math.max(0, counts.blocked - counts.needsInfo);

	const updateTarget = (index: number, targetPath: string) => {
		setLocalTargets(prev =>
			prev.map((target, currentIndex) =>
				currentIndex === index
					? {
							...target,
							targetPath,
							candidatePaths: uniqueStrings([targetPath, ...(target.candidatePaths ?? [])]),
						}
					: target
			)
		);
	};

	const handleDryRun = async () => {
		if (isRunning) {
			return;
		}
		setRunningAction({ key: 'global', kind: 'dry-run' });
		try {
			onStrictChange(localStrict);
			await onDryRun(localTargets, localStrict);
		} finally {
			setRunningAction(null);
		}
	};

	const handleApply = async () => {
		if (isRunning || !canApplyFromModal) {
			return;
		}

		setRunningAction({ key: 'global', kind: 'apply' });
		try {
			onStrictChange(localStrict);
			await onApply(localTargets, localStrict);
		} finally {
			setRunningAction(null);
		}
	};

	const handleTargetDryRun = async (index: number) => {
		if (isRunning) {
			return;
		}

		const target = displayTargets[index];

		if (!target) {
			return;
		}

		const key = getLocalTargetKey(target, index);
		setRunningAction({ key, kind: 'dry-run' });

		try {
			onStrictChange(localStrict);
			await onDryRun([target], localStrict, {
				mergeOutput: true,
			});
		} finally {
			setRunningAction(null);
		}
	};

	const handleTargetApply = async (index: number) => {
		if (isRunning) {
			return;
		}

		const target = displayTargets[index];
		if (!target?.targetPath.trim()) {
			return;
		}

		const key = getLocalTargetKey(target, index);
		setRunningAction({ key, kind: 'apply' });

		try {
			onStrictChange(localStrict);
			await onApply([target], localStrict, {
				mergeOutput: true,
			});
		} finally {
			setRunningAction(null);
		}
	};

	const globalDryRunning = runningAction?.key === 'global' && runningAction.kind === 'dry-run';
	const globalApplying = runningAction?.key === 'global' && runningAction.kind === 'apply';

	return createPortal(
		<dialog ref={dialogRef} className="modal" onClose={onClose} data-disable-chat-shortcuts="true">
			<div className="modal-box bg-base-100 flex max-h-[calc(100dvh-2rem)] w-11/12 max-w-5xl flex-col overflow-hidden rounded-2xl p-0 shadow-2xl">
				<div className="border-base-300 bg-base-100 border-b px-4 py-3 sm:px-5">
					<div className="flex items-start justify-between gap-4">
						<div className="min-w-0 flex-1">
							<h3 className="flex min-w-0 items-center gap-2 text-base font-semibold">
								<FiGitPullRequest size={16} className="shrink-0" />
								<span className="truncate">Apply unified diff</span>
							</h3>
							<div className="text-base-content/60 mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
								<span>{summary}</span>
								{patchDiagnosticCounts.total > 0 ? (
									<span
										className={`badge badge-outline badge-sm ${getDiagnosticSummaryBadgeClassName(
											patchDiagnosticCounts
										)}`}
										title={formatDiagnosticsTitle(patchDiagnostics)}
									>
										{patchDiagnosticCounts.total} patch diag
										{patchDiagnosticCounts.total === 1 ? '' : 's'}
									</span>
								) : null}
								{missingCount > 0 ? (
									<span className="badge badge-outline badge-warning badge-sm">{missingCount} need paths</span>
								) : null}
							</div>
							<div className="mt-2 flex flex-wrap gap-1.5 text-xs">
								{renderStatusCountBadge('applicable', counts.applicable, 'success')}
								{renderStatusCountBadge('applied', counts.applied, 'success')}
								{renderStatusCountBadge('already applied', counts.alreadyApplied, 'info')}
								{counts.needsInfo > 0 ? (
									<span className="badge badge-outline badge-sm badge-warning">{counts.needsInfo} need info</span>
								) : null}
								{renderStatusCountBadge('blocked', blockedFileCount, 'error')}
								{renderStatusCountBadge('pending', counts.unknown, 'neutral')}
							</div>
						</div>

						<button
							type="button"
							className="btn btn-ghost btn-sm btn-circle"
							onClick={() => dialogRef.current?.close()}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>
				</div>

				<div className="min-h-0 flex-1 overflow-y-auto overscroll-contain p-4 sm:px-5">
					{output?.message || error ? (
						<div className="border-base-300 bg-base-100 mb-4 rounded-xl border px-3 py-2 text-sm">
							<div className="flex items-start gap-2">
								<span
									className={`badge badge-outline badge-sm shrink-0 ${getBadgeToneClassName(
										error ? 'error' : output?.ok ? 'success' : 'warning'
									)}`}
								>
									{error ? 'Error' : output?.ok ? 'Ready' : 'Notice'}
								</span>
								<span className="min-w-0 leading-5">{error || output?.message}</span>
							</div>
						</div>
					) : null}

					{renderDiagnosticsPanel({
						title: 'Patch diagnostics',
						description: 'Applies to the whole patch, not one file section.',
						diagnostics: patchDiagnostics,
						className: 'mb-4',
						footer:
							fileDiagnostics.length > 0
								? `${fileDiagnostics.length} file-specific diagnostic${
										fileDiagnostics.length === 1 ? '' : 's'
									} shown inside file sections.`
								: undefined,
					})}

					<div className="border-base-300 bg-base-100 mb-4 flex flex-col gap-3 rounded-xl border px-3 py-2 sm:flex-row sm:items-center sm:justify-between">
						<label
							className="flex items-center gap-2 text-sm"
							title="Strict disables fuzzy matching in the backend tool"
						>
							<input
								type="checkbox"
								className="checkbox checkbox-xs"
								checked={localStrict}
								onChange={event => {
									setLocalStrict(event.target.checked);
								}}
							/>
							<span>Strict matching</span>
						</label>

						{missingCount > 0 ? (
							<div className="badge badge-outline badge-warning badge-sm">
								{missingCount} target path{missingCount === 1 ? '' : 's'} need attention.
							</div>
						) : null}
					</div>

					<div className="space-y-3">
						{!hasAnyTargets ? (
							<div className="bg-base-100 border-base-300 rounded-xl border p-4 text-sm">
								No file target information could be extracted. Try a dry run, or provide a complete unified diff.
							</div>
						) : null}

						{displayTargets.map((target, index) => {
							const missing = !target.targetPath.trim();
							const inputId = `diff-apply-target-${target.fileKey ?? index}`;
							const candidates = uniqueStrings([
								target.resolvedPath,
								target.targetPath,
								...(target.candidatePaths ?? []),
								target.newPath,
								target.oldPath,
								...candidatePaths,
							]).filter(path => path !== '/dev/null');

							const targetDiagnostics = uniqueDiagnostics(target.diagnostics ?? []);
							const targetKey = getLocalTargetKey(target, index);
							const isTargetDryRunning = runningAction?.key === targetKey && runningAction.kind === 'dry-run';
							const isTargetApplying = runningAction?.key === targetKey && runningAction.kind === 'apply';
							const isProblem = isTargetProblem(target, missing);
							const targetVisualState = getTargetVisualState(target, missing, targetDiagnostics);
							const targetCardClassName = getTargetCardClassName(targetVisualState);
							const targetBadgeClassName = getTargetStatusBadgeClassName(targetVisualState);
							const targetMessageClassName = getTargetMessageClassName(targetVisualState);
							const displayPath = getTargetDisplayPath(target);

							return (
								<div key={targetKey} className={`rounded-xl border p-4 shadow-sm ${targetCardClassName}`}>
									<div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
										<div className="min-w-0 flex-1">
											<div className="flex min-w-0 flex-wrap items-center gap-2">
												<span className={`badge badge-outline badge-sm ${targetBadgeClassName}`}>
													{getTargetStatusLabel(target, missing)}
												</span>
												<span
													className="max-w-full min-w-0 truncate font-mono text-sm font-semibold"
													title={displayPath}
												>
													{displayPath}
												</span>
											</div>

											<div className="mt-2 flex flex-wrap items-center gap-1.5 text-xs">
												{typeof target.hunks === 'number' ? (
													<span className="badge badge-ghost badge-sm">
														{target.hunks} hunk{target.hunks === 1 ? '' : 's'}
													</span>
												) : null}
												{typeof target.addedLines === 'number' ? (
													<span className="badge badge-ghost badge-sm">+{target.addedLines}</span>
												) : null}
												{typeof target.deletedLines === 'number' ? (
													<span className="badge badge-ghost badge-sm">-{target.deletedLines}</span>
												) : null}
												{target.fileKey ? (
													<span className="badge badge-ghost badge-sm font-mono">{target.fileKey}</span>
												) : null}
												{target.sectionKeys && target.sectionKeys.length > 1 ? (
													<span className="badge badge-ghost badge-sm">{target.sectionKeys.length} sections</span>
												) : null}
												{targetDiagnostics.length > 0 ? renderDiagnosticSeveritySummary(targetDiagnostics) : null}
											</div>

											{target.oldPath || target.newPath ? (
												<div className="text-base-content/60 mt-2 grid gap-1 text-[11px]">
													{target.oldPath ? renderPathMeta('old', target.oldPath) : null}
													{target.newPath ? renderPathMeta('new', target.newPath) : null}
												</div>
											) : null}

											{target.message ? (
												<div className={`mt-2 rounded-lg px-3 py-2 text-xs/5 ${targetMessageClassName}`}>
													{target.message}
												</div>
											) : null}
										</div>

										<div className="flex shrink-0 flex-wrap gap-2 sm:justify-end">
											<button
												type="button"
												className="btn btn-xs btn-outline"
												disabled={isRunning}
												onClick={() => {
													void handleTargetDryRun(index);
												}}
											>
												{isTargetDryRunning ? <span className="loading loading-spinner loading-xs" /> : null}
												Dry run
											</button>

											<button
												type="button"
												className="btn btn-xs btn-primary"
												disabled={isRunning || isProblem}
												onClick={() => {
													void handleTargetApply(index);
												}}
											>
												{isTargetApplying ? <span className="loading loading-spinner loading-xs" /> : null}
												Apply
											</button>
										</div>
									</div>

									<label className="text-base-content/70 mt-3 mb-1 block text-xs font-medium" htmlFor={inputId}>
										Target file path
									</label>

									<input
										id={inputId}
										className={`input input-sm w-full font-mono text-xs ${missing ? 'input-error' : ''}`}
										value={target.targetPath}
										onChange={event => {
											updateTarget(index, event.target.value);
										}}
										placeholder="Enter local target file path"
										spellCheck={false}
									/>

									{missing ? (
										<div className="text-error mt-1 text-xs">
											Target path is required before applying this file patch.
										</div>
									) : null}

									{candidates.length > 0 ? (
										<details className="group border-base-300 bg-base-100 mt-3 overflow-hidden rounded-lg border">
											<summary className="flex cursor-pointer list-none items-center justify-between gap-3 px-3 py-2 text-xs font-semibold">
												<span>Candidate paths</span>
												<span className="text-base-content/50 inline-flex items-center gap-1 font-normal">
													<FiChevronRight size={11} className="transition group-open:rotate-90" />
													{candidates.length} options
												</span>
											</summary>
											<div className="border-base-300 border-t px-3 py-2">
												<ul className="grid gap-1.5">
													{candidates.slice(0, 60).map(candidate => (
														<li key={candidate} className="min-w-0">
															<button
																type="button"
																className="btn btn-xs btn-ghost h-auto min-h-0 w-full justify-start rounded-md p-2 text-left font-mono text-[11px]/4 whitespace-normal"
																title={candidate}
																onClick={() => {
																	updateTarget(index, candidate);
																}}
															>
																<span className="min-w-0 break-all">{candidate}</span>
															</button>
														</li>
													))}
													{candidates.length > 60 ? (
														<li className="text-base-content/50 px-2 py-1 text-xs">
															+{candidates.length - 60} more candidate paths
														</li>
													) : null}
												</ul>
											</div>
										</details>
									) : null}

									{targetDiagnostics.length > 0 ? (
										<div className="mt-3">
											{renderDiagnosticsPanel({
												title: 'File diagnostics',
												diagnostics: targetDiagnostics,
											})}
										</div>
									) : null}
								</div>
							);
						})}
					</div>
				</div>

				<div className="border-base-300 bg-base-100 flex shrink-0 flex-wrap items-center justify-end gap-2 border-t px-4 py-3 sm:px-5">
					<button type="button" className="btn btn-sm" onClick={() => dialogRef.current?.close()}>
						Close
					</button>

					<button
						type="button"
						className="btn btn-sm"
						disabled={isRunning}
						onClick={() => {
							void handleDryRun();
						}}
					>
						{globalDryRunning ? <span className="loading loading-spinner loading-xs" /> : null}
						Dry run all
					</button>

					<button
						type="button"
						className="btn btn-sm btn-primary"
						disabled={!canApplyFromModal}
						title="Runs a dry run first, then applies only if the backend reports it is safe."
						onClick={() => {
							void handleApply();
						}}
					>
						{globalApplying ? <span className="loading loading-spinner loading-xs" /> : null}
						Apply all
					</button>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>,
		document.body
	);
}
