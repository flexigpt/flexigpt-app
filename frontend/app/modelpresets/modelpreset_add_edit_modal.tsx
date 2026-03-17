import { type ChangeEvent, type SubmitEventHandler, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiUpload, FiX } from 'react-icons/fi';

import {
	OutputFormatKind,
	type OutputParam,
	OutputVerbosity,
	type ProviderName,
	ReasoningLevel,
	type ReasoningParam,
	ReasoningSummaryStyle,
	ReasoningType,
} from '@/spec/inference';
import {
	type ModelPreset,
	type ModelPresetID,
	type PatchModelPresetPayload,
	type PostModelPresetPayload,
} from '@/spec/modelpreset';

import { Dropdown } from '@/components/dropdown';
import { ModalBackdrop } from '@/components/modal_backdrop';
import { ReadOnlyValue } from '@/components/read_only_value';

const OUTPUT_FORMAT_NONE = '__none__' as const;
type OutputFormatKindSelection = OutputFormatKind | typeof OUTPUT_FORMAT_NONE;

const OUTPUT_VERBOSITY_NONE = '__none__' as const;
type OutputVerbositySelection = OutputVerbosity | typeof OUTPUT_VERBOSITY_NONE;

const outputFormatKindItems: Record<OutputFormatKindSelection, { isEnabled: boolean; displayName: string }> = {
	[OUTPUT_FORMAT_NONE]: { isEnabled: true, displayName: 'Default / Unset' },
	[OutputFormatKind.Text]: { isEnabled: true, displayName: 'Text' },
	[OutputFormatKind.JSONSchema]: { isEnabled: true, displayName: 'JSON Schema' },
};

const outputVerbosityItems: Record<OutputVerbositySelection, { isEnabled: boolean; displayName: string }> = {
	[OUTPUT_VERBOSITY_NONE]: { isEnabled: true, displayName: 'Default / Unset' },
	[OutputVerbosity.Low]: { isEnabled: true, displayName: 'Low' },
	[OutputVerbosity.Medium]: { isEnabled: true, displayName: 'Medium' },
	[OutputVerbosity.High]: { isEnabled: true, displayName: 'High' },
	[OutputVerbosity.Max]: { isEnabled: true, displayName: 'Max' },
};

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

const reasoningSummaryStyleItems: Record<ReasoningSummaryStyle, { isEnabled: boolean; displayName: string }> = {
	[ReasoningSummaryStyle.Auto]: { isEnabled: true, displayName: 'Auto' },
	[ReasoningSummaryStyle.Concise]: { isEnabled: true, displayName: 'Concise' },
	[ReasoningSummaryStyle.Detailed]: { isEnabled: true, displayName: 'Detailed' },
};

/** Defaults we apply while in Add mode. */
const AddModeDefaults = {
	stream: true,
	maxPromptLength: 2048,
	maxOutputLength: 1024,
	temperature: undefined as number | undefined,
	timeout: 300,
};

const DEFAULT_REASONING_TOKENS = 1024;

function arraysEqual(a: string[] = [], b: string[] = []): boolean {
	if (a.length !== b.length) return false;
	return a.every((value, index) => value === b[index]);
}

function parseOptionalNumber(val: string): number | undefined {
	const trimmed = val.trim();
	if (!trimmed) return undefined;
	const parsed = Number(trimmed);
	return Number.isNaN(parsed) ? undefined : parsed;
}

function buildOutputParamFromForm(
	outputFormatKind: OutputFormatKindSelection,
	outputVerbosity: OutputVerbositySelection,
	initialOutputParam?: OutputParam
): OutputParam | undefined {
	const formatKind = outputFormatKind !== OUTPUT_FORMAT_NONE ? outputFormatKind : undefined;
	const verbosity = outputVerbosity !== OUTPUT_VERBOSITY_NONE ? outputVerbosity : undefined;
	if (!formatKind && !verbosity) return undefined;

	const format =
		formatKind !== undefined
			? initialOutputParam?.format?.kind === formatKind
				? initialOutputParam.format
				: { kind: formatKind }
			: undefined;

	return {
		...(format && { format }),
		...(verbosity && { verbosity }),
	};
}

