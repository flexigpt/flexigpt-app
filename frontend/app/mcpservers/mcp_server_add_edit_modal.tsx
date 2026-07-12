import type { ChangeEvent, SubmitEventHandler } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiPlus, FiTrash2, FiUpload, FiX } from 'react-icons/fi';

import type { MCPServerConfig, MCPToolPolicyOverride, PutMCPServerPayload } from '@/spec/mcp';
import { MCPApprovalRule, MCPExecutionMode, MCPHTTPAuthMode, MCPTransportType, MCPTrustLevel } from '@/spec/mcp';

import { omitManyKeys } from '@/lib/obj_utils';
import { validateSlug } from '@/lib/text_utils';

import type { DropdownItem } from '@/components/dropdown';
import { Dropdown } from '@/components/dropdown';
import { ModalBackdrop } from '@/components/modal_backdrop';

import type { MCPServerUpsertInput } from '@/mcpservers/lib/mcp_server_utils';
import {
	getDefaultMCPAppsPolicy,
	getDefaultMCPServerPolicy,
	getMCPApprovalRuleLabel,
	getMCPExecutionModeLabel,
	getMCPHTTPAuthModeLabel,
	getMCPTransportLabel,
	getMCPTrustLevelLabel,
	MCP_OAUTH_CLIENT_CREDENTIALS_SLOT,
	parseMCPObjectJSON,
	parseMCPStringRecordJSON,
	stringifyMCPJSON,
} from '@/mcpservers/lib/mcp_server_utils';

type ModalMode = 'add' | 'edit';

const TRANSPORT_DROPDOWN_ITEMS: Record<MCPTransportType, DropdownItem> = {
	[MCPTransportType.MCPTransportTypeStreamableHTTP]: { isEnabled: true },
	[MCPTransportType.MCPTransportTypeStdio]: { isEnabled: true },
};

const TRUST_DROPDOWN_ITEMS: Record<MCPTrustLevel, DropdownItem> = {
	[MCPTrustLevel.MCPTrustLevelUntrusted]: { isEnabled: true },
	[MCPTrustLevel.MCPTrustLevelTrusted]: { isEnabled: true },
};

const AUTH_MODE_DROPDOWN_ITEMS: Record<MCPHTTPAuthMode, DropdownItem> = {
	[MCPHTTPAuthMode.MCPHTTPAuthNone]: { isEnabled: true },
	[MCPHTTPAuthMode.MCPHTTPAuthAPIKey]: { isEnabled: true },
	[MCPHTTPAuthMode.MCPHTTPAuthOAuth]: { isEnabled: true },
	[MCPHTTPAuthMode.MCPHTTPAuthClientCredentials]: { isEnabled: true },
};

const APPROVAL_RULE_DROPDOWN_ITEMS: Record<MCPApprovalRule, DropdownItem> = {
	[MCPApprovalRule.MCPApprovalRuleAsk]: { isEnabled: true },
	[MCPApprovalRule.MCPApprovalRuleAllow]: { isEnabled: true },
	[MCPApprovalRule.MCPApprovalRuleDeny]: { isEnabled: true },
};

const EXECUTION_MODE_DROPDOWN_ITEMS: Record<MCPExecutionMode, DropdownItem> = {
	[MCPExecutionMode.MCPExecutionModeManual]: { isEnabled: true },
	[MCPExecutionMode.MCPExecutionModeAuto]: { isEnabled: true },
};

function isIPv4LoopbackHost(host: string): boolean {
	const parts = host.split('.').map(Number);

	return (
		parts.length === 4 && parts.every(part => Number.isInteger(part) && part >= 0 && part <= 255) && parts[0] === 127
	);
}

function isLoopbackHTTPHost(host: string): boolean {
	const normalized = host.trim().toLowerCase().replace(/^\[/, '').replace(/\]$/, '');

	if (!normalized) {
		return false;
	}
	if (normalized === 'localhost') {
		return true;
	}
	if (normalized === '::1') {
		return true;
	}
	if (normalized === '0:0:0:0:0:0:0:1') {
		return true;
	}

	return isIPv4LoopbackHost(normalized);
}

interface AddEditMCPServerModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (serverData: MCPServerUpsertInput) => Promise<void>;
	initialData?: MCPServerConfig;
	existingServerIDs: string[];
	prefillServers?: MCPServerConfig[];
	mode?: ModalMode;
}

interface ErrorState {
	serverID?: string;
	displayName?: string;
	stdioCommand?: string;
	stdioEnvJSON?: string;
	stdioStartupTimeoutMS?: string;
	stdioSecrets?: string;
	httpURL?: string;
	httpTimeoutMS?: string;
	httpClientCredentials?: string;
	httpClientIDMetadataDocumentURL?: string;
	httpAPIKey?: string;
	policies?: string;
	toolPoliciesJSON?: string;
}

interface SecretEnvRow {
	rowID: string;
	envName: string;
	originalEnvName?: string;
	slot: string;
	existingSecretRef?: string;
	secretValue: string;
	deleteExisting: boolean;
}

interface MCPServerFormData {
	serverID: string;
	displayName: string;
	enabled: boolean;
	transport: MCPTransportType;
	trustLevel: MCPTrustLevel;
	stdioCommand: string;
	stdioArgsText: string;
	stdioWorkingDir: string;
	stdioEnvJSON: string;
	stdioStartupTimeoutMS: string;
	stdioSecretRows: SecretEnvRow[];

	httpURL: string;
	httpTimeoutMS: string;
	httpAuthMode: MCPHTTPAuthMode;
	httpClientCredentialRef: string;
	httpClientCredentialsSecret: string;
	httpDeleteClientCredentials: boolean;
	httpClientIDMetadataDocumentURL: string;

	apiKeyHeaderName: string;
	apiKeyOriginalHeaderName: string;
	apiKeyValuePrefix: string;
	apiKeyValue: string;
	apiKeyExistingRef?: string;
	apiKeyDeleteExisting: boolean;

	defaultApprovalRule: MCPApprovalRule;
	defaultExecutionMode: MCPExecutionMode;
	requireApprovalForUnknownRisk: boolean;
	requireApprovalForWrite: boolean;
	requireApprovalForDestructive: boolean;

	appsPolicyEnabled: boolean;
	allowAppInitiatedToolCalls: boolean;
	requireApprovalForOpenLink: boolean;
	requireApprovalForContextUpdates: boolean;

	toolPoliciesJSON: string;
}

const ENV_NAME_RE = /^[A-Za-z_][A-Za-z0-9_]*$/;

function sameHTTPHeaderName(a: string, b: string): boolean {
	return a.trim().toLowerCase() === b.trim().toLowerCase();
}

function getDefaultAPIKeyValuePrefix(headerName: string): string {
	return sameHTTPHeaderName(headerName, 'Authorization') ? 'Bearer ' : '';
}

function hasInvalidHTTPHeaderValueChars(value: string): boolean {
	// oxlint-disable-next-line no-control-regex
	return /[\r\n\u0000]/.test(value);
}

