import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { OutputVerbosity, ReasoningLevel } from '@/spec/inference';
import type { IncludePreviousMessages, UIModelOption } from '@/spec/model';
import type { RestorableConversationContext, SessionConfiguration, SessionModelDefaultsOverride } from '@/spec/session';
import { ReasoningType } from '@/spec/inference';
import { DefaultUIModelOption } from '@/spec/model';
import { SessionModelDefaultField } from '@/spec/session';

import { getSupportedReasoningLevels, supportsOutputVerbosity } from '@/lib/model_capability_helpers';

import type { ModelCatalogUnavailableReason } from '@/apis/model_management';
import { modelManagementAPI } from '@/apis/baseapi';
import { isModelCatalogUnavailableError } from '@/apis/model_management';

import {
	buildModelOptionsWithPreviousMessages,
	findRestorableModelOption,
	isHybridReasoningModel,
	resolveRestoredSelectedModel,
} from '@/chats/composer/models/model_option_helpers';
import {
	setHybridReasoningPreference,
	setHybridTokensPreference,
	setOutputVerbosityPreference,
	setReasoningLevelPreference,
	setTemperaturePreference,
} from '@/chats/composer/models/request_preferences';

type ComposerModelSelectionSource = 'session' | 'agent';

function requestPatchFromSessionOverrides(overrides: SessionModelDefaultsOverride) {
	const defaults = {
		...(overrides.stream === undefined ? {} : { stream: overrides.stream }),
		...(overrides.maxPromptTokens === undefined ? {} : { maxPromptTokens: overrides.maxPromptTokens }),
		...(overrides.maxOutputTokens === undefined ? {} : { maxOutputTokens: overrides.maxOutputTokens }),
		...(overrides.temperature === undefined ? {} : { temperature: overrides.temperature }),
		...(overrides.systemPrompt === undefined ? {} : { systemPrompt: overrides.systemPrompt }),
		...(overrides.timeoutMS === undefined ? {} : { timeoutMS: overrides.timeoutMS }),
		...(overrides.reasoning ? { reasoning: structuredClone(overrides.reasoning) } : {}),
		...(overrides.cacheControl ? { cacheControl: structuredClone(overrides.cacheControl) } : {}),
		...(overrides.output ? { output: structuredClone(overrides.output) } : {}),
		...(overrides.stopSequences ? { stopSequences: [...overrides.stopSequences] } : {}),
		...(overrides.adapterParameters ? { adapterParameters: structuredClone(overrides.adapterParameters) } : {}),
	};

	const knownClearFields = new Set(Object.values(SessionModelDefaultField));
	const clear = (overrides.clear ?? []).filter(field =>
		knownClearFields.has(field as string as SessionModelDefaultField)
	);

	if (Object.keys(defaults).length === 0 && clear.length === 0) {
		return undefined;
	}

	return {
		...(Object.keys(defaults).length > 0 ? { defaults } : {}),
		...(clear.length > 0 ? { clear } : {}),
	};
}

function includePreviousMessagesFromSession(configuration: SessionConfiguration): IncludePreviousMessages {
	if (configuration.history.all) {
		return 'all';
	}

	return Math.max(0, configuration.history.includePreviousUserTurns ?? 0);
}

export interface ComposerContextController {
	modelOptions: UIModelOption;

	selectedModel: UIModelOption;
	allOptions: UIModelOption[];
	modelOptionsLoaded: boolean;
	hasRunnableModel: boolean;
	modelCatalogUnavailableReason: ModelCatalogUnavailableReason | null;
	modelCatalogError: string | null;

	modelSelectionSource: ComposerModelSelectionSource;
	isHybridReasoningEnabled: boolean;
	includePreviousMessages: IncludePreviousMessages;

	handleSetSelectedModel: Dispatch<SetStateAction<UIModelOption>>;
	handleSetIsHybridReasoningEnabled: (enabled?: boolean) => void;
	setIncludePreviousMessages: Dispatch<SetStateAction<IncludePreviousMessages>>;
	setModelSelectionSource: (source: ComposerModelSelectionSource) => void;
	setAgentModelPreview: (model: UIModelOption | undefined, agentOwnsModel: boolean) => void;

	setTemperature(temp?: number): void;
	setReasoningLevel(level?: ReasoningLevel): void;
	setHybridTokens(tokens?: number): void;
	setOutputVerbosity(verbosity?: OutputVerbosity): void;

