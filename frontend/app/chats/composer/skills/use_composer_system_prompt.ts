import { useCallback, useMemo, useRef, useState } from 'react';

import type { AssistantPreset } from '@/spec/assistantpreset';
import type { ModelParam } from '@/spec/inference';
import { PREVIOUS_CONVO_SYSTEM_PROMPT_BUNDLEID, PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY } from '@/spec/modelpreset';

import { dedupeStringArray } from '@/lib/obj_utils';

import { skillStoreAPI } from '@/apis/baseapi';

import { loadSkillOptions } from '@/assistantpresets/lib/assistant_preset_catalog';
import { buildSkillRefKey } from '@/assistantpresets/lib/assistant_preset_utils';
import type { AssistantPresetPreparedApplication } from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import type { SystemInstructionSource } from '@/chats/composer/skills/prompt_utils';
import { buildEffectiveSystemPrompt } from '@/chats/composer/skills/prompt_utils';
import { getSkillInstructionPromptEligibilityReason } from '@/skills/lib/skill_artifact_utils';

interface ComposerSystemPromptPreparedSelection {
	hasIncludeModelSystemPromptSelection: boolean;
	nextIncludeModelSystemPrompt: boolean;
	hasInstructionSourceSelection: boolean;
	nextSelectedInstructionSourceKeys: string[];
	preparedInstructionSources: SystemInstructionSource[];
}

interface RenderedInstructionSkillSource {
	identityKey?: string;
	displayName: string;
	prompt: string;
	skillRef: { bundleID: string; skillSlug: string; skillID: string };
}

type IncludeModelDefaultPreference = boolean | 'auto';

export interface ComposerSystemPromptController {
	modelDefaultPrompt: string;
	instructionSources: SystemInstructionSource[];
	includeModelDefault: boolean;
	selectedInstructionSourceKeys: string[];
	resolvedSystemPrompt: string;
	setIncludeModelDefault: (next: boolean) => void;
	toggleInstructionSource: (identityKey: string) => void;
	addAndSelectInstructionSkillSource: (draft: RenderedInstructionSkillSource) => void;
	clearInstructionSources: () => void;
	clearAllSystemPromptState: () => void;
	resetForNewConversation: (modelDefaultPrompt: string) => void;
	restoreConversationContext: (modelDefaultPrompt: string, modelParam?: ModelParam) => void;
	prepareAssistantPresetInstructionSources: (preset: AssistantPreset) => Promise<ComposerSystemPromptPreparedSelection>;
	applyPreparedAssistantPresetInstructionSources: (
		prepared: Pick<
			AssistantPresetPreparedApplication,
			| 'hasIncludeModelSystemPromptSelection'
			| 'nextIncludeModelSystemPrompt'
			| 'hasInstructionSourceSelection'
			| 'nextSelectedInstructionSourceKeys'
			| 'preparedInstructionSources'
		>
	) => void;
}

function buildPreviousConversationSystemPromptItem(prompt: string): SystemInstructionSource {
	return {
		identityKey: PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY,
		sourceKind: 'restored-conversation',
		bundleID: PREVIOUS_CONVO_SYSTEM_PROMPT_BUNDLEID,
		bundleDisplayName: 'Conversation',
		bundleSlug: 'conversation',
		displayName: 'Previous conversation prompt',
		sourceSlug: 'previous-conversation-prompt',
		text: prompt,
		isBuiltIn: true,
	};
}

function buildInstructionSkillPromptItem(draft: RenderedInstructionSkillSource): SystemInstructionSource {
	const stableKey = `skill-instructions:${buildSkillRefKey(draft.skillRef)}`;

	return {
		identityKey: draft.identityKey ?? stableKey,
		sourceKind: 'skill',
		bundleID: draft.skillRef.bundleID,
		bundleDisplayName: 'Skill instructions',
		bundleSlug: draft.skillRef.bundleID,
		displayName: draft.displayName,
		sourceSlug: draft.skillRef.skillSlug,
		text: draft.prompt,
		isBuiltIn: false,
	};
}

function deriveRestoredPromptSelectionState(
	modelDefaultPrompt: string,
	modelParam?: ModelParam
): {
	restoredConversationSystemPrompt: string | null;
	includeModelDefaultPreference: IncludeModelDefaultPreference;
	selectedInstructionSourceKeys: string[];
} {
	if (!modelParam) {
		return {
			restoredConversationSystemPrompt: null,
			includeModelDefaultPreference: modelDefaultPrompt.trim() ? true : 'auto',
			selectedInstructionSourceKeys: [],
		};
	}

	const restoredSystemPrompt = modelParam.systemPrompt?.trim() ?? '';
	if (restoredSystemPrompt) {
		return {
			restoredConversationSystemPrompt: restoredSystemPrompt,
			includeModelDefaultPreference: false,
			selectedInstructionSourceKeys: [PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY],
		};
	}

	return {
		restoredConversationSystemPrompt: null,
		includeModelDefaultPreference: false,
		selectedInstructionSourceKeys: [],
	};
}

