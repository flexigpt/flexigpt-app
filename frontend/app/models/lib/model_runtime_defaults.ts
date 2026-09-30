import type {
	CacheControlKind,
	JSONSchemaParam,
	OutputFormatKind,
	OutputVerbosity,
	ProviderSDKType,
	ReasoningLevel,
	ReasoningSummaryStyle,
	ReasoningType,
} from '@/spec/inference';
import type { ModelCapabilities, ModelDefaults } from '@/spec/model';
import {
	OutputFormatKind as OutputFormatKindValue,
	ReasoningLevel as ReasoningLevelValue,
	ReasoningSummaryStyle as ReasoningSummaryStyleValue,
	ReasoningType as ReasoningTypeValue,
} from '@/spec/inference';

import { formatJSON, MAX_JSON_SCHEMA_INPUT_CHARS, tryParseJSONObject } from '@/lib/jsonschema_utils';

import type { CacheControlTTLSelection } from '@/models/lib/cache_control';
import {
	buildCacheControlFromForm,
	CACHE_CONTROL_TTL_PROVIDER_DEFAULT,
	getInitialCacheControlKind,
	getInitialCacheControlTTLSelection,
	resolveSupportedCacheControlKinds,
	resolveSupportedCacheControlTTLs,
} from '@/models/lib/cache_control';
import {
	getStopSequencesPolicy,
	getSupportedOutputFormats,
	getTopLevelCacheControlCapabilities,
	isOutputFormatKind,
	isOutputVerbosity,
	isReasoningLevel,
	isReasoningSummaryStyle,
	isReasoningType,
	supportsOutputVerbosity,
} from '@/models/lib/capabilities';

export const OUTPUT_FORMAT_NONE = '__none__' as const;
export const OUTPUT_VERBOSITY_NONE = '__none__' as const;
export const OPTIONAL_BOOLEAN_UNSET = '__unset__' as const;
export const DEFAULT_REASONING_TOKENS = 1024;

export type OutputFormatSelection = OutputFormatKind | typeof OUTPUT_FORMAT_NONE;
export type OutputVerbositySelection = OutputVerbosity | typeof OUTPUT_VERBOSITY_NONE;
export type StrictSelection = typeof OPTIONAL_BOOLEAN_UNSET | 'true' | 'false';

export interface ModelRuntimeDefaultsForm {
	stream: boolean;
	maxPromptTokens: string;
	maxOutputTokens: string;
	temperature: string;
	reasoningEnabled: boolean;
	reasoningType: ReasoningType;
	reasoningLevel: ReasoningLevel;
	reasoningTokens: string;
	reasoningSummaryStyle: ReasoningSummaryStyle;
	systemPrompt: string;
	timeoutSeconds: string;
	cacheControlEnabled: boolean;
	cacheControlKind: CacheControlKind | '';
	cacheControlTTL: CacheControlTTLSelection;
	cacheControlKey: string;
	stopSequencesRaw: string;
	outputFormatKind: OutputFormatSelection;
	outputVerbosity: OutputVerbositySelection;
	outputJSONSchemaName: string;
	outputJSONSchemaDescription: string;
	outputJSONSchemaRaw: string;
	outputJSONSchemaStrict: StrictSelection;
}

export type ModelRuntimeDefaultsErrors = Partial<Record<keyof ModelRuntimeDefaultsForm, string>>;

export function parseOptionalPositiveInteger(value: string): number | undefined {
	const trimmed = value.trim();
	if (!trimmed) {
		return undefined;
	}

	const numberValue = Number(trimmed);
	if (!Number.isFinite(numberValue) || !Number.isInteger(numberValue) || numberValue <= 0) {
		return Number.NaN;
	}

	return numberValue;
}

function parseTemperature(value: string): number | undefined {
	const trimmed = value.trim();
	if (!trimmed) {
		return undefined;
	}

	const numberValue = Number(trimmed);
	if (!Number.isFinite(numberValue) || numberValue < 0 || numberValue > 1) {
		return Number.NaN;
	}

	return numberValue;
}

export function parseStopSequences(raw: string): string[] {
	const lines = raw
		.split(/\r?\n/g)
		.map(s => s.trim())
		.filter(Boolean);

	const out: string[] = [];
	if (lines.length === 0) {
		return out;
	}

	const seen = new Set<string>();

	for (const s of lines) {
		if (seen.has(s)) {
			continue;
		}
		seen.add(s);
		out.push(s);
	}
	return out;
}

export function parseOptionalStopSequences(raw: string): string[] | undefined {
	const values = parseStopSequences(raw);
	return values.length > 0 ? values : undefined;
}

