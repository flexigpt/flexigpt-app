import type { SubmitEventHandler } from 'react';
import { useMemo, useState } from 'react';
import { FiAlertCircle } from 'react-icons/fi';

import type { ModelAuthentication, ModelConnection, ModelHeaderPatch, ModelProviderDocument } from '@/spec/model';
import { ModelArtifactType, ModelAuthenticationMode } from '@/spec/model';

import { getErrorMessage } from '@/lib/error_utils';
import {
	httpHeadersEqual,
	omitSensitiveHTTPHeaders,
	parseHTTPHeadersJSON,
	validateHTTPHeaderName,
} from '@/lib/http_input_utils';
import { formatJSON } from '@/lib/jsonschema_utils';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import type { ModelProviderManagementItem } from '@/apis/model_management';

import { Dropdown } from '@/components/dropdown';
import { MANAGEMENT_MODAL_FORM_CLASS } from '@/components/managementui/management_class_consts';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';
import { ReadOnlyValue } from '@/components/read_only_value';

import { parseProviderAuthentication, parseProviderConnection } from '@/models/lib/document';
import { makeUniqueLogicalName, providerSDKDropdownItems } from '@/models/lib/management_form_utils';
import { ModelProviderModalMode, ProviderCredentialAction } from '@/models/lib/management_types';
import { getProviderSDKOption } from '@/models/lib/provider_sdk';

const COPY_SOURCE_NONE = '__copy_source_none__';

type ProviderFormField =
	'displayName' | 'adapter' | 'origin' | 'chatPath' | 'apiKeyHeaderKey' | 'defaultHeadersRawJSON' | 'apiKey' | 'form';

type ProviderFormErrors = Partial<Record<ProviderFormField, string>>;

interface ProviderFormState {
	logicalName: string;
	displayName: string;
	adapter: string;
	origin: string;
	chatPath: string;
	apiKeyHeaderKey: string;
	defaultHeadersRawJSON: string;
	enabled: boolean;
	credential: string;
	authenticationMode: ModelAuthenticationMode;
	headerRemovals: string[];
}

export interface ProviderEditorSubmission {
	document: ModelProviderDocument;
	enabled: boolean;
	credentialAction: ProviderCredentialAction;
	credential: string;
}

interface ModelProviderAddEditModalProps {
	isOpen: boolean;
	mode: ModelProviderModalMode;
	provider?: ModelProviderManagementItem;
	allProviders?: ModelProviderManagementItem[];
	onClose: () => void;
	onSubmit: (submission: ProviderEditorSubmission) => Promise<void>;
}

function safeHeadersFromDocument(document: ModelProviderDocument | undefined): Record<string, string> {
	try {
		return omitSensitiveHTTPHeaders(document?.connection?.headers?.set) ?? {};
	} catch {
		return {};
	}
}

function safeHeadersJSON(headers: Record<string, string>): string {
	try {
		return formatJSON(headers) || '{}';
	} catch {
		return '{}';
	}
}

function initialFormState(
	provider: ModelProviderManagementItem | undefined,
	existingLogicalNames: string[]
): ProviderFormState {
	const document = provider?.view.document;
	const sdkOption = getProviderSDKOption(document?.adapter ?? 'openai.responses');
	const authentication = document?.authentication;

	return {
		logicalName: document?.name ?? makeUniqueLogicalName(document?.displayName ?? '', existingLogicalNames, 'provider'),
		displayName: document?.displayName ?? '',
		adapter: document?.adapter ?? 'openai.responses',
		origin: document?.connection?.origin ?? '',
		chatPath: document?.connection?.path ?? sdkOption?.defaultPath ?? '/v1/responses',
		apiKeyHeaderKey: authentication?.headerName ?? sdkOption?.defaultAPIKeyHeader ?? 'Authorization',
		defaultHeadersRawJSON: safeHeadersJSON(safeHeadersFromDocument(document)),
		enabled: provider?.list.enabled ?? true,
		credential: '',
		authenticationMode: authentication?.mode ?? ModelAuthenticationMode.APIKeyHeader,
		headerRemovals: [...(document?.connection?.headers?.remove ?? [])],
	};
}

function rawHeadersEqual(raw: string, expected: Record<string, string>): boolean {
	try {
		return httpHeadersEqual(parseHTTPHeadersJSON(raw, 'Default headers'), expected);
	} catch {
		return false;
	}
}

