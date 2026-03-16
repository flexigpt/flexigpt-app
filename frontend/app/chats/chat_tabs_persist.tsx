import type { Conversation } from '@/spec/conversation';

import { ensureMakeID } from '@/lib/uuid_utils';

import { initConversation } from '@/chats/conversation/hydration_helper';

// ---------------- Tabs persistence ----------------
const CHAT_TABS_PERSIST_KEY = 'app.chats.tabs.v1';

function canUseStorage(): boolean {
	return typeof window !== 'undefined' && typeof localStorage !== 'undefined';
}

type PersistedChatsPageStateV1 = {
	v: 1;
	selectedTabId: string;
	tabs: Array<{
		tabId: string;
		conversationId: string;
		title: string;
		isPersisted: boolean;
		manualTitleLocked: boolean;
	}>;
	scrollTopByTab?: Record<string, number>;
	lastActivatedAtByTab?: Record<string, number>;
};

export const MAX_TABS = 8;

export type ChatTabState = {
	tabId: string;
	conversation: Conversation;

	// streaming (state only for coarse UI like spinner)
	isBusy: boolean;
	isHydrating: boolean;

	// persistence/title behavior
	isPersisted: boolean;
	manualTitleLocked: boolean;

	// edit state
	editingMessageId: string | null;
};

export function isScratchTab(t: ChatTabState) {
	// "New conversation" = local scratch, not a stored convo
	return !t.isPersisted && t.conversation.messages.length === 0;
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

function readPersistedChatsPageState(): PersistedChatsPageStateV1 | null {
	if (!canUseStorage()) return null;
	try {
		const raw = localStorage.getItem(CHAT_TABS_PERSIST_KEY);
		if (!raw) return null;
		const parsed = JSON.parse(raw) as PersistedChatsPageStateV1;
		if (!parsed || parsed.v !== 1) return null;
		if (!Array.isArray(parsed.tabs)) return null;
		return parsed;
	} catch {
		return null;
	}
}

export function writePersistedChatsPageState(state: PersistedChatsPageStateV1) {
	if (!canUseStorage()) return;
	try {
		localStorage.setItem(CHAT_TABS_PERSIST_KEY, JSON.stringify(state));
	} catch {
		// ignore quota / disabled storage; app still works without persistence
	}
}

export type InitialChatsModel = {
	restoredFromStorage: boolean;
	selectedTabId: string;
	tabs: ChatTabState[];
	scrollTopByTab: Record<string, number>;
	lastActivatedAtByTab: Record<string, number>;
};

export function buildInitialChatsModel(): InitialChatsModel {
	const persisted = readPersistedChatsPageState();
	if (!persisted) {
		const t = createEmptyTab();
		return {
			restoredFromStorage: false,
			selectedTabId: t.tabId,
			tabs: [t],
			scrollTopByTab: {},
			lastActivatedAtByTab: {},
		};
	}

	const seen = new Set<string>();
	const sanitizedTabs = persisted.tabs
		.filter(t => {
			if (!t?.tabId || typeof t.tabId !== 'string') return false;
			if (seen.has(t.tabId)) return false;
			seen.add(t.tabId);
			return true;
		})
		.slice(0, MAX_TABS);

	const tabs: ChatTabState[] = sanitizedTabs.map(t => {
		// Create a lightweight placeholder conversation; real content is rehydrated on mount.
		const conv = initConversation();
		if (t.isPersisted && t.conversationId) conv.id = t.conversationId;
		conv.title = t.title || conv.title;
		conv.messages = [];

		return {
			tabId: t.tabId,
			conversation: conv,
			isBusy: false,
			isHydrating: t.isPersisted,
			isPersisted: t.isPersisted,
			manualTitleLocked: t.manualTitleLocked,
			editingMessageId: null,
		};
	});

	const nonEmptyTabs = tabs.length > 0 ? tabs : [createEmptyTab()];
	const selected =
		persisted.selectedTabId && nonEmptyTabs.some(t => t.tabId === persisted.selectedTabId)
			? persisted.selectedTabId
			: nonEmptyTabs[0].tabId;

	return {
		restoredFromStorage: true,
		selectedTabId: selected,
		tabs: nonEmptyTabs,
		scrollTopByTab: persisted.scrollTopByTab ?? {},
		lastActivatedAtByTab: persisted.lastActivatedAtByTab ?? {},
	};
}
