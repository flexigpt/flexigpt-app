import { OutputFormatKind, OutputVerbosity, ReasoningLevel } from '@/spec/inference';

const OUTPUT_FORMAT_DISPLAY_NAMES: Record<OutputFormatKind, string> = {
	[OutputFormatKind.Text]: 'Text',
	[OutputFormatKind.JSONSchema]: 'JSON Schema',
};

const OUTPUT_VERBOSITY_DISPLAY_NAMES: Record<OutputVerbosity, string> = {
	[OutputVerbosity.Low]: 'Low',
	[OutputVerbosity.Medium]: 'Medium',
	[OutputVerbosity.High]: 'High',
	[OutputVerbosity.XHigh]: 'XHigh',
	[OutputVerbosity.Max]: 'Max',
};

const REASONING_LEVEL_DISPLAY_NAMES: Record<ReasoningLevel, string> = {
	[ReasoningLevel.None]: 'None',
	[ReasoningLevel.Minimal]: 'Minimal',
	[ReasoningLevel.Low]: 'Low',
	[ReasoningLevel.Medium]: 'Medium',
	[ReasoningLevel.High]: 'High',
	[ReasoningLevel.XHigh]: 'XHigh',
	[ReasoningLevel.Max]: 'Max',
};

export const OUTPUT_VERBOSITY_VALUES: readonly OutputVerbosity[] = [
	OutputVerbosity.Low,
	OutputVerbosity.Medium,
	OutputVerbosity.High,
	OutputVerbosity.XHigh,
	OutputVerbosity.Max,
];

export const COMPOSER_DEFAULT_REASONING_LEVELS: readonly ReasoningLevel[] = [
	ReasoningLevel.None,
	ReasoningLevel.Minimal,
	ReasoningLevel.Low,
	ReasoningLevel.Medium,
	ReasoningLevel.High,
];

export function getOutputFormatDisplayName(value?: OutputFormatKind): string {
	if (value === undefined) {
		return 'Default';
	}

	return OUTPUT_FORMAT_DISPLAY_NAMES[value] ?? 'Unknown';
}

export function getOutputVerbosityDisplayName(value?: OutputVerbosity): string {
	if (value === undefined) {
		return 'Default';
	}

	return OUTPUT_VERBOSITY_DISPLAY_NAMES[value] ?? 'Unknown';
}

export function getReasoningLevelDisplayName(value?: ReasoningLevel): string {
	if (value === undefined) {
		return 'Default';
	}

	return REASONING_LEVEL_DISPLAY_NAMES[value] ?? 'Unknown';
}
