import { useCallback, useMemo, useRef, useState } from 'react';

import type { ArtifactRef } from '@/spec/artifact';
import type { ModelParam } from '@/spec/inference';
import type { SessionConfiguration, SessionInstructionOverride } from '@/spec/session';
import type { SkillRef } from '@/spec/skill';
import { PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY, PREVIOUS_CONVO_SYSTEM_PROMPT_PLUGINID } from '@/spec/model';
import { SessionModelDefaultField } from '@/spec/session';
import { SessionCollectionOverrideMode } from '@/spec/session_base';

import { dedupeStringArray } from '@/lib/obj_utils';

import { skillRefKey } from '@/apis/skill_management';

import type { SystemInstructionSource } from '@/chats/composer/skills/prompt_utils';
import { buildEffectiveSystemPrompt } from '@/chats/composer/skills/prompt_utils';

interface RenderedInstructionSkillSource {
	identityKey?: string;
	displayName: string;
	prompt: string;
	skillRef: ArtifactRef;
	sourceTags?: string[];
}

export interface AgentInstructionSourceDeclaration {
	occurrencePath: string;
	kind: 'skill' | 'text';
	artifact: ArtifactRef;
	displayName: string;
}

const AGENT_INSTRUCTION_SOURCE_PREFIX = 'agent-instruction:';

export interface AgentSystemPromptController {
	modelDefaultPrompt: string;
	instructionSources: SystemInstructionSource[];
	includeModelDefault: boolean;
	selectedInstructionSourceKeys: string[];
	selectedInstructionSkillRefs: SkillRef[];
	resolvedSystemPrompt: string;
	selectedSessionInstructionOverrides: SessionInstructionOverride[];
	additionalSessionInstructionText: string;

	setIncludeModelDefault(next: boolean): void;
	toggleInstructionSource(identityKey: string): void;
	upsertAndSelectInstructionSource(source: SystemInstructionSource): void;
	removeInstructionSource(identityKey: string): void;
	addAndSelectInstructionSkillSource(source: RenderedInstructionSkillSource): void;

	replaceAgentInstructionSources(sources: AgentInstructionSourceDeclaration[]): void;
	clearAgentInstructionSources(): void;
	setAgentInstructionSourceSelected(occurrencePath: string, selected: boolean): void;
	clearSessionInstructionSources(): void;
	clearInstructionSources(): void;
	clearAllSystemPromptState(): void;
	resetForNewConversation(modelDefaultPrompt: string): void;
	restoreConversationContext(modelDefaultPrompt: string, modelParam?: ModelParam): void;
	restoreSessionConfiguration(configuration: SessionConfiguration): void;
}

type IncludeModelDefaultPreference = boolean | 'auto';

function previousConversationPrompt(prompt: string): SystemInstructionSource {
	return {
		identityKey: PREVIOUS_CONVO_SYSTEM_PROMPT_IDENTITY_KEY,
		sourceKind: 'restored-conversation',
		pluginID: PREVIOUS_CONVO_SYSTEM_PROMPT_PLUGINID,
		pluginDisplayName: 'Conversation',
		pluginSlug: 'conversation',
		displayName: 'Previous conversation prompt',
		sourceSlug: 'previous-conversation-prompt',
		text: prompt,
		isBuiltIn: true,
	};
}

function normalizeSkillSourceTags(sourceTags?: string[] | null): string[] {
	const seen = new Set<string>();

	for (const sourceTag of sourceTags ?? []) {
		const value = sourceTag.trim();
		if (value) {
			seen.add(value);
		}
	}

	return [...seen];
}

function instructionSkillSource(source: RenderedInstructionSkillSource): SystemInstructionSource {
	return {
		identityKey: source.identityKey ?? `skill-instructions:${skillRefKey(source.skillRef)}`,
		sourceKind: 'skill',
		pluginID: source.skillRef.rootID,
		pluginDisplayName: 'Skill instructions',
		pluginSlug: source.skillRef.rootID,
		displayName: source.displayName,
		sourceSlug: source.skillRef.artifactID,
		text: source.prompt,
		sourceTags: normalizeSkillSourceTags(source.sourceTags),
		isBuiltIn: false,
		skillRef: source.skillRef,
	};
}

function agentInstructionSource(source: AgentInstructionSourceDeclaration): SystemInstructionSource {
	return {
		identityKey: `${AGENT_INSTRUCTION_SOURCE_PREFIX}${source.occurrencePath}`,
		sourceKind: 'agent',
		pluginID: 'agent',
		pluginDisplayName: 'Agent',
		pluginSlug: 'agent',
		displayName: source.displayName,
		sourceSlug: source.artifact.artifactID,
		// Agent instruction content remains backend-owned. The empty text is
		// deliberate: this source is provenance and exclusion UI, never a
		// frontend-authored Session instruction payload.
		text: '',
		isBuiltIn: false,
		...(source.kind === 'skill' ? { skillRef: source.artifact } : {}),
		agentOccurrencePath: source.occurrencePath,
		agentArtifact: source.artifact,
	};
}

function agentInstructionSourceKey(occurrencePath: string): string {
	return `${AGENT_INSTRUCTION_SOURCE_PREFIX}${occurrencePath}`;
}

