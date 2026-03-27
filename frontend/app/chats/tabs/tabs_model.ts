import type { Conversation } from '@/spec/conversation';

import { ensureMakeID } from '@/lib/uuid_utils';

import { initConversation } from '@/chats/conversation/hydration_helper';

export const MAX_TABS = 8;

export type ChatTabState = {
	tabId: string;
	conversation: Conversation;

	isBusy: boolean;
	isHydrating: boolean;

	isPersisted: boolean;
	manualTitleLocked: boolean;

	editingMessageId: string | null;
};

type NormalizedTabsResult = {
	tabs: ChatTabState[];
	removedTabIds: string[];
	addedScratchTabId: string | null;
};

export function isScratchTab(tab: ChatTabState): boolean {
	return !tab.isPersisted && tab.conversation.messages.length === 0;
}

export function createEmptyTab(tabId: string = ensureMakeID()): ChatTabState {
	return {
		tabId,
		conversation: initConversation(),
		isBusy: false,
		isHydrating: false,
		isPersisted: false,
		manualTitleLocked: false,
		editingMessageId: null,
	};
}

export function toTimestampMap(source?: Record<string, number>): Map<string, number> {
	const map = new Map<string, number>();
	if (!source) return map;

	for (const [tabId, ts] of Object.entries(source)) {
		if (typeof ts === 'number') {
			map.set(tabId, ts);
		}
	}

	return map;
}

function pickLRUEvictionCandidateFromMap(
	current: ChatTabState[],
	activeId: string,
	lastActivatedAt: Map<string, number>
): ChatTabState | null {
	if (current.length < MAX_TABS) return null;

	const base = current.filter(tab => tab.tabId !== activeId && !isScratchTab(tab));
	const nonBusy = base.filter(tab => !tab.isBusy);
	const candidates = nonBusy.length > 0 ? nonBusy : base;

	if (candidates.length === 0) return null;

	const ts = (id: string) => lastActivatedAt.get(id) ?? 0;
	return candidates.reduce((lru, tab) => (ts(tab.tabId) < ts(lru.tabId) ? tab : lru), candidates[0]);
}

export function normalizeTabsForInvariants(
	current: ChatTabState[],
	activeId: string,
	lastActivatedAt: Map<string, number>
): NormalizedTabsResult {
	let next = current.slice();
	const removedTabIds = new Set<string>();
	let addedScratchTabId: string | null = null;

	const scratchTabs = next.filter(isScratchTab);
	const selected = next.find(tab => tab.tabId === activeId) ?? null;
	const keepScratch =
		scratchTabs.length === 0
			? null
			: selected && isScratchTab(selected)
				? selected
				: scratchTabs[scratchTabs.length - 1];

	if (keepScratch) {
		for (const tab of scratchTabs) {
			if (tab.tabId !== keepScratch.tabId) {
				removedTabIds.add(tab.tabId);
			}
		}

		if (removedTabIds.size > 0) {
			next = next.filter(tab => !removedTabIds.has(tab.tabId));
		}

		const keepIdx = next.findIndex(tab => tab.tabId === keepScratch.tabId);
		if (keepIdx !== -1 && keepIdx !== next.length - 1) {
			next = [...next.slice(0, keepIdx), ...next.slice(keepIdx + 1), keepScratch];
		}
	} else {
		while (next.length >= MAX_TABS) {
			const victim = pickLRUEvictionCandidateFromMap(next, activeId, lastActivatedAt);
			if (!victim) break;

			removedTabIds.add(victim.tabId);
			next = next.filter(tab => tab.tabId !== victim.tabId);
		}

		const scratch = createEmptyTab();
		addedScratchTabId = scratch.tabId;
		next = [...next, scratch];
	}

	while (next.length > MAX_TABS) {
		const victim = pickLRUEvictionCandidateFromMap(next, activeId, lastActivatedAt);
		if (!victim) break;

		removedTabIds.add(victim.tabId);
		next = next.filter(tab => tab.tabId !== victim.tabId);
	}

	if (next.length === 0) {
		const scratch = createEmptyTab();
		addedScratchTabId = scratch.tabId;
		next = [scratch];
	}

	return {
		tabs: next,
		removedTabIds: [...removedTabIds],
		addedScratchTabId,
	};
}
