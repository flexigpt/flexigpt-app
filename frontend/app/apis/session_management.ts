// oxlint-disable typescript/parameter-properties
import type {
	CancelRunCommand,
	CompleteCallAsErrorCommand,
	ContinueRunCommand,
	CreateSessionCommand,
	DeleteSessionCommand,
	EditAndResendCommand,
	ExecuteCallsCommand,
	PauseRunCommand,
	RenameSessionCommand,
	ResolveWaitCommand,
	ResumeRunCommand,
	RetryCallCommand,
	SessionRunObservation,
	SessionRunObserver,
	SessionRunSnapshot,
	SessionRuntimeListItem,
	SessionSnapshot,
	SessionSnapshotObservation,
	SessionSnapshotObserver,
	SessionUserMessageSkillTemplateCommand,
	SessionUserMessageSkillTemplateResult,
	SetAutomationHoldCommand,
	SkipCallCommand,
	StartRunCommand,
	UpdateContinuationInputCommand,
	UpdateSessionConfigurationCommand,
} from '@/spec/session';
import type { RunID, SessionCommandReceipt, SessionID, SessionRuntimeEvent } from '@/spec/session_base';
import type {
	MCPAppExecuteToolCommand,
	MCPAppOpenViewCommand,
	MCPAppReadResourceRequest,
	MCPAppResourceReadResult,
	MCPAppStageContextCommand,
} from '@/spec/session_mcp';
import type {
	SessionStoreConversationProjection,
	SessionStoreListSessionRunsRequest,
	SessionStoreListSessionsRequest,
	SessionStoreRunListItem,
	SessionStoreSearchSessionsRequest,
	SessionStoreSessionPage,
} from '@/spec/sessionstore';
import { SessionRunObservationKind, SessionSnapshotObservationKind } from '@/spec/session';
import { SessionRuntimeEventKind } from '@/spec/session_base';

import type { ISessionManagementAPI, ISessionRuntimeAPI, ISessionStoreAPI } from '@/apis/interface';

interface SessionObserverState {
	sessionID: SessionID;
	listeners: Set<SessionSnapshotObserver>;
	lastEventSequence: number;
	lastSnapshotSequence: number;
	pendingEvent?: SessionRuntimeEvent;
	refreshQueued: boolean;
	refreshRequested: boolean;
	refreshRunning: boolean;
}

/**
 * Application-facing Session/Run façade.
 *
 * This is intentionally the only frontend-facing entry point for the new
 * Session Runtime. Wails transport implementations remain private to
 * `baseapi.ts`; chat migration can consume this façade without importing a
 * Wails binding or a domain-specific execution API.
 *
 * Runtime events are notifications only. `observeSession` and `observeRun`
 * always re-read authoritative runtime snapshots and never interpret
 * RuntimeEvent.Data as state.
 */
export class SessionManagementAPI implements ISessionManagementAPI {
	private readonly sessionObservers = new Map<SessionID, SessionObserverState>();
	private runtimeEventUnsubscribe: (() => void) | undefined;

	constructor(
		private readonly runtime: ISessionRuntimeAPI,
		private readonly store: ISessionStoreAPI
	) {}

	subscribeRuntimeEvents(listener: (event: SessionRuntimeEvent) => void): () => void {
		return this.runtime.subscribeRuntimeEvents(listener);
	}

	observeSession(sessionID: SessionID, listener: SessionSnapshotObserver): () => void {
		if (!sessionID.trim()) {
			throw new Error('Session ID is required.');
		}

		let state = this.sessionObservers.get(sessionID);

		if (!state) {
			state = {
				sessionID,
				listeners: new Set<SessionSnapshotObserver>(),
				lastEventSequence: -1,
				lastSnapshotSequence: -1,
				refreshQueued: false,
				refreshRequested: false,
				refreshRunning: false,
			};
			this.sessionObservers.set(sessionID, state);
		}

		state.listeners.add(listener);
		this.ensureRuntimeEventSubscription();
		this.requestSnapshot(state);

		let stopped = false;

		return () => {
			if (stopped) {
				return;
			}

			stopped = true;
			state?.listeners.delete(listener);

			if (state && state.listeners.size === 0 && this.sessionObservers.get(sessionID) === state) {
				this.sessionObservers.delete(sessionID);
				this.stopRuntimeEventSubscriptionWhenUnused();
			}
		};
	}