function strictToSelection(value: boolean | undefined): StrictSelection {
	if (value === true) {
		return 'true';
	}
	if (value === false) {
		return 'false';
	}
	return OPTIONAL_BOOLEAN_UNSET;
}

function selectionToStrict(value: StrictSelection): boolean | undefined {
	if (value === 'true') {
		return true;
	}
	if (value === 'false') {
		return false;
	}
	return undefined;
}

function formatOutputSchema(schema: unknown): string {
	return formatJSON(schema);
}

export function createModelRuntimeDefaultsForm(
	defaults: ModelDefaults | undefined,
	providerSDKType: ProviderSDKType,
	capabilities: ModelCapabilities | undefined,
	isNewModel: boolean
): ModelRuntimeDefaultsForm {
	const cacheCapabilities = getTopLevelCacheControlCapabilities(providerSDKType, capabilities);
	const supportedCacheKinds = resolveSupportedCacheControlKinds(
		cacheCapabilities?.supportedKinds,
		defaults?.cacheControl
	);
	const supportedCacheTTLs = resolveSupportedCacheControlTTLs(cacheCapabilities?.supportedTTLs, defaults?.cacheControl);
	const schema = defaults?.output?.format?.jsonSchema;
	const reasoningType = isReasoningType(defaults?.reasoning?.type)
		? defaults.reasoning.type
		: ReasoningTypeValue.SingleWithLevels;
	const reasoningLevel = isReasoningLevel(defaults?.reasoning?.level)
		? defaults.reasoning.level
		: ReasoningLevelValue.Medium;

	return {
		stream: defaults?.stream ?? true,
		maxPromptTokens:
			defaults?.maxPromptTokens !== undefined ? String(defaults.maxPromptTokens) : isNewModel ? '2048' : '',
		maxOutputTokens:
			defaults?.maxOutputTokens !== undefined ? String(defaults.maxOutputTokens) : isNewModel ? '1024' : '',
		temperature: defaults?.temperature !== undefined ? String(defaults.temperature) : '',
		reasoningEnabled: Boolean(defaults?.reasoning),
		reasoningType,
		reasoningLevel,
		reasoningTokens: String(defaults?.reasoning?.tokens ?? DEFAULT_REASONING_TOKENS),
		reasoningSummaryStyle: isReasoningSummaryStyle(defaults?.reasoning?.summaryStyle)
			? defaults.reasoning.summaryStyle
			: ReasoningSummaryStyleValue.Auto,
		systemPrompt: defaults?.systemPrompt ?? '',
		timeoutSeconds:
			defaults?.timeoutMS !== undefined ? String(Math.ceil(defaults.timeoutMS / 1000)) : isNewModel ? '300' : '',
		cacheControlEnabled: Boolean(defaults?.cacheControl),
		cacheControlKind: getInitialCacheControlKind(defaults?.cacheControl, supportedCacheKinds),
		cacheControlTTL: getInitialCacheControlTTLSelection(defaults?.cacheControl, supportedCacheTTLs),
		cacheControlKey: defaults?.cacheControl?.key ?? '',
		stopSequencesRaw: defaults?.stopSequences?.join('\n') ?? '',
		outputFormatKind: isOutputFormatKind(defaults?.output?.format?.kind)
			? defaults.output.format.kind
			: OUTPUT_FORMAT_NONE,
		outputVerbosity: isOutputVerbosity(defaults?.output?.verbosity) ? defaults.output.verbosity : OUTPUT_VERBOSITY_NONE,
		outputJSONSchemaName: schema?.name ?? '',
		outputJSONSchemaDescription: schema?.description ?? '',
		outputJSONSchemaRaw: formatOutputSchema(schema?.schema),
		outputJSONSchemaStrict: strictToSelection(schema?.strict),
	};
}

export function createFallbackModelRuntimeDefaultsForm(): ModelRuntimeDefaultsForm {
	return {
		stream: true,
		maxPromptTokens: '2048',
		maxOutputTokens: '1024',
		temperature: '',
		reasoningEnabled: false,
		reasoningType: ReasoningTypeValue.SingleWithLevels,
		reasoningLevel: ReasoningLevelValue.Medium,
		reasoningTokens: String(DEFAULT_REASONING_TOKENS),
		reasoningSummaryStyle: ReasoningSummaryStyleValue.Auto,
		systemPrompt: '',
		timeoutSeconds: '300',
		cacheControlEnabled: false,
		cacheControlKind: '',
		cacheControlTTL: CACHE_CONTROL_TTL_PROVIDER_DEFAULT,
		cacheControlKey: '',
		stopSequencesRaw: '',
		outputFormatKind: OUTPUT_FORMAT_NONE,
		outputVerbosity: OUTPUT_VERBOSITY_NONE,
		outputJSONSchemaName: '',
		outputJSONSchemaDescription: '',
		outputJSONSchemaRaw: '',
		outputJSONSchemaStrict: OPTIONAL_BOOLEAN_UNSET,
	};
}