function buildConnection(form: ProviderFormState): ModelConnection {
	const headers = parseHTTPHeadersJSON(form.defaultHeadersRawJSON, 'Default headers');
	const headerPatch: ModelHeaderPatch | undefined =
		Object.keys(headers).length > 0 || form.headerRemovals.length > 0
			? {
					...(Object.keys(headers).length > 0 ? { set: headers } : {}),
					...(form.headerRemovals.length > 0 ? { remove: form.headerRemovals } : {}),
				}
			: undefined;

	return parseProviderConnection(
		JSON.stringify({
			origin: form.origin,
			path: form.chatPath,
			...(headerPatch ? { headers: headerPatch } : {}),
		})
	);
}

function buildAuthentication(
	form: ProviderFormState,
	existingAuthentication: ModelAuthentication | undefined
): ModelAuthentication {
	if (form.authenticationMode === ModelAuthenticationMode.None) {
		return {
			mode: ModelAuthenticationMode.None,
		};
	}

	return parseProviderAuthentication(
		JSON.stringify({
			...existingAuthentication,
			mode: form.authenticationMode,
			headerName: form.apiKeyHeaderKey.trim(),
		})
	);
}

function buildDocument(
	form: ProviderFormState,
	existingDocument: ModelProviderDocument | undefined
): ModelProviderDocument {
	return {
		...existingDocument,
		type: ModelArtifactType.Provider,
		name: existingDocument?.name ?? form.logicalName,
		displayName: form.displayName.trim(),
		adapter: form.adapter,
		connection: buildConnection(form),
		authentication: buildAuthentication(form, existingDocument?.authentication),
	};
}

function validateForm(
	form: ProviderFormState,
	mode: ModelProviderModalMode,
	existingDocument: ModelProviderDocument | undefined
): ProviderFormErrors {
	const errors: ProviderFormErrors = {};

	try {
		if (!form.displayName.trim()) {
			errors.displayName = 'Provider name is required.';
		}

		if (!getProviderSDKOption(form.adapter)) {
			errors.adapter = 'Select a supported compatibility mode.';
		}

		try {
			parseHTTPHeadersJSON(form.defaultHeadersRawJSON, 'Default headers');
		} catch (error) {
			errors.defaultHeadersRawJSON = getErrorMessage(error, 'Default headers are invalid.');
		}

		if (!errors.defaultHeadersRawJSON) {
			try {
				buildConnection(form);
			} catch (error) {
				errors.origin = getErrorMessage(error, 'Connection settings are invalid.');
			}
		}

		if (form.authenticationMode !== ModelAuthenticationMode.None) {
			const headerError = validateHTTPHeaderName(form.apiKeyHeaderKey, 'API-key header name');
			if (headerError) {
				errors.apiKeyHeaderKey = headerError;
			}
		}

		if (!errors.apiKeyHeaderKey) {
			try {
				buildAuthentication(form, existingDocument?.authentication);
			} catch (error) {
				errors.apiKeyHeaderKey = getErrorMessage(error, 'Authentication settings are invalid.');
			}
		}

		if (
			mode === ModelProviderModalMode.Add &&
			form.authenticationMode !== ModelAuthenticationMode.None &&
			!form.credential.trim()
		) {
			errors.apiKey = 'An API key is required for this provider.';
		}
	} catch (error) {
		errors.form = getErrorMessage(error, 'Provider settings could not be validated.');
	}

	return errors;
}