export function useComposerSystemPrompt(args: {
	modelDefaultPrompt: string;
	modelOptionsLoaded: boolean;
}): ComposerSystemPromptController {
	const { modelDefaultPrompt, modelOptionsLoaded } = args;

	const [rawSelectedInstructionSourceKeys, setRawSelectedInstructionSourceKeys] = useState<string[]>([]);
	const [includeModelDefaultPreference, setIncludeModelDefaultPreference] =
		useState<IncludeModelDefaultPreference>('auto');
	const [restoredConversationSystemPrompt, setRestoredConversationSystemPrompt] = useState<string | null>(null);
	const [skillInstructionSources, setSkillInstructionSources] = useState<SystemInstructionSource[]>([]);
	const includeModelDefault =
		includeModelDefaultPreference === 'auto'
			? modelOptionsLoaded && Boolean(modelDefaultPrompt.trim())
			: includeModelDefaultPreference;

	const syntheticPreviousConversationPrompt = useMemo(() => {
		const prompt = restoredConversationSystemPrompt?.trim();
		return prompt ? buildPreviousConversationSystemPromptItem(prompt) : null;
	}, [restoredConversationSystemPrompt]);

	const instructionSources = useMemo(
		() =>
			syntheticPreviousConversationPrompt
				? [syntheticPreviousConversationPrompt, ...skillInstructionSources]
				: [...skillInstructionSources],
		[skillInstructionSources, syntheticPreviousConversationPrompt]
	);

	const instructionSourcesByKey = useMemo(
		() => new Map(instructionSources.map(item => [item.identityKey, item])),
		[instructionSources]
	);

	const selectedInstructionSourceKeys = useMemo(
		() => rawSelectedInstructionSourceKeys.filter(key => instructionSourcesByKey.has(key)),
		[instructionSourcesByKey, rawSelectedInstructionSourceKeys]
	);

	const selectedInstructionSourceKeysRef = useRef(selectedInstructionSourceKeys);

	const setPromptSelectionState = useCallback(
		(
			nextRestoredConversationSystemPrompt: string | null,
			nextIncludeModelDefaultPreference: IncludeModelDefaultPreference,
			nextSelectedPromptKeys: string[]
		) => {
			setRestoredConversationSystemPrompt(nextRestoredConversationSystemPrompt);
			setIncludeModelDefaultPreference(nextIncludeModelDefaultPreference);

			const nextKeys = dedupeStringArray(nextSelectedPromptKeys);
			selectedInstructionSourceKeysRef.current = nextKeys;
			setRawSelectedInstructionSourceKeys(nextKeys);
		},
		[]
	);

	const resolvedSystemPrompt = useMemo(
		() =>
			buildEffectiveSystemPrompt({
				modelDefaultPrompt,
				includeModelDefault,
				selectedInstructionSourceKeys,
				instructionSourcesByKey,
			}),
		[includeModelDefault, instructionSourcesByKey, modelDefaultPrompt, selectedInstructionSourceKeys]
	);

	const setIncludeModelDefault = useCallback((next: boolean) => {
		setIncludeModelDefaultPreference(next);
	}, []);

	const toggleInstructionSource = useCallback((identityKey: string) => {
		setRawSelectedInstructionSourceKeys(prev => {
			const next = prev.includes(identityKey) ? prev.filter(item => item !== identityKey) : [...prev, identityKey];
			selectedInstructionSourceKeysRef.current = next;
			return next;
		});
	}, []);

	const addAndSelectInstructionSkillSource = useCallback((draft: RenderedInstructionSkillSource) => {
		const item = buildInstructionSkillPromptItem(draft);

		setSkillInstructionSources(prev => {
			const byKey = new Map(prev.map(existing => [existing.identityKey, existing] as const));
			// oxlint-disable-next-line unicorn/no-immediate-mutation
			byKey.set(item.identityKey, item);
			return [...byKey.values()];
		});
		setRawSelectedInstructionSourceKeys(prev => {
			const next = dedupeStringArray([...prev, item.identityKey]);
			selectedInstructionSourceKeysRef.current = next;
			return next;
		});
	}, []);

	const clearInstructionSources = useCallback(() => {
		setPromptSelectionState(null, false, []);
	}, [setPromptSelectionState]);

	const clearAllSystemPromptState = useCallback(() => {
		setSkillInstructionSources([]);
		setPromptSelectionState(null, false, []);
	}, [setPromptSelectionState]);

	const resetForNewConversation = useCallback(
		(nextModelDefaultPrompt: string) => {
			setSkillInstructionSources([]);
			setPromptSelectionState(null, nextModelDefaultPrompt.trim() ? true : 'auto', []);
		},
		[setPromptSelectionState]
	);

	const restoreConversationContext = useCallback(
		(nextModelDefaultPrompt: string, modelParam?: ModelParam) => {
			setSkillInstructionSources([]);
			const restoredPromptState = deriveRestoredPromptSelectionState(nextModelDefaultPrompt, modelParam);
			setPromptSelectionState(
				restoredPromptState.restoredConversationSystemPrompt,
				restoredPromptState.includeModelDefaultPreference,
				restoredPromptState.selectedInstructionSourceKeys
			);
		},
		[setPromptSelectionState]
	);

	const prepareAssistantPresetInstructionSources = useCallback(
		async (preset: AssistantPreset): Promise<ComposerSystemPromptPreparedSelection> => {
			const hasIncludeModelSystemPromptSelection = preset.startingIncludeModelSystemPrompt !== undefined;
			const nextIncludeModelSystemPrompt = hasIncludeModelSystemPromptSelection
				? Boolean(preset.startingIncludeModelSystemPrompt)
				: includeModelDefault;

			const instructionSkillSelections = (preset.startingSkillSelections ?? []).filter(sel => sel.useAsInstructions);
			const hasInstructionSourceSelection = instructionSkillSelections.length > 0;
			let nextSelectedInstructionSourceKeys = selectedInstructionSourceKeysRef.current;
			const preparedInstructionSources: SystemInstructionSource[] = [];

			if (hasInstructionSourceSelection) {
				const requestedPromptKeys = dedupeStringArray(
					instructionSkillSelections.map(sel => `skill-instructions:${buildSkillRefKey(sel.skillRef)}`)
				);

				const skillOptions = await loadSkillOptions({ force: true });
				const skillOptionByKey = new Map(skillOptions.map(item => [item.key, item] as const));

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

					preparedInstructionSources.push(
						buildInstructionSkillPromptItem({
							identityKey: `skill-instructions:${skillKey}`,
							displayName:
								option.skillDefinition.displayName || option.skillDefinition.name || option.skillDefinition.slug,
							prompt: rendered.text,
							skillRef: selection.skillRef,
						})
					);
				}
				nextSelectedInstructionSourceKeys = requestedPromptKeys;
			}

			return {
				hasIncludeModelSystemPromptSelection,
				nextIncludeModelSystemPrompt,
				hasInstructionSourceSelection,
				nextSelectedInstructionSourceKeys,
				preparedInstructionSources,
			};
		},
		[includeModelDefault]
	);

	const applyPreparedAssistantPresetInstructionSources = useCallback(
		(
			prepared: Pick<
				AssistantPresetPreparedApplication,
				| 'hasIncludeModelSystemPromptSelection'
				| 'nextIncludeModelSystemPrompt'
				| 'hasInstructionSourceSelection'
				| 'nextSelectedInstructionSourceKeys'
				| 'preparedInstructionSources'
			>
		) => {
			if (prepared.hasIncludeModelSystemPromptSelection) {
				setIncludeModelDefaultPreference(prepared.nextIncludeModelSystemPrompt);
			}

			if (prepared.hasInstructionSourceSelection) {
				setSkillInstructionSources(prev => {
					const byKey = new Map(prev.map(item => [item.identityKey, item] as const));
					for (const item of prepared.preparedInstructionSources) {
						byKey.set(item.identityKey, item);
					}
					return [...byKey.values()];
				});
				selectedInstructionSourceKeysRef.current = [...prepared.nextSelectedInstructionSourceKeys];
				setRawSelectedInstructionSourceKeys(prepared.nextSelectedInstructionSourceKeys);
			}
		},
		[]
	);

	return {
		modelDefaultPrompt,
		instructionSources,
		includeModelDefault,
		selectedInstructionSourceKeys,
		resolvedSystemPrompt,
		setIncludeModelDefault,
		toggleInstructionSource,
		addAndSelectInstructionSkillSource,
		clearInstructionSources,
		clearAllSystemPromptState,
		resetForNewConversation,
		restoreConversationContext,
		prepareAssistantPresetInstructionSources,
		applyPreparedAssistantPresetInstructionSources,
	};
}
