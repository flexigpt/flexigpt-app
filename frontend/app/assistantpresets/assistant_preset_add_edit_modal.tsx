import { type SubmitEventHandler, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiHelpCircle, FiRefreshCw, FiX } from 'react-icons/fi';

import type { AssistantPreset, AssistantPresetStartingModelPresetPatch } from '@/spec/assistantpreset';
import type { JSONSchemaParam, OutputFormat, OutputParam } from '@/spec/inference';
import { OutputFormatKind, ReasoningLevel, ReasoningType } from '@/spec/inference';
import type { AssistantModelPresetOption } from '@/spec/modelpreset';
import type { AssistantInstructionTemplateOption } from '@/spec/prompt';
import type { AssistantSkillOption } from '@/spec/skill';
import { type AssistantToolOption, ToolImplType, ToolStoreChoiceType } from '@/spec/tool';

import { parseOptionalNumber, parsePositiveInteger } from '@/lib/obj_utils';
import { validateSlug } from '@/lib/text_utils';
import { DEFAULT_SEMVER, isSemverVersion, suggestNextMinorVersion } from '@/lib/version_utils';

import { Dropdown } from '@/components/dropdown';
import { ModalBackdrop } from '@/components/modal_backdrop';

import { AssistantPresetModelPatchEditor } from '@/assistantpresets/components/model_patch_editor';
import { OrderedRefSelectionSection } from '@/assistantpresets/components/ordered_ref_selection_section';
import { ToolSelectionSection } from '@/assistantpresets/components/tool_selection_section';
import {
	type AssistantPresetEditorCatalog,
	loadAssistantPresetEditorCatalog,
} from '@/assistantpresets/lib/assistant_preset_catalog';
import type {
	AssistantPresetFormData,
	ErrorState,
	ModalMode,
	ModelPatchFormData,
	OrderedDisplayItem,
	PresetItem,
	SimpleSelectableOption,
	ToolSelectionDisplayItem,
	TriStateBoolean,
} from '@/assistantpresets/lib/assistant_preset_editor_types';
import {
	type AssistantPresetUpsertInput,
	buildModelPresetRefKey,
	buildSkillRefKey,
	buildToolRefKey,
	clonePromptTemplateRef,
	cloneSkillRef,
	formatDateish,
	hasAssistantPresetModelPatch,
} from '@/assistantpresets/lib/assistant_preset_utils';
import { buildEffectiveModelParamFromModelPreset } from '@/modelpresets/lib/modelpreset_effective_defaults';
import { buildPromptTemplateRefKey } from '@/prompts/lib/prompt_template_ref';

interface AddEditAssistantPresetModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (presetData: AssistantPresetUpsertInput) => Promise<void>;
	initialData?: PresetItem;
	existingPresets: PresetItem[];
	mode?: ModalMode;
}

type JSONParseResult = { ok: true; value: unknown } | { ok: false; error: string };

const EMPTY_ERROR_STATE: ErrorState = {};
const EMPTY_MODEL_OPTIONS: AssistantModelPresetOption[] = [];
const EMPTY_INSTRUCTION_OPTIONS: AssistantInstructionTemplateOption[] = [];
const EMPTY_TOOL_OPTIONS: AssistantToolOption[] = [];
const EMPTY_SKILL_OPTIONS: AssistantSkillOption[] = [];
const TRI_STATE_DROPDOWN_ITEMS: Record<TriStateBoolean, { isEnabled: boolean }> = {
	'': { isEnabled: true },
	true: { isEnabled: true },
	false: { isEnabled: true },
};
const TRI_STATE_ORDERED_KEYS: TriStateBoolean[] = ['', 'true', 'false'];

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

function booleanToTriState(value?: boolean): TriStateBoolean {
	if (value === true) return 'true';
	if (value === false) return 'false';
	return '';
}

function triStateToBoolean(value: TriStateBoolean): boolean | undefined {
	if (value === 'true') return true;
	if (value === 'false') return false;
	return undefined;
}

function moveItem<T>(items: readonly T[], from: number, to: number): T[] {
	if (to < 0 || to >= items.length || from < 0 || from >= items.length) {
		return [...items];
	}

	const next = [...items];
	const [item] = next.splice(from, 1);

	if (item === undefined) {
		return next;
	}

	next.splice(to, 0, item);
	return next;
}

function removeItemAtIndex<T>(items: readonly T[], index: number): T[] {
	return items.filter((_, itemIndex) => itemIndex !== index);
}

function updateItemAtIndex<T>(items: readonly T[], index: number, updater: (item: T) => T): T[] {
	return items.map((item, itemIndex) => (itemIndex === index ? updater(item) : item));
}

