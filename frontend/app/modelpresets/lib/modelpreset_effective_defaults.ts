import { DefaultModelParams, type ModelParam } from '@/spec/inference';
import type { ModelPreset } from '@/spec/modelpreset';

/**
 * Canonical "effective runtime defaults" for a model preset.
 *
 * This is the single place that should decide how missing model-preset fields
 * fall back to app/runtime defaults.
 */
export function buildEffectiveModelParamFromModelPreset(modelPreset: ModelPreset): ModelParam {
	const next: ModelParam = {
		name: modelPreset.name,
		stream: modelPreset.stream ?? DefaultModelParams.stream,
		maxPromptLength: modelPreset.maxPromptLength ?? DefaultModelParams.maxPromptLength,
		maxOutputLength: modelPreset.maxOutputLength ?? DefaultModelParams.maxOutputLength,
		systemPrompt: modelPreset.systemPrompt ?? DefaultModelParams.systemPrompt,
		timeout: modelPreset.timeout ?? DefaultModelParams.timeout,

		temperature: modelPreset.temperature,
		reasoning: modelPreset.reasoning,

		outputParam: modelPreset.outputParam ?? DefaultModelParams.outputParam,
		stopSequences: modelPreset.stopSequences ?? DefaultModelParams.stopSequences,
		additionalParametersRawJSON: modelPreset.additionalParametersRawJSON,
	};

	// Preserve current runtime behavior:
	// if neither reasoning nor temperature is specified by the preset,
	// fall back to the default temperature.
	if (next.temperature === undefined && next.reasoning === undefined) {
		next.temperature = DefaultModelParams.temperature;
	}

	return next;
}
