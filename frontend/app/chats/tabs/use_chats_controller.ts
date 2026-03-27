import { type RefObject, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { useStoreState } from '@ariakit/react';
import { useTabStore } from '@ariakit/react/tab';

import type { Conversation, ConversationSearchItem } from '@/spec/conversation';
import { RoleEnum } from '@/spec/inference';

import { defaultShortcutConfig, type ShortcutConfig, useChatShortcuts } from '@/lib/keyboard_shortcuts';
import { omitManyKeys } from '@/lib/obj_utils';
import { sanitizeConversationTitle } from '@/lib/text_utils';
import { generateTitle } from '@/lib/title_utils';

import { conversationStoreAPI } from '@/apis/baseapi';

import type { ConversationAreaHandle } from '@/chats/conversation/conversation_area';
import { hydrateConversation, toStoreConversation } from '@/chats/conversation/conversation_persistence_mapper';
import { initConversation } from '@/chats/conversation/hydration_helper';
import type { ConversationSearchHandle } from '@/chats/search/conversation_search';
import {
	type ChatTabState,
	createEmptyTab,
	isScratchTab,
	MAX_TABS,
	normalizeTabsForInvariants,
	toTimestampMap,
} from '@/chats/tabs/tabs_model';
import {
	buildInitialChatsModel,
	type InitialChatsModel,
	writePersistedChatsPageState,
} from '@/chats/tabs/tabs_persistence';

type UseChatsControllerArgs = {
	conversationAreaRef: RefObject<ConversationAreaHandle | null>;
	searchRef: RefObject<ConversationSearchHandle | null>;
};

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

	const topItemIndexByTab: Record<string, number> = {};
	for (const [tabId, topItemIndex] of Object.entries(built.topItemIndexByTab ?? {})) {
		if (existingTabIds.has(tabId) && typeof topItemIndex === 'number') {
			topItemIndexByTab[tabId] = topItemIndex;
		}
	}
	if (createdFallbackTab) {
		topItemIndexByTab[createdFallbackTab.tabId] = 0;
	}
	if (normalized.addedScratchTabId) {
		topItemIndexByTab[normalized.addedScratchTabId] = 0;
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
		topItemIndexByTab,
		lastActivatedAtByTab,
	};
}

