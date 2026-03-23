import type {
	AssistantPreset,
	AssistantPresetStartingModelPresetPatch,
	PutAssistantPresetPayload,
} from '@/spec/assistantpreset';
import type { ModelPresetRef } from '@/spec/modelpreset';
import type { PromptTemplateRef } from '@/spec/prompt';
import type { SkillRef } from '@/spec/skill';
import type { ToolRef, ToolSelection } from '@/spec/tool';

export interface AssistantPresetUpsertInput extends PutAssistantPresetPayload {
	slug: string;
	version: string;
}

type AssistantPresetModelPatchLike =
	| AssistantPreset['startingModelPresetPatch']
	| AssistantPresetStartingModelPresetPatch
	| undefined;

function cloneJSONLike<T>(value: T): T {
	if (value === undefined) {
		return value;
	}
	return JSON.parse(JSON.stringify(value)) as T;
}

export function buildModelPresetRefKey(ref: ModelPresetRef): string {
	return `${ref.providerName}/${ref.modelPresetID}`;
}

export function buildToolRefKey(ref: ToolRef): string {
	return `${ref.bundleID}/${ref.toolSlug}@${ref.toolVersion}`;
}

export function buildSkillRefKey(ref: SkillRef): string {
	return `${ref.bundleID}/${ref.skillSlug}#${ref.skillID}`;
}

export function clonePromptTemplateRef(ref: PromptTemplateRef): PromptTemplateRef {
	return {
		bundleID: ref.bundleID,
		templateSlug: ref.templateSlug,
		templateVersion: ref.templateVersion,
	};
}

function cloneToolSelection(selection: ToolSelection): ToolSelection {
	return {
		toolRef: {
			bundleID: selection.toolRef.bundleID,
			toolSlug: selection.toolRef.toolSlug,
			toolVersion: selection.toolRef.toolVersion,
		},
		toolChoicePatch: selection.toolChoicePatch
			? {
					...selection.toolChoicePatch,
				}
			: undefined,
	};
}

export function cloneSkillRef(ref: SkillRef): SkillRef {
	return {
		bundleID: ref.bundleID,
		skillSlug: ref.skillSlug,
		skillID: ref.skillID,
	};
}

export function formatAssistantPresetModelRef(ref?: ModelPresetRef): string {
	if (!ref) {
		return '—';
	}
	return `${ref.providerName}/${ref.modelPresetID}`;
}

export function getAssistantPresetCounts(
	preset: Pick<
		AssistantPreset,
		'startingInstructionTemplateRefs' | 'startingToolSelections' | 'startingEnabledSkillRefs'
	>
) {
	return {
		instructions: preset.startingInstructionTemplateRefs?.length ?? 0,
		tools: preset.startingToolSelections?.length ?? 0,
		skills: preset.startingEnabledSkillRefs?.length ?? 0,
	};
}

export function formatDateish(value: string | Date | undefined | null): string {
	if (!value) {
		return '-';
	}

	if (value instanceof Date) {
		return value.toISOString();
	}

	return value;
}

export function hasAssistantPresetModelPatch(patch: AssistantPresetModelPatchLike): boolean {
	if (!patch) {
		return false;
	}

	return (
		patch.stream !== undefined ||
		patch.maxPromptLength !== undefined ||
		patch.maxOutputLength !== undefined ||
		patch.temperature !== undefined ||
		patch.outputParam !== undefined ||
		(patch.stopSequences?.length ?? 0) > 0 ||
		patch.reasoning !== undefined ||
		patch.timeout !== undefined ||
		(patch.additionalParametersRawJSON ?? '').trim().length > 0
	);
}

function cloneStartingModelPatch(
	patch?: AssistantPresetStartingModelPresetPatch
): AssistantPresetStartingModelPresetPatch | undefined {
	if (!patch || !hasAssistantPresetModelPatch(patch)) {
		return undefined;
	}

	return {
		...patch,
		stopSequences: patch.stopSequences ? [...patch.stopSequences] : undefined,
		reasoning: patch.reasoning
			? {
					...patch.reasoning,
				}
			: undefined,
		outputParam: patch.outputParam
			? {
					...patch.outputParam,
					format: patch.outputParam.format
						? {
								...patch.outputParam.format,
								jsonSchemaParam: patch.outputParam.format.jsonSchemaParam
									? {
											...patch.outputParam.format.jsonSchemaParam,
											schema: cloneJSONLike(patch.outputParam.format.jsonSchemaParam.schema),
										}
									: undefined,
							}
						: undefined,
				}
			: undefined,
	};
}

export function toPutAssistantPresetPayload(input: AssistantPresetUpsertInput): PutAssistantPresetPayload {
	const payload: PutAssistantPresetPayload = {
		displayName: input.displayName.trim(),
		isEnabled: input.isEnabled,
	};

	const description = input.description?.trim();
	if (description) {
		payload.description = description;
	}

	if (input.startingModelPresetRef) {
		payload.startingModelPresetRef = {
			providerName: input.startingModelPresetRef.providerName,
			modelPresetID: input.startingModelPresetRef.modelPresetID,
		};
	}

	const startingModelPresetPatch = cloneStartingModelPatch(input.startingModelPresetPatch);
	if (startingModelPresetPatch) {
		payload.startingModelPresetPatch = startingModelPresetPatch;
	}

	if (input.startingModelPresetRef && input.startingIncludeModelSystemPrompt !== undefined) {
		payload.startingIncludeModelSystemPrompt = input.startingIncludeModelSystemPrompt;
	}

	if ((input.startingInstructionTemplateRefs?.length ?? 0) > 0) {
		payload.startingInstructionTemplateRefs = input.startingInstructionTemplateRefs?.map(clonePromptTemplateRef);
	}

	if ((input.startingToolSelections?.length ?? 0) > 0) {
		payload.startingToolSelections = input.startingToolSelections?.map(cloneToolSelection);
	}

	if ((input.startingEnabledSkillRefs?.length ?? 0) > 0) {
		payload.startingEnabledSkillRefs = input.startingEnabledSkillRefs?.map(cloneSkillRef);
	}

	return payload;
}
