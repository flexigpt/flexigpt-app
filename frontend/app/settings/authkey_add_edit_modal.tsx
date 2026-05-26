import type { ChangeEvent, ReactNode, SubmitEventHandler } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiX } from 'react-icons/fi';

import type { ProviderName } from '@/spec/inference';
import type { ProviderPreset } from '@/spec/modelpreset';
import type { AuthKeyMeta } from '@/spec/setting';
import { AuthKeyTypeProvider } from '@/spec/setting';

import { omitManyKeys } from '@/lib/obj_utils';

import { aggregateAPI } from '@/apis/baseapi';
import { getAllProviderPresetsMap } from '@/apis/list_helper';

import type { DropdownItem } from '@/components/dropdown';
import { Dropdown } from '@/components/dropdown';

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
	onClose,
	onChanged,
	prefill = null,
	providerOnly = false,
	defaultKeyName = null,
	intro = null,
}: AddEditAuthKeyModalProps) {
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

	/* raw provider presets fetched from backend */
	const [providerPresets, setProviderPresets] = useState<Record<ProviderName, ProviderPreset>>({});

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const isUnmountingRef = useRef(false);

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
		if (formData.type !== AuthKeyTypeProvider || hasProviderPresets) return;

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

	const checkDuplicate = useCallback(
		(t: string, n: string) =>
			// Provider-only setup intentionally updates existing provider key metadata.
			// Do not block on duplicate (provider,keyName); setAuthKey is an upsert.
			providerOnly && t === AuthKeyTypeProvider
				? false
				: existing.some(k => {
						if (k.type !== t || k.keyName !== n) return false; // different pair
						// allow the SAME pair that we're editing / pre-filling
						if (isEdit && k === initial) return false;
						if (isPrefilled && prefill?.type === t && prefill.keyName === n) return false;
						return true;
					}),
		[existing, initial, isEdit, isPrefilled, prefill, providerOnly]
	);

	const validateField = useCallback(
		(field: keyof FormData, value: string, state: FormData, currentErrors: FormErrors): FormErrors => {
			const next = omitManyKeys(currentErrors, [field]) as FormErrors;

			switch (field) {
				case 'type':
					if (!value) next.type = 'Select a type';
					break;

				case 'newType':
					if (state.type === sentinelAddNew && !value.trim()) next.newType = 'New type required';
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
					if (!value.trim()) next.secret = 'Secret cannot be empty';
					break;
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
		if (noProviderAvailable) return false;
		const nextErrors = validateForm(formData);
		return Object.values(nextErrors).every(v => !v);
	}, [formData, noProviderAvailable, validateForm]);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async e => {
		e.preventDefault();

		const nextErrors = validateForm(formData);
		setErrors(nextErrors);

		if (noProviderAvailable || Object.values(nextErrors).some(Boolean)) return;

		const finalType = formData.type === sentinelAddNew ? formData.newType.trim() : formData.type;

		await aggregateAPI.setAuthKey(finalType, formData.keyName.trim(), formData.secret.trim());

		onChanged();
		requestClose();
	};

	return (
		<dialog ref={dialogRef} className="modal" onClose={handleDialogClose}>
			<div className="modal-box bg-base-200 flex max-h-[80vh] min-h-[40vh] max-w-3xl flex-col overflow-hidden rounded-2xl">
				{/* header */}
				<div className="mb-4 flex shrink-0 flex-col gap-2">
					<div className="flex items-center justify-between">
						<h3 className="text-lg font-bold">{modalTitle}</h3>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={requestClose}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>
					{intro ? (
						<div className="border-base-300 bg-base-100/70 rounded-2xl border p-4 text-sm leading-relaxed">{intro}</div>
					) : null}
				</div>

				<form onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col">
					<div className="flex min-h-0 flex-1 flex-col">
						<div className="min-h-0 flex-1 space-y-4 overflow-y-auto pr-1">
							{!providerOnly && (
								<div className="grid grid-cols-12 items-start gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Type*</span>
										<span className="label-text-alt tooltip tooltip-right" data-tip="Logical grouping of keys">
											<FiHelpCircle size={12} />
										</span>
									</label>
									<div className="col-span-9">
										{isReadOnly ? (
											<input className="input input-bordered w-full rounded-2xl" value={formData.type} disabled />
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
										{errors.type && <FieldError msg={errors.type} />}
									</div>
								</div>
							)}

							{!providerOnly && !isReadOnly && formData.type === sentinelAddNew && (
								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">New Type*</span>
									</label>
									<div className="col-span-9">
										<input
											type="text"
											name="newType"
											value={formData.newType}
											onChange={handleChange}
											className={`input input-bordered w-full rounded-2xl ${errors.newType ? 'input-error' : ''}`}
											spellCheck="false"
										/>
										{errors.newType && <FieldError msg={errors.newType} />}
									</div>
								</div>
							)}

							{/* KEY NAME  */}
							<div className="grid grid-cols-12 items-start gap-2">
								<label className="label col-span-3">
									<span className="label-text text-sm">
										{formData.type === AuthKeyTypeProvider ? 'Provider*' : 'Key Name*'}
									</span>
								</label>
								<div className="col-span-9">
									{/* provider-type dropdown (create) */}
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
											<input
												className="input input-bordered w-full rounded-2xl"
												value="All providers already configured"
												disabled
											/>
										)
									) : (
										/* simple text input (non-provider OR read-only mode) */
										<input
											type="text"
											name="keyName"
											value={formData.keyName}
											onChange={handleChange}
											className={`input input-bordered w-full rounded-2xl ${errors.keyName ? 'input-error' : ''}`}
											disabled={isReadOnly}
											spellCheck="false"
										/>
									)}
									{errors.keyName && <FieldError msg={errors.keyName} />}
								</div>
							</div>

							{/* SECRET -- */}
							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="label-text text-sm">Secret*</span>
								</label>
								<div className="col-span-9">
									<input
										type="password"
										name="secret"
										value={formData.secret}
										onChange={handleChange}
										placeholder={isEdit && initial?.nonEmpty ? 'Paste replacement secret' : 'Paste API key'}
										className={`input input-bordered w-full rounded-2xl ${errors.secret ? 'input-error' : ''}`}
										spellCheck="false"
										autoComplete="off"
									/>
									{errors.secret && <FieldError msg={errors.secret} />}
								</div>
							</div>
						</div>

						{/* ACTIONS - */}
						<div className="modal-action mt-2 flex shrink-0 justify-between">
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={requestClose}>
								Cancel
							</button>
							<button type="submit" disabled={!isAllValid} className="btn btn-primary rounded-xl">
								{submitLabel}
							</button>
						</div>
					</div>
				</form>
			</div>
		</dialog>
	);
}

function FieldError({ msg }: { msg?: string }) {
	return msg ? (
		<div className="label">
			<span className="label-text-alt text-error flex items-center gap-1">
				<FiAlertCircle size={12} /> {msg}
			</span>
		</div>
	) : null;
}

export function AddEditAuthKeyModal(props: AddEditAuthKeyModalProps) {
	if (!props.isOpen) return null;
	if (typeof document === 'undefined' || !document.body) return null;

	const modalKey = props.initial
		? `edit:${props.initial.type}:${props.initial.keyName}`
		: props.prefill
			? `prefill:${props.prefill.type}:${props.prefill.keyName}`
			: props.providerOnly
				? `provider-only:${props.defaultKeyName ?? 'default'}`
				: 'add';

	return createPortal(<AddEditAuthKeyModalContent key={modalKey} {...props} />, document.body);
}
