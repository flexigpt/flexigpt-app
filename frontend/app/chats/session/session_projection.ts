import type { Attachment } from '@/spec/attachment';
import type { InputUnion, OutputUnion, ReasoningContent, UIToolCall, UIToolOutput } from '@/spec/inference';
import type {
	Conversation,
	ConversationMessage,
	SessionAttachmentReference,
	SessionConfiguration,
	SessionModelOperation,
	SessionRun,
	SessionSnapshot,
	SessionToolCall,
	SessionToolResult,
	SessionTranscriptMessage,
} from '@/spec/session';
import type { MCPAppInstance, MCPAppRenderTarget, MCPContent } from '@/spec/session_mcp';
import { AttachmentKind } from '@/spec/attachment';
import { RoleEnum, Status, UIToolCallStatus } from '@/spec/inference';
import {
	SessionAttachmentReferenceKind,
	SessionMessageRole,
	SessionOperationState,
	SessionRunState,
	SessionToolCallState,
	SessionToolResultKind,
} from '@/spec/session_base';
import { ToolOutputKind, ToolStoreChoiceType } from '@/spec/tool';

import { redactInternalPresentationFields } from '@/lib/user_facing_presentation';

import { deriveUIFieldsFromOutputUnion } from '@/chats/conversation/completion_helper';
import { selectedSessionRun } from '@/chats/session/session_commands';

interface IndexedToolCall {
	call: SessionToolCall;
	run: SessionRun;
}

interface IndexedOperation {
	operation: SessionModelOperation;
	run: SessionRun;
}

interface SessionRunIndex {
	runsByOperationID: Map<string, SessionRun>;
	callsByOperationID: Map<string, SessionToolCall[]>;
	callsByID: Map<string, IndexedToolCall>;
	appsByRunID: Map<string, Map<string, MCPAppInstance>>;
	operationsByCurrentMessageID: Map<string, IndexedOperation>;
	persistedToolResultIDs: Set<string>;
}

// oxlint-disable-next-line typescript/no-unnecessary-type-parameters
export function decodeSessionOpaqueJSON<T>(value: unknown): T | undefined {
	if (value === null || value === undefined) {
		return undefined;
	}

	if (typeof value === 'string') {
		try {
			return JSON.parse(value) as T;
		} catch {
			return undefined;
		}
	}

	if (Array.isArray(value) && value.every(item => typeof item === 'number')) {
		try {
			const text = new TextDecoder('utf-8', { fatal: true }).decode(Uint8Array.from(value));
			return JSON.parse(text) as T;
		} catch {
			return undefined;
		}
	}

	if (typeof value === 'object') {
		return value as T;
	}

	return undefined;
}

function debugDetailsForOperation(operation: SessionModelOperation | undefined): unknown {
	if (operation?.debugDetails !== undefined && operation.debugDetails !== null) {
		return redactInternalPresentationFields(decodeSessionOpaqueJSON<unknown>(operation.debugDetails));
	}

	return undefined;
}

function debugDetailsErrorForOperation(operation: SessionModelOperation | undefined): string | undefined {
	const value = operation?.debugDetailsError;
	if (typeof value !== 'string') {
		return undefined;
	}

	const trimmed = value.trim();
	return trimmed || undefined;
}

