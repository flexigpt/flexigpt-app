import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { SessionSearchItem, SessionStoreSessionListItem } from '@/spec/sessionstore';

import { cleanSearchQuery } from '@/lib/title_utils';

import { sessionManagementAPI } from '@/apis/baseapi';

import type { SearchResult } from '@/chats/search/conversation_search_utils';
import {
	CACHE_EXPIRY_TIME,
	mergeUniqBy,
	sessionsToSearchResults,
	sortConversationSearchResults,
	uniqBy,
} from '@/chats/search/conversation_search_utils';
import { requireAcceptedSessionCommand, sessionCommandMeta } from '@/chats/session/session_commands';

interface SearchState {
	query: string;
	results: SearchResult[];
	nextToken?: string;
	hasMore: boolean;
	loading: boolean;
	error?: string;
	searchedMessages: boolean;
}

interface SearchCacheEntry {
	results: SearchResult[];
	nextToken?: string;
	timestamp: number;
}

const searchCache = new Map<string, SearchCacheEntry>();

interface UseConversationSearchArgs {
	onSelectSession: (item: SessionSearchItem) => Promise<void>;
	refreshKey: number;
}

function toSearchItem(value: SessionStoreSessionListItem): SessionSearchItem {
	return {
		id: value.id,
		title: value.title || 'Conversation',
		modifiedAt: value.modifiedAt ?? value.createdAt,
	};
}

