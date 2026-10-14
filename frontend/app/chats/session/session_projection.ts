import type { Attachment } from '@/spec/attachment';
import type { Conversation, ConversationMessage } from '@/spec/conversation';
import type { InputUnion, OutputUnion, UIToolCall, UIToolOutput } from '@/spec/inference';
import type { MCPApprovalRule, MCPAppVisibility, MCPContent, MCPExecutionMode } from '@/spec/mcp';
import type {
	SessionAppInstance,
	SessionAppRenderTarget,
	SessionAttachmentReference,
	SessionRun,
	SessionSnapshot,
	SessionToolCall,
	SessionToolResult,
	SessionTranscriptMessage,
} from '@/spec/session';
import { AttachmentKind } from '@/spec/attachment';
import { RoleEnum, Status, UIToolCallStatus } from '@/spec/inference';
import {
	SessionAttachmentReferenceKind,
	SessionMessageRole,
	SessionRunState,
	SessionToolBindingKind,
	SessionToolCallState,
	SessionToolResultKind,
} from '@/spec/session';
import { ToolOutputKind, ToolStoreChoiceType } from '@/spec/tool';

import { deriveUIFieldsFromOutputUnion } from '@/chats/conversation/completion_helper';
import { selectedSessionRun } from '@/chats/session/session_commands';

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

function messageStatus(message: SessionTranscriptMessage): Status {
	if (message.error || message.status === 'failed') {
		return Status.Failed;
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
		case SessionToolCallState.ExecutionQueued:
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
		approvalRule?: string;
		executionMode?: string;
		appResourceUri?: string;
		visibility?: string[];
	}>(call.binding.data);

	if (!binding?.server || !binding.toolName) {
		return undefined;
	}

	return {
		server: binding.server,
		toolName: binding.toolName,
		providerToolName: binding.providerToolName,
		choiceID: binding.choiceID,
		digest: binding.toolDigest,
		approvalRule: binding.approvalRule as MCPApprovalRule,
		executionMode: binding.executionMode as MCPExecutionMode,
		appResourceUri: binding.appResourceUri,
		visibility: binding.visibility as MCPAppVisibility[],
	};
}

function appRenderTarget(
	app: SessionAppInstance | undefined,
	sessionSnapshot: SessionSnapshot,
	run: SessionRun
): SessionAppRenderTarget | undefined {
	if (!app) {
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
	sessionSnapshot: SessionSnapshot,
	run: SessionRun,
	appsByID: Map<string, SessionAppInstance>
): UIToolOutput {
	const app = result.appInstanceID ? appsByID.get(result.appInstanceID) : undefined;
	const structured = decodeSessionOpaqueJSON(result.structured);
	const appPayload = decodeSessionOpaqueJSON<{
		resourceUri?: string;
		resourceURI?: string;
		mimeType?: string;
		content?: MCPContent[];
		structuredContent?: unknown;
		isError?: boolean;
	}>(result.app);

	const text =
		result.text ||
		result.error ||
		(structured === undefined ? 'Tool completed without textual output.' : JSON.stringify(structured, null, 2));

	return {
		id: result.id,
		callID: result.callID,
		name: result.name,
		choiceID: result.choiceID ?? '',
		type: toolChoiceType(result.type),
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
	const output = [];

	for (const raw of message.protocolOutputs ?? []) {
		const value = decodeSessionOpaqueJSON<OutputUnion>(raw);
		if (value) {
			output.push(value);
		}
	}

	return output;
}

function projectTranscriptMessage(
	message: SessionTranscriptMessage,
	sessionSnapshot: SessionSnapshot,
	run: SessionRun | undefined,
	appsByID: Map<string, SessionAppInstance>
): ConversationMessage {
	const outputs = protocolOutputs(message);
	const derived = outputs.length > 0 ? deriveUIFieldsFromOutputUnion(outputs, new Map()) : { uiContent: '' };
	const toolCalls =
		run && message.operationID
			? run.toolCalls
				? run.toolCalls
						.filter(call => call.originOperationID === message.operationID)
						.map(t => {
							return uiToolCallFromSession(t);
						})
				: undefined
			: undefined;
	const toolOutputs =
		run && message.toolResults
			? message.toolResults.map(result => toolOutputFromResult(result, sessionSnapshot, run, appsByID))
			: undefined;

	return {
		id: message.id,
		createdAt: message.createdAt,
		role: messageRole(message.role),
		status: messageStatus(message),
		// oxlint-disable-next-line no-warning-comments
		// TODO Here it says can specific only known props, text doesnt exist.
		// text: message.text,
		attachments: message.attachments?.map(attachmentFromSession),
		inputs: (message.protocolInputs ?? [])
			.map(raw => decodeSessionOpaqueJSON(raw))
			.filter(value => value !== undefined) as InputUnion[],
		outputs,
		error: message.error,
		uiContent: derived.uiContent || message.text || '',
		uiReasoningContents: derived.uiReasoningContents,
		uiToolCalls: toolCalls?.length ? toolCalls : derived.uiToolCalls,
		uiToolOutputs: toolOutputs?.length ? toolOutputs : derived.uiToolOutputs,
		uiCitations: derived.uiCitations,
	};
}

function projectActiveOperation(
	_sessionSnapshot: SessionSnapshot,
	run: SessionRun,
	_appsByID: Map<string, SessionAppInstance>
): ConversationMessage | undefined {
	const operation = run.modelOperations?.find(candidate => candidate.id === run.activeModelOperationID);

	if (!operation) {
		return undefined;
	}

	if (!operation.streamedText && !operation.streamedThinking) {
		return undefined;
	}

	const resultID = `session-stream:${run.id}:${operation.id}`;
	const toolCalls = run.toolCalls
		? run.toolCalls
				.filter(call => call.originOperationID === operation.id)
				.map(t => {
					return uiToolCallFromSession(t);
				})
		: undefined;

	return {
		id: resultID,
		createdAt: operation.startedAt || operation.request.current.createdAt,
		role: RoleEnum.Assistant,
		status: Status.InProgress,
		uiContent: operation.streamedText || '',
		// oxlint-disable-next-line no-warning-comments
		// TODO: Need to fix this. here it says thinking cannot be assigned to Reasoning content.
		// uiReasoningContents: operation.streamedThinking
		// 	? [
		// 			{
		// 				thinking: [operation.streamedThinking],
		// 			},
		// 		]
		// 	: undefined,
		uiToolCalls: toolCalls && toolCalls.length > 0 ? toolCalls : undefined,
	};
}

export function projectSessionSnapshotToConversation(snapshot: SessionSnapshot, preferredRunID?: string): Conversation {
	const run = selectedSessionRun(snapshot, preferredRunID);
	const appsByID = new Map((run?.apps ?? []).map(app => [app.id, app] as const));
	const messages = snapshot.session.transcript
		? snapshot.session.transcript.map(message => projectTranscriptMessage(message, snapshot, run, appsByID))
		: [];

	if (run && run.activeModelOperationID) {
		const hasPersistedAssistant = messages.some(
			message =>
				message.role === RoleEnum.Assistant && message.id !== `session-stream:${run.id}:${run.activeModelOperationID}`
		);
		const transient = hasPersistedAssistant ? undefined : projectActiveOperation(snapshot, run, appsByID);

		if (transient) {
			messages.push(transient);
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
