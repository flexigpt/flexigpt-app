import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { ProvidedSkill, SkillRef } from '@/spec/skill';
import { SkillSessionSyncMode } from '@/spec/skill';
import type {
	CreateFilesystemWorkspacePayload,
	WorkspaceContextView,
	WorkspaceConversationResourceSelectionRef,
	WorkspaceConversationSelection,
	WorkspaceConversationSkillSelectionRef,
	WorkspaceSkillView,
	WorkspaceView,
} from '@/spec/workspace';
import { WorkspaceRecordState, WorkspaceSkillInsert } from '@/spec/workspace';

import { workspaceAPI } from '@/apis/baseapi';

import type { LoadedWorkspaceSelectionCatalog } from '@/chats/composer/workspaces/workspace_selection_loader';
import { loadWorkspaceSelectionCatalog } from '@/chats/composer/workspaces/workspace_selection_loader';
import {
	createWorkspaceSkillRef,
	getWorkspaceSkillRefParts,
	isWorkspaceSkillRef,
	normalizeSkillRef,
	skillRefKey,
} from '@/skills/lib/skill_identity_utils';
import { sortWorkspaces } from '@/workspaces/lib/workspace_utils';

interface SkillSelectionApplyOptions {
	syncSession?: SkillSessionSyncMode;
	forceResetSession?: boolean;
}

interface UseComposerWorkspaceArgs {
	applySkillSelectionState: (
		enabled: SkillRef[],
		active: SkillRef[],
		options?: SkillSelectionApplyOptions
	) => Promise<void>;
	getCurrentEnabledSkillRefs: () => SkillRef[];
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
	workspaceSkillProvidersByRecordID: ReadonlyMap<string, ProvidedSkill>;
	changedCount: number;
	attentionCount: number;
	refreshWorkspaces: () => Promise<void>;
	attachWorkspace: (workspace: WorkspaceView) => Promise<void>;
	restoreSelection: (selection?: WorkspaceConversationSelection, syncSkills?: boolean) => Promise<void>;
	detachWorkspace: (syncSkills?: boolean) => Promise<void>;
	updateSelectionFromCurrentContents: () => Promise<void>;
	toggleContext: (context: WorkspaceContextView, selected: boolean) => void;
	toggleSkill: (skill: WorkspaceSkillView, selected: boolean) => Promise<void>;
	removeContextRef: (recordID: string) => void;
	removeSkillRef: (recordID: string) => Promise<void>;
	refreshSelectedWorkspace: () => Promise<void>;
	createFilesystemWorkspace: (payload: CreateFilesystemWorkspacePayload) => Promise<void>;
	getSelectionSnapshot: () => WorkspaceConversationSelection | undefined;
}

function contextIsEligible(context: WorkspaceContextView): boolean {
	return (
		context.enabled &&
		context.state === WorkspaceRecordState.Available &&
		context.catalogCurrent &&
		context.projectionValid &&
		!context.runtimeDisabled
	);
}

function skillProviderAllowsConversation(provider?: ProvidedSkill): boolean {
	return Boolean(
		provider &&
		provider.enabled &&
		provider.available &&
		provider.runtimeAllowed &&
		provider.catalogCurrent &&
		!provider.shadowed
	);
}

export function isWorkspaceSkillSessionEligible(skill: WorkspaceSkillView, provider?: ProvidedSkill): boolean {
	return (
		skill.skill.insert === WorkspaceSkillInsert.Instructions &&
		skill.skill.isEnabled &&
		skill.state === WorkspaceRecordState.Available &&
		skill.projectionValid &&
		skill.catalogCurrent &&
		!skill.runtimeDisabled &&
		skillProviderAllowsConversation(provider)
	);
}

function getWorkspaceSkillProvidersByRecordID(providedSkills: ProvidedSkill[]): Map<string, ProvidedSkill> {
	return new Map(
		providedSkills
			.filter(skill => skill.workspaceRecordID)
			.map(skill => [skill.workspaceRecordID as string, skill] as const)
	);
}

