import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { useStoreState } from '@ariakit/react';
import { useTabStore } from '@ariakit/react/tab';

import type {
	Conversation,
	ConversationMessage,
	ConversationSearchItem,
	StoreConversation,
	StoreConversationMessage,
} from '@/spec/conversation';
import { RoleEnum } from '@/spec/inference';

import { defaultShortcutConfig, useChatShortcuts } from '@/lib/keyboard_shortcuts';
import { omitManyKeys } from '@/lib/obj_utils';
import { sanitizeConversationTitle } from '@/lib/text_utils';
import { generateTitle } from '@/lib/title_utils';

import { useTitleBarContent } from '@/hooks/use_title_bar';

import { conversationStoreAPI } from '@/apis/baseapi';

import { PageFrame } from '@/components/page_frame';

import { ChatSearch, type ChatSearchHandle } from '@/chats/chat_search';
import { ChatTabsBar } from '@/chats/chat_tabs_bar';
import {
	buildInitialChatsModel,
	type ChatTabState,
	createEmptyTab,
	type InitialChatsModel,
	isScratchTab,
	MAX_TABS,
	writePersistedChatsPageState,
} from '@/chats/chat_tabs_persist';
import { ConversationArea, type ConversationAreaHandle } from '@/chats/conversation/conversation_area';
import { hydrateConversation, initConversation } from '@/chats/conversation/hydration_helper';

type NormalizedTabsResult = {
	tabs: ChatTabState[];
	removedTabIds: string[];
	addedScratchTabId: string | null;
};

function toTimestampMap(source?: Record<string, number>): Map<string, number> {
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

	// Never evict scratch; prefer not to evict busy, but allow if needed.
	const base = current.filter(t => t.tabId !== activeId && !isScratchTab(t));
	const nonBusy = base.filter(t => !t.isBusy);
	const candidates = nonBusy.length > 0 ? nonBusy : base;

	if (candidates.length === 0) return null;

	const ts = (id: string) => lastActivatedAt.get(id) ?? 0;
	return candidates.reduce((lru, tab) => (ts(tab.tabId) < ts(lru.tabId) ? tab : lru), candidates[0]);
}

