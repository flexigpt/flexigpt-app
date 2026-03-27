/* eslint-disable @typescript-eslint/no-unnecessary-type-parameters */
import type { ConversationSearchItem } from '@/spec/conversation';

export const CACHE_EXPIRY_TIME = 60_000;

export interface SearchResult {
	searchConversation: ConversationSearchItem;
	matchType: 'title' | 'message';
	snippet?: string;
}

export const uniqBy = <T, K>(arr: T[], getKey: (item: T) => K): T[] => {
	const seen = new Set<K>();
	const out: T[] = [];

	for (const item of arr) {
		const key = getKey(item);
		if (seen.has(key)) continue;
		seen.add(key);
		out.push(item);
	}

	return out;
};

export const mergeUniqBy = <T, K>(a: T[], b: T[], getKey: (item: T) => K): T[] => {
	const seen = new Set<K>();
	const out: T[] = [];

	for (const item of a) {
		const key = getKey(item);
		if (seen.has(key)) continue;
		seen.add(key);
		out.push(item);
	}

	for (const item of b) {
		const key = getKey(item);
		if (seen.has(key)) continue;
		seen.add(key);
		out.push(item);
	}

	return out;
};

export function sortConversationSearchResults(a: SearchResult, b: SearchResult): number {
	const tA = new Date(a.searchConversation.modifiedAt).getTime();
	const tB = new Date(b.searchConversation.modifiedAt).getTime();

	if (tA !== tB) return tB - tA;
	if (a.matchType === b.matchType) return 0;
	return a.matchType === 'title' ? -1 : 1;
}

export function conversationsToSearchResults(conversations: ConversationSearchItem[]): SearchResult[] {
	return conversations.map(conversation => ({
		searchConversation: conversation,
		matchType: 'title',
	}));
}
