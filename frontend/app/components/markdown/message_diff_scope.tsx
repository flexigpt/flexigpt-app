// oxlint-disable react/only-export-components
import type { ReactNode } from 'react';
import { createContext, memo, useContext, useEffect, useLayoutEffect, useMemo, useSyncExternalStore } from 'react';

import { getErrorMessage } from '@/lib/error_utils';

import type { DiffApplyController, DiffApplyFileView } from '@/components/markdown/diff_apply_controller';
import type { InteractiveDiffFile, ParsedInteractiveDiff } from '@/components/markdown/diff_apply_model';
import type { MessageDiffBlockSource } from '@/components/markdown/message_diff_inventory';
import { useDiffApplyController } from '@/components/markdown/diff_apply_controller';
import { DiffApplyModal } from '@/components/markdown/diff_apply_modal';
import { isDiffApplyOutcomeComplete, parseInteractiveDiff } from '@/components/markdown/diff_apply_model';

interface MessageDiffBlock extends MessageDiffBlockSource {
	parsed: ParsedInteractiveDiff;
}

type OperationKind = NonNullable<DiffApplyController['busy']>['kind'];

interface MessageDiffOperation {
	kind: OperationKind;
	total: number;
	completed: number;
	activeBlockID?: string;
	activeFileID?: string;
	stopping: boolean;
}

interface MessageDiffDetails {
	blockID: string;
	fileID?: string;
}

export interface MessageDiffSnapshot {
	blocks: MessageDiffBlock[];
	activeBlockIDs: string[];
	controllers: ReadonlyMap<string, DiffApplyController>;
	operation: MessageDiffOperation | null;
	details: MessageDiffDetails | null;
	targetsReady: boolean;
	isBusy: boolean;
	notice?: string;
}

export interface MessageDiffRow {
	blockID: string;
	file: InteractiveDiffFile;
	view?: DiffApplyFileView;
}

export interface MessageDiffSummary {
	total: number;
	eligible: number;
	completed: number;
	ready: number;
	needsInfo: number;
	blocked: number;
	partial: number;
	pending: number;
	unresolved: number;
}

interface MessageDiffTask {
	blockID: string;
	fileID?: string;
	total: number;
	run: (controller: DiffApplyController) => Promise<void>;
}

interface OperationToken {
	cancelled: boolean;
}

const EMPTY_SNAPSHOT: MessageDiffSnapshot = {
	blocks: [],
	activeBlockIDs: [],
	controllers: new Map(),
	operation: null,
	details: null,
	targetsReady: false,
	isBusy: false,
};

const subscribeToNothing = () => () => {};
const getEmptySnapshot = () => EMPTY_SNAPSHOT;

export class MessageDiffStore {
	private snapshot: MessageDiffSnapshot;
	private listeners = new Set<() => void>();
	private controllerWaiters = new Map<string, Array<(controller: DiffApplyController | undefined) => void>>();
	private operationToken: OperationToken | null = null;
	private disposed = false;

	public constructor(isBusy: boolean, targetsReady: boolean) {
		this.snapshot = {
			...EMPTY_SNAPSHOT,
			blocks: [],
			activeBlockIDs: [],
			controllers: new Map(),
			isBusy,
			targetsReady,
		};
	}

	public getSnapshot = (): MessageDiffSnapshot => this.snapshot;

	public subscribe = (listener: () => void): (() => void) => {
		this.listeners.add(listener);
		return () => {
			this.listeners.delete(listener);
		};
	};

	private update(update: Partial<MessageDiffSnapshot>): void {
		this.snapshot = { ...this.snapshot, ...update };

		for (const listener of this.listeners) {
			listener();
		}
	}

	public resume = (): void => {
		this.disposed = false;
	};

	public dispose = (): void => {
		this.disposed = true;
		this.cancel();

		for (const waiters of this.controllerWaiters.values()) {
			for (const resolve of waiters) {
				resolve(undefined);
			}
		}
		this.controllerWaiters.clear();
	};

	public configure = (isBusy: boolean, targetsReady: boolean): void => {
		if (this.snapshot.isBusy === isBusy && this.snapshot.targetsReady === targetsReady) {
			return;
		}

		this.update({ isBusy, targetsReady });

		if (isBusy && this.operationToken) {
			this.cancel();
		}
	};

	public setSources = (sources: MessageDiffBlockSource[]): void => {
		const previous = this.snapshot.blocks;
		const unchanged =
			previous.length === sources.length &&
			previous.every(
				(block, index) =>
					block.id === sources[index].id &&
					block.language === sources[index].language &&
					block.diffText === sources[index].diffText
			);

		if (unchanged) {
			return;
		}

		// The owning scope is replaced when the message source changes.
		// Parsing here creates stable file identities for all entry points.
		const blocks = sources.map(source => ({
			...source,
			parsed: parseInteractiveDiff(source.diffText, source.language),
		}));

		this.update({ blocks });
	};

