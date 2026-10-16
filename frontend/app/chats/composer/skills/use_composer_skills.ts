import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useMemo, useRef, useState } from 'react';

import type { SessionArtifactSelection } from '@/spec/session_base';
import type { SkillRef } from '@/spec/skill';
import type { WorkspaceRef } from '@/spec/workspace';
import { ArtifactState } from '@/spec/artifact';
import { SkillInsert, SkillSessionSyncMode } from '@/spec/skill';

import { getErrorMessage } from '@/lib/error_utils';
import { resolveStateUpdate } from '@/lib/hook_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import type { ComposerSkillCatalogItem } from '@/apis/skill_management';
import { skillManagementAPI } from '@/apis/baseapi';

import {
	areSkillRefListsEqual,
	clampActiveSkillRefsToEnabled,
	normalizeSkillRefs,
	skillRefKey,
} from '@/skills/lib/skill_identity_utils';

interface ApplySkillSelectionStateOptions {
	syncSession?: SkillSessionSyncMode;
	forceResetSession?: boolean;
	workspace?: WorkspaceRef | null;
	workspaceSkillRefs?: SkillRef[];
}

interface UseComposerSkillsResult {
	allSkills: ComposerSkillCatalogItem[];
	templateSkills: ComposerSkillCatalogItem[];
	skillsLoading: boolean;
	skillsLoadError: string | null;
	enabledSkillRefs: SkillRef[];
	activeSkillRefs: SkillRef[];
	installedEnabledSkillRefs: SkillRef[];
	installedActiveSkillRefs: SkillRef[];
	setEnabledSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	enableAllSkills: () => void;
	disableAllSkills: () => void;
	refreshSkills: () => Promise<void>;
	applySkillSelectionState: (
		nextEnabledInput: SkillRef[] | null | undefined,
		nextActiveInput: SkillRef[] | null | undefined,
		options?: ApplySkillSelectionStateOptions
	) => Promise<void>;
	applyWorkspaceSkillSelectionState: (
		workspace: WorkspaceRef | undefined,
		nextWorkspaceEnabledInput: SkillRef[] | null | undefined,
		nextWorkspaceActiveInput: SkillRef[] | null | undefined,
		options?: Omit<ApplySkillSelectionStateOptions, 'workspace' | 'workspaceSkillRefs'>
	) => Promise<void>;
	applyInstalledSkillSelectionState: (
		nextEnabledInput: SkillRef[] | null | undefined,
		nextActiveInput: SkillRef[] | null | undefined,
		options?: ApplySkillSelectionStateOptions
	) => Promise<void>;
	getCurrentEnabledSkillRefs: () => SkillRef[];
	getCurrentActiveSkillRefs: () => SkillRef[];
	getCurrentInstalledEnabledSkillRefs: () => SkillRef[];
	getCurrentInstalledActiveSkillRefs: () => SkillRef[];
	getCurrentInstalledEnabledSkillSelections: () => SessionArtifactSelection[];
	getCurrentActiveSkillSelections: () => SessionArtifactSelection[];
	getCurrentInstalledActiveSkillSelections: () => SessionArtifactSelection[];
}

function catalogItemRef(item: ComposerSkillCatalogItem): SkillRef {
	return {
		rootID: item.ref.rootID,
		artifactID: item.ref.artifactID,
	};
}

function catalogItemSelection(item: ComposerSkillCatalogItem): SessionArtifactSelection {
	return {
		rootID: item.ref.rootID,
		artifactID: item.ref.artifactID,
		expectedRevision: item.revision,
		...(item.definitionDigest ? { expectedDigest: item.definitionDigest } : {}),
	};
}

function stringSetsEqual(left: ReadonlySet<string>, right: ReadonlySet<string>): boolean {
	if (left.size !== right.size) {
		return false;
	}

	for (const value of left) {
		if (!right.has(value)) {
			return false;
		}
	}
	return true;
}

function isInstructionSkill(item: ComposerSkillCatalogItem): boolean {
	return item.insert === SkillInsert.Instructions;
}

