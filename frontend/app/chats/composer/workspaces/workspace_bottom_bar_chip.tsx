import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle, FiCheck, FiFolderPlus, FiRefreshCw, FiSettings, FiX } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import type { SkillRef } from '@/spec/skill';
import { ArtifactState } from '@/spec/artifact';
import { WorkspaceDirectoryOrigin } from '@/spec/workspace';

import {
	ActionTriggerChipContent,
	actionTriggerChipMenuButtonClasses,
	actionTriggerChipSurfaceClasses,
	ActionTriggerMenu,
	ActionTriggerMenuBody,
	ActionTriggerMenuHeader,
	actionTriggerMenuHeaderRowClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuMetaClasses,
} from '@/components/action_trigger_chip';
import { ComposerDropdownSelectedPanel } from '@/components/composer_dropdown_selected_panel';
import { HoverTip } from '@/components/hover_tip';
import { SearchableMenuInput } from '@/components/searchmenu/searchable_menu';
import { useSearchableMenuState } from '@/components/searchmenu/searchable_menu_utils';

import type {
	ComposerWorkspaceCandidate,
	ComposerWorkspaceController,
} from '@/chats/composer/workspaces/use_composer_workspace';
import { WorkspaceDirectorySelectionModal } from '@/chats/composer/workspaces/workspace_selection_modal';
import { getWorkspaceContentCounts } from '@/workspaces/workspace_contents';
import { WorkspaceDirectoryRegistrationModal } from '@/workspaces/workspace_directory_registration_modal';

interface WorkspaceBottomBarChipProps {
	store: MenuStore;
	state: ComposerWorkspaceController;
	activeSkillRefs: SkillRef[];
	isInputLocked?: boolean;
}

function workspaceKey(candidate: ComposerWorkspaceCandidate): string {
	const artifact = candidate.workspace.workspace.artifact;
	return `${artifact.rootID}:${artifact.id}`;
}

function workspaceDisplayName(candidate: ComposerWorkspaceCandidate): string {
	const artifact = candidate.workspace.workspace.artifact;
	return artifact.displayName || artifact.logicalName;
}

function workspaceSubtitle(candidate: ComposerWorkspaceCandidate): string {
	if (candidate.workspace.origin === WorkspaceDirectoryOrigin.Default) {
		return `${candidate.directory.root.displayName} · Automatic setup`;
	}

	return candidate.workspace.manifestLocator ?? candidate.workspace.workspace.artifact.binding.locator;
}

function workspaceIsAvailable(candidate: ComposerWorkspaceCandidate): boolean {
	const artifact = candidate.workspace.workspace.artifact;

	return candidate.directory.enabled && artifact.enabled && artifact.state === ArtifactState.Available;
}

function candidateMatchesSearch(candidate: ComposerWorkspaceCandidate, rawQuery: string): boolean {
	const query = rawQuery.trim().toLocaleLowerCase();

	if (!query) {
		return true;
	}

	const artifact = candidate.workspace.workspace.artifact;

	return [
		workspaceDisplayName(candidate),
		candidate.workspace.workspace.description,
		workspaceSubtitle(candidate),
		candidate.directory.root.displayName,
		candidate.directory.root.id,
		artifact.logicalName,
		artifact.binding.locator,
		candidate.workspace.origin,
	]
		.filter(Boolean)
		.join('\n')
		.toLocaleLowerCase()
		.includes(query);
}

function selectedWorkspaceName(state: ComposerWorkspaceController): string {
	if (state.workspace) {
		return workspaceDisplayName(state.workspace);
	}

	return state.selection?.displayName || 'Workspace';
}

export function WorkspaceBottomBarChip({
	store,
	state,
	activeSkillRefs,
	isInputLocked = false,
}: WorkspaceBottomBarChipProps) {
	useEffect(() => {
		if (isInputLocked) {
			store.hide();
		}
	}, [isInputLocked, store]);

	return (
		<WorkspaceBottomBarChipContent
			store={store}
			state={state}
			activeSkillRefs={activeSkillRefs}
			isInputLocked={isInputLocked}
		/>
	);
}

