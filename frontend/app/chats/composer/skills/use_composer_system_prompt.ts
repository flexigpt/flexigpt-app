import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { AssistantPreset } from '@/spec/assistantpreset';
import type { ModelParam } from '@/spec/inference';
import { PREVIOUS_CONVO_SYSTEM_PROMPT_BUNDLEID, PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY } from '@/spec/modelpreset';

import { dedupeStringArray } from '@/lib/obj_utils';

import { skillStoreAPI } from '@/apis/baseapi';

import { loadSkillOptions } from '@/assistantpresets/lib/assistant_preset_catalog';
import { buildSkillRefKey } from '@/assistantpresets/lib/assistant_preset_utils';
import type { AssistantPresetPreparedApplication } from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import type { SystemPromptItem } from '@/chats/composer/skills/prompt_utils';
import { buildEffectiveSystemPrompt, PromptRoleEnum } from '@/chats/composer/skills/prompt_utils';
import { getSkillInstructionPromptEligibilityReason } from '@/skills/lib/skill_artifact_utils';

interface ComposerSystemPromptPreparedSelection {
	hasIncludeModelSystemPromptSelection: boolean;
	nextIncludeModelSystemPrompt: boolean;
	hasInstructionTemplateSelection: boolean;
	nextSelectedPromptKeys: string[];
}

interface RenderedInstructionSkillPrompt {
	identityKey?: string;
	displayName: string;
	prompt: string;
	skillRef: { bundleID: string; skillSlug: string; skillID: string };
}

