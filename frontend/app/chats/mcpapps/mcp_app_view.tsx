import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FiAlertTriangle } from 'react-icons/fi';

import type { SessionToolResult } from '@/spec/session';
import type { MCPContent } from '@/spec/session_mcp';
import { MCP_APP_HTML_MIME_TYPE } from '@/spec/mcp';
import { SessionRunObservationKind } from '@/spec/session';
import { SessionToolCallSource, SessionToolResultKind } from '@/spec/session_base';
import { MCPContentType } from '@/spec/session_mcp';

import { isJSONObject } from '@/lib/jsonschema_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import { backendAPI, mcpManagementAPI, sessionManagementAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import type { MCPAppUIResourceMeta } from '@/chats/composer/mcp/mcp_apps_csp';
import type { MCPAppModelContextUpdatePayload, MCPAppUIMessage } from '@/chats/mcpapps/mcp_app_events';
import type { MCPAppToolCallResult } from '@/chats/mcpapps/mcp_app_rpc_router';
import type { JSONRPCResponse, MCPAppInstance } from '@/chats/mcpapps/mcp_app_types';
import { buildMCPAppAllowAttribute, buildMCPAppCSP, getMCPAppUIResourceMeta } from '@/chats/composer/mcp/mcp_apps_csp';
import { dispatchMCPAppUIMessage } from '@/chats/mcpapps/mcp_app_events';
import { buildMCPAppHostContext } from '@/chats/mcpapps/mcp_app_host_context';
import { MCPAppPostMessageBridge } from '@/chats/mcpapps/mcp_app_postmessage_bridge';
import { MCPAppRPCRouter } from '@/chats/mcpapps/mcp_app_rpc_router';
import { MCPAppSandbox } from '@/chats/mcpapps/mcp_app_sandbox';
import { requireAcceptedSessionCommand, sessionCommandMeta } from '@/chats/session/session_commands';
import { decodeSessionOpaqueJSON } from '@/chats/session/session_projection';

const APP_MIME = MCP_APP_HTML_MIME_TYPE;
const NORMALIZED_APP_MIME = APP_MIME.toLowerCase().replaceAll(/\s/g, '');

const DEFAULT_APP_HEIGHT = 480;
const MIN_APP_HEIGHT = 160;
const MAX_APP_HEIGHT = 1200;
const APP_TOOL_RESULT_TIMEOUT_MS = 10 * 60 * 1000;
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

interface PendingAppToolResult {
	resolve: (result: MCPAppToolCallResult) => void;
	reject: (error: Error) => void;
	timer: number;
}

interface BlockedExternalLink {
	url: string;
	reason: 'unsafe' | 'open-failed';
}

