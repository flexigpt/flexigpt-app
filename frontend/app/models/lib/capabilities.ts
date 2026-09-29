import type { OutputFormatKind, ReasoningLevel } from '@/spec/inference';
import type {
	ModelCacheCapabilities,
	ModelCacheControlCapabilities,
	ModelCapabilities,
	UIModelOption,
} from '@/spec/model';
import {
	CacheControlKind as CacheControlKindValue,
	CacheControlTTL as CacheControlTTLValue,
	ReasoningLevel as ReasoningLevelValue,
} from '@/spec/inference';

function pick<T>(modelValue: T | undefined, providerValue: T | undefined): T | undefined {
	return modelValue !== undefined ? modelValue : providerValue;
}

function mergeCacheControlCapabilities(
	provider: ModelCacheControlCapabilities | undefined,
	model: ModelCacheControlCapabilities | undefined
): ModelCacheControlCapabilities | undefined {
	if (!provider && !model) {
		return undefined;
	}

	return {
		supportsTTL: pick(model?.supportsTTL, provider?.supportsTTL),
		supportedKinds: pick(model?.supportedKinds, provider?.supportedKinds),
		supportedTTLs: pick(model?.supportedTTLs, provider?.supportedTTLs),
		supportsKey: pick(model?.supportsKey, provider?.supportsKey),
	};
}

function mergeCacheCapabilities(
	provider: ModelCacheCapabilities | undefined,
	model: ModelCacheCapabilities | undefined
): ModelCacheCapabilities | undefined {
	if (!provider && !model) {
		return undefined;
	}

	return {
		supportsAutomaticCaching: pick(model?.supportsAutomaticCaching, provider?.supportsAutomaticCaching),
		topLevel: mergeCacheControlCapabilities(provider?.topLevel, model?.topLevel),
		inputOutputContent: mergeCacheControlCapabilities(provider?.inputOutputContent, model?.inputOutputContent),
		reasoningContent: mergeCacheControlCapabilities(provider?.reasoningContent, model?.reasoningContent),
		toolChoice: mergeCacheControlCapabilities(provider?.toolChoice, model?.toolChoice),
		toolCall: mergeCacheControlCapabilities(provider?.toolCall, model?.toolCall),
		toolOutput: mergeCacheControlCapabilities(provider?.toolOutput, model?.toolOutput),
	};
}

