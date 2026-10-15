import type { Attachment } from '@/spec/attachment';
import type { InputUnion, OutputUnion, UIToolCall, UIToolOutput } from '@/spec/inference';
import type { MCPApprovalRule, MCPExecutionMode } from '@/spec/mcp';
import type {
	Conversation,
	ConversationMessage,
	SessionAttachmentReference,
	SessionRun,
	SessionSnapshot,
	SessionToolCall,
	SessionToolResult,
	SessionTranscriptMessage,
} from '@/spec/session';
import type {
	MCPAppContextUpdate,
	MCPAppInstance,
	MCPAppRenderTarget,
	MCPAppVisibility,
	MCPContent,
	MCPContentType,
} from '@/spec/session_mcp';
import { AttachmentKind } from '@/spec/attachment';
import { RoleEnum, Status, UIToolCallStatus } from '@/spec/inference';
import {
	SessionAttachmentReferenceKind,
	SessionMessageRole,
	SessionOperationState,
	SessionRunState,
	SessionToolBindingKind,
	SessionToolCallState,
	SessionToolResultKind,
} from '@/spec/session_base';
import { isMCPApprovalRule, isMCPAppVisibility, isMCPExecutionMode } from '@/spec/session_mcp';
import { ToolOutputKind, ToolStoreChoiceType } from '@/spec/tool';

import { deriveUIFieldsFromOutputUnion } from '@/chats/conversation/completion_helper';
import { selectedSessionRun } from '@/chats/session/session_commands';

interface IndexedToolCall {
	call: SessionToolCall;
	run: SessionRun;
}

