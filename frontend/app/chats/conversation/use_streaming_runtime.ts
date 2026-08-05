import type { RefObject } from 'react';
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef } from 'react';

import type { ChatTabState } from '@/chats/tabs/tabs_model';

interface StreamChannelBuffer {
	chunks: string[];
	display: string;
}

interface StreamPublishSchedule {
	timerID: number | null;
	frameID: number | null;
	lastPublishedAt: number;
}

export interface StreamBuffer {
	text: StreamChannelBuffer;
	thinking: StreamChannelBuffer;
}

interface UseStreamingRuntimeArgs {
	tabs: ChatTabState[];
	selectedTabId: string;
	selectedTabIdRef: RefObject<string>;
}

const BACKGROUND_STREAM_COMPACT_CHUNK_COUNT = 256;
const STREAM_RENDER_FPS = 30;
const STREAM_RENDER_INTERVAL_MS = 1000 / STREAM_RENDER_FPS;

function flushStreamChannel(channel: StreamChannelBuffer): void {
	if (channel.chunks.length === 0) {
		return;
	}

	channel.display += channel.chunks.join('');
	channel.chunks = [];
}

function readStreamChannel(channel: StreamChannelBuffer): string {
	if (channel.chunks.length === 0) {
		return channel.display;
	}

	return `${channel.display}${channel.chunks.join('')}`;
}

function getStreamClock(): number {
	return typeof performance !== 'undefined' ? performance.now() : Date.now();
}

