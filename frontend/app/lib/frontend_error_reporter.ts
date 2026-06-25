export interface FrontendErrorPayload {
	name: string;
	message: string;
	stack: string;
	phase: string;
	componentStack: string;
	href: string;
	userAgent: string;
	time: string;
	extraJson: string;
}

interface FrontendErrorLogger {
	error: (...args: unknown[]) => void;
}

interface ErrorExtra {
	phase?: string;
	componentStack?: string;
	[key: string]: unknown;
}

const LAST_ERROR_KEY = 'flexigpt.frontend.lastError';
const ERROR_LOG_KEY = 'flexigpt.frontend.errorLog';
const CRASH_COUNT_KEY = 'flexigpt.frontend.crashCount';
const CRASH_WINDOW_MS = 60_000;

let logger: FrontendErrorLogger | null = null;
let handlersInstalled = false;
let reporting = false;
let lastReportKey = '';
let lastReportAt = 0;

export function setFrontendErrorLogger(nextLogger: FrontendErrorLogger | null) {
	logger = nextLogger;
}

function truncate(value: string, maxLength: number): string {
	if (value.length <= maxLength) {
		return value;
	}
	return `${value.slice(0, maxLength)}...[truncated]`;
}

function safeString(value: unknown): string {
	try {
		return String(value);
	} catch {
		return '[Unstringifiable value]';
	}
}

function safeJson(value: unknown, maxLength = 20_000): string {
	const seen = new WeakSet<object>();

	try {
		const json = JSON.stringify(value, (_key, current) => {
			if (typeof current === 'bigint') {
				return current.toString();
			}

			if (typeof current === 'function') {
				// oxlint-disable-next-line typescript/no-unsafe-member-access
				return `[Function ${current.name || 'anonymous'}]`;
			}

			if (typeof current === 'object' && current !== null) {
				if (seen.has(current)) {
					return '[Circular]';
				}
				seen.add(current);
			}

			// oxlint-disable-next-line typescript/no-unsafe-return
			return current;
		});

		return truncate(json ?? safeString(value), maxLength);
	} catch {
		return truncate(safeString(value), maxLength);
	}
}

function normalizeFrontendError(error: unknown): { name: string; message: string; stack: string } {
	if (error instanceof Error) {
		return {
			name: error.name || 'Error',
			message: error.message || '[empty error message]',
			stack: error.stack || '',
		};
	}

	if (typeof DOMException !== 'undefined' && error instanceof DOMException) {
		return {
			name: error.name || 'DOMException',
			message: error.message || '[empty DOMException message]',
			stack: '',
		};
	}

	if (typeof error === 'object' && error !== null) {
		const maybe = error as Record<string, unknown>;

		return {
			name: typeof maybe.name === 'string' ? maybe.name : 'NonErrorObject',
			message: typeof maybe.message === 'string' ? maybe.message : safeJson(error),
			stack: typeof maybe.stack === 'string' ? maybe.stack : '',
		};
	}

	return {
		name: typeof error,
		message: safeString(error),
		stack: '',
	};
}

function makePayload(error: unknown, extra: ErrorExtra): FrontendErrorPayload {
	const normalized = normalizeFrontendError(error);

	return {
		name: truncate(normalized.name, 512),
		message: truncate(normalized.message, 8_192),
		stack: truncate(normalized.stack, 64_000),
		phase: truncate(extra.phase || 'unknown', 512),
		componentStack: truncate(typeof extra.componentStack === 'string' ? extra.componentStack : '', 64_000),
		href: typeof window !== 'undefined' ? window.location.href : '',
		userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : '',
		time: new Date().toISOString(),
		extraJson: safeJson(extra),
	};
}

function shouldSkipDuplicate(payload: FrontendErrorPayload): boolean {
	const now = Date.now();
	const key = `${payload.phase}:${payload.name}:${payload.message}:${payload.stack.slice(0, 500)}`;

	if (key === lastReportKey && now - lastReportAt < 2_000) {
		return true;
	}

	lastReportKey = key;
	lastReportAt = now;
	return false;
}

function persistPayload(payload: FrontendErrorPayload) {
	if (typeof window === 'undefined') {
		return;
	}

	try {
		window.localStorage.setItem(LAST_ERROR_KEY, safeJson(payload, 50_000));

		const raw = window.localStorage.getItem(ERROR_LOG_KEY);
		const existing = raw ? JSON.parse(raw) : [];
		const list = Array.isArray(existing) ? existing.slice(-9) : [];

		list.push(payload);

		window.localStorage.setItem(ERROR_LOG_KEY, safeJson(list, 100_000));
	} catch {
		// Never throw while handling an error.
	}
}