	applyAdvancedModel(updatedModel: UIModelOption): void;
	restoreConversationContext(context: RestorableConversationContext): UIModelOption | null;
	restoreSessionConfiguration(configuration: SessionConfiguration): UIModelOption | null;
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
	const [modelSelectionSource, setModelSelectionSourceState] = useState<ComposerModelSelectionSource>('session');

	const selectedModelRef = useRef(selectedModel);
	const allOptionsRef = useRef(allOptions);
	const defaultModelRef = useRef(DefaultUIModelOption);
	const hybridReasoningRef = useRef(isHybridReasoningEnabled);
	const pendingRestoreRef = useRef<RestorableConversationContext | null>(null);
	const pendingSessionConfigurationRef = useRef<SessionConfiguration | null>(null);
	const modelSelectionSourceRef = useRef<ComposerModelSelectionSource>('session');

	const setModelSelectionSource = useCallback((source: ComposerModelSelectionSource) => {
		modelSelectionSourceRef.current = source;
		setModelSelectionSourceState(source);
	}, []);

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

	const applyRestoredSessionConfiguration = useCallback(
		(configuration: SessionConfiguration, options: UIModelOption[]) => {
			const fallback = defaultModelRef.current ?? options[0] ?? DefaultUIModelOption;
			const resolved = configuration.model ? findRestorableModelOption(options, configuration.model) : undefined;
			const requestPatch = requestPatchFromSessionOverrides(configuration.modelOverrides);
			const restored = {
				...(resolved ?? fallback),
				...(requestPatch
					? {
							requestPatch,
						}
					: {}),
			} as UIModelOption;
			const reasoningCleared =
				configuration.modelOverrides.clear?.includes(SessionModelDefaultField.Reasoning) ?? false;
			const nextHybridReasoningEnabled = isHybridReasoningModel(restored) && !reasoningCleared;

			selectedModelRef.current = restored;
			hybridReasoningRef.current = nextHybridReasoningEnabled;

			setSelectedModel(restored);
			setIsHybridReasoningEnabled(nextHybridReasoningEnabled);
			setIncludePreviousMessages(includePreviousMessagesFromSession(configuration));

			// An omitted Session model means only that there is no explicit
			// override. Agent ownership is established later by Agent resolution,
			// not inferred from the absence of configuration.model.
			setModelSelectionSource('session');

			return restored;
		},
		[setModelSelectionSource]
	);

