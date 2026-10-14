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
	SessionRunSnapshot,
	SessionRuntimeListItem,
	SessionSnapshot,
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

import type { ISessionRuntimeAPI } from '@/apis/interface';
import {
	mcpAppResourceReadResultFromWails,
	sessionCommandReceiptFromWails,
	sessionRunSnapshotFromWails,
	sessionRuntimeEventFromWails,
	sessionRuntimeListItemFromWails,
	sessionSnapshotFromWails,
	sessionUserMessageSkillTemplateResultFromWails,
} from '@/apis/wailsapi/session_projection';
import { wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	CancelRun,
	CompleteCallAsError,
	ContinueRun,
	CreateSession,
	DeleteSession,
	EditAndResend,
	ExecuteAppTool,
	ExecuteCalls,
	GetRunSnapshot,
	GetSessionSnapshot,
	ListRuntimeSessions,
	OpenAppView,
	PauseRun,
	PrepareUserMessageSkillTemplate,
	ReadAppResource,
	RenameSession,
	ResolveWait,
	ResumeRun,
	RetryCall,
	SetAutomationHold,
	SkipCall,
	StageAppContext,
	StartRun,
	UpdateContinuationInput,
	UpdateSessionConfiguration,
} from '@/apis/wailsjs/go/main/SessionRuntimeWrapper';
import { EventsOff, EventsOn } from '@/apis/wailsjs/runtime/runtime';

const SESSION_RUNTIME_EVENT_NAME = 'session-runtime:event';

type RuntimeEventListener = (event: SessionRuntimeEvent) => void;

const runtimeEventListeners = new Set<RuntimeEventListener>();
let runtimeEventBridgeAttached = false;

function dispatchRuntimeEvent(raw: unknown): void {
	let event: SessionRuntimeEvent;

	try {
		event = sessionRuntimeEventFromWails(raw, SESSION_RUNTIME_EVENT_NAME);
	} catch (error) {
		console.error('Invalid session runtime event payload:', error);
		return;
	}

	for (const listener of runtimeEventListeners) {
		try {
			listener(event);
		} catch (error) {
			console.error('Session runtime event listener failed:', error);
		}
	}
}

function attachRuntimeEventBridge(): void {
	if (runtimeEventBridgeAttached) {
		return;
	}

	EventsOn(SESSION_RUNTIME_EVENT_NAME, dispatchRuntimeEvent);
	runtimeEventBridgeAttached = true;
}

function detachRuntimeEventBridgeWhenUnused(): void {
	if (!runtimeEventBridgeAttached || runtimeEventListeners.size > 0) {
		return;
	}

	EventsOff(SESSION_RUNTIME_EVENT_NAME);
	runtimeEventBridgeAttached = false;
}

/**
 * Runtime events are notification-only.
 *
 * Consumers must refresh GetSessionSnapshot/GetRunSnapshot rather than
 * mutating UI state directly from RuntimeEvent.Data. This replaces the old
 * request-scoped completion stream listeners such as text-${requestID}.
 */
export class WailsSessionRuntimeAPI implements ISessionRuntimeAPI {
	subscribeRuntimeEvents(listener: RuntimeEventListener): () => void {
		runtimeEventListeners.add(listener);
		attachRuntimeEventBridge();

		let stopped = false;
		return () => {
			if (stopped) {
				return;
			}

			stopped = true;
			runtimeEventListeners.delete(listener);
			detachRuntimeEventBridgeWhenUnused();
		};
	}

	async prepareUserMessageSkillTemplate(
		command: SessionUserMessageSkillTemplateCommand
	): Promise<SessionUserMessageSkillTemplateResult> {
		return sessionUserMessageSkillTemplateResultFromWails(
			await PrepareUserMessageSkillTemplate(command as Parameters<typeof PrepareUserMessageSkillTemplate>[0]),
			'PrepareUserMessageSkillTemplate'
		);
	}

