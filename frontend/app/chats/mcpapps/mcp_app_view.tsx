import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { FiAlertTriangle } from 'react-icons/fi';

import { type MCPContent, MCPContentType } from '@/spec/mcp';

import { backendAPI, mcpAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import {
	buildMCPAppAllowAttribute,
	buildMCPAppCSP,
	getMCPAppUIResourceMeta,
	type MCPAppUIResourceMeta,
} from '@/chats/composer/mcp/mcp_apps_csp';
import { buildMCPAppHostContext } from '@/chats/mcpapps/mcp_app_host_context';
import { MCPAppPostMessageBridge } from '@/chats/mcpapps/mcp_app_postmessage_bridge';
import { MCPAppRPCRouter } from '@/chats/mcpapps/mcp_app_rpc_router';
import { MCPAppSandbox } from '@/chats/mcpapps/mcp_app_sandbox';
import { type JSONRPCRequest, type JSONRPCResponse, type MCPAppInstance } from '@/chats/mcpapps/mcp_app_types';

const APP_MIME = 'text/html;profile=mcp-app';
type LoadedMCPAppResource = {
	html: string;
	mimeType: string;
	meta?: MCPAppUIResourceMeta;
};
interface MCPAppViewProps {
	instance: MCPAppInstance;
	toolInput?: unknown;
	toolResult?: { content?: MCPContent[]; structuredContent?: unknown; isError?: boolean };
	height?: number;
}

function isAppMime(mime?: string): boolean {
	if (!mime) return false;
	const norm = mime.toLowerCase().replace(/\s/g, '');
	return norm === APP_MIME || norm.startsWith(`${APP_MIME};`);
}

function decodeMCPBlob(blob: string | number[] | undefined): string {
	if (!blob) return '';
	if (typeof blob === 'string') {
		try {
			return atob(blob);
		} catch {
			return '';
		}
	}
	try {
		return new TextDecoder().decode(new Uint8Array(blob));
	} catch {
		return '';
	}
}

function extractAppHTML(contents?: MCPContent[]): LoadedMCPAppResource | null {
	for (const c of contents ?? []) {
		if (c.type !== MCPContentType.MCPContentTypeResource) continue;
		const res = c.resource;
		if (!res) continue;
		if (!isAppMime(res.mimeType)) continue;
		const html = typeof res.text === 'string' && res.text.length > 0 ? res.text : decodeMCPBlob(res.blob);
		if (!html.trim()) continue;
		return {
			html,
			mimeType: res.mimeType ?? APP_MIME,
			meta: getMCPAppUIResourceMeta(res),
		};
	}
	return null;
}

function parseToolInput(raw: unknown): unknown {
	if (typeof raw !== 'string') return raw;
	const trimmed = raw.trim();
	if (!trimmed) return undefined;
	try {
		return JSON.parse(trimmed);
	} catch {
		return raw;
	}
}

function isSafeExternalURL(raw: string): boolean {
	try {
		const url = new URL(raw);
		return url.protocol === 'http:' || url.protocol === 'https:';
	} catch {
		return false;
	}
}

export function MCPAppView({ instance, toolInput, toolResult, height = 480 }: MCPAppViewProps) {
	const [loadedResource, setLoadedResource] = useState<LoadedMCPAppResource | null>(null);
	const [loadError, setLoadError] = useState<string | null>(null);
	const [pendingURL, setPendingURL] = useState<string | null>(null);
	const [blockedURL, setBlockedURL] = useState<string | null>(null);
	const [viewInitialized, setViewInitialized] = useState(false);

	const approvalResolverRef = useRef<((ok: boolean) => void) | null>(null);

	const bridgeRef = useRef<MCPAppPostMessageBridge | null>(null);
	const iframeRef = useRef<HTMLIFrameElement | null>(null);

	useEffect(() => {
		let cancelled = false;
		// eslint-disable-next-line react-hooks/set-state-in-effect, react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setLoadError(null);
		// eslint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setLoadedResource(null);
		// eslint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setViewInitialized(false);

		void mcpAPI
			.readMCPResource(instance.bundleID, instance.serverID, instance.resourceUri)
			.then(resp => {
				if (cancelled) return;
				const extracted = extractAppHTML(resp?.contents);
				if (!extracted) {
					setLoadError('UI resource did not return MCP App HTML.');
					return;
				}
				setLoadedResource(extracted);
			})
			.catch((err: unknown) => {
				if (cancelled) return;
				setLoadError(err instanceof Error ? err.message : 'Failed to load MCP App.');
			});

		return () => {
			cancelled = true;
		};
	}, [instance.bundleID, instance.resourceUri, instance.serverID]);

	const requestOpenLinkApproval = useCallback(async (url: string) => {
		if (!isSafeExternalURL(url)) {
			setBlockedURL(url);
			return false;
		}
		return await new Promise<boolean>(resolve => {
			approvalResolverRef.current = resolve;
			setPendingURL(url);
		});
	}, []);

	const router = useMemo(
		() =>
			// eslint-disable-next-line react-hooks/refs
			new MCPAppRPCRouter({
				instance,
				requestOpenLinkApproval,
				onAppLog: () => {},
			}),
		[instance, requestOpenLinkApproval]
	);

	const handleIframeReady = useCallback(
		(iframe: HTMLIFrameElement) => {
			iframeRef.current = iframe;
			bridgeRef.current?.dispose();

			const bridge = new MCPAppPostMessageBridge({
				iframe,
				onRequest: async (req): Promise<JSONRPCResponse> => {
					if (req.method === 'ui/initialize' || req.method === 'initialize') {
						return {
							jsonrpc: '2.0',
							id: req.id,
							result: buildMCPAppHostContext({
								width: iframe.clientWidth,
								height: iframe.clientHeight,
								allowToolCalls: true,
							}),
						};
					}
					return router.handle(req as JSONRPCRequest);
				},
				onNotification: note => {
					if (note.method === 'ui/notifications/initialized' || note.method === 'notifications/initialized') {
						setViewInitialized(true);
						return;
					}
					if (note.method === 'notifications/message') {
						console.info('MCP App message', instance.instanceID, note.params);
					}
				},
			});
			bridgeRef.current = bridge;
		},
		[instance.instanceID, router]
	);

	useEffect(() => {
		const bridge = bridgeRef.current;
		if (!bridge || !viewInitialized) return;

		if (toolInput !== undefined) {
			bridge.sendNotification('ui/notifications/tool-input', {
				toolName: instance.toolName,
				toolUseID: instance.toolUseID,
				arguments: parseToolInput(toolInput),
			});
		}
	}, [instance.toolName, instance.toolUseID, toolInput, viewInitialized]);

	useEffect(() => {
		const bridge = bridgeRef.current;
		if (!bridge || !viewInitialized) return;

		if (toolResult !== undefined) {
			bridge.sendNotification('ui/notifications/tool-result', {
				toolName: instance.toolName,
				toolUseID: instance.toolUseID,
				...toolResult,
			});
		}
	}, [instance.toolName, instance.toolUseID, toolResult, viewInitialized]);

	useEffect(() => {
		return () => {
			bridgeRef.current?.sendNotification('ui/resource-teardown', {
				resourceUri: instance.resourceUri,
			});
			bridgeRef.current?.dispose();
			bridgeRef.current = null;
		};
	}, [instance.resourceUri]);

	if (loadError) {
		return (
			<div className="alert alert-warning rounded-2xl text-sm">
				<div className="flex items-center gap-2">
					<FiAlertTriangle size={14} />
					<span>MCP App failed to load: {loadError}</span>
				</div>
			</div>
		);
	}

	if (!loadedResource) {
		return <div className="text-base-content/60 text-xs">Loading MCP App…</div>;
	}

	return (
		<>
			<div className="border-base-content/10 bg-base-200 rounded-2xl border p-2">
				<div className="mb-2 flex items-center justify-between gap-2 text-xs">
					<span className="truncate font-semibold">{instance.displayName ?? instance.toolName}</span>
					<span className="text-base-content/60 truncate">{instance.serverID}</span>
				</div>
				<MCPAppSandbox
					html={loadedResource.html}
					csp={buildMCPAppCSP(loadedResource.meta)}
					title={`MCP App ${instance.toolName}`}
					onIframeReady={handleIframeReady}
					height={height}
					allow={buildMCPAppAllowAttribute(loadedResource.meta)}
				/>
			</div>

			<DeleteConfirmationModal
				isOpen={pendingURL !== null}
				title="Open external link?"
				message={`The MCP App for ${instance.serverID} wants to open:\n${pendingURL ?? ''}`}
				confirmButtonText="Open"
				onConfirm={() => {
					const url = pendingURL;
					setPendingURL(null);
					if (!url) {
						approvalResolverRef.current?.(false);
						approvalResolverRef.current = null;
						return;
					}
					try {
						backendAPI.openURL(url);
						approvalResolverRef.current?.(true);
					} catch (_err: unknown) {
						setBlockedURL(url);
						approvalResolverRef.current?.(false);
					}
					approvalResolverRef.current = null;
				}}
				onClose={() => {
					setPendingURL(null);
					approvalResolverRef.current?.(false);
					approvalResolverRef.current = null;
				}}
			/>

			<ActionDeniedAlertModal
				isOpen={blockedURL !== null}
				onClose={() => {
					setBlockedURL(null);
				}}
				message={`Could not open ${blockedURL ?? ''}.`}
			/>
		</>
	);
}
