import { FiTrash2 } from 'react-icons/fi';

import type { ConversationSearchItem } from '@/spec/conversation';

import { formatDateAsString } from '@/lib/date_utils';

import { HoverTip } from '@/components/ariakit_hover_tip';

import type { SearchResult } from '@/chats/search/conversation_search_utils';

type ConversationSearchRowMetaProps = {
	result: SearchResult;
	openConversationIdSet: Set<string>;
	onAskDelete: (item: ConversationSearchItem) => void;
};

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
						className="btn btn-ghost btn-xs btn-circle h-5 min-h-0 w-5 shrink-0 p-0 shadow-none"
						aria-label="Delete conversation"
						onClick={e => {
							e.stopPropagation();
							onAskDelete(conversation);
						}}
					>
						<FiTrash2 size={14} className="text-neutral-custom hover:text-error shrink-0" />
					</button>
				</HoverTip>
			)}
		</span>
	);
}

type ConversationSearchFlatRowProps = {
	result: SearchResult;
	index: number;
	isFocused: boolean;
	onPick: (item: ConversationSearchItem) => void;
	openConversationIdSet: Set<string>;
	onAskDelete: (item: ConversationSearchItem) => void;
};

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
			onClick={() => {
				onPick(result.searchConversation);
			}}
			className={`hover:bg-base-100 flex cursor-pointer items-center justify-between px-12 py-2 ${
				isFocused ? 'bg-base-100' : ''
			}`}
		>
			<span className="truncate">{result.searchConversation.title}</span>

			<span className="text-neutral-custom hidden text-xs lg:block">
				<ConversationSearchRowMeta
					result={result}
					openConversationIdSet={openConversationIdSet}
					onAskDelete={onAskDelete}
				/>
			</span>
		</li>
	);
}
