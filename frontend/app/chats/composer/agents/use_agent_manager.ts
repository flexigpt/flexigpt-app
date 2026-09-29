import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { ArtifactRef } from '@/spec/artifact';
import type { UIModelOption } from '@/spec/model';

import { getErrorMessage } from '@/lib/error_utils';

import type { AgentCatalogOption, AgentPreparedInstructionSource, PreparedAgentStarter } from '@/apis/agent_management';
import { agentManagementAPI } from '@/apis/baseapi';

import type { AgentSystemPromptController } from '@/chats/composer/skills/use_agent_system_prompt';
import { modelRefEqual } from '@/models/lib/document';

interface AgentComposerContext {
	selectedModel: UIModelOption;
	allOptions: UIModelOption[];
	modelOptionsLoaded: boolean;
	handleSetSelectedModel: Dispatch<SetStateAction<UIModelOption>>;
}

interface AgentRuntimeApplier {
	applyAgentRuntime(recipe: PreparedAgentStarter): void;
}

type PendingDefaultAction = 'ensure' | 'reset' | 'track';

const BASE_AGENT_NAME = 'base';
const NO_SELECTABLE_AGENT_ERROR = 'No selectable Agent is available.';

export interface AgentManagerState {
	agentOptions: AgentCatalogOption[];
	loading: boolean;
	error: string | null;
	actionError: string | null;
	isApplying: boolean;

	baseAgentKey: string | null;
	defaultAgentKey: string | null;
	isDefaultAgentSelected: boolean;

	selectedAgentKey: string | null;
	selectedAgent: AgentCatalogOption | null;
	preparedStarter: PreparedAgentStarter | null;
	isTrackingOnly: boolean;

	refreshAgents(): Promise<void>;
	selectAgent(agentRef: ArtifactRef): Promise<boolean>;
	ensureDefaultAgent(): Promise<boolean>;
	resetToDefaultAgent(): Promise<boolean>;
	reapplySelectedAgent(): Promise<boolean>;
	trackDefaultAgentWithoutApplying(): Promise<boolean>;
	clearAgentTracking(): void;
}

function agentKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

function agentInstructionSourceKey(agent: ArtifactRef, source: AgentPreparedInstructionSource): string {
	return `agent:${agentKey(agent)}:${source.kind}:${agentKey(source.artifact)}`;
}

function getRecipeError(recipe: PreparedAgentStarter): string | undefined {
	const blocking = recipe.issues.filter(issue => issue.severity === 'error');

	if (blocking.length === 0) {
		return undefined;
	}

	return blocking.map(issue => `${issue.path ? `${issue.path}: ` : ''}${issue.message}`).join('\n');
}

