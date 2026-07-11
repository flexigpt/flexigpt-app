import type { RefObject } from 'react';
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef } from 'react';

import type { ChatTabState } from '@/chats/tabs/tabs_model';

interface StreamChannelBuffer {
	chunks: string[];
	flushedIdx: number;
	display: string;
}

export interface StreamBuffer {
	text: StreamChannelBuffer;
	thinking: StreamChannelBuffer;
}

interface UseStreamingRuntimeArgs {
	tabs: ChatTabState[];
	selectedTabIdRef: RefObject<string>;
}

const SHORT_STREAM_RENDER_INTERVAL_MS = 32;
const LONG_STREAM_RENDER_INTERVAL_MS = 50;
const VERY_LONG_STREAM_RENDER_INTERVAL_MS = 80;

function getStreamRenderInterval(buffer: StreamBuffer): number {
	const displayedLength = buffer.text.display.length + buffer.thinking.display.length;
	if (displayedLength >= 64_000) {
		return VERY_LONG_STREAM_RENDER_INTERVAL_MS;
	}
	if (displayedLength >= 16_000) {
		return LONG_STREAM_RENDER_INTERVAL_MS;
	}
	return SHORT_STREAM_RENDER_INTERVAL_MS;
}

function flushStreamChannel(channel: StreamChannelBuffer): void {
	if (channel.flushedIdx >= channel.chunks.length) {
		return;
	}

	const pending =
		channel.flushedIdx === 0 ? channel.chunks.join('') : channel.chunks.slice(channel.flushedIdx).join('');

	channel.display += pending;
	channel.chunks = [];
	channel.flushedIdx = 0;
}

