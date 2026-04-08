import type { AssistantPreset } from '@/spec/assistantpreset';
import {
	type OutputFormatKind,
	type OutputVerbosity,
	type ReasoningLevel,
	type ReasoningSummaryStyle,
	type ReasoningType,
} from '@/spec/inference';
import type { PromptTemplateRef } from '@/spec/prompt';
import type { SkillSelection } from '@/spec/skill';
import type { ToolRef } from '@/spec/tool';

export interface PresetItem {
	preset: AssistantPreset;
	bundleID: string;
	assistantPresetSlug: string;
}

export type ModalMode = 'add' | 'edit' | 'view';
export type TriStateBoolean = '' | 'true' | 'false';

export type ErrorState = {
	displayName?: string;
	slug?: string;
	version?: string;
	modelPreset?: string;
	modelPatch?: string;
	startingInstructionTemplateRefs?: string;
	startingToolSelections?: string;
	startingSkillSelections?: string;
};

export type ModelPatchFormData = {
	enabled: boolean;
	stream: TriStateBoolean;
	maxPromptLength: string;
	maxOutputLength: string;
	temperature: string;
	timeout: string;
	stopSequencesText: string;
	additionalParametersRawJSON: string;

	reasoningEnabled: boolean;
	reasoningType: ReasoningType;
	reasoningLevel: ReasoningLevel;
	reasoningTokens: string;
	reasoningSummaryStyle: '' | ReasoningSummaryStyle;

	outputEnabled: boolean;
	outputVerbosity: '' | OutputVerbosity;
	outputFormatEnabled: boolean;
	outputFormatKind: OutputFormatKind;
	outputJSONSchemaName: string;
	outputJSONSchemaDescription: string;
	outputJSONSchemaRaw: string;
	outputJSONSchemaStrictMode: TriStateBoolean;
};

export type ToolSelectionFormItem = {
	toolRef: ToolRef;
	autoExecuteMode: TriStateBoolean;
	userArgSchemaInstance: string;
};

export type AssistantPresetFormData = {
	displayName: string;
	slug: string;
	description: string;
	isEnabled: boolean;
	version: string;

	startingModelPresetKey: string;
	startingIncludeModelSystemPrompt: TriStateBoolean;
	modelPatch: ModelPatchFormData;

	startingInstructionTemplateRefs: PromptTemplateRef[];
	startingToolSelections: ToolSelectionFormItem[];
	startingSkillSelections: SkillSelection[];
};

export interface SimpleSelectableOption {
	key: string;
	label: string;
}

export interface OrderedDisplayItem {
	key: string;
	title: string;
	subtitle: string;
	statusLabel?: string;
}

export interface ToolSelectionDisplayItem extends OrderedDisplayItem {
	autoExecuteMode: TriStateBoolean;
	autoExecuteLabel: string;
	userArgSchemaInstance: string;
	userArgsHint: string;
	userArgsEditable: boolean;
}
