import {
	type CacheControl,
	type CacheControlKind,
	type CacheControlTTL,
	DefaultModelParams,
	type ModelParam,
	type OutputParam,
	type ProviderName,
	ProviderSDKType,
	type ReasoningParam,
} from '@/spec/inference';

export const PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY = '__conversation__:previous-system-prompt';
export const PREVIOUS_CONVO_SYSTEM_PROMPT_BUNDLEID = '__conversation__';
export const DEFAULT_REASONING_TOKENS = 1024;

type ModelName = string;
export type ModelDisplayName = string;
type ModelSlug = string;
export type ModelPresetID = string;

export interface ModelPresetRef {
	providerName: ProviderName;
	modelPresetID: ModelPresetID;
}

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
export class CacheControlCapabilitiesOverride {
	supportedKinds?: CacheControlKind[];
	supportedTTLs?: CacheControlTTL[];
	supportsKey?: boolean;
}

/**
 * @public
 */
export class CacheCapabilitiesOverride {
	supportsAutomaticCaching?: boolean;
	topLevel?: CacheControlCapabilitiesOverride;
	inputOutputContent?: CacheControlCapabilitiesOverride;
	reasoningContent?: CacheControlCapabilitiesOverride;
	toolChoice?: CacheControlCapabilitiesOverride;
	toolCall?: CacheControlCapabilitiesOverride;
	toolOutput?: CacheControlCapabilitiesOverride;
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
	cacheCapabilities?: CacheCapabilitiesOverride;
}

/**
 * @public
 */
export interface ModelPresetPatch {
	stream?: boolean;
	maxPromptLength?: number;
	maxOutputLength?: number;
	temperature?: number;
	outputParam?: OutputParam;
	stopSequences?: string[];
	reasoning?: ReasoningParam;
	systemPrompt?: string;
	timeout?: number;
	cacheControl?: CacheControl;
	additionalParametersRawJSON?: string;
	capabilitiesOverride?: ModelCapabilitiesOverride;
}

export interface PostModelPresetPayload extends ModelPresetPatch {
	name: ModelName;
	slug: ModelSlug;
	displayName: ModelDisplayName;
	isEnabled: boolean;
}

export interface PatchModelPresetPayload extends ModelPresetPatch {
	name?: ModelName;
	slug?: ModelSlug;
	displayName?: ModelDisplayName;
	isEnabled?: boolean;
}

type ISODateString = string;

export interface ModelPreset extends ModelPresetPatch {
	schemaVersion: string;
	id: ModelPresetID;
	name: ModelName;
	slug: ModelSlug;
	displayName: ModelDisplayName;
	isEnabled: boolean;
	capabilitiesOverride?: ModelCapabilitiesOverride;
	createdAt: ISODateString;
	modifiedAt: ISODateString;
	isBuiltIn: boolean;
}

export interface PostProviderPresetPayload {
	displayName: ProviderDisplayName;
	sdkType: ProviderSDKType;
	isEnabled: boolean;
	origin: string;
	chatCompletionPathPrefix: string;

	apiKeyHeaderKey?: string;
	defaultHeaders?: Record<string, string>;
	capabilitiesOverride?: ModelCapabilitiesOverride;
}

export interface PatchProviderPresetPayload {
	displayName?: ProviderDisplayName;
	sdkType?: ProviderSDKType;
	isEnabled?: boolean;
	origin?: string;
	chatCompletionPathPrefix?: string;
	apiKeyHeaderKey?: string;
	defaultHeaders?: Record<string, string>;
	defaultModelPresetID?: ModelPresetID;
	capabilitiesOverride?: ModelCapabilitiesOverride;
}

export interface ProviderPreset {
	schemaVersion: string;
	name: ProviderName;
	displayName: ProviderDisplayName;
	sdkType: ProviderSDKType;
	isEnabled: boolean;
	createdAt: ISODateString;
	modifiedAt: ISODateString;
	isBuiltIn: boolean;
	origin: string;
	chatCompletionPathPrefix: string;
	apiKeyHeaderKey: string;
	defaultHeaders: Record<string, string>;
	capabilitiesOverride?: ModelCapabilitiesOverride;
	defaultModelPresetID: ModelPresetID;
	modelPresets: Record<ModelPresetID, ModelPreset>;
}

export type IncludePreviousMessages = number | 'all';

export interface UIChatOption extends ModelParam {
	providerName: ProviderName;
	providerSDKType: ProviderSDKType;
	modelPresetID: ModelPresetID;
	providerDisplayName: ProviderDisplayName;
	modelDisplayName: ModelDisplayName;
	/**
	 * How many earlier conversation messages to include in addition to the
	 * current submitted user message.
	 * - 'all' => full history
	 * - 0 => current message only
	 * - n => last n previous messages + current message
	 */
	includePreviousMessages: IncludePreviousMessages;
	/**
	 * Effective (provider + model merged) capability overrides for this selectable option.
	 * Merged stored capability overrides for this selectable option.
	 * This is still an override layer, not a fully-derived effective capability profile.
	 * Model-level override wins over provider-level override.
	 */
	capabilitiesOverride?: ModelCapabilitiesOverride;
}

export interface AssistantModelPresetOption {
	key: string;
	label: string;

	ref: ModelPresetRef;
	providerPreset: ProviderPreset;
	modelPreset: ModelPreset;

	isBuiltIn: boolean;
	isSelectable: boolean;
	isProviderEnabled: boolean;
	isModelEnabled: boolean;
	availabilityReason?: string;
}

export const DefaultUIChatOptions: UIChatOption = {
	...DefaultModelParams,
	providerName: 'no-provider',
	providerSDKType: ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions,
	modelPresetID: 'no-model',
	providerDisplayName: 'No Provider',
	modelDisplayName: 'No Model configured',
	includePreviousMessages: 'all',
	capabilitiesOverride: undefined,
};
