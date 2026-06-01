import { type ChangeEvent, type SubmitEventHandler, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiPlus, FiTrash2, FiX } from 'react-icons/fi';

import {
	MCPApprovalRule,
	MCPExecutionMode,
	MCPHTTPAuthMode,
	MCPServerAvailability,
	type MCPServerConfig,
	type MCPToolPolicyOverride,
	MCPTransportType,
	MCPTrustLevel,
	type PutMCPServerPayload,
} from '@/spec/mcp';

import { omitManyKeys } from '@/lib/obj_utils';
import { validateSlug } from '@/lib/text_utils';

import { ModalBackdrop } from '@/components/modal_backdrop';

import {
	getDefaultMCPAppsPolicy,
	getDefaultMCPServerPolicy,
	getMCPApprovalRuleLabel,
	getMCPAvailabilityLabel,
	getMCPExecutionModeLabel,
	getMCPHTTPAuthModeLabel,
	getMCPTransportLabel,
	getMCPTrustLevelLabel,
	MCP_OAUTH_CLIENT_CREDENTIALS_SLOT,
	type MCPServerUpsertInput,
	parseMCPObjectJSON,
	parseMCPStringRecordJSON,
	stringifyMCPJSON,
} from '@/mcpservers/lib/mcp_server_utils';

type ModalMode = 'add' | 'edit';

interface AddEditMCPServerModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (serverData: MCPServerUpsertInput) => Promise<void>;
	initialData?: MCPServerConfig;
	existingServerIDs: string[];
	mode?: ModalMode;
}

type ErrorState = {
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
	policies?: string;
	toolPoliciesJSON?: string;
};

type SecretEnvRow = {
	rowID: string;
	envName: string;
	slot: string;
	existingSecretRef?: string;
	secretValue: string;
	deleteExisting: boolean;
};

type MCPServerFormData = {
	serverID: string;
	displayName: string;
	enabled: boolean;
	transport: MCPTransportType;

	availability: MCPServerAvailability;
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
};

const ENV_NAME_RE = /^[A-Za-z_][A-Za-z0-9_]*$/;
const SECRET_SLOT_RE = /^[A-Za-z0-9_.:-]+$/;

