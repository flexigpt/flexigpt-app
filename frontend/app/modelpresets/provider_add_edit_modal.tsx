import type { ChangeEvent, SubmitEventHandler } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiUpload, FiX } from 'react-icons/fi';

import type { ProviderName } from '@/spec/inference';
import { ProviderSDKType, SDK_DEFAULTS, SDK_DISPLAY_NAME } from '@/spec/inference';
import type { PatchProviderPresetPayload, PostProviderPresetPayload, ProviderPreset } from '@/spec/modelpreset';

import { GenerateRandomNumberString } from '@/lib/encode_decode';
import {
	omitSensitiveHTTPHeaders,
	parseHTTPHeadersJSON,
	redactSensitiveHTTPHeaders,
	restoreRedactedHTTPHeaders,
	validateHTTPHeaderName,
	validateHTTPURLSecurity,
} from '@/lib/http_input_utils';
import { omitManyKeys } from '@/lib/obj_utils';
import { MessageEnterValidURL, validateUrlForInput } from '@/lib/url_utils';

import { useDialogController } from '@/hooks/use_dialog_controller';

import { Dropdown } from '@/components/dropdown';
import { MANAGEMENT_MODAL_FORM_CLASS } from '@/components/managementui/management_class_consts';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalBackdrop } from '@/components/modal/modal_backdrop';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';
import { ReadOnlyValue } from '@/components/read_only_value';

type ModalMode = 'add' | 'edit' | 'view';

function parseDefaultHeadersRawJSON(raw: string): Record<string, string> {
	return parseHTTPHeadersJSON(raw, 'Default headers');
}

function normalizeHeadersRecord(headers: Record<string, string>): Record<string, string> {
	return Object.fromEntries(Object.entries(headers).toSorted(([a], [b]) => a.localeCompare(b)));
}

function headersEqual(a: Record<string, string>, b: Record<string, string>): boolean {
	return JSON.stringify(normalizeHeadersRecord(a)) === JSON.stringify(normalizeHeadersRecord(b));
}

function usesSDKDefaultHeaders(raw: string, sdkType: ProviderSDKType): boolean {
	if (!raw.trim()) {
		return true;
	}

	try {
		return headersEqual(parseDefaultHeadersRawJSON(raw), SDK_DEFAULTS[sdkType].defaultHeaders);
	} catch {
		return false;
	}
}

interface ProviderFormData {
	providerName: string;
	displayName: string;
	sdkType: ProviderSDKType;
	isEnabled: boolean;
	origin: string;
	chatCompletionPathPrefix: string;
	apiKeyHeaderKey: string;
	defaultHeadersRawJSON: string;
	apiKey: string;
}

const DEFAULT_SDK_TYPE = ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions;
const DEFAULT_SDK_DEFAULTS = SDK_DEFAULTS[DEFAULT_SDK_TYPE];

const DEFAULT_FORM: ProviderFormData = {
	providerName: '',
	displayName: '',
	sdkType: DEFAULT_SDK_TYPE,
	isEnabled: true,
	origin: '',
	chatCompletionPathPrefix: DEFAULT_SDK_DEFAULTS.chatPath,
	apiKeyHeaderKey: DEFAULT_SDK_DEFAULTS.apiKeyHeaderKey,
	defaultHeadersRawJSON: JSON.stringify(DEFAULT_SDK_DEFAULTS.defaultHeaders, null, 2),
	apiKey: '',
};

type ErrorState = Partial<Record<keyof ProviderFormData, string>>;

function getInitialFormData(mode: ModalMode, initialPreset?: ProviderPreset): ProviderFormData {
	if ((mode === 'edit' || mode === 'view') && initialPreset) {
		return {
			providerName: initialPreset.name,
			displayName: initialPreset.displayName,
			sdkType: initialPreset.sdkType,
			isEnabled: initialPreset.isEnabled,
			origin: initialPreset.origin,
			chatCompletionPathPrefix: initialPreset.chatCompletionPathPrefix,
			apiKeyHeaderKey: initialPreset.apiKeyHeaderKey,
			defaultHeadersRawJSON: JSON.stringify(redactSensitiveHTTPHeaders(initialPreset.defaultHeaders) ?? {}, null, 2),
			apiKey: '',
		};
	}

	return { ...DEFAULT_FORM };
}

