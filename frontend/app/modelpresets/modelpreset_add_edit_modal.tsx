import type { ChangeEvent, SubmitEventHandler } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiUpload, FiX } from 'react-icons/fi';

import type {
	CacheControl,
	CacheControlKind,
	CacheControlTTL,
	JSONSchemaParam,
	OutputParam,
	ProviderName,
	ProviderSDKType,
	ReasoningParam,
} from '@/spec/inference';
import {
	OutputFormatKind,
	OutputVerbosity,
	ReasoningLevel,
	ReasoningSummaryStyle,
	ReasoningType,
} from '@/spec/inference';
import type {
	ModelCapabilitiesOverride,
	ModelPreset,
	ModelPresetID,
	PatchModelPresetPayload,
	PostModelPresetPayload,
} from '@/spec/modelpreset';
import { DEFAULT_REASONING_TOKENS } from '@/spec/modelpreset';

import { arraysEqual, parseOptionalNumber } from '@/lib/obj_utils';

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

import type { CacheControlTTLSelection } from '@/modelpresets/lib/cache_control_utils';
import {
	buildCacheControlFromForm,
	buildCacheControlKindDropdownItems,
	buildCacheControlTTLDropdownItems,
	CACHE_CONTROL_TTL_PROVIDER_DEFAULT,
	getInitialCacheControlKind,
	getInitialCacheControlTTLSelection,
	resolveSupportedCacheControlKinds,
	resolveSupportedCacheControlTTLs,
} from '@/modelpresets/lib/cache_control_utils';
import {
	getTopLevelCacheControlCapabilities,
	mergeModelCapabilitiesOverride,
} from '@/modelpresets/lib/capabilities_override';
import { cacheControlEqual, outputParamsEqual, reasoningEqual } from '@/modelpresets/lib/type_utils';

const OUTPUT_FORMAT_NONE = '__none__' as const;
type OutputFormatKindSelection = OutputFormatKind | typeof OUTPUT_FORMAT_NONE;

const OUTPUT_VERBOSITY_NONE = '__none__' as const;
type OutputVerbositySelection = OutputVerbosity | typeof OUTPUT_VERBOSITY_NONE;

type OptionalBooleanSelection = '' | 'true' | 'false';
type JSONObjectParseResult = { ok: true; value: Record<string, unknown> } | { ok: false; error: string };

const OPTIONAL_BOOLEAN_ORDERED_KEYS: OptionalBooleanSelection[] = ['', 'true', 'false'];

function booleanToOptionalBoolean(value?: boolean): OptionalBooleanSelection {
	if (value === true) {
		return 'true';
	}
	if (value === false) {
		return 'false';
	}
	return '';
}

function optionalBooleanToBoolean(value: OptionalBooleanSelection): boolean | undefined {
	if (value === 'true') {
		return true;
	}
	if (value === 'false') {
		return false;
	}
	return undefined;
}

function isJSONObjectLike(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function tryParseJSONObject(raw: string): JSONObjectParseResult {
	try {
		const value = JSON.parse(raw);
		if (!isJSONObjectLike(value)) {
			return { ok: false, error: 'JSON schema body must be a JSON object.' };
		}
		return { ok: true, value };
	} catch {
		return { ok: false, error: 'JSON schema body must be valid JSON.' };
	}
}

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
	[OutputVerbosity.XHigh]: { isEnabled: true, displayName: 'XHigh' },
	[OutputVerbosity.Max]: { isEnabled: true, displayName: 'Max' },
};

const outputJSONSchemaStrictModeItems: Record<OptionalBooleanSelection, { isEnabled: boolean; displayName: string }> = {
	'': { isEnabled: true, displayName: 'Leave unset' },
	true: { isEnabled: true, displayName: 'Strict' },
	false: { isEnabled: true, displayName: 'Non-strict' },
};

function getOptionalBooleanDisplayName(value: OptionalBooleanSelection): string {
	return outputJSONSchemaStrictModeItems[value].displayName;
}

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
	[ReasoningLevel.Max]: { isEnabled: true, displayName: 'Max' },
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

const EXISTING_FIELD_CLEAR_UNSUPPORTED_MESSAGE =
	'Clearing an existing value is unavailable until the backend supports nullable patch fields or explicit clear flags. Enter a replacement value instead.';

function buildCacheControlFromModelPresetForm(
	formData: ModelPresetFormData,
	supportedKinds: CacheControlKind[],
	supportsTTL: boolean,
	supportsKey: boolean
): CacheControl | undefined {
	return buildCacheControlFromForm({
		enabled: formData.cacheControlEnabled,
		kind: formData.cacheControlKind,
		supportedKinds,
		ttlSelection: formData.cacheControlTTL,
		key: formData.cacheControlKey,
		supportsTTL,
		supportsKey,
	});
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
	outputJSONSchemaName: string;
	outputJSONSchemaDescription: string;
	outputJSONSchemaRaw: string;
	outputJSONSchemaStrictMode: OptionalBooleanSelection;

	cacheControlEnabled: boolean;
	cacheControlKind: CacheControlKind | '';
	cacheControlTTL: CacheControlTTLSelection;
	cacheControlKey: string;
	stopSequencesRaw: string;
}

