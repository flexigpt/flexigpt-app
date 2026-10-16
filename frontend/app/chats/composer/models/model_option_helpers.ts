import type { ArtifactRef } from '@/spec/artifact';
import type { ModelParam } from '@/spec/inference';
import type { IncludePreviousMessages, UIModelOption } from '@/spec/model';
import { ReasoningType } from '@/spec/inference';

import { modelRefEqual } from '@/apis/model_management';

export function isHybridReasoningModel(model: UIModelOption): boolean {
	return model.reasoning?.type === ReasoningType.HybridWithTokens;
}

export function buildModelOptionsWithPreviousMessages(
	selectedModel: UIModelOption,
	includePreviousMessages: IncludePreviousMessages
): UIModelOption {
	const base: UIModelOption = {
		...selectedModel,
		includePreviousMessages,
	};

	return base;
}

function hasOwn(value: object, key: string): boolean {
	return Object.hasOwn(value, key);
}

function pickUniqueModelOption(options: UIModelOption[]): UIModelOption | undefined {
	return options.length === 1 ? options[0] : undefined;
}

function applyPersistedModelParam(
	base: UIModelOption,
	modelParam?: ModelParam,
	options?: { preserveName?: boolean }
): UIModelOption {
	if (!modelParam) {
		return base;
	}

	const next: UIModelOption = {
		...base,
		stream: modelParam.stream,
		maxPromptLength: modelParam.maxPromptLength,
		maxOutputLength: modelParam.maxOutputLength,
		timeout: modelParam.timeout,
	};

	if (options?.preserveName !== false) {
		next.name = modelParam.name;
	}

	if (hasOwn(modelParam, 'temperature')) {
		next.temperature = modelParam.temperature;
	} else {
		delete next.temperature;
	}

	if (hasOwn(modelParam, 'reasoning')) {
		next.reasoning = modelParam.reasoning;
	} else {
		delete next.reasoning;
	}

	if (hasOwn(modelParam, 'outputParam')) {
		next.outputParam = modelParam.outputParam;
	} else {
		delete next.outputParam;
	}

	if (hasOwn(modelParam, 'stopSequences')) {
		next.stopSequences = modelParam.stopSequences;
	} else {
		delete next.stopSequences;
	}

	if (hasOwn(modelParam, 'cacheControl')) {
		next.cacheControl = modelParam.cacheControl;
	} else {
		delete next.cacheControl;
	}

	if (hasOwn(modelParam, 'additionalParametersRawJSON')) {
		next.additionalParametersRawJSON = modelParam.additionalParametersRawJSON;
	} else {
		delete next.additionalParametersRawJSON;
	}

	return next;
}

export function findRestorableModelOption(
	allOptions: UIModelOption[],
	modelRef?: ArtifactRef,
	modelParam?: ModelParam
): UIModelOption | undefined {
	const modelName = modelParam?.name?.trim();

	if (modelRef) {
		const direct = allOptions.find(option => modelRefEqual(option.model, modelRef));
		if (direct) {
			return direct;
		}
	}

	if (modelName) {
		const byName = pickUniqueModelOption(allOptions.filter(option => option.name === modelName));
		if (byName) {
			return byName;
		}

		return pickUniqueModelOption(allOptions.filter(option => option.modelDisplayName === modelName));
	}

	return undefined;
}

export function resolveRestoredSelectedModel(
	allOptions: UIModelOption[],
	fallback: UIModelOption,
	modelRef?: ArtifactRef,
	modelParam?: ModelParam
): UIModelOption {
	const resolved = findRestorableModelOption(allOptions, modelRef, modelParam);

	if (resolved) {
		return applyPersistedModelParam(resolved, modelParam, {
			preserveName: true,
		});
	}

	return applyPersistedModelParam(fallback, modelParam, {
		preserveName: false,
	});
}