function outputParamsEqual(a?: OutputParam, b?: OutputParam): boolean {
	if (!a && !b) return true;
	if (!a || !b) return false;

	if (a.verbosity !== b.verbosity) return false;

	const aFormat = a.format;
	const bFormat = b.format;
	if (!!aFormat !== !!bFormat) return false;
	if (!aFormat && !bFormat) return true;
	if (!aFormat || !bFormat) return false;

	return (
		aFormat.kind === bFormat.kind &&
		JSON.stringify(aFormat.jsonSchemaParam ?? null) === JSON.stringify(bFormat.jsonSchemaParam ?? null)
	);
}

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
	reasoningSummaryStyle: ReasoningSummaryStyle;

	systemPrompt: string;
	timeout: string;

	outputFormatKind: OutputFormatKindSelection;
	outputVerbosity: OutputVerbositySelection;
	stopSequencesRaw: string;
}

type ModalMode = 'add' | 'edit' | 'view';

interface AddEditModelPresetModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (
		modelPresetID: ModelPresetID,
		modelData: PostModelPresetPayload | PatchModelPresetPayload
	) => Promise<void>;
	providerName: ProviderName;
	mode?: ModalMode;
	initialModelID?: ModelPresetID;
	initialData?: ModelPreset;
	existingModels: Record<ModelPresetID, ModelPreset>;
	allModelPresets: Record<ProviderName, Record<ModelPresetID, ModelPreset>>;
}

function buildReasoningFromForm(
	formData: ModelPresetFormData,
	initialReasoning?: ReasoningParam
): ReasoningParam | undefined {
	if (!formData.reasoningSupport) return undefined;

	const type = formData.reasoningType ?? ReasoningType.SingleWithLevels;
	return {
		type,
		level: formData.reasoningLevel ?? ReasoningLevel.Medium,
		tokens:
			type === ReasoningType.HybridWithTokens
				? parseOrDefault(formData.reasoningTokens ?? '', DEFAULT_REASONING_TOKENS)
				: initialReasoning?.type === ReasoningType.SingleWithLevels
					? initialReasoning.tokens
					: DEFAULT_REASONING_TOKENS,
		summaryStyle: formData.reasoningSummaryStyle,
	};
}

function reasoningEqual(a?: ReasoningParam, b?: ReasoningParam): boolean {
	if (!a && !b) return true;
	if (!a || !b) return false;
	return a.type === b.type && a.level === b.level && a.tokens === b.tokens && a.summaryStyle === b.summaryStyle;
}

interface AddEditModelPresetModalContentProps extends Omit<AddEditModelPresetModalProps, 'mode'> {
	mode: ModalMode;
}

function parseStopSequencesRaw(raw: string): string[] {
	const parts = raw
		.split(/\r?\n/)
		.flatMap(line => line.split(','))
		.map(s => s.trim())
		.filter(Boolean);

	return Array.from(new Set(parts));
}

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

function parseOrDefault(val: string, def: number): number {
	const n = Number(val);
	return val.trim() === '' || Number.isNaN(n) ? def : n;
}

function getInitialModelPresetFormData(mode: ModalMode, initialData?: ModelPreset): ModelPresetFormData {
	if ((mode === 'edit' || mode === 'view') && initialData) {
		return {
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
			reasoningSummaryStyle: initialData.reasoning?.summaryStyle ?? ReasoningSummaryStyle.Auto,
			systemPrompt: initialData.systemPrompt ?? '',
			timeout: initialData.timeout !== undefined ? String(initialData.timeout) : '',
			outputFormatKind: initialData.outputParam?.format?.kind ?? OUTPUT_FORMAT_NONE,
			outputVerbosity: initialData.outputParam?.verbosity ?? OUTPUT_VERBOSITY_NONE,
			stopSequencesRaw: initialData.stopSequences?.join('\n') ?? '',
		};
	}

	return {
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
		reasoningSummaryStyle: ReasoningSummaryStyle.Auto,
		reasoningTokens: '',
		systemPrompt: '',
		timeout: String(AddModeDefaults.timeout ?? ''),
		outputFormatKind: OUTPUT_FORMAT_NONE,
		outputVerbosity: OUTPUT_VERBOSITY_NONE,
		stopSequencesRaw: '',
	};
}

