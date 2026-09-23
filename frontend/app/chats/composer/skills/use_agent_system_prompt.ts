import { useCallback, useMemo, useRef, useState } from 'react';

import type { ArtifactRef } from '@/spec/artifact';
import type { ModelParam } from '@/spec/inference';
import { PREVIOUS_CONVO_SYSTEM_PROMPT_BUNDLEID, PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY } from '@/spec/modelpreset';

import { dedupeStringArray } from '@/lib/obj_utils';

import type { SystemInstructionSource } from '@/chats/composer/skills/prompt_utils';
import { buildEffectiveSystemPrompt } from '@/chats/composer/skills/prompt_utils';
import { normalizeSkillSourceTags } from '@/skills/lib/skill_artifact_utils';
import { skillRefKey } from '@/skills/lib/skill_identity_utils';

interface RenderedInstructionSkillSource {
	identityKey?: string;
	displayName: string;
	prompt: string;
	skillRef: ArtifactRef;
	sourceTags?: string[];
}

export interface AgentSystemPromptController {
	modelDefaultPrompt: string;
	instructionSources: SystemInstructionSource[];
	includeModelDefault: boolean;
	selectedInstructionSourceKeys: string[];
	resolvedSystemPrompt: string;

	setIncludeModelDefault(next: boolean): void;
	toggleInstructionSource(identityKey: string): void;
	upsertAndSelectInstructionSource(source: SystemInstructionSource): void;
	removeInstructionSource(identityKey: string): void;
	addAndSelectInstructionSkillSource(source: RenderedInstructionSkillSource): void;

	clearInstructionSources(): void;
	clearAllSystemPromptState(): void;
	resetForNewConversation(modelDefaultPrompt: string): void;
	restoreConversationContext(modelDefaultPrompt: string, modelParam?: ModelParam): void;
}

type IncludeModelDefaultPreference = boolean | 'auto';

