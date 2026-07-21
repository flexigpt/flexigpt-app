import type { RefObject } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';

const SCROLL_AT_TOP_THRESHOLD = 8;
const SCROLL_AT_BOTTOM_THRESHOLD = 128;
const SCROLL_AUTO_FOLLOW_THRESHOLD = 24;
const MESSAGE_SCROLL_POSITION_TOLERANCE = 1;

function getDistanceFromBottom(el: HTMLElement): number {
	return Math.max(0, el.scrollHeight - el.clientHeight - el.scrollTop);
}

function isElementAtBottom(el: HTMLElement): boolean {
	return getDistanceFromBottom(el) <= SCROLL_AT_BOTTOM_THRESHOLD;
}

function shouldElementAutoFollow(el: HTMLElement): boolean {
	return getDistanceFromBottom(el) <= SCROLL_AUTO_FOLLOW_THRESHOLD;
}

function isElementAtTop(el: HTMLElement): boolean {
	return el.scrollTop <= SCROLL_AT_TOP_THRESHOLD;
}

function getMessageScrollTop(el: HTMLElement, message: HTMLElement): number {
	const scrollerContentTop = el.getBoundingClientRect().top + el.clientTop;
	const messageTop = el.scrollTop + message.getBoundingClientRect().top - scrollerContentTop;

	return Math.max(0, messageTop);
}

function findFirstMessageIndexAfter(messageTops: number[], threshold: number): number {
	let low = 0;
	let high = messageTops.length;

	while (low < high) {
		const middle = low + Math.floor((high - low) / 2);
		if (messageTops[middle] <= threshold) {
			low = middle + 1;
		} else {
			high = middle;
		}
	}

	return low < messageTops.length ? low : -1;
}

function getCurrentMessageIndex(el: HTMLElement, messageTops: number[]): number {
	if (messageTops.length === 0) {
		return -1;
	}

	const currentTop = el.scrollTop;
	const nextIndex = findFirstMessageIndexAfter(messageTops, currentTop + MESSAGE_SCROLL_POSITION_TOLERANCE);

	if (nextIndex < 0) {
		return messageTops.length - 1;
	}

	return Math.max(0, nextIndex - 1);
}

function getPreviousMessageIndex(el: HTMLElement, messageTops: number[]): number {
	if (messageTops.length === 0) {
		return -1;
	}
	if (isElementAtTop(el)) {
		return -1;
	}

	const currentIndex = getCurrentMessageIndex(el, messageTops);
	return currentIndex > 0 ? currentIndex - 1 : -1;
}

function getNextMessageIndex(el: HTMLElement, messageTops: number[]): number {
	if (messageTops.length === 0 || isElementAtBottom(el)) {
		return -1;
	}

	// At absolute scroll top, the first message can be offset by the
	// scroller's padding. Treat that first message as the current anchor
	// so Down moves to the second message rather than re-targeting the first.
	const currentTop = isElementAtTop(el) ? (messageTops[0] ?? el.scrollTop) : el.scrollTop;

	return findFirstMessageIndexAfter(messageTops, currentTop + MESSAGE_SCROLL_POSITION_TOLERANCE);
}

function scrollMessageAtIndex(el: HTMLElement, messageTops: number[], index: number): boolean {
	const targetTop = messageTops[index];
	if (targetTop === undefined) {
		return false;
	}

	const maxScrollTop = Math.max(0, el.scrollHeight - el.clientHeight);

	el.scrollTo({
		top: Math.min(maxScrollTop, targetTop),
		behavior: 'auto',
	});
	return true;
}

function getMessageJumpAvailability(el: HTMLElement | null, messageTops: number[]) {
	if (!el || messageTops.length <= 1) {
		return { canJumpToPreviousMessage: false, canJumpToNextMessage: false };
	}

	return {
		canJumpToPreviousMessage: getPreviousMessageIndex(el, messageTops) >= 0,
		canJumpToNextMessage: getNextMessageIndex(el, messageTops) >= 0,
	};
}

