import { type ChangeEvent, type SubmitEventHandler, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiUpload, FiX } from 'react-icons/fi';

import { type ProviderName, ReasoningLevel, ReasoningType } from '@/spec/inference';
import { type ModelPreset, type ModelPresetID } from '@/spec/modelpreset';

import { Dropdown } from '@/components/dropdown';
import { ModalBackdrop } from '@/components/modal_backdrop';
import { ReadOnlyValue } from '@/components/read_only_value';

const reasoningTypeItems: Record<ReasoningType, { isEnabled: boolean; displayName: string }> = {
	[ReasoningType.SingleWithLevels]: {
		isEnabled: true,
		displayName: 'Reasoning only, with Levels',
	},
	[ReasoningType.HybridWithTokens]: {
		isEnabled: true,
		displayName: 'Hybrid, with Reasoning Tokens',
	},
};

const reasoningLevelItems: Record<ReasoningLevel, { isEnabled: boolean; displayName: string }> = {
	[ReasoningLevel.None]: { isEnabled: true, displayName: 'None' },
	[ReasoningLevel.Minimal]: { isEnabled: true, displayName: 'Minimal' },
	[ReasoningLevel.Low]: { isEnabled: true, displayName: 'Low' },
	[ReasoningLevel.Medium]: { isEnabled: true, displayName: 'Medium' },
	[ReasoningLevel.High]: { isEnabled: true, displayName: 'High' },
	[ReasoningLevel.XHigh]: { isEnabled: true, displayName: 'XHigh' },
};

/** Defaults we *apply* while in **Add** mode (place-holders also use them). */
const AddModeDefaults: ModelPreset = {
	id: '',
	name: '',
	slug: '',
	displayName: '',
	isEnabled: true,
	isBuiltIn: false,
	stream: true,
	maxPromptLength: 2048,
	maxOutputLength: 1024,
	temperature: undefined,
	reasoning: undefined,
	systemPrompt: '',
	timeout: 300,
	additionalParametersRawJSON: undefined,
};

const DEFAULT_REASONING_TOKENS = 1024;

interface ModelPresetFormData {
	presetLabel: string;
	name: string;
	isEnabled: boolean;
	stream: boolean;
	maxPromptLength: string;
	maxOutputLength: string;
	temperature: string;

	reasoningSupport: boolean;
	reasoningType?: ReasoningType;
	reasoningLevel?: ReasoningLevel;
	reasoningTokens?: string;

	systemPrompt: string;
	timeout: string;
}

type ModalMode = 'add' | 'edit' | 'view';

interface AddEditModelPresetModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (modelPresetID: ModelPresetID, modelData: ModelPreset) => void;
	providerName: ProviderName;

	/** If not provided, mode is inferred: add when no initialModelID, else edit. */
	mode?: ModalMode;

	initialModelID?: ModelPresetID; // ⇒ edit/view mode when truthy
	initialData?: ModelPreset;

	existingModels: Record<ModelPresetID, ModelPreset>;
	allModelPresets: Record<ProviderName, Record<ModelPresetID, ModelPreset>>;
}