function makeRowID(): string {
	return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
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
	if (!trimmed) return undefined;

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
	const stdioSecretRows: SecretEnvRow[] = Object.entries(initialData?.stdio?.secretEnvRefs ?? {}).map(
		([envName, secretRef]) => ({
			rowID: makeRowID(),
			envName,
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

		availability: initialData?.availability ?? MCPServerAvailability.MCPServerAvailabilityManual,
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

function validateOAuthClientCredentials(raw: string, requireClientSecret: boolean): string | undefined {
	const value = raw.trim();
	if (!value) return undefined;

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

	if (requireClientSecret && (typeof obj.clientSecret !== 'string' || obj.clientSecret.trim().length === 0)) {
		return 'Client credentials auth requires a non-empty clientSecret string.';
	}

	if (obj.clientSecret !== undefined && typeof obj.clientSecret !== 'string') {
		return 'clientSecret must be a string when provided.';
	}

	return undefined;
}

function validateClientIDMetadataURL(raw: string): string | undefined {
	const value = raw.trim();
	if (!value) return undefined;

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
	mode,
}: AddEditMCPServerModalProps) {
	const effectiveMode: ModalMode = mode ?? (initialData ? 'edit' : 'add');
	const isEditMode = effectiveMode === 'edit';

	const [formData, setFormData] = useState<MCPServerFormData>(() => getInitialFormData(initialData));
	const [errors, setErrors] = useState<ErrorState>({});
	const [submitError, setSubmitError] = useState('');

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const isUnmountingRef = useRef(false);

	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) return;

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
		const dialog = dialogRef.current;

		if (dialog?.open) {
			dialog.close();
			return;
		}

		onClose();
	};

	const handleDialogClose = () => {
		if (isUnmountingRef.current) return;
		onClose();
	};

	const validateForm = (state: MCPServerFormData): ErrorState => {
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
				nextErrors.serverID = 'Server ID already exists in this bundle.';
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
			if (!state.stdioCommand.trim()) {
				nextErrors.stdioCommand = 'Command is required for stdio servers.';
			}

			try {
				parseMCPStringRecordJSON(state.stdioEnvJSON, 'Environment');
				nextErrors = omitManyKeys(nextErrors, ['stdioEnvJSON']);
			} catch (error) {
				nextErrors.stdioEnvJSON = error instanceof Error ? error.message : 'Environment must be valid JSON.';
			}

			if (state.stdioStartupTimeoutMS.trim() && normalizePositiveInteger(state.stdioStartupTimeoutMS) === undefined) {
				nextErrors.stdioStartupTimeoutMS = 'Startup timeout must be a positive integer.';
			} else {
				nextErrors = omitManyKeys(nextErrors, ['stdioStartupTimeoutMS']);
			}

			const seenEnvNames = new Set<string>();
			for (const row of state.stdioSecretRows) {
				const envName = row.envName.trim();
				const slot = row.slot.trim();

				if (!envName) {
					nextErrors.stdioSecrets = 'Every secret env row needs an environment variable name.';
					break;
				}

				if (!ENV_NAME_RE.test(envName)) {
					nextErrors.stdioSecrets = 'Secret env names must match [A-Za-z_][A-Za-z0-9_]*.';
					break;
				}

				if (seenEnvNames.has(envName)) {
					nextErrors.stdioSecrets = 'Secret env names must be unique.';
					break;
				}

				seenEnvNames.add(envName);

				if (!slot) {
					nextErrors.stdioSecrets = 'Every secret env row needs a slot.';
					break;
				}

				if (!SECRET_SLOT_RE.test(slot)) {
					nextErrors.stdioSecrets = 'Secret slots may only contain letters, numbers, underscore, dash, dot, and colon.';
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

			const hasExistingClientCredentials =
				state.httpAuthMode !== MCPHTTPAuthMode.MCPHTTPAuthNone &&
				Boolean(state.httpClientCredentialRef.trim()) &&
				!state.httpDeleteClientCredentials;

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
			parseMCPObjectJSON<Record<string, MCPToolPolicyOverride>>(state.toolPoliciesJSON, 'Tool policies');
			nextErrors = omitManyKeys(nextErrors, ['toolPoliciesJSON']);
		} catch (error) {
			nextErrors.toolPoliciesJSON = error instanceof Error ? error.message : 'Tool policies must be valid JSON.';
		}

		return nextErrors;
	};

	const setFormDataAndValidate = (next: MCPServerFormData) => {
		setFormData(next);
		setErrors(validateForm(next));
	};

	const handleInput = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
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
				if (row.rowID !== rowID) return row;

				const merged = { ...row, ...patch };

				if (patch.envName !== undefined && (!row.slot || row.slot === row.envName)) {
					merged.slot = patch.envName.trim();
				}

				return merged;
			}),
		};

		setFormDataAndValidate(next);
	};

	const removeSecretRow = (rowID: string) => {
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

		const toolPolicies = parseMCPObjectJSON<Record<string, MCPToolPolicyOverride>>(
			formData.toolPoliciesJSON,
			'Tool policies'
		);

		const payloadBase = {
			displayName: formData.displayName.trim(),
			enabled: formData.enabled,
			transport: formData.transport,
			availability: formData.availability,
			trustLevel: formData.trustLevel,
			defaultPolicy,
			toolPolicies,
			appsPolicy,
		};

		if (formData.transport === MCPTransportType.MCPTransportTypeStdio) {
			const secretEnvRefs: Record<string, string> = {};

			for (const row of formData.stdioSecretRows) {
				const envName = row.envName.trim();

				if (row.existingSecretRef && !row.deleteExisting) {
					secretEnvRefs[envName] = row.existingSecretRef;
				}
			}

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
				stdioSecretEnv: formData.stdioSecretRows.map(row => ({
					envName: row.envName.trim(),
					slot: row.slot.trim(),
					existingSecretRef: row.existingSecretRef,
					secretValue: row.secretValue,
					deleteExisting: row.deleteExisting,
				})),
			};
		}

		const existingClientCredentialRef = formData.httpClientCredentialRef.trim();
		const shouldDeleteExistingClientCredentials =
			Boolean(existingClientCredentialRef) &&
			(formData.httpDeleteClientCredentials || formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthNone);
		const clientCredentialRef =
			formData.httpAuthMode !== MCPHTTPAuthMode.MCPHTTPAuthNone &&
			existingClientCredentialRef &&
			!shouldDeleteExistingClientCredentials
				? formData.httpClientCredentialRef.trim()
				: undefined;
		const streamableHttp = {
			url: formData.httpURL.trim(),
			timeoutMS: normalizePositiveInteger(formData.httpTimeoutMS),
			authMode: formData.httpAuthMode,
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

		const initialPayload: PutMCPServerPayload | undefined = needsClientCredentialsStaging
			? {
					...payloadBase,
					streamableHttp: {
						...streamableHttp,
						authMode: MCPHTTPAuthMode.MCPHTTPAuthOAuth,
						clientCredentialRef: undefined,
						clientIDMetadataDocumentURL: undefined,
					},
				}
			: undefined;

		return {
			serverID: formData.serverID.trim(),
			initialPayload,
			payload: finalPayload,
			stdioSecretEnv: [],
			oauthClientCredentials:
				formData.httpAuthMode !== MCPHTTPAuthMode.MCPHTTPAuthNone ||
				existingClientCredentialRef ||
				shouldDeleteExistingClientCredentials
					? {
							slot: MCP_OAUTH_CLIENT_CREDENTIALS_SLOT,
							existingSecretRef: existingClientCredentialRef || undefined,
							secretValue: formData.httpClientCredentialsSecret,
							deleteExisting: shouldDeleteExistingClientCredentials,
						}
					: undefined,
		};
	};

	const isAllValid = useMemo(
		() => Object.keys(validateForm(formData)).length === 0,
		// validateForm captures stable modal props for one mount.
		// eslint-disable-next-line react-hooks/exhaustive-deps
		[formData]
	);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();
		e.stopPropagation();

		setSubmitError('');

		const nextErrors = validateForm(formData);
		setErrors(nextErrors);

		if (Object.keys(nextErrors).length > 0) return;

		void onSubmit(buildServerInput())
			.then(() => {
				requestClose();
			})
			.catch((error: unknown) => {
				const msg = error instanceof Error ? error.message : 'Failed to save MCP server.';
				setSubmitError(msg);
			});
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
						>
							<FiX size={12} />
						</button>
					</div>

					<form noValidate onSubmit={handleSubmit} className="space-y-4">
						{submitError && (
							<div className="alert alert-error rounded-2xl text-sm">
								<div className="flex items-center gap-2">
									<FiAlertCircle size={14} />
									<span>{submitError}</span>
								</div>
							</div>
						)}

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Server ID*</span>
								<span className="label-text-alt tooltip tooltip-right" data-tip="Stable lower-case server identifier.">
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
									className={`input input-bordered w-full rounded-xl ${errors.serverID ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									autoFocus={!isEditMode}
									aria-invalid={Boolean(errors.serverID)}
								/>
								{errors.serverID && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.serverID}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Display Name*</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="displayName"
									value={formData.displayName}
									onChange={handleInput}
									className={`input input-bordered w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.displayName)}
								/>
								{errors.displayName && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.displayName}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3 cursor-pointer">
								<span className="label-text text-sm">Enabled</span>
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
								<span className="label-text text-sm">Transport*</span>
							</label>
							<div className="col-span-9">
								<select
									name="transport"
									value={formData.transport}
									onChange={handleInput}
									className="select select-bordered w-full rounded-xl"
								>
									{Object.values(MCPTransportType).map(transport => (
										<option key={transport} value={transport}>
											{getMCPTransportLabel(transport)}
										</option>
									))}
								</select>
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Availability</span>
							</label>
							<div className="col-span-4">
								<select
									name="availability"
									value={formData.availability}
									onChange={handleInput}
									className="select select-bordered w-full rounded-xl"
								>
									{Object.values(MCPServerAvailability).map(availability => (
										<option key={availability} value={availability}>
											{getMCPAvailabilityLabel(availability)}
										</option>
									))}
								</select>
							</div>

							<label className="label col-span-2">
								<span className="label-text text-sm">Trust</span>
							</label>
							<div className="col-span-3">
								<select
									name="trustLevel"
									value={formData.trustLevel}
									onChange={handleInput}
									className="select select-bordered w-full rounded-xl"
								>
									{Object.values(MCPTrustLevel).map(trustLevel => (
										<option key={trustLevel} value={trustLevel}>
											{getMCPTrustLevelLabel(trustLevel)}
										</option>
									))}
								</select>
							</div>
						</div>

						{formData.transport === MCPTransportType.MCPTransportTypeStdio && (
							<>
								<div className="divider">Stdio</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Command*</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="stdioCommand"
											value={formData.stdioCommand}
											onChange={handleInput}
											className={`input input-bordered w-full rounded-xl ${errors.stdioCommand ? 'input-error' : ''}`}
											spellCheck="false"
											autoComplete="off"
										/>
										{errors.stdioCommand && (
											<div className="label">
												<span className="label-text-alt text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.stdioCommand}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-start gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Args</span>
										<span className="label-text-alt tooltip tooltip-right" data-tip="One argument per line.">
											<FiHelpCircle size={12} />
										</span>
									</label>
									<div className="col-span-9">
										<textarea
											name="stdioArgsText"
											value={formData.stdioArgsText}
											onChange={handleInput}
											className="textarea textarea-bordered h-28 w-full rounded-xl"
											spellCheck="false"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Working Dir</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="stdioWorkingDir"
											value={formData.stdioWorkingDir}
											onChange={handleInput}
											className="input input-bordered w-full rounded-xl"
											spellCheck="false"
											autoComplete="off"
										/>
									</div>
								</div>

								<div className="grid grid-cols-12 items-start gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Env JSON</span>
										<span className="label-text-alt tooltip tooltip-right" data-tip="Plain non-secret env object.">
											<FiHelpCircle size={12} />
										</span>
									</label>
									<div className="col-span-9">
										<textarea
											name="stdioEnvJSON"
											value={formData.stdioEnvJSON}
											onChange={handleInput}
											className={`textarea textarea-bordered h-28 w-full rounded-xl ${
												errors.stdioEnvJSON ? 'textarea-error' : ''
											}`}
											spellCheck="false"
											placeholder='{"NODE_ENV":"production"}'
										/>
										{errors.stdioEnvJSON && (
											<div className="label">
												<span className="label-text-alt text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.stdioEnvJSON}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Startup Timeout MS</span>
									</label>
									<div className="col-span-9">
										<input
											type="number"
											name="stdioStartupTimeoutMS"
											value={formData.stdioStartupTimeoutMS}
											onChange={handleInput}
											className={`input input-bordered w-full rounded-xl ${
												errors.stdioStartupTimeoutMS ? 'input-error' : ''
											}`}
											min={1}
										/>
										{errors.stdioStartupTimeoutMS && (
											<div className="label">
												<span className="label-text-alt text-error flex items-center gap-1">
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
															<span className="label-text text-sm">Env Name</span>
														</label>
														<input
															value={row.envName}
															onChange={e => {
																updateSecretRow(row.rowID, { envName: e.target.value });
															}}
															className="input input-bordered w-full rounded-xl"
															spellCheck="false"
														/>
													</div>

													<div className="col-span-12 md:col-span-3">
														<label className="label py-1">
															<span className="label-text text-sm">Slot</span>
														</label>
														<input
															value={row.slot}
															onChange={e => {
																updateSecretRow(row.rowID, { slot: e.target.value });
															}}
															className="input input-bordered w-full rounded-xl"
															spellCheck="false"
														/>
													</div>

													<div className="col-span-12 md:col-span-5">
														<label className="label py-1">
															<span className="label-text text-sm">
																{row.existingSecretRef ? 'Replace Secret Value' : 'Secret Value'}
															</span>
														</label>
														<input
															type="password"
															value={row.secretValue}
															onChange={e => {
																updateSecretRow(row.rowID, { secretValue: e.target.value });
															}}
															className="input input-bordered w-full rounded-xl"
															autoComplete="new-password"
														/>
														{row.existingSecretRef && (
															<div className="label">
																<span className="label-text-alt text-base-content/70">
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
																<span className="label-text text-sm">Delete existing stored secret</span>
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
										<span className="label-text text-sm">URL*</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="httpURL"
											value={formData.httpURL}
											onChange={handleInput}
											className={`input input-bordered w-full rounded-xl ${errors.httpURL ? 'input-error' : ''}`}
											spellCheck="false"
											autoComplete="off"
											placeholder="https://example.com/mcp"
										/>
										{errors.httpURL && (
											<div className="label">
												<span className="label-text-alt text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpURL}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Timeout MS</span>
									</label>
									<div className="col-span-9">
										<input
											type="number"
											name="httpTimeoutMS"
											value={formData.httpTimeoutMS}
											onChange={handleInput}
											className={`input input-bordered w-full rounded-xl ${errors.httpTimeoutMS ? 'input-error' : ''}`}
											min={1}
										/>
										{errors.httpTimeoutMS && (
											<div className="label">
												<span className="label-text-alt text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors.httpTimeoutMS}
												</span>
											</div>
										)}
									</div>
								</div>

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Auth Mode</span>
									</label>
									<div className="col-span-9">
										<select
											name="httpAuthMode"
											value={formData.httpAuthMode}
											onChange={handleInput}
											className="select select-bordered w-full rounded-xl"
										>
											{Object.values(MCPHTTPAuthMode).map(modeValue => (
												<option key={modeValue} value={modeValue}>
													{getMCPHTTPAuthModeLabel(modeValue)}
												</option>
											))}
										</select>
									</div>
								</div>

								{formData.httpAuthMode !== MCPHTTPAuthMode.MCPHTTPAuthNone && (
									<>
										<div className="grid grid-cols-12 items-start gap-2">
											<label className="label col-span-3">
												<span className="label-text text-sm">Client Credentials Secret</span>
												<span
													className="label-text-alt tooltip tooltip-right"
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
													className={`textarea textarea-bordered h-28 w-full rounded-xl ${
														errors.httpClientCredentials ? 'textarea-error' : ''
													}`}
													spellCheck="false"
													placeholder='{"clientID":"...","clientSecret":"..."}'
												/>
												{formData.httpClientCredentialRef && (
													<div className="label">
														<span className="label-text-alt text-base-content/70">
															Existing credential secret configured. Leave blank to keep it.
														</span>
													</div>
												)}
												{errors.httpClientCredentials && (
													<div className="label">
														<span className="label-text-alt text-error flex items-center gap-1">
															<FiAlertCircle size={12} /> {errors.httpClientCredentials}
														</span>
													</div>
												)}
											</div>
										</div>

										{formData.httpClientCredentialRef && (
											<div className="grid grid-cols-12 items-center gap-2">
												<div className="col-span-3"></div>
												<label className="label col-span-9 cursor-pointer justify-start gap-3">
													<input
														type="checkbox"
														name="httpDeleteClientCredentials"
														checked={formData.httpDeleteClientCredentials}
														onChange={handleInput}
														className="checkbox checkbox-sm"
													/>
													<span className="label-text text-sm">Delete existing client credentials secret</span>
												</label>
											</div>
										)}

										{formData.httpAuthMode === MCPHTTPAuthMode.MCPHTTPAuthOAuth && (
											<div className="grid grid-cols-12 items-center gap-2">
												<label className="label col-span-3">
													<span className="label-text text-sm">Client ID Metadata URL</span>
												</label>
												<div className="col-span-9">
													<input
														type="text"
														name="httpClientIDMetadataDocumentURL"
														value={formData.httpClientIDMetadataDocumentURL}
														onChange={handleInput}
														className={`input input-bordered w-full rounded-xl ${
															errors.httpClientIDMetadataDocumentURL ? 'input-error' : ''
														}`}
														spellCheck="false"
														autoComplete="off"
														placeholder="https://client.example.com/flexigpt-mcp-client.json"
													/>
													{errors.httpClientIDMetadataDocumentURL && (
														<div className="label">
															<span className="label-text-alt text-error flex items-center gap-1">
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
								<span className="label-text text-sm">Approval</span>
							</label>
							<div className="col-span-4">
								<select
									name="defaultApprovalRule"
									value={formData.defaultApprovalRule}
									onChange={handleInput}
									className="select select-bordered w-full rounded-xl"
								>
									{Object.values(MCPApprovalRule).map(rule => (
										<option key={rule} value={rule}>
											{getMCPApprovalRuleLabel(rule)}
										</option>
									))}
								</select>
							</div>

							<label className="label col-span-2">
								<span className="label-text text-sm">Execution</span>
							</label>
							<div className="col-span-3">
								<select
									name="defaultExecutionMode"
									value={formData.defaultExecutionMode}
									onChange={handleInput}
									className="select select-bordered w-full rounded-xl"
								>
									{Object.values(MCPExecutionMode).map(modeValue => (
										<option key={modeValue} value={modeValue}>
											{getMCPExecutionModeLabel(modeValue)}
										</option>
									))}
								</select>
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<div className="col-span-3"></div>
							<div className="col-span-9 grid grid-cols-1 gap-2 md:grid-cols-3">
								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="requireApprovalForUnknownRisk"
										checked={formData.requireApprovalForUnknownRisk}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="label-text text-sm">Unknown risk</span>
								</label>

								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="requireApprovalForWrite"
										checked={formData.requireApprovalForWrite}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="label-text text-sm">Write</span>
								</label>

								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="requireApprovalForDestructive"
										checked={formData.requireApprovalForDestructive}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="label-text text-sm">Destructive</span>
								</label>
							</div>
						</div>

						<div className="divider">Apps Policy</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<div className="col-span-3"></div>
							<div className="col-span-9 grid grid-cols-1 gap-2 md:grid-cols-2">
								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="appsPolicyEnabled"
										checked={formData.appsPolicyEnabled}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="label-text text-sm">Enable MCP Apps</span>
								</label>

								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="allowAppInitiatedToolCalls"
										checked={formData.allowAppInitiatedToolCalls}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="label-text text-sm">Allow app tool calls</span>
								</label>

								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="requireApprovalForOpenLink"
										checked={formData.requireApprovalForOpenLink}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="label-text text-sm">Approve open link</span>
								</label>

								<label className="label cursor-pointer justify-start gap-3">
									<input
										type="checkbox"
										name="requireApprovalForContextUpdates"
										checked={formData.requireApprovalForContextUpdates}
										onChange={handleInput}
										className="checkbox checkbox-sm"
									/>
									<span className="label-text text-sm">Approve context updates</span>
								</label>
							</div>
						</div>

						<div className="divider">Tool Policy Overrides</div>

						<div className="grid grid-cols-12 items-start gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Tool Policies JSON</span>
								<span
									className="label-text-alt tooltip tooltip-right"
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
									className={`textarea textarea-bordered h-36 w-full rounded-xl ${
										errors.toolPoliciesJSON ? 'textarea-error' : ''
									}`}
									spellCheck="false"
									placeholder='{"tool_name":{"toolName":"tool_name","approvalRule":"ask","executionMode":"manual"}}'
								/>
								{errors.toolPoliciesJSON && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.toolPoliciesJSON}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="modal-action">
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={requestClose}>
								Cancel
							</button>
							<button type="submit" className="btn btn-primary rounded-xl" disabled={!isAllValid}>
								Save
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
	if (!props.isOpen) return null;
	if (typeof document === 'undefined' || !document.body) return null;

	const remountKey = props.initialData
		? `${props.mode ?? 'auto'}:${props.initialData.bundleID}:${props.initialData.id}:${props.initialData.modifiedAt}`
		: `${props.mode ?? 'auto'}:new`;

	return createPortal(<AddEditMCPServerModalContent key={remountKey} {...props} />, document.body);
}