function buildOutputParamFromForm(formData: ModelPresetFormData): OutputParam | undefined {
	const formatKind = formData.outputFormatKind !== OUTPUT_FORMAT_NONE ? formData.outputFormatKind : undefined;
	const verbosity = formData.outputVerbosity !== OUTPUT_VERBOSITY_NONE ? formData.outputVerbosity : undefined;
	if (!formatKind && !verbosity) {
		return undefined;
	}

	let format: OutputParam['format'] = undefined;

	if (formatKind === OutputFormatKind.JSONSchema) {
		const jsonSchemaParam: JSONSchemaParam = {
			name: formData.outputJSONSchemaName.trim(),
		};
		const description = formData.outputJSONSchemaDescription.trim();
		const schemaRaw = formData.outputJSONSchemaRaw.trim();
		const strict = optionalBooleanToBoolean(formData.outputJSONSchemaStrictMode);

		if (description) {
			jsonSchemaParam.description = description;
		}
		if (schemaRaw) {
			const parsed = tryParseJSONObject(schemaRaw);
			if (parsed.ok) {
				jsonSchemaParam.schema = parsed.value;
			}
		}
		if (strict !== undefined) {
			jsonSchemaParam.strict = strict;
		}

		format = { kind: formatKind, jsonSchemaParam };
	} else if (formatKind) {
		format = { kind: formatKind };
	}

	return {
		...(format ? { format } : {}),
		...(verbosity ? { verbosity } : {}),
	};
}

type ModalMode = 'add' | 'edit' | 'view';

interface AddEditModelPresetModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (
		modelPresetID: ModelPresetID,
		modelData: PostModelPresetPayload | PatchModelPresetPayload
	) => Promise<void>;
	providerSDKType: ProviderSDKType;
	providerCapabilitiesOverride?: ModelCapabilitiesOverride;
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
	if (!formData.reasoningSupport) {
		return undefined;
	}

	const type = formData.reasoningType ?? ReasoningType.SingleWithLevels;
	let tokens = DEFAULT_REASONING_TOKENS;
	if (type === ReasoningType.HybridWithTokens) {
		const n = parseOptionalNumber(formData.reasoningTokens ?? '', DEFAULT_REASONING_TOKENS);
		if (n) {
			tokens = n;
		}
	} else if (initialReasoning?.type === ReasoningType.SingleWithLevels) {
		tokens = initialReasoning.tokens;
	}
	return {
		type,
		level: formData.reasoningLevel ?? ReasoningLevel.Medium,
		tokens: tokens,
		summaryStyle: formData.reasoningSummaryStyle,
	};
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

	return [...new Set(parts)];
}

type ValidationField =
	| 'modelPresetID'
	| 'name'
	| 'presetLabel'
	| 'temperature'
	| 'maxPromptLength'
	| 'maxOutputLength'
	| 'timeout'
	| 'cacheControlKey'
	| 'reasoningTokens'
	| 'outputJSONSchemaName'
	| 'outputJSONSchemaRaw';

type ValidationErrors = Partial<Record<ValidationField, string>>;

const calcNumericError = (
	field: ValidationField,
	strVal: string,
	minOrRange?: { min?: number; max?: number; integer?: boolean }
): string | undefined => {
	if (strVal.trim() === '') {
		return;
	}
	const num = Number(strVal);
	if (Number.isNaN(num)) {
		return `${field} must be a valid number.`;
	}
	if (minOrRange?.integer && !Number.isInteger(num)) {
		return `${field} must be a whole number.`;
	}
	if (minOrRange?.min !== undefined && num < minOrRange.min) {
		return `${field} must be ≥ ${minOrRange.min}.`;
	}
	if (minOrRange?.max !== undefined && num > minOrRange.max) {
		return `${field} must be ≤ ${minOrRange.max}.`;
	}
};

function getInitialModelPresetFormData(
	mode: ModalMode,
	initialData: ModelPreset | undefined,
	supportedCacheKinds: CacheControlKind[],
	supportedCacheTTLs: CacheControlTTL[]
): ModelPresetFormData {
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
			outputJSONSchemaName: initialData.outputParam?.format?.jsonSchemaParam?.name ?? '',
			outputJSONSchemaDescription: initialData.outputParam?.format?.jsonSchemaParam?.description ?? '',
			outputJSONSchemaRaw: initialData.outputParam?.format?.jsonSchemaParam?.schema
				? JSON.stringify(initialData.outputParam.format.jsonSchemaParam.schema, null, 2)
				: '',
			outputJSONSchemaStrictMode: booleanToOptionalBoolean(initialData.outputParam?.format?.jsonSchemaParam?.strict),
			cacheControlEnabled: !!initialData.cacheControl,
			cacheControlKind: getInitialCacheControlKind(initialData.cacheControl, supportedCacheKinds),
			cacheControlTTL: getInitialCacheControlTTLSelection(initialData.cacheControl, supportedCacheTTLs),
			cacheControlKey: initialData.cacheControl?.key ?? '',
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
		outputJSONSchemaName: '',
		outputJSONSchemaDescription: '',
		outputJSONSchemaRaw: '',
		outputJSONSchemaStrictMode: '',
		cacheControlEnabled: false,
		cacheControlKind: supportedCacheKinds[0] ?? '',
		cacheControlTTL: CACHE_CONTROL_TTL_PROVIDER_DEFAULT,
		cacheControlKey: '',
		stopSequencesRaw: '',
	};
}