export function useConversationSearch({ onSelectSession, refreshKey }: UseConversationSearchArgs) {
	const [searchState, setSearchState] = useState<SearchState>({
		query: '',
		results: [],
		loading: false,
		hasMore: false,
		searchedMessages: false,
	});

	const [recentSessions, setRecentSessions] = useState<SessionSearchItem[]>([]);
	const [deleteTarget, setDeleteTarget] = useState<SessionSearchItem | null>(null);
	const [deleteLoading, setDeleteLoading] = useState(false);

	const debounceTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const recentsNextTokenRef = useRef<string | undefined>(undefined);
	const recentsHasMoreRef = useRef(false);
	const recentSessionsRef = useRef<SessionSearchItem[]>([]);
	const searchRequestSeqRef = useRef(0);
	const searchStateRef = useRef(searchState);

	const invalidatePendingSearches = useCallback(() => {
		searchRequestSeqRef.current += 1;
	}, []);

	useEffect(() => {
		recentSessionsRef.current = recentSessions;
	}, [recentSessions]);

	useEffect(() => {
		searchStateRef.current = searchState;
	}, [searchState]);

	const loadRecentConversations = useCallback(async (token?: string, append = false) => {
		try {
			if (!append) {
				setSearchState(prev => ({ ...prev, loading: true, error: undefined }));
			}

			const page = await sessionManagementAPI.listSessions({
				pageToken: token,
				pageSize: 20,
			});
			const sessions = page.items.map(toSearchItem);
			const nextToken = page.nextPageToken;
			const trimmedNext = nextToken?.trim() || '';
			const nextRecentSessions = append
				? mergeUniqBy(recentSessionsRef.current, sessions, session => session.id)
				: [...sessions];

			recentsNextTokenRef.current = trimmedNext || undefined;
			recentsHasMoreRef.current = !!trimmedNext;

			recentSessionsRef.current = nextRecentSessions;
			setRecentSessions(nextRecentSessions);

			setSearchState(prev => {
				const activeQuery = prev.query;

				if (activeQuery.length > 0 && activeQuery.length < 3) {
					const filtered = nextRecentSessions.filter(session =>
						session.title.toLowerCase().includes(activeQuery.toLowerCase())
					);

					return {
						...prev,
						results: sessionsToSearchResults(filtered),
						loading: false,
						nextToken: trimmedNext,
						hasMore: !!trimmedNext,
						error: undefined,
						searchedMessages: false,
					};
				}

				if (activeQuery !== '' && !append) {
					return {
						...prev,
						loading: false,
						nextToken: trimmedNext,
						hasMore: !!trimmedNext,
					};
				}

				const previousSessions = append ? prev.results.map(result => result.searchSession) : [];
				const mergedSessions = append
					? mergeUniqBy(previousSessions, sessions, session => session.id)
					: nextRecentSessions;

				return {
					...prev,
					results: sessionsToSearchResults(mergedSessions),
					loading: false,
					nextToken: trimmedNext,
					hasMore: !!trimmedNext,
					error: undefined,
					searchedMessages: false,
				};
			});
		} catch (error) {
			console.error(error);
			setSearchState(prev => ({ ...prev, loading: false, error: 'Failed to load conversations' }));
		}
	}, []);

	const performSearch = useCallback(
		async (rawQuery: string, token?: string, append = false) => {
			const query = cleanSearchQuery(rawQuery);

			if (query === '') {
				invalidatePendingSearches();
				setSearchState(prev => ({
					...prev,
					results: [],
					loading: false,
					hasMore: false,
					nextToken: undefined,
					error: undefined,
					searchedMessages: false,
				}));
				return;
			}

			const requestSeq = append ? searchRequestSeqRef.current : searchRequestSeqRef.current + 1;
			if (!append) {
				searchRequestSeqRef.current = requestSeq;
				setSearchState(prev => ({
					...prev,
					loading: true,
					error: undefined,
					hasMore: false,
					searchedMessages: false,
				}));
			}

			try {
				const page = await sessionManagementAPI.searchSessions({
					query,
					pageToken: token,
					pageSize: 20,
				});
				const res = {
					sessions: page.items.map(toSearchItem),
					nextToken: page.nextPageToken,
				};
				if (requestSeq !== searchRequestSeqRef.current) {
					return;
				}

				const nextToken = res.nextToken?.trim() || '';
				const pageResults: SearchResult[] = res.sessions.map(searchSession => ({
					searchSession,
					matchType: searchSession.title.toLowerCase().includes(rawQuery.toLowerCase()) ? 'title' : 'message',
					snippet: '',
				}));

				const uniquePage = uniqBy(pageResults, result => result.searchSession.id);

				setSearchState(prev => {
					const results = append
						? mergeUniqBy(prev.results, uniquePage, result => result.searchSession.id)
						: uniquePage;

					return {
						...prev,
						results,
						nextToken,
						hasMore: !!nextToken,
						loading: false,
						searchedMessages: true,
					};
				});

				if (!append) {
					searchCache.set(query, {
						results: uniquePage,
						nextToken,
						timestamp: Date.now(),
					});

					while (searchCache.size > 5) {
						const oldest = [...searchCache.entries()].reduce((a, b) => (a[1].timestamp < b[1].timestamp ? a : b))[0];
						searchCache.delete(oldest);
					}
				}
			} catch {
				if (requestSeq !== searchRequestSeqRef.current) {
					return;
				}

				setSearchState(prev => ({
					...prev,
					loading: false,
					error: 'Search failed. Please try again.',
					hasMore: false,
					searchedMessages: true,
				}));
			}
		},
		[invalidatePendingSearches]
	);

	const filterLocalResults = useCallback(
		(query: string) => {
			const filtered = recentSessions.filter(session => session.title.toLowerCase().includes(query.toLowerCase()));

			setSearchState(prev => ({
				...prev,
				results: sessionsToSearchResults(filtered),
				loading: false,
				hasMore: recentsHasMoreRef.current,
				nextToken: recentsNextTokenRef.current,
				error: undefined,
				searchedMessages: false,
			}));
		},
		[recentSessions]
	);

	const setQuery = useCallback(
		(value: string) => {
			setSearchState(prev => ({ ...prev, query: value, searchedMessages: false }));

			if (debounceTimeoutRef.current) {
				clearTimeout(debounceTimeoutRef.current);
				debounceTimeoutRef.current = null;
			}

			if (!value.trim()) {
				invalidatePendingSearches();
				setSearchState(prev => ({
					...prev,
					results: sessionsToSearchResults(recentSessionsRef.current),
					loading: false,
					hasMore: recentsHasMoreRef.current,
					nextToken: recentsNextTokenRef.current,
					error: undefined,
					searchedMessages: false,
				}));
				return;
			}

			if (value.length < 3) {
				invalidatePendingSearches();
				filterLocalResults(value);
				return;
			}

			const cleaned = cleanSearchQuery(value);
			const cached = searchCache.get(cleaned);

			if (cached && Date.now() - cached.timestamp < CACHE_EXPIRY_TIME) {
				invalidatePendingSearches();
				setSearchState(prev => ({
					...prev,
					results: cached.results,
					nextToken: cached.nextToken,
					hasMore: !!cached.nextToken,
					searchedMessages: true,
					loading: false,
					error: undefined,
				}));
				return;
			}

			debounceTimeoutRef.current = setTimeout(() => {
				void performSearch(value);
			}, 300);
		},
		[filterLocalResults, invalidatePendingSearches, performSearch]
	);

	const openSearch = useCallback(async () => {
		if (recentSessions.length === 0) {
			await loadRecentConversations();
			return;
		}

		if (!searchState.query) {
			setSearchState(prev => ({
				...prev,
				results: sessionsToSearchResults(recentSessions),
			}));
		}
	}, [loadRecentConversations, recentSessions, searchState.query]);

	const searchNow = useCallback(() => {
		if (debounceTimeoutRef.current) {
			clearTimeout(debounceTimeoutRef.current);
			debounceTimeoutRef.current = null;
		}

		const query = searchState.query.trim();
		if (!query) {
			return;
		}

		void performSearch(searchState.query);
	}, [performSearch, searchState.query]);

	const pickSession = useCallback(
		async (session: SessionSearchItem) => {
			await onSelectSession(session);
			setSearchState(prev => ({ ...prev, query: '', searchedMessages: false }));
		},
		[onSelectSession]
	);

	const askDeleteSession = useCallback((session: SessionSearchItem) => {
		setDeleteTarget(session);
	}, []);

	const cancelDeleteConversation = useCallback(() => {
		if (!deleteLoading) {
			setDeleteTarget(null);
		}
	}, [deleteLoading]);

	const confirmDeleteConversation = useCallback(async () => {
		if (!deleteTarget) {
			return;
		}

		setDeleteLoading(true);

		try {
			const snapshot = await sessionManagementAPI.getSessionSnapshot(deleteTarget.id);
			requireAcceptedSessionCommand(
				await sessionManagementAPI.deleteSession({
					meta: sessionCommandMeta(snapshot.session.revision),
					sessionID: deleteTarget.id,
				}),
				'Delete Session'
			);

			const nextRecentSessions = recentSessionsRef.current.filter(session => session.id !== deleteTarget.id);

			recentSessionsRef.current = nextRecentSessions;
			setRecentSessions(nextRecentSessions);

			setSearchState(prev => ({
				...prev,
				results: prev.results.filter(result => result.searchSession.id !== deleteTarget.id),
			}));

			searchCache.forEach((entry, key) => {
				const filtered = entry.results.filter(result => result.searchSession.id !== deleteTarget.id);

				if (filtered.length === 0) {
					searchCache.delete(key);
				} else if (filtered.length !== entry.results.length) {
					searchCache.set(key, { ...entry, results: filtered });
				}
			});

			if (searchState.query.trim() === '') {
				void loadRecentConversations();
			}
		} catch (error) {
			console.error(error);
		} finally {
			setDeleteLoading(false);
			setDeleteTarget(null);
		}
	}, [deleteTarget, loadRecentConversations, searchState.query]);

	const retryCurrentMode = useCallback(() => {
		if (!searchState.query.trim() || (searchState.query.length < 3 && !searchState.searchedMessages)) {
			void loadRecentConversations();
			return;
		}

		void performSearch(searchState.query);
	}, [loadRecentConversations, performSearch, searchState.query, searchState.searchedMessages]);

	const showSearchAllHintShortQuery =
		searchState.query.length > 0 &&
		searchState.query.length < 3 &&
		!searchState.loading &&
		!searchState.searchedMessages;

	const isLocalMode = searchState.query.length === 0 || showSearchAllHintShortQuery;

	const loadMore = useCallback(() => {
		if (searchState.loading || !searchState.hasMore || !searchState.nextToken) {
			return;
		}

		if (isLocalMode) {
			void loadRecentConversations(searchState.nextToken, true);
		} else {
			void performSearch(searchState.query, searchState.nextToken, true);
		}
	}, [isLocalMode, loadRecentConversations, performSearch, searchState]);

	useEffect(() => {
		invalidatePendingSearches();
		searchCache.clear();

		const current = searchStateRef.current;
		const hasActiveFullTextSearch =
			current.query.trim().length > 0 && (current.query.length >= 3 || current.searchedMessages);

		if (hasActiveFullTextSearch) {
			void performSearch(current.query);
			return;
		}

		void loadRecentConversations();
	}, [invalidatePendingSearches, loadRecentConversations, performSearch, refreshKey]);

	useEffect(() => {
		return () => {
			if (debounceTimeoutRef.current) {
				clearTimeout(debounceTimeoutRef.current);
			}
		};
	}, []);

	const orderedResults = useMemo(
		() => (isLocalMode ? [...searchState.results].toSorted(sortConversationSearchResults) : searchState.results),
		[isLocalMode, searchState.results]
	);

	return {
		query: searchState.query,
		results: orderedResults,
		loading: searchState.loading,
		error: searchState.error,
		hasMore: searchState.hasMore,
		searchedMessages: searchState.searchedMessages,
		showSearchAllHintShortQuery,
		setQuery,
		openSearch,
		searchNow,
		loadMore,
		retryCurrentMode,
		pickSession,
		deleteTarget,
		deleteLoading,
		askDeleteSession,
		cancelDeleteConversation,
		confirmDeleteConversation,
	};
}