	useEffect(() => {
		let cancelled = false;

		void modelManagementAPI
			.getCatalog()
			.then(result => {
				if (cancelled) {
					return;
				}

				const nextDefault = result.defaultOption;

				allOptionsRef.current = result.options;
				defaultModelRef.current = nextDefault;

				setAllOptions(result.options);
				setModelOptionsLoaded(true);
				setHasRunnableModel(true);
				setModelCatalogUnavailableReason(null);
				setModelCatalogError(null);

				const pendingSessionConfiguration = pendingSessionConfigurationRef.current;
				if (pendingSessionConfiguration) {
					pendingSessionConfigurationRef.current = null;
					applyRestoredSessionConfiguration(pendingSessionConfiguration, result.options);
					return;
				}

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
	}, [applyRestoredContext, applyRestoredSessionConfiguration]);

	const handleSetSelectedModel = useCallback(
		(update: SetStateAction<UIModelOption>) => {
			setModelSelectionSource('session');
			applySelectedModel(update, {
				syncHybridReasoning: true,
			});
		},
		[applySelectedModel, setModelSelectionSource]
	);

	const handleSetIsHybridReasoningEnabled = useCallback(
		(enabled?: boolean) => {
			const fallback = isHybridReasoningModel(selectedModelRef.current);
			const next = enabled ?? fallback;

			applySelectedModel(current => ({
				...current,
				requestPatch: setHybridReasoningPreference(current.requestPatch, enabled),
			}));

			hybridReasoningRef.current = next;
			setIsHybridReasoningEnabled(next);
		},
		[applySelectedModel]
	);

	const setAgentModelPreview = useCallback(
		(model: UIModelOption | undefined, agentOwnsModel: boolean) => {
			if (model) {
				applySelectedModel(model, { syncHybridReasoning: true });
			} else if (!agentOwnsModel && modelSelectionSourceRef.current === 'agent') {
				// Moving from an Agent-owned Model to an Agent with no Model must
				// not leave the previous Agent's model looking like the fallback.
				// Preserve an explicit user/session choice, but otherwise return
				// to the catalog default.
				const fallback = defaultModelRef.current ?? allOptionsRef.current[0] ?? selectedModelRef.current;
				applySelectedModel(fallback, { syncHybridReasoning: true });
			}

			setModelSelectionSource(agentOwnsModel ? 'agent' : 'session');
		},
		[applySelectedModel, setModelSelectionSource]
	);

	const setTemperature = useCallback(
		(temperature?: number) => {
			applySelectedModel(current => ({
				...current,
				requestPatch: setTemperaturePreference(current.requestPatch, temperature),
			}));
		},
		[applySelectedModel]
	);

	const setReasoningLevel = useCallback(
		(level: ReasoningLevel) => {
			applySelectedModel(current => {
				if (current.reasoning?.type !== ReasoningType.SingleWithLevels) {
					return current;
				}

				return {
					...current,
					requestPatch: setReasoningLevelPreference(current.requestPatch, level),
				};
			});
		},
		[applySelectedModel]
	);

	const setHybridTokens = useCallback(
		(tokens?: number) => {
			applySelectedModel(current => ({
				...current,
				requestPatch: setHybridTokensPreference(current.requestPatch, tokens),
			}));
		},
		[applySelectedModel]
	);

	const setOutputVerbosity = useCallback(
		(verbosity?: OutputVerbosity) => {
			applySelectedModel(current => ({
				...current,
				requestPatch: setOutputVerbosityPreference(current.requestPatch, verbosity),
			}));
		},
		[applySelectedModel]
	);

	const applyAdvancedModel = useCallback(
		(updated: UIModelOption) => {
			applySelectedModel(updated);
		},
		[applySelectedModel]
	);

	const restoreConversationContext = useCallback(
		(context: RestorableConversationContext) => {
			pendingRestoreRef.current = context;
			pendingSessionConfigurationRef.current = null;

			if (!modelOptionsLoaded) {
				return null;
			}

			const selected = applyRestoredContext(context, allOptionsRef.current);
			pendingRestoreRef.current = null;
			return selected;
		},
		[applyRestoredContext, modelOptionsLoaded]
	);

	const restoreSessionConfiguration = useCallback(
		(configuration: SessionConfiguration) => {
			pendingRestoreRef.current = null;

			// See applyRestoredSessionConfiguration: do not render an Agent
			// ownership badge until the selected Agent actually resolves a Model.
			setModelSelectionSource('session');

			if (!modelOptionsLoaded) {
				pendingSessionConfigurationRef.current = configuration;
				return null;
			}

			pendingSessionConfigurationRef.current = null;
			return applyRestoredSessionConfiguration(configuration, allOptionsRef.current);
		},
		[applyRestoredSessionConfiguration, modelOptionsLoaded, setModelSelectionSource]
	);

	const resetForNewConversation = useCallback(() => {
		pendingRestoreRef.current = null;
		pendingSessionConfigurationRef.current = null;

		const next = defaultModelRef.current ?? allOptionsRef.current[0] ?? DefaultUIModelOption;
		setModelSelectionSource('session');

		selectedModelRef.current = next;
		hybridReasoningRef.current = isHybridReasoningModel(next);

		setSelectedModel(next);
		setIsHybridReasoningEnabled(isHybridReasoningModel(next));
		setIncludePreviousMessages(next.includePreviousMessages);

		return next;
	}, [setModelSelectionSource]);

	const modelOptions = useMemo(
		() => buildModelOptionsWithPreviousMessages(selectedModel, includePreviousMessages),
		[includePreviousMessages, selectedModel]
	);

	return {
		modelOptions,
		selectedModel,
		allOptions,
		modelOptionsLoaded,
		hasRunnableModel,
		modelCatalogUnavailableReason,
		modelCatalogError,
		modelSelectionSource,
		isHybridReasoningEnabled,
		includePreviousMessages,
		handleSetSelectedModel,
		handleSetIsHybridReasoningEnabled,
		setIncludePreviousMessages,
		setModelSelectionSource,
		setAgentModelPreview,
		setTemperature,
		setReasoningLevel,
		setHybridTokens,
		setOutputVerbosity,
		applyAdvancedModel,
		restoreConversationContext,
		restoreSessionConfiguration,
		resetForNewConversation,
		verbosityEnabled: supportsOutputVerbosity(selectedModel.capabilities),
		reasoningLevelOptions: getSupportedReasoningLevels(selectedModel.capabilities),
	};
}
