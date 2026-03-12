import { useEffect, useRef, useState } from 'react';

// Utility that talks to the worker and returns a Promise<string>
let worker: Worker | undefined;

let seq = 0;
const waiting = new Map<
	number,
	{
		resolve: (html: string) => void;
		reject: (err: unknown) => void;
	}
>();

export function ensureWorker(): Worker {
	if (worker) return worker;

	worker = new Worker(new URL('./highlight_worker.ts', import.meta.url), { type: 'module' });

	worker.onmessage = (evt: MessageEvent) => {
		const { id, html, error } = evt.data as {
			id: number;
			html?: string;
			error?: unknown;
		};

		const rec = waiting.get(id);
		if (!rec) return;

		if (error) rec.reject(error);
		else rec.resolve(html as string);

		waiting.delete(id);
	};

	console.log('highlight worker initialized');
	return worker;
}

function highlightAsync(code: string, lang: string): Promise<string> {
	return new Promise((resolve, reject) => {
		const id = seq++;
		waiting.set(id, { resolve, reject });
		ensureWorker().postMessage({ id, code, lang });
	});
}

export function useHighlight(code: string, lang: string) {
	const [html, setHtml] = useState<string | null>(null);
	const ticket = useRef(0);

	useEffect(() => {
		const myId = ++ticket.current;

		if (!code.trim()) {
			return;
		}

		highlightAsync(code, lang)
			.then(h => {
				if (ticket.current === myId) {
					setHtml(h);
				}
			})
			.catch((err: unknown) => {
				console.error(err);
				if (ticket.current === myId) {
					setHtml('');
				}
			});
	}, [code, lang]);

	return code.trim() ? html : '';
}