function AddEditModelPresetModalContent({
	onClose,
	onSubmit,
	providerName,
	providerSDKType,
	providerCapabilitiesOverride,
	mode,
	initialModelID,
	initialData,
	existingModels,
	allModelPresets,
}: AddEditModelPresetModalContentProps) {
	const isEditMode = mode === 'edit';
	const isViewMode = mode === 'view';
	const isReadOnly = isViewMode;

	const effectiveCapabilities = useMemo(
		() => mergeModelCapabilitiesOverride(providerCapabilitiesOverride, initialData?.capabilitiesOverride),
		[providerCapabilitiesOverride, initialData?.capabilitiesOverride]
	);
	const topLevelCacheCapabilities = useMemo(
		() => getTopLevelCacheControlCapabilities(providerSDKType, effectiveCapabilities),
		[effectiveCapabilities, providerSDKType]
	);
	const supportedCacheKinds = useMemo(
		() => resolveSupportedCacheControlKinds(topLevelCacheCapabilities?.supportedKinds, initialData?.cacheControl),
		[topLevelCacheCapabilities?.supportedKinds, initialData?.cacheControl]
	);
	const supportedCacheTTLs = useMemo(
		() => resolveSupportedCacheControlTTLs(topLevelCacheCapabilities?.supportedTTLs, initialData?.cacheControl),
		[topLevelCacheCapabilities?.supportedTTLs, initialData?.cacheControl]
	);
	const supportsManualCacheControl = supportedCacheKinds.length > 0;
	const supportsCacheTTL = topLevelCacheCapabilities?.supportsTTL !== false;
	const supportsCacheKey =
		topLevelCacheCapabilities?.supportsKey === true || Boolean(initialData?.cacheControl?.key?.trim());
	const canDisableCacheControl = mode === 'add' || !initialData?.cacheControl;
	const cannotClearExistingReasoning = isEditMode && initialData?.reasoning !== undefined;
	const cannotClearExistingOutput = isEditMode && initialData?.outputParam !== undefined;
	const cannotClearExistingCacheTTL = isEditMode && initialData?.cacheControl?.ttl !== undefined;
	const cannotClearExistingCacheKey = isEditMode && Boolean(initialData?.cacheControl?.key?.trim());
	const hasUnsupportedExistingClear =
		cannotClearExistingReasoning ||
		cannotClearExistingOutput ||
		Boolean(initialData?.cacheControl) ||
		initialData?.temperature !== undefined ||
		initialData?.timeout !== undefined;
	const cacheControlKindItems = useMemo(
		() => buildCacheControlKindDropdownItems(supportedCacheKinds),
		[supportedCacheKinds]
	);
	const cacheControlTTLItems = useMemo(
		() => buildCacheControlTTLDropdownItems(supportedCacheTTLs),
		[supportedCacheTTLs]
	);

	const [modelPresetID, setModelPresetID] = useState<ModelPresetID>(() => initialModelID ?? '');
	const [formData, setFormData] = useState<ModelPresetFormData>(() =>
		getInitialModelPresetFormData(mode, initialData, supportedCacheKinds, supportedCacheTTLs)
	);
	const [prefillMode, setPrefillMode] = useState(false);
	const [selectedPrefillKey, setSelectedPrefillKey] = useState<string | null>(null);
	const [errors, setErrors] = useState<ValidationErrors>({});
	const [isSubmitting, setIsSubmitting] = useState(false);
	const [submitError, setSubmitError] = useState('');

	const { dialogRef, requestClose, handleClose, handleCancel, unmountingRef } = useDialogController({
		onClose,
		blockCancel: !isReadOnly,
		isBusy: isSubmitting,
	});

	const modelPresetIdInputRef = useRef<HTMLInputElement | null>(null);
	const modelNameInputRef = useRef<HTMLInputElement | null>(null);

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

	const prefillKeys = useMemo(() => Object.keys(prefillSourceMap), [prefillSourceMap]);

	const prefillDropdownItems: Record<PrefillKey, { isEnabled: boolean; displayName: string }> = useMemo(() => {
		const out: Record<PrefillKey, { isEnabled: boolean; displayName: string }> = {};
		for (const [key, mp] of Object.entries(prefillSourceMap)) {
			const [prov] = key.split('::');
			out[key] = { isEnabled: true, displayName: `${prov} / ${mp.displayName || mp.id}` };
		}
		return out;
	}, [prefillSourceMap]);

	useEffect(() => {
		const focusTimer = window.setTimeout(() => {
			if (mode === 'add') {
				modelPresetIdInputRef.current?.focus();
			} else if (!isReadOnly) {
				modelNameInputRef.current?.focus();
			}
		}, 0);

		return () => {
			window.clearTimeout(focusTimer);
		};
	}, [mode, isReadOnly]);

	const applyPrefill = (key: PrefillKey) => {
		const src = prefillSourceMap[key];
		if (!src) {
			return;
		}

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
			outputFormatKind: src.outputParam?.format?.kind ?? OUTPUT_FORMAT_NONE,
			outputVerbosity: src.outputParam?.verbosity ?? OUTPUT_VERBOSITY_NONE,
			outputJSONSchemaName: src.outputParam?.format?.jsonSchemaParam?.name ?? '',
			outputJSONSchemaDescription: src.outputParam?.format?.jsonSchemaParam?.description ?? '',
			outputJSONSchemaRaw: src.outputParam?.format?.jsonSchemaParam?.schema
				? JSON.stringify(src.outputParam.format.jsonSchemaParam.schema, null, 2)
				: '',
			outputJSONSchemaStrictMode: booleanToOptionalBoolean(src.outputParam?.format?.jsonSchemaParam?.strict),
			cacheControlEnabled: supportsManualCacheControl && !!src.cacheControl,
			cacheControlKind: getInitialCacheControlKind(src.cacheControl, supportedCacheKinds),
			cacheControlTTL: getInitialCacheControlTTLSelection(src.cacheControl, supportedCacheTTLs),
			cacheControlKey:
				supportsCacheKey && src.cacheControl?.key
					? src.cacheControl.key
					: prev.cacheControlEnabled && !supportsCacheKey
						? ''
						: prev.cacheControlKey,
			stopSequencesRaw: src.stopSequences?.join('\n') ?? prev.stopSequencesRaw,
		}));
	};

	const buildPatchPayload = useCallback((): PatchModelPresetPayload => {
		if (!initialData) {
			return {};
		}

		const patch: PatchModelPresetPayload = {};
		const nextName = formData.name.trim();
		const nextDisplayName = formData.presetLabel.trim();

		if (nextName !== initialData.name) {
			patch.name = nextName;
		}
		if (nextDisplayName !== initialData.displayName) {
			patch.displayName = nextDisplayName;
		}
		if (formData.isEnabled !== initialData.isEnabled) {
			patch.isEnabled = formData.isEnabled;
		}
		if (formData.stream !== (initialData.stream ?? false)) {
			patch.stream = formData.stream;
		}

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

		if (formData.systemPrompt !== (initialData.systemPrompt ?? '')) {
			patch.systemPrompt = formData.systemPrompt;
		}

		const nextTimeout = parseOptionalNumber(formData.timeout);
		if (nextTimeout !== undefined && nextTimeout !== initialData.timeout) {
			patch.timeout = nextTimeout;
		}

		const nextCacheControl = buildCacheControlFromModelPresetForm(
			formData,
			supportedCacheKinds,
			supportsCacheTTL,
			supportsCacheKey
		);
		if (nextCacheControl && !cacheControlEqual(nextCacheControl, initialData.cacheControl)) {
			patch.cacheControl = nextCacheControl;
		}

		const nextOutputParam = buildOutputParamFromForm(formData);
		if (!outputParamsEqual(nextOutputParam, initialData.outputParam) && nextOutputParam) {
			patch.outputParam = nextOutputParam;
		}

		const nextStopSequences = parseStopSequencesRaw(formData.stopSequencesRaw);
		if (!arraysEqual(nextStopSequences, initialData.stopSequences ?? [])) {
			patch.stopSequences = nextStopSequences;
		}

		const nextReasoning = buildReasoningFromForm(formData, initialData.reasoning);
		if (!reasoningEqual(nextReasoning, initialData.reasoning) && nextReasoning) {
			patch.reasoning = nextReasoning;
		}

		return patch;
	}, [formData, initialData, supportedCacheKinds, supportsCacheKey, supportsCacheTTL]);

	/**
	 * Validates the form with optional overrides for the next-state values.
	 * Pass overrides when calling from handleChange so errors reflect the
	 * new value immediately (before React batches the state update).
	 */
	const computeValidation = useCallback(
		(overrides?: {
			formDataOverride?: ModelPresetFormData;
			modelPresetIDOverride?: ModelPresetID;
		}): ValidationErrors => {
			const fd = overrides?.formDataOverride ?? formData;
			const mpid = overrides?.modelPresetIDOverride ?? modelPresetID;

			if (isReadOnly) {
				return {};
			}

			const nextErrors: ValidationErrors = {};

			if (!isEditMode) {
				const idTrim = mpid.trim();
				if (!idTrim) {
					nextErrors.modelPresetID = 'Model Preset ID is required.';
				} else if (!/^[a-zA-Z0-9-]+$/.test(idTrim)) {
					nextErrors.modelPresetID = 'Only letters, numbers, and hyphens allowed.';
				} else if (Object.hasOwn(existingModels, idTrim)) {
					nextErrors.modelPresetID = 'Model Preset ID must be unique.';
				}
			}

			if (!fd.name.trim()) {
				nextErrors.name = 'Model Name is required.';
			}
			if (!fd.presetLabel.trim()) {
				nextErrors.presetLabel = 'Preset Label is required.';
			}

			const maybeValidateNumeric = (field: ValidationField, range?: { min?: number; max?: number }) => {
				const value = field === 'modelPresetID' ? mpid : (fd[field as keyof ModelPresetFormData] as string);
				nextErrors[field] = calcNumericError(field, value, range);
			};

			maybeValidateNumeric('temperature', { min: 0, max: 1 });
			maybeValidateNumeric('maxPromptLength', { min: 1 });
			maybeValidateNumeric('maxOutputLength', { min: 1 });
			maybeValidateNumeric('timeout', { min: 1 });

			if (isEditMode && initialData?.temperature !== undefined && !fd.temperature.trim()) {
				nextErrors.temperature = EXISTING_FIELD_CLEAR_UNSUPPORTED_MESSAGE;
			}

			if (isEditMode && initialData?.timeout !== undefined && !fd.timeout.trim()) {
				nextErrors.timeout = EXISTING_FIELD_CLEAR_UNSUPPORTED_MESSAGE;
			}

			if (cannotClearExistingCacheKey && !fd.cacheControlKey.trim()) {
				nextErrors.cacheControlKey = EXISTING_FIELD_CLEAR_UNSUPPORTED_MESSAGE;
			}

			if (fd.reasoningSupport && fd.reasoningType === ReasoningType.HybridWithTokens) {
				if ((fd.reasoningTokens ?? '').trim() === '') {
					nextErrors.reasoningTokens = 'Reasoning Tokens is required for Hybrid mode.';
				} else {
					maybeValidateNumeric('reasoningTokens', { min: 1024 });
				}
			}

			if (fd.outputFormatKind === OutputFormatKind.JSONSchema) {
				if (!fd.outputJSONSchemaName.trim()) {
					nextErrors.outputJSONSchemaName = 'JSON Schema output requires a schema name.';
				}

				const schemaRaw = fd.outputJSONSchemaRaw.trim();
				if (schemaRaw) {
					const parsed = tryParseJSONObject(schemaRaw);
					if (!parsed.ok) {
						nextErrors.outputJSONSchemaRaw = parsed.error;
					}
				}
			}

			const hasTemperature = fd.temperature.trim() !== '';
			const effectiveHasTemperature = hasTemperature || (isEditMode && initialData?.temperature !== undefined);
			const effectiveHasReasoning = fd.reasoningSupport || (isEditMode && !!initialData?.reasoning);

			if (!effectiveHasReasoning && !effectiveHasTemperature) {
				nextErrors.temperature = 'Provide either Temperature or enable Reasoning for new presets.';
			}

			return Object.fromEntries(
				Object.entries(nextErrors).filter(([key]) => nextErrors[key as ValidationField] !== undefined)
			) as ValidationErrors;
		},
		[
			cannotClearExistingCacheKey,
			existingModels,
			formData,
			initialData?.reasoning,
			initialData?.temperature,
			initialData?.timeout,
			isEditMode,
			isReadOnly,
			modelPresetID,
		]
	);

	const runValidation = useCallback(() => computeValidation(), [computeValidation]);

	const isAllValid = useMemo(() => {
		if (isReadOnly) {
			return true;
		}
		return Object.keys(runValidation()).length === 0;
	}, [isReadOnly, runValidation]);

	const hasPatchChanges = useMemo(() => {
		if (!isEditMode) {
			return true;
		}
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

		setSubmitError('');
		const nextCacheControl = buildCacheControlFromModelPresetForm(
			formData,
			supportedCacheKinds,
			supportsCacheTTL,
			supportsCacheKey
		);
		const nextOutputParam = buildOutputParamFromForm(formData);
		const nextReasoning = buildReasoningFromForm(formData);

		const validationErrors = runValidation();
		setErrors(validationErrors);
		if (Object.keys(validationErrors).length > 0) {
			return;
		}

		const finalModelPresetID = modelPresetID.trim();

		const payload: PostModelPresetPayload | PatchModelPresetPayload =
			mode === 'add'
				? {
						name: formData.name.trim(),
						slug: finalModelPresetID,
						displayName: formData.presetLabel.trim(),
						isEnabled: formData.isEnabled,
						stream: formData.stream,
						maxPromptLength: parseOptionalNumber(formData.maxPromptLength, AddModeDefaults.maxPromptLength),
						maxOutputLength: parseOptionalNumber(formData.maxOutputLength, AddModeDefaults.maxOutputLength),
						timeout: parseOptionalNumber(formData.timeout, AddModeDefaults.timeout),
						systemPrompt: formData.systemPrompt,
						...(nextCacheControl && { cacheControl: nextCacheControl }),
						...(formData.temperature.trim() !== '' && {
							temperature: parseOptionalNumber(formData.temperature, 0.1),
						}),
						...(nextOutputParam && {
							outputParam: nextOutputParam,
						}),
						...(nextReasoning && {
							reasoning: nextReasoning,
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
			requestClose(true);
		} catch (error) {
			if (!unmountingRef.current) {
				setSubmitError(error instanceof Error && error.message.trim() ? error.message : 'Failed to save model preset.');
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

	const handleChange = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
		if (isSubmitting) {
			return;
		}

		const target = e.target as HTMLInputElement;
		const { name, value, type, checked } = target;

		if (name === 'modelPresetID') {
			setModelPresetID(value);
			// Show ID field errors immediately without waiting for submit.
			setErrors(computeValidation({ modelPresetIDOverride: value }));
			return;
		}

		const nextValue: string | boolean = type === 'checkbox' ? checked : value;
		const fieldName = name as keyof ModelPresetFormData;
		// Compute next state synchronously so errors reflect the new value
		// before React processes the batched setFormData update.
		const nextFormData = { ...formData, [fieldName]: nextValue } as ModelPresetFormData;

		setFormData(prev => ({ ...prev, [fieldName]: nextValue }));
		setErrors(computeValidation({ formDataOverride: nextFormData }));
	};

	const title = mode === 'add' ? 'Add Model Preset' : mode === 'edit' ? 'Edit Model Preset' : 'View Model Preset';

	return (
		<dialog ref={dialogRef} className="modal" onClose={handleClose} onCancel={handleCancel}>
			<div className="modal-box bg-base-200 flex max-h-[85vh] w-[calc(100%-1rem)] max-w-4xl flex-col overflow-hidden rounded-2xl p-0">
				<ModalHeader
					title={title}
					description={
						<span>
							Provider: <span className="font-mono">{providerName}</span>
							{initialData
								? ` · ${initialData.isBuiltIn ? 'Built-in preset' : 'Custom preset'}`
								: ' · New custom preset'}
						</span>
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

						{isEditMode && !isReadOnly && (
							<div className="border-info/30 bg-info/10 rounded-xl border px-3 py-2 text-xs">
								Only changed fields are sent while editing.
							</div>
						)}

						{isEditMode && hasUnsupportedExistingClear ? (
							<div className="border-warning/40 bg-warning/10 rounded-xl border px-3 py-2 text-xs">
								Existing model configuration can be changed, but some values cannot be cleared yet. The current patch
								API omits undefined fields, so treating an empty form field as a clear operation would silently leave
								the stored value unchanged.
							</div>
						) : null}

						{mode === 'add' && (
							<ModalSection
								title="Copy an existing preset"
								description="Copy non-secret runtime defaults from another configured model preset."
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
												title={
													prefillKeys.length === 0 ? 'No existing model presets are available to copy.' : undefined
												}
											>
												<FiUpload size={14} />
												<span className="ml-1">Copy Existing Preset</span>
											</button>
										)}

										{prefillMode && (
											<>
												<Dropdown<PrefillKey>
													dropdownItems={prefillDropdownItems}
													orderedKeys={prefillKeys}
													selectedKey={selectedPrefillKey ?? ('' as PrefillKey)}
													onChange={key => {
														setSelectedPrefillKey(key);
														applyPrefill(key);
														setPrefillMode(false);
													}}
													disabled={prefillKeys.length === 0}
													filterDisabled={false}
													title="Select model preset to copy"
													getDisplayName={k => prefillDropdownItems[k]?.displayName ?? 'Select model preset to copy'}
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
							title="Identity"
							description="The preset ID is stable. The model name is the provider-facing API model identifier."
						>
							<ModalField
								label="Model Preset ID"
								htmlFor={mode === 'add' ? 'model-preset-id' : undefined}
								required
								hint="Unique identifier. Letters, numbers, and hyphens only."
								error={errors.modelPresetID}
							>
								{mode === 'add' ? (
									<input
										id="model-preset-id"
										ref={modelPresetIdInputRef}
										name="modelPresetID"
										type="text"
										className={`input w-full rounded-xl ${errors.modelPresetID ? 'input-error' : ''}`}
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
							</ModalField>

							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="text-sm">Model Name*</span>
									<span
										className="tooltip tooltip-right"
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
										className={`input w-full rounded-xl ${errors.name ? 'input-error' : ''}`}
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
											<span className="text-error flex items-center gap-1">
												<FiAlertCircle size={12} />
												{errors.name}
											</span>
										</div>
									)}
								</div>
							</div>

							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="text-sm">Preset Label*</span>
									<span className="tooltip tooltip-right" data-tip="Friendly name shown in the UI">
										<FiHelpCircle size={12} />
									</span>
								</label>
								<div className="col-span-9">
									<input
										name="presetLabel"
										type="text"
										className={`input w-full rounded-xl ${errors.presetLabel ? 'input-error' : ''}`}
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
											<span className="text-error flex items-center gap-1">
												<FiAlertCircle size={12} />
												{errors.presetLabel}
											</span>
										</div>
									)}
								</div>
							</div>
						</ModalSection>

						<ModalSection title="Presentation" description="Controls whether this preset is selectable and streamed.">
							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3 cursor-pointer">
									<span className="text-sm">Enabled</span>
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
									<span className="text-sm">Streaming</span>
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
						</ModalSection>

						<ModalSection
							title="Reasoning"
							description="Enable provider-specific reasoning configuration when supported."
						>
							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3 cursor-pointer">
									<span className="text-sm">Supports Reasoning</span>
									<span className="tooltip tooltip-right" data-tip="If enabled, configure below">
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
										disabled={isReadOnly || isSubmitting || cannotClearExistingReasoning}
										title={
											cannotClearExistingReasoning
												? 'Removing stored reasoning configuration is not supported by the current patch API.'
												: undefined
										}
									/>
									{cannotClearExistingReasoning ? (
										<span className="text-base-content/70 ml-2 text-xs">
											Existing reasoning configuration cannot be cleared yet.
										</span>
									) : null}
								</div>
							</div>

							{formData.reasoningSupport && (
								<>
									<div className="grid grid-cols-12 items-center gap-2">
										<label className="label col-span-3">
											<span className="text-sm">Reasoning Type</span>
										</label>
										<div className="col-span-9">
											{isReadOnly ? (
												<ReadOnlyValue
													value={
														reasoningTypeItems[formData.reasoningType ?? ReasoningType.SingleWithLevels].displayName
													}
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
												<span className="text-sm">Reasoning Level</span>
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
												<span className="text-sm">Reasoning Tokens</span>
											</label>
											<div className="col-span-9">
												<input
													name="reasoningTokens"
													type="text"
													className={`input w-full rounded-xl ${errors.reasoningTokens ? 'input-error' : ''}`}
													value={formData.reasoningTokens}
													onChange={handleChange}
													placeholder="e.g. 1024"
													spellCheck="false"
													disabled={isReadOnly || isSubmitting}
												/>
												{errors.reasoningTokens && (
													<div className="label">
														<span className="text-error flex items-center gap-1">
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
											<span className="text-sm">Reasoning Summary</span>
											<span className="tooltip tooltip-right" data-tip="Optional, reasoning summary style.">
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
						</ModalSection>

						<ModalSection title="Runtime limits" description="Set request limits and the normal sampling parameters.">
							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="text-sm">Temperature (0-1)</span>
									<span className="tooltip tooltip-right" data-tip="This or Reasoning is needed">
										<FiHelpCircle size={12} />
									</span>
								</label>
								<div className="col-span-9">
									<input
										name="temperature"
										type="text"
										className={`input w-full rounded-xl ${errors.temperature ? 'input-error' : ''}`}
										value={formData.temperature}
										onChange={handleChange}
										placeholder={numPlaceholder('temperature')}
										spellCheck="false"
										readOnly={isReadOnly}
										disabled={isSubmitting}
									/>
									{errors.temperature && (
										<div className="label">
											<span className="text-error flex items-center gap-1">
												<FiAlertCircle size={12} /> {errors.temperature}
											</span>
										</div>
									)}
								</div>
							</div>

							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="text-sm">Timeout (seconds)</span>
								</label>
								<div className="col-span-9">
									<input
										name="timeout"
										type="text"
										className={`input w-full rounded-xl ${errors.timeout ? 'input-error' : ''}`}
										value={formData.timeout}
										onChange={handleChange}
										placeholder={numPlaceholder('timeout')}
										spellCheck="false"
										readOnly={isReadOnly}
										disabled={isSubmitting}
									/>
									{errors.timeout && (
										<div className="label">
											<span className="text-error flex items-center gap-1">
												<FiAlertCircle size={12} /> {errors.timeout}
											</span>
										</div>
									)}
								</div>
							</div>

							{(['maxPromptLength', 'maxOutputLength'] as const).map(field => (
								<div className="grid grid-cols-12 items-center gap-2" key={field}>
									<label className="label col-span-3">
										<span className="text-sm">
											{field === 'maxPromptLength' ? 'Max Prompt Tokens' : 'Max Output Tokens'}
										</span>
									</label>
									<div className="col-span-9">
										<input
											name={field}
											type="text"
											className={`input w-full rounded-xl ${errors[field] ? 'input-error' : ''}`}
											value={formData[field]}
											onChange={handleChange}
											placeholder={numPlaceholder(field)}
											spellCheck="false"
											readOnly={isReadOnly}
											disabled={isSubmitting}
										/>
										{errors[field] && (
											<div className="label">
												<span className="text-error flex items-center gap-1">
													<FiAlertCircle size={12} /> {errors[field]}
												</span>
											</div>
										)}
									</div>
								</div>
							))}
						</ModalSection>

						{supportsManualCacheControl && (
							<ModalSection
								title="Cache control"
								description="Configure request-level cache behavior when the selected provider supports it."
							>
								<div className="grid grid-cols-12 items-center gap-2">
									<label className="label col-span-3 cursor-pointer">
										<span className="text-sm">Cache Control</span>
										<span
											className="tooltip tooltip-right"
											data-tip="Request-level manual cache control for this preset."
										>
											<FiHelpCircle size={12} />
										</span>
									</label>
									<div className="col-span-9 flex items-center gap-2">
										<input
											type="checkbox"
											name="cacheControlEnabled"
											className="toggle toggle-accent disabled:opacity-80"
											checked={formData.cacheControlEnabled}
											onChange={() => {
												setFormData(prev => ({
													...prev,
													cacheControlEnabled: !prev.cacheControlEnabled,
													cacheControlKind: prev.cacheControlKind || supportedCacheKinds[0] || '',
												}));
											}}
											disabled={isReadOnly || isSubmitting || !canDisableCacheControl}
										/>
										{!canDisableCacheControl && (
											<span className="text-xs opacity-70">
												Existing cache control can be changed, but clearing it is not supported by the current patch
												API.
											</span>
										)}
									</div>
								</div>

								{formData.cacheControlEnabled && (
									<>
										<div className="grid grid-cols-12 items-center gap-2">
											<label className="label col-span-3">
												<span className="text-sm">Cache Kind</span>
											</label>
											<div className="col-span-9">
												{isReadOnly ? (
													<ReadOnlyValue
														value={
															cacheControlKindItems[formData.cacheControlKind as CacheControlKind]?.displayName ?? '—'
														}
													/>
												) : (
													<Dropdown<CacheControlKind>
														dropdownItems={cacheControlKindItems}
														selectedKey={(formData.cacheControlKind || supportedCacheKinds[0]) as CacheControlKind}
														onChange={kind => {
															setFormData(prev => ({ ...prev, cacheControlKind: kind }));
														}}
														filterDisabled={false}
														title="Select Cache Kind"
														getDisplayName={k => cacheControlKindItems[k].displayName}
													/>
												)}
											</div>
										</div>

										<div className="grid grid-cols-12 items-center gap-2">
											<label className="label col-span-3">
												<span className="text-sm">Cache TTL</span>
											</label>
											<div className="col-span-9">
												{isReadOnly ? (
													<ReadOnlyValue value={cacheControlTTLItems[formData.cacheControlTTL].displayName} />
												) : (
													<Dropdown<CacheControlTTLSelection>
														dropdownItems={cacheControlTTLItems}
														selectedKey={formData.cacheControlTTL}
														onChange={ttl => {
															if (cannotClearExistingCacheTTL && ttl === CACHE_CONTROL_TTL_PROVIDER_DEFAULT) {
																return;
															}
															setFormData(prev => ({ ...prev, cacheControlTTL: ttl }));
														}}
														filterDisabled={false}
														title="Select Cache TTL"
														getDisplayName={k => cacheControlTTLItems[k].displayName}
													/>
												)}
											</div>
											{cannotClearExistingCacheTTL ? (
												<div className="text-base-content/70 col-span-12 text-xs sm:col-span-9 sm:col-start-4">
													An existing explicit cache TTL cannot be cleared to provider default until clear semantics are
													supported by the API.
												</div>
											) : null}
										</div>

										{supportsCacheKey && (
											<div className="grid grid-cols-12 items-center gap-2">
												<label className="label col-span-3">
													<span className="text-sm">Cache Key</span>
												</label>
												<div className="col-span-9">
													<input
														name="cacheControlKey"
														type="text"
														className={`input w-full rounded-xl ${errors.cacheControlKey ? 'input-error' : ''}`}
														value={formData.cacheControlKey}
														onChange={handleChange}
														placeholder="Optional request cache key"
														autoComplete="off"
														spellCheck="false"
														readOnly={isReadOnly}
														disabled={isSubmitting}
													/>
													{errors.cacheControlKey ? (
														<div className="label">
															<span className="text-error text-xs">{errors.cacheControlKey}</span>
														</div>
													) : null}
												</div>
											</div>
										)}
									</>
								)}
							</ModalSection>
						)}

						<ModalSection
							title="Prompt and stopping behavior"
							description="Set the optional default system prompt and stop sequences used for this model preset."
						>
							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="text-sm">System Prompt</span>
								</label>
								<div className="col-span-9">
									<textarea
										name="systemPrompt"
										className="textarea h-24 w-full rounded-xl"
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
									<span className="text-sm">Stop Sequences</span>
									<span className="tooltip tooltip-right" data-tip="One per line (commas also supported).">
										<FiHelpCircle size={12} />
									</span>
								</label>
								<div className="col-span-9">
									{isReadOnly ? (
										<ReadOnlyValue value={parseStopSequencesRaw(formData.stopSequencesRaw).join(', ') || '—'} />
									) : (
										<textarea
											name="stopSequencesRaw"
											className="textarea h-20 w-full rounded-xl"
											value={formData.stopSequencesRaw}
											onChange={handleChange}
											placeholder={'e.g.\n###\n</final>'}
											spellCheck="false"
											disabled={isSubmitting}
										/>
									)}
								</div>
							</div>
						</ModalSection>

						<ModalSection
							title="Output behavior"
							description="Configure provider output formatting and verbosity defaults."
						>
							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="text-sm">Output Format Kind</span>
									<span
										className="tooltip tooltip-right"
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
												setFormData(previous => {
													if (
														cannotClearExistingOutput &&
														k === OUTPUT_FORMAT_NONE &&
														previous.outputVerbosity === OUTPUT_VERBOSITY_NONE
													) {
														return previous;
													}
													return { ...previous, outputFormatKind: k };
												});
											}}
											filterDisabled={false}
											title="Select Output Format Kind"
											getDisplayName={k => outputFormatKindItems[k].displayName}
										/>
									)}
									{cannotClearExistingOutput ? (
										<div className="label">
											<span className="text-base-content/70 text-xs">
												At least one existing output setting must remain configured until the API supports clearing.
											</span>
										</div>
									) : null}
								</div>
							</div>

							{formData.outputFormatKind === OutputFormatKind.JSONSchema ? (
								<>
									<ModalField
										label="JSON Schema Name"
										htmlFor="model-output-json-schema-name"
										required
										error={errors.outputJSONSchemaName}
									>
										{isReadOnly ? (
											<ReadOnlyValue value={formData.outputJSONSchemaName || '—'} />
										) : (
											<input
												id="model-output-json-schema-name"
												name="outputJSONSchemaName"
												type="text"
												className={`input w-full rounded-xl ${errors.outputJSONSchemaName ? 'input-error' : ''}`}
												value={formData.outputJSONSchemaName}
												onChange={handleChange}
												autoComplete="off"
												spellCheck="false"
												disabled={isSubmitting}
											/>
										)}
									</ModalField>

									<ModalField label="Strict Mode">
										{isReadOnly ? (
											<ReadOnlyValue value={getOptionalBooleanDisplayName(formData.outputJSONSchemaStrictMode)} />
										) : (
											<Dropdown<OptionalBooleanSelection>
												dropdownItems={outputJSONSchemaStrictModeItems}
												orderedKeys={OPTIONAL_BOOLEAN_ORDERED_KEYS}
												selectedKey={formData.outputJSONSchemaStrictMode}
												onChange={outputJSONSchemaStrictMode => {
													setFormData(previous => ({
														...previous,
														outputJSONSchemaStrictMode,
													}));
												}}
												filterDisabled={false}
												title="Select JSON Schema strictness"
												getDisplayName={getOptionalBooleanDisplayName}
												disabled={isSubmitting}
											/>
										)}
									</ModalField>

									<ModalField label="JSON Schema Description" htmlFor="model-output-json-schema-description">
										{isReadOnly ? (
											<ReadOnlyValue value={formData.outputJSONSchemaDescription || '—'} />
										) : (
											<input
												id="model-output-json-schema-description"
												name="outputJSONSchemaDescription"
												type="text"
												className="input w-full rounded-xl"
												value={formData.outputJSONSchemaDescription}
												onChange={handleChange}
												autoComplete="off"
												spellCheck="false"
												disabled={isSubmitting}
											/>
										)}
									</ModalField>

									<ModalField
										label="JSON Schema Body"
										htmlFor="model-output-json-schema-body"
										error={errors.outputJSONSchemaRaw}
										align="start"
									>
										{isReadOnly ? (
											<pre className="bg-base-300 max-h-56 overflow-auto rounded-xl p-3 text-xs whitespace-pre-wrap">
												{formData.outputJSONSchemaRaw || '—'}
											</pre>
										) : (
											<textarea
												id="model-output-json-schema-body"
												name="outputJSONSchemaRaw"
												className={`textarea h-40 w-full rounded-xl font-mono text-xs ${
													errors.outputJSONSchemaRaw ? 'textarea-error' : ''
												}`}
												value={formData.outputJSONSchemaRaw}
												onChange={handleChange}
												spellCheck="false"
												placeholder='{"type":"object","properties":{}}'
												disabled={isSubmitting}
											/>
										)}
									</ModalField>
								</>
							) : null}

							<div className="grid grid-cols-12 items-center gap-2">
								<label className="label col-span-3">
									<span className="text-sm">Output Verbosity/Effort</span>
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
												setFormData(previous => {
													if (
														cannotClearExistingOutput &&
														v === OUTPUT_VERBOSITY_NONE &&
														previous.outputFormatKind === OUTPUT_FORMAT_NONE
													) {
														return previous;
													}
													return { ...previous, outputVerbosity: v };
												});
											}}
											filterDisabled={false}
											title="Select Output Verbosity"
											getDisplayName={k => outputVerbosityItems[k].displayName}
										/>
									)}
								</div>
							</div>
						</ModalSection>

						{isViewMode && initialData ? (
							<ModalSection title="Metadata">
								<ManagementInfoGrid>
									<ManagementInfoRow label="Provider" mono>
										{providerName}
									</ManagementInfoRow>
									<ManagementInfoRow label="Preset ID" mono>
										{initialData.id}
									</ManagementInfoRow>
									<ManagementInfoRow label="Slug" mono>
										{initialData.slug}
									</ManagementInfoRow>
									<ManagementInfoRow label="Built-in">{initialData.isBuiltIn ? 'Yes' : 'No'}</ManagementInfoRow>
									<ManagementInfoRow label="Created">{initialData.createdAt}</ManagementInfoRow>
									<ManagementInfoRow label="Modified">{initialData.modifiedAt}</ManagementInfoRow>
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
								disabled={!isAllValid || (isEditMode && !hasPatchChanges) || isSubmitting}
							>
								{isSubmitting ? 'Saving…' : isEditMode ? 'Save Changes' : 'Add Preset'}
							</button>
						)}
					</ModalActions>
				</form>
			</div>
			<ModalBackdrop enabled={isReadOnly} />
		</dialog>
	);
}

export function AddEditModelPresetModal(props: AddEditModelPresetModalProps) {
	if (!props.isOpen) {
		return null;
	}
	if (typeof document === 'undefined' || !document.body) {
		return null;
	}

	const inferredMode: ModalMode = props.initialModelID ? 'edit' : 'add';
	const effectiveMode: ModalMode = props.mode ?? inferredMode;

	const modalKey =
		effectiveMode === 'add'
			? `add-model:${props.providerName}`
			: `${effectiveMode}:${props.providerName}:${props.initialData?.id ?? props.initialModelID ?? 'unknown-model'}`;

	return createPortal(<AddEditModelPresetModalContent key={modalKey} {...props} mode={effectiveMode} />, document.body);
}
