import { Events } from '@wailsio/runtime';

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
	SessionInstructionSkillPreviewRequest,
	SessionInstructionSkillPreviewResult,
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
import { SessionWaitResolution } from '@/spec/session_base';

import type { ISessionRuntimeAPI } from '@/apis/interface';
import type { ResolveWaitCommand as WailsResolveWaitCommand } from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session/models';
import {
	mcpAppResourceReadResultFromWails,
	sessionCommandReceiptFromWails,
	sessionRunSnapshotFromWails,
	sessionRuntimeEventFromWails,
	sessionRuntimeListItemFromWails,
	sessionSnapshotFromWails,
	sessionUserMessageSkillTemplateResultFromWails,
} from '@/apis/wailsapi/session_projection';
import { enumToWails, requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
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
	PreviewInstructionSkill,
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
} from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/cmd/agentgo/sessionruntimewrapper';

const SESSION_RUNTIME_EVENT_NAME = 'session-runtime:event';

type RuntimeEventListener = (event: SessionRuntimeEvent) => void;

const runtimeEventListeners = new Set<RuntimeEventListener>();
let runtimeEventBridgeAttached = false;
let runtimeEventBridgeUnsubscribe: (() => void) | undefined;

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

	// Events.On supplies the emitted custom-event payload itself. Do not use
	// payload.data here: SessionRuntimeEvent has its own data field.
	runtimeEventBridgeUnsubscribe = Events.On(SESSION_RUNTIME_EVENT_NAME, payload => {
		dispatchRuntimeEvent(payload);
	});
	runtimeEventBridgeAttached = true;
}

function detachRuntimeEventBridgeWhenUnused(): void {
	if (!runtimeEventBridgeAttached || runtimeEventListeners.size > 0) {
		return;
	}

	const unsubscribe = runtimeEventBridgeUnsubscribe;
	runtimeEventBridgeUnsubscribe = undefined;
	runtimeEventBridgeAttached = false;

	unsubscribe?.();
}

function userMessageSkillTemplateRequestForWails(command: SessionUserMessageSkillTemplateCommand) {
	return {
		...command,
		sessionID: command.sessionID ?? '',
		arguments: command.arguments ?? null,
	};
}

function resolveWaitCommandToWails(command: ResolveWaitCommand): WailsResolveWaitCommand {
	return {
		...command,
		resolution: enumToWails<SessionWaitResolution, WailsResolveWaitCommand['resolution']>(
			command.resolution,
			SessionWaitResolution,
			'ResolveWaitCommand.resolution'
		),
	};
}

/**
 * Runtime events are notification-only.
 *
 * Consumers must refresh GetSessionSnapshot/GetRunSnapshot rather than
 * mutating UI state directly from RuntimeEvent.Data. This replaces the old
 * request-scoped completion stream listeners such as text-${requestID}.
 */
export class WailsSessionRuntimeAPI implements ISessionRuntimeAPI {
	async previewInstructionSkill(
		command: SessionInstructionSkillPreviewRequest
	): Promise<SessionInstructionSkillPreviewResult> {
		return requiredObject<SessionInstructionSkillPreviewResult>(
			await PreviewInstructionSkill(command as Parameters<typeof PreviewInstructionSkill>[0]),
			'PreviewInstructionSkill'
		);
	}

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
			await PrepareUserMessageSkillTemplate(
				userMessageSkillTemplateRequestForWails(command) as Parameters<typeof PrepareUserMessageSkillTemplate>[0]
			),
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
		return sessionCommandReceiptFromWails(await ResolveWait(resolveWaitCommandToWails(command)), 'ResolveWait');
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