function indexRuns(snapshot: SessionSnapshot): SessionRunIndex {
	const runsByOperationID = new Map<string, SessionRun>();
	const callsByOperationID = new Map<string, SessionToolCall[]>();
	const callsByID = new Map<string, IndexedToolCall>();
	const appsByRunID = new Map<string, Map<string, MCPAppInstance>>();
	const operationsByCurrentMessageID = new Map<string, IndexedOperation>();
	const persistedToolResultIDs = new Set<string>();

	for (const message of snapshot.session.transcript ?? []) {
		for (const result of message.toolResults ?? []) {
			persistedToolResultIDs.add(result.id);
		}
	}

	for (const run of snapshot.runs) {
		appsByRunID.set(run.id, new Map((run.apps ?? []).map(app => [app.id, app] as const)));

		for (const operation of run.modelOperations ?? []) {
			runsByOperationID.set(operation.id, run);
			// A result-only continuation can refer to the same current user
			// message. Its later configuration must not replace the original
			// message's configuration during rehydration.
			if (!operationsByCurrentMessageID.has(operation.request.current.id)) {
				operationsByCurrentMessageID.set(operation.request.current.id, { operation, run });
			}
		}

		for (const call of run.toolCalls ?? []) {
			const calls = callsByOperationID.get(call.originOperationID) ?? [];
			calls.push(call);
			callsByOperationID.set(call.originOperationID, calls);
			callsByID.set(call.id, { call, run });
		}
	}

	return {
		runsByOperationID,
		callsByOperationID,
		callsByID,
		appsByRunID,
		operationsByCurrentMessageID,
		persistedToolResultIDs,
	};
}

function messageRole(role: SessionMessageRole): RoleEnum {
	switch (role) {
		case SessionMessageRole.System:
			return RoleEnum.System;
		case SessionMessageRole.Developer:
			return RoleEnum.Developer;
		case SessionMessageRole.Assistant:
			return RoleEnum.Assistant;
		case SessionMessageRole.Tool:
			return RoleEnum.Tool;
		default:
			return RoleEnum.User;
	}
}

function messageStatus(message: SessionTranscriptMessage, operation?: SessionModelOperation): Status {
	if (message.status === Status.Cancelled || operation?.state === SessionOperationState.Cancelled) {
		return Status.Cancelled;
	}
	if (
		message.status === Status.Incomplete ||
		message.status === 'interrupted' ||
		operation?.state === SessionOperationState.Interrupted
	) {
		return Status.Incomplete;
	}
	if (
		message.error ||
		operation?.error ||
		message.status === 'failed' ||
		operation?.state === SessionOperationState.Failed
	) {
		return Status.Failed;
	}
	if (message.status === 'completed') {
		return Status.Completed;
	}
	return Status.None;
}

function terminalReasoningContents(
	operation: SessionModelOperation | undefined,
	derived: ReasoningContent[] | undefined,
	terminal: boolean
): ReasoningContent[] | undefined {
	const streamedThinking = operation?.streamedThinking ?? '';
	if (!terminal || !streamedThinking.trim() || !operation) {
		return derived;
	}

	const summary = derived?.flatMap(value => value.summary ?? []) ?? [];
	return [
		{
			id: `session-terminal-thinking:${operation.id}`,
			role: RoleEnum.Assistant,
			...(summary.length > 0 ? { summary } : {}),
			thinking: [streamedThinking],
		},
	];
}

