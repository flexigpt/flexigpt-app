/* eslint-disable react-you-might-not-need-an-effect/no-adjust-state-on-prop-change */
/* eslint-disable react-hooks/set-state-in-effect */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import {
	FiAlertTriangle,
	FiCheckCircle,
	FiChevronRight,
	FiEye,
	FiGitPullRequest,
	FiInfo,
	FiLoader,
	FiX,
} from 'react-icons/fi';

import {
	type ApplyUnifiedDiffDiagnostic,
	ApplyUnifiedDiffDiagnosticLevel,
	type ApplyUnifiedDiffFileOut,
	type ApplyUnifiedDiffFileTarget,
	type ApplyUnifiedDiffOut,
	ApplyUnifiedDiffStatus,
} from '@/spec/unified_diff';

import { aggregateAPI } from '@/apis/baseapi';

import {
	buildEditableTargetsFromOutput,
	buildUnifiedDiffTextForTarget,
	collectFileLevelDiagnostics,
	collectOutputDiagnostics,
	collectPatchLevelDiagnostics,
	editableTargetsToFileTargets,
	type EditableUnifiedDiffTarget,
	looksLikeUnifiedDiff,
	parseUnifiedDiffForUI,
	summaryLabel,
	uniqueDiagnostics,
} from '@/components/markdown/unified_diff_block';
import { ModalBackdrop } from '@/components/modal_backdrop';

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

interface DiffApplyRunOptions {
	diffText?: string;
	mergeOutput?: boolean;
}

interface FileStatusCounts {
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

interface DiagnosticSeverityCounts {
	total: number;
	error: number;
	warning: number;
	info: number;
}

type HeaderButtonTone = 'neutral' | 'success' | 'warning' | 'error' | 'info';
type ModalRunningAction = { key: string; kind: 'dry-run' | 'apply' };
type TargetVisualState = 'neutral' | 'info' | 'success' | 'warning' | 'error';

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
			if (output?.fileTargets && output.fileTargets.length > 0) {
				setFileTargets(output.fileTargets);
				return output.fileTargets;
			}

			const editable = buildEditableTargetsFromOutput(output, fallbackParsed);
			const next = editableTargetsToFileTargets(editable);

			if (next.length > 0) {
				setFileTargets(next);
				return next;
			}

			if (previousTargets) {
				setFileTargets(previousTargets);
				return previousTargets;
			}

