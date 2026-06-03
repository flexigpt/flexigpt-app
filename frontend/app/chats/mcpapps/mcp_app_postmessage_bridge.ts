import {
	isJSONRPCNotification,
	isJSONRPCRequest,
	isJSONRPCResponse,
	type JSONRPCMessage,
	type JSONRPCNotification,
	type JSONRPCRequest,
	type JSONRPCResponse,
} from '@/chats/mcpapps/mcp_app_types';

export type AppRequestHandler = (req: JSONRPCRequest) => Promise<JSONRPCResponse>;
export type AppNotificationHandler = (note: JSONRPCNotification) => void;

type PendingHostRequest = {
	resolve: (response: JSONRPCResponse) => void;
	reject: (error: Error) => void;
	timer: number;
};

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
	private nextHostRequestID = 1;
	private readonly pendingHostRequests = new Map<number | string, PendingHostRequest>();

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
		for (const pending of this.pendingHostRequests.values()) {
			window.clearTimeout(pending.timer);
			pending.reject(new Error('MCP App bridge disposed.'));
		}
		this.pendingHostRequests.clear();
	}

	sendRequest(method: string, params?: unknown, timeoutMS = 1000): Promise<JSONRPCResponse> {
		if (this.disposed) {
			return Promise.reject(new Error('MCP App bridge disposed.'));
		}

		const id = `host-${this.nextHostRequestID++}`;
		const req: JSONRPCRequest = {
			jsonrpc: '2.0',
			id,
			method,
			params,
		};

		return new Promise<JSONRPCResponse>((resolve, reject) => {
			const timer = window.setTimeout(() => {
				this.pendingHostRequests.delete(id);
				reject(new Error(`MCP App request ${method} timed out.`));
			}, timeoutMS);

			this.pendingHostRequests.set(id, { resolve, reject, timer });
			this.post(req);
		});
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
		if (isJSONRPCResponse(data)) {
			const pending = this.pendingHostRequests.get(data.id);
			if (!pending) return;

			this.pendingHostRequests.delete(data.id);
			window.clearTimeout(pending.timer);
			if (data.error) {
				pending.reject(new Error(data.error.message));
			} else {
				pending.resolve(data);
			}
			return;
		}

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