function attachmentFromSession(value: SessionAttachmentReference): Attachment {
	switch (value.kind) {
		case SessionAttachmentReferenceKind.File:
			return {
				kind: AttachmentKind.file,
				label: value.label ?? value.file?.name ?? 'Attachment',
				mode: value.mode as Attachment['mode'],
				availableContentBlockModes: value.availableModes as Attachment['availableContentBlockModes'],
				fileRef: value.file
					? {
							path: value.file.path ?? '',
							origPath: value.file.originalPath ?? value.file.path ?? '',
							name: value.file.name ?? '',
							exists: value.file.exists,
							isDir: value.file.isDirectory ?? false,
							size: value.file.size,
							origSize: value.file.originalSize ?? 0,
							modTime: value.file.modifiedAt,
							origModTime: value.file.originalModifiedAt ?? '',
						}
					: undefined,
			};

		case SessionAttachmentReferenceKind.Image:
			return {
				kind: AttachmentKind.image,
				label: value.label ?? value.image?.name ?? 'Image',
				mode: value.mode as Attachment['mode'],
				availableContentBlockModes: value.availableModes as Attachment['availableContentBlockModes'],
				imageRef: value.image
					? {
							path: value.image.path ?? '',
							origPath: value.image.originalPath ?? value.image.path ?? '',
							name: value.image.name ?? '',
							exists: value.image.exists,
							isDir: value.image.isDirectory ?? false,
							size: value.image.size,
							origSize: value.image.originalSize ?? 0,
							modTime: value.image.modifiedAt,
							origModTime: value.image.originalModifiedAt ?? '',
							mimeType: value.image.mimeType,
							width: value.image.width,
							height: value.image.height,
							format: value.image.format,
						}
					: undefined,
			};

		case SessionAttachmentReferenceKind.URL:
			return {
				kind: AttachmentKind.url,
				label: value.label ?? value.url?.original ?? value.url?.url ?? 'URL',
				mode: value.mode as Attachment['mode'],
				availableContentBlockModes: value.availableModes as Attachment['availableContentBlockModes'],
				urlRef: value.url
					? {
							url: value.url.url,
							normalized: value.url.normalized,
							origNormalized: value.url.original ?? value.url.url,
						}
					: undefined,
			};

		default:
			return {
				kind: AttachmentKind.docIndex,
				label: value.label ?? value.generic?.originalHandle ?? value.generic?.handle ?? 'Attachment',
				mode: value.mode as Attachment['mode'],
				availableContentBlockModes: value.availableModes as Attachment['availableContentBlockModes'],
				genericRef: value.generic
					? {
							handle: value.generic.handle,
							origHandle: value.generic.originalHandle ?? value.generic.handle,
						}
					: undefined,
			};
	}
}

function toolCallStatus(state: SessionToolCallState): UIToolCallStatus {
	switch (state) {
		case SessionToolCallState.Running:
		case SessionToolCallState.Evaluating:
		case SessionToolCallState.EvaluationQueued:
		case SessionToolCallState.ExecutionQueued:
		case SessionToolCallState.Waiting:
			return UIToolCallStatus.Running;

		case SessionToolCallState.Resolved:
		case SessionToolCallState.Observed:
			return UIToolCallStatus.Succeeded;

		case SessionToolCallState.Failed:
		case SessionToolCallState.Cancelled:
			return UIToolCallStatus.Failed;

		default:
			return UIToolCallStatus.Pending;
	}
}

function selectedToolResult(call: SessionToolCall): SessionToolResult | undefined {
	if (!call.selectedResultID) {
		return undefined;
	}

	return call.results?.find(result => result.id === call.selectedResultID);
}

function toolChoiceType(value: string | undefined): ToolStoreChoiceType {
	switch (value) {
		case ToolStoreChoiceType.Custom:
			return ToolStoreChoiceType.Custom;
		case ToolStoreChoiceType.WebSearch:
			return ToolStoreChoiceType.WebSearch;
		default:
			return ToolStoreChoiceType.Function;
	}
}

function appRenderTarget(
	app: MCPAppInstance | undefined,
	sessionSnapshot: SessionSnapshot,
	run: SessionRun | undefined
): MCPAppRenderTarget | undefined {
	if (!app || !run) {
		return undefined;
	}

	return {
		sessionID: app.sessionID,
		runID: app.runID,
		appInstanceID: app.id,
		serverID: app.serverID,
		resourceURI: app.resourceURI,
		sessionRevision: sessionSnapshot.session.revision,
		runRevision: run.revision,
	};
}

