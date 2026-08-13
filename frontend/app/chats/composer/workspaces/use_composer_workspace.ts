import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { ArtifactRef } from '@/spec/artifact';
import { ArtifactState } from '@/spec/artifact';
import type { SkillRef } from '@/spec/skill';
import { SkillSessionSyncMode } from '@/spec/skill';
import type {
	CreateFilesystemWorkspaceInput,
	WorkspaceContextView,
	WorkspaceConversationResourceSelectionRef,
	WorkspaceConversationSelection,
	WorkspaceConversationSkillSelectionRef,
	WorkspaceRef,
	WorkspaceSkillView,
	WorkspaceView,
} from '@/spec/workspace';
import { WorkspaceSkillInsert } from '@/spec/workspace';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { workspaceAPI } from '@/apis/baseapi';

import type { LoadedWorkspaceSelectionCatalog } from '@/chats/composer/workspaces/workspace_selection_loader';
import { loadWorkspaceSelectionCatalog } from '@/chats/composer/workspaces/workspace_selection_loader';
import { normalizeSkillRefs, skillRefKey } from '@/skills/lib/skill_identity_utils';
import {
	artifactRefKey,
	createFilesystemWorkspaceCollection,
	listAllWorkspaces,
	workspaceRefKey,
	workspaceRefsEqual,
} from '@/workspaces/lib/workspace_api_utils';
import { sortWorkspaces } from '@/workspaces/lib/workspace_utils';

interface SkillSelectionApplyOptions {
	syncSession?: SkillSessionSyncMode;
	forceResetSession?: boolean;
}

interface UseComposerWorkspaceArgs {
	applyWorkspaceSkillSelectionState: (
		workspace: WorkspaceRef | undefined,
		workspaceEnabled: SkillRef[],
		workspaceActive: SkillRef[],
		options?: SkillSelectionApplyOptions
	) => Promise<void>;
	getCurrentActiveSkillRefs: () => SkillRef[];
}

export interface ComposerWorkspaceController {
	workspaces: WorkspaceView[];
	workspacesLoading: boolean;
	workspacesLoadError: string | null;
	selection: WorkspaceConversationSelection | undefined;
	workspace: WorkspaceView | undefined;
	contexts: WorkspaceContextView[];
	skills: WorkspaceSkillView[];
	selectionLoading: boolean;
	catalogKnown: boolean;
	catalogRevision?: number;
	selectionError: string | null;
	blockingError: string | null;
	selectedContextIDs: Set<string>;
	selectedSkillIDs: Set<string>;
	missingContextRefs: WorkspaceConversationResourceSelectionRef[];
	missingSkillRefs: WorkspaceConversationSkillSelectionRef[];
	changedCount: number;
	attentionCount: number;
	refreshWorkspaces: () => Promise<void>;
	attachWorkspace: (workspace: WorkspaceView) => Promise<void>;
	restoreSelection: (selection?: WorkspaceConversationSelection, syncSkills?: boolean) => Promise<void>;
	detachWorkspace: (syncSkills?: boolean) => Promise<void>;
	updateSelectionFromCurrentContents: () => Promise<void>;
	toggleContext: (context: WorkspaceContextView, selected: boolean) => void;
	toggleSkill: (skill: WorkspaceSkillView, selected: boolean) => Promise<void>;
	removeContextRef: (artifact: ArtifactRef) => void;
	removeSkillRef: (artifact: ArtifactRef) => Promise<void>;
	refreshSelectedWorkspace: () => Promise<void>;
	createFilesystemWorkspace: (payload: CreateFilesystemWorkspaceInput) => Promise<void>;
	getSelectionSnapshot: () => WorkspaceConversationSelection | undefined;
}

function contextIsEligible(context: WorkspaceContextView): boolean {
	return (
		context.enabled &&
		context.state === ArtifactState.Available &&
		context.catalogCurrent &&
		context.projectionValid &&
		!context.runtimeDisabled
	);
}