export function AddEditModelPresetModal({
	isOpen,
	onClose,
	onSubmit,
	providerName,
	mode,
	initialModelID,
	initialData,
	existingModels,
	allModelPresets,
}: AddEditModelPresetModalProps) {
	const inferredMode: ModalMode = initialModelID ? 'edit' : 'add';
	const effectiveMode: ModalMode = mode ?? inferredMode;

	const isEditMode = effectiveMode === 'edit';
	const isViewMode = effectiveMode === 'view';
	const isReadOnly = isViewMode;

	const [defaultValues] = useState<ModelPreset>(AddModeDefaults);
	const [modelPresetID, setModelPresetID] = useState<ModelPresetID>(initialModelID ?? ('' as ModelPresetID));

	const dialogRef = useRef<HTMLDialogElement | null>(null);

	const blankForm: ModelPresetFormData = {
		presetLabel: '',
		name: '',
		isEnabled: true,
		stream: false,
		maxPromptLength: '',
		maxOutputLength: '',
		temperature: '',
		reasoningSupport: false,
		reasoningType: ReasoningType.SingleWithLevels,
		reasoningLevel: ReasoningLevel.Medium,
		reasoningTokens: '',
		systemPrompt: '',
		timeout: '',
	};

	const [formData, setFormData] = useState<ModelPresetFormData>(blankForm);

	/* ────────────────────── PREFILL ‑ from existing ──────────────────── */
	/** composite-key:  `${provider}::${modelID}`  */
	type PrefillKey = string;

	const prefillSourceMap = useMemo<Record<PrefillKey, ModelPreset>>(() => {
		const m: Record<PrefillKey, ModelPreset> = {};
		for (const [prov, presets] of Object.entries(allModelPresets)) {
			for (const [mid, mp] of Object.entries(presets)) {
				m[`${prov}::${mid}`] = mp;
			}
		}
		return m;
	}, [allModelPresets]);

	const prefillDropdownItems: Record<PrefillKey, { isEnabled: boolean; displayName: string }> = useMemo(() => {
		const o: Record<PrefillKey, { isEnabled: boolean; displayName: string }> = {};
		for (const [key, mp] of Object.entries(prefillSourceMap)) {
			const [prov] = key.split('::');
			o[key] = { isEnabled: true, displayName: `${prov} / ${mp.displayName || mp.id}` };
		}
		return o;
	}, [prefillSourceMap]);

	const [prefillMode, setPrefillMode] = useState(false);
	const [selectedPrefillKey, setSelectedPrefillKey] = useState<PrefillKey | null>(null);

	const applyPrefill = (key: PrefillKey) => {
		const src = prefillSourceMap[key];

		setFormData(prev => ({
			...prev,

			/* IDs are *NOT* copied intentionally */
			name: src.name,
			presetLabel: src.displayName,

			stream: src.stream ?? prev.stream,
			isEnabled: true, // always create *enabled* presets when adding

			maxPromptLength: src.maxPromptLength !== undefined ? String(src.maxPromptLength) : prev.maxPromptLength,
			maxOutputLength: src.maxOutputLength !== undefined ? String(src.maxOutputLength) : prev.maxOutputLength,
			temperature: src.temperature !== undefined ? String(src.temperature) : prev.temperature,

			reasoningSupport: !!src.reasoning,
			reasoningType: src.reasoning?.type ?? prev.reasoningType,
			reasoningLevel: src.reasoning?.level ?? prev.reasoningLevel,
			reasoningTokens: src.reasoning?.tokens !== undefined ? String(src.reasoning.tokens) : prev.reasoningTokens,

			systemPrompt: src.systemPrompt ?? prev.systemPrompt,
			timeout: src.timeout !== undefined ? String(src.timeout) : prev.timeout,
		}));
	};

	/* ───────────────────────── load / reset ──────────────────────────── */
	useEffect(() => {
		if (!isOpen) return;

		const loadData = () => {
			if ((isEditMode || isViewMode) && initialData) {
				setFormData({
					presetLabel: initialData.displayName,
					name: initialData.name,
					isEnabled: initialData.isEnabled,
					stream: initialData.stream ?? false,
					maxPromptLength: initialData.maxPromptLength !== undefined ? String(initialData.maxPromptLength) : '',
					maxOutputLength: initialData.maxOutputLength !== undefined ? String(initialData.maxOutputLength) : '',
					temperature: initialData.temperature !== undefined ? String(initialData.temperature) : '',
					reasoningSupport: !!initialData.reasoning,
					reasoningType: initialData.reasoning?.type ?? ReasoningType.SingleWithLevels,
					reasoningLevel: initialData.reasoning?.level ?? ReasoningLevel.Medium,
					reasoningTokens: initialData.reasoning?.tokens !== undefined ? String(initialData.reasoning.tokens) : '',
					systemPrompt: initialData.systemPrompt ?? '',
					timeout: initialData.timeout !== undefined ? String(initialData.timeout) : '',
				});
				setModelPresetID(initialData.id);
			} else {
				setFormData({
					presetLabel: '',
					name: '',
					isEnabled: true,
					stream: AddModeDefaults.stream ?? false,
					maxPromptLength: String(AddModeDefaults.maxPromptLength ?? ''),
					maxOutputLength: String(AddModeDefaults.maxOutputLength ?? ''),
					temperature: '',
					reasoningSupport: false,
					reasoningType: ReasoningType.SingleWithLevels,
					reasoningLevel: ReasoningLevel.Medium,
					reasoningTokens: '',
					systemPrompt: '',
					timeout: String(AddModeDefaults.timeout ?? ''),
				});
				setModelPresetID('' as ModelPresetID);
				setPrefillMode(false);
				setSelectedPrefillKey(null);
			}

			setErrors({});
		};

		loadData();
	}, [isOpen, isEditMode, isViewMode, initialData]);

	// Open/close native dialog
	useEffect(() => {
		if (!isOpen) return;

		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) dialog.showModal();

		return () => {
			if (dialog.open) dialog.close();
		};
	}, [isOpen]);

	const handleDialogClose = () => {
		onClose();
	};

	type ValidationField =
		| 'modelPresetID'
		| 'name'
		| 'presetLabel'
		| 'temperature'
		| 'maxPromptLength'
		| 'maxOutputLength'
		| 'timeout'
		| 'reasoningTokens';

	type ValidationErrors = Partial<Record<ValidationField, string>>;

	const [errors, setErrors] = useState<ValidationErrors>({});

	const calcNumericError = (
		field: ValidationField,
		strVal: string,
		minOrRange?: { min?: number; max?: number }
	): string | undefined => {
		if (strVal.trim() === '') return;
		const num = Number(strVal);
		if (Number.isNaN(num)) return `${field} must be a valid number.`;
		if (minOrRange?.min !== undefined && num < minOrRange.min) return `${field} must be ≥ ${minOrRange.min}.`;
		if (minOrRange?.max !== undefined && num > minOrRange.max) return `${field} must be ≤ ${minOrRange.max}.`;
		return;
	};

	const runValidation = (): ValidationErrors => {
		if (isReadOnly) return {};

		const v: ValidationErrors = {};

		/* id */
		if (!isEditMode) {
			const idTrim = modelPresetID.trim();
			if (!idTrim) v.modelPresetID = 'Model Preset ID is required.';
			else if (!/^[a-zA-Z0-9-]+$/.test(idTrim)) v.modelPresetID = 'Only letters, numbers, and hyphens allowed.';
			else if (Object.prototype.hasOwnProperty.call(existingModels, idTrim))
				v.modelPresetID = 'Model Preset ID must be unique.';
		}

		/* required strings */
		if (!formData.name.trim()) v.name = 'Model Name is required.';
		if (!formData.presetLabel.trim()) v.presetLabel = 'Preset Label is required.';

		/* numeric fields */
		const maybe = (f: ValidationField, rng?: { min?: number; max?: number }) => {
			let value: string;
			if (f === 'modelPresetID') value = modelPresetID;
			else value = formData[f as keyof ModelPresetFormData] as string;

			v[f] = calcNumericError(f, value, rng);
		};

		maybe('temperature', { min: 0, max: 1 });
		maybe('maxPromptLength', { min: 1 });
		maybe('maxOutputLength', { min: 1 });
		maybe('timeout', { min: 1 });

		// Reasoning tokens required ONLY for HybridWithTokens
		if (formData.reasoningSupport && formData.reasoningType === ReasoningType.HybridWithTokens) {
			if ((formData.reasoningTokens ?? '').trim() === '') {
				v.reasoningTokens = 'Reasoning Tokens is required for Hybrid mode.';
			} else {
				maybe('reasoningTokens', { min: 1024 });
			}
		}

		/* at least one of "reasoning" or "temperature" should be present */
		const hasTemp = formData.temperature.trim() !== '';
		if (!formData.reasoningSupport && !hasTemp) {
			v.temperature = 'Provide either Temperature or enable Reasoning.';
		}

		return Object.fromEntries(
			Object.entries(v).filter(([key]) => v[key as ValidationField] !== undefined)
		) as ValidationErrors;
	};

	/* helper - string ⇒ number (keep default if blank / NaN) */
	const parseOrDefault = (val: string, def: number) => {
		const n = Number(val);
		return val.trim() === '' || Number.isNaN(n) ? def : n;
	};

	const handleFieldChange = (field: keyof ModelPresetFormData, value: string | boolean) => {
		setFormData(prev => ({ ...prev, [field]: value }));
	};

	const handleChange = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
		const { name, value, type, checked } = e.target as HTMLInputElement;

		if (name === 'modelPresetID') {
			setModelPresetID(value);
			return;
		}

		if (type === 'checkbox') handleFieldChange(name as any, checked);
		else handleFieldChange(name as any, value);
	};

	const isAllValid = useMemo(() => {
		if (isReadOnly) return true;
		const currentErrors = runValidation();
		return Object.keys(currentErrors).length === 0;
	}, [formData, modelPresetID, isReadOnly]);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();
		e.stopPropagation();

		if (isReadOnly) {
			dialogRef.current?.close();
			return;
		}

		const newErrors = runValidation();
		setErrors(newErrors);
		if (Object.keys(newErrors).length !== 0) return;

		const finalModelPresetID = modelPresetID.trim();

		const finalData: ModelPreset = {
			id: finalModelPresetID,
			slug: finalModelPresetID,
			isBuiltIn: false,

			name: formData.name.trim(),
			displayName: formData.presetLabel.trim(),
			isEnabled: formData.isEnabled,
			stream: formData.stream,

			maxPromptLength: parseOrDefault(formData.maxPromptLength, defaultValues.maxPromptLength ?? 2048),
			maxOutputLength: parseOrDefault(formData.maxOutputLength, defaultValues.maxOutputLength ?? 1024),

			timeout: parseOrDefault(formData.timeout, defaultValues.timeout ?? 300),
			systemPrompt: formData.systemPrompt,

			...(formData.temperature.trim() !== '' && {
				temperature: parseOrDefault(formData.temperature, 0.1),
			}),
		};

		if (formData.reasoningSupport) {
			const type = formData.reasoningType ?? ReasoningType.SingleWithLevels;

			finalData.reasoning = {
				type,
				level: formData.reasoningLevel ?? ReasoningLevel.Medium,
				tokens:
					type === ReasoningType.HybridWithTokens
						? parseOrDefault(formData.reasoningTokens ?? '', DEFAULT_REASONING_TOKENS)
						: DEFAULT_REASONING_TOKENS,
			};
		}

		onSubmit(finalModelPresetID, finalData);
		dialogRef.current?.close();
	};

	const numPlaceholder = (field: keyof ModelPreset) => {
		const v = defaultValues[field];
		return v === undefined || typeof v === 'object' ? 'Default: N/A' : `Default: ${String(v)}`;
	};

	if (!isOpen) return null;

	const title =
		effectiveMode === 'add' ? 'Add Model Preset' : effectiveMode === 'edit' ? 'Edit Model Preset' : 'View Model Preset';

	return createPortal(
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={e => {
				// Form mode (add/edit): block Esc close. View mode: allow.
				if (!isReadOnly) e.preventDefault();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-3xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between">
						<div className="flex flex-col">
							<h3 className="text-lg font-bold">{title}</h3>
							<div className="flex items-center text-xs opacity-60">
								<span>Provider: {providerName}</span>
								{initialData?.isBuiltIn && <span>,built-in</span>}
								{!initialData?.isBuiltIn && <span>,custom</span>}
							</div>
						</div>

						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={() => dialogRef.current?.close()}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					<form noValidate onSubmit={handleSubmit} className="space-y-4">
						{/* ───── Prefill (ADD mode only) ───── */}
						{effectiveMode === 'add' && (
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
										>
											<FiUpload size={14} />
											<span className="ml-1">Copy Existing Preset</span>
										</button>
									)}

									{prefillMode && (
										<>
											<Dropdown<PrefillKey>
												dropdownItems={prefillDropdownItems}
												selectedKey={selectedPrefillKey ?? ('' as PrefillKey)}
												onChange={key => {
													setSelectedPrefillKey(key);
													applyPrefill(key);
													setPrefillMode(false);
												}}
												filterDisabled={false}
												title="Select model preset to copy"
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
											>
												<FiX size={12} />
											</button>
										</>
									)}
								</div>
							</div>
						)}

						{/* ModelPresetID */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Model Preset ID*</span>
								<span
									className="label-text-alt tooltip tooltip-right"
									data-tip="Unique identifier. Letters, numbers, hyphen."
								>
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								{effectiveMode === 'add' ? (
									<input
										name="modelPresetID"
										type="text"
										className={`input input-bordered w-full rounded-xl ${errors.modelPresetID ? 'input-error' : ''}`}
										value={modelPresetID}
										onChange={handleChange}
										placeholder="e.g. gpt4-preset"
										autoComplete="off"
										spellCheck="false"
									/>
								) : (
									<ReadOnlyValue value={modelPresetID} />
								)}
								{errors.modelPresetID && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} />
											{errors.modelPresetID}
										</span>
									</div>
								)}
							</div>
						</div>

						{/* Model Name */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Model Name*</span>
								<span
									className="label-text-alt tooltip tooltip-right"
									data-tip="The name you send to the completions API (e.g. gpt-4)"
								>
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									name="name"
									type="text"
									className={`input input-bordered w-full rounded-xl ${errors.name ? 'input-error' : ''}`}
									value={formData.name}
									onChange={handleChange}
									placeholder="e.g. gpt-4, claude-3-opus-20240229"
									autoComplete="off"
									spellCheck="false"
									autoFocus={!isReadOnly}
									readOnly={isReadOnly}
								/>
								{errors.name && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} />
											{errors.name}
										</span>
									</div>
								)}
							</div>
						</div>

						{/* Preset Label */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Preset Label*</span>
								<span className="label-text-alt tooltip tooltip-right" data-tip="Friendly name shown in the UI">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									name="presetLabel"
									type="text"
									className={`input input-bordered w-full rounded-xl ${errors.presetLabel ? 'input-error' : ''}`}
									value={formData.presetLabel}
									onChange={handleChange}
									placeholder="e.g. GPT-4 (Creative)"
									autoComplete="off"
									spellCheck="false"
									readOnly={isReadOnly}
								/>
								{errors.presetLabel && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} />
											{errors.presetLabel}
										</span>
									</div>
								)}
							</div>
						</div>

						{/* Toggles */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3 cursor-pointer">
								<span className="label-text text-sm">Enabled</span>
							</label>
							<div className="col-span-9">
								<input
									type="checkbox"
									name="isEnabled"
									className="toggle toggle-accent disabled:opacity-80"
									checked={formData.isEnabled}
									onChange={handleChange}
									disabled={isReadOnly}
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3 cursor-pointer">
								<span className="label-text text-sm">Streaming</span>
							</label>
							<div className="col-span-9">
								<input
									type="checkbox"
									name="stream"
									className="toggle toggle-accent disabled:opacity-80"
									checked={formData.stream}
									onChange={handleChange}
									disabled={isReadOnly}
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3 cursor-pointer">
								<span className="label-text text-sm">Supports Reasoning</span>
								<span className="label-text-alt tooltip tooltip-right" data-tip="If enabled, configure below">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									type="checkbox"
									name="reasoningSupport"
									className="toggle toggle-accent disabled:opacity-80"
									checked={formData.reasoningSupport}
									onChange={handleChange}
									disabled={isReadOnly}
								/>
							</div>
						</div>

						{/* Reasoning controls */}
						{formData.reasoningSupport && (
							<>
								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Reasoning Type</span>
									</label>
									<div className="col-span-9">
										{isReadOnly ? (
											<ReadOnlyValue
												value={reasoningTypeItems[formData.reasoningType ?? ReasoningType.SingleWithLevels].displayName}
											/>
										) : (
											<Dropdown<ReasoningType>
												dropdownItems={reasoningTypeItems}
												selectedKey={formData.reasoningType ?? ReasoningType.SingleWithLevels}
												onChange={t => {
													setFormData(p => ({ ...p, reasoningType: t }));
												}}
												filterDisabled={false}
												title="Select Reasoning Type"
												getDisplayName={k => reasoningTypeItems[k].displayName}
											/>
										)}
									</div>
								</div>

								{formData.reasoningType === ReasoningType.SingleWithLevels && (
									<div className="grid grid-cols-12 items-center gap-2">
										<label className="label col-span-3">
											<span className="label-text text-sm">Reasoning Level</span>
										</label>
										<div className="col-span-9">
											{isReadOnly ? (
												<ReadOnlyValue
													value={reasoningLevelItems[formData.reasoningLevel ?? ReasoningLevel.Medium].displayName}
												/>
											) : (
												<Dropdown<ReasoningLevel>
													dropdownItems={reasoningLevelItems}
													selectedKey={formData.reasoningLevel ?? ReasoningLevel.Medium}
													onChange={lvl => {
														setFormData(p => ({ ...p, reasoningLevel: lvl }));
													}}
													filterDisabled={false}
													title="Select Reasoning Level"
													getDisplayName={k => reasoningLevelItems[k].displayName}
												/>
											)}
										</div>
									</div>
								)}

								{formData.reasoningType === ReasoningType.HybridWithTokens && (
									<div className="grid grid-cols-12 items-center gap-2">
										<label className="label col-span-3">
											<span className="label-text text-sm">Reasoning Tokens</span>
										</label>
										<div className="col-span-9">
											<input
												name="reasoningTokens"
												type="text"
												className={`input input-bordered w-full rounded-xl ${errors.reasoningTokens ? 'input-error' : ''}`}
												value={formData.reasoningTokens}
												onChange={handleChange}
												placeholder="e.g. 1024"
												spellCheck="false"
												disabled={isReadOnly}
											/>
											{errors.reasoningTokens && (
												<div className="label">
													<span className="label-text-alt text-error flex items-center gap-1">
														<FiAlertCircle size={12} />
														{errors.reasoningTokens}
													</span>
												</div>
											)}
										</div>
									</div>
								)}
							</>
						)}

						{/* Temperature */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Temperature (0-1)</span>
								<span className="label-text-alt tooltip tooltip-right" data-tip="This or Reasoning is needed">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									name="temperature"
									type="text"
									className={`input input-bordered w-full rounded-xl ${errors.temperature ? 'input-error' : ''}`}
									value={formData.temperature}
									onChange={handleChange}
									placeholder={numPlaceholder('temperature')}
									spellCheck="false"
									readOnly={isReadOnly}
								/>
								{errors.temperature && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.temperature}
										</span>
									</div>
								)}
							</div>
						</div>

						{/* Timeout */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Timeout (seconds)</span>
							</label>
							<div className="col-span-9">
								<input
									name="timeout"
									type="text"
									className={`input input-bordered w-full rounded-xl ${errors.timeout ? 'input-error' : ''}`}
									value={formData.timeout}
									onChange={handleChange}
									placeholder={numPlaceholder('timeout')}
									spellCheck="false"
									readOnly={isReadOnly}
								/>
								{errors.timeout && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.timeout}
										</span>
									</div>
								)}
							</div>
						</div>

						{/* Prompt / Output lengths */}
						{(['maxPromptLength', 'maxOutputLength'] as const).map(f => (
							<div className="grid grid-cols-12 items-center gap-2" key={f}>
								<label className="label col-span-3">
									<span className="label-text text-sm">
										{f === 'maxPromptLength' ? 'Max Prompt Tokens' : 'Max Output Tokens'}
									</span>
								</label>
								<div className="col-span-9">
									<input
										name={f}
										type="text"
										className={`input input-bordered w-full rounded-xl ${errors[f] ? 'input-error' : ''}`}
										value={formData[f]}
										onChange={handleChange}
										placeholder={numPlaceholder(f)}
										spellCheck="false"
										readOnly={isReadOnly}
									/>
									{errors[f] && (
										<div className="label">
											<span className="label-text-alt text-error flex items-center gap-1">
												<FiAlertCircle size={12} /> {errors[f]}
											</span>
										</div>
									)}
								</div>
							</div>
						))}

						{/* System Prompt */}
						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">System Prompt</span>
							</label>
							<div className="col-span-9">
								<textarea
									name="systemPrompt"
									className="textarea textarea-bordered h-24 w-full rounded-xl"
									value={formData.systemPrompt}
									onChange={handleChange}
									placeholder="Enter instructions here…"
									spellCheck="false"
									disabled={isReadOnly}
								/>
							</div>
						</div>

						{/* ACTIONS */}
						<div className="modal-action">
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={() => dialogRef.current?.close()}>
								{isReadOnly ? 'Close' : 'Cancel'}
							</button>

							{!isReadOnly && (
								<button type="submit" className="btn btn-primary rounded-xl" disabled={!isAllValid}>
									{isEditMode ? 'Save Changes' : 'Add Preset'}
								</button>
							)}
						</div>
					</form>
				</div>
			</div>
			<ModalBackdrop enabled={isReadOnly} />
		</dialog>,
		document.body
	);
}
