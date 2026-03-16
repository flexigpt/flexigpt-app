import { type Dispatch, type SetStateAction, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { type OutputVerbosity, type ReasoningLevel, ReasoningType } from '@/spec/inference';
import { DefaultUIChatOptions, type IncludePreviousMessages, type UIChatOption } from '@/spec/modelpreset';

import {
	getSupportedReasoningLevels,
	sanitizeUIChatOptionByCapabilities,
	supportsOutputVerbosity,
} from '@/chats/assitantcontexts/capabilities_override_helper';
import { getChatInputOptions } from '@/chats/assitantcontexts/context_uichatoption_helper';
import { buildEffectiveSystemPrompt } from '@/chats/assitantcontexts/system_prompt_utils';
import { type SystemPromptItem, useSystemPrompts } from '@/chats/assitantcontexts/use_system_prompts';
import { useSetSystemPromptForChat } from '@/chats/events/set_system_prompt';

function isHybridReasoningModel(model: UIChatOption): boolean {
	return model.reasoning?.type === ReasoningType.HybridWithTokens;
}

function buildFinalOptions(
	selectedModel: UIChatOption,
	includePreviousMessages: IncludePreviousMessages,
	isHybridReasoningEnabled: boolean,
	includeModelDefault: boolean,
	selectedPromptIds: string[],
	promptsById: Map<string, SystemPromptItem>
): UIChatOption {
	const base: UIChatOption = {
		...selectedModel,
		includePreviousMessages,
		systemPrompt: buildEffectiveSystemPrompt({
			modelDefaultPrompt: selectedModel.systemPrompt,
			includeModelDefault,
			selectedPromptIds,
			promptsById,
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

	isHybridReasoningEnabled: boolean;
	includePreviousMessages: IncludePreviousMessages;
	prompts: SystemPromptItem[];
	selectedPromptIds: string[];
	includeModelDefault: boolean;

	handleSetSelectedModel: Dispatch<SetStateAction<UIChatOption>>;
	handleSetIsHybridReasoningEnabled: Dispatch<SetStateAction<boolean>>;
	setIncludePreviousMessages: Dispatch<SetStateAction<IncludePreviousMessages>>;
	setIncludeModelDefault: Dispatch<SetStateAction<boolean>>;

	setTemperature: (temp: number) => void;
	setReasoningLevel: (level: ReasoningLevel) => void;
	setHybridTokens: (tokens: number) => void;
	setOutputVerbosity: (verbosity?: OutputVerbosity) => void;

	togglePromptSelection: (id: string) => void;
	addAndSelectPrompt: (prompt: string) => void;
	removeSavedPrompt: (id: string) => void;
	clearSelectedPromptSources: () => void;

	applyAdvancedModel: (updatedModel: UIChatOption) => void;

	verbosityEnabled: boolean;
	reasoningLevelOptions: ReasoningLevel[];
};

export function useAssistantContextState({ active }: { active: boolean }): AssistantContextController {
	const [selectedModel, setSelectedModel] = useState<UIChatOption>(DefaultUIChatOptions);
	const [allOptions, setAllOptions] = useState<UIChatOption[]>([DefaultUIChatOptions]);

	const [isHybridReasoningEnabled, setIsHybridReasoningEnabled] = useState(true);
	const [includePreviousMessages, setIncludePreviousMessages] = useState<IncludePreviousMessages>(
		DefaultUIChatOptions.includePreviousMessages
	);
	const [rawSelectedPromptIds, setRawSelectedPromptIds] = useState<string[]>([]);
	const [includeModelDefault, setIncludeModelDefault] = useState<boolean>(
		Boolean(DefaultUIChatOptions.systemPrompt.trim())
	);

	const selectedModelRef = useRef(selectedModel);
	const isHybridReasoningEnabledRef = useRef(isHybridReasoningEnabled);

	useEffect(() => {
		selectedModelRef.current = selectedModel;
	}, [selectedModel]);

	useEffect(() => {
		isHybridReasoningEnabledRef.current = isHybridReasoningEnabled;
	}, [isHybridReasoningEnabled]);

	const { prompts, addPrompt, ensurePrompt, deletePrompt } = useSystemPrompts();

	const promptsById = useMemo(() => new Map(prompts.map(item => [item.id, item])), [prompts]);

	const selectedPromptIds = useMemo(
		() => rawSelectedPromptIds.filter(id => promptsById.has(id)),
		[promptsById, rawSelectedPromptIds]
	);

	const chatOptions = useMemo(
		() =>
			buildFinalOptions(
				selectedModel,
				includePreviousMessages,
				isHybridReasoningEnabled,
				includeModelDefault,
				selectedPromptIds,
				promptsById
			),
		[
			includeModelDefault,
			includePreviousMessages,
			isHybridReasoningEnabled,
			promptsById,
			selectedModel,
			selectedPromptIds,
		]
	);

	const applySelectedModel = useCallback(
		(
			action: SetStateAction<UIChatOption>,
			options?: {
				syncHybridFromModel?: boolean;
				syncModelDefaultSelectionFromModel?: boolean;
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

			setSelectedModel(nextSelectedModel);

			if (nextIsHybridReasoningEnabled !== currentIsHybridReasoningEnabled) {
				setIsHybridReasoningEnabled(nextIsHybridReasoningEnabled);
			}

			if (options?.syncModelDefaultSelectionFromModel) {
				setIncludeModelDefault(Boolean(nextSelectedModel.systemPrompt.trim()));
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

			setSelectedModel(nextSelectedModel);
			setAllOptions(r.allOptions);
			setIsHybridReasoningEnabled(nextIsHybridReasoningEnabled);
			setIncludeModelDefault(Boolean(nextSelectedModel.systemPrompt.trim()));
		})();

		return () => {
			cancelled = true;
		};
	}, []);

	const handleSetSelectedModel = useCallback(
		(action: SetStateAction<UIChatOption>) => {
			applySelectedModel(action, {
				syncHybridFromModel: true,
				syncModelDefaultSelectionFromModel: true,
			});
		},
		[applySelectedModel]
	);

	const handleSetIsHybridReasoningEnabled = useCallback((action: SetStateAction<boolean>) => {
		const currentIsHybridReasoningEnabled = isHybridReasoningEnabledRef.current;
		const nextIsHybridReasoningEnabled =
			typeof action === 'function' ? action(currentIsHybridReasoningEnabled) : action;
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

	const togglePromptSelection = useCallback((id: string) => {
		setRawSelectedPromptIds(prev => (prev.includes(id) ? prev.filter(itemId => itemId !== id) : [...prev, id]));
	}, []);

	const addAndSelectPrompt = useCallback(
		(prompt: string) => {
			const item = addPrompt(prompt);
			if (!item) return;

			setRawSelectedPromptIds(prev => (prev.includes(item.id) ? prev : [...prev, item.id]));
		},
		[addPrompt]
	);

	const removeSavedPrompt = useCallback(
		(id: string) => {
			deletePrompt(id);
			setRawSelectedPromptIds(prev => prev.filter(itemId => itemId !== id));
		},
		[deletePrompt]
	);

	const clearSelectedPromptSources = useCallback(() => {
		setIncludeModelDefault(false);
		setRawSelectedPromptIds([]);
	}, []);

	const handleSetSystemPromptForChat = useCallback(
		(prompt: string) => {
			if (!active) return;

			const trimmed = prompt.trim();
			if (!trimmed) return;

			const item = ensurePrompt(trimmed);
			if (!item) return;

			setIncludeModelDefault(false);
			setRawSelectedPromptIds([item.id]);
		},
		[active, ensurePrompt]
	);

	useSetSystemPromptForChat(handleSetSystemPromptForChat);

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

		isHybridReasoningEnabled,
		includePreviousMessages,
		prompts,
		selectedPromptIds,
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
		removeSavedPrompt,
		clearSelectedPromptSources,

		applyAdvancedModel,

		verbosityEnabled,
		reasoningLevelOptions,
	};
}
