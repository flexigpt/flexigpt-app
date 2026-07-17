import { FiTrash2 } from 'react-icons/fi';

import type { ConversationSearchItem } from '@/spec/conversation';

import { formatDateAsString } from '@/lib/date_utils';

import { HoverTip } from '@/components/hover_tip';

import type { SearchResult } from '@/chats/search/conversation_search_utils';

interface ConversationSearchRowMetaProps {
	result: SearchResult;
	openConversationIdSet: Set<string>;
	onAskDelete: (item: ConversationSearchItem) => void;
}

export function ConversationSearchRowMeta({
	result,
	openConversationIdSet,
	onAskDelete,
}: ConversationSearchRowMetaProps) {
	const conversation = result.searchConversation;
	const isOpenConversation = openConversationIdSet.has(conversation.id);

	return (
		<span className="inline-flex items-center gap-4">
			{result.matchType === 'message' && <span className="max-w-48 truncate">{result.snippet}</span>}
			<span className="whitespace-nowrap">{formatDateAsString(conversation.modifiedAt)}</span>

			{!isOpenConversation && (
				<HoverTip content="Delete conversation" placement="right">
					<button
						type="button"
						className="btn btn-ghost btn-xs btn-circle size-5 min-h-0 shrink-0 p-0 shadow-none"
						aria-label="Delete conversation"
						onClick={e => {
							e.stopPropagation();
							onAskDelete(conversation);
						}}
					>
						<FiTrash2 size={14} className="app-text-neutral hover:text-error shrink-0" />
					</button>
				</HoverTip>
			)}
		</span>
	);
}

interface ConversationSearchFlatRowProps {
	result: SearchResult;
	index: number;
	isFocused: boolean;
	onPick: (item: ConversationSearchItem) => void;
	openConversationIdSet: Set<string>;
	onAskDelete: (item: ConversationSearchItem) => void;
}

export function ConversationSearchFlatRow({
	result,
	index,
	isFocused,
	onPick,
	openConversationIdSet,
	onAskDelete,
}: ConversationSearchFlatRowProps) {
	return (
		<li
			key={result.searchConversation.id}
			data-index={index}
			className={`hover:bg-base-100 flex cursor-pointer items-center justify-between px-12 py-2 ${
				isFocused ? 'bg-base-100' : ''
			}`}
		>
			<button
				type="button"
				className="flex min-w-0 flex-1 cursor-pointer items-center text-left"
				onClick={() => {
					onPick(result.searchConversation);
				}}
			>
				<span className="truncate">{result.searchConversation.title}</span>
			</button>

			<span className="app-text-neutral hidden text-xs lg:block">
				<ConversationSearchRowMeta
					result={result}
					openConversationIdSet={openConversationIdSet}
					onAskDelete={onAskDelete}
				/>
			</span>
		</li>
	);
}
