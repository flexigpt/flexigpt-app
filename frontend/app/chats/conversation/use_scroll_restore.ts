import { type RefObject, useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';

const SCROLL_AT_TOP_THRESHOLD = 8;
const SCROLL_AT_BOTTOM_THRESHOLD = 128;

function getDistanceFromBottom(el: HTMLElement): number {
	return Math.max(0, el.scrollHeight - el.clientHeight - el.scrollTop);
}

function isElementAtBottom(el: HTMLElement): boolean {
	return getDistanceFromBottom(el) <= SCROLL_AT_BOTTOM_THRESHOLD;
}

function isElementAtTop(el: HTMLElement): boolean {
	return el.scrollTop <= SCROLL_AT_TOP_THRESHOLD;
}

type UseScrollRestoreArgs = {
	selectedTabId: string;
	selectedTabIdRef: RefObject<string>;
	activeTabIsHydrating: boolean;
	messageCount: number;
	initialScrollTopByTab?: Record<string, number>;
};

export function useScrollRestore({
	selectedTabId,
	selectedTabIdRef,
	activeTabIsHydrating,
	messageCount,
	initialScrollTopByTab,
}: UseScrollRestoreArgs) {
	const [isAtBottom, setIsAtBottom] = useState(true);
	const [isAtTop, setIsAtTop] = useState(true);

	const scrollContainerRef = useRef<HTMLDivElement | null>(null);
	const scrollContentRef = useRef<HTMLDivElement | null>(null);
	const scrollTopByTabRef = useRef(new Map<string, number>());
	const shouldAutoFollowRef = useRef(true);

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

	const persistScrollPosition = useCallback((tabId: string, el: HTMLElement) => {
		scrollTopByTabRef.current.set(tabId, el.scrollTop);
	}, []);

	const updateScrollState = useCallback((el: HTMLElement | null) => {
		if (!el) {
			shouldAutoFollowRef.current = true;
			setIsAtTop(true);
			setIsAtBottom(true);
			return;
		}

		const nextAtTop = isElementAtTop(el);
		const nextAtBottom = isElementAtBottom(el);

		shouldAutoFollowRef.current = nextAtBottom;
		setIsAtTop(nextAtTop);
		setIsAtBottom(nextAtBottom);
	}, []);

	const setScrollContainerRef = useCallback(
		(el: HTMLDivElement | null) => {
			scrollContainerRef.current = el;
			updateScrollState(el);
		},
		[updateScrollState]
	);

	const setScrollContentRef = useCallback((el: HTMLDivElement | null) => {
		scrollContentRef.current = el;
	}, []);

	const handleScroll = useCallback(() => {
		const el = scrollContainerRef.current;
		const tabId = selectedTabIdRef.current;
		if (!el || !tabId) return;

		persistScrollPosition(tabId, el);
		updateScrollState(el);
	}, [persistScrollPosition, selectedTabIdRef, updateScrollState]);

	const scrollElementToBottom = useCallback((el: HTMLElement | null) => {
		if (!el) return;
		el.scrollTo({ top: el.scrollHeight, behavior: 'auto' });
	}, []);

	const previousSelectedTabIdRef = useRef<string | null>(null);
	const previousHydratingRef = useRef(activeTabIsHydrating);

	useLayoutEffect(() => {
		const el = scrollContainerRef.current;
		const selectedChanged = previousSelectedTabIdRef.current !== selectedTabId;
		const justFinishedHydrating =
			previousSelectedTabIdRef.current === selectedTabId && previousHydratingRef.current && !activeTabIsHydrating;

		if (el && selectedTabId && !activeTabIsHydrating && (selectedChanged || justFinishedHydrating)) {
			const nextTop = scrollTopByTabRef.current?.get(selectedTabId) ?? 0;
			el.scrollTo({ top: nextTop, behavior: 'auto' });
			persistScrollPosition(selectedTabId, el);
			updateScrollState(el);
		} else if (el) {
			updateScrollState(el);
		}

		previousSelectedTabIdRef.current = selectedTabId;
		previousHydratingRef.current = activeTabIsHydrating;
	}, [activeTabIsHydrating, persistScrollPosition, selectedTabId, updateScrollState]);

	const previousRenderRef = useRef({
		tabId: selectedTabId,
		messageCount,
	});

	useLayoutEffect(() => {
		const el = scrollContainerRef.current;
		const previous = previousRenderRef.current;
		const sameTab = previous.tabId === selectedTabId;
		const messageCountIncreased = sameTab && messageCount > previous.messageCount;

		if (el && selectedTabId && !activeTabIsHydrating && messageCountIncreased && shouldAutoFollowRef.current) {
			scrollElementToBottom(el);
			persistScrollPosition(selectedTabId, el);
			updateScrollState(el);
		} else if (el) {
			updateScrollState(el);
		}

		previousRenderRef.current = {
			tabId: selectedTabId,
			messageCount,
		};
	}, [
		activeTabIsHydrating,
		messageCount,
		persistScrollPosition,
		scrollElementToBottom,
		selectedTabId,
		updateScrollState,
	]);

	useEffect(() => {
		const contentEl = scrollContentRef.current;
		if (!contentEl) return;
		if (typeof ResizeObserver === 'undefined') return;

		const observer = new ResizeObserver(() => {
			const scrollerEl = scrollContainerRef.current;
			const tabId = selectedTabIdRef.current;
			if (!scrollerEl || !tabId) return;

			if (!activeTabIsHydrating && shouldAutoFollowRef.current) {
				scrollElementToBottom(scrollerEl);
				persistScrollPosition(tabId, scrollerEl);
			}

			updateScrollState(scrollerEl);
		});
		observer.observe(contentEl);
		return () => {
			observer.disconnect();
		};
	}, [
		activeTabIsHydrating,
		persistScrollPosition,
		scrollElementToBottom,
		selectedTabId,
		selectedTabIdRef,
		updateScrollState,
	]);

	const scrollTabToBottomSoon = useCallback(
		(tabId: string) => {
			requestAnimationFrame(() => {
				requestAnimationFrame(() => {
					if (selectedTabIdRef.current !== tabId) return;

					const el = scrollContainerRef.current;
					if (!el) return;

					scrollElementToBottom(el);
					persistScrollPosition(tabId, el);
					updateScrollState(el);
				});
			});
		},
		[persistScrollPosition, scrollElementToBottom, selectedTabIdRef, updateScrollState]
	);

	const scrollActiveToTop = useCallback(() => {
		const el = scrollContainerRef.current;
		const tabId = selectedTabIdRef.current;
		if (!el || !tabId) return;

		el.scrollTo({ top: 0, behavior: 'auto' });
		persistScrollPosition(tabId, el);
		updateScrollState(el);
	}, [persistScrollPosition, selectedTabIdRef, updateScrollState]);

	const scrollActiveToBottom = useCallback(() => {
		const lastIndex = messageCount - 1;
		if (lastIndex < 0) return;

		const el = scrollContainerRef.current;
		const tabId = selectedTabIdRef.current;
		if (!el || !tabId) return;

		scrollElementToBottom(el);
		persistScrollPosition(tabId, el);
		updateScrollState(el);
	}, [messageCount, persistScrollPosition, scrollElementToBottom, selectedTabIdRef, updateScrollState]);

	const resetScrollToTop = useCallback(
		(tabId: string) => {
			scrollTopByTabRef.current.set(tabId, 0);

			if (selectedTabIdRef.current !== tabId) return;
			const el = scrollContainerRef.current;
			if (!el) return;

			el.scrollTo({ top: 0, behavior: 'auto' });
			persistScrollPosition(tabId, el);
			updateScrollState(el);
		},
		[persistScrollPosition, selectedTabIdRef, updateScrollState]
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

	const disposeScrollRuntime = useCallback((tabId: string) => {
		scrollTopByTabRef.current.delete(tabId);
	}, []);

	return {
		setScrollContainerRef,
		setScrollContentRef,
		handleScroll,
		isAtBottom,
		isAtTop,
		scrollActiveToTop,
		scrollActiveToBottom,
		scrollTabToBottomSoon,
		setScrollTopForTab,
		resetScrollToTop,
		getScrollTopByTabSnapshot,
		disposeScrollRuntime,
	};
}