function resolveWorkspaceSessionSkillRefs(
	selection: WorkspaceConversationSelection | undefined,
	workspaceSkills: WorkspaceSkillView[],
	workspaceSkillProvidersByRecordID: ReadonlyMap<string, ProvidedSkill>
): SkillRef[] {
	if (!selection) {
		return [];
	}

	const skillsByRecordID = new Map(
		workspaceSkills.filter(skill => skill.rootID === selection.rootID).map(skill => [skill.recordID, skill] as const)
	);
	const refs: SkillRef[] = [];

	for (const selectionRef of selection.skillRefs ?? []) {
		const skill = skillsByRecordID.get(selectionRef.recordID);
		const provider = workspaceSkillProvidersByRecordID.get(selectionRef.recordID);

		if (!skill || !isWorkspaceSkillSessionEligible(skill, provider)) {
			continue;
		}

		const normalized = normalizeSkillRef({ identity: selectionRef.identity });
		const parts = normalized ? getWorkspaceSkillRefParts(normalized) : undefined;

		if (!normalized || !parts || parts.rootID !== selection.rootID || parts.recordID !== selectionRef.recordID) {
			continue;
		}

		refs.push(normalized);
	}

	return refs;
}

function contextSelectionRef(context: WorkspaceContextView): WorkspaceConversationResourceSelectionRef {
	return {
		recordID: context.recordID,
		name: context.name,
		locator: context.locator,
		definitionDigest: context.definitionDigest,
		recordRevision: context.recordRevision,
	};
}

function skillSelectionRef(skill: WorkspaceSkillView): WorkspaceConversationSkillSelectionRef {
	const ref = createWorkspaceSkillRef(skill.rootID, skill.recordID);
	if (!ref) {
		throw new Error('Workspace Skill has an invalid root or record identity.');
	}

	return {
		recordID: skill.recordID,
		identity: ref.identity,
		name: skill.skill.name,
		displayName: skill.skill.displayName,
		locator: skill.locator,
		definitionDigest: skill.definitionDigest,
		recordRevision: skill.recordRevision,
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
		// oxlint-disable-next-line oxc/no-map-spread
		contextRefs: (selection.contextRefs ?? []).map(ref => ({ ...ref })),
		// oxlint-disable-next-line oxc/no-map-spread
		skillRefs: (selection.skillRefs ?? []).map(ref => ({ ...ref })),
	};
}

function getErrorMessage(error: unknown, fallback: string): string {
	return error instanceof Error && error.message.trim() ? error.message : fallback;
}

