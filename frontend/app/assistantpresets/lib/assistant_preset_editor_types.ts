import type { AssistantPreset } from '@/spec/assistantpreset';
import type {
	OutputFormatKind,
	OutputVerbosity,
	ReasoningLevel,
	ReasoningSummaryStyle,
	ReasoningType,
} from '@/spec/inference';
import type { MCPConversationContext } from '@/spec/mcp';
import type { SkillSelection } from '@/spec/skill';
import type { ToolRef } from '@/spec/tool';

export interface PresetItem {
	preset: AssistantPreset;
	bundleID: string;
	assistantPresetSlug: string;
}

export type ModalMode = 'add' | 'edit' | 'view';
export type TriStateBoolean = '' | 'true' | 'false';

export interface ErrorState {
	displayName?: string;
	slug?: string;
	version?: string;
	modelPreset?: string;
	modelPatch?: string;
	startingToolSelections?: string;
	startingSkillSelections?: string;
	startingMCPContext?: string;
}

export interface ModelPatchFormData {
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
}

interface ToolSelectionFormItem {
	toolRef: ToolRef;
	autoExecuteMode: TriStateBoolean;
	userArgSchemaInstance: string;
}

export interface AssistantPresetFormData {
	displayName: string;
	slug: string;
	version: string;
	description: string;
	isEnabled: boolean;

	startingText: string;
	startingModelPresetKey: string;
	startingIncludeModelSystemPrompt: TriStateBoolean;
	modelPatch: ModelPatchFormData;

	startingToolSelections: ToolSelectionFormItem[];
	startingSkillSelections: SkillSelection[];
	startingMCPContext?: MCPConversationContext;
}

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

export interface SkillSelectionDisplayItem extends OrderedDisplayItem {
	preLoadAsActive: boolean;
	useAsInstructions: boolean;
	canUseAsInstructions: boolean;
	useAsInstructionsDisabledReason?: string;
	canPreLoadAsActive: boolean;
	preLoadAsActiveDisabledReason?: string;
}
