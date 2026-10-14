import { FiTrash2 } from 'react-icons/fi';

import type { SessionSearchItem } from '@/spec/sessionstore';

import { formatDateAsString } from '@/lib/date_utils';

import { HoverTip } from '@/components/hover_tip';

import type { SearchResult } from '@/chats/search/conversation_search_utils';
import { sessionSearchDate } from '@/chats/search/conversation_search_utils';

interface ConversationSearchRowMetaProps {
	result: SearchResult;
	openSessionIdSet: Set<string>;
	onAskDelete: (item: SessionSearchItem) => void;
}

export function ConversationSearchRowMeta({ result, openSessionIdSet, onAskDelete }: ConversationSearchRowMetaProps) {
	const session = result.searchSession;
	const isOpenSession = openSessionIdSet.has(session.id);
	const modifiedAt = sessionSearchDate(session);

	return (
		<span className="inline-flex items-center gap-4">
			{result.matchType === 'message' && <span className="max-w-48 truncate">{result.snippet}</span>}
			<span className="whitespace-nowrap">{formatDateAsString(modifiedAt)}</span>

			{!isOpenSession && (
				<HoverTip content="Delete conversation" placement="right">
					<button
						type="button"
						className="btn btn-ghost btn-xs btn-circle size-5 min-h-0 shrink-0 p-0 shadow-none"
						aria-label="Delete conversation"
						onClick={e => {
							e.stopPropagation();
							onAskDelete(session);
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
	onPick: (item: SessionSearchItem) => void;
	openSessionIdSet: Set<string>;
	onAskDelete: (item: SessionSearchItem) => void;
}

export function ConversationSearchFlatRow({
	result,
	index,
	isFocused,
	onPick,
	openSessionIdSet,
	onAskDelete,
}: ConversationSearchFlatRowProps) {
	return (
		<li
			key={result.searchSession.id}
			data-index={index}
			className={`hover:bg-base-100 flex cursor-pointer items-center justify-between px-12 py-2 ${
				isFocused ? 'bg-base-100' : ''
			}`}
		>
			<button
				type="button"
				className="flex min-w-0 flex-1 cursor-pointer items-center text-left"
				onClick={() => {
					onPick(result.searchSession);
				}}
			>
				<span className="truncate">{result.searchSession.title}</span>
			</button>

			<span className="app-text-neutral hidden text-xs lg:block">
				<ConversationSearchRowMeta result={result} openSessionIdSet={openSessionIdSet} onAskDelete={onAskDelete} />
			</span>
		</li>
	);
}
