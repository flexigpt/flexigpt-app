/**
 * @public
 */
export interface ToolCapabilities {
	supportedToolTypes: string[];
	supportedToolPolicyModes: string[];
	supportsParallelToolCalls: boolean;
	maxForcedTools: number;
}

/**
 * @public
 */
export interface OutputCapabilities {
	supportedOutputFormats: string[];
	supportsVerbosity: boolean;
}

/**
 * @public
 */
export interface StopSequenceCapabilities {
	isSupported: boolean;
	disallowedWithReasoning: boolean;
	maxSequences: number;
}

/**
 * @public
 */
export interface ReasoningCapabilities {
	supportedReasoningTypes: string[];
	supportedReasoningLevels: string[];
	supportsSummaryStyle: boolean;
	supportsEncryptedReasoningInput: boolean;
	temperatureDisallowedWhenEnabled: boolean;
}

/**
 * @public
 */
export interface ModelCapabilities {
	modalitiesIn: string[];
	modalitiesOut: string[];
	reasoningCapabilities?: ReasoningCapabilities;
	stopSequenceCapabilities?: StopSequenceCapabilities;
	outputCapabilities?: OutputCapabilities;
	toolCapabilities?: ToolCapabilities;
}
