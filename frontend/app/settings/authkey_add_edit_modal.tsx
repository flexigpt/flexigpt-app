import type { ChangeEvent, SubmitEventHandler } from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiX } from 'react-icons/fi';

import type { ProviderName } from '@/spec/inference';
import type { ProviderPreset } from '@/spec/modelpreset';
import type { AuthKeyMeta } from '@/spec/setting';
import { AuthKeyTypeProvider } from '@/spec/setting';

import { omitManyKeys } from '@/lib/obj_utils';

import { settingstoreAPI } from '@/apis/baseapi';
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
}

const sentinelAddNew = '__add_new__';

interface FormData {
	type: string; // existing type | sentinelAddNew
	keyName: string; // provider name | free string
	secret: string; // the secret the user types
	newType: string; // only when sentinelAddNew chosen
}

export function AddEditAuthKeyModal({
	isOpen,
	initial,
	existing,
	onClose,
	onChanged,
	prefill = null,
}: AddEditAuthKeyModalProps) {
	const isEdit = Boolean(initial); // “edit” = we already have that record
	const isPrefilled = !isEdit && !!prefill; // “add”, but (type,keyName) should be fixed
	const isReadOnly = isEdit || isPrefilled; // helper for rendering

	const [form, setForm] = useState<FormData>({
		type: initial?.type ?? prefill?.type ?? AuthKeyTypeProvider,
		keyName: initial?.keyName ?? prefill?.keyName ?? '',
		secret: '',
		newType: '',
	});
	const [errors, setErrors] = useState<Partial<Record<keyof FormData, string>>>({});

	/* raw provider presets fetched from backend */
	const [providerPresets, setProviderPresets] = useState<Record<ProviderName, ProviderPreset>>({});

	const dialogRef = useRef<HTMLDialogElement | null>(null);

	/* list of *types* that already exist (for dropdown) */
	const existingTypes = useMemo(() => [...new Set(existing.map(k => k.type))], [existing]);

	/* provider names that ALREADY have an auth-key */
	const usedProviderNames = useMemo(
		() => new Set(existing.filter(k => k.type === AuthKeyTypeProvider).map(k => k.keyName)),
		[existing]
	);

	/* provider presets still AVAILABLE for creating new key */
	const availableProviderPresets = useMemo(() => {
		const allowed = new Set<ProviderName>();

		// allow the current provider name when editing OR when pre-filled
		if (isEdit && initial?.type === AuthKeyTypeProvider) {
			allowed.add(initial.keyName);
		}
		if (isPrefilled && prefill.type === AuthKeyTypeProvider) {
			allowed.add(prefill.keyName);
		}

		const out: Record<ProviderName, ProviderPreset> = {};
		Object.entries(providerPresets).forEach(([name, preset]) => {
			if (!usedProviderNames.has(name) || allowed.has(name)) {
				out[name] = preset;
			}
		});
		return out;
	}, [providerPresets, usedProviderNames, isEdit, initial, isPrefilled, prefill]);

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
		!isEdit && !isPrefilled && form.type === AuthKeyTypeProvider && Object.keys(providerDropdownItems).length === 0;

	/* type dropdown items (existing + sentinel) */
	const typeDropdownItems = useMemo(() => {
		const obj: Record<string, DropdownItem> = {};
		existingTypes.forEach(t => (obj[t] = { isEnabled: true }));
		obj[sentinelAddNew] = { isEnabled: true };
		return obj;
	}, [existingTypes]);

	/* reset the form every time the modal is opened for a given target */
	useEffect(() => {
		if (!isOpen) return;

		setForm({
			type: initial?.type ?? prefill?.type ?? AuthKeyTypeProvider,
			keyName: initial?.keyName ?? prefill?.keyName ?? '',
			secret: '',
			newType: '',
		});
		setErrors({});
	}, [isOpen, initial, prefill]);

	/* fetch provider presets once needed */
	useEffect(() => {
		if (isOpen && form.type === AuthKeyTypeProvider && Object.keys(providerPresets).length === 0) {
			(async () => {
				try {
					const prov = await getAllProviderPresetsMap(true);
					setProviderPresets(prov);
				} catch (err) {
					console.error('Failed fetching provider presets', err);
				}
			})();
		}
	}, [isOpen, form.type, providerPresets]);

	// Open the dialog natively when isOpen becomes true
	useEffect(() => {
		if (!isOpen) return;

		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			dialog.showModal();
		}

		return () => {
			// If the component unmounts while the dialog is still open, close it.
			if (dialog.open) {
				dialog.close();
			}
		};
	}, [isOpen]);

	// Sync parent state whenever the dialog is closed (Esc or dialog.close()).
	const handleDialogClose = () => {
		onClose();
	};

	const checkDuplicate = (t: string, n: string) =>
		existing.some(k => {
			if (k.type !== t || k.keyName !== n) return false; // different pair
			// allow the SAME pair that we're editing / pre-filling
			if (isEdit && k === initial) return false;
			if (isPrefilled && prefill.type === t && prefill.keyName === n) return false;
			return true;
		});

	const validate = (field: keyof FormData, value: string) => {
		const next = omitManyKeys(errors, [field]) as Partial<Record<keyof FormData, string>>;

		switch (field) {
			case 'type':
				if (!value) next.type = 'Select a type';
				break;

			case 'newType':
				if (form.type === sentinelAddNew && !value.trim()) next.newType = 'New type required';
				break;

			case 'keyName': {
				if (!value.trim()) next.keyName = 'Key name is required';
				else {
					const t = form.type === sentinelAddNew ? form.newType : form.type;
					if (checkDuplicate(t, value.trim())) next.keyName = 'Duplicate (type, key) pair';
				}
				break;
			}

			case 'secret':
				if (!value.trim()) next.secret = 'Secret cannot be empty';
				break;
		}
		setErrors(next);
	};

	const handleChange = (e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
		const { name, value } = e.target;
		setForm(prev => ({ ...prev, [name]: value }));
		validate(name as keyof FormData, value);
	};

	const handleTypeSelect = (k: string) => {
		setForm(prev => ({
			...prev,
			type: k,
			newType: '',
			/* reset key-name when switching away from provider */
			keyName: k === AuthKeyTypeProvider ? prev.keyName : '',
		}));
		validate('type', k);
	};

	const handleKeyNameSelect = (k: ProviderName) => {
		setForm(prev => ({ ...prev, keyName: k }));
		validate('keyName', k);
	};

	/* overall validity */
	const isAllValid = useMemo(() => {
		if (noProviderAvailable) return false;

		if (!form.secret.trim()) return false;
		if (form.type === sentinelAddNew && !form.newType.trim()) return false;
		if (!form.keyName.trim()) return false;

		return Object.values(errors).every(v => !v);
	}, [form, errors, noProviderAvailable]);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async e => {
		e.preventDefault();

		// explicit validation trigger
		(Object.entries(form) as [keyof FormData, string][]).forEach(([k, v]) => {
			validate(k, v);
		});
		if (!isAllValid) return;

		const finalType = form.type === sentinelAddNew ? form.newType.trim() : form.type;

		await settingstoreAPI.setAuthKey(finalType, form.keyName.trim(), form.secret.trim());

		onChanged();
		// Close via native dialog API; this will trigger handleDialogClose -> parent onClose()
		dialogRef.current?.close();
	};

	/* closed → nothing to render */
	if (!isOpen) return null;

	return createPortal(
		<dialog ref={dialogRef} className="modal" onClose={handleDialogClose}>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-auto rounded-2xl">
				{/* header */}
				<div className="mb-4 flex items-center justify-between">
					<h3 className="text-lg font-bold">{isEdit ? 'Edit Auth Key' : 'Add Auth Key'}</h3>
					<button
						type="button"
						className="btn btn-sm btn-circle bg-base-300"
						onClick={() => dialogRef.current?.close()}
						aria-label="Close"
					>
						<FiX size={12} />
					</button>
				</div>

				<form onSubmit={handleSubmit} className="space-y-4">
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-3">
							<span className="label-text text-sm">Type*</span>
							<span className="label-text-alt tooltip tooltip-right" data-tip="Logical grouping of keys">
								<FiHelpCircle size={12} />
							</span>
						</label>
						<div className="col-span-9">
							{isReadOnly ? (
								<input className="input input-bordered w-full rounded-2xl" value={form.type} disabled />
							) : (
								<Dropdown<string>
									dropdownItems={typeDropdownItems}
									selectedKey={form.type}
									onChange={handleTypeSelect}
									filterDisabled={false}
									title="Select type"
									getDisplayName={k => (k === sentinelAddNew ? 'Add new type…' : k)}
								/>
							)}
							{errors.type && <FieldError msg={errors.type} />}
						</div>
					</div>

					{/* NEW TYPE (only when sentinel)  */}
					{!isReadOnly && form.type === sentinelAddNew && (
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">New Type*</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									name="newType"
									value={form.newType}
									onChange={handleChange}
									className={`input input-bordered w-full rounded-2xl ${errors.newType ? 'input-error' : ''}`}
									spellCheck="false"
								/>
								{errors.newType && <FieldError msg={errors.newType} />}
							</div>
						</div>
					)}

					{/* KEY NAME  */}
					<div className="grid grid-cols-12 items-center gap-2">
						<label className="label col-span-3">
							<span className="label-text text-sm">Key Name*</span>
						</label>
						<div className="col-span-9">
							{/* provider-type dropdown (create) */}
							{!isReadOnly && form.type === AuthKeyTypeProvider ? (
								Object.keys(providerDropdownItems).length > 0 ? (
									<Dropdown<ProviderName>
										dropdownItems={providerDropdownItems}
										selectedKey={form.keyName}
										onChange={handleKeyNameSelect}
										filterDisabled={false}
										title="Select provider"
										getDisplayName={k => availableProviderPresets[k].displayName || k}
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
									value={form.keyName}
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
								value={form.secret}
								onChange={handleChange}
								className={`input input-bordered w-full rounded-2xl ${errors.secret ? 'input-error' : ''}`}
								spellCheck="false"
							/>
							{errors.secret && <FieldError msg={errors.secret} />}
						</div>
					</div>

					{/* ACTIONS - */}
					<div className="modal-action mt-6">
						<button type="button" className="btn bg-base-300 rounded-xl" onClick={() => dialogRef.current?.close()}>
							Cancel
						</button>
						<button type="submit" disabled={!isAllValid} className="btn btn-primary rounded-xl">
							{isEdit ? 'Update' : 'Add'}
						</button>
					</div>
				</form>
			</div>
		</dialog>,
		document.body
	);
}

/* ───────────────────────── field-error helper ───────────────────────── */
function FieldError({ msg }: { msg?: string }) {
	return msg ? (
		<div className="label">
			<span className="label-text-alt text-error flex items-center gap-1">
				<FiAlertCircle size={12} /> {msg}
			</span>
		</div>
	) : null;
}