function formatPayloadForLog(payload: FrontendErrorPayload): string {
	const parts = [
		`[frontend-error] phase=${payload.phase} name=${payload.name} time=${payload.time}`,
		`href=${payload.href}`,
		`message=${payload.message}`,
	];

	if (payload.stack) {
		parts.push(`stack:\n${payload.stack}`);
	}
	if (payload.componentStack) {
		parts.push(`componentStack:\n${payload.componentStack}`);
	}
	if (payload.extraJson && payload.extraJson !== '{}') {
		parts.push(`extra=${payload.extraJson}`);
	}

	return parts.join('\n');
}

export function reportFrontendError(error: unknown, extra: ErrorExtra = {}) {
	if (reporting) {
		return;
	}

	reporting = true;

	try {
		const payload = makePayload(error, extra);

		if (shouldSkipDuplicate(payload)) {
			return;
		}

		persistPayload(payload);

		// Keep browser console useful in dev and in WebView devtools.
		console.error('Frontend error:', payload);

		if (logger) {
			try {
				logger.error(formatPayloadForLog(payload));
			} catch (loggerError) {
				console.error('Failed to write frontend error to backend logger:', loggerError);
			}
		}
	} finally {
		reporting = false;
	}
}

export function installGlobalFrontendErrorHandlers() {
	if (typeof window === 'undefined' || handlersInstalled) {
		return;
	}

	handlersInstalled = true;

	window.addEventListener(
		'error',
		event => {
			if (event instanceof ErrorEvent) {
				reportFrontendError(event.error ?? event.message, {
					phase: 'window.error',
					filename: event.filename,
					lineno: event.lineno,
					colno: event.colno,
				});
				return;
			}

			reportFrontendError('Unknown window error', {
				phase: 'window.error',
			});
		},
		true
	);

	window.addEventListener('unhandledrejection', event => {
		// Important: log only. Do not show fatal UI for unhandled promises.
		reportFrontendError(event.reason, {
			phase: 'window.unhandledrejection',
		});
	});
}

export function recordFrontendCrash(source: string): number {
	if (typeof window === 'undefined') {
		return 1;
	}

	try {
		const now = Date.now();
		const raw = window.sessionStorage.getItem(CRASH_COUNT_KEY);
		const parsed = raw ? (JSON.parse(raw) as { count?: unknown; firstAt?: unknown }) : {};

		let count = 1;
		let firstAt = now;

		if (typeof parsed.firstAt === 'number' && now - parsed.firstAt < CRASH_WINDOW_MS) {
			firstAt = parsed.firstAt;
			count = (typeof parsed.count === 'number' ? parsed.count : 0) + 1;
		}

		window.sessionStorage.setItem(CRASH_COUNT_KEY, JSON.stringify({ count, firstAt, source }));

		return count;
	} catch {
		return 1;
	}
}

export function reloadFrontend() {
	if (typeof window === 'undefined') {
		return;
	}
	window.location.reload();
}

export function resetLocalFrontendStateAndReload() {
	if (typeof window === 'undefined') {
		return;
	}

	// oxlint-disable-next-line no-alert
	const confirmed = window.confirm(
		'Reset local UI state and reload? This clears localStorage/sessionStorage only. Backend data is not deleted.'
	);

	if (!confirmed) {
		return;
	}

	try {
		window.localStorage.clear();
		window.sessionStorage.clear();
	} catch {
		// Ignore.
	}

	window.location.reload();
}

export function renderFatalStartupScreen(error: unknown) {
	if (typeof document === 'undefined') {
		return;
	}

	reportFrontendError(error, {
		phase: 'startup.fatal_screen',
	});

	const normalized = normalizeFrontendError(error);
	const host = document.body ?? document.documentElement;

	const container = document.createElement('main');
	container.style.minHeight = '100vh';
	container.style.display = 'flex';
	container.style.alignItems = 'center';
	container.style.justifyContent = 'center';
	container.style.padding = '24px';
	container.style.fontFamily = 'system-ui, sans-serif';

	const panel = document.createElement('section');
	panel.style.width = '100%';
	panel.style.maxWidth = '720px';

	const title = document.createElement('h1');
	title.textContent = 'FlexiGPT interface failed to start';

	const message = document.createElement('p');
	message.textContent = normalized.message || 'An unexpected frontend startup error occurred.';

	const reload = document.createElement('button');
	reload.type = 'button';
	reload.textContent = 'Reload UI';
	reload.onclick = () => {
		reloadFrontend();
	};

	const reset = document.createElement('button');
	reset.type = 'button';
	reset.textContent = 'Reset local UI state';
	reset.style.marginLeft = '8px';
	reset.onclick = () => {
		resetLocalFrontendStateAndReload();
	};

	panel.append(title, message, reload, reset);
	container.append(panel);

	host.replaceChildren(container);
}
