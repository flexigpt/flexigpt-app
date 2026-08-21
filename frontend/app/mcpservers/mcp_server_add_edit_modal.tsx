import type { ChangeEvent, SubmitEventHandler } from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';

import { FiAlertCircle, FiPlus, FiTrash2 } from 'react-icons/fi';

import {
	MCPApprovalRule,
	MCPExecutionMode,
	MCPHTTPAuthMode,
	MCPTransportType,
	MCPTrustLevel,
} from '@/spec/mcp_artifact';

import { validateHTTPURLSecurity } from '@/lib/http_input_utils';
import { validateSlug } from '@/lib/text_utils';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import type { DropdownItem } from '@/components/dropdown';
import { Dropdown } from '@/components/dropdown';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import type {
	MCPBundleView,
	MCPServerDraft,
	MCPServerView,
	MCPStdioSecretDraft,
} from '@/mcpservers/lib/mcp_management';
import { serverDraftFromView } from '@/mcpservers/lib/mcp_management';
import {
	getMCPApprovalRuleLabel,
	getMCPExecutionModeLabel,
	getMCPHTTPAuthModeLabel,
	getMCPTransportLabel,
	getMCPTrustLevelLabel,
} from '@/mcpservers/lib/mcp_server_utils';

interface AddEditMCPServerModalProps {
	isOpen: boolean;
	bundle: MCPBundleView;
	initialServer?: MCPServerView;
	existingLogicalNames: string[];
	onClose: () => void;
	onSubmit: (draft: MCPServerDraft) => Promise<void>;
}

interface ErrorState {
	logicalName?: string;
	displayName?: string;
	stdioCommand?: string;
	stdioEnvironment?: string;
	stdioSecrets?: string;
	stdioTimeout?: string;
	httpURL?: string;
	httpHeaders?: string;
	httpTimeout?: string;
	httpAPIKey?: string;
	httpCredentials?: string;
	httpMetadataURL?: string;
	toolPolicies?: string;
}

const ENV_NAME_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*$/;

const TRANSPORT_ITEMS: Record<MCPTransportType, DropdownItem> = {
	[MCPTransportType.Stdio]: { isEnabled: true },
	[MCPTransportType.StreamableHTTP]: { isEnabled: true },
};

const TRUST_ITEMS: Record<MCPTrustLevel, DropdownItem> = {
	[MCPTrustLevel.Untrusted]: { isEnabled: true },
	[MCPTrustLevel.Trusted]: { isEnabled: true },
};

const AUTH_ITEMS: Record<MCPHTTPAuthMode, DropdownItem> = {
	[MCPHTTPAuthMode.None]: { isEnabled: true },
	[MCPHTTPAuthMode.APIKey]: { isEnabled: true },
	[MCPHTTPAuthMode.OAuth]: { isEnabled: true },
	[MCPHTTPAuthMode.ClientCredentials]: { isEnabled: true },
};

const APPROVAL_ITEMS: Record<MCPApprovalRule, DropdownItem> = {
	[MCPApprovalRule.Ask]: { isEnabled: true },
	[MCPApprovalRule.Allow]: { isEnabled: true },
	[MCPApprovalRule.Deny]: { isEnabled: true },
};

const EXECUTION_ITEMS: Record<MCPExecutionMode, DropdownItem> = {
	[MCPExecutionMode.Manual]: { isEnabled: true },
	[MCPExecutionMode.Auto]: { isEnabled: true },
};

function emptySecretRow(): MCPStdioSecretDraft {
	return {
		envName: '',
		secretValue: '',
		deleteExisting: false,
	};
}

function parseStringRecord(raw: string, label: string): Record<string, string> {
	const value = raw.trim();

	if (!value) {
		return {};
	}

	let parsed: unknown;

	try {
		parsed = JSON.parse(value);
	} catch {
		throw new Error(`${label} must be valid JSON.`);
	}

	if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
		throw new Error(`${label} must be a JSON object.`);
	}

	const output: Record<string, string> = {};

	for (const [key, item] of Object.entries(parsed)) {
		if (typeof item !== 'string') {
			throw new TypeError(`${label} values must all be strings.`);
		}

		output[key] = item;
	}

	return output;
}

function parseObjectRecord(raw: string, label: string): Record<string, any> {
	const value = raw.trim();

	if (!value) {
		return {};
	}

	let parsed: unknown;

	try {
		parsed = JSON.parse(value);
	} catch {
		throw new Error(`${label} must be valid JSON.`);
	}

	if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
		throw new Error(`${label} must be a JSON object.`);
	}

	return parsed as Record<string, any>;
}

function stringify(value: unknown): string {
	if (!value || (typeof value === 'object' && Object.keys(value).length === 0)) {
		return '';
	}

	return JSON.stringify(value, null, 2);
}

function parseOAuthCredentials(raw: string, clientSecretRequired: boolean): string | undefined {
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

	const object = parsed as Record<string, unknown>;

	if (typeof object.clientID !== 'string' || !object.clientID.trim()) {
		return 'Client credentials require a non-empty clientID.';
	}

	if (object.clientID !== object.clientID.trim()) {
		return 'clientID must not have leading or trailing whitespace.';
	}

	if (object.clientSecret !== undefined && typeof object.clientSecret !== 'string') {
		return 'clientSecret must be a string when supplied.';
	}

	if (clientSecretRequired && (typeof object.clientSecret !== 'string' || !object.clientSecret.trim())) {
		return 'Client credentials auth requires a non-empty clientSecret.';
	}

	return undefined;
}