			return [];
		},
		[fallbackParsed]
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

				const key = part.replace(/\n+$/g, '');
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
				const output = await aggregateAPI.applyUnifiedDiff({
					diffText: requestDiffText,
					dryRun: true,
					strict: nextStrict,
					fileTargets: targets.length > 0 ? targets : undefined,
					candidatePaths: normalizedCandidatePaths.length > 0 ? normalizedCandidatePaths : undefined,
				});

				if (seq !== requestSeqRef.current) return undefined;

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
			setFileTargets([]);
			return;
		}

		setStrict(false);
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

		const applyTargets =
			dryRunOutput.fileTargets && dryRunOutput.fileTargets.length > 0 ? dryRunOutput.fileTargets : targets;

		setControlState(previous => ({
			...previous,
			status: 'applying',
			message: 'Applying diff.',
			output: options?.mergeOutput ? previous.output : dryRunViewOutput,
			error: undefined,
		}));
		const applySeq = ++requestSeqRef.current;

		try {
			const output = await aggregateAPI.applyUnifiedDiff({
				diffText: requestDiffText,
				dryRun: false,
				strict: nextStrict,
				fileTargets: applyTargets.length > 0 ? applyTargets : undefined,
				candidatePaths: normalizedCandidatePaths.length > 0 ? normalizedCandidatePaths : undefined,
			});
			if (applySeq !== requestSeqRef.current) return;

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
	const headerStatusSummary = getHeaderStatusSummary(statusCounts, state.output);

	const editableTargetsForHeader = buildEditableTargetsFromOutput(state.output, fallbackParsed);
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
			<div className="text-code flex min-w-0 flex-wrap items-center gap-1.5">
				<FiChevronRight size={12} className="shrink-0" />

				<button
					type="button"
					className={getControlButtonClassName()}
					onClick={() => {
						setIsDetailsOpen(true);
					}}
					title="Open diff details"
					aria-label="Open diff details"
				>
					<FiEye size={12} className="mb-0.5" />
					<span className="leading-none">Details</span>
				</button>

				{isRequestBusy ? (
					<button type="button" className={getControlButtonClassName('info')} disabled title={title}>
						<div className="mb-0.5">{icon}</div>
						{label}
					</button>
				) : hasDryRunResult ? (
					<>
						<span className="max-w-80 truncate px-1 text-[11px] opacity-75" title={headerStatusSummary}>
							{headerStatusSummary}
						</span>

						{statusCounts.applicable > 0 ? (
							<button
								type="button"
								className={getControlButtonClassName('success')}
								disabled={!canHeaderApply}
								onClick={handleHeaderApply}
								title={`${statusCounts.applicable} applicable file patch${statusCounts.applicable === 1 ? '' : 'es'}`}
							>
								<FiGitPullRequest size={12} className="mb-0.5" />
								Apply {statusCounts.applicable}
							</button>
						) : null}

						{statusCounts.blocked > 0 ? (
							<button
								type="button"
								className={getControlButtonClassName('error')}
								onClick={() => {
									setIsDetailsOpen(true);
								}}
								title={`${statusCounts.blocked} blocked file patch${statusCounts.blocked === 1 ? '' : 'es'}`}
							>
								<FiAlertTriangle size={12} className="mb-0.5" />
								Blocked {statusCounts.blocked}
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
								<FiInfo size={12} className="shrink-0" />
								Need info {statusCounts.needsInfo}
							</button>
						) : null}
						{statusCounts.alreadyApplied > 0 ? (
							<button
								type="button"
								className={getControlButtonClassName('info')}
								onClick={() => {
									setIsDetailsOpen(true);
								}}
								title={`${statusCounts.alreadyApplied} already applied file patch${
									statusCounts.alreadyApplied === 1 ? '' : 'es'
								}`}
							>
								<FiCheckCircle size={12} className="mb-0.5" />
								Already Applied {statusCounts.alreadyApplied}
							</button>
						) : null}

						{statusCounts.applied > 0 ? (
							<button
								type="button"
								className={getControlButtonClassName('success')}
								onClick={() => {
									setIsDetailsOpen(true);
								}}
								title={`${statusCounts.applied} applied file patch${statusCounts.applied === 1 ? '' : 'es'}`}
							>
								<FiCheckCircle size={12} className="mb-0.5" />
								Applied {statusCounts.applied}
							</button>
						) : null}

						{statusCounts.notApplicable > 0 ? (
							<button
								type="button"
								className={getControlButtonClassName()}
								onClick={() => {
									setIsDetailsOpen(true);
								}}
								title={`${statusCounts.notApplicable} not applicable file patch${statusCounts.notApplicable === 1 ? '' : 'es'}`}
							>
								Not applicable {statusCounts.notApplicable}
							</button>
						) : null}

						{statusCounts.unknown > 0 ? (
							<button
								type="button"
								className={getControlButtonClassName()}
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
						className={getControlButtonClassName()}
						disabled={state.status === 'idle'}
						onClick={() => {
							setIsDetailsOpen(true);
						}}
						title={`${buttonTitle}\n${title}`}
					>
						<div className="mb-0.5">{icon}</div>
						{label}
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

function DiffApplyModal({
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
	const displayTargets = useMemo(() => groupEditableTargetsForModal(localTargets), [localTargets]);

	const [localStrict, setLocalStrict] = useState(strict);
	const [runningAction, setRunningAction] = useState<ModalRunningAction | null>(null);
	const isRunning = runningAction !== null;
	const patchDiagnostics = uniqueDiagnostics([
		...(fallbackParsed.diagnostics ?? []),
		...collectPatchLevelDiagnostics(output),
	]);

	useEffect(() => {
		if (!isOpen) return;

		const fromOutput = buildEditableTargetsFromOutput(output, fallbackParsed);
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

		setLocalTargets(Array.from(byKey.values()));
		// eslint-disable-next-line react-you-might-not-need-an-effect/no-derived-state
		setLocalStrict(strict);
	}, [fallbackParsed, fileTargets, isOpen, output, strict]);

	useEffect(() => {
		if (!isOpen) return;

		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			dialog.showModal();
		}

		return () => {
			if (dialog.open) dialog.close();
		};
	}, [isOpen]);

	if (!isOpen || typeof document === 'undefined') return null;

	const fileDiagnostics = collectFileLevelDiagnostics(output);
	const summary = summaryLabel(output, fallbackParsed);
	const counts = buildFileStatusCounts(output, fallbackParsed);
	const missingCount = displayTargets.filter(target => !target.targetPath.trim()).length;
	const canApplyFromModal = missingCount === 0 && !isRunning;
	const hasAnyTargets = displayTargets.length > 0;

	const updateTarget = (targetToUpdate: EditableUnifiedDiffTarget, targetPath: string) => {
		setLocalTargets(prev =>
			prev.map(target =>
				editableTargetsMatch(target, targetToUpdate)
					? {
							...target,
							targetPath,
							candidatePaths: uniqueStrings([targetPath, ...target.candidatePaths]),
						}
					: target
			)
		);
	};

	const handleDryRun = async () => {
		if (isRunning) return;
		setRunningAction({ key: 'global', kind: 'dry-run' });
		try {
			onStrictChange(localStrict);
			await onDryRun(localTargets, localStrict);
		} finally {
			setRunningAction(null);
		}
	};

	const handleApply = async () => {
		if (isRunning || !canApplyFromModal) return;

		setRunningAction({ key: 'global', kind: 'apply' });
		try {
			onStrictChange(localStrict);
			await onApply(localTargets, localStrict);
		} finally {
			setRunningAction(null);
		}
	};

	const handleTargetDryRun = async (index: number) => {
		if (isRunning) return;

		const target = displayTargets[index];

		if (!target) return;

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
		if (isRunning) return;

		const target = displayTargets[index];
		if (!target || !target.targetPath.trim()) return;

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
			<div className="modal-box bg-base-200 max-h-[84vh] max-w-5xl overflow-hidden rounded-2xl p-0 shadow-2xl">
				<div className="border-base-300 bg-base-100/70 border-b px-6 py-4">
					<div className="flex items-start justify-between gap-4">
						<div className="min-w-0">
							<h3 className="flex items-center gap-2 text-lg font-bold">
								<FiGitPullRequest size={16} />
								Apply unified diff
							</h3>
							<div className="text-base-content/70 mt-1 text-xs">{summary}</div>
							<div className="mt-3 flex flex-wrap gap-2 text-xs">
								<span className="badge badge-outline badge-success">{counts.applicable} applicable</span>
								<span className={`badge badge-outline ${counts.notApplicable > 0 ? 'badge-warning' : 'badge-ghost'}`}>
									{counts.notApplicable} not applicable
								</span>
								{counts.blocked > 0 ? (
									<span className="badge badge-outline badge-error">{counts.blocked} blocked</span>
								) : null}
								{counts.needsInfo > 0 ? (
									<span className="badge badge-outline badge-warning">{counts.needsInfo} need info</span>
								) : null}
								{counts.applied > 0 ? (
									<span className="badge badge-outline badge-success">{counts.applied} applied</span>
								) : null}
								{counts.alreadyApplied > 0 ? (
									<span className="badge badge-outline badge-info">{counts.alreadyApplied} already applied</span>
								) : null}
							</div>
						</div>

						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={() => dialogRef.current?.close()}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>
				</div>

				<div className="flex-1 overflow-y-auto p-6">
					{output?.message || error ? (
						<div
							className={`alert mb-4 py-2 text-sm ${error ? 'alert-error' : output?.ok ? 'alert-success' : 'alert-warning'}`}
						>
							<span>{error || output?.message}</span>
						</div>
					) : null}
					{patchDiagnostics.length > 0 ? (
						<div
							className={`mb-4 rounded-xl border p-4 shadow-sm ${getDiagnosticSectionClassName(
								getHighestDiagnosticLevel(patchDiagnostics)
							)}`}
						>
							<div className="mb-2 flex items-start justify-between gap-3">
								<div>
									<div className="flex items-center gap-2 text-sm font-semibold">
										<FiInfo size={14} className="text-info" />
										Patch-level diagnostics
									</div>
									<div className="text-base-content/60 mt-1 text-xs">
										These belong to the patch as a whole, not to a single file section.
									</div>
								</div>
								{renderDiagnosticSeveritySummary(patchDiagnostics)}
							</div>
							<div className="space-y-2">
								{patchDiagnostics.slice(0, 8).map((diagnostic, index) => {
									const display = getDiagnosticDisplay(diagnostic.level);
									return (
										<div
											key={`${diagnostic.level}-${diagnostic.code ?? 'nocode'}-${diagnostic.message}-${index}`}
											className={`rounded-lg border px-3 py-2 text-xs ${display.containerClassName}`}
										>
											<div className="flex items-start gap-2">
												<div className="mt-0.5 shrink-0">{display.icon}</div>
												<div className="min-w-0 flex-1">
													<div className="flex flex-wrap items-center gap-2">
														<span className={`badge badge-sm ${display.badgeClassName}`}>{display.label}</span>
														{diagnostic.code ? (
															<span className="badge badge-ghost badge-sm font-mono">{diagnostic.code}</span>
														) : null}
													</div>
													<div className="mt-1 leading-5 whitespace-pre-wrap">{diagnostic.message}</div>
												</div>
											</div>
										</div>
									);
								})}
								{patchDiagnostics.length > 8 ? (
									<div className="text-base-content/50 text-xs">
										+{patchDiagnostics.length - 8} more patch-level diagnostics
									</div>
								) : null}
								{fileDiagnostics.length > 0 ? (
									<div className="border-base-300 bg-base-100/70 text-base-content/70 rounded-lg border px-3 py-2 text-xs">
										{fileDiagnostics.length} file-specific diagnostic
										{fileDiagnostics.length === 1 ? '' : 's'} are shown inside the file sections below.
									</div>
								) : null}
							</div>
						</div>
					) : null}
					<div className="bg-base-100 border-base-300 mb-5 flex flex-col gap-3 rounded-xl border p-3 sm:flex-row sm:items-center sm:justify-between">
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
							<div className="text-warning text-xs">
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
							const highestTargetDiagnosticLevel = getHighestDiagnosticLevel(targetDiagnostics);

							return (
								<div key={targetKey} className={`rounded-xl border p-4 shadow-sm ${targetCardClassName}`}>
									<div className="mb-3 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
										<div className="min-w-0">
											<div className="mb-2 flex flex-wrap items-center gap-2 text-xs">
												<span className={`badge badge-outline ${targetBadgeClassName}`}>
													{getTargetStatusLabel(target, missing)}
												</span>
												{target.fileKey ? <span className="badge badge-ghost font-mono">{target.fileKey}</span> : null}
												{target.sectionKeys && target.sectionKeys.length > 1 ? (
													<span className="badge badge-ghost">{target.sectionKeys.length} sections</span>
												) : null}
												{target.oldPath ? (
													<span className="badge badge-outline max-w-full justify-start font-mono">
														<span className="max-w-[18rem] truncate">old: {target.oldPath}</span>
													</span>
												) : null}
												{target.newPath ? (
													<span className="badge badge-outline max-w-full justify-start font-mono">
														<span className="max-w-[18rem] truncate">new: {target.newPath}</span>
													</span>
												) : null}
											</div>

											{renderDiagnosticSeveritySummary(targetDiagnostics)}

											{target.message ? (
												<div className={`rounded-lg px-3 py-2 text-xs ${targetMessageClassName}`}>{target.message}</div>
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

									<div className="mb-3 flex flex-wrap gap-2 text-xs">
										{typeof target.hunks === 'number' ? (
											<span className="badge badge-ghost">
												{target.hunks} hunk{target.hunks === 1 ? '' : 's'}
											</span>
										) : null}
										{typeof target.addedLines === 'number' ? (
											<span className="badge badge-ghost">+{target.addedLines}</span>
										) : null}
										{typeof target.deletedLines === 'number' ? (
											<span className="badge badge-ghost">-{target.deletedLines}</span>
										) : null}
									</div>

									<label className="mb-1 block text-sm font-semibold" htmlFor={inputId}>
										Target file path
									</label>

									<input
										id={inputId}
										className={`input input-sm input-bordered w-full ${missing ? 'input-error' : ''}`}
										value={target.targetPath}
										onChange={event => {
											updateTarget(target, event.target.value);
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
										<details className="group border-base-300 bg-base-200/60 mt-3 overflow-hidden rounded-lg border">
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
																className="btn btn-xs btn-ghost h-auto min-h-0 w-full justify-start rounded-md px-2 py-2 text-left font-mono text-[11px] leading-4 whitespace-normal"
																title={candidate}
																onClick={() => {
																	updateTarget(target, candidate);
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
										<div
											className={`mt-3 rounded-lg border p-3 text-xs ${
												highestTargetDiagnosticLevel === ApplyUnifiedDiffDiagnosticLevel.Error
													? 'border-error/30 bg-error/10 text-error'
													: highestTargetDiagnosticLevel === ApplyUnifiedDiffDiagnosticLevel.Warning
														? 'border-warning/30 bg-warning/10 text-warning'
														: 'border-base-300 bg-base-200/70 text-base-content/70'
											}`}
										>
											<div className="mb-1 flex items-center gap-1 font-semibold">
												{highestTargetDiagnosticLevel === ApplyUnifiedDiffDiagnosticLevel.Error ? (
													<FiAlertTriangle size={12} className="text-error" />
												) : highestTargetDiagnosticLevel === ApplyUnifiedDiffDiagnosticLevel.Warning ? (
													<FiAlertTriangle size={12} className="text-warning" />
												) : (
													<FiInfo size={12} className="text-info" />
												)}
												File diagnostics
											</div>
											{renderDiagnosticSeveritySummary(targetDiagnostics)}

											<div className="space-y-2">
												{targetDiagnostics.slice(0, 8).map((diagnostic, diagIndex) => {
													const display = getDiagnosticDisplay(diagnostic.level);
													return (
														<div
															key={`${diagnostic.level}-${diagnostic.code ?? 'nocode'}-${diagnostic.message}-${diagIndex}`}
															className={`rounded-md border px-3 py-2 ${display.containerClassName}`}
														>
															<div className="flex items-start gap-2">
																<div className="mt-0.5 shrink-0">{display.icon}</div>
																<div className="min-w-0 flex-1">
																	<div className="flex flex-wrap items-center gap-2">
																		<span className={`badge badge-xs ${display.badgeClassName}`}>{display.label}</span>
																		{diagnostic.code ? (
																			<span className="badge badge-ghost badge-xs font-mono">{diagnostic.code}</span>
																		) : null}
																	</div>
																	<div className="mt-1 leading-5 whitespace-pre-wrap">{diagnostic.message}</div>
																</div>
															</div>
														</div>
													);
												})}
												{targetDiagnostics.length > 8 ? (
													<div className="text-base-content/50 text-xs">
														+{targetDiagnostics.length - 8} more file diagnostics
													</div>
												) : null}
											</div>
										</div>
									) : null}
								</div>
							);
						})}
					</div>
				</div>

				<div className="border-base-300 bg-base-100/80 flex flex-wrap items-center justify-end gap-2 border-t px-6 py-4">
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
						onClick={() => {
							void handleApply();
						}}
					>
						{globalApplying ? <span className="loading loading-spinner loading-xs" /> : null}
						Dry run and apply all
					</button>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>,
		document.body
	);
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

function buildFileStatusCounts(
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
				counts.applicable = output.ok ? total : 0;
				counts.blocked = output.ok ? 0 : total;
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
			case ApplyUnifiedDiffStatus.Error:
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
				case ApplyUnifiedDiffStatus.Error:
				default:
					counts.error += 1;
					counts.blocked += 1;
					break;
			}
		}
	}

	counts.unknown = Math.max(0, total - files.length);
	counts.notApplicable = Math.max(0, total - counts.applicable);

	return counts;
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

	return parts.join('\n');
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
		case ApplyUnifiedDiffStatus.Conflict:
		case ApplyUnifiedDiffStatus.Error:
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
		neutral: 'text-code',
		success: 'text-success',
		warning: 'text-warning',
		error: 'text-error',
		info: 'text-info',
	};

	return `inline-flex h-6 items-center gap-1 rounded-full border border-base-300 bg-base-100/60 px-2 text-[11px] font-medium leading-none whitespace-nowrap shadow-none transition-colors hover:border-base-300 hover:bg-base-100/80 hover:opacity-100 disabled:cursor-not-allowed disabled:bg-base-100/40 disabled:opacity-50 ${toneClassName[tone]}`;
}

function getDiagnosticSeverityCounts(diagnostics: ApplyUnifiedDiffDiagnostic[]): DiagnosticSeverityCounts {
	const counts: DiagnosticSeverityCounts = {
		total: diagnostics.length,
		error: 0,
		warning: 0,
		info: 0,
	};

	for (const diagnostic of diagnostics) {
		switch (diagnostic.level) {
			case ApplyUnifiedDiffDiagnosticLevel.Error:
				counts.error += 1;
				break;
			case ApplyUnifiedDiffDiagnosticLevel.Warning:
				counts.warning += 1;
				break;
			case ApplyUnifiedDiffDiagnosticLevel.Info:
			default:
				counts.info += 1;
				break;
		}
	}

	return counts;
}

function renderDiagnosticSeveritySummary(diagnostics: ApplyUnifiedDiffDiagnostic[]) {
	const counts = getDiagnosticSeverityCounts(diagnostics);
	if (counts.total === 0) return null;

	return (
		<div className="flex flex-wrap gap-1.5 text-[11px]">
			{counts.error > 0 ? (
				<span className="badge badge-outline badge-error">
					{counts.error} error{counts.error === 1 ? '' : 's'}
				</span>
			) : null}
			{counts.warning > 0 ? (
				<span className="badge badge-outline badge-warning">
					{counts.warning} warning{counts.warning === 1 ? '' : 's'}
				</span>
			) : null}
			{counts.info > 0 ? (
				<span className="badge badge-outline badge-info">
					{counts.info} info{counts.info === 1 ? '' : 's'}
				</span>
			) : null}
		</div>
	);
}

function getDiagnosticSectionClassName(level?: ApplyUnifiedDiffDiagnosticLevel): string {
	switch (level) {
		case ApplyUnifiedDiffDiagnosticLevel.Error:
			return 'border-error/30 bg-error/5 border-l-4 border-l-error';
		case ApplyUnifiedDiffDiagnosticLevel.Warning:
			return 'border-warning/30 bg-warning/5 border-l-4 border-l-warning';
		case ApplyUnifiedDiffDiagnosticLevel.Info:
		default:
			return 'border-info/30 bg-info/5 border-l-4 border-l-info';
	}
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

	if (target.status === ApplyUnifiedDiffStatus.Applicable) return 'success';
	if (target.status === ApplyUnifiedDiffStatus.Applied || target.status === ApplyUnifiedDiffStatus.AlreadyApplied)
		return 'info';
	if (diagnostics.some(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Info)) return 'info';
	return 'neutral';
}

function getTargetMessageClassName(visualState: TargetVisualState): string {
	switch (visualState) {
		case 'error':
			return 'bg-error/10 text-error';
		case 'warning':
			return 'bg-warning/10 text-warning';
		case 'success':
			return 'bg-success/10 text-success';
		case 'info':
			return 'bg-info/10 text-info';
		case 'neutral':
		default:
			return 'bg-base-200 text-base-content/70';
	}
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

	return haveSharedPathIdentity(
		[left.targetPath, left.resolvedPath, left.newPath, left.oldPath],
		[right.targetPath, right.resolvedPath, right.newPath, right.oldPath]
	);
}

function fileTargetsMatch(left: ApplyUnifiedDiffFileTarget, right: ApplyUnifiedDiffFileTarget): boolean {
	if (left.fileKey && right.fileKey && left.fileKey === right.fileKey) return true;
	return haveSharedPathIdentity(
		[left.targetPath, left.newPath, left.oldPath],
		[right.targetPath, right.newPath, right.oldPath]
	);
}

function upsertEditableTargetForModal(
	byKey: Map<string, EditableUnifiedDiffTarget>,
	target: EditableUnifiedDiffTarget
) {
	const existingEntry = Array.from(byKey.entries()).find(([, existing]) => editableTargetsMatch(existing, target));
	const key = existingEntry?.[0] ?? getLocalTargetKey(target, byKey.size);
	const existing = existingEntry?.[1];

	byKey.set(key, mergeEditableTargetForModal(existing, target));
}

function editableTargetsMatch(left: EditableUnifiedDiffTarget, right: EditableUnifiedDiffTarget): boolean {
	const leftKeys = uniqueStrings([left.fileKey, ...(left.sectionKeys ?? [])]);
	const rightKeys = uniqueStrings([right.fileKey, ...(right.sectionKeys ?? [])]);

	if (leftKeys.some(key => rightKeys.includes(key))) return true;

	return haveSharedPathIdentity(
		[left.targetPath, left.resolvedPath, left.newPath, left.oldPath, ...(left.candidatePaths ?? [])],
		[right.targetPath, right.resolvedPath, right.newPath, right.oldPath, ...(right.candidatePaths ?? [])]
	);
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

function getLocalTargetKey(target: EditableUnifiedDiffTarget, index: number): string {
	return (
		target.fileKey ||
		target.sectionKeys?.join('|') ||
		getPathIdentity(target.targetPath) ||
		getPathIdentity(target.newPath) ||
		getPathIdentity(target.oldPath) ||
		`target-${index}`
	);
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

function getHeaderStatusSummary(counts: FileStatusCounts, output?: ApplyUnifiedDiffOut): string {
	const totalLabel = `${counts.total} file patch${counts.total === 1 ? '' : 'es'} detected`;

	if (!output) return totalLabel;

	if (output.dryRun) {
		return `${totalLabel} · ${counts.applicable} applicable · ${counts.notApplicable} not applicable`;
	}

	if (output.status === ApplyUnifiedDiffStatus.Applied) {
		return `${totalLabel} · ${counts.applied} applied`;
	}

	if (output.status === ApplyUnifiedDiffStatus.AlreadyApplied) {
		return `${totalLabel} · ${counts.alreadyApplied} already applied`;
	}
	if (output.status === ApplyUnifiedDiffStatus.NeedsInfo) {
		return `${totalLabel} · ${counts.needsInfo} need info`;
	}

	if (output.status === ApplyUnifiedDiffStatus.Conflict || output.status === ApplyUnifiedDiffStatus.Error) {
		return `${totalLabel} · ${counts.blocked} blocked`;
	}

	return `${totalLabel} · ${counts.applicable} applicable`;
}

function getDiagnosticDisplay(level: ApplyUnifiedDiffDiagnosticLevel) {
	switch (level) {
		case ApplyUnifiedDiffDiagnosticLevel.Error:
			return {
				label: 'Error',
				badgeClassName: 'badge-error',
				containerClassName: 'border-error/30 bg-error/10 text-error border-l-4 border-l-error',
				icon: <FiX size={12} />,
			};
		case ApplyUnifiedDiffDiagnosticLevel.Warning:
			return {
				label: 'Warning',
				badgeClassName: 'badge-warning',
				containerClassName: 'border-warning/30 bg-warning/10 text-warning border-l-4 border-l-warning',
				icon: <FiAlertTriangle size={12} />,
			};
		case ApplyUnifiedDiffDiagnosticLevel.Info:
		default:
			return {
				label: 'Info',
				badgeClassName: 'badge-info',
				containerClassName: 'border-info/30 bg-info/10 text-info border-l-4 border-l-info',
				icon: <FiInfo size={12} />,
			};
	}
}

function getTargetCardClassName(visualState: TargetVisualState): string {
	switch (visualState) {
		case 'error':
			return 'border-error/40 border-l-4 border-l-error bg-error/5';
		case 'warning':
			return 'border-warning/40 border-l-4 border-l-warning bg-warning/5';
		case 'success':
			return 'border-success/40 border-l-4 border-l-success bg-success/5';
		case 'info':
			return 'border-info/40 border-l-4 border-l-info bg-info/5';
		case 'neutral':
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
		case 'neutral':
		default:
			return 'badge-ghost';
	}
}

function getTargetStatusLabel(target: EditableUnifiedDiffTarget, missing: boolean): string {
	if (missing) return 'needs path';
	if (target.status) return target.status.replaceAll('_', ' ');
	if (target.ok === false) return 'blocked';
	return 'pending';
}

function getHighestDiagnosticLevel(
	diagnostics: ApplyUnifiedDiffDiagnostic[]
): ApplyUnifiedDiffDiagnosticLevel | undefined {
	if (diagnostics.some(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Error)) {
		return ApplyUnifiedDiffDiagnosticLevel.Error;
	}

	if (diagnostics.some(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Warning)) {
		return ApplyUnifiedDiffDiagnosticLevel.Warning;
	}

	if (diagnostics.some(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Info)) {
		return ApplyUnifiedDiffDiagnosticLevel.Info;
	}

	return undefined;
}

function mergeNumberMax(left: number | undefined, right: number | undefined): number | undefined {
	if (typeof left === 'number' && typeof right === 'number') return Math.max(left, right);
	if (typeof right === 'number') return right;
	return left;
}

function haveSharedPathIdentity(left: Array<string | undefined>, right: Array<string | undefined>): boolean {
	const leftSet = new Set(left.map(getPathIdentity).filter(Boolean));
	if (leftSet.size === 0) return false;
	return right.map(getPathIdentity).some(identity => !!identity && leftSet.has(identity));
}

function getPathIdentity(value: string | undefined | null): string {
	const trimmed = value?.trim();
	if (!trimmed || trimmed === '/dev/null') return '';

	return trimmed
		.replaceAll('\\', '/')
		.replace(/\/+/g, '/')
		.replace(/^(?:\.\/)+/, '');
}

function joinDiffTextParts(parts: string[]): string {
	return parts.map(part => part.replace(/\n+$/g, '')).join('\n');
}

function isDiffPartHunkComplete(diffPart: string, expectedHunks: number | undefined): boolean {
	if (typeof expectedHunks !== 'number' || expectedHunks <= 0) return true;
	return diffPart.split('\n').filter(line => line.startsWith('@@')).length === expectedHunks;
}

function groupEditableTargetsForModal(targets: EditableUnifiedDiffTarget[]): EditableUnifiedDiffTarget[] {
	const byKey = new Map<string, EditableUnifiedDiffTarget>();
	for (const target of targets) {
		upsertEditableTargetForModal(byKey, target);
	}
	return Array.from(byKey.values());
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

function getErrorMessage(error: unknown): string {
	if (error instanceof Error && error.message.trim()) return error.message;
	return 'Unexpected error while checking or applying unified diff.';
}
