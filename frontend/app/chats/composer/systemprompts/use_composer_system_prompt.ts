import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { AssistantPreset } from '@/spec/assistantpreset';
import type { ModelParam } from '@/spec/inference';
import { PREVIOUS_CONVO_SYSTEM_PROMPT_BUNDLEID, PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY } from '@/spec/modelpreset';
import { type PromptBundle, PromptRoleEnum } from '@/spec/prompt';

import { dedupeStringArray } from '@/lib/obj_utils';

import { loadInstructionTemplateOptions } from '@/assistantpresets/lib/assistant_preset_catalog';
import type { AssistantPresetPreparedApplication } from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import { buildPromptTemplateRefKey } from '@/prompts/lib/prompt_template_ref';
import { buildEffectiveSystemPrompt } from '@/prompts/lib/system_prompt_utils';
import type { SystemPromptDraft, SystemPromptItem } from '@/prompts/lib/use_system_prompts';
import { useSystemPrompts } from '@/prompts/lib/use_system_prompts';

export interface ComposerSystemPromptPreparedSelection {
	hasIncludeModelSystemPromptSelection: boolean;
	nextIncludeModelSystemPrompt: boolean;
	hasInstructionTemplateSelection: boolean;
	nextSelectedPromptKeys: string[];
}

export interface ComposerSystemPromptController {
	modelDefaultPrompt: string;
	prompts: SystemPromptItem[];
	systemPromptBundles: PromptBundle[];
	preferredSystemPromptBundleID: string | null;
	systemPromptsLoading: boolean;
	systemPromptError: string | null;
	includeModelDefault: boolean;
	selectedPromptKeys: string[];
	resolvedSystemPrompt: string;
	setIncludeModelDefault: (next: boolean) => void;
	togglePromptSelection: (identityKey: string) => void;
	addAndSelectPrompt: (draft: SystemPromptDraft) => Promise<void>;
	clearSelectedPromptSources: () => void;
	refreshSystemPrompts: () => Promise<void>;
	getExistingSystemPromptVersions: (bundleID: string, slug: string) => string[];
	resetForNewConversation: (modelDefaultPrompt: string) => void;
	restoreConversationContext: (modelDefaultPrompt: string, modelParam?: ModelParam) => void;
	prepareAssistantPresetSelections: (preset: AssistantPreset) => Promise<ComposerSystemPromptPreparedSelection>;
	applyPreparedAssistantPresetSelections: (
		prepared: Pick<
			AssistantPresetPreparedApplication,
			| 'hasIncludeModelSystemPromptSelection'
			| 'nextIncludeModelSystemPrompt'
			| 'hasInstructionTemplateSelection'
			| 'nextSelectedPromptKeys'
		>
	) => void;
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
	modelDefaultPrompt: string,
	modelParam?: ModelParam
): {
	restoredConversationSystemPrompt: string | null;
	includeModelDefault: boolean;
	selectedPromptKeys: string[];
} {
	if (!modelParam) {
		return {
			restoredConversationSystemPrompt: null,
			includeModelDefault: Boolean(modelDefaultPrompt.trim()),
			selectedPromptKeys: [],
		};
	}

	const restoredSystemPrompt = modelParam.systemPrompt?.trim() ?? '';
	if (restoredSystemPrompt) {
		return {
			restoredConversationSystemPrompt: restoredSystemPrompt,
			includeModelDefault: false,
			selectedPromptKeys: [PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY],
		};
	}

	return {
		restoredConversationSystemPrompt: null,
		includeModelDefault: false,
		selectedPromptKeys: [],
	};
}

