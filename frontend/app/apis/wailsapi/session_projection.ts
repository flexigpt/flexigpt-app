import type {
	Session,
	SessionRun,
	SessionRunSnapshot,
	SessionRuntimeListItem,
	SessionSnapshot,
	SessionUserMessageSkillTemplateResult,
} from '@/spec/session';
import type { SessionCommandReceipt, SessionOpaqueJSON, SessionRuntimeEvent } from '@/spec/session_base';
import type { MCPAppResourceReadResult } from '@/spec/session_mcp';
import type {
	SessionStoreConversationProjection,
	SessionStoreRunListItem,
	SessionStoreSessionListItem,
	SessionStoreSessionPage,
} from '@/spec/sessionstore';
import { SessionRuntimeEventKind } from '@/spec/session_base';

import {
	enumFromWails,
	optionalWailsString,
	requiredObject,
	requireWailsBoolean,
	requireWailsFiniteNumber,
	requireWailsString,
	wailsArrayOrEmpty,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';

function stringArrayFromWails(value: unknown, field: string): string[] {
	return wailsArrayOrEmpty(value, field).map((item, index) => requireWailsString(item, `${field}[${index}]`));
}

export function sessionCommandReceiptFromWails(value: unknown, operation: string): SessionCommandReceipt {
	const receipt = requiredObject<Record<string, unknown>>(value, operation);

	return {
		commandID: requireWailsString(receipt.commandID, `${operation}.commandID`),
		accepted: requireWailsBoolean(receipt.accepted, `${operation}.accepted`),
		duplicate:
			receipt.duplicate === undefined ? undefined : requireWailsBoolean(receipt.duplicate, `${operation}.duplicate`),
		sessionID: optionalWailsString(receipt.sessionID, `${operation}.sessionID`),
		runID: optionalWailsString(receipt.runID, `${operation}.runID`),
		sessionRevision:
			receipt.sessionRevision === undefined
				? undefined
				: requireWailsFiniteNumber(receipt.sessionRevision, `${operation}.sessionRevision`),
		runRevision:
			receipt.runRevision === undefined
				? undefined
				: requireWailsFiniteNumber(receipt.runRevision, `${operation}.runRevision`),
		operationIDs: stringArrayFromWails(receipt.operationIDs, `${operation}.operationIDs`),
		toolCallIDs: stringArrayFromWails(receipt.toolCallIDs, `${operation}.toolCallIDs`),
		waitIDs: stringArrayFromWails(receipt.waitIDs, `${operation}.waitIDs`),
		alreadyActiveToolCallIDs: stringArrayFromWails(
			receipt.alreadyActiveToolCallIDs,
			`${operation}.alreadyActiveToolCallIDs`
		),
		starterText: optionalWailsString(receipt.starterText, `${operation}.starterText`),
		message: optionalWailsString(receipt.message, `${operation}.message`),
		preparationFailure:
			receipt.preparationFailure === null || receipt.preparationFailure === undefined
				? undefined
				: requiredObject<NonNullable<SessionCommandReceipt['preparationFailure']>>(
						receipt.preparationFailure,
						`${operation}.preparationFailure`
					),
	};
}

export function sessionUserMessageSkillTemplateResultFromWails(
	value: unknown,
	operation: string
): SessionUserMessageSkillTemplateResult {
	const result = requiredObject<Record<string, unknown>>(value, operation);

	return {
		skill: requiredObject(result.skill, `${operation}.skill`),
		text: requireWailsString(result.text, `${operation}.text`),
	};
}

export function sessionFromWails(value: unknown, operation: string): Session {
	return requiredObject<Session>(value, operation);
}

function sessionRunFromWails(value: unknown, operation: string): SessionRun {
	return requiredObject<SessionRun>(value, operation);
}

export function sessionSnapshotFromWails(value: unknown, operation: string): SessionSnapshot {
	const snapshot = requiredObject<Record<string, unknown>>(value, operation);

	return {
		session: sessionFromWails(snapshot.session, `${operation}.session`),
		runs: wailsObjectArrayOrEmpty(snapshot.runs, `${operation}.runs`).map((run, index) =>
			sessionRunFromWails(run, `${operation}.runs[${index}]`)
		),
		sequence: requireWailsFiniteNumber(snapshot.sequence, `${operation}.sequence`),
	};
}

export function sessionRunSnapshotFromWails(value: unknown, operation: string): SessionRunSnapshot {
	const snapshot = requiredObject<Record<string, unknown>>(value, operation);

	return {
		sessionID: requireWailsString(snapshot.sessionID, `${operation}.sessionID`),
		sessionRevision: requireWailsFiniteNumber(snapshot.sessionRevision, `${operation}.sessionRevision`),
		run: sessionRunFromWails(snapshot.run, `${operation}.run`),
	};
}

export function sessionRuntimeListItemFromWails(value: unknown, operation: string): SessionRuntimeListItem {
	return requiredObject<SessionRuntimeListItem>(value, operation);
}

export function sessionRuntimeEventFromWails(value: unknown, operation: string): SessionRuntimeEvent {
	const event = requiredObject<Record<string, unknown>>(value, operation);

	return {
		sessionID: requireWailsString(event.sessionID, `${operation}.sessionID`),
		runID: optionalWailsString(event.runID, `${operation}.runID`),
		sequence: requireWailsFiniteNumber(event.sequence, `${operation}.sequence`),
		runSequence:
			event.runSequence === undefined
				? undefined
				: requireWailsFiniteNumber(event.runSequence, `${operation}.runSequence`),
		commandID: optionalWailsString(event.commandID, `${operation}.commandID`),
		operationID: optionalWailsString(event.operationID, `${operation}.operationID`),
		toolCallID: optionalWailsString(event.toolCallID, `${operation}.toolCallID`),
		toolAttemptID: optionalWailsString(event.toolAttemptID, `${operation}.toolAttemptID`),
		waitID: optionalWailsString(event.waitID, `${operation}.waitID`),
		kind: enumFromWails(event.kind, SessionRuntimeEventKind, `${operation}.kind`),
		at: requireWailsString(event.at, `${operation}.at`),
		data: event.data === undefined ? undefined : (event.data as SessionOpaqueJSON),
	};
}

export function mcpAppResourceReadResultFromWails(value: unknown, operation: string): MCPAppResourceReadResult {
	return requiredObject<MCPAppResourceReadResult>(value, operation);
}

function sessionStoreSessionListItemFromWails(value: unknown, operation: string): SessionStoreSessionListItem {
	return requiredObject<SessionStoreSessionListItem>(value, operation);
}

export function sessionStoreSessionPageFromWails(value: unknown, operation: string): SessionStoreSessionPage {
	const page = requiredObject<Record<string, unknown>>(value, operation);

	return {
		items: wailsObjectArrayOrEmpty(page.items, `${operation}.items`).map((item, index) =>
			sessionStoreSessionListItemFromWails(item, `${operation}.items[${index}]`)
		),
		nextPageToken: optionalWailsString(page.nextPageToken, `${operation}.nextPageToken`) || undefined,
	};
}

export function sessionStoreConversationProjectionFromWails(
	value: unknown,
	operation: string
): SessionStoreConversationProjection {
	return requiredObject<SessionStoreConversationProjection>(value, operation);
}

export function sessionStoreRunListItemFromWails(value: unknown, operation: string): SessionStoreRunListItem {
	return requiredObject<SessionStoreRunListItem>(value, operation);
}