function sessionInstructionSources(configuration: SessionConfiguration): SystemInstructionSource[] {
	const sources = new Map<string, SystemInstructionSource>();
	let replacementStarted = false;

	for (const override of configuration.instructions ?? []) {
		switch (override.mode) {
			case SessionCollectionOverrideMode.Inherit:
			case SessionCollectionOverrideMode.Remove:
				continue;

			case SessionCollectionOverrideMode.Replace:
				if (!replacementStarted) {
					sources.clear();
					replacementStarted = true;
				}
				break;

			case SessionCollectionOverrideMode.Add:
				break;
		}

		const text = override.value?.text?.trim();
		if (!text) {
			continue;
		}

		sources.set(override.key, {
			identityKey: `session-instruction:${override.key}`,
			sourceKind: 'session',
			pluginID: 'session',
			pluginDisplayName: 'Session',
			pluginSlug: 'session',
			displayName: override.value?.label?.trim() || 'Session instructions',
			sourceSlug: override.key,
			text,
			isBuiltIn: false,
			sessionOverride: structuredClone(override),
		});
	}

	return [...sources.values()];
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

	const selectedInstructionSkillRefs = useMemo(() => {
		const selected = new Set(selectedInstructionSourceKeys);
		const seen = new Set<string>();
		const refs: SkillRef[] = [];

		for (const source of sources) {
			if (source.sourceKind !== 'skill' || !source.skillRef || !selected.has(source.identityKey)) {
				continue;
			}

			const key = skillRefKey(source.skillRef);
			if (!seen.has(key)) {
				seen.add(key);
				refs.push({ rootID: source.skillRef.rootID, artifactID: source.skillRef.artifactID });
			}
		}
		return refs;
	}, [selectedInstructionSourceKeys, sources]);

	const selectedSessionInstructionOverrides = useMemo(() => {
		const selectedKeys = new Set(selectedInstructionSourceKeys);

		return sources.flatMap(source => {
			if (source.sourceKind !== 'session' || !source.sessionOverride || !selectedKeys.has(source.identityKey)) {
				return [];
			}

			return [structuredClone(source.sessionOverride)];
		});
	}, [selectedInstructionSourceKeys, sources]);

	const nonSessionInstructionSourcesByKey = useMemo(
		() =>
			new Map(
				sources
					.filter(source => source.sourceKind !== 'session' && source.sourceKind !== 'agent')
					.map(source => [source.identityKey, source])
			),
		[sources]
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

	const additionalSessionInstructionText = useMemo(
		() =>
			buildEffectiveSystemPrompt({
				modelDefaultPrompt: '',
				includeModelDefault: false,
				selectedInstructionSourceKeys: selectedInstructionSourceKeys.filter(key =>
					nonSessionInstructionSourcesByKey.has(key)
				),
				instructionSourcesByKey: nonSessionInstructionSourcesByKey,
			}),
		[nonSessionInstructionSourcesByKey, selectedInstructionSourceKeys]
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

	const replaceAgentInstructionSources = useCallback(
		(nextSources: AgentInstructionSourceDeclaration[]) => {
			const byKey = new Map<string, SystemInstructionSource>();

			for (const source of nextSources) {
				const normalized = agentInstructionSource(source);
				byKey.set(normalized.identityKey, normalized);
			}

			const normalizedSources = [...byKey.values()];
			const normalizedKeys = normalizedSources.map(source => source.identityKey);

			setDynamicSources(previous => [
				...normalizedSources,
				...previous.filter(source => source.sourceKind !== 'agent'),
			]);

			setSelectedKeys([
				...normalizedKeys,
				...selectedKeysRef.current.filter(key => !key.startsWith(AGENT_INSTRUCTION_SOURCE_PREFIX)),
			]);
		},
		[setSelectedKeys]
	);

	const clearAgentInstructionSources = useCallback(() => {
		setDynamicSources(previous => previous.filter(source => source.sourceKind !== 'agent'));
		setSelectedKeys(selectedKeysRef.current.filter(key => !key.startsWith(AGENT_INSTRUCTION_SOURCE_PREFIX)));
	}, [setSelectedKeys]);

	const setAgentInstructionSourceSelected = useCallback(
		(occurrencePath: string, selected: boolean) => {
			const key = agentInstructionSourceKey(occurrencePath);
			setSelectedKeys(
				selected ? [...selectedKeysRef.current, key] : selectedKeysRef.current.filter(candidate => candidate !== key)
			);
		},
		[setSelectedKeys]
	);

	const clearSessionInstructionSources = useCallback(() => {
		const retained = selectedKeysRef.current.filter(key => {
			const source = sourceByKey.get(key);
			return source?.sourceKind === 'agent' || source?.sourceKind === 'restored-conversation';
		});
		setSelectedKeys(retained);
	}, [setSelectedKeys, sourceByKey]);

	const clearInstructionSources = useCallback(() => {
		clearSessionInstructionSources();
	}, [clearSessionInstructionSources]);

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

	const restoreSessionConfiguration = useCallback(
		(configuration: SessionConfiguration) => {
			const clearModelDefault = configuration.modelOverrides.clear?.includes(SessionModelDefaultField.SystemPrompt);
			const s = sessionInstructionSources(configuration);

			setRestoredPrompt(null);
			setIncludePreference(clearModelDefault ? false : 'auto');
			setDynamicSources(s);
			setSelectedKeys(s.map(source => source.identityKey));
		},
		[setSelectedKeys]
	);

	return {
		modelDefaultPrompt,
		instructionSources: sources,
		includeModelDefault,
		selectedInstructionSourceKeys,
		selectedInstructionSkillRefs,
		resolvedSystemPrompt,
		selectedSessionInstructionOverrides,
		additionalSessionInstructionText,
		setIncludeModelDefault,
		toggleInstructionSource,
		upsertAndSelectInstructionSource,
		removeInstructionSource,
		addAndSelectInstructionSkillSource,
		replaceAgentInstructionSources,
		clearAgentInstructionSources,
		setAgentInstructionSourceSelected,
		clearSessionInstructionSources,
		clearInstructionSources,
		clearAllSystemPromptState,
		resetForNewConversation,
		restoreConversationContext,
		restoreSessionConfiguration,
	};
}
