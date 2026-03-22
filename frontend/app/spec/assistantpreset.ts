import type { ModelPresetPatch, ModelPresetRef } from '@/spec/modelpreset';
import type { PromptTemplateRef } from '@/spec/prompt';
import type { SkillRef } from '@/spec/skill';
import type { ToolSelection } from '@/spec/tool';

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
	icon?: string;
	isEnabled: boolean;
	isBuiltIn: boolean;
	startingModelPresetRef?: ModelPresetRef;
	startingModelPresetPatch?: ModelPresetPatch;
	startingIncludeModelSystemPrompt?: boolean;
	startingInstructionTemplateRefs?: PromptTemplateRef[];
	startingToolSelections?: ToolSelection[];
	startingEnabledSkillRefs?: SkillRef[];
	createdAt: Date; // Go type: time
	modifiedAt: Date; // Go type: time
}

/**
 * @public
 */
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

/**
 * @public
 */
export interface AssistantPresetListItem {
	bundleID: string;
	bundleSlug: string;
	assistantPresetSlug: string;
	assistantPresetVersion: string;
	displayName: string;
	description?: string;
	icon?: string;
	isEnabled: boolean;
	isBuiltIn: boolean;
	modifiedAt?: Date; // Go type: time
}
