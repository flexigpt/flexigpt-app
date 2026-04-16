import {
	CacheControlKind,
	CacheControlTTL,
	DefaultModelParams,
	OutputFormatKind,
	ProviderSDKType,
	ReasoningLevel,
	ReasoningType,
} from '@/spec/inference';
import type {
	CacheCapabilitiesOverride,
	CacheControlCapabilitiesOverride,
	ModelCapabilitiesOverride,
	UIChatOption,
} from '@/spec/modelpreset';

function pick<T>(modelVal: T | undefined, providerVal: T | undefined): T | undefined {
	return modelVal !== undefined ? modelVal : providerVal;
}

function isReasoningLevel(v: string): v is ReasoningLevel {
	return (Object.values(ReasoningLevel) as string[]).includes(v);
}

function isReasoningType(v: string): v is ReasoningType {
	return (Object.values(ReasoningType) as string[]).includes(v);
}

function isOutputFormatKind(v: string): v is OutputFormatKind {
	return (Object.values(OutputFormatKind) as string[]).includes(v);
}

export const CACHE_CONTROL_KIND_LABELS: Record<CacheControlKind, string> = {
	[CacheControlKind.Ephemeral]: 'Ephemeral',
};

export const CACHE_CONTROL_TTL_LABELS: Record<CacheControlTTL, string> = {
	[CacheControlTTL.TTL5m]: '5 minutes',
	[CacheControlTTL.TTL1h]: '1 hour',
	[CacheControlTTL.TTL24h]: '24 hours',
	[CacheControlTTL.TTLInMemory]: 'In-memory',
};

function mergeCacheControlCapabilities(
	providerVal?: CacheControlCapabilitiesOverride,
	modelVal?: CacheControlCapabilitiesOverride
): CacheControlCapabilitiesOverride | undefined {
	if (!providerVal && !modelVal) return undefined;

	const merged: CacheControlCapabilitiesOverride = {
		supportedKinds: pick(modelVal?.supportedKinds, providerVal?.supportedKinds),
		supportedTTLs: pick(modelVal?.supportedTTLs, providerVal?.supportedTTLs),
		supportsKey: pick(modelVal?.supportsKey, providerVal?.supportsKey),
		supportsTTL: pick(modelVal?.supportsTTL, providerVal?.supportsTTL),
	};

	const hasAny =
		merged.supportedKinds !== undefined ||
		merged.supportedTTLs !== undefined ||
		merged.supportsKey !== undefined ||
		merged.supportsTTL !== undefined;

	return hasAny ? merged : undefined;
}

function mergeCacheCapabilities(
	providerVal?: CacheCapabilitiesOverride,
	modelVal?: CacheCapabilitiesOverride
): CacheCapabilitiesOverride | undefined {
	if (!providerVal && !modelVal) return undefined;

	const merged: CacheCapabilitiesOverride = {
		supportsAutomaticCaching: pick(modelVal?.supportsAutomaticCaching, providerVal?.supportsAutomaticCaching),
		topLevel: mergeCacheControlCapabilities(providerVal?.topLevel, modelVal?.topLevel),
		inputOutputContent: mergeCacheControlCapabilities(providerVal?.inputOutputContent, modelVal?.inputOutputContent),
		reasoningContent: mergeCacheControlCapabilities(providerVal?.reasoningContent, modelVal?.reasoningContent),
		toolChoice: mergeCacheControlCapabilities(providerVal?.toolChoice, modelVal?.toolChoice),
		toolCall: mergeCacheControlCapabilities(providerVal?.toolCall, modelVal?.toolCall),
		toolOutput: mergeCacheControlCapabilities(providerVal?.toolOutput, modelVal?.toolOutput),
	};

	const hasAny =
		merged.supportsAutomaticCaching !== undefined ||
		merged.topLevel !== undefined ||
		merged.inputOutputContent !== undefined ||
		merged.reasoningContent !== undefined ||
		merged.toolChoice !== undefined ||
		merged.toolCall !== undefined ||
		merged.toolOutput !== undefined;

	return hasAny ? merged : undefined;
}

const ORDERED_REASONING_LEVELS: ReasoningLevel[] = [
	ReasoningLevel.None,
	ReasoningLevel.Minimal,
	ReasoningLevel.Low,
	ReasoningLevel.Medium,
	ReasoningLevel.High,
	ReasoningLevel.XHigh,
];

