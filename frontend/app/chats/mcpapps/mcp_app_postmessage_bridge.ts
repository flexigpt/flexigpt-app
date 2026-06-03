import {
	isJSONRPCNotification,
	isJSONRPCRequest,
	type JSONRPCMessage,
	type JSONRPCNotification,
	type JSONRPCRequest,
	type JSONRPCResponse,
} from '@/chats/mcpapps/mcp_app_types';

export type AppRequestHandler = (req: JSONRPCRequest) => Promise<JSONRPCResponse>;
export type AppNotificationHandler = (note: JSONRPCNotification) => void;

/**
 * Low-level postMessage bridge between the FlexiGPT host and one MCP App
 * iframe. Validates source window and message shape, dispatches requests and
 * notifications, and lets the host send messages back.
 *
 * The iframe has a unique opaque origin (sandbox="allow-scripts" without
 * allow-same-origin). For null-origin iframes, browsers require targetOrigin
 * "*". This is safe because the iframe is isolated by the sandbox.
 */
export class MCPAppPostMessageBridge {
	private readonly iframe: HTMLIFrameElement;
	private readonly onRequest: AppRequestHandler;
	private readonly onNotification: AppNotificationHandler;
	private readonly listener: (e: MessageEvent) => void;
	private disposed = false;

	constructor(args: {
		iframe: HTMLIFrameElement;
		onRequest: AppRequestHandler;
		onNotification: AppNotificationHandler;
	}) {
		this.iframe = args.iframe;
		this.onRequest = args.onRequest;
		this.onNotification = args.onNotification;
		this.listener = e => {
			void this.handle(e);
		};
		window.addEventListener('message', this.listener);
	}

	dispose() {
		if (this.disposed) return;
		this.disposed = true;
		window.removeEventListener('message', this.listener);
	}

	sendNotification(method: string, params?: unknown) {
		const note: JSONRPCNotification = { jsonrpc: '2.0', method, params };
		this.post(note);
	}

	private async handle(e: MessageEvent) {
		if (this.disposed) return;
		// Source check: must come from our iframe's contentWindow.
		if (!this.iframe.contentWindow || e.source !== this.iframe.contentWindow) return;
		const data = e.data as unknown;
		if (!data || typeof data !== 'object') return;

		if (isJSONRPCRequest(data)) {
			try {
				const resp = await this.onRequest(data);
				this.post(resp);
			} catch (err) {
				this.post({
					jsonrpc: '2.0',
					id: data.id,
					error: {
						code: -32603,
						message: err instanceof Error ? err.message : 'Internal host error',
					},
				});
			}
			return;
		}
		if (isJSONRPCNotification(data)) {
			this.onNotification(data);
		}
	}

	private post(msg: JSONRPCMessage) {
		try {
			this.iframe.contentWindow?.postMessage(msg, '*');
		} catch {
			// iframe may have detached
		}
	}
}
