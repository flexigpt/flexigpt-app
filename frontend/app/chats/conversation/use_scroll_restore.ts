import type { RefObject } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';

const SCROLL_AT_TOP_THRESHOLD = 8;
const SCROLL_AT_BOTTOM_THRESHOLD = 128;
const SCROLL_AUTO_FOLLOW_THRESHOLD = 24;
const MESSAGE_SCROLL_MARGIN = 12;

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

function getCurrentMessageIndex(el: HTMLElement, messages: HTMLElement[]): number {
	if (messages.length === 0) {
		return -1;
	}

	const scanTop = el.scrollTop + MESSAGE_SCROLL_MARGIN;
	let lo = 0;
	let hi = messages.length - 1;
	let result = messages.length - 1;

	while (lo <= hi) {
		const mid = Math.floor((lo + hi) / 2);
		const message = messages[mid];
		const messageBottom = message.offsetTop + message.offsetHeight;

		if (messageBottom > scanTop) {
			result = mid;
			hi = mid - 1;
		} else {
			lo = mid + 1;
		}
	}

	return Number(messages[result]?.dataset.chatMessageIndex ?? -1);
}

function scrollMessageAtIndex(el: HTMLElement, messages: HTMLElement[], index: number): boolean {
	const target = messages[index];
	if (!target) {
		return false;
	}

	el.scrollTo({
		top: Math.max(0, target.offsetTop - MESSAGE_SCROLL_MARGIN),
		behavior: 'auto',
	});
	return true;
}

function getMessageJumpAvailability(el: HTMLElement | null, messages: HTMLElement[]) {
	if (!el || messages.length <= 1) {
		return { canJumpToPreviousMessage: false, canJumpToNextMessage: false };
	}

	const currentIndex = getCurrentMessageIndex(el, messages);

	return {
		canJumpToPreviousMessage: currentIndex > 0,
		canJumpToNextMessage: currentIndex >= 0 && currentIndex < messages.length - 1,
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
	const messageElementsRef = useRef<HTMLElement[]>([]);

	const scrollTopByTabRef = useRef(new Map<string, number>());
	const shouldAutoFollowRef = useRef(true);

	const seededScrollFromStorageRef = useRef(false);
	const activeTabIsHydratingRef = useRef(activeTabIsHydrating);

	const refreshMessageElementCache = useCallback(() => {
		const contentEl = scrollContentRef.current;
		if (!contentEl) {
			messageElementsRef.current = [];
			return;
		}

		const next: HTMLElement[] = [];
		for (const child of contentEl.children) {
			if (child instanceof HTMLElement && child.dataset.chatMessageIndex !== undefined) {
				next.push(child);
			}
		}

		messageElementsRef.current = next;
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

			const jumpState = includeMessageJumps
				? getMessageJumpAvailability(el, messageElementsRef.current)
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

		[setScrollIndicatorStateIfChanged]
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
			scheduleScrollStateUpdate(true);
		},
		[scheduleScrollStateUpdate]
	);

	const setScrollContentRef = useCallback((el: HTMLDivElement | null) => {
		scrollContentRef.current = el;
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

		const messages = messageElementsRef.current;
		const currentIndex = getCurrentMessageIndex(el, messages);
		if (currentIndex <= 0) {
			return;
		}

		if (scrollMessageAtIndex(el, messages, currentIndex - 1)) {
			persistScrollPosition(tabId, el);
			updateScrollStateNow(el);
		}
	}, [persistScrollPosition, selectedTabIdRef, updateScrollStateNow]);

	const scrollActiveToNextMessage = useCallback(() => {
		const el = scrollContainerRef.current;
		const tabId = selectedTabIdRef.current;
		if (!el || !tabId) {
			return;
		}

		const messages = messageElementsRef.current;
		const currentIndex = getCurrentMessageIndex(el, messages);
		if (currentIndex < 0 || currentIndex >= messageCount - 1) {
			return;
		}

		if (scrollMessageAtIndex(el, messages, currentIndex + 1)) {
			persistScrollPosition(tabId, el);
			updateScrollStateNow(el);
		}
	}, [messageCount, persistScrollPosition, selectedTabIdRef, updateScrollStateNow]);

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