function ModelProviderAddEditModalContent({
	mode,
	provider,
	allProviders = [],
	onSubmit,
}: Omit<ModelProviderAddEditModalProps, 'isOpen' | 'onClose'>) {
	const readOnly = mode === ModelProviderModalMode.View;
	const existingLogicalNames = useMemo(() => allProviders.map(item => item.view.document.name), [allProviders]);
	const [form, setForm] = useState<ProviderFormState>(() => initialFormState(provider, existingLogicalNames));
	const [errors, setErrors] = useState<ProviderFormErrors>({});
	const [copySourceKey, setCopySourceKey] = useState(COPY_SOURCE_NONE);
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);
	const { requestClose, unmountingRef } = useModalDialogController();

	const validation = useMemo(
		() => validateForm(form, mode, provider?.view.document),
		[form, mode, provider?.view.document]
	);

	const sdkItems = useMemo(() => providerSDKDropdownItems(form.adapter), [form.adapter]);

	const copySourceItems = useMemo(() => {
		const items: Record<string, { isEnabled: boolean; displayName: string }> = {
			[COPY_SOURCE_NONE]: {
				isEnabled: true,
				displayName: 'Choose a provider',
			},
		};

		for (const item of allProviders) {
			items[`${item.list.ref.rootID}:${item.list.ref.artifactID}`] = {
				isEnabled: true,
				displayName: item.list.displayName || 'Provider',
			};
		}

		return items;
	}, [allProviders]);

	const copySourceKeys = useMemo(() => Object.keys(copySourceItems), [copySourceItems]);

	const title =
		mode === ModelProviderModalMode.Add
			? 'Add Provider'
			: mode === ModelProviderModalMode.Edit
				? 'Edit Provider'
				: 'View Provider';

	const updateForm = (next: ProviderFormState) => {
		setForm(next);
		setErrors(validateForm(next, mode, provider?.view.document));
	};

	const applyCopy = (sourceKey: string) => {
		if (sourceKey === COPY_SOURCE_NONE) {
			return;
		}

		try {
			const source = allProviders.find(item => `${item.list.ref.rootID}:${item.list.ref.artifactID}` === sourceKey);
			if (!source) {
				setSubmitError('The selected provider is no longer available.');
				return;
			}

			const sourceDocument = source.view.document;
			const sourceSDK = getProviderSDKOption(sourceDocument.adapter);
			const sourceAuthentication = sourceDocument.authentication;
			const displayName = `${source.list.displayName || 'Provider'} Copy`;

			updateForm({
				logicalName: makeUniqueLogicalName(displayName, existingLogicalNames, 'provider'),
				displayName,
				adapter: sourceDocument.adapter,
				origin: sourceDocument.connection?.origin ?? '',
				chatPath: sourceDocument.connection?.path ?? sourceSDK?.defaultPath ?? '/v1/responses',
				apiKeyHeaderKey: sourceAuthentication?.headerName ?? sourceSDK?.defaultAPIKeyHeader ?? 'Authorization',
				defaultHeadersRawJSON: safeHeadersJSON(safeHeadersFromDocument(sourceDocument)),
				enabled: true,
				credential: '',
				authenticationMode: sourceAuthentication?.mode ?? ModelAuthenticationMode.APIKeyHeader,
				headerRemovals: [],
			});
		} catch (error) {
			setSubmitError(getErrorMessage(error, 'Provider settings could not be copied.'));
		}
	};

	const changeSDK = (adapter: string) => {
		const previousSDK = getProviderSDKOption(form.adapter);
		const nextSDK = getProviderSDKOption(adapter);
		if (!nextSDK) {
			setErrors(previous => ({
				...previous,
				adapter: 'This compatibility mode is not supported.',
			}));
			return;
		}

		const usesPreviousPath = !form.chatPath || form.chatPath === previousSDK?.defaultPath;
		const usesPreviousHeader = !form.apiKeyHeaderKey || form.apiKeyHeaderKey === previousSDK?.defaultAPIKeyHeader;
		const usesPreviousHeaders = previousSDK
			? rawHeadersEqual(form.defaultHeadersRawJSON, previousSDK.defaultHeaders)
			: false;

		updateForm({
			...form,
			adapter,
			chatPath: usesPreviousPath ? nextSDK.defaultPath : form.chatPath,
			apiKeyHeaderKey: usesPreviousHeader ? nextSDK.defaultAPIKeyHeader : form.apiKeyHeaderKey,
			defaultHeadersRawJSON: usesPreviousHeaders ? safeHeadersJSON(nextSDK.defaultHeaders) : form.defaultHeadersRawJSON,
		});
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		event.stopPropagation();

		if (readOnly) {
			requestClose();
			return;
		}

		setErrors(validation);
		if (Object.keys(validation).length > 0) {
			return;
		}

		let document: ModelProviderDocument;
		try {
			document = buildDocument(form, provider?.view.document);
		} catch (error) {
			setSubmitError(getErrorMessage(error, 'Provider settings could not be prepared.'));
			return;
		}

		setIsSubmitting(true);
		setSubmitError('');

		void (async () => {
			try {
				await onSubmit({
					document,
					enabled: form.enabled,
					credentialAction: form.credential.trim() ? ProviderCredentialAction.Set : ProviderCredentialAction.Unchanged,
					credential: form.credential.trim(),
				});
				requestClose(true);
			} catch (error) {
				if (!unmountingRef.current) {
					setSubmitError(getErrorMessage(error, 'Provider could not be saved.'));
				}
			} finally {
				if (!unmountingRef.current) {
					setIsSubmitting(false);
				}
			}
		})();
	};

	return (
		<div className="modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-4xl flex-col overflow-hidden rounded-2xl p-0">
			<ModalHeader
				title={title}
				description={
					readOnly
						? 'Inspect provider compatibility and connection settings.'
						: 'Configure provider compatibility, connection settings, credentials, and availability.'
				}
				onClose={() => {
					requestClose();
				}}
				closeDisabled={isSubmitting}
			/>

			<form noValidate onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col" aria-busy={isSubmitting}>
				<div className={`app-scrollbar-thin min-h-0 flex-1 overflow-y-auto p-4 sm:p-6 ${MANAGEMENT_MODAL_FORM_CLASS}`}>
					{submitError ? (
						<div className="alert alert-error rounded-xl text-sm" role="alert">
							<div className="flex items-center gap-2">
								<FiAlertCircle size={15} />
								<span>{submitError}</span>
							</div>
						</div>
					) : null}

					{Object.keys(errors).length > 0 ? (
						<div className="alert alert-error rounded-xl text-sm" role="alert">
							<div className="flex items-start gap-2">
								<FiAlertCircle size={15} className="mt-0.5 shrink-0" />
								<ul className="list-disc pl-4">
									{Object.entries(errors).map(([field, message]) => (
										<li key={field}>{message}</li>
									))}
								</ul>
							</div>
						</div>
					) : null}

					{mode === ModelProviderModalMode.Add && allProviders.length > 0 ? (
						<ModalSection
							title="Copy an Existing Provider"
							description="Copies endpoint and compatibility settings. Credentials are never copied."
						>
							<ModalField label="Copy From">
								<Dropdown<string>
									dropdownItems={copySourceItems}
									orderedKeys={copySourceKeys}
									selectedKey={copySourceKey}
									onChange={key => {
										setCopySourceKey(key);
										applyCopy(key);
									}}
									filterDisabled={false}
									title="Select a provider to copy"
									getDisplayName={key => copySourceItems[key]?.displayName ?? 'Choose a provider'}
									disabled={isSubmitting}
								/>
							</ModalField>
						</ModalSection>
					) : null}

					<ModalSection
						title="Identity and Compatibility"
						description="Choose the compatible API before configuring the endpoint."
					>
						<ModalField label="Display Name" htmlFor="provider-display-name" required error={errors.displayName}>
							{readOnly ? (
								<ReadOnlyValue value={form.displayName || '—'} />
							) : (
								<input
									id="provider-display-name"
									type="text"
									className={`input w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
									value={form.displayName}
									onChange={event => {
										const displayName = event.target.value;
										updateForm({
											...form,
											displayName,
											logicalName:
												mode === ModelProviderModalMode.Add
													? makeUniqueLogicalName(displayName, existingLogicalNames, 'provider')
													: form.logicalName,
										});
									}}
									autoComplete="off"
									spellCheck="false"
									autoFocus
									disabled={isSubmitting}
								/>
							)}
						</ModalField>

						<ModalField label="SDK Type" error={errors.adapter}>
							{readOnly ? (
								<ReadOnlyValue
									value={getProviderSDKOption(form.adapter)?.displayName ?? 'Unsupported compatibility mode'}
								/>
							) : (
								<Dropdown<string>
									dropdownItems={sdkItems}
									selectedKey={form.adapter}
									onChange={changeSDK}
									filterDisabled={false}
									title="Select SDK type"
									getDisplayName={key => sdkItems[key]?.displayName ?? 'Unsupported compatibility mode'}
									disabled={isSubmitting}
								/>
							)}
						</ModalField>
					</ModalSection>

					<ModalSection
						title="Connection"
						description="Configure the origin, chat path, credential header, and stable non-secret headers."
					>
						<ModalField label="Origin" htmlFor="provider-origin" required error={errors.origin}>
							{readOnly ? (
								<ReadOnlyValue value={form.origin || '—'} />
							) : (
								<input
									id="provider-origin"
									type="url"
									className={`input w-full rounded-xl ${errors.origin ? 'input-error' : ''}`}
									value={form.origin}
									onChange={event => {
										updateForm({
											...form,
											origin: event.target.value,
										});
									}}
									placeholder="https://api.example.com"
									autoComplete="off"
									spellCheck="false"
									disabled={isSubmitting}
								/>
							)}
						</ModalField>

						<ModalField label="Chat Path" htmlFor="provider-chat-path" required error={errors.origin}>
							{readOnly ? (
								<ReadOnlyValue value={form.chatPath || 'Adapter default'} />
							) : (
								<input
									id="provider-chat-path"
									type="text"
									className="input w-full rounded-xl"
									value={form.chatPath}
									onChange={event => {
										updateForm({
											...form,
											chatPath: event.target.value,
										});
									}}
									placeholder="/v1/responses"
									autoComplete="off"
									spellCheck="false"
									disabled={isSubmitting}
								/>
							)}
						</ModalField>

						{form.authenticationMode !== ModelAuthenticationMode.None ? (
							<ModalField label="API-Key Header" htmlFor="provider-api-key-header" error={errors.apiKeyHeaderKey}>
								{readOnly ? (
									<ReadOnlyValue value={form.apiKeyHeaderKey || '—'} />
								) : (
									<input
										id="provider-api-key-header"
										type="text"
										className={`input w-full rounded-xl ${errors.apiKeyHeaderKey ? 'input-error' : ''}`}
										value={form.apiKeyHeaderKey}
										onChange={event => {
											updateForm({
												...form,
												apiKeyHeaderKey: event.target.value,
											});
										}}
										autoComplete="off"
										spellCheck="false"
										disabled={isSubmitting}
									/>
								)}
							</ModalField>
						) : null}

						<ModalField
							label="Default Headers (JSON)"
							htmlFor="provider-default-headers"
							error={errors.defaultHeadersRawJSON}
							align="start"
						>
							{readOnly ? (
								<pre className="bg-base-300 max-h-56 overflow-auto rounded-xl p-3 text-xs whitespace-pre-wrap">
									{form.defaultHeadersRawJSON || '{}'}
								</pre>
							) : (
								<>
									<textarea
										id="provider-default-headers"
										className={`textarea h-28 w-full rounded-xl font-mono text-xs ${
											errors.defaultHeadersRawJSON ? 'textarea-error' : ''
										}`}
										value={form.defaultHeadersRawJSON}
										onChange={event => {
											updateForm({
												...form,
												defaultHeadersRawJSON: event.target.value,
											});
										}}
										placeholder='{"Content-Type":"application/json"}'
										spellCheck="false"
										disabled={isSubmitting}
									/>
									{errors.defaultHeadersRawJSON ? (
										<div className="label">
											<span className="text-error text-xs">{errors.defaultHeadersRawJSON}</span>
										</div>
									) : null}
								</>
							)}
						</ModalField>
					</ModalSection>

					<ModalSection
						title="Security and Availability"
						description="Credentials are write-only. Leaving an existing credential blank preserves it."
					>
						{!readOnly && form.authenticationMode !== ModelAuthenticationMode.None ? (
							<ModalField
								label="API Key"
								htmlFor="provider-api-key"
								required={mode === ModelProviderModalMode.Add}
								error={errors.apiKey}
							>
								<input
									id="provider-api-key"
									type="password"
									autoComplete="new-password"
									className={`input w-full rounded-xl ${errors.apiKey ? 'input-error' : ''}`}
									value={form.credential}
									onChange={event => {
										updateForm({
											...form,
											credential: event.target.value,
										});
									}}
									placeholder={
										mode === ModelProviderModalMode.Edit && provider?.apiKey.configured
											? 'Leave blank to keep the current API key'
											: ''
									}
									disabled={isSubmitting}
								/>
							</ModalField>
						) : null}

						<ModalField label="Enabled">
							{readOnly ? (
								<ReadOnlyValue value={form.enabled ? 'Enabled' : 'Disabled'} />
							) : (
								<input
									type="checkbox"
									className="toggle toggle-accent"
									checked={form.enabled}
									onChange={event => {
										updateForm({
											...form,
											enabled: event.target.checked,
										});
									}}
									disabled={isSubmitting}
								/>
							)}
						</ModalField>
					</ModalSection>
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
						{readOnly ? 'Close' : 'Cancel'}
					</button>

					{!readOnly ? (
						<button
							type="submit"
							className="btn btn-primary rounded-xl"
							disabled={isSubmitting || Object.keys(validation).length > 0}
						>
							{isSubmitting ? 'Saving...' : mode === ModelProviderModalMode.Add ? 'Add Provider' : 'Save Changes'}
						</button>
					) : null}
				</ModalActions>
			</form>
		</div>
	);
}

export function ModelProviderAddEditModal({
	isOpen,
	mode,
	provider,
	allProviders,
	onClose,
	onSubmit,
}: ModelProviderAddEditModalProps) {
	if (!isOpen) {
		return null;
	}

	const key =
		mode === ModelProviderModalMode.Add
			? 'add-provider'
			: `${mode}:${provider?.list.ref.rootID ?? 'provider'}:${
					provider?.list.ref.artifactID ?? 'unknown'
				}:${provider?.list.revision ?? 0}`;

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose} blockCancel={mode !== ModelProviderModalMode.View}>
			<ModelProviderAddEditModalContent
				key={key}
				mode={mode}
				provider={provider}
				allProviders={allProviders}
				onSubmit={onSubmit}
			/>
		</ModalDialog>
	);
}
