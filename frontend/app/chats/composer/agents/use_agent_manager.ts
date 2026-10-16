import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { AgentCapabilityOccurrence, AgentResolution } from '@/spec/agent';
import type { ArtifactRef, CapabilityTarget } from '@/spec/artifact';
import type { UIModelOption } from '@/spec/model';
import type { SessionConfiguration } from '@/spec/session';
import { CapabilityResolutionStatus, CapabilityTargetForm } from '@/spec/artifact';
import { SessionCollectionOverrideMode } from '@/spec/session_base';

import { getErrorMessage } from '@/lib/error_utils';

import type { AgentCatalogOption } from '@/apis/agent_management';
import { agentManagementAPI } from '@/apis/baseapi';

import type { AgentCapabilityExclusion } from '@/chats/composer/editor/editor_types';

interface AgentComposerContext {
	selectedModel: UIModelOption;
	allOptions: UIModelOption[];
	modelOptionsLoaded: boolean;
	modelSelectionSource: 'session' | 'agent';
	handleSetSelectedModel: Dispatch<SetStateAction<UIModelOption>>;
	setModelSelectionSource: (source: 'session' | 'agent') => void;
	setAgentModelPreview: (model?: UIModelOption) => void;
}

interface PendingRestoredAgent {
	agent: ArtifactRef;
	modelSelectionSource: 'session' | 'agent';
	exclusionKeys: string[];
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
	selectedAgentResolution: AgentResolution | null;
	preparedStarter: null;
	agentModelRef?: ArtifactRef;
	isTrackingOnly: boolean;
	capabilityExclusions: AgentCapabilityExclusion[];

	refreshAgents(): Promise<void>;
	selectAgent(agentRef: ArtifactRef): Promise<boolean>;
	ensureDefaultAgent(): Promise<boolean>;
	resetToDefaultAgent(): Promise<boolean>;
	reapplySelectedAgent(): Promise<boolean>;
	trackDefaultAgentWithoutApplying(): Promise<boolean>;
	clearAgentTracking(): void;
	restoreSelectedAgent(
		agent?: ArtifactRef,
		modelSelectionSource?: 'session' | 'agent',
		configuration?: SessionConfiguration
	): Promise<boolean>;
	getCapabilityExclusion(occurrence: AgentCapabilityOccurrence): AgentCapabilityExclusion | undefined;
	toggleCapabilityExclusion(occurrence: AgentCapabilityOccurrence): void;
}

function agentKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

function capabilityTargetKey(target: CapabilityTarget): string {
	if (target.form === CapabilityTargetForm.Artifact && target.artifact) {
		return `artifact:${target.artifact.rootID}:${target.artifact.artifactID}`;
	}

	return ['direct', target.providerIdentity ?? '', target.providerLocalID ?? '', target.type, target.name].join(':');
}

function cloneCapabilityTarget(target: CapabilityTarget): CapabilityTarget {
	return {
		...target,
		...(target.artifact
			? {
					artifact: {
						rootID: target.artifact.rootID,
						artifactID: target.artifact.artifactID,
					},
				}
			: {}),
	};
}

function capabilityExclusionForOccurrence(occurrence: AgentCapabilityOccurrence): AgentCapabilityExclusion | undefined {
	if (occurrence.status !== CapabilityResolutionStatus.Available || !occurrence.target) {
		return undefined;
	}

	switch (occurrence.type) {
		case 'text':
			if (occurrence.target.form !== CapabilityTargetForm.Artifact) {
				return undefined;
			}
			return {
				key: capabilityTargetKey(occurrence.target),
				kind: 'instruction',
				target: cloneCapabilityTarget(occurrence.target),
			};

		case 'tool':
			return {
				key: capabilityTargetKey(occurrence.target),
				kind: 'tool',
				target: cloneCapabilityTarget(occurrence.target),
			};

		case 'skill':
			if (occurrence.target.form !== CapabilityTargetForm.Artifact) {
				return undefined;
			}
			return {
				key: capabilityTargetKey(occurrence.target),
				kind: 'skill',
				target: cloneCapabilityTarget(occurrence.target),
			};

		case 'mcp':
			if (occurrence.target.form !== CapabilityTargetForm.Artifact) {
				return undefined;
			}
			return {
				key: capabilityTargetKey(occurrence.target),
				kind: 'mcp',
				target: cloneCapabilityTarget(occurrence.target),
			};

		default:
			return undefined;
	}
}

function resolvedAgentModelRef(resolution: AgentResolution | null): ArtifactRef | undefined {
	for (const occurrence of resolution?.capabilities.occurrences ?? []) {
		if (
			occurrence.type === 'model' &&
			occurrence.status === CapabilityResolutionStatus.Available &&
			occurrence.target?.form === CapabilityTargetForm.Artifact &&
			occurrence.target.artifact
		) {
			return occurrence.target.artifact;
		}
	}

	return undefined;
}

