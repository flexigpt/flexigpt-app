import type { Session, SessionRunSnapshot } from '@/spec/session';
import type { SessionID } from '@/spec/session_base';
import type {
	SessionStoreConversationProjection,
	SessionStoreListSessionRunsRequest,
	SessionStoreListSessionsRequest,
	SessionStoreRunListItem,
	SessionStoreSearchSessionsRequest,
	SessionStoreSessionPage,
} from '@/spec/sessionstore';

import type { ISessionStoreAPI } from '@/apis/interface';
import {
	sessionFromWails,
	sessionRunSnapshotFromWails,
	sessionStoreConversationProjectionFromWails,
	sessionStoreRunListItemFromWails,
	sessionStoreSessionPageFromWails,
} from '@/apis/wailsapi/session_projection';
import { requireWailsString, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	ExportSession,
	ListSessionRuns,
	ListSessions,
	ReadSession,
	ReadSessionConversation,
	ReadSessionRun,
	SearchSessions,
} from '@/apis/wailsjs/go/main/SessionStoreWrapper';

export class WailsSessionStoreAPI implements ISessionStoreAPI {
	async readSession(sessionID: SessionID, forceFetch = false): Promise<Session> {
		return sessionFromWails(
			await ReadSession(sessionID as Parameters<typeof ReadSession>[0], forceFetch),
			'ReadSession'
		);
	}

	async readConversation(sessionID: SessionID, forceFetch = false): Promise<SessionStoreConversationProjection> {
		return sessionStoreConversationProjectionFromWails(
			await ReadSessionConversation(sessionID as Parameters<typeof ReadSessionConversation>[0], forceFetch),
			'ReadSessionConversation'
		);
	}

	async readRun(sessionID: SessionID, runID: string, forceFetch = false): Promise<SessionRunSnapshot> {
		return sessionRunSnapshotFromWails(
			await ReadSessionRun(
				sessionID as Parameters<typeof ReadSessionRun>[0],
				runID as Parameters<typeof ReadSessionRun>[1],
				forceFetch
			),
			'ReadSessionRun'
		);
	}

	async listSessions(request: SessionStoreListSessionsRequest = {}): Promise<SessionStoreSessionPage> {
		return sessionStoreSessionPageFromWails(
			await ListSessions(request as Parameters<typeof ListSessions>[0]),
			'ListSessions'
		);
	}

	async searchSessions(request: SessionStoreSearchSessionsRequest): Promise<SessionStoreSessionPage> {
		return sessionStoreSessionPageFromWails(
			await SearchSessions(request as Parameters<typeof SearchSessions>[0]),
			'SearchSessions'
		);
	}

	async listSessionRuns(request: SessionStoreListSessionRunsRequest): Promise<SessionStoreRunListItem[]> {
		return wailsObjectArrayOrEmpty(
			await ListSessionRuns(request as Parameters<typeof ListSessionRuns>[0]),
			'ListSessionRuns'
		).map((value, index) => sessionStoreRunListItemFromWails(value, `ListSessionRuns[${index}]`));
	}

	async exportSession(sessionID: SessionID): Promise<string> {
		return requireWailsString(await ExportSession(sessionID as Parameters<typeof ExportSession>[0]), 'ExportSession');
	}
}