	async createSession(command: CreateSessionCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			await CreateSession(command as Parameters<typeof CreateSession>[0]),
			'CreateSession'
		);
	}

	async updateSessionConfiguration(command: UpdateSessionConfigurationCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			await UpdateSessionConfiguration(command as Parameters<typeof UpdateSessionConfiguration>[0]),
			'UpdateSessionConfiguration'
		);
	}

	async renameSession(command: RenameSessionCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			await RenameSession(command as Parameters<typeof RenameSession>[0]),
			'RenameSession'
		);
	}

	async deleteSession(command: DeleteSessionCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			await DeleteSession(command as Parameters<typeof DeleteSession>[0]),
			'DeleteSession'
		);
	}

	async startRun(command: StartRunCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(await StartRun(command as Parameters<typeof StartRun>[0]), 'StartRun');
	}

	async updateContinuationInput(command: UpdateContinuationInputCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			await UpdateContinuationInput(command as Parameters<typeof UpdateContinuationInput>[0]),
			'UpdateContinuationInput'
		);
	}

	async continueRun(command: ContinueRunCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			await ContinueRun(command as Parameters<typeof ContinueRun>[0]),
			'ContinueRun'
		);
	}

	async executeCalls(command: ExecuteCallsCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			await ExecuteCalls(command as Parameters<typeof ExecuteCalls>[0]),
			'ExecuteCalls'
		);
	}

	async retryCall(command: RetryCallCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(await RetryCall(command as Parameters<typeof RetryCall>[0]), 'RetryCall');
	}

	async skipCall(command: SkipCallCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(await SkipCall(command as Parameters<typeof SkipCall>[0]), 'SkipCall');
	}

	async completeCallAsError(command: CompleteCallAsErrorCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			// oxlint-disable-next-line unicorn/throw-new-error
			await CompleteCallAsError(command as Parameters<typeof CompleteCallAsError>[0]),
			'CompleteCallAsError'
		);
	}

	async resolveWait(command: ResolveWaitCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			await ResolveWait(command as Parameters<typeof ResolveWait>[0]),
			'ResolveWait'
		);
	}

	async setAutomationHold(command: SetAutomationHoldCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			await SetAutomationHold(command as Parameters<typeof SetAutomationHold>[0]),
			'SetAutomationHold'
		);
	}

	async pauseRun(command: PauseRunCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(await PauseRun(command as Parameters<typeof PauseRun>[0]), 'PauseRun');
	}

	async resumeRun(command: ResumeRunCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(await ResumeRun(command as Parameters<typeof ResumeRun>[0]), 'ResumeRun');
	}

	async cancelRun(command: CancelRunCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(await CancelRun(command as Parameters<typeof CancelRun>[0]), 'CancelRun');
	}

	async editAndResend(command: EditAndResendCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			await EditAndResend(command as Parameters<typeof EditAndResend>[0]),
			'EditAndResend'
		);
	}

	async openAppView(command: MCPAppOpenViewCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			await OpenAppView(command as Parameters<typeof OpenAppView>[0]),
			'OpenAppView'
		);
	}

	async executeAppTool(command: MCPAppExecuteToolCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			await ExecuteAppTool(command as Parameters<typeof ExecuteAppTool>[0]),
			'ExecuteAppTool'
		);
	}

	async stageAppContext(command: MCPAppStageContextCommand): Promise<SessionCommandReceipt> {
		return sessionCommandReceiptFromWails(
			await StageAppContext(command as Parameters<typeof StageAppContext>[0]),
			'StageAppContext'
		);
	}

	async readAppResource(request: MCPAppReadResourceRequest): Promise<MCPAppResourceReadResult> {
		return mcpAppResourceReadResultFromWails(
			await ReadAppResource(request as Parameters<typeof ReadAppResource>[0]),
			'ReadAppResource'
		);
	}

	async getSessionSnapshot(sessionID: SessionID): Promise<SessionSnapshot> {
		return sessionSnapshotFromWails(
			await GetSessionSnapshot(sessionID as Parameters<typeof GetSessionSnapshot>[0]),
			'GetSessionSnapshot'
		);
	}

	async getRunSnapshot(runID: RunID): Promise<SessionRunSnapshot> {
		return sessionRunSnapshotFromWails(
			await GetRunSnapshot(runID as Parameters<typeof GetRunSnapshot>[0]),
			'GetRunSnapshot'
		);
	}

	async listRuntimeSessions(): Promise<SessionRuntimeListItem[]> {
		return wailsObjectArrayOrEmpty(await ListRuntimeSessions(), 'ListRuntimeSessions').map((value, index) =>
			sessionRuntimeListItemFromWails(value, `ListRuntimeSessions[${index}]`)
		);
	}
}
