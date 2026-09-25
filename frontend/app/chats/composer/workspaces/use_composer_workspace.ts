import { useCallback, useMemo, useRef, useState } from 'react';

import type { ArtifactRef, CapabilityOccurrence } from '@/spec/artifact';
import type { SkillRef } from '@/spec/skill';
import type {
	WorkspaceConversationResourceSelectionRef,
	WorkspaceConversationSelection,
	WorkspaceDirectoryView,
	WorkspacePromptContribution,
	WorkspaceRef,
	WorkspaceRuntimePlan,
	WorkspaceSkill,
} from '@/spec/workspace';
import { ArtifactState } from '@/spec/artifact';
import { SkillSessionSyncMode } from '@/spec/skill';
import { WorkspaceInsertTarget } from '@/spec/workspace';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { workspaceManagementAPI } from '@/apis/baseapi';

interface SkillSelectionApplyOptions {
	syncSession?: SkillSessionSyncMode;
	forceResetSession?: boolean;
}

export interface UseComposerWorkspaceArgs {
	applyWorkspaceSkillSelectionState: (
		workspace: WorkspaceRef | undefined,
		workspaceEnabled: SkillRef[],
		workspaceActive: SkillRef[],
		options?: SkillSelectionApplyOptions
	) => Promise<void>;
	getCurrentActiveSkillRefs: () => SkillRef[];
}

export interface ComposerWorkspaceCandidate {
	directory: WorkspaceDirectoryView;
	workspace: WorkspaceDirectoryView['workspaces'][number];
}

export interface ComposerWorkspaceController {
	directories: WorkspaceDirectoryView[];
	workspaces: ComposerWorkspaceCandidate[];
	workspacesLoading: boolean;
	workspacesLoadError: string | null;

	selection: WorkspaceConversationSelection | undefined;
	workspace: ComposerWorkspaceCandidate | undefined;
	plan: WorkspaceRuntimePlan | undefined;
	contexts: WorkspacePromptContribution[];
	skills: WorkspaceSkill[];

	selectionLoading: boolean;
	catalogKnown: boolean;
	selectionError: string | null;
	blockingError: string | null;
	selectedContextIDs: Set<string>;
	selectedSkillIDs: Set<string>;
	missingContextRefs: WorkspaceConversationResourceSelectionRef[];
	missingSkillRefs: WorkspaceConversationResourceSelectionRef[];
	changedCount: number;
	attentionCount: number;
	capabilityIssues: CapabilityOccurrence[];

	refreshWorkspaces: () => Promise<void>;
	attachWorkspace: (workspace: ComposerWorkspaceCandidate) => Promise<void>;
	restoreSelection: (selection?: WorkspaceConversationSelection, syncSkills?: boolean) => Promise<void>;
	detachWorkspace: (syncSkills?: boolean) => Promise<void>;
	updateSelectionFromCurrentContents: () => Promise<void>;
	toggleContext: (context: WorkspacePromptContribution, selected: boolean) => void;
	toggleSkill: (skill: WorkspaceSkill, selected: boolean) => Promise<void>;
	removeContextRef: (artifact: ArtifactRef) => void;
	removeSkillRef: (artifact: ArtifactRef) => Promise<void>;
	refreshSelectedWorkspace: () => Promise<void>;
	createWorkspaceDirectory: (path: string) => Promise<void>;
	getSelectionSnapshot: () => WorkspaceConversationSelection | undefined;
}

function refKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

function workspaceRefOf(candidate: ComposerWorkspaceCandidate): ArtifactRef {
	return {
		rootID: candidate.workspace.workspace.artifact.rootID,
		artifactID: candidate.workspace.workspace.artifact.id,
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
		contextRefs: selection.contextRefs?.map(ref => ({
			...ref,
			artifact: { ...ref.artifact },
		})),
		skillRefs: selection.skillRefs?.map(ref => ({
			...ref,
			artifact: { ...ref.artifact },
		})),
	};
}

function contextRefOf(value: WorkspacePromptContribution): WorkspaceConversationResourceSelectionRef {
	return {
		artifact: { ...value.artifact },
		name: value.name,
		locator: value.locator,
		definitionDigest: value.definitionDigest,
		artifactRevision: value.artifactRevision,
	};
}

function skillRefOf(value: WorkspaceSkill): WorkspaceConversationResourceSelectionRef {
	return {
		artifact: { ...value.artifact },
		name: value.name,
		locator: value.locator,
		definitionDigest: value.definitionDigest,
		artifactRevision: value.artifactRevision,
	};
}

