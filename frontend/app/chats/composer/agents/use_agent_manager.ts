import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { ArtifactRef } from '@/spec/artifact';
import type { UIModelOption } from '@/spec/model';
import { CapabilityResolutionStatus } from '@/spec/artifact';

import { getErrorMessage } from '@/lib/error_utils';

import type { AgentCatalogOption } from '@/apis/agent_management';
import { agentManagementAPI } from '@/apis/baseapi';

interface AgentComposerContext {
	selectedModel: UIModelOption;
	allOptions: UIModelOption[];
	modelOptionsLoaded: boolean;
	handleSetSelectedModel: Dispatch<SetStateAction<UIModelOption>>;
	setModelSelectionSource: (source: 'session' | 'agent') => void;
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
	preparedStarter: null;
	isTrackingOnly: boolean;

	refreshAgents(): Promise<void>;
	selectAgent(agentRef: ArtifactRef): Promise<boolean>;
	ensureDefaultAgent(): Promise<boolean>;
	resetToDefaultAgent(): Promise<boolean>;
	reapplySelectedAgent(): Promise<boolean>;
	trackDefaultAgentWithoutApplying(): Promise<boolean>;
	clearAgentTracking(): void;
	restoreSelectedAgent(agent?: ArtifactRef): Promise<boolean>;
}

function agentKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

export function useAgentManager(context: AgentComposerContext): AgentManagerState {
	const [agentOptions, setAgentOptions] = useState<AgentCatalogOption[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [actionError, setActionError] = useState<string | null>(null);
	const [isApplying, setIsApplying] = useState(false);
	const [isTrackingOnly, setIsTrackingOnly] = useState(false);
	const [selectedAgentKey, setSelectedAgentKey] = useState<string | null>(null);

	const selectedAgentKeyRef = useRef<string | null>(null);
	const isApplyingRef = useRef(false);
	const applyRequestSeqRef = useRef(0);
	const catalogRequestSeqRef = useRef(0);
	const pendingDefaultActionRef = useRef<PendingDefaultAction | null>(null);
	const initialDefaultRequestRef = useRef(false);
	const autoEnsureCatalogKeyRef = useRef<string | null>(null);
	const pendingRestoredAgentRef = useRef<ArtifactRef | null>(null);

	const setTrackedAgent = useCallback((aKey: string | null, trackingOnly = false) => {
		selectedAgentKeyRef.current = aKey;
		setSelectedAgentKey(aKey);

		setIsTrackingOnly(trackingOnly);
	}, []);

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
			const requestSequence = applyRequestSeqRef.current + 1;
			applyRequestSeqRef.current = requestSequence;
			isApplyingRef.current = true;
			setIsApplying(true);

			const option = agentOptions.find(value => value.key === agentKey(agentRef));

			if (!option) {
				setActionError('The selected Agent is no longer available.');
				isApplyingRef.current = false;
				setIsApplying(false);
				return false;
			}

			if (!option.isSelectable) {
				setActionError(option.availabilityReason ?? 'The selected Agent is not currently available.');
				isApplyingRef.current = false;
				setIsApplying(false);
				return false;
			}

			if (!context.modelOptionsLoaded) {
				setActionError('Model options are still loading. Try again in a moment.');
				isApplyingRef.current = false;
				setIsApplying(false);
				return false;
			}

			try {
				const resolution = await agentManagementAPI.resolveAgent(agentRef);
				if (applyRequestSeqRef.current !== requestSequence) {
					return false;
				}
				const unavailableRequired = resolution.capabilities.occurrences.find(
					occurrence => occurrence.required && occurrence.status !== CapabilityResolutionStatus.Available
				);

				if (unavailableRequired) {
					setActionError(unavailableRequired.message || 'A required Agent capability is unavailable.');
					return false;
				}

				context.setModelSelectionSource('agent');
				setTrackedAgent(option.key);
				return true;
			} catch (e: unknown) {
				setActionError(getErrorMessage(e, 'Failed to validate the Agent selection.'));
				return false;
			} finally {
				if (applyRequestSeqRef.current === requestSequence) {
					isApplyingRef.current = false;
					setIsApplying(false);
				}
			}
		},
		[agentOptions, context, setTrackedAgent]
	);

	const restoreSelectedAgent = useCallback(
		async (agent?: ArtifactRef): Promise<boolean> => {
			cancelActiveAgentRequest();

			if (!agent) {
				pendingRestoredAgentRef.current = null;
				context.setModelSelectionSource('session');
				setTrackedAgent(null);
				setActionError(null);
				return true;
			}

			if (loading) {
				pendingRestoredAgentRef.current = agent;
				return false;
			}

			const option = agentOptions.find(candidate => candidate.key === agentKey(agent));
			if (!option) {
				pendingRestoredAgentRef.current = null;
				setTrackedAgent(null);
				setActionError('The Session-selected Agent is no longer available in the current catalog.');
				return false;
			}

			pendingRestoredAgentRef.current = null;
			context.setModelSelectionSource('agent');
			setTrackedAgent(option.key, true);
			setActionError(null);
			return true;
		},
		[agentOptions, cancelActiveAgentRequest, context, loading, setTrackedAgent]
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
		setTrackedAgent(defaultAgent.key, true);
		return true;
	}, [cancelActiveAgentRequest, defaultAgent, hasTrackedAgent, loading, setTrackedAgent]);

	const clearAgentTracking = useCallback(() => {
		cancelActiveAgentRequest();
		pendingDefaultActionRef.current = null;
		autoEnsureCatalogKeyRef.current = null;
		context.setModelSelectionSource('session');
		setTrackedAgent(null);
		setActionError(null);
	}, [cancelActiveAgentRequest, context, setTrackedAgent]);

	const catalogIdentity = useMemo(
		() => agentOptions.map(option => `${option.key}:${option.isSelectable ? '1' : '0'}`).join('|'),
		[agentOptions]
	);

	useEffect(() => {
		if (initialDefaultRequestRef.current) {
			return;
		}

		initialDefaultRequestRef.current = true;
		void Promise.resolve().then(() => ensureDefaultAgent());
	}, [ensureDefaultAgent]);

	useEffect(() => {
		const pendingRestoredAgent = pendingRestoredAgentRef.current;
		if (pendingRestoredAgent && !loading) {
			pendingRestoredAgentRef.current = null;
			void Promise.resolve().then(() => restoreSelectedAgent(pendingRestoredAgent));
			return;
		}

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

		void Promise.resolve().then(() => {
			clearAgentTracking();
			return ensureDefaultAgent();
		});
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
		preparedStarter: null,
		isTrackingOnly: selectedAgent ? isTrackingOnly : false,
		refreshAgents,
		selectAgent,
		ensureDefaultAgent,
		resetToDefaultAgent,
		reapplySelectedAgent,
		trackDefaultAgentWithoutApplying,
		clearAgentTracking,
		restoreSelectedAgent,
	};
}
