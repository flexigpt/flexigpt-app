import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { AgentCapabilityOccurrence, AgentResolution } from '@/spec/agent';
import type { ArtifactRef, CapabilityTarget } from '@/spec/artifact';
import type { UIModelOption } from '@/spec/model';
import type { SessionActiveSkillOverride, SessionConfiguration } from '@/spec/session';
import { artifactRefKey, CapabilityResolutionStatus, CapabilityTargetForm } from '@/spec/artifact';
import { SessionActivityOverrideMode, SessionCollectionOverrideMode } from '@/spec/session_base';

import { getErrorMessage } from '@/lib/error_utils';

import type { AgentCatalogOption, AgentStarterPreview } from '@/apis/agent_management';
import { agentManagementAPI } from '@/apis/baseapi';

import type {
	AgentInstructionSourceDeclaration,
	AgentMCPProjection,
	AgentSkillProjection,
	AgentToolProjection,
} from '@/chats/composer/agents/agent_projection';
import type { AgentCapabilityExclusion } from '@/chats/composer/editor/editor_types';
import {
	agentSkillActivityChange,
	effectiveAgentSkillActive,
	projectAgentInstructionSources,
	projectAgentMCPCapabilities,
	projectAgentSkillCapabilities,
	projectAgentToolCapabilities,
	sameAgentDefinition,
} from '@/chats/composer/agents/agent_projection';
import { sessionCapabilityKey } from '@/chats/session/session_capability_key';

interface AgentComposerContext {
	selectedModel: UIModelOption;
	allOptions: UIModelOption[];
	modelOptionsLoaded: boolean;
	modelSelectionSource: 'session' | 'agent';
	handleSetSelectedModel: Dispatch<SetStateAction<UIModelOption>>;
	setModelSelectionSource: (source: 'session' | 'agent') => void;
	setAgentModelPreview: (model: UIModelOption | undefined, agentOwnsModel: boolean) => void;
}

interface AgentManagerOptions {
	applyUserMessageStarter?: (text: string) => boolean;
	replaceAgentInstructionSources?: (sources: AgentInstructionSourceDeclaration[]) => void;
	clearAgentInstructionSources?: () => void;
	setAgentInstructionSourceSelected?: (occurrencePath: string, selected: boolean) => void;
}

interface PendingRestoredAgent {
	agent: ArtifactRef;
	modelSelectionSource: 'session' | 'agent';
	exclusionKeys: string[];
	configuration?: SessionConfiguration;
}

type PendingDefaultAction = 'ensure' | 'reset' | 'track';

const BASE_AGENT_NAME = 'base';
const NO_SELECTABLE_AGENT_ERROR = 'No selectable Agent is available.';

export interface AgentManagerState {
	selectedAgentKey: string | null;
	selectedAgent: AgentCatalogOption | null;
	selectedAgentResolution: AgentResolution | null;
	preparedStarter: AgentStarterPreview | null;
	agentModelRef?: ArtifactRef;
	isTrackingOnly: boolean;
	capabilityExclusions: AgentCapabilityExclusion[];
	releasedCapabilityExclusionKeys: string[];
	agentSkillActivityOverrides: SessionActiveSkillOverride[];
	releasedAgentSkillActivityOverrideKeys: string[];
	agentSelectionPinRefreshRequested: boolean;
	agentToolProjections: AgentToolProjection[];
	agentSkillProjections: AgentSkillProjection[];
	agentMCPProjections: AgentMCPProjection[];

	agentOptions: AgentCatalogOption[];
	loading: boolean;
	error: string | null;
	actionError: string | null;
	isApplying: boolean;
	baseAgentKey: string | null;
	defaultAgentKey: string | null;
	isDefaultAgentSelected: boolean;

	refreshAgents(): Promise<void>;
	selectAgent(agentRef: ArtifactRef): Promise<boolean>;
	ensureDefaultAgent(): Promise<boolean>;
	resetToDefaultAgent(): Promise<boolean>;
	reapplySelectedAgent(): Promise<boolean>;
	resetModelToAgent(): Promise<boolean>;
	acknowledgeAgentSelectionPinRefresh(): void;
	trackDefaultAgentWithoutApplying(): Promise<boolean>;
	clearAgentTracking(): void;
	restoreSelectedAgent(
		agent?: ArtifactRef,
		modelSelectionSource?: 'session' | 'agent',
		configuration?: SessionConfiguration
	): Promise<boolean>;
	getCapabilityExclusion(occurrence: AgentCapabilityOccurrence): AgentCapabilityExclusion | undefined;
	toggleCapabilityExclusion(occurrence: AgentCapabilityOccurrence): void;
	toggleCapabilityExclusionByPath(path: string): void;
	clearCapabilityExclusions(kinds?: AgentCapabilityExclusion['kind'][]): void;
	isAgentSkillActive(projection: AgentSkillProjection): boolean;
	setAgentSkillActive(projection: AgentSkillProjection, active: boolean): void;
	clearAgentSkillActivityOverrides(): void;
}

