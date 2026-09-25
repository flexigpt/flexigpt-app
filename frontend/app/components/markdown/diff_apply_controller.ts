import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';

import type { ApplyUnifiedDiffDiagnostic } from '@/spec/unified_diff';
import { ApplyUnifiedDiffDiagnosticLevel } from '@/spec/unified_diff';

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
	normalizeInteractiveTargetPath,
	normalizeInteractiveTargetPaths,
} from '@/components/markdown/diff_apply_model';

const DRY_RUN_CONCURRENCY = 3;
const HUNK_DRY_RUN_CONCURRENCY = 2;

type DiffApplyOperationKind = 'check' | 'apply' | 'check-hunks' | 'apply-hunks';

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
	targetPath: string;
	candidates: string[];
	fingerprint: string;
	fullOutcome?: DiffApplyOutcome;
	partialOutcome?: DiffApplyOutcome;
	hunkOutcomes: Record<string, DiffApplyOutcome>;
	selectedHunkIDs: string[];
	status: DiffApplyOutcomeStatus | 'pending';
	canCheck: boolean;
	hasInvalidPathInput: boolean;
}

export interface DiffApplyController {
	files: DiffApplyFileView[];
	patchDiagnostics: ApplyUnifiedDiffDiagnostic[];
	patchHasErrors: boolean;
	strict: boolean;
	busy: DiffApplyBusyState | null;
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
	checkAll: () => Promise<void>;
	checkFile: (fileID: string) => Promise<void>;
	applyReady: () => Promise<void>;
	applyFile: (fileID: string) => Promise<void>;
	checkHunks: (fileID: string) => Promise<void>;
	applySelectedHunks: (fileID: string) => Promise<void>;
}

interface StoredFileState {
	fingerprint: string;
	fullOutcome?: DiffApplyOutcome;
	partialOutcome?: DiffApplyOutcome;
	hunkOutcomes: Record<string, DiffApplyOutcome>;
	selectedHunkIDs: string[];
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

	const next = async () => {
		while (shouldContinue()) {
			const currentIndex = nextIndex;
			nextIndex += 1;

			const item = items[currentIndex];

			if (item === undefined) {
				return;
			}

			await worker(item);
		}
	};

	for (let index = 0; index < Math.min(concurrency, items.length); index += 1) {
		workers.push(next());
	}

	await Promise.all(workers);
}

function createFingerprint(fileID: string, targetPath: string, strict: boolean): string {
	return `${fileID}\u0000${targetPath}\u0000${strict ? 'strict' : 'fuzzy'}`;
}

function createStoredFileState(fingerprint: string): StoredFileState {
	return {
		fingerprint,
		hunkOutcomes: {},
		selectedHunkIDs: [],
	};
}

function hasErrorDiagnostics(diagnostics: ApplyUnifiedDiffDiagnostic[]): boolean {
	return diagnostics.some(diagnostic => diagnostic.level === ApplyUnifiedDiffDiagnosticLevel.Error);
}