interface ScrollIndicatorState {
	isAtBottom: boolean;
	isAtTop: boolean;
	canJumpToPreviousMessage: boolean;
	canJumpToNextMessage: boolean;
}

const INITIAL_SCROLL_INDICATOR_STATE: ScrollIndicatorState = {
	isAtBottom: true,
	isAtTop: true,
	canJumpToPreviousMessage: false,
	canJumpToNextMessage: false,
};

interface UseScrollRestoreArgs {
	selectedTabId: string;
	selectedTabIdRef: RefObject<string>;
	activeTabIsHydrating: boolean;
	messageCount: number;
	initialScrollTopByTab?: Record<string, number>;
}

export function useScrollRestore({
	selectedTabId,
	selectedTabIdRef,
	activeTabIsHydrating,
	messageCount,
	initialScrollTopByTab,
}: UseScrollRestoreArgs) {
	const [scrollIndicatorState, setScrollIndicatorState] =
		useState<ScrollIndicatorState>(INITIAL_SCROLL_INDICATOR_STATE);
	const { isAtBottom, isAtTop, canJumpToPreviousMessage, canJumpToNextMessage } = scrollIndicatorState;
	const scrollIndicatorStateRef = useRef<ScrollIndicatorState>(INITIAL_SCROLL_INDICATOR_STATE);

	const scrollContainerRef = useRef<HTMLDivElement | null>(null);
	const scrollContentRef = useRef<HTMLDivElement | null>(null);
	const messageScrollTopsRef = useRef<number[]>([]);
	const messagePositionsDirtyRef = useRef(true);

	const scrollTopByTabRef = useRef(new Map<string, number>());
	const shouldAutoFollowRef = useRef(true);

	const seededScrollFromStorageRef = useRef(false);
	const activeTabIsHydratingRef = useRef(activeTabIsHydrating);

	const refreshMessageElementCache = useCallback(() => {
		const contentEl = scrollContentRef.current;
		const scrollerEl = scrollContainerRef.current;
		if (!contentEl || !scrollerEl) {
			messageScrollTopsRef.current = [];
			messagePositionsDirtyRef.current = false;
			return;
		}

		const nextTops: number[] = [];
		for (const child of contentEl.children) {
			if (child instanceof HTMLElement && child.dataset.chatMessageIndex !== undefined) {
				nextTops.push(getMessageScrollTop(scrollerEl, child));
			}
		}

		messageScrollTopsRef.current = nextTops;
		messagePositionsDirtyRef.current = false;
	}, []);

	useLayoutEffect(() => {
		activeTabIsHydratingRef.current = activeTabIsHydrating;
	}, [activeTabIsHydrating]);

	const scrollStateRafRef = useRef<number | null>(null);
	const scrollStateShouldIncludeJumpsRef = useRef(false);
	const resizeObserverRafRef = useRef<number | null>(null);
	const messageJumpTimerRef = useRef<number | null>(null);
	const observedContentSizeRef = useRef<{ width: number; height: number } | null>(null);

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

	const setScrollIndicatorStateIfChanged = useCallback((next: ScrollIndicatorState) => {
		const prev = scrollIndicatorStateRef.current;
		if (
			prev.isAtBottom === next.isAtBottom &&
			prev.isAtTop === next.isAtTop &&
			prev.canJumpToPreviousMessage === next.canJumpToPreviousMessage &&
			prev.canJumpToNextMessage === next.canJumpToNextMessage
		) {
			return;
		}

		scrollIndicatorStateRef.current = next;
		setScrollIndicatorState(next);
	}, []);

	const updateScrollStateNow = useCallback(
		(el: HTMLElement | null, includeMessageJumps = true) => {
			if (!el) {
				shouldAutoFollowRef.current = true;
				setScrollIndicatorStateIfChanged(INITIAL_SCROLL_INDICATOR_STATE);
				return;
			}

			const nextAtTop = isElementAtTop(el);
			const nextAtBottom = isElementAtBottom(el);
			shouldAutoFollowRef.current = shouldElementAutoFollow(el);

			if (includeMessageJumps && messagePositionsDirtyRef.current) {
				refreshMessageElementCache();
			}

			const jumpState = includeMessageJumps
				? getMessageJumpAvailability(el, messageScrollTopsRef.current)
				: {
						canJumpToPreviousMessage: scrollIndicatorStateRef.current.canJumpToPreviousMessage,
						canJumpToNextMessage: scrollIndicatorStateRef.current.canJumpToNextMessage,
					};

			setScrollIndicatorStateIfChanged({
				isAtBottom: nextAtBottom,
				isAtTop: nextAtTop,
				canJumpToPreviousMessage: jumpState.canJumpToPreviousMessage,
				canJumpToNextMessage: jumpState.canJumpToNextMessage,
			});
		},

		[refreshMessageElementCache, setScrollIndicatorStateIfChanged]
	);

	const scheduleScrollStateUpdate = useCallback(
		(includeMessageJumps = false) => {
			if (includeMessageJumps) {
				scrollStateShouldIncludeJumpsRef.current = true;
			}

			if (scrollStateRafRef.current !== null) {
				return;
			}

			scrollStateRafRef.current = window.requestAnimationFrame(() => {
				scrollStateRafRef.current = null;
				const shouldIncludeJumps = scrollStateShouldIncludeJumpsRef.current;
				scrollStateShouldIncludeJumpsRef.current = false;
				updateScrollStateNow(scrollContainerRef.current, shouldIncludeJumps);
			});
		},
		[updateScrollStateNow]
	);

	const scheduleMessageJumpStateUpdate = useCallback(() => {
		if (messageJumpTimerRef.current !== null) {
			window.clearTimeout(messageJumpTimerRef.current);
		}

		messageJumpTimerRef.current = window.setTimeout(() => {
			messageJumpTimerRef.current = null;
			scheduleScrollStateUpdate(true);
		}, 120);
	}, [scheduleScrollStateUpdate]);

	useEffect(() => {
		return () => {
			if (scrollStateRafRef.current !== null) {
				window.cancelAnimationFrame(scrollStateRafRef.current);
				scrollStateRafRef.current = null;
			}
			if (resizeObserverRafRef.current !== null) {
				window.cancelAnimationFrame(resizeObserverRafRef.current);
				resizeObserverRafRef.current = null;
			}
			if (messageJumpTimerRef.current !== null) {
				window.clearTimeout(messageJumpTimerRef.current);
				messageJumpTimerRef.current = null;
			}
		};
	}, []);

	const setScrollContainerRef = useCallback(
		(el: HTMLDivElement | null) => {
			scrollContainerRef.current = el;
			messagePositionsDirtyRef.current = true;
			scheduleScrollStateUpdate(true);
		},
		[scheduleScrollStateUpdate]
	);

	const setScrollContentRef = useCallback((el: HTMLDivElement | null) => {
		scrollContentRef.current = el;
		messagePositionsDirtyRef.current = true;
	}, []);

	const handleScroll = useCallback(() => {
		const el = scrollContainerRef.current;
		const tabId = selectedTabIdRef.current;
		if (!el || !tabId) {
			return;
		}

		shouldAutoFollowRef.current = shouldElementAutoFollow(el);
		persistScrollPosition(tabId, el);
		scheduleScrollStateUpdate(false);
		scheduleMessageJumpStateUpdate();
	}, [persistScrollPosition, scheduleMessageJumpStateUpdate, scheduleScrollStateUpdate, selectedTabIdRef]);

	const scrollElementToBottom = useCallback((el: HTMLElement | null) => {
		if (!el) {
			return;
		}
		const nextTop = Math.max(0, el.scrollHeight - el.clientHeight);
		shouldAutoFollowRef.current = true;

		if (Math.abs(el.scrollTop - nextTop) > 1) {
			el.scrollTop = nextTop;
		}
	}, []);

	const previousRenderRef = useRef({
		selectedTabId: '',
		activeTabIsHydrating: true,
		messageCount: 0,
	});

	useLayoutEffect(() => {
		refreshMessageElementCache();

		const el = scrollContainerRef.current;
		const previous = previousRenderRef.current;

		const selectedChanged = previous.selectedTabId !== selectedTabId;
		const justFinishedHydrating =
			previous.selectedTabId === selectedTabId && previous.activeTabIsHydrating && !activeTabIsHydrating;
		const messageCountIncreased = previous.selectedTabId === selectedTabId && messageCount > previous.messageCount;

		if (el && selectedTabId && !activeTabIsHydrating && (selectedChanged || justFinishedHydrating)) {
			const nextTop = scrollTopByTabRef.current.get(selectedTabId) ?? 0;
			shouldAutoFollowRef.current = false;
			el.scrollTop = nextTop;
			persistScrollPosition(selectedTabId, el);
			scheduleScrollStateUpdate(true);
		} else if (el && selectedTabId && !activeTabIsHydrating && messageCountIncreased && shouldAutoFollowRef.current) {
			scrollElementToBottom(el);
			persistScrollPosition(selectedTabId, el);
			scheduleScrollStateUpdate(true);
		} else if (el) {
			scheduleScrollStateUpdate(true);
		}

		previousRenderRef.current = {
			selectedTabId,
			activeTabIsHydrating,
			messageCount,
		};
	}, [
		activeTabIsHydrating,
		messageCount,
		persistScrollPosition,
		scrollElementToBottom,
		selectedTabId,
		refreshMessageElementCache,
		scheduleScrollStateUpdate,
	]);

	useEffect(() => {
		const contentEl = scrollContentRef.current;
		if (!contentEl) {
			return;
		}
		if (typeof ResizeObserver === 'undefined') {
			return;
		}

		observedContentSizeRef.current = null;

		const observer = new ResizeObserver(entries => {
			const entry = entries[0];
			const width = entry?.contentRect.width ?? 0;
			const height = entry?.contentRect.height ?? 0;
			const previous = observedContentSizeRef.current;

			if (previous && previous.width === width && previous.height === height) {
				return;
			}

			observedContentSizeRef.current = { width, height };
			messagePositionsDirtyRef.current = true;

			if (resizeObserverRafRef.current !== null) {
				return;
			}

			resizeObserverRafRef.current = window.requestAnimationFrame(() => {
				resizeObserverRafRef.current = null;

				const scrollerEl = scrollContainerRef.current;
				const tabId = selectedTabIdRef.current;
				if (!scrollerEl || !tabId) {
					return;
				}

				if (!activeTabIsHydratingRef.current && shouldAutoFollowRef.current) {
					scrollElementToBottom(scrollerEl);
					persistScrollPosition(tabId, scrollerEl);
				}

				updateScrollStateNow(scrollerEl, false);

				scheduleMessageJumpStateUpdate();
			});
		});
		observer.observe(contentEl);
		return () => {
			observer.disconnect();
			if (resizeObserverRafRef.current !== null) {
				window.cancelAnimationFrame(resizeObserverRafRef.current);
				resizeObserverRafRef.current = null;
			}
			if (messageJumpTimerRef.current !== null) {
				window.clearTimeout(messageJumpTimerRef.current);
				messageJumpTimerRef.current = null;
			}
		};
	}, [
		persistScrollPosition,
		scheduleMessageJumpStateUpdate,
		scrollElementToBottom,
		selectedTabIdRef,
		updateScrollStateNow,
	]);

	const scrollTabToBottomSoon = useCallback(
		(tabId: string) => {
			requestAnimationFrame(() => {
				requestAnimationFrame(() => {
					if (selectedTabIdRef.current !== tabId) {
						return;
					}

					const el = scrollContainerRef.current;
					if (!el) {
						return;
					}

					scrollElementToBottom(el);
					persistScrollPosition(tabId, el);
					updateScrollStateNow(el);
				});
			});
		},
		[persistScrollPosition, scrollElementToBottom, selectedTabIdRef, updateScrollStateNow]
	);

	const scrollActiveToTop = useCallback(() => {
		const el = scrollContainerRef.current;
		const tabId = selectedTabIdRef.current;
		if (!el || !tabId) {
			return;
		}

		el.scrollTo({ top: 0, behavior: 'auto' });
		persistScrollPosition(tabId, el);
		updateScrollStateNow(el);
	}, [persistScrollPosition, selectedTabIdRef, updateScrollStateNow]);

	const scrollActiveToBottom = useCallback(() => {
		const lastIndex = messageCount - 1;
		if (lastIndex < 0) {
			return;
		}

		const el = scrollContainerRef.current;
		const tabId = selectedTabIdRef.current;
		if (!el || !tabId) {
			return;
		}

		scrollElementToBottom(el);
		persistScrollPosition(tabId, el);
		updateScrollStateNow(el);
	}, [messageCount, persistScrollPosition, scrollElementToBottom, selectedTabIdRef, updateScrollStateNow]);

	const scrollActiveToPreviousMessage = useCallback(() => {
		const el = scrollContainerRef.current;
		const tabId = selectedTabIdRef.current;
		if (!el || !tabId) {
			return;
		}

		if (messagePositionsDirtyRef.current) {
			refreshMessageElementCache();
		}

		const messageTops = messageScrollTopsRef.current;
		const previousIndex = getPreviousMessageIndex(el, messageTops);
		if (previousIndex < 0) {
			return;
		}

		if (scrollMessageAtIndex(el, messageTops, previousIndex)) {
			persistScrollPosition(tabId, el);
			updateScrollStateNow(el);
		}
	}, [persistScrollPosition, refreshMessageElementCache, selectedTabIdRef, updateScrollStateNow]);

	const scrollActiveToNextMessage = useCallback(() => {
		const el = scrollContainerRef.current;
		const tabId = selectedTabIdRef.current;
		if (!el || !tabId) {
			return;
		}

		if (messagePositionsDirtyRef.current) {
			refreshMessageElementCache();
		}

		const messageTops = messageScrollTopsRef.current;
		const nextIndex = getNextMessageIndex(el, messageTops);
		if (nextIndex < 0) {
			return;
		}

		if (scrollMessageAtIndex(el, messageTops, nextIndex)) {
			persistScrollPosition(tabId, el);
			updateScrollStateNow(el);
		}
	}, [persistScrollPosition, refreshMessageElementCache, selectedTabIdRef, updateScrollStateNow]);

	const scrollActivePageBy = useCallback(
		(direction: 1 | -1) => {
			const el = scrollContainerRef.current;
			const tabId = selectedTabIdRef.current;
			if (!el || !tabId) {
				return;
			}

			const delta = Math.max(120, el.clientHeight * 0.85) * direction;
			el.scrollTo({
				top: Math.max(0, Math.min(el.scrollHeight - el.clientHeight, el.scrollTop + delta)),
				behavior: 'auto',
			});
			persistScrollPosition(tabId, el);
			updateScrollStateNow(el);
		},
		[persistScrollPosition, selectedTabIdRef, updateScrollStateNow]
	);

	const resetScrollToTop = useCallback(
		(tabId: string) => {
			scrollTopByTabRef.current.set(tabId, 0);

			if (selectedTabIdRef.current !== tabId) {
				return;
			}
			const el = scrollContainerRef.current;
			if (!el) {
				return;
			}

			el.scrollTo({ top: 0, behavior: 'auto' });
			persistScrollPosition(tabId, el);
			updateScrollStateNow(el);
		},
		[persistScrollPosition, selectedTabIdRef, updateScrollStateNow]
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
		canJumpToPreviousMessage,
		canJumpToNextMessage,
		scrollActiveToTop,
		scrollActiveToBottom,
		scrollActiveToPreviousMessage,
		scrollActiveToNextMessage,
		scrollActivePageBy,
		scrollTabToBottomSoon,
		setScrollTopForTab,
		resetScrollToTop,
		getScrollTopByTabSnapshot,
		disposeScrollRuntime,
	};
}