function isPositiveInteger(value: string): boolean {
	if (!value.trim()) {
		return true;
	}

	const numeric = Number(value);
	return Number.isSafeInteger(numeric) && numeric > 0;
}

interface FormState {
	logicalName: string;
	displayName: string;
	enabled: boolean;
	transport: MCPTransportType;
	trustLevel: MCPTrustLevel;

	stdioCommand: string;
	stdioArgsText: string;
	stdioEnvironmentJSON: string;
	stdioTimeoutMS: string;
	stdioSecrets: MCPStdioSecretDraft[];

	httpURL: string;
	httpHeadersJSON: string;
	httpTimeoutMS: string;
	httpAuthMode: MCPHTTPAuthMode;
	httpAPIKeyHeaderName: string;
	httpAPIKeyPrefix: string;
	httpAPIKeySuffix: string;
	httpAPIKeyValue: string;
	httpAPIKeyDeleteExisting: boolean;
	httpUseOAuthClientCredentials: boolean;
	httpOAuthCredentialsJSON: string;
	httpOAuthDeleteExisting: boolean;
	httpClientIDMetadataDocumentURL: string;

	defaultApprovalRule: MCPApprovalRule;
	defaultExecutionMode: MCPExecutionMode;
	requireApprovalForUnknownRisk: boolean;
	requireApprovalForWrite: boolean;
	requireApprovalForDestructive: boolean;

	appsEnabled: boolean;
	appsAllowToolCalls: boolean;
	appsRequireOpenLinkApproval: boolean;
	appsRequireContextUpdateApproval: boolean;

	toolPoliciesJSON: string;
}

function draftToForm(draft: MCPServerDraft): FormState {
	return {
		logicalName: draft.logicalName,
		displayName: draft.displayName,
		enabled: draft.enabled,
		transport: draft.transport,
		trustLevel: draft.trustLevel,

		stdioCommand: draft.stdioCommand,
		stdioArgsText: draft.stdioArgs.join('\n'),
		stdioEnvironmentJSON: stringify(draft.stdioEnv),
		stdioTimeoutMS: draft.stdioStartupTimeoutMS ? String(draft.stdioStartupTimeoutMS) : '',
		stdioSecrets: draft.stdioSecrets,

		httpURL: draft.httpURL,
		httpHeadersJSON: stringify(draft.httpHeaders),
		httpTimeoutMS: draft.httpTimeoutMS ? String(draft.httpTimeoutMS) : '',
		httpAuthMode: draft.httpAuthMode,
		httpAPIKeyHeaderName: draft.httpAPIKey?.headerName ?? 'Authorization',
		httpAPIKeyPrefix: draft.httpAPIKey?.valuePrefix ?? 'Bearer ',
		httpAPIKeySuffix: draft.httpAPIKey?.valueSuffix ?? '',
		httpAPIKeyValue: '',
		httpAPIKeyDeleteExisting: draft.httpAPIKey?.deleteExisting ?? false,
		httpUseOAuthClientCredentials: draft.httpOAuthClientCredentials.useClientCredentials,
		httpOAuthCredentialsJSON: '',
		httpOAuthDeleteExisting: draft.httpOAuthClientCredentials.deleteExisting,
		httpClientIDMetadataDocumentURL: draft.httpClientIDMetadataDocumentURL,

		defaultApprovalRule: draft.defaultPolicy.defaultApprovalRule,
		defaultExecutionMode: draft.defaultPolicy.defaultExecutionMode,
		requireApprovalForUnknownRisk: draft.defaultPolicy.requireApprovalForUnknownRisk,
		requireApprovalForWrite: draft.defaultPolicy.requireApprovalForWrite,
		requireApprovalForDestructive: draft.defaultPolicy.requireApprovalForDestructive,

		appsEnabled: draft.appsPolicy.enabled,
		appsAllowToolCalls: draft.appsPolicy.allowAppInitiatedToolCalls,
		appsRequireOpenLinkApproval: draft.appsPolicy.requireApprovalForOpenLink,
		appsRequireContextUpdateApproval: draft.appsPolicy.requireApprovalForContextUpdates,

		toolPoliciesJSON: stringify(draft.toolPolicies),
	};
}

