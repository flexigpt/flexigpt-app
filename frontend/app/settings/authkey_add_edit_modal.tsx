import type { ChangeEvent, ReactNode, SubmitEventHandler } from 'react';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { FiAlertCircle } from 'react-icons/fi';

import type { ProviderName } from '@/spec/inference';
import type { ProviderPreset } from '@/spec/modelpreset';
import type { AuthKeyMeta } from '@/spec/setting';
import { AuthKeyTypeProvider } from '@/spec/setting';

import { omitManyKeys } from '@/lib/obj_utils';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { aggregateAPI } from '@/apis/baseapi';
import { getAllProviderPresetsMap } from '@/apis/list_helper';

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

/* ────────────────────────── props & helpers ────────────────────────── */
interface AddEditAuthKeyModalProps {
	isOpen: boolean;
	initial: AuthKeyMeta | null; // “edit” when NOT null
	existing: AuthKeyMeta[]; // list of existing keys
	onClose: () => void;
	onChanged: () => void; // parent should refetch on success
	// In pure "add" mode (initial === null) you can still send a pre-selected (type, keyName).
	// These fields will be rendered already filled-in and read-only,
	// exactly like in edit-mode, but the entry will still be created.
	prefill?: { type: string; keyName: string } | null;
	// Provider-only mode is useful for first-run onboarding.
	// It hides the generic key type selector, shows the provider dropdown, and writes provider auth keys.
	providerOnly?: boolean;
	// Optional default provider to preselect in provider-only mode.
	defaultKeyName?: string | null;
	// Optional explanatory content shown near the top of the modal.
	intro?: ReactNode;
}

const sentinelAddNew = '__add_new__';

interface FormData {
	type: string; // existing type | sentinelAddNew
	keyName: string; // provider name | free string
	secret: string; // the secret the user types
	newType: string; // only when sentinelAddNew chosen
}

type FormErrors = Partial<Record<keyof FormData, string>>;

function getInitialFormData(
	initial: AuthKeyMeta | null,
	prefill: { type: string; keyName: string } | null,
	defaultKeyName: string | null
): FormData {
	return {
		type: initial?.type ?? prefill?.type ?? AuthKeyTypeProvider,
		keyName: initial?.keyName ?? prefill?.keyName ?? defaultKeyName ?? '',
		secret: '',
		newType: '',
	};
}