function capabilityTargetKey(target: CapabilityTarget): string {
	return sessionCapabilityKey(target);
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

export function useAgentManager(context: AgentComposerContext, options: AgentManagerOptions = {}): AgentManagerState {
	const [agentOptions, setAgentOptions] = useState<AgentCatalogOption[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [actionError, setActionError] = useState<string | null>(null);
	const [isApplying, setIsApplying] = useState(false);
	const [isTrackingOnly, setIsTrackingOnly] = useState(false);
	const [selectedAgentKey, setSelectedAgentKey] = useState<string | null>(null);
	const [configuredDefaultAgentRef, setConfiguredDefaultAgentRef] = useState<ArtifactRef | null>(null);
	const [excludedCapabilityKeys, setExcludedCapabilityKeys] = useState<string[]>([]);
	const [releasedCapabilityExclusionKeys, setReleasedCapabilityExclusionKeys] = useState<string[]>([]);
	const [preparedStarter, setPreparedStarter] = useState<AgentStarterPreview | null>(null);
	const [selectedAgentResolution, setSelectedAgentResolution] = useState<AgentResolution | null>(null);
	const [agentSelectionPinRefreshRequested, setAgentSelectionPinRefreshRequested] = useState(false);
	const [agentSkillActivityOverrides, setAgentSkillActivityOverrides] = useState<SessionActiveSkillOverride[]>([]);
	const [releasedAgentSkillActivityOverrideKeys, setReleasedAgentSkillActivityOverrideKeys] = useState<string[]>([]);

	const selectedAgentKeyRef = useRef<string | null>(null);
	const isApplyingRef = useRef(false);
	const starterApplierRef = useRef<AgentManagerOptions['applyUserMessageStarter']>(options.applyUserMessageStarter);
	const agentInstructionSourcesReplacerRef = useRef<AgentManagerOptions['replaceAgentInstructionSources']>(
		options.replaceAgentInstructionSources
	);
	const agentInstructionSourcesClearerRef = useRef<AgentManagerOptions['clearAgentInstructionSources']>(
		options.clearAgentInstructionSources
	);
	const agentInstructionSourceSelectionRef = useRef<AgentManagerOptions['setAgentInstructionSourceSelected']>(
		options.setAgentInstructionSourceSelected
	);
	const activeAgentRequestAbortControllerRef = useRef<AbortController | null>(null);
	const applyRequestSeqRef = useRef(0);
	const restoreRequestSeqRef = useRef(0);
	const modelResetRequestSeqRef = useRef(0);
	const catalogRequestSeqRef = useRef(0);
	const pendingDefaultActionRef = useRef<PendingDefaultAction | null>(null);
	const initialDefaultRequestRef = useRef(false);
	const autoEnsureCatalogKeyRef = useRef<string | null>(null);
	const pendingRestoredAgentRef = useRef<PendingRestoredAgent | null>(null);
	const excludedCapabilityKeysRef = useRef<string[]>([]);

	useEffect(() => {
		starterApplierRef.current = options.applyUserMessageStarter;
		agentInstructionSourcesReplacerRef.current = options.replaceAgentInstructionSources;
		agentInstructionSourcesClearerRef.current = options.clearAgentInstructionSources;
		agentInstructionSourceSelectionRef.current = options.setAgentInstructionSourceSelected;
	}, [
		options.applyUserMessageStarter,
		options.clearAgentInstructionSources,
		options.replaceAgentInstructionSources,
		options.setAgentInstructionSourceSelected,
	]);

	const applyPreparedStarter = useCallback(
		async (starter: AgentStarterPreview, signal: AbortSignal): Promise<boolean> => {
			if (signal.aborted) {
				return false;
			}

			setPreparedStarter(starter);

			const text = starter.userMessageStarterText?.trim();
			if (text) {
				// This is intentionally best-effort. A non-empty Composer draft
				// wins over an Agent user-message starter.
				starterApplierRef.current?.(text);
			}
			return true;
		},
		[]
	);

	const publishAgentInstructionSources = useCallback((resolution: AgentResolution, starter?: AgentStarterPreview) => {
		const declarations = projectAgentInstructionSources(resolution, starter?.instructionTextArtifacts ?? []);
		agentInstructionSourcesReplacerRef.current?.(declarations);

		const excluded = new Set(excludedCapabilityKeysRef.current);
		for (const declaration of declarations) {
			const occurrence = resolution.capabilities.occurrences.find(
				candidate => candidate.path === declaration.occurrencePath
			);

			if (!occurrence?.target) {
				continue;
			}

			if (excluded.has(capabilityTargetKey(occurrence.target))) {
				agentInstructionSourceSelectionRef.current?.(declaration.occurrencePath, false);
			}
		}
	}, []);

	const replaceExcludedCapabilityKeys = useCallback((keys: readonly string[]) => {
		const next = [
			...new Set(
				keys.filter(key => {
					return key.trim().length > 0;
				})
			),
		];
		excludedCapabilityKeysRef.current = next;
		setExcludedCapabilityKeys(next);
	}, []);

	const setTrackedAgent = useCallback(
		(aKey: string | null, trackingOnly = false, resolution: AgentResolution | null = null) => {
			selectedAgentKeyRef.current = aKey;
			setSelectedAgentKey(aKey);
			setSelectedAgentResolution(resolution);
			setIsTrackingOnly(trackingOnly);
		},
		[]
	);

	const cancelActiveAgentRequest = useCallback(() => {
		applyRequestSeqRef.current += 1;
		activeAgentRequestAbortControllerRef.current?.abort();
		activeAgentRequestAbortControllerRef.current = null;
		isApplyingRef.current = false;
		setIsApplying(false);
	}, []);

	const fetchAgentOptions = useCallback(async (force: boolean, requestSeq: number) => {
		try {
			const [nextOptions, defaultAgent] = await Promise.all([
				agentManagementAPI.listAgentCatalogOptions(force),
				agentManagementAPI.getDefaultAgent().catch(() => null),
			]);
			if (catalogRequestSeqRef.current !== requestSeq) {
				return;
			}

			setAgentOptions(nextOptions);
			setConfiguredDefaultAgentRef(defaultAgent);
		} catch (loadError) {
			if (catalogRequestSeqRef.current !== requestSeq) {
				return;
			}

			setAgentOptions([]);
			setError(getErrorMessage(loadError, 'Failed to load Agents.'));
			setConfiguredDefaultAgentRef(null);
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
		sameAgentDefinition(selectedAgentResolution.agent, selectedAgent.agent)
			? selectedAgentResolution
			: null;

	const baseAgent = useMemo(
		() =>
			agentOptions.find(option => option.agent.builtIn && option.agent.name.trim().toLowerCase() === BASE_AGENT_NAME) ??
			null,
		[agentOptions]
	);
	const configuredDefaultAgent = useMemo(
		() =>
			configuredDefaultAgentRef
				? (agentOptions.find(
						option => option.isSelectable && artifactRefKey(option.ref) === artifactRefKey(configuredDefaultAgentRef)
					) ?? null)
				: null,
		[agentOptions, configuredDefaultAgentRef]
	);
	const defaultAgent = useMemo(
		() =>
			configuredDefaultAgent ??
			(baseAgent?.isSelectable ? baseAgent : (agentOptions.find(option => option.isSelectable) ?? null)),
		[agentOptions, baseAgent, configuredDefaultAgent]
	);
	const isDefaultAgentSelected = selectedAgent !== null && selectedAgent.key === defaultAgent?.key;

	const agentModelRef = useMemo(
		() => resolvedAgentModelRef(currentSelectedAgentResolution),
		[currentSelectedAgentResolution]
	);

	const agentToolProjections = useMemo(
		() => projectAgentToolCapabilities(currentSelectedAgentResolution),
		[currentSelectedAgentResolution]
	);
	const agentSkillProjections = useMemo(
		() => projectAgentSkillCapabilities(currentSelectedAgentResolution),
		[currentSelectedAgentResolution]
	);
	const agentMCPProjections = useMemo(
		() => projectAgentMCPCapabilities(currentSelectedAgentResolution),
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

	const restoreAgentSkillActivityOverrides = useCallback(
		(resolution: AgentResolution, configuration?: SessionConfiguration) => {
			const knownKeys = new Set(projectAgentSkillCapabilities(resolution).map(projection => projection.key));
			const restored = (configuration?.activeSkills ?? []).filter(
				override =>
					knownKeys.has(override.key) &&
					(override.mode === SessionActivityOverrideMode.Activate ||
						override.mode === SessionActivityOverrideMode.Deactivate)
			);
			setAgentSkillActivityOverrides(restored.map(override => structuredClone(override)));
			setReleasedAgentSkillActivityOverrideKeys([]);
		},
		[]
	);

	useEffect(() => {
		if (!selectedAgent || currentSelectedAgentResolution) {
			return;
		}

		let cancelled = false;
		const controller = new AbortController();

		const refreshSelectedAgentResolution = async () => {
			try {
				const resolution = await agentManagementAPI.resolveAgent(selectedAgent.ref);

				if (cancelled || controller.signal.aborted) {
					return;
				}

				let starter: AgentStarterPreview | undefined;
				try {
					starter = await agentManagementAPI.prepareAgentStarterFromResolution(resolution);
				} catch {
					starter = undefined;
				}

				if (cancelled || controller.signal.aborted) {
					return;
				}

				publishAgentInstructionSources(resolution, starter);
				if (context.modelSelectionSource === 'agent') {
					context.setAgentModelPreview(
						resolveAgentModelOption(resolution),
						resolvedAgentModelRef(resolution) !== undefined
					);
				}
				setPreparedStarter(starter ?? null);
				setSelectedAgentResolution(resolution);
			} catch {
				if (!cancelled) {
					setSelectedAgentResolution(null);
				}
			}
		};

		void refreshSelectedAgentResolution();

		return () => {
			cancelled = true;
			controller.abort();
		};
	}, [context, currentSelectedAgentResolution, publishAgentInstructionSources, resolveAgentModelOption, selectedAgent]);

	const hasTrackedAgent = useCallback(() => {
		const trackedKey = selectedAgentKeyRef.current;
		return trackedKey !== null && agentOptions.some(option => option.key === trackedKey);
	}, [agentOptions]);

	const selectAgent = useCallback(
		async (agentRef: ArtifactRef): Promise<boolean> => {
			if (isApplyingRef.current) {
				return false;
			}

			const option = agentOptions.find(value => value.key === artifactRefKey(agentRef));
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

			restoreRequestSeqRef.current += 1;
			pendingRestoredAgentRef.current = null;

			if (selectedAgentKeyRef.current !== option.key) {
				setSelectedAgentResolution(null);
				agentInstructionSourcesClearerRef.current?.();
			}

			const requestSequence = applyRequestSeqRef.current + 1;
			const isReapplyingCurrentAgent = selectedAgentKeyRef.current === option.key;
			const requestAbortController = new AbortController();
			const isCurrentRequest = () =>
				applyRequestSeqRef.current === requestSequence && !requestAbortController.signal.aborted;

			applyRequestSeqRef.current = requestSequence;
			activeAgentRequestAbortControllerRef.current = requestAbortController;
			isApplyingRef.current = true;
			setIsApplying(true);
			setActionError(null);

			try {
				const resolution = await agentManagementAPI.resolveAgent(agentRef);
				if (!isCurrentRequest()) {
					return false;
				}

				const unavailableRequired = resolution.capabilities.occurrences.find(
					occurrence => occurrence.required && occurrence.status !== CapabilityResolutionStatus.Available
				);
				if (unavailableRequired) {
					setActionError(unavailableRequired.message || 'A required Agent capability is unavailable.');
					return false;
				}

				const starter = await agentManagementAPI.prepareAgentStarterFromResolution(resolution);
				if (!isCurrentRequest()) {
					return false;
				}
				if (!starter.canApply) {
					const issue = starter.issues.find(value => value.severity === 'error');
					setActionError(issue?.message || 'The Agent starter could not be prepared.');
					return false;
				}

				const starterApplied = await applyPreparedStarter(starter, requestAbortController.signal);
				if (!starterApplied || !isCurrentRequest()) {
					return false;
				}

				const localAgentModelRef = resolvedAgentModelRef(resolution);
				context.setAgentModelPreview(resolveAgentModelOption(resolution), localAgentModelRef !== undefined);
				if (!isReapplyingCurrentAgent) {
					replaceExcludedCapabilityKeys([]);
					setReleasedCapabilityExclusionKeys([]);
					setAgentSkillActivityOverrides([]);
					setReleasedAgentSkillActivityOverrideKeys([]);
				}
				publishAgentInstructionSources(resolution, starter);
				setTrackedAgent(option.key, false, resolution);
				return true;
			} catch (e: unknown) {
				if (isCurrentRequest()) {
					setActionError(getErrorMessage(e, 'Failed to validate the Agent selection.'));
				}
				return false;
			} finally {
				if (activeAgentRequestAbortControllerRef.current === requestAbortController) {
					activeAgentRequestAbortControllerRef.current = null;
				}
				if (applyRequestSeqRef.current === requestSequence) {
					isApplyingRef.current = false;
					setIsApplying(false);
				}
			}
		},
		[
			agentOptions,
			applyPreparedStarter,
			context,
			publishAgentInstructionSources,
			replaceExcludedCapabilityKeys,
			resolveAgentModelOption,
			setTrackedAgent,
		]
	);

	const restoreSelectedAgent = useCallback(
		async (
			agent?: ArtifactRef,
			modelSelectionSource: 'session' | 'agent' = 'agent',
			configuration?: SessionConfiguration
		): Promise<boolean> => {
			const restoreRequest = restoreRequestSeqRef.current + 1;
			restoreRequestSeqRef.current = restoreRequest;
			cancelActiveAgentRequest();
			const exclusionKeys = agentRemovalKeys(configuration);
			agentInstructionSourcesClearerRef.current?.();
			setReleasedCapabilityExclusionKeys([]);
			setReleasedAgentSkillActivityOverrideKeys([]);

			// Do not claim Agent ownership until its resolved capability plan proves it.
			context.setModelSelectionSource('session');

			if (!agent) {
				pendingRestoredAgentRef.current = null;
				setPreparedStarter(null);
				replaceExcludedCapabilityKeys([]);
				setAgentSkillActivityOverrides([]);
				setTrackedAgent(null);
				setActionError(null);
				setAgentSelectionPinRefreshRequested(false);
				return true;
			}

			if (loading) {
				pendingRestoredAgentRef.current = {
					agent,
					modelSelectionSource,
					exclusionKeys,
					configuration: configuration ? structuredClone(configuration) : undefined,
				};
				return false;
			}

			const option = agentOptions.find(candidate => candidate.key === artifactRefKey(agent));
			if (!option) {
				pendingRestoredAgentRef.current = null;
				replaceExcludedCapabilityKeys([]);
				setPreparedStarter(null);
				setAgentSkillActivityOverrides([]);
				setTrackedAgent(null);
				setActionError('The Session-selected Agent is no longer available in the current catalog.');
				return false;
			}

			pendingRestoredAgentRef.current = null;
			try {
				const resolution = await agentManagementAPI.resolveAgent(agent);
				if (restoreRequestSeqRef.current !== restoreRequest) {
					return false;
				}

				const starter = await agentManagementAPI.prepareAgentStarterFromResolution(resolution).catch(() => undefined);
				if (restoreRequestSeqRef.current !== restoreRequest) {
					return false;
				}

				replaceExcludedCapabilityKeys(exclusionKeys);
				restoreAgentSkillActivityOverrides(resolution, configuration);
				publishAgentInstructionSources(resolution, starter);
				if (modelSelectionSource === 'agent') {
					const localAgentModelRef = resolvedAgentModelRef(resolution);
					context.setAgentModelPreview(resolveAgentModelOption(resolution), localAgentModelRef !== undefined);
				}
				setTrackedAgent(option.key, true, resolution);
			} catch {
				if (restoreRequestSeqRef.current !== restoreRequest) {
					return false;
				}

				// A stale/missing Agent must not leave the Model control falsely
				// marked as Agent-owned.
				context.setModelSelectionSource('session');
				replaceExcludedCapabilityKeys(exclusionKeys);
				setAgentSkillActivityOverrides([]);
				setTrackedAgent(option.key, true);
			}
			if (restoreRequestSeqRef.current !== restoreRequest) {
				return false;
			}
			setActionError(null);
			setAgentSelectionPinRefreshRequested(false);
			return true;
		},
		[
			agentOptions,
			cancelActiveAgentRequest,
			context,
			loading,
			publishAgentInstructionSources,
			replaceExcludedCapabilityKeys,
			resolveAgentModelOption,
			restoreAgentSkillActivityOverrides,
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

		const applied = await selectAgent(option.ref);
		if (applied) {
			setAgentSelectionPinRefreshRequested(true);
		}
		return applied;
	}, [agentOptions, selectAgent]);

	const acknowledgeAgentSelectionPinRefresh = useCallback(() => {
		setAgentSelectionPinRefreshRequested(false);
	}, []);

	const resetModelToAgent = useCallback(async (): Promise<boolean> => {
		const option = selectedAgentKeyRef.current
			? agentOptions.find(candidate => candidate.key === selectedAgentKeyRef.current)
			: undefined;
		if (!option) {
			setActionError('No active Agent model is available.');
			return false;
		}

		const request = modelResetRequestSeqRef.current + 1;
		modelResetRequestSeqRef.current = request;

		try {
			const resolution = await agentManagementAPI.resolveAgent(option.ref);
			if (modelResetRequestSeqRef.current !== request || selectedAgentKeyRef.current !== option.key) {
				return false;
			}

			context.setAgentModelPreview(
				resolveAgentModelOption(resolution),
				resolvedAgentModelRef(resolution) !== undefined
			);

			if (!sameAgentDefinition(currentSelectedAgentResolution?.agent, resolution.agent)) {
				setAgentSelectionPinRefreshRequested(true);
			}

			setSelectedAgentResolution(resolution);
			setActionError(null);
			return true;
		} catch (cause) {
			setActionError(getErrorMessage(cause, 'The Agent model could not be restored.'));
			return false;
		}
	}, [agentOptions, context, currentSelectedAgentResolution, resolveAgentModelOption]);

	const trackDefaultAgentWithoutApplying = useCallback(async (): Promise<boolean> => {
		if (hasTrackedAgent()) {
			pendingDefaultActionRef.current = null;
			return true;
		}

		// A restore must invalidate a still-running initial default application,
		// otherwise it can overwrite restored Composer state when it resolves.
		cancelActiveAgentRequest();
		agentInstructionSourcesClearerRef.current?.();

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
		restoreRequestSeqRef.current += 1;
		modelResetRequestSeqRef.current += 1;
		pendingDefaultActionRef.current = null;
		pendingRestoredAgentRef.current = null;
		agentInstructionSourcesClearerRef.current?.();
		autoEnsureCatalogKeyRef.current = null;
		context.setModelSelectionSource('session');
		replaceExcludedCapabilityKeys([]);
		setPreparedStarter(null);
		setReleasedCapabilityExclusionKeys([]);
		setAgentSkillActivityOverrides([]);
		setReleasedAgentSkillActivityOverrideKeys([]);
		setAgentSelectionPinRefreshRequested(false);
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

	const toggleCapabilityExclusion = useCallback(
		(occurrence: AgentCapabilityOccurrence) => {
			const exclusion = capabilityExclusionForOccurrence(occurrence);
			if (!exclusion) {
				setActionError(
					'This Agent contribution cannot be removed because it has no canonical Artifact or capability target.'
				);
				return;
			}

			const wasExcluded = excludedCapabilityKeys.includes(exclusion.key);
			if (
				exclusion.kind === 'skill' &&
				exclusion.target.form === CapabilityTargetForm.Artifact &&
				exclusion.target.artifact
			) {
				agentInstructionSourceSelectionRef.current?.(occurrence.path, wasExcluded);
			}
			if (exclusion.kind === 'instruction') {
				agentInstructionSourceSelectionRef.current?.(occurrence.path, wasExcluded);
			}

			setReleasedCapabilityExclusionKeys(previous =>
				wasExcluded ? [...new Set([...previous, exclusion.key])] : previous.filter(key => key !== exclusion.key)
			);

			setExcludedCapabilityKeys(previous => {
				const next = previous.includes(exclusion.key)
					? previous.filter(key => key !== exclusion.key)
					: [...previous, exclusion.key];
				excludedCapabilityKeysRef.current = next;
				return next;
			});
		},
		[excludedCapabilityKeys]
	);

	const toggleCapabilityExclusionByPath = useCallback(
		(path: string) => {
			const occurrence = currentSelectedAgentResolution?.capabilities.occurrences.find(
				candidate => candidate.path === path
			);
			if (occurrence) {
				toggleCapabilityExclusion(occurrence);
			}
		},
		[currentSelectedAgentResolution, toggleCapabilityExclusion]
	);

	const clearCapabilityExclusions = useCallback(
		(kinds?: AgentCapabilityExclusion['kind'][]) => {
			const allowedKinds = kinds ? new Set(kinds) : undefined;
			const releasable = new Set<string>();

			for (const occurrence of currentSelectedAgentResolution?.capabilities.occurrences ?? []) {
				const exclusion = capabilityExclusionForOccurrence(occurrence);
				if (
					!exclusion ||
					!excludedCapabilityKeysRef.current.includes(exclusion.key) ||
					(allowedKinds && !allowedKinds.has(exclusion.kind))
				) {
					continue;
				}

				releasable.add(exclusion.key);
				if (exclusion.kind === 'instruction') {
					agentInstructionSourceSelectionRef.current?.(occurrence.path, true);
				}
			}

			if (releasable.size === 0) {
				return;
			}

			setReleasedCapabilityExclusionKeys(previous => [...new Set([...previous, ...releasable])]);
			setExcludedCapabilityKeys(previous => {
				const next = previous.filter(key => !releasable.has(key));
				excludedCapabilityKeysRef.current = next;
				return next;
			});
		},
		[currentSelectedAgentResolution]
	);

	const isAgentSkillActive = useCallback(
		(projection: AgentSkillProjection) =>
			effectiveAgentSkillActive(
				projection,
				agentSkillActivityOverrides.find(override => override.key === projection.key)
			),
		[agentSkillActivityOverrides]
	);

	const setAgentSkillActive = useCallback((projection: AgentSkillProjection, active: boolean) => {
		const change = agentSkillActivityChange(projection, active);
		const releasedOverrideKey = change.releasedOverrideKey;
		if (releasedOverrideKey) {
			setReleasedAgentSkillActivityOverrideKeys(previous => [...new Set([...previous, releasedOverrideKey])]);
			setAgentSkillActivityOverrides(previous =>
				previous.filter(override => override.key !== change.releasedOverrideKey)
			);
			return;
		}
		const nextOverride = change.override;
		if (!nextOverride) {
			return;
		}

		setReleasedAgentSkillActivityOverrideKeys(previous => previous.filter(key => key !== nextOverride?.key));
		setAgentSkillActivityOverrides(previous => [
			...previous.filter(override => override.key !== nextOverride.key),
			nextOverride,
		]);
	}, []);

	const clearAgentSkillActivityOverrides = useCallback(() => {
		setAgentSkillActivityOverrides(previous => {
			if (previous.length === 0) {
				return previous;
			}
			setReleasedAgentSkillActivityOverrideKeys(released => [
				...new Set([...released, ...previous.map(override => override.key)]),
			]);
			return [];
		});
	}, []);

	const catalogIdentity = useMemo(
		() =>
			[
				...agentOptions.map(option => `${option.key}:${option.isSelectable ? '1' : '0'}`),
				configuredDefaultAgentRef ? artifactRefKey(configuredDefaultAgentRef) : 'automatic',
			].join('|'),
		[agentOptions, configuredDefaultAgentRef]
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
				restoreSelectedAgent(
					pendingRestoredAgent.agent,
					pendingRestoredAgent.modelSelectionSource,
					pendingRestoredAgent.configuration
				)
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
		preparedStarter,
		agentModelRef,
		isTrackingOnly: selectedAgent ? isTrackingOnly : false,
		capabilityExclusions,
		releasedCapabilityExclusionKeys,
		agentSkillActivityOverrides,
		releasedAgentSkillActivityOverrideKeys,
		agentSelectionPinRefreshRequested,
		agentToolProjections,
		agentSkillProjections,
		agentMCPProjections,
		refreshAgents,
		selectAgent,
		ensureDefaultAgent,
		resetToDefaultAgent,
		reapplySelectedAgent,
		resetModelToAgent,
		acknowledgeAgentSelectionPinRefresh,
		trackDefaultAgentWithoutApplying,
		clearAgentTracking,
		restoreSelectedAgent,
		getCapabilityExclusion,
		toggleCapabilityExclusion,
		toggleCapabilityExclusionByPath,
		clearCapabilityExclusions,
		isAgentSkillActive,
		setAgentSkillActive,
		clearAgentSkillActivityOverrides,
	};
}
