import type { ChangeEvent, SubmitEventHandler } from 'react';
import { useCallback, useMemo, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiPlus, FiTrash2, FiUpload, FiX } from 'react-icons/fi';

import type { MCPServerConfig, MCPToolPolicyOverride, PutMCPServerPayload } from '@/spec/mcp';
import { MCPApprovalRule, MCPExecutionMode, MCPHTTPAuthMode, MCPTransportType, MCPTrustLevel } from '@/spec/mcp';

import { validateHTTPURLSecurity } from '@/lib/http_input_utils';
import { omitManyKeys } from '@/lib/obj_utils';
import { validateSlug } from '@/lib/text_utils';

import { useDialogController } from '@/hooks/use_dialog_controller';

import type { DropdownItem } from '@/components/dropdown';
import { Dropdown } from '@/components/dropdown';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

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

	const { dialogRef, requestClose, handleClose, handleCancel, unmountingRef } = useDialogController({
		onClose,
		blockCancel: true,
		isBusy: isSubmitting,
	});

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
					const urlError = validateHTTPURLSecurity(rawURL, 'MCP server URL');
					if (urlError) {
						nextErrors.httpURL = urlError;
					} else {
						nextErrors = omitManyKeys(nextErrors, ['httpURL']);
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
			// Setup definitions are server metadata, not editable form fields.
			// Preserve them so ordinary edits do not erase setup requirements.
			...(initialData?.setup ? { setup: initialData.setup } : {}),
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

		const existingSecretHeaderRefs = initialData?.streamableHttp?.secretHeaderRefs ?? {};
		const existingAPIKeyRef = formData.apiKeyExistingRef?.trim() ?? '';
		const apiKeyHeaderChanged =
			Boolean(existingAPIKeyRef) &&
			Boolean(originalAPIKeyHeaderName) &&
			!sameHTTPHeaderName(apiKeyHeaderName, originalAPIKeyHeaderName);
		const shouldDeleteAPIKey =
			Boolean(existingAPIKeyRef) && (formData.apiKeyDeleteExisting || !isAPIKey || apiKeyHeaderChanged);
		const apiKeyRef = isAPIKey && existingAPIKeyRef && !shouldDeleteAPIKey ? existingAPIKeyRef : undefined;
		const apiKeyFullValue = formData.apiKeyValue ? `${formData.apiKeyValuePrefix}${formData.apiKeyValue}` : '';
		const shouldRemoveManagedAPIKeyRef = (headerName: string) =>
			Boolean(existingAPIKeyRef) &&
			Boolean(originalAPIKeyHeaderName) &&
			sameHTTPHeaderName(headerName, originalAPIKeyHeaderName) &&
			(shouldDeleteAPIKey || apiKeyHeaderChanged || !isAPIKey);
		let secretHeaderRefs: Record<string, string> = Object.fromEntries(
			Object.entries(existingSecretHeaderRefs).filter(([headerName]) => !shouldRemoveManagedAPIKeyRef(headerName))
		);

		if (apiKeyRef) {
			for (const headerName of Object.keys(secretHeaderRefs)) {
				if (sameHTTPHeaderName(headerName, apiKeyHeaderName)) {
					secretHeaderRefs = omitManyKeys(secretHeaderRefs, [headerName]);
				}
			}

			secretHeaderRefs[apiKeyHeaderName] = apiKeyRef;
		}

		const streamableHttp = {
			url: formData.httpURL.trim(),
			timeoutMS: normalizePositiveInteger(formData.httpTimeoutMS),
			authMode: formData.httpAuthMode,
			// General headers are not editable in this modal. Preserve them on
			// PUT rather than silently deleting existing non-secret headers.
			...(initialData?.streamableHttp?.headers ? { headers: { ...initialData.streamableHttp.headers } } : {}),
			secretHeaderRefs: Object.keys(secretHeaderRefs).length > 0 ? secretHeaderRefs : undefined,
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
			if (!unmountingRef.current) {
				requestClose(true);
			}
		} catch (error) {
			if (!unmountingRef.current) {
				const msg = error instanceof Error ? error.message : 'Failed to save MCP server.';
				setSubmitError(msg);
			}
		} finally {
			if (!unmountingRef.current) {
				setIsSubmitting(false);
			}
		}
	};

	const headerTitle = isEditMode ? 'Edit MCP Server' : 'Add MCP Server';
	const headerDescription = isEditMode
		? 'Update transport, credentials, runtime configuration, and policy without exposing stored secrets.'
		: 'Create a custom MCP server with explicit transport, trust, credential, and policy settings.';

	return (
		<dialog ref={dialogRef} className="modal" onClose={handleClose} onCancel={handleCancel}>
			<div className="modal-box bg-base-200 flex max-h-[85vh] w-[calc(100%-1rem)] max-w-4xl flex-col overflow-hidden rounded-2xl p-0">
				<ModalHeader
					title={headerTitle}
					description={headerDescription}
					onClose={() => {
						requestClose();
					}}
					closeDisabled={isSubmitting}
				/>

				<form noValidate onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col" aria-busy={isSubmitting}>
					<div className="app-scrollbar-thin min-h-0 flex-1 space-y-4 overflow-y-auto p-4 sm:p-6">
						{submitError && (
							<div className="alert alert-error rounded-2xl text-sm">
								<div className="flex items-center gap-2">
									<FiAlertCircle size={14} />
									<span>{submitError}</span>
								</div>
							</div>
						)}

						{effectiveMode === 'add' && (
							<ModalSection
								title="Copy an existing server"
								description="Copy non-secret configuration from an existing MCP server. Stored secrets are never copied."
							>
								<ModalField label="Prefill from Existing">
									<div className="flex items-center gap-2">
										{!prefillMode && (
											<button
												type="button"
												className="btn btn-sm btn-ghost flex items-center rounded-xl"
												onClick={() => {
													setPrefillMode(true);
												}}
												disabled={prefillKeys.length === 0}
												aria-disabled={prefillKeys.length === 0}
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
													inlineMenu={true}
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
								</ModalField>
							</ModalSection>
						)}

						<ModalSection
							title="Identity and availability"
							description="The server ID is stable after creation. Enable the server only when its configuration is ready."
						>
							<ModalField
								label="Server ID"
								htmlFor="mcp-server-id"
								required
								hint="Stable lower-case server identifier."
								error={errors.serverID}
							>
								<input
									id="mcp-server-id"
									type="text"
									name="serverID"
									value={formData.serverID}
									onChange={handleInput}
									readOnly={isEditMode}
									className={`input w-full rounded-xl ${errors.serverID ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									autoFocus={!isEditMode}
									disabled={isSubmitting}
									aria-invalid={Boolean(errors.serverID)}
								/>
							</ModalField>

							<ModalField label="Display Name" htmlFor="mcp-server-display-name" required error={errors.displayName}>
								<input
									id="mcp-server-display-name"
									type="text"
									name="displayName"
									value={formData.displayName}
									onChange={handleInput}
									className={`input w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.displayName)}
									disabled={isSubmitting}
								/>
							</ModalField>

							<ModalField label="Enabled" htmlFor="mcp-server-enabled">
								<input
									id="mcp-server-enabled"
									type="checkbox"
									name="enabled"
									checked={formData.enabled}
									onChange={handleInput}
									className="toggle toggle-accent"
									disabled={isSubmitting}
								/>
							</ModalField>
						</ModalSection>

						<ModalSection
							title="Transport and trust"
							description="Choose how FlexiGPT connects to this server and whether the server should be treated as trusted."
						>
							<ModalField label="Transport" required>
								<Dropdown
									dropdownItems={TRANSPORT_DROPDOWN_ITEMS}
									selectedKey={formData.transport}
									onChange={transport => {
										setFormDataAndValidate({ ...formData, transport });
									}}
									getDisplayName={getMCPTransportLabel}
									disabled={isSubmitting}
									title="Transport"
									inlineMenu={true}
								/>
							</ModalField>

							<ModalField label="Trust">
								<Dropdown
									dropdownItems={TRUST_DROPDOWN_ITEMS}
									selectedKey={formData.trustLevel}
									onChange={trustLevel => {
										setFormDataAndValidate({ ...formData, trustLevel });
									}}
									getDisplayName={getMCPTrustLevelLabel}
									disabled={isSubmitting}
									title="Trust level"
									inlineMenu={true}
								/>
							</ModalField>
						</ModalSection>

						{formData.transport === MCPTransportType.MCPTransportTypeStdio && (
							<ModalSection
								title="Stdio runtime"
								description="Configure the executable process. Plain environment values are stored in configuration; secret values use the secure server secret store."
							>
								<ModalField label="Command" htmlFor="mcp-stdio-command" required error={errors.stdioCommand}>
									<input
										id="mcp-stdio-command"
										type="text"
										name="stdioCommand"
										value={formData.stdioCommand}
										onChange={handleInput}
										className={`input w-full rounded-xl ${errors.stdioCommand ? 'input-error' : ''}`}
										spellCheck="false"
										autoComplete="off"
										disabled={isSubmitting}
									/>
								</ModalField>

								<ModalField
									label="Arguments"
									htmlFor="mcp-stdio-args"
									hint="Use one process argument per line."
									align="start"
								>
									<textarea
										id="mcp-stdio-args"
										name="stdioArgsText"
										value={formData.stdioArgsText}
										onChange={handleInput}
										className="textarea h-28 w-full rounded-xl"
										spellCheck="false"
										disabled={isSubmitting}
									/>
								</ModalField>

								<ModalField label="Working Directory" htmlFor="mcp-stdio-working-directory">
									<input
										id="mcp-stdio-working-directory"
										type="text"
										name="stdioWorkingDir"
										value={formData.stdioWorkingDir}
										onChange={handleInput}
										className="input w-full rounded-xl"
										spellCheck="false"
										autoComplete="off"
										disabled={isSubmitting}
									/>
								</ModalField>

								<ModalField
									label="Environment JSON"
									htmlFor="mcp-stdio-environment"
									hint="Plain non-secret environment object. Use Secret Environment Variables below for credentials."
									error={errors.stdioEnvJSON}
									align="start"
								>
									<textarea
										id="mcp-stdio-environment"
										name="stdioEnvJSON"
										value={formData.stdioEnvJSON}
										onChange={handleInput}
										className={`textarea h-28 w-full rounded-xl ${errors.stdioEnvJSON ? 'textarea-error' : ''}`}
										spellCheck="false"
										placeholder='{"NODE_ENV":"production"}'
										disabled={isSubmitting}
									/>
								</ModalField>

								<ModalField
									label="Startup Timeout"
									htmlFor="mcp-stdio-startup-timeout"
									hint="Optional positive timeout in milliseconds."
									error={errors.stdioStartupTimeoutMS}
								>
									<input
										id="mcp-stdio-startup-timeout"
										type="number"
										name="stdioStartupTimeoutMS"
										value={formData.stdioStartupTimeoutMS}
										onChange={handleInput}
										className={`input w-full rounded-xl ${errors.stdioStartupTimeoutMS ? 'input-error' : ''}`}
										min={1}
										disabled={isSubmitting}
									/>
								</ModalField>

								<ModalSection
									title="Secret environment variables"
									description="Secret values are stored separately. Existing secret values are never shown."
									className="bg-base-100/40"
									actions={
										<button type="button" className="btn btn-sm btn-ghost rounded-xl" onClick={addSecretRow}>
											<FiPlus size={14} />
											<span className="ml-1">Add Secret</span>
										</button>
									}
								>
									{errors.stdioSecrets && (
										<div className="alert alert-error rounded-2xl text-sm">
											<FiAlertCircle size={12} /> {errors.stdioSecrets}
										</div>
									)}

									<div className="space-y-3">
										{formData.stdioSecretRows.map(row => (
											<div key={row.rowID} className="border-base-content/10 bg-base-100 rounded-2xl border p-3">
												<div className="mb-3 flex items-center justify-between gap-3">
													<div className="min-w-0">
														<div className="truncate text-sm font-semibold">{row.envName || 'New secret variable'}</div>
														<div className="text-base-content/70 text-xs">
															Secret values are never displayed after saving.
														</div>
													</div>
													<button
														type="button"
														className="btn btn-ghost btn-sm rounded-xl"
														onClick={() => {
															removeSecretRow(row.rowID);
														}}
														title="Remove secret environment variable"
														disabled={isSubmitting}
													>
														<FiTrash2 size={14} />
													</button>
												</div>

												<div className="space-y-3">
													<ModalField label="Environment Variable" htmlFor={`mcp-secret-env-name-${row.rowID}`}>
														<input
															id={`mcp-secret-env-name-${row.rowID}`}
															value={row.envName}
															onChange={e => {
																updateSecretRow(row.rowID, { envName: e.target.value });
															}}
															className="input w-full rounded-xl"
															spellCheck="false"
															disabled={isSubmitting}
														/>
													</ModalField>

													<ModalField label="Secret Slot" htmlFor={`mcp-secret-slot-${row.rowID}`}>
														<input
															id={`mcp-secret-slot-${row.rowID}`}
															value={row.envName.trim() || '(matches env name)'}
															readOnly
															className="input bg-base-200 w-full rounded-xl"
															spellCheck="false"
															title="Stdio secret slots are derived from the environment variable name."
														/>
													</ModalField>

													<ModalField label={row.existingSecretRef ? 'Replace Secret Value' : 'Secret Value'}>
														<input
															type="password"
															value={row.secretValue}
															onChange={e => {
																updateSecretRow(row.rowID, { secretValue: e.target.value });
															}}
															className="input w-full rounded-xl"
															autoComplete="new-password"
															disabled={isSubmitting}
														/>
														{row.existingSecretRef && (
															<p className="text-base-content/70 mt-1 text-xs">
																Existing secret configured. Leave blank to keep it.
															</p>
														)}
													</ModalField>

													{row.existingSecretRef && (
														<ModalField
															label="Delete Existing Stored Secret"
															htmlFor={`mcp-secret-delete-${row.rowID}`}
														>
															<input
																id={`mcp-secret-delete-${row.rowID}`}
																type="checkbox"
																className="checkbox checkbox-sm"
																checked={row.deleteExisting}
																onChange={e => {
																	updateSecretRow(row.rowID, { deleteExisting: e.target.checked });
																}}
																disabled={isSubmitting}
															/>
														</ModalField>
													)}
												</div>
											</div>
										))}

										{formData.stdioSecretRows.length === 0 && (
											<div className="text-base-content/70 text-center text-sm">No secret env variables.</div>
										)}
									</div>
								</ModalSection>
							</ModalSection>
						)}

						{formData.transport === MCPTransportType.MCPTransportTypeStreamableHTTP && (
							<ModalSection
								title="Streamable HTTP runtime"
								description="Remote endpoints must use HTTPS. Plain HTTP is allowed only for localhost and loopback development endpoints."
							>
								<ModalField label="URL" htmlFor="mcp-http-url" required error={errors.httpURL}>
									<input
										id="mcp-http-url"
										type="text"
										name="httpURL"
										value={formData.httpURL}
										onChange={handleInput}
										className={`input w-full rounded-xl ${errors.httpURL ? 'input-error' : ''}`}
										spellCheck="false"
										autoComplete="off"
										placeholder="https://example.com/mcp"
										disabled={isSubmitting}
									/>
								</ModalField>

								<ModalField
									label="Timeout"
									htmlFor="mcp-http-timeout"
									hint="Optional positive timeout in milliseconds."
									error={errors.httpTimeoutMS}
								>
									<input
										id="mcp-http-timeout"
										type="number"
										name="httpTimeoutMS"
										value={formData.httpTimeoutMS}
										onChange={handleInput}
										className={`input w-full rounded-xl ${errors.httpTimeoutMS ? 'input-error' : ''}`}
										min={1}
										disabled={isSubmitting}
									/>
								</ModalField>

								<ModalField label="Auth Mode">
									<Dropdown
										dropdownItems={AUTH_MODE_DROPDOWN_ITEMS}
										selectedKey={formData.httpAuthMode}
										onChange={httpAuthMode => {
											setFormDataAndValidate({ ...formData, httpAuthMode });
										}}
										getDisplayName={getMCPHTTPAuthModeLabel}
										disabled={isSubmitting}
										title="Auth mode"
										inlineMenu={true}
									/>
								</ModalField>

								{formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthAPIKey && (
									<ModalSection
										title="API key credentials"
										description="The key value is write-only and stored in the MCP server secret store."
										className="bg-base-100/40"
									>
										<ModalField label="Header Name" htmlFor="mcp-api-key-header-name" error={errors.httpAPIKey}>
											<input
												id="mcp-api-key-header-name"
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
												disabled={isSubmitting}
											/>
										</ModalField>

										<ModalField
											label="Value Prefix"
											htmlFor="mcp-api-key-prefix"
											hint='Prepended to the key, for example "Bearer ".'
										>
											<input
												id="mcp-api-key-prefix"
												type="text"
												name="apiKeyValuePrefix"
												value={formData.apiKeyValuePrefix}
												onChange={handleInput}
												className="input w-full rounded-xl"
												spellCheck="false"
												autoComplete="off"
												placeholder="Bearer "
												disabled={isSubmitting}
											/>
										</ModalField>

										<ModalField
											label={formData.apiKeyExistingRef ? 'Replace API Key' : 'API Key'}
											htmlFor="mcp-api-key-value"
											error={errors.httpAPIKey}
										>
											<input
												id="mcp-api-key-value"
												type="password"
												name="apiKeyValue"
												value={formData.apiKeyValue}
												onChange={handleInput}
												className={`input w-full rounded-xl ${errors.httpAPIKey ? 'input-error' : ''}`}
												autoComplete="new-password"
												disabled={isSubmitting}
											/>
											{formData.apiKeyExistingRef && (
												<p className="text-base-content/70 mt-1 text-xs">
													Existing key configured. Leave blank to keep it.
												</p>
											)}
										</ModalField>

										{formData.apiKeyExistingRef && (
											<ModalField label="Delete Existing API Key" htmlFor="mcp-delete-api-key">
												<input
													id="mcp-delete-api-key"
													type="checkbox"
													name="apiKeyDeleteExisting"
													checked={formData.apiKeyDeleteExisting}
													onChange={handleInput}
													className="checkbox checkbox-sm"
													disabled={isSubmitting}
												/>
											</ModalField>
										)}
									</ModalSection>
								)}

								{(formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth ||
									formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthClientCredentials) && (
									<ModalSection
										title="OAuth client credentials"
										description="Credentials are stored as a write-only server secret. OAuth authorization state is managed separately."
										className="bg-base-100/40"
									>
										<ModalField
											label="Client Credentials Secret"
											htmlFor="mcp-client-credentials"
											hint='JSON: {"clientID":"...","clientSecret":"..."}'
											error={errors.httpClientCredentials}
											align="start"
										>
											<textarea
												id="mcp-client-credentials"
												name="httpClientCredentialsSecret"
												value={formData.httpClientCredentialsSecret}
												onChange={handleInput}
												className={`textarea h-28 w-full rounded-xl ${
													errors.httpClientCredentials ? 'textarea-error' : ''
												}`}
												spellCheck="false"
												placeholder='{"clientID":"...","clientSecret":"..."}'
												disabled={isSubmitting}
											/>
											{formData.httpClientCredentialRef && (
												<p className="text-base-content/70 mt-1 text-xs">
													Existing credential secret configured. Leave blank to keep it.
												</p>
											)}
										</ModalField>

										{formData.httpClientCredentialRef && (
											<ModalField label="Delete Existing Client Credentials" htmlFor="mcp-delete-client-credentials">
												<input
													id="mcp-delete-client-credentials"
													type="checkbox"
													name="httpDeleteClientCredentials"
													checked={formData.httpDeleteClientCredentials}
													onChange={handleInput}
													className="checkbox checkbox-sm"
													disabled={isSubmitting}
												/>
											</ModalField>
										)}

										{formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth && (
											<ModalField
												label="Client ID Metadata URL"
												htmlFor="mcp-client-id-metadata-url"
												error={errors.httpClientIDMetadataDocumentURL}
											>
												<input
													id="mcp-client-id-metadata-url"
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
													disabled={isSubmitting}
												/>
											</ModalField>
										)}
									</ModalSection>
								)}
							</ModalSection>
						)}

						<ModalSection
							title="Default tool policy"
							description="These defaults apply when a discovered tool does not have a more specific override."
						>
							<ModalField label="Execution">
								<Dropdown
									dropdownItems={EXECUTION_MODE_DROPDOWN_ITEMS}
									selectedKey={formData.defaultExecutionMode}
									onChange={defaultExecutionMode => {
										setFormDataAndValidate({ ...formData, defaultExecutionMode });
									}}
									getDisplayName={getMCPExecutionModeLabel}
									disabled={isSubmitting}
									title="Default execution"
									inlineMenu={true}
								/>
							</ModalField>

							<ModalField label="Approval">
								<Dropdown
									dropdownItems={APPROVAL_RULE_DROPDOWN_ITEMS}
									selectedKey={formData.defaultApprovalRule}
									onChange={defaultApprovalRule => {
										setFormDataAndValidate({ ...formData, defaultApprovalRule });
									}}
									getDisplayName={getMCPApprovalRuleLabel}
									disabled={isSubmitting}
									title="Default approval"
									inlineMenu={true}
								/>
							</ModalField>

							<ModalField
								label="Require approval for"
								hint="These checks add approval even when the default approval rule allows a call."
								align="start"
							>
								<div className="grid grid-cols-1 gap-3 md:grid-cols-3">
									<label className="label cursor-pointer justify-start gap-3">
										<input
											type="checkbox"
											name="requireApprovalForUnknownRisk"
											checked={formData.requireApprovalForUnknownRisk}
											onChange={handleInput}
											className="checkbox checkbox-sm"
											disabled={isSubmitting}
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
											disabled={isSubmitting}
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
											disabled={isSubmitting}
										/>
										<span className="text-sm">Destructive</span>
									</label>
								</div>
							</ModalField>
						</ModalSection>

						<ModalSection
							title="MCP Apps policy"
							description="Controls whether MCP Apps are exposed and what app-originated actions require approval."
						>
							<ModalField label="App behavior" align="start">
								<div className="grid grid-cols-1 gap-3 md:grid-cols-2">
									<label className="label cursor-pointer justify-start gap-3">
										<input
											type="checkbox"
											name="appsPolicyEnabled"
											checked={formData.appsPolicyEnabled}
											onChange={handleInput}
											className="checkbox checkbox-sm"
											disabled={isSubmitting}
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
											disabled={isSubmitting}
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
											disabled={isSubmitting}
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
											disabled={isSubmitting}
										/>
										<span className="text-sm">Approve context updates</span>
									</label>
								</div>
							</ModalField>
						</ModalSection>

						<ModalSection
							title="Per-tool policy overrides"
							description="Optional overrides keyed by discovered tool name. These take precedence over the default policy."
						>
							<ModalField
								label="Tool Policies JSON"
								htmlFor="mcp-tool-policy-overrides"
								hint="Optional object keyed by tool name. Each value follows MCPToolPolicyOverride."
								error={errors.toolPoliciesJSON}
								align="start"
							>
								<textarea
									id="mcp-tool-policy-overrides"
									name="toolPoliciesJSON"
									value={formData.toolPoliciesJSON}
									onChange={handleInput}
									className={`textarea h-36 w-full rounded-xl ${errors.toolPoliciesJSON ? 'textarea-error' : ''}`}
									spellCheck="false"
									placeholder='{"tool_name":{"toolName":"tool_name","approvalRule":"ask","executionMode":"manual"}}'
									disabled={isSubmitting}
								/>
							</ModalField>
						</ModalSection>

						{isEditMode && initialData ? (
							<ModalSection title="Metadata">
								<ManagementInfoGrid>
									<ManagementInfoRow label="Bundle ID" mono>
										{initialData.bundleID}
									</ManagementInfoRow>
									<ManagementInfoRow label="Server ID" mono>
										{initialData.id}
									</ManagementInfoRow>
									<ManagementInfoRow label="Built-in">{initialData.isBuiltIn ? 'Yes' : 'No'}</ManagementInfoRow>
									<ManagementInfoRow label="Created">{initialData.createdAt}</ManagementInfoRow>
									<ManagementInfoRow label="Modified">{initialData.modifiedAt}</ManagementInfoRow>
								</ManagementInfoGrid>
							</ModalSection>
						) : null}
					</div>

					<ModalActions>
						<button
							type="button"
							className="btn bg-base-300 rounded-xl"
							onClick={() => {
								requestClose();
							}}
							disabled={isSubmitting}
						>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={!isAllValid || isSubmitting}>
							{isSubmitting ? 'Saving...' : 'Save'}
						</button>
					</ModalActions>
				</form>
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
