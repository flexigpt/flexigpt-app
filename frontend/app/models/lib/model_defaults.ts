import type { CacheControl, ModelParam, OutputParam, ReasoningParam } from '@/spec/inference';
import type { ModelDefaults, ModelOutputDefaults, ModelRequestPatch, UIModelOption } from '@/spec/model';
import { DefaultModelParams } from '@/spec/inference';
import { ModelRequestClearField } from '@/spec/model';

import { jsonEqual } from '@/lib/jsonschema_utils';

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

function outputDefaultsFromParam(value: OutputParam | undefined): ModelOutputDefaults | undefined {
	if (!value) {
		return undefined;
	}

	return {
		...(value.verbosity !== undefined ? { verbosity: value.verbosity } : {}),
		...(value.format
			? {
					format: {
						kind: value.format.kind,
						...(value.format.jsonSchemaParam
							? {
									jsonSchema: {
										name: value.format.jsonSchemaParam.name,
										description: value.format.jsonSchemaParam.description,
										schema: value.format.jsonSchemaParam.schema,
										strict: value.format.jsonSchemaParam.strict,
									},
								}
							: {}),
					},
				}
			: {}),
	};
}

function parseAdapterParameters(value: string | undefined): Record<string, unknown> | undefined {
	if (!value?.trim()) {
		return undefined;
	}

	let parsed: unknown;
	try {
		parsed = JSON.parse(value);
	} catch {
		throw new Error('Additional adapter parameters must be valid JSON.');
	}

	if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
		throw new Error('Additional adapter parameters must be a JSON object.');
	}

	return parsed as Record<string, unknown>;
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
	const source = option.sourceModelParam;
	if (!source) {
		return undefined;
	}

	const defaults: ModelDefaults = {};
	const clear: ModelRequestClearField[] = [];

	if (option.stream !== source.stream) {
		defaults.stream = option.stream;
	}
	if (option.maxPromptLength !== source.maxPromptLength) {
		defaults.maxPromptTokens = option.maxPromptLength;
	}
	if (option.maxOutputLength !== source.maxOutputLength) {
		defaults.maxOutputTokens = option.maxOutputLength;
	}
	if (option.systemPrompt !== source.systemPrompt) {
		defaults.systemPrompt = option.systemPrompt;
	}
	if (option.timeout !== source.timeout) {
		defaults.timeoutMS = option.timeout * 1000;
	}

	if (option.temperature === undefined && source.temperature !== undefined) {
		clear.push(ModelRequestClearField.Temperature);
	} else if (option.temperature !== undefined && option.temperature !== source.temperature) {
		defaults.temperature = option.temperature;
	}

	if (option.reasoning === undefined && source.reasoning !== undefined) {
		clear.push(ModelRequestClearField.Reasoning);
	} else if (option.reasoning !== undefined && !jsonEqual(option.reasoning, source.reasoning)) {
		defaults.reasoning = option.reasoning;
	}

	if (option.cacheControl === undefined && source.cacheControl !== undefined) {
		clear.push(ModelRequestClearField.CacheControl);
	} else if (option.cacheControl !== undefined && !jsonEqual(option.cacheControl, source.cacheControl)) {
		defaults.cacheControl = option.cacheControl;
	}

	if (option.outputParam === undefined && source.outputParam !== undefined) {
		clear.push(ModelRequestClearField.Output);
	} else if (option.outputParam !== undefined && !jsonEqual(option.outputParam, source.outputParam)) {
		defaults.output = outputDefaultsFromParam(option.outputParam);
	}

	if (option.stopSequences === undefined && source.stopSequences !== undefined) {
		clear.push(ModelRequestClearField.StopSequences);
	} else if (option.stopSequences !== undefined && !jsonEqual(option.stopSequences, source.stopSequences)) {
		defaults.stopSequences = option.stopSequences;
	}

	if (option.additionalParametersRawJSON === undefined && source.additionalParametersRawJSON !== undefined) {
		clear.push(ModelRequestClearField.AdapterParameters);
	} else if (option.additionalParametersRawJSON !== source.additionalParametersRawJSON) {
		const adapterParameters = parseAdapterParameters(option.additionalParametersRawJSON);
		if (adapterParameters) {
			defaults.adapterParameters = adapterParameters;
		}
	}

	if (Object.keys(defaults).length === 0 && clear.length === 0) {
		return undefined;
	}

	return {
		...(Object.keys(defaults).length > 0 ? { defaults } : {}),
		...(clear.length > 0 ? { clear } : {}),
	};
}
