import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { ConversationSearchItem } from '@/spec/conversation';

import { cleanSearchQuery } from '@/lib/title_utils';

import { conversationStoreAPI } from '@/apis/baseapi';

import {
	CACHE_EXPIRY_TIME,
	conversationsToSearchResults,
	mergeUniqBy,
	type SearchResult,
	sortConversationSearchResults,
	uniqBy,
} from '@/chats/search/conversation_search_utils';

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

type UseConversationSearchArgs = {
	onSelectConversation: (item: ConversationSearchItem) => Promise<void>;
	refreshKey: number;
};

export function useConversationSearch({ onSelectConversation, refreshKey }: UseConversationSearchArgs) {
	const [searchState, setSearchState] = useState<SearchState>({
		query: '',
		results: [],
		loading: false,
		hasMore: false,
		searchedMessages: false,
	});

	const [recentConversations, setRecentConversations] = useState<ConversationSearchItem[]>([]);
	const [deleteTarget, setDeleteTarget] = useState<ConversationSearchItem | null>(null);
	const [deleteLoading, setDeleteLoading] = useState(false);

	const debounceTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const recentsNextTokenRef = useRef<string | undefined>(undefined);
	const recentsHasMoreRef = useRef(false);
	const recentConversationsRef = useRef<ConversationSearchItem[]>([]);
	const searchRequestSeqRef = useRef(0);

	const invalidatePendingSearches = useCallback(() => {
		searchRequestSeqRef.current += 1;
	}, []);

	useEffect(() => {
		recentConversationsRef.current = recentConversations;
	}, [recentConversations]);

	const loadRecentConversations = useCallback(async (token?: string, append = false) => {
		try {
			if (!append) {
				setSearchState(prev => ({ ...prev, loading: true, error: undefined }));
			}

			const { conversations, nextToken } = await conversationStoreAPI.listConversations(token, 20);
			const trimmedNext = nextToken?.trim() || '';
			const nextRecentConversations = append
				? mergeUniqBy(recentConversationsRef.current, conversations, conversation => conversation.id)
				: [...conversations];

			recentsNextTokenRef.current = trimmedNext || undefined;
			recentsHasMoreRef.current = !!trimmedNext;

			recentConversationsRef.current = nextRecentConversations;
			setRecentConversations(nextRecentConversations);

			setSearchState(prev => {
				const activeQuery = prev.query;

				if (activeQuery.length > 0 && activeQuery.length < 3) {
					const filtered = nextRecentConversations.filter(conversation =>
						conversation.title.toLowerCase().includes(activeQuery.toLowerCase())
					);

					return {
						...prev,
						results: conversationsToSearchResults(filtered),
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

				const previousConversations = append ? prev.results.map(result => result.searchConversation) : [];
				const mergedConversations = append
					? mergeUniqBy(previousConversations, conversations, conversation => conversation.id)
					: nextRecentConversations;

				return {
					...prev,
					results: conversationsToSearchResults(mergedConversations),
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
				const res = await conversationStoreAPI.searchConversations(query, token, 20);
				if (requestSeq !== searchRequestSeqRef.current) return;

				const nextToken = res.nextToken?.trim() || '';
				const pageResults: SearchResult[] = res.conversations.map(searchConversation => ({
					searchConversation,
					matchType: searchConversation.title.toLowerCase().includes(rawQuery.toLowerCase()) ? 'title' : 'message',
					snippet: '',
				}));

				const uniquePage = uniqBy(pageResults, result => result.searchConversation.id);

				setSearchState(prev => {
					const results = append
						? mergeUniqBy(prev.results, uniquePage, result => result.searchConversation.id)
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
				if (requestSeq !== searchRequestSeqRef.current) return;

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
			const filtered = recentConversations.filter(conversation =>
				conversation.title.toLowerCase().includes(query.toLowerCase())
			);

			setSearchState(prev => ({
				...prev,
				results: conversationsToSearchResults(filtered),
				loading: false,
				hasMore: recentsHasMoreRef.current,
				nextToken: recentsNextTokenRef.current,
				error: undefined,
				searchedMessages: false,
			}));
		},
		[recentConversations]
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
					results: conversationsToSearchResults(recentConversationsRef.current),
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
		if (!recentConversations.length) {
			await loadRecentConversations();
			return;
		}

		if (!searchState.query) {
			setSearchState(prev => ({
				...prev,
				results: conversationsToSearchResults(recentConversations),
			}));
		}
	}, [loadRecentConversations, recentConversations, searchState.query]);

	const searchNow = useCallback(() => {
		if (debounceTimeoutRef.current) {
			clearTimeout(debounceTimeoutRef.current);
			debounceTimeoutRef.current = null;
		}

		const query = searchState.query.trim();
		if (!query) return;

		void performSearch(searchState.query);
	}, [performSearch, searchState.query]);

	const pickConversation = useCallback(
		async (conversation: ConversationSearchItem) => {
			await onSelectConversation(conversation);
			setSearchState(prev => ({ ...prev, query: '', searchedMessages: false }));
		},
		[onSelectConversation]
	);

	const askDeleteConversation = useCallback((conversation: ConversationSearchItem) => {
		setDeleteTarget(conversation);
	}, []);

	const cancelDeleteConversation = useCallback(() => {
		if (!deleteLoading) {
			setDeleteTarget(null);
		}
	}, [deleteLoading]);

	const confirmDeleteConversation = useCallback(async () => {
		if (!deleteTarget) return;

		setDeleteLoading(true);

		try {
			await conversationStoreAPI.deleteConversation(deleteTarget.id, deleteTarget.title);

			const nextRecentConversations = recentConversationsRef.current.filter(
				conversation => conversation.id !== deleteTarget.id
			);

			recentConversationsRef.current = nextRecentConversations;
			setRecentConversations(nextRecentConversations);

			setSearchState(prev => ({
				...prev,
				results: prev.results.filter(result => result.searchConversation.id !== deleteTarget.id),
			}));

			searchCache.forEach((entry, key) => {
				const filtered = entry.results.filter(result => result.searchConversation.id !== deleteTarget.id);

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
		if (searchState.loading || !searchState.hasMore || !searchState.nextToken) return;

		if (isLocalMode) {
			void loadRecentConversations(searchState.nextToken, true);
		} else {
			void performSearch(searchState.query, searchState.nextToken, true);
		}
	}, [isLocalMode, loadRecentConversations, performSearch, searchState]);

	useEffect(() => {
		invalidatePendingSearches();
		searchCache.clear();
		// eslint-disable-next-line react-hooks/set-state-in-effect
		void loadRecentConversations();
	}, [invalidatePendingSearches, loadRecentConversations, refreshKey]);

	useEffect(() => {
		return () => {
			if (debounceTimeoutRef.current) {
				clearTimeout(debounceTimeoutRef.current);
			}
		};
	}, []);

	const orderedResults = useMemo(
		() => (isLocalMode ? [...searchState.results].sort(sortConversationSearchResults) : searchState.results),
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
		pickConversation,
		deleteTarget,
		deleteLoading,
		askDeleteConversation,
		cancelDeleteConversation,
		confirmDeleteConversation,
	};
}