function makeRowID(): string {
	return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function buildMCPServerPrefillKey(server: MCPServerConfig): string {
	return `${server.bundleID}:${server.id}`;
}

function splitArgsText(argsText: string): string[] | undefined {
	const args = argsText
		.split('\n')
		.map(arg => arg.trim())
		.filter(Boolean);

	return args.length > 0 ? args : undefined;
}

function normalizePositiveInteger(value: string): number | undefined {
	const trimmed = value.trim();
	if (!trimmed) {
		return undefined;
	}

	const parsed = Number(trimmed);
	if (!Number.isInteger(parsed) || parsed <= 0) {
		return undefined;
	}

	return parsed;
}

function stringifyArgs(args?: string[]): string {
	return (args ?? []).join('\n');
}

function getInitialFormData(initialData: MCPServerConfig | undefined): MCPServerFormData {
	const defaultPolicy = initialData?.defaultPolicy ?? getDefaultMCPServerPolicy();
	const appsPolicy = initialData?.appsPolicy ?? getDefaultMCPAppsPolicy();
	const existingSecretHeaderRefs = initialData?.streamableHttp?.secretHeaderRefs ?? {};
	const isAPIKeyMode = initialData?.streamableHttp?.authMode === MCPHTTPAuthMode.MCPHTTPAuthAPIKey;
	const apiKeyHeaderName =
		Object.keys(existingSecretHeaderRefs).find(key => key.toLowerCase() === 'authorization') ??
		Object.keys(existingSecretHeaderRefs)[0] ??
		'Authorization';
	const stdioSecretRows: SecretEnvRow[] = Object.entries(initialData?.stdio?.secretEnvRefs ?? {}).map(
		([envName, secretRef]) => ({
			rowID: makeRowID(),
			envName,
			originalEnvName: envName,
			slot: envName,
			existingSecretRef: secretRef,
			secretValue: '',
			deleteExisting: false,
		})
	);

	return {
		serverID: initialData?.id ?? '',
		displayName: initialData?.displayName ?? '',
		enabled: initialData?.enabled ?? true,
		transport: initialData?.transport ?? MCPTransportType.MCPTransportTypeStreamableHTTP,
		trustLevel: initialData?.trustLevel ?? MCPTrustLevel.MCPTrustLevelUntrusted,

		stdioCommand: initialData?.stdio?.command ?? '',
		stdioArgsText: stringifyArgs(initialData?.stdio?.args),
		stdioWorkingDir: initialData?.stdio?.workingDir ?? '',
		stdioEnvJSON: stringifyMCPJSON(initialData?.stdio?.env),
		stdioStartupTimeoutMS: initialData?.stdio?.startupTimeoutMS ? String(initialData.stdio.startupTimeoutMS) : '',
		stdioSecretRows,

		httpURL: initialData?.streamableHttp?.url ?? '',
		httpTimeoutMS: initialData?.streamableHttp?.timeoutMS ? String(initialData.streamableHttp.timeoutMS) : '',
		httpAuthMode: initialData?.streamableHttp?.authMode ?? MCPHTTPAuthMode.MCPHTTPAuthNone,
		httpClientCredentialRef: initialData?.streamableHttp?.clientCredentialRef ?? '',
		httpClientCredentialsSecret: '',
		httpDeleteClientCredentials: false,
		httpClientIDMetadataDocumentURL: initialData?.streamableHttp?.clientIDMetadataDocumentURL ?? '',

		apiKeyHeaderName,
		apiKeyOriginalHeaderName: isAPIKeyMode ? apiKeyHeaderName : '',
		apiKeyValuePrefix: getDefaultAPIKeyValuePrefix(apiKeyHeaderName),
		apiKeyValue: '',
		apiKeyExistingRef: isAPIKeyMode ? existingSecretHeaderRefs[apiKeyHeaderName] : undefined,
		apiKeyDeleteExisting: false,

		defaultApprovalRule: defaultPolicy.defaultApprovalRule,
		defaultExecutionMode: defaultPolicy.defaultExecutionMode,
		requireApprovalForUnknownRisk: defaultPolicy.requireApprovalForUnknownRisk,
		requireApprovalForWrite: defaultPolicy.requireApprovalForWrite,
		requireApprovalForDestructive: defaultPolicy.requireApprovalForDestructive,

		appsPolicyEnabled: appsPolicy.enabled,
		allowAppInitiatedToolCalls: appsPolicy.allowAppInitiatedToolCalls,
		requireApprovalForOpenLink: appsPolicy.requireApprovalForOpenLink,
		requireApprovalForContextUpdates: appsPolicy.requireApprovalForContextUpdates,

		toolPoliciesJSON: stringifyMCPJSON(initialData?.toolPolicies),
	};
}

function buildCopiedMCPServerFormData(source: MCPServerConfig, current: MCPServerFormData): MCPServerFormData {
	const copied = getInitialFormData(source);

	return {
		...copied,
		serverID: current.serverID,
		enabled: true,
		stdioSecretRows: copied.stdioSecretRows.map(row =>
			Object.assign({}, row, {
				rowID: makeRowID(),
				existingSecretRef: undefined,
				originalEnvName: undefined,
				secretValue: '',
				deleteExisting: false,
			})
		),

		httpClientCredentialRef: '',
		httpClientCredentialsSecret: '',
		httpDeleteClientCredentials: false,

		apiKeyOriginalHeaderName: '',
		apiKeyValue: '',
		apiKeyExistingRef: undefined,
		apiKeyDeleteExisting: false,
	};
}

function validateOAuthClientCredentials(raw: string, requireClientSecret: boolean): string | undefined {
	const value = raw.trim();
	if (!value) {
		return undefined;
	}

	let parsed: unknown;

	try {
		parsed = JSON.parse(value);
	} catch {
		return 'Client credentials must be valid JSON.';
	}

	if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
		return 'Client credentials must be a JSON object.';
	}

	const obj = parsed as Record<string, unknown>;
	const allowedKeys = new Set(['clientID', 'clientSecret']);

	for (const key of Object.keys(obj)) {
		if (!allowedKeys.has(key)) {
			return `Unsupported client credentials field "${key}". Allowed fields: clientID, clientSecret.`;
		}
	}
	if (typeof obj.clientID !== 'string' || obj.clientID.trim().length === 0) {
		return 'Client credentials must include a non-empty clientID string.';
	}

	if (obj.clientID.trim() !== obj.clientID) {
		return 'clientID must not have leading or trailing whitespace.';
	}

	if (obj.clientSecret !== undefined && typeof obj.clientSecret !== 'string') {
		return 'clientSecret must be a string when provided.';
	}

	if (obj.clientSecret !== undefined && obj.clientSecret.trim().length === 0) {
		return 'clientSecret must not be only whitespace.';
	}

	if (requireClientSecret && (typeof obj.clientSecret !== 'string' || obj.clientSecret.trim().length === 0)) {
		return 'Client credentials auth requires a non-empty clientSecret string.';
	}

	return undefined;
}

function validateClientIDMetadataURL(raw: string): string | undefined {
	const value = raw.trim();
	if (!value) {
		return undefined;
	}

	try {
		const url = new URL(value);
		if (url.protocol !== 'https:') {
			return 'Client ID metadata URL must use https.';
		}
		if (!url.host) {
			return 'Client ID metadata URL host is required.';
		}
		if (url.username || url.password) {
			return 'Client ID metadata URL must not include user info.';
		}
		if (!url.pathname || url.pathname === '/') {
			return 'Client ID metadata URL must include a path.';
		}
		if (url.hash) {
			return 'Client ID metadata URL must not include a fragment.';
		}
		return undefined;
	} catch {
		return 'Client ID metadata URL must be valid.';
	}
}