export function useStreamingRuntime({ tabs, selectedTabId, selectedTabIdRef }: UseStreamingRuntimeArgs) {
	const tabIdSet = useMemo(() => new Set(tabs.map(tab => tab.tabId)), [tabs]);
	const tabIdSetRef = useRef(tabIdSet);

	const tabExists = useCallback((tabId: string) => tabIdSetRef.current.has(tabId), []);

	const abortRefs = useRef(new Map<string, { current: AbortController | null }>());
	const requestIdByTabRef = useRef(new Map<string, string | null>());

	const streamBuffersRef = useRef(new Map<string, StreamBuffer>());
	const streamVersionRef = useRef(new Map<string, number>());
	const streamListenersRef = useRef(new Map<string, Set<() => void>>());
	const streamPublishSchedulesRef = useRef(new Map<string, StreamPublishSchedule>());

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
				text: { chunks: [], display: '' },
				thinking: { chunks: [], display: '' },
			};
			streamBuffersRef.current.set(tabId, buffer);
		}
		return buffer;
	}, []);

	const clearStreamBuffer = useCallback(
		(tabId: string) => {
			const buffer = getStreamBuffer(tabId);

			buffer.text.chunks = [];
			buffer.text.display = '';

			buffer.thinking.chunks = [];
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

		return readStreamChannel(buffer.text);
	}, []);

	const getFullStreamThinkingForTab = useCallback((tabId: string) => {
		const buffer = streamBuffersRef.current.get(tabId);
		if (!buffer) {
			return '';
		}

		return readStreamChannel(buffer.thinking);
	}, []);

	// Visible stream getters intentionally expose only `display`, never pending
	// chunks. This guarantees that rendering cannot bypass the frame scheduler
	// because of an unrelated React render.
	const getVisibleStreamTextForTab = useCallback((tabId: string) => {
		return streamBuffersRef.current.get(tabId)?.text.display ?? '';
	}, []);

	const getVisibleStreamThinkingForTab = useCallback((tabId: string) => {
		return streamBuffersRef.current.get(tabId)?.thinking.display ?? '';
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

	const emitStreamUpdate = useCallback(
		(tabId: string) => {
			bumpStreamVersion(tabId);

			const listeners = streamListenersRef.current.get(tabId);
			if (!listeners) {
				return;
			}

			for (const cb of listeners) {
				cb();
			}
		},
		[bumpStreamVersion]
	);

	const getPublishSchedule = useCallback((tabId: string): StreamPublishSchedule => {
		let schedule = streamPublishSchedulesRef.current.get(tabId);
		if (!schedule) {
			schedule = {
				timerID: null,
				frameID: null,
				lastPublishedAt: Number.NEGATIVE_INFINITY,
			};
			streamPublishSchedulesRef.current.set(tabId, schedule);
		}
		return schedule;
	}, []);

	const cancelScheduledStreamPublish = useCallback((tabId: string) => {
		const schedule = streamPublishSchedulesRef.current.get(tabId);
		if (!schedule) {
			return;
		}

		if (schedule.timerID !== null) {
			window.clearTimeout(schedule.timerID);
			schedule.timerID = null;
		}
		if (schedule.frameID !== null) {
			window.cancelAnimationFrame(schedule.frameID);
			schedule.frameID = null;
		}
	}, []);

	const publishStream = useCallback(
		(tabId: string, force = false) => {
			// Background tabs retain complete stream state, but they do not cause
			// React work until shown again.
			if (!force && selectedTabIdRef.current !== tabId) {
				flushStreamForTab(tabId);
				return;
			}

			flushStreamForTab(tabId);
			getPublishSchedule(tabId).lastPublishedAt = getStreamClock();
			emitStreamUpdate(tabId);
		},
		[emitStreamUpdate, flushStreamForTab, getPublishSchedule, selectedTabIdRef]
	);

	const notifyStreamSoon = useCallback(
		(tabId: string) => {
			const buffer = getStreamBuffer(tabId);

			if (selectedTabIdRef.current !== tabId) {
				cancelScheduledStreamPublish(tabId);

				const pendingChunkCount = buffer.text.chunks.length + buffer.thinking.chunks.length;
				if (pendingChunkCount >= BACKGROUND_STREAM_COMPACT_CHUNK_COUNT) {
					flushStreamForTab(tabId);
				}
				return;
			}

			const schedule = getPublishSchedule(tabId);
			if (schedule.timerID !== null || schedule.frameID !== null) {
				return;
			}

			const requestFrame = () => {
				if (schedule.frameID !== null) {
					return;
				}

				schedule.frameID = window.requestAnimationFrame(frameTime => {
					schedule.frameID = null;

					const remaining = STREAM_RENDER_INTERVAL_MS - (frameTime - schedule.lastPublishedAt);
					if (remaining > 0) {
						schedule.timerID = window.setTimeout(() => {
							schedule.timerID = null;
							requestFrame();
						}, Math.ceil(remaining));
						return;
					}

					publishStream(tabId);
				});
			};

			const elapsed = getStreamClock() - schedule.lastPublishedAt;
			if (elapsed >= STREAM_RENDER_INTERVAL_MS) {
				requestFrame();
				return;
			}

			schedule.timerID = window.setTimeout(
				() => {
					schedule.timerID = null;
					requestFrame();
				},
				Math.ceil(STREAM_RENDER_INTERVAL_MS - elapsed)
			);
		},
		[
			cancelScheduledStreamPublish,
			flushStreamForTab,
			getPublishSchedule,
			getStreamBuffer,
			publishStream,
			selectedTabIdRef,
		]
	);

	// Explicit lifecycle publication. This is deliberately not used for normal
	// chunks; normal chunks always use notifyStreamSoon().
	const notifyStreamNow = useCallback(
		(tabId: string) => {
			cancelScheduledStreamPublish(tabId);
			publishStream(tabId, true);
		},
		[cancelScheduledStreamPublish, publishStream]
	);

	// A background stream may have buffered chunks without a publish. Make it
	// visible synchronously when the user activates that tab.
	useLayoutEffect(() => {
		if (!selectedTabId) {
			return;
		}

		const buffer = streamBuffersRef.current.get(selectedTabId);
		if (!buffer || (buffer.text.chunks.length === 0 && buffer.thinking.chunks.length === 0)) {
			return;
		}

		notifyStreamNow(selectedTabId);
	}, [notifyStreamNow, selectedTabId]);

	const clearStreamForTab = useCallback(
		(tabId: string) => {
			clearStreamBuffer(tabId);
			notifyStreamNow(tabId);
		},
		[clearStreamBuffer, notifyStreamNow]
	);

	const disposeStreamRuntime = useCallback(
		(tabId: string) => {
			cancelScheduledStreamPublish(tabId);

			const abortRef = getAbortRef(tabId);
			abortRef.current?.abort();
			abortRef.current = null;

			abortRefs.current.delete(tabId);
			requestIdByTabRef.current.delete(tabId);
			streamBuffersRef.current.delete(tabId);
			streamVersionRef.current.delete(tabId);
			streamListenersRef.current.delete(tabId);
			streamPublishSchedulesRef.current.delete(tabId);
		},
		[cancelScheduledStreamPublish, getAbortRef]
	);

	useEffect(() => {
		const abortRefsCurrent = abortRefs.current;
		const streamPublishSchedulesRefCurrent = streamPublishSchedulesRef.current;
		return () => {
			try {
				for (const refObj of abortRefsCurrent.values()) {
					refObj.current?.abort();
				}
			} catch {
				// ignore
			}

			for (const schedule of streamPublishSchedulesRefCurrent.values()) {
				if (schedule.timerID !== null) {
					window.clearTimeout(schedule.timerID);
				}
				if (schedule.frameID !== null) {
					window.cancelAnimationFrame(schedule.frameID);
				}
			}
			streamPublishSchedulesRefCurrent.clear();
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
		getVisibleStreamTextForTab,
		getVisibleStreamThinkingForTab,
		getStreamVersionSnapshot,
		subscribeToStream,
		notifyStreamNow,
		notifyStreamSoon,
		requestIdByTabRef,
		disposeStreamRuntime,
	};
}