function toolOutputFromResult(
	result: SessionToolResult,
	call: SessionToolCall | undefined,
	run: SessionRun | undefined,
	sessionSnapshot: SessionSnapshot,
	appsByID: Map<string, MCPAppInstance> | undefined
): UIToolOutput {
	const app = result.appInstanceID ? appsByID?.get(result.appInstanceID) : undefined;
	const displayName = call?.displayName?.trim() || 'Tool';
	const appPayload = decodeSessionOpaqueJSON<{
		resourceUri?: string;
		resourceURI?: string;
		mimeType?: string;
		content?: MCPContent[];
		structuredContent?: unknown;
		isError?: boolean;
		allowAppInitiatedToolCalls?: boolean;
		requireApprovalForOpenLink?: boolean;
		requireApprovalForContextUpdates?: boolean;
	}>(result.app);

	const text = result.text || result.error || 'Tool completed without textual output.';

	return {
		id: result.id,
		callID: result.callID,
		name: displayName,
		displayName,
		description: call?.description,
		appToolName: call?.name,
		choiceID: result.choiceID ?? call?.choiceID ?? '',
		type: toolChoiceType(result.type || call?.type),
		summary: result.kind === SessionToolResultKind.Success ? `Result: ${displayName}` : `Tool error: ${displayName}`,
		toolOutputs: result.contents?.length ? result.contents : [{ kind: ToolOutputKind.Text, textItem: { text } }],
		isError: result.kind !== SessionToolResultKind.Success,
		isSkipped: result.kind === SessionToolResultKind.Skipped,
		errorMessage: result.error,
		arguments: call
			? typeof call.arguments === 'string'
				? call.arguments
				: JSON.stringify(call.arguments ?? {})
			: undefined,
		mcpApp:
			app || appPayload?.resourceUri || appPayload?.resourceURI
				? {
						resourceUri: app?.resourceURI ?? appPayload?.resourceUri ?? appPayload?.resourceURI,
						mimeType: appPayload?.mimeType,
						content: appPayload?.content,
						structuredContent: appPayload?.structuredContent,
						isError: appPayload?.isError,
						allowAppInitiatedToolCalls: appPayload?.allowAppInitiatedToolCalls,
						requireApprovalForOpenLink: appPayload?.requireApprovalForOpenLink,
						requireApprovalForContextUpdates: appPayload?.requireApprovalForContextUpdates,
					}
				: undefined,
		sessionApp: appRenderTarget(app, sessionSnapshot, run),
	};
}

function uiToolCallFromSession(call: SessionToolCall): UIToolCall {
	const displayName = call.displayName?.trim() || 'Tool';
	return {
		id: call.id,
		callID: call.providerCallID || call.id,
		name: displayName,
		displayName,
		description: call.description,
		arguments: typeof call.arguments === 'string' ? call.arguments : JSON.stringify(call.arguments ?? {}),
		choiceID: call.choiceID ?? '',
		type: toolChoiceType(call.type),
		status: toolCallStatus(call.state),
		automatic: call.automatic,
		errorMessage: call.lastFailure,
	};
}

export function projectSessionRunToolActivity(
	snapshot: SessionSnapshot,
	run: SessionRun | undefined
): {
	calls: UIToolCall[];
	outputs: UIToolOutput[];
} {
	if (!run || !sessionRunIsActive(run)) {
		return {
			calls: [],
			outputs: [],
		};
	}

	const appsByID = new Map((run.apps ?? []).map(app => [app.id, app] as const));
	const persistedResultIDs = new Set(
		(snapshot.session.transcript ?? []).flatMap(message => (message.toolResults ?? []).map(result => result.id))
	);
	// Model operations are appended by the Session Runtime. Once the next
	// operation exists, calls/results from the preceding operation belong to
	// history rather than the composer.
	const latestOperationID = run.modelOperations?.at(-1)?.id ?? run.activeModelOperationID;
	const composerCalls = (run.toolCalls ?? []).filter(call => {
		if (latestOperationID && call.originOperationID !== latestOperationID) {
			return false;
		}

		const result = selectedToolResult(call);
		const unresolved = call.state !== SessionToolCallState.Resolved && call.state !== SessionToolCallState.Observed;

		return unresolved || Boolean(result && !persistedResultIDs.has(result.id));
	});
	const results = composerCalls
		.flatMap(call => {
			const result = selectedToolResult(call);
			return result && !persistedResultIDs.has(result.id) ? [{ call, result }] : [];
		})
		.toSorted((left, right) => left.result.createdAt.localeCompare(right.result.createdAt));

	return {
		calls: composerCalls.map(c => {
			return uiToolCallFromSession(c);
		}),
		outputs: results.map(({ call, result }) => toolOutputFromResult(result, call, run, snapshot, appsByID)),
	};
}