function AddEditAuthKeyModalContent({
	initial,
	existing,

	onChanged,
	prefill = null,
	providerOnly = false,
	defaultKeyName = null,
	intro = null,
	onBusyChange,
}: Omit<AddEditAuthKeyModalProps, 'isOpen' | 'onClose'> & { onBusyChange?: (isBusy: boolean) => void }) {
	const isEdit = Boolean(initial); // “edit” = we already have that record
	const isPrefilled = !isEdit && !!prefill; // “add”, but (type,keyName) should be fixed
	const isReadOnly = isEdit || isPrefilled; // helper for rendering

	const modalTitle = providerOnly
		? initial?.nonEmpty
			? 'Update Provider API Key'
			: 'Add Provider API Key'
		: isEdit
			? 'Edit Auth Key'
			: 'Add Auth Key';

	const submitLabel = isEdit && initial?.nonEmpty ? 'Update' : 'Add';

	const [formData, setFormData] = useState<FormData>(() => getInitialFormData(initial, prefill, defaultKeyName));
	const [errors, setErrors] = useState<FormErrors>({});
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	/* raw provider presets fetched from backend */
	const [providerPresets, setProviderPresets] = useState<Record<ProviderName, ProviderPreset>>({});

	const { requestClose, unmountingRef } = useModalDialogController();
	useEffect(() => {
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-pass-live-state-to-parent
		onBusyChange?.(isSubmitting);
		return () => {
			onBusyChange?.(false);
		};
	}, [isSubmitting, onBusyChange]);

	/* list of *types* that already exist (for dropdown) */
	const existingTypes = useMemo(() => [...new Set(existing.map(k => k.type))], [existing]);

	/* provider names that ALREADY have an auth-key */
	const usedProviderNames = useMemo(
		() => new Set(existing.filter(k => k.type === AuthKeyTypeProvider).map(k => k.keyName)),
		[existing]
	);

	/* provider presets still AVAILABLE for creating new key */
	const availableProviderPresets = useMemo(() => {
		// First-run/provider-only setup should allow selecting any provider preset.
		// If a built-in empty auth-key metadata row already exists, submitting will update it.
		// That is exactly what a novice user expects when choosing a provider and pasting a key.
		if (providerOnly && !isEdit && !isPrefilled) {
			return providerPresets;
		}

		const allowed = new Set<ProviderName>();

		// allow the current provider name when editing OR when pre-filled
		if (isEdit && initial?.type === AuthKeyTypeProvider) {
			allowed.add(initial.keyName);
		}
		if (isPrefilled && prefill?.type === AuthKeyTypeProvider) {
			allowed.add(prefill.keyName);
		}

		const out: Record<ProviderName, ProviderPreset> = {};
		Object.entries(providerPresets).forEach(([name, preset]) => {
			const providerName = name;
			if (!usedProviderNames.has(name) || allowed.has(providerName)) {
				out[providerName] = preset;
			}
		});
		return out;
	}, [providerOnly, providerPresets, usedProviderNames, isEdit, initial, isPrefilled, prefill]);

	/* dropdown items for provider-name selection (create-mode only) */
	const providerDropdownItems = useMemo(() => {
		const obj: Record<ProviderName, DropdownItem> = {};
		Object.keys(availableProviderPresets).forEach(name => {
			obj[name] = {
				isEnabled: true,
			};
		});
		return obj;
	}, [availableProviderPresets]);

	/* whether *no* provider is available to create a new key for */
	const noProviderAvailable =
		!isEdit && !isPrefilled && formData.type === AuthKeyTypeProvider && Object.keys(providerDropdownItems).length === 0;

	/* type dropdown items (existing + sentinel) */
	const typeDropdownItems = useMemo(() => {
		const obj: Record<string, DropdownItem> = {};
		existingTypes.forEach(t => {
			obj[t] = { isEnabled: true };
		});
		obj[sentinelAddNew] = { isEnabled: true };
		return obj;
	}, [existingTypes]);

	const hasProviderPresets = Object.keys(providerPresets).length > 0;

	/* fetch provider presets once needed */
	useEffect(() => {
		if (formData.type !== AuthKeyTypeProvider || hasProviderPresets) {
			return;
		}

		let cancelled = false;

		void (async () => {
			try {
				const prov = await getAllProviderPresetsMap(true);
				if (!cancelled) {
					setProviderPresets(prov);
				}
			} catch (err) {
				console.error('Failed fetching provider presets', err);
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [formData.type, hasProviderPresets]);

	const checkDuplicate = useCallback(
		(t: string, n: string) =>
			// Provider-only setup intentionally updates existing provider key metadata.
			// Do not block on duplicate (provider,keyName); setAuthKey is an upsert.
			providerOnly && t === AuthKeyTypeProvider
				? false
				: existing.some(k => {
						if (k.type !== t || k.keyName !== n) {
							return false;
						} // different pair
						// allow the SAME pair that we're editing / pre-filling
						if (isEdit && k === initial) {
							return false;
						}
						if (isPrefilled && prefill?.type === t && prefill.keyName === n) {
							return false;
						}
						return true;
					}),
		[existing, initial, isEdit, isPrefilled, prefill, providerOnly]
	);

	const validateField = useCallback(
		(field: keyof FormData, value: string, state: FormData, currentErrors: FormErrors): FormErrors => {
			const next = omitManyKeys(currentErrors, [field]) as FormErrors;

			switch (field) {
				case 'type':
					if (!value) {
						next.type = 'Select a type';
					}
					break;

				case 'newType':
					if (state.type === sentinelAddNew && !value.trim()) {
						next.newType = 'New type required';
					}
					break;

				case 'keyName': {
					if (!value.trim()) {
						next.keyName = 'Key name is required';
					} else {
						const finalType = state.type === sentinelAddNew ? state.newType.trim() : state.type;
						if (finalType && checkDuplicate(finalType, value.trim())) {
							next.keyName = 'Duplicate (type, key) pair';
						}
					}
					break;
				}

				case 'secret':
					if (!value.trim()) {
						next.secret = 'Secret cannot be empty';
					}
					break;

				default:
				// Ok.
			}

			return next;
		},
		[checkDuplicate]
	);

	const validateForm = useCallback(
		(state: FormData): FormErrors => {
			let next: FormErrors = {};
			next = validateField('type', state.type, state, next);
			next = validateField('newType', state.newType, state, next);
			next = validateField('keyName', state.keyName, state, next);
			next = validateField('secret', state.secret, state, next);
			return next;
		},
		[validateField]
	);

	const handleChange = (e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
		const { name, value } = e.target;
		const field = name as keyof FormData;
		const nextState = { ...formData, [field]: value };

		setFormData(nextState);
		setErrors(prev => {
			let next = validateField(field, value, nextState, prev);

			if (field === 'type' || field === 'newType') {
				next = validateField('keyName', nextState.keyName, nextState, next);
			}

			return next;
		});
	};

	const handleTypeSelect = (k: string) => {
		const nextState: FormData = {
			...formData,
			type: k,
			newType: '',
			/* reset key-name when switching away from provider */
			keyName: k === AuthKeyTypeProvider ? formData.keyName : '',
		};

		setFormData(nextState);
		setErrors(prev => {
			let next = validateField('type', k, nextState, prev);
			next = validateField('newType', nextState.newType, nextState, next);
			next = validateField('keyName', nextState.keyName, nextState, next);
			return next;
		});
	};

	const handleKeyNameSelect = (k: ProviderName) => {
		const nextState: FormData = { ...formData, keyName: k };
		setFormData(nextState);
		setErrors(prev => validateField('keyName', k, nextState, prev));
	};

	/* overall validity */
	const isAllValid = useMemo(() => {
		if (noProviderAvailable) {
			return false;
		}
		const nextErrors = validateForm(formData);
		return Object.values(nextErrors).every(v => !v);
	}, [formData, noProviderAvailable, validateForm]);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async e => {
		e.preventDefault();

		if (isSubmitting) {
			return;
		}

		const nextErrors = validateForm(formData);
		setErrors(nextErrors);

		if (noProviderAvailable || Object.values(nextErrors).some(Boolean)) {
			return;
		}

		const finalType = formData.type === sentinelAddNew ? formData.newType.trim() : formData.type;

		setSubmitError('');
		setIsSubmitting(true);
		try {
			await aggregateAPI.setAuthKey(finalType, formData.keyName.trim(), formData.secret.trim());

			if (!unmountingRef.current) {
				onChanged();
				requestClose(true);
			}
		} catch (error) {
			if (!unmountingRef.current) {
				setSubmitError(
					error instanceof Error && error.message.trim() ? error.message : 'Failed to save the authentication key.'
				);
			}
		} finally {
			if (!unmountingRef.current) {
				setIsSubmitting(false);
			}
		}
	};

	return (
		<>
			<div className="modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-3xl flex-col overflow-hidden rounded-2xl p-0">
				<form noValidate onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col" aria-busy={isSubmitting}>
					<ModalHeader
						title={modalTitle}
						description={
							providerOnly
								? 'Store or replace the selected provider credential in the configured secure key store.'
								: 'Credentials are write-only. Existing secret values are never shown in this form.'
						}
						onClose={() => {
							requestClose();
						}}
						closeDisabled={isSubmitting}
					/>

					<div className="app-scrollbar-thin min-h-0 flex-1 space-y-4 overflow-y-auto p-4 sm:p-6">
						{intro ? (
							<div className="border-base-300 bg-base-100/70 rounded-2xl border p-4 text-sm/relaxed">{intro}</div>
						) : null}

						{submitError ? (
							<div className="alert alert-error rounded-2xl text-sm" role="alert">
								<FiAlertCircle className="shrink-0" size={14} />
								<span className="wrap-break-word">{submitError}</span>
							</div>
						) : null}

						<ModalSection
							title="Key target"
							description="Choose the logical key type and the provider or key name that owns this credential."
						>
							{!providerOnly && (
								<ModalField label="Type" required hint="Logical grouping of keys." error={errors.type}>
									{isReadOnly ? (
										<input className="input w-full rounded-2xl" value={formData.type} disabled />
									) : (
										<Dropdown<string>
											dropdownItems={typeDropdownItems}
											selectedKey={formData.type}
											onChange={handleTypeSelect}
											filterDisabled={false}
											title="Select type"
											getDisplayName={k => (k === sentinelAddNew ? 'Add new type…' : k)}
											inlineMenu
											maxMenuHeight={220}
										/>
									)}
								</ModalField>
							)}

							{!providerOnly && !isReadOnly && formData.type === sentinelAddNew && (
								<ModalField label="New Type" htmlFor="auth-key-new-type" required error={errors.newType}>
									<input
										id="auth-key-new-type"
										type="text"
										name="newType"
										value={formData.newType}
										onChange={handleChange}
										className={`input w-full rounded-2xl ${errors.newType ? 'input-error' : ''}`}
										spellCheck="false"
									/>
								</ModalField>
							)}

							<ModalField
								label={formData.type === AuthKeyTypeProvider ? 'Provider' : 'Key Name'}
								htmlFor="auth-key-name"
								required
								error={errors.keyName}
							>
								{!isReadOnly && formData.type === AuthKeyTypeProvider ? (
									Object.keys(providerDropdownItems).length > 0 ? (
										<Dropdown<ProviderName>
											dropdownItems={providerDropdownItems}
											selectedKey={formData.keyName}
											onChange={handleKeyNameSelect}
											filterDisabled={false}
											title="Select provider"
											getDisplayName={k => availableProviderPresets[k].displayName || k}
											inlineMenu
											maxMenuHeight={220}
										/>
									) : (
										/* no provider left – show disabled input */
										<input className="input w-full rounded-2xl" value="All providers already configured" disabled />
									)
								) : (
									<input
										id="auth-key-name"
										type="text"
										name="keyName"
										value={formData.keyName}
										onChange={handleChange}
										className={`input w-full rounded-2xl ${errors.keyName ? 'input-error' : ''}`}
										disabled={isReadOnly}
										spellCheck="false"
									/>
								)}
							</ModalField>
						</ModalSection>

						<ModalSection
							title="Secret"
							description={
								isEdit && initial?.nonEmpty
									? 'Enter a replacement value. Leaving this form without saving preserves the existing secret.'
									: 'The secret is stored through the configured secure key store and is not displayed later.'
							}
						>
							<ModalField label="Secret" htmlFor="auth-key-secret" required error={errors.secret}>
								<input
									id="auth-key-secret"
									type="password"
									name="secret"
									value={formData.secret}
									onChange={handleChange}
									placeholder={isEdit && initial?.nonEmpty ? 'Paste replacement secret' : 'Paste API key'}
									className={`input w-full rounded-2xl ${errors.secret ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
								/>
							</ModalField>
						</ModalSection>

						{initial ? (
							<ModalSection title="Stored metadata">
								<ManagementInfoGrid>
									<ManagementInfoRow label="Type" mono>
										{initial.type}
									</ManagementInfoRow>
									<ManagementInfoRow label="Key name" mono>
										{initial.keyName}
									</ManagementInfoRow>
									<ManagementInfoRow label="Secret configured">{initial.nonEmpty ? 'Yes' : 'No'}</ManagementInfoRow>
									<ManagementInfoRow label="SHA-256" mono>
										{initial.nonEmpty ? initial.sha256 : '—'}
									</ManagementInfoRow>
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
						<button type="submit" disabled={!isAllValid || isSubmitting} className="btn btn-primary rounded-xl">
							{isSubmitting ? 'Saving...' : submitLabel}
						</button>
					</ModalActions>
				</form>
			</div>
			<ModalBackdrop enabled={!isSubmitting} />
		</>
	);
}

export function AddEditAuthKeyModal(props: AddEditAuthKeyModalProps) {
	const [isSubmitting, setIsSubmitting] = useState(false);
	if (!props.isOpen) {
		return null;
	}

	const modalKey = props.initial
		? `edit:${props.initial.type}:${props.initial.keyName}`
		: props.prefill
			? `prefill:${props.prefill.type}:${props.prefill.keyName}`
			: props.providerOnly
				? `provider-only:${props.defaultKeyName ?? 'default'}`
				: 'add';

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose} isBusy={isSubmitting}>
			<AddEditAuthKeyModalContent key={modalKey} {...props} onBusyChange={setIsSubmitting} />
		</ModalDialog>
	);
}