export function mergeModelCapabilitiesOverride(
	providerOverride?: ModelCapabilitiesOverride,
	modelOverride?: ModelCapabilitiesOverride
): ModelCapabilitiesOverride | undefined {
	if (!providerOverride && !modelOverride) return undefined;

	const p = providerOverride;
	const m = modelOverride;

	const merged: ModelCapabilitiesOverride = {
		modalitiesIn: pick(m?.modalitiesIn, p?.modalitiesIn),
		modalitiesOut: pick(m?.modalitiesOut, p?.modalitiesOut),

		paramDialect:
			m?.paramDialect || p?.paramDialect
				? {
						maxOutputTokensParamName: pick(
							m?.paramDialect?.maxOutputTokensParamName,
							p?.paramDialect?.maxOutputTokensParamName
						),
						toolChoiceParamStyle: pick(m?.paramDialect?.toolChoiceParamStyle, p?.paramDialect?.toolChoiceParamStyle),
					}
				: undefined,

		reasoningCapabilities:
			m?.reasoningCapabilities || p?.reasoningCapabilities
				? {
						supportsReasoningConfig: pick(
							m?.reasoningCapabilities?.supportsReasoningConfig,
							p?.reasoningCapabilities?.supportsReasoningConfig
						),
						supportedReasoningTypes: pick(
							m?.reasoningCapabilities?.supportedReasoningTypes,
							p?.reasoningCapabilities?.supportedReasoningTypes
						),
						supportedReasoningLevels: pick(
							m?.reasoningCapabilities?.supportedReasoningLevels,
							p?.reasoningCapabilities?.supportedReasoningLevels
						),
						supportsSummaryStyle: pick(
							m?.reasoningCapabilities?.supportsSummaryStyle,
							p?.reasoningCapabilities?.supportsSummaryStyle
						),
						supportsEncryptedReasoningInput: pick(
							m?.reasoningCapabilities?.supportsEncryptedReasoningInput,
							p?.reasoningCapabilities?.supportsEncryptedReasoningInput
						),
						temperatureDisallowedWhenEnabled: pick(
							m?.reasoningCapabilities?.temperatureDisallowedWhenEnabled,
							p?.reasoningCapabilities?.temperatureDisallowedWhenEnabled
						),
					}
				: undefined,

		stopSequenceCapabilities:
			m?.stopSequenceCapabilities || p?.stopSequenceCapabilities
				? {
						isSupported: pick(m?.stopSequenceCapabilities?.isSupported, p?.stopSequenceCapabilities?.isSupported),
						disallowedWithReasoning: pick(
							m?.stopSequenceCapabilities?.disallowedWithReasoning,
							p?.stopSequenceCapabilities?.disallowedWithReasoning
						),
						maxSequences: pick(m?.stopSequenceCapabilities?.maxSequences, p?.stopSequenceCapabilities?.maxSequences),
					}
				: undefined,

		outputCapabilities:
			m?.outputCapabilities || p?.outputCapabilities
				? {
						supportedOutputFormats: pick(
							m?.outputCapabilities?.supportedOutputFormats,
							p?.outputCapabilities?.supportedOutputFormats
						),
						supportsVerbosity: pick(m?.outputCapabilities?.supportsVerbosity, p?.outputCapabilities?.supportsVerbosity),
					}
				: undefined,

		toolCapabilities:
			m?.toolCapabilities || p?.toolCapabilities
				? {
						supportedToolTypes: pick(m?.toolCapabilities?.supportedToolTypes, p?.toolCapabilities?.supportedToolTypes),
						supportedToolPolicyModes: pick(
							m?.toolCapabilities?.supportedToolPolicyModes,
							p?.toolCapabilities?.supportedToolPolicyModes
						),
						supportedClientToolOutputFormats: pick(
							m?.toolCapabilities?.supportedClientToolOutputFormats,
							p?.toolCapabilities?.supportedClientToolOutputFormats
						),
						supportsParallelToolCalls: pick(
							m?.toolCapabilities?.supportsParallelToolCalls,
							p?.toolCapabilities?.supportsParallelToolCalls
						),
						maxForcedTools: pick(m?.toolCapabilities?.maxForcedTools, p?.toolCapabilities?.maxForcedTools),
					}
				: undefined,

		cacheCapabilities: mergeCacheCapabilities(p?.cacheCapabilities, m?.cacheCapabilities),
	};

	// If everything is undefined, collapse to undefined
	const hasAny =
		merged.modalitiesIn ||
		merged.modalitiesOut ||
		merged.reasoningCapabilities ||
		merged.stopSequenceCapabilities ||
		merged.outputCapabilities ||
		merged.toolCapabilities ||
		merged.cacheCapabilities;
	return hasAny ? merged : undefined;
}

