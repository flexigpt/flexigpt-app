import {
	DefaultModelParams,
	type ModelParam,
	type OutputParam,
	type ProviderName,
	ProviderSDKType,
	type ReasoningParam,
} from '@/spec/inference';

type ModelName = string;
export type ModelDisplayName = string;
type ModelSlug = string;
export type ModelPresetID = string;

export type ProviderDisplayName = string;

/**
 * @public
 */
export interface ToolCapabilitiesOverride {
	supportedToolTypes?: string[];
	supportedToolPolicyModes?: string[];
	supportsParallelToolCalls?: boolean;
	maxForcedTools?: number;
}

/**
 * @public
 */
export interface OutputCapabilitiesOverride {
	supportedOutputFormats?: string[];
	supportsVerbosity?: boolean;
}

/**
 * @public
 */
export interface StopSequenceCapabilitiesOverride {
	isSupported?: boolean;
	disallowedWithReasoning?: boolean;
	maxSequences?: number;
}

/**
 * @public
 */
export interface ReasoningCapabilitiesOverride {
	supportedReasoningTypes?: string[];
	supportedReasoningLevels?: string[];
	supportsSummaryStyle?: boolean;
	supportsEncryptedReasoningInput?: boolean;
	temperatureDisallowedWhenEnabled?: boolean;
}

/**
 * @public
 */
export interface ModelCapabilitiesOverride {
	modalitiesIn?: string[];
	modalitiesOut?: string[];
	reasoningCapabilities?: ReasoningCapabilitiesOverride;
	stopSequenceCapabilities?: StopSequenceCapabilitiesOverride;
	outputCapabilities?: OutputCapabilitiesOverride;
	toolCapabilities?: ToolCapabilitiesOverride;
}

export interface PutModelPresetPayload {
	name: ModelName;
	slug: ModelSlug;
	displayName: ModelDisplayName;
	isEnabled: boolean;
	stream?: boolean;
	maxPromptLength?: number;
	maxOutputLength?: number;
	temperature?: number;
	outputParam?: OutputParam;
	stopSequences?: string[];
	reasoning?: ReasoningParam;
	systemPrompt?: string;
	timeout?: number;
	additionalParametersRawJSON?: string;
	capabilitiesOverride?: ModelCapabilitiesOverride;
}

export interface ModelPreset extends PutModelPresetPayload {
	id: ModelPresetID;
	isBuiltIn: boolean;
}

export interface PutProviderPresetPayload {
	displayName: ProviderDisplayName;
	sdkType: ProviderSDKType;
	isEnabled: boolean;
	origin: string;
	chatCompletionPathPrefix: string;
	apiKeyHeaderKey: string;
	defaultHeaders: Record<string, string>;
	capabilitiesOverride?: ModelCapabilitiesOverride;
}

export interface ProviderPreset extends PutProviderPresetPayload {
	name: ProviderName;
	isBuiltIn: boolean;
	defaultModelPresetID: ModelPresetID;
	modelPresets: Record<ModelPresetID, ModelPreset>;
}

export interface UIChatOption extends ModelParam {
	providerName: ProviderName;
	providerSDKType: ProviderSDKType;
	modelPresetID: ModelPresetID;
	providerDisplayName: ProviderDisplayName;
	modelDisplayName: ModelDisplayName;
	disablePreviousMessages: boolean;
}

export const DefaultUIChatOptions: UIChatOption = {
	...DefaultModelParams,
	providerName: 'no-provider',
	providerSDKType: ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions,
	modelPresetID: 'no-model',
	providerDisplayName: 'No Provider',
	modelDisplayName: 'No Model configured',
	disablePreviousMessages: false,
};
