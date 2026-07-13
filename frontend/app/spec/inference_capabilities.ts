export interface ToolCapabilities {
	supportedToolTypes: string[];
	supportedToolPolicyModes: string[];
	supportsParallelToolCalls: boolean;
	maxForcedTools: number;
}

export interface OutputCapabilities {
	supportedOutputFormats: string[];
	supportsVerbosity: boolean;
}

export interface StopSequenceCapabilities {
	isSupported: boolean;
	disallowedWithReasoning: boolean;
	maxSequences: number;
}

export interface ReasoningCapabilities {
	supportedReasoningTypes: string[];
	supportedReasoningLevels: string[];
	supportsSummaryStyle: boolean;
	supportsEncryptedReasoningInput: boolean;
	temperatureDisallowedWhenEnabled: boolean;
}

export interface ModelCapabilities {
	modalitiesIn: string[];
	modalitiesOut: string[];
	reasoningCapabilities?: ReasoningCapabilities;
	stopSequenceCapabilities?: StopSequenceCapabilities;
	outputCapabilities?: OutputCapabilities;
	toolCapabilities?: ToolCapabilities;
}
