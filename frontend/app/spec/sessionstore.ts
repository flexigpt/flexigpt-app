import type { SessionRunOutcome, SessionTranscriptMessage } from '@/spec/session';
import type { RunID, SessionID } from '@/spec/session_base';

export interface SessionStoreConversationProjection {
	schemaVersion: string;
	sessionID: SessionID;
	sessionRevision: number;
	title?: string;
	createdAt: string;
	modifiedAt: string;
	messages: SessionTranscriptMessage[];
	associatedRunIDs?: RunID[];
}

export interface SessionStoreListSessionsRequest {
	pageToken?: string;
	pageSize?: number;
}

export interface SessionStoreSearchSessionsRequest {
	query: string;
	pageToken?: string;
	pageSize?: number;
}

export interface SessionStoreSessionListItem {
	id: SessionID;
	revision: number;
	title?: string;
	createdAt: string;
	modifiedAt: string;
	runCount: number;
	activeRunIDs?: RunID[];
}

/**
 * Search rows are SessionStore-owned metadata projections. They intentionally
 * do not expose a separate Conversation domain identity.
 */
export interface SessionSearchItem {
	id: SessionID;
	title: string;
	modifiedAt?: string;
}

export interface SessionStoreSessionPage {
	items: SessionStoreSessionListItem[];
	nextPageToken?: string;
}

export interface SessionStoreListSessionRunsRequest {
	sessionID: SessionID;
	activeOnly?: boolean;
	limit?: number;
}

export interface SessionStoreRunListItem {
	id: RunID;
	revision: number;
	state: string;
	createdAt: string;
	updatedAt: string;
	pendingWaitCount: number;
	outcome?: SessionRunOutcome;
}