interface SessionRunIndex {
	runsByOperationID: Map<string, SessionRun>;
	callsByOperationID: Map<string, SessionToolCall[]>;
	callsByID: Map<string, IndexedToolCall>;
	appsByRunID: Map<string, Map<string, MCPAppInstance>>;
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

function indexRuns(snapshot: SessionSnapshot): SessionRunIndex {
	const runsByOperationID = new Map<string, SessionRun>();
	const callsByOperationID = new Map<string, SessionToolCall[]>();
	const callsByID = new Map<string, IndexedToolCall>();
	const appsByRunID = new Map<string, Map<string, MCPAppInstance>>();

	for (const run of snapshot.runs) {
		appsByRunID.set(run.id, new Map((run.apps ?? []).map(app => [app.id, app] as const)));

		for (const operation of run.modelOperations ?? []) {
			runsByOperationID.set(operation.id, run);
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

function messageStatus(
	message: SessionTranscriptMessage,
	operationError?: NonNullable<SessionTranscriptMessage['error']>
): Status {
	if (message.error || operationError || message.status === 'failed') {
		return Status.Failed;
	}
	if (message.status === Status.Cancelled || message.status === 'interrupted') {
		return Status.Cancelled;
	}
	if (message.status === Status.Incomplete) {
		return Status.Incomplete;
	}
	if (message.status === 'completed') {
		return Status.Completed;
	}
	return Status.None;
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

function mcpSelectionFromBinding(call: SessionToolCall): UIToolCall['mcpToolSelection'] {
	if (call.binding.kind !== SessionToolBindingKind.MCP) {
		return undefined;
	}

	const binding = decodeSessionOpaqueJSON<{
		server?: string;
		providerToolName?: string;
		choiceID?: string;
		toolName?: string;
		toolDigest?: string;
		approvalRule?: unknown;
		executionMode?: unknown;
		appResourceUri?: string;
		visibility?: unknown;
	}>(call.binding.data);

	if (!binding?.server || !binding.toolName) {
		return undefined;
	}

	const visibility = Array.isArray(binding.visibility) ? binding.visibility.filter(isMCPAppVisibility) : undefined;

	return {
		server: binding.server,
		toolName: binding.toolName,
		providerToolName: binding.providerToolName,
		choiceID: binding.choiceID,
		digest: binding.toolDigest,
		approvalRule: isMCPApprovalRule(binding.approvalRule) ? (binding.approvalRule as MCPApprovalRule) : undefined,
		executionMode: isMCPExecutionMode(binding.executionMode) ? (binding.executionMode as MCPExecutionMode) : undefined,
		appResourceUri: binding.appResourceUri,
		visibility: visibility as MCPAppVisibility[] | undefined,
	};
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
	const structured = decodeSessionOpaqueJSON(result.structured);
	const appPayload = decodeSessionOpaqueJSON<{
		resourceUri?: string;
		resourceURI?: string;
		mimeType?: string;
		content?: MCPContent[];
		structuredContent?: unknown;
		isError?: boolean;
	}>(result.app);

	const structuredText = structured === undefined ? '' : JSON.stringify(structured, null, 2) || '';
	const text = result.text || result.error || structuredText || 'Tool completed without textual output.';

	return {
		id: result.id,
		callID: result.callID,
		name: result.name,
		choiceID: result.choiceID ?? call?.choiceID ?? '',
		type: toolChoiceType(result.type || call?.type),
		summary: result.kind === SessionToolResultKind.Success ? `Result: ${result.name}` : `Tool error: ${result.name}`,
		toolOutputs: [
			{
				kind: ToolOutputKind.Text,
				textItem: {
					text,
				},
			},
		],
		isError: result.kind !== SessionToolResultKind.Success,
		isSkipped: result.kind === SessionToolResultKind.Skipped,
		errorMessage: result.error,
		mcpToolSelection: call ? mcpSelectionFromBinding(call) : undefined,
		mcpApp:
			app || appPayload?.resourceUri || appPayload?.resourceURI
				? {
						resourceUri: app?.resourceURI ?? appPayload?.resourceUri ?? appPayload?.resourceURI,
						mimeType: appPayload?.mimeType,
						content: appPayload?.content,
						structuredContent: appPayload?.structuredContent,
						isError: appPayload?.isError,
					}
				: undefined,
		sessionApp: appRenderTarget(app, sessionSnapshot, run),
	};
}

function uiToolCallFromSession(call: SessionToolCall): UIToolCall {
	return {
		id: call.id,
		callID: call.providerCallID || call.id,
		name: call.name,
		arguments: typeof call.arguments === 'string' ? call.arguments : JSON.stringify(call.arguments ?? {}),
		choiceID: call.choiceID ?? '',
		type: toolChoiceType(call.type),
		status: toolCallStatus(call.state),
		errorMessage: call.lastFailure,
		mcpToolSelection: mcpSelectionFromBinding(call),
	};
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

function appContextUpdatesFromSession(values: MCPAppContextUpdate[] | undefined) {
	return values?.map(update => ({
		instanceID: update.appInstanceID,
		server: update.serverID,
		resourceUri: update.resourceURI,
		viewID: update.viewID,
		content: update.content?.map(content => ({
			type: content.type as MCPContentType,
			text: content.text,
			uri: content.uri,
			name: content.name,
			description: content.description,
			mimeType: content.mimeType,
		})),
		structuredContent: decodeSessionOpaqueJSON(update.structuredContent),
		updatedAt: update.updatedAt,
	}));
}

function projectTranscriptMessage(
	message: SessionTranscriptMessage,
	sessionSnapshot: SessionSnapshot,
	index: SessionRunIndex
): ConversationMessage {
	const outputs = protocolOutputs(message);
	const operationRun = message.operationID ? index.runsByOperationID.get(message.operationID) : undefined;
	const operation = operationRun?.modelOperations?.find(candidate => candidate.id === message.operationID);
	const operationCalls = message.operationID ? (index.callsByOperationID.get(message.operationID) ?? []) : [];
	const appsByID = operationRun ? index.appsByRunID.get(operationRun.id) : undefined;
	const derived = outputs.length > 0 ? deriveUIFieldsFromOutputUnion(outputs, new Map()) : { uiContent: '' };

	const toolResults =
		message.toolResults
			?.map(result => {
				const indexed = index.callsByID.get(result.callID);
				const resultRun = indexed?.run ?? operationRun;
				const resultApps = resultRun ? index.appsByRunID.get(resultRun.id) : appsByID;

				return toolOutputFromResult(result, indexed?.call, resultRun, sessionSnapshot, resultApps);
			})
			.filter(Boolean) ?? [];

	return {
		id: message.id,
		createdAt: message.createdAt,
		role: messageRole(message.role),
		status: messageStatus(message, operation?.error),
		attachments: message.attachments?.map(attachmentFromSession),
		mcpAppContextUpdates: appContextUpdatesFromSession(message.mcpAppContext),
		inputs: (message.protocolInputs ?? [])
			.map(raw => decodeSessionOpaqueJSON(raw))
			.filter(value => value !== undefined) as InputUnion[],
		outputs,
		error: message.error ?? operation?.error,
		debugDetails: message.runtimeMetadata,
		uiContent: derived.uiContent || message.text || '',
		uiReasoningContents: derived.uiReasoningContents,
		uiToolCalls:
			operationCalls.length > 0
				? operationCalls.map(t => {
						return uiToolCallFromSession(t);
					})
				: derived.uiToolCalls,
		uiToolOutputs: toolResults.length > 0 ? toolResults : derived.uiToolOutputs,
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
