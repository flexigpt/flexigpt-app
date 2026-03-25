import type { AssistantPreset } from '@/spec/assistantpreset';
import type { UIChatOption } from '@/spec/modelpreset';
import type { SkillRef } from '@/spec/skill';
import { type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import { stripUndefinedDeep } from '@/lib/obj_utils';

import { sanitizeUIChatOptionByCapabilities } from '@/chats/inputarea/assitantcontexts/capabilities_override_helper';
import type { WebSearchChoiceTemplate } from '@/chats/inputarea/tools/websearch_utils';

export interface AssistantPresetOptionItem {
	key: string;
	bundleID: string;
	bundleSlug: string;
	bundleDisplayName: string;
	displayName: string;
	description?: string;
	preset: AssistantPreset;
	label: string;

	isSelectable: boolean;
	availabilityReason?: string;
}

export interface AssistantPresetPreparedRuntimeSelections {
	hasToolsSelection: boolean;
	conversationToolChoices: ToolStoreChoice[];
	webSearchChoices: ToolStoreChoice[];
	hasSkillsSelection: boolean;
	enabledSkillRefs: SkillRef[];
}

export interface AssistantPresetComparisonState {
	model?: Record<string, unknown>;
	instructions?: string[];
	tools?: {
		conversationToolChoices: AssistantPresetNormalizedToolChoice[];
		webSearchChoices: AssistantPresetNormalizedToolChoice[];
	};
	skills?: string[];
}

export interface AssistantPresetPreparedApplication {
	presetKey: string;
	option: AssistantPresetOptionItem;
	preset: AssistantPreset;

	hasModelSelection: boolean;
	nextSelectedModel: UIChatOption;

	hasIncludeModelSystemPromptSelection: boolean;
	nextIncludeModelSystemPrompt: boolean;

	hasInstructionTemplateSelection: boolean;
	nextSelectedPromptKeys: string[];

	runtimeSelections: AssistantPresetPreparedRuntimeSelections;
	comparisonState: AssistantPresetComparisonState;
}

export interface AssistantPresetRuntimeSnapshot {
	conversationToolChoices: ToolStoreChoice[];
	webSearchChoices: ToolStoreChoice[];
	enabledSkillRefs: SkillRef[];
}

interface AssistantPresetNormalizedToolChoice {
	toolType: ToolStoreChoiceType;
	bundleID: string;
	toolSlug: string;
	toolVersion: string;
	autoExecute: boolean;
	userArgSchemaInstance?: string;
}

export interface AssistantPresetModificationSummary {
	model: boolean;
	instructions: boolean;
	tools: boolean;
	skills: boolean;
	any: boolean;
	modifiedLabels: string[];
}

export const EMPTY_ASSISTANT_PRESET_RUNTIME_SNAPSHOT: AssistantPresetRuntimeSnapshot = {
	conversationToolChoices: [],
	webSearchChoices: [],
	enabledSkillRefs: [],
};

export const EMPTY_ASSISTANT_PRESET_MODIFICATION_SUMMARY: AssistantPresetModificationSummary = {
	model: false,
	instructions: false,
	tools: false,
	skills: false,
	any: false,
	modifiedLabels: [],
};

function hasOwn(value: object, key: string): boolean {
	return Object.prototype.hasOwnProperty.call(value, key);
}

function cloneJSONLike<T>(value: T): T {
	if (value === undefined) {
		return value;
	}
	return JSON.parse(JSON.stringify(value)) as T;
}

function mergePatchObject<T>(baseValue: T | undefined, patchValue: unknown): T {
	if (patchValue === undefined) {
		return baseValue as T;
	}

	if (patchValue === null || Array.isArray(patchValue) || typeof patchValue !== 'object') {
		return cloneJSONLike(patchValue) as T;
	}

	const baseObject =
		baseValue && typeof baseValue === 'object' && !Array.isArray(baseValue)
			? (baseValue as Record<string, unknown>)
			: {};

	const next: Record<string, unknown> = {
		...baseObject,
	};

	for (const [key, nestedPatchValue] of Object.entries(patchValue as Record<string, unknown>)) {
		next[key] = mergePatchObject(baseObject[key], nestedPatchValue);
	}

	return next as T;
}

function pickManagedPatchShape(patchValue: unknown, currentValue: unknown): unknown {
	if (patchValue === undefined) {
		return undefined;
	}

	if (patchValue === null || Array.isArray(patchValue) || typeof patchValue !== 'object') {
		return cloneJSONLike(currentValue);
	}

	const currentObject =
		currentValue && typeof currentValue === 'object' && !Array.isArray(currentValue)
			? (currentValue as Record<string, unknown>)
			: {};

	const next: Record<string, unknown> = {};

	for (const [key, nestedPatchValue] of Object.entries(patchValue as Record<string, unknown>)) {
		next[key] = pickManagedPatchShape(nestedPatchValue, currentObject[key]);
	}

	return next;
}

function areComparableValuesEqual(left: unknown, right: unknown): boolean {
	return JSON.stringify(stripUndefinedDeep(left)) === JSON.stringify(stripUndefinedDeep(right));
}

export function buildAssistantPresetIdentityKey(
	bundleID: string,
	assistantPresetSlug: string,
	version: string
): string {
	return `${bundleID}/${assistantPresetSlug}@${version}`;
}

export function applyAssistantPresetModelPatch(
	base: UIChatOption,
	patch?: AssistantPreset['startingModelPresetPatch']
): UIChatOption {
	if (!patch) {
		return sanitizeUIChatOptionByCapabilities({
			...base,
		});
	}

	const next: UIChatOption = {
		...base,
	};

	const patchHasTemperature = hasOwn(patch, 'temperature');
	const patchHasReasoning = hasOwn(patch, 'reasoning');

	if (hasOwn(patch, 'stream') && patch.stream !== undefined) {
		next.stream = patch.stream;
	}

	if (hasOwn(patch, 'maxPromptLength') && patch.maxPromptLength !== undefined) {
		next.maxPromptLength = patch.maxPromptLength;
	}

	if (hasOwn(patch, 'maxOutputLength') && patch.maxOutputLength !== undefined) {
		next.maxOutputLength = patch.maxOutputLength;
	}

	if (patchHasTemperature) {
		if (patch.temperature === undefined) {
			delete next.temperature;
		} else {
			next.temperature = patch.temperature;
			if (!patchHasReasoning) {
				delete next.reasoning;
			}
		}
	}

	if (patchHasReasoning) {
		if (patch.reasoning === undefined) {
			delete next.reasoning;
		} else {
			next.reasoning = mergePatchObject(next.reasoning, patch.reasoning);
			if (!patchHasTemperature) {
				delete next.temperature;
			}
		}
	}

	if (hasOwn(patch, 'outputParam')) {
		if (patch.outputParam === undefined) {
			delete next.outputParam;
		} else {
			next.outputParam = mergePatchObject(next.outputParam, patch.outputParam);
		}
	}

	if (hasOwn(patch, 'stopSequences')) {
		next.stopSequences = patch.stopSequences ? [...patch.stopSequences] : undefined;
	}

	if (hasOwn(patch, 'timeout') && patch.timeout !== undefined) {
		next.timeout = patch.timeout;
	}

	if (hasOwn(patch, 'additionalParametersRawJSON')) {
		next.additionalParametersRawJSON = patch.additionalParametersRawJSON?.trim() || undefined;
	}

	return sanitizeUIChatOptionByCapabilities(next);
}

export function buildAssistantPresetModelComparisonState(
	preset: AssistantPreset,
	selectedModel: UIChatOption,
	includeModelSystemPrompt: boolean
): Record<string, unknown> | undefined {
	const modelState: Record<string, unknown> = {};

	if (preset.startingModelPresetRef) {
		modelState.modelRef = {
			providerName: selectedModel.providerName,
			modelPresetID: selectedModel.modelPresetID,
		};
	}

	const patch = preset.startingModelPresetPatch;
	if (patch) {
		if (hasOwn(patch, 'stream')) {
			modelState.stream = selectedModel.stream;
		}

		if (hasOwn(patch, 'maxPromptLength')) {
			modelState.maxPromptLength = selectedModel.maxPromptLength;
		}

		if (hasOwn(patch, 'maxOutputLength')) {
			modelState.maxOutputLength = selectedModel.maxOutputLength;
		}

		if (hasOwn(patch, 'temperature')) {
			modelState.temperature = selectedModel.temperature;
		}

		if (hasOwn(patch, 'reasoning')) {
			modelState.reasoning = pickManagedPatchShape(patch.reasoning, selectedModel.reasoning);
		}

		if (hasOwn(patch, 'outputParam')) {
			modelState.outputParam = pickManagedPatchShape(patch.outputParam, selectedModel.outputParam);
		}

		if (hasOwn(patch, 'stopSequences')) {
			modelState.stopSequences = selectedModel.stopSequences;
		}

		if (hasOwn(patch, 'timeout')) {
			modelState.timeout = selectedModel.timeout;
		}

		if (hasOwn(patch, 'additionalParametersRawJSON')) {
			modelState.additionalParametersRawJSON = selectedModel.additionalParametersRawJSON;
		}
	}

	if (preset.startingIncludeModelSystemPrompt !== undefined) {
		modelState.includeModelSystemPrompt = includeModelSystemPrompt;
	}

	return Object.keys(modelState).length > 0 ? modelState : undefined;
}

export function normalizeAssistantPresetToolChoices(
	choices: Array<
		Pick<
			ToolStoreChoice,
			'bundleID' | 'toolSlug' | 'toolVersion' | 'toolType' | 'autoExecute' | 'userArgSchemaInstance'
		>
	>
): AssistantPresetNormalizedToolChoice[] {
	return choices.map(choice => ({
		toolType: choice.toolType,
		bundleID: choice.bundleID,
		toolSlug: choice.toolSlug,
		toolVersion: choice.toolVersion,
		autoExecute: choice.autoExecute,
		userArgSchemaInstance: choice.userArgSchemaInstance?.trim() || undefined,
	}));
}

export function normalizeAssistantPresetSkillRefs(refs: SkillRef[]): string[] {
	return refs.map(ref => `${ref.bundleID}/${ref.skillSlug}#${ref.skillID}`);
}

export function mapAssistantPresetWebSearchTemplatesToChoices(templates: WebSearchChoiceTemplate[]): ToolStoreChoice[] {
	return templates.map((template, index) => ({
		choiceID: `assistant-preset-web-search:${index}:${template.bundleID}:${template.toolSlug}:${template.toolVersion}`,
		bundleID: template.bundleID,
		bundleSlug: template.bundleSlug,
		toolID: template.toolID,
		toolSlug: template.toolSlug,
		toolVersion: template.toolVersion,
		toolType: template.toolType ?? ToolStoreChoiceType.WebSearch,
		displayName: template.displayName,
		description: template.description,
		autoExecute: template.autoExecute,
		userArgSchemaInstance: template.userArgSchemaInstance,
	}));
}

export function getAssistantPresetModificationSummary(args: {
	preparedApplication: AssistantPresetPreparedApplication | null;
	currentSelectedModel: UIChatOption;
	currentIncludeModelSystemPrompt: boolean;
	currentSelectedPromptKeys: string[];
	currentRuntimeSnapshot: AssistantPresetRuntimeSnapshot;
}): AssistantPresetModificationSummary {
	const { preparedApplication } = args;
	if (!preparedApplication) {
		return EMPTY_ASSISTANT_PRESET_MODIFICATION_SUMMARY;
	}

	const currentModelState = buildAssistantPresetModelComparisonState(
		preparedApplication.preset,
		args.currentSelectedModel,
		args.currentIncludeModelSystemPrompt
	);

	const currentToolsState = {
		conversationToolChoices: normalizeAssistantPresetToolChoices(args.currentRuntimeSnapshot.conversationToolChoices),
		webSearchChoices: normalizeAssistantPresetToolChoices(args.currentRuntimeSnapshot.webSearchChoices),
	};

	const currentSkillsState = normalizeAssistantPresetSkillRefs(args.currentRuntimeSnapshot.enabledSkillRefs);

	const model = preparedApplication.comparisonState.model
		? !areComparableValuesEqual(preparedApplication.comparisonState.model, currentModelState)
		: false;

	const instructions = preparedApplication.comparisonState.instructions
		? !areComparableValuesEqual(preparedApplication.comparisonState.instructions, [...args.currentSelectedPromptKeys])
		: false;

	const tools = preparedApplication.comparisonState.tools
		? !areComparableValuesEqual(preparedApplication.comparisonState.tools, currentToolsState)
		: false;

	const skills = preparedApplication.comparisonState.skills
		? !areComparableValuesEqual(preparedApplication.comparisonState.skills, currentSkillsState)
		: false;

	const modifiedLabels: string[] = [];
	if (model) modifiedLabels.push('Model');
	if (instructions) modifiedLabels.push('Instructions');
	if (tools) modifiedLabels.push('Tools');
	if (skills) modifiedLabels.push('Skills');

	return {
		model,
		instructions,
		tools,
		skills,
		any: modifiedLabels.length > 0,
		modifiedLabels,
	};
}