export function useAgentManager(
	context: AgentComposerContext,
	runtimeCompiler: AgentRuntimeApplier,
	systemPrompt: AgentSystemPromptController
): AgentManagerState {
	const [agentOptions, setAgentOptions] = useState<AgentCatalogOption[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [actionError, setActionError] = useState<string | null>(null);
	const [isApplying, setIsApplying] = useState(false);
	const [isTrackingOnly, setIsTrackingOnly] = useState(false);
	const [selectedAgentKey, setSelectedAgentKey] = useState<string | null>(null);
	const [preparedStarter, setPreparedStarter] = useState<PreparedAgentStarter | null>(null);

	const selectedAgentKeyRef = useRef<string | null>(null);
	const isApplyingRef = useRef(false);
	const applyRequestSeqRef = useRef(0);
	const catalogRequestSeqRef = useRef(0);
	const pendingDefaultActionRef = useRef<PendingDefaultAction | null>(null);
	const initialDefaultRequestRef = useRef(false);
	const autoEnsureCatalogKeyRef = useRef<string | null>(null);
	const appliedInstructionSourceKeysRef = useRef<Set<string>>(new Set());

	const setTrackedAgent = useCallback(
		(aKey: string | null, starter: PreparedAgentStarter | null, trackingOnly = false) => {
			selectedAgentKeyRef.current = aKey;
			setSelectedAgentKey(aKey);
			setPreparedStarter(starter);
			setIsTrackingOnly(trackingOnly);
		},
		[]
	);

	const cancelActiveAgentRequest = useCallback(() => {
		applyRequestSeqRef.current += 1;
		isApplyingRef.current = false;
		setIsApplying(false);
	}, []);

	const loadAgentOptions = useCallback(async (force = false) => {
		const requestSeq = catalogRequestSeqRef.current + 1;
		catalogRequestSeqRef.current = requestSeq;
		setLoading(true);
		setError(null);

		try {
			const nextOptions = await agentManagementAPI.listAgentCatalogOptions(force);
			if (catalogRequestSeqRef.current !== requestSeq) {
				return;
			}

			setAgentOptions(nextOptions);
		} catch (loadError) {
			if (catalogRequestSeqRef.current !== requestSeq) {
				return;
			}

			setAgentOptions([]);
			setError(getErrorMessage(loadError, 'Failed to load Agents.'));
		} finally {
			if (catalogRequestSeqRef.current === requestSeq) {
				setLoading(false);
			}
		}
	}, []);

	const refreshAgents = useCallback(async () => {
		autoEnsureCatalogKeyRef.current = null;
		await loadAgentOptions(true);
	}, [loadAgentOptions]);

	useEffect(() => {
		// oxlint-disable-next-line react/set-state-in-effect
		void loadAgentOptions(false);
	}, [loadAgentOptions]);

	const selectedAgent = useMemo(
		() => agentOptions.find(option => option.key === selectedAgentKey) ?? null,
		[agentOptions, selectedAgentKey]
	);

	const baseAgent = useMemo(
		() =>
			agentOptions.find(option => option.agent.builtIn && option.agent.name.trim().toLowerCase() === BASE_AGENT_NAME) ??
			null,
		[agentOptions]
	);
	const defaultAgent = useMemo(
		() => (baseAgent?.isSelectable ? baseAgent : (agentOptions.find(option => option.isSelectable) ?? null)),
		[agentOptions, baseAgent]
	);
	const isDefaultAgentSelected = selectedAgent !== null && selectedAgent.key === defaultAgent?.key;

	const hasTrackedAgent = useCallback(() => {
		const trackedKey = selectedAgentKeyRef.current;
		return trackedKey !== null && agentOptions.some(option => option.key === trackedKey);
	}, [agentOptions]);

	const selectAgent = useCallback(
		async (agentRef: ArtifactRef): Promise<boolean> => {
			if (isApplyingRef.current) {
				return false;
			}

			const option = agentOptions.find(value => value.key === agentKey(agentRef));

			if (!option) {
				setActionError('The selected Agent is no longer available.');
				return false;
			}

			if (!option.isSelectable) {
				setActionError(option.availabilityReason ?? 'The selected Agent is not currently available.');
				return false;
			}

			if (!context.modelOptionsLoaded) {
				setActionError('Model options are still loading. Try again in a moment.');
				return false;
			}

			const requestSeq = applyRequestSeqRef.current + 1;
			applyRequestSeqRef.current = requestSeq;
			pendingDefaultActionRef.current = null;
			setActionError(null);
			isApplyingRef.current = true;
			setIsApplying(true);

			try {
				const recipe = await agentManagementAPI.prepareAgentStarter(agentRef);
				if (applyRequestSeqRef.current !== requestSeq) {
					return false;
				}

				const recipeError = getRecipeError(recipe);

				if (recipeError) {
					setActionError(recipeError);
					return false;
				}

				let nextModel = context.selectedModel;

				if (recipe.modelRef) {
					const resolvedModel = context.allOptions.find(o => modelRefEqual(o.model, recipe.modelRef));

					if (!resolvedModel) {
						setActionError(
							`Agent model "${recipe.modelRef.rootID}/${recipe.modelRef.artifactID}" is not currently selectable in Composer.`
						);
						return false;
					}

					nextModel = resolvedModel;
				}

				for (const tool of recipe.toolSelections) {
					if (tool.requiredSDKType && tool.requiredSDKType !== (nextModel.providerSDKType as string)) {
						setActionError(
							`Tool "${tool.choice.displayName || tool.choice.target.name}" requires provider SDK "${tool.requiredSDKType}", but the Agent Model uses "${nextModel.providerSDKType}".`
						);
						return false;
					}
				}

				context.handleSetSelectedModel({
					...nextModel,
				});

				if (recipe.includeModelSystemPrompt !== undefined) {
					systemPrompt.setIncludeModelDefault(recipe.includeModelSystemPrompt);
				}

				const instructionSources: AgentPreparedInstructionSource[] =
					recipe.instructionSources.length > 0
						? recipe.instructionSources
						: recipe.instructionText.trim()
							? [
									{
										kind: 'text',
										artifact: option.ref,
										displayName: `${option.displayName} instructions`,
										text: recipe.instructionText,
									},
								]
							: [];

				const visibleInstructionSources = instructionSources.filter(source => source.text.trim());
				const nextInstructionSourceKeys = new Set(
					visibleInstructionSources.map(source => agentInstructionSourceKey(option.ref, source))
				);

				for (const previousKey of appliedInstructionSourceKeysRef.current) {
					if (nextInstructionSourceKeys.has(previousKey)) {
						continue;
					}
					systemPrompt.removeInstructionSource(previousKey);
				}

				for (const source of visibleInstructionSources) {
					const identityKey = agentInstructionSourceKey(option.ref, source);

					systemPrompt.upsertAndSelectInstructionSource({
						identityKey,
						sourceKind: 'agent',
						bundleID: agentKey(option.ref),
						bundleDisplayName: option.displayName,
						bundleSlug: option.agent.name,
						displayName: source.displayName,
						sourceSlug: source.artifact.artifactID,
						text: source.text,
						sourceTags: source.sourceTags,
						isBuiltIn: option.agent.builtIn,
						skillRef: source.kind === 'skill' ? source.artifact : undefined,
					});
				}

				appliedInstructionSourceKeysRef.current = nextInstructionSourceKeys;
				runtimeCompiler.applyAgentRuntime(recipe);
				setTrackedAgent(option.key, recipe);
				return true;
			} catch (prepareError) {
				if (applyRequestSeqRef.current !== requestSeq) {
					return false;
				}

				setActionError(getErrorMessage(prepareError, 'Failed to prepare the Agent starter recipe.'));
				return false;
			} finally {
				if (applyRequestSeqRef.current === requestSeq) {
					isApplyingRef.current = false;
					setIsApplying(false);
				}
			}
		},
		[agentOptions, context, runtimeCompiler, setTrackedAgent, systemPrompt]
	);

	const applyDefaultAgent = useCallback(
		async (force: boolean): Promise<boolean> => {
			if (!force && hasTrackedAgent()) {
				pendingDefaultActionRef.current = null;
				return true;
			}

			if (loading || !context.modelOptionsLoaded || isApplyingRef.current) {
				pendingDefaultActionRef.current = force ? 'reset' : 'ensure';
				return false;
			}

			if (!defaultAgent) {
				pendingDefaultActionRef.current = null;
				setActionError(NO_SELECTABLE_AGENT_ERROR);
				return false;
			}

			pendingDefaultActionRef.current = null;
			return selectAgent(defaultAgent.ref);
		},
		[context.modelOptionsLoaded, defaultAgent, hasTrackedAgent, loading, selectAgent]
	);

	const ensureDefaultAgent = useCallback(() => applyDefaultAgent(false), [applyDefaultAgent]);
	const resetToDefaultAgent = useCallback(() => applyDefaultAgent(true), [applyDefaultAgent]);

	const reapplySelectedAgent = useCallback(async (): Promise<boolean> => {
		const trackedKey = selectedAgentKeyRef.current;
		const option = trackedKey ? agentOptions.find(agent => agent.key === trackedKey) : undefined;

		if (!option) {
			setActionError('No active Agent is available to reapply.');
			return false;
		}

		return selectAgent(option.ref);
	}, [agentOptions, selectAgent]);

	const trackDefaultAgentWithoutApplying = useCallback(async (): Promise<boolean> => {
		if (hasTrackedAgent()) {
			pendingDefaultActionRef.current = null;
			return true;
		}

		// A restore must invalidate a still-running initial default application,
		// otherwise it can overwrite restored Composer state when it resolves.
		cancelActiveAgentRequest();

		if (loading) {
			pendingDefaultActionRef.current = 'track';
			return false;
		}

		if (!defaultAgent) {
			pendingDefaultActionRef.current = null;
			setActionError(NO_SELECTABLE_AGENT_ERROR);
			return false;
		}

		pendingDefaultActionRef.current = null;
		setActionError(null);
		setTrackedAgent(defaultAgent.key, null, true);
		return true;
	}, [cancelActiveAgentRequest, defaultAgent, hasTrackedAgent, loading, setTrackedAgent]);

	const clearAgentTracking = useCallback(() => {
		cancelActiveAgentRequest();
		pendingDefaultActionRef.current = null;
		autoEnsureCatalogKeyRef.current = null;
		setTrackedAgent(null, null);
		setActionError(null);
		// Retain applied Composer state and instruction-source keys. The next
		// Agent application can remove only Agent-owned stale sources.
	}, [cancelActiveAgentRequest, setTrackedAgent]);

	const catalogIdentity = useMemo(
		() => agentOptions.map(option => `${option.key}:${option.isSelectable ? '1' : '0'}`).join('|'),
		[agentOptions]
	);

	useEffect(() => {
		if (initialDefaultRequestRef.current) {
			return;
		}

		initialDefaultRequestRef.current = true;
		void ensureDefaultAgent();
	}, [ensureDefaultAgent]);

	useEffect(() => {
		const pendingAction = pendingDefaultActionRef.current;
		if (!pendingAction) {
			return;
		}

		if (pendingAction === 'track') {
			if (loading) {
				return;
			}

			pendingDefaultActionRef.current = null;
			// oxlint-disable-next-line react/set-state-in-effect
			void trackDefaultAgentWithoutApplying();
			return;
		}

		if (loading || !context.modelOptionsLoaded || isApplying || isApplyingRef.current) {
			return;
		}

		pendingDefaultActionRef.current = null;
		if (pendingAction === 'reset') {
			void resetToDefaultAgent();
		} else {
			void ensureDefaultAgent();
		}
	}, [
		context.modelOptionsLoaded,
		ensureDefaultAgent,
		isApplying,
		loading,
		resetToDefaultAgent,
		trackDefaultAgentWithoutApplying,
	]);

	useEffect(() => {
		if (selectedAgentKey === null || selectedAgent !== null) {
			return;
		}

		// oxlint-disable-next-line react/set-state-in-effect
		clearAgentTracking();
		void ensureDefaultAgent();
	}, [clearAgentTracking, ensureDefaultAgent, selectedAgent, selectedAgentKey]);

	useEffect(() => {
		if (loading || !context.modelOptionsLoaded || isApplying || isApplyingRef.current || hasTrackedAgent()) {
			return;
		}

		if (autoEnsureCatalogKeyRef.current === catalogIdentity) {
			return;
		}

		autoEnsureCatalogKeyRef.current = catalogIdentity;
		void ensureDefaultAgent();
	}, [catalogIdentity, context.modelOptionsLoaded, ensureDefaultAgent, hasTrackedAgent, isApplying, loading]);

	return {
		agentOptions,
		loading,
		error,
		actionError,
		isApplying,
		baseAgentKey: baseAgent?.key ?? null,
		defaultAgentKey: defaultAgent?.key ?? null,
		isDefaultAgentSelected,

		selectedAgentKey: selectedAgent?.key ?? null,
		selectedAgent,
		preparedStarter: selectedAgent ? preparedStarter : null,
		isTrackingOnly: selectedAgent ? isTrackingOnly : false,
		refreshAgents,
		selectAgent,
		ensureDefaultAgent,
		resetToDefaultAgent,
		reapplySelectedAgent,
		trackDefaultAgentWithoutApplying,
		clearAgentTracking,
	};
}