export function useDiffApplyController(
	parsed: ParsedInteractiveDiff,
	candidatePaths: string[] = [],
	workspaceRoots: string[] = []
): DiffApplyController {
	const [strict, setStrictState] = useState(false);
	const [targetInputs, setTargetInputs] = useState<Record<string, string>>({});
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

	const patchHasErrors = useMemo(() => hasErrorDiagnostics(parsed.diagnostics), [parsed.diagnostics]);

	const files = useMemo<DiffApplyFileView[]>(
		() =>
			parsed.files.map(file => {
				const suggestion = targetSuggestions.get(file.id);
				const pathInput = targetInputs[file.id] ?? suggestion?.targetPath ?? '';
				const targetPath = normalizeInteractiveTargetPath(pathInput);
				const fingerprint = createFingerprint(file.id, targetPath, strict);
				const stored = states[file.id];
				const state = stored?.fingerprint === fingerprint ? stored : undefined;
				const fullOutcome = state?.fullOutcome;
				const partialOutcome = state?.partialOutcome;

				const status: DiffApplyFileView['status'] =
					patchHasErrors || !file.canApplyWhole
						? 'blocked'
						: !targetPath
							? 'needs-info'
							: (fullOutcome?.status ?? partialOutcome?.status ?? 'pending');

				return {
					file,
					pathInput,
					targetPath,
					candidates: suggestion?.candidates ?? [],
					fingerprint,
					fullOutcome,
					partialOutcome,
					hunkOutcomes: state?.hunkOutcomes ?? {},
					selectedHunkIDs: state?.selectedHunkIDs ?? [],
					status,
					canCheck: !patchHasErrors && file.canApplyWhole && !!targetPath,
					hasInvalidPathInput: !!pathInput.trim() && !targetPath,
				};
			}),
		[patchHasErrors, parsed.files, states, strict, targetInputs, targetSuggestions]
	);

	useLayoutEffect(() => {
		filesRef.current = files;
	}, [files]);

	useEffect(() => {
		mountedRef.current = true;

		return () => {
			mountedRef.current = false;

			if (operationRef.current) {
				operationRef.current.cancelled = true;
			}
		};
	}, []);

	useEffect(() => {
		strictRef.current = strict;
	}, [strict]);

	const findCurrentFile = useCallback((fileID: string): DiffApplyFileView | undefined => {
		return filesRef.current.find(file => file.file.id === fileID);
	}, []);

	const isCurrentScope = useCallback((file: DiffApplyFileView): boolean => {
		const current = filesRef.current.find(candidate => candidate.file.id === file.file.id);
		return current?.fingerprint === file.fingerprint;
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
		(operation: ActiveOperation, count = 1): void => {
			updateBusy(operation, current => ({
				...current,
				completed: Math.min(current.total, current.completed + count),
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
				const current = existing?.fingerprint === file.fingerprint ? existing : createStoredFileState(file.fingerprint);

				return {
					...previous,
					[file.file.id]: update(current),
				};
			});
		},
		[isCurrentScope]
	);

	const setFullOutcome = useCallback(
		(file: DiffApplyFileView, outcome: DiffApplyOutcome): void => {
			updateFileState(file, previous => ({
				...previous,
				fullOutcome: outcome,
				partialOutcome: undefined,
				hunkOutcomes: {},
				selectedHunkIDs: [],
			}));
		},
		[updateFileState]
	);

	const clearHunkOutcomes = useCallback(
		(file: DiffApplyFileView): void => {
			updateFileState(file, previous => ({
				...previous,
				partialOutcome: undefined,
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

	const setPartialOutcome = useCallback(
		(file: DiffApplyFileView, outcome: DiffApplyOutcome): void => {
			updateFileState(file, previous => ({
				...previous,
				partialOutcome: outcome,
			}));
		},
		[updateFileState]
	);

	const setPartialTerminalOutcome = useCallback(
		(file: DiffApplyFileView, outcome: DiffApplyOutcome): void => {
			updateFileState(file, previous => ({
				...previous,
				fullOutcome: outcome,
				partialOutcome: outcome,
				hunkOutcomes: {},
				selectedHunkIDs: [],
			}));
		},
		[updateFileState]
	);

	const pinTargetPath = useCallback((file: DiffApplyFileView): void => {
		setTargetInputs(previous =>
			previous[file.file.id] === undefined
				? {
						...previous,
						[file.file.id]: file.targetPath,
					}
				: previous
		);
	}, []);

	const outcomeForFileProblem = useCallback(
		(file: DiffApplyFileView, phase: DiffApplyPhase): DiffApplyOutcome => {
			if (patchHasErrors) {
				return createDiffApplyOutcome(
					phase,
					'blocked',
					'The patch has top-level parser errors that must be resolved first.'
				);
			}

			if (!file.file.canApplyWhole) {
				return createDiffApplyOutcome(phase, 'blocked', 'This file section has parser errors and cannot be sent.');
			}

			if (!file.targetPath) {
				return createDiffApplyOutcome(
					phase,
					'needs-info',
					file.hasInvalidPathInput
						? 'Target path must be an absolute path without dot segments.'
						: 'Choose or enter an absolute target file path.'
				);
			}

			return createDiffApplyOutcome(phase, 'blocked', 'This file section is not ready for this operation.');
		},
		[patchHasErrors]
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
				const output = await aggregateAPI.applyUnifiedDiff({
					diffText,
					dryRun: phase === 'dry-run',
					strict: requestStrict,
					fileTargets: [
						{
							oldPath: file.file.oldPath,
							newPath: file.file.newPath,
							targetPath: file.targetPath,
						},
					],
					candidatePaths: [file.targetPath],
				});

				return interpretScopedApplyResult(output, phase, file.targetPath);
			} catch (error) {
				return createDiffApplyOutcome(phase, 'blocked', getErrorMessage(error, 'Unexpected diff request error.'));
			} finally {
				markActive(operation, file.file.id, false);
				markCompleted(operation);
			}
		},
		[isOperationCurrent, markActive, markCompleted]
	);

	const buildRequest = useCallback((file: DiffApplyFileView, selectedHunkIDs?: string[]): string => {
		return buildInteractiveDiffRequest(file.file, file.targetPath, selectedHunkIDs);
	}, []);

	const checkFiles = useCallback(
		async (fileIDs?: string[]): Promise<void> => {
			const selected = fileIDs
				? filesRef.current.filter(file => fileIDs.includes(file.file.id))
				: [...filesRef.current];

			if (selected.length === 0) {
				return;
			}

			const operation = startOperation('check', selected.length);

			if (!operation) {
				return;
			}

			const requestStrict = strictRef.current;
			const runnable: DiffApplyFileView[] = [];

			try {
				for (const file of selected) {
					if (!file.canCheck) {
						setFullOutcome(file, outcomeForFileProblem(file, 'dry-run'));
						markCompleted(operation);
						continue;
					}

					pinTargetPath(file);
					runnable.push(file);
				}

				await runWithConcurrency(
					runnable,
					DRY_RUN_CONCURRENCY,
					() => isOperationCurrent(operation),
					async file => {
						let diffText: string;

						try {
							diffText = buildRequest(file);
						} catch (error) {
							setFullOutcome(
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
							setFullOutcome(file, outcome);
						}
					}
				);
			} finally {
				finishOperation(operation);
			}
		},
		[
			buildRequest,
			finishOperation,
			isOperationCurrent,
			markCompleted,
			outcomeForFileProblem,
			pinTargetPath,
			requestOutcome,
			setFullOutcome,
			startOperation,
		]
	);

	const applyFiles = useCallback(
		async (fileIDs?: string[], onlyReady = false): Promise<void> => {
			const selected = fileIDs
				? filesRef.current.filter(file => fileIDs.includes(file.file.id))
				: filesRef.current.filter(file => !onlyReady || file.fullOutcome?.status === 'ready');

			if (selected.length === 0) {
				return;
			}

			const operation = startOperation('apply', selected.length * 2);

			if (!operation) {
				return;
			}

			const requestStrict = strictRef.current;

			try {
				for (const file of selected) {
					if (!isOperationCurrent(operation)) {
						break;
					}

					if (!file.canCheck) {
						setFullOutcome(file, outcomeForFileProblem(file, 'apply'));
						markCompleted(operation, 2);
						continue;
					}

					await withWriteSlot(async () => {
						if (!isOperationCurrent(operation, file)) {
							return;
						}

						pinTargetPath(file);

						let diffText: string;

						try {
							diffText = buildRequest(file);
						} catch (error) {
							setFullOutcome(
								file,
								createDiffApplyOutcome(
									'apply',
									'blocked',
									getErrorMessage(error, 'Could not isolate this file section.')
								)
							);
							markCompleted(operation, 2);
							return;
						}

						const dryRunOutcome = await requestOutcome(operation, file, 'dry-run', diffText, requestStrict);

						if (dryRunOutcome && isCurrentScope(file)) {
							setFullOutcome(file, dryRunOutcome);
						}

						if (!dryRunOutcome) {
							return;
						}

						if (dryRunOutcome.status === 'already-applied') {
							markCompleted(operation);
							return;
						}

						if (dryRunOutcome.status !== 'ready' || !isOperationCurrent(operation, file)) {
							markCompleted(operation);
							return;
						}

						const applyOutcome = await requestOutcome(operation, file, 'apply', diffText, requestStrict);

						// A write can finish after Stop was pressed. Record it
						// when the target is still current so the UI remains truthful.
						if (applyOutcome && isCurrentScope(file)) {
							setFullOutcome(file, applyOutcome);
						}
					});
				}
			} finally {
				finishOperation(operation);
			}
		},
		[
			buildRequest,
			finishOperation,
			isCurrentScope,
			isOperationCurrent,
			markCompleted,
			outcomeForFileProblem,
			pinTargetPath,
			requestOutcome,
			setFullOutcome,
			startOperation,
		]
	);

	const checkHunks = useCallback(
		async (fileID: string): Promise<void> => {
			const file = findCurrentFile(fileID);

			if (!file) {
				return;
			}

			if (!file.canCheck) {
				setPartialOutcome(file, outcomeForFileProblem(file, 'dry-run'));
				return;
			}

			if (!file.file.canApplyPartial) {
				setPartialOutcome(
					file,
					createDiffApplyOutcome(
						'dry-run',
						'blocked',
						'Partial application is available only for ordinary, non-overlapping text modifications.'
					)
				);
				return;
			}

			if (file.file.hunks.length > MAX_INTERACTIVE_HUNK_PROBES) {
				setPartialOutcome(
					file,
					createDiffApplyOutcome(
						'dry-run',
						'blocked',
						`This file has ${file.file.hunks.length} hunks. Per-hunk checks are limited to ${MAX_INTERACTIVE_HUNK_PROBES} hunks.`
					)
				);
				return;
			}

			const operation = startOperation('check-hunks', file.file.hunks.length);

			if (!operation) {
				return;
			}

			const requestStrict = strictRef.current;
			pinTargetPath(file);
			clearHunkOutcomes(file);

			try {
				await runWithConcurrency(
					file.file.hunks,
					HUNK_DRY_RUN_CONCURRENCY,
					() => isOperationCurrent(operation, file),
					async hunk => {
						let diffText: string;

						try {
							diffText = buildRequest(file, [hunk.id]);
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
			buildRequest,
			clearHunkOutcomes,
			findCurrentFile,
			finishOperation,
			isOperationCurrent,
			markCompleted,
			outcomeForFileProblem,
			pinTargetPath,
			requestOutcome,
			setHunkOutcome,
			setPartialOutcome,
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
				setPartialOutcome(
					file,
					createDiffApplyOutcome('dry-run', 'needs-info', 'Select at least one hunk that passed its dry run.')
				);
				return;
			}

			const operation = startOperation('apply-hunks', 2);

			if (!operation) {
				return;
			}

			const requestStrict = strictRef.current;

			try {
				await withWriteSlot(async () => {
					if (!isOperationCurrent(operation, file)) {
						return;
					}

					pinTargetPath(file);

					let diffText: string;

					try {
						diffText = buildRequest(file, selectedHunkIDs);
					} catch (error) {
						setPartialOutcome(
							file,
							createDiffApplyOutcome(
								'apply',
								'blocked',
								getErrorMessage(error, 'Could not construct the selected-hunk patch.')
							)
						);
						markCompleted(operation, 2);
						return;
					}

					const dryRunOutcome = await requestOutcome(operation, file, 'dry-run', diffText, requestStrict);

					if (dryRunOutcome && isCurrentScope(file)) {
						setPartialOutcome(file, dryRunOutcome);
					}

					if (!dryRunOutcome) {
						return;
					}

					const toPartialOutcome = (outcome: DiffApplyOutcome): DiffApplyOutcome => {
						if (
							selectedHunkIDs.length === file.file.hunks.length ||
							(outcome.status !== 'applied' && outcome.status !== 'already-applied')
						) {
							return outcome;
						}

						return {
							...outcome,
							status: 'partial',
							message:
								`${selectedHunkIDs.length} of ${file.file.hunks.length} hunks ` +
								(outcome.status === 'already-applied' ? 'are already present.' : 'were applied.') +
								' The remaining hunks were not submitted and must be checked again.',
						};
					};

					if (dryRunOutcome.status === 'already-applied') {
						setPartialTerminalOutcome(file, toPartialOutcome(dryRunOutcome));
						markCompleted(operation);
						return;
					}

					if (dryRunOutcome.status !== 'ready' || !isOperationCurrent(operation, file)) {
						markCompleted(operation);
						return;
					}

					const applyOutcome = await requestOutcome(operation, file, 'apply', diffText, requestStrict);

					if (applyOutcome && isCurrentScope(file)) {
						setPartialTerminalOutcome(file, toPartialOutcome(applyOutcome));
					}
				});
			} finally {
				finishOperation(operation);
			}
		},
		[
			buildRequest,
			findCurrentFile,
			finishOperation,
			isCurrentScope,
			isOperationCurrent,
			markCompleted,
			pinTargetPath,
			requestOutcome,
			setPartialOutcome,
			setPartialTerminalOutcome,
			startOperation,
		]
	);

	const setTargetPath = useCallback((fileID: string, value: string): void => {
		if (operationRef.current) {
			return;
		}

		setTargetInputs(previous => ({
			...previous,
			[fileID]: value,
		}));
	}, []);

	const setStrict = useCallback((value: boolean): void => {
		if (operationRef.current) {
			return;
		}

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

	const statusCounts = useMemo(() => {
		const counts = {
			ready: 0,
			applied: 0,
			alreadyApplied: 0,
			blocked: 0,
			needsInfo: 0,
			pending: 0,
			partial: 0,
		};

		for (const file of files) {
			switch (file.status) {
				case 'ready':
					counts.ready += 1;
					break;
				case 'applied':
					counts.applied += 1;
					break;
				case 'already-applied':
					counts.alreadyApplied += 1;
					break;
				case 'blocked':
					counts.blocked += 1;
					break;
				case 'needs-info':
					counts.needsInfo += 1;
					break;
				case 'partial':
					counts.partial += 1;
					break;
				default:
					counts.pending += 1;
					break;
			}
		}

		return counts;
	}, [files]);

	return {
		files,
		patchDiagnostics: parsed.diagnostics,
		patchHasErrors,
		strict,
		busy,
		readyFileCount: statusCounts.ready,
		appliedFileCount: statusCounts.applied,
		alreadyAppliedFileCount: statusCounts.alreadyApplied,
		blockedFileCount: statusCounts.blocked,
		needsInfoFileCount: statusCounts.needsInfo,
		pendingFileCount: statusCounts.pending,
		partialFileCount: statusCounts.partial,
		setStrict,
		setTargetPath,
		setHunkSelected,
		cancel,
		checkAll: () => checkFiles(),
		checkFile: fileID => checkFiles([fileID]),
		applyReady: () => applyFiles(undefined, true),
		applyFile: fileID => applyFiles([fileID]),
		checkHunks,
		applySelectedHunks,
	};
}
