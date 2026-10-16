import type { ModelCapabilities } from '@/spec/model';
import {
	OutputFormatKind,
	OutputVerbosity,
	ReasoningLevel,
	ReasoningSummaryStyle,
	ReasoningType,
} from '@/spec/inference';

const orderedReasoningLevels: ReasoningLevel[] = [
	ReasoningLevel.None,
	ReasoningLevel.Minimal,
	ReasoningLevel.Low,
	ReasoningLevel.Medium,
	ReasoningLevel.High,
	ReasoningLevel.XHigh,
	ReasoningLevel.Max,
];

export function isReasoningLevel(value: unknown): value is ReasoningLevel {
	return typeof value === 'string' && (Object.values(ReasoningLevel) as string[]).includes(value);
}

export function isReasoningType(value: unknown): value is ReasoningType {
	return typeof value === 'string' && (Object.values(ReasoningType) as string[]).includes(value);
}

export function isOutputFormatKind(value: unknown): value is OutputFormatKind {
	return typeof value === 'string' && (Object.values(OutputFormatKind) as string[]).includes(value);
}

export function isOutputVerbosity(value: unknown): value is OutputVerbosity {
	return typeof value === 'string' && (Object.values(OutputVerbosity) as string[]).includes(value);
}

export function isReasoningSummaryStyle(value: unknown): value is ReasoningSummaryStyle {
	return typeof value === 'string' && (Object.values(ReasoningSummaryStyle) as string[]).includes(value);
}

export function getSupportedReasoningLevels(capabilities: ModelCapabilities | undefined): ReasoningLevel[] {
	const configured = capabilities?.reasoningCapabilities?.supportedReasoningLevels;

	if (!configured || configured.length === 0) {
		return [...orderedReasoningLevels];
	}

	const enabled = new Set(
		configured.filter(l => {
			return isReasoningLevel(l);
		})
	);
	const supported = orderedReasoningLevels.filter(level => enabled.has(level));

	return supported.length > 0 ? supported : [ReasoningLevel.Low, ReasoningLevel.Medium, ReasoningLevel.High];
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

	const supported = configured.filter(o => {
		return isOutputFormatKind(o);
	});
	return supported.length > 0 ? supported : undefined;
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