function formToDraft(form: FormState, initial: MCPServerDraft): MCPServerDraft {
	const stdioSecrets = form.stdioSecrets.map(row => ({
		...row,
		envName: row.envName.trim(),
	}));

	const apiKeyEnabled = form.httpAuthMode === MCPHTTPAuthMode.APIKey;
	const apiKey = apiKeyEnabled
		? {
				...initial.httpAPIKey,
				headerName: form.httpAPIKeyHeaderName.trim(),
				valuePrefix: form.httpAPIKeyPrefix,
				valueSuffix: form.httpAPIKeySuffix,
				secretValue: form.httpAPIKeyValue,
				deleteExisting: form.httpAPIKeyDeleteExisting,
			}
		: undefined;

	return {
		...initial,
		logicalName: form.logicalName.trim(),
		displayName: form.displayName.trim(),
		enabled: form.enabled,
		transport: form.transport,
		trustLevel: form.trustLevel,

		stdioCommand: form.stdioCommand.trim(),
		stdioArgs: form.stdioArgsText
			.split('\n')
			.map(value => value.trim())
			.filter(Boolean),
		stdioEnv: parseStringRecord(form.stdioEnvironmentJSON, 'Environment'),
		stdioStartupTimeoutMS: form.stdioTimeoutMS.trim() ? Number(form.stdioTimeoutMS) : undefined,
		stdioSecrets,

		httpURL: form.httpURL.trim(),
		httpHeaders: parseStringRecord(form.httpHeadersJSON, 'HTTP headers'),
		httpTimeoutMS: form.httpTimeoutMS.trim() ? Number(form.httpTimeoutMS) : undefined,
		httpAuthMode: form.httpAuthMode,
		httpAPIKey: apiKey,
		httpOAuthClientCredentials: {
			...initial.httpOAuthClientCredentials,
			useClientCredentials:
				form.httpAuthMode === MCPHTTPAuthMode.ClientCredentials ||
				(form.httpAuthMode === MCPHTTPAuthMode.OAuth && form.httpUseOAuthClientCredentials),
			secretJSON: form.httpOAuthCredentialsJSON,
			deleteExisting: form.httpOAuthDeleteExisting,
		},
		httpClientIDMetadataDocumentURL:
			form.httpAuthMode === MCPHTTPAuthMode.OAuth ? form.httpClientIDMetadataDocumentURL.trim() : '',

		defaultPolicy: {
			defaultApprovalRule: form.defaultApprovalRule,
			defaultExecutionMode: form.defaultExecutionMode,
			requireApprovalForUnknownRisk: form.requireApprovalForUnknownRisk,
			requireApprovalForWrite: form.requireApprovalForWrite,
			requireApprovalForDestructive: form.requireApprovalForDestructive,
		},
		toolPolicies: parseObjectRecord(form.toolPoliciesJSON, 'Tool policies') as MCPServerDraft['toolPolicies'],
		appsPolicy: {
			enabled: form.appsEnabled,
			allowAppInitiatedToolCalls: form.appsAllowToolCalls,
			requireApprovalForOpenLink: form.appsRequireOpenLinkApproval,
			requireApprovalForContextUpdates: form.appsRequireContextUpdateApproval,
		},
	};
}