export function validateModelRuntimeDefaultsForm(
	form: ModelRuntimeDefaultsForm,
	providerSDKType: ProviderSDKType,
	capabilities: ModelCapabilities | undefined,
	requireTemperatureOrReasoning: boolean
): ModelRuntimeDefaultsErrors {
	const errors: ModelRuntimeDefaultsErrors = {};
	const maxPromptTokens = parseOptionalPositiveInteger(form.maxPromptTokens);
	const maxOutputTokens = parseOptionalPositiveInteger(form.maxOutputTokens);
	const timeoutSeconds = parseOptionalPositiveInteger(form.timeoutSeconds);
	const temperature = parseTemperature(form.temperature);
	const stopPolicy = getStopSequencesPolicy(capabilities);
	const supportedFormats = getSupportedOutputFormats(capabilities);

	if (Number.isNaN(maxPromptTokens)) {
		errors.maxPromptTokens = 'Max prompt tokens must be a positive whole number.';
	}
	if (Number.isNaN(maxOutputTokens)) {
		errors.maxOutputTokens = 'Max output tokens must be a positive whole number.';
	}
	if (Number.isNaN(timeoutSeconds)) {
		errors.timeoutSeconds = 'Timeout must be a positive whole number of seconds.';
	}
	if (Number.isNaN(temperature)) {
		errors.temperature = 'Temperature must be between 0 and 1.';
	}

	if (requireTemperatureOrReasoning && !form.reasoningEnabled && form.temperature.trim() === '') {
		errors.temperature = 'Provide a temperature or enable reasoning.';
	}

	if (
		form.reasoningEnabled &&
		capabilities?.reasoningCapabilities?.temperatureDisallowedWhenEnabled &&
		form.temperature.trim()
	) {
		errors.temperature = 'Temperature is not supported while reasoning is enabled for this model.';
	}

	if (form.reasoningEnabled && form.reasoningType === ReasoningTypeValue.HybridWithTokens) {
		const reasoningTokens = parseOptionalPositiveInteger(form.reasoningTokens);
		if (Number.isNaN(reasoningTokens) || reasoningTokens === undefined || reasoningTokens < DEFAULT_REASONING_TOKENS) {
			errors.reasoningTokens = `Reasoning tokens must be a whole number of at least ${DEFAULT_REASONING_TOKENS}.`;
		}
	}

	if (
		form.outputFormatKind !== OUTPUT_FORMAT_NONE &&
		supportedFormats &&
		!supportedFormats.includes(form.outputFormatKind)
	) {
		errors.outputFormatKind = 'This output format is not supported by the selected model/provider.';
	}

	if (form.outputFormatKind === OutputFormatKindValue.JSONSchema) {
		if (!form.outputJSONSchemaName.trim()) {
			errors.outputJSONSchemaName = 'JSON Schema output requires a schema name.';
		}

		const rawSchema = form.outputJSONSchemaRaw.trim();
		if (!rawSchema) {
			errors.outputJSONSchemaRaw = 'JSON Schema output requires a schema body.';
		} else {
			const parsed = tryParseJSONObject(rawSchema, 'JSON schema body', MAX_JSON_SCHEMA_INPUT_CHARS);
			if (!parsed.ok) {
				errors.outputJSONSchemaRaw = parsed.error;
			}
		}
	}

	if (!supportsOutputVerbosity(capabilities) && form.outputVerbosity !== OUTPUT_VERBOSITY_NONE) {
		errors.outputVerbosity = 'Output verbosity is not supported by the selected model/provider.';
	}

	const stopSequences = parseStopSequences(form.stopSequencesRaw);
	if (stopPolicy.disallowedWithReasoning && form.reasoningEnabled && stopSequences.length > 0) {
		errors.stopSequencesRaw = 'Stop sequences are not supported while reasoning is enabled.';
	} else if (!stopPolicy.isSupported && stopSequences.length > 0) {
		errors.stopSequencesRaw = 'Stop sequences are not supported by the selected model/provider.';
	} else if (stopSequences.length > stopPolicy.maxSequences) {
		errors.stopSequencesRaw = `At most ${stopPolicy.maxSequences} stop sequences are allowed.`;
	} else if (stopSequences.some(value => value.length > 256)) {
		errors.stopSequencesRaw = 'Each stop sequence must be at most 256 characters.';
	}

	const cacheCapabilities = getTopLevelCacheControlCapabilities(providerSDKType, capabilities);
	const supportedCacheKinds = resolveSupportedCacheControlKinds(cacheCapabilities?.supportedKinds);
	if (form.cacheControlEnabled && supportedCacheKinds.length === 0) {
		errors.cacheControlKind = 'Manual cache control is not supported by the selected model/provider.';
	}

	return errors;
}