export function useComposerSkills(): UseComposerSkillsResult {
	const [enabledSkillRefs, setEnabledSkillRefsRaw] = useState<SkillRef[]>([]);
	const [activeSkillRefs, setActiveSkillRefsRaw] = useState<SkillRef[]>([]);
	const [workspaceSkillRefKeys, setWorkspaceSkillRefKeys] = useState<ReadonlySet<string>>(() => new Set());

	const loadComposerSkills = useCallback(async (_signal: AbortSignal) => {
		return skillManagementAPI.listComposerSkillCatalog();
	}, []);

	const {
		data: composerSkills,
		error: catalogError,
		isLoading: skillsLoading,
		reloadOrThrow,
	} = useAsyncResource(loadComposerSkills, {
		initialData: [] satisfies ComposerSkillCatalogItem[],
	});

	const skillsLoadError = useMemo(() => {
		return catalogError ? getErrorMessage(catalogError, 'Failed to load the Skills catalog.') : null;
	}, [catalogError]);

	const enabledSkillRefsRef = useRef<SkillRef[]>([]);
	const activeSkillRefsRef = useRef<SkillRef[]>([]);
	const workspaceRefRef = useRef<WorkspaceRef | undefined>(undefined);
	const workspaceSkillRefKeysRef = useRef<Set<string>>(new Set());

	const runtimeResolvedSkills = useMemo(() => composerSkills.filter(skill => skill.runtimeResolved), [composerSkills]);

	const allSkills = useMemo(
		() =>
			composerSkills.filter(s => {
				return s.insert !== SkillInsert.UserMessage;
			}),
		[composerSkills]
	);
	const templateSkills = useMemo(
		() => composerSkills.filter(skill => skill.insert === SkillInsert.UserMessage),
		[composerSkills]
	);

	const catalogByRefKey = useMemo(
		() => new Map(runtimeResolvedSkills.map(skill => [skillRefKey(catalogItemRef(skill)), skill])),
		[runtimeResolvedSkills]
	);

	const updateEnabledSkillRefsState = useCallback((next: SkillRef[]) => {
		enabledSkillRefsRef.current = next;
		setEnabledSkillRefsRaw(prev => (areSkillRefListsEqual(prev, next) ? prev : next));
	}, []);

	const updateActiveSkillRefsState = useCallback((next: SkillRef[]) => {
		const clampedNext = clampActiveSkillRefsToEnabled(enabledSkillRefsRef.current, next);
		activeSkillRefsRef.current = clampedNext;
		setActiveSkillRefsRaw(prev => (areSkillRefListsEqual(prev, clampedNext) ? prev : clampedNext));
	}, []);

	const updateWorkspaceSkillRefKeysState = useCallback((next: ReadonlySet<string>) => {
		const copied = new Set(next);
		workspaceSkillRefKeysRef.current = copied;
		setWorkspaceSkillRefKeys(previous => {
			return stringSetsEqual(previous, copied) ? previous : copied;
		});
	}, []);

	const getCurrentEnabledSkillRefs = useCallback(() => {
		return enabledSkillRefsRef.current;
	}, []);

	const getCurrentActiveSkillRefs = useCallback(() => {
		return activeSkillRefsRef.current;
	}, []);

	const getCurrentInstalledEnabledSkillRefs = useCallback(() => {
		const workspaceKeys = workspaceSkillRefKeysRef.current;
		return enabledSkillRefsRef.current.filter(ref => !workspaceKeys.has(skillRefKey(ref)));
	}, []);

	const getCurrentInstalledActiveSkillRefs = useCallback(() => {
		const workspaceKeys = workspaceSkillRefKeysRef.current;
		return activeSkillRefsRef.current.filter(ref => !workspaceKeys.has(skillRefKey(ref)));
	}, []);

	const installedEnabledSkillRefs = useMemo(
		() => enabledSkillRefs.filter(ref => !workspaceSkillRefKeys.has(skillRefKey(ref))),
		[enabledSkillRefs, workspaceSkillRefKeys]
	);

	const installedActiveSkillRefs = useMemo(
		() => activeSkillRefs.filter(ref => !workspaceSkillRefKeys.has(skillRefKey(ref))),
		[activeSkillRefs, workspaceSkillRefKeys]
	);

	const applySkillSelectionState = useCallback(
		async (
			nextEnabledInput: SkillRef[] | null | undefined,
			nextActiveInput: SkillRef[] | null | undefined,
			options?: ApplySkillSelectionStateOptions
		) => {
			const replacesWorkspace = options?.workspace !== undefined;
			const replacesWorkspaceSkillRefs = options?.workspaceSkillRefs !== undefined;
			const previousWorkspaceSkillRefKeys = workspaceSkillRefKeysRef.current;
			const nextWorkspaceSkillRefs = replacesWorkspaceSkillRefs
				? normalizeSkillRefs(options?.workspaceSkillRefs)
				: undefined;
			const nextWorkspaceSkillRefKeys = nextWorkspaceSkillRefs
				? new Set(
						nextWorkspaceSkillRefs.map(k => {
							return skillRefKey(k);
						})
					)
				: previousWorkspaceSkillRefKeys;

			workspaceRefRef.current = replacesWorkspace ? (options?.workspace ?? undefined) : workspaceRefRef.current;
			if (replacesWorkspaceSkillRefs) {
				updateWorkspaceSkillRefKeysState(nextWorkspaceSkillRefKeys);
			}
			const nextEnabled = normalizeSkillRefs(nextEnabledInput);
			updateEnabledSkillRefsState(nextEnabled);
			updateActiveSkillRefsState(clampActiveSkillRefsToEnabled(nextEnabled, nextActiveInput));
		},
		[updateActiveSkillRefsState, updateEnabledSkillRefsState, updateWorkspaceSkillRefKeysState]
	);

	const applyWorkspaceSkillSelectionState = useCallback(
		async (
			nextWorkspace: WorkspaceRef | undefined,
			nextWorkspaceEnabledInput: SkillRef[] | null | undefined,
			nextWorkspaceActiveInput: SkillRef[] | null | undefined,
			options?: Omit<ApplySkillSelectionStateOptions, 'workspace' | 'workspaceSkillRefs'>
		) => {
			const previousWorkspaceKeys = workspaceSkillRefKeysRef.current;
			const retainedInstalledEnabled = enabledSkillRefsRef.current.filter(
				ref => !previousWorkspaceKeys.has(skillRefKey(ref))
			);
			const retainedInstalledActive = activeSkillRefsRef.current.filter(
				ref => !previousWorkspaceKeys.has(skillRefKey(ref))
			);
			const nextWorkspaceEnabled = normalizeSkillRefs(nextWorkspaceEnabledInput);
			const nextWorkspaceEnabledKeys = new Set(
				nextWorkspaceEnabled.map(k => {
					return skillRefKey(k);
				})
			);
			const nextWorkspaceActive = normalizeSkillRefs(nextWorkspaceActiveInput).filter(ref =>
				nextWorkspaceEnabledKeys.has(skillRefKey(ref))
			);

			await applySkillSelectionState(
				[...retainedInstalledEnabled, ...nextWorkspaceEnabled],
				[...retainedInstalledActive, ...nextWorkspaceActive],
				{
					...options,
					workspace: nextWorkspace ?? null,
					workspaceSkillRefs: nextWorkspaceEnabled,
				}
			);
		},
		[applySkillSelectionState]
	);

	const applyInstalledSkillSelectionState = useCallback(
		async (
			nextInstalledEnabledInput: SkillRef[] | null | undefined,
			nextInstalledActiveInput: SkillRef[] | null | undefined,
			options?: ApplySkillSelectionStateOptions
		) => {
			const workspaceKeys = workspaceSkillRefKeysRef.current;
			const retainedWorkspaceEnabled = getCurrentEnabledSkillRefs().filter(ref => workspaceKeys.has(skillRefKey(ref)));
			const retainedWorkspaceActive = getCurrentActiveSkillRefs().filter(ref => workspaceKeys.has(skillRefKey(ref)));
			const nextInstalledEnabled = normalizeSkillRefs(nextInstalledEnabledInput).filter(
				ref => !workspaceKeys.has(skillRefKey(ref))
			);
			const nextInstalledActive = normalizeSkillRefs(nextInstalledActiveInput).filter(
				ref => !workspaceKeys.has(skillRefKey(ref))
			);

			await applySkillSelectionState(
				[...nextInstalledEnabled, ...retainedWorkspaceEnabled],
				[...nextInstalledActive, ...retainedWorkspaceActive],
				options
			);
		},
		[applySkillSelectionState, getCurrentActiveSkillRefs, getCurrentEnabledSkillRefs]
	);

	const setEnabledSkillRefs = useCallback<Dispatch<SetStateAction<SkillRef[]>>>(
		update => {
			const prevEnabled = enabledSkillRefsRef.current;
			const nextEnabled = resolveStateUpdate(update, prevEnabled);
			void applySkillSelectionState(nextEnabled, activeSkillRefsRef.current, {
				syncSession: SkillSessionSyncMode.None,
			});
		},
		[applySkillSelectionState]
	);

	const setActiveSkillRefs = useCallback<Dispatch<SetStateAction<SkillRef[]>>>(
		update => {
			const prevActive = activeSkillRefsRef.current;
			const nextActive = resolveStateUpdate(update, prevActive);
			void applySkillSelectionState(enabledSkillRefsRef.current, nextActive, {
				syncSession: SkillSessionSyncMode.None,
			});
		},
		[applySkillSelectionState]
	);

	const refreshSkills = useCallback(async () => {
		skillManagementAPI.invalidateComposerSkillsCatalog();
		try {
			await reloadOrThrow();
		} catch {
			// useAsyncResource retains the error for the Composer controls.
		}
	}, [reloadOrThrow]);

	const enableAllSkills = useCallback(() => {
		const workspaceKeys = workspaceSkillRefKeysRef.current;
		const selectable = allSkills
			.filter(
				skill =>
					skill.enabled && skill.runtimeResolved && skill.state === ArtifactState.Available && isInstructionSkill(skill)
			)
			.map(s => {
				return catalogItemRef(s);
			});

		void applySkillSelectionState(
			[...enabledSkillRefsRef.current.filter(ref => workspaceKeys.has(skillRefKey(ref))), ...selectable],
			activeSkillRefsRef.current,
			{ syncSession: SkillSessionSyncMode.None }
		);
	}, [allSkills, applySkillSelectionState]);

	const disableAllSkills = useCallback(() => {
		const workspaceKeys = workspaceSkillRefKeysRef.current;
		const nextEnabled = enabledSkillRefsRef.current.filter(ref => workspaceKeys.has(skillRefKey(ref)));
		const nextActive = activeSkillRefsRef.current.filter(ref => workspaceKeys.has(skillRefKey(ref)));
		void applySkillSelectionState(nextEnabled, nextActive, {
			syncSession: SkillSessionSyncMode.None,
		});
	}, [applySkillSelectionState]);

	const selectionForRef = useCallback(
		(ref: SkillRef): SessionArtifactSelection => {
			const item = catalogByRefKey.get(skillRefKey(ref));
			return item ? catalogItemSelection(item) : { rootID: ref.rootID, artifactID: ref.artifactID };
		},
		[catalogByRefKey]
	);

	return {
		allSkills,
		templateSkills,
		skillsLoading,
		skillsLoadError,
		enabledSkillRefs,
		activeSkillRefs,
		installedEnabledSkillRefs,
		installedActiveSkillRefs,
		setEnabledSkillRefs,
		setActiveSkillRefs,
		enableAllSkills,
		disableAllSkills,
		refreshSkills,
		applySkillSelectionState,
		applyWorkspaceSkillSelectionState,
		applyInstalledSkillSelectionState,
		getCurrentEnabledSkillRefs,
		getCurrentActiveSkillRefs,
		getCurrentInstalledEnabledSkillRefs,
		getCurrentInstalledActiveSkillRefs,
		getCurrentInstalledEnabledSkillSelections: () =>
			getCurrentInstalledEnabledSkillRefs().map(r => {
				return selectionForRef(r);
			}),
		getCurrentActiveSkillSelections: () =>
			getCurrentActiveSkillRefs().map(r => {
				return selectionForRef(r);
			}),
		getCurrentInstalledActiveSkillSelections: () =>
			getCurrentInstalledActiveSkillRefs().map(r => {
				return selectionForRef(r);
			}),
	};
}