interface AddEditProviderPresetModalProps {
	isOpen: boolean;
	mode: ModalMode;
	onClose: () => void;
	onSubmit: (
		providerName: ProviderName,
		payload: PostProviderPresetPayload | PatchProviderPresetPayload,
		apiKey: string | null
	) => Promise<void>;
	existingProviderNames: ProviderName[];
	allProviderPresets: Record<ProviderName, ProviderPreset>;
	initialPreset?: ProviderPreset;
	apiKeyAlreadySet?: boolean;
}

function AddEditProviderPresetModalContent({
	mode,
	onClose,
	onSubmit,
	existingProviderNames,
	allProviderPresets,
	initialPreset,
	apiKeyAlreadySet = false,
}: AddEditProviderPresetModalProps) {
	const isReadOnly = mode === 'view';

	const [formData, setFormData] = useState<ProviderFormData>(() => getInitialFormData(mode, initialPreset));
	const [errors, setErrors] = useState<ErrorState>({});
	const [prefillMode, setPrefillMode] = useState(false);
	const [selectedPrefillKey, setSelectedPrefillKey] = useState<ProviderName | null>(null);
	const [isSubmitting, setIsSubmitting] = useState(false);
	const [submitError, setSubmitError] = useState('');

	const { dialogRef, requestClose, handleClose, handleCancel, unmountingRef } = useDialogController({
		onClose,
		blockCancel: !isReadOnly,
		isBusy: isSubmitting,
	});

	const providerNameInputRef = useRef<HTMLInputElement | null>(null);
	const displayNameInputRef = useRef<HTMLInputElement | null>(null);
	const originInputRef = useRef<HTMLInputElement | null>(null);

	const prefillDropdownItems: Record<ProviderName, { isEnabled: boolean; displayName: string }> = useMemo(() => {
		const out = {} as Record<ProviderName, { isEnabled: boolean; displayName: string }>;

		for (const [name, preset] of Object.entries(allProviderPresets)) {
			out[name] = { isEnabled: true, displayName: preset.displayName || name };
		}

		return out;
	}, [allProviderPresets]);

	const prefillKeys = useMemo(() => Object.keys(prefillDropdownItems), [prefillDropdownItems]);

	const sdkDropdownItems: Record<ProviderSDKType, { isEnabled: boolean; displayName: string }> = useMemo(
		() => ({
			[ProviderSDKType.ProviderSDKTypeAnthropic]: {
				isEnabled: true,
				displayName: SDK_DISPLAY_NAME[ProviderSDKType.ProviderSDKTypeAnthropic],
			},
			[ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions]: {
				isEnabled: true,
				displayName: SDK_DISPLAY_NAME[ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions],
			},
			[ProviderSDKType.ProviderSDKTypeOpenAIResponses]: {
				isEnabled: true,
				displayName: SDK_DISPLAY_NAME[ProviderSDKType.ProviderSDKTypeOpenAIResponses],
			},
			[ProviderSDKType.ProviderSDKTypeGoogleGenerateContent]: {
				isEnabled: true,
				displayName: SDK_DISPLAY_NAME[ProviderSDKType.ProviderSDKTypeGoogleGenerateContent],
			},
		}),
		[]
	);

	useEffect(() => {
		const focusTimer = window.setTimeout(() => {
			if (mode === 'add') {
				providerNameInputRef.current?.focus();
			} else if (!isReadOnly) {
				displayNameInputRef.current?.focus();
			}
		}, 0);

		return () => {
			window.clearTimeout(focusTimer);
		};
	}, [isReadOnly, mode]);

	const validateField = useCallback(
		(
			field: keyof ProviderFormData,
			val: string | boolean | ProviderSDKType,
			currentErrors: ErrorState,
			originInput: HTMLInputElement | null = null
		): ErrorState => {
			if (isReadOnly) {
				return currentErrors;
			}

			let newErrs: ErrorState = { ...currentErrors };
			const v = typeof val === 'string' ? val.trim() : val;

			if (field === 'providerName' && mode === 'add') {
				if (!v) {
					newErrs.providerName = 'Provider name required.';
				} else if (typeof v === 'string' && !/^[\w-]+$/.test(v)) {
					newErrs.providerName = 'Letters, numbers, dash & underscore only.';
				} else if (typeof v === 'string' && existingProviderNames.includes(v)) {
					newErrs.providerName = 'Provider already exists.';
				} else {
					newErrs = omitManyKeys(newErrs, ['providerName']);
				}
			}

			if (field === 'displayName') {
				if (!v) {
					newErrs.displayName = 'Display name required.';
				} else {
					newErrs = omitManyKeys(newErrs, ['displayName']);
				}
			}

			if (field === 'origin') {
				const { error, normalized } = validateUrlForInput(String(val), originInput, { required: true });
				const securityError = normalized ? validateHTTPURLSecurity(normalized, 'Provider origin') : undefined;
				if (error || securityError) {
					newErrs.origin = error ?? securityError;
				} else {
					newErrs = omitManyKeys(newErrs, ['origin']);
				}
			}

			if (field === 'chatCompletionPathPrefix') {
				if (!String(v).trim()) {
					newErrs.chatCompletionPathPrefix = 'Chat path required.';
				} else if (!String(v).startsWith('/')) {
					newErrs.chatCompletionPathPrefix = 'Chat path must start with "/".';
				} else if (/[\r\n\u0000]/.test(String(v))) {
					newErrs.chatCompletionPathPrefix = 'Chat path must not contain control characters.';
				} else {
					newErrs = omitManyKeys(newErrs, ['chatCompletionPathPrefix']);
				}
			}

			if (field === 'apiKeyHeaderKey') {
				const headerNameError = String(v).trim() ? validateHTTPHeaderName(String(v), 'API-key header name') : undefined;
				if (headerNameError) {
					newErrs.apiKeyHeaderKey = headerNameError;
				} else {
					newErrs = omitManyKeys(newErrs, ['apiKeyHeaderKey']);
				}
			}

			if (field === 'defaultHeadersRawJSON') {
				if (v) {
					try {
						parseDefaultHeadersRawJSON(String(v));
						newErrs = omitManyKeys(newErrs, ['defaultHeadersRawJSON']);
					} catch (error) {
						newErrs.defaultHeadersRawJSON =
							error instanceof Error
								? error.message
								: 'Default headers must be a JSON object containing string values.';
					}
				} else {
					newErrs = omitManyKeys(newErrs, ['defaultHeadersRawJSON']);
				}
			}

			if (field === 'apiKey' && mode === 'add') {
				if (!v) {
					newErrs.apiKey = 'API key required.';
				} else {
					newErrs = omitManyKeys(newErrs, ['apiKey']);
				}
			}

			if (field === 'sdkType') {
				if (!Object.values(ProviderSDKType).includes(val as ProviderSDKType)) {
					newErrs.sdkType = 'Invalid SDK type.';
				} else {
					newErrs = omitManyKeys(newErrs, ['sdkType']);
				}
			}

			return newErrs;
		},
		[existingProviderNames, isReadOnly, mode]
	);

	const validateForm = useCallback(
		(state: ProviderFormData): ErrorState => {
			if (isReadOnly) {
				return {};
			}

			let next: ErrorState = {};
			next = validateField('providerName', state.providerName, next);
			next = validateField('displayName', state.displayName, next);
			next = validateField('origin', state.origin, next);
			next = validateField('chatCompletionPathPrefix', state.chatCompletionPathPrefix, next);
			next = validateField('apiKeyHeaderKey', state.apiKeyHeaderKey, next);
			next = validateField('defaultHeadersRawJSON', state.defaultHeadersRawJSON, next);
			if (mode === 'add' || state.apiKey.trim()) {
				next = validateField('apiKey', state.apiKey, next);
			}
			next = validateField('sdkType', state.sdkType, next);

			if (!next.defaultHeadersRawJSON) {
				try {
					restoreRedactedHTTPHeaders(
						parseDefaultHeadersRawJSON(state.defaultHeadersRawJSON),
						mode === 'edit' ? initialPreset?.defaultHeaders : undefined
					);
				} catch (error) {
					next.defaultHeadersRawJSON =
						error instanceof Error ? error.message : 'Sensitive default headers could not be resolved.';
				}
			}

			return next;
		},
		[initialPreset?.defaultHeaders, isReadOnly, mode, validateField]
	);

	const buildPatchPayload = useCallback(
		(
			state: ProviderFormData,
			normalizedOrigin: string,
			defaultHeaders: Record<string, string>
		): PatchProviderPresetPayload => {
			if (!initialPreset) {
				return {};
			}

			const patch: PatchProviderPresetPayload = {};
			if (state.displayName.trim() !== initialPreset.displayName) {
				patch.displayName = state.displayName.trim();
			}
			if (state.sdkType !== initialPreset.sdkType) {
				patch.sdkType = state.sdkType;
			}
			if (state.isEnabled !== initialPreset.isEnabled) {
				patch.isEnabled = state.isEnabled;
			}
			if (normalizedOrigin !== initialPreset.origin) {
				patch.origin = normalizedOrigin;
			}
			if (state.chatCompletionPathPrefix.trim() !== initialPreset.chatCompletionPathPrefix) {
				patch.chatCompletionPathPrefix = state.chatCompletionPathPrefix.trim();
			}
			if (state.apiKeyHeaderKey.trim() !== initialPreset.apiKeyHeaderKey) {
				patch.apiKeyHeaderKey = state.apiKeyHeaderKey.trim();
			}
			if (!headersEqual(defaultHeaders, initialPreset.defaultHeaders ?? {})) {
				patch.defaultHeaders = defaultHeaders;
			}
			return patch;
		},
		[initialPreset]
	);

	const applyPrefill = (key: ProviderName) => {
		const src = allProviderPresets[key];
		if (!src) {
			return;
		}

		const next: ProviderFormData = {
			...formData,
			displayName: `${src.displayName}-${GenerateRandomNumberString(3)}`,
			sdkType: src.sdkType,
			isEnabled: true,
			origin: src.origin,
			chatCompletionPathPrefix: src.chatCompletionPathPrefix,
			apiKeyHeaderKey: src.apiKeyHeaderKey,
			defaultHeadersRawJSON: JSON.stringify(omitSensitiveHTTPHeaders(src.defaultHeaders) ?? {}, null, 2),
			apiKey: '',
		};

		setFormData(next);
		setErrors(validateForm(next));
	};

	const handleInput = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
		if (isSubmitting) {
			return;
		}

		const target = e.target as HTMLInputElement;
		const { name, value, type, checked } = target;
		const newVal = type === 'checkbox' ? checked : value;

		setFormData(prev => ({ ...prev, [name]: newVal }));

		if (
			['providerName', 'displayName', 'origin', 'chatCompletionPathPrefix', 'defaultHeadersRawJSON', 'apiKey'].includes(
				name
			)
		) {
			setErrors(prev =>
				validateField(name as keyof ProviderFormData, newVal, prev, name === 'origin' ? originInputRef.current : null)
			);
		}
	};

	const onSdkTypeChange = (key: ProviderSDKType) => {
		if (isReadOnly || isSubmitting) {
			return;
		}

		const defaults = SDK_DEFAULTS[key];
		const previousDefaults = SDK_DEFAULTS[formData.sdkType];
		const shouldReplacePath =
			!formData.chatCompletionPathPrefix.trim() ||
			formData.chatCompletionPathPrefix.trim() === previousDefaults.chatPath;
		const shouldReplaceHeaderKey =
			!formData.apiKeyHeaderKey.trim() || formData.apiKeyHeaderKey.trim() === previousDefaults.apiKeyHeaderKey;
		const shouldReplaceHeaders = usesSDKDefaultHeaders(formData.defaultHeadersRawJSON, formData.sdkType);
		const next: ProviderFormData = {
			...formData,
			sdkType: key,
			chatCompletionPathPrefix: shouldReplacePath ? defaults.chatPath : formData.chatCompletionPathPrefix,
			apiKeyHeaderKey: shouldReplaceHeaderKey ? defaults.apiKeyHeaderKey : formData.apiKeyHeaderKey,
			defaultHeadersRawJSON: shouldReplaceHeaders
				? JSON.stringify(defaults.defaultHeaders, null, 2)
				: formData.defaultHeadersRawJSON,
		};

		setFormData(next);
		setErrors(prev => validateField('sdkType', key, prev));
	};

	const hasEffectiveChanges = useMemo(() => {
		if (mode !== 'edit') {
			return true;
		}
		if (formData.apiKey.trim()) {
			return true;
		}
		if (!initialPreset) {
			return false;
		}

		let defaultHeaders: Record<string, string>;
		try {
			defaultHeaders = restoreRedactedHTTPHeaders(
				parseDefaultHeadersRawJSON(formData.defaultHeadersRawJSON),
				mode === 'edit' ? initialPreset?.defaultHeaders : undefined
			);
		} catch {
			return true;
		}

		const { normalized } = validateUrlForInput(formData.origin, null, { required: true });
		const normalizedOrigin = normalized ?? formData.origin.trim();

		return Object.keys(buildPatchPayload(formData, normalizedOrigin, defaultHeaders)).length > 0;
	}, [buildPatchPayload, formData, initialPreset, mode]);

	const allValid = useMemo(() => {
		if (isReadOnly) {
			return true;
		}

		const validationErrors = validateForm(formData);
		const hasErr = Object.values(validationErrors).some(Boolean);
		const requiredFilled =
			formData.providerName.trim() &&
			formData.displayName.trim() &&
			formData.origin.trim() &&
			formData.chatCompletionPathPrefix.trim() &&
			(mode === 'add' ? formData.apiKey.trim() : true);

		return !hasErr && Boolean(requiredFilled);
	}, [formData, isReadOnly, mode, validateForm]);

	const submitForm = async () => {
		if (isReadOnly) {
			requestClose();
			return;
		}

		setSubmitError('');
		const finalErrors = validateForm(formData);
		setErrors(finalErrors);
		if (Object.values(finalErrors).some(Boolean)) {
			return;
		}

		const originInput = originInputRef.current;
		const { normalized: normalizedOrigin, error: originError } = validateUrlForInput(formData.origin, originInput, {
			required: true,
		});

		if (!normalizedOrigin || originError) {
			setErrors(prev => ({ ...prev, origin: originError ?? MessageEnterValidURL }));
			originInput?.focus();
			return;
		}

		let defaultHeaders: Record<string, string>;
		try {
			defaultHeaders = restoreRedactedHTTPHeaders(
				parseDefaultHeadersRawJSON(formData.defaultHeadersRawJSON),
				mode === 'edit' ? initialPreset?.defaultHeaders : undefined
			);
		} catch (error) {
			setErrors(previous => ({
				...previous,
				defaultHeadersRawJSON: error instanceof Error ? error.message : 'Default headers could not be resolved.',
			}));
			return;
		}

		const providerName = formData.providerName.trim();
		const apiKey = formData.apiKey.trim() || null;
		const trimmedApiKeyHeaderKey = formData.apiKeyHeaderKey.trim();
		const trimmedChatPath = formData.chatCompletionPathPrefix.trim();

		const payload: PostProviderPresetPayload | PatchProviderPresetPayload =
			mode === 'add'
				? {
						displayName: formData.displayName.trim(),
						sdkType: formData.sdkType,
						isEnabled: formData.isEnabled,
						origin: normalizedOrigin,
						chatCompletionPathPrefix: trimmedChatPath,
						...(trimmedApiKeyHeaderKey && { apiKeyHeaderKey: trimmedApiKeyHeaderKey }),
						...(Object.keys(defaultHeaders).length > 0 && { defaultHeaders }),
					}
				: buildPatchPayload(formData, normalizedOrigin, defaultHeaders);

		if (mode === 'edit' && Object.keys(payload).length === 0 && !apiKey) {
			requestClose();
			return;
		}

		setIsSubmitting(true);
		try {
			await onSubmit(providerName, payload, apiKey);
			requestClose(true);
		} catch (error) {
			if (!unmountingRef.current) {
				setSubmitError(error instanceof Error && error.message.trim() ? error.message : 'Failed to save provider.');
			}
		} finally {
			if (!unmountingRef.current) {
				setIsSubmitting(false);
			}
		}
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();
		e.stopPropagation();
		void submitForm();
	};

	const title = mode === 'add' ? 'Add Provider' : mode === 'edit' ? 'Edit Provider' : 'View Provider';

	return (
		<dialog ref={dialogRef} className="modal" onClose={handleClose} onCancel={handleCancel}>
			<div className="modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-4xl flex-col overflow-hidden rounded-2xl p-0">
				<ModalHeader
					title={title}
					description={
						isReadOnly
							? 'Inspect provider compatibility and connection settings.'
							: 'Configure provider compatibility, endpoints, defaults, and credentials.'
					}
					onClose={() => {
						requestClose();
					}}
					closeDisabled={isSubmitting}
				/>

				<form noValidate onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col" aria-busy={isSubmitting}>
					<div
						className={`app-scrollbar-thin min-h-0 flex-1 overflow-y-auto p-4 sm:p-6 ${MANAGEMENT_MODAL_FORM_CLASS}`}
					>
						{submitError ? (
							<div className="alert alert-error rounded-2xl text-sm" role="alert">
								<div className="flex items-center gap-2">
									<FiAlertCircle size={14} />
									<span className="wrap-break-word">{submitError}</span>
								</div>
							</div>
						) : null}

						{mode === 'edit' && !isReadOnly && (
							<div className="border-info/30 bg-info/10 rounded-xl border px-3 py-2 text-xs">
								Only changed provider fields are sent while editing.
								<br />
								Leaving API-Key blank keeps the current stored secret.
							</div>
						)}

						{mode === 'add' && (
							<ModalSection
								title="Copy an existing provider"
								description="Copy endpoint and compatibility settings. API keys are never copied."
							>
								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="text-sm">Prefill from Existing</span>
									</label>

									<div className="col-span-9 flex items-center gap-2">
										{!prefillMode && (
											<button
												type="button"
												className="btn btn-sm btn-ghost flex items-center rounded-xl"
												onClick={() => {
													setPrefillMode(true);
												}}
												disabled={isSubmitting || prefillKeys.length === 0}
												title={prefillKeys.length === 0 ? 'No existing providers are available to copy.' : undefined}
											>
												<FiUpload size={14} />
												<span className="ml-1">Copy Existing Provider</span>
											</button>
										)}

										{prefillMode && (
											<>
												<Dropdown<ProviderName>
													dropdownItems={prefillDropdownItems}
													orderedKeys={prefillKeys}
													selectedKey={selectedPrefillKey ?? ('' as ProviderName)}
													onChange={key => {
														setSelectedPrefillKey(key);
														applyPrefill(key);
														setPrefillMode(false);
													}}
													disabled={prefillKeys.length === 0}
													filterDisabled={false}
													title="Select provider to copy"
													getDisplayName={k => prefillDropdownItems[k]?.displayName ?? 'Select provider to copy'}
												/>
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													onClick={() => {
														setPrefillMode(false);
														setSelectedPrefillKey(null);
													}}
													title="Cancel prefill"
													disabled={isSubmitting}
												>
													<FiX size={12} />
												</button>
											</>
										)}
									</div>
								</div>
							</ModalSection>
						)}

						<ModalSection
							title="Identity and compatibility"
							description="Choose the compatible SDK before configuring the provider endpoint."
						>
							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="text-sm">SDK Type*</span>
									<span
										className="tooltip tooltip-right"
										data-tip="Select the backend SDK/API compatibility for this provider."
									>
										<FiHelpCircle size={12} />
									</span>
								</label>
								<div className="col-span-9">
									{isReadOnly ? (
										<ReadOnlyValue value={SDK_DISPLAY_NAME[formData.sdkType]} />
									) : (
										<Dropdown<ProviderSDKType>
											dropdownItems={sdkDropdownItems}
											selectedKey={formData.sdkType}
											onChange={onSdkTypeChange}
											filterDisabled={false}
											title="Select SDK Type"
											getDisplayName={k => sdkDropdownItems[k].displayName}
										/>
									)}
									{errors.sdkType && (
										<div className="label">
											<span className="text-error flex items-center gap-1">
												<FiAlertCircle size={12} /> {errors.sdkType}
											</span>
										</div>
									)}
								</div>
							</div>

							<ModalField
								label="Provider ID"
								htmlFor="provider-id"
								required
								hint={mode === 'add' ? 'Unique identifier using letters, numbers, dashes, and underscores.' : undefined}
								error={errors.providerName}
							>
								<input
									id="provider-id"
									ref={providerNameInputRef}
									type="text"
									name="providerName"
									value={formData.providerName}
									onChange={handleInput}
									className={`input w-full rounded-xl ${errors.providerName ? 'input-error' : ''}`}
									readOnly={mode !== 'add'}
									spellCheck="false"
									autoComplete="off"
									disabled={isSubmitting}
								/>
							</ModalField>

							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="text-sm">Display Name*</span>
								</label>
								<div className="col-span-9">
									<input
										ref={displayNameInputRef}
										type="text"
										name="displayName"
										value={formData.displayName}
										onChange={handleInput}
										className={`input w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
										spellCheck="false"
										autoComplete="off"
										readOnly={isReadOnly}
										disabled={isSubmitting}
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
						</ModalSection>

						<ModalSection
							title="Connection"
							description="Configure the provider origin, chat endpoint path, and stable request headers."
						>
							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="text-sm">Origin*</span>
								</label>
								<div className="col-span-9">
									<input
										ref={originInputRef}
										type="url"
										name="origin"
										value={formData.origin}
										onChange={handleInput}
										className={`input w-full rounded-xl ${errors.origin ? 'input-error' : ''}`}
										spellCheck="false"
										autoComplete="off"
										placeholder="https://api.example.com OR api.example.com"
										readOnly={isReadOnly}
										disabled={isSubmitting}
									/>
									{errors.origin && (
										<div className="label">
											<span className="text-error flex items-center gap-1">
												<FiAlertCircle size={12} /> {errors.origin}
											</span>
										</div>
									)}
								</div>
							</div>

							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="text-sm">Chat Path*</span>
									<span className="tooltip tooltip-right" data-tip="Endpoint path for chat completions.">
										<FiHelpCircle size={12} />
									</span>
								</label>
								<div className="col-span-9">
									<input
										type="text"
										name="chatCompletionPathPrefix"
										value={formData.chatCompletionPathPrefix}
										onChange={handleInput}
										className="input w-full rounded-xl"
										spellCheck="false"
										autoComplete="off"
										readOnly={isReadOnly}
										disabled={isSubmitting}
									/>
									{errors.chatCompletionPathPrefix && (
										<div className="label">
											<span className="text-error flex items-center gap-1">
												<FiAlertCircle size={12} /> {errors.chatCompletionPathPrefix}
											</span>
										</div>
									)}
								</div>
							</div>

							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="text-sm">API-Key Header Key</span>
								</label>
								<div className="col-span-9">
									<input
										type="text"
										name="apiKeyHeaderKey"
										value={formData.apiKeyHeaderKey}
										onChange={handleInput}
										className={`input w-full rounded-xl ${errors.apiKeyHeaderKey ? 'input-error' : ''}`}
										spellCheck="false"
										autoComplete="off"
										readOnly={isReadOnly}
										disabled={isSubmitting}
										aria-invalid={Boolean(errors.apiKeyHeaderKey)}
									/>
									{errors.apiKeyHeaderKey ? (
										<div className="label">
											<span className="text-error flex items-center gap-1">
												<FiAlertCircle size={12} /> {errors.apiKeyHeaderKey}
											</span>
										</div>
									) : null}
								</div>
							</div>

							<div className="grid grid-cols-12 items-start gap-2">
								<label className="label col-span-3">
									<span className="text-sm">Default Headers (JSON)</span>
								</label>
								<div className="col-span-9">
									<textarea
										name="defaultHeadersRawJSON"
										value={formData.defaultHeadersRawJSON}
										onChange={handleInput}
										className={`textarea h-24 w-full rounded-xl ${errors.defaultHeadersRawJSON ? 'textarea-error' : ''}`}
										spellCheck="false"
										readOnly={isReadOnly}
										disabled={isSubmitting}
									/>
									{errors.defaultHeadersRawJSON && (
										<div className="label">
											<span className="text-error flex items-center gap-1">
												<FiAlertCircle size={12} /> {errors.defaultHeadersRawJSON}
											</span>
										</div>
									)}
								</div>
							</div>
						</ModalSection>

						<ModalSection
							title="Security and availability"
							description="Provider API keys remain write-only. Leaving an existing key blank preserves it."
						>
							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3 flex flex-col items-start gap-0.5">
									<span className="text-sm">API-Key*</span>
									{(mode === 'edit' || mode === 'view') && apiKeyAlreadySet && (
										<span className="text-xs">
											{mode === 'view' ? '(managed separately; not shown)' : '(leave blank to keep current)'}
										</span>
									)}
								</label>
								<div className="col-span-9">
									<input
										type="password"
										name="apiKey"
										value={formData.apiKey}
										onChange={handleInput}
										className={`input w-full rounded-xl ${errors.apiKey ? 'input-error' : ''}`}
										placeholder={(mode === 'edit' || mode === 'view') && apiKeyAlreadySet ? '********' : ''}
										spellCheck="false"
										autoComplete="new-password"
										readOnly={isReadOnly}
										disabled={isSubmitting}
									/>
									{errors.apiKey && (
										<div className="label">
											<span className="text-error flex items-center gap-1">
												<FiAlertCircle size={12} /> {errors.apiKey}
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
										name="isEnabled"
										checked={formData.isEnabled}
										onChange={handleInput}
										className="toggle toggle-accent"
										disabled={isReadOnly || isSubmitting}
									/>
								</div>
							</div>
						</ModalSection>

						{isReadOnly && initialPreset ? (
							<ModalSection title="Metadata">
								<ManagementInfoGrid>
									<ManagementInfoRow label="Provider ID" mono>
										{initialPreset.name}
									</ManagementInfoRow>
									<ManagementInfoRow label="Built-in">{initialPreset.isBuiltIn ? 'Yes' : 'No'}</ManagementInfoRow>
									<ManagementInfoRow label="Default model" mono>
										{initialPreset.defaultModelPresetID || '—'}
									</ManagementInfoRow>
									<ManagementInfoRow label="Created">{initialPreset.createdAt}</ManagementInfoRow>
									<ManagementInfoRow label="Modified">{initialPreset.modifiedAt}</ManagementInfoRow>
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
							{isReadOnly ? 'Close' : 'Cancel'}
						</button>

						{!isReadOnly && (
							<button
								type="submit"
								className="btn btn-primary rounded-xl"
								disabled={!allValid || (mode === 'edit' && !hasEffectiveChanges) || isSubmitting}
							>
								{isSubmitting ? 'Saving…' : mode === 'add' ? 'Add Provider' : 'Save'}
							</button>
						)}
					</ModalActions>
				</form>
			</div>
			<ModalBackdrop enabled={isReadOnly} />
		</dialog>
	);
}

export function AddEditProviderPresetModal(props: AddEditProviderPresetModalProps) {
	if (!props.isOpen) {
		return null;
	}
	if (typeof document === 'undefined' || !document.body) {
		return null;
	}

	const modalKey =
		props.mode === 'add'
			? 'add-provider'
			: `${props.mode}:${props.initialPreset?.name ?? 'provider-without-name'}:${
					props.initialPreset?.modifiedAt ?? 'unknown-modified'
				}`;

	return createPortal(<AddEditProviderPresetModalContent key={modalKey} {...props} />, document.body);
}
