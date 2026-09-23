import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { ArtifactRef } from '@/spec/artifact';
import type { UIChatOption } from '@/spec/modelpreset';

import { getErrorMessage } from '@/lib/error_utils';

import type { AgentCatalogOption, PreparedAgentStarter } from '@/apis/agent_management';
import { agentManagementAPI } from '@/apis/baseapi';

import type { AgentSystemPromptController } from '@/chats/composer/skills/use_agent_system_prompt';

interface AgentComposerContext {
	selectedModel: UIChatOption;
	allOptions: UIChatOption[];
	modelOptionsLoaded: boolean;
	handleSetSelectedModel: (next: UIChatOption) => void;
}

interface AgentRuntimeApplier {
	applyAgentRuntime(recipe: PreparedAgentStarter): void;
}

export interface AgentManagerState {
	agentOptions: AgentCatalogOption[];
	loading: boolean;
	error: string | null;
	actionError: string | null;
	isApplying: boolean;

	selectedAgentKey: string | null;
	selectedAgent: AgentCatalogOption | null;
	preparedStarter: PreparedAgentStarter | null;

	refreshAgents(): Promise<void>;
	selectAgent(agentRef: ArtifactRef): Promise<boolean>;
	ensureDefaultAgent(): Promise<boolean>;
	clearAgentTracking(): void;
}

function agentKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
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
	const [selectedAgentKey, setSelectedAgentKey] = useState<string | null>(null);
	const [preparedStarter, setPreparedStarter] = useState<PreparedAgentStarter | null>(null);

	const pendingDefaultRef = useRef(false);
	const appliedInstructionSourceKeyRef = useRef<string | null>(null);
	const refreshAgents = useCallback(async () => {
		setLoading(true);
		setError(null);

		try {
			setAgentOptions(await agentManagementAPI.listAgentCatalogOptions());
		} catch (loadError) {
			setAgentOptions([]);
			setError(getErrorMessage(loadError, 'Failed to load Agents.'));
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		// oxlint-disable-next-line react/set-state-in-effect
		void refreshAgents();
	}, [refreshAgents]);

	const selectedAgent = useMemo(
		() => agentOptions.find(option => option.key === selectedAgentKey) ?? null,
		[agentOptions, selectedAgentKey]
	);

	const selectAgent = useCallback(
		async (agentRef: ArtifactRef): Promise<boolean> => {
			if (isApplying) {
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

			setActionError(null);
			setIsApplying(true);

			try {
				const recipe = await agentManagementAPI.prepareAgentStarter(agentRef);
				const recipeError = getRecipeError(recipe);

				if (recipeError) {
					setActionError(recipeError);
					return false;
				}

				let nextModel = context.selectedModel;

				if (recipe.modelPresetRef) {
					const resolvedModel = context.allOptions.find(
						model =>
							model.providerName === recipe.modelPresetRef?.providerName &&
							model.modelPresetID === recipe.modelPresetRef?.modelPresetID
					);

					if (!resolvedModel) {
						setActionError(
							`Agent Model "${recipe.modelPresetRef.providerName}/${recipe.modelPresetRef.modelPresetID}" is not currently selectable in Composer.`
						);
						return false;
					}

					nextModel = resolvedModel;
				}

				for (const tool of recipe.toolSelections) {
					if (tool.requiredSDKType && tool.requiredSDKType !== (nextModel.providerSDKType as string)) {
						setActionError(
							`Tool "${tool.choice.displayName || tool.choice.toolSlug}" requires provider SDK "${tool.requiredSDKType}", but the Agent Model uses "${nextModel.providerSDKType}".`
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

				const nextInstructionSourceKey = `agent:${option.ref.rootID}:${option.ref.artifactID}:instructions`;
				const previousInstructionSourceKey = appliedInstructionSourceKeyRef.current;

				if (previousInstructionSourceKey && previousInstructionSourceKey !== nextInstructionSourceKey) {
					systemPrompt.removeInstructionSource(previousInstructionSourceKey);
				}

				if (recipe.instructionText.trim()) {
					systemPrompt.upsertAndSelectInstructionSource({
						identityKey: nextInstructionSourceKey,
						sourceKind: 'agent',
						bundleID: option.ref.rootID,
						bundleDisplayName: 'Agent',
						bundleSlug: option.agent.builtIn ? 'built-in' : 'managed',
						displayName: `${option.displayName} instructions`,
						sourceSlug: option.agent.name,
						text: recipe.instructionText,
						isBuiltIn: option.agent.builtIn,
					});
					appliedInstructionSourceKeyRef.current = nextInstructionSourceKey;
				} else if (previousInstructionSourceKey) {
					systemPrompt.removeInstructionSource(previousInstructionSourceKey);
					appliedInstructionSourceKeyRef.current = null;
				}
				runtimeCompiler.applyAgentRuntime(recipe);
				setPreparedStarter(recipe);
				setSelectedAgentKey(option.key);
				return true;
			} catch (prepareError) {
				setActionError(getErrorMessage(prepareError, 'Failed to prepare the Agent starter recipe.'));
				return false;
			} finally {
				setIsApplying(false);
			}
		},
		[agentOptions, context, isApplying, runtimeCompiler, systemPrompt]
	);

	const ensureDefaultAgent = useCallback(async (): Promise<boolean> => {
		if (loading || !context.modelOptionsLoaded) {
			pendingDefaultRef.current = true;
			return false;
		}

		const defaultAgent =
			agentOptions.find(option => option.agent.builtIn && option.agent.name === 'base' && option.isSelectable) ??
			agentOptions.find(option => option.isSelectable);

		if (!defaultAgent) {
			setActionError('No selectable Agent is available.');
			return false;
		}

		pendingDefaultRef.current = false;
		return selectAgent(defaultAgent.ref);
	}, [agentOptions, context.modelOptionsLoaded, loading, selectAgent]);

	useEffect(() => {
		if (!pendingDefaultRef.current || loading || !context.modelOptionsLoaded) {
			return;
		}

		void ensureDefaultAgent();
	}, [context.modelOptionsLoaded, ensureDefaultAgent, loading]);

	const clearAgentTracking = useCallback(() => {
		setSelectedAgentKey(null);
		setPreparedStarter(null);
		setActionError(null);
		// Deliberately retain applied Composer state and instruction sources.
		// Agent selection is a starter recipe, not a live runtime binding.
	}, []);

	return {
		agentOptions,
		loading,
		error,
		actionError,
		isApplying,
		selectedAgentKey,
		selectedAgent,
		preparedStarter,
		refreshAgents,
		selectAgent,
		ensureDefaultAgent,
		clearAgentTracking,
	};
}