function agentRemovalKeys(configuration?: SessionConfiguration): string[] {
	if (!configuration) {
		return [];
	}

	const keys = new Set<string>();
	for (const override of configuration.tools ?? []) {
		if (override.mode === SessionCollectionOverrideMode.Remove) {
			keys.add(override.key);
		}
	}
	for (const override of configuration.skills ?? []) {
		if (override.mode === SessionCollectionOverrideMode.Remove) {
			keys.add(override.key);
		}
	}
	for (const override of configuration.instructions ?? []) {
		if (override.mode === SessionCollectionOverrideMode.Remove) {
			keys.add(override.key);
		}
	}
	for (const override of configuration.mcp ?? []) {
		if (override.mode === SessionCollectionOverrideMode.Remove) {
			keys.add(override.key);
		}
	}

	return [...keys];
}

export function useAgentManager(context: AgentComposerContext): AgentManagerState {
	const [agentOptions, setAgentOptions] = useState<AgentCatalogOption[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [actionError, setActionError] = useState<string | null>(null);
	const [isApplying, setIsApplying] = useState(false);
	const [isTrackingOnly, setIsTrackingOnly] = useState(false);
	const [selectedAgentKey, setSelectedAgentKey] = useState<string | null>(null);
	const [excludedCapabilityKeys, setExcludedCapabilityKeys] = useState<string[]>([]);
	const [selectedAgentResolution, setSelectedAgentResolution] = useState<AgentResolution | null>(null);

	const selectedAgentKeyRef = useRef<string | null>(null);
	const isApplyingRef = useRef(false);
	const applyRequestSeqRef = useRef(0);
	const catalogRequestSeqRef = useRef(0);
	const pendingDefaultActionRef = useRef<PendingDefaultAction | null>(null);
	const initialDefaultRequestRef = useRef(false);
	const autoEnsureCatalogKeyRef = useRef<string | null>(null);
	const pendingRestoredAgentRef = useRef<PendingRestoredAgent | null>(null);

	const setTrackedAgent = useCallback(
		(aKey: string | null, trackingOnly = false, resolution: AgentResolution | null = null) => {
			selectedAgentKeyRef.current = aKey;
			setSelectedAgentKey(aKey);
			setSelectedAgentResolution(resolution);
			setIsTrackingOnly(trackingOnly);
		},
		[]
	);

	const replaceExcludedCapabilityKeys = useCallback((keys: readonly string[]) => {
		setExcludedCapabilityKeys([
			...new Set(
				keys.filter(key => {
					return key.trim().length > 0;
				})
			),
		]);
	}, []);

	const cancelActiveAgentRequest = useCallback(() => {
		applyRequestSeqRef.current += 1;
		isApplyingRef.current = false;
		setIsApplying(false);
	}, []);

	const fetchAgentOptions = useCallback(async (force: boolean, requestSeq: number) => {
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

	const loadAgentOptions = useCallback(
		(force = false) => {
			const requestSeq = catalogRequestSeqRef.current + 1;
			catalogRequestSeqRef.current = requestSeq;
			setLoading(true);
			setError(null);
			return fetchAgentOptions(force, requestSeq);
		},
		[fetchAgentOptions]
	);

	const refreshAgents = useCallback(async () => {
		autoEnsureCatalogKeyRef.current = null;
		await loadAgentOptions(true);
	}, [loadAgentOptions]);

	useEffect(() => {
		const requestSeq = catalogRequestSeqRef.current + 1;
		catalogRequestSeqRef.current = requestSeq;
		// oxlint-disable-next-line react/set-state-in-effect
		void fetchAgentOptions(false, requestSeq);
	}, [fetchAgentOptions]);

	const selectedAgent = useMemo(
		() => agentOptions.find(option => option.key === selectedAgentKey) ?? null,
		[agentOptions, selectedAgentKey]
	);

	const currentSelectedAgentResolution =
		selectedAgent !== null &&
		selectedAgentResolution !== null &&
		agentKey(selectedAgentResolution.agent.ref) === selectedAgent.key
			? selectedAgentResolution
			: null;

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

	const agentModelRef = useMemo(
		() => resolvedAgentModelRef(currentSelectedAgentResolution),
		[currentSelectedAgentResolution]
	);

	const resolveAgentModelOption = useCallback(
		(resolution: AgentResolution): UIModelOption | undefined => {
			const ref = resolvedAgentModelRef(resolution);
			if (!ref) {
				return undefined;
			}

			return context.allOptions.find(
				option => option.model?.rootID === ref.rootID && option.model?.artifactID === ref.artifactID
			);
		},
		[context.allOptions]
	);

	const capabilityExclusions = useMemo(() => {
		const excluded = new Set(excludedCapabilityKeys);
		const output: AgentCapabilityExclusion[] = [];

		for (const occurrence of currentSelectedAgentResolution?.capabilities.occurrences ?? []) {
			const exclusion = capabilityExclusionForOccurrence(occurrence);
			if (exclusion && excluded.has(exclusion.key)) {
				output.push(exclusion);
			}
		}

		return output;
	}, [currentSelectedAgentResolution, excludedCapabilityKeys]);

	useEffect(() => {
		if (!selectedAgent || currentSelectedAgentResolution) {
			return;
		}

		let cancelled = false;

		void agentManagementAPI
			.resolveAgent(selectedAgent.ref)
			.then(resolution => {
				if (!cancelled) {
					if (context.modelSelectionSource === 'agent') {
						context.setAgentModelPreview(resolveAgentModelOption(resolution));
					}
					setSelectedAgentResolution(resolution);
				}
			})
			.catch(() => {
				if (!cancelled) {
					setSelectedAgentResolution(null);
				}
			});

		return () => {
			cancelled = true;
		};
	}, [context, currentSelectedAgentResolution, resolveAgentModelOption, selectedAgent]);

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

				context.setAgentModelPreview(resolveAgentModelOption(resolution));
				replaceExcludedCapabilityKeys([]);
				context.setModelSelectionSource('agent');
				setTrackedAgent(option.key, false, resolution);
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
		[agentOptions, context, replaceExcludedCapabilityKeys, resolveAgentModelOption, setTrackedAgent]
	);

	const restoreSelectedAgent = useCallback(
		async (
			agent?: ArtifactRef,
			modelSelectionSource: 'session' | 'agent' = 'agent',
			configuration?: SessionConfiguration
		): Promise<boolean> => {
			cancelActiveAgentRequest();
			const exclusionKeys = agentRemovalKeys(configuration);

			if (!agent) {
				pendingRestoredAgentRef.current = null;
				context.setModelSelectionSource('session');
				replaceExcludedCapabilityKeys([]);
				setTrackedAgent(null);
				setActionError(null);
				return true;
			}

			if (loading) {
				pendingRestoredAgentRef.current = {
					agent,
					modelSelectionSource,
					exclusionKeys,
				};
				return false;
			}

			const option = agentOptions.find(candidate => candidate.key === agentKey(agent));
			if (!option) {
				pendingRestoredAgentRef.current = null;
				replaceExcludedCapabilityKeys([]);
				setTrackedAgent(null);
				setActionError('The Session-selected Agent is no longer available in the current catalog.');
				return false;
			}

			pendingRestoredAgentRef.current = null;
			replaceExcludedCapabilityKeys(exclusionKeys);
			context.setModelSelectionSource(modelSelectionSource);
			try {
				const resolution = await agentManagementAPI.resolveAgent(agent);
				if (modelSelectionSource === 'agent') {
					context.setAgentModelPreview(resolveAgentModelOption(resolution));
				}
				setTrackedAgent(option.key, true, resolution);
			} catch {
				setTrackedAgent(option.key, true);
			}
			setActionError(null);
			return true;
		},
		[
			agentOptions,
			cancelActiveAgentRequest,
			context,
			loading,
			replaceExcludedCapabilityKeys,
			resolveAgentModelOption,
			setTrackedAgent,
		]
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
		replaceExcludedCapabilityKeys([]);
		setTrackedAgent(null);
		setActionError(null);
	}, [cancelActiveAgentRequest, context, replaceExcludedCapabilityKeys, setTrackedAgent]);

	const getCapabilityExclusion = useCallback(
		(occurrence: AgentCapabilityOccurrence): AgentCapabilityExclusion | undefined => {
			const exclusion = capabilityExclusionForOccurrence(occurrence);
			if (!exclusion || !excludedCapabilityKeys.includes(exclusion.key)) {
				return undefined;
			}
			return exclusion;
		},
		[excludedCapabilityKeys]
	);

	const toggleCapabilityExclusion = useCallback((occurrence: AgentCapabilityOccurrence) => {
		const exclusion = capabilityExclusionForOccurrence(occurrence);
		if (!exclusion) {
			setActionError(
				'This Agent contribution cannot be removed because it has no canonical Artifact or capability target.'
			);
			return;
		}

		setExcludedCapabilityKeys(previous =>
			previous.includes(exclusion.key) ? previous.filter(key => key !== exclusion.key) : [...previous, exclusion.key]
		);
	}, []);

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
			void Promise.resolve().then(() =>
				restoreSelectedAgent(pendingRestoredAgent.agent, pendingRestoredAgent.modelSelectionSource, {
					agent: pendingRestoredAgent.agent,
					modelOverrides: {},
					history: {},
					tools: pendingRestoredAgent.exclusionKeys.map(key => ({
						key,
						mode: SessionCollectionOverrideMode.Remove,
					})),
				})
			);
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
			void Promise.resolve().then(() => trackDefaultAgentWithoutApplying());
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
		restoreSelectedAgent,
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
		selectedAgentResolution: currentSelectedAgentResolution,
		preparedStarter: null,
		agentModelRef,
		isTrackingOnly: selectedAgent ? isTrackingOnly : false,
		capabilityExclusions,
		refreshAgents,
		selectAgent,
		ensureDefaultAgent,
		resetToDefaultAgent,
		reapplySelectedAgent,
		trackDefaultAgentWithoutApplying,
		clearAgentTracking,
		restoreSelectedAgent,
		getCapabilityExclusion,
		toggleCapabilityExclusion,
	};
}