export function useChatsController({ conversationAreaRef, searchRef }: UseChatsControllerArgs) {
	const [initialModel] = useState<InitialChatsModel>(() => buildNormalizedInitialModel());
	const initialSelectedTabId = initialModel.selectedTabId;

	const lastActivatedAtRef = useRef(toTimestampMap(initialModel.lastActivatedAtByTab));
	const touchTab = useCallback((tabId: string) => {
		lastActivatedAtRef.current.set(tabId, Date.now());
	}, []);

	const [tabs, setTabs] = useState<ChatTabState[]>(() => initialModel.tabs);
	const tabsRef = useRef(initialModel.tabs);

	const tabStore = useTabStore({ defaultSelectedId: initialSelectedTabId });
	const storeSelectedTabId = useStoreState(tabStore, 'selectedId') ?? initialSelectedTabId;

	const selectedTabId = useMemo(() => {
		if (tabs.some(tab => tab.tabId === storeSelectedTabId)) {
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

	const openConversationIds = useMemo(() => tabs.map(tab => tab.conversation.id).filter(Boolean), [tabs]);

	const scrollTopSnapshotRef = useRef(initialModel.scrollTopByTab ?? {});
	const topItemIndexSnapshotRef = useRef(initialModel.topItemIndexByTab ?? {});
	const persistNowRef = useRef<() => void>(() => {});
	const schedulePersistRef = useRef<() => void>(() => {});
	const saveQueueByTabRef = useRef(new Map<string, Promise<void>>());

	const disposeTabRuntime = useCallback(
		(tabId: string) => {
			conversationAreaRef.current?.disposeTabRuntime(tabId);
			lastActivatedAtRef.current.delete(tabId);
		},
		[conversationAreaRef]
	);

	const enqueueSaveForTab = useCallback((tabId: string, operation: () => Promise<void>) => {
		const previous = saveQueueByTabRef.current.get(tabId) ?? Promise.resolve();

		const next = previous
			.catch(() => {
				// ignore previous error so next save still runs
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
		schedulePersistRef.current();
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
				let nextTopItemIndexSnapshot = { ...topItemIndexSnapshotRef.current };

				for (const removedTabId of removedTabIds) {
					disposeTabRuntime(removedTabId);
					nextScrollSnapshot = omitManyKeys(nextScrollSnapshot, [removedTabId]);
					nextTopItemIndexSnapshot = omitManyKeys(nextTopItemIndexSnapshot, [removedTabId]);
				}

				if (addedScratchTabId) {
					touchTab(addedScratchTabId);
					conversationAreaRef.current?.setScrollTopForTab(addedScratchTabId, 0);
					nextScrollSnapshot[addedScratchTabId] = 0;
					nextTopItemIndexSnapshot[addedScratchTabId] = 0;
				}

				scrollTopSnapshotRef.current = nextScrollSnapshot;
				topItemIndexSnapshotRef.current = nextTopItemIndexSnapshot;
			}

			commitTabs(normalizedTabs);
			return normalizedTabs;
		},
		[commitTabs, conversationAreaRef, disposeTabRuntime, touchTab]
	);

	const updateTab = useCallback(
		(tabId: string, updater: (tab: ChatTabState) => ChatTabState) => {
			const current = tabsRef.current;
			const idx = current.findIndex(tab => tab.tabId === tabId);
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

	const [searchRefreshKey, setSearchRefreshKey] = useState(0);
	const bumpSearchKey = useCallback(async () => {
		await new Promise(resolve => setTimeout(resolve, 50));
		setSearchRefreshKey(key => key + 1);
	}, []);

	const persistNow = useCallback(() => {
		const tabsSnapshot = tabsRef.current.slice(0, MAX_TABS);

		const scrollObj: Record<string, number> = {
			...(scrollTopSnapshotRef.current ?? {}),
			...(conversationAreaRef.current?.getScrollTopByTabSnapshot() ?? {}),
		};
		scrollTopSnapshotRef.current = scrollObj;

		const topItemIndexObj: Record<string, number> = {
			...(topItemIndexSnapshotRef.current ?? {}),
			...(conversationAreaRef.current?.getTopItemIndexByTabSnapshot() ?? {}),
		};
		topItemIndexSnapshotRef.current = topItemIndexObj;

		const lruObj: Record<string, number> = {};
		for (const [key, value] of lastActivatedAtRef.current.entries()) {
			lruObj[key] = value;
		}

		writePersistedChatsPageState({
			v: 1,
			selectedTabId: selectedTabIdRef.current,
			tabs: tabsSnapshot.map(tab => ({
				tabId: tab.tabId,
				conversationId: tab.conversation.id,
				title: tab.conversation.title,
				isPersisted: tab.isPersisted,
				manualTitleLocked: tab.manualTitleLocked,
			})),
			scrollTopByTab: scrollObj,
			topItemIndexByTab: topItemIndexObj,
			lastActivatedAtByTab: lruObj,
		});
	}, [conversationAreaRef]);

	useEffect(() => {
		persistNowRef.current = persistNow;
	}, [persistNow]);

	const persistTimerRef = useRef<number | null>(null);

	const schedulePersist = useCallback(() => {
		if (persistTimerRef.current !== null) {
			window.clearTimeout(persistTimerRef.current);
		}

		persistTimerRef.current = window.setTimeout(() => {
			persistTimerRef.current = null;
			persistNow();
		}, 250);
	}, [persistNow]);

	useEffect(() => {
		schedulePersistRef.current = schedulePersist;
	}, [schedulePersist]);

	useEffect(() => {
		// eslint-disable-next-line react-you-might-not-need-an-effect/no-pass-live-state-to-parent
		schedulePersist();
	}, [schedulePersist, selectedTabId]);

	useEffect(() => {
		return () => {
			if (persistTimerRef.current !== null) {
				window.clearTimeout(persistTimerRef.current);
			}
		};
	}, []);

	useEffect(() => {
		const onPageHide = () => {
			persistNow();
		};

		const onVisibilityChange = () => {
			if (document.visibilityState === 'hidden') {
				persistNow();
			}
		};

		window.addEventListener('pagehide', onPageHide);
		document.addEventListener('visibilitychange', onVisibilityChange);

		return () => {
			window.removeEventListener('pagehide', onPageHide);
			document.removeEventListener('visibilitychange', onVisibilityChange);
		};
	}, [persistNow]);

	useEffect(() => {
		return () => {
			persistNow();
		};
	}, [persistNow]);

	const saveUpdatedConversation = useCallback(
		(tabId: string, updatedConv: Conversation, titleWasExternallyChanged = false) => {
			const tab = tabsRef.current.find(t => t.tabId === tabId);
			if (!tab) return;

			let newTitle = updatedConv.title;

			const allowAutoTitle = !titleWasExternallyChanged && !tab.manualTitleLocked;
			if (allowAutoTitle && updatedConv.messages.length <= 4) {
				const userMessages = updatedConv.messages.filter(message => message.role === RoleEnum.User);

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
			if (titleChangedByFunction) {
				updatedConv.title = newTitle;
			}

			const titleChanged = titleWasExternallyChanged || titleChangedByFunction;
			const storeConversation = toStoreConversation(updatedConv);
			const needsFullSave = !tab.isPersisted || titleChanged;

			enqueueSaveForTab(tabId, async () => {
				if (needsFullSave) {
					await conversationStoreAPI.putConversation(storeConversation);
					await bumpSearchKey();
					persistNowRef.current();
					return;
				}

				await conversationStoreAPI.putMessagesToConversation(
					storeConversation.id,
					storeConversation.title,
					storeConversation.messages
				);
			});

			updateTab(tabId, current => ({
				...current,
				conversation: { ...updatedConv, messages: [...updatedConv.messages] },
				isPersisted: true,
				manualTitleLocked: titleWasExternallyChanged ? true : current.manualTitleLocked,
				isHydrating: false,
			}));
		},
		[bumpSearchKey, enqueueSaveForTab, updateTab]
	);

	const openNewTab = useCallback(() => {
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
	}, [conversationAreaRef, selectTab]);

	const closeTab = useCallback(
		(tabId: string) => {
			const current = tabsRef.current;
			const idx = current.findIndex(tab => tab.tabId === tabId);
			if (idx === -1) return;

			const wasActive = tabId === selectedTabIdRef.current;

			disposeTabRuntime(tabId);

			scrollTopSnapshotRef.current = omitManyKeys({ ...scrollTopSnapshotRef.current }, [tabId]);
			topItemIndexSnapshotRef.current = omitManyKeys({ ...topItemIndexSnapshotRef.current }, [tabId]);

			const baseNextTabs = current.filter(tab => tab.tabId !== tabId);

			if (wasActive) {
				const right = current[idx + 1];
				const left = idx > 0 ? current[idx - 1] : undefined;

				const preferredNextSelectedId =
					(left && left.tabId !== tabId ? left.tabId : right && right.tabId !== tabId ? right.tabId : '') ||
					baseNextTabs[0]?.tabId ||
					'';

				if (preferredNextSelectedId) {
					selectTab(preferredNextSelectedId);
				}
			}

			const normalizedNextTabs = normalizeAndCommitTabs(baseNextTabs);

			if (!wasActive) return;

			const finalNextSelectedId = normalizedNextTabs.some(tab => tab.tabId === selectedTabIdRef.current)
				? selectedTabIdRef.current
				: (normalizedNextTabs[0]?.tabId ?? '');

			if (finalNextSelectedId && finalNextSelectedId !== selectedTabIdRef.current) {
				selectTab(finalNextSelectedId);
			}
		},
		[disposeTabRuntime, normalizeAndCommitTabs, selectTab]
	);

	const cycleTabBy = useCallback(
		(delta: number) => {
			const current = tabsRef.current;
			if (current.length < 2) return;

			const activeId = selectedTabIdRef.current;
			const idx = current.findIndex(tab => tab.tabId === activeId);
			const from = idx >= 0 ? idx : 0;
			const nextIndex = (from + delta + current.length) % current.length;
			const nextId = current[nextIndex]?.tabId;
			if (!nextId) return;

			selectTab(nextId);
			requestAnimationFrame(() => {
				conversationAreaRef.current?.focusInput(nextId);
			});
		},
		[conversationAreaRef, selectTab]
	);

	const selectNextTab = useCallback(() => {
		cycleTabBy(1);
	}, [cycleTabBy]);

	const selectPrevTab = useCallback(() => {
		cycleTabBy(-1);
	}, [cycleTabBy]);

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

	const loadConversationIntoTab = useCallback(
		async (tabId: string, item: ConversationSearchItem) => {
			updateTab(tabId, tab => ({
				...tab,
				isHydrating: true,
				isBusy: false,
				editingMessageId: null,
			}));

			try {
				const selectedChat = await conversationStoreAPI.getConversation(item.id, item.title, true);
				if (!selectedChat) {
					updateTab(tabId, tab => ({
						...tab,
						isHydrating: false,
						isBusy: false,
						editingMessageId: null,
					}));
					return;
				}

				const hydrated = hydrateConversation(selectedChat);

				conversationAreaRef.current?.clearStreamForTab(tabId);

				updateTab(tabId, tab => ({
					...tab,
					conversation: hydrated,
					isPersisted: true,
					manualTitleLocked: false,
					editingMessageId: null,
					isBusy: false,
					isHydrating: true,
				}));

				conversationAreaRef.current?.resetScrollToTop(tabId);

				requestAnimationFrame(() => {
					conversationAreaRef.current?.syncComposerFromConversation(tabId, hydrated);

					updateTab(tabId, tab => ({
						...tab,
						isHydrating: false,
						isBusy: false,
						editingMessageId: null,
					}));
				});
			} catch (error) {
				console.error(error);
				updateTab(tabId, tab => ({
					...tab,
					isHydrating: false,
					isBusy: false,
					editingMessageId: null,
				}));
			}
		},
		[conversationAreaRef, updateTab]
	);

	const handleSelectConversation = useCallback(
		async (item: ConversationSearchItem) => {
			const alreadyOpen = tabsRef.current.find(tab => tab.conversation.id === item.id);
			if (alreadyOpen) {
				selectTab(alreadyOpen.tabId);
				return;
			}

			const scratch = tabsRef.current.find(isScratchTab);
			const targetId = scratch?.tabId ?? selectedTabIdRef.current;
			if (!targetId) return;

			selectTab(targetId);
			await loadConversationIntoTab(targetId, item);

			requestAnimationFrame(() => {
				conversationAreaRef.current?.focusInput(targetId);
			});
		},
		[conversationAreaRef, loadConversationIntoTab, selectTab]
	);

	useEffect(() => {
		if (!initialModel.restoredFromStorage) return;

		let cancelled = false;

		(async () => {
			const snapshot = tabsRef.current.slice(0, MAX_TABS);

			for (const tab of snapshot) {
				if (cancelled) return;
				if (!tab.isPersisted) continue;
				if (!tab.conversation.id) continue;

				try {
					const stored = await conversationStoreAPI.getConversation(tab.conversation.id, tab.conversation.title, true);
					if (cancelled) return;

					if (!stored) {
						updateTab(tab.tabId, prev => ({
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

					conversationAreaRef.current?.clearStreamForTab(tab.tabId);

					updateTab(tab.tabId, prev => ({
						...prev,
						isBusy: false,
						isHydrating: true,
						editingMessageId: null,
						isPersisted: true,
						conversation: hydrated,
					}));

					requestAnimationFrame(() => {
						if (cancelled) return;

						conversationAreaRef.current?.syncComposerFromConversation(tab.tabId, hydrated);
						updateTab(tab.tabId, prev => ({
							...prev,
							isBusy: false,
							isHydrating: false,
							editingMessageId: null,
						}));
					});
				} catch (error) {
					console.error(error);
					updateTab(tab.tabId, prev => ({
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
	}, [conversationAreaRef, initialModel.restoredFromStorage, updateTab]);

	// eslint-disable-next-line @typescript-eslint/no-unnecessary-type-arguments
	const [shortcutConfig] = useState<ShortcutConfig>(defaultShortcutConfig);

	useChatShortcuts({
		config: shortcutConfig,
		isBusy: false,
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
			focusSearch: () => {
				searchRef.current?.focusInput();
			},
			focusInput: () => {
				conversationAreaRef.current?.focusInput(selectedTabIdRef.current);
			},
			insertTemplate: () => {
				conversationAreaRef.current?.openTemplateMenu(selectedTabIdRef.current);
			},
			insertTool: () => {
				conversationAreaRef.current?.openToolMenu(selectedTabIdRef.current);
			},
			insertAttachment: () => {
				conversationAreaRef.current?.openAttachmentMenu(selectedTabIdRef.current);
			},
		},
	});

	const getConversationForExport = useCallback(async (): Promise<string> => {
		const tab = tabsRef.current.find(current => current.tabId === selectedTabIdRef.current);
		if (!tab) return JSON.stringify(null, null, 2);

		const selectedChat = await conversationStoreAPI.getConversation(tab.conversation.id, tab.conversation.title, true);
		return JSON.stringify(selectedChat ?? null, null, 2);
	}, []);

	const tabBarItems = useMemo(
		() =>
			tabs.map(tab => ({
				tabId: tab.tabId,
				title: tab.conversation.title,
				isBusy: tab.isBusy || tab.isHydrating,
				isEmpty: tab.conversation.messages.length === 0,
				renameEnabled: tab.conversation.messages.length > 0,
			})),
		[tabs]
	);

	return {
		tabStore,
		tabs,
		selectedTabId,
		initialScrollTopByTab: initialModel.scrollTopByTab,
		initialTopItemIndexByTab: initialModel.topItemIndexByTab,
		shortcutConfig,
		updateTab,
		saveUpdatedConversation,
		openNewTab,
		closeTab,
		renameTabTitle,
		handleSelectConversation,
		getConversationForExport,
		tabBarItems,
		openConversationIds,
		searchRefreshKey,
		maxTabs: MAX_TABS,
	};
}