function previousConversationPrompt(prompt: string): SystemInstructionSource {
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

function instructionSkillSource(source: RenderedInstructionSkillSource): SystemInstructionSource {
	return {
		identityKey: source.identityKey ?? `skill-instructions:${skillRefKey(source.skillRef)}`,
		sourceKind: 'skill',
		bundleID: source.skillRef.rootID,
		bundleDisplayName: 'Skill instructions',
		bundleSlug: source.skillRef.rootID,
		displayName: source.displayName,
		sourceSlug: source.skillRef.artifactID,
		text: source.prompt,
		sourceTags: normalizeSkillSourceTags(source.sourceTags),
		isBuiltIn: false,
		skillRef: source.skillRef,
	};
}

function restoredPromptState(modelDefaultPrompt: string, modelParam?: ModelParam) {
	const restored = modelParam?.systemPrompt?.trim() ?? '';

	if (restored) {
		return {
			prompt: restored,
			includeModelDefault: false as IncludeModelDefaultPreference,
			selectedKeys: [PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY],
		};
	}

	return {
		prompt: null,
		includeModelDefault: modelDefaultPrompt.trim() ? true : ('auto' as IncludeModelDefaultPreference),
		selectedKeys: [],
	};
}

export function useAgentSystemPrompt(args: {
	modelDefaultPrompt: string;
	modelOptionsLoaded: boolean;
}): AgentSystemPromptController {
	const { modelDefaultPrompt, modelOptionsLoaded } = args;

	const [includePreference, setIncludePreference] = useState<IncludeModelDefaultPreference>('auto');
	const [restoredPrompt, setRestoredPrompt] = useState<string | null>(null);
	const [dynamicSources, setDynamicSources] = useState<SystemInstructionSource[]>([]);
	const [rawSelectedKeys, setRawSelectedKeys] = useState<string[]>([]);
	const selectedKeysRef = useRef<string[]>([]);

	const includeModelDefault =
		includePreference === 'auto' ? modelOptionsLoaded && Boolean(modelDefaultPrompt.trim()) : includePreference;

	const sources = useMemo(() => {
		const restored = restoredPrompt?.trim() ? previousConversationPrompt(restoredPrompt.trim()) : null;
		return restored ? [restored, ...dynamicSources] : dynamicSources;
	}, [dynamicSources, restoredPrompt]);

	const sourceByKey = useMemo(() => new Map(sources.map(source => [source.identityKey, source] as const)), [sources]);

	const selectedInstructionSourceKeys = useMemo(
		() => rawSelectedKeys.filter(key => sourceByKey.has(key)),
		[rawSelectedKeys, sourceByKey]
	);

	const resolvedSystemPrompt = useMemo(
		() =>
			buildEffectiveSystemPrompt({
				modelDefaultPrompt,
				includeModelDefault,
				selectedInstructionSourceKeys,
				instructionSourcesByKey: sourceByKey,
			}),
		[includeModelDefault, modelDefaultPrompt, selectedInstructionSourceKeys, sourceByKey]
	);

	const setSelectedKeys = useCallback((next: string[]) => {
		const deduped = dedupeStringArray(next);
		selectedKeysRef.current = deduped;
		setRawSelectedKeys(deduped);
	}, []);

	const setIncludeModelDefault = useCallback((next: boolean) => {
		setIncludePreference(next);
	}, []);

	const toggleInstructionSource = useCallback((identityKey: string) => {
		setRawSelectedKeys(previous => {
			const next = previous.includes(identityKey)
				? previous.filter(key => key !== identityKey)
				: [...previous, identityKey];

			selectedKeysRef.current = next;
			return next;
		});
	}, []);

	const upsertAndSelectInstructionSource = useCallback(
		(source: SystemInstructionSource) => {
			setDynamicSources(previous => {
				const byKey = new Map(previous.map(item => [item.identityKey, item] as const));
				// oxlint-disable-next-line unicorn/no-immediate-mutation
				byKey.set(source.identityKey, source);
				return [...byKey.values()];
			});

			setSelectedKeys([...selectedKeysRef.current, source.identityKey]);
		},
		[setSelectedKeys]
	);

	const removeInstructionSource = useCallback(
		(identityKey: string) => {
			setDynamicSources(previous => previous.filter(source => source.identityKey !== identityKey));
			setSelectedKeys(selectedKeysRef.current.filter(key => key !== identityKey));
		},
		[setSelectedKeys]
	);

	const addAndSelectInstructionSkillSource = useCallback(
		(source: RenderedInstructionSkillSource) => {
			upsertAndSelectInstructionSource(instructionSkillSource(source));
		},
		[upsertAndSelectInstructionSource]
	);

	const clearInstructionSources = useCallback(() => {
		setRestoredPrompt(null);
		setIncludePreference(false);
		setSelectedKeys([]);
	}, [setSelectedKeys]);

	const clearAllSystemPromptState = useCallback(() => {
		setDynamicSources([]);
		setRestoredPrompt(null);
		setIncludePreference(false);
		setSelectedKeys([]);
	}, [setSelectedKeys]);

	const resetForNewConversation = useCallback(
		(nextModelDefaultPrompt: string) => {
			setDynamicSources([]);
			setRestoredPrompt(null);
			setIncludePreference(nextModelDefaultPrompt.trim() ? true : 'auto');
			setSelectedKeys([]);
		},
		[setSelectedKeys]
	);

	const restoreConversationContext = useCallback(
		(nextModelDefaultPrompt: string, modelParam?: ModelParam) => {
			const restored = restoredPromptState(nextModelDefaultPrompt, modelParam);

			setDynamicSources([]);
			setRestoredPrompt(restored.prompt);
			setIncludePreference(restored.includeModelDefault);
			setSelectedKeys(restored.selectedKeys);
		},
		[setSelectedKeys]
	);

	return {
		modelDefaultPrompt,
		instructionSources: sources,
		includeModelDefault,
		selectedInstructionSourceKeys,
		resolvedSystemPrompt,
		setIncludeModelDefault,
		toggleInstructionSource,
		upsertAndSelectInstructionSource,
		removeInstructionSource,
		addAndSelectInstructionSkillSource,
		clearInstructionSources,
		clearAllSystemPromptState,
		resetForNewConversation,
		restoreConversationContext,
	};
}