export function mergeModelCapabilities(
	provider: ModelCapabilities | undefined,
	model: ModelCapabilities | undefined
): ModelCapabilities | undefined {
	if (!provider && !model) {
		return undefined;
	}

	return {
		modalitiesIn: pick(model?.modalitiesIn, provider?.modalitiesIn),
		modalitiesOut: pick(model?.modalitiesOut, provider?.modalitiesOut),
		reasoningCapabilities:
			provider?.reasoningCapabilities || model?.reasoningCapabilities
				? {
						supportsReasoningConfig: pick(
							model?.reasoningCapabilities?.supportsReasoningConfig,
							provider?.reasoningCapabilities?.supportsReasoningConfig
						),
						supportedReasoningTypes: pick(
							model?.reasoningCapabilities?.supportedReasoningTypes,
							provider?.reasoningCapabilities?.supportedReasoningTypes
						),
						supportedReasoningLevels: pick(
							model?.reasoningCapabilities?.supportedReasoningLevels,
							provider?.reasoningCapabilities?.supportedReasoningLevels
						),
						hybridTokenBudgetCapabilities: pick(
							model?.reasoningCapabilities?.hybridTokenBudgetCapabilities,
							provider?.reasoningCapabilities?.hybridTokenBudgetCapabilities
						),
						supportsSummaryStyle: pick(
							model?.reasoningCapabilities?.supportsSummaryStyle,
							provider?.reasoningCapabilities?.supportsSummaryStyle
						),
						supportsReasoningContext: pick(
							model?.reasoningCapabilities?.supportsReasoningContext,
							provider?.reasoningCapabilities?.supportsReasoningContext
						),
						supportsReasoningMode: pick(
							model?.reasoningCapabilities?.supportsReasoningMode,
							provider?.reasoningCapabilities?.supportsReasoningMode
						),
						supportsEncryptedReasoningInput: pick(
							model?.reasoningCapabilities?.supportsEncryptedReasoningInput,
							provider?.reasoningCapabilities?.supportsEncryptedReasoningInput
						),
						temperatureDisallowedWhenEnabled: pick(
							model?.reasoningCapabilities?.temperatureDisallowedWhenEnabled,
							provider?.reasoningCapabilities?.temperatureDisallowedWhenEnabled
						),
					}
				: undefined,
		stopSequenceCapabilities:
			provider?.stopSequenceCapabilities || model?.stopSequenceCapabilities
				? {
						isSupported: pick(
							model?.stopSequenceCapabilities?.isSupported,
							provider?.stopSequenceCapabilities?.isSupported
						),
						disallowedWithReasoning: pick(
							model?.stopSequenceCapabilities?.disallowedWithReasoning,
							provider?.stopSequenceCapabilities?.disallowedWithReasoning
						),
						maxSequences: pick(
							model?.stopSequenceCapabilities?.maxSequences,
							provider?.stopSequenceCapabilities?.maxSequences
						),
					}
				: undefined,
		outputCapabilities:
			provider?.outputCapabilities || model?.outputCapabilities
				? {
						supportedOutputFormats: pick(
							model?.outputCapabilities?.supportedOutputFormats,
							provider?.outputCapabilities?.supportedOutputFormats
						),
						supportsVerbosity: pick(
							model?.outputCapabilities?.supportsVerbosity,
							provider?.outputCapabilities?.supportsVerbosity
						),
					}
				: undefined,
		toolCapabilities:
			provider?.toolCapabilities || model?.toolCapabilities
				? {
						supportedToolTypes: pick(
							model?.toolCapabilities?.supportedToolTypes,
							provider?.toolCapabilities?.supportedToolTypes
						),
						supportedToolPolicyModes: pick(
							model?.toolCapabilities?.supportedToolPolicyModes,
							provider?.toolCapabilities?.supportedToolPolicyModes
						),
						supportsParallelToolCalls: pick(
							model?.toolCapabilities?.supportsParallelToolCalls,
							provider?.toolCapabilities?.supportsParallelToolCalls
						),
						maxForcedTools: pick(model?.toolCapabilities?.maxForcedTools, provider?.toolCapabilities?.maxForcedTools),
						supportedClientToolOutputFormats: pick(
							model?.toolCapabilities?.supportedClientToolOutputFormats,
							provider?.toolCapabilities?.supportedClientToolOutputFormats
						),
					}
				: undefined,
		cacheCapabilities: mergeCacheCapabilities(provider?.cacheCapabilities, model?.cacheCapabilities),
		paramDialect:
			provider?.paramDialect || model?.paramDialect
				? {
						maxOutputTokensParamName: pick(
							model?.paramDialect?.maxOutputTokensParamName,
							provider?.paramDialect?.maxOutputTokensParamName
						),
						toolChoiceParamStyle: pick(
							model?.paramDialect?.toolChoiceParamStyle,
							provider?.paramDialect?.toolChoiceParamStyle
						),
					}
				: undefined,
	};
}

const orderedReasoningLevels: ReasoningLevel[] = [
	ReasoningLevelValue.None,
	ReasoningLevelValue.Minimal,
	ReasoningLevelValue.Low,
	ReasoningLevelValue.Medium,
	ReasoningLevelValue.High,
	ReasoningLevelValue.XHigh,
	ReasoningLevelValue.Max,
];

export function getSupportedReasoningLevels(capabilities: ModelCapabilities | undefined): ReasoningLevel[] {
	const configured = capabilities?.reasoningCapabilities?.supportedReasoningLevels;

	if (!configured || configured.length === 0) {
		return [...orderedReasoningLevels];
	}

	const enabled = new Set(configured);
	return orderedReasoningLevels.filter(level => enabled.has(level));
}