export function useStreamingRuntime({ tabs, selectedTabIdRef }: UseStreamingRuntimeArgs) {
	const tabIdSet = useMemo(() => new Set(tabs.map(tab => tab.tabId)), [tabs]);
	const tabIdSetRef = useRef(tabIdSet);

	const tabExists = useCallback((tabId: string) => tabIdSetRef.current.has(tabId), []);

	const abortRefs = useRef(new Map<string, { current: AbortController | null }>());
	const requestIdByTabRef = useRef(new Map<string, string | null>());
	const tokensReceivedByTabRef = useRef(new Map<string, boolean | null>());

	const streamBuffersRef = useRef(new Map<string, StreamBuffer>());
	const streamVersionRef = useRef(new Map<string, number>());
	const streamListenersRef = useRef(new Map<string, Set<() => void>>());
	const notifyTimersRef = useRef(new Map<string, number | null>());

	useLayoutEffect(() => {
		tabIdSetRef.current = tabIdSet;
	}, [tabIdSet]);

	const getAbortRef = useCallback((tabId: string) => {
		let refObj = abortRefs.current.get(tabId);
		if (!refObj) {
			refObj = { current: null };
			abortRefs.current.set(tabId, refObj);
		}
		return refObj;
	}, []);

	const getStreamBuffer = useCallback((tabId: string) => {
		let buffer = streamBuffersRef.current.get(tabId);
		if (!buffer) {
			buffer = {
				text: { chunks: [], flushedIdx: 0, display: '' },
				thinking: { chunks: [], flushedIdx: 0, display: '' },
			};
			streamBuffersRef.current.set(tabId, buffer);
		}
		return buffer;
	}, []);

	const clearStreamBuffer = useCallback(
		(tabId: string) => {
			const pendingTimer = notifyTimersRef.current.get(tabId) ?? null;
			if (pendingTimer !== null) {
				window.clearTimeout(pendingTimer);
				notifyTimersRef.current.set(tabId, null);
			}

			const buffer = getStreamBuffer(tabId);

			buffer.text.chunks = [];
			buffer.text.flushedIdx = 0;
			buffer.text.display = '';

			buffer.thinking.chunks = [];
			buffer.thinking.flushedIdx = 0;
			buffer.thinking.display = '';
		},
		[getStreamBuffer]
	);

	const flushStreamForTab = useCallback(
		(tabId: string) => {
			const buffer = getStreamBuffer(tabId);

			flushStreamChannel(buffer.text);
			flushStreamChannel(buffer.thinking);
		},
		[getStreamBuffer]
	);

	const getFullStreamTextForTab = useCallback((tabId: string) => {
		const buffer = streamBuffersRef.current.get(tabId);
		if (!buffer) {
			return '';
		}

		const channel = buffer.text;
		flushStreamChannel(channel);

		return channel.display;
	}, []);

	const getFullStreamThinkingForTab = useCallback((tabId: string) => {
		const buffer = streamBuffersRef.current.get(tabId);
		if (!buffer) {
			return '';
		}

		const channel = buffer.thinking;
		flushStreamChannel(channel);

		return channel.display;
	}, []);

	const bumpStreamVersion = useCallback((tabId: string) => {
		const nextVersion = (streamVersionRef.current.get(tabId) ?? 0) + 1;
		streamVersionRef.current.set(tabId, nextVersion);
	}, []);

	const getStreamVersionSnapshot = useCallback((tabId: string) => {
		return streamVersionRef.current.get(tabId) ?? 0;
	}, []);

	const subscribeToStream = useCallback((tabId: string, cb: () => void) => {
		let set = streamListenersRef.current.get(tabId);
		if (!set) {
			set = new Set();
			streamListenersRef.current.set(tabId, set);
		}
		set.add(cb);

		return () => {
			const listeners = streamListenersRef.current.get(tabId);
			listeners?.delete(cb);
			if (listeners && listeners.size === 0) {
				streamListenersRef.current.delete(tabId);
			}
		};
	}, []);

	const notifyStreamNow = useCallback(
		(tabId: string) => {
			flushStreamForTab(tabId);
			bumpStreamVersion(tabId);

			const listeners = streamListenersRef.current.get(tabId);
			if (!listeners) {
				return;
			}

			for (const cb of listeners) {
				cb();
			}
		},
		[bumpStreamVersion, flushStreamForTab]
	);

	const notifyStreamSoon = useCallback(
		(tabId: string) => {
			if (selectedTabIdRef.current !== tabId) {
				return;
			}

			const existingTimer = notifyTimersRef.current.get(tabId) ?? null;
			if (existingTimer !== null) {
				return;
			}

			const buffer = getStreamBuffer(tabId);
			const renderInterval = getStreamRenderInterval(buffer);
			const timer = window.setTimeout(() => {
				notifyTimersRef.current.set(tabId, null);

				if (!tabIdSetRef.current.has(tabId) || selectedTabIdRef.current !== tabId) {
					return;
				}

				notifyStreamNow(tabId);
			}, renderInterval);

			notifyTimersRef.current.set(tabId, timer);
		},
		[getStreamBuffer, notifyStreamNow, selectedTabIdRef]
	);

	const clearStreamForTab = useCallback(
		(tabId: string) => {
			clearStreamBuffer(tabId);
			notifyStreamNow(tabId);
		},
		[clearStreamBuffer, notifyStreamNow]
	);

	const disposeStreamRuntime = useCallback(
		(tabId: string) => {
			const abortRef = getAbortRef(tabId);
			abortRef.current?.abort();
			abortRef.current = null;

			abortRefs.current.delete(tabId);
			requestIdByTabRef.current.delete(tabId);
			tokensReceivedByTabRef.current.delete(tabId);
			streamBuffersRef.current.delete(tabId);
			streamVersionRef.current.delete(tabId);

			const timer = notifyTimersRef.current.get(tabId) ?? null;
			if (timer !== null) {
				window.clearTimeout(timer);
			}
			notifyTimersRef.current.delete(tabId);
			streamListenersRef.current.delete(tabId);
		},
		[getAbortRef]
	);

	useEffect(() => {
		const abortRefsCurrent = abortRefs.current;
		const notifyTimersCurrent = notifyTimersRef.current;

		return () => {
			try {
				for (const refObj of abortRefsCurrent.values()) {
					refObj.current?.abort();
				}
			} catch {
				// ignore
			}

			for (const timer of notifyTimersCurrent.values()) {
				if (timer !== null) {
					window.clearTimeout(timer);
				}
			}
		};
	}, []);

	return {
		tabExists,
		getAbortRef,
		getStreamBuffer,
		clearStreamBuffer,
		clearStreamForTab,
		getFullStreamTextForTab,
		getFullStreamThinkingForTab,
		getStreamVersionSnapshot,
		subscribeToStream,
		notifyStreamNow,
		notifyStreamSoon,
		requestIdByTabRef,
		tokensReceivedByTabRef,
		disposeStreamRuntime,
	};
}
