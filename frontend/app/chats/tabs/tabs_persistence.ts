import { initConversation } from '@/chats/conversation/hydration_helper';
import { type ChatTabState, createEmptyTab, MAX_TABS } from '@/chats/tabs/tabs_model';

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

export type InitialChatsModel = {
	restoredFromStorage: boolean;
	selectedTabId: string;
	tabs: ChatTabState[];
	scrollTopByTab: Record<string, number>;
	lastActivatedAtByTab: Record<string, number>;
};

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

export function writePersistedChatsPageState(state: PersistedChatsPageStateV1): void {
	if (!canUseStorage()) return;

	try {
		localStorage.setItem(CHAT_TABS_PERSIST_KEY, JSON.stringify(state));
	} catch {
		// ignore
	}
}

export function buildInitialChatsModel(): InitialChatsModel {
	const persisted = readPersistedChatsPageState();

	if (!persisted) {
		const tab = createEmptyTab();
		return {
			restoredFromStorage: false,
			selectedTabId: tab.tabId,
			tabs: [tab],
			scrollTopByTab: {},
			lastActivatedAtByTab: {},
		};
	}

	const seen = new Set<string>();
	const sanitizedTabs = persisted.tabs
		.filter(tab => {
			if (!tab?.tabId || typeof tab.tabId !== 'string') return false;
			if (seen.has(tab.tabId)) return false;
			seen.add(tab.tabId);
			return true;
		})
		.slice(0, MAX_TABS);

	const tabs: ChatTabState[] = sanitizedTabs.map(tab => {
		const conversation = initConversation();
		if (tab.isPersisted && tab.conversationId) {
			conversation.id = tab.conversationId;
		}
		conversation.title = tab.title || conversation.title;
		conversation.messages = [];

		return {
			tabId: tab.tabId,
			conversation,
			isLoaded: !tab.isPersisted,
			isBusy: false,
			isHydrating: false,
			isPersisted: tab.isPersisted,
			manualTitleLocked: tab.manualTitleLocked,
			editingMessageId: null,
		};
	});

	const nonEmptyTabs = tabs.length > 0 ? tabs : [createEmptyTab()];
	const selectedTabId =
		persisted.selectedTabId && nonEmptyTabs.some(tab => tab.tabId === persisted.selectedTabId)
			? persisted.selectedTabId
			: nonEmptyTabs[0].tabId;

	return {
		restoredFromStorage: true,
		selectedTabId,
		tabs: nonEmptyTabs,
		scrollTopByTab: persisted.scrollTopByTab ?? {},
		lastActivatedAtByTab: persisted.lastActivatedAtByTab ?? {},
	};
}
