import type { ConversationSearchItem } from '@/spec/conversation';

import { extractTimeFromUUIDv7Str } from '@/lib/uuid_utils';

export const CACHE_EXPIRY_TIME = 60_000;

export interface SearchResult {
	searchConversation: ConversationSearchItem;
	matchType: 'title' | 'message';
	snippet?: string;
}

/**
 * Search-list rendering is the local UI boundary where an ISO wire timestamp
 * becomes a Date. If an older/incomplete search row has no modifiedAt value,
 * retain the old UUIDv7 timestamp fallback without putting Date objects back
 * into the application conversation contract.
 */
export function conversationSearchDate(conversation: ConversationSearchItem): Date {
	const modifiedAt = conversation.modifiedAt ? new Date(conversation.modifiedAt) : undefined;

	return modifiedAt && !Number.isNaN(modifiedAt.getTime()) ? modifiedAt : extractTimeFromUUIDv7Str(conversation.id);
}

// oxlint-disable-next-line typescript/no-unnecessary-type-parameters
export const uniqBy = <T, K>(arr: T[], getKey: (item: T) => K): T[] => {
	const seen = new Set<K>();
	const out: T[] = [];

	for (const item of arr) {
		const key = getKey(item);
		if (seen.has(key)) {
			continue;
		}
		seen.add(key);
		out.push(item);
	}

	return out;
};

// oxlint-disable-next-line typescript/no-unnecessary-type-parameters
export const mergeUniqBy = <T, K>(a: T[], b: T[], getKey: (item: T) => K): T[] => {
	const seen = new Set<K>();
	const out: T[] = [];

	for (const item of a) {
		const key = getKey(item);
		if (seen.has(key)) {
			continue;
		}
		seen.add(key);
		out.push(item);
	}

	for (const item of b) {
		const key = getKey(item);
		if (seen.has(key)) {
			continue;
		}
		seen.add(key);
		out.push(item);
	}

	return out;
};

export function sortConversationSearchResults(a: SearchResult, b: SearchResult): number {
	const tA = conversationSearchDate(a.searchConversation).getTime();
	const tB = conversationSearchDate(b.searchConversation).getTime();

	if (tA !== tB) {
		return tB - tA;
	}
	if (a.matchType === b.matchType) {
		return 0;
	}
	return a.matchType === 'title' ? -1 : 1;
}

export function conversationsToSearchResults(conversations: ConversationSearchItem[]): SearchResult[] {
	return conversations.map(conversation => ({
		searchConversation: conversation,
		matchType: 'title',
	}));
}