/**
 * Session transcript projections intentionally do not resolve Tool Artifacts
 * synchronously. Edit hydration must use the configuration of the original
 * operation, not the current Session configuration or the calls the model
 * happened to make.
 */
export function sessionConfigurationForMessage(
	snapshot: SessionSnapshot,
	messageID: string
): SessionConfiguration | undefined {
	const message = snapshot.session.transcript?.find(candidate => candidate.id === messageID);

	for (const run of snapshot.runs) {
		const operation = run.modelOperations?.find(candidate =>
			message?.operationID ? candidate.id === message.operationID : candidate.request.current.id === messageID
		);

		if (operation) {
			return operation.request.configuration;
		}
	}

	return undefined;
}

function protocolOutputs(message: SessionTranscriptMessage): OutputUnion[] {
	const output: OutputUnion[] = [];

	for (const raw of message.protocolOutputs ?? []) {
		const value = decodeSessionOpaqueJSON<OutputUnion>(raw);
		if (value) {
			output.push(value);
		}
	}

	return output;
}

/*
 * `injectedContext` is intentionally not copied into ConversationMessage
 * content. It is backend-authored semantic replay material. ProtocolInputs
 * remain authoritative when present; InjectedContext is the safe fallback
 * used by runtime setup when protocol inputs are unavailable.
 */
