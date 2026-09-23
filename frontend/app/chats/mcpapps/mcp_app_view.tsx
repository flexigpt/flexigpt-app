import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FiAlertTriangle } from 'react-icons/fi';

import type { ArtifactRef } from '@/spec/artifact';
import type { MCPAppsPolicy, MCPContent } from '@/spec/mcp';
import { MCP_APP_HTML_MIME_TYPE, MCPContentType } from '@/spec/mcp';

import { isJSONObject } from '@/lib/jsonschema_utils';

import { backendAPI, mcpManagementAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import type { MCPAppUIResourceMeta } from '@/chats/composer/mcp/mcp_apps_csp';
import type { MCPAppModelContextUpdatePayload, MCPAppUIMessage } from '@/chats/mcpapps/mcp_app_events';
import type { JSONRPCResponse, MCPAppInstance } from '@/chats/mcpapps/mcp_app_types';
import { MCPApprovalModal } from '@/chats/composer/mcp/mcp_approval_modal';
import { buildMCPAppAllowAttribute, buildMCPAppCSP, getMCPAppUIResourceMeta } from '@/chats/composer/mcp/mcp_apps_csp';
import { useMCPApproval } from '@/chats/composer/mcp/use_mcp_approval';
import { dispatchMCPAppModelContextUpdate, dispatchMCPAppUIMessage } from '@/chats/mcpapps/mcp_app_events';
import { buildMCPAppHostContext } from '@/chats/mcpapps/mcp_app_host_context';
import { MCPAppPostMessageBridge } from '@/chats/mcpapps/mcp_app_postmessage_bridge';
import { MCPAppRPCRouter } from '@/chats/mcpapps/mcp_app_rpc_router';
import { MCPAppSandbox } from '@/chats/mcpapps/mcp_app_sandbox';

const APP_MIME = MCP_APP_HTML_MIME_TYPE;
const NORMALIZED_APP_MIME = APP_MIME.toLowerCase().replaceAll(/\s/g, '');
const UNKNOWN_APP_POLICY: MCPAppsPolicy = {
	enabled: false,
	allowAppInitiatedToolCalls: false,
	requireApprovalForOpenLink: true,
	requireApprovalForContextUpdates: true,
};

const DEFAULT_APP_HEIGHT = 480;
const MIN_APP_HEIGHT = 160;
const MAX_APP_HEIGHT = 1200;
const MAX_APPROVAL_PREVIEW_LENGTH = 4000;
const MAX_EXTERNAL_URL_LENGTH = 8192;

interface LoadedMCPAppResource {
	html: string;
	meta?: MCPAppUIResourceMeta;
}
interface MCPAppViewProps {
	instance: MCPAppInstance;
	toolInput?: unknown;
	toolResult?: { content?: MCPContent[]; structuredContent?: unknown; isError?: boolean };
	height?: number;
}

interface BlockedExternalLink {
	url: string;
	reason: 'unsafe' | 'open-failed';
}

function isAppMime(mime?: string): boolean {
	if (!mime) {
		return false;
	}
	const norm = mime.toLowerCase().replaceAll(/\s/g, '');
	return norm === NORMALIZED_APP_MIME || norm.startsWith(`${NORMALIZED_APP_MIME};`);
}

function decodeMCPBlob(blob: string | number[] | undefined): string {
	if (blob === undefined || (typeof blob === 'string' && blob.length === 0)) {
		return '';
	}
	if (typeof blob === 'string') {
		try {
			const binary = atob(blob);
			const bytes = new Uint8Array(binary.length);
			for (let i = 0; i < binary.length; i += 1) {
				// oxlint-disable-next-line unicorn/prefer-code-point
				bytes[i] = binary.charCodeAt(i);
			}
			return new TextDecoder().decode(bytes);
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
		if (c.type !== MCPContentType.Resource) {
			continue;
		}
		const res = c.resource;
		if (!res) {
			continue;
		}
		if (!isAppMime(res.mimeType)) {
			continue;
		}
		const text = typeof res.text === 'string' ? res.text : '';
		const html = text.trim() ? text : decodeMCPBlob(res.blob);
		if (!html.trim()) {
			continue;
		}
		return {
			html,
			meta: getMCPAppUIResourceMeta(res),
		};
	}
	return null;
}

function parseToolInput(raw: unknown): unknown {
	if (typeof raw !== 'string') {
		return raw;
	}
	const trimmed = raw.trim();
	if (!trimmed) {
		return undefined;
	}
	try {
		return JSON.parse(trimmed);
	} catch {
		return raw;
	}
}

function normalizeSafeExternalURL(raw: string): string | null {
	if (!raw || raw.length > MAX_EXTERNAL_URL_LENGTH) {
		return null;
	}
	try {
		const url = new URL(raw);
		if (url.protocol !== 'http:' && url.protocol !== 'https:') {
			return null;
		}
		if (url.username || url.password) {
			return null;
		}
		return url.href;
	} catch {
		return null;
	}
}

function clampAppHeight(value: number): number {
	if (!Number.isFinite(value)) {
		return DEFAULT_APP_HEIGHT;
	}
	return Math.max(MIN_APP_HEIGHT, Math.min(MAX_APP_HEIGHT, Math.round(value)));
}

function truncateForDisplay(value: string, maximumLength = MAX_APPROVAL_PREVIEW_LENGTH): string {
	if (value.length <= maximumLength) {
		return value;
	}
	return `${value.slice(0, maximumLength)}\n[Preview truncated]`;
}

function formatApprovalPreview(value: unknown): string {
	try {
		const serialized = typeof value === 'string' ? value : (JSON.stringify(value, null, 2) ?? String(value));
		return truncateForDisplay(serialized);
	} catch {
		return '[Preview unavailable]';
	}
}

function disposeMCPAppBridge(bridge: MCPAppPostMessageBridge, resourceUri: string, reason: string): void {
	try {
		void bridge.sendRequest('ui/resource-teardown', { resourceUri, reason }, 500).catch(() => undefined);
	} catch {
		// The bridge may already be disconnected.
	}

	// Posting the teardown request is synchronous. Dispose immediately so the
	// unmounted view cannot continue handling incoming messages for 500 ms.
	bridge.dispose();
}

function getMCPAppViewKey(instance: MCPAppInstance): string {
	return JSON.stringify([
		instance.instanceID,
		instance.server,
		instance.resourceUri,
		instance.toolName,
		instance.toolUseID,
	]);
}

function buildToolResultNotificationParams(
	toolName: string,
	toolUseID: string,
	toolResult: NonNullable<MCPAppViewProps['toolResult']>
): {
	toolName: string;
	toolUseID: string;
	content?: MCPContent[];
	structuredContent?: Record<string, unknown>;
	isError?: boolean;
} {
	const params: {
		toolName: string;
		toolUseID: string;
		content?: MCPContent[];
		structuredContent?: Record<string, unknown>;
		isError?: boolean;
	} = {
		toolName,
		toolUseID,
	};

	if (Array.isArray(toolResult.content)) {
		params.content = toolResult.content;
	}

	if (isJSONObject(toolResult.structuredContent)) {
		params.structuredContent = toolResult.structuredContent;
	}

	if (typeof toolResult.isError === 'boolean') {
		params.isError = toolResult.isError;
	}

	return params;
}

function buildToolInputNotificationParams(
	toolName: string,
	toolUseID: string,
	rawToolInput: unknown
): {
	toolName: string;
	toolUseID: string;
	arguments?: Record<string, unknown>;
} {
	const params: {
		toolName: string;
		toolUseID: string;
		arguments?: Record<string, unknown>;
	} = {
		toolName,
		toolUseID,
	};

	const parsed = parseToolInput(rawToolInput);

	if (isJSONObject(parsed)) {
		params.arguments = parsed;
	}

	return params;
}

export function MCPAppView(props: MCPAppViewProps) {
	return <MCPAppViewContent key={getMCPAppViewKey(props.instance)} {...props} />;
}

function MCPAppViewContent({ instance, toolInput, toolResult, height = DEFAULT_APP_HEIGHT }: MCPAppViewProps) {
	const [loadedResource, setLoadedResource] = useState<LoadedMCPAppResource | null>(null);
	const [loadError, setLoadError] = useState<string | null>(null);
	const [pendingURL, setPendingURL] = useState<string | null>(null);
	const [pendingUIMessage, setPendingUIMessage] = useState<MCPAppUIMessage | null>(null);
	const [pendingContextUpdate, setPendingContextUpdate] = useState<MCPAppModelContextUpdatePayload | null>(null);
	const [serverArtifact, setServerArtifact] = useState<ArtifactRef | null>(null);
	const [blockedLink, setBlockedLink] = useState<BlockedExternalLink | null>(null);
	const [initializedBridgeVersion, setInitializedBridgeVersion] = useState(0);
	const [appsPolicy, setAppsPolicy] = useState<MCPAppsPolicy | null>(null);
	const [policyError, setPolicyError] = useState<string | null>(null);
	const [appRequestedHeight, setAppRequestedHeight] = useState<number | null>(null);

	const approvalResolverRef = useRef<((ok: boolean) => void) | null>(null);
	const mountedRef = useRef(true);
	const mcpApproval = useMCPApproval();

	const bridgeRef = useRef<MCPAppPostMessageBridge | null>(null);
	const bridgeVersionRef = useRef(0);
	const sizeAnimationFrameRef = useRef<number | null>(null);
	const lastToolInputNotificationRef = useRef<{
		bridgeVersion: number;
		toolName: string;
		toolUseID: string;
		value: unknown;
	} | null>(null);
	const lastToolResultNotificationRef = useRef<{
		bridgeVersion: number;
		toolName: string;
		toolUseID: string;
		value: MCPAppViewProps['toolResult'];
	} | null>(null);

	const server = instance.server;
	const appsEnabled = appsPolicy?.enabled === true;
	const effectiveAppsPolicy = appsPolicy ?? UNKNOWN_APP_POLICY;
	const serverLabel = serverArtifact ? `${serverArtifact.rootID}/${serverArtifact.artifactID}` : server;
	const frameHeight = appRequestedHeight ?? clampAppHeight(height);

	useEffect(() => {
		if (!appsEnabled) {
			return;
		}

		let cancelled = false;

		void mcpManagementAPI
			.readMCPResource(server, instance.resourceUri)
			.then(resp => {
				if (cancelled) {
					return;
				}
				const extracted = extractAppHTML(resp?.contents);
				if (!extracted) {
					setLoadError('UI resource did not return MCP App HTML.');
					return;
				}
				setLoadedResource(extracted);
			})
			.catch((err: unknown) => {
				if (cancelled) {
					return;
				}
				setLoadError(err instanceof Error ? err.message : 'Failed to load MCP App.');
			});

		return () => {
			cancelled = true;
		};
	}, [appsEnabled, instance.resourceUri, server]);

	useEffect(() => {
		let cancelled = false;

		void mcpManagementAPI
			.artifactRefForRuntimeServerID(server)
			.then(async artifact => {
				if (cancelled) {
					return null;
				}
				return {
					artifact,
					resolved: await mcpManagementAPI.inspectMCPServer(artifact),
				};
			})
			.then(result => {
				if (cancelled || !result) {
					return;
				}

				const { artifact, resolved } = result;
				setServerArtifact(artifact);
				const nextPolicy = resolved.policy.body.appsPolicy;
				if (!nextPolicy) {
					setPolicyError('The server did not return an MCP Apps policy.');
					return;
				}
				if (!nextPolicy.enabled) {
					setPolicyError('MCP Apps is currently disabled for this server.');
					return;
				}
				setAppsPolicy(nextPolicy);
			})
			.catch((err: unknown) => {
				if (cancelled) {
					return;
				}
				setPolicyError(err instanceof Error ? err.message : 'Could not verify MCP Apps policy.');
			});

		return () => {
			cancelled = true;
		};
	}, [server]);

	useEffect(() => {
		return () => {
			if (sizeAnimationFrameRef.current !== null) {
				window.cancelAnimationFrame(sizeAnimationFrameRef.current);
			}
		};
	}, []);

	const requestOpenLinkApproval = useCallback(
		async (url: string) => {
			if (!mountedRef.current) {
				return false;
			}

			const safeURL = normalizeSafeExternalURL(url);
			if (!safeURL) {
				setBlockedLink({ url, reason: 'unsafe' });
				return false;
			}
			if (approvalResolverRef.current) {
				return false;
			}
			if (!effectiveAppsPolicy.requireApprovalForOpenLink) {
				try {
					backendAPI.openURL(safeURL);
					return true;
				} catch {
					if (mountedRef.current) {
						setBlockedLink({ url: safeURL, reason: 'open-failed' });
					}
					return false;
				}
			}
			return await new Promise<boolean>(resolve => {
				approvalResolverRef.current = resolve;
				setPendingURL(safeURL);
			});
		},
		[effectiveAppsPolicy.requireApprovalForOpenLink]
	);

	const applySizeChanged = useCallback((params: unknown) => {
		if (!params || typeof params !== 'object') {
			return;
		}
		const heightValue = (params as Record<string, unknown>).height;
		if (typeof heightValue !== 'number' || !Number.isFinite(heightValue) || heightValue <= 0) {
			return;
		}
		const nextHeight = clampAppHeight(heightValue);

		if (sizeAnimationFrameRef.current !== null) {
			window.cancelAnimationFrame(sizeAnimationFrameRef.current);
		}
		sizeAnimationFrameRef.current = window.requestAnimationFrame(() => {
			sizeAnimationFrameRef.current = null;
			setAppRequestedHeight(current => (current !== null && Math.abs(current - nextHeight) < 4 ? current : nextHeight));
		});
	}, []);

	const router = useMemo(
		() =>
			// oxlint-disable-next-line react/refs
			new MCPAppRPCRouter({
				instance,
				requestOpenLinkApproval,
				requestMCPApproval: mcpApproval.requestMCPApproval,
				requestUIMessageApproval: async message => {
					if (approvalResolverRef.current) {
						return false;
					}
					return await new Promise<boolean>(resolve => {
						approvalResolverRef.current = resolve;
						setPendingUIMessage(message);
					});
				},
				onUIMessage: message => {
					dispatchMCPAppUIMessage(instance, message);
				},
				requestModelContextUpdateApproval: async update => {
					if (!effectiveAppsPolicy.requireApprovalForContextUpdates) {
						return true;
					}
					if (approvalResolverRef.current) {
						return false;
					}

					return await new Promise<boolean>(resolve => {
						approvalResolverRef.current = resolve;
						setPendingContextUpdate(update);
					});
				},
				onModelContextUpdate: update => {
					dispatchMCPAppModelContextUpdate(instance, update);
				},
				onAppLog: () => {},
			}),
		[
			effectiveAppsPolicy.requireApprovalForContextUpdates,
			instance,
			mcpApproval.requestMCPApproval,
			requestOpenLinkApproval,
		]
	);

	const routerRef = useRef(router);
	useEffect(() => {
		routerRef.current = router;
	}, [router]);

	const sandboxConfiguration = useMemo(() => {
		if (!loadedResource) {
			return null;
		}
		return {
			csp: buildMCPAppCSP(loadedResource.meta),
			allow: buildMCPAppAllowAttribute(loadedResource.meta),
		};
	}, [loadedResource]);

	const handleIframeReady = useCallback(
		(iframe: HTMLIFrameElement) => {
			const bridgeVersion = bridgeVersionRef.current + 1;
			bridgeVersionRef.current = bridgeVersion;

			const previousBridge = bridgeRef.current;
			bridgeRef.current = null;
			if (previousBridge) {
				const pendingResolver = approvalResolverRef.current;
				if (pendingResolver) {
					approvalResolverRef.current = null;
					setPendingURL(null);
					setPendingUIMessage(null);
					setPendingContextUpdate(null);
					pendingResolver(false);
				}
				disposeMCPAppBridge(previousBridge, instance.resourceUri, 'iframe replaced');
			}

			if (sizeAnimationFrameRef.current !== null) {
				window.cancelAnimationFrame(sizeAnimationFrameRef.current);
				sizeAnimationFrameRef.current = null;
			}

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
								allowToolCalls: effectiveAppsPolicy.enabled && effectiveAppsPolicy.allowAppInitiatedToolCalls,
							}),
						};
					}
					return routerRef.current.handle(req);
				},
				onNotification: note => {
					if (bridgeVersionRef.current !== bridgeVersion) {
						return;
					}
					if (note.method === 'ui/notifications/initialized' || note.method === 'notifications/initialized') {
						setInitializedBridgeVersion(bridgeVersion);
						return;
					}
					if (note.method === 'ui/notifications/size-changed') {
						applySizeChanged(note.params);
					}
				},
			});
			bridgeRef.current = bridge;
		},
		[
			applySizeChanged,
			effectiveAppsPolicy.allowAppInitiatedToolCalls,
			effectiveAppsPolicy.enabled,
			instance.resourceUri,
		]
	);

	useEffect(() => {
		const bridge = bridgeRef.current;
		if (!bridge || initializedBridgeVersion === 0 || initializedBridgeVersion !== bridgeVersionRef.current) {
			return;
		}

		if (toolInput !== undefined) {
			const lastNotification = lastToolInputNotificationRef.current;
			if (
				lastNotification?.bridgeVersion === initializedBridgeVersion &&
				lastNotification.toolName === instance.toolName &&
				lastNotification.toolUseID === instance.toolUseID &&
				Object.is(lastNotification.value, toolInput)
			) {
				return;
			}

			bridge.sendNotification(
				'ui/notifications/tool-input',
				buildToolInputNotificationParams(instance.toolName, instance.toolUseID, toolInput)
			);
			lastToolInputNotificationRef.current = {
				bridgeVersion: initializedBridgeVersion,
				toolName: instance.toolName,
				toolUseID: instance.toolUseID,
				value: toolInput,
			};
		}
	}, [initializedBridgeVersion, instance.toolName, instance.toolUseID, toolInput]);

	useEffect(() => {
		const bridge = bridgeRef.current;
		if (!bridge || initializedBridgeVersion === 0 || initializedBridgeVersion !== bridgeVersionRef.current) {
			return;
		}

		if (toolResult !== undefined) {
			const lastNotification = lastToolResultNotificationRef.current;
			if (
				lastNotification?.bridgeVersion === initializedBridgeVersion &&
				lastNotification.toolName === instance.toolName &&
				lastNotification.toolUseID === instance.toolUseID &&
				Object.is(lastNotification.value, toolResult)
			) {
				return;
			}

			bridge.sendNotification(
				'ui/notifications/tool-result',
				buildToolResultNotificationParams(instance.toolName, instance.toolUseID, toolResult)
			);
			lastToolResultNotificationRef.current = {
				bridgeVersion: initializedBridgeVersion,
				toolName: instance.toolName,
				toolUseID: instance.toolUseID,
				value: toolResult,
			};
		}
	}, [initializedBridgeVersion, instance.toolName, instance.toolUseID, toolResult]);

	useEffect(() => {
		return () => {
			bridgeVersionRef.current += 1;
			const bridge = bridgeRef.current;
			bridgeRef.current = null;
			if (!bridge) {
				return;
			}

			disposeMCPAppBridge(bridge, instance.resourceUri, 'view unmounted');
		};
	}, [instance.resourceUri]);

	useEffect(() => {
		mountedRef.current = true;
		return () => {
			mountedRef.current = false;
			approvalResolverRef.current?.(false);
			approvalResolverRef.current = null;
		};
	}, []);

	if (!appsPolicy && !policyError) {
		return (
			<output className="text-base-content/60 text-xs" aria-live="polite">
				Verifying MCP App policy…
			</output>
		);
	}

	if (policyError) {
		return (
			<div className="alert alert-warning rounded-2xl text-sm" role="alert">
				<div className="flex items-center gap-2">
					<FiAlertTriangle size={14} aria-hidden="true" />
					<span>MCP App is unavailable: {policyError}</span>
				</div>
			</div>
		);
	}
	if (loadError) {
		return (
			<div className="alert alert-warning rounded-2xl text-sm" role="alert">
				<div className="flex items-center gap-2">
					<FiAlertTriangle size={14} aria-hidden="true" />
					<span>MCP App failed to load: {loadError}</span>
				</div>
			</div>
		);
	}

	if (!loadedResource || !sandboxConfiguration) {
		return (
			<output className="text-base-content/60 text-xs" aria-live="polite">
				Loading MCP App…
			</output>
		);
	}

	return (
		<>
			<div className="border-base-content/10 bg-base-200 rounded-2xl border p-2">
				<div className="mb-2 flex items-center justify-between gap-2 text-xs">
					<span className="truncate font-semibold">{instance.displayName ?? instance.toolName}</span>
					<span
						className="text-base-content/60 truncate"
						title={serverArtifact ? `${serverArtifact.rootID}/${serverArtifact.artifactID}` : server}
					>
						{serverLabel}
					</span>
				</div>
				<MCPAppSandbox
					html={loadedResource.html}
					csp={sandboxConfiguration.csp}
					title={`MCP App ${instance.toolName}`}
					onIframeReady={handleIframeReady}
					height={frameHeight}
					allow={sandboxConfiguration.allow}
				/>
			</div>

			<DeleteConfirmationModal
				isOpen={pendingURL !== null}
				title="Open external link?"
				message={`The MCP App for ${serverLabel} wants to open:\n${truncateForDisplay(pendingURL ?? '')}`}
				confirmButtonText="Open"
				onConfirm={() => {
					const url = pendingURL;
					const resolver = approvalResolverRef.current;
					approvalResolverRef.current = null;
					setPendingURL(null);
					if (!url || !resolver) {
						resolver?.(false);
						return;
					}

					void (async () => {
						try {
							backendAPI.openURL(url);
							resolver(true);
						} catch {
							if (mountedRef.current) {
								setBlockedLink({ url, reason: 'open-failed' });
							}
							resolver(false);
						}
					})();
				}}
				onClose={() => {
					const resolver = approvalResolverRef.current;
					approvalResolverRef.current = null;
					setPendingURL(null);
					resolver?.(false);
				}}
			/>
			<DeleteConfirmationModal
				isOpen={pendingUIMessage !== null}
				title="Add message from MCP App?"
				message={`The MCP App for ${serverLabel} wants to add this draft message:\n\n${truncateForDisplay(pendingUIMessage?.text ?? '')}`}
				confirmButtonText="Add draft"
				onConfirm={() => {
					const resolver = approvalResolverRef.current;
					approvalResolverRef.current = null;
					setPendingUIMessage(null);
					resolver?.(true);
				}}
				onClose={() => {
					const resolver = approvalResolverRef.current;
					approvalResolverRef.current = null;
					setPendingUIMessage(null);
					resolver?.(false);
				}}
			/>

			<DeleteConfirmationModal
				isOpen={pendingContextUpdate !== null}
				title="Allow MCP App model context?"
				message={`The MCP App for ${serverLabel} wants to add this context to the next model request:\n\n${
					pendingContextUpdate ? formatApprovalPreview(pendingContextUpdate) : ''
				}`}
				confirmButtonText="Allow"
				onConfirm={() => {
					const resolver = approvalResolverRef.current;
					approvalResolverRef.current = null;
					setPendingContextUpdate(null);
					resolver?.(true);
				}}
				onClose={() => {
					const resolver = approvalResolverRef.current;
					approvalResolverRef.current = null;
					setPendingContextUpdate(null);
					resolver?.(false);
				}}
			/>
			<ActionDeniedAlertModal
				isOpen={blockedLink !== null}
				onClose={() => {
					setBlockedLink(null);
				}}
				message={
					blockedLink
						? blockedLink.reason === 'unsafe'
							? `Blocked an unsafe or unsupported external link:\n${truncateForDisplay(blockedLink.url)}\n\nOnly HTTP and HTTPS links without embedded credentials are allowed.`
							: `Could not open:\n${truncateForDisplay(blockedLink.url)}`
						: ''
				}
			/>
			<MCPApprovalModal
				approvalRequest={mcpApproval.approvalRequest}
				isResolving={mcpApproval.isResolving}
				error={mcpApproval.approvalError}
				onResolve={r => {
					return mcpApproval.resolveMCPApproval(r);
				}}
			/>
		</>
	);
}