	public publishController = (blockID: string, controller: DiffApplyController): void => {
		if (this.snapshot.controllers.get(blockID) === controller) {
			return;
		}

		const controllers = new Map<string, DiffApplyController>([
			...this.snapshot.controllers.entries(),
			[blockID, controller],
		]);
		this.update({ controllers });

		const waiters = this.controllerWaiters.get(blockID);
		if (waiters) {
			this.controllerWaiters.delete(blockID);
			for (const resolve of waiters) {
				resolve(controller);
			}
		}
	};

	private ensureController(blockID: string): Promise<DiffApplyController | undefined> {
		if (this.disposed || !this.snapshot.blocks.some(block => block.id === blockID)) {
			return Promise.resolve(undefined);
		}

		const existing = this.snapshot.controllers.get(blockID);
		if (existing) {
			return Promise.resolve(existing);
		}

		const pending = new Promise<DiffApplyController | undefined>(resolve => {
			const waiters = this.controllerWaiters.get(blockID) ?? [];
			waiters.push(resolve);
			this.controllerWaiters.set(blockID, waiters);
		});

		if (!this.snapshot.activeBlockIDs.includes(blockID)) {
			this.update({ activeBlockIDs: [...this.snapshot.activeBlockIDs, blockID] });
		}

		return pending;
	}

	private canMutate(): boolean {
		return (
			!this.disposed &&
			!this.snapshot.isBusy &&
			this.snapshot.targetsReady &&
			!this.operationToken &&
			![...this.snapshot.controllers.values()].some(controller => controller.busy !== null)
		);
	}

	private async runTasks(kind: OperationKind, tasks: MessageDiffTask[]): Promise<void> {
		if (!this.canMutate() || tasks.length === 0) {
			return;
		}

		// Claim synchronously, before waiting for a lazy controller mount.
		const token: OperationToken = { cancelled: false };
		this.operationToken = token;
		this.update({
			notice: undefined,
			operation: {
				kind,
				total: tasks.reduce((total, task) => total + task.total, 0),
				completed: 0,
				stopping: false,
			},
		});

		try {
			for (const task of tasks) {
				if (this.disposed || token.cancelled) {
					break;
				}

				const operation = this.snapshot.operation;
				if (!operation) {
					break;
				}

				this.update({
					operation: {
						...operation,
						activeBlockID: task.blockID,
						activeFileID: task.fileID,
					},
				});

				const mountedController = await this.ensureController(task.blockID);
				if (!mountedController || this.disposed || token.cancelled) {
					break;
				}

				const controller = this.snapshot.controllers.get(task.blockID) ?? mountedController;
				await task.run(controller);

				const current = this.snapshot.operation;
				if (current && this.operationToken === token) {
					this.update({
						operation: {
							...current,
							completed: Math.min(current.total, current.completed + task.total),
							activeBlockID: undefined,
							activeFileID: undefined,
						},
					});
				}
			}
		} catch (error) {
			if (!this.disposed) {
				this.update({ notice: getErrorMessage(error, 'The diff operation could not be completed.') });
			}
		} finally {
			if (this.operationToken === token) {
				this.operationToken = null;
				this.update({ operation: null });
			}
		}
	}

	private reviewTask(blockID: string, fileID?: string): MessageDiffTask | undefined {
		const block = this.snapshot.blocks.find(candidate => candidate.id === blockID);
		if (!block) {
			return undefined;
		}

		if (fileID) {
			if (!block.parsed.files.some(file => file.id === fileID)) {
				return undefined;
			}
			return {
				blockID,
				fileID,
				total: 1,
				run: controller => controller.reviewFile(fileID),
			};
		}

		if (block.parsed.files.length === 0) {
			return undefined;
		}

		return {
			blockID,
			total: block.parsed.files.length,
			run: controller => controller.reviewAll(),
		};
	}

	private applyTasks(blockID?: string): MessageDiffTask[] {
		const tasks: MessageDiffTask[] = [];

		for (const block of this.snapshot.blocks) {
			if (blockID && block.id !== blockID) {
				continue;
			}

			const controller = this.snapshot.controllers.get(block.id);
			for (const file of controller?.files ?? []) {
				if (!file.canApply || isDiffApplyOutcomeComplete(file.status)) {
					continue;
				}

				const fingerprint = file.fingerprint;
				tasks.push({
					blockID: block.id,
					fileID: file.file.id,
					total: 1,
					run: current => {
						const live = current.files.find(candidate => candidate.file.id === file.file.id);
						if (
							!live ||
							live.fingerprint !== fingerprint ||
							!live.canApply ||
							isDiffApplyOutcomeComplete(live.status)
						) {
							return Promise.resolve();
						}
						return current.applyFile(file.file.id);
					},
				});
			}
		}

		return tasks;
	}

