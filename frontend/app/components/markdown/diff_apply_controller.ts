import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';

import type { ApplyUnifiedDiffDiagnostic } from '@/spec/unified_diff';

import { getErrorMessage } from '@/lib/error_utils';

import { aggregateAPI } from '@/apis/baseapi';

import type {
	DiffApplyOutcome,
	DiffApplyOutcomeStatus,
	DiffApplyPhase,
	InteractiveDiffFile,
	InteractiveDiffHunk,
	ParsedInteractiveDiff,
} from '@/components/markdown/diff_apply_model';
import {
	buildInteractiveDiffRequest,
	createDiffApplyOutcome,
	inferInteractiveDiffTargets,
	interpretScopedApplyResult,
	MAX_INTERACTIVE_HUNK_PROBES,
	normalizeInteractiveTargetPaths,
} from '@/components/markdown/diff_apply_model';

const EMPTY_PATHS: string[] = [];
const REVIEW_CONCURRENCY = 3;
const HUNK_REVIEW_CONCURRENCY = 2;

type DiffApplyOperationKind = 'review' | 'apply' | 'review-hunks' | 'apply-hunks';

interface DiffApplyBusyState {
	id: number;
	kind: DiffApplyOperationKind;
	total: number;
	completed: number;
	activeFileIDs: string[];
	stopping: boolean;
}

export interface DiffApplyFileView {
	file: InteractiveDiffFile;
	pathInput: string;

	/**
	 * Explicit target entered or selected by the user.
	 */
	targetPath: string;

	/**
	 * Only an explicit user target is sent back to the backend. Backend target
	 * resolution remains display information, not a frontend decision.
	 */
	effectiveTargetPath: string;

	candidates: string[];
	fingerprint: string;
	reviewOutcome?: DiffApplyOutcome;
	applyOutcome?: DiffApplyOutcome;
	hunkOutcomes: Record<string, DiffApplyOutcome>;
	selectedHunkIDs: string[];
	status: DiffApplyOutcomeStatus | 'pending';
	canApply: boolean;
	canTryHunks: boolean;
}

export interface DiffApplyController {
	files: DiffApplyFileView[];
	patchDiagnostics: ApplyUnifiedDiffDiagnostic[];
	strict: boolean;
	busy: DiffApplyBusyState | null;
	hasReviewResult: boolean;
	reviewedFileCount: number;
	readyFileCount: number;
	appliedFileCount: number;
	alreadyAppliedFileCount: number;
	blockedFileCount: number;
	needsInfoFileCount: number;
	pendingFileCount: number;
	partialFileCount: number;
	setStrict: (value: boolean) => void;
	setTargetPath: (fileID: string, value: string) => void;
	setHunkSelected: (fileID: string, hunkID: string, selected: boolean) => void;
	cancel: () => void;
	reviewAll: () => Promise<void>;
	reviewFile: (fileID: string) => Promise<void>;
	applyReady: () => Promise<void>;
	applyFile: (fileID: string) => Promise<void>;
	reviewHunks: (fileID: string) => Promise<void>;
	applySelectedHunks: (fileID: string) => Promise<void>;
}

interface StoredFileState {
	fingerprint: string;
	source: InteractiveDiffFile;
	reviewOutcome?: DiffApplyOutcome;
	applyOutcome?: DiffApplyOutcome;
	hunkOutcomes: Record<string, DiffApplyOutcome>;
	selectedHunkIDs: string[];
}

interface TargetInputState {
	source: InteractiveDiffFile;
	value: string;
}

interface ActiveOperation {
	id: number;
	cancelled: boolean;
}

let writeQueue: Promise<void> = Promise.resolve();

async function withWriteSlot<T>(work: () => Promise<T>): Promise<T> {
	const previous = writeQueue;
	let release: (() => void) | undefined;

	writeQueue = new Promise<void>(resolve => {
		release = resolve;
	});

	await previous;

	try {
		return await work();
	} finally {
		release?.();
	}
}

async function runWithConcurrency<T>(
	items: T[],
	concurrency: number,
	shouldContinue: () => boolean,
	worker: (item: T) => Promise<void>
): Promise<void> {
	let nextIndex = 0;
	const workers: Array<Promise<void>> = [];

	const runWorker = async () => {
		while (shouldContinue()) {
			const index = nextIndex;
			nextIndex += 1;

			const item = items[index];

			if (item === undefined) {
				return;
			}

			await worker(item);
		}
	};

	for (let index = 0; index < Math.min(concurrency, items.length); index += 1) {
		workers.push(runWorker());
	}

	await Promise.all(workers);
}