export function supportsReasoningSummaryStyle(capabilities: ModelCapabilities | undefined): boolean {
	return capabilities?.reasoningCapabilities?.supportsSummaryStyle !== false;
}

export function supportsOutputVerbosity(capabilities: ModelCapabilities | undefined): boolean {
	return capabilities?.outputCapabilities?.supportsVerbosity !== false;
}

export function getSupportedOutputFormats(capabilities: ModelCapabilities | undefined): OutputFormatKind[] | undefined {
	const configured = capabilities?.outputCapabilities?.supportedOutputFormats;

	if (!configured || configured.length === 0) {
		return undefined;
	}

	return configured;
}

export function getStopSequencesPolicy(capabilities: ModelCapabilities | undefined): {
	isSupported: boolean;
	disallowedWithReasoning: boolean;
	maxSequences: number;
} {
	return {
		isSupported: capabilities?.stopSequenceCapabilities?.isSupported !== false,
		disallowedWithReasoning: capabilities?.stopSequenceCapabilities?.disallowedWithReasoning === true,
		maxSequences: capabilities?.stopSequenceCapabilities?.maxSequences ?? 16,
	};
}

function sdkBaseCacheCapabilities(sdkType: string): ModelCacheCapabilities | undefined {
	switch (sdkType) {
		case 'providerSDKTypeAnthropicMessages':
			return {
				supportsAutomaticCaching: false,
				topLevel: {
					supportedKinds: [CacheControlKindValue.Ephemeral],
					supportedTTLs: [CacheControlTTLValue.TTL5m, CacheControlTTLValue.TTL1h],
					supportsKey: false,
				},
			};
		case 'providerSDKTypeOpenAIResponses':
			return {
				supportsAutomaticCaching: true,
				topLevel: {
					supportedKinds: [CacheControlKindValue.Ephemeral],
					supportedTTLs: [CacheControlTTLValue.TTLInMemory, CacheControlTTLValue.TTL24h],
					supportsKey: true,
				},
			};
		default:
			return undefined;
	}
}

export function getEffectiveCacheCapabilities(
	sdkType: string,
	capabilities: ModelCapabilities | undefined
): ModelCacheCapabilities | undefined {
	return mergeCacheCapabilities(sdkBaseCacheCapabilities(sdkType), capabilities?.cacheCapabilities);
}

export function getTopLevelCacheControlCapabilities(
	sdkType: string,
	capabilities: ModelCapabilities | undefined
): ModelCacheControlCapabilities | undefined {
	return getEffectiveCacheCapabilities(sdkType, capabilities)?.topLevel;
}

export function sanitizeUIModelOptionByCapabilities(option: UIModelOption): UIModelOption {
	const next: UIModelOption = {
		...option,
	};

	const supportedReasoning = next.capabilities?.reasoningCapabilities?.supportedReasoningTypes;

	if (
		next.reasoning &&
		supportedReasoning &&
		supportedReasoning.length > 0 &&
		!supportedReasoning.includes(next.reasoning.type)
	) {
		delete next.reasoning;
	}

	if (next.reasoning && next.capabilities?.reasoningCapabilities?.temperatureDisallowedWhenEnabled) {
		delete next.temperature;
	}

	if (next.capabilities?.outputCapabilities?.supportsVerbosity === false && next.outputParam) {
		const output = {
			...next.outputParam,
		};
		delete output.verbosity;
		next.outputParam = output.format ? output : undefined;
	}

	const stopPolicy = getStopSequencesPolicy(next.capabilities);
	if (!stopPolicy.isSupported || (stopPolicy.disallowedWithReasoning && next.reasoning)) {
		delete next.stopSequences;
	} else if (next.stopSequences && next.stopSequences.length > stopPolicy.maxSequences) {
		next.stopSequences = next.stopSequences.slice(0, stopPolicy.maxSequences);
	}

	return next;
}
