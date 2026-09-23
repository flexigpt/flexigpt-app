import type { ForwardedRef } from 'react';
import { forwardRef, useCallback, useMemo, useRef } from 'react';

interface MCPAppSandboxProps {
	html: string;
	csp: string;
	title: string;
	onIframeReady: (iframe: HTMLIFrameElement) => void;
	height?: number;
	allow?: string;
}

function assignForwardedRef<T>(ref: ForwardedRef<T>, value: T | null): void {
	if (typeof ref === 'function') {
		ref(value);
		return;
	}
	if (ref) {
		ref.current = value;
	}
}

/**
 * Sandboxed iframe host for MCP App HTML. We do not allow same-origin, so the
 * iframe gets a unique opaque origin and cannot reach the parent DOM or any
 * Wails runtime bindings.
 *
 * The CSP is injected as a <meta http-equiv> tag because srcdoc cannot carry
 * Content-Security-Policy response headers.
 */
export const MCPAppSandbox = forwardRef<HTMLIFrameElement, MCPAppSandboxProps>(function MCPAppSandbox(
	{ html, csp, title, onIframeReady, height = 480, allow },
	forwardedRef
) {
	const notifiedIframeRef = useRef<HTMLIFrameElement | null>(null);
	const setIframeRef = useCallback(
		(iframe: HTMLIFrameElement | null) => {
			assignForwardedRef(forwardedRef, iframe);
			if (!iframe || notifiedIframeRef.current === iframe) {
				return;
			}

			notifiedIframeRef.current = iframe;
			onIframeReady(iframe);
		},
		[forwardedRef, onIframeReady]
	);
	const wrapped = useMemo(() => injectCSPMeta(html, csp), [csp, html]);

	return (
		<iframe
			ref={setIframeRef}
			title={title}
			srcDoc={wrapped}
			sandbox="allow-scripts"
			referrerPolicy="no-referrer"
			allow={allow}
			loading="lazy"
			className="bg-base-100 w-full rounded-2xl"
			style={{ height, border: '0' }}
		/>
	);
});

function injectCSPMeta(html: string, csp: string): string {
	const parsed = new DOMParser().parseFromString(html, 'text/html');
	const meta = parsed.createElement('meta');
	meta.httpEquiv = 'Content-Security-Policy';
	meta.content = csp;
	parsed.head.prepend(meta);

	return `<!DOCTYPE html>\n${parsed.documentElement.outerHTML}`;
}
