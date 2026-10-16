import type { OutputVerbosity, ReasoningLevel } from '@/spec/inference';
import type { ModelDefaults, ModelRequestPatch } from '@/spec/model';
import { ReasoningType } from '@/spec/inference';
import { SessionModelDefaultField } from '@/spec/session';

import { omitManyKeys } from '@/lib/obj_utils';

const CLEAR_FIELD_DEFAULT_KEY: Record<SessionModelDefaultField, keyof ModelDefaults> = {
	[SessionModelDefaultField.AdapterParameters]: 'adapterParameters',
	[SessionModelDefaultField.CacheControl]: 'cacheControl',
	[SessionModelDefaultField.MaxOutputTokens]: 'maxOutputTokens',
	[SessionModelDefaultField.MaxPromptTokens]: 'maxPromptTokens',
	[SessionModelDefaultField.Output]: 'output',
	[SessionModelDefaultField.Reasoning]: 'reasoning',
	[SessionModelDefaultField.StopSequences]: 'stopSequences',
	[SessionModelDefaultField.Stream]: 'stream',
	[SessionModelDefaultField.SystemPrompt]: 'systemPrompt',
	[SessionModelDefaultField.Temperature]: 'temperature',
	[SessionModelDefaultField.TimeoutMS]: 'timeoutMS',
};

function clonePatch(patch?: ModelRequestPatch): ModelRequestPatch {
	return patch ? structuredClone(patch) : {};
}

function compactDefaults(defaults: ModelDefaults) {
	if (defaults.reasoning && Object.keys(defaults.reasoning).length === 0) {
		delete defaults.reasoning;
	}
	if (defaults.output && Object.keys(defaults.output).length === 0) {
		delete defaults.output;
	}
}

function finishPatch(patch: ModelRequestPatch): ModelRequestPatch | undefined {
	if (patch.defaults) {
		compactDefaults(patch.defaults);
		if (Object.keys(patch.defaults).length === 0) {
			delete patch.defaults;
		}
	}

	if (patch.clear?.length === 0) {
		delete patch.clear;
	}

	return patch.defaults || patch.clear?.length ? patch : undefined;
}

export function editRequestDefaults(
	patch: ModelRequestPatch | undefined,
	edit: (defaults: ModelDefaults) => void
): ModelRequestPatch | undefined {
	const next = clonePatch(patch);
	const defaults = structuredClone(next.defaults ?? {});

	edit(defaults);

	next.defaults = defaults;
	return finishPatch(next);
}

export function removeRequestClear(
	patch: ModelRequestPatch | undefined,
	field: SessionModelDefaultField
): ModelRequestPatch | undefined {
	const next = clonePatch(patch);
	next.clear = next.clear?.filter(value => value !== field);
	return finishPatch(next);
}

export function inheritRequestDefault(
	patch: ModelRequestPatch | undefined,
	field: SessionModelDefaultField
): ModelRequestPatch | undefined {
	const next = clonePatch(patch);
	const key = CLEAR_FIELD_DEFAULT_KEY[field];

	if (next.defaults) {
		next.defaults = omitManyKeys(next.defaults, [key]);
	}
	next.clear = next.clear?.filter(value => value !== field);

	return finishPatch(next);
}

export function clearRequestDefault(
	patch: ModelRequestPatch | undefined,
	field: SessionModelDefaultField
): ModelRequestPatch | undefined {
	const next = clonePatch(patch);
	const key = CLEAR_FIELD_DEFAULT_KEY[field];

	if (next.defaults) {
		next.defaults = omitManyKeys(next.defaults, [key]);
	}
	if (!next.clear?.includes(field)) {
		next.clear = [...(next.clear ?? []), field];
	}

	return finishPatch(next);
}

export function setTemperaturePreference(
	patch: ModelRequestPatch | undefined,
	temperature?: number
): ModelRequestPatch | undefined {
	if (temperature === undefined) {
		return inheritRequestDefault(patch, SessionModelDefaultField.Temperature);
	}

	const next = editRequestDefaults(patch, defaults => {
		defaults.temperature = temperature;
	});
	return removeRequestClear(next, SessionModelDefaultField.Temperature);
}

export function setReasoningLevelPreference(
	patch: ModelRequestPatch | undefined,
	level?: ReasoningLevel
): ModelRequestPatch | undefined {
	const next = editRequestDefaults(patch, defaults => {
		const reasoning = { ...defaults.reasoning };

		if (level === undefined) {
			delete reasoning.type;
			delete reasoning.level;
		} else {
			reasoning.type = ReasoningType.SingleWithLevels;
			reasoning.level = level;
		}

		if (Object.keys(reasoning).length === 0) {
			delete defaults.reasoning;
		} else {
			defaults.reasoning = reasoning;
		}
	});

	return removeRequestClear(next, SessionModelDefaultField.Reasoning);
}

export function setHybridReasoningPreference(
	patch: ModelRequestPatch | undefined,
	enabled?: boolean
): ModelRequestPatch | undefined {
	if (enabled === undefined) {
		return inheritRequestDefault(patch, SessionModelDefaultField.Reasoning);
	}

	if (!enabled) {
		return clearRequestDefault(patch, SessionModelDefaultField.Reasoning);
	}

	const next = editRequestDefaults(patch, defaults => {
		defaults.reasoning = {
			...defaults.reasoning,
			type: ReasoningType.HybridWithTokens,
		};
	});

	return removeRequestClear(next, SessionModelDefaultField.Reasoning);
}

export function setHybridTokensPreference(
	patch: ModelRequestPatch | undefined,
	tokens?: number
): ModelRequestPatch | undefined {
	const next = editRequestDefaults(patch, defaults => {
		const reasoning = { ...defaults.reasoning };

		if (tokens === undefined) {
			delete reasoning.tokens;
		} else {
			reasoning.tokens = tokens;
		}

		if (Object.keys(reasoning).length === 0) {
			delete defaults.reasoning;
		} else {
			defaults.reasoning = reasoning;
		}
	});

	return removeRequestClear(next, SessionModelDefaultField.Reasoning);
}

export function setOutputVerbosityPreference(
	patch: ModelRequestPatch | undefined,
	verbosity?: OutputVerbosity
): ModelRequestPatch | undefined {
	const next = editRequestDefaults(patch, defaults => {
		const output = { ...defaults.output };

		if (verbosity === undefined) {
			delete output.verbosity;
		} else {
			output.verbosity = verbosity;
		}

		if (Object.keys(output).length === 0) {
			delete defaults.output;
		} else {
			defaults.output = output;
		}
	});

	return removeRequestClear(next, SessionModelDefaultField.Output);
}