export function getSupportedReasoningLevels(cap?: ModelCapabilitiesOverride): ReasoningLevel[] {
	const raw = cap?.reasoningCapabilities?.supportedReasoningLevels;
	if (!raw || raw.length === 0) {
		// Default UI list (existing behavior, but now includes XHigh)
		return [
			ReasoningLevel.None,
			ReasoningLevel.Minimal,
			ReasoningLevel.Low,
			ReasoningLevel.Medium,
			ReasoningLevel.High,
		];
	}

	const set = new Set(raw.filter(isReasoningLevel));
	const ordered = ORDERED_REASONING_LEVELS.filter(l => set.has(l));
	return ordered.length > 0 ? ordered : [ReasoningLevel.Low, ReasoningLevel.Medium, ReasoningLevel.High];
}

export function supportsReasoningSummaryStyle(cap?: ModelCapabilitiesOverride): boolean {
	return cap?.reasoningCapabilities?.supportsSummaryStyle !== false;
}

export function supportsOutputVerbosity(cap?: ModelCapabilitiesOverride): boolean {
	return cap?.outputCapabilities?.supportsVerbosity !== false;
}

export function getSupportedOutputFormats(cap?: ModelCapabilitiesOverride): OutputFormatKind[] | undefined {
	const raw = cap?.outputCapabilities?.supportedOutputFormats;
	if (!raw || raw.length === 0) return undefined; // undefined means "no restriction"
	const out = raw.filter(isOutputFormatKind);
	return out.length > 0 ? out : undefined;
}

export function getStopSequencesPolicy(cap?: ModelCapabilitiesOverride): {
	isSupported: boolean;
	disallowedWithReasoning: boolean;
	maxSequences: number;
} {
	return {
		isSupported: cap?.stopSequenceCapabilities?.isSupported !== false,
		disallowedWithReasoning: cap?.stopSequenceCapabilities?.disallowedWithReasoning === true,
		maxSequences: cap?.stopSequenceCapabilities?.maxSequences ?? 16,
	};
}

function getSDKBaseCacheCapabilities(providerSDKType: ProviderSDKType): CacheCapabilitiesOverride | undefined {
	switch (providerSDKType) {
		case ProviderSDKType.ProviderSDKTypeAnthropic:
			return {
				supportsAutomaticCaching: false,
				topLevel: {
					supportedKinds: [CacheControlKind.Ephemeral],
					supportedTTLs: [CacheControlTTL.TTL5m, CacheControlTTL.TTL1h],
					supportsKey: false,
				},
			};
		case ProviderSDKType.ProviderSDKTypeOpenAIResponses:
			return {
				supportsAutomaticCaching: true,
				topLevel: {
					supportedKinds: [CacheControlKind.Ephemeral],
					supportedTTLs: [CacheControlTTL.TTLInMemory, CacheControlTTL.TTL24h],
					supportsKey: true,
				},
			};
		case (ProviderSDKType.ProviderSDKTypeOpenAIChatCompletions, ProviderSDKType.ProviderSDKTypeGoogleGenerateContent):
		default:
			return {
				supportsAutomaticCaching: false,
			};
	}
}

export function getEffectiveCacheCapabilities(
	providerSDKType: ProviderSDKType,
	cap?: ModelCapabilitiesOverride
): CacheCapabilitiesOverride | undefined {
	return mergeCacheCapabilities(getSDKBaseCacheCapabilities(providerSDKType), cap?.cacheCapabilities);
}