function isJSONObjectLike(value: unknown): value is Record<string, any> {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function tryParseJSONRaw(raw: string): JSONParseResult {
	try {
		return {
			ok: true,
			value: JSON.parse(raw),
		};
	} catch {
		return {
			ok: false,
			error: 'Must be valid JSON.',
		};
	}
}

function getTriStateLabel(value: TriStateBoolean): string {
	if (value === 'true') return 'Force On';
	if (value === 'false') return 'Force Off';
	return 'Not Set';
}

function getToolAutoExecuteLabel(value: TriStateBoolean): string {
	if (value === 'true') return 'Force On';
	if (value === 'false') return 'Force Off';
	return 'Tool Default';
}

function getIncludeModelSystemPromptLabel(value: TriStateBoolean): string {
	if (value === 'true') return 'Include';
	if (value === 'false') return 'Do Not Include';
	return 'Not Set';
}

function getSuggestedNextVersion(initialData: PresetItem, existingPresets: PresetItem[]): string {
	return suggestNextMinorVersion(
		initialData.preset.version,
		existingPresets.filter(item => item.preset.slug === initialData.preset.slug).map(item => item.preset.version)
	).suggested;
}

function buildModelPatchSeedFormData(modelOption?: AssistantModelPresetOption): ModelPatchFormData {
	const base = getDefaultModelPatchFormData();
	if (!modelOption) {
		return base;
	}

	const effective = buildEffectiveModelParamFromModelPreset(modelOption.modelPreset);

	return {
		...base,
		enabled: true,
		stream: booleanToTriState(effective.stream),
		maxPromptLength: String(effective.maxPromptLength),
		maxOutputLength: String(effective.maxOutputLength),
		temperature: effective.temperature !== undefined ? String(effective.temperature) : '',
		timeout: String(effective.timeout),
		stopSequencesText: (effective.stopSequences ?? []).join('\n'),
		additionalParametersRawJSON: effective.additionalParametersRawJSON ?? '',

		reasoningEnabled: effective.reasoning !== undefined,
		reasoningType: effective.reasoning?.type ?? base.reasoningType,
		reasoningLevel: effective.reasoning?.level ?? base.reasoningLevel,
		reasoningTokens:
			effective.reasoning?.tokens !== undefined ? String(effective.reasoning.tokens) : base.reasoningTokens,
		reasoningSummaryStyle: effective.reasoning?.summaryStyle ?? '',

		outputEnabled: effective.outputParam !== undefined,
		outputVerbosity: effective.outputParam?.verbosity ?? '',
		outputFormatEnabled: effective.outputParam?.format !== undefined,
		outputFormatKind: effective.outputParam?.format?.kind ?? base.outputFormatKind,
		outputJSONSchemaName: effective.outputParam?.format?.jsonSchemaParam?.name ?? '',
		outputJSONSchemaDescription: effective.outputParam?.format?.jsonSchemaParam?.description ?? '',
		outputJSONSchemaRaw: effective.outputParam?.format?.jsonSchemaParam?.schema
			? JSON.stringify(effective.outputParam.format.jsonSchemaParam.schema, null, 2)
			: '',
		outputJSONSchemaStrictMode: booleanToTriState(effective.outputParam?.format?.jsonSchemaParam?.strict),
	};
}

function getDefaultModelPatchFormData(): ModelPatchFormData {
	return {
		enabled: false,
		stream: '',
		maxPromptLength: '',
		maxOutputLength: '',
		temperature: '',
		timeout: '',
		stopSequencesText: '',
		additionalParametersRawJSON: '',

		reasoningEnabled: false,
		reasoningType: ReasoningType.SingleWithLevels,
		reasoningLevel: ReasoningLevel.Medium,
		reasoningTokens: '1024',
		reasoningSummaryStyle: '',

		outputEnabled: false,
		outputVerbosity: '',
		outputFormatEnabled: false,
		outputFormatKind: OutputFormatKind.Text,
		outputJSONSchemaName: '',
		outputJSONSchemaDescription: '',
		outputJSONSchemaRaw: '',
		outputJSONSchemaStrictMode: '',
	};
}

function getInitialModelPatchFormData(patch?: AssistantPreset['startingModelPresetPatch']): ModelPatchFormData {
	const base = getDefaultModelPatchFormData();

	if (!patch) {
		return base;
	}

	return {
		enabled: hasAssistantPresetModelPatch(patch),
		stream: booleanToTriState(patch.stream),
		maxPromptLength: patch.maxPromptLength !== undefined ? String(patch.maxPromptLength) : '',
		maxOutputLength: patch.maxOutputLength !== undefined ? String(patch.maxOutputLength) : '',
		temperature: patch.temperature !== undefined ? String(patch.temperature) : '',
		timeout: patch.timeout !== undefined ? String(patch.timeout) : '',
		stopSequencesText: (patch.stopSequences ?? []).join('\n'),
		additionalParametersRawJSON: patch.additionalParametersRawJSON ?? '',

		reasoningEnabled: patch.reasoning !== undefined,
		reasoningType: patch.reasoning?.type ?? base.reasoningType,
		reasoningLevel: patch.reasoning?.level ?? base.reasoningLevel,
		reasoningTokens: patch.reasoning?.tokens !== undefined ? String(patch.reasoning.tokens) : base.reasoningTokens,
		reasoningSummaryStyle: patch.reasoning?.summaryStyle ?? '',

		outputEnabled: patch.outputParam !== undefined,
		outputVerbosity: patch.outputParam?.verbosity ?? '',
		outputFormatEnabled: patch.outputParam?.format !== undefined,
		outputFormatKind: patch.outputParam?.format?.kind ?? OutputFormatKind.Text,
		outputJSONSchemaName: patch.outputParam?.format?.jsonSchemaParam?.name ?? '',
		outputJSONSchemaDescription: patch.outputParam?.format?.jsonSchemaParam?.description ?? '',
		outputJSONSchemaRaw: patch.outputParam?.format?.jsonSchemaParam?.schema
			? JSON.stringify(patch.outputParam.format.jsonSchemaParam.schema, null, 2)
			: '',
		outputJSONSchemaStrictMode: booleanToTriState(patch.outputParam?.format?.jsonSchemaParam?.strict),
	};
}

function getInitialFormData(
	initialData: PresetItem | undefined,
	existingPresets: PresetItem[],
	isEditMode: boolean
): AssistantPresetFormData {
	if (initialData) {
		const src = initialData.preset;
		const nextVersion = isEditMode ? getSuggestedNextVersion(initialData, existingPresets) : src.version;

		return {
			displayName: src.displayName,
			slug: src.slug,
			description: src.description ?? '',
			isEnabled: src.isEnabled,
			version: nextVersion,
			startingModelPresetKey: src.startingModelPresetRef ? buildModelPresetRefKey(src.startingModelPresetRef) : '',
			startingIncludeModelSystemPrompt: booleanToTriState(src.startingIncludeModelSystemPrompt),
			modelPatch: getInitialModelPatchFormData(src.startingModelPresetPatch),
			startingInstructionTemplateRefs: (src.startingInstructionTemplateRefs ?? []).map(clonePromptTemplateRef),
			startingToolSelections: (src.startingToolSelections ?? []).map(selection => ({
				toolRef: {
					bundleID: selection.toolRef.bundleID,
					toolSlug: selection.toolRef.toolSlug,
					toolVersion: selection.toolRef.toolVersion,
				},
				autoExecuteMode: booleanToTriState(selection.toolChoicePatch?.autoExecute),
				userArgSchemaInstance: selection.toolChoicePatch?.userArgSchemaInstance ?? '',
			})),
			startingEnabledSkillRefs: (src.startingEnabledSkillRefs ?? []).map(cloneSkillRef),
		};
	}

	return {
		displayName: '',
		slug: '',
		description: '',
		isEnabled: true,
		version: DEFAULT_SEMVER,
		startingModelPresetKey: '',
		startingIncludeModelSystemPrompt: '',
		modelPatch: getDefaultModelPatchFormData(),
		startingInstructionTemplateRefs: [],
		startingToolSelections: [],
		startingEnabledSkillRefs: [],
	};
}

function hasModelPatchFormValues(modelPatch: ModelPatchFormData): boolean {
	return (
		modelPatch.stream !== '' ||
		modelPatch.maxPromptLength.trim().length > 0 ||
		modelPatch.maxOutputLength.trim().length > 0 ||
		modelPatch.temperature.trim().length > 0 ||
		modelPatch.timeout.trim().length > 0 ||
		modelPatch.stopSequencesText.trim().length > 0 ||
		modelPatch.additionalParametersRawJSON.trim().length > 0 ||
		modelPatch.reasoningEnabled ||
		modelPatch.outputEnabled
	);
}

function buildModelPatchFromFormData(
	modelPatch: ModelPatchFormData
): AssistantPresetStartingModelPresetPatch | undefined {
	if (!modelPatch.enabled) {
		return undefined;
	}

	const patch: AssistantPresetStartingModelPresetPatch = {};

	const stream = triStateToBoolean(modelPatch.stream);
	if (stream !== undefined) {
		patch.stream = stream;
	}

	const maxPromptLength = parsePositiveInteger(modelPatch.maxPromptLength);
	if (maxPromptLength !== undefined) {
		patch.maxPromptLength = maxPromptLength;
	}

	const maxOutputLength = parsePositiveInteger(modelPatch.maxOutputLength);
	if (maxOutputLength !== undefined) {
		patch.maxOutputLength = maxOutputLength;
	}

	const temperature = parseOptionalNumber(modelPatch.temperature);
	if (temperature !== undefined) {
		patch.temperature = temperature;
	}

	const timeout = parsePositiveInteger(modelPatch.timeout);
	if (timeout !== undefined) {
		patch.timeout = timeout;
	}

	const stopSequences = modelPatch.stopSequencesText
		.split('\n')
		.map(item => item.trim())
		.filter(Boolean);
	if (stopSequences.length > 0) {
		patch.stopSequences = stopSequences;
	}

	if (modelPatch.additionalParametersRawJSON.trim()) {
		patch.additionalParametersRawJSON = modelPatch.additionalParametersRawJSON.trim();
	}

	const reasoningTokens = parsePositiveInteger(modelPatch.reasoningTokens);
	if (modelPatch.reasoningEnabled && reasoningTokens !== undefined) {
		patch.reasoning = {
			type: modelPatch.reasoningType,
			level: modelPatch.reasoningLevel,
			tokens: reasoningTokens,
			...(modelPatch.reasoningSummaryStyle ? { summaryStyle: modelPatch.reasoningSummaryStyle } : {}),
		};
	}

	if (modelPatch.outputEnabled) {
		const outputParam: OutputParam = {};

		if (modelPatch.outputVerbosity) {
			outputParam.verbosity = modelPatch.outputVerbosity;
		}

		if (modelPatch.outputFormatEnabled) {
			const format: OutputFormat = {
				kind: modelPatch.outputFormatKind,
			};

			if (modelPatch.outputFormatKind === OutputFormatKind.JSONSchema) {
				const jsonSchemaParam: JSONSchemaParam = {
					name: modelPatch.outputJSONSchemaName.trim(),
				};

				const description = modelPatch.outputJSONSchemaDescription.trim();
				if (description) {
					jsonSchemaParam.description = description;
				}

				if (modelPatch.outputJSONSchemaRaw.trim()) {
					const parsed = tryParseJSONRaw(modelPatch.outputJSONSchemaRaw.trim());
					if (parsed.ok && isJSONObjectLike(parsed.value)) {
						jsonSchemaParam.schema = parsed.value;
					}
				}

				const strict = triStateToBoolean(modelPatch.outputJSONSchemaStrictMode);
				if (strict !== undefined) {
					jsonSchemaParam.strict = strict;
				}

				format.jsonSchemaParam = jsonSchemaParam;
			}

			outputParam.format = format;
		}

		if (outputParam.format !== undefined || outputParam.verbosity !== undefined) {
			patch.outputParam = outputParam;
		}
	}

	return hasAssistantPresetModelPatch(patch) ? patch : undefined;
}

function createOptionMap<T extends { key: string }>(items: readonly T[]): Map<string, T> {
	return new Map(items.map(item => [item.key, item] as const));
}

function getEffectiveOptionKey(availableOptions: readonly SimpleSelectableOption[], requestedKey: string): string {
	if (availableOptions.length === 0) {
		return '';
	}

	return availableOptions.some(option => option.key === requestedKey) ? requestedKey : availableOptions[0].key;
}

function getAssistantPresetToolModelCompatibilityError(
	toolOption: AssistantToolOption,
	modelOption?: AssistantModelPresetOption
): string | undefined {
	if (!modelOption) {
		return undefined;
	}

	if (toolOption.toolDefinition.type !== ToolImplType.SDK) {
		return undefined;
	}

	const requiredSDKType = toolOption.toolDefinition.sdkImpl?.sdkType?.trim();
	const toolLabel =
		toolOption.toolDefinition.displayName || toolOption.toolDefinition.slug || toolOption.toolDefinition.id;

	if (!requiredSDKType) {
		return `Tool "${toolLabel}" is missing SDK metadata and cannot be applied safely.`;
	}

	if (requiredSDKType !== (modelOption.providerPreset.sdkType as string)) {
		return `Tool "${toolLabel}" requires provider SDK "${requiredSDKType}", but the selected starting model uses "${modelOption.providerPreset.sdkType}".`;
	}

	return undefined;
}

function AddEditAssistantPresetModalContent({
	onClose,
	onSubmit,
	initialData,
	existingPresets,
	mode,
}: AddEditAssistantPresetModalProps) {
	const effectiveMode: ModalMode = mode ?? (initialData ? 'edit' : 'add');
	const isViewMode = effectiveMode === 'view';
	const isEditMode = effectiveMode === 'edit';

	const [formData, setFormData] = useState<AssistantPresetFormData>(() =>
		getInitialFormData(initialData, existingPresets, isEditMode)
	);
	const [submitError, setSubmitError] = useState('');

	const [catalog, setCatalog] = useState<AssistantPresetEditorCatalog | null>(null);
	const [catalogLoading, setCatalogLoading] = useState(true);
	const [catalogError, setCatalogError] = useState('');

	const [nextInstructionKey, setNextInstructionKey] = useState('');
	const [nextToolKey, setNextToolKey] = useState('');
	const [nextSkillKey, setNextSkillKey] = useState('');

	const initialPresetID = initialData?.preset?.id;
	const initialPresetSlug = initialData?.preset?.slug;
	const initialPresetVersion = initialData?.preset?.version;

	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const isUnmountingRef = useRef(false);

	const loadCatalog = useCallback(async () => {
		setCatalogLoading(true);
		setCatalogError('');

		try {
			const loaded = await loadAssistantPresetEditorCatalog();
			setCatalog(loaded);
		} catch (error) {
			console.error('Failed to load assistant preset editor catalog:', error);
			setCatalogError(getErrorMessage(error, 'Failed to load models, prompts, tools, and skills.'));
		} finally {
			setCatalogLoading(false);
		}
	}, []);

	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) return;

		if (!dialog.open) {
			try {
				dialog.showModal();
			} catch {
				// keep safe
			}
		}

		return () => {
			isUnmountingRef.current = true;

			if (dialog.open) {
				dialog.close();
			}
		};
	}, []);

	useEffect(() => {
		void loadCatalog();
	}, [loadCatalog]);

	const requestClose = useCallback(() => {
		const dialog = dialogRef.current;

		if (dialog?.open) {
			dialog.close();
			return;
		}

		onClose();
	}, [onClose]);

	const handleDialogClose = useCallback(() => {
		if (isUnmountingRef.current) return;
		onClose();
	}, [onClose]);

	const modelPresetOptions = catalog?.modelPresetOptions ?? EMPTY_MODEL_OPTIONS;
	const instructionOptions = catalog?.instructionTemplateOptions ?? EMPTY_INSTRUCTION_OPTIONS;
	const toolOptions = catalog?.toolOptions ?? EMPTY_TOOL_OPTIONS;
	const skillOptions = catalog?.skillOptions ?? EMPTY_SKILL_OPTIONS;

	const modelOptionByKey = useMemo(() => createOptionMap(modelPresetOptions), [modelPresetOptions]);
	const instructionOptionByKey = useMemo(() => createOptionMap(instructionOptions), [instructionOptions]);
	const toolOptionByKey = useMemo(() => createOptionMap(toolOptions), [toolOptions]);
	const skillOptionByKey = useMemo(() => createOptionMap(skillOptions), [skillOptions]);
	const selectedStartingModelOption = formData.startingModelPresetKey
		? modelOptionByKey.get(formData.startingModelPresetKey)
		: undefined;

	const availableInstructionOptions = useMemo<SimpleSelectableOption[]>(() => {
		const selected = new Set(formData.startingInstructionTemplateRefs.map(ref => buildPromptTemplateRefKey(ref)));

		return instructionOptions
			.filter(option => option.isSelectable && !selected.has(option.key))
			.map(option => ({
				key: option.key,
				label: option.label,
			}));
	}, [instructionOptions, formData.startingInstructionTemplateRefs]);

	const availableToolOptions = useMemo<SimpleSelectableOption[]>(() => {
		const selected = new Set(formData.startingToolSelections.map(selection => buildToolRefKey(selection.toolRef)));
		const hasSelectedWebSearchTool = formData.startingToolSelections.some(selection => {
			const option = toolOptionByKey.get(buildToolRefKey(selection.toolRef));
			return option?.toolDefinition.llmToolType === ToolStoreChoiceType.WebSearch;
		});

		return toolOptions
			.filter(option => {
				if (!option.isSelectable || selected.has(option.key)) {
					return false;
				}

				if (hasSelectedWebSearchTool && option.toolDefinition.llmToolType === ToolStoreChoiceType.WebSearch) {
					return false;
				}

				return getAssistantPresetToolModelCompatibilityError(option, selectedStartingModelOption) === undefined;
			})
			.map(option => ({
				key: option.key,
				label: option.label,
			}));
	}, [formData.startingToolSelections, toolOptions, toolOptionByKey, selectedStartingModelOption]);

	const availableSkillOptions = useMemo<SimpleSelectableOption[]>(() => {
		const selected = new Set(formData.startingEnabledSkillRefs.map(ref => buildSkillRefKey(ref)));

		return skillOptions
			.filter(option => option.isSelectable && !selected.has(option.key))
			.map(option => ({
				key: option.key,
				label: option.label,
			}));
	}, [skillOptions, formData.startingEnabledSkillRefs]);

	const effectiveNextInstructionKey = useMemo(
		() => getEffectiveOptionKey(availableInstructionOptions, nextInstructionKey),
		[availableInstructionOptions, nextInstructionKey]
	);

	const effectiveNextToolKey = useMemo(
		() => getEffectiveOptionKey(availableToolOptions, nextToolKey),
		[availableToolOptions, nextToolKey]
	);

	const effectiveNextSkillKey = useMemo(
		() => getEffectiveOptionKey(availableSkillOptions, nextSkillKey),
		[availableSkillOptions, nextSkillKey]
	);

	const validateForm = useCallback(
		(state: AssistantPresetFormData): ErrorState => {
			const nextErrors: ErrorState = {};

			const displayName = state.displayName.trim();
			const slug = state.slug.trim();
			const version = state.version.trim();

			if (!displayName) {
				nextErrors.displayName = 'This field is required.';
			}

			if (!isEditMode) {
				if (!slug) {
					nextErrors.slug = 'This field is required.';
				} else {
					const slugError = validateSlug(slug);
					if (slugError) {
						nextErrors.slug = slugError;
					} else {
						const slugClash = existingPresets.some(
							item => item.preset.slug === slug && item.preset.id !== initialPresetID
						);

						if (slugClash) {
							nextErrors.slug = 'Slug already in use. Use New Version from an existing preset instead.';
						}
					}
				}
			}

			if (!version) {
				nextErrors.version = 'Version is required.';
			} else if (isEditMode && initialPresetVersion && version === initialPresetVersion) {
				nextErrors.version = 'New version must be different from the current version.';
			} else {
				const slugToCheck = isEditMode ? (initialPresetSlug ?? slug) : slug;
				const versionClash =
					Boolean(slugToCheck) &&
					existingPresets.some(item => item.preset.slug === slugToCheck && item.preset.version === version);

				if (versionClash) {
					nextErrors.version = 'That version already exists for this slug.';
				}
			}

			if (state.startingModelPresetKey) {
				const selectedModelOption = modelOptionByKey.get(state.startingModelPresetKey);

				if (!selectedModelOption) {
					nextErrors.modelPreset = 'Selected model preset no longer exists.';
				} else if (!selectedModelOption.isSelectable) {
					nextErrors.modelPreset = selectedModelOption.availabilityReason ?? 'Selected model preset is not available.';
				}
			}

			if (state.modelPatch.enabled) {
				if (!state.startingModelPresetKey) {
					nextErrors.modelPatch = 'Select a starting model preset before defining a starting model patch.';
				}

				if (
					state.modelPatch.maxPromptLength.trim() &&
					parsePositiveInteger(state.modelPatch.maxPromptLength) === undefined
				) {
					nextErrors.modelPatch = 'Max prompt length must be a positive integer.';
				}

				if (
					!nextErrors.modelPatch &&
					state.modelPatch.maxOutputLength.trim() &&
					parsePositiveInteger(state.modelPatch.maxOutputLength) === undefined
				) {
					nextErrors.modelPatch = 'Max output length must be a positive integer.';
				}

				if (
					!nextErrors.modelPatch &&
					state.modelPatch.temperature.trim() &&
					parseOptionalNumber(state.modelPatch.temperature) === undefined
				) {
					nextErrors.modelPatch = 'Temperature must be a valid number.';
				}

				if (
					!nextErrors.modelPatch &&
					state.modelPatch.timeout.trim() &&
					parsePositiveInteger(state.modelPatch.timeout) === undefined
				) {
					nextErrors.modelPatch = 'Timeout must be a positive integer.';
				}

				if (!nextErrors.modelPatch && state.modelPatch.additionalParametersRawJSON.trim()) {
					const parsed = tryParseJSONRaw(state.modelPatch.additionalParametersRawJSON.trim());
					if (!parsed.ok) {
						nextErrors.modelPatch = 'Additional parameters raw JSON must be valid JSON.';
					}
				}

				if (!nextErrors.modelPatch && state.modelPatch.reasoningEnabled) {
					if (!state.modelPatch.reasoningTokens.trim()) {
						nextErrors.modelPatch = 'Reasoning tokens are required when reasoning override is enabled.';
					} else if (parsePositiveInteger(state.modelPatch.reasoningTokens) === undefined) {
						nextErrors.modelPatch = 'Reasoning tokens must be a positive integer.';
					}
				}

				if (
					!nextErrors.modelPatch &&
					state.modelPatch.outputEnabled &&
					state.modelPatch.outputFormatEnabled &&
					state.modelPatch.outputFormatKind === OutputFormatKind.JSONSchema
				) {
					if (!state.modelPatch.outputJSONSchemaName.trim()) {
						nextErrors.modelPatch = 'JSON schema output format requires a schema name.';
					} else if (state.modelPatch.outputJSONSchemaRaw.trim()) {
						const parsed = tryParseJSONRaw(state.modelPatch.outputJSONSchemaRaw.trim());
						if (!parsed.ok) {
							nextErrors.modelPatch = 'JSON schema body must be valid JSON.';
						} else if (!isJSONObjectLike(parsed.value)) {
							nextErrors.modelPatch = 'JSON schema body must be a JSON object.';
						}
					}
				}
			}

			if (state.startingInstructionTemplateRefs.length > 0) {
				const keys = state.startingInstructionTemplateRefs.map(ref => buildPromptTemplateRefKey(ref));
				if (new Set(keys).size !== keys.length) {
					nextErrors.startingInstructionTemplateRefs = 'Instruction template selections must be unique.';
				} else {
					const invalid = state.startingInstructionTemplateRefs.find(ref => {
						const option = instructionOptionByKey.get(buildPromptTemplateRefKey(ref));
						return !option || !option.isSelectable;
					});

					if (invalid) {
						nextErrors.startingInstructionTemplateRefs =
							'Every selected instruction template must still exist, be enabled, be instructions-only, and already be resolved.';
					}
				}
			}

			if (state.startingToolSelections.length > 0) {
				const keys = state.startingToolSelections.map(selection => buildToolRefKey(selection.toolRef));
				if (new Set(keys).size !== keys.length) {
					nextErrors.startingToolSelections = 'Tool selections must be unique.';
				} else {
					const selectedModelOption = state.startingModelPresetKey
						? modelOptionByKey.get(state.startingModelPresetKey)
						: undefined;
					const invalidTool = state.startingToolSelections.find(selection => {
						const option = toolOptionByKey.get(buildToolRefKey(selection.toolRef));
						return !option || !option.isSelectable;
					});

					if (invalidTool) {
						nextErrors.startingToolSelections = 'Every selected tool must still exist and be enabled.';
					} else {
						const webSearchToolCount = state.startingToolSelections.reduce((count, selection) => {
							const option = toolOptionByKey.get(buildToolRefKey(selection.toolRef));
							return count + (option?.toolDefinition.llmToolType === ToolStoreChoiceType.WebSearch ? 1 : 0);
						}, 0);

						if (webSearchToolCount > 1) {
							nextErrors.startingToolSelections =
								'Only one web-search tool may be selected in an assistant preset. The chat runtime currently restores a single active web-search configuration.';
						}
					}

					if (!nextErrors.startingToolSelections && selectedModelOption) {
						const incompatibleToolSelection = state.startingToolSelections.find(selection => {
							const option = toolOptionByKey.get(buildToolRefKey(selection.toolRef));
							return option
								? getAssistantPresetToolModelCompatibilityError(option, selectedModelOption) !== undefined
								: false;
						});

						if (incompatibleToolSelection) {
							const option = toolOptionByKey.get(buildToolRefKey(incompatibleToolSelection.toolRef));
							nextErrors.startingToolSelections =
								(option && getAssistantPresetToolModelCompatibilityError(option, selectedModelOption)) ??
								'Selected tool is not compatible with the chosen starting model.';
						}
					}

					if (!nextErrors.startingToolSelections) {
						const invalidArgsSelection = state.startingToolSelections.find(selection => {
							const option = toolOptionByKey.get(buildToolRefKey(selection.toolRef));
							const raw = selection.userArgSchemaInstance.trim();
							if (!raw) return false;
							if (!option?.hasUserArgSchema) return true;

							const parsed = tryParseJSONRaw(raw);
							return !parsed.ok;
						});

						if (invalidArgsSelection) {
							const option = toolOptionByKey.get(buildToolRefKey(invalidArgsSelection.toolRef));
							nextErrors.startingToolSelections = option?.hasUserArgSchema
								? 'Tool user-args instances must be valid JSON.'
								: 'Tool args may only be provided for tools that expose a user-args schema.';
						}
					}
				}
			}

			if (state.startingEnabledSkillRefs.length > 0) {
				const keys = state.startingEnabledSkillRefs.map(ref => buildSkillRefKey(ref));
				if (new Set(keys).size !== keys.length) {
					nextErrors.startingEnabledSkillRefs = 'Skill selections must be unique.';
				} else {
					const invalid = state.startingEnabledSkillRefs.find(ref => {
						const option = skillOptionByKey.get(buildSkillRefKey(ref));
						return !option || !option.isSelectable;
					});

					if (invalid) {
						nextErrors.startingEnabledSkillRefs = 'Every selected skill must still exist and be enabled.';
					}
				}
			}

			return nextErrors;
		},
		[
			existingPresets,
			initialPresetID,
			initialPresetSlug,
			initialPresetVersion,
			instructionOptionByKey,
			isEditMode,
			modelOptionByKey,
			skillOptionByKey,
			toolOptionByKey,
		]
	);

	const errors = useMemo(
		() => (isViewMode ? EMPTY_ERROR_STATE : validateForm(formData)),
		[formData, isViewMode, validateForm]
	);

	const updateFormData = useCallback((updater: (prev: AssistantPresetFormData) => AssistantPresetFormData) => {
		setFormData(prev => updater(prev));
	}, []);

	const suggestedNextVersion = useMemo(() => {
		if (!initialData) return DEFAULT_SEMVER;
		return getSuggestedNextVersion(initialData, existingPresets);
	}, [initialData, existingPresets]);

	const currentModelOption = selectedStartingModelOption;

	const hasMissingSelectedModel = Boolean(formData.startingModelPresetKey) && currentModelOption === undefined;

	const seedModelPatchFromSelectedModel = useCallback(() => {
		if (!currentModelOption) {
			return;
		}

		updateFormData(prev => ({
			...prev,
			modelPatch: buildModelPatchSeedFormData(currentModelOption),
		}));
	}, [currentModelOption, updateFormData]);

	const updateModelPatch = useCallback(
		(patch: Partial<ModelPatchFormData>) => {
			updateFormData(prev => ({
				...prev,
				modelPatch:
					patch.enabled === true &&
					!prev.modelPatch.enabled &&
					!hasModelPatchFormValues(prev.modelPatch) &&
					currentModelOption
						? buildModelPatchSeedFormData(currentModelOption)
						: {
								...prev.modelPatch,
								...patch,
							},
			}));
		},
		[currentModelOption, updateFormData]
	);

	const modelPresetDropdownItems = useMemo<Record<string, { isEnabled: boolean }>>(() => {
		const items: Record<string, { isEnabled: boolean }> = {
			'': { isEnabled: true },
		};

		if (hasMissingSelectedModel && formData.startingModelPresetKey) {
			items[formData.startingModelPresetKey] = { isEnabled: false };
		}

		for (const option of modelPresetOptions) {
			items[option.key] = { isEnabled: option.isSelectable };
		}

		return items;
	}, [formData.startingModelPresetKey, hasMissingSelectedModel, modelPresetOptions]);

	const modelPresetOrderedKeys = useMemo(() => {
		const keys = [''];

		if (hasMissingSelectedModel && formData.startingModelPresetKey) {
			keys.push(formData.startingModelPresetKey);
		}

		return [...keys, ...modelPresetOptions.map(option => option.key)];
	}, [formData.startingModelPresetKey, hasMissingSelectedModel, modelPresetOptions]);

	const instructionDisplayItems = useMemo<OrderedDisplayItem[]>(
		() =>
			formData.startingInstructionTemplateRefs.map(ref => {
				const key = buildPromptTemplateRefKey(ref);
				const option = instructionOptionByKey.get(key);

				return {
					key,
					title: option?.label ?? `${ref.templateSlug}@${ref.templateVersion} — ${ref.bundleID}`,
					subtitle: option
						? `${option.bundleDisplayName} · ${option.template.slug}@${option.template.version}`
						: 'Reference no longer exists in catalog.',
					statusLabel: option
						? option.isSelectable
							? undefined
							: (option.availabilityReason ?? 'Unavailable')
						: 'Missing reference',
				};
			}),
		[formData.startingInstructionTemplateRefs, instructionOptionByKey]
	);

	const toolDisplayItems = useMemo<ToolSelectionDisplayItem[]>(
		() =>
			formData.startingToolSelections.map(selection => {
				const key = buildToolRefKey(selection.toolRef);
				const option = toolOptionByKey.get(key);
				const rawUserArgs = selection.userArgSchemaInstance.trim();
				const hasStaleArgsWithoutSchema = rawUserArgs.length > 0 && option !== undefined && !option.hasUserArgSchema;
				const canEditUserArgs = option?.hasUserArgSchema || rawUserArgs.length > 0;

				return {
					key,
					title:
						option?.label ??
						`${selection.toolRef.toolSlug}@${selection.toolRef.toolVersion} — ${selection.toolRef.bundleID}`,
					subtitle: option
						? `${option.bundleDisplayName} · ${option.toolDefinition.type}`
						: 'Reference no longer exists in catalog.',
					statusLabel: option
						? hasStaleArgsWithoutSchema
							? 'Args schema missing'
							: option.isSelectable
								? undefined
								: (option.availabilityReason ?? 'Unavailable')
						: 'Missing reference',

					autoExecuteMode: selection.autoExecuteMode,
					autoExecuteLabel: getToolAutoExecuteLabel(selection.autoExecuteMode),
					userArgSchemaInstance: selection.userArgSchemaInstance,
					userArgsHint: option?.hasUserArgSchema
						? 'This tool exposes a user-args schema.'
						: hasStaleArgsWithoutSchema
							? 'This preset still contains saved args, but the tool no longer exposes a user-args schema.'
							: 'This tool does not expose a user-args schema.',
					userArgsEditable: canEditUserArgs,
				};
			}),
		[formData.startingToolSelections, toolOptionByKey]
	);

	const skillDisplayItems = useMemo<OrderedDisplayItem[]>(
		() =>
			formData.startingEnabledSkillRefs.map(ref => {
				const key = buildSkillRefKey(ref);
				const option = skillOptionByKey.get(key);

				return {
					key,
					title: option?.label ?? `${ref.skillSlug} — ${ref.bundleID}`,
					subtitle: option
						? `${option.bundleDisplayName} · ${option.skillDefinition.type}`
						: 'Reference no longer exists in catalog.',
					statusLabel: option
						? option.isSelectable
							? undefined
							: (option.availabilityReason ?? 'Unavailable')
						: 'Missing reference',
				};
			}),
		[formData.startingEnabledSkillRefs, skillOptionByKey]
	);

	const isAllValid =
		isViewMode || (Boolean(catalog) && !catalogLoading && !catalogError && Object.keys(errors).length === 0);

	const handleCatalogRetry = useCallback(() => {
		void loadCatalog();
	}, [loadCatalog]);

	const handleInstructionOptionKeyChange = useCallback((key: string) => {
		setNextInstructionKey(key);
	}, []);

	const handleToolOptionKeyChange = useCallback((key: string) => {
		setNextToolKey(key);
	}, []);

	const handleSkillOptionKeyChange = useCallback((key: string) => {
		setNextSkillKey(key);
	}, []);

	const handleAddInstructionTemplate = useCallback(() => {
		const option = instructionOptions.find(item => item.key === effectiveNextInstructionKey);
		if (!option) return;

		updateFormData(prev => ({
			...prev,
			startingInstructionTemplateRefs: [...prev.startingInstructionTemplateRefs, clonePromptTemplateRef(option.ref)],
		}));
		setNextInstructionKey('');
	}, [effectiveNextInstructionKey, instructionOptions, updateFormData]);

	const handleMoveInstructionUp = useCallback(
		(index: number) => {
			updateFormData(prev => ({
				...prev,
				startingInstructionTemplateRefs: moveItem(prev.startingInstructionTemplateRefs, index, index - 1),
			}));
		},
		[updateFormData]
	);

	const handleMoveInstructionDown = useCallback(
		(index: number) => {
			updateFormData(prev => ({
				...prev,
				startingInstructionTemplateRefs: moveItem(prev.startingInstructionTemplateRefs, index, index + 1),
			}));
		},
		[updateFormData]
	);

	const handleRemoveInstruction = useCallback(
		(index: number) => {
			updateFormData(prev => ({
				...prev,
				startingInstructionTemplateRefs: removeItemAtIndex(prev.startingInstructionTemplateRefs, index),
			}));
		},
		[updateFormData]
	);

	const handleAddToolSelection = useCallback(() => {
		const option = toolOptions.find(item => item.key === effectiveNextToolKey);
		if (!option) return;

		updateFormData(prev => ({
			...prev,
			startingToolSelections: [
				...prev.startingToolSelections,
				{
					toolRef: {
						bundleID: option.toolRef.bundleID,
						toolSlug: option.toolRef.toolSlug,
						toolVersion: option.toolRef.toolVersion,
					},
					autoExecuteMode: '',
					userArgSchemaInstance: '',
				},
			],
		}));
		setNextToolKey('');
	}, [effectiveNextToolKey, toolOptions, updateFormData]);

	const handleMoveToolUp = useCallback(
		(index: number) => {
			updateFormData(prev => ({
				...prev,
				startingToolSelections: moveItem(prev.startingToolSelections, index, index - 1),
			}));
		},
		[updateFormData]
	);

	const handleMoveToolDown = useCallback(
		(index: number) => {
			updateFormData(prev => ({
				...prev,
				startingToolSelections: moveItem(prev.startingToolSelections, index, index + 1),
			}));
		},
		[updateFormData]
	);

	const handleRemoveTool = useCallback(
		(index: number) => {
			updateFormData(prev => ({
				...prev,
				startingToolSelections: removeItemAtIndex(prev.startingToolSelections, index),
			}));
		},
		[updateFormData]
	);

	const handleToolAutoExecuteChange = useCallback(
		(index: number, value: TriStateBoolean) => {
			updateFormData(prev => ({
				...prev,
				startingToolSelections: updateItemAtIndex(prev.startingToolSelections, index, item => ({
					...item,
					autoExecuteMode: value,
				})),
			}));
		},
		[updateFormData]
	);

	const handleToolUserArgsChange = useCallback(
		(index: number, value: string) => {
			updateFormData(prev => ({
				...prev,
				startingToolSelections: updateItemAtIndex(prev.startingToolSelections, index, item => ({
					...item,
					userArgSchemaInstance: value,
				})),
			}));
		},
		[updateFormData]
	);

	const handleAddSkillRef = useCallback(() => {
		const option = skillOptions.find(item => item.key === effectiveNextSkillKey);
		if (!option) return;

		updateFormData(prev => ({
			...prev,
			startingEnabledSkillRefs: [...prev.startingEnabledSkillRefs, cloneSkillRef(option.ref)],
		}));
		setNextSkillKey('');
	}, [effectiveNextSkillKey, skillOptions, updateFormData]);

	const handleMoveSkillUp = useCallback(
		(index: number) => {
			updateFormData(prev => ({
				...prev,
				startingEnabledSkillRefs: moveItem(prev.startingEnabledSkillRefs, index, index - 1),
			}));
		},
		[updateFormData]
	);

	const handleMoveSkillDown = useCallback(
		(index: number) => {
			updateFormData(prev => ({
				...prev,
				startingEnabledSkillRefs: moveItem(prev.startingEnabledSkillRefs, index, index + 1),
			}));
		},
		[updateFormData]
	);

	const handleRemoveSkill = useCallback(
		(index: number) => {
			updateFormData(prev => ({
				...prev,
				startingEnabledSkillRefs: removeItemAtIndex(prev.startingEnabledSkillRefs, index),
			}));
		},
		[updateFormData]
	);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async e => {
		e.preventDefault();
		e.stopPropagation();

		if (isViewMode) return;

		setSubmitError('');

		if (catalogLoading || !catalog) {
			setSubmitError('The editor catalog could not be loaded yet. Please retry.');
			return;
		}

		const nextErrors = validateForm(formData);
		if (Object.keys(nextErrors).length > 0) {
			return;
		}

		try {
			let startingModelPresetRef: AssistantPresetUpsertInput['startingModelPresetRef'];

			if (formData.startingModelPresetKey) {
				const selectedModelOption = modelOptionByKey.get(formData.startingModelPresetKey);

				if (!selectedModelOption || !selectedModelOption.isSelectable) {
					setSubmitError(selectedModelOption?.availabilityReason ?? 'Selected model preset is not available.');
					return;
				}

				startingModelPresetRef = selectedModelOption.ref;
			}

			const startingToolSelections = formData.startingToolSelections.map(selection => {
				const toolOption = toolOptionByKey.get(buildToolRefKey(selection.toolRef));
				const autoExecute = triStateToBoolean(selection.autoExecuteMode);
				const userArgSchemaInstance = toolOption?.hasUserArgSchema ? selection.userArgSchemaInstance.trim() : '';

				const toolChoicePatch =
					autoExecute === undefined && !userArgSchemaInstance
						? undefined
						: {
								...(autoExecute !== undefined ? { autoExecute } : {}),
								...(userArgSchemaInstance ? { userArgSchemaInstance } : {}),
							};

				return {
					toolRef: {
						bundleID: selection.toolRef.bundleID,
						toolSlug: selection.toolRef.toolSlug,
						toolVersion: selection.toolRef.toolVersion,
					},
					toolChoicePatch,
				};
			});

			const payload: AssistantPresetUpsertInput = {
				displayName: formData.displayName.trim(),
				slug: (initialData?.preset.slug ?? formData.slug).trim(),
				description: formData.description.trim(),
				isEnabled: formData.isEnabled,
				version: formData.version.trim(),
				startingModelPresetRef,
				startingModelPresetPatch: buildModelPatchFromFormData(formData.modelPatch),
				startingIncludeModelSystemPrompt: startingModelPresetRef
					? triStateToBoolean(formData.startingIncludeModelSystemPrompt)
					: undefined,
				startingInstructionTemplateRefs: formData.startingInstructionTemplateRefs.map(clonePromptTemplateRef),
				startingToolSelections,
				startingEnabledSkillRefs: formData.startingEnabledSkillRefs.map(cloneSkillRef),
			};

			await onSubmit(payload);
			requestClose();
		} catch (error) {
			setSubmitError(getErrorMessage(error, 'Failed to save assistant preset.'));
		}
	};

	const headerTitle =
		effectiveMode === 'view'
			? 'View Assistant Preset'
			: effectiveMode === 'edit'
				? 'Create New Assistant Preset Version'
				: 'Add Assistant Preset';

	return (
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={e => {
				if (!isViewMode) e.preventDefault();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-5xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between">
						<h3 className="text-lg font-bold">{headerTitle}</h3>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={requestClose}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

					<form noValidate onSubmit={handleSubmit} className="space-y-4">
						{submitError && (
							<div className="alert alert-error rounded-2xl text-sm">
								<div className="flex items-center gap-2">
									<FiAlertCircle size={14} />
									<span>{submitError}</span>
								</div>
							</div>
						)}

						{catalogLoading && (
							<div className="alert rounded-2xl text-sm">
								<span className="loading loading-spinner loading-sm" />
								<span>Loading models, instruction templates, tools, and skills…</span>
							</div>
						)}

						{catalogError && (
							<div className="alert alert-warning rounded-2xl text-sm">
								<div className="flex grow items-center gap-2">
									<FiAlertCircle size={14} />
									<span>{catalogError}</span>
								</div>
								<button type="button" className="btn btn-sm rounded-xl" onClick={handleCatalogRetry}>
									<FiRefreshCw size={14} />
									<span className="ml-1">Retry</span>
								</button>
							</div>
						)}

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Display Name*</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									value={formData.displayName}
									onChange={e => {
										const value = e.target.value;
										updateFormData(prev => ({ ...prev, displayName: value }));
									}}
									readOnly={isViewMode}
									className={`input input-bordered w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									autoFocus={!isViewMode}
									aria-invalid={Boolean(errors.displayName)}
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
								<span className="label-text text-sm">Slug*</span>
								<span className="label-text-alt tooltip tooltip-right" data-tip="Short URL-friendly identifier">
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									value={formData.slug}
									onChange={e => {
										const value = e.target.value;
										updateFormData(prev => ({ ...prev, slug: value }));
									}}
									className={`input input-bordered w-full rounded-xl ${errors.slug ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									readOnly={isViewMode || isEditMode}
									aria-invalid={Boolean(errors.slug)}
								/>
								{errors.slug && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.slug}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Version*</span>
								<span
									className="label-text-alt tooltip tooltip-right"
									data-tip="Versions are immutable. Edit creates a new version."
								>
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<input
									type="text"
									value={formData.version}
									onChange={e => {
										const value = e.target.value;
										updateFormData(prev => ({ ...prev, version: value }));
									}}
									readOnly={isViewMode}
									className={`input input-bordered w-full rounded-xl ${errors.version ? 'input-error' : ''}`}
									spellCheck="false"
									autoComplete="off"
									aria-invalid={Boolean(errors.version)}
									placeholder={DEFAULT_SEMVER}
								/>
								{isEditMode && initialData?.preset && (
									<div className="label">
										<span className="label-text-alt text-base-content/70 text-xs">
											Current: {initialData.preset.version} · Suggested next: {suggestedNextVersion}
											{!isSemverVersion(initialData.preset.version) ? ' (current is not semver)' : ''}
										</span>
									</div>
								)}
								{errors.version && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.version}
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
									checked={formData.isEnabled}
									onChange={e => {
										updateFormData(prev => ({ ...prev, isEnabled: e.target.checked }));
									}}
									className="toggle toggle-accent"
									disabled={isViewMode}
								/>
							</div>
						</div>

						<div className="grid grid-cols-12 items-start gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Description</span>
							</label>
							<div className="col-span-9">
								<textarea
									value={formData.description}
									onChange={e => {
										const value = e.target.value;
										updateFormData(prev => ({ ...prev, description: value }));
									}}
									readOnly={isViewMode}
									className="textarea textarea-bordered h-20 w-full rounded-xl"
									spellCheck="false"
								/>
							</div>
						</div>

						<div className="divider">Starting Model</div>

						<div className="grid grid-cols-12 items-start gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Model Preset</span>
								<span
									className="label-text-alt tooltip tooltip-right"
									data-tip="Optional starting model preset reference. Must resolve and be enabled."
								>
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<div className={errors.modelPreset ? 'ring-error/50 rounded-2xl ring-1' : ''}>
									<Dropdown<string>
										dropdownItems={modelPresetDropdownItems}
										orderedKeys={modelPresetOrderedKeys}
										selectedKey={formData.startingModelPresetKey}
										onChange={value => {
											updateFormData(prev => ({
												...prev,
												startingModelPresetKey: value,
												startingIncludeModelSystemPrompt: value ? prev.startingIncludeModelSystemPrompt : '',
											}));
										}}
										disabled={isViewMode}
										filterDisabled={false}
										placeholderLabel="None"
										title="Select model preset"
										getDisplayName={key => {
											if (!key) return 'None';
											const option = modelOptionByKey.get(key);
											if (option) {
												return option.isSelectable
													? option.label
													: `${option.label} — ${option.availabilityReason ?? 'Unavailable'}`;
											}
											return `Unavailable (${key})`;
										}}
									/>
								</div>

								{currentModelOption && (
									<div className="label">
										<span
											className={`label-text-alt text-xs ${
												currentModelOption.isSelectable ? 'text-base-content/70' : 'text-warning'
											}`}
										>
											{currentModelOption.isSelectable
												? `${currentModelOption.providerPreset.displayName || currentModelOption.providerPreset.name} / ${
														currentModelOption.modelPreset.displayName || currentModelOption.modelPreset.name
													}`
												: currentModelOption.availabilityReason}
										</span>
									</div>
								)}

								{errors.modelPreset && (
									<div className="label">
										<span className="label-text-alt text-error flex items-center gap-1">
											<FiAlertCircle size={12} /> {errors.modelPreset}
										</span>
									</div>
								)}
							</div>
						</div>

						<div className="grid grid-cols-12 items-center gap-2">
							<label className="label col-span-3">
								<span className="label-text text-sm">Include Model System Prompt</span>
								<span
									className="label-text-alt tooltip tooltip-right"
									data-tip="Optional. Not Set leaves the user's current choice unchanged."
								>
									<FiHelpCircle size={12} />
								</span>
							</label>
							<div className="col-span-9">
								<Dropdown<TriStateBoolean>
									dropdownItems={TRI_STATE_DROPDOWN_ITEMS}
									orderedKeys={TRI_STATE_ORDERED_KEYS}
									selectedKey={formData.startingIncludeModelSystemPrompt}
									onChange={value => {
										updateFormData(prev => ({
											...prev,
											startingIncludeModelSystemPrompt: value,
										}));
									}}
									disabled={isViewMode || !formData.startingModelPresetKey}
									filterDisabled={false}
									placeholderLabel="Not Set"
									title="Model system prompt inclusion"
									getDisplayName={getIncludeModelSystemPromptLabel}
								/>
							</div>
						</div>

						<AssistantPresetModelPatchEditor
							isViewMode={isViewMode}
							modelPatch={formData.modelPatch}
							error={errors.modelPatch}
							onPatchChange={updateModelPatch}
							canSeedFromSelectedModel={Boolean(currentModelOption)}
							onSeedFromSelectedModel={seedModelPatchFromSelectedModel}
						/>

						<div className="divider">Instruction Templates</div>

						<p className="text-base-content/70 text-xs">
							Only enabled, already-resolved system prompt templates may be selected.
						</p>

						{errors.startingInstructionTemplateRefs && (
							<div className="text-error flex items-center gap-1 text-sm">
								<FiAlertCircle size={12} /> {errors.startingInstructionTemplateRefs}
							</div>
						)}

						<OrderedRefSelectionSection
							isViewMode={isViewMode}
							availableOptions={availableInstructionOptions}
							selectedOptionKey={effectiveNextInstructionKey}
							onSelectedOptionKeyChange={handleInstructionOptionKeyChange}
							onAdd={handleAddInstructionTemplate}
							emptyOptionsLabel="No eligible instruction templates available"
							items={instructionDisplayItems}
							emptyState="No instruction templates selected."
							onMoveUp={handleMoveInstructionUp}
							onMoveDown={handleMoveInstructionDown}
							onRemove={handleRemoveInstruction}
						/>

						<div className="divider">Tool Selections</div>

						{errors.startingToolSelections && (
							<div className="text-error flex items-center gap-1 text-sm">
								<FiAlertCircle size={12} /> {errors.startingToolSelections}
							</div>
						)}

						<ToolSelectionSection
							isViewMode={isViewMode}
							availableOptions={availableToolOptions}
							selectedOptionKey={effectiveNextToolKey}
							onSelectedOptionKeyChange={handleToolOptionKeyChange}
							onAdd={handleAddToolSelection}
							emptyOptionsLabel="No eligible tools available"
							items={toolDisplayItems}
							emptyState="No tool selections configured."
							onMoveUp={handleMoveToolUp}
							onMoveDown={handleMoveToolDown}
							onRemove={handleRemoveTool}
							onAutoExecuteChange={handleToolAutoExecuteChange}
							onUserArgsChange={handleToolUserArgsChange}
						/>

						<div className="divider">Enabled Skills</div>

						{errors.startingEnabledSkillRefs && (
							<div className="text-error flex items-center gap-1 text-sm">
								<FiAlertCircle size={12} /> {errors.startingEnabledSkillRefs}
							</div>
						)}

						<OrderedRefSelectionSection
							isViewMode={isViewMode}
							availableOptions={availableSkillOptions}
							selectedOptionKey={effectiveNextSkillKey}
							onSelectedOptionKeyChange={handleSkillOptionKeyChange}
							onAdd={handleAddSkillRef}
							emptyOptionsLabel="No eligible skills available"
							items={skillDisplayItems}
							emptyState="No skills selected."
							onMoveUp={handleMoveSkillUp}
							onMoveDown={handleMoveSkillDown}
							onRemove={handleRemoveSkill}
						/>

						{isViewMode && initialData?.preset && (
							<>
								<div className="divider">Metadata</div>
								<div className="grid grid-cols-12 gap-2 text-sm">
									<div className="col-span-3 font-semibold">Version</div>
									<div className="col-span-9">{initialData.preset.version}</div>

									<div className="col-span-3 font-semibold">Built-in</div>
									<div className="col-span-9">{initialData.preset.isBuiltIn ? 'Yes' : 'No'}</div>

									<div className="col-span-3 font-semibold">Model Ref</div>
									<div className="col-span-9">
										{currentModelOption?.label ?? (formData.startingModelPresetKey || '—')}
									</div>

									<div className="col-span-3 font-semibold">Model Patch</div>
									<div className="col-span-9">
										{hasAssistantPresetModelPatch(initialData.preset.startingModelPresetPatch) ? 'Configured' : 'None'}
									</div>

									<div className="col-span-3 font-semibold">Include Model System Prompt</div>
									<div className="col-span-9">
										{formData.startingModelPresetKey
											? getIncludeModelSystemPromptLabel(formData.startingIncludeModelSystemPrompt)
											: '—'}
									</div>

									<div className="col-span-3 font-semibold">Created</div>
									<div className="col-span-9">{formatDateish(initialData.preset.createdAt)}</div>

									<div className="col-span-3 font-semibold">Modified</div>
									<div className="col-span-9">{formatDateish(initialData.preset.modifiedAt)}</div>

									<div className="col-span-3 font-semibold">Patch Stream</div>
									<div className="col-span-9">{getTriStateLabel(formData.modelPatch.stream)}</div>
								</div>
							</>
						)}

						<div className="modal-action">
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={requestClose}>
								{isViewMode ? 'Close' : 'Cancel'}
							</button>
							{!isViewMode && (
								<button type="submit" className="btn btn-primary rounded-xl" disabled={!isAllValid}>
									Save
								</button>
							)}
						</div>
					</form>
				</div>
			</div>
			<ModalBackdrop enabled={isViewMode} />
		</dialog>
	);
}

export function AddEditAssistantPresetModal(props: AddEditAssistantPresetModalProps) {
	if (!props.isOpen) return null;
	if (typeof document === 'undefined' || !document.body) return null;

	const remountKey = props.initialData
		? `${props.mode ?? 'auto'}:${props.initialData.bundleID}:${props.initialData.preset.id}:${props.initialData.preset.version}:${String(props.initialData.preset.modifiedAt)}`
		: `${props.mode ?? 'auto'}:new`;

	return createPortal(<AddEditAssistantPresetModalContent key={remountKey} {...props} />, document.body);
}