export interface ComposerSystemPromptController {
	modelDefaultPrompt: string;
	prompts: SystemPromptItem[];
	includeModelDefault: boolean;
	selectedPromptKeys: string[];
	resolvedSystemPrompt: string;
	setIncludeModelDefault: (next: boolean) => void;
	togglePromptSelection: (identityKey: string) => void;
	addAndSelectInstructionSkillPrompt: (draft: RenderedInstructionSkillPrompt) => void;
	clearSelectedPromptSources: () => void;
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

function buildInstructionSkillPromptItem(draft: RenderedInstructionSkillPrompt): SystemPromptItem {
	const now = new Date().toISOString();
	const stableKey = `skill-instructions:${buildSkillRefKey(draft.skillRef)}`;

	return {
		identityKey: draft.identityKey ?? stableKey,
		bundleID: draft.skillRef.bundleID,
		bundleDisplayName: 'Skill instructions',
		bundleSlug: draft.skillRef.bundleID,
		displayName: draft.displayName,
		templateSlug: draft.skillRef.skillSlug,
		templateVersion: 'skill',
		role: PromptRoleEnum.System,
		prompt: draft.prompt,
		isBuiltIn: false,
		createdAt: now,
		modifiedAt: now,
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
	const [skillInstructionPrompts, setSkillInstructionPrompts] = useState<SystemPromptItem[]>([]);
	const initializedFromModelRef = useRef(initializedFromModel);

	useEffect(() => {
		initializedFromModelRef.current = initializedFromModel;
	}, [initializedFromModel]);

	// Initialize includeModelDefault once when model options finish loading,
	// unless a restore/reset has already set the flag. Done in an effect so
	// we don't trigger an extra render-during-render cycle on every mount.
	useEffect(() => {
		if (!modelOptionsLoaded || initializedFromModelRef.current) {
			return;
		}
		initializedFromModelRef.current = true;
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setInitializedFromModelState(true);
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setIncludeModelDefaultState(Boolean(modelDefaultPrompt.trim()));
	}, [modelOptionsLoaded, modelDefaultPrompt]);

	const syntheticPreviousConversationPrompt = useMemo(() => {
		const prompt = restoredConversationSystemPrompt?.trim();
		return prompt ? buildPreviousConversationSystemPromptItem(prompt) : null;
	}, [restoredConversationSystemPrompt]);

	const prompts = useMemo(
		() =>
			syntheticPreviousConversationPrompt
				? [syntheticPreviousConversationPrompt, ...skillInstructionPrompts]
				: [...skillInstructionPrompts],
		[skillInstructionPrompts, syntheticPreviousConversationPrompt]
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

	const addAndSelectInstructionSkillPrompt = useCallback((draft: RenderedInstructionSkillPrompt) => {
		const item = buildInstructionSkillPromptItem({
			...draft,
			identityKey: draft.identityKey ?? `skill-instructions:${buildSkillRefKey(draft.skillRef)}:${Date.now()}`,
		});

		setSkillInstructionPrompts(prev => [...prev.filter(existing => existing.identityKey !== item.identityKey), item]);
		setRawSelectedPromptKeys(prev => {
			const next = prev.includes(item.identityKey) ? prev : [...prev, item.identityKey];
			selectedPromptKeysRef.current = next;
			return next;
		});
	}, []);

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

	const promptsByKeyRef = useRef(promptsByKey);
	useEffect(() => {
		promptsByKeyRef.current = promptsByKey;
	}, [promptsByKey]);

	const prepareAssistantPresetSelections = useCallback(
		async (preset: AssistantPreset): Promise<ComposerSystemPromptPreparedSelection> => {
			const hasIncludeModelSystemPromptSelection = preset.startingIncludeModelSystemPrompt !== undefined;
			const nextIncludeModelSystemPrompt = hasIncludeModelSystemPromptSelection
				? Boolean(preset.startingIncludeModelSystemPrompt)
				: includeModelDefaultRef.current;

			const instructionSkillSelections = (preset.startingSkillSelections ?? []).filter(sel => sel.useAsInstructions);
			const hasInstructionTemplateSelection = instructionSkillSelections.length > 0;
			let nextSelectedPromptKeys = selectedPromptKeysRef.current;

			if (hasInstructionTemplateSelection) {
				const requestedPromptKeys = dedupeStringArray(
					instructionSkillSelections.map(sel => `skill-instructions:${buildSkillRefKey(sel.skillRef)}`)
				);

				const skillOptions = await loadSkillOptions();
				const skillOptionByKey = new Map(skillOptions.map(item => [item.key, item] as const));
				const renderedPromptItems: SystemPromptItem[] = [];

				for (const selection of instructionSkillSelections) {
					const skillKey = buildSkillRefKey(selection.skillRef);
					const option = skillOptionByKey.get(skillKey);

					if (!option || !option.isSelectable) {
						throw new Error(
							option?.availabilityReason ??
								`Instruction skill "${selection.skillRef.bundleID}/${selection.skillRef.skillSlug}#${selection.skillRef.skillID}" is not currently available.`
						);
					}

					const reason = getSkillInstructionPromptEligibilityReason(option.skillDefinition);
					if (reason) {
						throw new Error(
							`${option.skillDefinition.displayName || option.skillDefinition.name || option.skillDefinition.slug}: ${reason}`
						);
					}

					const rendered = await skillStoreAPI.renderSkill(selection.skillRef, {});
					if (rendered.insert !== 'instructions') {
						throw new Error(`Skill "${option.skillDefinition.slug}" did not render as instruction text.`);
					}

					renderedPromptItems.push(
						buildInstructionSkillPromptItem({
							identityKey: `skill-instructions:${skillKey}`,
							displayName:
								option.skillDefinition.displayName || option.skillDefinition.name || option.skillDefinition.slug,
							prompt: rendered.text,
							skillRef: selection.skillRef,
						})
					);
				}

				setSkillInstructionPrompts(prev => {
					const byKey = new Map(prev.map(item => [item.identityKey, item] as const));
					for (const item of renderedPromptItems) {
						byKey.set(item.identityKey, item);
					}
					return [...byKey.values()];
				});
				nextSelectedPromptKeys = requestedPromptKeys;
			}

			return {
				hasIncludeModelSystemPromptSelection,
				nextIncludeModelSystemPrompt,
				hasInstructionTemplateSelection,
				nextSelectedPromptKeys,
			};
		},
		[]
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
		includeModelDefault,
		selectedPromptKeys,
		resolvedSystemPrompt,
		setIncludeModelDefault,
		togglePromptSelection,
		addAndSelectInstructionSkillPrompt,
		clearSelectedPromptSources,
		resetForNewConversation,
		restoreConversationContext,
		prepareAssistantPresetSelections,
		applyPreparedAssistantPresetSelections,
	};
}
