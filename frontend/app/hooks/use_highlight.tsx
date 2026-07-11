import { useEffect, useRef, useState } from 'react';

let worker: Worker | undefined;

let seq = 0;
const HIGHLIGHT_CACHE_MAX_ENTRIES = 128;
const HIGHLIGHT_CACHE_MAX_CHARS = 8 * 1024 * 1024;

const highlightCache = new Map<string, string>();
const inFlightByKey = new Map<string, Promise<string>>();
let highlightCacheChars = 0;

const waiting = new Map<
	number,
	{
		resolve: (html: string) => void;
		reject: (err: unknown) => void;
	}
>();

function getHighlightKey(code: string, lang: string): string {
	return `${lang.trim().toLowerCase()}\u0000${code}`;
}

function cacheHighlightResult(key: string, html: string): void {
	const previous = highlightCache.get(key);
	if (previous !== undefined) {
		highlightCacheChars -= key.length + previous.length;
		highlightCache.delete(key);
	}

	highlightCache.set(key, html);
	highlightCacheChars += key.length + html.length;

	while (highlightCache.size > HIGHLIGHT_CACHE_MAX_ENTRIES || highlightCacheChars > HIGHLIGHT_CACHE_MAX_CHARS) {
		const oldest = highlightCache.entries().next().value as [string, string] | undefined;
		if (!oldest) {
			break;
		}

		highlightCache.delete(oldest[0]);
		highlightCacheChars -= oldest[0].length + oldest[1].length;
	}
}

export function ensureWorker(): Worker {
	if (worker) {
		return worker;
	}

	worker = new Worker(new URL('./highlight_worker.ts', import.meta.url), { type: 'module' });

	worker.onmessage = (evt: MessageEvent) => {
		const { id, html, error } = evt.data as {
			id: number;
			html?: string;
			error?: unknown;
		};

		const rec = waiting.get(id);
		if (!rec) {
			return;
		}

		if (error) {
			rec.reject(error);
		} else {
			rec.resolve(html as string);
		}

		waiting.delete(id);
	};

	worker.onerror = error => {
		for (const record of waiting.values()) {
			record.reject(error);
		}
		waiting.clear();
		worker = undefined;
	};

	return worker;
}

function highlightAsync(code: string, lang: string): Promise<string> {
	return new Promise((resolve, reject) => {
		const id = seq++;
		waiting.set(id, { resolve, reject });
		// oxlint-disable-next-line unicorn/require-post-message-target-origin
		ensureWorker().postMessage({ id, code, lang });
	});
}

function highlightWithCache(code: string, lang: string): Promise<string> {
	const key = getHighlightKey(code, lang);
	const cached = highlightCache.get(key);
	if (cached !== undefined) {
		return Promise.resolve(cached);
	}

	const inFlight = inFlightByKey.get(key);
	if (inFlight) {
		return inFlight;
	}

	const request = highlightAsync(code, lang).then(html => {
		cacheHighlightResult(key, html);
		return html;
	});

	inFlightByKey.set(key, request);
	void request.then(
		() => {
			if (inFlightByKey.get(key) === request) {
				inFlightByKey.delete(key);
			}
		},
		() => {
			if (inFlightByKey.get(key) === request) {
				inFlightByKey.delete(key);
			}
		}
	);

	return request;
}

export function useHighlight(code: string, lang: string, enabled = true) {
	const key = getHighlightKey(code, lang);
	const [result, setResult] = useState<{ key: string; html: string | null }>({
		key: '',
		html: null,
	});
	const ticket = useRef(0);

	useEffect(() => {
		const myId = ++ticket.current;

		if (!enabled || !code.trim()) {
			return;
		}

		highlightWithCache(code, lang)
			.then(h => {
				if (ticket.current === myId) {
					setResult({ key, html: h });
				}
			})
			.catch((err: unknown) => {
				console.error(err);
				if (ticket.current === myId) {
					setResult({ key, html: '' });
				}
			});

		return () => {
			if (ticket.current === myId) {
				ticket.current += 1;
			}
		};
	}, [code, enabled, key, lang]);

	if (!code.trim()) {
		return '';
	}
	if (!enabled) {
		return null;
	}

	return result.key === key ? result.html : (highlightCache.get(key) ?? null);
}