interface ResolvedServerLabel {
	runtimeID: string;
	label: string;
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
			const bytes = Uint8Array.from(binary, character => character.codePointAt(0) ?? 0);
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

function formatModelContextUpdatePreview(update: MCPAppModelContextUpdatePayload): string {
	return formatApprovalPreview({
		...(update.content === undefined ? {} : { content: update.content }),
		...(update.structuredContent === undefined ? {} : { structuredContent: update.structuredContent }),
	});
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
		instance.callID,
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

function stringifyOpaqueValue(value: unknown): string {
	try {
		return JSON.stringify(value, null, 2) ?? '';
	} catch {
		return '';
	}
}

function sessionToolResultToAppToolCallResult(result: SessionToolResult): MCPAppToolCallResult {
	const appPayload = decodeSessionOpaqueJSON<{
		content?: MCPContent[];
		structuredContent?: unknown;
		isError?: boolean;
	}>(result.app);
	const structuredContent = appPayload?.structuredContent ?? decodeSessionOpaqueJSON(result.structured);
	const fallbackText =
		result.text ||
		result.error ||
		(structuredContent === undefined ? '' : stringifyOpaqueValue(structuredContent)) ||
		'Tool completed without textual output.';

	return {
		content: Array.isArray(appPayload?.content)
			? appPayload.content
			: [
					{
						type: MCPContentType.Text,
						text: fallbackText,
					},
				],
		...(isJSONObject(structuredContent) ? { structuredContent } : {}),
		isError: appPayload?.isError ?? result.kind !== SessionToolResultKind.Success,
	};
}

export function MCPAppView(props: MCPAppViewProps) {
	return <MCPAppViewContent key={getMCPAppViewKey(props.instance)} {...props} />;
}

function MCPAppViewContent({ instance, toolInput, toolResult, height = DEFAULT_APP_HEIGHT }: MCPAppViewProps) {
	const [routerInstance] = useState(() => instance);
	const [loadedResource, setLoadedResource] = useState<LoadedMCPAppResource | null>(null);
	const [loadError, setLoadError] = useState<string | null>(null);
	const [pendingURL, setPendingURL] = useState<string | null>(null);
	const [pendingUIMessage, setPendingUIMessage] = useState<MCPAppUIMessage | null>(null);
	const [pendingContextUpdate, setPendingContextUpdate] = useState<MCPAppModelContextUpdatePayload | null>(null);
	const [blockedLink, setBlockedLink] = useState<BlockedExternalLink | null>(null);
	const [initializedBridgeVersion, setInitializedBridgeVersion] = useState(0);
	const [appRequestedHeight, setAppRequestedHeight] = useState<number | null>(null);
	const [sessionError, setSessionError] = useState<string | null>(null);
	const [viewID] = useState(() => `mcp-app-view-${getUUIDv7()}`);
	const [resolvedServerLabel, setResolvedServerLabel] = useState<ResolvedServerLabel | null>(null);
	const serverLabel = resolvedServerLabel?.runtimeID === instance.server ? resolvedServerLabel.label : 'MCP server';
	const revisionsRef = useRef({
		sessionRevision: instance.session.sessionRevision,
		runRevision: instance.session.runRevision,
	});

	const approvalResolverRef = useRef<((ok: boolean) => void) | null>(null);
	const mountedRef = useRef(true);
	const isMCPAppViewActive = useCallback(() => mountedRef.current, []);

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
	const pendingAppToolResultsRef = useRef(new Map<string, PendingAppToolResult>());
	const completedAppToolResultsRef = useRef(new Map<string, MCPAppToolCallResult>());
	const frameHeight = appRequestedHeight ?? clampAppHeight(height);

	useEffect(() => {
		let cancelled = false;
		const runtimeID = instance.server;

		void mcpManagementAPI
			.findComposerServerByRuntimeID(runtimeID)
			.then(server => {
				if (cancelled || !server) {
					return;
				}

				setResolvedServerLabel({
					runtimeID,
					label: server.displayName.trim() || server.logicalName.trim() || 'MCP server',
				});
			})
			.catch(() => {
				// The runtime ID remains a useful fallback when the management
				// catalog no longer contains the server.
			});

		return () => {
			cancelled = true;
		};
	}, [instance.server]);

	useEffect(() => {
		let cancelled = false;

		void (async () => {
			try {
				const opened = requireAcceptedSessionCommand(
					await sessionManagementAPI.openAppView({
						meta: sessionCommandMeta(revisionsRef.current.sessionRevision, revisionsRef.current.runRevision),
						sessionID: instance.session.sessionID,
						runID: instance.session.runID,
						appInstanceID: instance.session.appInstanceID,
						viewID,
					}),
					'Open MCP App view'
				);

				revisionsRef.current = {
					sessionRevision: opened.sessionRevision ?? revisionsRef.current.sessionRevision,
					runRevision: opened.runRevision ?? revisionsRef.current.runRevision,
				};

				const raw = await sessionManagementAPI.readAppResource({
					sessionID: instance.session.sessionID,
					runID: instance.session.runID,
					appInstanceID: instance.session.appInstanceID,
					viewID,
					resourceURI: instance.resourceUri,
				});

				if (cancelled) {
					return;
				}
				const response = decodeSessionOpaqueJSON<{ contents?: MCPContent[] }>(raw.content);
				const extracted = extractAppHTML(response?.contents);
				if (!extracted) {
					setLoadError('UI resource did not return MCP App HTML.');
					return;
				}
				setLoadedResource(extracted);
			} catch (error) {
				if (cancelled) {
					return;
				}
				setSessionError(error instanceof Error ? error.message : 'Failed to open the Session-scoped MCP App.');
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [
		instance.resourceUri,
		instance.session.appInstanceID,
		instance.session.runID,
		instance.session.sessionID,
		viewID,
	]);

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

			if (approvalResolverRef.current) {
				return false;
			}
			if (!safeURL) {
				setBlockedLink({ url, reason: 'unsafe' });
				return false;
			}
			if (routerInstance.requireOpenLinkApproval === false) {
				try {
					backendAPI.openURL(safeURL);
					return true;
				} catch {
					setBlockedLink({ url: safeURL, reason: 'open-failed' });
					return false;
				}
			}
			return await new Promise<boolean>(resolve => {
				approvalResolverRef.current = resolve;
				setPendingURL(safeURL);
			});
		},
		[routerInstance.requireOpenLinkApproval]
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

	const settleAppToolResult = useCallback((callID: string, result: MCPAppToolCallResult) => {
		completedAppToolResultsRef.current.set(callID, result);
		const pending = pendingAppToolResultsRef.current.get(callID);
		if (!pending) {
			return;
		}

		pendingAppToolResultsRef.current.delete(callID);
		window.clearTimeout(pending.timer);
		pending.resolve(result);
	}, []);

	const waitForAppToolResult = useCallback((callID: string): Promise<MCPAppToolCallResult> => {
		const completed = completedAppToolResultsRef.current.get(callID);
		if (completed) {
			return Promise.resolve(completed);
		}

		return new Promise<MCPAppToolCallResult>((resolve, reject) => {
			const timer = window.setTimeout(() => {
				pendingAppToolResultsRef.current.delete(callID);
				reject(new Error('Timed out waiting for the MCP App Tool result.'));
			}, APP_TOOL_RESULT_TIMEOUT_MS);

			pendingAppToolResultsRef.current.set(callID, {
				resolve,
				reject,
				timer,
			});
		});
	}, []);

	useEffect(() => {
		const pendingAppToolResults = pendingAppToolResultsRef.current;
		const completedAppToolResults = completedAppToolResultsRef.current;

		return () => {
			for (const pending of pendingAppToolResults.values()) {
				window.clearTimeout(pending.timer);
				pending.reject(new Error('MCP App view was closed before the Tool result arrived.'));
			}
			pendingAppToolResults.clear();
			completedAppToolResults.clear();
		};
	}, []);

	const router = useMemo(
		() =>
			// oxlint-disable-next-line react/refs
			new MCPAppRPCRouter({
				instance: routerInstance,
				isActive: isMCPAppViewActive,
				allowAppToolCalls: routerInstance.allowToolCalls ?? true,
				executeAppTool: async (toolName, args) => {
					const receipt = requireAcceptedSessionCommand(
						await sessionManagementAPI.executeAppTool({
							meta: sessionCommandMeta(revisionsRef.current.sessionRevision, revisionsRef.current.runRevision),
							sessionID: instance.session.sessionID,
							runID: instance.session.runID,
							appInstanceID: instance.session.appInstanceID,
							viewID,
							toolName,
							arguments: args,
						}),
						'Execute MCP App Tool'
					);
					revisionsRef.current = {
						sessionRevision: receipt.sessionRevision ?? revisionsRef.current.sessionRevision,
						runRevision: receipt.runRevision ?? revisionsRef.current.runRevision,
					};
					const callID = receipt.toolCallIDs?.[0];
					if (!callID) {
						return { accepted: true };
					}

					return waitForAppToolResult(callID);
				},
				readAppResource: async resourceURI => {
					const response = await sessionManagementAPI.readAppResource({
						sessionID: instance.session.sessionID,
						runID: instance.session.runID,
						appInstanceID: instance.session.appInstanceID,
						viewID,
						resourceURI,
					});
					return decodeSessionOpaqueJSON(response.content);
				},
				stageModelContext: async update => {
					const receipt = requireAcceptedSessionCommand(
						await sessionManagementAPI.stageAppContext({
							meta: sessionCommandMeta(revisionsRef.current.sessionRevision, revisionsRef.current.runRevision),
							sessionID: instance.session.sessionID,
							runID: instance.session.runID,
							appInstanceID: instance.session.appInstanceID,
							viewID,
							update: {
								appInstanceID: instance.session.appInstanceID,
								viewID,
								serverID: instance.session.serverID,
								resourceURI: instance.session.resourceURI,
								content: update.content?.map(content => ({
									type: content.type,
									text: content.text ?? content.resource?.text,
									uri: content.uri ?? content.resource?.uri,
									name: content.name,
									description: content.description,
									mimeType: content.mimeType ?? content.resource?.mimeType,
									data: content.data ?? content.resource?.blob,
								})),
								structuredContent: update.structuredContent,
							},
						}),
						'Stage MCP App context'
					);
					revisionsRef.current = {
						sessionRevision: receipt.sessionRevision ?? revisionsRef.current.sessionRevision,
						runRevision: receipt.runRevision ?? revisionsRef.current.runRevision,
					};
				},
				requestModelContextUpdateApproval: async update => {
					if (routerInstance.requireContextUpdateApproval === false) {
						return true;
					}
					if (!mountedRef.current || approvalResolverRef.current) {
						return false;
					}
					return await new Promise<boolean>(resolve => {
						approvalResolverRef.current = resolve;
						setPendingContextUpdate(update);
					});
				},
				requestOpenLinkApproval,
				requestUIMessageApproval: async message => {
					if (!mountedRef.current || approvalResolverRef.current) {
						return false;
					}
					return await new Promise<boolean>(resolve => {
						approvalResolverRef.current = resolve;
						setPendingUIMessage(message);
					});
				},
				onUIMessage: message => {
					dispatchMCPAppUIMessage(routerInstance, message);
				},
			}),
		[
			instance.session.appInstanceID,
			instance.session.resourceURI,
			instance.session.runID,
			instance.session.serverID,
			instance.session.sessionID,
			isMCPAppViewActive,
			requestOpenLinkApproval,
			routerInstance,
			viewID,
			waitForAppToolResult,
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
								allowToolCalls: routerInstance.allowToolCalls ?? true,
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
		[applySizeChanged, instance.resourceUri, routerInstance.allowToolCalls]
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
		const sentResultIDs = new Set<string>();

		return sessionManagementAPI.observeRun(instance.session.sessionID, instance.session.runID, observation => {
			if (observation.kind !== SessionRunObservationKind.Snapshot || !observation.snapshot) {
				return;
			}

			revisionsRef.current = {
				sessionRevision: observation.snapshot.sessionRevision,
				runRevision: observation.snapshot.run.revision,
			};

			for (const call of observation.snapshot.run.toolCalls ?? []) {
				if (call.source !== SessionToolCallSource.App || call.appInstanceID !== instance.session.appInstanceID) {
					continue;
				}

				for (const result of call.results ?? []) {
					const appToolResult = sessionToolResultToAppToolCallResult(result);
					settleAppToolResult(call.id, appToolResult);

					if (sentResultIDs.has(result.id)) {
						continue;
					}
					sentResultIDs.add(result.id);

					const bridge = bridgeRef.current;
					if (!bridge) {
						continue;
					}

					bridge.sendNotification('ui/notifications/tool-result', {
						toolName: result.name,
						toolUseID: result.providerCallID || result.callID,
						content: appToolResult.content,
						structuredContent: appToolResult.structuredContent,
						isError: appToolResult.isError,
					});
				}
			}
		});
	}, [instance.session.appInstanceID, instance.session.runID, instance.session.sessionID, settleAppToolResult]);

	useEffect(() => {
		mountedRef.current = true;
		return () => {
			mountedRef.current = false;
			approvalResolverRef.current?.(false);
			approvalResolverRef.current = null;
		};
	}, []);

	if (sessionError) {
		return (
			<div className="alert alert-warning rounded-2xl text-sm" role="alert">
				<div className="flex items-center gap-2">
					<FiAlertTriangle size={14} aria-hidden="true" />
					<span>MCP App is unavailable: {sessionError}</span>
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
					<span className="text-base-content/60 truncate" title={serverLabel}>
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

					try {
						backendAPI.openURL(url);
						resolver(true);
					} catch {
						if (mountedRef.current) {
							setBlockedLink({ url, reason: 'open-failed' });
						}
						resolver(false);
					}
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
					pendingContextUpdate ? formatModelContextUpdatePreview(pendingContextUpdate) : ''
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
		</>
	);
}
