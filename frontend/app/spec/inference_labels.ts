import { CacheControlKind, CacheControlTTL, OutputFormatKind, OutputVerbosity, ReasoningLevel } from '@/spec/inference';

const OUTPUT_FORMAT_DISPLAY_NAMES: Record<OutputFormatKind, string> = {
	[OutputFormatKind.Text]: 'Text',
	[OutputFormatKind.JSONSchema]: 'JSON Schema',
};

export function getOutputFormatDisplayName(value?: OutputFormatKind): string {
	if (value === undefined) {
		return 'Default';
	}

	return OUTPUT_FORMAT_DISPLAY_NAMES[value] ?? 'Unknown';
}

const OUTPUT_VERBOSITY_DISPLAY_NAMES: Record<OutputVerbosity, string> = {
	[OutputVerbosity.Low]: 'Low',
	[OutputVerbosity.Medium]: 'Medium',
	[OutputVerbosity.High]: 'High',
	[OutputVerbosity.XHigh]: 'XHigh',
	[OutputVerbosity.Max]: 'Max',
};

export function getOutputVerbosityDisplayName(value?: OutputVerbosity): string {
	if (value === undefined) {
		return 'Default';
	}

	return OUTPUT_VERBOSITY_DISPLAY_NAMES[value] ?? 'Unknown';
}

const REASONING_LEVEL_DISPLAY_NAMES: Record<ReasoningLevel, string> = {
	[ReasoningLevel.None]: 'None',
	[ReasoningLevel.Minimal]: 'Minimal',
	[ReasoningLevel.Low]: 'Low',
	[ReasoningLevel.Medium]: 'Medium',
	[ReasoningLevel.High]: 'High',
	[ReasoningLevel.XHigh]: 'XHigh',
	[ReasoningLevel.Max]: 'Max',
};

export function getReasoningLevelDisplayName(value?: ReasoningLevel): string {
	if (value === undefined) {
		return 'Default';
	}

	return REASONING_LEVEL_DISPLAY_NAMES[value] ?? 'Unknown';
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