function createFingerprint(fileID: string, targetPath: string, strict: boolean): string {
	return `${fileID}\u0000${targetPath}\u0000${strict ? 'strict' : 'fuzzy'}`;
}

function createStoredState(file: DiffApplyFileView): StoredFileState {
	return {
		fingerprint: file.fingerprint,
		source: file.file,
		hunkOutcomes: {},
		selectedHunkIDs: [],
	};
}

function uniquePaths(paths: Array<string | undefined>): string[] {
	return normalizeInteractiveTargetPaths(paths);
}

export function useDiffApplyController(
	parsed: ParsedInteractiveDiff,
	candidatePaths: string[] = EMPTY_PATHS,
	workspaceRoots: string[] = EMPTY_PATHS
): DiffApplyController {
	const [strict, setStrictState] = useState(false);
	const [targetInputs, setTargetInputs] = useState<Record<string, TargetInputState>>({});
	const [states, setStates] = useState<Record<string, StoredFileState>>({});
	const [busy, setBusy] = useState<DiffApplyBusyState | null>(null);

	const mountedRef = useRef(true);
	const operationRef = useRef<ActiveOperation | null>(null);
	const operationSequenceRef = useRef(0);
	const filesRef = useRef<DiffApplyFileView[]>([]);
	const strictRef = useRef(strict);

	const normalizedCandidatePaths = useMemo(() => normalizeInteractiveTargetPaths(candidatePaths), [candidatePaths]);
	const normalizedWorkspaceRoots = useMemo(() => normalizeInteractiveTargetPaths(workspaceRoots), [workspaceRoots]);
	const targetSuggestions = useMemo(
		() => inferInteractiveDiffTargets(parsed.files, normalizedCandidatePaths, normalizedWorkspaceRoots),
		[normalizedCandidatePaths, normalizedWorkspaceRoots, parsed.files]
	);

	const files = useMemo<DiffApplyFileView[]>(
		() =>
			parsed.files.map(file => {
				const suggestion = targetSuggestions.get(file.id);
				const targetInput = targetInputs[file.id];
				const pathInput = targetInput?.source === file ? targetInput.value : '';
				const explicitTargetPath = pathInput.trim();
				const fingerprint = createFingerprint(file.id, explicitTargetPath, strict);
				const stored = states[file.id];
				const state = stored?.fingerprint === fingerprint && stored.source === file ? stored : undefined;
				const reviewOutcome = state?.reviewOutcome;
				const applyOutcome = state?.applyOutcome;
				const effectiveTargetPath = explicitTargetPath;
				const status = applyOutcome?.status ?? reviewOutcome?.status ?? 'pending';
				const canTryHunks =
					(reviewOutcome?.status === 'blocked' || reviewOutcome?.status === 'needs-info') && file.hunks.length > 0;

				return {
					file,
					pathInput,
					targetPath: explicitTargetPath,
					effectiveTargetPath,
					candidates: uniquePaths([...(suggestion?.candidates ?? []), reviewOutcome?.resolvedTargetPath]),
					fingerprint,
					reviewOutcome,
					applyOutcome,
					hunkOutcomes: state?.hunkOutcomes ?? {},
					selectedHunkIDs: state?.selectedHunkIDs ?? [],
					status,
					canApply: reviewOutcome?.status === 'ready' && !applyOutcome,
					canTryHunks,
				};
			}),
		[parsed.files, states, strict, targetInputs, targetSuggestions]
	);

	useLayoutEffect(() => {
		filesRef.current = files;
		strictRef.current = strict;
	}, [files, strict]);

	useEffect(() => {
		mountedRef.current = true;

		return () => {
			mountedRef.current = false;

			if (operationRef.current) {
				operationRef.current.cancelled = true;
			}
		};
	}, []);

	const findCurrentFile = useCallback((fileID: string): DiffApplyFileView | undefined => {
		return filesRef.current.find(file => file.file.id === fileID);
	}, []);

	const isCurrentScope = useCallback((file: DiffApplyFileView): boolean => {
		const current = filesRef.current.find(candidate => candidate.file.id === file.file.id);

		return mountedRef.current && current?.fingerprint === file.fingerprint && current.file === file.file;
	}, []);

	const isOperationCurrent = useCallback(
		(operation: ActiveOperation, file?: DiffApplyFileView): boolean => {
			if (!mountedRef.current || operationRef.current !== operation || operation.cancelled) {
				return false;
			}

			return !file || isCurrentScope(file);
		},
		[isCurrentScope]
	);

	const updateBusy = useCallback(
		(operation: ActiveOperation, update: (current: DiffApplyBusyState) => DiffApplyBusyState): void => {
			if (!mountedRef.current || operationRef.current !== operation) {
				return;
			}

			setBusy(previous => {
				if (!previous || previous.id !== operation.id) {
					return previous;
				}

				return update(previous);
			});
		},
		[]
	);

	const markActive = useCallback(
		(operation: ActiveOperation, fileID: string, active: boolean): void => {
			updateBusy(operation, current => ({
				...current,
				activeFileIDs: active
					? current.activeFileIDs.includes(fileID)
						? current.activeFileIDs
						: [...current.activeFileIDs, fileID]
					: current.activeFileIDs.filter(id => id !== fileID),
			}));
		},
		[updateBusy]
	);

	const markCompleted = useCallback(
		(operation: ActiveOperation): void => {
			updateBusy(operation, current => ({
				...current,
				completed: Math.min(current.total, current.completed + 1),
			}));
		},
		[updateBusy]
	);

	const startOperation = useCallback((kind: DiffApplyOperationKind, total: number): ActiveOperation | undefined => {
		if (operationRef.current || !mountedRef.current) {
			return undefined;
		}

		const operation: ActiveOperation = {
			id: ++operationSequenceRef.current,
			cancelled: false,
		};

		operationRef.current = operation;
		setBusy({
			id: operation.id,
			kind,
			total: Math.max(1, total),
			completed: 0,
			activeFileIDs: [],
			stopping: false,
		});

		return operation;
	}, []);

	const finishOperation = useCallback((operation: ActiveOperation): void => {
		if (operationRef.current !== operation) {
			return;
		}

		operationRef.current = null;

		if (mountedRef.current) {
			setBusy(null);
		}
	}, []);

	const updateFileState = useCallback(
		(file: DiffApplyFileView, update: (current: StoredFileState) => StoredFileState): void => {
			if (!isCurrentScope(file)) {
				return;
			}

			setStates(previous => {
				const existing = previous[file.file.id];
				const current =
					existing?.fingerprint === file.fingerprint && existing.source === file.file
						? existing
						: createStoredState(file);

				return {
					...previous,
					[file.file.id]: update(current),
				};
			});
		},
		[isCurrentScope]
	);

	const clearForReview = useCallback(
		(file: DiffApplyFileView): void => {
			updateFileState(file, previous => ({
				...previous,
				reviewOutcome: undefined,
				applyOutcome: undefined,
				hunkOutcomes: {},
				selectedHunkIDs: [],
			}));
		},
		[updateFileState]
	);

	const setReviewOutcome = useCallback(
		(file: DiffApplyFileView, outcome: DiffApplyOutcome): void => {
			updateFileState(file, previous => ({
				...previous,
				reviewOutcome: outcome,
				applyOutcome: undefined,
				hunkOutcomes: {},
				selectedHunkIDs: [],
			}));
		},
		[updateFileState]
	);

	const setApplyOutcome = useCallback(
		(file: DiffApplyFileView, outcome: DiffApplyOutcome, clearHunks = true): void => {
			updateFileState(file, previous => ({
				...previous,
				applyOutcome: outcome,
				hunkOutcomes: clearHunks ? {} : previous.hunkOutcomes,
				selectedHunkIDs: clearHunks ? [] : previous.selectedHunkIDs,
			}));
		},
		[updateFileState]
	);

	const clearHunkOutcomes = useCallback(
		(file: DiffApplyFileView): void => {
			updateFileState(file, previous => ({
				...previous,
				applyOutcome: undefined,
				hunkOutcomes: {},
				selectedHunkIDs: [],
			}));
		},
		[updateFileState]
	);

	const setHunkOutcome = useCallback(
		(file: DiffApplyFileView, hunk: InteractiveDiffHunk, outcome: DiffApplyOutcome): void => {
			updateFileState(file, previous => {
				const selectedHunkIDs =
					outcome.status === 'ready' && !previous.selectedHunkIDs.includes(hunk.id)
						? [...previous.selectedHunkIDs, hunk.id]
						: previous.selectedHunkIDs;

				return {
					...previous,
					hunkOutcomes: {
						...previous.hunkOutcomes,
						[hunk.id]: outcome,
					},
					selectedHunkIDs,
				};
			});
		},
		[updateFileState]
	);

	const requestOutcome = useCallback(
		async (
			operation: ActiveOperation,
			file: DiffApplyFileView,
			phase: DiffApplyPhase,
			diffText: string,
			requestStrict: boolean
		): Promise<DiffApplyOutcome | undefined> => {
			if (!isOperationCurrent(operation, file)) {
				return undefined;
			}

			markActive(operation, file.file.id, true);

			try {
				/*
				 * The backend fuzzy applier owns target resolution. We only
				 * provide an explicit file target when the UI has one.
				 */
				const fileTargets = file.effectiveTargetPath
					? [
							{
								oldPath: file.file.oldPath,
								newPath: file.file.newPath,
								targetPath: file.effectiveTargetPath,
							},
						]
					: undefined;

				const output = await aggregateAPI.applyUnifiedDiff({
					diffText,
					dryRun: phase === 'dry-run',
					strict: requestStrict,
					fileTargets,
					candidatePaths: file.candidates.length > 0 ? file.candidates : undefined,
				});

				return interpretScopedApplyResult(output, phase);
			} catch (error) {
				const message = getErrorMessage(error, 'Unexpected diff request error.');

				return createDiffApplyOutcome(
					phase,
					'blocked',
					phase === 'apply' ? `The apply request failed. Review again before retrying. ${message}` : message
				);
			} finally {
				markActive(operation, file.file.id, false);
				markCompleted(operation);
			}
		},
		[isOperationCurrent, markActive, markCompleted]
	);

	const reviewFiles = useCallback(
		async (fileIDs?: string[]): Promise<void> => {
			const selected = fileIDs
				? filesRef.current.filter(file => fileIDs.includes(file.file.id))
				: [...filesRef.current];

			if (selected.length === 0) {
				return;
			}

			const operation = startOperation('review', selected.length);

			if (!operation) {
				return;
			}

			const requestStrict = strictRef.current;

			try {
				for (const file of selected) {
					clearForReview(file);
				}

				await runWithConcurrency(
					selected,
					REVIEW_CONCURRENCY,
					() => isOperationCurrent(operation),
					async file => {
						let diffText: string;

						try {
							diffText = buildInteractiveDiffRequest(file.file, file.effectiveTargetPath);
						} catch (error) {
							setReviewOutcome(
								file,
								createDiffApplyOutcome(
									'dry-run',
									'blocked',
									getErrorMessage(error, 'Could not isolate this file section.')
								)
							);
							markCompleted(operation);
							return;
						}

						const outcome = await requestOutcome(operation, file, 'dry-run', diffText, requestStrict);

						if (outcome && isOperationCurrent(operation, file)) {
							setReviewOutcome(file, outcome);
						}
					}
				);
			} finally {
				finishOperation(operation);
			}
		},
		[
			clearForReview,
			finishOperation,
			isOperationCurrent,
			markCompleted,
			requestOutcome,
			setReviewOutcome,
			startOperation,
		]
	);

	const applyFiles = useCallback(
		async (fileIDs?: string[]): Promise<void> => {
			const selected = fileIDs
				? filesRef.current.filter(file => fileIDs.includes(file.file.id) && file.canApply)
				: filesRef.current.filter(file => file.canApply);

			if (selected.length === 0) {
				return;
			}

			const operation = startOperation('apply', selected.length);

			if (!operation) {
				return;
			}

			const requestStrict = strictRef.current;

			try {
				for (const file of selected) {
					if (!isOperationCurrent(operation)) {
						break;
					}

					await withWriteSlot(async () => {
						if (!isOperationCurrent(operation, file)) {
							return;
						}

						let diffText: string;

						try {
							/*
							 * This is deliberately a direct apply. The user
							 * explicitly reviewed first, and backend apply
							 * semantics decide whether the current file still
							 * accepts this fuzzy patch.
							 */
							diffText = buildInteractiveDiffRequest(file.file, file.effectiveTargetPath);
						} catch (error) {
							setApplyOutcome(
								file,
								createDiffApplyOutcome(
									'apply',
									'blocked',
									getErrorMessage(error, 'Could not isolate this file section.')
								)
							);
							markCompleted(operation);
							return;
						}

						const outcome = await requestOutcome(operation, file, 'apply', diffText, requestStrict);

						/*
						 * A write may complete after Stop. Keep the result if
						 * the source/target scope still matches.
						 */
						if (outcome && isCurrentScope(file)) {
							setApplyOutcome(file, outcome);
						}
					});
				}
			} finally {
				finishOperation(operation);
			}
		},
		[
			finishOperation,
			isCurrentScope,
			isOperationCurrent,
			markCompleted,
			requestOutcome,
			setApplyOutcome,
			startOperation,
		]
	);

	const reviewHunks = useCallback(
		async (fileID: string): Promise<void> => {
			const file = findCurrentFile(fileID);

			if (!file || !file.canTryHunks || file.file.hunks.length === 0) {
				return;
			}

			if (file.file.hunks.length > MAX_INTERACTIVE_HUNK_PROBES) {
				setApplyOutcome(
					file,
					createDiffApplyOutcome(
						'dry-run',
						'blocked',
						`This file has ${file.file.hunks.length} hunks. Individual hunk review is limited to ${MAX_INTERACTIVE_HUNK_PROBES} hunks.`
					),
					false
				);
				return;
			}

			const operation = startOperation('review-hunks', file.file.hunks.length);

			if (!operation) {
				return;
			}

			const requestStrict = strictRef.current;
			clearHunkOutcomes(file);

			try {
				await runWithConcurrency(
					file.file.hunks,
					HUNK_REVIEW_CONCURRENCY,
					() => isOperationCurrent(operation, file),
					async hunk => {
						let diffText: string;

						try {
							diffText = buildInteractiveDiffRequest(file.file, file.effectiveTargetPath, [hunk.id]);
						} catch (error) {
							setHunkOutcome(
								file,
								hunk,
								createDiffApplyOutcome('dry-run', 'blocked', getErrorMessage(error, 'Could not isolate this hunk.'))
							);
							markCompleted(operation);
							return;
						}

						const outcome = await requestOutcome(operation, file, 'dry-run', diffText, requestStrict);

						if (outcome && isOperationCurrent(operation, file)) {
							setHunkOutcome(file, hunk, outcome);
						}
					}
				);
			} finally {
				finishOperation(operation);
			}
		},
		[
			clearHunkOutcomes,
			findCurrentFile,
			finishOperation,
			isOperationCurrent,
			markCompleted,
			requestOutcome,
			setApplyOutcome,
			setHunkOutcome,
			startOperation,
		]
	);

	const applySelectedHunks = useCallback(
		async (fileID: string): Promise<void> => {
			const file = findCurrentFile(fileID);

			if (!file) {
				return;
			}

			const selectedHunkIDs = file.file.hunks
				.filter(hunk => file.selectedHunkIDs.includes(hunk.id) && file.hunkOutcomes[hunk.id]?.status === 'ready')
				.map(hunk => hunk.id);

			if (selectedHunkIDs.length === 0) {
				return;
			}

			const operation = startOperation('apply-hunks', 1);

			if (!operation) {
				return;
			}

			const requestStrict = strictRef.current;

			try {
				await withWriteSlot(async () => {
					if (!isOperationCurrent(operation, file)) {
						return;
					}

					let diffText: string;

					try {
						/*
						 * Do not perform another frontend hunk validation or
						 * coordinate rewrite here. The selected exact source
						 * is sent to the backend fuzzy applier.
						 */
						diffText = buildInteractiveDiffRequest(file.file, file.effectiveTargetPath, selectedHunkIDs);
					} catch (error) {
						setApplyOutcome(
							file,
							createDiffApplyOutcome(
								'apply',
								'blocked',
								getErrorMessage(error, 'Could not build the selected-hunk request.')
							)
						);
						markCompleted(operation);
						return;
					}

					const outcome = await requestOutcome(operation, file, 'apply', diffText, requestStrict);

					if (!outcome || !isCurrentScope(file)) {
						return;
					}

					const finalOutcome =
						(outcome.status === 'applied' || outcome.status === 'already-applied') &&
						selectedHunkIDs.length < file.file.hunks.length
							? {
									...outcome,
									status: 'partial' as const,
									message:
										`${selectedHunkIDs.length} of ${file.file.hunks.length} hunks ` +
										(outcome.status === 'already-applied' ? 'are already present.' : 'were applied.') +
										' Remaining hunks were not submitted.',
								}
							: outcome;

					setApplyOutcome(file, finalOutcome);
				});
			} finally {
				finishOperation(operation);
			}
		},
		[
			findCurrentFile,
			finishOperation,
			isCurrentScope,
			isOperationCurrent,
			markCompleted,
			requestOutcome,
			setApplyOutcome,
			startOperation,
		]
	);

	const setTargetPath = useCallback(
		(fileID: string, value: string): void => {
			if (operationRef.current) {
				return;
			}

			const file = findCurrentFile(fileID);
			if (!file) {
				return;
			}

			setStates(previous => {
				if (!previous[fileID]) {
					return previous;
				}

				const next = { ...previous };
				Reflect.deleteProperty(next, fileID);
				return next;
			});

			setTargetInputs(previous => ({
				...previous,
				[fileID]: {
					source: file.file,
					value,
				},
			}));
		},
		[findCurrentFile]
	);

	const setStrict = useCallback((value: boolean): void => {
		if (operationRef.current || strictRef.current === value) {
			return;
		}

		strictRef.current = value;
		setStates({});
		setStrictState(value);
	}, []);

	const setHunkSelected = useCallback(
		(fileID: string, hunkID: string, selected: boolean): void => {
			if (operationRef.current) {
				return;
			}

			const file = findCurrentFile(fileID);

			if (!file || file.hunkOutcomes[hunkID]?.status !== 'ready') {
				return;
			}

			updateFileState(file, previous => {
				const selectedHunkIDs = new Set(previous.selectedHunkIDs);

				if (selected) {
					selectedHunkIDs.add(hunkID);
				} else {
					selectedHunkIDs.delete(hunkID);
				}

				return {
					...previous,
					selectedHunkIDs: [...selectedHunkIDs],
				};
			});
		},
		[findCurrentFile, updateFileState]
	);

	const cancel = useCallback((): void => {
		const operation = operationRef.current;

		if (!operation) {
			return;
		}

		operation.cancelled = true;

		updateBusy(operation, current => ({
			...current,
			stopping: true,
		}));
	}, [updateBusy]);

	const counts = useMemo(() => {
		const next = {
			reviewed: 0,
			ready: 0,
			applied: 0,
			alreadyApplied: 0,
			blocked: 0,
			needsInfo: 0,
			pending: 0,
			partial: 0,
		};

		for (const file of files) {
			if (file.reviewOutcome) {
				next.reviewed += 1;
			}

			switch (file.status) {
				case 'ready':
					next.ready += 1;
					break;
				case 'applied':
					next.applied += 1;
					break;
				case 'already-applied':
					next.alreadyApplied += 1;
					break;
				case 'blocked':
					next.blocked += 1;
					break;
				case 'needs-info':
					next.needsInfo += 1;
					break;
				case 'partial':
					next.partial += 1;
					break;
				default:
					next.pending += 1;
					break;
			}
		}

		return next;
	}, [files]);

	const reviewAll = useCallback((): Promise<void> => reviewFiles(), [reviewFiles]);
	const reviewFile = useCallback((fileID: string): Promise<void> => reviewFiles([fileID]), [reviewFiles]);
	const applyReady = useCallback((): Promise<void> => applyFiles(), [applyFiles]);
	const applyFile = useCallback((fileID: string): Promise<void> => applyFiles([fileID]), [applyFiles]);

	return {
		files,
		patchDiagnostics: parsed.diagnostics,
		strict,
		busy,
		hasReviewResult: counts.reviewed > 0,
		reviewedFileCount: counts.reviewed,
		readyFileCount: counts.ready,
		appliedFileCount: counts.applied,
		alreadyAppliedFileCount: counts.alreadyApplied,
		blockedFileCount: counts.blocked,
		needsInfoFileCount: counts.needsInfo,
		pendingFileCount: counts.pending,
		partialFileCount: counts.partial,
		setStrict,
		setTargetPath,
		setHunkSelected,
		cancel,
		reviewAll,
		reviewFile,
		applyReady,
		applyFile,
		reviewHunks,
		applySelectedHunks,
	};
}