	observeRun(sessionID: SessionID, runID: RunID, listener: SessionRunObserver): () => void {
		if (!runID.trim()) {
			throw new Error('Run ID is required.');
		}

		return this.observeSession(sessionID, observation => {
			switch (observation.kind) {
				case SessionSnapshotObservationKind.Snapshot: {
					if (!observation.snapshot) {
						return;
					}

					const run = observation.snapshot.runs.find(candidate => candidate.id === runID);

					if (!run) {
						const unavailable: SessionRunObservation = {
							kind: SessionRunObservationKind.Unavailable,
							sessionID,
							runID,
							event: observation.event,
						};
						listener(unavailable);
						return;
					}

					const snapshot: SessionRunObservation = {
						kind: SessionRunObservationKind.Snapshot,
						event: observation.event,
						snapshot: {
							sessionID,
							sessionRevision: observation.snapshot.session.revision,
							run,
						},
					};
					listener(snapshot);
					return;
				}

				case SessionSnapshotObservationKind.Deleted: {
					const deleted: SessionRunObservation = {
						kind: SessionRunObservationKind.Deleted,
						sessionID,
						runID,
						event: observation.event,
					};
					listener(deleted);
					return;
				}

				case SessionSnapshotObservationKind.Error: {
					const failure: SessionRunObservation = {
						kind: SessionRunObservationKind.Error,
						sessionID,
						runID,
						event: observation.event,
						error: observation.error,
					};
					listener(failure);
				}
			}
		});
	}

	dispose(): void {
		const unsubscribe = this.runtimeEventUnsubscribe;
		this.runtimeEventUnsubscribe = undefined;
		this.sessionObservers.clear();

		unsubscribe?.();
	}

	prepareUserMessageSkillTemplate(
		command: SessionUserMessageSkillTemplateCommand
	): Promise<SessionUserMessageSkillTemplateResult> {
		return this.runtime.prepareUserMessageSkillTemplate(command);
	}

	openAppView(command: MCPAppOpenViewCommand): Promise<SessionCommandReceipt> {
		return this.runtime.openAppView(command);
	}

	executeAppTool(command: MCPAppExecuteToolCommand): Promise<SessionCommandReceipt> {
		return this.runtime.executeAppTool(command);
	}

	stageAppContext(command: MCPAppStageContextCommand): Promise<SessionCommandReceipt> {
		return this.runtime.stageAppContext(command);
	}

	readAppResource(request: MCPAppReadResourceRequest): Promise<MCPAppResourceReadResult> {
		return this.runtime.readAppResource(request);
	}

	createSession(command: CreateSessionCommand): Promise<SessionCommandReceipt> {
		return this.runtime.createSession(command);
	}

	updateSessionConfiguration(command: UpdateSessionConfigurationCommand): Promise<SessionCommandReceipt> {
		return this.runtime.updateSessionConfiguration(command);
	}

	renameSession(command: RenameSessionCommand): Promise<SessionCommandReceipt> {
		return this.runtime.renameSession(command);
	}

	deleteSession(command: DeleteSessionCommand): Promise<SessionCommandReceipt> {
		return this.runtime.deleteSession(command);
	}

	startRun(command: StartRunCommand): Promise<SessionCommandReceipt> {
		return this.runtime.startRun(command);
	}

	updateContinuationInput(command: UpdateContinuationInputCommand): Promise<SessionCommandReceipt> {
		return this.runtime.updateContinuationInput(command);
	}

	continueRun(command: ContinueRunCommand): Promise<SessionCommandReceipt> {
		return this.runtime.continueRun(command);
	}

	executeCalls(command: ExecuteCallsCommand): Promise<SessionCommandReceipt> {
		return this.runtime.executeCalls(command);
	}

	retryCall(command: RetryCallCommand): Promise<SessionCommandReceipt> {
		return this.runtime.retryCall(command);
	}

	skipCall(command: SkipCallCommand): Promise<SessionCommandReceipt> {
		return this.runtime.skipCall(command);
	}

	completeCallAsError(command: CompleteCallAsErrorCommand): Promise<SessionCommandReceipt> {
		return this.runtime.completeCallAsError(command);
	}

	resolveWait(command: ResolveWaitCommand): Promise<SessionCommandReceipt> {
		return this.runtime.resolveWait(command);
	}

	setAutomationHold(command: SetAutomationHoldCommand): Promise<SessionCommandReceipt> {
		return this.runtime.setAutomationHold(command);
	}

	pauseRun(command: PauseRunCommand): Promise<SessionCommandReceipt> {
		return this.runtime.pauseRun(command);
	}

	resumeRun(command: ResumeRunCommand): Promise<SessionCommandReceipt> {
		return this.runtime.resumeRun(command);
	}

	cancelRun(command: CancelRunCommand): Promise<SessionCommandReceipt> {
		return this.runtime.cancelRun(command);
	}

	editAndResend(command: EditAndResendCommand): Promise<SessionCommandReceipt> {
		return this.runtime.editAndResend(command);
	}

	getSessionSnapshot(sessionID: SessionID): Promise<SessionSnapshot> {
		return this.runtime.getSessionSnapshot(sessionID);
	}

	getRunSnapshot(runID: RunID): Promise<SessionRunSnapshot> {
		return this.runtime.getRunSnapshot(runID);
	}