function normalizeTabsForInvariants(
	current: ChatTabState[],
	activeId: string,
	lastActivatedAt: Map<string, number>
): NormalizedTabsResult {
	let next = current.slice();
	const removedTabIds = new Set<string>();
	let addedScratchTabId: string | null = null;

	const scratchTabs = next.filter(isScratchTab);
	const selected = next.find(t => t.tabId === activeId) ?? null;
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

function buildNormalizedInitialModel(): InitialChatsModel {
	const built = buildInitialChatsModel();

	const createdFallbackTab = built.tabs.length === 0 ? createEmptyTab() : null;
	const baseTabs = built.tabs.length > 0 ? built.tabs : createdFallbackTab ? [createdFallbackTab] : [];
	const baseSelectedTabId = baseTabs.some(tab => tab.tabId === built.selectedTabId)
		? built.selectedTabId
		: (baseTabs[0]?.tabId ?? '');

	const lastActivatedAt = toTimestampMap(built.lastActivatedAtByTab);
	const normalized = normalizeTabsForInvariants(baseTabs, baseSelectedTabId, lastActivatedAt);

	for (const removedTabId of normalized.removedTabIds) {
		lastActivatedAt.delete(removedTabId);
	}
	if (createdFallbackTab) {
		lastActivatedAt.set(createdFallbackTab.tabId, Date.now());
	}
	if (normalized.addedScratchTabId) {
		lastActivatedAt.set(normalized.addedScratchTabId, Date.now());
	}

	const existingTabIds = new Set(normalized.tabs.map(tab => tab.tabId));
	const selectedTabId = normalized.tabs.some(tab => tab.tabId === baseSelectedTabId)
		? baseSelectedTabId
		: (normalized.tabs[0]?.tabId ?? '');

	const scrollTopByTab: Record<string, number> = {};
	for (const [tabId, scrollTop] of Object.entries(built.scrollTopByTab ?? {})) {
		if (existingTabIds.has(tabId) && typeof scrollTop === 'number') {
			scrollTopByTab[tabId] = scrollTop;
		}
	}
	if (createdFallbackTab) {
		scrollTopByTab[createdFallbackTab.tabId] = 0;
	}
	if (normalized.addedScratchTabId) {
		scrollTopByTab[normalized.addedScratchTabId] = 0;
	}

	const lastActivatedAtByTab: Record<string, number> = {};
	for (const [tabId, ts] of lastActivatedAt.entries()) {
		if (existingTabIds.has(tabId)) {
			lastActivatedAtByTab[tabId] = ts;
		}
	}

	return {
		...built,
		tabs: normalized.tabs,
		selectedTabId,
		scrollTopByTab,
		lastActivatedAtByTab,
	};
}

function toStoreConversationMessage(message: ConversationMessage): StoreConversationMessage {
	const {
		uiContent: _uiContent,
		uiDebugDetails: _uiDebugDetails,
		uiReasoningContents: _uiReasoningContents,
		uiToolCalls: _uiToolCalls,
		uiToolOutputs: _uiToolOutputs,
		uiCitations: _uiCitations,
		...storeMessage
	} = message;

	return storeMessage;
}

function toStoreConversation(conversation: Conversation): StoreConversation {
	return {
		...conversation,
		messages: conversation.messages.map(toStoreConversationMessage),
	};
}

// eslint-disable-next-line no-restricted-exports
export default function ChatsPage() {
	const [initialModel] = useState<InitialChatsModel>(() => buildNormalizedInitialModel());
	const initialSelectedTabId = initialModel.selectedTabId;

	// ---------------- Tabs state ----------------
	const lastActivatedAtRef = useRef(toTimestampMap(initialModel.lastActivatedAtByTab));
	const touchTab = useCallback((tabId: string) => {
		lastActivatedAtRef.current.set(tabId, Date.now());
	}, []);

	const [tabs, setTabs] = useState<ChatTabState[]>(() => initialModel.tabs);
	const tabsRef = useRef(initialModel.tabs);

	const tabStore = useTabStore({ defaultSelectedId: initialSelectedTabId });
	const storeSelectedTabId = useStoreState(tabStore, 'selectedId') ?? initialSelectedTabId;

	const selectedTabId = useMemo(() => {
		if (tabs.some(t => t.tabId === storeSelectedTabId)) {
			return storeSelectedTabId;
		}
		return tabs[0]?.tabId ?? initialSelectedTabId;
	}, [initialSelectedTabId, storeSelectedTabId, tabs]);

	const selectedTabIdRef = useRef(selectedTabId);
	useEffect(() => {
		selectedTabIdRef.current = selectedTabId;
	}, [selectedTabId]);

	const selectTab = useCallback(
		(nextTabId: string) => {
			if (!nextTabId) return;
			selectedTabIdRef.current = nextTabId;
			tabStore.setSelectedId(nextTabId);
		},
		[tabStore]
	);

	const openConversationIds = useMemo(() => tabs.map(t => t.conversation.id).filter(Boolean), [tabs]);

	// ---------------- Conversation area (conversation runtime + UI) ----------------
	const conversationAreaRef = useRef<ConversationAreaHandle | null>(null);

	// ---------------- UI refs ----------------
	const searchRef = useRef<ChatSearchHandle | null>(null);

	// ---------------- Persistence scratch state ----------------
	const scrollTopSnapshotRef = useRef(initialModel.scrollTopByTab ?? {});
	const persistNowRef = useRef<() => void>(() => {});
	const saveQueueByTabRef = useRef(new Map<string, Promise<void>>());

	// ---------------- Helpers ----------------
	const disposeTabRuntime = useCallback((tabId: string) => {
		conversationAreaRef.current?.disposeTabRuntime(tabId);
		lastActivatedAtRef.current.delete(tabId);
	}, []);

	const enqueueSaveForTab = useCallback((tabId: string, operation: () => Promise<void>) => {
		const previous = saveQueueByTabRef.current.get(tabId) ?? Promise.resolve();

		const next = previous
			.catch(() => {
				// Swallow previous errors so later saves still run.
			})
			.then(operation)
			.catch((error: unknown) => {
				console.error('Failed to persist conversation state:', error);
			});

		saveQueueByTabRef.current.set(tabId, next);

		void next.finally(() => {
			if (saveQueueByTabRef.current.get(tabId) === next) {
				saveQueueByTabRef.current.delete(tabId);
			}
		});
	}, []);

	const commitTabs = useCallback((nextTabs: ChatTabState[]) => {
		tabsRef.current = nextTabs;
		setTabs(nextTabs);
	}, []);

	const normalizeAndCommitTabs = useCallback(
		(nextTabs: ChatTabState[]) => {
			const {
				tabs: normalizedTabs,
				removedTabIds,
				addedScratchTabId,
			} = normalizeTabsForInvariants(nextTabs, selectedTabIdRef.current, lastActivatedAtRef.current);

			if (removedTabIds.length > 0 || addedScratchTabId) {
				let nextScrollSnapshot = { ...scrollTopSnapshotRef.current };

				for (const removedTabId of removedTabIds) {
					disposeTabRuntime(removedTabId);
					nextScrollSnapshot = omitManyKeys(nextScrollSnapshot, [removedTabId]);
				}

				if (addedScratchTabId) {
					touchTab(addedScratchTabId);
					conversationAreaRef.current?.setScrollTopForTab(addedScratchTabId, 0);
					nextScrollSnapshot[addedScratchTabId] = 0;
				}

				scrollTopSnapshotRef.current = nextScrollSnapshot;
			}

			commitTabs(normalizedTabs);
			return normalizedTabs;
		},
		[commitTabs, disposeTabRuntime, touchTab]
	);

	const updateTab = useCallback(
		(tabId: string, updater: (t: ChatTabState) => ChatTabState) => {
			const current = tabsRef.current;
			const idx = current.findIndex(t => t.tabId === tabId);
			if (idx === -1) return;

			const next = current.slice();
			next[idx] = updater(next[idx]);
			normalizeAndCommitTabs(next);
		},
		[normalizeAndCommitTabs]
	);

	useEffect(() => {
		touchTab(selectedTabId);
	}, [selectedTabId, touchTab]);

	// ---------------- Search refresh key ----------------
	const [searchRefreshKey, setSearchRefreshKey] = useState(0);
	const bumpSearchKey = useCallback(async () => {
		await new Promise(resolve => setTimeout(resolve, 50));
		setSearchRefreshKey(k => k + 1);
	}, []);

	// ---------------- Persistence ----------------
	const persistNow = useCallback(() => {
		const tabsSnapshot = tabsRef.current.slice(0, MAX_TABS);

		const scrollObj =
			conversationAreaRef.current?.getScrollTopByTabSnapshot() ??
			scrollTopSnapshotRef.current ??
			({} as Record<string, number>);
		scrollTopSnapshotRef.current = scrollObj;

		const lruObj: Record<string, number> = {};
		for (const [k, v] of lastActivatedAtRef.current.entries()) lruObj[k] = v;

		writePersistedChatsPageState({
			v: 1,
			selectedTabId: selectedTabIdRef.current,
			tabs: tabsSnapshot.map(t => ({
				tabId: t.tabId,
				conversationId: t.conversation.id,
				title: t.conversation.title,
				isPersisted: t.isPersisted,
				manualTitleLocked: t.manualTitleLocked,
			})),
			scrollTopByTab: scrollObj,
			lastActivatedAtByTab: lruObj,
		});
	}, []);

	useEffect(() => {
		persistNowRef.current = persistNow;
	}, [persistNow]);

	const persistTimerRef = useRef<number | null>(null);
	const schedulePersist = useCallback(() => {
		if (persistTimerRef.current) window.clearTimeout(persistTimerRef.current);
		persistTimerRef.current = window.setTimeout(() => {
			persistTimerRef.current = null;
			persistNow();
		}, 250);
	}, [persistNow]);

	useEffect(() => {
		// Persist actual tab/selection changes, not just pagehide/unmount.
		// Without this, stale tab metadata (especially old conversation titles)
		// can survive across restarts and break title-based conversation lookup.
		schedulePersist();
	}, [schedulePersist, selectedTabId, tabs]);

	useEffect(() => {
		const onPageHide = () => {
			persistNow();
		};
		const onVis = () => {
			if (document.visibilityState === 'hidden') persistNow();
		};

		window.addEventListener('pagehide', onPageHide);
		document.addEventListener('visibilitychange', onVis);

		return () => {
			window.removeEventListener('pagehide', onPageHide);
			document.removeEventListener('visibilitychange', onVis);
		};
	}, [persistNow]);

	// On unmount: flush persistence (route navigation safety)
	useEffect(() => {
		return () => {
			persistNow();
		};
	}, [persistNow]);

	// ---------------- Save conversation ----------------
	const saveUpdatedConversation = useCallback(
		(tabId: string, updatedConv: Conversation, titleWasExternallyChanged = false) => {
			const tab = tabsRef.current.find(t => t.tabId === tabId);
			if (!tab) return;

			let newTitle = updatedConv.title;

			const allowAutoTitle = !titleWasExternallyChanged && !tab.manualTitleLocked;
			if (allowAutoTitle && updatedConv.messages.length <= 4) {
				const userMessages = updatedConv.messages.filter(m => m.role === RoleEnum.User);
				if (userMessages.length === 1) {
					newTitle = generateTitle(userMessages[0].uiContent).title;
				} else if (userMessages.length === 2) {
					const c1 = generateTitle(userMessages[0].uiContent);
					const c2 = generateTitle(userMessages[1].uiContent);
					newTitle = c2.score > c1.score ? c2.title : c1.title;
				}
				newTitle = sanitizeConversationTitle(newTitle);
			}

			const titleChangedByFunction = newTitle !== updatedConv.title;
			if (titleChangedByFunction) updatedConv.title = newTitle;

			const titleChanged = titleWasExternallyChanged || titleChangedByFunction;
			const storeConversation = toStoreConversation(updatedConv);
			const needsFullSave = !tab.isPersisted || titleChanged;

			enqueueSaveForTab(tabId, async () => {
				if (needsFullSave) {
					await conversationStoreAPI.putConversation(storeConversation);
					await bumpSearchKey();

					// Tab restore metadata (title / persisted identity) must be flushed
					// only after the full conversation save has succeeded.
					persistNowRef.current();
					return;
				}

				await conversationStoreAPI.putMessagesToConversation(
					storeConversation.id,
					storeConversation.title,
					storeConversation.messages
				);
			});

			updateTab(tabId, t => ({
				...t,
				conversation: { ...updatedConv, messages: [...updatedConv.messages] },
				isPersisted: true,
				manualTitleLocked: titleWasExternallyChanged ? true : t.manualTitleLocked,
				isHydrating: false,
			}));
		},
		[bumpSearchKey, enqueueSaveForTab, updateTab]
	);

	// ---------------- Tab actions ----------------
	const openNewTab = useCallback(() => {
		// Always go to the single scratch tab.
		const scratch = tabsRef.current.find(isScratchTab) ?? null;
		const targetId = scratch?.tabId ?? selectedTabIdRef.current;
		if (!targetId) return;

		selectTab(targetId);
		requestAnimationFrame(() => {
			const area = conversationAreaRef.current;
			if (!area) return;
			void area.resetComposerForNewConversation(targetId).finally(() => {
				area.focusInput(targetId);
			});
		});
	}, [selectTab]);

	const closeTab = useCallback(
		(tabId: string) => {
			const current = tabsRef.current;
			const idx = current.findIndex(t => t.tabId === tabId);
			if (idx === -1) return;

			const wasActive = tabId === selectedTabIdRef.current;

			// Stop any async work / cleanup runtime refs immediately.
			disposeTabRuntime(tabId);

			let nextScrollSnapshot = { ...scrollTopSnapshotRef.current };
			nextScrollSnapshot = omitManyKeys(nextScrollSnapshot, [tabId]);

			scrollTopSnapshotRef.current = nextScrollSnapshot;

			const baseNextTabs = current.filter(t => t.tabId !== tabId);
			const normalizedNextTabs = normalizeAndCommitTabs(baseNextTabs);

			if (!wasActive) return;

			const right = current[idx + 1];
			const left = idx > 0 ? current[idx - 1] : undefined;
			const preferredNextSelectedId =
				(left && left.tabId !== tabId ? left.tabId : right && right.tabId !== tabId ? right.tabId : '') ||
				normalizedNextTabs[0]?.tabId ||
				'';

			if (!preferredNextSelectedId) return;

			// Keep refs immediately valid for persistence and callbacks.
			selectedTabIdRef.current = preferredNextSelectedId;

			requestAnimationFrame(() => {
				const currentTabs = tabsRef.current;
				const nextSelectedId = currentTabs.some(t => t.tabId === preferredNextSelectedId)
					? preferredNextSelectedId
					: currentTabs[0]?.tabId;

				if (nextSelectedId) {
					selectTab(nextSelectedId);
				}
			});
		},
		[disposeTabRuntime, normalizeAndCommitTabs, selectTab]
	);

	const cycleTabBy = useCallback(
		(delta: number) => {
			const current = tabsRef.current;
			if (current.length < 2) return;

			const activeId = selectedTabIdRef.current;
			const idx = current.findIndex(t => t.tabId === activeId);
			const from = idx >= 0 ? idx : 0;
			const nextIndex = (from + delta + current.length) % current.length;
			const nextId = current[nextIndex]?.tabId;
			if (!nextId) return;

			selectTab(nextId);
			requestAnimationFrame(() => conversationAreaRef.current?.focusInput(nextId));
		},
		[selectTab]
	);

	const selectNextTab = useCallback(() => {
		cycleTabBy(1);
	}, [cycleTabBy]);

	const selectPrevTab = useCallback(() => {
		cycleTabBy(-1);
	}, [cycleTabBy]);

	// ---------------- Rename ----------------
	const renameTabTitle = useCallback(
		(tabId: string, newTitle: string) => {
			const tab = tabsRef.current.find(t => t.tabId === tabId);
			if (!tab) return;

			const sanitized = sanitizeConversationTitle(newTitle.trim());
			if (!sanitized || sanitized === tab.conversation.title) return;

			const updatedConv: Conversation = {
				...tab.conversation,
				title: sanitized,
				modifiedAt: new Date(),
			};

			saveUpdatedConversation(tabId, updatedConv, true);
		},
		[saveUpdatedConversation]
	);

	// ---------------- Search select behavior ----------------
	const loadConversationIntoTab = useCallback(
		async (tabId: string, item: ConversationSearchItem) => {
			updateTab(tabId, t => ({
				...t,
				isHydrating: true,
				isBusy: false,
				editingMessageId: null,
			}));

			try {
				const selectedChat = await conversationStoreAPI.getConversation(item.id, item.title, true);
				if (!selectedChat) {
					updateTab(tabId, t => ({
						...t,
						isHydrating: false,
						isBusy: false,
						editingMessageId: null,
					}));
					return;
				}

				const hydrated = hydrateConversation(selectedChat);

				// Reset stream buffer for that tab
				conversationAreaRef.current?.clearStreamForTab(tabId);

				updateTab(tabId, t => ({
					...t,
					conversation: hydrated,
					isPersisted: true,
					manualTitleLocked: false,
					editingMessageId: null,
					isBusy: false,
					isHydrating: true,
				}));

				// Reset scroll position for that tab
				conversationAreaRef.current?.resetScrollToTop(tabId);

				requestAnimationFrame(() => {
					conversationAreaRef.current?.syncComposerFromConversation(tabId, hydrated);
					updateTab(tabId, t => ({
						...t,
						isHydrating: false,
						isBusy: false,
						editingMessageId: null,
					}));
				});
			} catch (e) {
				console.error(e);
				updateTab(tabId, t => ({
					...t,
					isHydrating: false,
					isBusy: false,
					editingMessageId: null,
				}));
			}
		},
		[updateTab]
	);

	const handleSelectConversation = useCallback(
		async (item: ConversationSearchItem) => {
			// If already open, just activate it.
			const already = tabsRef.current.find(t => t.conversation.id === item.id);
			if (already) {
				selectTab(already.tabId);
				return;
			}

			// Always load into scratch tab; after load it becomes a normal tab and
			// mutation-time normalization creates the new scratch on the right.
			const scratch = tabsRef.current.find(isScratchTab);
			const targetId = scratch?.tabId ?? selectedTabIdRef.current;
			if (!targetId) return;

			selectTab(targetId);
			await loadConversationIntoTab(targetId, item);
			requestAnimationFrame(() => conversationAreaRef.current?.focusInput(targetId));
		},
		[loadConversationIntoTab, selectTab]
	);

	// ---------------- Rehydrate tabs on mount (from Go-backed store) ----------------
	useEffect(() => {
		if (!initialModel.restoredFromStorage) return;

		let cancelled = false;

		(async () => {
			const snapshot = tabsRef.current.slice(0, MAX_TABS);

			for (const t of snapshot) {
				if (cancelled) return;
				if (!t.isPersisted) continue;
				if (!t.conversation.id) continue;

				try {
					const stored = await conversationStoreAPI.getConversation(t.conversation.id, t.conversation.title, true);
					if (cancelled) return;

					if (!stored) {
						// Conversation missing -> degrade to scratch; mutation-time normalization will clean up.
						updateTab(t.tabId, prev => ({
							...prev,
							isBusy: false,
							isHydrating: false,
							isPersisted: false,
							manualTitleLocked: false,
							editingMessageId: null,
							conversation: initConversation(),
						}));
						continue;
					}

					const hydrated = hydrateConversation(stored);

					// Clear transient streaming buffer for restored tabs
					conversationAreaRef.current?.clearStreamForTab(t.tabId);

					updateTab(t.tabId, prev => ({
						...prev,
						isBusy: false,
						isHydrating: true,
						editingMessageId: null,
						isPersisted: true,
						conversation: hydrated,
					}));

					requestAnimationFrame(() => {
						if (cancelled) return;
						conversationAreaRef.current?.syncComposerFromConversation(t.tabId, hydrated);
						updateTab(t.tabId, prev => ({
							...prev,
							isBusy: false,
							isHydrating: false,
							editingMessageId: null,
						}));
					});
				} catch (e) {
					console.error(e);
					updateTab(t.tabId, prev => ({
						...prev,
						isBusy: false,
						isHydrating: false,
						isPersisted: false,
						manualTitleLocked: false,
						editingMessageId: null,
						conversation: initConversation(),
					}));
				}
			}
		})().catch(console.error);

		return () => {
			cancelled = true;
		};
	}, [initialModel.restoredFromStorage, updateTab]);

	// ---------------- Shortcuts ----------------
	const [shortcutConfig] = useState(defaultShortcutConfig);

	useChatShortcuts({
		config: shortcutConfig,
		isBusy: false, // new tabs allowed even if another tab is busy
		handlers: {
			newChat: () => {
				openNewTab();
			},
			closeChat: () => {
				closeTab(selectedTabIdRef.current);
			},
			nextChat: () => {
				selectNextTab();
			},
			previousChat: () => {
				selectPrevTab();
			},
			focusSearch: () => searchRef.current?.focusInput(),
			focusInput: () => conversationAreaRef.current?.focusInput(selectedTabIdRef.current),
			insertTemplate: () => conversationAreaRef.current?.openTemplateMenu(selectedTabIdRef.current),
			insertTool: () => conversationAreaRef.current?.openToolMenu(selectedTabIdRef.current),
			insertAttachment: () => conversationAreaRef.current?.openAttachmentMenu(selectedTabIdRef.current),
		},
	});

	// ---------------- Titlebar: search only (like before) ----------------
	useTitleBarContent(
		{
			center: (
				<div className="mx-auto flex w-4/5 items-center justify-center">
					<ChatSearch
						ref={searchRef}
						compact={true}
						onSelectConversation={handleSelectConversation}
						refreshKey={searchRefreshKey}
						openConversationIds={openConversationIds}
					/>
				</div>
			),
		},
		[handleSelectConversation, openConversationIds, searchRefreshKey]
	);

	// ---------------- Export (active tab) ----------------
	const getConversationForExport = useCallback(async (): Promise<string> => {
		const t = tabsRef.current.find(x => x.tabId === selectedTabIdRef.current);
		if (!t) return JSON.stringify(null, null, 2);

		const selectedChat = await conversationStoreAPI.getConversation(t.conversation.id, t.conversation.title, true);
		return JSON.stringify(selectedChat ?? null, null, 2);
	}, []);

	// ---------------- Render helpers ----------------
	const tabBarItems = useMemo(
		() =>
			tabs.map(t => ({
				tabId: t.tabId,
				title: t.conversation.title,
				isBusy: t.isBusy || t.isHydrating,
				isEmpty: t.conversation.messages.length === 0,
				renameEnabled: t.conversation.messages.length > 0,
			})),
		[tabs]
	);

	return (
		<PageFrame contentScrollable={false}>
			<div className="grid h-full w-full grid-rows-[auto_1fr_auto] overflow-hidden">
				<div className="relative row-start-1 row-end-2 min-h-0 min-w-0 p-0">
					<ChatTabsBar
						store={tabStore}
						selectedTabId={selectedTabId}
						tabs={tabBarItems}
						maxTabs={MAX_TABS}
						onNewTab={openNewTab}
						onCloseTab={closeTab}
						onRenameTab={renameTabTitle}
						getConversationForExport={getConversationForExport}
					/>
				</div>

				<ConversationArea
					ref={conversationAreaRef}
					tabs={tabs}
					selectedTabId={selectedTabId}
					shortcutConfig={shortcutConfig}
					initialScrollTopByTab={initialModel.scrollTopByTab}
					updateTab={updateTab}
					saveUpdatedConversation={saveUpdatedConversation}
				/>
			</div>
		</PageFrame>
	);
}
