import { type Dispatch, type SetStateAction, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { RestorableConversationContext } from '@/spec/conversation';
import { type ModelParam, type OutputVerbosity, type ReasoningLevel, ReasoningType } from '@/spec/inference';
import {
	DefaultUIChatOptions,
	type IncludePreviousMessages,
	type ModelPresetRef,
	PREVIOUS_CONVO_SYSTEM_PROMPT_BUNDLEID,
	PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY,
	type UIChatOption,
} from '@/spec/modelpreset';
import { type PromptBundle, PromptRoleEnum } from '@/spec/prompt';
import { type Tool, ToolImplType, type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import { dedupeStringArray } from '@/lib/obj_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import {
	loadInstructionTemplateOptions,
	loadSkillOptions,
	loadToolOptions,
} from '@/assistantpresets/lib/assistant_preset_catalog';
import {
	buildModelPresetRefKey,
	buildSkillRefKey,
	buildToolRefKey,
} from '@/assistantpresets/lib/assistant_preset_utils';
import {
	applyAssistantPresetModelPatch,
	type AssistantPresetOptionItem,
	type AssistantPresetPreparedApplication,
	buildAssistantPresetModelComparisonState,
	normalizeAssistantPresetSkillRefs,
	normalizeAssistantPresetToolChoices,
} from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import { useAssistantPresets } from '@/chats/composer/assistantpresets/use_assistant_presets';
import {
	getSupportedReasoningLevels,
	sanitizeUIChatOptionByCapabilities,
	supportsOutputVerbosity,
} from '@/modelpresets/lib/capabilities_override';
import { getChatInputOptions } from '@/modelpresets/lib/uichatoption_helper';
import { buildPromptTemplateRefKey } from '@/prompts/lib/prompt_template_ref';
import { buildEffectiveSystemPrompt } from '@/prompts/lib/system_prompt_utils';
import type { SystemPromptDraft } from '@/prompts/lib/use_system_prompts';
import { type SystemPromptItem, useSystemPrompts } from '@/prompts/lib/use_system_prompts';
import { normalizeSkillSelectionsToRefs } from '@/skills/lib/skill_identity_utils';

function isHybridReasoningModel(model: UIChatOption): boolean {
	return model.reasoning?.type === ReasoningType.HybridWithTokens;
}

function hasOwn(obj: object, key: string): boolean {
	return Object.prototype.hasOwnProperty.call(obj, key);
}

function applyPersistedModelParamToSelectedModel(
	base: UIChatOption,
	modelParam?: ModelParam,
	options?: { preserveName?: boolean }
): UIChatOption {
	if (!modelParam) return sanitizeUIChatOptionByCapabilities(base);

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

	if (hasOwn(modelParam, 'temperature')) next.temperature = modelParam.temperature;
	else delete next.temperature;

	if (hasOwn(modelParam, 'reasoning')) next.reasoning = modelParam.reasoning;
	else delete next.reasoning;

	if (hasOwn(modelParam, 'outputParam')) next.outputParam = modelParam.outputParam;
	else delete next.outputParam;

	if (hasOwn(modelParam, 'stopSequences')) next.stopSequences = modelParam.stopSequences;
	else delete next.stopSequences;

	if (hasOwn(modelParam, 'additionalParametersRawJSON')) {
		next.additionalParametersRawJSON = modelParam.additionalParametersRawJSON;
	} else {
		delete next.additionalParametersRawJSON;
	}

	// IMPORTANT:
	// selectedModel.systemPrompt in UI state represents the model-default prompt.
	// The restored effective conversation prompt is handled separately as a
	// synthetic selectable prompt source ("previous convo system prompt").
	return sanitizeUIChatOptionByCapabilities(next);
}

function pickUniqueModelOption(options: UIChatOption[]): UIChatOption | undefined {
	return options.length === 1 ? options[0] : undefined;
}

function findRestorableModelOption(
	allOptions: UIChatOption[],
	modelPresetRef?: ModelPresetRef,
	modelParam?: ModelParam
): UIChatOption | undefined {
	const providerName = modelPresetRef?.providerName?.trim();
	const modelPresetID = modelPresetRef?.modelPresetID?.trim();
	const modelName = modelParam?.name?.trim();

	const providerScopedOptions = providerName
		? allOptions.filter(option => option.providerName === providerName)
		: allOptions;

	// 1) Strongest match: exact provider + preset id
	if (providerName && modelPresetID) {
		const exactRef = providerScopedOptions.find(option => option.modelPresetID === modelPresetID);
		if (exactRef) return exactRef;
	}

	// 2) If provider is known but preset id is stale/missing, prefer matching by
	//    underlying model name ONLY within that provider, and only if unique.
	if (providerName && modelName) {
		const byProviderAndName = providerScopedOptions.filter(option => option.name === modelName);
		const uniqueByProviderAndName = pickUniqueModelOption(byProviderAndName);
		if (uniqueByProviderAndName) return uniqueByProviderAndName;

		// Secondary provider-scoped fallback in case very old persisted state used
		// a display-ish name instead of the raw model name.
		const byProviderAndDisplayName = providerScopedOptions.filter(option => option.modelDisplayName === modelName);
		const uniqueByProviderAndDisplayName = pickUniqueModelOption(byProviderAndDisplayName);
		if (uniqueByProviderAndDisplayName) return uniqueByProviderAndDisplayName;

		// Ambiguous within provider -> do not guess.
		return undefined;
	}

	// 3) No provider info: only accept globally unique matches.
	if (modelName) {
		const byGlobalName = allOptions.filter(option => option.name === modelName);
		const uniqueByGlobalName = pickUniqueModelOption(byGlobalName);
		if (uniqueByGlobalName) return uniqueByGlobalName;

		const byGlobalDisplayName = allOptions.filter(option => option.modelDisplayName === modelName);
		const uniqueByGlobalDisplayName = pickUniqueModelOption(byGlobalDisplayName);
		if (uniqueByGlobalDisplayName) return uniqueByGlobalDisplayName;
	}

	return undefined;
}

function resolveRestoredSelectedModel(
	allOptions: UIChatOption[],
	fallbackSelectedModel: UIChatOption,
	modelPresetRef?: ModelPresetRef,
	modelParam?: ModelParam
): UIChatOption {
	const resolvedOption = findRestorableModelOption(allOptions, modelPresetRef, modelParam);
	if (resolvedOption) {
		return applyPersistedModelParamToSelectedModel(resolvedOption, modelParam, {
			preserveName: true,
		});
	}

	// IMPORTANT:
	// If the exact conversation model preset is no longer selectable (disabled,
	// deleted, no API key, etc.), do NOT project its stale provider/model IDs
	// back into selectedModel. That re-selects a non-existent option in UI state.
	//
	// Also do not guess an arbitrary sibling model from the same provider and do
	// not carry forward the stale `modelParam.name` onto a different live preset.
	// Restore onto a real available option and only carry over non-identity
	// advanced settings as best-effort state.
	const fallbackAvailable = fallbackSelectedModel ?? allOptions[0] ?? DefaultUIChatOptions;
	return applyPersistedModelParamToSelectedModel(fallbackAvailable, modelParam, {
		preserveName: false,
	});
}

function buildPreviousConversationSystemPromptItem(prompt: string): SystemPromptItem {
	return {
		identityKey: PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY,
		bundleID: PREVIOUS_CONVO_SYSTEM_PROMPT_BUNDLEID,
		bundleDisplayName: 'Conversation',
		displayName: 'Previous conversation prompt',
		templateSlug: 'previous-conversation-prompt',
		templateVersion: 'persisted',
		role: PromptRoleEnum.System,
		prompt,
		isBuiltIn: true,
	} as SystemPromptItem;
}

function deriveRestoredPromptSelectionState(
	nextSelectedModel: UIChatOption,
	modelParam?: ModelParam
): {
	restoredConversationSystemPrompt: string | null;
	includeModelDefault: boolean;
	selectedPromptKeys: string[];
} {
	// No persisted model params at all:
	// behave like a fresh conversation for prompt-selection UI.
	if (!modelParam) {
		return {
			restoredConversationSystemPrompt: null,
			includeModelDefault: Boolean(nextSelectedModel.systemPrompt.trim()),
			selectedPromptKeys: [],
		};
	}

	// Persisted effective system prompt exists:
	// restore it as ONE synthetic selectable source and do not keep any old
	// prompt selections from prior tab-local state.
	const restoredSystemPrompt = modelParam.systemPrompt?.trim() ?? '';
	if (restoredSystemPrompt) {
		return {
			restoredConversationSystemPrompt: restoredSystemPrompt,
			includeModelDefault: false,
			selectedPromptKeys: [PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY],
		};
	}

	// Persisted model params explicitly had no effective system prompt.
	return {
		restoredConversationSystemPrompt: null,
		includeModelDefault: false,
		selectedPromptKeys: [],
	};
}

function getPresetToolModelCompatibilityError(toolDefinition: Tool, selectedModel: UIChatOption): string | undefined {
	if (toolDefinition.type !== ToolImplType.SDK) {
		return undefined;
	}

	const requiredSDKType = toolDefinition.sdkImpl?.sdkType?.trim();
	const toolLabel = toolDefinition.displayName || toolDefinition.slug || toolDefinition.id;

	if (!requiredSDKType) {
		return `Tool "${toolLabel}" is missing SDK metadata and cannot be applied safely.`;
	}

	if (requiredSDKType !== (selectedModel.providerSDKType as string)) {
		return `Tool "${toolLabel}" requires provider SDK "${requiredSDKType}", but the selected starting model uses "${selectedModel.providerSDKType}".`;
	}

	return undefined;
}

function buildFinalOptions(
	selectedModel: UIChatOption,
	includePreviousMessages: IncludePreviousMessages,
	isHybridReasoningEnabled: boolean,
	includeModelDefault: boolean,
	selectedPromptKeys: string[],
	promptsByKey: Map<string, SystemPromptItem>
): UIChatOption {
	const base: UIChatOption = {
		...selectedModel,
		includePreviousMessages,
		systemPrompt: buildEffectiveSystemPrompt({
			modelDefaultPrompt: selectedModel.systemPrompt,
			includeModelDefault,
			selectedPromptKeys,
			promptsByKey,
		}),
	};

	if (selectedModel.reasoning?.type === ReasoningType.HybridWithTokens && !isHybridReasoningEnabled) {
		const modifiedOptions = { ...base };
		delete modifiedOptions.reasoning;

		if (modifiedOptions.temperature === undefined) {
			modifiedOptions.temperature = DefaultUIChatOptions.temperature;
		}

		return sanitizeUIChatOptionByCapabilities(modifiedOptions);
	}

	return sanitizeUIChatOptionByCapabilities(base);
}

export type AssistantContextController = {
	chatOptions: UIChatOption;

	selectedModel: UIChatOption;
	allOptions: UIChatOption[];
	modelOptionsLoaded: boolean;

	isHybridReasoningEnabled: boolean;
	includePreviousMessages: IncludePreviousMessages;
	prompts: SystemPromptItem[];
	selectedPromptKeys: string[];
	systemPromptBundles: PromptBundle[];
	preferredSystemPromptBundleID: string | null;
	systemPromptsLoading: boolean;
	systemPromptError: string | null;
	includeModelDefault: boolean;

	handleSetSelectedModel: Dispatch<SetStateAction<UIChatOption>>;
	handleSetIsHybridReasoningEnabled: Dispatch<SetStateAction<boolean>>;
	setIncludePreviousMessages: Dispatch<SetStateAction<IncludePreviousMessages>>;
	setIncludeModelDefault: Dispatch<SetStateAction<boolean>>;

	setTemperature: (temp: number) => void;
	setReasoningLevel: (level: ReasoningLevel) => void;
	setHybridTokens: (tokens: number) => void;
	setOutputVerbosity: (verbosity?: OutputVerbosity) => void;

	togglePromptSelection: (identityKey: string) => void;
	addAndSelectPrompt: (draft: SystemPromptDraft) => Promise<void>;
	clearSelectedPromptSources: () => void;
	refreshSystemPrompts: () => Promise<void>;
	getExistingSystemPromptVersions: (bundleID: string, slug: string) => string[];

	assistantPresetOptions: AssistantPresetOptionItem[];
	assistantPresetsLoading: boolean;
	assistantPresetError: string | null;
	refreshAssistantPresets: () => Promise<void>;
	prepareAssistantPresetApplication: (presetKey: string) => Promise<AssistantPresetPreparedApplication | null>;
	applyPreparedAssistantPreset: (prepared: AssistantPresetPreparedApplication) => void;

	applyAdvancedModel: (updatedModel: UIChatOption) => void;

	restoreConversationContext: (context: RestorableConversationContext) => void;
	resetForNewConversation: () => void;

	verbosityEnabled: boolean;
	reasoningLevelOptions: ReasoningLevel[];
};

export function useAssistantContextState(): AssistantContextController {
	const [selectedModel, setSelectedModel] = useState(DefaultUIChatOptions);
	const [allOptions, setAllOptions] = useState([DefaultUIChatOptions]);
	const [optionsLoaded, setOptionsLoaded] = useState(false);

	const [isHybridReasoningEnabled, setIsHybridReasoningEnabled] = useState(true);
	const [includePreviousMessages, setIncludePreviousMessages] = useState<IncludePreviousMessages>(
		DefaultUIChatOptions.includePreviousMessages
	);
	const [rawSelectedPromptKeys, setRawSelectedPromptKeys] = useState<string[]>([]);
	const [includeModelDefault, setIncludeModelDefaultState] = useState(
		Boolean(DefaultUIChatOptions.systemPrompt.trim())
	);
	const [restoredConversationSystemPrompt, setRestoredConversationSystemPrompt] = useState<string | null>(null);

	const selectedModelRef = useRef(selectedModel);
	const isHybridReasoningEnabledRef = useRef(isHybridReasoningEnabled);
	const allOptionsRef = useRef(allOptions);
	const defaultLoadedOptionRef = useRef(DefaultUIChatOptions);
	const pendingRestoreContextRef = useRef<RestorableConversationContext | null>(null);

	useEffect(() => {
		selectedModelRef.current = selectedModel;
	}, [selectedModel]);

	useEffect(() => {
		isHybridReasoningEnabledRef.current = isHybridReasoningEnabled;
	}, [isHybridReasoningEnabled]);

	useEffect(() => {
		allOptionsRef.current = allOptions;
	}, [allOptions]);

	const {
		prompts: storedPrompts,
		bundles: systemPromptBundles,
		preferredBundleID: preferredSystemPromptBundleID,
		loading: systemPromptsLoading,
		error: systemPromptError,
		addPrompt,
		refreshPrompts,
		getExistingVersions,
	} = useSystemPrompts();

	const {
		presetOptions: rawAssistantPresetOptions,
		loading: assistantPresetsLoading,
		error: assistantPresetError,
		refreshPresets: refreshAssistantPresets,
	} = useAssistantPresets();

	const assistantPresetOptions = useMemo(() => {
		// Before model options finish loading, keep the raw preset catalog state.
		if (!optionsLoaded) {
			return rawAssistantPresetOptions;
		}

		const selectableModelPresetKeys = new Set(
			allOptions.map(option =>
				buildModelPresetRefKey({
					providerName: option.providerName,
					modelPresetID: option.modelPresetID,
				})
			)
		);

		return rawAssistantPresetOptions.map(option => {
			if (!option.isSelectable) return option;

			const startingModelPresetRef = option.preset.startingModelPresetRef;
			if (!startingModelPresetRef) return option;

			const modelKey = buildModelPresetRefKey(startingModelPresetRef);
			if (selectableModelPresetKeys.has(modelKey)) return option;

			return {
				...option,
				isSelectable: false,
				availabilityReason: `Starting model preset "${modelKey}" is not currently selectable in chat (disabled, missing API key, or unavailable).`,
			};
		});
	}, [allOptions, optionsLoaded, rawAssistantPresetOptions]);

	const syntheticPreviousConversationPrompt = useMemo(() => {
		const prompt = restoredConversationSystemPrompt?.trim();
		return prompt ? buildPreviousConversationSystemPromptItem(prompt) : null;
	}, [restoredConversationSystemPrompt]);

	const prompts = useMemo(
		() =>
			syntheticPreviousConversationPrompt ? [syntheticPreviousConversationPrompt, ...storedPrompts] : storedPrompts,
		[storedPrompts, syntheticPreviousConversationPrompt]
	);

	const promptsByKey = useMemo(() => new Map(prompts.map(item => [item.identityKey, item])), [prompts]);
	const promptsByKeyRef = useRef(promptsByKey);

	useEffect(() => {
		promptsByKeyRef.current = promptsByKey;
	}, [promptsByKey]);

	const selectedPromptKeys = useMemo(
		() => rawSelectedPromptKeys.filter(key => promptsByKey.has(key)),
		[promptsByKey, rawSelectedPromptKeys]
	);

	const includeModelDefaultRef = useRef(includeModelDefault);
	useEffect(() => {
		includeModelDefaultRef.current = includeModelDefault;
	}, [includeModelDefault]);

	const setIncludeModelDefault = useCallback((action: SetStateAction<boolean>) => {
		const currentValue = includeModelDefaultRef.current;
		const nextValue = typeof action === 'function' ? action(currentValue) : action;
		includeModelDefaultRef.current = nextValue;
		setIncludeModelDefaultState(nextValue);
	}, []);

	const selectedPromptKeysRef = useRef(selectedPromptKeys);
	useEffect(() => {
		selectedPromptKeysRef.current = selectedPromptKeys;
	}, [selectedPromptKeys]);

	const setPromptSelectionState = useCallback(
		(
			nextRestoredConversationSystemPrompt: string | null,
			nextIncludeModelDefault: boolean,
			nextSelectedPromptKeys: string[]
		) => {
			setRestoredConversationSystemPrompt(nextRestoredConversationSystemPrompt);
			includeModelDefaultRef.current = nextIncludeModelDefault;
			setIncludeModelDefaultState(nextIncludeModelDefault);

			const nextKeys = [...nextSelectedPromptKeys];
			selectedPromptKeysRef.current = nextKeys;
			setRawSelectedPromptKeys(nextKeys);
		},
		[]
	);

	const chatOptions = useMemo(
		() =>
			buildFinalOptions(
				selectedModel,
				includePreviousMessages,
				isHybridReasoningEnabled,
				includeModelDefault,
				selectedPromptKeys,
				promptsByKey
			),
		[
			includeModelDefault,
			includePreviousMessages,
			isHybridReasoningEnabled,
			promptsByKey,
			selectedModel,
			selectedPromptKeys,
		]
	);

	const applyRestoredConversationContext = useCallback(
		(context: RestorableConversationContext, availableOptions: UIChatOption[]) => {
			const nextSelectedModel = resolveRestoredSelectedModel(
				availableOptions,
				defaultLoadedOptionRef.current,
				context.modelPresetRef,
				context.modelParam
			);

			selectedModelRef.current = nextSelectedModel;
			setSelectedModel(nextSelectedModel);

			const nextHybridEnabled = isHybridReasoningModel(nextSelectedModel);
			isHybridReasoningEnabledRef.current = nextHybridEnabled;
			setIsHybridReasoningEnabled(nextHybridEnabled);

			const restoredPromptState = deriveRestoredPromptSelectionState(nextSelectedModel, context.modelParam);
			setPromptSelectionState(
				restoredPromptState.restoredConversationSystemPrompt,
				restoredPromptState.includeModelDefault,
				restoredPromptState.selectedPromptKeys
			);
			// We intentionally do not restore includePreviousMessages yet.
			// But hydration must still reset it so stale per-tab UI state cannot
			// leak into restored conversations, especially old/stale ones.
			setIncludePreviousMessages(nextSelectedModel.includePreviousMessages);
		},
		[setPromptSelectionState]
	);

	const restoreConversationContext = useCallback(
		(context: RestorableConversationContext) => {
			pendingRestoreContextRef.current = context;
			if (!optionsLoaded) return;
			applyRestoredConversationContext(context, allOptionsRef.current);
			pendingRestoreContextRef.current = null;
		},
		[applyRestoredConversationContext, optionsLoaded]
	);

	const resetForNewConversation = useCallback(() => {
		pendingRestoreContextRef.current = null;

		const nextSelectedModel = sanitizeUIChatOptionByCapabilities(
			defaultLoadedOptionRef.current ?? allOptionsRef.current[0] ?? DefaultUIChatOptions
		);
		selectedModelRef.current = nextSelectedModel;
		setSelectedModel(nextSelectedModel);

		const nextHybridEnabled = isHybridReasoningModel(nextSelectedModel);
		isHybridReasoningEnabledRef.current = nextHybridEnabled;
		setIsHybridReasoningEnabled(nextHybridEnabled);

		setPromptSelectionState(null, Boolean(nextSelectedModel.systemPrompt.trim()), []);
		setIncludePreviousMessages(nextSelectedModel.includePreviousMessages);
	}, [setPromptSelectionState]);

	const applySelectedModel = useCallback(
		(
			action: SetStateAction<UIChatOption>,
			options?: {
				syncHybridFromModel?: boolean;
			}
		) => {
			const currentSelectedModel = selectedModelRef.current;
			const currentIsHybridReasoningEnabled = isHybridReasoningEnabledRef.current;

			const nextSelectedModel =
				typeof action === 'function'
					? (action as (prevState: UIChatOption) => UIChatOption)(currentSelectedModel)
					: action;

			const nextIsHybridReasoningEnabled = options?.syncHybridFromModel
				? isHybridReasoningModel(nextSelectedModel)
				: currentIsHybridReasoningEnabled;

			// Keep refs in sync immediately so same-tick follow-up logic
			// cannot read stale selected model / reasoning mode.
			selectedModelRef.current = nextSelectedModel;
			isHybridReasoningEnabledRef.current = nextIsHybridReasoningEnabled;

			setSelectedModel(nextSelectedModel);

			if (nextIsHybridReasoningEnabled !== currentIsHybridReasoningEnabled) {
				setIsHybridReasoningEnabled(nextIsHybridReasoningEnabled);
			}
		},
		[]
	);

	useEffect(() => {
		let cancelled = false;

		void (async () => {
			const r = await getChatInputOptions();
			if (cancelled) return;

			const nextSelectedModel = sanitizeUIChatOptionByCapabilities(r.default);
			const nextIsHybridReasoningEnabled = isHybridReasoningModel(nextSelectedModel);

			setAllOptions(r.allOptions);
			allOptionsRef.current = r.allOptions;
			defaultLoadedOptionRef.current = nextSelectedModel;
			setOptionsLoaded(true);

			const pendingRestore = pendingRestoreContextRef.current;
			if (pendingRestore) {
				pendingRestoreContextRef.current = null;
				applyRestoredConversationContext(pendingRestore, r.allOptions);
				return;
			}

			setSelectedModel(nextSelectedModel);
			setIsHybridReasoningEnabled(nextIsHybridReasoningEnabled);
			setIncludePreviousMessages(nextSelectedModel.includePreviousMessages);
			setPromptSelectionState(null, Boolean(nextSelectedModel.systemPrompt.trim()), []);
		})();

		return () => {
			cancelled = true;
		};
	}, [applyRestoredConversationContext, setPromptSelectionState]);

	const handleSetSelectedModel = useCallback(
		(action: SetStateAction<UIChatOption>) => {
			applySelectedModel(action, {
				syncHybridFromModel: true,
			});
		},
		[applySelectedModel]
	);

	const handleSetIsHybridReasoningEnabled = useCallback((action: SetStateAction<boolean>) => {
		const currentIsHybridReasoningEnabled = isHybridReasoningEnabledRef.current;
		const nextIsHybridReasoningEnabled =
			typeof action === 'function' ? action(currentIsHybridReasoningEnabled) : action;
		isHybridReasoningEnabledRef.current = nextIsHybridReasoningEnabled;
		setIsHybridReasoningEnabled(nextIsHybridReasoningEnabled);
	}, []);

	const setTemperature = useCallback(
		(temp: number) => {
			const clampedTemp = Math.max(0, Math.min(1, temp));
			applySelectedModel(prev => ({ ...prev, temperature: clampedTemp }));
		},
		[applySelectedModel]
	);

	const setReasoningLevel = useCallback(
		(newLevel: ReasoningLevel) => {
			applySelectedModel(prev => ({
				...prev,
				reasoning: {
					type: ReasoningType.SingleWithLevels,
					level: newLevel,
					tokens: 1024,
					summaryStyle: prev.reasoning?.summaryStyle,
				},
			}));
		},
		[applySelectedModel]
	);

	const setHybridTokens = useCallback(
		(tokens: number) => {
			applySelectedModel(prev => {
				if (!prev.reasoning || prev.reasoning.type !== ReasoningType.HybridWithTokens) return prev;
				return { ...prev, reasoning: { ...prev.reasoning, tokens } };
			});
		},
		[applySelectedModel]
	);

	const setOutputVerbosity = useCallback(
		(verbosity?: OutputVerbosity) => {
			applySelectedModel(prev => {
				const next = { ...(prev.outputParam ?? {}) };

				if (verbosity === undefined) {
					delete next.verbosity;
				} else {
					next.verbosity = verbosity;
				}

				const hasAny = !!next.verbosity || !!next.format;

				return {
					...prev,
					outputParam: hasAny ? next : undefined,
				};
			});
		},
		[applySelectedModel]
	);

	const togglePromptSelection = useCallback((identityKey: string) => {
		setRawSelectedPromptKeys(prev => {
			const next = prev.includes(identityKey) ? prev.filter(item => item !== identityKey) : [...prev, identityKey];
			selectedPromptKeysRef.current = next;
			return next;
		});
	}, []);

	const addAndSelectPrompt = useCallback(
		async (draft: SystemPromptDraft) => {
			const item = await addPrompt(draft);
			setRawSelectedPromptKeys(prev => {
				const next = prev.includes(item.identityKey) ? prev : [...prev, item.identityKey];
				selectedPromptKeysRef.current = next;
				return next;
			});
		},
		[addPrompt]
	);

	const clearSelectedPromptSources = useCallback(() => {
		includeModelDefaultRef.current = false;
		setIncludeModelDefaultState(false);
		selectedPromptKeysRef.current = [];
		setRawSelectedPromptKeys([]);
	}, []);

	const prepareAssistantPresetApplication = useCallback(
		async (presetKey: string): Promise<AssistantPresetPreparedApplication | null> => {
			const option = assistantPresetOptions.find(item => item.key === presetKey);
			if (!option) {
				return null;
			}

			const { preset } = option;
			const currentSelectedModel = selectedModelRef.current;
			let nextSelectedModel = currentSelectedModel;
			let hasModelSelection = false;

			if (preset.startingModelPresetRef) {
				if (!optionsLoaded) {
					throw new Error('Model presets are still loading. Try again in a moment.');
				}

				const matchedModel = allOptionsRef.current.find(
					item =>
						item.providerName === preset.startingModelPresetRef?.providerName &&
						item.modelPresetID === preset.startingModelPresetRef?.modelPresetID
				);

				if (!matchedModel) {
					throw new Error(
						`Model preset "${preset.startingModelPresetRef.providerName}/${preset.startingModelPresetRef.modelPresetID}" is not currently selectable.`
					);
				}

				nextSelectedModel = {
					...matchedModel,
				};
				hasModelSelection = true;
			}

			if (preset.startingModelPresetPatch) {
				nextSelectedModel = applyAssistantPresetModelPatch(nextSelectedModel, preset.startingModelPresetPatch);
				hasModelSelection = true;
			}

			const hasIncludeModelSystemPromptSelection = preset.startingIncludeModelSystemPrompt !== undefined;
			const nextIncludeModelSystemPrompt = hasIncludeModelSystemPromptSelection
				? Boolean(preset.startingIncludeModelSystemPrompt)
				: includeModelDefaultRef.current;

			const hasInstructionTemplateSelection = (preset.startingInstructionTemplateRefs?.length ?? 0) > 0;
			let nextSelectedPromptKeys = selectedPromptKeysRef.current;
			if (hasInstructionTemplateSelection) {
				const requestedPromptKeys = dedupeStringArray(
					(preset.startingInstructionTemplateRefs ?? []).map(buildPromptTemplateRefKey)
				);

				const instructionOptions = await loadInstructionTemplateOptions();
				const instructionOptionByKey = new Map(instructionOptions.map(item => [item.key, item] as const));

				const invalidPromptKey = requestedPromptKeys.find(key => {
					const promptOption = instructionOptionByKey.get(key);
					return !promptOption || !promptOption.isSelectable;
				});

				if (invalidPromptKey) {
					const invalidPromptOption = instructionOptionByKey.get(invalidPromptKey);
					throw new Error(
						invalidPromptOption?.availabilityReason ??
							`Instruction template "${invalidPromptKey}" is not currently available.`
					);
				}

				// Keep the prompt store in sync so selectedPromptKeys can immediately
				// resolve to concrete prompt sources after apply.
				await refreshPrompts();
				nextSelectedPromptKeys = requestedPromptKeys;
			}

			const requestedToolSelections = preset.startingToolSelections ?? [];
			const hasToolsSelection = requestedToolSelections.length > 0;
			const conversationToolChoices: ToolStoreChoice[] = [];
			const webSearchChoices: ToolStoreChoice[] = [];

			if (hasToolsSelection) {
				const toolOptions = await loadToolOptions();
				const toolOptionByKey = new Map(toolOptions.map(item => [item.key, item] as const));
				for (const selection of requestedToolSelections) {
					const toolOption = toolOptionByKey.get(buildToolRefKey(selection.toolRef));
					if (!toolOption || !toolOption.isSelectable) {
						throw new Error(
							toolOption?.availabilityReason ??
								`Tool "${selection.toolRef.bundleID}/${selection.toolRef.toolSlug}@${selection.toolRef.toolVersion}" is not currently available.`
						);
					}

					const rawUserArgs = selection.toolChoicePatch?.userArgSchemaInstance?.trim();
					if (rawUserArgs && !toolOption.hasUserArgSchema) {
						throw new Error(
							`Tool "${selection.toolRef.bundleID}/${selection.toolRef.toolSlug}@${selection.toolRef.toolVersion}" no longer exposes user args, but this assistant preset still contains saved args for it.`
						);
					}

					if (rawUserArgs) {
						try {
							JSON.parse(rawUserArgs);
						} catch {
							throw new Error(
								`Tool "${selection.toolRef.bundleID}/${selection.toolRef.toolSlug}@${selection.toolRef.toolVersion}" contains invalid saved args JSON.`
							);
						}
					}

					const toolDefinition = toolOption.toolDefinition;
					const compatibilityError = getPresetToolModelCompatibilityError(toolDefinition, nextSelectedModel);
					if (compatibilityError) {
						throw new Error(compatibilityError);
					}

					const toolChoice: ToolStoreChoice = {
						choiceID: getUUIDv7(),
						bundleID: selection.toolRef.bundleID,
						toolID: toolDefinition.id,
						toolSlug: toolDefinition.slug,
						toolVersion: toolDefinition.version,
						toolType: toolDefinition.llmToolType,
						displayName: toolDefinition.displayName,
						description: toolDefinition.description,
						autoExecute: selection.toolChoicePatch?.autoExecute ?? toolDefinition.autoExecReco,
						userArgSchemaInstance: rawUserArgs || undefined,
					};

					if (toolDefinition.llmToolType === ToolStoreChoiceType.WebSearch) {
						webSearchChoices.push(toolChoice);
					} else {
						conversationToolChoices.push(toolChoice);
					}
				}
			}

			const requestedSkillSels = preset.startingSkillSelections ?? [];
			// Preset semantics: empty or missing means "no opinion", not "clear current skills".
			const hasSkillsSelection = requestedSkillSels.length > 0;
			const enabledSkillRefs = hasSkillsSelection ? normalizeSkillSelectionsToRefs(requestedSkillSels) : [];
			const activeSkillRefs = hasSkillsSelection
				? normalizeSkillSelectionsToRefs(requestedSkillSels.filter(sel => sel.preLoadAsActive))
				: [];

			if (hasSkillsSelection) {
				const skillOptions = await loadSkillOptions();
				const skillOptionByKey = new Map(skillOptions.map(item => [item.key, item] as const));

				const invalidSkillSel = requestedSkillSels.find(sel => {
					const skillOption = skillOptionByKey.get(buildSkillRefKey(sel.skillRef));
					return !skillOption || !skillOption.isSelectable;
				});

				if (invalidSkillSel) {
					const invalidSkillOption = skillOptionByKey.get(buildSkillRefKey(invalidSkillSel.skillRef));
					throw new Error(
						invalidSkillOption?.availabilityReason ??
							`Skill "${invalidSkillSel.skillRef.bundleID}/${invalidSkillSel.skillRef.skillSlug}#${invalidSkillSel.skillRef.skillID}" is not currently available.`
					);
				}
			}

			return {
				presetKey,
				option,
				preset,
				hasModelSelection,
				nextSelectedModel,
				hasIncludeModelSystemPromptSelection,
				nextIncludeModelSystemPrompt,
				hasInstructionTemplateSelection,
				nextSelectedPromptKeys,
				runtimeSelections: {
					hasToolsSelection,
					conversationToolChoices,
					webSearchChoices,
					hasSkillsSelection,
					enabledSkillRefs,
					activeSkillRefs,
				},
				comparisonState: {
					model: buildAssistantPresetModelComparisonState(preset, nextSelectedModel, nextIncludeModelSystemPrompt),
					instructions: hasInstructionTemplateSelection ? [...nextSelectedPromptKeys] : undefined,
					tools: hasToolsSelection
						? {
								conversationToolChoices: normalizeAssistantPresetToolChoices(conversationToolChoices),
								webSearchChoices: normalizeAssistantPresetToolChoices(webSearchChoices),
							}
						: undefined,
					skills: hasSkillsSelection ? normalizeAssistantPresetSkillRefs(enabledSkillRefs) : undefined,
				},
			};
		},
		[assistantPresetOptions, optionsLoaded, refreshPrompts]
	);

	const applyPreparedAssistantPreset = useCallback(
		(prepared: AssistantPresetPreparedApplication) => {
			if (prepared.hasModelSelection) {
				applySelectedModel(prepared.nextSelectedModel, {
					syncHybridFromModel: true,
				});
			}

			if (prepared.hasIncludeModelSystemPromptSelection) {
				includeModelDefaultRef.current = prepared.nextIncludeModelSystemPrompt;
				setIncludeModelDefaultState(prepared.nextIncludeModelSystemPrompt);
			}

			if (prepared.hasInstructionTemplateSelection) {
				selectedPromptKeysRef.current = [...prepared.nextSelectedPromptKeys];
				setRawSelectedPromptKeys(prepared.nextSelectedPromptKeys);
			}
		},
		[applySelectedModel]
	);

	const applyAdvancedModel = useCallback(
		(updatedModel: UIChatOption) => {
			applySelectedModel(sanitizeUIChatOptionByCapabilities(updatedModel));
		},
		[applySelectedModel]
	);

	const verbosityEnabled = supportsOutputVerbosity(selectedModel.capabilitiesOverride);
	const reasoningLevelOptions = getSupportedReasoningLevels(selectedModel.capabilitiesOverride);

	return {
		chatOptions,

		selectedModel,
		allOptions,
		modelOptionsLoaded: optionsLoaded,

		isHybridReasoningEnabled,
		includePreviousMessages,
		prompts,
		selectedPromptKeys,
		systemPromptBundles,
		preferredSystemPromptBundleID,
		systemPromptsLoading,
		systemPromptError,
		includeModelDefault,

		handleSetSelectedModel,
		handleSetIsHybridReasoningEnabled,
		setIncludePreviousMessages,
		setIncludeModelDefault,

		setTemperature,
		setReasoningLevel,
		setHybridTokens,
		setOutputVerbosity,

		togglePromptSelection,
		addAndSelectPrompt,
		clearSelectedPromptSources,
		refreshSystemPrompts: refreshPrompts,
		getExistingSystemPromptVersions: getExistingVersions,

		assistantPresetOptions,
		assistantPresetsLoading,
		assistantPresetError,
		refreshAssistantPresets,
		prepareAssistantPresetApplication,
		applyPreparedAssistantPreset,

		applyAdvancedModel,

		restoreConversationContext,
		resetForNewConversation,
		verbosityEnabled,
		reasoningLevelOptions,
	};
}