export function useComposerSystemPrompt(args: {
	modelDefaultPrompt: string;
	modelOptionsLoaded: boolean;
}): ComposerSystemPromptController {
	const { modelDefaultPrompt, modelOptionsLoaded } = args;

	const [rawSelectedPromptKeys, setRawSelectedPromptKeys] = useState<string[]>([]);
	const [includeModelDefault, setIncludeModelDefaultState] = useState(false);
	const [restoredConversationSystemPrompt, setRestoredConversationSystemPrompt] = useState<string | null>(null);
	const [initializedFromModel, setInitializedFromModelState] = useState(false);
	const initializedFromModelRef = useRef(initializedFromModel);

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

	useEffect(() => {
		initializedFromModelRef.current = initializedFromModel;
	}, [initializedFromModel]);

	// Initialize includeModelDefault once when model options finish loading,
	// unless a restore/reset has already set the flag.
	if (modelOptionsLoaded && !initializedFromModel) {
		setInitializedFromModelState(true);
		setIncludeModelDefaultState(Boolean(modelDefaultPrompt.trim()));
	}

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

	const selectedPromptKeys = useMemo(
		() => rawSelectedPromptKeys.filter(key => promptsByKey.has(key)),
		[promptsByKey, rawSelectedPromptKeys]
	);

	const includeModelDefaultRef = useRef(includeModelDefault);
	useEffect(() => {
		includeModelDefaultRef.current = includeModelDefault;
	}, [includeModelDefault]);

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

	const resolvedSystemPrompt = useMemo(
		() =>
			buildEffectiveSystemPrompt({
				modelDefaultPrompt,
				includeModelDefault,
				selectedPromptKeys,
				promptsByKey,
			}),
		[includeModelDefault, modelDefaultPrompt, promptsByKey, selectedPromptKeys]
	);

	const setIncludeModelDefault = useCallback((next: boolean) => {
		includeModelDefaultRef.current = next;
		setIncludeModelDefaultState(next);
	}, []);

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
		setPromptSelectionState(null, false, []);
	}, [setPromptSelectionState]);

	const resetForNewConversation = useCallback(
		(nextModelDefaultPrompt: string) => {
			initializedFromModelRef.current = true;
			setPromptSelectionState(null, Boolean(nextModelDefaultPrompt.trim()), []);
		},
		[setPromptSelectionState]
	);

	const restoreConversationContext = useCallback(
		(nextModelDefaultPrompt: string, modelParam?: ModelParam) => {
			initializedFromModelRef.current = true;
			const restoredPromptState = deriveRestoredPromptSelectionState(nextModelDefaultPrompt, modelParam);
			setPromptSelectionState(
				restoredPromptState.restoredConversationSystemPrompt,
				restoredPromptState.includeModelDefault,
				restoredPromptState.selectedPromptKeys
			);
		},
		[setPromptSelectionState]
	);

	const prepareAssistantPresetSelections = useCallback(
		async (preset: AssistantPreset): Promise<ComposerSystemPromptPreparedSelection> => {
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

				await refreshPrompts();
				nextSelectedPromptKeys = requestedPromptKeys;
			}

			return {
				hasIncludeModelSystemPromptSelection,
				nextIncludeModelSystemPrompt,
				hasInstructionTemplateSelection,
				nextSelectedPromptKeys,
			};
		},
		[refreshPrompts]
	);

	const applyPreparedAssistantPresetSelections = useCallback(
		(
			prepared: Pick<
				AssistantPresetPreparedApplication,
				| 'hasIncludeModelSystemPromptSelection'
				| 'nextIncludeModelSystemPrompt'
				| 'hasInstructionTemplateSelection'
				| 'nextSelectedPromptKeys'
			>
		) => {
			if (prepared.hasIncludeModelSystemPromptSelection) {
				includeModelDefaultRef.current = prepared.nextIncludeModelSystemPrompt;
				setIncludeModelDefaultState(prepared.nextIncludeModelSystemPrompt);
			}

			if (prepared.hasInstructionTemplateSelection) {
				selectedPromptKeysRef.current = [...prepared.nextSelectedPromptKeys];
				setRawSelectedPromptKeys(prepared.nextSelectedPromptKeys);
			}
		},
		[]
	);

	return {
		modelDefaultPrompt,
		prompts,
		systemPromptBundles,
		preferredSystemPromptBundleID,
		systemPromptsLoading,
		systemPromptError,
		includeModelDefault,
		selectedPromptKeys,
		resolvedSystemPrompt,
		setIncludeModelDefault,
		togglePromptSelection,
		addAndSelectPrompt,
		clearSelectedPromptSources,
		refreshSystemPrompts: refreshPrompts,
		getExistingSystemPromptVersions: getExistingVersions,
		resetForNewConversation,
		restoreConversationContext,
		prepareAssistantPresetSelections,
		applyPreparedAssistantPresetSelections,
	};
}
