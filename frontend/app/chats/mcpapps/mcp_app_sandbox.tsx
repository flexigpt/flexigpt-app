import { forwardRef, useEffect, useRef } from 'react';

interface MCPAppSandboxProps {
	html: string;
	csp: string;
	title: string;
	onIframeReady: (iframe: HTMLIFrameElement) => void;
	height?: number;
	allow?: string;
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
	_ref
) {
	const iframeRef = useRef<HTMLIFrameElement | null>(null);

	useEffect(() => {
		const el = iframeRef.current;
		if (!el) return;
		onIframeReady(el);
	}, [onIframeReady]);

	const wrapped = injectCSPMeta(html, csp);

	return (
		<iframe
			ref={iframeRef}
			title={title}
			srcDoc={wrapped}
			sandbox="allow-scripts"
			referrerPolicy="no-referrer"
			allow={allow}
			loading="lazy"
			className="bg-base-100 border-base-content/10 w-full rounded-2xl border"
			style={{ height, border: '0' }}
		/>
	);
});

function injectCSPMeta(html: string, csp: string): string {
	const meta = `<meta http-equiv="Content-Security-Policy" content="${escapeAttr(csp)}">`;
	if (/<head[^>]*>/i.test(html)) {
		return html.replace(/<head[^>]*>/i, match => `${match}\n${meta}`);
	}
	if (/<html[^>]*>/i.test(html)) {
		return html.replace(/<html[^>]*>/i, match => `${match}\n<head>${meta}</head>`);
	}
	return `<!DOCTYPE html><html><head>${meta}</head><body>${html}</body></html>`;
}

function escapeAttr(value: string): string {
	return value.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
