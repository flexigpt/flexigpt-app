import type { AssistantPreset, PutAssistantPresetPayload } from '@/spec/assistantpreset';
import type { MCPConversationContext } from '@/spec/mcp';
import type { ModelPresetRef } from '@/spec/modelpreset';
import type { SkillRef, SkillSelection } from '@/spec/skill';
import type { ToolRef, ToolSelection } from '@/spec/tool';

import { cloneJSONLike } from '@/lib/jsonschema_utils';

export interface AssistantPresetUpsertInput extends PutAssistantPresetPayload {
	slug: string;
	version: string;
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

export function cloneMCPConversationContext(context?: MCPConversationContext): MCPConversationContext | undefined {
	if (!context) {
		return undefined;
	}

	return cloneJSONLike(context);
}

export function hasAssistantPresetMCPContext(context?: MCPConversationContext): boolean {
	if (!context) {
		return false;
	}

	return (
		(context.servers?.length ?? 0) > 0 ||
		(context.resources?.length ?? 0) > 0 ||
		(context.resourceTemplates?.length ?? 0) > 0 ||
		(context.prompts?.length ?? 0) > 0
	);
}

function cloneSkillRef(ref: SkillRef): SkillRef {
	return {
		bundleID: ref.bundleID,
		skillSlug: ref.skillSlug,
		skillID: ref.skillID,
	};
}

export function cloneSkillSelection(sel: SkillSelection): SkillSelection {
	return {
		skillRef: cloneSkillRef(sel.skillRef),
		preLoadAsActive: sel.preLoadAsActive,
		useAsInstructions: sel.useAsInstructions,
	};
}

export function formatAssistantPresetModelRef(ref?: ModelPresetRef): string {
	if (!ref) {
		return '—';
	}
	return `${ref.providerName}/${ref.modelPresetID}`;
}

export function getAssistantPresetCounts(
	preset: Pick<AssistantPreset, 'startingToolSelections' | 'startingSkillSelections' | 'startingMCPContext'>
) {
	return {
		instructions: (preset.startingSkillSelections ?? []).filter(selection => selection.useAsInstructions).length,
		tools: preset.startingToolSelections?.length ?? 0,
		skills: (preset.startingSkillSelections ?? []).filter(selection => !selection.useAsInstructions).length,
		mcp:
			(preset.startingMCPContext?.servers?.length ?? 0) +
			(preset.startingMCPContext?.resources?.length ?? 0) +
			(preset.startingMCPContext?.resourceTemplates?.length ?? 0) +
			(preset.startingMCPContext?.prompts?.length ?? 0),
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

export function toPutAssistantPresetPayload(input: AssistantPresetUpsertInput): PutAssistantPresetPayload {
	const payload: PutAssistantPresetPayload = {
		displayName: input.displayName.trim(),
		isEnabled: input.isEnabled,
	};

	const description = input.description?.trim();
	if (description) {
		payload.description = description;
	}

	const startingText = input.startingText?.trim();
	if (startingText) {
		payload.startingText = startingText;
	}

	if (input.startingModelPresetRef) {
		payload.startingModelPresetRef = {
			providerName: input.startingModelPresetRef.providerName,
			modelPresetID: input.startingModelPresetRef.modelPresetID,
		};
	}

	if (input.startingModelPresetRef && input.startingIncludeModelSystemPrompt !== undefined) {
		payload.startingIncludeModelSystemPrompt = input.startingIncludeModelSystemPrompt;
	}

	if ((input.startingToolSelections?.length ?? 0) > 0) {
		payload.startingToolSelections = input.startingToolSelections?.map(cloneToolSelection);
	}

	if ((input.startingSkillSelections?.length ?? 0) > 0) {
		payload.startingSkillSelections = input.startingSkillSelections?.map(cloneSkillSelection);
	}

	if (hasAssistantPresetMCPContext(input.startingMCPContext)) {
		payload.startingMCPContext = cloneMCPConversationContext(input.startingMCPContext);
	}

	return payload;
}