export function getTopLevelCacheControlCapabilities(
	providerSDKType: ProviderSDKType,
	cap?: ModelCapabilitiesOverride
): CacheControlCapabilitiesOverride | undefined {
	return getEffectiveCacheCapabilities(providerSDKType, cap)?.topLevel;
}

export function sanitizeUIChatOptionByCapabilities(option: UIChatOption): UIChatOption {
	const cap = option.capabilitiesOverride;
	const topLevelCacheCapabilities = getTopLevelCacheControlCapabilities(option.providerSDKType, cap);

	if (!cap && !topLevelCacheCapabilities && !option.cacheControl) return option;

	let next: UIChatOption = { ...option };

	// --- Reasoning sanitization ---
	const supportedTypesRaw = cap?.reasoningCapabilities?.supportedReasoningTypes;
	if (supportedTypesRaw && next.reasoning) {
		const supportedTypes = new Set(supportedTypesRaw.filter(isReasoningType));
		if (supportedTypes.size > 0 && !supportedTypes.has(next.reasoning.type)) {
			delete next.reasoning;
		}
	}

	if (next.reasoning?.type === ReasoningType.SingleWithLevels) {
		const allowedLevels = getSupportedReasoningLevels(cap);
		if (!allowedLevels.includes(next.reasoning.level)) {
			next = {
				...next,
				reasoning: {
					...next.reasoning,
					level: allowedLevels.includes(ReasoningLevel.Medium) ? ReasoningLevel.Medium : allowedLevels[0],
				},
			};
		}
	}

	if (next.reasoning && cap?.reasoningCapabilities?.supportsSummaryStyle === false) {
		next = { ...next, reasoning: { ...next.reasoning } };
		if (next.reasoning) {
			delete next.reasoning.summaryStyle;
		}
	}

	// Some models disallow temperature whenever reasoning is enabled
	if (next.reasoning && cap?.reasoningCapabilities?.temperatureDisallowedWhenEnabled) {
		next = { ...next };
		delete next.temperature;
	}

	// If reasoning got removed and temperature is missing, ensure we still have a valid default
	if (!next.reasoning && next.temperature === undefined) {
		next.temperature = DefaultModelParams.temperature;
	}

	// --- Output sanitization ---
	if (cap?.outputCapabilities?.supportsVerbosity === false && next.outputParam?.verbosity) {
		const op = { ...(next.outputParam ?? {}) };
		delete op.verbosity;
		next.outputParam = op.format ? op : undefined;
	}

	const supportedFormats = getSupportedOutputFormats(cap);
	if (supportedFormats && next.outputParam?.format?.kind) {
		if (!supportedFormats.includes(next.outputParam.format.kind)) {
			const op = { ...(next.outputParam ?? {}) };
			delete op.format;
			next.outputParam = op.verbosity ? op : undefined;
		}
	}

	// --- Stop sequence sanitization ---
	const stopPolicy = getStopSequencesPolicy(cap);
	if (!stopPolicy.isSupported) {
		next.stopSequences = undefined;
	} else if (next.stopSequences?.length) {
		if (next.stopSequences.length > stopPolicy.maxSequences) {
			next.stopSequences = next.stopSequences.slice(0, stopPolicy.maxSequences);
		}
		if (stopPolicy.disallowedWithReasoning && !!next.reasoning) {
			next.stopSequences = undefined;
		}
	}

	// --- Cache-control sanitization ---
	if (!topLevelCacheCapabilities) {
		delete next.cacheControl;
	} else if (next.cacheControl) {
		const supportedKinds = topLevelCacheCapabilities.supportedKinds ?? [];
		const supportedTTLs = topLevelCacheCapabilities.supportedTTLs ?? [];

		if (supportedKinds.length > 0 && !supportedKinds.includes(next.cacheControl.kind)) {
			next.cacheControl = {
				...next.cacheControl,
				kind: supportedKinds[0],
			};
		}

		if (next.cacheControl.ttl && supportedTTLs.length > 0 && !supportedTTLs.includes(next.cacheControl.ttl)) {
			const cc = { ...next.cacheControl };
			delete cc.ttl;
			next.cacheControl = cc;
		}

		if (!topLevelCacheCapabilities.supportsKey && next.cacheControl.key) {
			const cc = { ...next.cacheControl };
			delete cc.key;
			next.cacheControl = cc;
		}
	}

	return next;
}