export function buildModelDefaultsFromRuntimeForm(
	form: ModelRuntimeDefaultsForm,
	providerSDKType: ProviderSDKType,
	capabilities: ModelCapabilities | undefined
): ModelDefaults {
	const defaults: ModelDefaults = {
		stream: form.stream,
		systemPrompt: form.systemPrompt,
	};
	const maxPromptTokens = parseOptionalPositiveInteger(form.maxPromptTokens);
	const maxOutputTokens = parseOptionalPositiveInteger(form.maxOutputTokens);
	const timeoutSeconds = parseOptionalPositiveInteger(form.timeoutSeconds);
	const temperature = parseTemperature(form.temperature);
	const stopPolicy = getStopSequencesPolicy(capabilities);
	const cacheCapabilities = getTopLevelCacheControlCapabilities(providerSDKType, capabilities);
	const supportedCacheKinds = resolveSupportedCacheControlKinds(cacheCapabilities?.supportedKinds);

	if (maxPromptTokens !== undefined && !Number.isNaN(maxPromptTokens)) {
		defaults.maxPromptTokens = maxPromptTokens;
	}
	if (maxOutputTokens !== undefined && !Number.isNaN(maxOutputTokens)) {
		defaults.maxOutputTokens = maxOutputTokens;
	}
	if (timeoutSeconds !== undefined && !Number.isNaN(timeoutSeconds)) {
		defaults.timeoutMS = timeoutSeconds * 1000;
	}
	if (temperature !== undefined && !Number.isNaN(temperature)) {
		defaults.temperature = temperature;
	}

	if (form.reasoningEnabled) {
		defaults.reasoning = {
			type: form.reasoningType,
			level: form.reasoningLevel,
			tokens:
				form.reasoningType === ReasoningTypeValue.HybridWithTokens
					? Number(form.reasoningTokens)
					: DEFAULT_REASONING_TOKENS,
			summaryStyle: form.reasoningSummaryStyle,
		};
	}

	const cacheControl = buildCacheControlFromForm({
		enabled: form.cacheControlEnabled,
		kind: form.cacheControlKind,
		supportedKinds: supportedCacheKinds,
		ttlSelection: form.cacheControlTTL,
		key: form.cacheControlKey,
		supportsTTL: cacheCapabilities?.supportsTTL ?? true,
		supportsKey: cacheCapabilities?.supportsKey === true,
	});
	if (cacheControl) {
		defaults.cacheControl = cacheControl;
	}

	const output: NonNullable<ModelDefaults['output']> = {};
	if (form.outputVerbosity !== OUTPUT_VERBOSITY_NONE) {
		output.verbosity = form.outputVerbosity;
	}
	if (form.outputFormatKind === OutputFormatKindValue.Text) {
		output.format = {
			kind: OutputFormatKindValue.Text,
		};
	}
	if (form.outputFormatKind === OutputFormatKindValue.JSONSchema) {
		const parsedSchema = tryParseJSONObject(form.outputJSONSchemaRaw, 'JSON schema body', MAX_JSON_SCHEMA_INPUT_CHARS);
		const jsonSchema: JSONSchemaParam = {
			name: form.outputJSONSchemaName.trim(),
			...(form.outputJSONSchemaDescription.trim()
				? {
						description: form.outputJSONSchemaDescription.trim(),
					}
				: {}),
			...(parsedSchema.ok ? { schema: parsedSchema.value } : {}),
			...(selectionToStrict(form.outputJSONSchemaStrict) !== undefined
				? {
						strict: selectionToStrict(form.outputJSONSchemaStrict),
					}
				: {}),
		};

		output.format = {
			kind: OutputFormatKindValue.JSONSchema,
			jsonSchema,
		};
	}
	if (output.verbosity || output.format) {
		defaults.output = output;
	}

	const stopSequences = parseStopSequences(form.stopSequencesRaw);
	if (
		stopPolicy.isSupported &&
		!(stopPolicy.disallowedWithReasoning && form.reasoningEnabled) &&
		stopSequences.length > 0
	) {
		defaults.stopSequences = stopSequences;
	}

	return defaults;
}