function AddEditMCPServerModalContent({
	onClose,
	onSubmit,
	initialData,
	existingServerIDs,
	prefillServers = [],
	mode,
}: AddEditMCPServerModalProps) {
	const effectiveMode: ModalMode = mode ?? (initialData ? 'edit' : 'add');
	const isEditMode = effectiveMode === 'edit';

	const [formData, setFormData] = useState<MCPServerFormData>(() => getInitialFormData(initialData));
	const [prefillMode, setPrefillMode] = useState(false);
	const [selectedPrefillKey, setSelectedPrefillKey] = useState<string | null>(null);
	const [errors, setErrors] = useState<ErrorState>({});
	const [submitError, setSubmitError] = useState('');
	const [deletedStdioSecretRows, setDeletedStdioSecretRows] = useState<SecretEnvRow[]>([]);
	const [isSubmitting, setIsSubmitting] = useState(false);

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const isUnmountingRef = useRef(false);

	const prefillSourceMap = useMemo<Record<string, MCPServerConfig>>(
		() => Object.fromEntries(prefillServers.map(server => [buildMCPServerPrefillKey(server), server] as const)),
		[prefillServers]
	);

	const prefillKeys = useMemo(() => Object.keys(prefillSourceMap), [prefillSourceMap]);

	const prefillDropdownItems = useMemo<Record<string, { isEnabled: boolean; displayName: string }>>(
		() =>
			Object.fromEntries(
				Object.entries(prefillSourceMap).map(([key, server]) => [
					key,
					{
						isEnabled: true,
						displayName: `${server.displayName || server.id} — ${server.bundleID} (${server.id})`,
					},
				])
			),
		[prefillSourceMap]
	);

	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) {
			return;
		}

		if (!dialog.open) {
			try {
				dialog.showModal();
			} catch {
				// Ignore showModal errors and keep rendering safely.
			}
		}

		return () => {
			isUnmountingRef.current = true;

			if (dialog.open) {
				dialog.close();
			}
		};
	}, []);

	const requestClose = () => {
		if (isSubmitting) {
			return;
		}

		const dialog = dialogRef.current;

		if (dialog?.open) {
			dialog.close();
			return;
		}

		onClose();
	};

	const handleDialogClose = () => {
		if (isUnmountingRef.current) {
			return;
		}
		onClose();
	};

	const validateForm = useCallback(
		(state: MCPServerFormData): ErrorState => {
			let nextErrors: ErrorState = {};
			const serverID = state.serverID.trim();
			const displayName = state.displayName.trim();

			if (!serverID) {
				nextErrors.serverID = 'Server ID is required.';
			} else {
				const err = validateSlug(serverID);
				if (err) {
					nextErrors.serverID = err;
				} else if (!isEditMode && existingServerIDs.includes(serverID)) {
					nextErrors.serverID = 'Server ID already exists.';
				} else {
					nextErrors = omitManyKeys(nextErrors, ['serverID']);
				}
			}

			if (!displayName) {
				nextErrors.displayName = 'Display name is required.';
			} else {
				nextErrors = omitManyKeys(nextErrors, ['displayName']);
			}

			if (state.transport === MCPTransportType.MCPTransportTypeStdio) {
				let parsedStdioEnv: Record<string, string> | undefined;
				if (!state.stdioCommand.trim()) {
					nextErrors.stdioCommand = 'Command is required for stdio servers.';
				}

				try {
					parsedStdioEnv = parseMCPStringRecordJSON(state.stdioEnvJSON, 'Environment');
					nextErrors = omitManyKeys(nextErrors, ['stdioEnvJSON']);
				} catch (error) {
					nextErrors.stdioEnvJSON = error instanceof Error ? error.message : 'Environment must be valid JSON.';
				}

				if (state.stdioStartupTimeoutMS.trim() && normalizePositiveInteger(state.stdioStartupTimeoutMS) === undefined) {
					nextErrors.stdioStartupTimeoutMS = 'Startup timeout must be a positive integer.';
				} else {
					nextErrors = omitManyKeys(nextErrors, ['stdioStartupTimeoutMS']);
				}
				const plainEnvNames = new Set(
					Object.keys(parsedStdioEnv ?? {})
						.map(key => key.trim().toLowerCase())
						.filter(Boolean)
				);
				const seenEnvNames = new Set<string>();
				for (const row of state.stdioSecretRows) {
					const envName = row.envName.trim();

					if (!envName) {
						nextErrors.stdioSecrets = 'Every secret env row needs an environment variable name.';
						break;
					}

					if (!ENV_NAME_RE.test(envName)) {
						nextErrors.stdioSecrets = 'Secret env names must match [A-Za-z_][A-Za-z0-9_]*.';
						break;
					}

					const normalizedEnvName = envName.toLowerCase();
					if (seenEnvNames.has(normalizedEnvName)) {
						nextErrors.stdioSecrets = 'Secret env names must be unique.';
						break;
					}
					if (plainEnvNames.has(envName.toLowerCase())) {
						nextErrors.stdioSecrets = `Secret env ${envName} is also present in plain Env JSON.`;
						break;
					}

					seenEnvNames.add(normalizedEnvName);

					if (
						row.existingSecretRef &&
						row.originalEnvName &&
						row.originalEnvName !== envName &&
						!row.secretValue.trim() &&
						!row.deleteExisting
					) {
						nextErrors.stdioSecrets = `Changing ${row.originalEnvName} to ${envName} requires a replacement secret value.`;
						break;
					}

					if (!row.existingSecretRef && !row.secretValue.trim()) {
						nextErrors.stdioSecrets = `Secret value is required for ${envName}.`;
						break;
					}
				}

				if (!nextErrors.stdioSecrets) {
					nextErrors = omitManyKeys(nextErrors, ['stdioSecrets']);
				}
			}

			if (state.transport === MCPTransportType.MCPTransportTypeStreamableHTTP) {
				const rawURL = state.httpURL.trim();

				if (!rawURL) {
					nextErrors.httpURL = 'URL is required for Streamable HTTP servers.';
				} else {
					try {
						const url = new URL(rawURL);
						if (url.protocol !== 'http:' && url.protocol !== 'https:') {
							nextErrors.httpURL = 'URL must use http or https.';
						} else if (url.protocol === 'http:' && !isLoopbackHTTPHost(url.hostname)) {
							nextErrors.httpURL =
								'HTTP URLs are only allowed for localhost or loopback hosts. Use HTTPS for remote servers.';
						} else {
							nextErrors = omitManyKeys(nextErrors, ['httpURL']);
						}
					} catch {
						nextErrors.httpURL = 'URL must be valid.';
					}
				}

				if (state.httpTimeoutMS.trim() && normalizePositiveInteger(state.httpTimeoutMS) === undefined) {
					nextErrors.httpTimeoutMS = 'Timeout must be a positive integer.';
				} else {
					nextErrors = omitManyKeys(nextErrors, ['httpTimeoutMS']);
				}

				const credentialAuthMode =
					state.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth ||
					state.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthClientCredentials;

				if (state.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthAPIKey) {
					const headerName = state.apiKeyHeaderName.trim();
					const originalHeaderName = state.apiKeyOriginalHeaderName.trim();
					const headerNameChanged =
						Boolean(state.apiKeyExistingRef) &&
						Boolean(originalHeaderName) &&
						!sameHTTPHeaderName(headerName, originalHeaderName);
					const hasExisting = Boolean(state.apiKeyExistingRef) && !state.apiKeyDeleteExisting && !headerNameChanged;
					const hasNew = Boolean(state.apiKeyValue.trim());
					const fullHeaderValue = `${state.apiKeyValuePrefix}${state.apiKeyValue}`;

					if (!headerName) {
						nextErrors.httpAPIKey = 'API key header name is required.';
					} else if (!/^[A-Za-z0-9!#$%&'*+.^_`|~-]+$/.test(headerName)) {
						nextErrors.httpAPIKey = 'API key header name contains invalid characters.';
					} else if (headerNameChanged && !hasNew) {
						nextErrors.httpAPIKey = `Changing the header name from ${originalHeaderName} requires entering the API key again.`;
					} else if (!hasExisting && !hasNew) {
						nextErrors.httpAPIKey = 'API key value is required.';
					} else if (hasNew && hasInvalidHTTPHeaderValueChars(fullHeaderValue)) {
						nextErrors.httpAPIKey = 'API key header value must not contain CR, LF, or NUL.';
					} else {
						nextErrors = omitManyKeys(nextErrors, ['httpAPIKey']);
					}
				} else {
					nextErrors = omitManyKeys(nextErrors, ['httpAPIKey']);
				}

				const hasExistingClientCredentials =
					credentialAuthMode && Boolean(state.httpClientCredentialRef.trim()) && !state.httpDeleteClientCredentials;

				const hasNewClientCredentials = Boolean(state.httpClientCredentialsSecret.trim());

				if (
					state.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthClientCredentials &&
					!hasExistingClientCredentials &&
					!hasNewClientCredentials
				) {
					nextErrors.httpClientCredentials = 'Client credentials auth requires a credentials secret.';
				} else {
					const secretError = validateOAuthClientCredentials(
						state.httpClientCredentialsSecret,
						state.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthClientCredentials
					);
					if (secretError) {
						nextErrors.httpClientCredentials = secretError;
					} else {
						nextErrors = omitManyKeys(nextErrors, ['httpClientCredentials']);
					}
				}
				if (state.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth) {
					const metadataURLError = validateClientIDMetadataURL(state.httpClientIDMetadataDocumentURL);
					if (metadataURLError) {
						nextErrors.httpClientIDMetadataDocumentURL = metadataURLError;
					} else {
						nextErrors = omitManyKeys(nextErrors, ['httpClientIDMetadataDocumentURL']);
					}
				} else {
					nextErrors = omitManyKeys(nextErrors, ['httpClientIDMetadataDocumentURL']);
				}
			}

			try {
				parseMCPObjectJSON(state.toolPoliciesJSON, 'Tool policies');

				nextErrors = omitManyKeys(nextErrors, ['toolPoliciesJSON']);
			} catch (error) {
				nextErrors.toolPoliciesJSON = error instanceof Error ? error.message : 'Tool policies must be valid JSON.';
			}

			return nextErrors;
		},
		[existingServerIDs, isEditMode]
	);

	const setFormDataAndValidate = (next: MCPServerFormData) => {
		if (isSubmitting) {
			return;
		}

		setFormData(next);
		setErrors(validateForm(next));
	};

	const applyPrefill = (key: string) => {
		const source = prefillSourceMap[key];
		if (!source) {
			return;
		}

		const next = buildCopiedMCPServerFormData(source, formData);

		setDeletedStdioSecretRows([]);
		setFormDataAndValidate(next);
		setSubmitError('');
		setSelectedPrefillKey(key);
		setPrefillMode(false);
	};

	const handleInput = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
		if (isSubmitting) {
			return;
		}

		const target = e.target as HTMLInputElement;
		const { name, value, type, checked } = target;
		const newVal = type === 'checkbox' ? checked : value;
		const next = { ...formData, [name]: newVal } as MCPServerFormData;

		setFormDataAndValidate(next);
	};

	const addSecretRow = () => {
		const next: MCPServerFormData = {
			...formData,
			stdioSecretRows: [
				...formData.stdioSecretRows,
				{
					rowID: makeRowID(),
					envName: '',
					slot: '',
					secretValue: '',
					deleteExisting: false,
				},
			],
		};

		setFormDataAndValidate(next);
	};

	const updateSecretRow = (rowID: string, patch: Partial<SecretEnvRow>) => {
		const next: MCPServerFormData = {
			...formData,
			stdioSecretRows: formData.stdioSecretRows.map(row => {
				if (row.rowID !== rowID) {
					return row;
				}

				const merged = { ...row, ...patch };

				if (patch.envName !== undefined) {
					merged.slot = patch.envName.trim();
				}

				return merged;
			}),
		};

		setFormDataAndValidate(next);
	};

	const removeSecretRow = (rowID: string) => {
		const rowToRemove = formData.stdioSecretRows.find(row => row.rowID === rowID);
		if (rowToRemove?.existingSecretRef) {
			const originalSlot = (rowToRemove.originalEnvName ?? rowToRemove.slot ?? rowToRemove.envName).trim();
			setDeletedStdioSecretRows(prev => [
				...prev.filter(row => row.existingSecretRef !== rowToRemove.existingSecretRef),
				{
					...rowToRemove,
					envName: originalSlot,
					slot: originalSlot,
					secretValue: '',
					deleteExisting: true,
				},
			]);
		}

		const next: MCPServerFormData = {
			...formData,
			stdioSecretRows: formData.stdioSecretRows.filter(row => row.rowID !== rowID),
		};

		setFormDataAndValidate(next);
	};

	const buildServerInput = (): MCPServerUpsertInput => {
		const defaultPolicy = {
			defaultApprovalRule: formData.defaultApprovalRule,
			defaultExecutionMode: formData.defaultExecutionMode,
			requireApprovalForUnknownRisk: formData.requireApprovalForUnknownRisk,
			requireApprovalForWrite: formData.requireApprovalForWrite,
			requireApprovalForDestructive: formData.requireApprovalForDestructive,
		};

		const appsPolicy = {
			enabled: formData.appsPolicyEnabled,
			allowAppInitiatedToolCalls: formData.allowAppInitiatedToolCalls,
			requireApprovalForOpenLink: formData.requireApprovalForOpenLink,
			requireApprovalForContextUpdates: formData.requireApprovalForContextUpdates,
		};

		const toolPolicies = parseMCPObjectJSON(formData.toolPoliciesJSON, 'Tool policies') as
			Record<string, MCPToolPolicyOverride> | undefined;

		const payloadBase = {
			displayName: formData.displayName.trim(),
			enabled: formData.enabled,
			transport: formData.transport,
			trustLevel: formData.trustLevel,
			defaultPolicy,
			toolPolicies,
			appsPolicy,
		};

		if (formData.transport === MCPTransportType.MCPTransportTypeStdio) {
			const secretEnvRefs: Record<string, string> = {};

			for (const row of formData.stdioSecretRows) {
				const envName = row.envName.trim();

				const envNameUnchanged = !row.originalEnvName || row.originalEnvName === envName;
				if (row.existingSecretRef && !row.deleteExisting && !row.secretValue.trim() && envNameUnchanged) {
					secretEnvRefs[envName] = row.existingSecretRef;
				}
			}
			const deletedSecretOps = deletedStdioSecretRows.map(row => {
				const deleteSlot = (row.originalEnvName ?? row.slot ?? row.envName).trim();
				return {
					envName: deleteSlot,
					slot: deleteSlot,
					deleteSlot,
					existingSecretRef: row.existingSecretRef,
					secretValue: '',
					deleteExisting: true,
				};
			});

			const activeSecretOps = formData.stdioSecretRows.map(row => {
				const envName = row.envName.trim();
				const originalSlot = (row.originalEnvName ?? row.slot).trim();
				const renamedExistingSecret =
					Boolean(row.existingSecretRef) && Boolean(row.originalEnvName) && row.originalEnvName !== envName;

				return {
					envName,
					slot: row.slot.trim(),
					deleteSlot: originalSlot || row.slot.trim(),
					existingSecretRef: row.existingSecretRef,
					secretValue: row.secretValue,
					deleteExisting: row.deleteExisting || renamedExistingSecret,
				};
			});

			return {
				serverID: formData.serverID.trim(),
				payload: {
					...payloadBase,
					stdio: {
						command: formData.stdioCommand.trim(),
						args: splitArgsText(formData.stdioArgsText),
						workingDir: formData.stdioWorkingDir.trim() || undefined,
						env: parseMCPStringRecordJSON(formData.stdioEnvJSON, 'Environment'),
						secretEnvRefs: Object.keys(secretEnvRefs).length > 0 ? secretEnvRefs : undefined,
						startupTimeoutMS: normalizePositiveInteger(formData.stdioStartupTimeoutMS),
					},
				},
				stdioSecretEnv: [...deletedSecretOps, ...activeSecretOps],
			};
		}

		const credentialAuthMode =
			formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth ||
			formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthClientCredentials;
		const isAPIKey = formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthAPIKey;

		const existingClientCredentialRef = formData.httpClientCredentialRef.trim();
		const shouldDeleteExistingClientCredentials =
			Boolean(existingClientCredentialRef) && (formData.httpDeleteClientCredentials || !credentialAuthMode);
		const clientCredentialRef =
			credentialAuthMode && existingClientCredentialRef && !shouldDeleteExistingClientCredentials
				? existingClientCredentialRef
				: undefined;

		const apiKeyHeaderName = formData.apiKeyHeaderName.trim() || 'Authorization';
		const originalAPIKeyHeaderName = formData.apiKeyOriginalHeaderName.trim();

		const existingAPIKeyRef = formData.apiKeyExistingRef?.trim() ?? '';
		const apiKeyHeaderChanged =
			Boolean(existingAPIKeyRef) &&
			Boolean(originalAPIKeyHeaderName) &&
			!sameHTTPHeaderName(apiKeyHeaderName, originalAPIKeyHeaderName);
		const shouldDeleteAPIKey =
			Boolean(existingAPIKeyRef) && (formData.apiKeyDeleteExisting || !isAPIKey || apiKeyHeaderChanged);
		const apiKeyRef = isAPIKey && existingAPIKeyRef && !shouldDeleteAPIKey ? existingAPIKeyRef : undefined;
		const apiKeyFullValue = formData.apiKeyValue ? `${formData.apiKeyValuePrefix}${formData.apiKeyValue}` : '';

		const streamableHttp = {
			url: formData.httpURL.trim(),
			timeoutMS: normalizePositiveInteger(formData.httpTimeoutMS),
			authMode: formData.httpAuthMode,
			secretHeaderRefs: apiKeyRef ? { [apiKeyHeaderName]: apiKeyRef } : undefined,
			clientCredentialRef,
			clientIDMetadataDocumentURL:
				formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth
					? formData.httpClientIDMetadataDocumentURL.trim() || undefined
					: undefined,
		};

		const finalPayload: PutMCPServerPayload = {
			...payloadBase,
			streamableHttp,
		};

		const needsClientCredentialsStaging =
			formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthClientCredentials &&
			!clientCredentialRef &&
			Boolean(formData.httpClientCredentialsSecret.trim());
		const needsAPIKeyStaging = isAPIKey && !apiKeyRef && Boolean(apiKeyFullValue);

		let initialPayload: PutMCPServerPayload | undefined;
		if (needsClientCredentialsStaging) {
			initialPayload = {
				...payloadBase,
				streamableHttp: {
					...streamableHttp,
					// Stage as no-auth so creating the server does not enter the
					// interactive OAuth flow before the client-credentials secret exists.
					authMode: MCPHTTPAuthMode.MCPHTTPAuthNone,
					clientCredentialRef: undefined,
					clientIDMetadataDocumentURL: undefined,
					secretHeaderRefs: undefined,
				},
			};
		} else if (needsAPIKeyStaging) {
			initialPayload = {
				...payloadBase,
				streamableHttp: {
					...streamableHttp,
					authMode: MCPHTTPAuthMode.MCPHTTPAuthNone,
					clientCredentialRef: undefined,
					clientIDMetadataDocumentURL: undefined,
					secretHeaderRefs: undefined,
				},
			};
		}
		return {
			serverID: formData.serverID.trim(),
			initialPayload,
			payload: finalPayload,
			stdioSecretEnv: [],
			oauthClientCredentials:
				credentialAuthMode || existingClientCredentialRef || shouldDeleteExistingClientCredentials
					? {
							slot: MCP_OAUTH_CLIENT_CREDENTIALS_SLOT,
							existingSecretRef: existingClientCredentialRef || undefined,
							secretValue: formData.httpClientCredentialsSecret,
							deleteExisting: shouldDeleteExistingClientCredentials,
						}
					: undefined,
			httpHeaderSecret:
				isAPIKey || (Boolean(existingAPIKeyRef) && shouldDeleteAPIKey)
					? {
							headerName: apiKeyHeaderName,
							slot: apiKeyHeaderName,
							deleteSlot: originalAPIKeyHeaderName || apiKeyHeaderName,
							existingSecretRef: existingAPIKeyRef || undefined,
							secretValue: apiKeyFullValue,
							deleteExisting: shouldDeleteAPIKey,
						}
					: undefined,
		};
	};

	const isAllValid = useMemo(() => Object.keys(validateForm(formData)).length === 0, [formData, validateForm]);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async e => {
		e.preventDefault();
		e.stopPropagation();

		if (isSubmitting) {
			return;
		}

		setSubmitError('');

		const nextErrors = validateForm(formData);
		setErrors(nextErrors);

		if (Object.keys(nextErrors).length > 0) {
			return;
		}

		setIsSubmitting(true);
		try {
			await onSubmit(buildServerInput());
			if (!isUnmountingRef.current) {
				const dialog = dialogRef.current;
				if (dialog?.open) {
					dialog.close();
				} else {
					onClose();
				}
			}
		} catch (error) {
			if (!isUnmountingRef.current) {
				const msg = error instanceof Error ? error.message : 'Failed to save MCP server.';
				setSubmitError(msg);
			}
		} finally {
			if (!isUnmountingRef.current) {
				setIsSubmitting(false);
			}
		}
	};

	const headerTitle = isEditMode ? 'Edit MCP Server' : 'Add MCP Server';

	return (
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={e => {
				e.preventDefault();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-4xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between">
						<h3 className="text-lg font-bold">{headerTitle}</h3>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={requestClose}
							aria-label="Close"
							disabled={isSubmitting}
						>
							<FiX size={12} />
						</button>
					</div>

					<form noValidate onSubmit={handleSubmit} className="space-y-4" aria-busy={isSubmitting}>
						{submitError && (
							<div className="alert alert-error rounded-2xl text-sm">
								<div className="flex items-center gap-2">
									<FiAlertCircle size={14} />
									<span>{submitError}</span>
								</div>
							</div>
						)}

						{effectiveMode === 'add' && (
							<div className="grid grid-cols-12 items-start gap-2">
								<label className="label col-span-3">
									<span className="text-sm">Prefill from Existing</span>
								</label>

								<div className="col-span-9 space-y-2">
									<div className="flex items-center gap-2">
										{!prefillMode && (
											<button
												type="button"
												className="btn btn-sm btn-ghost flex items-center rounded-xl"
												onClick={() => {
													setPrefillMode(true);
												}}
												disabled={prefillKeys.length === 0}
												title={prefillKeys.length === 0 ? 'No existing MCP servers are available to copy.' : undefined}
											>
												<FiUpload size={14} />
												<span className="ml-1">Copy Existing Server</span>
											</button>
										)}

										{prefillMode && (
											<>
												<Dropdown<string>
													dropdownItems={prefillDropdownItems}
													orderedKeys={prefillKeys}
													selectedKey={selectedPrefillKey ?? ''}
													onChange={applyPrefill}
													disabled={prefillKeys.length === 0}
													filterDisabled={false}
													title="Select MCP server to copy"
													getDisplayName={key => prefillDropdownItems[key]?.displayName ?? 'Select MCP server to copy'}
												/>
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													onClick={() => {
														setPrefillMode(false);
														setSelectedPrefillKey(null);
													}}
													title="Cancel prefill"
												>
													<FiX size={12} />
												</button>
											</>
										)}
									</div>
									<p className="text-base-content/70 text-xs">
										Secrets are never copied. Enter replacement secret values where needed before saving.
									</p>
								</div>
							</div>
						)}

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="text-sm">Server ID*</span>
								<span className="tooltip tooltip-right" data-tip="Stable lower-case server identifier.">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="serverID"
									value={formData.serverID}
									onChange={handleInput}
									readOnly={isEditMode}
									className={`input w-full rounded-xl ${errors.serverID ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									autoFocus={!isEditMode}
									aria-invalid={Boolean(errors.serverID)}
								/>
								{errors.serverID && (
									<div className="label">
										<span className="text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.serverID}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="text-sm">Display Name*</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="displayName"
									value={formData.displayName}
									onChange={handleInput}
									className={`input w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.displayName)}
								/>
								{errors.displayName && (
									<div className="label">
										<span className="text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.displayName}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3 cursor-pointer">
								<span className="text-sm">Enabled</span>
							</label>
							<div className="col-span-9">
								<input
									type="checkbox"
									name="enabled"
									checked={formData.enabled}
									onChange={handleInput}
									className="toggle toggle-accent"
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="text-sm">Transport*</span>
							</label>
							<div className="col-span-9">
								<Dropdown
									dropdownItems={TRANSPORT_DROPDOWN_ITEMS}
									selectedKey={formData.transport}
									onChange={transport => {
										setFormDataAndValidate({ ...formData, transport });
									}}
									getDisplayName={getMCPTransportLabel}
									title="Transport"
									inlineMenu={true}
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="text-sm">Trust</span>
							</label>
							<div className="col-span-9">
								<Dropdown
									dropdownItems={TRUST_DROPDOWN_ITEMS}
									selectedKey={formData.trustLevel}
									onChange={trustLevel => {
										setFormDataAndValidate({ ...formData, trustLevel });
									}}
									getDisplayName={getMCPTrustLevelLabel}
									title="Trust level"
									inlineMenu={true}
								/>
							</div>
						</div>

						{formData.transport === MCPTransportType.MCPTransportTypeStdio && (
							<>
								<div className="divider">Stdio</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="text-sm">Command*</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="stdioCommand"
											value={formData.stdioCommand}
											onChange={handleInput}
											className={`input w-full rounded-xl ${errors.stdioCommand ? 'input-error' : ''}`}
											spellCheck="false"
											autoComplete="off"
										/>
										{errors.stdioCommand && (
											<div className="label">
												<span className="text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.stdioCommand}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-start gap-2">
									<label className="label col-span-3">
										<span className="text-sm">Args</span>
										<span className="tooltip tooltip-right" data-tip="One argument per line.">
											<FiHelpCircle size={12} />
										</span>
									</label>
									<div className="col-span-9">
										<textarea
											name="stdioArgsText"
											value={formData.stdioArgsText}
											onChange={handleInput}
											className="textarea h-28 w-full rounded-xl"
											spellCheck="false"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="text-sm">Working Dir</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="stdioWorkingDir"
											value={formData.stdioWorkingDir}
											onChange={handleInput}
											className="input w-full rounded-xl"
											spellCheck="false"
											autoComplete="off"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-start gap-2">
									<label className="label col-span-3">
										<span className="text-sm">Env JSON</span>
										<span className="tooltip tooltip-right" data-tip="Plain non-secret env object.">
											<FiHelpCircle size={12} />
										</span>
									</label>
									<div className="col-span-9">
										<textarea
											name="stdioEnvJSON"
											value={formData.stdioEnvJSON}
											onChange={handleInput}
											className={`textarea h-28 w-full rounded-xl ${errors.stdioEnvJSON ? 'textarea-error' : ''}`}
											spellCheck="false"
											placeholder='{"NODE_ENV":"production"}'
										/>
										{errors.stdioEnvJSON && (
											<div className="label">
												<span className="text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.stdioEnvJSON}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="text-sm">Startup Timeout MS</span>
									</label>
									<div className="col-span-9">
										<input
											type="number"
											name="stdioStartupTimeoutMS"
											value={formData.stdioStartupTimeoutMS}
											onChange={handleInput}
											className={`input w-full rounded-xl ${errors.stdioStartupTimeoutMS ? 'input-error' : ''}`}
											min={1}
										/>
										{errors.stdioStartupTimeoutMS && (
											<div className="label">
												<span className="text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.stdioStartupTimeoutMS}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="border-base-content/10 rounded-2xl border p-3">
									<div className="mb-3 flex items-center justify-between">
										<div>
											<div className="text-sm font-semibold">Secret Env</div>
											<p className="text-base-content/70 text-xs">
												Secret values are stored separately. Existing secret values are never shown.
											</p>
										</div>
										<button type="button" className="btn btn-sm btn-ghost rounded-xl" onClick={addSecretRow}>
											<FiPlus size={14} />
											<span className="ml-1">Add Secret</span>
										</button>
									</div>

									{errors.stdioSecrets && (
										<div className="text-error mb-3 flex items-center gap-1 text-sm">
											<FiAlertCircle size={12} /> {errors.stdioSecrets}
										</div>
									)}

									<div className="space-y-3">
										{formData.stdioSecretRows.map(row => (
											<div key={row.rowID} className="bg-base-100 rounded-2xl p-3">
												<div className="grid grid-cols-12 gap-2">
													<div className="col-span-12 md:col-span-3">
														<label className="label py-1">
															<span className="text-sm">Env Name</span>
														</label>
														<input
															value={row.envName}
															onChange={e => {
																updateSecretRow(row.rowID, { envName: e.target.value });
															}}
															className="input w-full rounded-xl"
															spellCheck="false"
														/>
													</div>

													<div className="col-span-12 md:col-span-3">
														<label className="label py-1">
															<span className="text-sm">Secret Slot</span>
														</label>
														<input
															value={row.envName.trim() || '(matches env name)'}
															readOnly
															className="input bg-base-200 w-full rounded-xl"
															spellCheck="false"
															title="Stdio secret slots are derived from the environment variable name."
														/>
													</div>

													<div className="col-span-12 md:col-span-5">
														<label className="label py-1">
															<span className="text-sm">
																{row.existingSecretRef ? 'Replace Secret Value' : 'Secret Value'}
															</span>
														</label>
														<input
															type="password"
															value={row.secretValue}
															onChange={e => {
																updateSecretRow(row.rowID, { secretValue: e.target.value });
															}}
															className="input w-full rounded-xl"
															autoComplete="new-password"
														/>
														{row.existingSecretRef && (
															<div className="label">
																<span className="text-base-content/70">
																	Existing secret configured. Leave blank to keep it.
																</span>
															</div>
														)}
													</div>

													<div className="col-span-12 flex items-end justify-end md:col-span-1">
														<button
															type="button"
															className="btn btn-ghost btn-sm rounded-xl"
															onClick={() => {
																removeSecretRow(row.rowID);
															}}
															title="Remove row"
														>
															<FiTrash2 size={14} />
														</button>
													</div>

													{row.existingSecretRef && (
														<div className="col-span-12">
															<label className="label cursor-pointer justify-start gap-3">
																<input
																	type="checkbox"
																	className="checkbox checkbox-sm"
																	checked={row.deleteExisting}
																	onChange={e => {
																		updateSecretRow(row.rowID, { deleteExisting: e.target.checked });
																	}}
																/>
																<span className="text-sm">Delete existing stored secret</span>
															</label>
														</div>
													)}
												</div>
											</div>
										))}

										{formData.stdioSecretRows.length === 0 && (
											<div className="text-base-content/70 text-center text-sm">No secret env variables.</div>
										)}
									</div>
								</div>
							</>
						)}

						{formData.transport === MCPTransportType.MCPTransportTypeStreamableHTTP && (
							<>
								<div className="divider">Streamable HTTP</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="text-sm">URL*</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="httpURL"
											value={formData.httpURL}
											onChange={handleInput}
											className={`input w-full rounded-xl ${errors.httpURL ? 'input-error' : ''}`}
											spellCheck="false"
											autoComplete="off"
											placeholder="https://example.com/mcp"
										/>
										{errors.httpURL && (
											<div className="label">
												<span className="text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpURL}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="text-sm">Timeout MS</span>
									</label>
									<div className="col-span-9">
										<input
											type="number"
											name="httpTimeoutMS"
											value={formData.httpTimeoutMS}
											onChange={handleInput}
											className={`input w-full rounded-xl ${errors.httpTimeoutMS ? 'input-error' : ''}`}
											min={1}
										/>
										{errors.httpTimeoutMS && (
											<div className="label">
												<span className="text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpTimeoutMS}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="text-sm">Auth Mode</span>
									</label>
									<div className="col-span-9">
										<Dropdown
											dropdownItems={AUTH_MODE_DROPDOWN_ITEMS}
											selectedKey={formData.httpAuthMode}
											onChange={httpAuthMode => {
												setFormDataAndValidate({ ...formData, httpAuthMode });
											}}
											getDisplayName={getMCPHTTPAuthModeLabel}
											title="Auth mode"
											inlineMenu={true}
										/>
									</div>
								</div>

								{formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthAPIKey && (
									<>
										<div className="grid grid-cols-12 items-center gap-2">
											<label className="label col-span-3">
												<span className="text-sm">Header Name</span>
											</label>
											<div className="col-span-9">
												<input
													type="text"
													name="apiKeyHeaderName"
													value={formData.apiKeyHeaderName}
													onChange={e => {
														const nextHeaderName = e.target.value;
														const previousDefaultPrefix = getDefaultAPIKeyValuePrefix(formData.apiKeyHeaderName);
														const shouldUpdatePrefix = formData.apiKeyValuePrefix === previousDefaultPrefix;

														setFormDataAndValidate({
															...formData,
															apiKeyHeaderName: nextHeaderName,
															apiKeyValuePrefix: shouldUpdatePrefix
																? getDefaultAPIKeyValuePrefix(nextHeaderName)
																: formData.apiKeyValuePrefix,
														});
													}}
													className="input w-full rounded-xl"
													spellCheck="false"
													autoComplete="off"
													placeholder="Authorization"
												/>
											</div>
										</div>
										<div className="grid grid-cols-12 items-center gap-2">
											<label className="label col-span-3">
												<span className="text-sm">Value Prefix</span>
												<span className="tooltip tooltip-right" data-tip='Prepended to the key, e.g. "Bearer ".'>
													<FiHelpCircle size={12} />
												</span>
											</label>
											<div className="col-span-9">
												<input
													type="text"
													name="apiKeyValuePrefix"
													value={formData.apiKeyValuePrefix}
													onChange={handleInput}
													className="input w-full rounded-xl"
													spellCheck="false"
													autoComplete="off"
													placeholder="Bearer "
												/>
											</div>
										</div>
										<div className="grid grid-cols-12 items-start gap-2">
											<label className="label col-span-3">
												<span className="text-sm">{formData.apiKeyExistingRef ? 'Replace API Key' : 'API Key'}</span>
											</label>
											<div className="col-span-9">
												<input
													type="password"
													name="apiKeyValue"
													value={formData.apiKeyValue}
													onChange={handleInput}
													className={`input w-full rounded-xl ${errors.httpAPIKey ? 'input-error' : ''}`}
													autoComplete="new-password"
												/>
												{formData.apiKeyExistingRef && (
													<div className="label">
														<span className="text-base-content/70">
															Existing key configured. Leave blank to keep it.
														</span>
													</div>
												)}
												{errors.httpAPIKey && (
													<div className="label">
														<span className="text-error flex items-center gap-1">
															<FiAlertCircle size={12} /> {errors.httpAPIKey}
														</span>
													</div>
												)}
											</div>
										</div>
										{formData.apiKeyExistingRef && (
											<div className="grid grid-cols-12 items-center gap-2">
												<div className="col-span-3" />
												<label className="label col-span-9 cursor-pointer justify-start gap-3">
													<input
														type="checkbox"
														name="apiKeyDeleteExisting"
														checked={formData.apiKeyDeleteExisting}
														onChange={handleInput}
														className="checkbox checkbox-sm"
													/>
													<span className="text-sm">Delete existing API key secret</span>
												</label>
											</div>
										)}
									</>
								)}

								{(formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth ||
									formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthClientCredentials) && (
									<>
										<div className="grid grid-cols-12 items-start gap-2">
											<label className="label col-span-3">
												<span className="text-sm">Client Credentials Secret</span>
												<span
													className="tooltip tooltip-right"
													data-tip='JSON: {"clientID":"...","clientSecret":"..."}'
												>
													<FiHelpCircle size={12} />
												</span>
											</label>
											<div className="col-span-9">
												<textarea
													name="httpClientCredentialsSecret"
													value={formData.httpClientCredentialsSecret}
													onChange={handleInput}
													className={`textarea h-28 w-full rounded-xl ${
														errors.httpClientCredentials ? 'textarea-error' : ''
													}`}
													spellCheck="false"
													placeholder='{"clientID":"...","clientSecret":"..."}'
												/>
												{formData.httpClientCredentialRef && (
													<div className="label">
														<span className="text-base-content/70">
															Existing credential secret configured. Leave blank to keep it.
														</span>
													</div>
												)}
												{errors.httpClientCredentials && (
													<div className="label">
														<span className="text-error flex items-center gap-1">
															<FiAlertCircle size={12} /> {errors.httpClientCredentials}
														</span>
													</div>
												)}
											</div>
										</div>

										{formData.httpClientCredentialRef && (
											<div className="grid grid-cols-12 items-center gap-2">
												<div className="col-span-3" />
												<label className="label col-span-9 cursor-pointer justify-start gap-3">
													<input
														type="checkbox"
														name="httpDeleteClientCredentials"
														checked={formData.httpDeleteClientCredentials}
														onChange={handleInput}
														className="checkbox checkbox-sm"
													/>
													<span className="text-sm">Delete existing client credentials secret</span>
												</label>
											</div>
										)}

										{formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth && (
											<div className="grid grid-cols-12 items-center gap-2">
												<label className="label col-span-3">
													<span className="text-sm">Client ID Metadata URL</span>
												</label>
												<div className="col-span-9">
													<input
														type="text"
														name="httpClientIDMetadataDocumentURL"
														value={formData.httpClientIDMetadataDocumentURL}
														onChange={handleInput}
														className={`input w-full rounded-xl ${
															errors.httpClientIDMetadataDocumentURL ? 'input-error' : ''
														}`}
														spellCheck="false"
														autoComplete="off"
														placeholder="https://client.example.com/flexigpt-mcp-client.json"
													/>
													{errors.httpClientIDMetadataDocumentURL && (
														<div className="label">
															<span className="text-error flex items-center gap-1">
																<FiAlertCircle size={12} /> {errors.httpClientIDMetadataDocumentURL}
															</span>
														</div>
													)}
												</div>
											</div>
										)}
									</>
								)}
							</>
						)}

						<div className="divider">Default Policy</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="text-sm">Execution</span>
							</label>
							<div className="col-span-9">
								<Dropdown
									dropdownItems={EXECUTION_MODE_DROPDOWN_ITEMS}
									selectedKey={formData.defaultExecutionMode}
									onChange={defaultExecutionMode => {
										setFormDataAndValidate({ ...formData, defaultExecutionMode });
									}}
									getDisplayName={getMCPExecutionModeLabel}
									title="Default execution"
									inlineMenu={true}
								/>
							</div>
						</div>
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="text-sm">Approval</span>
							</label>
							<div className="col-span-9">
								<Dropdown
									dropdownItems={APPROVAL_RULE_DROPDOWN_ITEMS}
									selectedKey={formData.defaultApprovalRule}
									onChange={defaultApprovalRule => {
										setFormDataAndValidate({ ...formData, defaultApprovalRule });
									}}
									getDisplayName={getMCPApprovalRuleLabel}
									title="Default approval"
									inlineMenu={true}
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<div className="col-span-3" />
							<div className="col-span-9 grid grid-cols-1 gap-2 md:grid-cols-3">
								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="requireApprovalForUnknownRisk"
										checked={formData.requireApprovalForUnknownRisk}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="text-sm">Unknown risk</span>
								</label>

								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="requireApprovalForWrite"
										checked={formData.requireApprovalForWrite}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="text-sm">Write</span>
								</label>

								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="requireApprovalForDestructive"
										checked={formData.requireApprovalForDestructive}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="text-sm">Destructive</span>
								</label>
							</div>
						</div>

						<div className="divider">Apps Policy</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<div className="col-span-3" />
							<div className="col-span-9 grid grid-cols-1 gap-2 md:grid-cols-2">
								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="appsPolicyEnabled"
										checked={formData.appsPolicyEnabled}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="text-sm">Advertise and render MCP Apps</span>
								</label>

								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="allowAppInitiatedToolCalls"
										checked={formData.allowAppInitiatedToolCalls}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="text-sm">Allow app-initiated tool calls</span>
								</label>

								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="requireApprovalForOpenLink"
										checked={formData.requireApprovalForOpenLink}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="text-sm">Approve open link</span>
								</label>

								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="requireApprovalForContextUpdates"
										checked={formData.requireApprovalForContextUpdates}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="text-sm">Approve context updates</span>
								</label>
							</div>
						</div>

						<div className="divider">Tool Policy Overrides</div>

						<div className="grid grid-cols-12 items-start gap-2">
							<label className="label col-span-3">
								<span className="text-sm">Tool Policies JSON</span>
								<span
									className="tooltip tooltip-right"
									data-tip="Optional map keyed by tool name. Values match MCPToolPolicyOverride."
								>
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<textarea
									name="toolPoliciesJSON"
									value={formData.toolPoliciesJSON}
									onChange={handleInput}
									className={`textarea h-36 w-full rounded-xl ${errors.toolPoliciesJSON ? 'textarea-error' : ''}`}
									spellCheck="false"
									placeholder='{"tool_name":{"toolName":"tool_name","approvalRule":"ask","executionMode":"manual"}}'
								/>
								{errors.toolPoliciesJSON && (
									<div className="label">
										<span className="text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.toolPoliciesJSON}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="modal-action">
							<button
								type="button"
								className="btn bg-base-300 rounded-xl"
								onClick={requestClose}
								disabled={isSubmitting}
							>
								Cancel
							</button>
							<button type="submit" className="btn btn-primary rounded-xl" disabled={!isAllValid || isSubmitting}>
								{isSubmitting ? 'Saving...' : 'Save'}
							</button>
						</div>
					</form>
				</div>
			</div>
			<ModalBackdrop enabled={false} />
		</dialog>
	);
}

export function AddEditMCPServerModal(props: AddEditMCPServerModalProps) {
	if (!props.isOpen) {
		return null;
	}
	if (typeof document === 'undefined' || !document.body) {
		return null;
	}

	const remountKey = props.initialData
		? `${props.mode ?? 'auto'}:${props.initialData.bundleID}:${props.initialData.id}:${props.initialData.modifiedAt}`
		: `${props.mode ?? 'auto'}:new`;

	return createPortal(<AddEditMCPServerModalContent key={remountKey} {...props} />, document.body);
}