function AddEditMCPServerModalContent({
	bundle,
	initialServer,
	existingLogicalNames,
	onSubmit,
}: Omit<AddEditMCPServerModalProps, 'isOpen' | 'onClose'>) {
	const initialDraft = useMemo(() => serverDraftFromView(initialServer), [initialServer]);
	const [form, setForm] = useState<FormState>(() => draftToForm(initialDraft));
	const [errors, setErrors] = useState<ErrorState>({});
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);
	const logicalNameInputRef = useRef<HTMLInputElement | null>(null);
	const { requestClose, unmountingRef } = useModalDialogController();

	const isEdit = Boolean(initialServer);

	useEffect(() => {
		if (isEdit) {
			return;
		}

		const frame = window.requestAnimationFrame(() => {
			logicalNameInputRef.current?.focus({ preventScroll: true });
		});

		return () => {
			window.cancelAnimationFrame(frame);
		};
	}, [isEdit]);

	const validate = (next: FormState): ErrorState => {
		const validation: ErrorState = {};
		const logicalName = next.logicalName.trim();

		if (!logicalName) {
			validation.logicalName = 'Logical name is required.';
		} else {
			const error = validateSlug(logicalName);
			if (error) {
				validation.logicalName = error;
			} else if (!isEdit && existingLogicalNames.includes(logicalName)) {
				validation.logicalName = 'A server with this logical name already exists.';
			}
		}

		if (!next.displayName.trim()) {
			validation.displayName = 'Display name is required.';
		}

		if (next.transport === MCPTransportType.Stdio) {
			if (!next.stdioCommand.trim()) {
				validation.stdioCommand = 'Command is required for stdio servers.';
			}

			try {
				parseStringRecord(next.stdioEnvironmentJSON, 'Environment');
			} catch (error) {
				validation.stdioEnvironment = error instanceof Error ? error.message : 'Invalid environment JSON.';
			}

			if (!isPositiveInteger(next.stdioTimeoutMS)) {
				validation.stdioTimeout = 'Startup timeout must be a positive integer.';
			}

			const seen = new Set<string>();

			for (const row of next.stdioSecrets) {
				const name = row.envName.trim();

				if (!ENV_NAME_PATTERN.test(name)) {
					validation.stdioSecrets = 'Secret environment names must match [A-Za-z_][A-Za-z0-9_]*.';
					break;
				}

				const key = name.toLocaleLowerCase();

				if (seen.has(key)) {
					validation.stdioSecrets = 'Secret environment names must be unique.';
					break;
				}

				seen.add(key);

				if (!row.existingSecretRef && !row.secretValue) {
					validation.stdioSecrets = `Secret value is required for ${name}.`;
					break;
				}
			}
		}

		if (next.transport === MCPTransportType.StreamableHTTP) {
			const urlError = validateHTTPURLSecurity(next.httpURL.trim(), 'MCP server URL');

			if (urlError) {
				validation.httpURL = urlError;
			}

			try {
				parseStringRecord(next.httpHeadersJSON, 'HTTP headers');
			} catch (error) {
				validation.httpHeaders = error instanceof Error ? error.message : 'Invalid HTTP headers JSON.';
			}

			if (!isPositiveInteger(next.httpTimeoutMS)) {
				validation.httpTimeout = 'Timeout must be a positive integer.';
			}

			if (next.httpAuthMode === MCPHTTPAuthMode.APIKey) {
				if (!next.httpAPIKeyHeaderName.trim()) {
					validation.httpAPIKey = 'API key header name is required.';
				} else if (!/^[A-Za-z0-9!#$%&'*+.^_`|~-]+$/.test(next.httpAPIKeyHeaderName.trim())) {
					validation.httpAPIKey = 'API key header name contains invalid characters.';
				} else if (!initialDraft.httpAPIKey?.existingSecretRef && !next.httpAPIKeyValue) {
					validation.httpAPIKey = 'API key value is required.';
				}
			}

			if (
				next.httpAuthMode === MCPHTTPAuthMode.ClientCredentials &&
				!initialDraft.httpOAuthClientCredentials.existingSecretRef &&
				!next.httpOAuthCredentialsJSON.trim()
			) {
				validation.httpCredentials = 'Client credentials auth requires an OAuth client credential secret.';
			}

			const credentialsError = parseOAuthCredentials(
				next.httpOAuthCredentialsJSON,
				next.httpAuthMode === MCPHTTPAuthMode.ClientCredentials
			);

			if (credentialsError) {
				validation.httpCredentials = credentialsError;
			}

			if (next.httpAuthMode === MCPHTTPAuthMode.OAuth && next.httpClientIDMetadataDocumentURL.trim()) {
				try {
					const url = new URL(next.httpClientIDMetadataDocumentURL.trim());

					if (
						url.protocol !== 'https:' ||
						!url.hostname ||
						url.username ||
						url.password ||
						url.hash ||
						!url.pathname ||
						url.pathname === '/'
					) {
						validation.httpMetadataURL =
							'Client ID metadata URL must be an HTTPS URL with a document path and no credentials or fragment.';
					}
				} catch {
					validation.httpMetadataURL = 'Client ID metadata URL must be valid.';
				}
			}
		}

		try {
			parseObjectRecord(next.toolPoliciesJSON, 'Tool policies');
		} catch (error) {
			validation.toolPolicies = error instanceof Error ? error.message : 'Tool policies must be valid JSON.';
		}

		return validation;
	};

	const setFormAndValidate = (next: FormState) => {
		if (isSubmitting) {
			return;
		}

		setForm(next);
		setErrors(validate(next));
	};

	const handleInput = (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
		const target = event.target as HTMLInputElement;
		const value = target.type === 'checkbox' ? target.checked : target.value;
		setFormAndValidate({
			...form,
			[target.name]: value,
		});
	};

	const addSecretRow = () => {
		setFormAndValidate({
			...form,
			stdioSecrets: [...form.stdioSecrets, emptySecretRow()],
		});
	};

	const updateSecretRow = (index: number, patch: Partial<MCPStdioSecretDraft>) => {
		setFormAndValidate({
			...form,
			stdioSecrets: form.stdioSecrets.map((row, rowIndex) => (rowIndex === index ? { ...row, ...patch } : row)),
		});
	};

	const removeSecretRow = (index: number) => {
		setFormAndValidate({
			...form,
			stdioSecrets: form.stdioSecrets.filter((_, rowIndex) => rowIndex !== index),
		});
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async event => {
		event.preventDefault();
		event.stopPropagation();

		if (isSubmitting) {
			return;
		}

		const nextErrors = validate(form);
		setErrors(nextErrors);
		setSubmitError('');

		if (Object.keys(nextErrors).length > 0) {
			return;
		}

		setIsSubmitting(true);

		try {
			await onSubmit(formToDraft(form, initialDraft));

			if (!unmountingRef.current) {
				requestClose(true);
			}
		} catch (error) {
			if (!unmountingRef.current) {
				setSubmitError(error instanceof Error ? error.message : 'Failed to save MCP server.');
			}
		} finally {
			if (!unmountingRef.current) {
				setIsSubmitting(false);
			}
		}
	};

	const allValid = Object.keys(validate(form)).length === 0;

	return (
		<>
			<div className="modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-4xl flex-col overflow-hidden rounded-2xl p-0">
				<ModalHeader
					title={isEdit ? 'Edit MCP Server' : 'Add MCP Server'}
					description={
						isEdit
							? 'Update the portable server Definition and its installation-local secret bindings.'
							: `Add a server to ${bundle.displayName}. Secrets remain outside the portable MCP document.`
					}
					onClose={() => {
						requestClose();
					}}
					closeDisabled={isSubmitting}
				/>

				<form noValidate onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col">
					<div className="app-scrollbar-thin min-h-0 flex-1 space-y-4 overflow-y-auto p-4 sm:p-6">
						{submitError ? (
							<div className="alert alert-error rounded-2xl text-sm">
								<FiAlertCircle size={14} />
								<span>{submitError}</span>
							</div>
						) : null}

						<ModalSection title="Identity and availability">
							<ModalField
								label="Logical Name"
								htmlFor="mcp-logical-name"
								required
								hint="Portable logical name inside this MCP Bundle. It cannot be changed after creation."
								error={errors.logicalName}
							>
								<input
									id="mcp-logical-name"
									ref={logicalNameInputRef}
									type="text"
									name="logicalName"
									value={form.logicalName}
									readOnly={isEdit}
									disabled={isSubmitting}
									autoComplete="off"
									spellCheck="false"
									className={`input w-full rounded-xl ${errors.logicalName ? 'input-error' : ''}`}
									onChange={handleInput}
								/>
							</ModalField>

							<ModalField label="Display Name" htmlFor="mcp-display-name" required error={errors.displayName}>
								<input
									id="mcp-display-name"
									type="text"
									name="displayName"
									value={form.displayName}
									disabled={isSubmitting}
									autoComplete="off"
									spellCheck="false"
									className={`input w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
									onChange={handleInput}
								/>
							</ModalField>

							<ModalField label="Enabled" htmlFor="mcp-server-enabled">
								<input
									id="mcp-server-enabled"
									type="checkbox"
									name="enabled"
									checked={form.enabled}
									disabled={isSubmitting}
									className="toggle toggle-accent"
									onChange={handleInput}
								/>
							</ModalField>
						</ModalSection>

						<ModalSection title="Transport and trust">
							<ModalField label="Transport">
								<Dropdown
									dropdownItems={TRANSPORT_ITEMS}
									selectedKey={form.transport}
									onChange={transport => {
										setFormAndValidate({
											...form,
											transport,
										});
									}}
									getDisplayName={getMCPTransportLabel}
									disabled={isSubmitting}
									title="Transport"
									inlineMenu={true}
								/>
							</ModalField>

							<ModalField label="Trust">
								<Dropdown
									dropdownItems={TRUST_ITEMS}
									selectedKey={form.trustLevel}
									onChange={trustLevel => {
										setFormAndValidate({
											...form,
											trustLevel,
										});
									}}
									getDisplayName={getMCPTrustLevelLabel}
									disabled={isSubmitting}
									title="Trust level"
									inlineMenu={true}
								/>
							</ModalField>
						</ModalSection>

						{form.transport === MCPTransportType.Stdio ? (
							<ModalSection
								title="Stdio runtime"
								description="Secret values are materialized only into declared process environment variables."
							>
								<ModalField label="Command" htmlFor="mcp-stdio-command" required error={errors.stdioCommand}>
									<input
										id="mcp-stdio-command"
										type="text"
										name="stdioCommand"
										value={form.stdioCommand}
										disabled={isSubmitting}
										autoComplete="off"
										spellCheck="false"
										className={`input w-full rounded-xl ${errors.stdioCommand ? 'input-error' : ''}`}
										onChange={handleInput}
									/>
								</ModalField>

								<ModalField label="Arguments" htmlFor="mcp-stdio-args" hint="One argument per line." align="start">
									<textarea
										id="mcp-stdio-args"
										name="stdioArgsText"
										value={form.stdioArgsText}
										disabled={isSubmitting}
										spellCheck="false"
										className="textarea h-28 w-full rounded-xl"
										onChange={handleInput}
									/>
								</ModalField>

								<ModalField
									label="Environment JSON"
									htmlFor="mcp-stdio-environment"
									hint="Plain non-secret environment values."
									error={errors.stdioEnvironment}
									align="start"
								>
									<textarea
										id="mcp-stdio-environment"
										name="stdioEnvironmentJSON"
										value={form.stdioEnvironmentJSON}
										disabled={isSubmitting}
										spellCheck="false"
										placeholder='{"NODE_ENV":"production"}'
										className={`textarea h-28 w-full rounded-xl ${errors.stdioEnvironment ? 'textarea-error' : ''}`}
										onChange={handleInput}
									/>
								</ModalField>

								<ModalField
									label="Startup Timeout"
									htmlFor="mcp-stdio-timeout"
									hint="Optional positive timeout in milliseconds."
									error={errors.stdioTimeout}
								>
									<input
										id="mcp-stdio-timeout"
										type="number"
										name="stdioTimeoutMS"
										value={form.stdioTimeoutMS}
										min={1}
										disabled={isSubmitting}
										className={`input w-full rounded-xl ${errors.stdioTimeout ? 'input-error' : ''}`}
										onChange={handleInput}
									/>
								</ModalField>

								<ModalSection
									title="Secret environment variables"
									description="Existing secrets are represented only as configured bindings."
									className="bg-base-100/40"
									actions={
										<button
											type="button"
											className="btn btn-sm btn-ghost rounded-xl"
											disabled={isSubmitting}
											onClick={addSecretRow}
										>
											<FiPlus size={14} />
											<span className="ml-1">Add Secret</span>
										</button>
									}
								>
									{errors.stdioSecrets ? (
										<div className="alert alert-error rounded-2xl text-sm">
											<FiAlertCircle size={12} />
											<span>{errors.stdioSecrets}</span>
										</div>
									) : null}

									<div className="space-y-3">
										{form.stdioSecrets.map((row, index) => (
											<div
												key={`${row.inputName ?? 'new'}:${index}`}
												className="border-base-content/10 bg-base-100 rounded-2xl border p-3"
											>
												<div className="mb-3 flex items-center justify-between gap-3">
													<div>
														<div className="font-semibold">{row.envName || 'New secret variable'}</div>
														<div className="text-base-content/60 text-xs">
															Secret values are never displayed after saving.
														</div>
													</div>
													<button
														type="button"
														className="btn btn-ghost btn-sm rounded-xl"
														disabled={isSubmitting}
														title="Remove secret environment variable"
														onClick={() => {
															removeSecretRow(index);
														}}
													>
														<FiTrash2 size={14} />
													</button>
												</div>

												<div className="space-y-3">
													<ModalField label="Environment Variable">
														<input
															type="text"
															value={row.envName}
															disabled={isSubmitting}
															autoComplete="off"
															spellCheck="false"
															className="input w-full rounded-xl"
															onChange={event => {
																updateSecretRow(index, {
																	envName: event.target.value,
																});
															}}
														/>
													</ModalField>

													<ModalField label={row.existingSecretRef ? 'Replace Secret Value' : 'Secret Value'}>
														<input
															type="password"
															value={row.secretValue}
															disabled={isSubmitting}
															autoComplete="new-password"
															className="input w-full rounded-xl"
															onChange={event => {
																updateSecretRow(index, {
																	secretValue: event.target.value,
																});
															}}
														/>
														{row.existingSecretRef ? (
															<p className="text-base-content/60 mt-1 text-xs">
																Configured. Leave blank to keep the existing secret.
															</p>
														) : null}
													</ModalField>

													{row.existingSecretRef ? (
														<label className="label cursor-pointer justify-start gap-3">
															<input
																type="checkbox"
																className="checkbox checkbox-sm"
																checked={row.deleteExisting}
																disabled={isSubmitting}
																onChange={event => {
																	updateSecretRow(index, {
																		deleteExisting: event.target.checked,
																	});
																}}
															/>
															<span className="text-sm">Delete existing stored secret</span>
														</label>
													) : null}
												</div>
											</div>
										))}

										{form.stdioSecrets.length === 0 ? (
											<div className="text-base-content/60 text-center text-sm">No secret environment variables.</div>
										) : null}
									</div>
								</ModalSection>
							</ModalSection>
						) : null}

						{form.transport === MCPTransportType.StreamableHTTP ? (
							<ModalSection
								title="Streamable HTTP runtime"
								description="Remote endpoints must use HTTPS. Plain HTTP is only accepted for loopback development endpoints."
							>
								<ModalField label="URL" htmlFor="mcp-http-url" required error={errors.httpURL}>
									<input
										id="mcp-http-url"
										type="text"
										name="httpURL"
										value={form.httpURL}
										disabled={isSubmitting}
										autoComplete="off"
										spellCheck="false"
										placeholder="https://example.com/mcp"
										className={`input w-full rounded-xl ${errors.httpURL ? 'input-error' : ''}`}
										onChange={handleInput}
									/>
								</ModalField>

								<ModalField
									label="Headers JSON"
									htmlFor="mcp-http-headers"
									hint="Plain non-secret headers. API-key secrets are configured below."
									error={errors.httpHeaders}
									align="start"
								>
									<textarea
										id="mcp-http-headers"
										name="httpHeadersJSON"
										value={form.httpHeadersJSON}
										disabled={isSubmitting}
										spellCheck="false"
										placeholder='{"Accept":"application/json"}'
										className={`textarea h-28 w-full rounded-xl ${errors.httpHeaders ? 'textarea-error' : ''}`}
										onChange={handleInput}
									/>
								</ModalField>

								<ModalField
									label="Timeout"
									htmlFor="mcp-http-timeout"
									hint="Optional positive timeout in milliseconds."
									error={errors.httpTimeout}
								>
									<input
										id="mcp-http-timeout"
										type="number"
										name="httpTimeoutMS"
										value={form.httpTimeoutMS}
										min={1}
										disabled={isSubmitting}
										className={`input w-full rounded-xl ${errors.httpTimeout ? 'input-error' : ''}`}
										onChange={handleInput}
									/>
								</ModalField>

								<ModalField label="Auth Mode">
									<Dropdown
										dropdownItems={AUTH_ITEMS}
										selectedKey={form.httpAuthMode}
										onChange={httpAuthMode => {
											setFormAndValidate({
												...form,
												httpAuthMode,
											});
										}}
										getDisplayName={getMCPHTTPAuthModeLabel}
										disabled={isSubmitting}
										title="Auth mode"
										inlineMenu={true}
									/>
								</ModalField>

								{form.httpAuthMode === MCPHTTPAuthMode.APIKey ? (
									<ModalSection
										title="API key secret"
										description="The secret value is stored separately and substituted only into this declared header."
										className="bg-base-100/40"
									>
										<ModalField label="Header Name" error={errors.httpAPIKey}>
											<input
												type="text"
												name="httpAPIKeyHeaderName"
												value={form.httpAPIKeyHeaderName}
												disabled={isSubmitting}
												autoComplete="off"
												spellCheck="false"
												className={`input w-full rounded-xl ${errors.httpAPIKey ? 'input-error' : ''}`}
												onChange={handleInput}
											/>
										</ModalField>

										<ModalField label="Value Prefix" hint='For example, "Bearer ".'>
											<input
												type="text"
												name="httpAPIKeyPrefix"
												value={form.httpAPIKeyPrefix}
												disabled={isSubmitting}
												autoComplete="off"
												className="input w-full rounded-xl"
												onChange={handleInput}
											/>
										</ModalField>

										<ModalField label="Value Suffix">
											<input
												type="text"
												name="httpAPIKeySuffix"
												value={form.httpAPIKeySuffix}
												disabled={isSubmitting}
												autoComplete="off"
												className="input w-full rounded-xl"
												onChange={handleInput}
											/>
										</ModalField>

										<ModalField label="API Key" error={errors.httpAPIKey}>
											<input
												type="password"
												name="httpAPIKeyValue"
												value={form.httpAPIKeyValue}
												disabled={isSubmitting}
												autoComplete="new-password"
												className={`input w-full rounded-xl ${errors.httpAPIKey ? 'input-error' : ''}`}
												onChange={handleInput}
											/>
											{initialDraft.httpAPIKey?.existingSecretRef ? (
												<p className="text-base-content/60 mt-1 text-xs">
													Configured. Leave blank to keep the existing key.
												</p>
											) : null}
										</ModalField>

										{initialDraft.httpAPIKey?.existingSecretRef ? (
											<label className="label cursor-pointer justify-start gap-3">
												<input
													type="checkbox"
													name="httpAPIKeyDeleteExisting"
													checked={form.httpAPIKeyDeleteExisting}
													disabled={isSubmitting}
													className="checkbox checkbox-sm"
													onChange={handleInput}
												/>
												<span className="text-sm">Delete existing API key</span>
											</label>
										) : null}
									</ModalSection>
								) : null}

								{form.httpAuthMode === MCPHTTPAuthMode.OAuth ||
								form.httpAuthMode === MCPHTTPAuthMode.ClientCredentials ? (
									<ModalSection
										title="OAuth client credentials"
										description="OAuth client credentials are local secret values. Authorization-code OAuth can omit them when dynamic registration is supported."
										className="bg-base-100/40"
									>
										{form.httpAuthMode === MCPHTTPAuthMode.OAuth ? (
											<label className="label cursor-pointer justify-start gap-3">
												<input
													type="checkbox"
													name="httpUseOAuthClientCredentials"
													checked={form.httpUseOAuthClientCredentials}
													disabled={isSubmitting}
													className="checkbox checkbox-sm"
													onChange={handleInput}
												/>
												<span className="text-sm">Use pre-registered OAuth client credentials</span>
											</label>
										) : null}

										{form.httpAuthMode === MCPHTTPAuthMode.ClientCredentials || form.httpUseOAuthClientCredentials ? (
											<ModalField
												label="Client Credentials JSON"
												hint='{"clientID":"...","clientSecret":"..."}'
												error={errors.httpCredentials}
												align="start"
											>
												<textarea
													name="httpOAuthCredentialsJSON"
													value={form.httpOAuthCredentialsJSON}
													disabled={isSubmitting}
													spellCheck="false"
													placeholder='{"clientID":"...","clientSecret":"..."}'
													className={`textarea h-28 w-full rounded-xl ${
														errors.httpCredentials ? 'textarea-error' : ''
													}`}
													onChange={handleInput}
												/>
												{initialDraft.httpOAuthClientCredentials.existingSecretRef ? (
													<p className="text-base-content/60 mt-1 text-xs">
														Configured. Leave blank to keep the existing credentials.
													</p>
												) : null}
											</ModalField>
										) : null}

										{initialDraft.httpOAuthClientCredentials.existingSecretRef ? (
											<label className="label cursor-pointer justify-start gap-3">
												<input
													type="checkbox"
													name="httpOAuthDeleteExisting"
													checked={form.httpOAuthDeleteExisting}
													disabled={isSubmitting}
													className="checkbox checkbox-sm"
													onChange={handleInput}
												/>
												<span className="text-sm">Delete existing OAuth client credentials</span>
											</label>
										) : null}

										{form.httpAuthMode === MCPHTTPAuthMode.OAuth ? (
											<ModalField label="Client ID Metadata URL" error={errors.httpMetadataURL}>
												<input
													type="text"
													name="httpClientIDMetadataDocumentURL"
													value={form.httpClientIDMetadataDocumentURL}
													disabled={isSubmitting}
													autoComplete="off"
													spellCheck="false"
													placeholder="https://client.example.com/flexigpt-mcp-client.json"
													className={`input w-full rounded-xl ${errors.httpMetadataURL ? 'input-error' : ''}`}
													onChange={handleInput}
												/>
											</ModalField>
										) : null}
									</ModalSection>
								) : null}
							</ModalSection>
						) : null}

						<ModalSection
							title="Default tool policy"
							description="The policy is stored as an inline MCP Policy Artifact in the Bundle document."
						>
							<ModalField label="Execution">
								<Dropdown
									dropdownItems={EXECUTION_ITEMS}
									selectedKey={form.defaultExecutionMode}
									onChange={defaultExecutionMode => {
										setFormAndValidate({
											...form,
											defaultExecutionMode,
										});
									}}
									getDisplayName={getMCPExecutionModeLabel}
									disabled={isSubmitting}
									title="Default execution"
									inlineMenu={true}
								/>
							</ModalField>

							<ModalField label="Approval">
								<Dropdown
									dropdownItems={APPROVAL_ITEMS}
									selectedKey={form.defaultApprovalRule}
									onChange={defaultApprovalRule => {
										setFormAndValidate({
											...form,
											defaultApprovalRule,
										});
									}}
									getDisplayName={getMCPApprovalRuleLabel}
									disabled={isSubmitting}
									title="Default approval"
									inlineMenu={true}
								/>
							</ModalField>

							<ModalField label="Require approval for" align="start">
								<div className="grid grid-cols-1 gap-3 md:grid-cols-3">
									<label className="label cursor-pointer justify-start gap-3">
										<input
											type="checkbox"
											name="requireApprovalForUnknownRisk"
											checked={form.requireApprovalForUnknownRisk}
											disabled={isSubmitting}
											className="checkbox checkbox-sm"
											onChange={handleInput}
										/>
										<span className="text-sm">Unknown risk</span>
									</label>
									<label className="label cursor-pointer justify-start gap-3">
										<input
											type="checkbox"
											name="requireApprovalForWrite"
											checked={form.requireApprovalForWrite}
											disabled={isSubmitting}
											className="checkbox checkbox-sm"
											onChange={handleInput}
										/>
										<span className="text-sm">Write</span>
									</label>
									<label className="label cursor-pointer justify-start gap-3">
										<input
											type="checkbox"
											name="requireApprovalForDestructive"
											checked={form.requireApprovalForDestructive}
											disabled={isSubmitting}
											className="checkbox checkbox-sm"
											onChange={handleInput}
										/>
										<span className="text-sm">Destructive</span>
									</label>
								</div>
							</ModalField>
						</ModalSection>

						<ModalSection title="MCP Apps policy">
							<ModalField label="App behavior" align="start">
								<div className="grid grid-cols-1 gap-3 md:grid-cols-2">
									<label className="label cursor-pointer justify-start gap-3">
										<input
											type="checkbox"
											name="appsEnabled"
											checked={form.appsEnabled}
											disabled={isSubmitting}
											className="checkbox checkbox-sm"
											onChange={handleInput}
										/>
										<span className="text-sm">Advertise and render MCP Apps</span>
									</label>
									<label className="label cursor-pointer justify-start gap-3">
										<input
											type="checkbox"
											name="appsAllowToolCalls"
											checked={form.appsAllowToolCalls}
											disabled={isSubmitting}
											className="checkbox checkbox-sm"
											onChange={handleInput}
										/>
										<span className="text-sm">Allow app-initiated tool calls</span>
									</label>
									<label className="label cursor-pointer justify-start gap-3">
										<input
											type="checkbox"
											name="appsRequireOpenLinkApproval"
											checked={form.appsRequireOpenLinkApproval}
											disabled={isSubmitting}
											className="checkbox checkbox-sm"
											onChange={handleInput}
										/>
										<span className="text-sm">Approve open-link actions</span>
									</label>
									<label className="label cursor-pointer justify-start gap-3">
										<input
											type="checkbox"
											name="appsRequireContextUpdateApproval"
											checked={form.appsRequireContextUpdateApproval}
											disabled={isSubmitting}
											className="checkbox checkbox-sm"
											onChange={handleInput}
										/>
										<span className="text-sm">Approve context updates</span>
									</label>
								</div>
							</ModalField>
						</ModalSection>

						<ModalSection
							title="Per-tool policy overrides"
							description="Optional object keyed by discovered MCP tool name."
						>
							<ModalField label="Tool Policies JSON" error={errors.toolPolicies} align="start">
								<textarea
									name="toolPoliciesJSON"
									value={form.toolPoliciesJSON}
									disabled={isSubmitting}
									spellCheck="false"
									placeholder='{"tool_name":{"toolName":"tool_name","approvalRule":"ask"}}'
									className={`textarea h-36 w-full rounded-xl ${errors.toolPolicies ? 'textarea-error' : ''}`}
									onChange={handleInput}
								/>
							</ModalField>
						</ModalSection>

						{initialServer ? (
							<ModalSection title="Artifact metadata">
								<ManagementInfoGrid>
									<ManagementInfoRow label="Artifact ID" mono>
										{initialServer.ref.artifactID}
									</ManagementInfoRow>
									<ManagementInfoRow label="Root ID" mono>
										{initialServer.ref.rootID}
									</ManagementInfoRow>
									<ManagementInfoRow label="Artifact Revision">{initialServer.artifact.revision}</ManagementInfoRow>
									<ManagementInfoRow label="Built-in">{initialServer.builtIn ? 'Yes' : 'No'}</ManagementInfoRow>
								</ManagementInfoGrid>
							</ModalSection>
						) : null}
					</div>

					<ModalActions>
						<button
							type="button"
							className="btn bg-base-300 rounded-xl"
							disabled={isSubmitting}
							onClick={() => {
								requestClose();
							}}
						>
							Cancel
						</button>
						<button type="submit" className="btn btn-primary rounded-xl" disabled={!allValid || isSubmitting}>
							{isSubmitting ? 'Saving...' : 'Save'}
						</button>
					</ModalActions>
				</form>
			</div>
			<ModalBackdrop enabled={false} />
		</>
	);
}

export function AddEditMCPServerModal(props: AddEditMCPServerModalProps) {
	if (!props.isOpen) {
		return null;
	}

	const modalKey = props.initialServer
		? `${props.initialServer.ref.rootID}:${props.initialServer.ref.artifactID}:${props.initialServer.artifact.revision}`
		: 'new';

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose} blockCancel>
			<AddEditMCPServerModalContent key={modalKey} {...props} />
		</ModalDialog>
	);
}
