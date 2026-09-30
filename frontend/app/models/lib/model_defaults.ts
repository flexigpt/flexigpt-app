import type { CacheControl, ModelParam, OutputParam, ReasoningParam } from '@/spec/inference';
import type { ModelDefaults, ModelOutputDefaults, ModelRequestPatch, UIModelOption } from '@/spec/model';
import { DefaultModelParams } from '@/spec/inference';

function mergeDefaults(
	providerDefaults: ModelDefaults | undefined,
	modelDefaults: ModelDefaults | undefined
): ModelDefaults {
	const provider = providerDefaults ?? {};
	const model = modelDefaults ?? {};

	return {
		...provider,
		...model,
		...(provider.reasoning || model.reasoning
			? {
					reasoning: {
						...provider.reasoning,
						...model.reasoning,
					},
				}
			: {}),
		...(provider.cacheControl || model.cacheControl
			? {
					cacheControl: {
						...provider.cacheControl,
						...model.cacheControl,
					} as CacheControl,
				}
			: {}),
		...(provider.output || model.output
			? {
					output: {
						...provider.output,
						...model.output,
						...(provider.output?.format || model.output?.format
							? {
									format: {
										...provider.output?.format,
										...model.output?.format,
									},
								}
							: {}),
					},
				}
			: {}),
	};
}

function outputParamFromDefaults(value: ModelOutputDefaults | undefined): OutputParam | undefined {
	if (!value) {
		return undefined;
	}

	const output: OutputParam = {};

	if (value.verbosity !== undefined) {
		output.verbosity = value.verbosity;
	}

	if (value.format?.kind) {
		output.format = {
			kind: value.format.kind,
			...(value.format.jsonSchema
				? {
						jsonSchemaParam: {
							name: value.format.jsonSchema.name ?? '',
							description: value.format.jsonSchema.description,
							schema: typeof value.format.jsonSchema.schema === 'boolean' ? undefined : value.format.jsonSchema.schema,
							strict: value.format.jsonSchema.strict,
						},
					}
				: {}),
		};
	}

	return output.format || output.verbosity ? output : undefined;
}

export function buildModelParamFromDefaults(
	providerModelID: string,
	providerDefaults: ModelDefaults | undefined,
	modelDefaults: ModelDefaults | undefined
): ModelParam {
	const defaults = mergeDefaults(providerDefaults, modelDefaults);
	const timeoutMS = typeof defaults.timeoutMS === 'number' ? defaults.timeoutMS : DefaultModelParams.timeout * 1000;
	const reasoning = defaults.reasoning as ReasoningParam | undefined;
	const configuredTemperature = typeof defaults.temperature === 'number' ? defaults.temperature : undefined;

	return {
		name: providerModelID,
		stream: typeof defaults.stream === 'boolean' ? defaults.stream : DefaultModelParams.stream,
		maxPromptLength:
			typeof defaults.maxPromptTokens === 'number' ? defaults.maxPromptTokens : DefaultModelParams.maxPromptLength,
		maxOutputLength:
			typeof defaults.maxOutputTokens === 'number' ? defaults.maxOutputTokens : DefaultModelParams.maxOutputLength,
		temperature: configuredTemperature ?? (reasoning === undefined ? DefaultModelParams.temperature : undefined),
		reasoning,
		systemPrompt: typeof defaults.systemPrompt === 'string' ? defaults.systemPrompt : DefaultModelParams.systemPrompt,
		timeout: Math.ceil(timeoutMS / 1000),
		cacheControl: defaults.cacheControl as CacheControl | undefined,
		outputParam: outputParamFromDefaults(defaults.output),
		stopSequences: defaults.stopSequences,
		additionalParametersRawJSON:
			defaults.adapterParameters && Object.keys(defaults.adapterParameters).length > 0
				? JSON.stringify(defaults.adapterParameters)
				: undefined,
	};
}

export function buildRequestPatch(option: UIModelOption): ModelRequestPatch | undefined {
	const patch = option.requestPatch;
	if (!patch || (!patch.defaults && (!patch.clear || patch.clear.length === 0))) {
		return undefined;
	}

	return structuredClone(patch);
}
