import type {
	RunID,
	Session,
	SessionID,
	SessionRunOutcome,
	SessionRunSnapshot,
	SessionTranscriptMessage,
} from '@/spec/session';

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

export interface SessionStoreAPIShape {
	readSession(sessionID: SessionID, forceFetch?: boolean): Promise<Session>;
	readConversation(sessionID: SessionID, forceFetch?: boolean): Promise<SessionStoreConversationProjection>;
	readRun(sessionID: SessionID, runID: RunID, forceFetch?: boolean): Promise<SessionRunSnapshot>;
	listSessions(request?: SessionStoreListSessionsRequest): Promise<SessionStoreSessionPage>;
	searchSessions(request: SessionStoreSearchSessionsRequest): Promise<SessionStoreSessionPage>;
	listSessionRuns(request: SessionStoreListSessionRunsRequest): Promise<SessionStoreRunListItem[]>;
	exportSession(sessionID: SessionID): Promise<string>;
}
