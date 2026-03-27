import { type RefObject, useCallback, useEffect, useMemo, useRef } from 'react';

import type { ChatTabState } from '@/chats/tabs/tabs_model';

export type StreamChannelBuffer = {
	chunks: string[];
	flushedIdx: number;
	display: string;
};

export type StreamBuffer = {
	text: StreamChannelBuffer;
	thinking: StreamChannelBuffer;
};

type UseStreamingRuntimeArgs = {
	tabs: ChatTabState[];
	selectedTabIdRef: RefObject<string>;
};

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

	useEffect(() => {
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

			const flushChannel = (channel: StreamChannelBuffer) => {
				if (channel.flushedIdx < channel.chunks.length) {
					channel.display += channel.chunks.slice(channel.flushedIdx).join('');
					channel.flushedIdx = channel.chunks.length;
				}
			};

			flushChannel(buffer.text);
			flushChannel(buffer.thinking);
		},
		[getStreamBuffer]
	);

	const getFullStreamTextForTab = useCallback((tabId: string) => {
		const buffer = streamBuffersRef.current.get(tabId);
		if (!buffer) return '';

		const channel = buffer.text;
		if (channel.flushedIdx < channel.chunks.length) {
			channel.display += channel.chunks.slice(channel.flushedIdx).join('');
			channel.flushedIdx = channel.chunks.length;
		}

		return channel.display;
	}, []);

	const getFullStreamThinkingForTab = useCallback((tabId: string) => {
		const buffer = streamBuffersRef.current.get(tabId);
		if (!buffer) return '';

		const channel = buffer.thinking;
		if (channel.flushedIdx < channel.chunks.length) {
			channel.display += channel.chunks.slice(channel.flushedIdx).join('');
			channel.flushedIdx = channel.chunks.length;
		}

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
			if (!listeners) return;

			for (const cb of listeners) {
				cb();
			}
		},
		[bumpStreamVersion, flushStreamForTab]
	);

	const notifyStreamSoon = useCallback(
		(tabId: string) => {
			if (selectedTabIdRef.current !== tabId) return;

			const existingTimer = notifyTimersRef.current.get(tabId) ?? null;
			if (existingTimer !== null) return;

			const timer = window.setTimeout(() => {
				notifyTimersRef.current.set(tabId, null);
				notifyStreamNow(tabId);
			}, 140);

			notifyTimersRef.current.set(tabId, timer);
		},
		[notifyStreamNow, selectedTabIdRef]
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
