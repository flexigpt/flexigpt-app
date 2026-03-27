import { type RefObject, useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';

import { type ListRange, type VirtuosoHandle } from 'react-virtuoso';

import type { Conversation } from '@/spec/conversation';

import type { SavedVirtuosoState } from '@/chats/conversation/virtuoso_utils';
import type { ChatTabState } from '@/chats/tabs/tabs_model';

function getConversationModifiedAtMs(conversation?: Conversation): number {
	const raw = conversation?.modifiedAt;
	if (!raw) return 0;

	const ms = raw instanceof Date ? raw.getTime() : new Date(raw).getTime();
	return Number.isFinite(ms) ? ms : 0;
}

function getSavedVirtuosoStateRecord(
	stateByTabRef: RefObject<Map<string, SavedVirtuosoState>>,
	tabId: string
): SavedVirtuosoState | undefined {
	return stateByTabRef.current?.get(tabId);
}

function getMapNumberValue(mapRef: RefObject<Map<string, number>>, key: string): number | undefined {
	return mapRef.current?.get(key);
}

type UseScrollRestoreArgs = {
	tabsRef: RefObject<ChatTabState[]>;
	selectedTabId: string;
	selectedTabIdRef: RefObject<string>;
	activeTabIsHydrating: boolean;
	messageCount: number;
	activeConversationModifiedAtMs: number;
	activeLastMessageId: string | null;
	initialScrollTopByTab?: Record<string, number>;
	initialTopItemIndexByTab?: Record<string, number>;
};

export function useScrollRestore({
	tabsRef,
	selectedTabId,
	selectedTabIdRef,
	activeTabIsHydrating,
	messageCount,
	activeConversationModifiedAtMs,
	activeLastMessageId,
	initialScrollTopByTab,
	initialTopItemIndexByTab,
}: UseScrollRestoreArgs) {
	const virtuosoRef = useRef<VirtuosoHandle>(null);
	const virtuosoStateByTabRef = useRef(new Map<string, SavedVirtuosoState>());
	const restoredInitialScrollByTabRef = useRef(new Set<string>());

	const scrollTopByTabRef = useRef(new Map<string, number>());
	const topItemIndexByTabRef = useRef(new Map<string, number>());

	const seededScrollFromStorageRef = useRef(false);
	useEffect(() => {
		if (!seededScrollFromStorageRef.current) {
			seededScrollFromStorageRef.current = true;
			if (initialScrollTopByTab) {
				for (const [tabId, top] of Object.entries(initialScrollTopByTab)) {
					if (typeof top === 'number') {
						scrollTopByTabRef.current.set(tabId, top);
					}
				}
			}
		}
	}, [initialScrollTopByTab]);

	const seededTopItemIndexFromStorageRef = useRef(false);
	useEffect(() => {
		if (!seededTopItemIndexFromStorageRef.current) {
			seededTopItemIndexFromStorageRef.current = true;
			if (initialTopItemIndexByTab) {
				for (const [tabId, topItemIndex] of Object.entries(initialTopItemIndexByTab)) {
					if (typeof topItemIndex === 'number') {
						topItemIndexByTabRef.current.set(tabId, topItemIndex);
					}
				}
			}
		}
	}, [initialTopItemIndexByTab]);

	const shouldPreserveRestoreSeed = useCallback(
		(tabId: string) => {
			if (restoredInitialScrollByTabRef.current.has(tabId)) return false;

			const tab = tabsRef.current.find(t => t.tabId === tabId);
			if (!tab) return false;

			const savedScrollTop = scrollTopByTabRef.current.get(tabId) ?? 0;
			const savedTopItemIndex = topItemIndexByTabRef.current.get(tabId) ?? 0;

			return tab.isHydrating || savedScrollTop > 0 || savedTopItemIndex > 0;
		},
		[tabsRef]
	);

	const saveVirtuosoStateForTab = useCallback(
		(tabId: string, handle: VirtuosoHandle | null) => {
			if (!handle) return;

			const tab = tabsRef.current.find(t => t.tabId === tabId);
			const tabMessages = tab?.conversation.messages ?? [];
			const savedStateMeta = {
				modifiedAtMs: getConversationModifiedAtMs(tab?.conversation),
				messageCount: tabMessages.length,
				lastMessageId: tabMessages[tabMessages.length - 1]?.id ?? null,
			};

			handle.getState(state => {
				virtuosoStateByTabRef.current.set(tabId, {
					snapshot: state,
					...savedStateMeta,
				});
			});
		},
		[tabsRef]
	);

	const [isAtBottom, setIsAtBottom] = useState(true);
	const [isAtTop, setIsAtTop] = useState(true);

	const scrollListenerCleanupRef = useRef<(() => void) | null>(null);

	const handleScrollerRef = useCallback(
		(el: HTMLElement | Window | null) => {
			scrollListenerCleanupRef.current?.();
			scrollListenerCleanupRef.current = null;

			const htmlEl = el instanceof HTMLElement ? el : null;
			if (!htmlEl) return;

			const tabId = selectedTabIdRef.current;
			const handler = () => {
				if (!tabId) return;
				if (shouldPreserveRestoreSeed(tabId)) return;
				scrollTopByTabRef.current.set(tabId, htmlEl.scrollTop);
			};

			htmlEl.addEventListener('scroll', handler, { passive: true });
			scrollListenerCleanupRef.current = () => {
				htmlEl.removeEventListener('scroll', handler);
			};
		},
		[selectedTabIdRef, shouldPreserveRestoreSeed]
	);

	useLayoutEffect(() => {
		const tabId = selectedTabId;
		const handle = virtuosoRef.current;

		return () => {
			saveVirtuosoStateForTab(tabId, handle);
		};
	}, [saveVirtuosoStateForTab, selectedTabId]);

	useEffect(() => {
		return () => {
			scrollListenerCleanupRef.current?.();
		};
	}, []);

	const scrollTabToBottomSoon = useCallback(
		(tabId: string) => {
			requestAnimationFrame(() => {
				requestAnimationFrame(() => {
					if (selectedTabIdRef.current !== tabId) return;

					const tab = tabsRef.current.find(t => t.tabId === tabId);
					const lastIndex = (tab?.conversation.messages.length ?? 0) - 1;
					if (lastIndex < 0) return;

					virtuosoRef.current?.scrollToIndex({ index: lastIndex, align: 'end', behavior: 'auto' });
				});
			});
		},
		[selectedTabIdRef, tabsRef]
	);

	// eslint-disable-next-line react-hooks/refs
	const activeVirtuosoStateRecord = getSavedVirtuosoStateRecord(virtuosoStateByTabRef, selectedTabId);

	const activeVirtuosoState =
		activeVirtuosoStateRecord &&
		activeVirtuosoStateRecord.modifiedAtMs === activeConversationModifiedAtMs &&
		activeVirtuosoStateRecord.messageCount === messageCount &&
		activeVirtuosoStateRecord.lastMessageId === activeLastMessageId
			? activeVirtuosoStateRecord.snapshot
			: undefined;

	useEffect(() => {
		if (activeVirtuosoStateRecord && !activeVirtuosoState) {
			virtuosoStateByTabRef.current.delete(selectedTabId);
		}
	}, [activeVirtuosoState, activeVirtuosoStateRecord, selectedTabId]);

	// eslint-disable-next-line react-hooks/refs
	const savedTopItemIndex = getMapNumberValue(topItemIndexByTabRef, selectedTabId);

	const hasInitialTopItemIndex =
		!activeVirtuosoState &&
		typeof savedTopItemIndex === 'number' &&
		Number.isFinite(savedTopItemIndex) &&
		savedTopItemIndex > 0 &&
		savedTopItemIndex < messageCount;

	// eslint-disable-next-line react-hooks/refs
	const savedInitialScrollTop = getMapNumberValue(scrollTopByTabRef, selectedTabId) ?? 0;

	const initialPositionProps = activeVirtuosoState
		? {}
		: hasInitialTopItemIndex
			? {
					initialTopMostItemIndex: {
						index: savedTopItemIndex,
						align: 'start' as const,
					},
				}
			: Number.isFinite(savedInitialScrollTop) && savedInitialScrollTop > 0
				? {
						initialScrollTop: savedInitialScrollTop,
					}
				: {};

	const scrollActiveToTop = useCallback(() => {
		virtuosoRef.current?.scrollToIndex({ index: 0, align: 'start', behavior: 'auto' });
	}, []);

	const scrollActiveToBottom = useCallback(() => {
		const lastIndex = messageCount - 1;
		if (lastIndex < 0) return;

		virtuosoRef.current?.scrollToIndex({ index: lastIndex, align: 'end', behavior: 'auto' });
	}, [messageCount]);

	const resetScrollToTop = useCallback(
		(tabId: string) => {
			scrollTopByTabRef.current.set(tabId, 0);
			topItemIndexByTabRef.current.set(tabId, 0);
			virtuosoStateByTabRef.current.delete(tabId);
			restoredInitialScrollByTabRef.current.add(tabId);

			if (selectedTabIdRef.current !== tabId) return;
			virtuosoRef.current?.scrollTo({ top: 0 });
		},
		[selectedTabIdRef]
	);

	const setScrollTopForTab = useCallback((tabId: string, top: number) => {
		scrollTopByTabRef.current.set(tabId, top);
	}, []);

	const getScrollTopByTabSnapshot = useCallback(() => {
		const obj: Record<string, number> = {};
		for (const [key, value] of scrollTopByTabRef.current.entries()) {
			obj[key] = value;
		}
		return obj;
	}, []);

	const getTopItemIndexByTabSnapshot = useCallback(() => {
		const obj: Record<string, number> = {};
		for (const [key, value] of topItemIndexByTabRef.current.entries()) {
			obj[key] = value;
		}
		return obj;
	}, []);

	const disposeScrollRuntime = useCallback((tabId: string) => {
		virtuosoStateByTabRef.current.delete(tabId);
		restoredInitialScrollByTabRef.current.delete(tabId);
		scrollTopByTabRef.current.delete(tabId);
		topItemIndexByTabRef.current.delete(tabId);
	}, []);

	const previousSelectedTabIdRef = useRef<string | null>(null);

	useEffect(() => {
		const previousSelectedTabId = previousSelectedTabIdRef.current;
		const switchedTabs = previousSelectedTabId !== selectedTabId;

		if (switchedTabs && !activeVirtuosoState && messageCount > 0) {
			restoredInitialScrollByTabRef.current.add(selectedTabId);
		}

		previousSelectedTabIdRef.current = selectedTabId;
	}, [selectedTabId, activeVirtuosoState, messageCount]);

	const previousSelectedRenderRef = useRef({
		tabId: selectedTabId,
		isHydrating: activeTabIsHydrating,
		messageCount,
	});

	useEffect(() => {
		const previous = previousSelectedRenderRef.current;
		const sameTab = previous.tabId === selectedTabId;
		const justBecameRenderable =
			sameTab && ((previous.isHydrating && !activeTabIsHydrating) || (previous.messageCount === 0 && messageCount > 0));

		if (
			selectedTabId &&
			justBecameRenderable &&
			!activeTabIsHydrating &&
			messageCount > 0 &&
			!activeVirtuosoState &&
			!restoredInitialScrollByTabRef.current.has(selectedTabId)
		) {
			const restoreTabId = selectedTabId;
			const restoreTopItemIndex = topItemIndexByTabRef.current.get(restoreTabId);
			const canRestoreByTopItemIndex =
				typeof restoreTopItemIndex === 'number' && restoreTopItemIndex > 0 && restoreTopItemIndex < messageCount;
			const restoreScrollTop = scrollTopByTabRef.current.get(restoreTabId) ?? 0;

			requestAnimationFrame(() => {
				if (selectedTabIdRef.current !== restoreTabId) return;

				if (canRestoreByTopItemIndex) {
					const currentTab = tabsRef.current.find(t => t.tabId === restoreTabId);
					const currentMessageCount = currentTab?.conversation.messages.length ?? 0;
					if (currentMessageCount > 0) {
						virtuosoRef.current?.scrollToIndex({
							index: Math.min(restoreTopItemIndex, currentMessageCount - 1),
							align: 'start',
							behavior: 'auto',
						});
					}
				} else if (restoreScrollTop > 0) {
					virtuosoRef.current?.scrollTo({ top: restoreScrollTop, behavior: 'auto' });
				}

				restoredInitialScrollByTabRef.current.add(restoreTabId);
			});
		}

		previousSelectedRenderRef.current = {
			tabId: selectedTabId,
			isHydrating: activeTabIsHydrating,
			messageCount,
		};
	}, [activeTabIsHydrating, activeVirtuosoState, messageCount, selectedTabId, selectedTabIdRef, tabsRef]);

	const followActiveOutput = useCallback(
		(atBottom: boolean) => {
			if (activeTabIsHydrating) return false;
			return atBottom;
		},
		[activeTabIsHydrating]
	);

	const handleRangeChanged = useCallback(
		(range: ListRange) => {
			if (!selectedTabId) return;
			if (shouldPreserveRestoreSeed(selectedTabId)) return;
			topItemIndexByTabRef.current.set(selectedTabId, range.startIndex);
		},
		[selectedTabId, shouldPreserveRestoreSeed]
	);

	const handleAtBottomStateChange = useCallback((atBottom: boolean) => {
		setIsAtBottom(atBottom);
	}, []);

	const handleAtTopStateChange = useCallback((atTop: boolean) => {
		setIsAtTop(atTop);
	}, []);

	return {
		virtuosoRef,
		activeVirtuosoState,
		initialPositionProps,
		followActiveOutput,
		handleScrollerRef,
		handleRangeChanged,
		isAtBottom,
		isAtTop,
		handleAtBottomStateChange,
		handleAtTopStateChange,
		scrollActiveToTop,
		scrollActiveToBottom,
		scrollTabToBottomSoon,
		setScrollTopForTab,
		resetScrollToTop,
		getScrollTopByTabSnapshot,
		getTopItemIndexByTabSnapshot,
		disposeScrollRuntime,
	};
}
