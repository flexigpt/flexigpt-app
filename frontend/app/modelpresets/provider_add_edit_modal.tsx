import type { ChangeEvent, SubmitEventHandler } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiUpload, FiX } from 'react-icons/fi';

import { type ProviderName, ProviderSDKType, SDK_DEFAULTS, SDK_DISPLAY_NAME } from '@/spec/inference';
import type { PatchProviderPresetPayload, PostProviderPresetPayload, ProviderPreset } from '@/spec/modelpreset';

import { GenerateRandomNumberString } from '@/lib/encode_decode';
import { omitManyKeys } from '@/lib/obj_utils';
import { MessageEnterValidURL, validateUrlForInput } from '@/lib/url_utils';

import { Dropdown } from '@/components/dropdown';
import { ModalBackdrop } from '@/components/modal_backdrop';
import { ReadOnlyValue } from '@/components/read_only_value';

type ModalMode = 'add' | 'edit' | 'view';

function parseDefaultHeadersRawJSON(raw: string): Record<string, string> {
	if (!raw.trim()) return {};
	return JSON.parse(raw.trim()) as Record<string, string>;
}

function normalizeHeadersRecord(headers: Record<string, string>): Record<string, string> {
	return Object.fromEntries(Object.entries(headers).sort(([a], [b]) => a.localeCompare(b)));
}

function headersEqual(a: Record<string, string>, b: Record<string, string>): boolean {
	return JSON.stringify(normalizeHeadersRecord(a)) === JSON.stringify(normalizeHeadersRecord(b));
}

type ProviderFormData = {
	providerName: string;
	displayName: string;
	sdkType: ProviderSDKType;
	isEnabled: boolean;
	origin: string;
	chatCompletionPathPrefix: string;
	apiKeyHeaderKey: string;
	defaultHeadersRawJSON: string;
	apiKey: string;
};