function AddEditModelPresetModalContent({
	onClose,
	onSubmit,
	providerName,
	mode,
	initialModelID,
	initialData,
	existingModels,
	allModelPresets,
}: AddEditModelPresetModalContentProps) {
	const isEditMode = mode === 'edit';
	const isViewMode = mode === 'view';
	const isReadOnly = isViewMode;

	const [modelPresetID, setModelPresetID] = useState<ModelPresetID>(() => initialModelID ?? '');
	const [formData, setFormData] = useState<ModelPresetFormData>(() => getInitialModelPresetFormData(mode, initialData));
	const [prefillMode, setPrefillMode] = useState(false);
	const [selectedPrefillKey, setSelectedPrefillKey] = useState<string | null>(null);
	const [errors, setErrors] = useState<ValidationErrors>({});
	const [isSubmitting, setIsSubmitting] = useState(false);

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const modelPresetIdInputRef = useRef<HTMLInputElement | null>(null);
	const modelNameInputRef = useRef<HTMLInputElement | null>(null);
	const isUnmountingRef = useRef(false);

	type PrefillKey = string;

	const prefillSourceMap = useMemo<Record<PrefillKey, ModelPreset>>(() => {
		const out: Record<PrefillKey, ModelPreset> = {};
		for (const [prov, presets] of Object.entries(allModelPresets)) {
			for (const [mid, mp] of Object.entries(presets)) {
				out[`${prov}::${mid}`] = mp;
			}
		}
		return out;
	}, [allModelPresets]);

	const prefillDropdownItems: Record<PrefillKey, { isEnabled: boolean; displayName: string }> = useMemo(() => {
		const out: Record<PrefillKey, { isEnabled: boolean; displayName: string }> = {};
		for (const [key, mp] of Object.entries(prefillSourceMap)) {
			const [prov] = key.split('::');
			out[key] = { isEnabled: true, displayName: `${prov} / ${mp.displayName || mp.id}` };
		}
		return out;
	}, [prefillSourceMap]);

	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			try {
				dialog.showModal();
			} catch {
				// Keep rendering safely if showModal fails.
			}
		}

		const focusTimer = window.setTimeout(() => {
			if (mode === 'add') {
				modelPresetIdInputRef.current?.focus();
			} else if (!isReadOnly) {
				modelNameInputRef.current?.focus();
			}
		}, 0);

		return () => {
			isUnmountingRef.current = true;
			window.clearTimeout(focusTimer);

			if (dialog.open) {
				dialog.close();
			}
		};
	}, [mode, isReadOnly]);

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

	const applyPrefill = (key: PrefillKey) => {
		const src = prefillSourceMap[key];
		if (!src) return;

		setFormData(prev => ({
			...prev,
			name: src.name,
			presetLabel: src.displayName,
			stream: src.stream ?? prev.stream,
			isEnabled: true,
			maxPromptLength: src.maxPromptLength !== undefined ? String(src.maxPromptLength) : prev.maxPromptLength,
			maxOutputLength: src.maxOutputLength !== undefined ? String(src.maxOutputLength) : prev.maxOutputLength,
			temperature: src.temperature !== undefined ? String(src.temperature) : prev.temperature,
			reasoningSupport: !!src.reasoning,
			reasoningType: src.reasoning?.type ?? prev.reasoningType,
			reasoningLevel: src.reasoning?.level ?? prev.reasoningLevel,
			reasoningTokens: src.reasoning?.tokens !== undefined ? String(src.reasoning.tokens) : prev.reasoningTokens,
			reasoningSummaryStyle: src.reasoning?.summaryStyle ?? prev.reasoningSummaryStyle,
			systemPrompt: src.systemPrompt ?? prev.systemPrompt,
			timeout: src.timeout !== undefined ? String(src.timeout) : prev.timeout,
			outputFormatKind: src.outputParam?.format?.kind ?? prev.outputFormatKind ?? OUTPUT_FORMAT_NONE,
			outputVerbosity: src.outputParam?.verbosity ?? prev.outputVerbosity ?? OUTPUT_VERBOSITY_NONE,
			stopSequencesRaw: src.stopSequences?.join('\n') ?? prev.stopSequencesRaw,
		}));
	};

	const buildPatchPayload = useCallback((): PatchModelPresetPayload => {
		if (!initialData) return {};

		const patch: PatchModelPresetPayload = {};
		const nextName = formData.name.trim();
		const nextDisplayName = formData.presetLabel.trim();

		if (nextName !== initialData.name) patch.name = nextName;
		if (nextDisplayName !== initialData.displayName) patch.displayName = nextDisplayName;
		if (formData.isEnabled !== initialData.isEnabled) patch.isEnabled = formData.isEnabled;
		if (formData.stream !== (initialData.stream ?? false)) patch.stream = formData.stream;

		const nextMaxPromptLength = parseOptionalNumber(formData.maxPromptLength);
		if (nextMaxPromptLength !== undefined && nextMaxPromptLength !== initialData.maxPromptLength) {
			patch.maxPromptLength = nextMaxPromptLength;
		}

		const nextMaxOutputLength = parseOptionalNumber(formData.maxOutputLength);
		if (nextMaxOutputLength !== undefined && nextMaxOutputLength !== initialData.maxOutputLength) {
			patch.maxOutputLength = nextMaxOutputLength;
		}

		const nextTemperature = parseOptionalNumber(formData.temperature);
		if (nextTemperature !== undefined && nextTemperature !== initialData.temperature) {
			patch.temperature = nextTemperature;
		}

		if (formData.systemPrompt !== (initialData.systemPrompt ?? '')) patch.systemPrompt = formData.systemPrompt;

		const nextTimeout = parseOptionalNumber(formData.timeout);
		if (nextTimeout !== undefined && nextTimeout !== initialData.timeout) patch.timeout = nextTimeout;

		const nextOutputParam = buildOutputParamFromForm(
			formData.outputFormatKind,
			formData.outputVerbosity,
			initialData.outputParam
		);
		if (!outputParamsEqual(nextOutputParam, initialData.outputParam) && nextOutputParam)
			patch.outputParam = nextOutputParam;

		const nextStopSequences = parseStopSequencesRaw(formData.stopSequencesRaw);
		if (!arraysEqual(nextStopSequences, initialData.stopSequences ?? [])) patch.stopSequences = nextStopSequences;

		const nextReasoning = buildReasoningFromForm(formData, initialData.reasoning);
		if (!reasoningEqual(nextReasoning, initialData.reasoning) && nextReasoning) patch.reasoning = nextReasoning;

		return patch;
	}, [formData, initialData]);

	const runValidation = useCallback((): ValidationErrors => {
		if (isReadOnly) return {};

		const nextErrors: ValidationErrors = {};

		if (!isEditMode) {
			const idTrim = modelPresetID.trim();
			if (!idTrim) nextErrors.modelPresetID = 'Model Preset ID is required.';
			else if (!/^[a-zA-Z0-9-]+$/.test(idTrim))
				nextErrors.modelPresetID = 'Only letters, numbers, and hyphens allowed.';
			else if (Object.prototype.hasOwnProperty.call(existingModels, idTrim))
				nextErrors.modelPresetID = 'Model Preset ID must be unique.';
		}

		if (!formData.name.trim()) nextErrors.name = 'Model Name is required.';
		if (!formData.presetLabel.trim()) nextErrors.presetLabel = 'Preset Label is required.';

		const maybeValidateNumeric = (field: ValidationField, range?: { min?: number; max?: number }) => {
			const value =
				field === 'modelPresetID' ? modelPresetID : (formData[field as keyof ModelPresetFormData] as string);
			nextErrors[field] = calcNumericError(field, value, range);
		};

		maybeValidateNumeric('temperature', { min: 0, max: 1 });
		maybeValidateNumeric('maxPromptLength', { min: 1 });
		maybeValidateNumeric('maxOutputLength', { min: 1 });
		maybeValidateNumeric('timeout', { min: 1 });

		if (formData.reasoningSupport && formData.reasoningType === ReasoningType.HybridWithTokens) {
			if ((formData.reasoningTokens ?? '').trim() === '') {
				nextErrors.reasoningTokens = 'Reasoning Tokens is required for Hybrid mode.';
			} else {
				maybeValidateNumeric('reasoningTokens', { min: 1024 });
			}
		}

		const hasTemperature = formData.temperature.trim() !== '';
		const effectiveHasTemperature = hasTemperature || (isEditMode && initialData?.temperature !== undefined);
		const effectiveHasReasoning = formData.reasoningSupport || (isEditMode && !!initialData?.reasoning);

		if (!effectiveHasReasoning && !effectiveHasTemperature) {
			nextErrors.temperature = 'Provide either Temperature or enable Reasoning for new presets.';
		}

		return Object.fromEntries(
			Object.entries(nextErrors).filter(([key]) => nextErrors[key as ValidationField] !== undefined)
		) as ValidationErrors;
	}, [
		existingModels,
		formData,
		initialData?.reasoning,
		initialData?.temperature,
		isEditMode,
		isReadOnly,
		modelPresetID,
	]);

	const isAllValid = useMemo(() => {
		if (isReadOnly) return true;
		return Object.keys(runValidation()).length === 0;
	}, [isReadOnly, runValidation]);

	const hasPatchChanges = useMemo(() => {
		if (!isEditMode) return true;
		return Object.keys(buildPatchPayload()).length > 0;
	}, [buildPatchPayload, isEditMode]);

	const numPlaceholder = (field: keyof typeof AddModeDefaults) => {
		const v = AddModeDefaults[field];
		return v === undefined || typeof v === 'object' ? 'Default: N/A' : `Default: ${String(v)}`;
	};

	const submitForm = async () => {
		if (isReadOnly) {
			requestClose();
			return;
		}

		const validationErrors = runValidation();
		setErrors(validationErrors);
		if (Object.keys(validationErrors).length !== 0) return;

		const finalModelPresetID = modelPresetID.trim();

		const payload: PostModelPresetPayload | PatchModelPresetPayload =
			mode === 'add'
				? {
						name: formData.name.trim(),
						slug: finalModelPresetID,
						displayName: formData.presetLabel.trim(),
						isEnabled: formData.isEnabled,
						stream: formData.stream,
						maxPromptLength: parseOrDefault(formData.maxPromptLength, AddModeDefaults.maxPromptLength),
						maxOutputLength: parseOrDefault(formData.maxOutputLength, AddModeDefaults.maxOutputLength),
						timeout: parseOrDefault(formData.timeout, AddModeDefaults.timeout),
						systemPrompt: formData.systemPrompt,
						...(formData.temperature.trim() !== '' && {
							temperature: parseOrDefault(formData.temperature, 0.1),
						}),
						...(buildOutputParamFromForm(formData.outputFormatKind, formData.outputVerbosity) && {
							outputParam: buildOutputParamFromForm(formData.outputFormatKind, formData.outputVerbosity),
						}),
						...(buildReasoningFromForm(formData) && {
							reasoning: buildReasoningFromForm(formData),
						}),
						...(parseStopSequencesRaw(formData.stopSequencesRaw).length > 0 && {
							stopSequences: parseStopSequencesRaw(formData.stopSequencesRaw),
						}),
					}
				: buildPatchPayload();

		if (mode === 'edit' && Object.keys(payload).length === 0) {
			return;
		}

		if ('outputParam' in payload && payload.outputParam === undefined) {
			delete payload.outputParam;
		}
		if ('reasoning' in payload && payload.reasoning === undefined) {
			delete payload.reasoning;
		}

		setIsSubmitting(true);
		try {
			await onSubmit(finalModelPresetID, payload);
			requestClose();
		} catch {
			// Keep the modal open so the user does not lose their form on failed save.
		} finally {
			setIsSubmitting(false);
		}
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		e.preventDefault();
		e.stopPropagation();
		void submitForm();
	};

	const handleChange = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
		if (isSubmitting) return;

		const target = e.target as HTMLInputElement;
		const { name, value, type, checked } = target;

		if (name === 'modelPresetID') {
			setModelPresetID(value);
			return;
		}

		const nextValue: string | boolean = type === 'checkbox' ? checked : value;
		const fieldName = name as keyof ModelPresetFormData;

		setFormData(prev => ({
			...prev,
			[fieldName]: nextValue,
		}));
	};

	const title = mode === 'add' ? 'Add Model Preset' : mode === 'edit' ? 'Edit Model Preset' : 'View Model Preset';

	return (
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={e => {
				if (!isReadOnly || isSubmitting) {
					e.preventDefault();
				}
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
							onClick={requestClose}
							aria-label="Close"
							disabled={isSubmitting}
						>
							<FiX size={12} />
						</button>
					</div>

					<form noValidate onSubmit={handleSubmit} className="space-y-4">
						{isEditMode && !isReadOnly && (
							<div className="border-info/30 bg-info/10 rounded-xl border px-3 py-2 text-xs">
								Only changed fields are sent while editing.
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
								<span className="label-text text-sm">Model Preset ID*</span>
								<span
									className="label-text-alt tooltip tooltip-right"
									data-tip="Unique identifier. Letters, numbers, hyphen."
								>
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								{mode === 'add' ? (
									<input
										ref={modelPresetIdInputRef}
										name="modelPresetID"
										type="text"
										className={`input input-bordered w-full rounded-xl ${errors.modelPresetID ? 'input-error' : ''}`}
										value={modelPresetID}
										onChange={handleChange}
										placeholder="e.g. gpt4-preset"
										autoComplete="off"
										spellCheck="false"
										disabled={isSubmitting}
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
									ref={modelNameInputRef}
									name="name"
									type="text"
									className={`input input-bordered w-full rounded-xl ${errors.name ? 'input-error' : ''}`}
									value={formData.name}
									onChange={handleChange}
									placeholder="e.g. gpt-4, claude-3-opus-20240229"
									autoComplete="off"
									spellCheck="false"
									readOnly={isReadOnly}
									disabled={isSubmitting}
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
									disabled={isSubmitting}
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
									disabled={isReadOnly || isSubmitting}
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
									disabled={isReadOnly || isSubmitting}
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
									disabled={isReadOnly || isSubmitting}
								/>
							</div>
						</div>

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
													setFormData(prev => ({ ...prev, reasoningType: t }));
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
														setFormData(prev => ({ ...prev, reasoningLevel: lvl }));
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
												disabled={isReadOnly || isSubmitting}
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

								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3">
										<span className="label-text text-sm">Reasoning Summary</span>
										<span
											className="label-text-alt tooltip tooltip-right"
											data-tip="Optional, reasoning summary style."
										>
											<FiHelpCircle size={12} />
										</span>
									</label>
									<div className="col-span-9">
										{isReadOnly ? (
											<ReadOnlyValue value={reasoningSummaryStyleItems[formData.reasoningSummaryStyle].displayName} />
										) : (
											<Dropdown<ReasoningSummaryStyle>
												dropdownItems={reasoningSummaryStyleItems}
												selectedKey={formData.reasoningSummaryStyle}
												onChange={style => {
													setFormData(prev => ({ ...prev, reasoningSummaryStyle: style }));
												}}
												filterDisabled={false}
												title="Select Reasoning Summary Style"
												getDisplayName={k => reasoningSummaryStyleItems[k].displayName}
											/>
										)}
									</div>
								</div>
							</>
						)}

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
									disabled={isSubmitting}
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
									disabled={isSubmitting}
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

						{(['maxPromptLength', 'maxOutputLength'] as const).map(field => (
							<div className="grid grid-cols-12 items-center gap-2" key={field}>
								<label className="label col-span-3">
									<span className="label-text text-sm">
										{field === 'maxPromptLength' ? 'Max Prompt Tokens' : 'Max Output Tokens'}
									</span>
								</label>
								<div className="col-span-9">
									<input
										name={field}
										type="text"
										className={`input input-bordered w-full rounded-xl ${errors[field] ? 'input-error' : ''}`}
										value={formData[field]}
										onChange={handleChange}
										placeholder={numPlaceholder(field)}
										spellCheck="false"
										readOnly={isReadOnly}
										disabled={isSubmitting}
									/>
									{errors[field] && (
										<div className="label">
											<span className="label-text-alt text-error flex items-center gap-1">
												<FiAlertCircle size={12} /> {errors[field]}
											</span>
										</div>
									)}
								</div>
							</div>
						))}

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
									disabled={isReadOnly || isSubmitting}
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-start gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Stop Sequences</span>
								<span className="label-text-alt tooltip tooltip-right" data-tip="One per line (commas also supported).">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								{isReadOnly ? (
									<ReadOnlyValue value={parseStopSequencesRaw(formData.stopSequencesRaw).join(', ') || '—'} />
								) : (
									<textarea
										name="stopSequencesRaw"
										className="textarea textarea-bordered h-20 w-full rounded-xl"
										value={formData.stopSequencesRaw}
										onChange={handleChange}
										placeholder={'e.g.\n###\n</final>'}
										spellCheck="false"
										disabled={isSubmitting}
									/>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Output Format Kind</span>
								<span
									className="label-text-alt tooltip tooltip-right"
									data-tip="Provider-specific output format identifier (if supported)."
								>
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								{isReadOnly ? (
									<ReadOnlyValue
										value={
											formData.outputFormatKind === OUTPUT_FORMAT_NONE
												? '—'
												: outputFormatKindItems[formData.outputFormatKind].displayName
										}
									/>
								) : (
									<Dropdown<OutputFormatKindSelection>
										dropdownItems={outputFormatKindItems}
										selectedKey={formData.outputFormatKind}
										onChange={k => {
											setFormData(prev => ({ ...prev, outputFormatKind: k }));
										}}
										filterDisabled={false}
										title="Select Output Format Kind"
										getDisplayName={k => outputFormatKindItems[k].displayName}
									/>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Output Verbosity/Effort</span>
							</label>
							<div className="col-span-9">
								{isReadOnly ? (
									<ReadOnlyValue
										value={
											formData.outputVerbosity === OUTPUT_VERBOSITY_NONE
												? '—'
												: outputVerbosityItems[formData.outputVerbosity].displayName
										}
									/>
								) : (
									<Dropdown<OutputVerbositySelection>
										dropdownItems={outputVerbosityItems}
										selectedKey={formData.outputVerbosity}
										onChange={v => {
											setFormData(prev => ({ ...prev, outputVerbosity: v }));
										}}
										filterDisabled={false}
										title="Select Output Verbosity"
										getDisplayName={k => outputVerbosityItems[k].displayName}
									/>
								)}
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
									disabled={!isAllValid || (isEditMode && !hasPatchChanges) || isSubmitting}
								>
									{isSubmitting ? 'Saving…' : isEditMode ? 'Save Changes' : 'Add Preset'}
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

export function AddEditModelPresetModal(props: AddEditModelPresetModalProps) {
	if (!props.isOpen) return null;
	if (typeof document === 'undefined' || !document.body) return null;

	const inferredMode: ModalMode = props.initialModelID ? 'edit' : 'add';
	const effectiveMode: ModalMode = props.mode ?? inferredMode;

	const modalKey =
		effectiveMode === 'add'
			? `add-model:${props.providerName}`
			: `${effectiveMode}:${props.providerName}:${props.initialData?.id ?? props.initialModelID ?? 'unknown-model'}`;

	return createPortal(<AddEditModelPresetModalContent key={modalKey} {...props} mode={effectiveMode} />, document.body);
}