export function isWorkspaceSkillConversationAvailable(skill: WorkspaceSkillView): boolean {
	return (
		skill.skill.isEnabled &&
		skill.state === ArtifactState.Available &&
		skill.projectionValid &&
		skill.catalogCurrent &&
		!skill.runtimeDisabled
	);
}

function isWorkspaceSkillSessionEligible(skill: WorkspaceSkillView): boolean {
	return skill.skill.insert === WorkspaceSkillInsert.Instructions && isWorkspaceSkillConversationAvailable(skill);
}

function resolveWorkspaceSessionSkillRefs(
	selection: WorkspaceConversationSelection | undefined,
	workspaceSkills: WorkspaceSkillView[]
): SkillRef[] {
	if (!selection) {
		return [];
	}

	const skillsByArtifactKey = new Map(
		workspaceSkills
			.filter(skill => workspaceRefsEqual(skill.workspace, selection.workspace))
			.map(skill => [artifactRefKey(skill.artifact), skill] as const)
	);
	const refs: SkillRef[] = [];

	for (const selectionRef of selection.skillRefs ?? []) {
		const key = artifactRefKey(selectionRef.artifact);
		const skill = skillsByArtifactKey.get(key);

		if (!skill || !isWorkspaceSkillSessionEligible(skill)) {
			continue;
		}
		refs.push(skill.artifact);
	}

	return normalizeSkillRefs(refs);
}

function contextSelectionRef(context: WorkspaceContextView): WorkspaceConversationResourceSelectionRef {
	return {
		artifact: { ...context.artifact },
		name: context.name,
		locator: context.locator,
		definitionDigest: context.definitionDigest,
		artifactRevision: context.recordRevision,
	};
}

function skillSelectionRef(skill: WorkspaceSkillView): WorkspaceConversationSkillSelectionRef {
	return {
		artifact: { ...skill.artifact },
		name: skill.skill.name,
		displayName: skill.skill.displayName,
		locator: skill.locator,
		definitionDigest: skill.definitionDigest,
		artifactRevision: skill.recordRevision,
		insert: skill.skill.insert,
	};
}

function cloneSelection(
	selection: WorkspaceConversationSelection | undefined
): WorkspaceConversationSelection | undefined {
	if (!selection) {
		return undefined;
	}
	return {
		...selection,
		workspace: { ...selection.workspace },
		// oxlint-disable-next-line oxc/no-map-spread
		contextRefs: (selection.contextRefs ?? []).map(ref => ({
			...ref,
			artifact: { ...ref.artifact },
		})),
		// oxlint-disable-next-line oxc/no-map-spread
		skillRefs: (selection.skillRefs ?? []).map(ref => ({
			...ref,
			artifact: { ...ref.artifact },
		})),
	};
}

function getErrorMessage(error: unknown, fallback: string): string {
	return error instanceof Error && error.message.trim() ? error.message : fallback;
}

async function loadComposerWorkspaceList(signal: AbortSignal): Promise<WorkspaceView[]> {
	const loaded = await listAllWorkspaces();
	throwIfAborted(signal);
	return sortWorkspaces(loaded);
}