	listRuntimeSessions(): Promise<SessionRuntimeListItem[]> {
		return this.runtime.listRuntimeSessions();
	}

	readSession(sessionID: SessionID, forceFetch = false) {
		return this.store.readSession(sessionID, forceFetch);
	}

	readConversation(sessionID: SessionID, forceFetch = false): Promise<SessionStoreConversationProjection> {
		return this.store.readConversation(sessionID, forceFetch);
	}

	readRun(sessionID: SessionID, runID: RunID, forceFetch = false): Promise<SessionRunSnapshot> {
		return this.store.readRun(sessionID, runID, forceFetch);
	}

	listSessions(request: SessionStoreListSessionsRequest = {}): Promise<SessionStoreSessionPage> {
		return this.store.listSessions(request);
	}

	searchSessions(request: SessionStoreSearchSessionsRequest): Promise<SessionStoreSessionPage> {
		return this.store.searchSessions(request);
	}

	listSessionRuns(request: SessionStoreListSessionRunsRequest): Promise<SessionStoreRunListItem[]> {
		return this.store.listSessionRuns(request);
	}

	exportSession(sessionID: SessionID): Promise<string> {
		return this.store.exportSession(sessionID);
	}

	private ensureRuntimeEventSubscription(): void {
		if (this.runtimeEventUnsubscribe || this.sessionObservers.size === 0) {
			return;
		}

		this.runtimeEventUnsubscribe = this.runtime.subscribeRuntimeEvents(event => {
			this.handleRuntimeEvent(event);
		});
	}

	private stopRuntimeEventSubscriptionWhenUnused(): void {
		if (this.sessionObservers.size > 0) {
			return;
		}

		const unsubscribe = this.runtimeEventUnsubscribe;
		this.runtimeEventUnsubscribe = undefined;
		unsubscribe?.();
	}

	private handleRuntimeEvent(event: SessionRuntimeEvent): void {
		const state = this.sessionObservers.get(event.sessionID);

		if (!state) {
			return;
		}

		if (event.kind === SessionRuntimeEventKind.SessionDeleted) {
			state.lastEventSequence = Math.max(state.lastEventSequence, event.sequence);
			this.emitSessionObservation(state, {
				kind: SessionSnapshotObservationKind.Deleted,
				event,
			});
			this.sessionObservers.delete(event.sessionID);
			this.stopRuntimeEventSubscriptionWhenUnused();
			return;
		}

		if (event.sequence <= state.lastEventSequence) {
			return;
		}

		state.lastEventSequence = event.sequence;
		this.requestSnapshot(state, event);
	}

	private requestSnapshot(state: SessionObserverState, event?: SessionRuntimeEvent): void {
		if (this.sessionObservers.get(state.sessionID) !== state) {
			return;
		}

		if (event) {
			state.pendingEvent = event;
		}

		state.refreshRequested = true;

		if (state.refreshQueued || state.refreshRunning) {
			return;
		}

		state.refreshQueued = true;

		void Promise.resolve().then(() => {
			void this.flushSnapshot(state);
		});
	}

	private async flushSnapshot(state: SessionObserverState): Promise<void> {
		state.refreshQueued = false;

		if (state.refreshRunning || this.sessionObservers.get(state.sessionID) !== state) {
			return;
		}

		state.refreshRunning = true;

		try {
			while (state.refreshRequested && this.sessionObservers.get(state.sessionID) === state) {
				state.refreshRequested = false;

				const event = state.pendingEvent;
				state.pendingEvent = undefined;

				try {
					const snapshot = await this.runtime.getSessionSnapshot(state.sessionID);

					if (this.sessionObservers.get(state.sessionID) !== state) {
						return;
					}

					if (snapshot.sequence < state.lastSnapshotSequence) {
						continue;
					}

					state.lastEventSequence = Math.max(state.lastEventSequence, snapshot.sequence);
					state.lastSnapshotSequence = snapshot.sequence;
					this.emitSessionObservation(state, {
						kind: SessionSnapshotObservationKind.Snapshot,
						snapshot,
						event,
					});
				} catch (error) {
					if (this.sessionObservers.get(state.sessionID) !== state) {
						return;
					}

					this.emitSessionObservation(state, {
						kind: SessionSnapshotObservationKind.Error,
						error,
						event,
					});
				}
			}
		} finally {
			state.refreshRunning = false;

			if (state.refreshRequested && this.sessionObservers.get(state.sessionID) === state) {
				this.requestSnapshot(state);
			}
		}
	}

	private emitSessionObservation(state: SessionObserverState, observation: SessionSnapshotObservation): void {
		for (const listener of state.listeners) {
			try {
				listener(observation);
			} catch (error) {
				console.error('Session snapshot observer failed:', error);
			}
		}
	}
}