	public reviewAll = (): Promise<void> => {
		const tasks = this.snapshot.blocks.flatMap(block => {
			const task = this.reviewTask(block.id);
			return task ? [task] : [];
		});
		return this.runTasks('review', tasks);
	};

	public reviewBlock = (blockID: string): Promise<void> => {
		const task = this.reviewTask(blockID);
		return this.runTasks('review', task ? [task] : []);
	};

	public reviewFile = (blockID: string, fileID: string): Promise<void> => {
		const task = this.reviewTask(blockID, fileID);
		return this.runTasks('review', task ? [task] : []);
	};

	public applyAll = (): Promise<void> => this.runTasks('apply', this.applyTasks());

	public applyBlock = (blockID: string): Promise<void> => this.runTasks('apply', this.applyTasks(blockID));

	public applyFile = (blockID: string, fileID: string): Promise<void> => {
		const tasks = this.applyTasks(blockID).filter(task => task.fileID === fileID);
		return this.runTasks('apply', tasks);
	};

	private runFileOperation(
		blockID: string,
		fileID: string,
		kind: 'review-hunks' | 'apply-hunks' | 'apply'
	): Promise<void> {
		const controller = this.snapshot.controllers.get(blockID);
		const file = controller?.files.find(candidate => candidate.file.id === fileID);
		if (!file) {
			return Promise.resolve();
		}

		return this.runTasks(kind, [
			{
				blockID,
				fileID,
				total: kind === 'review-hunks' ? Math.max(1, file.file.hunks.length) : 1,
				run: current =>
					kind === 'review-hunks'
						? current.reviewHunks(fileID)
						: kind === 'apply-hunks'
							? current.applySelectedHunks(fileID)
							: current.applyFile(fileID),
			},
		]);
	}

	public cancel = (): void => {
		if (!this.operationToken) {
			return;
		}

		this.operationToken.cancelled = true;

		for (const controller of this.snapshot.controllers.values()) {
			if (controller.busy) {
				controller.cancel();
			}
		}

		if (this.snapshot.operation) {
			this.update({
				operation: { ...this.snapshot.operation, stopping: true },
			});
		}
	};

	public openDetails = (blockID: string, fileID?: string): void => {
		if (this.disposed || this.snapshot.isBusy) {
			return;
		}

		this.update({ details: { blockID, fileID } });
		void this.ensureController(blockID);
	};

	public closeDetails = (): void => {
		this.update({ details: null });
	};

	public getProgress = (): { completed: number; total: number } => {
		const operation = this.snapshot.operation;
		if (!operation) {
			return { completed: 0, total: 0 };
		}

		const nativeBusy = operation.activeBlockID ? this.snapshot.controllers.get(operation.activeBlockID)?.busy : null;

		return {
			completed: Math.min(operation.total, operation.completed + (nativeBusy?.completed ?? 0)),
			total: operation.total,
		};
	};

	/**
	 * Modal commands also use the message lock. The underlying controller
	 * remains the owner of targets, strict mode, outcomes, and hunk state.
	 */
	public getControllerForUI = (blockID: string): DiffApplyController | undefined => {
		const controller = this.snapshot.controllers.get(blockID);
		if (!controller) {
			return undefined;
		}

		const operation = this.snapshot.operation;
		const progress = this.getProgress();
		const active = operation?.activeBlockID === blockID;
		const nativeBusy = controller.busy;

		return {
			...controller,
			busy: operation
				? {
						id: 0,
						kind: operation.kind,
						total: progress.total,
						completed: progress.completed,
						activeFileIDs: active
							? nativeBusy?.activeFileIDs.length
								? nativeBusy.activeFileIDs
								: operation.activeFileID
									? [operation.activeFileID]
									: []
							: [],
						stopping: operation.stopping,
					}
				: nativeBusy,
			setStrict: value => {
				if (this.canMutate()) {
					controller.setStrict(value);
				}
			},
			setTargetPath: (fileID, value) => {
				if (this.canMutate()) {
					controller.setTargetPath(fileID, value);
				}
			},
			setHunkSelected: (fileID, hunkID, selected) => {
				if (this.canMutate()) {
					controller.setHunkSelected(fileID, hunkID, selected);
				}
			},
			cancel: this.cancel,
			reviewAll: () => this.reviewBlock(blockID),
			reviewFile: fileID => this.reviewFile(blockID, fileID),
			applyReady: () => this.applyBlock(blockID),
			// Keep the modal's explicit per-file submission behavior.
			applyFile: fileID => this.runFileOperation(blockID, fileID, 'apply'),
			reviewHunks: fileID => this.runFileOperation(blockID, fileID, 'review-hunks'),
			applySelectedHunks: fileID => this.runFileOperation(blockID, fileID, 'apply-hunks'),
		};
	};
}

