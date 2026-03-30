import type { ChangeEvent, KeyboardEvent } from 'react';
import { forwardRef, useCallback, useImperativeHandle, useMemo, useRef, useState } from 'react';

import { FiSearch } from 'react-icons/fi';

import { Popover, usePopoverStore, useStoreState } from '@ariakit/react';

import type { ConversationSearchItem } from '@/spec/conversation';

import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import { ConversationSearchDropdown } from '@/chats/search/conversation_search_dropdown';
import { useConversationSearch } from '@/chats/search/use_conversation_search';

export interface ConversationSearchHandle {
	focusInput: () => void;
}

interface ConversationSearchProps {
	onSelectConversation: (item: ConversationSearchItem) => Promise<void>;
	refreshKey: number;
	openConversationIds: string[];
	compact: boolean;
}

export const ConversationSearch = forwardRef<ConversationSearchHandle, ConversationSearchProps>(
	function ConversationSearch({ onSelectConversation, refreshKey, openConversationIds, compact }, ref) {
		const inputRef = useRef<HTMLInputElement>(null);
		const [focusedIndex, setFocusedIndex] = useState(-1);

		const popover = usePopoverStore({
			placement: 'bottom-start',
		});

		const isOpen = useStoreState(popover, 'open');
		const openConversationIdSet = useMemo(() => new Set(openConversationIds.filter(Boolean)), [openConversationIds]);

		const {
			query,
			results,
			loading,
			error,
			hasMore,
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
		} = useConversationSearch({
			onSelectConversation,
			refreshKey,
		});

		const setAnchorElement = useCallback(
			(el: HTMLDivElement | null) => {
				popover.setAnchorElement(el);
			},
			[popover]
		);

		const handleFocus = useCallback(() => {
			popover.setOpen(true);
			void openSearch();
		}, [openSearch, popover]);

		const handleInputChange = useCallback(
			(event: ChangeEvent<HTMLInputElement>) => {
				const value = event.target.value;

				if (!isOpen && value.trim()) {
					popover.setOpen(true);
				}

				setFocusedIndex(-1);
				setQuery(value);
			},
			[isOpen, popover, setQuery]
		);

		const handlePick = useCallback(
			async (conversation: ConversationSearchItem) => {
				popover.setOpen(false);
				setFocusedIndex(-1);
				inputRef.current?.blur();
				void pickConversation(conversation);
			},
			[pickConversation, popover]
		);

		const handleKeyDown = useCallback(
			(event: KeyboardEvent<HTMLInputElement>) => {
				if (!isOpen) return;

				switch (event.key) {
					case 'ArrowDown':
						event.preventDefault();
						if (results.length) {
							setFocusedIndex(index => (index + 1) % results.length);
						}
						break;

					case 'ArrowUp':
						event.preventDefault();
						if (results.length) {
							setFocusedIndex(index => (index - 1 + results.length) % results.length);
						}
						break;

					case 'Enter':
						event.preventDefault();
						if (focusedIndex >= 0 && focusedIndex < results.length) {
							void handlePick(results[focusedIndex].searchConversation);
						} else if (!loading && query.trim()) {
							searchNow();
						}
						break;

					case 'Escape':
						popover.setOpen(false);
						setFocusedIndex(-1);
						if (query) {
							setQuery('');
						}
						break;
				}
			},
			[focusedIndex, handlePick, isOpen, loading, popover, query, results, searchNow, setQuery]
		);

		useImperativeHandle(ref, () => ({
			focusInput: () => {
				inputRef.current?.focus();
			},
		}));

		return (
			<div className="h-full w-full p-0">
				<div
					ref={setAnchorElement}
					className="bg-base-100 border-base-300 focus-within:border-base-300 m-0 flex h-8 items-center rounded-xl border px-2 py-0 shadow-none transition-colors"
				>
					<FiSearch size={14} className="text-neutral-custom mx-2 shrink-0" />

					<input
						ref={inputRef}
						type="text"
						value={query}
						onChange={handleInputChange}
						onFocus={handleFocus}
						onKeyDown={handleKeyDown}
						placeholder="Search conversations..."
						className="placeholder:text-neutral-custom w-full bg-transparent text-sm outline-none"
						spellCheck={false}
					/>

					{loading && <span className="loading loading-dots loading-sm" />}
				</div>

				<Popover
					store={popover}
					portal
					sameWidth
					className="app-no-drag z-999"
					gutter={compact ? 8 : 12}
					autoFocusOnShow={false}
					autoFocusOnHide={false}
				>
					<ConversationSearchDropdown
						results={results}
						loading={loading}
						error={error}
						hasMore={hasMore}
						onLoadMore={loadMore}
						onRetry={retryCurrentMode}
						focusedIndex={focusedIndex}
						onPick={handlePick}
						onAskDelete={askDeleteConversation}
						query={query}
						showSearchAllHintShortQuery={showSearchAllHintShortQuery}
						openConversationIdSet={openConversationIdSet}
					/>
				</Popover>

				<DeleteConfirmationModal
					isOpen={!!deleteTarget}
					onClose={cancelDeleteConversation}
					onConfirm={confirmDeleteConversation}
					title="Delete conversation?"
					message={`Are you sure you want to delete "${deleteTarget?.title}"? This action cannot be undone.`}
					confirmButtonText={deleteLoading ? 'Deleting…' : 'Delete'}
				/>
			</div>
		);
	}
);
