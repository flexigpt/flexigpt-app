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

import { type ApplyUnifiedDiffFileTarget, type ApplyUnifiedDiffOut, ApplyUnifiedDiffStatus } from '@/spec/unified_diff';

import { aggregateAPI } from '@/apis/baseapi';

import {
	buildEditableTargetsFromOutput,
	collectOutputDiagnostics,
	editableTargetsToFileTargets,
	type EditableUnifiedDiffTarget,
	looksLikeUnifiedDiff,
	parseUnifiedDiffForUI,
	summaryLabel,
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

export function DiffApplyControl({ language, diffText, isBusy, candidatePaths }: DiffApplyControlProps) {
	const isDiffLike = useMemo(() => looksLikeUnifiedDiff(diffText, language), [diffText, language]);
	const fallbackParsed = useMemo(() => parseUnifiedDiffForUI(diffText, language), [diffText, language]);
	const normalizedCandidatePaths = useMemo(() => uniqueStrings(candidatePaths ?? []), [candidatePaths]);

	const [state, setState] = useState<DiffApplyState>({ status: 'idle' });
	const [fileTargets, setFileTargets] = useState<ApplyUnifiedDiffFileTarget[]>([]);
	const [strict, setStrict] = useState(false);
	const [isDetailsOpen, setIsDetailsOpen] = useState(false);

	const requestSeqRef = useRef(0);

	const blockKey = useMemo(
		() => `${language}\u0000${diffText}\u0000${normalizedCandidatePaths.join('\u0000')}`,
		[diffText, language, normalizedCandidatePaths]
	);
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

	const runDryRun = useCallback(
		async (targets: ApplyUnifiedDiffFileTarget[], nextStrict: boolean) => {
			const seq = ++requestSeqRef.current;

			setState({
				status: 'checking',
				message: 'Checking whether this diff can be applied.',
			});

			try {
				const output = await aggregateAPI.applyUnifiedDiff({
					diffText,
					dryRun: true,
					strict: nextStrict,
					fileTargets: targets.length > 0 ? targets : undefined,
					candidatePaths: normalizedCandidatePaths.length > 0 ? normalizedCandidatePaths : undefined,
				});

				if (seq !== requestSeqRef.current) return undefined;

				deriveAndStoreTargets(output, targets);

				setState({
					status: mapOutputToControlStatus(output),
					message: output.message,
					output,
				});

				return output;
			} catch (error) {
				if (seq !== requestSeqRef.current) return undefined;

				setState({
					status: 'blocked',
					error: getErrorMessage(error),
				});
				return undefined;
			}
		},
		[deriveAndStoreTargets, diffText, normalizedCandidatePaths]
	);

	useEffect(() => {
		requestSeqRef.current += 1;

		if (!isDiffLike || isBusy) {
			setState({ status: 'idle' });
			setFileTargets([]);
			return;
		}

		setStrict(false);
		setFileTargets([]);

		void runDryRun([], false);
	}, [blockKey, isBusy, isDiffLike, runDryRun]);

	const runApply = async (targets: ApplyUnifiedDiffFileTarget[], nextStrict: boolean) => {
		setState({
			status: 'checking',
			message: 'Rechecking diff before applying.',
		});

		const dryRunOutput = await runDryRun(targets, nextStrict);
		if (!dryRunOutput) return;

		if (!dryRunOutput.ok) {
			setState({
				status: mapOutputToControlStatus(dryRunOutput),
				message: dryRunOutput.message,
				output: dryRunOutput,
			});
			return;
		}

		if (dryRunOutput.status === ApplyUnifiedDiffStatus.AlreadyApplied) {
			setState({
				status: 'already-applied',
				message: dryRunOutput.message || 'Unified diff is already applied.',
				output: dryRunOutput,
			});
			return;
		}

		const applyTargets =
			dryRunOutput.fileTargets && dryRunOutput.fileTargets.length > 0 ? dryRunOutput.fileTargets : targets;

		setState({
			status: 'applying',
			message: 'Applying diff.',
			output: dryRunOutput,
		});
		const applySeq = ++requestSeqRef.current;

		try {
			const output = await aggregateAPI.applyUnifiedDiff({
				diffText,
				dryRun: false,
				strict: nextStrict,
				fileTargets: applyTargets.length > 0 ? applyTargets : undefined,
				candidatePaths: normalizedCandidatePaths.length > 0 ? normalizedCandidatePaths : undefined,
			});
			if (applySeq !== requestSeqRef.current) return;

			deriveAndStoreTargets(output, applyTargets);

			setState({
				status: mapOutputToControlStatus(output),
				message: output.message,
				output,
			});
		} catch (error) {
			if (applySeq !== requestSeqRef.current) return;

			setState({
				status: 'blocked',
				error: getErrorMessage(error),
			});
		}
	};

	if (!isDiffLike || isBusy) return null;

	const title = buildTitle(state, fallbackParsed);
	const buttonTitle = getButtonTitle(state.status);
	const label = getButtonLabel(state.status);
	const icon = getButtonIcon(state.status);

	const canMainApply = state.status === 'ready';

	return (
		<>
			<div className="text-code flex items-center justify-between gap-2">
				<FiChevronRight size={12} />
				{canMainApply ? (
					<div className="flex items-center gap-2">
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
							<span className="leading-none">View Details</span>
						</button>
						<FiChevronRight size={12} />
					</div>
				) : null}

				<button
					type="button"
					className={getControlButtonClassName()}
					disabled={state.status === 'idle' || state.status === 'checking' || state.status === 'applying'}
					onClick={() => {
						if (canMainApply) {
							void runApply(fileTargets, strict);
							return;
						}
						setIsDetailsOpen(true);
					}}
					title={`${buttonTitle}\n${title}`}
				>
					<div className="mb-0.5">{icon}</div>
					{label}
				</button>
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
				onDryRun={async (nextTargets, nextStrict) => {
					const normalized = editableTargetsToFileTargets(nextTargets);
					setFileTargets(normalized);
					await runDryRun(normalized, nextStrict);
				}}
				onApply={async (nextTargets, nextStrict) => {
					const normalized = editableTargetsToFileTargets(nextTargets);
					setFileTargets(normalized);
					await runApply(normalized, nextStrict);
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
	onDryRun: (targets: EditableUnifiedDiffTarget[], strict: boolean) => Promise<void>;
	onApply: (targets: EditableUnifiedDiffTarget[], strict: boolean) => Promise<void>;
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
	const [localStrict, setLocalStrict] = useState(strict);
	const [isRunning, setIsRunning] = useState(false);

	useEffect(() => {
		if (!isOpen) return;

		const fromOutput = buildEditableTargetsFromOutput(output, fallbackParsed);
		const byKey = new Map<string, EditableUnifiedDiffTarget>();

		for (const target of fromOutput) {
			byKey.set(target.fileKey || `${target.oldPath ?? ''}\u0000${target.newPath ?? ''}`, target);
		}

		for (const target of fileTargets) {
			const key = target.fileKey || `${target.oldPath ?? ''}\u0000${target.newPath ?? ''}`;
			const existing = byKey.get(key);

			byKey.set(key, {
				...existing,
				fileKey: target.fileKey || existing?.fileKey,
				oldPath: target.oldPath || existing?.oldPath,
				newPath: target.newPath || existing?.newPath,
				targetPath: target.targetPath || existing?.targetPath || '',
				candidatePaths: uniqueStrings([target.targetPath, ...(existing?.candidatePaths ?? [])]),
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

	const summary = summaryLabel(output, fallbackParsed);
	const diagnostics = uniqueStrings([...collectOutputDiagnostics(output), ...(fallbackParsed.diagnostics ?? [])]);
	const missingCount = localTargets.filter(target => !target.targetPath.trim()).length;
	const canApplyFromModal = missingCount === 0 && !isRunning;
	const hasAnyTargets = localTargets.length > 0;

	const updateTarget = (index: number, targetPath: string) => {
		setLocalTargets(prev =>
			prev.map((target, i) =>
				i === index
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
		setIsRunning(true);
		try {
			onStrictChange(localStrict);
			await onDryRun(localTargets, localStrict);
		} finally {
			setIsRunning(false);
		}
	};

	const handleApply = async () => {
		if (isRunning || !canApplyFromModal) return;

		setIsRunning(true);
		try {
			onStrictChange(localStrict);
			await onApply(localTargets, localStrict);
		} finally {
			setIsRunning(false);
		}
	};

	return createPortal(
		<dialog ref={dialogRef} className="modal" onClose={onClose} data-disable-chat-shortcuts="true">
			<div className="modal-box bg-base-200 max-h-[82vh] max-w-4xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[82vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between gap-4">
						<div>
							<h3 className="flex items-center gap-2 text-lg font-bold">
								<FiGitPullRequest size={16} />
								Apply unified diff
							</h3>
							<div className="text-base-content/70 mt-1 text-xs">{summary}</div>
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

					{output?.message || error ? (
						<div className={`alert mb-4 py-2 text-sm ${output?.ok ? 'alert-success' : 'alert-warning'}`}>
							<span>{error || output?.message}</span>
						</div>
					) : null}

					{diagnostics.length > 0 ? (
						<div className="bg-base-100 border-base-300 mb-4 rounded-xl border p-3 text-xs">
							<div className="mb-1 font-semibold">Diagnostics</div>
							<ul className="list-disc space-y-1 pl-4">
								{diagnostics.slice(0, 16).map((item, index) => (
									<li key={`${item}-${index}`}>{item}</li>
								))}
							</ul>
						</div>
					) : null}

					<div className="mb-4 flex items-center justify-between gap-4">
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

					<div className="space-y-4">
						{!hasAnyTargets ? (
							<div className="bg-base-100 border-base-300 rounded-xl border p-4 text-sm">
								No file target information could be extracted. Try a dry run, or provide a complete unified diff.
							</div>
						) : null}
						{localTargets.map((target, index) => {
							const missing = !target.targetPath.trim();
							const inputId = `diff-apply-target-${target.fileKey ?? index}`;
							const candidates = uniqueStrings([
								target.targetPath,
								...(target.candidatePaths ?? []),
								target.newPath,
								target.oldPath,
								...candidatePaths,
							]).filter(path => path !== '/dev/null');

							return (
								<div
									key={`${target.fileKey ?? index}-${target.oldPath ?? ''}-${target.newPath ?? ''}`}
									className="bg-base-100 border-base-300 rounded-xl border p-3"
								>
									<div className="mb-2 flex flex-wrap items-center gap-2 text-xs">
										{target.fileKey ? <span className="badge badge-ghost">{target.fileKey}</span> : null}
										{target.status ? <span className="badge badge-outline">{target.status}</span> : null}
										{target.oldPath ? (
											<span className="badge badge-outline max-w-full truncate">old: {target.oldPath}</span>
										) : null}
										{target.newPath ? (
											<span className="badge badge-outline max-w-full truncate">new: {target.newPath}</span>
										) : null}
									</div>

									{target.message ? <div className="text-base-content/70 mb-2 text-xs">{target.message}</div> : null}
									<div className="mb-2 flex flex-wrap gap-2 text-xs">
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
										<div className="mt-2 flex flex-wrap gap-2">
											{candidates.slice(0, 18).map(candidate => (
												<button
													key={candidate}
													type="button"
													className="btn btn-xs btn-ghost max-w-full"
													title={candidate}
													onClick={() => {
														updateTarget(index, candidate);
													}}
												>
													<span className="max-w-96 truncate">{candidate}</span>
												</button>
											))}
										</div>
									) : null}

									{target.diagnostics && target.diagnostics.length > 0 ? (
										<div className="text-base-content/60 mt-2 text-xs">
											{target.diagnostics.slice(0, 4).map((diag, diagIndex) => (
												<div key={`${diag}-${diagIndex}`}>{diag}</div>
											))}
										</div>
									) : null}
								</div>
							);
						})}
					</div>

					<div className="modal-action">
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
							{isRunning ? <span className="loading loading-spinner loading-xs" /> : null}
							Dry run
						</button>

						<button
							type="button"
							className="btn btn-sm btn-primary"
							disabled={!canApplyFromModal}
							onClick={() => {
								void handleApply();
							}}
						>
							{isRunning ? <span className="loading loading-spinner loading-xs" /> : null}
							Dry run and apply
						</button>
					</div>
				</div>
			</div>

			<ModalBackdrop enabled={true} />
		</dialog>,
		document.body
	);
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

function getControlButtonClassName(): string {
	return 'inline-flex items-center gap-1 leading-none whitespace-nowrap border-none bg-transparent shadow-none hover:opacity-60 disabled:bg-transparent disabled:opacity-50';
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