const MessageDiffContext = createContext<MessageDiffStore | null>(null);

export function useMessageDiffScope(): MessageDiffStore | null {
	return useContext(MessageDiffContext);
}

export function useMessageDiffSnapshot(store: MessageDiffStore | null): MessageDiffSnapshot {
	return useSyncExternalStore(
		store?.subscribe ?? subscribeToNothing,
		store?.getSnapshot ?? getEmptySnapshot,
		getEmptySnapshot
	);
}

export function getMessageDiffRows(snapshot: MessageDiffSnapshot): MessageDiffRow[] {
	return snapshot.blocks.flatMap(block => {
		const controller = snapshot.controllers.get(block.id);
		return block.parsed.files.map(file => ({
			blockID: block.id,
			file,
			view: controller?.files.find(candidate => candidate.file === file),
		}));
	});
}

export function getMessageDiffSummary(snapshot: MessageDiffSnapshot): MessageDiffSummary {
	const summary: MessageDiffSummary = {
		total: 0,
		eligible: 0,
		completed: 0,
		ready: 0,
		needsInfo: 0,
		blocked: 0,
		partial: 0,
		pending: 0,
		unresolved: 0,
	};

	for (const row of getMessageDiffRows(snapshot)) {
		summary.total += 1;
		if (row.file.id === 'unsplit-section') {
			summary.unresolved += 1;
		}

		const status = row.view?.status ?? 'pending';
		if (row.view?.canApply && !isDiffApplyOutcomeComplete(status)) {
			summary.eligible += 1;
		}

		switch (status) {
			case 'applied':
			case 'already-applied':
				summary.completed += 1;
				break;
			case 'ready':
				summary.ready += 1;
				break;
			case 'needs-info':
				summary.needsInfo += 1;
				break;
			case 'blocked':
				summary.blocked += 1;
				break;
			case 'partial':
				summary.partial += 1;
				break;
			default:
				summary.pending += 1;
				break;
		}
	}

	return summary;
}

const MessageDiffSession = memo(function MessageDiffSession({
	store,
	block,
	candidatePaths,
	workspaceRoots,
}: {
	store: MessageDiffStore;
	block: MessageDiffBlock;
	candidatePaths?: string[];
	workspaceRoots?: string[];
}) {
	const controller = useDiffApplyController(block.parsed, candidatePaths, workspaceRoots);

	useLayoutEffect(() => {
		store.publishController(block.id, controller);
	}, [block.id, controller, store]);

	return null;
});

function MessageDiffRuntime({
	store,
	candidatePaths,
	workspaceRoots,
}: {
	store: MessageDiffStore;
	candidatePaths?: string[];
	workspaceRoots?: string[];
}) {
	const snapshot = useMessageDiffSnapshot(store);
	const detailsController = snapshot.details ? store.getControllerForUI(snapshot.details.blockID) : undefined;

	return (
		<>
			{snapshot.blocks
				.filter(block => snapshot.activeBlockIDs.includes(block.id))
				.map(block => (
					<MessageDiffSession
						key={block.id}
						store={store}
						block={block}
						candidatePaths={candidatePaths}
						workspaceRoots={workspaceRoots}
					/>
				))}

			{snapshot.details && detailsController ? (
				<DiffApplyModal
					isOpen
					onClose={store.closeDetails}
					controller={detailsController}
					focusFileID={snapshot.details.fileID}
				/>
			) : null}
		</>
	);
}

export function MessageDiffScope({
	messageID,
	source,
	isBusy,
	candidatePaths,
	candidatePathsReady = true,
	workspaceRoots,
	children,
}: {
	messageID: string;
	source: string;
	isBusy: boolean;
	candidatePaths?: string[];
	candidatePathsReady?: boolean;
	workspaceRoots?: string[];
	children: ReactNode;
}) {
	const store = useMemo(
		() => new MessageDiffStore(isBusy, candidatePathsReady),
		// A source replacement invalidates all previous review authorization.
		// eslint-disable-next-line react-hooks/exhaustive-deps
		[messageID, source]
	);

	useLayoutEffect(() => {
		store.configure(isBusy, candidatePathsReady);
	}, [candidatePathsReady, isBusy, store]);

	useEffect(() => {
		store.resume();
		return store.dispose;
	}, [store]);

	return (
		<MessageDiffContext.Provider value={store}>
			<MessageDiffRuntime
				key={`${messageID}\u0000${source}`}
				store={store}
				candidatePaths={candidatePaths}
				workspaceRoots={workspaceRoots}
			/>
			{children}
		</MessageDiffContext.Provider>
	);
}
