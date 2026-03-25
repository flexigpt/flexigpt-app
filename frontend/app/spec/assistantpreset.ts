import type { ModelPresetPatch, ModelPresetRef } from '@/spec/modelpreset';
import type { PromptTemplateRef } from '@/spec/prompt';
import type { SkillRef } from '@/spec/skill';
import type { ToolSelection } from '@/spec/tool';

/**
 * @public
 */
export const BASE_ASSISTANT_PRESET_SLUG = 'base';

/**
 * Assistant preset write-time subset of ModelPresetPatch.
 *
 * Mirrors Go validation for assistant presets:
 * - systemPrompt must be nil
 * - capabilitiesOverride must be nil
 */
export type AssistantPresetStartingModelPresetPatch = Omit<
	ModelPresetPatch,
	'systemPrompt' | 'capabilitiesOverride'
> & {
	systemPrompt?: never;
	capabilitiesOverride?: never;
};

/**
 * @public
 */
export interface AssistantPreset {
	schemaVersion: string;
	id: string;
	slug: string;
	version: string;
	displayName: string;
	description?: string;
	isEnabled: boolean;
	isBuiltIn: boolean;
	startingModelPresetRef?: ModelPresetRef;
	startingModelPresetPatch?: AssistantPresetStartingModelPresetPatch;
	startingIncludeModelSystemPrompt?: boolean;
	startingInstructionTemplateRefs?: PromptTemplateRef[];
	startingToolSelections?: ToolSelection[];
	startingEnabledSkillRefs?: SkillRef[];
	createdAt: Date; // Go type: time
	modifiedAt: Date; // Go type: time
}

export interface AssistantPresetBundle {
	schemaVersion: string;
	id: string;
	slug: string;
	displayName: string;
	description?: string;
	isEnabled: boolean;
	isBuiltIn: boolean;
	createdAt: Date; // Go type: time
	modifiedAt: Date; // Go type: time
	softDeletedAt?: Date; // Go type: time
}

export interface AssistantPresetListItem {
	bundleID: string;
	bundleSlug: string;
	assistantPresetSlug: string;
	assistantPresetVersion: string;
	displayName: string;
	description?: string;
	isEnabled: boolean;
	isBuiltIn: boolean;
	modifiedAt?: Date; // Go type: time
}

/**
 * App-facing write payload for putAssistantPreset().
 * No "Body" wrapper type on the frontend boundary.
 */
export interface PutAssistantPresetPayload {
	displayName: string;
	description?: string;
	isEnabled: boolean;
	startingModelPresetRef?: ModelPresetRef;
	startingModelPresetPatch?: AssistantPresetStartingModelPresetPatch;
	startingIncludeModelSystemPrompt?: boolean;
	startingInstructionTemplateRefs?: PromptTemplateRef[];
	startingToolSelections?: ToolSelection[];
	startingEnabledSkillRefs?: SkillRef[];
}
