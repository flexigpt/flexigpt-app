import { useCallback, useEffect, useRef } from 'react';

import type { ConversationSearchItem } from '@/spec/conversation';

import { GroupedDropdown } from '@/components/date_grouped_dropdown';

import { ConversationSearchFlatRow, ConversationSearchRowMeta } from '@/chats/search/conversation_search_row';
import type { SearchResult } from '@/chats/search/conversation_search_utils';

interface ConversationSearchDropdownProps {
	results: SearchResult[];
	loading: boolean;
	error?: string;
	hasMore: boolean;
	onLoadMore: () => void;
	onRetry: () => void;
	focusedIndex: number;
	onPick: (item: ConversationSearchItem) => void;
	onAskDelete: (item: ConversationSearchItem) => void;
	query: string;
	showSearchAllHintShortQuery: boolean;
	openConversationIdSet: Set<string>;
}

export function ConversationSearchDropdown({
	results,
	loading,
	error,
	hasMore,
	onLoadMore,
	onRetry,
	focusedIndex,
	onPick,
	onAskDelete,
	query,
	showSearchAllHintShortQuery,
	openConversationIdSet,
}: ConversationSearchDropdownProps) {
	const scrollRef = useRef<HTMLDivElement>(null);
	const shouldGroup = query.length === 0 || showSearchAllHintShortQuery;

	const handleScroll = useCallback(() => {
		const container = scrollRef.current;
		if (!container || !hasMore || loading) return;

		if ((container.scrollTop + container.clientHeight) / container.scrollHeight >= 0.8) {
			onLoadMore();
		}
	}, [hasMore, loading, onLoadMore]);

	useEffect(() => {
		if (focusedIndex < 0) return;
		const el = scrollRef.current?.querySelector(`[data-index="${focusedIndex}"]`);
		el?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
	}, [focusedIndex]);

	if (error) {
		return (
			<div className="bg-base-200 rounded-2xl p-4 text-center shadow-lg">
				<p className="text-error mb-2 text-sm">{error}</p>
				<button className="btn btn-sm btn-primary" onClick={onRetry}>
					Retry
				</button>
			</div>
		);
	}

	const SEARCHED_ALL = 'Searched all titles & messages';
	const PRESS_ENTER_TO_SEARCH = 'Press Enter to search messages';
	const NO_CONVERSATIONS = 'No conversations yet';

	const getTopBarContent = () => {
		let left = '';
		let right = '';

		if (loading && !results.length) {
			left = 'Searching...';
		} else if (!results.length && !loading) {
			if (query.length < 3) {
				if (showSearchAllHintShortQuery) {
					left = query ? `No title results for "${query}" ` : NO_CONVERSATIONS;
					right = PRESS_ENTER_TO_SEARCH;
				} else {
					left = query ? `No results for "${query}"` : NO_CONVERSATIONS;
					right = SEARCHED_ALL;
				}
			} else {
				left = query ? `No results for "${query}"` : NO_CONVERSATIONS;
				right = query ? SEARCHED_ALL : '';
			}
		} else {
			if (query.length === 0) {
				left = 'Titles';
				right = 'Type to search';
			} else if (query.length < 3) {
				if (showSearchAllHintShortQuery) {
					left = 'Title matches';
					right = PRESS_ENTER_TO_SEARCH;
				} else {
					left = 'Title & message matches';
					right = SEARCHED_ALL;
				}
			} else {
				left = 'Title & message matches';
			}
		}

		return { left, right };
	};

	const { left: barLeft, right: barRight } = getTopBarContent();

	return (
		<div className="bg-base-200 overflow-hidden rounded-2xl shadow-lg">
			<div className="text-neutral-custom border-base-300 sticky top-0 flex items-center justify-between border-b px-8 py-1 text-xs">
				<span className="truncate">{barLeft}</span>
				{barRight && <span className="shrink-0 pl-4">{barRight}</span>}
			</div>

			{!results.length && !loading ? (
				<div className="text-neutral-custom py-8 text-center text-sm">
					{query ? 'Try refining your search' : 'Start a conversation to see it here'}
				</div>
			) : (
				<div ref={scrollRef} className="max-h-[60dvh] overflow-y-auto antialiased" onScroll={handleScroll}>
					{shouldGroup ? (
						<GroupedDropdown<SearchResult>
							items={results}
							focused={focusedIndex}
							getDate={result => new Date(result.searchConversation.modifiedAt)}
							getKey={result => result.searchConversation.id}
							getLabel={result => <span className="truncate">{result.searchConversation.title}</span>}
							onPick={result => {
								onPick(result.searchConversation);
							}}
							renderItemExtra={result => (
								<ConversationSearchRowMeta
									result={result}
									openConversationIdSet={openConversationIdSet}
									onAskDelete={onAskDelete}
								/>
							)}
						/>
					) : (
						<ul className="w-full text-sm">
							{results.map((result, index) => (
								<ConversationSearchFlatRow
									key={result.searchConversation.id}
									result={result}
									index={index}
									isFocused={index === focusedIndex}
									onPick={onPick}
									openConversationIdSet={openConversationIdSet}
									onAskDelete={onAskDelete}
								/>
							))}
						</ul>
					)}

					{loading && (
						<div className="flex items-center justify-center py-4">
							<span className="text-neutral-custom text-sm">{results.length ? 'Loading more...' : 'Searching...'}</span>
							<span className="loading loading-dots loading-sm" />
						</div>
					)}

					{!loading && !hasMore && results.length > 0 && query && (
						<div className="text-neutral-custom border-base-300 border-t py-1 text-center text-xs">End of results</div>
					)}
				</div>
			)}
		</div>
	);
}