function selectionFromPlan(
	candidate: ComposerWorkspaceCandidate,
	plan: WorkspaceRuntimePlan
): WorkspaceConversationSelection {
	return {
		workspace: workspaceRefOf(candidate),
		displayName: candidate.workspace.workspace.artifact.displayName,
		workspaceRevision: candidate.workspace.workspace.artifact.revision,
		contextRefs: plan.prompt.contributions.map(contextRefOf),
		skillRefs: plan.skills.skills
			.filter(skill => skill.insert === WorkspaceInsertTarget.Instructions)
			.map(r => {
				return skillRefOf(r);
			}),
	};
}

function candidatesFor(directories: WorkspaceDirectoryView[]): ComposerWorkspaceCandidate[] {
	return directories.flatMap(directory =>
		directory.workspaces.map(workspace => ({
			directory,
			workspace,
		}))
	);
}

function candidateForRef(
	candidates: ComposerWorkspaceCandidate[],
	ref: ArtifactRef
): ComposerWorkspaceCandidate | undefined {
	return candidates.find(candidate => refKey(workspaceRefOf(candidate)) === refKey(ref));
}

function isCandidateAvailable(candidate: ComposerWorkspaceCandidate): boolean {
	const artifact = candidate.workspace.workspace.artifact;
	return candidate.directory.enabled && artifact.enabled && artifact.state === ArtifactState.Available;
}