export function useComposerWorkspace({
	applySkillSelectionState,
	getCurrentEnabledSkillRefs,
	getCurrentActiveSkillRefs,
}: UseComposerWorkspaceArgs): ComposerWorkspaceController {
	const [workspaces, setWorkspaces] = useState<WorkspaceView[]>([]);
	const [workspacesLoading, setWorkspacesLoading] = useState(false);
	const [workspacesLoadError, setWorkspacesLoadError] = useState<string | null>(null);

	const [selection, setSelectionState] = useState<WorkspaceConversationSelection>();
	const [workspace, setWorkspace] = useState<WorkspaceView>();
	const [contexts, setContexts] = useState<WorkspaceContextView[]>([]);
	const [skills, setSkills] = useState<WorkspaceSkillView[]>([]);
	const [selectionLoading, setSelectionLoading] = useState(false);
	const [catalogKnown, setCatalogKnown] = useState(false);
	const [catalogRevision, setCatalogRevision] = useState<number | undefined>(undefined);
	const [providedSkills, setProvidedSkills] = useState<ProvidedSkill[]>([]);
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

	const workspaceSkillProvidersByRecordID = useMemo(
		() => getWorkspaceSkillProvidersByRecordID(providedSkills),
		[providedSkills]
	);

	const refreshWorkspaces = useCallback(async () => {
		setWorkspacesLoading(true);
		setWorkspacesLoadError(null);
		try {
			const next = sortWorkspaces(await workspaceAPI.listWorkspaces());
			if (mountedRef.current) {
				setWorkspaces(next);
			}
		} catch (error) {
			if (mountedRef.current) {
				setWorkspacesLoadError(getErrorMessage(error, 'Workspaces could not be loaded.'));
			}
		} finally {
			if (mountedRef.current) {
				setWorkspacesLoading(false);
			}
		}
	}, []);

	useEffect(() => {
		// oxlint-disable-next-line jsreact-hooks/set-state-in-effect
		void refreshWorkspaces();
	}, [refreshWorkspaces]);

	const replaceWorkspaceSkillRefs = useCallback(
		async (
			nextSelection: WorkspaceConversationSelection | undefined,
			syncSession: SkillSelectionApplyOptions['syncSession'],
			loadedCatalog?: Pick<LoadedWorkspaceSelectionCatalog, 'skills' | 'providedSkills'>,
			forceResetSession = false
		) => {
			const installedEnabled = getCurrentEnabledSkillRefs().filter(ref => !isWorkspaceSkillRef(ref));
			const installedActive = getCurrentActiveSkillRefs().filter(ref => !isWorkspaceSkillRef(ref));

			const workspaceSkills = loadedCatalog?.skills ?? skills;
			const providersByRecordID = loadedCatalog
				? getWorkspaceSkillProvidersByRecordID(loadedCatalog.providedSkills)
				: workspaceSkillProvidersByRecordID;
			const workspaceEnabled = resolveWorkspaceSessionSkillRefs(nextSelection, workspaceSkills, providersByRecordID);

			const selectedKeys = new Set(
				workspaceEnabled.map(r => {
					return skillRefKey(r);
				})
			);
			const retainedWorkspaceActive = getCurrentActiveSkillRefs().filter(
				ref => isWorkspaceSkillRef(ref) && selectedKeys.has(skillRefKey(ref))
			);

			await applySkillSelectionState(
				[...installedEnabled, ...workspaceEnabled],
				[...installedActive, ...retainedWorkspaceActive],
				{
					syncSession,
					forceResetSession,
				}
			);
		},
		[
			applySkillSelectionState,
			getCurrentActiveSkillRefs,
			getCurrentEnabledSkillRefs,
			skills,
			workspaceSkillProvidersByRecordID,
		]
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
			setProvidedSkills([]);
			setCatalogKnown(false);
			setCatalogRevision(nextSelection.catalogRevision);
			setSelectionLoading(true);
			setSelectionError(null);

			const loaded = await loadWorkspaceSelectionCatalog(nextSelection.rootID);
			if (!mountedRef.current || loadVersionRef.current !== version) {
				return;
			}

			setWorkspace(loaded.workspace);
			setContexts(loaded.contexts);
			setSkills(loaded.skills);
			setProvidedSkills(loaded.providedSkills);
			setCatalogKnown(loaded.catalogKnown);
			setCatalogRevision(loaded.catalogRevision ?? nextSelection.catalogRevision);

			await replaceWorkspaceSkillRefs(nextSelection, syncSession, loaded, forceResetSession);
			if (!mountedRef.current || loadVersionRef.current !== version) {
				return;
			}

			setSelectionError(loaded.errors.length > 0 ? loaded.errors.join(' ') : null);
			setSelectionLoading(false);
		},
		[replaceSelection, replaceWorkspaceSkillRefs]
	);

	const attachWorkspace = useCallback(
		async (nextWorkspace: WorkspaceView) => {
			const version = loadVersionRef.current + 1;
			loadVersionRef.current = version;
			setSelectionLoading(true);
			setSelectionError(null);

			try {
				const loaded = await loadWorkspaceSelectionCatalog(nextWorkspace.rootID);
				if (!mountedRef.current || loadVersionRef.current !== version) {
					return;
				}
				if (!loaded.workspace) {
					throw new Error(loaded.errors.join(' ') || 'The selected Workspace is unavailable.');
				}

				const providersByRecordID = new Map(
					loaded.providedSkills
						.filter(skill => skill.workspaceRecordID)
						.map(skill => [skill.workspaceRecordID as string, skill] as const)
				);

				const nextSelection: WorkspaceConversationSelection = {
					rootID: loaded.workspace.rootID,
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
						.filter(r => {
							return isWorkspaceSkillSessionEligible(r, providersByRecordID.get(r.recordID));
						})
						.map(r => {
							return skillSelectionRef(r);
						}),
				};

				replaceSelection(nextSelection);
				setWorkspace(loaded.workspace);
				setContexts(loaded.contexts);
				setSkills(loaded.skills);
				setProvidedSkills(loaded.providedSkills);
				setCatalogKnown(loaded.catalogKnown);
				setCatalogRevision(loaded.catalogRevision);
				await replaceWorkspaceSkillRefs(nextSelection, SkillSessionSyncMode.EnsureIfEnabled, loaded);
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
		[replaceSelection, replaceWorkspaceSkillRefs]
	);

	const restoreSelection = useCallback(
		async (nextSelection?: WorkspaceConversationSelection, syncSkills = true) => {
			if (!nextSelection) {
				await replaceWorkspaceSkillRefs(
					undefined,
					syncSkills ? SkillSessionSyncMode.IfSessionExists : SkillSessionSyncMode.None
				);
				replaceSelection(undefined);
				setWorkspace(undefined);
				setContexts([]);
				setSkills([]);
				setProvidedSkills([]);
				setCatalogKnown(false);
				setCatalogRevision(undefined);
				setSelectionError(null);
				return;
			}

			await loadSelection(nextSelection, syncSkills ? SkillSessionSyncMode.EnsureIfEnabled : SkillSessionSyncMode.None);
		},
		[loadSelection, replaceSelection, replaceWorkspaceSkillRefs]
	);

	const detachWorkspace = useCallback(
		async (syncSkills = true) => {
			loadVersionRef.current += 1;
			replaceSelection(undefined);
			setWorkspace(undefined);
			setContexts([]);
			setSkills([]);
			setProvidedSkills([]);
			setCatalogKnown(false);
			setCatalogRevision(undefined);
			setSelectionError(null);
			setSelectionLoading(false);
			await replaceWorkspaceSkillRefs(
				undefined,
				syncSkills ? SkillSessionSyncMode.IfSessionExists : SkillSessionSyncMode.None
			);
		},
		[replaceSelection, replaceWorkspaceSkillRefs]
	);

	const updateSelectionFromCurrentContents = useCallback(async () => {
		if (!workspace) {
			return;
		}

		const current = selectionRef.current;
		const nextSelection: WorkspaceConversationSelection = {
			rootID: workspace.rootID,
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
				.filter(r => {
					return isWorkspaceSkillSessionEligible(r, workspaceSkillProvidersByRecordID.get(r.recordID));
				})
				.map(r => {
					return skillSelectionRef(r);
				}),
		};

		replaceSelection(nextSelection);
		const shouldResetExistingSession =
			(current?.skillRefs?.length ?? 0) > 0 || (nextSelection.skillRefs && nextSelection.skillRefs.length > 0);
		await replaceWorkspaceSkillRefs(
			nextSelection,
			SkillSessionSyncMode.IfSessionExists,
			undefined,
			shouldResetExistingSession
		);
	}, [
		catalogRevision,
		contexts,
		replaceSelection,
		replaceWorkspaceSkillRefs,
		skills,
		workspace,
		workspaceSkillProvidersByRecordID,
	]);

	const toggleContext = useCallback(
		(context: WorkspaceContextView, selected: boolean) => {
			const current = selectionRef.current;
			if (!current || (selected && !contextIsEligible(context))) {
				return;
			}

			const byID = new Map((current.contextRefs ?? []).map(ref => [ref.recordID, ref]));

			if (selected) {
				byID.set(context.recordID, contextSelectionRef(context));
			} else {
				byID.delete(context.recordID);
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
			if (
				!current ||
				(selected && !isWorkspaceSkillSessionEligible(skill, workspaceSkillProvidersByRecordID.get(skill.recordID)))
			) {
				return;
			}

			const byID = new Map((current.skillRefs ?? []).map(ref => [ref.recordID, ref]));

			if (selected) {
				byID.set(skill.recordID, skillSelectionRef(skill));
			} else {
				byID.delete(skill.recordID);
			}

			const nextSelection = {
				...current,
				skillRefs: [...byID.values()],
			};

			replaceSelection(nextSelection);
			await replaceWorkspaceSkillRefs(nextSelection, SkillSessionSyncMode.IfSessionExists);
		},
		[replaceSelection, replaceWorkspaceSkillRefs, workspaceSkillProvidersByRecordID]
	);

	const removeContextRef = useCallback(
		(recordID: string) => {
			const current = selectionRef.current;
			if (!current) {
				return;
			}

			replaceSelection({
				...current,
				contextRefs: (current.contextRefs ?? []).filter(ref => ref.recordID !== recordID),
			});
		},
		[replaceSelection]
	);

	const removeSkillRef = useCallback(
		async (recordID: string) => {
			const current = selectionRef.current;
			if (!current) {
				return;
			}

			const nextSelection = {
				...current,
				skillRefs: (current.skillRefs ?? []).filter(ref => ref.recordID !== recordID),
			};
			replaceSelection(nextSelection);
			await replaceWorkspaceSkillRefs(nextSelection, SkillSessionSyncMode.IfSessionExists);
		},
		[replaceSelection, replaceWorkspaceSkillRefs]
	);

	const refreshSelectedWorkspace = useCallback(async () => {
		const current = selectionRef.current;
		if (!current) {
			return;
		}

		setSelectionLoading(true);
		setSelectionError(null);
		try {
			await workspaceAPI.refreshWorkspace(current.rootID);
			if (selectionRef.current?.rootID === current.rootID) {
				await loadSelection(current, SkillSessionSyncMode.IfSessionExists, (current.skillRefs?.length ?? 0) > 0);
			}
			await refreshWorkspaces();
		} catch (error) {
			setSelectionError(getErrorMessage(error, 'The selected Workspace could not be refreshed.'));
		} finally {
			setSelectionLoading(false);
		}
	}, [loadSelection, refreshWorkspaces]);

	const createFilesystemWorkspace = useCallback(
		async (payload: CreateFilesystemWorkspacePayload) => {
			const normalizePath = (value: string) =>
				value.trim().replaceAll('\\', '/').replaceAll(/\/+$/g, '').toLocaleLowerCase();
			const requestedPath = normalizePath(payload.rootPath);
			const existing = workspaces.find(
				candidate => candidate.primaryPath && normalizePath(candidate.primaryPath) === requestedPath
			);

			if (existing) {
				await attachWorkspace(existing);
				return;
			}

			const created = await workspaceAPI.createFilesystemWorkspace(payload);
			setWorkspaces(previous => sortWorkspaces([...previous.filter(w => w.rootID !== created.rootID), created]));
			await refreshWorkspaces();

			try {
				await workspaceAPI.refreshWorkspace(created.rootID);
				const refreshed = await workspaceAPI.getWorkspace(created.rootID);
				await attachWorkspace(refreshed);
			} catch (error) {
				const fallbackSelection: WorkspaceConversationSelection = {
					rootID: created.rootID,
					displayName: created.displayName,
					workspaceRevision: created.revision,
					contextRefs: [],
					skillRefs: [],
				};

				replaceSelection(fallbackSelection);
				setWorkspace(created);
				setContexts([]);
				setSkills([]);
				setProvidedSkills([]);
				setCatalogKnown(false);
				setCatalogRevision(undefined);
				await replaceWorkspaceSkillRefs(fallbackSelection, SkillSessionSyncMode.EnsureIfEnabled);
				setSelectionError(
					`Workspace was created, but initial discovery failed. Retry refresh before selecting Context or Skills. ${getErrorMessage(
						error,
						''
					)}`.trim()
				);
			}
		},
		[attachWorkspace, refreshWorkspaces, replaceSelection, replaceWorkspaceSkillRefs, workspaces]
	);

	const selectedContextIDs = useMemo(
		() => new Set((selection?.contextRefs ?? []).map(ref => ref.recordID)),
		[selection]
	);
	const selectedSkillIDs = useMemo(() => new Set((selection?.skillRefs ?? []).map(ref => ref.recordID)), [selection]);

	const currentContextByID = useMemo(() => new Map(contexts.map(context => [context.recordID, context])), [contexts]);
	const currentSkillByID = useMemo(() => new Map(skills.map(skill => [skill.recordID, skill])), [skills]);

	const missingContextRefs = useMemo(
		() => (catalogKnown ? (selection?.contextRefs ?? []).filter(ref => !currentContextByID.has(ref.recordID)) : []),
		[catalogKnown, currentContextByID, selection]
	);
	const missingSkillRefs = useMemo(
		() => (catalogKnown ? (selection?.skillRefs ?? []).filter(ref => !currentSkillByID.has(ref.recordID)) : []),
		[catalogKnown, currentSkillByID, selection]
	);

	const changedCount = useMemo(() => {
		if (!catalogKnown) {
			return 0;
		}

		let count = 0;

		for (const ref of selection?.contextRefs ?? []) {
			const current = currentContextByID.get(ref.recordID);
			if (current && ref.definitionDigest && current.definitionDigest !== ref.definitionDigest) {
				count += 1;
			}
		}

		for (const ref of selection?.skillRefs ?? []) {
			const current = currentSkillByID.get(ref.recordID);
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
			if (selectedContextIDs.has(context.recordID) && !contextIsEligible(context)) {
				count += 1;
			}
		}

		for (const skill of skills) {
			if (
				selectedSkillIDs.has(skill.recordID) &&
				!isWorkspaceSkillSessionEligible(skill, workspaceSkillProvidersByRecordID.get(skill.recordID))
			) {
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
		workspaceSkillProvidersByRecordID,
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
		workspaceSkillProvidersByRecordID,
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
