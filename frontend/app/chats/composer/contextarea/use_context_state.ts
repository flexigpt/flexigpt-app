import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { RestorableConversationContext } from '@/spec/conversation';
import type { ModelParam, OutputVerbosity, ReasoningLevel } from '@/spec/inference';
import type { IncludePreviousMessages, ModelPresetRef, UIChatOption } from '@/spec/modelpreset';
import { ReasoningType } from '@/spec/inference';
import { DefaultUIChatOptions } from '@/spec/modelpreset';

import {
	getSupportedReasoningLevels,
	sanitizeUIChatOptionByCapabilities,
	supportsOutputVerbosity,
} from '@/modelpresets/lib/capabilities_override';
import { getChatInputOptions } from '@/modelpresets/lib/uichatoption_helper';

type ChatInputOptionsResult = Awaited<ReturnType<typeof getChatInputOptions>>;

let chatInputOptionsCache: ChatInputOptionsResult | undefined;
let chatInputOptionsPromise: Promise<ChatInputOptionsResult> | undefined;

function loadChatInputOptionsCached(options?: { forceRefresh?: boolean }): Promise<ChatInputOptionsResult> {
	if (!options?.forceRefresh) {
		if (chatInputOptionsCache) {
			return Promise.resolve(chatInputOptionsCache);
		}

		if (chatInputOptionsPromise) {
			return chatInputOptionsPromise;
		}
	} else if (chatInputOptionsPromise) {
		return chatInputOptionsPromise;
	}

	const request = getChatInputOptions().then(result => {
		chatInputOptionsCache = result;
		return result;
	});

	chatInputOptionsPromise = request;

	void request.finally(() => {
		if (chatInputOptionsPromise === request) {
			chatInputOptionsPromise = undefined;
		}
	});

	return request;
}

function hasOwn(value: object, key: string): boolean {
	return Object.hasOwn(value, key);
}

function isHybridReasoningModel(model: UIChatOption): boolean {
	return model.reasoning?.type === ReasoningType.HybridWithTokens;
}

function pickUniqueModelOption(options: UIChatOption[]): UIChatOption | undefined {
	return options.length === 1 ? options[0] : undefined;
}