export function useComposerWorkspace({
	applyWorkspaceSkillSelectionState,
	getCurrentActiveSkillRefs,
}: UseComposerWorkspaceArgs): ComposerWorkspaceController {
	const loadDirectories = useCallback(async (signal: AbortSignal) => {
		const directories = await workspaceManagementAPI.listComposerWorkspaceDirectories();
		throwIfAborted(signal);
		return directories;
	}, []);

	const {
		data: directories,
		error: directoriesError,
		isLoading: initialDirectoriesLoading,
		isRefreshing: directoriesRefreshing,
		reloadOrThrow,
		setData: setDirectories,
	} = useAsyncResource(loadDirectories, {
		initialData: [] as WorkspaceDirectoryView[],
	});

	const [selection, setSelection] = useState<WorkspaceConversationSelection>();
	const [workspace, setWorkspace] = useState<ComposerWorkspaceCandidate>();
	const [plan, setPlan] = useState<WorkspaceRuntimePlan>();
	const [selectionLoading, setSelectionLoading] = useState(false);
	const [selectionError, setSelectionError] = useState<string | null>(null);

	const requestVersion = useRef(0);
	const selectionRef = useRef<WorkspaceConversationSelection | undefined>(undefined);

	const workspaces = useMemo(() => candidatesFor(directories), [directories]);

	const replaceSelection = useCallback((next: WorkspaceConversationSelection | undefined) => {
		const cloned = cloneSelection(next);
		selectionRef.current = cloned;
		setSelection(cloned);
	}, []);

	const syncWorkspaceSkills = useCallback(
		async (
			nextSelection: WorkspaceConversationSelection | undefined,
			nextPlan: WorkspaceRuntimePlan | undefined,
			syncSession: SkillSessionSyncMode
		) => {
			const selected = new Set((nextSelection?.skillRefs ?? []).map(ref => refKey(ref.artifact)));
			const enabled = (nextPlan?.skills.skills ?? [])
				.filter(skill => skill.insert === WorkspaceInsertTarget.Instructions && selected.has(refKey(skill.artifact)))
				.map(skill => skill.artifact as SkillRef);

			const enabledKeys = new Set(enabled.map(value => refKey(value)));
			const active = getCurrentActiveSkillRefs().filter(value => enabledKeys.has(refKey(value)));

			await applyWorkspaceSkillSelectionState(nextSelection?.workspace, enabled, active, { syncSession });
		},
		[applyWorkspaceSkillSelectionState, getCurrentActiveSkillRefs]
	);

	const resolveCandidate = useCallback(
		async (
			candidate: ComposerWorkspaceCandidate,
			existingSelection?: WorkspaceConversationSelection,
			syncSession = SkillSessionSyncMode.EnsureIfEnabled
		) => {
			const version = requestVersion.current + 1;
			requestVersion.current = version;
			setSelectionLoading(true);
			setSelectionError(null);

			try {
				const nextPlan = await workspaceManagementAPI.resolveWorkspaceRuntimePlan(workspaceRefOf(candidate), {
					requireComplete: false,
				});
				if (requestVersion.current !== version) {
					return;
				}

				const nextSelection = cloneSelection(existingSelection) ?? selectionFromPlan(candidate, nextPlan);

				replaceSelection(nextSelection);
				setWorkspace(candidate);
				setPlan(nextPlan);
				await syncWorkspaceSkills(nextSelection, nextPlan, syncSession);

				if (requestVersion.current === version) {
					setSelectionError(null);
				}
			} catch (cause) {
				if (requestVersion.current === version) {
					setPlan(undefined);
					setWorkspace(undefined);
					setSelectionError(cause instanceof Error ? cause.message : 'The selected Workspace could not be resolved.');
				}
				throw cause;
			} finally {
				if (requestVersion.current === version) {
					setSelectionLoading(false);
				}
			}
		},
		[replaceSelection, syncWorkspaceSkills]
	);

	const refreshWorkspaces = useCallback(async () => {
		workspaceManagementAPI.invalidateComposerWorkspaceCatalog();

		try {
			await reloadOrThrow();
		} catch {
			// The async resource publishes the error while retaining existing data.
		}
	}, [reloadOrThrow]);

	const attachWorkspace = useCallback(
		async (candidate: ComposerWorkspaceCandidate) => {
			if (!isCandidateAvailable(candidate)) {
				throw new Error('This Workspace is not currently enabled and available.');
			}
			await resolveCandidate(candidate);
		},
		[resolveCandidate]
	);

	const detachWorkspace = useCallback(
		async (syncSkills = true) => {
			requestVersion.current += 1;
			replaceSelection(undefined);
			setWorkspace(undefined);
			setPlan(undefined);
			setSelectionError(null);
			setSelectionLoading(false);

			if (syncSkills) {
				await syncWorkspaceSkills(undefined, undefined, SkillSessionSyncMode.IfSessionExists);
			}
		},
		[replaceSelection, syncWorkspaceSkills]
	);

	const restoreSelection = useCallback(
		async (nextSelection?: WorkspaceConversationSelection, syncSkills = true) => {
			if (!nextSelection) {
				await detachWorkspace(syncSkills);
				return;
			}

			let nextDirectories = directories;
			let candidate = candidateForRef(candidatesFor(nextDirectories), nextSelection.workspace);

			if (!candidate) {
				nextDirectories = await workspaceManagementAPI.listComposerWorkspaceDirectories();
				setDirectories(nextDirectories);
				candidate = candidateForRef(candidatesFor(nextDirectories), nextSelection.workspace);
			}

			if (!candidate) {
				replaceSelection(nextSelection);
				setWorkspace(undefined);
				setPlan(undefined);
				setSelectionError('The Workspace selected by this conversation is no longer effective in its directory.');
				await syncWorkspaceSkills(
					undefined,
					undefined,
					syncSkills ? SkillSessionSyncMode.IfSessionExists : SkillSessionSyncMode.None
				);
				return;
			}

			await resolveCandidate(
				candidate,
				nextSelection,
				syncSkills ? SkillSessionSyncMode.EnsureIfEnabled : SkillSessionSyncMode.None
			);
		},
		[detachWorkspace, directories, replaceSelection, resolveCandidate, setDirectories, syncWorkspaceSkills]
	);

	const updateSelectionFromCurrentContents = useCallback(async () => {
		if (!workspace || !plan) {
			return;
		}

		const next = selectionFromPlan(workspace, plan);
		replaceSelection(next);
		await syncWorkspaceSkills(next, plan, SkillSessionSyncMode.IfSessionExists);
	}, [plan, replaceSelection, syncWorkspaceSkills, workspace]);

	const toggleContext = useCallback(
		(context: WorkspacePromptContribution, selected: boolean) => {
			const current = selectionRef.current;
			if (!current) {
				return;
			}

			const refs = new Map((current.contextRefs ?? []).map(ref => [refKey(ref.artifact), ref]));
			if (selected) {
				refs.set(refKey(context.artifact), contextRefOf(context));
			} else {
				refs.delete(refKey(context.artifact));
			}

			replaceSelection({
				...current,
				contextRefs: [...refs.values()],
			});
		},
		[replaceSelection]
	);

	const toggleSkill = useCallback(
		async (skill: WorkspaceSkill, selected: boolean) => {
			if (skill.insert !== WorkspaceInsertTarget.Instructions) {
				return;
			}

			const current = selectionRef.current;
			if (!current) {
				return;
			}

			const refs = new Map((current.skillRefs ?? []).map(ref => [refKey(ref.artifact), ref]));
			if (selected) {
				refs.set(refKey(skill.artifact), skillRefOf(skill));
			} else {
				refs.delete(refKey(skill.artifact));
			}

			const next = {
				...current,
				skillRefs: [...refs.values()],
			};
			replaceSelection(next);
			await syncWorkspaceSkills(next, plan, SkillSessionSyncMode.IfSessionExists);
		},
		[plan, replaceSelection, syncWorkspaceSkills]
	);

	const removeContextRef = useCallback(
		(artifact: ArtifactRef) => {
			const current = selectionRef.current;
			if (!current) {
				return;
			}

			replaceSelection({
				...current,
				contextRefs: (current.contextRefs ?? []).filter(ref => refKey(ref.artifact) !== refKey(artifact)),
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

			const next = {
				...current,
				skillRefs: (current.skillRefs ?? []).filter(ref => refKey(ref.artifact) !== refKey(artifact)),
			};
			replaceSelection(next);
			await syncWorkspaceSkills(next, plan, SkillSessionSyncMode.IfSessionExists);
		},
		[plan, replaceSelection, syncWorkspaceSkills]
	);

	const refreshSelectedWorkspace = useCallback(async () => {
		const current = workspace;
		const currentSelection = selectionRef.current;
		if (!current || !currentSelection) {
			return;
		}

		const refreshed = await workspaceManagementAPI.refreshWorkspaceDirectory(current.directory.ref);
		setDirectories(previous =>
			previous.map(directory => (directory.ref.rootID === refreshed.ref.rootID ? refreshed : directory))
		);

		const candidate = candidateForRef(
			candidatesFor([...directories.filter(directory => directory.ref.rootID !== refreshed.ref.rootID), refreshed]),
			currentSelection.workspace
		);

		if (!candidate) {
			setWorkspace(undefined);
			setPlan(undefined);
			setSelectionError('The selected Workspace is no longer effective after refresh.');
			return;
		}

		await resolveCandidate(candidate, currentSelection, SkillSessionSyncMode.IfSessionExists);
	}, [directories, resolveCandidate, setDirectories, workspace]);

	const createWorkspaceDirectory = useCallback(
		async (path: string) => {
			const directory = await workspaceManagementAPI.registerWorkspaceDirectory(path);
			setDirectories(previous => {
				const withoutCurrent = previous.filter(value => value.ref.rootID !== directory.ref.rootID);
				return [...withoutCurrent, directory];
			});

			const candidate = candidatesFor([directory])[0];
			if (!candidate) {
				throw new Error(
					'The directory was registered, but it has no effective Workspace. Fix any Workspace manifest diagnostics and refresh.'
				);
			}

			await attachWorkspace(candidate);
		},
		[attachWorkspace, setDirectories]
	);

	const selectedContextIDs = useMemo(
		() => new Set((selection?.contextRefs ?? []).map(ref => refKey(ref.artifact))),
		[selection]
	);
	const selectedSkillIDs = useMemo(
		() => new Set((selection?.skillRefs ?? []).map(ref => refKey(ref.artifact))),
		[selection]
	);

	const contexts = plan?.prompt.contributions ?? [];
	const skills = plan?.skills.skills ?? [];
	const contextsByRef = new Map(contexts.map(value => [refKey(value.artifact), value]));
	const skillsByRef = new Map(skills.map(value => [refKey(value.artifact), value]));

	const missingContextRefs = (selection?.contextRefs ?? []).filter(ref => !contextsByRef.has(refKey(ref.artifact)));
	const missingSkillRefs = (selection?.skillRefs ?? []).filter(ref => !skillsByRef.has(refKey(ref.artifact)));

	const changedCount = [...(selection?.contextRefs ?? []), ...(selection?.skillRefs ?? [])].filter(ref => {
		const current = contextsByRef.get(refKey(ref.artifact)) ?? skillsByRef.get(refKey(ref.artifact));
		return (
			current !== undefined &&
			((ref.definitionDigest !== undefined && ref.definitionDigest !== current.definitionDigest) ||
				(ref.artifactRevision !== undefined && ref.artifactRevision !== current.artifactRevision))
		);
	}).length;

	const capabilityIssues = (plan?.capabilities.occurrences ?? []).filter(
		occurrence => occurrence.status !== 'available'
	);
	const blockingError = selectionLoading
		? 'Workspace selection is still resolving. Wait for it to finish before sending.'
		: selection && !workspace
			? (selectionError ?? 'The selected Workspace is no longer effective. Detach it or select another Workspace.')
			: workspace && !isCandidateAvailable(workspace)
				? 'The selected Workspace is disabled or unavailable.'
				: null;

	return {
		directories,
		workspaces,
		workspacesLoading: initialDirectoriesLoading || directoriesRefreshing,
		workspacesLoadError: directoriesError
			? directoriesError instanceof Error
				? directoriesError.message
				: 'Workspace directories could not be loaded.'
			: null,
		selection,
		workspace,
		plan,
		contexts,
		skills,
		selectionLoading,
		catalogKnown: plan !== undefined,
		selectionError,
		blockingError,
		selectedContextIDs,
		selectedSkillIDs,
		missingContextRefs,
		missingSkillRefs,
		changedCount,
		attentionCount:
			missingContextRefs.length +
			missingSkillRefs.length +
			changedCount +
			capabilityIssues.length +
			(selectionError ? 1 : 0),
		capabilityIssues,
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
		createWorkspaceDirectory,
		getSelectionSnapshot: () => cloneSelection(selectionRef.current),
	};
}