const DEFAULT_FORM: ProviderFormData = {
	providerName: '',
	displayName: '',
	sdkType: ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions,
	isEnabled: true,
	origin: '',
	chatCompletionPathPrefix: '/v1/chat/completions',
	apiKeyHeaderKey: 'Authorization',
	defaultHeadersRawJSON: '',
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
			defaultHeadersRawJSON: JSON.stringify(initialPreset.defaultHeaders ?? {}, null, 2),
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

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const providerNameInputRef = useRef<HTMLInputElement | null>(null);
	const displayNameInputRef = useRef<HTMLInputElement | null>(null);
	const originInputRef = useRef<HTMLInputElement | null>(null);
	const isUnmountingRef = useRef(false);

	const prefillDropdownItems: Record<ProviderName, { isEnabled: boolean; displayName: string }> = useMemo(() => {
		const out = {} as Record<ProviderName, { isEnabled: boolean; displayName: string }>;

		for (const [name, preset] of Object.entries(allProviderPresets)) {
			out[name] = { isEnabled: true, displayName: preset.displayName || name };
		}

		return out;
	}, [allProviderPresets]);

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
		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			try {
				dialog.showModal();
			} catch {
				// Ignore showModal errors and keep rendering safely.
			}
		}

		const focusTimer = window.setTimeout(() => {
			if (mode === 'add') {
				providerNameInputRef.current?.focus();
			} else if (!isReadOnly) {
				displayNameInputRef.current?.focus();
			}
		}, 0);

		return () => {
			isUnmountingRef.current = true;
			window.clearTimeout(focusTimer);

			if (dialog.open) {
				dialog.close();
			}
		};
	}, [isReadOnly, mode]);

	const requestClose = useCallback(() => {
		if (isSubmitting) return;

		const dialog = dialogRef.current;

		if (dialog?.open) {
			dialog.close();
			return;
		}

		onClose();
	}, [isSubmitting, onClose]);

	const handleDialogClose = () => {
		if (isUnmountingRef.current) return;
		onClose();
	};

	const validateField = useCallback(
		(
			field: keyof ProviderFormData,
			val: string | boolean | ProviderSDKType,
			currentErrors: ErrorState,
			originInput: HTMLInputElement | null = null
		): ErrorState => {
			if (isReadOnly) return currentErrors;

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
				if (!v) newErrs.displayName = 'Display name required.';
				else newErrs = omitManyKeys(newErrs, ['displayName']);
			}

			if (field === 'origin') {
				const { error } = validateUrlForInput(String(val), originInput, { required: true });
				if (error) newErrs.origin = error;
				else newErrs = omitManyKeys(newErrs, ['origin']);
			}

			if (field === 'chatCompletionPathPrefix') {
				if (!String(v).trim()) newErrs.chatCompletionPathPrefix = 'Chat path required.';
				else newErrs = omitManyKeys(newErrs, ['chatCompletionPathPrefix']);
			}

			if (field === 'defaultHeadersRawJSON') {
				if (v) {
					try {
						JSON.parse(String(v));
						newErrs = omitManyKeys(newErrs, ['defaultHeadersRawJSON']);
					} catch {
						newErrs.defaultHeadersRawJSON = 'Invalid JSON.';
					}
				} else {
					newErrs = omitManyKeys(newErrs, ['defaultHeadersRawJSON']);
				}
			}

			if (field === 'apiKey' && mode === 'add') {
				if (!v) newErrs.apiKey = 'API key required.';
				else newErrs = omitManyKeys(newErrs, ['apiKey']);
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
			if (isReadOnly) return {};

			let next: ErrorState = {};
			next = validateField('providerName', state.providerName, next);
			next = validateField('displayName', state.displayName, next);
			next = validateField('origin', state.origin, next);
			next = validateField('chatCompletionPathPrefix', state.chatCompletionPathPrefix, next);
			next = validateField('defaultHeadersRawJSON', state.defaultHeadersRawJSON, next);
			if (mode === 'add' || state.apiKey.trim()) next = validateField('apiKey', state.apiKey, next);
			next = validateField('sdkType', state.sdkType, next);
			return next;
		},
		[isReadOnly, mode, validateField]
	);

	const buildPatchPayload = useCallback(
		(
			state: ProviderFormData,
			normalizedOrigin: string,
			defaultHeaders: Record<string, string>
		): PatchProviderPresetPayload => {
			if (!initialPreset) return {};

			const patch: PatchProviderPresetPayload = {};
			if (state.displayName.trim() !== initialPreset.displayName) patch.displayName = state.displayName.trim();
			if (state.sdkType !== initialPreset.sdkType) patch.sdkType = state.sdkType;
			if (state.isEnabled !== initialPreset.isEnabled) patch.isEnabled = state.isEnabled;
			if (normalizedOrigin !== initialPreset.origin) patch.origin = normalizedOrigin;
			if (state.chatCompletionPathPrefix.trim() !== initialPreset.chatCompletionPathPrefix)
				patch.chatCompletionPathPrefix = state.chatCompletionPathPrefix.trim();
			if (state.apiKeyHeaderKey.trim() !== initialPreset.apiKeyHeaderKey)
				patch.apiKeyHeaderKey = state.apiKeyHeaderKey.trim();
			if (!headersEqual(defaultHeaders, initialPreset.defaultHeaders ?? {})) patch.defaultHeaders = defaultHeaders;
			return patch;
		},
		[initialPreset]
	);

	const applyPrefill = (key: ProviderName) => {
		const src = allProviderPresets[key];
		if (!src) return;

		const next: ProviderFormData = {
			...formData,
			displayName: `${src.displayName}-${GenerateRandomNumberString(3)}`,
			sdkType: src.sdkType,
			isEnabled: true,
			origin: src.origin,
			chatCompletionPathPrefix: src.chatCompletionPathPrefix,
			apiKeyHeaderKey: src.apiKeyHeaderKey,
			defaultHeadersRawJSON: JSON.stringify(src.defaultHeaders, null, 2),
			apiKey: '',
		};

		setFormData(next);
		setErrors(validateForm(next));
	};

	const handleInput = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
		if (isSubmitting) return;

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
		if (isReadOnly || isSubmitting) return;

		const defaults = SDK_DEFAULTS[key];
		const next: ProviderFormData = {
			...formData,
			sdkType: key,
			chatCompletionPathPrefix: formData.chatCompletionPathPrefix.trim()
				? formData.chatCompletionPathPrefix
				: defaults.chatPath,
			apiKeyHeaderKey: formData.apiKeyHeaderKey.trim() ? formData.apiKeyHeaderKey : defaults.apiKeyHeaderKey,
			defaultHeadersRawJSON: formData.defaultHeadersRawJSON.trim()
				? formData.defaultHeadersRawJSON
				: JSON.stringify(defaults.defaultHeaders, null, 2),
		};

		setFormData(next);
		setErrors(prev => validateField('sdkType', key, prev));
	};

	const hasEffectiveChanges = useMemo(() => {
		if (mode !== 'edit') return true;
		if (formData.apiKey.trim()) return true;
		if (!initialPreset) return false;

		let defaultHeaders: Record<string, string>;
		try {
			defaultHeaders = parseDefaultHeadersRawJSON(formData.defaultHeadersRawJSON);
		} catch {
			return true;
		}

		const { normalized } = validateUrlForInput(formData.origin, originInputRef.current, { required: true });
		const normalizedOrigin = normalized ?? formData.origin.trim();

		return Object.keys(buildPatchPayload(formData, normalizedOrigin, defaultHeaders)).length > 0;
	}, [buildPatchPayload, formData, initialPreset, mode]);

	const allValid = useMemo(() => {
		if (isReadOnly) return true;

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

		const finalErrors = validateForm(formData);
		setErrors(finalErrors);
		if (Object.values(finalErrors).some(Boolean)) return;

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
			defaultHeaders = parseDefaultHeadersRawJSON(formData.defaultHeadersRawJSON);
		} catch {
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
			requestClose();
		} catch {
			// Keep modal open so the user keeps their form on failed save.
		} finally {
			setIsSubmitting(false);
		}
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();
		e.stopPropagation();
		void submitForm();
	};

	const title = mode === 'add' ? 'Add Provider' : mode === 'edit' ? 'Edit Provider' : 'View Provider';

	return (
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={e => {
				if (!isReadOnly || isSubmitting) e.preventDefault();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-4/5 overflow-hidden rounded-2xl p-0 xl:max-w-3/5">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between">
						<h3 className="text-lg font-bold">{title}</h3>
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

					<form noValidate onSubmit={handleSubmit} className="space-y-4">
						{mode === 'edit' && !isReadOnly && (
							<div className="border-info/30 bg-info/10 rounded-xl border px-3 py-2 text-xs">
								Only changed provider fields are sent while editing.
								<br />
								Leaving API-Key blank keeps the current stored secret.
							</div>
						)}

						{mode === 'add' && (
							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="label-text text-sm">Prefill from Existing</span>
								</label>

								<div className="col-span-9 flex items-center gap-2">
									{!prefillMode && (
										<button
											type="button"
											className="btn btn-sm btn-ghost flex items-center rounded-xl"
											onClick={() => {
												setPrefillMode(true);
											}}
											disabled={isSubmitting}
										>
											<FiUpload size={14} />
											<span className="ml-1">Copy Existing Provider</span>
										</button>
									)}

									{prefillMode && (
										<>
											<Dropdown<ProviderName>
												dropdownItems={prefillDropdownItems}
												selectedKey={selectedPrefillKey ?? ('' as ProviderName)}
												onChange={key => {
													setSelectedPrefillKey(key);
													applyPrefill(key);
													setPrefillMode(false);
												}}
												filterDisabled={false}
												title="Select provider to copy"
												getDisplayName={k => prefillDropdownItems[k].displayName}
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
						)}

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">SDK Type*</span>
								<span
									className="label-text-alt tooltip tooltip-right"
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
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.sdkType}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Provider ID*</span>
								{mode === 'add' && (
									<span
										className="label-text-alt tooltip tooltip-right"
										data-tip="Unique identifier (letters, numbers, dash, underscore)."
									>
										<FiHelpCircle size={12} />
									</span>
								)}
							</label>
							<div className="col-span-9">
								<input
									ref={providerNameInputRef}
									type="text"
									name="providerName"
									value={formData.providerName}
									onChange={handleInput}
									className={`input input-bordered w-full rounded-xl ${errors.providerName ? 'input-error' : ''}`}
									readOnly={mode !== 'add'}
									spellCheck="false"
									autoComplete="off"
									disabled={isSubmitting}
								/>
								{errors.providerName && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.providerName}
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
									ref={displayNameInputRef}
									type="text"
									name="displayName"
									value={formData.displayName}
									onChange={handleInput}
									className={`input input-bordered w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									readOnly={isReadOnly}
									disabled={isSubmitting}
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
							<label className="label col-span-3">
								<span className="label-text text-sm">Origin*</span>
							</label>
							<div className="col-span-9">
								<input
									ref={originInputRef}
									type="url"
									name="origin"
									value={formData.origin}
									onChange={handleInput}
									className={`input input-bordered w-full rounded-xl ${errors.origin ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									placeholder="https://api.example.com OR api.example.com"
									readOnly={isReadOnly}
									disabled={isSubmitting}
								/>
								{errors.origin && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.origin}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Chat Path*</span>
								<span className="label-text-alt tooltip tooltip-right" data-tip="Endpoint path for chat completions.">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="chatCompletionPathPrefix"
									value={formData.chatCompletionPathPrefix}
									onChange={handleInput}
									className="input input-bordered w-full rounded-xl"
									spellCheck="false"
									autoComplete="off"
									readOnly={isReadOnly}
									disabled={isSubmitting}
								/>
								{errors.chatCompletionPathPrefix && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.chatCompletionPathPrefix}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">API-Key Header Key</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="apiKeyHeaderKey"
									value={formData.apiKeyHeaderKey}
									onChange={handleInput}
									className="input input-bordered w-full rounded-xl"
									spellCheck="false"
									autoComplete="off"
									readOnly={isReadOnly}
									disabled={isSubmitting}
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-start gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Default Headers (JSON)</span>
							</label>
							<div className="col-span-9">
								<textarea
									name="defaultHeadersRawJSON"
									value={formData.defaultHeadersRawJSON}
									onChange={handleInput}
									className={`textarea textarea-bordered h-24 w-full rounded-xl ${
										errors.defaultHeadersRawJSON ? 'textarea-error' : ''
									}`}
									spellCheck="false"
									readOnly={isReadOnly}
									disabled={isSubmitting}
								/>
								{errors.defaultHeadersRawJSON && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.defaultHeadersRawJSON}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3 flex flex-col items-start gap-0.5">
								<span className="label-text text-sm">API-Key*</span>
								{(mode === 'edit' || mode === 'view') && apiKeyAlreadySet && (
									<span className="label-text-alt text-xs">
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
									className={`input input-bordered w-full rounded-xl ${errors.apiKey ? 'input-error' : ''}`}
									placeholder={(mode === 'edit' || mode === 'view') && apiKeyAlreadySet ? '********' : ''}
									spellCheck="false"
									autoComplete="off"
									readOnly={isReadOnly}
									disabled={isSubmitting}
								/>
								{errors.apiKey && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.apiKey}
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
									name="isEnabled"
									checked={formData.isEnabled}
									onChange={handleInput}
									className="toggle toggle-accent"
									disabled={isReadOnly || isSubmitting}
								/>
							</div>
						</div>

						<div className="modal-action">
							<button
								type="button"
								className="btn bg-base-300 rounded-xl"
								onClick={requestClose}
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
						</div>
					</form>
				</div>
			</div>
			<ModalBackdrop enabled={isReadOnly} />
		</dialog>
	);
}

export function AddEditProviderPresetModal(props: AddEditProviderPresetModalProps) {
	if (!props.isOpen) return null;
	if (typeof document === 'undefined' || !document.body) return null;

	const modalKey =
		props.mode === 'add' ? 'add-provider' : `${props.mode}:${props.initialPreset?.name ?? 'provider-without-name'}`;

	return createPortal(<AddEditProviderPresetModalContent key={modalKey} {...props} />, document.body);
}