function applyPersistedModelParam(
	base: UIChatOption,
	modelParam?: ModelParam,
	options?: { preserveName?: boolean }
): UIChatOption {
	if (!modelParam) {
		return sanitizeUIChatOptionByCapabilities(base);
	}

	const next: UIChatOption = {
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

	return sanitizeUIChatOptionByCapabilities(next);
}

function findRestorableModelOption(
	allOptions: UIChatOption[],
	modelPresetRef?: ModelPresetRef,
	modelParam?: ModelParam
): UIChatOption | undefined {
	const providerName = modelPresetRef?.providerName?.trim();
	const modelPresetID = modelPresetRef?.modelPresetID?.trim();
	const modelName = modelParam?.name?.trim();

	const providerOptions = providerName ? allOptions.filter(option => option.providerName === providerName) : allOptions;

	if (providerName && modelPresetID) {
		const direct = providerOptions.find(option => option.modelPresetID === modelPresetID);
		if (direct) {
			return direct;
		}
	}

	if (providerName && modelName) {
		const byName = pickUniqueModelOption(providerOptions.filter(option => option.name === modelName));
		if (byName) {
			return byName;
		}

		return pickUniqueModelOption(providerOptions.filter(option => option.modelDisplayName === modelName));
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
	allOptions: UIChatOption[],
	fallback: UIChatOption,
	modelPresetRef?: ModelPresetRef,
	modelParam?: ModelParam
): UIChatOption {
	const resolved = findRestorableModelOption(allOptions, modelPresetRef, modelParam);

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
	selectedModel: UIChatOption,
	includePreviousMessages: IncludePreviousMessages,
	isHybridReasoningEnabled: boolean
): UIChatOption {
	const base: UIChatOption = {
		...selectedModel,
		includePreviousMessages,
	};

	if (selectedModel.reasoning?.type === ReasoningType.HybridWithTokens && !isHybridReasoningEnabled) {
		const next = {
			...base,
		};

		delete next.reasoning;

		if (next.temperature === undefined) {
			next.temperature = DefaultUIChatOptions.temperature;
		}

		return sanitizeUIChatOptionByCapabilities(next);
	}

	return sanitizeUIChatOptionByCapabilities(base);
}

export interface ComposerContextController {
	chatOptions: UIChatOption;

	selectedModel: UIChatOption;
	allOptions: UIChatOption[];
	modelOptionsLoaded: boolean;

	isHybridReasoningEnabled: boolean;
	includePreviousMessages: IncludePreviousMessages;

	handleSetSelectedModel: Dispatch<SetStateAction<UIChatOption>>;
	handleSetIsHybridReasoningEnabled: Dispatch<SetStateAction<boolean>>;
	setIncludePreviousMessages: Dispatch<SetStateAction<IncludePreviousMessages>>;

	setTemperature(temp: number): void;
	setReasoningLevel(level: ReasoningLevel): void;
	setHybridTokens(tokens: number): void;
	setOutputVerbosity(verbosity?: OutputVerbosity): void;

	applyAdvancedModel(updatedModel: UIChatOption): void;
	restoreConversationContext(context: RestorableConversationContext): UIChatOption | null;
	resetForNewConversation(): UIChatOption;

	verbosityEnabled: boolean;
	reasoningLevelOptions: ReasoningLevel[];
}

export function useComposerContextState(): ComposerContextController {
	const [selectedModel, setSelectedModel] = useState(DefaultUIChatOptions);
	const [allOptions, setAllOptions] = useState([DefaultUIChatOptions]);
	const [modelOptionsLoaded, setModelOptionsLoaded] = useState(false);
	const [isHybridReasoningEnabled, setIsHybridReasoningEnabled] = useState(true);
	const [includePreviousMessages, setIncludePreviousMessages] = useState<IncludePreviousMessages>(
		DefaultUIChatOptions.includePreviousMessages
	);

	const selectedModelRef = useRef(selectedModel);
	const allOptionsRef = useRef(allOptions);
	const defaultModelRef = useRef(DefaultUIChatOptions);
	const hybridReasoningRef = useRef(isHybridReasoningEnabled);
	const pendingRestoreRef = useRef<RestorableConversationContext | null>(null);

	const applySelectedModel = useCallback(
		(
			update: SetStateAction<UIChatOption>,
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

	const applyRestoredContext = useCallback((context: RestorableConversationContext, options: UIChatOption[]) => {
		const fallback = defaultModelRef.current ?? options[0] ?? DefaultUIChatOptions;
		const next = resolveRestoredSelectedModel(options, fallback, context.modelPresetRef, context.modelParam);

		selectedModelRef.current = next;
		hybridReasoningRef.current = isHybridReasoningModel(next);

		setSelectedModel(next);
		setIsHybridReasoningEnabled(isHybridReasoningModel(next));
		setIncludePreviousMessages(next.includePreviousMessages);

		return next;
	}, []);

	useEffect(() => {
		let cancelled = false;

		void loadChatInputOptionsCached({ forceRefresh: true })
			.then(result => {
				if (cancelled) {
					return;
				}

				const nextDefault = sanitizeUIChatOptionByCapabilities(result.default);

				allOptionsRef.current = result.allOptions;
				defaultModelRef.current = nextDefault;

				setAllOptions(result.allOptions);
				setModelOptionsLoaded(true);

				const pendingRestore = pendingRestoreRef.current;
				if (pendingRestore) {
					pendingRestoreRef.current = null;
					applyRestoredContext(pendingRestore, result.allOptions);
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
				}
			});

		return () => {
			cancelled = true;
		};
	}, [applyRestoredContext]);

	const handleSetSelectedModel = useCallback(
		(update: SetStateAction<UIChatOption>) => {
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
		(updated: UIChatOption) => {
			applySelectedModel(sanitizeUIChatOptionByCapabilities(updated));
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

		const next = sanitizeUIChatOptionByCapabilities(
			defaultModelRef.current ?? allOptionsRef.current[0] ?? DefaultUIChatOptions
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
		verbosityEnabled: supportsOutputVerbosity(selectedModel.capabilitiesOverride),
		reasoningLevelOptions: getSupportedReasoningLevels(selectedModel.capabilitiesOverride),
	};
}
