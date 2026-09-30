import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { ArtifactRef } from '@/spec/artifact';
import type { RestorableConversationContext } from '@/spec/conversation';
import type { ModelParam, OutputVerbosity, ReasoningLevel } from '@/spec/inference';
import type { IncludePreviousMessages, UIModelOption } from '@/spec/model';
import { ReasoningType } from '@/spec/inference';
import { DefaultUIModelOption } from '@/spec/model';

import type { ModelCatalogUnavailableReason } from '@/apis/model_management';
import { modelManagementAPI } from '@/apis/baseapi';
import { isModelCatalogUnavailableError } from '@/apis/model_management';

import {
	getSupportedReasoningLevels,
	sanitizeUIModelOptionByCapabilities,
	supportsOutputVerbosity,
} from '@/models/lib/capabilities';
import { modelRefEqual } from '@/models/lib/document';

function hasOwn(value: object, key: string): boolean {
	return Object.hasOwn(value, key);
}

function isHybridReasoningModel(model: UIModelOption): boolean {
	return model.reasoning?.type === ReasoningType.HybridWithTokens;
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
		return sanitizeUIModelOptionByCapabilities(base);
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

	return sanitizeUIModelOptionByCapabilities(next);
}

function findRestorableModelOption(
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

function resolveRestoredSelectedModel(
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

function buildChatOptions(
	selectedModel: UIModelOption,
	includePreviousMessages: IncludePreviousMessages,
	isHybridReasoningEnabled: boolean
): UIModelOption {
	const base: UIModelOption = {
		...selectedModel,
		includePreviousMessages,
	};

	if (selectedModel.reasoning?.type === ReasoningType.HybridWithTokens && !isHybridReasoningEnabled) {
		const next = {
			...base,
		};

		delete next.reasoning;

		if (next.temperature === undefined) {
			next.temperature = DefaultUIModelOption.temperature;
		}

		return sanitizeUIModelOptionByCapabilities(next);
	}

	return sanitizeUIModelOptionByCapabilities(base);
}

export interface ComposerContextController {
	chatOptions: UIModelOption;

	selectedModel: UIModelOption;
	allOptions: UIModelOption[];
	modelOptionsLoaded: boolean;
	hasRunnableModel: boolean;
	modelCatalogUnavailableReason: ModelCatalogUnavailableReason | null;
	modelCatalogError: string | null;

	isHybridReasoningEnabled: boolean;
	includePreviousMessages: IncludePreviousMessages;

	handleSetSelectedModel: Dispatch<SetStateAction<UIModelOption>>;
	handleSetIsHybridReasoningEnabled: Dispatch<SetStateAction<boolean>>;
	setIncludePreviousMessages: Dispatch<SetStateAction<IncludePreviousMessages>>;

	setTemperature(temp: number): void;
	setReasoningLevel(level: ReasoningLevel): void;
	setHybridTokens(tokens: number): void;
	setOutputVerbosity(verbosity?: OutputVerbosity): void;

	applyAdvancedModel(updatedModel: UIModelOption): void;
	restoreConversationContext(context: RestorableConversationContext): UIModelOption | null;
	resetForNewConversation(): UIModelOption;

	verbosityEnabled: boolean;
	reasoningLevelOptions: ReasoningLevel[];
}

export function useComposerContextState(): ComposerContextController {
	const [selectedModel, setSelectedModel] = useState(DefaultUIModelOption);
	const [allOptions, setAllOptions] = useState<UIModelOption[]>([]);
	const [modelOptionsLoaded, setModelOptionsLoaded] = useState(false);
	const [hasRunnableModel, setHasRunnableModel] = useState(false);
	const [modelCatalogUnavailableReason, setModelCatalogUnavailableReason] =
		useState<ModelCatalogUnavailableReason | null>(null);
	const [modelCatalogError, setModelCatalogError] = useState<string | null>(null);
	const [isHybridReasoningEnabled, setIsHybridReasoningEnabled] = useState(true);
	const [includePreviousMessages, setIncludePreviousMessages] = useState<IncludePreviousMessages>(
		DefaultUIModelOption.includePreviousMessages
	);

	const selectedModelRef = useRef(selectedModel);
	const allOptionsRef = useRef(allOptions);
	const defaultModelRef = useRef(DefaultUIModelOption);
	const hybridReasoningRef = useRef(isHybridReasoningEnabled);
	const pendingRestoreRef = useRef<RestorableConversationContext | null>(null);

	const applySelectedModel = useCallback(
		(
			update: SetStateAction<UIModelOption>,
			options?: {
				syncHybridReasoning?: boolean;
			}
		) => {
			const current = selectedModelRef.current;
			const next = typeof update === 'function' ? update(current) : update;
			const nextHybrid =
				options?.syncHybridReasoning === true ? isHybridReasoningModel(next) : hybridReasoningRef.current;

			selectedModelRef.current = next;
			hybridReasoningRef.current = nextHybrid;

			setSelectedModel(next);
			setIsHybridReasoningEnabled(nextHybrid);
		},
		[]
	);

	const applyRestoredContext = useCallback((context: RestorableConversationContext, options: UIModelOption[]) => {
		const fallback = defaultModelRef.current ?? options[0] ?? DefaultUIModelOption;
		const next = resolveRestoredSelectedModel(options, fallback, context.modelRef, context.modelParam);

		selectedModelRef.current = next;
		hybridReasoningRef.current = isHybridReasoningModel(next);

		setSelectedModel(next);
		setIsHybridReasoningEnabled(isHybridReasoningModel(next));
		setIncludePreviousMessages(next.includePreviousMessages);

		return next;
	}, []);

	useEffect(() => {
		let cancelled = false;

		void modelManagementAPI
			.getCatalog()
			.then(result => {
				if (cancelled) {
					return;
				}

				const nextDefault = sanitizeUIModelOptionByCapabilities(result.defaultOption);

				allOptionsRef.current = result.options;
				defaultModelRef.current = nextDefault;

				setAllOptions(result.options);
				setModelOptionsLoaded(true);
				setHasRunnableModel(true);
				setModelCatalogUnavailableReason(null);
				setModelCatalogError(null);

				const pendingRestore = pendingRestoreRef.current;
				if (pendingRestore) {
					pendingRestoreRef.current = null;
					applyRestoredContext(pendingRestore, result.options);
					return;
				}

				selectedModelRef.current = nextDefault;
				hybridReasoningRef.current = isHybridReasoningModel(nextDefault);

				setSelectedModel(nextDefault);
				setIsHybridReasoningEnabled(isHybridReasoningModel(nextDefault));
				setIncludePreviousMessages(nextDefault.includePreviousMessages);
			})
			.catch((error: unknown) => {
				if (!cancelled) {
					console.error('Failed to load Composer model options:', error);

					allOptionsRef.current = [];
					defaultModelRef.current = DefaultUIModelOption;
					selectedModelRef.current = DefaultUIModelOption;
					hybridReasoningRef.current = false;

					setAllOptions([]);
					setSelectedModel(DefaultUIModelOption);
					setIsHybridReasoningEnabled(false);
					setIncludePreviousMessages(DefaultUIModelOption.includePreviousMessages);
					setModelOptionsLoaded(false);
					setHasRunnableModel(false);
					setModelCatalogUnavailableReason(isModelCatalogUnavailableError(error) ? error.reason : null);
					setModelCatalogError(
						isModelCatalogUnavailableError(error) ? null : 'Model configuration could not be loaded.'
					);
				}
			});

		return () => {
			cancelled = true;
		};
	}, [applyRestoredContext]);

	const handleSetSelectedModel = useCallback(
		(update: SetStateAction<UIModelOption>) => {
			applySelectedModel(update, {
				syncHybridReasoning: true,
			});
		},
		[applySelectedModel]
	);

	const handleSetIsHybridReasoningEnabled = useCallback((update: SetStateAction<boolean>) => {
		const next = typeof update === 'function' ? update(hybridReasoningRef.current) : update;

		hybridReasoningRef.current = next;
		setIsHybridReasoningEnabled(next);
	}, []);

	const setTemperature = useCallback(
		(temperature: number) => {
			applySelectedModel(current => ({
				...current,
				temperature: Math.max(0, Math.min(1, temperature)),
			}));
		},
		[applySelectedModel]
	);

	const setReasoningLevel = useCallback(
		(level: ReasoningLevel) => {
			applySelectedModel(current => ({
				...current,
				reasoning: {
					type: ReasoningType.SingleWithLevels,
					level,
					tokens: current.reasoning?.tokens ?? 1024,
					summaryStyle: current.reasoning?.summaryStyle,
				},
			}));
		},
		[applySelectedModel]
	);

	const setHybridTokens = useCallback(
		(tokens: number) => {
			applySelectedModel(current => {
				if (current.reasoning?.type !== ReasoningType.HybridWithTokens) {
					return current;
				}

				return {
					...current,
					reasoning: {
						...current.reasoning,
						tokens,
					},
				};
			});
		},
		[applySelectedModel]
	);

	const setOutputVerbosity = useCallback(
		(verbosity: OutputVerbosity) => {
			applySelectedModel(current => {
				const outputParam = {
					...current.outputParam,
				};

				if (verbosity === undefined) {
					delete outputParam.verbosity;
				} else {
					outputParam.verbosity = verbosity;
				}

				return {
					...current,
					outputParam: outputParam.verbosity || outputParam.format ? outputParam : undefined,
				};
			});
		},
		[applySelectedModel]
	);

	const applyAdvancedModel = useCallback(
		(updated: UIModelOption) => {
			applySelectedModel(sanitizeUIModelOptionByCapabilities(updated));
		},
		[applySelectedModel]
	);

	const restoreConversationContext = useCallback(
		(context: RestorableConversationContext) => {
			pendingRestoreRef.current = context;

			if (!modelOptionsLoaded) {
				return null;
			}

			const selected = applyRestoredContext(context, allOptionsRef.current);
			pendingRestoreRef.current = null;
			return selected;
		},
		[applyRestoredContext, modelOptionsLoaded]
	);

	const resetForNewConversation = useCallback(() => {
		pendingRestoreRef.current = null;

		const next = sanitizeUIModelOptionByCapabilities(
			defaultModelRef.current ?? allOptionsRef.current[0] ?? DefaultUIModelOption
		);

		selectedModelRef.current = next;
		hybridReasoningRef.current = isHybridReasoningModel(next);

		setSelectedModel(next);
		setIsHybridReasoningEnabled(isHybridReasoningModel(next));
		setIncludePreviousMessages(next.includePreviousMessages);

		return next;
	}, []);

	const chatOptions = useMemo(
		() => buildChatOptions(selectedModel, includePreviousMessages, isHybridReasoningEnabled),
		[includePreviousMessages, isHybridReasoningEnabled, selectedModel]
	);

	return {
		chatOptions,
		selectedModel,
		allOptions,
		modelOptionsLoaded,
		hasRunnableModel,
		modelCatalogUnavailableReason,
		modelCatalogError,
		isHybridReasoningEnabled,
		includePreviousMessages,
		handleSetSelectedModel,
		handleSetIsHybridReasoningEnabled,
		setIncludePreviousMessages,
		setTemperature,
		setReasoningLevel,
		setHybridTokens,
		setOutputVerbosity,
		applyAdvancedModel,
		restoreConversationContext,
		resetForNewConversation,
		verbosityEnabled: supportsOutputVerbosity(selectedModel.capabilities),
		reasoningLevelOptions: getSupportedReasoningLevels(selectedModel.capabilities),
	};
}