function projectTranscriptMessage(
	message: SessionTranscriptMessage,
	sessionSnapshot: SessionSnapshot,
	index: SessionRunIndex
): ConversationMessage {
	const outputs = protocolOutputs(message);
	const indexedOperation = message.operationID
		? (() => {
				const run = index.runsByOperationID.get(message.operationID);
				const operation = run?.modelOperations?.find(candidate => candidate.id === message.operationID);
				return operation && run ? { operation, run } : undefined;
			})()
		: index.operationsByCurrentMessageID.get(message.id);
	const operationRun = indexedOperation?.run;
	const operation = indexedOperation?.operation;
	const operationCalls = operation ? (index.callsByOperationID.get(operation.id) ?? []) : [];
	const appsByID = operationRun ? index.appsByRunID.get(operationRun.id) : undefined;
	const derived = outputs.length > 0 ? deriveUIFieldsFromOutputUnion(outputs, new Map()) : { uiContent: '' };
	const modelRef = operation?.request.configuration.model
		? {
				rootID: operation.request.configuration.model.rootID,
				artifactID: operation.request.configuration.model.artifactID,
			}
		: operation?.preparation.resolution.model?.used
			? {
					rootID: operation.preparation.resolution.model.used.rootID,
					artifactID: operation.preparation.resolution.model.used.artifactID,
				}
			: undefined;

	const isAssistantMessage = message.role === SessionMessageRole.Assistant;
	const projectedStatus = messageStatus(message, operation);
	const uiReasoningContents = terminalReasoningContents(
		operation,
		derived.uiReasoningContents,
		projectedStatus === Status.Cancelled || projectedStatus === Status.Failed || projectedStatus === Status.Incomplete
	);

	// Usage/debug data belongs to the provider completion, represented by the
	// assistant output. The current user message also points at the operation
	// for replay/configuration purposes, but must not receive a duplicate
	// request/response debugger modal.
	const completionUsage = isAssistantMessage ? operation?.usage : undefined;
	const completionDebugDetails = isAssistantMessage ? debugDetailsForOperation(operation) : undefined;
	const completionDebugDetailsError = isAssistantMessage ? debugDetailsErrorForOperation(operation) : undefined;

	const projectToolResult = (
		result: SessionToolResult,
		call: SessionToolCall | undefined,
		run: SessionRun | undefined
	): UIToolOutput => {
		const apps = run ? index.appsByRunID.get(run.id) : appsByID;
		return toolOutputFromResult(result, call, run, sessionSnapshot, apps);
	};

	const transcriptToolOutputs =
		message.toolResults
			?.map(result => {
				const indexed = index.callsByID.get(result.callID);
				const resultRun = indexed?.run ?? operationRun;
				return projectToolResult(result, indexed?.call, resultRun);
			})
			.filter(Boolean) ?? [];
	const pendingOperationToolOutputs =
		message.role === SessionMessageRole.Assistant
			? operationCalls.flatMap(call => {
					const result = selectedToolResult(call);
					if (!result || index.persistedToolResultIDs.has(result.id)) {
						return [];
					}

					return [projectToolResult(result, call, operationRun)];
				})
			: [];

	return {
		id: message.id,
		createdAt: message.createdAt,
		role: messageRole(message.role),
		status: projectedStatus,
		modelRef,
		attachments: message.attachments?.map(attachmentFromSession),
		operationResolution: operation?.preparation.resolution,
		appContextUpdates: message.mcpAppContext,
		usage: completionUsage,
		inputs: (message.protocolInputs ?? [])
			.map(raw => decodeSessionOpaqueJSON(raw))
			.filter(value => value !== undefined) as InputUnion[],
		outputs,
		error: message.error ?? (message.role === SessionMessageRole.Assistant ? operation?.error : undefined),
		debugDetails: completionDebugDetails,
		debugDetailsError: completionDebugDetailsError,
		// Session Text is the canonicalized final stream. Prefer it over a
		// partial or error-shaped provider payload so cancellation never
		// replaces already displayed assistant content.
		uiContent: message.text || derived.uiContent || '',
		uiReasoningContents,
		uiToolCalls: isAssistantMessage
			? operationCalls.length > 0
				? operationCalls.map(t => {
						return uiToolCallFromSession(t);
					})
				: derived.uiToolCalls
			: undefined,
		uiToolOutputs:
			transcriptToolOutputs.length > 0
				? transcriptToolOutputs
				: pendingOperationToolOutputs.length > 0
					? pendingOperationToolOutputs
					: derived.uiToolOutputs,
		uiCitations: derived.uiCitations,
	};
}

function projectActiveOperation(run: SessionRun): ConversationMessage | undefined {
	const operation = run.modelOperations?.find(candidate => candidate.id === run.activeModelOperationID);

	if (!operation) {
		return undefined;
	}

	const status =
		operation.state === SessionOperationState.Failed
			? Status.Failed
			: operation.state === SessionOperationState.Cancelled
				? Status.Cancelled
				: operation.state === SessionOperationState.Interrupted
					? Status.Incomplete
					: Status.InProgress;

	const toolCalls = run.toolCalls
		?.filter(call => call.originOperationID === operation.id)
		.map(t => {
			return uiToolCallFromSession(t);
		});

	return {
		id: `session-stream:${run.id}:${operation.id}`,
		createdAt: operation.startedAt || operation.request.current.createdAt,
		role: RoleEnum.Assistant,
		status,
		error: operation.error,
		operationResolution: operation.preparation.resolution,
		uiContent: operation.streamedText || '',
		uiReasoningContents: operation.streamedThinking
			? [
					{
						id: `session-thinking:${run.id}:${operation.id}`,
						role: RoleEnum.Assistant,
						thinking: [operation.streamedThinking],
					},
				]
			: undefined,
		uiToolCalls: toolCalls?.length ? toolCalls : undefined,
	};
}

function latestTranscriptUserMessageID(transcript: SessionTranscriptMessage[]): string | undefined {
	for (let index = transcript.length - 1; index >= 0; index -= 1) {
		const message = transcript[index];
		if (message.role === SessionMessageRole.User) {
			return message.id;
		}
	}

	return undefined;
}