export function useComposerWorkspace({
	applyWorkspaceSkillSelectionState,
	getCurrentActiveSkillRefs,
}: UseComposerWorkspaceArgs): ComposerWorkspaceController {
	const loadWorkspaceList = useCallback((signal: AbortSignal) => loadComposerWorkspaceList(signal), []);
	const {
		data: workspaces,
		error: workspaceListError,
		isLoading: isInitialWorkspaceListLoading,
		isRefreshing: isWorkspaceListRefreshing,
		reloadOrThrow: reloadWorkspaceList,
		setData: setWorkspaces,
	} = useAsyncResource(loadWorkspaceList, {
		initialData: [] as WorkspaceView[],
	});
	const workspacesLoading = isInitialWorkspaceListLoading || isWorkspaceListRefreshing;
	const workspacesLoadError = workspaceListError
		? getErrorMessage(workspaceListError, 'Workspaces could not be loaded.')
		: null;

	const [selection, setSelectionState] = useState<WorkspaceConversationSelection>();
	const [workspace, setWorkspace] = useState<WorkspaceView>();
	const [contexts, setContexts] = useState<WorkspaceContextView[]>([]);
	const [skills, setSkills] = useState<WorkspaceSkillView[]>([]);
	const [selectionLoading, setSelectionLoading] = useState(false);
	const [catalogKnown, setCatalogKnown] = useState(false);
	const [catalogRevision, setCatalogRevision] = useState<number | undefined>(undefined);

	const [selectionError, setSelectionError] = useState<string | null>(null);

	const selectionRef = useRef<WorkspaceConversationSelection | undefined>(undefined);
	const loadVersionRef = useRef(0);
	const mountedRef = useRef(true);

	useEffect(() => {
		mountedRef.current = true;
		return () => {
			mountedRef.current = false;
			loadVersionRef.current += 1;
		};
	}, []);

	const replaceSelection = useCallback((next?: WorkspaceConversationSelection) => {
		const cloned = cloneSelection(next);
		selectionRef.current = cloned;
		setSelectionState(cloned);
	}, []);

	const refreshWorkspaces = useCallback(async () => {
		try {
			await reloadWorkspaceList();
		} catch {
			// `useAsyncResource` retains the previous list and exposes the error.
		}
	}, [reloadWorkspaceList]);

	const applyResolvedWorkspaceSkillRefs = useCallback(
		async (
			nextSelection: WorkspaceConversationSelection | undefined,
			syncSession: SkillSelectionApplyOptions['syncSession'],
			loadedCatalog?: LoadedWorkspaceSelectionCatalog,
			forceResetSession = false
		) => {
			const workspaceSkills = loadedCatalog?.skills ?? skills;
			const workspaceEnabled = resolveWorkspaceSessionSkillRefs(nextSelection, workspaceSkills);

			const selectedKeys = new Set(
				workspaceEnabled.map(k => {
					return skillRefKey(k);
				})
			);
			const retainedWorkspaceActive = getCurrentActiveSkillRefs().filter(ref => selectedKeys.has(skillRefKey(ref)));

			await applyWorkspaceSkillSelectionState(nextSelection?.workspace, workspaceEnabled, retainedWorkspaceActive, {
				syncSession,
				forceResetSession,
			});
		},
		[applyWorkspaceSkillSelectionState, getCurrentActiveSkillRefs, skills]
	);

	const loadSelection = useCallback(
		async (
			nextSelection: WorkspaceConversationSelection,
			syncSession: SkillSessionSyncMode,
			forceResetSession = false
		) => {
			const version = loadVersionRef.current + 1;
			loadVersionRef.current = version;

			replaceSelection(nextSelection);
			setWorkspace(undefined);
			setContexts([]);
			setSkills([]);
			setCatalogKnown(false);
			setCatalogRevision(nextSelection.catalogRevision);
			setSelectionLoading(true);
			setSelectionError(null);

			try {
				const loaded = await loadWorkspaceSelectionCatalog(nextSelection.workspace);
				if (!mountedRef.current || loadVersionRef.current !== version) {
					return;
				}

				setWorkspace(loaded.workspace);
				setContexts(loaded.contexts);
				setSkills(loaded.skills);
				setCatalogKnown(loaded.catalogKnown);
				setCatalogRevision(loaded.catalogRevision ?? nextSelection.catalogRevision);

				await applyResolvedWorkspaceSkillRefs(
					loaded.workspace ? nextSelection : undefined,
					syncSession,
					loaded,
					forceResetSession
				);
				if (!mountedRef.current || loadVersionRef.current !== version) {
					return;
				}

				setSelectionError(loaded.errors.length > 0 ? loaded.errors.join(' ') : null);
			} catch (error) {
				if (!mountedRef.current || loadVersionRef.current !== version) {
					return;
				}

				// Do not leave Workspace refs from a failed selection in the
				// current Skill Runtime state.
				await applyResolvedWorkspaceSkillRefs(undefined, syncSession, undefined, forceResetSession);
				if (!mountedRef.current || loadVersionRef.current !== version) {
					return;
				}

				setSelectionError(getErrorMessage(error, 'The selected Workspace could not be loaded.'));
			} finally {
				if (mountedRef.current && loadVersionRef.current === version) {
					setSelectionLoading(false);
				}
			}
		},
		[applyResolvedWorkspaceSkillRefs, replaceSelection]
	);

	const attachWorkspace = useCallback(
		async (nextWorkspace: WorkspaceView) => {
			const version = loadVersionRef.current + 1;
			loadVersionRef.current = version;
			setSelectionLoading(true);
			setSelectionError(null);

			try {
				const loaded = await loadWorkspaceSelectionCatalog(nextWorkspace.workspace);
				if (!mountedRef.current || loadVersionRef.current !== version) {
					return;
				}
				if (!loaded.workspace) {
					throw new Error(loaded.errors.join(' ') || 'The selected Workspace is unavailable.');
				}

				const nextSelection: WorkspaceConversationSelection = {
					workspace: { ...loaded.workspace.workspace },
					displayName: loaded.workspace.displayName,
					workspaceRevision: loaded.workspace.revision,
					catalogRevision: loaded.catalogRevision,
					contextRefs: loaded.contexts
						.filter(c => {
							return contextIsEligible(c);
						})
						.map(c => {
							return contextSelectionRef(c);
						}),
					skillRefs: loaded.skills
						.filter(r => isWorkspaceSkillSessionEligible(r))
						.map(r => {
							return skillSelectionRef(r);
						}),
				};

				replaceSelection(nextSelection);
				setWorkspace(loaded.workspace);
				setContexts(loaded.contexts);
				setSkills(loaded.skills);
				setCatalogKnown(loaded.catalogKnown);
				setCatalogRevision(loaded.catalogRevision);
				await applyResolvedWorkspaceSkillRefs(nextSelection, SkillSessionSyncMode.EnsureIfEnabled, loaded);
				if (!mountedRef.current || loadVersionRef.current !== version) {
					return;
				}
				setSelectionError(loaded.errors.length > 0 ? loaded.errors.join(' ') : null);
			} catch (error) {
				if (mountedRef.current && loadVersionRef.current === version) {
					setSelectionError(getErrorMessage(error, 'The Workspace could not be selected.'));
				}
				throw error;
			} finally {
				if (mountedRef.current && loadVersionRef.current === version) {
					setSelectionLoading(false);
				}
			}
		},
		[applyResolvedWorkspaceSkillRefs, replaceSelection]
	);

	const restoreSelection = useCallback(
		async (nextSelection?: WorkspaceConversationSelection, syncSkills = true) => {
			if (!nextSelection) {
				// Invalidate an attach/restore request that is still loading.
				// Without this, its completion can reattach a Workspace after a
				// no-Workspace conversation or edited message has been restored.
				loadVersionRef.current += 1;
				replaceSelection(undefined);
				setWorkspace(undefined);
				setContexts([]);
				setSkills([]);
				setCatalogKnown(false);
				setCatalogRevision(undefined);
				setSelectionLoading(false);
				setSelectionError(null);

				await applyResolvedWorkspaceSkillRefs(
					undefined,
					syncSkills ? SkillSessionSyncMode.IfSessionExists : SkillSessionSyncMode.None
				);
				return;
			}

			await loadSelection(nextSelection, syncSkills ? SkillSessionSyncMode.EnsureIfEnabled : SkillSessionSyncMode.None);
		},
		[applyResolvedWorkspaceSkillRefs, loadSelection, replaceSelection]
	);

	const detachWorkspace = useCallback(
		async (syncSkills = true) => {
			loadVersionRef.current += 1;
			replaceSelection(undefined);
			setWorkspace(undefined);
			setContexts([]);
			setSkills([]);
			setCatalogKnown(false);
			setCatalogRevision(undefined);
			setSelectionError(null);
			setSelectionLoading(false);
			await applyResolvedWorkspaceSkillRefs(
				undefined,
				syncSkills ? SkillSessionSyncMode.IfSessionExists : SkillSessionSyncMode.None
			);
		},
		[applyResolvedWorkspaceSkillRefs, replaceSelection]
	);

	const updateSelectionFromCurrentContents = useCallback(async () => {
		if (!workspace) {
			return;
		}

		const current = selectionRef.current;
		const nextSelection: WorkspaceConversationSelection = {
			workspace: { ...workspace.workspace },
			displayName: workspace.displayName,
			workspaceRevision: workspace.revision,
			catalogRevision: catalogRevision ?? current?.catalogRevision,
			contextRefs: contexts
				.filter(c => {
					return contextIsEligible(c);
				})
				.map(c => {
					return contextSelectionRef(c);
				}),
			skillRefs: skills
				.filter(r => isWorkspaceSkillSessionEligible(r))
				.map(r => {
					return skillSelectionRef(r);
				}),
		};

		replaceSelection(nextSelection);
		const shouldResetExistingSession =
			(current?.skillRefs?.length ?? 0) > 0 || (nextSelection.skillRefs && nextSelection.skillRefs.length > 0);
		await applyResolvedWorkspaceSkillRefs(
			nextSelection,
			SkillSessionSyncMode.IfSessionExists,
			undefined,
			shouldResetExistingSession
		);
	}, [catalogRevision, contexts, applyResolvedWorkspaceSkillRefs, replaceSelection, skills, workspace]);

	const toggleContext = useCallback(
		(context: WorkspaceContextView, selected: boolean) => {
			const current = selectionRef.current;
			if (!current || (selected && !contextIsEligible(context))) {
				return;
			}

			const byID = new Map((current.contextRefs ?? []).map(ref => [artifactRefKey(ref.artifact), ref]));
			const key = artifactRefKey(context.artifact);

			if (selected) {
				byID.set(key, contextSelectionRef(context));
			} else {
				byID.delete(key);
			}

			replaceSelection({
				...current,
				contextRefs: [...byID.values()],
			});
		},
		[replaceSelection]
	);

	const toggleSkill = useCallback(
		async (skill: WorkspaceSkillView, selected: boolean) => {
			const current = selectionRef.current;
			const key = artifactRefKey(skill.artifact);
			if (!current || (selected && !isWorkspaceSkillSessionEligible(skill))) {
				return;
			}

			const byID = new Map((current.skillRefs ?? []).map(ref => [artifactRefKey(ref.artifact), ref]));

			if (selected) {
				byID.set(key, skillSelectionRef(skill));
			} else {
				byID.delete(key);
			}

			const nextSelection = {
				...current,
				skillRefs: [...byID.values()],
			};

			replaceSelection(nextSelection);
			await applyResolvedWorkspaceSkillRefs(nextSelection, SkillSessionSyncMode.IfSessionExists);
		},
		[applyResolvedWorkspaceSkillRefs, replaceSelection]
	);

	const removeContextRef = useCallback(
		(artifact: ArtifactRef) => {
			const current = selectionRef.current;
			if (!current) {
				return;
			}

			replaceSelection({
				...current,
				contextRefs: (current.contextRefs ?? []).filter(
					ref => artifactRefKey(ref.artifact) !== artifactRefKey(artifact)
				),
			});
		},
		[replaceSelection]
	);

	const removeSkillRef = useCallback(
		async (artifact: ArtifactRef) => {
			const current = selectionRef.current;
			if (!current) {
				return;
			}

			const nextSelection = {
				...current,
				skillRefs: (current.skillRefs ?? []).filter(ref => artifactRefKey(ref.artifact) !== artifactRefKey(artifact)),
			};
			replaceSelection(nextSelection);
			await applyResolvedWorkspaceSkillRefs(nextSelection, SkillSessionSyncMode.IfSessionExists);
		},
		[applyResolvedWorkspaceSkillRefs, replaceSelection]
	);

	const refreshSelectedWorkspace = useCallback(async () => {
		const current = selectionRef.current;
		if (!current) {
			return;
		}

		const refreshVersion = loadVersionRef.current + 1;
		loadVersionRef.current = refreshVersion;
		setSelectionLoading(true);
		setSelectionError(null);
		try {
			await workspaceAPI.refreshWorkspace(current.workspace);
			if (
				!mountedRef.current ||
				loadVersionRef.current !== refreshVersion ||
				!workspaceRefsEqual(selectionRef.current?.workspace, current.workspace)
			) {
				return;
			}

			await loadSelection(current, SkillSessionSyncMode.IfSessionExists, (current.skillRefs?.length ?? 0) > 0);
			if (!mountedRef.current || !workspaceRefsEqual(selectionRef.current?.workspace, current.workspace)) {
				return;
			}

			await refreshWorkspaces();
		} catch (error) {
			if (mountedRef.current && loadVersionRef.current === refreshVersion) {
				setSelectionError(getErrorMessage(error, 'The selected Workspace could not be refreshed.'));
			}
		} finally {
			if (mountedRef.current && loadVersionRef.current === refreshVersion) {
				setSelectionLoading(false);
			}
		}
	}, [loadSelection, refreshWorkspaces]);

	const createFilesystemWorkspace = useCallback(
		async (payload: CreateFilesystemWorkspaceInput) => {
			const normalizePath = (value: string) => value.trim().replaceAll('\\', '/').replaceAll(/\/+$/g, '').toLowerCase();
			const requestedPath = normalizePath(payload.rootPath);
			const existing = workspaces.find(
				candidate => candidate.primaryPath && normalizePath(candidate.primaryPath) === requestedPath
			);

			if (existing) {
				await attachWorkspace(existing);
				return;
			}

			const preferredRootID = workspace?.workspace.rootID ?? workspaces[0]?.workspace.rootID;
			const created = await createFilesystemWorkspaceCollection(payload, preferredRootID);
			const createdKey = workspaceRefKey(created.workspace);
			setWorkspaces(previous =>
				sortWorkspaces([...previous.filter(w => workspaceRefKey(w.workspace) !== createdKey), created])
			);
			await refreshWorkspaces();

			try {
				await workspaceAPI.refreshWorkspace(created.workspace);
				const refreshed = await workspaceAPI.getWorkspace(created.workspace);
				await attachWorkspace(refreshed);
			} catch (error) {
				const fallbackSelection: WorkspaceConversationSelection = {
					workspace: { ...created.workspace },
					displayName: created.displayName,
					workspaceRevision: created.revision,
					contextRefs: [],
					skillRefs: [],
				};

				replaceSelection(fallbackSelection);
				setWorkspace(created);
				setContexts([]);
				setSkills([]);
				setCatalogKnown(false);
				setCatalogRevision(undefined);
				await applyResolvedWorkspaceSkillRefs(fallbackSelection, SkillSessionSyncMode.EnsureIfEnabled);
				setSelectionError(
					`Workspace was created, but initial discovery failed. Retry refresh before selecting Context or Skills. ${getErrorMessage(
						error,
						''
					)}`.trim()
				);
			}
		},
		[
			applyResolvedWorkspaceSkillRefs,
			attachWorkspace,
			setWorkspaces,
			refreshWorkspaces,
			replaceSelection,
			workspace?.workspace.rootID,
			workspaces,
		]
	);

	const selectedContextIDs = useMemo(
		() => new Set((selection?.contextRefs ?? []).map(ref => artifactRefKey(ref.artifact))),
		[selection]
	);
	const selectedSkillIDs = useMemo(
		() => new Set((selection?.skillRefs ?? []).map(ref => artifactRefKey(ref.artifact))),
		[selection]
	);

	const currentContextByID = useMemo(
		() => new Map(contexts.map(context => [artifactRefKey(context.artifact), context])),
		[contexts]
	);
	const currentSkillByID = useMemo(
		() => new Map(skills.map(skill => [artifactRefKey(skill.artifact), skill])),
		[skills]
	);

	const missingContextRefs = useMemo(
		() =>
			catalogKnown
				? (selection?.contextRefs ?? []).filter(ref => !currentContextByID.has(artifactRefKey(ref.artifact)))
				: [],
		[catalogKnown, currentContextByID, selection]
	);
	const missingSkillRefs = useMemo(
		() =>
			catalogKnown
				? (selection?.skillRefs ?? []).filter(ref => !currentSkillByID.has(artifactRefKey(ref.artifact)))
				: [],
		[catalogKnown, currentSkillByID, selection]
	);

	const changedCount = useMemo(() => {
		if (!catalogKnown) {
			return 0;
		}

		let count = 0;

		for (const ref of selection?.contextRefs ?? []) {
			const current = currentContextByID.get(artifactRefKey(ref.artifact));
			if (current && ref.definitionDigest && current.definitionDigest !== ref.definitionDigest) {
				count += 1;
			}
		}

		for (const ref of selection?.skillRefs ?? []) {
			const current = currentSkillByID.get(artifactRefKey(ref.artifact));
			if (current && ref.definitionDigest && current.definitionDigest !== ref.definitionDigest) {
				count += 1;
			}
		}

		return count;
	}, [currentContextByID, currentSkillByID, selection, catalogKnown]);

	const unusableSelectedCount = useMemo(() => {
		if (!catalogKnown) {
			return 0;
		}

		let count = missingContextRefs.length + missingSkillRefs.length;

		for (const context of contexts) {
			if (selectedContextIDs.has(artifactRefKey(context.artifact)) && !contextIsEligible(context)) {
				count += 1;
			}
		}

		for (const skill of skills) {
			if (selectedSkillIDs.has(artifactRefKey(skill.artifact)) && !isWorkspaceSkillSessionEligible(skill)) {
				count += 1;
			}
		}

		return count;
	}, [
		contexts,
		missingContextRefs.length,
		missingSkillRefs.length,
		selectedContextIDs,
		selectedSkillIDs,
		skills,
		catalogKnown,
	]);

	const selectedCount = (selection?.contextRefs?.length ?? 0) + (selection?.skillRefs?.length ?? 0);
	const usableSelectedCount = Math.max(0, selectedCount - unusableSelectedCount);

	const blockingError = selectionLoading
		? 'Workspace selection is still resolving. Wait for it to finish before sending.'
		: !selection
			? null
			: !workspace
				? (selectionError ??
					'The selected Workspace is unavailable. Detach it or choose another Workspace before sending.')
				: workspace && !workspace.enabled
					? 'The selected Workspace is disabled. Enable it in Workspace management or detach it before sending.'
					: catalogKnown && selectedCount > 0 && usableSelectedCount === 0
						? 'None of the selected Workspace resources are currently usable. Refresh, change the selection, or detach the Workspace before sending.'
						: null;

	const attentionCount =
		unusableSelectedCount + changedCount + (selectionError ? 1 : 0) + (workspace && !workspace.enabled ? 1 : 0);

	const getSelectionSnapshot = useCallback(() => cloneSelection(selectionRef.current), []);

	return {
		workspaces,
		workspacesLoading,
		workspacesLoadError,
		selection,
		workspace,
		contexts,
		skills,
		selectionLoading,
		catalogKnown,
		catalogRevision,
		selectionError,
		blockingError,
		selectedContextIDs,
		selectedSkillIDs,
		missingContextRefs,
		missingSkillRefs,
		changedCount,
		attentionCount,
		refreshWorkspaces,
		attachWorkspace,
		restoreSelection,
		detachWorkspace,
		updateSelectionFromCurrentContents,
		toggleContext,
		toggleSkill,
		removeContextRef,
		removeSkillRef,
		refreshSelectedWorkspace,
		createFilesystemWorkspace,
		getSelectionSnapshot,
	};
}