function WorkspaceBottomBarChipContent({
	store,
	state,
	activeSkillRefs,
	isInputLocked = false,
}: WorkspaceBottomBarChipProps) {
	const open = useStoreState(store, 'open');
	const [search, setSearch] = useSearchableMenuState(open);
	const [isDirectoryRegistrationOpen, setIsDirectoryRegistrationOpen] = useState(false);
	const [isSelectionOpen, setIsSelectionOpen] = useState(false);
	const [actionError, setActionError] = useState<string | null>(null);

	const visibleWorkspaces = useMemo(
		() => state.workspaces.filter(candidate => candidateMatchesSearch(candidate, search)),
		[search, state.workspaces]
	);

	const availableWorkspaces = useMemo(
		() =>
			visibleWorkspaces.filter(candidate => {
				return (
					state.selection?.workspace.rootID !== candidate.workspace.workspace.artifact.rootID ||
					state.selection?.workspace.artifactID !== candidate.workspace.workspace.artifact.id
				);
			}),
		[state.selection?.workspace.artifactID, state.selection?.workspace.rootID, visibleWorkspaces]
	);

	const selected = Boolean(state.selection);
	const selectedName = selectedWorkspaceName(state);
	const contentCounts = getWorkspaceContentCounts(state.plan, {
		selectedContextIDs: state.selectedContextIDs,
		selectedSkillIDs: state.selectedSkillIDs,
		selectedMCPIDs: state.selectedMCPIDs,
	});
	const selectedFileCount = contentCounts.instructionFiles + contentCounts.referenceFiles;
	const selectedSkillCount = contentCounts.skills;
	const selectedMCPCount = contentCounts.mcpServers;
	const workspaceHeaderMeta = search.trim()
		? `${availableWorkspaces.length} match${availableWorkspaces.length === 1 ? '' : 'es'}`
		: `${availableWorkspaces.length} available`;

	const hoverContent = selected ? 'Change the attached Workspace' : 'Choose a Workspace';

	const selectedWorkspacePanel = selected ? (
		<div className="border-secondary/30 bg-secondary/10 rounded-xl border p-2">
			<div className="truncate text-xs font-semibold">{selectedName}</div>
			<div className="text-base-content/60 mt-1 text-xs">
				{selectedFileCount} files · {selectedSkillCount} Skills · {selectedMCPCount} MCP servers
			</div>
		</div>
	) : undefined;

	return (
		<div className="relative shrink-0" data-bottom-bar-workspace>
			<HoverTip content={hoverContent} placement="top" wrapperElement="div">
				<div
					className={`${actionTriggerChipSurfaceClasses} ${
						selected
							? state.attentionCount > 0
								? 'border-warning/50 bg-warning/10'
								: 'border-secondary/50 bg-secondary/10'
							: open
								? 'border-base-300 bg-base-300/60'
								: 'border-transparent'
					} ${isInputLocked ? 'opacity-60' : ''}`}
				>
					<MenuButton
						store={store}
						disabled={isInputLocked}
						className={actionTriggerChipMenuButtonClasses}
						aria-label="Select Workspace"
					>
						<ActionTriggerChipContent
							icon={<FiFolderPlus size={14} />}
							label={selected ? selectedName : 'Workspace'}
							suffix={
								selected ? (
									state.attentionCount > 0 ? (
										<FiAlertCircle size={14} className="text-warning" />
									) : (
										<FiCheck size={14} className="shrink-0" />
									)
								) : null
							}
							open={open}
						/>
					</MenuButton>
				</div>
			</HoverTip>

			<ActionTriggerMenu store={store}>
				<ActionTriggerMenuHeader>
					<SearchableMenuInput
						open={open}
						query={search}
						onQueryChange={setSearch}
						placeholder="Search Workspaces…"
						disabled={state.workspacesLoading && state.workspaces.length === 0}
						onEscape={() => {
							store.hide();
						}}
					/>

					<div className={actionTriggerMenuHeaderRowClasses}>
						<div className="min-w-0 flex-1 truncate text-xs font-semibold">Repository Workspaces</div>
						<div className="flex shrink-0 items-center gap-2">
							<span className={actionTriggerMenuMetaClasses}>{workspaceHeaderMeta}</span>
							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								disabled={isInputLocked || state.workspacesLoading}
								onClick={() => {
									void state.refreshWorkspaces();
								}}
								aria-label="Refresh Workspace directory list"
								title="Refresh Workspace directory list"
							>
								<FiRefreshCw size={12} />
							</button>
						</div>
					</div>

					{state.workspacesLoadError ? (
						<div className="alert alert-warning rounded-xl text-xs">
							<FiAlertCircle size={14} />
							<span>{state.workspacesLoadError}</span>
						</div>
					) : null}

					{actionError ? (
						<div className="alert alert-error rounded-xl text-xs">
							<FiAlertCircle size={14} />
							<span>{actionError}</span>
						</div>
					) : null}
				</ActionTriggerMenuHeader>

				<ActionTriggerMenuBody className="space-y-2">
					{state.workspacesLoading && state.workspaces.length === 0 ? (
						<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
							Loading Workspace directories...
						</div>
					) : availableWorkspaces.length === 0 ? (
						<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
							{selected ? 'All matching Workspaces are already selected below.' : 'No effective Workspaces match.'}
						</div>
					) : (
						<div className="space-y-1">
							{availableWorkspaces.map(candidate => {
								const candidateKey = workspaceKey(candidate);
								const available = workspaceIsAvailable(candidate);

								return (
									<MenuItem
										key={candidateKey}
										hideOnClick={false}
										disabled={!available || isInputLocked}
										className={`${actionTriggerMenuItemClasses} items-start`}
										onClick={() => {
											if (!available) {
												return;
											}

											setActionError(null);

											void state
												.attachWorkspace(candidate)
												.then(() => {
													store.hide();
												})
												.catch((error: unknown) => {
													setActionError(error instanceof Error ? error.message : 'Workspace could not be selected.');
												});
										}}
									>
										<FiFolderPlus size={14} className="mt-0.5 shrink-0" />
										<div className="min-w-0 flex-1">
											<div className="truncate text-xs font-medium">{workspaceDisplayName(candidate)}</div>
											<div className="text-base-content/60 truncate font-mono text-[10px]">
												{workspaceSubtitle(candidate)}
											</div>
										</div>
										{!available ? (
											<span className="badge badge-warning badge-xs">Unavailable</span>
										) : candidate.workspace.origin === WorkspaceDirectoryOrigin.Default ? (
											<span className="badge badge-info badge-xs">Automatic</span>
										) : (
											<span className="badge badge-ghost badge-xs">Repository</span>
										)}
									</MenuItem>
								);
							})}
						</div>
					)}

					<div className="border-base-300 flex justify-end border-t pt-2">
						<button
							type="button"
							className="btn btn-ghost btn-sm rounded-xl"
							disabled={isInputLocked}
							onClick={() => {
								store.hide();
								setIsDirectoryRegistrationOpen(true);
							}}
						>
							<FiFolderPlus size={14} />
							Add Workspace Directory
						</button>
					</div>
				</ActionTriggerMenuBody>

				<ComposerDropdownSelectedPanel
					label="Added Workspace"
					ariaLabel="Added Workspace"
					isEmpty={!selected}
					emptyLabel="Choose a Workspace above to add its repository contents to this conversation."
					help="A Workspace can add instruction files, reference files, Skills, and MCP servers. Only one Workspace can be attached."
					summary={
						selected ? (
							<>
								<span>
									1 Workspace · {selectedFileCount} files · {selectedSkillCount} Skills · {selectedMCPCount} MCP
								</span>
								{state.attentionCount > 0 ? (
									<span className="text-warning">{state.attentionCount} need attention</span>
								) : null}
							</>
						) : undefined
					}
					actions={
						selected ? (
							<>
								<button
									type="button"
									className="btn btn-ghost btn-xs rounded-lg"
									onClick={() => {
										store.hide();
										setIsSelectionOpen(true);
									}}
									title="Edit Workspace selection"
								>
									<FiSettings size={12} />
								</button>
								<button
									type="button"
									className="btn btn-ghost btn-xs rounded-lg"
									disabled={state.selectionLoading}
									onClick={() => {
										void state.refreshSelectedWorkspace();
									}}
									title="Refresh Workspace status"
								>
									<FiRefreshCw size={12} />
								</button>
								<button
									type="button"
									className="btn btn-ghost btn-xs text-error rounded-lg"
									disabled={isInputLocked}
									onClick={() => {
										void state.detachWorkspace().catch((error: unknown) => {
											setActionError(error instanceof Error ? error.message : 'Workspace could not be detached.');
										});
									}}
									title="Detach Workspace"
								>
									<FiX size={12} />
								</button>
							</>
						) : undefined
					}
				>
					{selectedWorkspacePanel}
				</ComposerDropdownSelectedPanel>
			</ActionTriggerMenu>

			<WorkspaceDirectoryRegistrationModal
				isOpen={isDirectoryRegistrationOpen}
				onClose={() => {
					setIsDirectoryRegistrationOpen(false);
				}}
				title="Add Workspace Directory to Conversation"
				description="Choose a repository directory. Its effective Workspace will be selected for this conversation after discovery succeeds."
				onRegister={async path => {
					setActionError(null);

					try {
						await state.createWorkspaceDirectory(path);
						store.hide();
					} catch (error) {
						const message = error instanceof Error ? error.message : 'Workspace directory could not be registered.';
						setActionError(message);
						throw error;
					}
				}}
			/>

			<WorkspaceDirectorySelectionModal
				isOpen={isSelectionOpen}
				onClose={() => {
					setIsSelectionOpen(false);
				}}
				state={state}
				activeSkillRefs={activeSkillRefs}
				isInputLocked={isInputLocked}
			/>
		</div>
	);
}