function projectTerminalRunOutcome(
	run: SessionRun,
	transcript: SessionTranscriptMessage[]
): ConversationMessage | undefined {
	if (run.state !== SessionRunState.Failed && run.state !== SessionRunState.Cancelled) {
		return undefined;
	}

	const latestOperation = run.modelOperations?.at(-1);
	const latestUserMessageID = latestTranscriptUserMessageID(transcript);

	// EditAndResend removes the replaced transcript branch but retains its Run.
	// A fallback has no historical insertion point, so emit it only for the
	// current visible user turn.
	if (
		!latestOperation ||
		!latestOperation.request.current.id ||
		latestOperation.request.current.id !== latestUserMessageID
	) {
		return undefined;
	}

	const alreadyRepresented = transcript.some(message => {
		if (message.role !== SessionMessageRole.Assistant || message.operationID !== latestOperation.id) {
			return false;
		}

		const status = messageStatus(message, latestOperation);
		return status === Status.Cancelled || status === Status.Failed || status === Status.Incomplete;
	});

	if (alreadyRepresented) {
		return undefined;
	}

	const cancelled = run.state === SessionRunState.Cancelled;
	const outcomeMessage =
		run.outcome?.message?.trim() ||
		(cancelled ? 'The Run was cancelled before a terminal assistant message was recorded.' : 'The Run failed.');
	const terminalError = latestOperation?.error ?? {
		code: run.outcome?.code || (cancelled ? 'cancelled' : 'run_failed'),
		message: outcomeMessage,
	};

	return {
		id: `session-run-outcome:${run.id}:${run.revision}`,
		createdAt: run.updatedAt,
		role: RoleEnum.Assistant,
		status: cancelled ? Status.Cancelled : Status.Failed,
		error: terminalError,
		// Never replace terminal partial output with a generic cancellation
		// sentence. ChatMessage renders terminalError after this preserved body.
		uiContent: latestOperation?.streamedText ?? '',
		uiReasoningContents: latestOperation?.streamedThinking
			? [
					{
						id: `session-run-outcome-thinking:${run.id}:${latestOperation.id}`,
						role: RoleEnum.Assistant,
						thinking: [latestOperation.streamedThinking],
					},
				]
			: undefined,
	};
}

export function projectSessionSnapshotToConversation(snapshot: SessionSnapshot, preferredRunID?: string): Conversation {
	const selectedRun = selectedSessionRun(snapshot, preferredRunID);
	const index = indexRuns(snapshot);
	const messages =
		snapshot.session.transcript?.map(message => projectTranscriptMessage(message, snapshot, index)) ?? [];

	if (selectedRun?.activeModelOperationID) {
		const hasPersistedAssistantForActiveOperation = snapshot.session.transcript?.some(
			message =>
				message.role === SessionMessageRole.Assistant && message.operationID === selectedRun.activeModelOperationID
		);

		if (!hasPersistedAssistantForActiveOperation) {
			const transient = projectActiveOperation(selectedRun);
			if (transient) {
				messages.push(transient);
			}
		}
	}

	const terminalOutcome = selectedRun
		? projectTerminalRunOutcome(selectedRun, snapshot.session.transcript ?? [])
		: undefined;
	if (terminalOutcome) {
		messages.push(terminalOutcome);
	}

	return {
		schemaVersion: 'session-v1',
		id: snapshot.session.id,
		title: snapshot.session.title || 'Conversation',
		createdAt: snapshot.session.createdAt,
		modifiedAt: snapshot.session.updatedAt,
		messages,
	};
}

export function sessionRunIsActive(run: SessionRun | undefined): boolean {
	return Boolean(
		run &&
		run.state !== SessionRunState.Completed &&
		run.state !== SessionRunState.Failed &&
		run.state !== SessionRunState.Cancelled
	);
}
